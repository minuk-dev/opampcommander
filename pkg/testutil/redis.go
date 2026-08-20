package testutil

import (
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	redisTestContainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

const redisImage = "redis:7.4-alpine"

// Redis wraps a testcontainer running a Redis instance.
type Redis struct {
	*Base
	testcontainers.Container

	// Endpoint is the host:port the container is reachable at.
	Endpoint string
}

// StartRedis starts a Redis container and returns it with the address to reach it.
func (b *Base) StartRedis() *Redis {
	b.t.Helper()

	container, err := redisTestContainer.Run(b.t.Context(), redisImage)
	require.NoError(b.t, err)

	endpoint, err := container.Endpoint(b.t.Context(), "")
	require.NoError(b.t, err)

	b.t.Logf("Redis started at: %s", endpoint)

	return &Redis{
		Base:      b,
		Container: container,
		Endpoint:  endpoint,
	}
}
