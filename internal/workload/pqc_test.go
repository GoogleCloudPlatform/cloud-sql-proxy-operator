// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workload_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	corev1 "k8s.io/api/core/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

func generateLocalhostTLS13Cert(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal ECDSA key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("failed to load X509 key pair: %v", err)
	}
	return tlsCert, certPEM
}

func TestOperatorTLSKeyExchange(t *testing.T) {
	if tls.X25519MLKEM768 != tls.CurveID(0x11ec) {
		t.Fatalf("got tls.X25519MLKEM768 = 0x%x, want 0x11ec", uint16(tls.X25519MLKEM768))
	}

	tlsCert, caPEM := generateLocalhostTLS13Cert(t)

	testCases := []struct {
		name         string
		godebug      string
		serverCurves []tls.CurveID
		wantCurve    tls.CurveID
	}{
		{
			name:      "default negotiates X25519MLKEM768",
			wantCurve: tls.X25519MLKEM768,
		},
		{
			name:         "falls back to X25519 against classical-only server",
			serverCurves: []tls.CurveID{tls.X25519, tls.CurveP256},
			wantCurve:    tls.X25519,
		},
		{
			// Explicitly configure X25519MLKEM768 on the server so process-wide
			// GODEBUG=tlsmlkem=0 does not disable PQC on the test server itself,
			// proving the client transport opted out.
			name:         "respects GODEBUG tlsmlkem=0 opt-out",
			godebug:      "tlsmlkem=0",
			serverCurves: []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256},
			wantCurve:    tls.X25519,
		},
		{
			name:         "respects GODEBUG tlsmlkem=1 explicit opt-in",
			godebug:      "tlsmlkem=1",
			serverCurves: []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256},
			wantCurve:    tls.X25519MLKEM768,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.godebug != "" {
				t.Setenv("GODEBUG", tc.godebug)
			}

			var (
				mu          sync.Mutex
				serverState tls.ConnectionState
			)

			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.TLS != nil {
					mu.Lock()
					serverState = *r.TLS
					mu.Unlock()
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}))
			srv.TLS = &tls.Config{
				Certificates:     []tls.Certificate{tlsCert},
				MinVersion:       tls.VersionTLS13,
				CurvePreferences: tc.serverCurves,
			}
			srv.StartTLS()
			defer srv.Close()

			// Exercise the same k8s.io/client-go REST TLS transport builder used by
			// the operator controller when connecting to the Kubernetes API server.
			rt, err := rest.TransportFor(&rest.Config{
				Host: srv.URL,
				TLSClientConfig: rest.TLSClientConfig{
					CAData: caPEM,
				},
			})
			if err != nil {
				t.Fatalf("rest.TransportFor failed: %v", err)
			}

			httpClient := &http.Client{
				Transport: rt,
				Timeout:   5 * time.Second,
			}
			resp, err := httpClient.Get(srv.URL + "/version")
			if err != nil {
				t.Fatalf("HTTPS GET failed: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed reading response body: %v", err)
			}
			if resp.StatusCode != http.StatusOK || string(body) != "ok" {
				t.Fatalf("got HTTP %d (%q), want 200 (\"ok\")", resp.StatusCode, string(body))
			}

			if resp.TLS == nil {
				t.Fatal("client response TLS ConnectionState is nil")
			}
			if resp.TLS.Version != tls.VersionTLS13 {
				t.Errorf("client TLS version = 0x%x, want TLS 1.3 (0x%x)", resp.TLS.Version, tls.VersionTLS13)
			}
			if resp.TLS.CurveID != tc.wantCurve {
				t.Errorf("client CurveID = 0x%x (%v), want 0x%x (%v)",
					uint16(resp.TLS.CurveID), resp.TLS.CurveID, uint16(tc.wantCurve), tc.wantCurve)
			}

			mu.Lock()
			gotServerState := serverState
			mu.Unlock()
			if gotServerState.Version != tls.VersionTLS13 {
				t.Errorf("server TLS version = 0x%x, want TLS 1.3 (0x%x)", gotServerState.Version, tls.VersionTLS13)
			}
			if gotServerState.CurveID != tc.wantCurve {
				t.Errorf("server CurveID = 0x%x (%v), want 0x%x (%v)",
					uint16(gotServerState.CurveID), gotServerState.CurveID, uint16(tc.wantCurve), tc.wantCurve)
			}
		})
	}
}

func TestDefaultProxyImageSupportsPQC(t *testing.T) {
	const wantImageRepo = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:"
	if !strings.HasPrefix(workload.DefaultProxyImage, wantImageRepo) {
		t.Fatalf("DefaultProxyImage = %q, want prefix %q", workload.DefaultProxyImage, wantImageRepo)
	}

	tag := strings.TrimPrefix(workload.DefaultProxyImage, wantImageRepo)
	gotVersion, err := utilversion.ParseSemantic(tag)
	if err != nil {
		t.Fatalf("unable to parse DefaultProxyImage tag %q as semantic version: %v", tag, err)
	}

	// Cloud SQL Auth Proxy v2.15.2 is the first release with `go 1.24` in go.mod
	// where Go enables X25519MLKEM768 (tlsmlkem=1) by default.
	minPQCVersion := utilversion.MustParseSemantic("2.15.2")
	if !gotVersion.AtLeast(minPQCVersion) {
		t.Fatalf("DefaultProxyImage version %s is older than minimum PQC-capable version %s",
			gotVersion, minPQCVersion)
	}

	// Verify default workload configuration uses DefaultProxyImage and does not
	// override GODEBUG (leaving Go's default tlsmlkem=1 enabled).
	u := workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage, false)
	wl := podWorkload()
	apw := simpleAuthProxy("instance1", "project:region:db")
	apw.Spec.Instances[0].PortEnvName = "DB_PORT"

	if err := configureProxies(u, wl, []*cloudsqlapi.AuthProxyWorkload{apw}); err != nil {
		t.Fatalf("configureProxies failed: %v", err)
	}

	proxyContainer, err := findContainer(wl, workload.ContainerName(apw))
	if err != nil {
		t.Fatalf("findContainer failed: %v", err)
	}
	if proxyContainer.Image != workload.DefaultProxyImage {
		t.Errorf("got image %q, want %q", proxyContainer.Image, workload.DefaultProxyImage)
	}
	for _, ev := range proxyContainer.Env {
		if ev.Name == "GODEBUG" {
			t.Errorf("unexpected GODEBUG=%q set on default proxy container", ev.Value)
		}
	}
}

func TestSidecarGODEBUGOptOutViaContainerOverride(t *testing.T) {
	const manifestYAML = `
apiVersion: cloudsql.cloud.google.com/v1
kind: AuthProxyWorkload
metadata:
  name: instance1
  namespace: default
  generation: 1
spec:
  workloadSelector:
    kind: "Deployment"
    name: "busybox"
  instances:
    - connectionString: "project:region:db"
      port: 5000
  authProxyContainer:
    container:
      name: placeholder
      image: gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.26.3
      restartPolicy: Always
      args:
        - "--structured-logs"
        - "project:region:db?port=5000"
      env:
        - name: GODEBUG
          value: "tlsmlkem=0"
`

	testCases := []struct {
		name       string
		useSidecar bool
	}{
		{
			name:       "regular sidecar container (k8s < 1.29)",
			useSidecar: false,
		},
		{
			name:       "native initContainer sidecar (k8s >= 1.29)",
			useSidecar: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var apw cloudsqlapi.AuthProxyWorkload
			if err := yaml.Unmarshal([]byte(manifestYAML), &apw); err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}
			if !tc.useSidecar {
				apw.Spec.AuthProxyContainer.Container.RestartPolicy = nil
			}

			if _, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateCreate(context.Background(), &apw); err != nil {
				t.Fatalf("ValidateCreate failed for GODEBUG opt-out manifest: %v", err)
			}

			u := workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage, tc.useSidecar)
			wl := podWorkload()
			wl.Pod.Spec.Containers[0].Ports = []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

			if err := configureProxies(u, wl, []*cloudsqlapi.AuthProxyWorkload{&apw}); err != nil {
				t.Fatalf("configureProxies failed: %v", err)
			}

			wantName := workload.ContainerName(&apw)
			var proxyContainer *corev1.Container
			if tc.useSidecar {
				for i := range wl.Pod.Spec.InitContainers {
					if wl.Pod.Spec.InitContainers[i].Name == wantName {
						proxyContainer = &wl.Pod.Spec.InitContainers[i]
						break
					}
				}
				if proxyContainer == nil {
					t.Fatalf("proxy container %q not found in InitContainers", wantName)
				}
				if proxyContainer.RestartPolicy == nil || *proxyContainer.RestartPolicy != corev1.ContainerRestartPolicyAlways {
					t.Errorf("got RestartPolicy %v, want Always", proxyContainer.RestartPolicy)
				}
			} else {
				c, err := findContainer(wl, wantName)
				if err != nil {
					t.Fatalf("findContainer failed: %v", err)
				}
				proxyContainer = &c
			}

			var foundGODEBUG bool
			for _, ev := range proxyContainer.Env {
				if ev.Name == "GODEBUG" {
					foundGODEBUG = true
					if ev.Value != "tlsmlkem=0" {
						t.Errorf("got proxy container GODEBUG=%q, want \"tlsmlkem=0\"", ev.Value)
					}
				}
			}
			if !foundGODEBUG {
				t.Errorf("GODEBUG env var was not preserved on proxy container: %+v", proxyContainer.Env)
			}

			// Ensure GODEBUG is not leaked onto the application container and the
			// application container and pod annotations remain intact.
			appContainer, err := findContainer(wl, "busybox")
			if err != nil {
				t.Fatalf("application container busybox missing: %v", err)
			}
			for _, ev := range appContainer.Env {
				if ev.Name == "GODEBUG" {
					t.Errorf("unexpected GODEBUG=%q leaked onto application container", ev.Value)
				}
			}
			wantAnnKey, wantAnnVal := u.PodAnnotation(&apw)
			if gotAnnVal := wl.PodTemplateAnnotations()[wantAnnKey]; gotAnnVal != wantAnnVal {
				t.Errorf("pod annotation %q = %q, want %q", wantAnnKey, gotAnnVal, wantAnnVal)
			}
		})
	}
}
