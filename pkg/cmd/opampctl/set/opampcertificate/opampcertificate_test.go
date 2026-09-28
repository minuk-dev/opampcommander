package opampcertificate //nolint:testpackage // exercises the unexported certificate validation boundary

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

func TestValidateCertificateForAgent(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "agent-uid"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	//exhaustruct:ignore
	certificate := &v1.Certificate{Spec: v1.CertificateSpec{
		Cert:       string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}}

	require.NoError(t, validateCertificateForAgent(certificate, "agent-uid"))
	require.ErrorIs(t, validateCertificateForAgent(certificate, "other-agent"), errCertificateCNMismatch)

	certificate.Spec.PrivateKey = "invalid"

	assert.ErrorContains(t, validateCertificateForAgent(certificate, "agent-uid"), "invalid keypair")
}
