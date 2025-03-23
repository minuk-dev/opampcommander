package remoteconfig

import (
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

// RemoteConfig is a struct to manage remote config.
type RemoteConfig struct {
	RemoteConfigStatuses []StatusWithKey
	LastErrorMessage     string
	LastModifiedAt       time.Time
}

// StatusWithKey is a struct to manage status with key.
type StatusWithKey struct {
	Key   vo.Hash
	Value Status
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

func New() RemoteConfig {
	return RemoteConfig{
		RemoteConfigStatuses: make([]StatusWithKey, 0),
		LastErrorMessage:     "",
		LastModifiedAt:       time.Now(),
	}
}

func FromOpAMPStatus(status protobufs.RemoteConfigStatuses) Status {
	return Status(status)
}

// SetStatus sets status with key.
func (r *RemoteConfig) SetStatus(newSK StatusWithKey) {
	r.updateLastModifiedAt()

	for i, statusWithKey := range r.RemoteConfigStatuses {
		if statusWithKey.Key.Equal(newSK.Key) {
			r.RemoteConfigStatuses[i] = newSK

			return
		}
	}
}

// GetStatus gets status with key.
func (r *RemoteConfig) GetStatus(key vo.Hash) Status {
	for _, statusWithKey := range r.RemoteConfigStatuses {
		if statusWithKey.Key.Equal(key) {
			return statusWithKey.Value
		}
	}

	return StatusUnset
}

// ListStatuses lists status with key.
func (r *RemoteConfig) ListStatuses() []StatusWithKey {
	return r.RemoteConfigStatuses
}

// SetLastErrorMessage sets last error message.
func (r *RemoteConfig) SetLastErrorMessage(errorMessage string) {
	r.updateLastModifiedAt()
	r.LastErrorMessage = errorMessage
}

func (r *RemoteConfig) updateLastModifiedAt() {
	r.LastModifiedAt = time.Now()
}

func (r Status) WithKey(key vo.Hash) StatusWithKey {
	return StatusWithKey{
		Key:   key,
		Value: r,
	}
}
