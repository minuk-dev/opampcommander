package agent_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// Check that subcommands exist
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 2)

	// Check that both new-instance-uid and connection-settings subcommands exist
	subcommandNames := make([]string, len(subcommands))
	for i, sc := range subcommands {
		subcommandNames[i] = sc.Name()
	}
	assert.Contains(t, subcommandNames, "new-instance-uid")
	assert.Contains(t, subcommandNames, "connection-settings")
}

func TestNewInstanceUIDCommand(t *testing.T) {
	t.Parallel()

	// given
	options := agent.CommandOptions{
		GlobalConfig: &config.GlobalConfig{},
	}

	// when
	cmd := agent.NewCommand(options)

	// Find new-instance-uid subcommand
	var subCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "new-instance-uid" {
			subCmd = sc

			break
		}
	}

	require.NotNil(t, subCmd, "new-instance-uid subcommand should exist")

	// then - Check args validation
	err := subCmd.Args(subCmd, []string{"arg1", "arg2"})
	require.NoError(t, err)

	err = subCmd.Args(subCmd, []string{"arg1"})
	require.Error(t, err)

	err = subCmd.Args(subCmd, []string{"arg1", "arg2", "arg3"})
	require.Error(t, err)
}
