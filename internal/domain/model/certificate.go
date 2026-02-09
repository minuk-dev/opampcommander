package model

// Certificate represents a TLS certificate used for secure communications.
type Certificate struct {
	Metadata CertificateMetadata
	Spec     CertificateSpec
	Status   CertificateStatus
}

// CertificateMetadata represents metadata information for a certificate.
type CertificateMetadata struct {
	Name       string
	Attributes Attributes
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
