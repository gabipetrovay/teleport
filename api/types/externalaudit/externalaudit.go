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

package externalaudit

import (
	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/types/header"
	"github.com/gravitational/teleport/api/types/header/convert/legacy"
	"github.com/gravitational/teleport/api/utils"
)

// ExternalAudit
type ExternalAudit struct {
	// ResourceHeader is the common resource header for all resources.
	header.ResourceHeader

	// Spec is the specification for the external audit.
	Spec ExternalAuditSpec `json:"spec" yaml:"spec"`
}

// ExternalAuditSpec is the specification for an external audit.
type ExternalAuditSpec struct {
	// IntegrationName is name of existing OIDC intagration used to
	// generate AWS credentials.
	IntegrationName string `json:"integration_name" yaml:"integration_name"`
	// SessionsRecordingsURI is s3 path used to store sessions recordings.
	SessionsRecordingsURI string `json:"sessions_recordings_uri" yaml:"sessions_recordings_uri"`
	// AthenaWorkspace is workspace used by Athena audit logs during queries.
	AthenaWorkspace string `json:"athena_workspace" yaml:"athena_workspace"`
	// GlueDatabase is database used by Athena audit logs during queries.
	GlueDatabase string `json:"glue_database" yaml:"glue_database"`
	// GlueTable is table used by Athena audit logs during queries.
	GlueTable string `json:"glue_table" yaml:"glue_table"`
	// AuditEventsLongTermURI is s3 path used to store batched parquet files with
	// audit events, partitioned by event date.
	AuditEventsLongTermURI string `json:"audit_events_long_term_uri" yaml:"audit_events_long_term_uri"`
	// AthenaResultsURI is s3 path used to store temporary results generated by
	// Athena engine.
	AthenaResultsURI string `json:"athena_results_uri" yaml:"athena_results_uri"`
}

// NewExternalAudit will create a new external audit.
func NewExternalAudit(metadata header.Metadata, spec ExternalAuditSpec) (*ExternalAudit, error) {
	externalaudit := &ExternalAudit{
		ResourceHeader: header.ResourceHeaderFromMetadata(metadata),
		Spec:           spec,
	}

	if err := externalaudit.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	return externalaudit, nil
}

// CheckAndSetDefaults validates fields and populates empty fields with default values.
func (a *ExternalAudit) CheckAndSetDefaults() error {
	a.SetKind(types.KindExternalAudit)
	a.SetVersion(types.V1)

	if err := a.ResourceHeader.CheckAndSetDefaults(); err != nil {
		return trace.Wrap(err)
	}

	if a.Spec.IntegrationName == "" {
		return trace.BadParameter("external audit integration_name required")
	}
	if a.Spec.SessionsRecordingsURI == "" {
		return trace.BadParameter("external audit sessions_recordings_uri required")
	}
	if a.Spec.AthenaWorkspace == "" {
		return trace.BadParameter("external audit athena_workspace required")
	}
	if a.Spec.GlueDatabase == "" {
		return trace.BadParameter("external audit glue_database required")
	}
	if a.Spec.GlueTable == "" {
		return trace.BadParameter("external audit glue_table required")
	}
	if a.Spec.AuditEventsLongTermURI == "" {
		return trace.BadParameter("external audit audit_events_long_term_uri required")
	}
	if a.Spec.AthenaResultsURI == "" {
		return trace.BadParameter("external audit athena_results_uri required")
	}

	return nil
}

// GetMetadata returns metadata. This is specifically for conforming to the Resource interface,
// and should be removed when possible.
func (a *ExternalAudit) GetMetadata() types.Metadata {
	return legacy.FromHeaderMetadata(a.Metadata)
}

// MatchSearch goes through select field values of a resource
// and tries to match against the list of search values.
func (a *ExternalAudit) MatchSearch(values []string) bool {
	fieldVals := append(utils.MapToStrings(a.GetAllLabels()), a.GetName())
	return types.MatchSearch(fieldVals, values, nil)
}

// CloneResource returns a copy of the resource as types.ResourceWithLabels.
func (a *ExternalAudit) CloneResource() types.ResourceWithLabels {
	var copy *ExternalAudit
	utils.StrictObjectToStruct(a, &copy)
	return copy
}

// ClusterExternalAudit
type ClusterExternalAudit struct {
	// ResourceHeader is the common resource header for all resources.
	header.ResourceHeader

	// Spec is the specification for the external audit.
	Spec ClusterExternalAuditSpec `json:"spec" yaml:"spec"`
}

// ClusterExternalAuditSpec is the specification for an external audit.
type ClusterExternalAuditSpec struct {
	// ExternalAuditName is name of existing external audit configuration
	// that will be used as cluster external audit.
	ExternalAuditName string `json:"external_audit_name" yaml:"external_audit_name"`
}

// NewClusterExternalAudit will create a new cluster external audit.
func NewClusterExternalAudit(metadata header.Metadata, spec ClusterExternalAuditSpec) (*ClusterExternalAudit, error) {
	clusterexternalaudit := &ClusterExternalAudit{
		ResourceHeader: header.ResourceHeaderFromMetadata(metadata),
		Spec:           spec,
	}

	if err := clusterexternalaudit.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	return clusterexternalaudit, nil
}

// CheckAndSetDefaults validates fields and populates empty fields with default values.
func (a *ClusterExternalAudit) CheckAndSetDefaults() error {
	a.SetKind(types.KindClusterExternalAudit)
	a.SetVersion(types.V1)
	a.SetName(types.KindClusterExternalAudit)

	if err := a.ResourceHeader.CheckAndSetDefaults(); err != nil {
		return trace.Wrap(err)
	}
	if a.Spec.ExternalAuditName == "" {
		return trace.BadParameter("external audit name required")
	}
	return nil
}

// GetMetadata returns metadata. This is specifically for conforming to the Resource interface,
// and should be removed when possible.
func (a *ClusterExternalAudit) GetMetadata() types.Metadata {
	return legacy.FromHeaderMetadata(a.Metadata)
}

// MatchSearch goes through select field values of a resource
// and tries to match against the list of search values.
func (a *ClusterExternalAudit) MatchSearch(values []string) bool {
	fieldVals := append(utils.MapToStrings(a.GetAllLabels()), a.GetName())
	return types.MatchSearch(fieldVals, values, nil)
}

// CloneResource returns a copy of the resource as types.ResourceWithLabels.
func (a *ClusterExternalAudit) CloneResource() types.ResourceWithLabels {
	var copy *ClusterExternalAudit
	utils.StrictObjectToStruct(a, &copy)
	return copy
}
