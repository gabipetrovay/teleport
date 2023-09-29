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

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/types/externalaudit"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/services/local/generic"
)

const (
	externalAuditPrefix        = "external_audit"
	clusterExternalAuditPrefix = "cluster_external_audit"
	externalAuditMaxPageSize   = 100
)

// ExternalAuditService manages external audit resources in the Backend.
type ExternalAuditService struct {
	externalAuditService *generic.Service[*externalaudit.ExternalAudit]

	backend backend.Backend
}

// NewExternalAuditService creates a new ExternalAuditService.
func NewExternalAuditService(backend backend.Backend) (*ExternalAuditService, error) {
	externalAuditService, err := generic.NewService(&generic.ServiceConfig[*externalaudit.ExternalAudit]{
		Backend:       backend,
		PageLimit:     externalAuditMaxPageSize,
		ResourceKind:  types.KindExternalAudit,
		BackendPrefix: externalAuditPrefix,
		MarshalFunc:   services.MarshalExternalAudit,
		UnmarshalFunc: services.UnmarshalExternalAudit,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &ExternalAuditService{
		externalAuditService: externalAuditService,
		backend:              backend,
	}, nil
}

// GetExternalAudits returns the all external audit resources.
func (s *ExternalAuditService) GetExternalAudits(ctx context.Context) ([]*externalaudit.ExternalAudit, error) {
	externalAudits, err := s.externalAuditService.GetResources(ctx)
	return externalAudits, trace.Wrap(err)
}

// GetExternalAudit returns the specified external audit resource.
func (s *ExternalAuditService) GetExternalAudit(ctx context.Context, name string) (*externalaudit.ExternalAudit, error) {
	externalAudit, err := s.externalAuditService.GetResource(ctx, name)
	return externalAudit, trace.Wrap(err)
}

// CreateExternalAudit creates external audit resource.
func (s *ExternalAuditService) CreateExternalAudit(ctx context.Context, externalAudit *externalaudit.ExternalAudit) (*externalaudit.ExternalAudit, error) {
	return externalAudit, trace.Wrap(s.externalAuditService.CreateResource(ctx, externalAudit))
}

// DeleteExternalAudit removes the specified external audit resource.
func (s *ExternalAuditService) DeleteExternalAudit(ctx context.Context, name string) error {
	return trace.Wrap(s.externalAuditService.DeleteResource(ctx, name))
}

func (s *ExternalAuditService) SetClusterExternalAudit(ctx context.Context, in *externalaudit.ClusterExternalAudit) error {
	value, err := services.MarshalClusterExternalAudit(in)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = s.backend.Put(ctx, backend.Item{
		Key:   backend.Key(clusterExternalAuditPrefix),
		Value: value,
	})
	return trace.Wrap(err)
}

func (s *ExternalAuditService) GetClusterExternalAudit(ctx context.Context) (*externalaudit.ExternalAudit, error) {
	item, err := s.backend.Get(ctx, backend.Key(clusterExternalAuditPrefix))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	clusterExternalAudit, err := services.UnmarshalClusterExternalAudit(item.Value)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	out, err := s.GetExternalAudit(ctx, clusterExternalAudit.Spec.ExternalAuditName)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return out, nil
}

func (s *ExternalAuditService) DeleteClusterExternalAudit(ctx context.Context) error {
	err := s.backend.Delete(ctx, backend.Key(clusterExternalAuditPrefix))
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}
