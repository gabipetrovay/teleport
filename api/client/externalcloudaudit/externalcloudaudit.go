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

package externalcloudaudit

import (
	"context"

	"github.com/gravitational/trace"

	externalcloudauditv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/externalcloudaudit/v1"
	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	conv "github.com/gravitational/teleport/api/types/externalcloudaudit/convert/v1"
)

// Client is an external cloud audit client that conforms to the following lib/services interfaces:
// * services.Externalaudits
type Client struct {
	grpcClient externalcloudauditv1.ExternalCloudAuditServiceClient
}

// NewClient creates a new external cloud audit client.
func NewClient(grpcClient externalcloudauditv1.ExternalCloudAuditServiceClient) *Client {
	return &Client{
		grpcClient: grpcClient,
	}
}

// GetExternalCloudAudit returns the specified external cloud audit resource.
func (c *Client) GetExternalCloudAudit(ctx context.Context, name string) (*externalcloudaudit.ExternalCloudAudit, error) {
	resp, err := c.grpcClient.GetExternalCloudAudit(ctx, &externalcloudauditv1.GetExternalCloudAuditRequest{
		Name: name,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	externalAudit, err := conv.FromProto(resp.GetExternalCloudAudit())
	return externalAudit, trace.Wrap(err)
}

// CreateExternalCloudAudit creates external cloud audit resource.
func (c *Client) CreateExternalCloudAudit(ctx context.Context, in *externalcloudaudit.ExternalCloudAudit) (*externalcloudaudit.ExternalCloudAudit, error) {
	resp, err := c.grpcClient.CreateExternalCloudAudit(ctx, &externalcloudauditv1.CreateExternalCloudAuditRequest{
		ExternalCloudAudit: conv.ToProto(in),
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	out, err := conv.FromProto(resp.GetExternalCloudAudit())
	return out, trace.Wrap(err)
}

// DeleteExternalCloudAudit deletes external cloud audit resource.
func (c *Client) DeleteExternalCloudAudit(ctx context.Context, name string) error {
	_, err := c.grpcClient.DeleteExternalCloudAudit(ctx, &externalcloudauditv1.DeleteExternalCloudAuditRequest{
		Name: name,
	})
	return trace.Wrap(err)
}

// EnableClusterExternalCloudAudit sets cluster external cloud audit resource.
func (c *Client) EnableClusterExternalCloudAudit(ctx context.Context, in *externalcloudaudit.ClusterExternalCloudAudit) error {
	_, err := c.grpcClient.EnableClusterExternalCloudAudit(ctx, &externalcloudauditv1.EnableClusterExternalCloudAuditRequest{
		ClusterExternalCloudAudit: conv.ToProtoClusterExternalAudit(in),
	})
	return trace.Wrap(err)
}

// DisableClusterExternalCloudAudit deletes cluster external cloud audit resource.
func (c *Client) DisableClusterExternalCloudAudit(ctx context.Context) error {
	_, err := c.grpcClient.DisableClusterExternalCloudAudit(ctx, &externalcloudauditv1.DisableClusterExternalCloudAuditRequest{})
	return trace.Wrap(err)
}
