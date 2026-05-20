package testutil

import (
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const mongoDBImage = "mongo:4.4.10"

// MongoDB wraps a testcontainer running a MongoDB instance. The container is
// started as a single-node replica set so MongoDB transactions are available.
type MongoDB struct {
	*Base
	testcontainers.Container

	URI string
}

// StartMongoDB starts a MongoDB container as a single-node replica set and
// returns a MongoDB instance. The returned URI already includes
// directConnection=true (see [WithDirectConnection]) so any client that
// receives it can talk to the container from the host without needing to
// resolve the rs.initiate-registered internal IP.
func (b *Base) StartMongoDB() *MongoDB {
	b.t.Helper()

	container, err := mongoTestContainer.Run(
		b.t.Context(),
		mongoDBImage,
		mongoTestContainer.WithReplicaSet("rs0"),
	)
	require.NoError(b.t, err)

	uri, err := container.ConnectionString(b.t.Context())
	require.NoError(b.t, err)

	return &MongoDB{
		Base:      b,
		Container: container,
		URI:       WithDirectConnection(uri),
	}
}

// WithDirectConnection appends directConnection=true to a MongoDB URI. The
// testcontainers mongodb module calls rs.initiate with the container's
// internal Docker IP, which is not reachable from the host. Setting
// directConnection=true makes the driver bypass SDAM topology discovery and
// talk to the published port directly; transactions still work because that
// single node IS the replica-set primary.
func WithDirectConnection(uri string) string {
	if strings.Contains(uri, "directConnection=") {
		return uri
	}

	if strings.Contains(uri, "?") {
		return uri + "&directConnection=true"
	}

	return uri + "?directConnection=true"
}
