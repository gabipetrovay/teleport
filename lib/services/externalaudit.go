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

package services

import (
	"context"

	"github.com/gravitational/trace"

	externalauditclient "github.com/gravitational/teleport/api/client/externalaudit"
	"github.com/gravitational/teleport/api/types/externalaudit"
	"github.com/gravitational/teleport/lib/utils"
)

var _ ExternalAudit = (*externalauditclient.Client)(nil)

// ExternalAuditGetter defines an interface for reading external audits.
type ExternalAuditGetter interface {
	// GetExternalAudit returns the specified external audit resource.
	GetExternalAudit(context.Context, string) (*externalaudit.ExternalAudit, error)
}

// ExternalAudit defines an interface for managing ExternalAudit.
type ExternalAudit interface {
	ExternalAuditGetter

	// CreateExternalAudit creates an external audit resource.
	CreateExternalAudit(context.Context, *externalaudit.ExternalAudit) (*externalaudit.ExternalAudit, error)
	// DeleteExternalAudit deletes an external audit resource.
	DeleteExternalAudit(context.Context, string) error
	// SetClusterExternalAudit sets cluster external audit resource.
	SetClusterExternalAudit(context.Context, *externalaudit.ClusterExternalAudit) error
	// DeleteClusterExternalAudit deletes an cluster external audit resource.
	DeleteClusterExternalAudit(context.Context) error
}

// UnmarshalExternalAudit unmarshals the external audit resource from JSON.
func UnmarshalExternalAudit(data []byte, opts ...MarshalOption) (*externalaudit.ExternalAudit, error) {
	if len(data) == 0 {
		return nil, trace.BadParameter("missing external audit data")
	}
	cfg, err := CollectOptions(opts)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	var out *externalaudit.ExternalAudit
	if err := utils.FastUnmarshal(data, &out); err != nil {
		return nil, trace.BadParameter(err.Error())
	}
	if err := out.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	if cfg.ID != 0 {
		out.SetResourceID(cfg.ID)
	}
	if cfg.Revision != "" {
		out.SetRevision(cfg.Revision)
	}
	if !cfg.Expires.IsZero() {
		out.SetExpiry(cfg.Expires)
	}
	return out, nil
}

// MarshalExternalAudit marshals the external audit resource to JSON.
func MarshalExternalAudit(externalaudit *externalaudit.ExternalAudit, opts ...MarshalOption) ([]byte, error) {
	if err := externalaudit.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	cfg, err := CollectOptions(opts)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if !cfg.PreserveResourceID {
		copy := *externalaudit
		copy.SetResourceID(0)
		copy.SetRevision("")
		externalaudit = &copy
	}
	return utils.FastMarshal(externalaudit)
}

// UnmarshalClusterExternalAudit unmarshals the cluster external audit resource from JSON.
func UnmarshalClusterExternalAudit(data []byte, opts ...MarshalOption) (*externalaudit.ClusterExternalAudit, error) {
	if len(data) == 0 {
		return nil, trace.BadParameter("missing external audit data")
	}
	cfg, err := CollectOptions(opts)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	var out *externalaudit.ClusterExternalAudit
	if err := utils.FastUnmarshal(data, &out); err != nil {
		return nil, trace.BadParameter(err.Error())
	}
	if err := out.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	if cfg.ID != 0 {
		out.SetResourceID(cfg.ID)
	}
	if cfg.Revision != "" {
		out.SetRevision(cfg.Revision)
	}
	if !cfg.Expires.IsZero() {
		out.SetExpiry(cfg.Expires)
	}
	return out, nil
}

// MarshalClusterExternalAudit marshals the cluster external audit resource to JSON.
func MarshalClusterExternalAudit(in *externalaudit.ClusterExternalAudit, opts ...MarshalOption) ([]byte, error) {
	if err := in.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	cfg, err := CollectOptions(opts)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if !cfg.PreserveResourceID {
		copy := *in
		copy.SetResourceID(0)
		copy.SetRevision("")
		in = &copy
	}
	return utils.FastMarshal(in)
}
