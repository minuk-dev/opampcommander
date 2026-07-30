package agentmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestServerID_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "server-1", agentmodel.ServerID("server-1").String())
	assert.Empty(t, agentmodel.ServerID("").String())
}

func TestServerAddress_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "10.0.0.5:8081", agentmodel.ServerAddress("10.0.0.5:8081").String())
	assert.Empty(t, agentmodel.ServerAddress("").String())
}
