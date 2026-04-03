package filter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/application/service/agentgroup/filter"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestSanity_Sanitize(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existing *agentmodel.AgentGroup
		updated  *agentmodel.AgentGroup
		want     *agentmodel.AgentGroup
	}{
		{
			name:     "nil existing returns updated as-is",
			existing: nil,
			updated: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
			},
			want: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
			},
		},
		{
			name: "nil updated returns nil",
			existing: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
			},
			updated: nil,
			want:    nil,
		},
		{
			name: "preserves createdAt from existing",
			existing: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.AgentGroupStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusTrue,
							Message: "Created",
						},
					},
				},
			},
			updated: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
				Spec: agentmodel.AgentGroupSpec{
					Priority: 10,
				},
			},
			want: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Spec: agentmodel.AgentGroupSpec{
					Priority: 10,
				},
				Status: agentmodel.AgentGroupStatus{
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
			existing: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.AgentGroupStatus{
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
			updated: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: newTime,
				},
				Status: agentmodel.AgentGroupStatus{
					Conditions: []model.Condition{
						{
							Type:    model.ConditionTypeCreated,
							Status:  model.ConditionStatusFalse,
							Message: "Should be overwritten",
						},
					},
				},
			},
			want: &agentmodel.AgentGroup{
				Metadata: agentmodel.AgentGroupMetadata{
					Name:      "test-group",
					CreatedAt: fixedTime,
				},
				Status: agentmodel.AgentGroupStatus{
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
