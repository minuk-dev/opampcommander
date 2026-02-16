package model

import "time"

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

// MarkAsDeleted marks the certificate as deleted.
func (c *Certificate) MarkAsDeleted(deletedAt time.Time, deletedBy string) {
	// Set the DeletedAt timestamp in metadata for soft delete filtering
	c.Metadata.DeletedAt = &deletedAt

	// Mark as deleted by adding a condition
	c.Status.Conditions = append(c.Status.Conditions, Condition{
		Type:               ConditionTypeDeleted,
		Status:             ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Certificate deleted",
	})
}

// CertificateMetadata represents metadata information for a certificate.
type CertificateMetadata struct {
	Name       string
	Attributes Attributes
	DeletedAt  *time.Time
}

// CertificateSpec represents the specification of a certificate.
type CertificateSpec struct {
	Cert       []byte
	PrivateKey []byte
	CaCert     []byte
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	Conditions []Condition
}
