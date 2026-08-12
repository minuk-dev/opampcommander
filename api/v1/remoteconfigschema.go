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
	// fixed set of classes, then by component type name.
	Components ComponentCatalog `json:"components"`
} // @name RemoteConfigSchemaSpec

// ComponentCatalog describes the components of a collector build, keyed by component
// class and then by component type name.
type ComponentCatalog map[string]map[string]Component // @name ComponentCatalog

// Component describes one component of a collector build: the signals it handles, how
// stable it is, where it comes from, and — when known — the settings it accepts.
// Everything but Type is optional, so a catalog built from a source that only knows
// the component names (e.g. `otelcol components` on an old collector) is still valid.
type Component struct {
	// Type is the component type name as written in a collector config
	// (e.g. "otlp", "memory_limiter").
	Type string `json:"type"`
	// Signals are the telemetry signals the component handles ("traces", "metrics",
	// "logs", "profiles"). Empty for extensions and for connectors, which use Pairs.
	Signals []string `json:"signals,omitempty"`
	// Stability is the component's stability level per signal (for a connector, per
	// "<from>_to_<to>" pair), e.g. {"metrics": "beta"}.
	Stability map[string]string `json:"stability,omitempty"`
	// Pairs are the signal conversions a connector supports.
	Pairs []SignalPair `json:"pairs,omitempty"`
	// Module is the Go module the component is built from.
	Module string `json:"module,omitempty"`
	// Fields is the root of the component's config field schema (a "map" field). When
	// nil, a config targeting this component is validated for existence only; when
	// set, it is validated field-by-field.
	Fields *ConfigField `json:"fields,omitempty"`
} // @name Component

// SignalPair is one signal conversion a connector supports: it consumes From and
// produces To.
type SignalPair struct {
	From string `json:"from"`
	To   string `json:"to"`
} // @name SignalPair

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
// statically (a third-party config, a recursive one) is left unchecked rather than
// reported as unknown. An empty Type accepts any value.
type ConfigField struct {
	Type     string                 `json:"type,omitempty"`
	Children map[string]ConfigField `json:"children,omitempty"`
	Open     bool                   `json:"open,omitempty"`
	Enum     []string               `json:"enum,omitempty"`
	Doc      string                 `json:"doc,omitempty"`
} // @name ConfigField

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

// RemoteConfigSchemaStatus represents the status of a remote config schema.
type RemoteConfigSchemaStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name RemoteConfigSchemaStatus
