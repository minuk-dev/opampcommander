package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestCertificate_ToAgentCertificate(t *testing.T) {
	t.Parallel()

	t.Run("Converts certificate spec to agent certificate", func(t *testing.T) {
		t.Parallel()

		cert := &model.Certificate{
			Metadata: model.CertificateMetadata{
				Name:       "test-cert",
				Attributes: model.Attributes{"env": "prod"},
			},
			Spec: model.CertificateSpec{
				Cert:       []byte("-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----"),
				PrivateKey: []byte("-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"),
				CaCert:     []byte("-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----"),
			},
			Status: model.CertificateStatus{
				Conditions: nil,
			},
		}

		agentCert := cert.ToAgentCertificate()

		require.NotNil(t, agentCert)
		assert.Equal(t, cert.Spec.Cert, agentCert.Cert)
		assert.Equal(t, cert.Spec.PrivateKey, agentCert.PrivateKey)
		assert.Equal(t, cert.Spec.CaCert, agentCert.CaCert)
	})

	t.Run("Handles empty certificate spec", func(t *testing.T) {
		t.Parallel()

		cert := &model.Certificate{
			Metadata: model.CertificateMetadata{
				Name: "empty-cert",
			},
			Spec:   model.CertificateSpec{},
			Status: model.CertificateStatus{},
		}

		agentCert := cert.ToAgentCertificate()

		require.NotNil(t, agentCert)
		assert.Empty(t, agentCert.Cert)
		assert.Empty(t, agentCert.PrivateKey)
		assert.Empty(t, agentCert.CaCert)
	})
}

func TestCertificate_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("Sets DeletedAt and adds deleted condition", func(t *testing.T) {
		t.Parallel()

		cert := &model.Certificate{
			Metadata: model.CertificateMetadata{
				Name: "test-cert",
			},
			Spec: model.CertificateSpec{
				Cert: []byte("test"),
			},
			Status: model.CertificateStatus{
				Conditions: []model.Condition{},
			},
		}

		deletedAt := time.Now()
		deletedBy := "test-user"

		cert.MarkAsDeleted(deletedAt, deletedBy)

		assert.False(t, cert.Metadata.DeletedAt.IsZero())
		assert.Equal(t, deletedAt, cert.Metadata.DeletedAt)

		require.Len(t, cert.Status.Conditions, 1)
		condition := cert.Status.Conditions[0]
		assert.Equal(t, model.ConditionTypeDeleted, condition.Type)
		assert.Equal(t, model.ConditionStatusTrue, condition.Status)
		assert.Equal(t, deletedAt, condition.LastTransitionTime)
		assert.Equal(t, deletedBy, condition.Reason)
		assert.Equal(t, "Certificate deleted", condition.Message)
	})

	t.Run("Appends to existing conditions", func(t *testing.T) {
		t.Parallel()

		createdAt := time.Now().Add(-time.Hour)
		cert := &model.Certificate{
			Metadata: model.CertificateMetadata{
				Name: "test-cert",
			},
			Spec: model.CertificateSpec{},
			Status: model.CertificateStatus{
				Conditions: []model.Condition{
					{
						Type:               model.ConditionTypeCreated,
						Status:             model.ConditionStatusTrue,
						LastTransitionTime: createdAt,
						Reason:             "system",
						Message:            "Certificate created",
					},
				},
			},
		}

		deletedAt := time.Now()
		cert.MarkAsDeleted(deletedAt, "admin")

		require.Len(t, cert.Status.Conditions, 2)
		assert.Equal(t, model.ConditionTypeCreated, cert.Status.Conditions[0].Type)
		assert.Equal(t, model.ConditionTypeDeleted, cert.Status.Conditions[1].Type)
	})
}

func TestCertificateMetadata(t *testing.T) {
	t.Parallel()

	t.Run("CertificateMetadata with all fields", func(t *testing.T) {
		t.Parallel()

		deletedAt := time.Now()
		metadata := model.CertificateMetadata{
			Name:       "my-cert",
			Attributes: model.Attributes{"team": "platform", "env": "staging"},
			DeletedAt:  deletedAt,
		}

		assert.Equal(t, "my-cert", metadata.Name)
		assert.Equal(t, "platform", metadata.Attributes["team"])
		assert.Equal(t, "staging", metadata.Attributes["env"])
		assert.False(t, metadata.DeletedAt.IsZero())
		assert.Equal(t, deletedAt, metadata.DeletedAt)
	})
}

func TestCertificateSpec(t *testing.T) {
	t.Parallel()

	t.Run("CertificateSpec stores binary data", func(t *testing.T) {
		t.Parallel()

		spec := model.CertificateSpec{
			Cert:       []byte{0x01, 0x02, 0x03},
			PrivateKey: []byte{0x04, 0x05, 0x06},
			CaCert:     []byte{0x07, 0x08, 0x09},
		}

		assert.Equal(t, []byte{0x01, 0x02, 0x03}, spec.Cert)
		assert.Equal(t, []byte{0x04, 0x05, 0x06}, spec.PrivateKey)
		assert.Equal(t, []byte{0x07, 0x08, 0x09}, spec.CaCert)
	})
}
