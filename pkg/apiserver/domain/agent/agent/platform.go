package agent

// EnvironmentKind classifies the compute substrate an agent runs on. It is the
// coarse axis distinguishing a Host (bare metal / VM) from a Container.
type EnvironmentKind string

const (
	// EnvironmentKindHost means the agent runs directly on a machine.
	EnvironmentKindHost EnvironmentKind = "host"
	// EnvironmentKindContainer means the agent runs inside a container.
	EnvironmentKindContainer EnvironmentKind = "container"
	// EnvironmentKindUnknown means the environment could not be determined from
	// the reported attributes.
	EnvironmentKindUnknown EnvironmentKind = "unknown"
)

// Platform is an extensible label for the orchestration/deployment environment.
// It is orthogonal to EnvironmentKind: a Container may be a "docker" or a
// "kubernetes" platform, and a Host may be "baremetal", "vm", etc. New
// environments are added as new Platform values rather than new aggregates.
type Platform string

const (
	// PlatformBareMetal is a physical machine with no cloud provider reported.
	PlatformBareMetal Platform = "baremetal"
	// PlatformVM is a virtual machine (a cloud provider was reported on a host).
	PlatformVM Platform = "vm"
	// PlatformDocker is a standalone container runtime (not orchestrated by k8s).
	PlatformDocker Platform = "docker"
	// PlatformKubernetes is a Kubernetes-managed container (pod).
	PlatformKubernetes Platform = "kubernetes"
	// PlatformECS is an AWS ECS-managed container.
	PlatformECS Platform = "ecs"
	// PlatformUnknown means the platform could not be determined.
	PlatformUnknown Platform = "unknown"
)
