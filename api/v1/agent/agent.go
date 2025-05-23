// Package agent provides the agent API for the server
package agent

// UpdateAgentConfigRequest is a struct that represents the request to update the agent configuration.
// It contains the target instance UID and the remote configuration data.
type UpdateAgentConfigRequest struct {
	RemoteConfig any `binding:"required" json:"remoteConfig"`
}
