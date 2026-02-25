package v1

const (
	// AgentPackageKind is the kind of the agent package resource.
	AgentPackageKind = "AgentPackage"
)

// AgentPackage represents an agent package resource.
type AgentPackage struct {
	Metadata AgentPackageMetadata `json:"metadata"`
	Spec     AgentPackageSpec     `json:"spec"`
	Status   AgentPackageStatus   `json:"status"`
} // @name AgentPackage

// AgentPackageMetadata represents the metadata of an agent package.
type AgentPackageMetadata struct {
	Name       string     `json:"name"`
	Attributes Attributes `json:"attributes"`
	DeletedAt  *Time      `json:"deletedAt,omitempty"`
} // @name AgentPackageMetadata

// AgentPackageSpec represents the specification of an agent package.
type AgentPackageSpec struct {
	PackageType string            `json:"packageType"`
	Version     string            `json:"version"`
	DownloadURL string            `json:"downloadUrl"`
	ContentHash []byte            `json:"contentHash"`
	Signature   []byte            `json:"signature"`
	Headers     map[string]string `json:"headers"`
	Hash        []byte            `json:"hash"`
} // @name AgentPackageSpec

// AgentPackageStatus represents the status of an agent package.
type AgentPackageStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name AgentPackageStatus
