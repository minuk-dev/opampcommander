package filter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/application/service/agentpackage/filter"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestSanity_Sanitize(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existing *model.AgentPackage
		updated  *model.AgentPackage
		want     *model.AgentPackage
	}{
		{
			name:     "nil existing returns updated as-is",
			existing: nil,
			updated: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: newTime,
				},
			},
			want: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: newTime,
				},
			},
		},
		{
			name: "nil updated returns nil",
			existing: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: fixedTime,
				},
			},
			updated: nil,
			want:    nil,
		},
		{
			name: "preserves createdAt from existing",
			existing: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: fixedTime,
				},
				Status: model.AgentPackageStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
			updated: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: newTime,
				},
				Spec: model.AgentPackageSpec{
					Version: "2.0.0",
				},
			},
			want: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: fixedTime,
				},
				Spec: model.AgentPackageSpec{
					Version: "2.0.0",
				},
				Status: model.AgentPackageStatus{
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
			existing: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: fixedTime,
				},
				Status: model.AgentPackageStatus{
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
			updated: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: newTime,
				},
				Status: model.AgentPackageStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusFalse,
							Message: "Should be overwritten",
						},
					},
				},
			},
			want: &model.AgentPackage{
				Metadata: model.AgentPackageMetadata{
					Name:      "test-package",
					CreatedAt: fixedTime,
				},
				Status: model.AgentPackageStatus{
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
