package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestAgentGroupMembershipListOptions(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		input   *model.ListOptions
		limit   int64
		invalid bool
	}{
		{name: "nil", limit: 50},
		{name: "zero", input: &model.ListOptions{}, limit: 50},
		{name: "explicit", input: &model.ListOptions{Limit: 1000, Continue: "next", NamePrefix: "group"}, limit: 1000},
		{name: "negative", input: &model.ListOptions{Limit: -1}, invalid: true},
		{name: "too large", input: &model.ListOptions{Limit: 1001}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options, err := model.AgentGroupMembershipListOptions(test.input)
			if test.invalid {
				require.ErrorIs(t, err, model.ErrInvalidArgument)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.limit, options.Limit)

			if test.input != nil {
				require.Equal(t, test.input.Continue, options.Continue)
				require.Equal(t, test.input.NamePrefix, options.NamePrefix)
				require.NotSame(t, test.input, options)
			}
		})
	}
}
