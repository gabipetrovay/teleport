/*
Copyright 2021 Gravitational, Inc.

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

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gravitational/trace/trail"

	"github.com/gravitational/teleport/api/client/proto"
)

// mockServer mocks an Auth Server.
type mockServer struct {
	addr string
	grpc *grpc.Server
}

func newMockServer(t *testing.T, addr string, service proto.AuthServiceServer) *mockServer {
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLSConfig(t))),
		grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			resp, err := handler(ctx, req)
			return resp, trail.ToGRPC(err)
		}),
	)

	m := &mockServer{
		addr: addr,
		grpc: server,
	}
	proto.RegisterAuthServiceServer(m.grpc, service)
	return m
}

func (m *mockServer) Stop() {
	m.grpc.Stop()
}

func (m *mockServer) Addr() string {
	return m.addr
}

type ConfigOpt func(*Config)

func WithConfig(cfg Config) ConfigOpt {
	return func(config *Config) {
		*config = cfg
	}
}

func (m *mockServer) NewClient(ctx context.Context, t *testing.T, opts ...ConfigOpt) (*Client, error) {
	cfg := Config{
		DialTimeout: time.Second,
		Addrs:       []string{m.addr},
		Credentials: []Credentials{
			LoadTLS(clientTLSConfig(t)),
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return New(ctx, cfg)
}

// startMockServer starts a new mock server. Parallel tests cannot use the same addr.
func startMockServer(t *testing.T, service proto.AuthServiceServer) *mockServer {
	l, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err)
	return startMockServerWithListener(t, l, service)
}

// startMockServerWithListener starts a new mock server with the provided listener
func startMockServerWithListener(t *testing.T, l net.Listener, service proto.AuthServiceServer) *mockServer {
	srv := newMockServer(t, l.Addr().String(), service)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.grpc.Serve(l)
	}()

	t.Cleanup(func() {
		srv.grpc.Stop()
		require.NoError(t, <-errCh)
	})

	return srv
}

func serverTLSConfig(t *testing.T) *tls.Config {
	serverTLSCert, err := tls.X509KeyPair([]byte(serverCertPEM), []byte(serverKeyPEM))
	require.NoError(t, err)

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM([]byte(caCertPEM))
	require.True(t, ok, "invalid TLS CA cert PEM")

	return &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
		RootCAs:      pool,
	}
}

func clientTLSConfig(t *testing.T) *tls.Config {
	clientTLSCert, err := tls.X509KeyPair([]byte(clientCertPEM), []byte(clientKeyPEM))
	require.NoError(t, err)

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM([]byte(caCertPEM))
	require.True(t, ok, "invalid TLS CA cert PEM")

	return &tls.Config{
		Certificates: []tls.Certificate{clientTLSCert},
		RootCAs:      pool,
	}
}

// test keys below were generated with openssl:
//
// openssl genpkey -algorithm ed25519 -out ca.key
// openssl req -new -x509 -sha256 -key ca.key -out ca.crt
//
// openssl genpkey -algorithm ed25519 -out server.key
// openssl req -new -sha256 -key server.key -out server.csr -subj "/CN=localhost"
// openssl x509 -req -in server.csr -extfile <(printf "subjectAltName=DNS:teleport.cluster.local") -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 1000 -sha256
//
// openssl genpkey -algorithm ed25519 -out client.key
// openssl req -new -sha256 -key client.key -out client.csr -subj "/CN=localhost"
// openssl x509 -req -in client.csr -extfile <(printf "subjectAltName=DNS:teleport.cluster.local") -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 1000 -sha256
var (
	serverKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIFgNuBtVLihdwBAcXSU8lPY+xPLoAIaysf4fpWZf4Ew/
-----END PRIVATE KEY-----`

	serverCertPEM = `-----BEGIN CERTIFICATE-----
MIIBgDCCATKgAwIBAgIUdE2uc+2CF0ql+iRRlIpalUy6cuQwBQYDK2VwMEUxCzAJ
BgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5l
dCBXaWRnaXRzIFB0eSBMdGQwHhcNMjMwODA4MTk1MDUzWhcNMjYwNTA0MTk1MDUz
WjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwKjAFBgMrZXADIQA1T1GupKBgqtUxq6h4
a+xlTZ0HZuqrZL9PHXpEr5wz36NlMGMwIQYDVR0RBBowGIIWdGVsZXBvcnQuY2x1
c3Rlci5sb2NhbDAdBgNVHQ4EFgQUVp9awOfzip6isncw5OFWpbmDWUIwHwYDVR0j
BBgwFoAUg/kYi7U63zNSIsOEtEsmRZ36AEowBQYDK2VwA0EAoqmaefDQ4CZysVdW
OTccRHlTPtpe5WH9D49TnzQAPAxT89+9QIXV26cWeC93FcLgr4jR88hjqPTaC/Z6
xCS3DA==
-----END CERTIFICATE-----`

	clientKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEICm2fDaf0tzZFzoowr9QghuRtwHdLN+EE1tlqNJnZ5lA
-----END PRIVATE KEY-----`

	clientCertPEM = `-----BEGIN CERTIFICATE-----
MIIBgDCCATKgAwIBAgIUUyPgxjzvhm+VzvY4upF1/54a5OEwBQYDK2VwMEUxCzAJ
BgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5l
dCBXaWRnaXRzIFB0eSBMdGQwHhcNMjMwODA4MTk1MTA5WhcNMjYwNTA0MTk1MTA5
WjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwKjAFBgMrZXADIQAT9lDnT3nrLlZRgnae
rDs7Ol/9H93d2NY41X5+0WUU2KNlMGMwIQYDVR0RBBowGIIWdGVsZXBvcnQuY2x1
c3Rlci5sb2NhbDAdBgNVHQ4EFgQUvVlm6DpdxZ3G+L0SWV7YfWS/GywwHwYDVR0j
BBgwFoAUg/kYi7U63zNSIsOEtEsmRZ36AEowBQYDK2VwA0EAfkv7cKnkl51g+BdQ
ICbjU+nWzYJ2ApkqPMk2JDUQHYeRYqK0VuBdJhgXj2JyWytV1INQBFSU6t6taVKv
aDkOAA==
-----END CERTIFICATE-----`

	caCertPEM = `-----BEGIN CERTIFICATE-----
MIIBnzCCAVGgAwIBAgIUOK2TPdroHp7iq4CJm0mw2Nw9cWIwBQYDK2VwMEUxCzAJ
BgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5l
dCBXaWRnaXRzIFB0eSBMdGQwHhcNMjMwODA4MTgzODAxWhcNMjMwOTA3MTgzODAx
WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwY
SW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMCowBQYDK2VwAyEADBpcnFF/qhMDpwC7
ZgsWc1dYDryQY4RwWrhFWcj0NS6jUzBRMB0GA1UdDgQWBBSD+RiLtTrfM1Iiw4S0
SyZFnfoASjAfBgNVHSMEGDAWgBSD+RiLtTrfM1Iiw4S0SyZFnfoASjAPBgNVHRMB
Af8EBTADAQH/MAUGAytlcANBACAxb2GkpukN/drkJvdmpRK1WnVG4YLV5x/o1sgx
c5u3DzCl0P4TGwlEGFwfaL2BFx3NPDRbQ6Cuv+PHuv+gigk=
-----END CERTIFICATE-----`
)
