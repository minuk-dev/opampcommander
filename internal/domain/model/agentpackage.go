package model

// AgentPackage represents an agent package resource.
type AgentPackage struct {
	Metadata AgentPackageMetadata
	Spec     AgentPackageSpec
	Status   AgentPackageStatus
}

// AgentPackageMetadata represents the metadata of an agent package.
type AgentPackageMetadata struct {
	Name       string
	Attributes Attributes
}

// AgentPackageSpec represents the specification of an agent package.
type AgentPackageSpec struct {
	PackageType string
	Version     string
	DownloadURL string
	ContentHash []byte
	Signature   []byte
	Headers     map[string]string
	Hash        []byte
}

// AgentPackageStatus represents the status of an agent package.
type AgentPackageStatus struct {
	Conditions []Condition
}
