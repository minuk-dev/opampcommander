package model

type AgentPackage struct {
	Metadata AgentPackageMetadata
	Spec     AgentPackageSpec
	Status   AgentPackageStatus
}

type AgentPackageMetadata struct {
	Name       string
	Attributes Attributes
}

type AgentPackageSpec struct {
	PackageType string
	Version     string
	DownloadURL string
	ContentHash []byte
	Signature   []byte
	Headers     map[string]string
	Hash        []byte
}

type AgentPackageStatus struct {
	Conditions []Condition
}
