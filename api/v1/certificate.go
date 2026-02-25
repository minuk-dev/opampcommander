package v1

const (
	// CertificateKind is the kind for Certificate resources.
	CertificateKind = "Certificate"
)

// Certificate represents a TLS certificate resource.
type Certificate struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the certificate.
	Metadata CertificateMetadata `json:"metadata"`
	// Spec contains the specification of the certificate.
	Spec CertificateSpec `json:"spec"`
	// Status contains the status of the certificate.
	Status CertificateStatus `json:"status,omitempty"`
} // @name Certificate

// CertificateMetadata represents metadata for a certificate.
type CertificateMetadata struct {
	// Name is the unique name of the certificate.
	Name string `json:"name"`
	// Attributes are optional key-value pairs for the certificate.
	Attributes Attributes `json:"attributes,omitempty"`
	// DeletedAt is the timestamp when the certificate was soft deleted.
	// If nil, the certificate is not deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
} // @name CertificateMetadata

// CertificateSpec represents the specification of a certificate.
type CertificateSpec struct {
	// Cert is the PEM-encoded certificate.
	Cert string `json:"cert,omitempty"`
	// PrivateKey is the PEM-encoded private key.
	PrivateKey string `json:"privateKey,omitempty"`
	// CaCert is the PEM-encoded CA certificate.
	CaCert string `json:"caCert,omitempty"`
} // @name CertificateSpec

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	// Conditions contains the conditions of the certificate.
	Conditions []Condition `json:"conditions,omitempty"`
} // @name CertificateStatus
