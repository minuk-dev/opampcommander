package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/agent"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()

	// given
	options := agent.CommandOptions{
		GlobalConfig: &config.GlobalConfig{},
	}

	// when
	cmd := agent.NewCommand(options)

	// then
	assert.NotNil(t, cmd)
	assert.Equal(t, "agent", cmd.Use)
	assert.Contains(t, cmd.Short, "Set agent configurations")
	
	// Check that new-instance-uid subcommand exists
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 1)
	assert.Contains(t, subcommands[0].Use, "new-instance-uid")
}

func TestNewInstanceUIDCommand(t *testing.T) {
	t.Parallel()

	// given
	options := agent.CommandOptions{
		GlobalConfig: &config.GlobalConfig{},
	}

	// when
	cmd := agent.NewCommand(options)
	subCmd := cmd.Commands()[0]

	// then - Check args validation
	err := subCmd.Args(subCmd, []string{"arg1", "arg2"})
	assert.NoError(t, err)
	
	err = subCmd.Args(subCmd, []string{"arg1"})
	assert.Error(t, err)
	
	err = subCmd.Args(subCmd, []string{"arg1", "arg2", "arg3"})
	assert.Error(t, err)
}