package client

import (
	"context"
	"testing"

	"github.com/gravitational/trace"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/metadata"
	"github.com/gravitational/teleport/api/types"
)

type mfaService struct {
	*proto.UnimplementedAuthServiceServer
}

func (s *mfaService) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	return &proto.PingResponse{}, nil
}

func (s *mfaService) CreateAuthenticateChallenge(ctx context.Context, req *proto.CreateAuthenticateChallengeRequest) (*proto.MFAAuthenticateChallenge, error) {
	return &proto.MFAAuthenticateChallenge{}, nil
}

const otpTestCode = "otp-test-code"

func (s *mfaService) UpsertRole(ctx context.Context, role *types.RoleV6) (*emptypb.Empty, error) {
	mfaResp, err := metadata.MFACredentialsFromContext(ctx)
	if trace.IsNotFound(err) {
		return nil, ErrAdminActionMFARequired
	} else if err != nil {
		return nil, trace.Wrap(ErrAdminActionMFARequired, "failed to retrieve MFA credentials from context with error: %v", err)
	}

	switch r := mfaResp.Response.(type) {
	case *proto.MFAAuthenticateResponse_TOTP:
		if r.TOTP.Code != otpTestCode {
			return nil, trace.AccessDenied("failed MFA verification")
		}
	default:
		return nil, trace.BadParameter("unexpected mfa response type %T", r)
	}

	return &emptypb.Empty{}, nil
}

// TestAdminRequestMFA test that MFA can be provided for admin requests through request metadata,
// both in the initial request with a call option or with a unary interceptor retry.
func TestAdminRequestMFA(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	server := startMockServer(t, &mfaService{})

	clt, err := server.NewClient(ctx, t)
	require.NoError(t, err)

	// UpsertRole should fail if the client is not set up to prompt for MFA
	// when an admin request returns ErrAdminActionMFARequired.

	err = clt.UpsertRole(ctx, &types.RoleV6{})
	require.Error(t, err)

	// Passing MFA credentials as a call option should pass the server MFA check.

	mfaTestResp := &proto.MFAAuthenticateResponse{
		Response: &proto.MFAAuthenticateResponse_TOTP{
			TOTP: &proto.TOTPResponse{
				Code: otpTestCode,
			},
		},
	}

	_, err = clt.grpc.UpsertRole(ctx, &types.RoleV6{}, metadata.WithMFACredentials(mfaTestResp))
	require.NoError(t, err)

	// UpsertRole should succeed when the client gets ErrAdminActionMFARequired
	// and automatically retries the request with MFA.

	clt.c.PromptAdminRequestMFA = func(ctx context.Context, chall *proto.MFAAuthenticateChallenge, proxyAddr string) (*proto.MFAAuthenticateResponse, error) {
		return mfaTestResp, nil
	}

	err = clt.UpsertRole(ctx, &types.RoleV6{})
	require.NoError(t, err)
}
