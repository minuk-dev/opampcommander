package mongodb // the selector schemas and their translation are unexported

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// TestSelectorSchemasCoverTheDomainAllowlists is the guard that keeps a field the
// API advertises as selectable from having no MongoDB translation — which would
// turn a 200 with a filtered list into a 400 from the adapter — and keeps a field
// this table can translate from going undocumented and unreachable.
func TestSelectorSchemasCoverTheDomainAllowlists(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		schema  selectorSchema
		allowed []string
	}{
		{"agent", agentSelectorSchema, agentmodel.AgentSelectableFields},
		{"agentgroup", agentGroupSelectorSchema, agentmodel.AgentGroupSelectableFields},
		{"agentpackage", agentPackageSelectorSchema, agentmodel.AgentPackageSelectableFields},
		{"agentremoteconfig", agentRemoteConfigSelectorSchema, agentmodel.AgentRemoteConfigSelectableFields},
		{"certificate", certificateSelectorSchema, agentmodel.CertificateSelectableFields},
		{"container", containerSelectorSchema, agentmodel.ContainerSelectableFields},
		{"endpoint", endpointSelectorSchema, agentmodel.EndpointSelectableFields},
		{"host", hostSelectorSchema, agentmodel.HostSelectableFields},
		{"namespace", namespaceSelectorSchema, agentmodel.NamespaceSelectableFields},
		{"remoteconfigschema", remoteConfigSchemaSelectorSchema, agentmodel.RemoteConfigSchemaSelectableFields},
		{"user", userSelectorSchema, usermodel.UserSelectableFields},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			translated := make([]string, 0, len(testCase.schema.fields))
			for field := range testCase.schema.fields {
				translated = append(translated, field)
			}

			assert.ElementsMatch(t, testCase.allowed, translated)

			assert.NotEqual(t, labelsUnsupported, testCase.schema.labelStorage,
				"every listed resource carries a label map")
			assert.NotEmpty(t, testCase.schema.namePath, "every listed resource is searchable by name")
		})
	}
}

func TestSelectorSchema_MapLabelConditions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		raw      string
		expected bson.M
	}{
		{"equals", "env=prod", bson.M{"metadata.attributes.env": "prod"}},
		{"not equals", "env!=prod", bson.M{"metadata.attributes.env": bson.M{"$ne": "prod"}}},
		{"in", "env in (prod,stg)", bson.M{"metadata.attributes.env": bson.M{"$in": []string{"prod", "stg"}}}},
		{"notin", "env notin (prod)", bson.M{"metadata.attributes.env": bson.M{"$nin": []string{"prod"}}}},
		{"exists", "env", bson.M{"metadata.attributes.env": bson.M{"$exists": true}}},
		{"not exists", "!env", bson.M{"metadata.attributes.env": bson.M{"$exists": false}}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			conditions := conditionsFor(t, endpointSelectorSchema, testCase.raw, "")
			require.Len(t, conditions, 1)
			assert.Equal(t, testCase.expected, conditions[0])
		})
	}
}

func TestSelectorSchema_KeyValueLabelConditions(t *testing.T) {
	t.Parallel()

	path := "metadata.description.identifyingAttributes"

	testCases := []struct {
		name     string
		raw      string
		expected bson.M
	}{
		{
			name: "equals",
			raw:  "service.namespace=payments",
			expected: bson.M{path: bson.M{"$elemMatch": bson.M{
				"key": "service.namespace", "value": "payments",
			}}},
		},
		{
			name: "not equals negates the whole elemMatch, so an agent without the attribute matches",
			raw:  "service.namespace!=payments",
			expected: bson.M{"$nor": bson.A{bson.M{path: bson.M{"$elemMatch": bson.M{
				"key": "service.namespace", "value": "payments",
			}}}}},
		},
		{
			name: "in",
			raw:  "service.namespace in (payments,billing)",
			expected: bson.M{path: bson.M{"$elemMatch": bson.M{
				"key": "service.namespace", "value": bson.M{"$in": []string{"payments", "billing"}},
			}}},
		},
		{
			name: "notin",
			raw:  "service.namespace notin (payments)",
			expected: bson.M{"$nor": bson.A{bson.M{path: bson.M{"$elemMatch": bson.M{
				"key": "service.namespace", "value": bson.M{"$in": []string{"payments"}},
			}}}}},
		},
		{
			name:     "exists",
			raw:      "service.namespace",
			expected: bson.M{path: bson.M{"$elemMatch": bson.M{"key": "service.namespace"}}},
		},
		{
			name: "not exists",
			raw:  "!service.namespace",
			expected: bson.M{"$nor": bson.A{bson.M{path: bson.M{"$elemMatch": bson.M{
				"key": "service.namespace",
			}}}}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			conditions := conditionsFor(t, agentSelectorSchema, testCase.raw, "")
			require.Len(t, conditions, 1)
			assert.Equal(t, testCase.expected, conditions[0])
		})
	}
}

func TestSelectorSchema_FieldConditions(t *testing.T) {
	t.Parallel()

	t.Run("string field", func(t *testing.T) {
		t.Parallel()

		conditions := conditionsFor(t, hostSelectorSchema, "", "spec.platform=vm")
		require.Len(t, conditions, 1)
		assert.Equal(t, bson.M{"spec.platform": "vm"}, conditions[0])
	})

	t.Run("bool field is stored as a bool, not the text", func(t *testing.T) {
		t.Parallel()

		conditions := conditionsFor(t, agentSelectorSchema, "", "status.healthy=true")
		require.Len(t, conditions, 1)
		assert.Equal(t, bson.M{"status.componentHealth.healthy": true}, conditions[0])
	})

	t.Run("connected uses the staleness-aware predicate, not the raw flag", func(t *testing.T) {
		t.Parallel()

		conditions := conditionsFor(t, agentSelectorSchema, "", "status.connected=true")
		require.Len(t, conditions, 1)
		assert.Equal(t, connectedMatchFilter(), conditions[0],
			"a field selector must mean exactly what the connected badge and ConnectedOnly mean")
	})

	t.Run("connected=false is the negation of the same predicate", func(t *testing.T) {
		t.Parallel()

		equalsFalse := conditionsFor(t, agentSelectorSchema, "", "status.connected=false")
		notEqualsTrue := conditionsFor(t, agentSelectorSchema, "", "status.connected!=true")

		assert.Equal(t, bson.M{"$nor": bson.A{connectedMatchFilter()}}, equalsFalse[0])
		assert.Equal(t, equalsFalse, notEqualsTrue)
	})
}

func TestSelectorSchema_NamePrefixIsARangeScan(t *testing.T) {
	t.Parallel()

	//exhaustruct:ignore
	conditions, err := endpointSelectorSchema.conditions(&model.ListOptions{NamePrefix: "prod-"})
	require.NoError(t, err)
	require.Len(t, conditions, 1)

	assert.Equal(t, bson.M{"metadata.name": bson.M{"$gte": "prod-", "$lt": "prod."}}, conditions[0],
		"a prefix search must be an index-servable range, never a regex built from client input")
}

func TestSelectorSchema_RejectsWhatItCannotExpress(t *testing.T) {
	t.Parallel()

	t.Run("unknown field", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		_, err := hostSelectorSchema.conditions(&model.ListOptions{
			FieldSelector: selector.FieldSelector{
				{Field: "spec.secret", Operator: selector.OpEquals, Value: "x"},
			},
		})
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	})

	t.Run("non-boolean value for a boolean field", func(t *testing.T) {
		t.Parallel()

		_, err := agentSelectorSchema.conditions(listOptionsFor(t, "", "status.healthy=yes"))
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	})

	t.Run("labels on a resource that has none", func(t *testing.T) {
		t.Parallel()

		_, err := noSelectorSchema.conditions(listOptionsFor(t, "env=prod", ""))
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	})

	t.Run("name search on a resource that has no name", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		_, err := noSelectorSchema.conditions(&model.ListOptions{NamePrefix: "a"})
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	})
}

func TestSelectorSchema_EmptyOptionsProduceNoConditions(t *testing.T) {
	t.Parallel()

	conditions, err := agentSelectorSchema.conditions(nil)
	require.NoError(t, err)
	assert.Empty(t, conditions)

	//exhaustruct:ignore
	conditions, err = agentSelectorSchema.conditions(&model.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, conditions)
}

func conditionsFor(t *testing.T, schema selectorSchema, labels string, fields string) []bson.M {
	t.Helper()

	conditions, err := schema.conditions(listOptionsFor(t, labels, fields))
	require.NoError(t, err)

	return conditions
}

func listOptionsFor(t *testing.T, labels string, fields string) *model.ListOptions {
	t.Helper()

	labelSelector, err := selector.ParseLabels(labels)
	require.NoError(t, err)

	fieldSelector, err := selector.ParseFields(fields)
	require.NoError(t, err)

	//exhaustruct:ignore
	return &model.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
}
