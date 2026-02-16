//go:build e2e

package apiserver_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// testCertificates holds dynamically generated test certificates.
type testCertificates struct {
	CertPEM   string
	KeyPEM    string
	CaCertPEM string
}

// generateTestCertificates creates self-signed certificates for testing.
func generateTestCertificates(t *testing.T) testCertificates {
	t.Helper()

	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate CA key")

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	// Self-sign CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err, "Failed to create CA certificate")

	// Generate server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate server key")

	// Create server certificate template
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Test Server",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	// Sign server certificate with CA
	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err, "Failed to parse CA certificate")

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err, "Failed to create server certificate")

	// Encode to PEM
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})

	return testCertificates{
		CertPEM:   string(serverCertPEM),
		KeyPEM:    string(serverKeyPEM),
		CaCertPEM: string(caCertPEM),
	}
}

// TestE2E_Certificate_CRUD tests the complete CRUD lifecycle of certificates.
func TestE2E_Certificate_CRUD(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	testCerts := generateTestCertificates(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_cert_crud")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Test 1: Create certificate
	certName := "test-cert-crud"
	t.Run("Create Certificate", func(t *testing.T) {
		//exhaustruct:ignore
		cert, err := opampClient.CertificateService.CreateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: certName,
				Attributes: map[string]string{
					"environment": "test",
					"team":        "platform",
				},
			},
			Spec: v1.CertificateSpec{
				Cert:       testCerts.CertPEM,
				PrivateKey: testCerts.KeyPEM,
				CaCert:     testCerts.CaCertPEM,
			},
		})
		require.NoError(t, err, "Failed to create certificate")
		assert.Equal(t, certName, cert.Metadata.Name)
		assert.Equal(t, testCerts.CertPEM, cert.Spec.Cert)
		assert.Equal(t, testCerts.KeyPEM, cert.Spec.PrivateKey)
		assert.Equal(t, testCerts.CaCertPEM, cert.Spec.CaCert)

		t.Logf("Created certificate: %s", certName)
	})

	// Test 2: Get certificate
	t.Run("Get Certificate", func(t *testing.T) {
		cert, err := opampClient.CertificateService.GetCertificate(ctx, certName)
		require.NoError(t, err, "Failed to get certificate")
		assert.Equal(t, certName, cert.Metadata.Name)
		assert.Equal(t, "test", cert.Metadata.Attributes["environment"])
		assert.Equal(t, "platform", cert.Metadata.Attributes["team"])

		t.Logf("Retrieved certificate: %s", certName)
	})

	// Test 3: List certificates
	t.Run("List Certificates", func(t *testing.T) {
		certs, err := opampClient.CertificateService.ListCertificates(ctx)
		require.NoError(t, err, "Failed to list certificates")
		assert.GreaterOrEqual(t, len(certs.Items), 1, "Should have at least one certificate")

		found := false
		for _, c := range certs.Items {
			if c.Metadata.Name == certName {
				found = true
				break
			}
		}
		assert.True(t, found, "Created certificate should be in the list")

		t.Logf("Listed %d certificates", len(certs.Items))
	})

	// Test 4: Update certificate
	t.Run("Update Certificate", func(t *testing.T) {
		//exhaustruct:ignore
		updatedCert, err := opampClient.CertificateService.UpdateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: certName,
				Attributes: map[string]string{
					"environment": "staging",
					"team":        "platform",
					"updated":     "true",
				},
			},
			Spec: v1.CertificateSpec{
				Cert:       testCerts.CertPEM,
				PrivateKey: testCerts.KeyPEM,
				CaCert:     testCerts.CaCertPEM,
			},
		})
		require.NoError(t, err, "Failed to update certificate")
		assert.Equal(t, "staging", updatedCert.Metadata.Attributes["environment"])
		assert.Equal(t, "true", updatedCert.Metadata.Attributes["updated"])

		t.Logf("Updated certificate: %s", certName)
	})

	// Test 5: Delete certificate
	t.Run("Delete Certificate", func(t *testing.T) {
		err := opampClient.CertificateService.DeleteCertificate(ctx, certName)
		require.NoError(t, err, "Failed to delete certificate")

		// Verify deletion (soft delete - should still exist but marked as deleted)
		cert, err := opampClient.CertificateService.GetCertificate(ctx, certName)
		require.NoError(t, err, "Certificate should still be retrievable after soft delete")

		// Check that it has a deleted condition
		hasDeletedCondition := false
		for _, condition := range cert.Status.Conditions {
			if condition.Type == v1.ConditionTypeDeleted && condition.Status == v1.ConditionStatusTrue {
				hasDeletedCondition = true
				break
			}
		}
		assert.True(t, hasDeletedCondition, "Certificate should have deleted condition")

		t.Logf("Deleted certificate: %s", certName)
	})
}

// TestE2E_Certificate_MultipleCertificates tests managing multiple certificates.
func TestE2E_Certificate_MultipleCertificates(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	testCerts := generateTestCertificates(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_cert_multi")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Create multiple certificates for different environments
	certNames := []string{"cert-dev", "cert-staging", "cert-prod"}

	for _, name := range certNames {
		//exhaustruct:ignore
		_, err := opampClient.CertificateService.CreateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: name,
				Attributes: map[string]string{
					"environment": name[5:], // Extract env from name (e.g., "dev" from "cert-dev")
				},
			},
			Spec: v1.CertificateSpec{
				Cert: testCerts.CertPEM,
			},
		})
		require.NoError(t, err, "Failed to create certificate: %s", name)
		t.Logf("Created certificate: %s", name)
	}

	// Verify all certificates are listed
	certs, err := opampClient.CertificateService.ListCertificates(ctx)
	require.NoError(t, err)

	foundCount := 0
	for _, c := range certs.Items {
		for _, name := range certNames {
			if c.Metadata.Name == name {
				foundCount++
				break
			}
		}
	}
	assert.Equal(t, len(certNames), foundCount, "All created certificates should be listed")

	// Cleanup
	for _, name := range certNames {
		err := opampClient.CertificateService.DeleteCertificate(ctx, name)
		require.NoError(t, err, "Failed to delete certificate: %s", name)
	}
}

// TestE2E_Certificate_PartialData tests certificates with only partial data (e.g., only cert, no key).
func TestE2E_Certificate_PartialData(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	testCerts := generateTestCertificates(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_cert_partial")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Test: Create certificate with only public cert (no private key)
	t.Run("Create Certificate without PrivateKey", func(t *testing.T) {
		certName := "cert-public-only"
		//exhaustruct:ignore
		cert, err := opampClient.CertificateService.CreateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: certName,
			},
			Spec: v1.CertificateSpec{
				Cert: testCerts.CertPEM,
			},
		})
		require.NoError(t, err, "Should create certificate with only public cert")
		assert.Equal(t, testCerts.CertPEM, cert.Spec.Cert)
		assert.Empty(t, cert.Spec.PrivateKey)

		// Cleanup
		_ = opampClient.CertificateService.DeleteCertificate(ctx, certName)
	})

	// Test: Create certificate with only CA cert
	t.Run("Create Certificate with only CaCert", func(t *testing.T) {
		certName := "cert-ca-only"
		//exhaustruct:ignore
		cert, err := opampClient.CertificateService.CreateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: certName,
			},
			Spec: v1.CertificateSpec{
				CaCert: testCerts.CaCertPEM,
			},
		})
		require.NoError(t, err, "Should create certificate with only CA cert")
		assert.Empty(t, cert.Spec.Cert)
		assert.Empty(t, cert.Spec.PrivateKey)
		assert.Equal(t, testCerts.CaCertPEM, cert.Spec.CaCert)

		// Cleanup
		_ = opampClient.CertificateService.DeleteCertificate(ctx, certName)
	})
}

// TestE2E_Certificate_NotFound tests error handling for non-existent certificates.
func TestE2E_Certificate_NotFound(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_cert_notfound")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Test: Get non-existent certificate
	_, err := opampClient.CertificateService.GetCertificate(ctx, "non-existent-cert")
	assert.Error(t, err, "Should return error for non-existent certificate")

	t.Logf("Correctly received error for non-existent certificate: %v", err)
}

// TestE2E_Certificate_Pagination tests certificate listing with pagination.
func TestE2E_Certificate_Pagination(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	testCerts := generateTestCertificates(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_cert_pagination")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Create 5 certificates
	numCerts := 5
	createdNames := make([]string, numCerts)

	for i := range numCerts {
		name := "cert-page-" + string(rune('a'+i))
		createdNames[i] = name
		//exhaustruct:ignore
		_, err := opampClient.CertificateService.CreateCertificate(ctx, &v1.Certificate{
			Metadata: v1.CertificateMetadata{
				Name: name,
			},
			Spec: v1.CertificateSpec{
				Cert: testCerts.CertPEM,
			},
		})
		require.NoError(t, err)
	}

	// Test: List with limit
	t.Run("List with limit", func(t *testing.T) {
		certs, err := opampClient.CertificateService.ListCertificates(ctx,
			client.WithLimit(2))
		require.NoError(t, err)
		assert.LessOrEqual(t, len(certs.Items), 2, "Should return at most 2 items")
	})

	// Test: List all (without pagination)
	t.Run("List all", func(t *testing.T) {
		certs, err := opampClient.CertificateService.ListCertificates(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(certs.Items), numCerts, "Should return all certificates")
	})

	// Cleanup
	for _, name := range createdNames {
		_ = opampClient.CertificateService.DeleteCertificate(ctx, name)
	}
}
