package agent

// Cloud is a descriptor value object that represents the cloud context an agent
// runs in. Its fields are derived from the OpenTelemetry "cloud.*"
// non-identifying resource attributes reported by the agent.
type Cloud struct {
	// Provider is the OpenTelemetry "cloud.provider" attribute
	// (e.g. "aws", "gcp", "azure").
	Provider string
	// Platform is the OpenTelemetry "cloud.platform" attribute
	// (e.g. "aws_ec2", "aws_ecs", "aws_eks", "gcp_kubernetes_engine").
	Platform string
	// Region is the OpenTelemetry "cloud.region" attribute.
	Region string
}

// IsZero reports whether no cloud attributes were reported.
func (c Cloud) IsZero() bool {
	return c.Provider == "" && c.Platform == "" && c.Region == ""
}
