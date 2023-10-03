// Copyright 2023 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	"github.com/gravitational/teleport/api/types/header"
)

func TestRoundtrip(t *testing.T) {
	t.Run("external cloud audit", func(t *testing.T) {
		externalAudit := newExternalCloudAudit(t, "audit-1")

		converted, err := FromProto(ToProto(externalAudit))
		require.NoError(t, err)

		require.Empty(t, cmp.Diff(externalAudit, converted))
	})

	t.Run("cluster external cloud audit", func(t *testing.T) {
		clusterExternalAudit, err := externalcloudaudit.NewClusterExternalCloudAudit(header.Metadata{}, externalcloudaudit.ClusterExternalCloudAuditSpec{
			ExternalCloudAuditName: "audit-1",
		})
		require.NoError(t, err)

		converted, err := FromProtoClusterExternalAudit(ToProtoClusterExternalAudit(clusterExternalAudit))
		require.NoError(t, err)

		require.Empty(t, cmp.Diff(clusterExternalAudit, converted))
	})
}

func newExternalCloudAudit(t *testing.T, name string) *externalcloudaudit.ExternalCloudAudit {
	t.Helper()

	out, err := externalcloudaudit.NewExternalCloudAudit(
		header.Metadata{
			Name: name,
		},
		externalcloudaudit.ExternalCloudAuditSpec{
			IntegrationName:        "integration1",
			SessionsRecordingsURI:  "s3://mybucket/myprefix",
			AthenaWorkspace:        "athena_workspace",
			GlueDatabase:           "teleport_db",
			GlueTable:              "teleport_table",
			AuditEventsLongTermURI: "s3://mybucket/myprefix",
			AthenaResultsURI:       "s3://mybucket/myprefix",
		},
	)
	require.NoError(t, err)
	return out
}
