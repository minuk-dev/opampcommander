package agentmodel

import (
	"slices"
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
}

// ComponentCatalog describes the components of a collector build, keyed by component
// class and then by component type name. The class keys are open-ended (typically
// "receivers", "processors", "exporters", "extensions", "connectors", matching the
// collector config sections and the OpAMP available_components report) rather than a
// fixed set, because OpAMP does not guarantee any particular component classes.
type ComponentCatalog map[string]map[string]Component

// Component describes one component of a collector build: the signals it handles, how
// stable it is, where it comes from, and — when known — the settings it accepts.
// Everything but Type is optional, so a catalog built from a source that only knows
// the component names is still usable for existence checks.
type Component struct {
	// Type is the component type name as written in a collector config.
	Type string
	// Signals are the telemetry signals the component handles. Empty for extensions
	// and for connectors, which use Pairs.
	Signals []string
	// Stability is the component's stability level per signal (for a connector, per
	// "<from>_to_<to>" pair).
	Stability map[string]string
	// Pairs are the signal conversions a connector supports.
	Pairs []SignalPair
	// Module is the Go module the component is built from.
	Module string
	// Fields is the root of the component's config field schema. When nil, a config
	// targeting this component is validated for existence only.
	Fields *ConfigField
}

// SignalPair is one signal conversion a connector supports: it consumes From and
// produces To.
type SignalPair struct {
	From string
	To   string
}

// Telemetry signals a component can handle.
const (
	SignalTraces   = "traces"
	SignalMetrics  = "metrics"
	SignalLogs     = "logs"
	SignalProfiles = "profiles"
)

// ConfigField describes the shape of one config field for structural + type
// validation. A "map" field carries its named settings in Children; a "list" field
// carries its element schema in Children under ConfigFieldItemKey. An Open field
// accepts keys beyond those in Children, which is how config that cannot be resolved
// statically is left unchecked rather than reported as unknown. An empty Type accepts
// any value.
type ConfigField struct {
	Type     string
	Children map[string]ConfigField
	Open     bool
	Enum     []string
	Doc      string
}

// ConfigFieldItemKey is the Children key under which a list field carries its element
// schema.
const ConfigFieldItemKey = "item"

// Config field types used by ConfigField.Type. An unset type accepts any value.
const (
	ConfigFieldTypeString   = "string"
	ConfigFieldTypeInt      = "int"
	ConfigFieldTypeFloat    = "float"
	ConfigFieldTypeBool     = "bool"
	ConfigFieldTypeDuration = "duration"
	ConfigFieldTypeMap      = "map"
	ConfigFieldTypeList     = "list"
)

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
			Binary:     "",
			Version:    "",
			Components: ComponentCatalog{},
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

// Component returns the component of the given class and type name, and whether the
// catalog describes it.
func (s *RemoteConfigSchema) Component(class string, name string) (Component, bool) {
	component, ok := s.Spec.Components[class][name]

	return component, ok
}

// SupportsSignal reports whether the component handles the given signal. A component
// whose signals are unknown (an existence-only catalog entry) supports every signal,
// so a shallow catalog never produces false alarms.
func (c Component) SupportsSignal(signal string) bool {
	if len(c.Signals) == 0 {
		return true
	}

	return slices.Contains(c.Signals, signal)
}

// SupportsPair reports whether the connector converts fromSignal into toSignal. A
// connector whose pairs are unknown supports every conversion.
func (c Component) SupportsPair(fromSignal string, toSignal string) bool {
	if len(c.Pairs) == 0 {
		return true
	}

	return slices.ContainsFunc(c.Pairs, func(pair SignalPair) bool {
		return pair.From == fromSignal && pair.To == toSignal
	})
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
