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

package proxy

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testingkubemock "github.com/gravitational/teleport/lib/kube/proxy/testing/kube_server"
)

func TestErrorRewriter(t *testing.T) {
	t.Parallel()
	const (
		gkeAutopilotCluster = "gke-autopilot"
		otherCluster        = "any-cluster"
		username            = "user"
	)
	// kubeMock is a Kubernetes API mock for the session tests.
	// Once a new session is created, this mock will write to
	// stdout and stdin (if available) the pod name, followed
	// by copying the contents of stdin into both streams.
	gkeKubeMock, err := testingkubemock.NewKubeAPIMock(
		testingkubemock.WithGetPodError(
			metav1.Status{
				Status: metav1.StatusFailure,
				Message: "groups \"system:masters\" is forbidden: User \"<user>\" cannot " +
					"impersonate resource \"groups\" in API group \"\" at the cluster scope: GKE " +
					"Warden authz [denied by user-impersonation-limitation]: impersonating system " +
					"identities are not allowed",
				Reason: metav1.StatusReasonForbidden,
				Code:   http.StatusForbidden,
			},
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { gkeKubeMock.Close() })

	otherKubeMock, err := testingkubemock.NewKubeAPIMock(
		testingkubemock.WithGetPodError(
			metav1.Status{
				Status:  metav1.StatusFailure,
				Message: "request denied",
				Reason:  metav1.StatusReasonForbidden,
				Code:    http.StatusForbidden,
			},
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { otherKubeMock.Close() })

	// creates a Kubernetes service with a configured cluster pointing to mock api server
	testCtx := SetupTestContext(
		context.Background(),
		t,
		TestConfig{
			Clusters: []KubeClusterConfig{
				{Name: gkeAutopilotCluster, APIEndpoint: gkeKubeMock.URL},
				{Name: otherCluster, APIEndpoint: otherKubeMock.URL},
			},
		},
	)
	// close tests
	t.Cleanup(func() { require.NoError(t, testCtx.Close()) })

	// create a user with full access to kubernetes Pods.
	// (kubernetes_user and kubernetes_groups specified)
	user, _ := testCtx.CreateUserAndRole(
		testCtx.Context,
		t,
		username,
		RoleSpec{
			Name:       username,
			KubeUsers:  roleKubeUsers,
			KubeGroups: []string{"system:masters"},
		},
	)

	type args struct {
		kubeCluster string
	}
	type want struct {
		getTestPodResult error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "rewrite gke autopilot error",
			args: args{
				kubeCluster: gkeAutopilotCluster,
			},
			want: want{
				getTestPodResult: &errors.StatusError{
					ErrStatus: metav1.Status{
						Status: metav1.StatusFailure,
						Message: "GKE Autopilot denied the request because it impersonates the " +
							"\"system:masters\" group.\nYour Teleport Roles [user:user] have " +
							"given access to the \"system:masters\" group for the cluster " +
							"\"gke-autopilot\".\nFor additional information and resolution, " +
							"please visit https://goteleport.com/docs/kubernetes-access/troubleshooting/#unable-to-connect-to-gke-autopilot-clusters\n",
						Reason: metav1.StatusReasonForbidden,
						Code:   http.StatusForbidden,
					},
				},
			},
		},
		{
			name: "don't rewrite other errors",
			args: args{
				kubeCluster: otherCluster,
			},
			want: want{
				getTestPodResult: &errors.StatusError{
					ErrStatus: metav1.Status{
						Status:  metav1.StatusFailure,
						Message: "request denied",
						Reason:  metav1.StatusReasonForbidden,
						Code:    http.StatusForbidden,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// generate a kube client with user certs for auth
			client, _ := testCtx.GenTestKubeClientTLSCert(
				t,
				user.GetName(),
				tt.args.kubeCluster,
			)

			_, err := client.CoreV1().Pods(metav1.NamespaceDefault).Get(
				testCtx.Context,
				"test-pod",
				metav1.GetOptions{},
			)
			require.Error(t, err)
			require.Equal(t, tt.want.getTestPodResult, err)
		})
	}
}
