package filter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/application/service/agentgroup/filter"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestSanity_Sanitize(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existing *model.AgentGroup
		updated  *model.AgentGroup
		want     *model.AgentGroup
	}{
		{
			name:     "nil existing returns updated as-is",
			existing: nil,
			updated: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
			},
			want: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
			},
		},
		{
			name: "nil updated returns nil",
			existing: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
			},
			updated: nil,
			want:    nil,
		},
		{
			name: "preserves createdAt from existing",
			existing: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: model.AgentGroupStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
			updated: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
				Spec: model.AgentGroupSpec{
					Priority: 10,
				},
			},
			want: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Spec: model.AgentGroupSpec{
					Priority: 10,
				},
				Status: model.AgentGroupStatus{
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
			name: "preserves conditions from existing",
			existing: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: model.AgentGroupStatus{
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
			updated: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
				Status: model.AgentGroupStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusFalse,
							Message: "Should be overwritten",
						},
					},
				},
			},
			want: &model.AgentGroup{
				Metadata: model.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: model.AgentGroupStatus{
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
