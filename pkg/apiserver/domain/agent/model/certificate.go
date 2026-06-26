package agentmodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Certificate represents a TLS certificate used for secure communications.
type Certificate struct {
	Metadata CertificateMetadata
	Spec     CertificateSpec
	Status   CertificateStatus
}

// ToAgentCertificate converts the certificate to an AgentCertificate.
func (c *Certificate) ToAgentCertificate() *AgentCertificate {
	return &AgentCertificate{
		Cert:       c.Spec.Cert,
		PrivateKey: c.Spec.PrivateKey,
		CaCert:     c.Spec.CaCert,
	}
}

// MarkAsCreated stamps the creation timestamp and records a Created condition.
func (c *Certificate) MarkAsCreated(createdAt time.Time, createdBy string) {
	c.Metadata.CreatedAt = createdAt

	c.Status.Conditions = append(c.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "Certificate created",
	})
}

// MarkAsUpdated records an Updated condition for the certificate.
func (c *Certificate) MarkAsUpdated(updatedAt time.Time, updatedBy string) {
	c.Status.Conditions = append(c.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeUpdated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: updatedAt,
		Reason:             updatedBy,
		Message:            "Certificate updated",
	})
}

// ApplyUpdate copies the mutable fields from incoming into the receiver while
// preserving immutable identity and lifecycle state (Name, Namespace, CreatedAt,
// DeletedAt, and Status conditions). Callers should load the stored certificate,
// ApplyUpdate the client-supplied one onto it, and persist the receiver.
func (c *Certificate) ApplyUpdate(incoming *Certificate) {
	c.Spec = incoming.Spec
	c.Metadata.Attributes = incoming.Metadata.Attributes
}

// MarkAsDeleted marks the certificate as deleted.
func (c *Certificate) MarkAsDeleted(deletedAt time.Time, deletedBy string) {
	// Set the DeletedAt timestamp in metadata for soft delete filtering
	c.Metadata.DeletedAt = deletedAt

	// Mark as deleted by adding a condition
	c.Status.Conditions = append(c.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Certificate deleted",
	})
}

// CertificateMetadata represents metadata information for a certificate.
type CertificateMetadata struct {
	Name      string
	Namespace string
	// Attributes are optional key-value pairs for the certificate.
	Attributes Attributes
	// CreatedAt is the timestamp when the certificate was created.
	CreatedAt time.Time
	// DeletedAt is the timestamp when the certificate was soft deleted.
	// If zero, the certificate is not deleted.
	DeletedAt time.Time
}

// CertificateSpec represents the specification of a certificate.
type CertificateSpec struct {
	Cert       []byte
	PrivateKey []byte
	CaCert     []byte
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	Conditions []model.Condition
}
