package agent

// Host is a descriptor value object that represents the machine (bare metal or
// VM) an agent runs on. Its fields are derived from the OpenTelemetry "host.*"
// non-identifying resource attributes reported by the agent.
type Host struct {
	// ID is the OpenTelemetry "host.id" attribute. It is stable across reboots
	// and is the preferred identity for the Host aggregate.
	ID string
	// Name is the OpenTelemetry "host.name" attribute (typically the hostname).
	Name string
	// Arch is the OpenTelemetry "host.arch" attribute (e.g. "amd64", "arm64").
	Arch string
	// Type is the OpenTelemetry "host.type" attribute. In cloud environments it
	// carries the instance type (e.g. "m5.xlarge") and hints VM vs bare metal.
	Type string
}

// IsZero reports whether no host attributes were reported.
func (h Host) IsZero() bool {
	return h.ID == "" && h.Name == "" && h.Arch == "" && h.Type == ""
}
