package agentmodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// RemoteConfigSchema is a domain aggregate that pins the collector build a remote
// config is validated against. Server-side remote-config validation can only be
// correct if it knows which collector distribution/version the config will run on,
// because the set of valid receivers/processors/exporters/extensions/connectors
// depends on the deployed OTel Collector build.
//
// A RemoteConfigSchema is a referenceable resource: an AgentRemoteConfig references
// it by name (see AgentRemoteConfigSpec.SchemaRef); this type only describes the
// available component catalog and does not itself validate or materialize any
// configuration (validation is layered on top separately).
type RemoteConfigSchema struct {
	Metadata RemoteConfigSchemaMetadata
	Spec     RemoteConfigSchemaSpec
	Status   RemoteConfigSchemaStatus
}

// RemoteConfigSchemaMetadata contains identity and lifecycle information for a
// schema. Together, Namespace and Name form the unique identity of the schema.
type RemoteConfigSchemaMetadata struct {
	// Name is the name of the schema.
	Name string
	// Namespace is the namespace of the schema.
	Namespace string
	// Attributes is a map of user-supplied attributes for the schema.
	Attributes Attributes
	// ResourceVersion is an optimistic-concurrency token. It is 0 for a schema that
	// has never been persisted and is incremented by the persistence layer on every
	// successful write. Callers must not set this by hand — load, mutate, save.
	ResourceVersion int64
	// CreatedAt is the timestamp when the schema was created.
	CreatedAt time.Time
	// DeletedAt is the timestamp when the schema was soft deleted.
	// If nil, the schema is not deleted.
	DeletedAt *time.Time
}

// RemoteConfigSchemaSpec contains the specification for the schema.
type RemoteConfigSchemaSpec struct {
	// Binary identifies the collector build/distribution the schema describes
	// (e.g. "otelcol", "otelcol-contrib", or a vendor/custom distribution). It is
	// an extensible free-form string, mirroring how Host/Container carry an
	// extensible Platform label.
	Binary string
	// Version is the collector version (semver) the schema describes.
	Version string
	// Components is the catalog of components a config for this binary may
	// reference. This is what a validator consults.
	Components ComponentCatalog
	// ComponentConfigs holds the config field schema of each component, keyed by
	// class then component name. Optional and additive: absent components are
	// validated for existence only, present ones field-by-field.
	ComponentConfigs ComponentConfigCatalog
}

// ComponentConfigCatalog maps component class -> component name -> the root config
// field schema (an object field) describing that component's config.
type ComponentConfigCatalog map[string]map[string]ConfigField

// Coarse config field kinds used by ConfigField.Type.
const (
	ConfigFieldTypeString   = "string"
	ConfigFieldTypeInt      = "int"
	ConfigFieldTypeFloat    = "float"
	ConfigFieldTypeBool     = "bool"
	ConfigFieldTypeDuration = "duration"
	ConfigFieldTypeObject   = "object"
	ConfigFieldTypeArray    = "array"
	ConfigFieldTypeMap      = "map"
	ConfigFieldTypeAny      = "any"
)

// ConfigField describes the shape of one config field for structural + type
// validation. For an object, Fields holds the named sub-fields; for an array or map,
// Elem holds the element/value schema. Type "any" accepts any value.
type ConfigField struct {
	Type   string
	Fields map[string]ConfigField
	Elem   *ConfigField
}

// ComponentCatalog lists the available components of a collector build, keyed by
// component class. The class keys are open-ended (typically "receivers",
// "processors", "exporters", "extensions", "connectors", matching the collector
// config sections and the OpAMP available_components report) rather than a fixed
// set, because OpAMP does not guarantee any particular component classes. Values
// are the available component names for that class (names only for now; per-component
// config schema can be layered on later).
type ComponentCatalog map[string][]string

// RemoteConfigSchemaStatus contains the observed state of the schema.
type RemoteConfigSchemaStatus struct {
	// Conditions is a list of conditions that apply to the schema.
	Conditions []model.Condition
}

// NewRemoteConfigSchema creates a new schema with the given identity, marking it as
// created by createdBy at createdAt.
func NewRemoteConfigSchema(
	namespace string,
	name string,
	attributes Attributes,
	createdAt time.Time,
	createdBy string,
) *RemoteConfigSchema {
	return &RemoteConfigSchema{
		Metadata: RemoteConfigSchemaMetadata{
			Name:            name,
			Namespace:       namespace,
			Attributes:      attributes,
			ResourceVersion: 0,
			CreatedAt:       createdAt,
			DeletedAt:       nil,
		},
		Spec: RemoteConfigSchemaSpec{
			Binary:           "",
			Version:          "",
			Components:       ComponentCatalog{},
			ComponentConfigs: nil,
		},
		Status: RemoteConfigSchemaStatus{
			Conditions: []model.Condition{
				{
					Type:               model.ConditionTypeCreated,
					Status:             model.ConditionStatusTrue,
					LastTransitionTime: createdAt,
					Reason:             createdBy,
					Message:            "RemoteConfigSchema created",
				},
			},
		},
	}
}

// IsDeleted returns true if the schema is marked as deleted.
func (s *RemoteConfigSchema) IsDeleted() bool {
	return s.Metadata.DeletedAt != nil
}

// MarkAsCreated stamps the creation timestamp and records a Created condition.
func (s *RemoteConfigSchema) MarkAsCreated(createdAt time.Time, createdBy string) {
	s.Metadata.CreatedAt = createdAt
	s.Status.Conditions = append(s.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "RemoteConfigSchema created",
	})
}

// ApplyUpdate copies the mutable fields from incoming into the receiver while
// preserving immutable identity and lifecycle state (Name, Namespace, CreatedAt,
// DeletedAt, and Status conditions). Callers should load the stored schema,
// ApplyUpdate the client-supplied one onto it, and persist the receiver — this
// keeps the identity intact and avoids forking a phantom record on update.
func (s *RemoteConfigSchema) ApplyUpdate(incoming *RemoteConfigSchema) {
	s.Spec = incoming.Spec
	s.Metadata.Attributes = incoming.Metadata.Attributes
}

// MarkDeleted marks the schema as deleted by setting DeletedAt and adding a
// deleted condition.
func (s *RemoteConfigSchema) MarkDeleted(deletedAt time.Time, deletedBy string) {
	s.Metadata.DeletedAt = &deletedAt
	s.Status.Conditions = append(s.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "RemoteConfigSchema deleted",
	})
}
