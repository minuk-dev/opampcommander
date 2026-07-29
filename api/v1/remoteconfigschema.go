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
} // @name RemoteConfigSchemaSpec

// ComponentCatalog lists the available components of a collector build, keyed by
// component class.
type ComponentCatalog map[string][]string // @name ComponentCatalog

// RemoteConfigSchemaStatus represents the status of a remote config schema.
type RemoteConfigSchemaStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name RemoteConfigSchemaStatus
