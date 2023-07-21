/*
Copyright 2020 Gravitational, Inc.

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

package client

import (
	"context"
	"errors"

	"github.com/gravitational/trace"
	"github.com/gravitational/trace/trail"
	"google.golang.org/grpc"

	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/metadata"
)

// ErrAdminActionMFARequired is returned when an admin action is missing required MFA verification.
var ErrAdminActionMFARequired = trace.AccessDenied("MFA is required for admin-level API request")

// RetryWithMFAUnaryInterceptor intercepts a GRPC client unary call to check if the
// error indicates that the client should retry with MFA verification.
func (c *Client) RetryWithMFAUnaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// MFA is not supported.
	if c.c.PromptAdminRequestMFA == nil {
		return trace.Wrap(invoker(ctx, method, req, reply, cc, opts...))
	}

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err == nil {
		return nil
	} else if !errors.Is(trail.FromGRPC(err), ErrAdminActionMFARequired) {
		return trace.Wrap(err)
	}

	pingResp, err := c.PingWithCache(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	chall, err := c.CreateAuthenticateChallenge(ctx, &proto.CreateAuthenticateChallengeRequest{
		Request: &proto.CreateAuthenticateChallengeRequest_ContextUser{},
	})
	if err != nil {
		return trace.Wrap(err)
	}

	resp, err := c.c.PromptAdminRequestMFA(ctx, chall, pingResp.ProxyPublicAddr)
	if err != nil {
		return trace.Wrap(err)
	}

	opts = append(opts, metadata.WithMFACredentials(resp))
	return trace.Wrap(invoker(ctx, method, req, reply, cc, opts...))
}
