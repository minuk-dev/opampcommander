package v1

const (
	// RemoteConfigSchemaKind is the kind for RemoteConfigSchema resources.
	RemoteConfigSchemaKind = "RemoteConfigSchema"
)

// RemoteConfigSchema pins the collector build (distribution + version) a remote
// config is validated against, describing the component catalog available to
// configs that target it.
type RemoteConfigSchema struct {
	Kind       string                     `json:"kind"`
	APIVersion string                     `json:"apiVersion"`
	Metadata   RemoteConfigSchemaMetadata `json:"metadata"`
	Spec       RemoteConfigSchemaSpec     `json:"spec"`
	Status     RemoteConfigSchemaStatus   `json:"status"`
} // @name RemoteConfigSchema

// RemoteConfigSchemaMetadata represents the metadata of a remote config schema.
type RemoteConfigSchemaMetadata struct {
	Name       string     `json:"name"`
	Namespace  string     `json:"namespace"`
	Attributes Attributes `json:"attributes"`
	CreatedAt  Time       `json:"createdAt"`
} // @name RemoteConfigSchemaMetadata

// RemoteConfigSchemaSpec represents the specification of a remote config schema.
type RemoteConfigSchemaSpec struct {
	// Binary identifies the collector build/distribution (e.g. "otelcol",
	// "otelcol-contrib", or a vendor/custom distribution). Extensible free-form string.
	Binary string `json:"binary"`
	// Version is the collector version (semver) the schema describes.
	Version string `json:"version"`
	// Components is the catalog of components a config for this binary may reference,
	// keyed by open-ended component class (e.g. "receivers", "processors",
	// "exporters", "extensions", "connectors") since OpAMP does not guarantee a
	// fixed set of classes. Values are the available component names (names only).
	Components ComponentCatalog `json:"components"`
	// ComponentConfigs holds the config field schema of each component, keyed by
	// component class then component name. It is optional and additive: when a
	// component has no entry here, only its existence is validated; when present, a
	// config targeting that component is validated field-by-field (unknown keys and
	// type mismatches). Populated from the collector's component config structs.
	ComponentConfigs ComponentConfigCatalog `json:"componentConfigs,omitempty"`
} // @name RemoteConfigSchemaSpec

// ComponentCatalog lists the available components of a collector build, keyed by
// component class.
type ComponentCatalog map[string][]string // @name ComponentCatalog

// ComponentConfigCatalog maps component class -> component name -> the root config
// field schema (an object field) describing that component's config.
type ComponentConfigCatalog map[string]map[string]ConfigField // @name ComponentConfigCatalog

// ConfigField describes the shape of one config field for structural + type
// validation. Type is a coarse kind (see the ConfigFieldType* constants). For an
// object, Fields holds the named sub-fields; for an array or map, Elem holds the
// element/value schema. A field of type "any" accepts any value (used where the
// component config is open-ended).
type ConfigField struct {
	Type   string                 `json:"type"`
	Fields map[string]ConfigField `json:"fields,omitempty"`
	Elem   *ConfigField           `json:"elem,omitempty"`
} // @name ConfigField

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

// RemoteConfigSchemaStatus represents the status of a remote config schema.
type RemoteConfigSchemaStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name RemoteConfigSchemaStatus
