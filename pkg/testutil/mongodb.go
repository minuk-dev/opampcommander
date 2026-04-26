package testutil

import (
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const mongoDBImage = "mongo:4.4.10"

// MongoDB wraps a testcontainer running a MongoDB instance.
type MongoDB struct {
	*Base
	testcontainers.Container

	URI string
}

// StartMongoDB starts a MongoDB container and returns a MongoDB instance.
func (b *Base) StartMongoDB() *MongoDB {
	b.t.Helper()

	container, err := mongoTestContainer.Run(b.t.Context(), mongoDBImage)
	require.NoError(b.t, err)

	uri, err := container.ConnectionString(b.t.Context())
	require.NoError(b.t, err)

	return &MongoDB{
		Base:      b,
		Container: container,
		URI:       uri,
	}
}
