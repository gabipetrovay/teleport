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

package reverseproxy

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIPv6Fix(t *testing.T) {
	testCases := []struct {
		desc     string
		clientIP string
		expected string
	}{
		{
			desc:     "empty",
			clientIP: "",
			expected: "",
		},
		{
			desc:     "ipv4 localhost",
			clientIP: "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			desc:     "ipv4",
			clientIP: "10.13.14.15",
			expected: "10.13.14.15",
		},
		{
			desc:     "ipv6 zone",
			clientIP: `fe80::d806:a55d:eb1b:49cc%vEthernet (vmxnet3 Ethernet Adapter - Virtual Switch)`,
			expected: "fe80::d806:a55d:eb1b:49cc",
		},
		{
			desc:     "ipv6 medium",
			clientIP: `fe80::1`,
			expected: "fe80::1",
		},
		{
			desc:     "ipv6 small",
			clientIP: `2000::`,
			expected: "2000::",
		},
		{
			desc:     "ipv6",
			clientIP: `2001:3452:4952:2837::`,
			expected: "2001:3452:4952:2837::",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			actual := ipv6fix(test.clientIP)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestRewriter(t *testing.T) {
	const hostname = "teleport-dev"
	testCases := []struct {
		desc       string
		reqHeaders http.Header
		tlsReq     bool
		hostReq    string
		remoteAddr string
		expected   http.Header
	}{
		{
			desc: "set x-real-ip",
			reqHeaders: http.Header{
				XForwardedFor: []string{"1.2.3.4"},
			},
			tlsReq:     true,
			hostReq:    "teleport.dev:3543",
			remoteAddr: "1.2.3.4:1234",
			expected: http.Header{
				XForwardedHost:   []string{"teleport.dev:3543"},
				XForwardedPort:   []string{"3543"},
				XForwardedProto:  []string{"https"},
				XForwardedServer: []string{hostname},
				XRealIP:          []string{"1.2.3.4"},
			},
		},
		{
			desc: "trust x-real-ip",
			reqHeaders: http.Header{
				XRealIP:       []string{"5.6.7.8"},
				XForwardedFor: []string{"1.2.3.4"},
			},
			tlsReq:     false,
			hostReq:    "teleport.dev:3543",
			remoteAddr: "1.2.3.4:1234",
			expected: http.Header{
				XForwardedHost:   []string{"teleport.dev:3543"},
				XForwardedPort:   []string{"3543"},
				XForwardedProto:  []string{"http"},
				XForwardedServer: []string{hostname},
				XRealIP:          []string{"5.6.7.8"},
			},
		},
		{
			desc: "trust x-real-ip and guess port from schema",
			reqHeaders: http.Header{
				XRealIP: []string{"5.6.7.8"},
			},
			tlsReq:     false,
			hostReq:    "teleport.dev",
			remoteAddr: "1.2.3.4:1234",
			expected: http.Header{
				XForwardedHost:   []string{"teleport.dev"},
				XForwardedPort:   []string{"80"},
				XForwardedProto:  []string{"http"},
				XForwardedServer: []string{hostname},
				XRealIP:          []string{"5.6.7.8"},
			},
		},
	}
	rewriter := NewHeaderRewriter()
	// set hostname to make sure it's the same in all tests.
	rewriter.Hostname = hostname
	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := &http.Request{
				Host:       test.hostReq,
				Header:     test.reqHeaders,
				RemoteAddr: test.remoteAddr,
			}
			if test.tlsReq {
				req.TLS = &tls.ConnectionState{}
			}
			rewriter.Rewrite(req)
			require.Equal(t, test.expected, req.Header)
		})
	}
}
