package entity

import (
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// RemoteConfigSchemaNameFieldName is the field name for name in MongoDB.
const RemoteConfigSchemaNameFieldName = "metadata.name"

// RemoteConfigSchemaResourceEntity is the MongoDB entity for a remote config schema resource.
type RemoteConfigSchemaResourceEntity struct {
	ID       *bson.ObjectID                       `bson:"_id,omitempty"`
	Metadata RemoteConfigSchemaResourceMetadata   `bson:"metadata"`
	Spec     RemoteConfigSchemaResourceSpec       `bson:"spec"`
	Status   RemoteConfigSchemaResourceEntityStat `bson:"status"`
}

// RemoteConfigSchemaResourceMetadata represents the metadata of a schema resource.
type RemoteConfigSchemaResourceMetadata struct {
	Name            string            `bson:"name"`
	Namespace       string            `bson:"namespace"`
	Attributes      map[string]string `bson:"attributes,omitempty"`
	ResourceVersion int64             `bson:"resourceVersion"` // Optimistic-concurrency token
	CreatedAt       time.Time         `bson:"createdAt"`
	DeletedAt       *time.Time        `bson:"deletedAt,omitempty"`
}

// RemoteConfigSchemaResourceSpec represents the specification of a schema resource.
type RemoteConfigSchemaResourceSpec struct {
	Binary  string `bson:"binary"`
	Version string `bson:"version"`
	// Components is the component catalog keyed by open-ended component class and then
	// by component type name. The domain Component is stored directly (bson marshals
	// it, including the nested config field schema).
	Components map[string]map[string]agentmodel.Component `bson:"components,omitempty"`
}

// RemoteConfigSchemaResourceEntityStat represents the status of a schema resource.
type RemoteConfigSchemaResourceEntityStat struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (e *RemoteConfigSchemaResourceEntity) ToDomain() *agentmodel.RemoteConfigSchema {
	return &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{
			Name:            e.Metadata.Name,
			Namespace:       e.Metadata.Namespace,
			Attributes:      e.Metadata.Attributes,
			ResourceVersion: e.Metadata.ResourceVersion,
			CreatedAt:       e.Metadata.CreatedAt,
			DeletedAt:       e.Metadata.DeletedAt,
		},
		Spec: agentmodel.RemoteConfigSchemaSpec{
			Binary:     e.Spec.Binary,
			Version:    e.Spec.Version,
			Components: agentmodel.ComponentCatalog(e.Spec.Components),
		},
		Status: agentmodel.RemoteConfigSchemaStatus{
			Conditions: lo.Map(e.Status.Conditions, func(c Condition, _ int) model.Condition {
				return c.ToDomain()
			}),
		},
	}
}

// RemoteConfigSchemaResourceEntityFromDomain converts domain model to entity.
func RemoteConfigSchemaResourceEntityFromDomain(
	schema *agentmodel.RemoteConfigSchema,
) *RemoteConfigSchemaResourceEntity {
	//nolint:exhaustruct // ID is set by MongoDB
	return &RemoteConfigSchemaResourceEntity{
		Metadata: RemoteConfigSchemaResourceMetadata{
			Name:            schema.Metadata.Name,
			Namespace:       schema.Metadata.Namespace,
			Attributes:      schema.Metadata.Attributes,
			ResourceVersion: schema.Metadata.ResourceVersion,
			CreatedAt:       schema.Metadata.CreatedAt,
			DeletedAt:       schema.Metadata.DeletedAt,
		},
		Spec: RemoteConfigSchemaResourceSpec{
			Binary:     schema.Spec.Binary,
			Version:    schema.Spec.Version,
			Components: map[string]map[string]agentmodel.Component(schema.Spec.Components),
		},
		Status: RemoteConfigSchemaResourceEntityStat{
			Conditions: lo.Map(schema.Status.Conditions, func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}
