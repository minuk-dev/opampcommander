package testutil

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestKafkaInitializeOnce(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "initialization error", err: io.ErrUnexpectedEOF},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			hook := func(context.Context, testcontainers.Container) error {
				calls++

				return test.err
			}
			req := testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
						{PostStarts: []testcontainers.ContainerHook{hook, hook}},
					},
				},
			}
			require.NoError(t, kafkaInitializeOnce(&req))

			for range 2 {
				for _, initialize := range req.LifecycleHooks[0].PostStarts {
					require.ErrorIs(t, initialize(t.Context(), nil), test.err)
				}
			}

			require.Equal(t, 2, calls, "each startup hook must run only once and preserve its result")
		})
	}
}
