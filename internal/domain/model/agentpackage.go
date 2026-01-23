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
	SourceURL string
}

type AgentPackageStatus struct {
}
