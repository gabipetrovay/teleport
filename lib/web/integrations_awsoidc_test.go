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

package web

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/gravitational/trace"
	"github.com/stretchr/testify/require"
)

func TestBuildDeployServiceConfigureIAMScript(t *testing.T) {
	isBadParamErrFn := func(tt require.TestingT, err error, i ...any) {
		require.True(tt, trace.IsBadParameter(err), "expected bad parameter, got %v", err)
	}

	ctx := context.Background()
	env := newWebPack(t, 1)

	// Unauthenticated client for script downloading.
	publicClt := env.proxies[0].newClient(t)
	pathVars := []string{
		"webapi",
		"scripts",
		"integrations",
		"configure",
		"deployservice-iam.sh",
	}
	endpoint := publicClt.Endpoint(pathVars...)

	tests := []struct {
		name                 string
		reqRelativeURL       string
		reqQuery             url.Values
		errCheck             require.ErrorAssertionFunc
		expectedTeleportArgs string
	}{
		{
			name: "valid",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"myRole"},
				"taskRole":        []string{"taskRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure deployservice-iam " +
				`--cluster=localhost ` +
				`--name=myintegration ` +
				`--aws-region=us-east-1 ` +
				`--role=myRole ` +
				`--task-role=taskRole`,
		},
		{
			name: "valid with symbols in role",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"Test+1=2,3.4@5-6_7"},
				"taskRole":        []string{"taskRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure deployservice-iam " +
				`--cluster=localhost ` +
				`--name=myintegration ` +
				`--aws-region=us-east-1 ` +
				`--role=Test+1=2,3.4@5-6_7 ` +
				`--task-role=taskRole`,
		},
		{
			name: "missing aws-region",
			reqQuery: url.Values{
				"role":            []string{"myRole"},
				"taskRole":        []string{"taskRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing role",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"taskRole":        []string{"taskRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing task role",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"role"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing integration name",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"role"},
				"taskRole":  []string{"taskRole"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "trying to inject escape sequence into query params",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"role"},
				"taskRole":        []string{"taskRole"},
				"integrationName": []string{"'; rm -rf /tmp/dir; echo '"},
			},
			errCheck: isBadParamErrFn,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := publicClt.Get(ctx, endpoint, tc.reqQuery)
			tc.errCheck(t, err)
			if err != nil {
				return
			}

			require.Contains(t, string(resp.Bytes()),
				fmt.Sprintf("teleportArgs='%s'\n", tc.expectedTeleportArgs),
			)
		})
	}
}

func TestBuildEICEConfigureIAMScript(t *testing.T) {
	isBadParamErrFn := func(tt require.TestingT, err error, i ...any) {
		require.True(tt, trace.IsBadParameter(err), "expected bad parameter, got %v", err)
	}

	ctx := context.Background()
	env := newWebPack(t, 1)

	// Unauthenticated client for script downloading.
	publicClt := env.proxies[0].newClient(t)
	pathVars := []string{
		"webapi",
		"scripts",
		"integrations",
		"configure",
		"eice-iam.sh",
	}
	endpoint := publicClt.Endpoint(pathVars...)

	tests := []struct {
		name                 string
		reqRelativeURL       string
		reqQuery             url.Values
		errCheck             require.ErrorAssertionFunc
		expectedTeleportArgs string
	}{
		{
			name: "valid",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"myRole"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure eice-iam " +
				"--aws-region=us-east-1 " +
				"--role=myRole",
		},
		{
			name: "valid with symbols in role",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"Test+1=2,3.4@5-6_7"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure eice-iam " +
				"--aws-region=us-east-1 " +
				"--role=Test+1=2,3.4@5-6_7",
		},
		{
			name: "missing aws-region",
			reqQuery: url.Values{
				"role": []string{"myRole"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing role",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "trying to inject escape sequence into query params",
			reqQuery: url.Values{
				"awsRegion": []string{"'; rm -rf /tmp/dir; echo '"},
				"role":      []string{"role"},
			},
			errCheck: isBadParamErrFn,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := publicClt.Get(ctx, endpoint, tc.reqQuery)
			tc.errCheck(t, err)
			if err != nil {
				return
			}

			require.Contains(t, string(resp.Bytes()),
				fmt.Sprintf("teleportArgs='%s'\n", tc.expectedTeleportArgs),
			)
		})
	}
}

func TestBuildAWSOIDCIdPConfigureScript(t *testing.T) {
	isBadParamErrFn := func(tt require.TestingT, err error, i ...any) {
		require.True(tt, trace.IsBadParameter(err), "expected bad parameter, got %v", err)
	}

	ctx := context.Background()
	env := newWebPack(t, 1)

	// Unauthenticated client for script downloading.
	publicClt := env.proxies[0].newClient(t)
	pathVars := []string{
		"webapi",
		"scripts",
		"integrations",
		"configure",
		"awsoidc-idp.sh",
	}
	endpoint := publicClt.Endpoint(pathVars...)

	proxyPublicURL := env.proxies[0].webURL

	tests := []struct {
		name                 string
		reqRelativeURL       string
		reqQuery             url.Values
		errCheck             require.ErrorAssertionFunc
		expectedTeleportArgs string
	}{
		{
			name: "valid",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"myRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure awsoidc-idp " +
				"--cluster=localhost " +
				"--name=myintegration " +
				"--aws-region=us-east-1 " +
				"--role=myRole " +
				"--proxy-public-url=" + proxyPublicURL.String(),
		},
		{
			name: "valid with symbols in role",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"Test+1=2,3.4@5-6_7"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure awsoidc-idp " +
				"--cluster=localhost " +
				"--name=myintegration " +
				"--aws-region=us-east-1 " +
				"--role=Test+1=2,3.4@5-6_7 " +
				"--proxy-public-url=" + proxyPublicURL.String(),
		},
		{
			name: "missing aws-region",
			reqQuery: url.Values{
				"role":            []string{"myRole"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing role",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"integrationName": []string{"myintegration"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing integration name",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"role"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "trying to inject escape sequence into query params",
			reqQuery: url.Values{
				"awsRegion":       []string{"us-east-1"},
				"role":            []string{"role"},
				"integrationName": []string{"'; rm -rf /tmp/dir; echo '"},
			},
			errCheck: isBadParamErrFn,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := publicClt.Get(ctx, endpoint, tc.reqQuery)
			tc.errCheck(t, err)
			if err != nil {
				return
			}

			require.Contains(t, string(resp.Bytes()),
				fmt.Sprintf("teleportArgs='%s'\n", tc.expectedTeleportArgs),
			)
		})
	}
}

func TestBuildListDatabasesConfigureIAMScript(t *testing.T) {
	isBadParamErrFn := func(tt require.TestingT, err error, i ...any) {
		require.True(tt, trace.IsBadParameter(err), "expected bad parameter, got %v", err)
	}

	ctx := context.Background()
	env := newWebPack(t, 1)

	// Unauthenticated client for script downloading.
	publicClt := env.proxies[0].newClient(t)
	pathVars := []string{
		"webapi",
		"scripts",
		"integrations",
		"configure",
		"listdatabases-iam.sh",
	}
	endpoint := publicClt.Endpoint(pathVars...)

	tests := []struct {
		name                 string
		reqRelativeURL       string
		reqQuery             url.Values
		errCheck             require.ErrorAssertionFunc
		expectedTeleportArgs string
	}{
		{
			name: "valid",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"myRole"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure listdatabases-iam " +
				"--aws-region=us-east-1 " +
				"--role=myRole",
		},
		{
			name: "valid with symbols in role",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
				"role":      []string{"Test+1=2,3.4@5-6_7"},
			},
			errCheck: require.NoError,
			expectedTeleportArgs: "integration configure listdatabases-iam " +
				"--aws-region=us-east-1 " +
				"--role=Test+1=2,3.4@5-6_7",
		},
		{
			name: "missing aws-region",
			reqQuery: url.Values{
				"role": []string{"myRole"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "missing role",
			reqQuery: url.Values{
				"awsRegion": []string{"us-east-1"},
			},
			errCheck: isBadParamErrFn,
		},
		{
			name: "trying to inject escape sequence into query params",
			reqQuery: url.Values{
				"awsRegion": []string{"'; rm -rf /tmp/dir; echo '"},
				"role":      []string{"role"},
			},
			errCheck: isBadParamErrFn,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := publicClt.Get(ctx, endpoint, tc.reqQuery)
			tc.errCheck(t, err)
			if err != nil {
				return
			}

			require.Contains(t, string(resp.Bytes()),
				fmt.Sprintf("teleportArgs='%s'\n", tc.expectedTeleportArgs),
			)
		})
	}
}
