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

	"github.com/gravitational/teleport/api/types/externalaudit"
	"github.com/gravitational/teleport/api/types/header"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/backend/memory"
)

func TestExternalAuditCRUD(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	mem, err := memory.New(memory.Config{
		Context: ctx,
		Clock:   clock,
	})
	require.NoError(t, err)

	service, err := NewExternalAuditService(backend.NewSanitizer(mem))
	require.NoError(t, err)

	audit1Name := "audit1"
	externalAudit1 := newExternalAudit(t, audit1Name, "s3://bucket1/ses-rec")
	audit2Name := "audit2"
	externalAudit2 := newExternalAudit(t, audit2Name, "s3://bucket2/ses-rec")

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
		_, err := service.CreateExternalAudit(ctx, externalAudit1)
		require.NoError(t, err)
		_, err = service.CreateExternalAudit(ctx, externalAudit2)
		require.NoError(t, err)

		// Then
		out, err := service.GetExternalAudit(ctx, audit1Name)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit1, out, cmpOpts...))
		out, err = service.GetExternalAudit(ctx, audit2Name)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit2, out, cmpOpts...))
		listOfAudits, err := service.GetExternalAudits(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff([]*externalaudit.ExternalAudit{externalAudit1, externalAudit2}, listOfAudits, cmpOpts...))
	})

	t.Run("get cluster external audit should be empty", func(t *testing.T) {
		// Given audit1 and audit2 as external_audit resource
		// When GetClusterExternalAudit is executed
		// Then NotFound error is returned.

		// When
		out, err := service.GetClusterExternalAudit(ctx)
		// Then
		require.True(t, trace.IsNotFound(err), "expected not found error, got %v", err)
		require.Nil(t, out)
	})
	t.Run("set audit1 to cluster external audit", func(t *testing.T) {
		// Given audit1 and audit2 as external_audit resource
		// When SetClusterExternalAudit is executed with audit1
		// Then GetClusterExternalAudit returns audit1.

		// When
		err := service.SetClusterExternalAudit(ctx, &externalaudit.ClusterExternalAudit{
			Spec: externalaudit.ClusterExternalAuditSpec{
				ExternalAuditName: audit1Name,
			},
		})
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalAudit(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit1, out, cmpOpts...))
	})
	t.Run("set audit2 to cluster external audit", func(t *testing.T) {
		// Given audit1 as cluster_external_audit
		// When SetClusterExternalAudit is executed with audit2
		// Then GetClusterExternalAudit returns audit2.

		// When
		err := service.SetClusterExternalAudit(ctx, &externalaudit.ClusterExternalAudit{
			Spec: externalaudit.ClusterExternalAuditSpec{
				ExternalAuditName: audit2Name,
			},
		})
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalAudit(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(externalAudit2, out, cmpOpts...))
	})
	t.Run("delete cluster external audit", func(t *testing.T) {
		// Given audit1 as cluster_external_audit
		// When DeleteClusterExternalAudit is executed
		// Then NotFound error is returned.

		// When
		err := service.DeleteClusterExternalAudit(ctx)
		require.NoError(t, err)
		// Then
		out, err := service.GetClusterExternalAudit(ctx)
		require.True(t, trace.IsNotFound(err), "expected not found error, got %v", err)
		require.Nil(t, out)
	})
	t.Run("delete audit2", func(t *testing.T) {
		// Given audit2 as external_audit
		// When DeleteExternalAudit is executed
		// Then GetExternalAudits reurns audit1.

		// When
		err := service.DeleteExternalAudit(ctx, audit2Name)
		require.NoError(t, err)
		// Then
		out, err := service.GetExternalAudits(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff([]*externalaudit.ExternalAudit{externalAudit1}, out, cmpOpts...))
	})
}

func newExternalAudit(t *testing.T, name, sessionsRecordingsURI string) *externalaudit.ExternalAudit {
	t.Helper()
	out, err := externalaudit.NewExternalAudit(header.Metadata{Name: name}, externalaudit.ExternalAuditSpec{
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
