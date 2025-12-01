package mongodb

import (
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
)

// SelectorToMatchConditions converts an AgentSelector to a list of MongoDB match conditions.
func SelectorToMatchConditions(selector entity.AgentSelector) []bson.M {
	// Build match conditions for identifying attributes
	identifyingConditions := IdentifyingAttributesSelectorToMatchConditions(selector.IdentifyingAttributes)

	// Build match conditions for non-identifying attributes
	nonIdentifyingConditions := NonIdentifyingAttributesSelectorToMatchConditions(selector.NonIdentifyingAttributes)

	// Combine all conditions
	allConditions := mergeConditions(identifyingConditions, nonIdentifyingConditions)

	return allConditions
}

// IdentifyingAttributesSelectorToMatchConditions converts identifying attributes to MongoDB match conditions.
func IdentifyingAttributesSelectorToMatchConditions(attributes map[string]string) []bson.M {
	conditions := make([]bson.M, 0, len(attributes))
	for key, value := range attributes {
		conditions = append(conditions, bson.M{
			entity.IdentifyingAttributesFieldName: bson.M{
				"$elemMatch": bson.M{
					"key":   key,
					"value": value,
				},
			},
		})
	}

	return conditions
}

// NonIdentifyingAttributesSelectorToMatchConditions converts non-identifying attributes to MongoDB match conditions.
func NonIdentifyingAttributesSelectorToMatchConditions(attributes map[string]string) []bson.M {
	conditions := make([]bson.M, 0, len(attributes))
	for key, value := range attributes {
		conditions = append(conditions, bson.M{
			entity.NonIdentifyingAttributesFieldName: bson.M{
				"$elemMatch": bson.M{
					"key":   key,
					"value": value,
				},
			},
		})
	}

	return conditions
}

func mergeConditions(conds ...[]bson.M) []bson.M {
	return lo.FlatMap(conds, func(cond []bson.M, _ int) []bson.M {
		return cond
	})
}
