package agentmodel

import (
	"bytes"
	"fmt"

	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
)

// RemoteConfigOffer returns the assigned configuration and its wire hash, including
// for agents that cannot accept it so the API can describe the desired state.
func (a *Agent) RemoteConfigOffer() (*protobufs.AgentRemoteConfig, error) {
	if !a.HasAssignedRemoteConfig() {
		return nil, nil //nolint:nilnil // No assigned configuration means there is no offer.
	}

	files := make(map[string]*protobufs.AgentConfigObject)
	for name, file := range a.Spec.RemoteConfig.ConfigMap.ConfigMap {
		files[name] = &protobufs.AgentConfigObject{Body: file.Body, ContentType: file.ContentType}
	}

	hash, err := vo.NewHashFromAny(files)
	if err != nil {
		return nil, fmt.Errorf("hash remote config: %w", err)
	}

	return &protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{ConfigMap: files}, ConfigHash: hash.Bytes(),
	}, nil
}

// ShouldOfferRemoteConfig keeps failed revisions suppressed even after the agent
// reports a successful rollback to a different hash. Agents without status reporting
// retain the existing redelivery behavior because there is no acknowledgment to use.
func (a *Agent) ShouldOfferRemoteConfig(hash []byte) bool {
	if !a.IsRemoteConfigSupported() {
		return false
	}

	if !a.Metadata.Capabilities.Has(agent.AgentCapabilityReportsRemoteConfig) {
		return true
	}

	status := a.Status.RemoteConfigStatus
	if len(status.LastFailedHash) > 0 && bytes.Equal(hash, status.LastFailedHash) {
		return false
	}

	return !bytes.Equal(hash, status.LastRemoteConfigHash) ||
		(status.Status != RemoteConfigStatusApplying && status.Status != RemoteConfigStatusApplied &&
			status.Status != RemoteConfigStatusFailed)
}

// RemoteConfigSyncStatus relates the last report to the desired configuration.
func (a *Agent) RemoteConfigSyncStatus(hash []byte) string {
	if len(hash) == 0 {
		return "Unassigned"
	}

	if !a.IsRemoteConfigSupported() {
		return "Unsupported"
	}

	status := a.Status.RemoteConfigStatus
	if bytes.Equal(hash, status.LastRemoteConfigHash) {
		switch status.Status {
		case RemoteConfigStatusApplied:
			return "Applied"
		case RemoteConfigStatusApplying:
			return "Applying"
		case RemoteConfigStatusFailed:
			return "Failed"
		case RemoteConfigStatusUnset:
		}
	}

	if len(status.LastFailedHash) > 0 && bytes.Equal(hash, status.LastFailedHash) {
		if status.Status == RemoteConfigStatusApplied {
			return "RolledBack"
		}

		return "Failed"
	}

	return "Pending"
}

// RetryRemoteConfig allows an operator to explicitly retry an unchanged revision.
func (a *Agent) RetryRemoteConfig() error {
	if !a.HasRemoteConfig() {
		return ErrUnsupportedAgentOperation
	}

	a.Status.RemoteConfigStatus = AgentRemoteConfigStatus{}

	return nil
}
