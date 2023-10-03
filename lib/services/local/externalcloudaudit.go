/*
Copyright 2023 Gravitational, Inc.

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

package local

import (
	"context"
	"errors"
	"time"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/services/local/generic"
)

const (
	externalCloudAuditPrefix        = "external_cloud_audit"
	clusterExternalCloudAuditPrefix = "cluster_external_cloud_audit"
	externalCloudAuditMaxPageSize   = 100
	externalCloudAuditLockName      = "external_cloud_audit_lock"
	externalCloudAuditLockTTL       = 10 * time.Second
)

var ErrExternalCloudAuditDeleteProtection = errors.New("external cloud audit cannot be deleted because it's used in cluster")

// ExternalCloudAuditService manages external cloud audit resources in the Backend.
type ExternalCloudAuditService struct {
	externalCloudAuditService *generic.Service[*externalcloudaudit.ExternalCloudAudit]

	backend backend.Backend
}

// NewExternalCloudAuditService creates a new ExternalAuditService.
func NewExternalCloudAuditService(backend backend.Backend) (*ExternalCloudAuditService, error) {
	externalAuditService, err := generic.NewService(&generic.ServiceConfig[*externalcloudaudit.ExternalCloudAudit]{
		Backend:       backend,
		PageLimit:     externalCloudAuditMaxPageSize,
		ResourceKind:  types.KindExternalCloudAudit,
		BackendPrefix: externalCloudAuditPrefix,
		MarshalFunc:   services.MarshalExternalCloudAudit,
		UnmarshalFunc: services.UnmarshalExternalCloudAudit,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &ExternalCloudAuditService{
		externalCloudAuditService: externalAuditService,
		backend:                   backend,
	}, nil
}

// GetExternalAudits returns the all external cloud audit resources.
func (s *ExternalCloudAuditService) GetExternalCloudAudits(ctx context.Context) ([]*externalcloudaudit.ExternalCloudAudit, error) {
	externalAudits, err := s.externalCloudAuditService.GetResources(ctx)
	return externalAudits, trace.Wrap(err)
}

// GetExternalAudit returns the specified external cloud audit resource.
func (s *ExternalCloudAuditService) GetExternalCloudAudit(ctx context.Context, name string) (*externalcloudaudit.ExternalCloudAudit, error) {
	externalAudit, err := s.externalCloudAuditService.GetResource(ctx, name)
	return externalAudit, trace.Wrap(err)
}

// CreateExternalAudit creates external cloud audit resource.
func (s *ExternalCloudAuditService) CreateExternalCloudAudit(ctx context.Context, externalAudit *externalcloudaudit.ExternalCloudAudit) (*externalcloudaudit.ExternalCloudAudit, error) {
	return externalAudit, trace.Wrap(s.externalCloudAuditService.CreateResource(ctx, externalAudit))
}

// DeleteExternalAudit removes the specified external cloud audit resource.
func (s *ExternalCloudAuditService) DeleteExternalCloudAudit(ctx context.Context, name string) error {
	// Lock is used to prevent race between EnableClusterExternalCloudAudit and DeleteExternalCloudAudit
	err := backend.RunWhileLocked(ctx, backend.RunWhileLockedConfig{
		LockConfiguration: backend.LockConfiguration{
			Backend:  s.backend,
			LockName: externalCloudAuditLockName,
			TTL:      externalCloudAuditLockTTL,
		},
	}, func(ctx context.Context) error {
		got, err := s.GetClusterExternalCloudAudit(ctx)
		if err != nil {
			if !trace.IsNotFound(err) {
				return trace.Wrap(err)
			}
			// Not found happens when we don't have any cluster external audit.
			// In that case we can remove external audit.
		} else {
			if got.GetName() == name {
				return trace.Wrap(ErrExternalCloudAuditDeleteProtection)
			}
		}
		return trace.Wrap(s.externalCloudAuditService.DeleteResource(ctx, name))
	})

	return trace.Wrap(err)
}

func (s *ExternalCloudAuditService) EnableClusterExternalCloudAudit(ctx context.Context, in *externalcloudaudit.ClusterExternalCloudAudit) error {
	value, err := services.MarshalClusterExternalCloudAudit(in)
	if err != nil {
		return trace.Wrap(err)
	}

	// Lock is used to prevent race between EnableClusterExternalCloudAudit and DeleteExternalCloudAudit
	err = backend.RunWhileLocked(ctx, backend.RunWhileLockedConfig{
		LockConfiguration: backend.LockConfiguration{
			Backend:  s.backend,
			LockName: externalCloudAuditLockName,
			TTL:      externalCloudAuditLockTTL,
		},
	}, func(ctx context.Context) error {
		_, err := s.GetExternalCloudAudit(ctx, in.Spec.ExternalCloudAuditName)
		if err != nil {
			if trace.IsNotFound(err) {
				return trace.BadParameter("cannot set %s as cluster external cloud audit resource, because external cloud audit not exists", in.Spec.ExternalCloudAuditName)
			}
			return trace.Wrap(err)
		}
		_, err = s.backend.Put(ctx, backend.Item{
			Key:   backend.Key(clusterExternalCloudAuditPrefix),
			Value: value,
		})
		return trace.Wrap(err)
	})

	return trace.Wrap(err)
}

func (s *ExternalCloudAuditService) GetClusterExternalCloudAudit(ctx context.Context) (*externalcloudaudit.ExternalCloudAudit, error) {
	item, err := s.backend.Get(ctx, backend.Key(clusterExternalCloudAuditPrefix))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	clusterExternalAudit, err := services.UnmarshalClusterExternalCloudAudit(item.Value)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	out, err := s.GetExternalCloudAudit(ctx, clusterExternalAudit.Spec.ExternalCloudAuditName)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return out, nil
}

func (s *ExternalCloudAuditService) DisableClusterExternalCloudAudit(ctx context.Context) error {
	err := s.backend.Delete(ctx, backend.Key(clusterExternalCloudAuditPrefix))
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}
