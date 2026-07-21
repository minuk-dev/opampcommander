//nolint:testpackage // white-box test of the unexported error-response builder
package opamp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateErrorServerToAgent pins the shape of the error-only ServerToAgent: it carries the
// instance UID and a structured error_response, and — per the OpAMP spec, where the agent
// ignores every other field when error_response is set — no desired-state fields.
func TestCreateErrorServerToAgent(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	svc := &Service{}

	msg := svc.createErrorServerToAgent(instanceUID,
		protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest,
		"boom")

	require.NotNil(t, msg)
	assert.Equal(t, instanceUID[:], msg.GetInstanceUid())

	require.NotNil(t, msg.GetErrorResponse())
	assert.Equal(t,
		protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest,
		msg.GetErrorResponse().GetType())
	assert.Equal(t, "boom", msg.GetErrorResponse().GetErrorMessage())

	// No desired-state fields must be set alongside the error.
	assert.Nil(t, msg.GetRemoteConfig())
	assert.Nil(t, msg.GetConnectionSettings())
	assert.Nil(t, msg.GetPackagesAvailable())
	assert.Nil(t, msg.GetCommand())
	assert.Zero(t, msg.GetCapabilities())
}
