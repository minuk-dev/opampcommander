package v1

const (
	// NamespaceKind is the kind of the namespace resource.
	NamespaceKind = "Namespace"
)

// Namespace represents a namespace resource that groups agent groups.
type Namespace struct {
	Metadata NamespaceMetadata `json:"metadata"`
	Status   NamespaceStatus   `json:"status"`
} // @name Namespace

// NamespaceMetadata represents the metadata of a namespace.
type NamespaceMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   Time              `json:"createdAt"`
	DeletedAt   *Time             `json:"deletedAt,omitempty"`
} // @name NamespaceMetadata

// NamespaceStatus represents the status of a namespace.
type NamespaceStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name NamespaceStatus
