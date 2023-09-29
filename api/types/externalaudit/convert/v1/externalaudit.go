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

package v1

import (
	"github.com/gravitational/trace"

	externalauditv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/externalaudit/v1"
	"github.com/gravitational/teleport/api/types/externalaudit"
	headerv1 "github.com/gravitational/teleport/api/types/header/convert/v1"
)

// FromProto converts a v1 external audit into an internal external audit object.
func FromProto(in *externalauditv1.ExternalAudit) (*externalaudit.ExternalAudit, error) {
	if in == nil {
		return nil, trace.BadParameter("external audit message is nil")
	}

	if in.Spec == nil {
		return nil, trace.BadParameter("spec is missing")
	}
	externalaudit, err := externalaudit.NewExternalAudit(headerv1.FromMetadataProto(in.Header.Metadata), externalaudit.ExternalAuditSpec{
		IntegrationName:        in.Spec.IntegrationName,
		SessionsRecordingsURI:  in.Spec.SessionsRecordingsUri,
		AthenaWorkspace:        in.Spec.AthenaWorkspace,
		GlueDatabase:           in.Spec.GlueDatabase,
		GlueTable:              in.Spec.GlueTable,
		AuditEventsLongTermURI: in.Spec.AuditEventsLongTermUri,
		AthenaResultsURI:       in.Spec.AthenaResultsUri,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return externalaudit, nil
}

// ToProto converts an internal external audit into a v1 external audit object.
func ToProto(in *externalaudit.ExternalAudit) *externalauditv1.ExternalAudit {
	return &externalauditv1.ExternalAudit{
		Header: headerv1.ToResourceHeaderProto(in.ResourceHeader),
		Spec: &externalauditv1.ExternalAuditSpec{
			IntegrationName:        in.Spec.IntegrationName,
			SessionsRecordingsUri:  in.Spec.SessionsRecordingsURI,
			AthenaWorkspace:        in.Spec.AthenaWorkspace,
			GlueDatabase:           in.Spec.GlueDatabase,
			GlueTable:              in.Spec.GlueTable,
			AuditEventsLongTermUri: in.Spec.AuditEventsLongTermURI,
			AthenaResultsUri:       in.Spec.AthenaResultsURI,
		},
	}
}

// FromProtoClusterExternalAudit converts a v1 cluster external audit into an
// internal cluster external audit object.
func FromProtoClusterExternalAudit(in *externalauditv1.ClusterExternalAudit) (*externalaudit.ClusterExternalAudit, error) {
	if in == nil {
		return nil, trace.BadParameter("cluster external audit message is nil")
	}

	if in.Spec == nil {
		return nil, trace.BadParameter("spec is missing")
	}
	externalaudit, err := externalaudit.NewClusterExternalAudit(headerv1.FromMetadataProto(in.Header.Metadata), externalaudit.ClusterExternalAuditSpec{
		ExternalAuditName: in.Spec.ExternalAuditName,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return externalaudit, nil
}

// ToProtoClusterExternalAudit converts an internal cluster external audit into
// a v1 cluster external audit object.
func ToProtoClusterExternalAudit(in *externalaudit.ClusterExternalAudit) *externalauditv1.ClusterExternalAudit {
	return &externalauditv1.ClusterExternalAudit{
		Header: headerv1.ToResourceHeaderProto(in.ResourceHeader),
		Spec: &externalauditv1.ClusterExternalAuditSpec{
			ExternalAuditName: in.Spec.ExternalAuditName,
		},
	}
}
