package filter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/application/service/certificate/filter"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestSanity_Sanitize(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existing *model.Certificate
		updated  *model.Certificate
		want     *model.Certificate
	}{
		{
			name:     "nil existing returns updated as-is",
			existing: nil,
			updated: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
			},
			want: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
			},
		},
		{
			name: "nil updated returns nil",
			existing: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
			},
			updated: nil,
			want:    nil,
		},
		{
			name: "preserves createdAt from existing",
			existing: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: model.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
			updated: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
				Spec: model.CertificateSpec{
					Cert: []byte("new-cert-data"),
				},
			},
			want: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Spec: model.CertificateSpec{
					Cert: []byte("new-cert-data"),
				},
				Status: model.CertificateStatus{
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
			existing: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: model.CertificateStatus{
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
			updated: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: newTime,
				},
				Status: model.CertificateStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusFalse,
							Message: "Should be overwritten",
						},
					},
				},
			},
			want: &model.Certificate{
				Metadata: model.CertificateMetadata{
					Name:      "test-cert",
					CreatedAt: fixedTime,
				},
				Status: model.CertificateStatus{
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
