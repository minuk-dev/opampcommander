package filter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/application/service/certificate/filter"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestSanity_Sanitize(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existing *agentmodel.Certificate
		updated  *agentmodel.Certificate
		want     *agentmodel.Certificate
	}{
		{
			name:     "nil existing returns updated as-is",
			existing: nil,
			updated: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
			},
			want: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
			},
		},
		{
			name: "nil updated returns nil",
			existing: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
			},
			updated: nil,
			want:    nil,
		},
		{
			name: "preserves createdAt from existing",
			existing: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
			updated: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
				Spec: agentmodel.CertificateSpec{
					Cert: []byte("new-cert-data"),
				},
			},
			want: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Spec: agentmodel.CertificateSpec{
					Cert: []byte("new-cert-data"),
				},
				Status: agentmodel.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
		},
		{
			name: "preserves status from existing",
			existing: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
						{
							Type:    model.ConditionTypeUpdated,
							Status:  model.ConditionStatusTrue,
							Message: "Updated",
						},
					},
				},
			},
			updated: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
				Status: agentmodel.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusFalse,
							Message: "Should be overwritten",
						},
					},
				},
			},
			want: &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
						{
							Type:    model.ConditionTypeUpdated,
							Status:  model.ConditionStatusTrue,
							Message: "Updated",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := filter.NewSanity()
			got := f.Sanitize(tt.existing, tt.updated)
			assert.Equal(t, tt.want, got)
		})
	}
}
