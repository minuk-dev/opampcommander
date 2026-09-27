package agentgroup //nolint:testpackage // exercises the unexported certificate validation boundary

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

func TestValidateCertificateForAgent(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "agent-uid"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	//exhaustruct:ignore
	certificate := &v1.Certificate{Spec: v1.CertificateSpec{
		Cert:       string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}}

	err = validateCertificateForAgent(certificate, "agent-uid")
	if err != nil {
		t.Fatal(err)
	}

	err = validateCertificateForAgent(certificate, "other-agent")
	if !errors.Is(err, errCertificateCNMismatch) {
		t.Fatalf("expected CN mismatch, got %v", err)
	}

	certificate.Spec.PrivateKey = "invalid"

	err = validateCertificateForAgent(certificate, "agent-uid")
	if err == nil {
		t.Fatal("invalid keypair was accepted")
	}
}
