package entity

import (
	"time"

	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// CertificateKeyFieldName is the key field name for certificate.
	CertificateKeyFieldName = "metadata.name"
)

// Certificate is the MongoDB entity for certificate.
type Certificate struct {
	Common `bson:",inline"`

	Metadata CertificateMetadata `bson:"metadata"`
	Spec     CertificateSpec     `bson:"spec"`
	Status   CertificateStatus   `bson:"status"`
}

// CertificateMetadata represents the metadata of a certificate.
type CertificateMetadata struct {
	Name       string            `bson:"name"`
	Attributes map[string]string `bson:"attributes,omitempty"`
	DeletedAt  *time.Time        `bson:"deletedAt,omitempty"`
}

// CertificateSpec represents the specification of a certificate.
type CertificateSpec struct {
	Cert       []byte `bson:"cert,omitempty"`
	PrivateKey []byte `bson:"privateKey,omitempty"`
	CaCert     []byte `bson:"caCert,omitempty"`
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (c *Certificate) ToDomain() *model.Certificate {
	return &model.Certificate{
		Metadata: c.Metadata.toDomain(),
		Spec:     c.Spec.toDomain(),
		Status:   c.Status.toDomain(),
	}
}

func (m *CertificateMetadata) toDomain() model.CertificateMetadata {
	return model.CertificateMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		DeletedAt:  m.DeletedAt,
	}
}

func (s *CertificateSpec) toDomain() model.CertificateSpec {
	return model.CertificateSpec{
		Cert:       s.Cert,
		PrivateKey: s.PrivateKey,
		CaCert:     s.CaCert,
	}
}

func (s *CertificateStatus) toDomain() model.CertificateStatus {
	return model.CertificateStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// CertificateFromDomain converts domain model to entity.
func CertificateFromDomain(domain *model.Certificate) *Certificate {
	return &Certificate{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: certificateMetadataFromDomain(domain.Metadata),
		Spec:     certificateSpecFromDomain(domain.Spec),
		Status:   certificateStatusFromDomain(domain.Status),
	}
}

func certificateMetadataFromDomain(m model.CertificateMetadata) CertificateMetadata {
	return CertificateMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		DeletedAt:  m.DeletedAt,
	}
}

func certificateSpecFromDomain(s model.CertificateSpec) CertificateSpec {
	return CertificateSpec{
		Cert:       s.Cert,
		PrivateKey: s.PrivateKey,
		CaCert:     s.CaCert,
	}
}

func certificateStatusFromDomain(s model.CertificateStatus) CertificateStatus {
	return CertificateStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
