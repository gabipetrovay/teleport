// Copyright 2023 Gravitational, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package externalaudit

import (
	"context"

	"github.com/gravitational/trace"

	externalauditv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/externalaudit/v1"
	"github.com/gravitational/teleport/api/types/externalaudit"
	conv "github.com/gravitational/teleport/api/types/externalaudit/convert/v1"
)

// Client is an external audit client that conforms to the following lib/services interfaces:
// * services.Externalaudits
type Client struct {
	grpcClient externalauditv1.ExternalAuditServiceClient
}

// NewClient creates a new external audit client.
func NewClient(grpcClient externalauditv1.ExternalAuditServiceClient) *Client {
	return &Client{
		grpcClient: grpcClient,
	}
}

// GetExternalAudit returns the specified external audit resource.
func (c *Client) GetExternalAudit(ctx context.Context, name string) (*externalaudit.ExternalAudit, error) {
	resp, err := c.grpcClient.GetExternalAudit(ctx, &externalauditv1.GetExternalAuditRequest{
		Name: name,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	externalAudit, err := conv.FromProto(resp.GetExternalAudit())
	return externalAudit, trace.Wrap(err)
}

// CreateExternalAudit creates external audit resource.
func (c *Client) CreateExternalAudit(ctx context.Context, in *externalaudit.ExternalAudit) (*externalaudit.ExternalAudit, error) {
	resp, err := c.grpcClient.CreateExternalAudit(ctx, &externalauditv1.CreateExternalAuditRequest{
		ExternalAudit: conv.ToProto(in),
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	out, err := conv.FromProto(resp.GetExternalAudit())
	return out, trace.Wrap(err)
}

// DeleteExternalAudit deletes external audit resource.
func (c *Client) DeleteExternalAudit(ctx context.Context, name string) error {
	_, err := c.grpcClient.DeleteExternalAudit(ctx, &externalauditv1.DeleteExternalAuditRequest{
		Name: name,
	})
	return trace.Wrap(err)
}

// SetClusterExternalAudit sets cluster external audit resource.
func (c *Client) SetClusterExternalAudit(ctx context.Context, in *externalaudit.ClusterExternalAudit) error {
	_, err := c.grpcClient.SetClusterExternalAudit(ctx, &externalauditv1.SetClusterExternalAuditRequest{
		ClusterExternalAudit: conv.ToProtoClusterExternalAudit(in),
	})
	return trace.Wrap(err)
}

// DeleteClusterExternalAudit deletes cluster external audit resource.
func (c *Client) DeleteClusterExternalAudit(ctx context.Context) error {
	_, err := c.grpcClient.DeleteClusterExternalAudit(ctx, &externalauditv1.DeleteClusterExternalAuditRequest{})
	return trace.Wrap(err)
}
