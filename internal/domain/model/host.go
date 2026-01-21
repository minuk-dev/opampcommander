package model

type HostEnvironment string

const (
	HostEnvironmentContainer HostEnvironment = "container"
	HostEnvironmentVM        HostEnvironment = "vm"
)

// Host is a value object that represents a host.
type Host struct {
	Environment HostEnvironment
}
