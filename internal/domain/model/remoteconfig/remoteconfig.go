// Package remoteconfig provides the remote config for opampcommander.
package remoteconfig

import (
	"fmt"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

// RemoteConfig is a struct to manage remote config.
type RemoteConfig struct {
	RemoteConfigCommands []Command
	LastErrorMessage     string
	LastModifiedAt       time.Time
}

// Command is a struct to manage status with key.
type Command struct {
	Key           vo.Hash
	Status        Status
	Config        []byte
	LastUpdatedAt time.Time
}

// Status is generated from agentToServer of OpAMP.
type Status int32

// Status is a struct to manage
// To manage simply, we use opamp-go's protobufs' value.
const (
	StatusUnset    Status = Status(int32(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_UNSET))
	StatusApplied  Status = Status(int32(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED))
	StatusApplying Status = Status(int32(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLYING))
	StatusFailed   Status = Status(int32(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED))
)

// New creates a new RemoteConfig instance.
func New() RemoteConfig {
	return RemoteConfig{
		RemoteConfigCommands: make([]Command, 0),
		LastErrorMessage:     "",
		LastModifiedAt:       time.Now(),
	}
}

// NewCommand creates a new Command instance.
func NewCommand(config any) (Command, error) {
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return Command{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	hash, err := vo.NewHash(configBytes)
	if err != nil {
		return Command{}, fmt.Errorf("failed to create hash: %w", err)
	}

	return Command{
		Key:           hash,
		Status:        StatusUnset,
		Config:        configBytes,
		LastUpdatedAt: time.Now(),
	}, nil
}

// FromOpAMPStatus converts OpAMP status to domain model.
func FromOpAMPStatus(status protobufs.RemoteConfigStatuses) Status {
	return Status(status)
}

// SetStatus sets status with key.
func (r *RemoteConfig) SetStatus(key vo.Hash, status Status) {
	r.updateLastModifiedAt()

	for i, statusWithKey := range r.RemoteConfigCommands {
		if statusWithKey.Key.Equal(key) {
			r.RemoteConfigCommands[i].Status = status

			return
		}
	}
}

// GetStatus gets status with key.
func (r *RemoteConfig) GetStatus(key vo.Hash) Status {
	for _, statusWithKey := range r.RemoteConfigCommands {
		if statusWithKey.Key.Equal(key) {
			return statusWithKey.Status
		}
	}

	return StatusUnset
}

// ListStatuses lists status with key.
func (r *RemoteConfig) ListStatuses() []Command {
	return r.RemoteConfigCommands
}

// SetLastErrorMessage sets last error message.
func (r *RemoteConfig) SetLastErrorMessage(errorMessage string) {
	r.updateLastModifiedAt()
	r.LastErrorMessage = errorMessage
}

// ApplyRemoteConfig applies remote config with key.
func (r *RemoteConfig) ApplyRemoteConfig(command Command) error {
	r.RemoteConfigCommands = append(r.RemoteConfigCommands, command)

	return nil
}

// IsManaged returns whether the agent is managed by the server.
// An agent is considered managed if it has any remote config commands.
func (r *RemoteConfig) IsManaged() bool {
	return len(r.RemoteConfigCommands) > 0
}

func (r *RemoteConfig) updateLastModifiedAt() {
	r.LastModifiedAt = time.Now()
}
