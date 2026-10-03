package certificate

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// ErrIssuerUnavailable indicates that server-side certificate issuance is disabled.
var ErrIssuerUnavailable = errors.New("OpAMP client certificate issuer is not configured")

var (
	errIncompleteIssuer = errors.New("issuerCertFile and issuerKeyFile require opampTLS.caFile")
	errIssuerNotCA      = errors.New("issuing certificate must be a CA with certificate-signing usage")
	errIssuerKey        = errors.New("issuing CA private key cannot sign certificates")
	errTrustedCAs       = errors.New("OpAMP client CA bundle contains no certificates")
	errIssuerExpired    = errors.New("issuing CA has expired")
)

const certificateSerialBits = 128

// Issuer signs agent client certificates with the configured CA.
type Issuer struct {
	ca    *x509.Certificate
	key   crypto.Signer
	chain []byte
}

// NewIssuer validates that the issuing CA chains to an OpAMP trusted CA.
func NewIssuer(certPEM, keyPEM, trustedPEM []byte) (*Issuer, error) {
	if len(certPEM) == 0 && len(keyPEM) == 0 {
		return &Issuer{}, nil
	}

	if len(certPEM) == 0 || len(keyPEM) == 0 || len(trustedPEM) == 0 {
		return nil, errIncompleteIssuer
	}

	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load issuing CA key pair: %w", err)
	}

	issuerCert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse issuing CA certificate: %w", err)
	}

	if !issuerCert.IsCA || issuerCert.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, errIssuerNotCA
	}

	key, ok := pair.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, errIssuerKey
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(trustedPEM) {
		return nil, errTrustedCAs
	}

	intermediates := x509.NewCertPool()

	for _, der := range pair.Certificate[1:] {
		intermediate, parseErr := x509.ParseCertificate(der)
		if parseErr != nil {
			return nil, fmt.Errorf("parse issuing CA chain: %w", parseErr)
		}

		intermediates.AddCert(intermediate)
	}
	//exhaustruct:ignore // Zero-value verify options intentionally use current time and no DNS name.
	_, err = issuerCert.Verify(x509.VerifyOptions{
		Roots: roots, Intermediates: intermediates,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return nil, fmt.Errorf("issuing CA is not trusted for OpAMP clients: %w", err)
	}

	return &Issuer{ca: issuerCert, key: key, chain: certPEM}, nil
}

// Enabled reports whether the server can issue client certificates.
func (i *Issuer) Enabled() bool { return i != nil && i.ca != nil }

// Issue returns a PEM certificate chain and private key for an agent UID.
func (i *Issuer) Issue(instanceUID string) ([]byte, []byte, error) {
	uid, err := uuid.Parse(instanceUID)
	if err != nil || uid == uuid.Nil || uid.String() != instanceUID {
		return nil, nil, fmt.Errorf("%w: instanceUid must be a canonical non-nil UUID", model.ErrInvalidArgument)
	}

	if !i.Enabled() {
		return nil, nil, ErrIssuerUnavailable
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate agent key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), certificateSerialBits))
	if err != nil {
		return nil, nil, fmt.Errorf("generate certificate serial: %w", err)
	}

	now := time.Now()
	notAfter := now.Add(30 * 24 * time.Hour)

	if notAfter.After(i.ca.NotAfter) {
		notAfter = i.ca.NotAfter
	}

	if !notAfter.After(now) {
		return nil, nil, errIssuerExpired
	}

	//exhaustruct:ignore // Only the agent identity, validity, and TLS usages are set.
	template := &x509.Certificate{
		//exhaustruct:ignore // CN is the OpAMP agent instance UID.
		SerialNumber: serial, Subject: pkix.Name{CommonName: instanceUID},
		NotBefore: now.Add(-time.Minute), NotAfter: notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, i.ca, &key.PublicKey, i.key)
	if err != nil {
		return nil, nil, fmt.Errorf("sign agent certificate: %w", err)
	}

	encodedKey, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("encode agent key: %w", err)
	}

	//exhaustruct:ignore // PEM headers are unnecessary.
	chain := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), i.chain...)
	//exhaustruct:ignore // PEM headers are unnecessary.
	return chain, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encodedKey}), nil
}
