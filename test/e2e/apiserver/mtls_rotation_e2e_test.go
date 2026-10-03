//go:build e2e

package apiserver_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// This test uses the upstream opamp-go WebSocket client and needs no Docker.
func TestE2E_OpAMPMTLSRotate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	base := testutil.NewBase(t)
	settings, roots, ca, caKey := mtlsFixture(t)
	server := base.StartStandaloneAPIServerWithOpAMPTLS(settings)
	require.Eventually(t, server.IsReady, 30*time.Second, 100*time.Millisecond)
	apiClient := client.New(server.Endpoint, client.WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}),
		client.WithBasicAuth(server.AdminUsername(), server.AdminPassword()))
	ctx := t.Context()
	uid := uuid.New()
	// The test acts as the external CA; APIServer only receives issued client credentials.
	register := func(name string) *v1.Certificate {
		cert, err := apiClient.CertificateService.CreateCertificate(ctx, "default", &v1.Certificate{
			Metadata: v1.CertificateMetadata{Name: name, Namespace: "default"},
			Spec:     mtlsClientCertificate(t, ca, caKey, uid),
		})
		require.NoError(t, err)
		return cert
	}
	oldCert := register("old-client")
	oldAgent := base.StartReferenceAgent(server.Port, testutil.WithReferenceAgentUID(uid),
		testutil.WithReferenceAgentTLSConfig(agentTLS(t, roots, oldCert)),
		testutil.WithReferenceAgentOpAMPSettings(),
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": "mtls-rotation"}))
	testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool { return a.Status.Connected },
		30*time.Second, 100*time.Millisecond, "old certificate should connect")
	newCert := register("new-client")
	newName := newCert.Metadata.Name
	endpoint := "wss://localhost:" + strconv.Itoa(server.Port) + "/api/v1/opamp"
	_, err := apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
		Metadata: v1.Metadata{Name: "mtls-rotation"},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "mtls-rotation"}},
			AgentConfig: &v1.AgentConfig{ConnectionSettings: &v1.ConnectionSettings{
				OpAMP: v1.OpAMPConnectionSettings{DestinationEndpoint: endpoint, CertificateName: &newName},
			}},
		},
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return oldAgent.OfferedClientCertificate() != nil },
		30*time.Second, 100*time.Millisecond, "old agent should receive the new certificate")
	require.Equal(t, []byte(newCert.Spec.Cert), oldAgent.OfferedClientCertificate().GetCert())
	oldAgent.ReconnectWithOfferedCertificate(t, roots)
	testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return a.Status.Connected && a.Status.ConnectionSettings.SyncStatus == "applied"
	}, 30*time.Second, 100*time.Millisecond, "new certificate should become active")
	rejectedAgent := base.StartReferenceAgent(server.Port, testutil.WithReferenceAgentUID(uid),
		testutil.WithReferenceAgentTLSConfig(agentTLS(t, roots, oldCert)),
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": "mtls-rotation"}))
	require.Eventually(t, func() bool { return len(rejectedAgent.ServerErrors()) > 0 },
		10*time.Second, 100*time.Millisecond, "replaced certificate should be rejected on reconnect")
	rejectedAgent.Stop()
}

func agentTLS(t *testing.T, roots *x509.CertPool, cert *v1.Certificate) *tls.Config {
	t.Helper()
	pair, err := tls.X509KeyPair([]byte(cert.Spec.Cert), []byte(cert.Spec.PrivateKey))
	require.NoError(t, err)
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{pair}}
}

func mtlsFixture(t *testing.T) (config.OpAMPTLSSettings, *x509.CertPool, *x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	now := time.Now()
	caTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test agent CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serverTemplate := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, ca, &serverKey.PublicKey, caKey)
	require.NoError(t, err)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	require.NoError(t, err)
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, data, 0600))
		return path
	}
	caFile := write("ca.pem", caPEM)
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(caPEM))
	return config.OpAMPTLSSettings{
		CertFile: write("server.crt", serverPEM), KeyFile: write("server.key", pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})),
		CAFile: caFile,
	}, roots, ca, caKey
}

func mtlsClientCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, uid uuid.UUID) v1.CertificateSpec {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: uid.String()},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: ca.NotAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return v1.CertificateSpec{
		Cert:       string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})),
		CaCert:     "",
	}
}
