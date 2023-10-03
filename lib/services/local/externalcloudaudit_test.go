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

package local

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gravitational/trace"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	"github.com/gravitational/teleport/api/types/header"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/backend/memory"
)

func TestExternalCloudAuditCRUD(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	mem, err := memory.New(memory.Config{
		Context: ctx,
		Clock:   clock,
	})
	require.NoError(t, err)

	service, err := NewExternalCloudAuditService(backend.NewSanitizer(mem))
	require.NoError(t, err)

	audit1Name := "audit1"
	externalAudit1 := newExternalCloudAudit(t, audit1Name, "s3://bucket1/ses-rec")
	audit2Name := "audit2"
	externalAudit2 := newExternalCloudAudit(t, audit2Name, "s3://bucket2/ses-rec")

	cmpOpts := []cmp.Option{
		cmpopts.IgnoreFields(header.Metadata{}, "ID", "Revision"),
	}

	t.Run("create externalAudits", func(t *testing.T) {
		// Given no resources
		// When externalAudit1 and externalAudit2 are created
		// Then externalAudit1 and externalAudit2 are returned on
		// GetExternalAudit based on name,
		// and GetExternalAudits returns both.

		// When
		_, err := service.CreateExternalCloudAudit(ctx, externalAudit1)
		require.NoError(t, err)
		_, err = service.CreateExternalCloudAudit(ctx, externalAudit2)
		require.NoError(t, err)

		// Then
		out, err := service.GetExternalCloudAudit(ctx, audit1Name)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit1, out, cmpOpts...))
		out, err = service.GetExternalCloudAudit(ctx, audit2Name)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit2, out, cmpOpts...))
		listOfAudits, err := service.GetExternalCloudAudits(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff([]*externalcloudaudit.ExternalCloudAudit{externalAudit1, externalAudit2}, listOfAudits, cmpOpts...))
	})

	t.Run("get cluster external cloud audit should be empty", func(t *testing.T) {
		// Given audit1 and audit2 as external_audit resource
		// When GetClusterExternalAudit is executed
		// Then NotFound error is returned.

		// When
		out, err := service.GetClusterExternalCloudAudit(ctx)
		// Then
		require.True(t, trace.IsNotFound(err), "expected not found error, got %v", err)
		require.Nil(t, out)
	})
	t.Run("set audit1 to cluster external cloud audit", func(t *testing.T) {
		// Given audit1 and audit2 as external_audit resource
		// When SetClusterExternalAudit is executed with audit1
		// Then GetClusterExternalAudit returns audit1.

		// When
		err := service.EnableClusterExternalCloudAudit(ctx, &externalcloudaudit.ClusterExternalCloudAudit{
			Spec: externalcloudaudit.ClusterExternalCloudAuditSpec{
				ExternalCloudAuditName: audit1Name,
			},
		})
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalCloudAudit(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit1, out, cmpOpts...))
	})
	t.Run("set audit2 to cluster external cloud audit", func(t *testing.T) {
		// Given audit1 as cluster_external_audit
		// When SetClusterExternalAudit is executed with audit2
		// Then GetClusterExternalAudit returns audit2.

		// When
		err := service.EnableClusterExternalCloudAudit(ctx, &externalcloudaudit.ClusterExternalCloudAudit{
			Spec: externalcloudaudit.ClusterExternalCloudAuditSpec{
				ExternalCloudAuditName: audit2Name,
			},
		})
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalCloudAudit(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit2, out, cmpOpts...))
	})
	t.Run("delete cluster external cloud audit", func(t *testing.T) {
		// Given audit1 as cluster_external_audit
		// When DeleteClusterExternalAudit is executed
		// Then NotFound error is returned.

		// When
		err := service.DisableClusterExternalCloudAudit(ctx)
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalCloudAudit(ctx)
		require.True(t, trace.IsNotFound(err), "expected not found error, got %v", err)
		require.Nil(t, out)
	})
	t.Run("delete audit2", func(t *testing.T) {
		// Given audit2 as external_audit
		// When DeleteExternalAudit is executed
		// Then GetExternalAudits reurns audit1.

		// When
		err := service.DeleteExternalCloudAudit(ctx, audit2Name)
		require.NoError(t, err)
		// Then
		out, err := service.GetExternalCloudAudits(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff([]*externalcloudaudit.ExternalCloudAudit{externalAudit1}, out, cmpOpts...))
	})
}

func newExternalCloudAudit(t *testing.T, name, sessionsRecordingsURI string) *externalcloudaudit.ExternalCloudAudit {
	t.Helper()
	out, err := externalcloudaudit.NewExternalCloudAudit(header.Metadata{Name: name}, externalcloudaudit.ExternalCloudAuditSpec{
		IntegrationName:        "aws-integration-1",
		SessionsRecordingsURI:  sessionsRecordingsURI,
		AthenaWorkspace:        "primary",
		GlueDatabase:           "teleport_db",
		GlueTable:              "teleport_table",
		AuditEventsLongTermURI: "s3://bucket/events",
		AthenaResultsURI:       "s3://bucket/results",
	})
	require.NoError(t, err)
	return out
}
