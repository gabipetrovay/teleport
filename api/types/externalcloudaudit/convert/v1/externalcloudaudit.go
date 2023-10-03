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

	externalcloudauditv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/externalcloudaudit/v1"
	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	headerv1 "github.com/gravitational/teleport/api/types/header/convert/v1"
)

// FromProto converts a v1 external cloud audit into an internal external cloud audit object.
func FromProto(in *externalcloudauditv1.ExternalCloudAudit) (*externalcloudaudit.ExternalCloudAudit, error) {
	if in == nil {
		return nil, trace.BadParameter("external cloud audit message is nil")
	}

	if in.Spec == nil {
		return nil, trace.BadParameter("spec is missing")
	}
	externalCloudAudit, err := externalcloudaudit.NewExternalCloudAudit(headerv1.FromMetadataProto(in.Header.Metadata), externalcloudaudit.ExternalCloudAuditSpec{
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
	return externalCloudAudit, nil
}

// ToProto converts an internal external cloud audit into a v1 external cloud audit object.
func ToProto(in *externalcloudaudit.ExternalCloudAudit) *externalcloudauditv1.ExternalCloudAudit {
	return &externalcloudauditv1.ExternalCloudAudit{
		Header: headerv1.ToResourceHeaderProto(in.ResourceHeader),
		Spec: &externalcloudauditv1.ExternalCloudAuditSpec{
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

// FromProtoClusterExternalAudit converts a v1 cluster external cloud audit into an
// internal cluster external cloud audit object.
func FromProtoClusterExternalAudit(in *externalcloudauditv1.ClusterExternalCloudAudit) (*externalcloudaudit.ClusterExternalCloudAudit, error) {
	if in == nil {
		return nil, trace.BadParameter("cluster external cloud audit message is nil")
	}

	if in.Spec == nil {
		return nil, trace.BadParameter("cluster external cloud audit spec is nil")
	}
	externalCloudAudit, err := externalcloudaudit.NewClusterExternalCloudAudit(headerv1.FromMetadataProto(in.Header.Metadata), externalcloudaudit.ClusterExternalCloudAuditSpec{
		ExternalCloudAuditName: in.Spec.ExternalCloudAuditName,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return externalCloudAudit, nil
}

// ToProtoClusterExternalAudit converts an internal cluster external cloud audit into
// a v1 cluster ClusterExternalCloudAudit object.
func ToProtoClusterExternalAudit(in *externalcloudaudit.ClusterExternalCloudAudit) *externalcloudauditv1.ClusterExternalCloudAudit {
	return &externalcloudauditv1.ClusterExternalCloudAudit{
		Header: headerv1.ToResourceHeaderProto(in.ResourceHeader),
		Spec: &externalcloudauditv1.ClusterExternalCloudAuditSpec{
			ExternalCloudAuditName: in.Spec.ExternalCloudAuditName,
		},
	}
}
