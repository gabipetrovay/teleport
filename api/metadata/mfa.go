/*
Copyright 2021 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metadata

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gravitational/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/gravitational/teleport/api/client/proto"
)

const mfaResponseToken = "mfa_challenge_response"

// WithMFACredentials can be called on a GRPC client request to attach
// MFA credentials to the GRPC metadata for requests that require MFA,
// like admin-level requests.
func WithMFACredentials(resp *proto.MFAAuthenticateResponse) grpc.CallOption {
	return grpc.PerRPCCredentials(&MFAPerRPCCredentials{MFAChallengeResponse: resp})
}

// MFACredentialsFromContext can be called from a GRPC server method to return
// MFA credentials added to the GRPC metadata for requests that require MFA,
// like admin-level requests.
func MFACredentialsFromContext(ctx context.Context) (*proto.MFAAuthenticateResponse, error) {
	values := metadata.ValueFromIncomingContext(ctx, mfaResponseToken)
	if len(values) == 0 {
		return nil, trace.NotFound("MFA credentials not found in the request metadata")
	}
	mfaChallengeResponseEnc := values[0]

	mfaChallengeResponseJSON, err := base64.StdEncoding.DecodeString(mfaChallengeResponseEnc)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	mfaChallengeResponse := &proto.MFAAuthenticateResponse{}
	if err := jsonpb.Unmarshal(bytes.NewReader([]byte(mfaChallengeResponseJSON)), mfaChallengeResponse); err != nil {
		return nil, trace.Wrap(err)
	}

	return mfaChallengeResponse, nil
}

// MFAPerRPCCredentials supplies PerRPCCredentials from an MFA challenge response.
type MFAPerRPCCredentials struct {
	MFAChallengeResponse *proto.MFAAuthenticateResponse
}

// GetRequestMetadata gets the request metadata as a map from a TokenSource.
func (mc *MFAPerRPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	ri, _ := credentials.RequestInfoFromContext(ctx)
	if err := credentials.CheckSecurityLevel(ri.AuthInfo, credentials.PrivacyAndIntegrity); err != nil {
		return nil, fmt.Errorf("unable to transfer MFA PerRPCCredentials: %v", err)
	}

	buf := new(bytes.Buffer)
	err := (&jsonpb.Marshaler{}).Marshal(buf, mc.MFAChallengeResponse)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	enc := base64.StdEncoding.EncodeToString(buf.Bytes())
	return map[string]string{
		mfaResponseToken: enc,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (mc *MFAPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}
