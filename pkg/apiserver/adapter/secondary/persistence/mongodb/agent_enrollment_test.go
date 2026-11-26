package mongodb_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

func TestAgentEnrollmentClaim(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := t.Context()
	database := startReplicaSetMongo(t).Database("enrollment")
	require.NoError(t, mongodb.VerifyInstallation(ctx, database, "installation", true))
	store := mongodb.NewAgentEnrollmentStore(database)
	key := port.AgentEnrollmentKey{InstallationUID: "installation", NamespaceUID: "namespace", InstanceUID: "instance"}

	firstResult := make(chan error, 1)
	go func() { firstResult <- store.Claim(ctx, key, "first-account") }()

	secondErr := store.Claim(ctx, key, "second-account")
	firstErr := <-firstResult

	winner, loser := "first-account", "second-account"
	if secondErr == nil {
		winner, loser = loser, winner

		require.ErrorIs(t, firstErr, port.ErrAgentEnrollmentConflict)
	} else {
		require.NoError(t, firstErr)
		require.ErrorIs(t, secondErr, port.ErrAgentEnrollmentConflict)
	}

	// Recreating the repository models a gateway restart with the same database.
	restarted := mongodb.NewAgentEnrollmentStore(database)
	require.NoError(t, restarted.Claim(ctx, key, winner))
	require.ErrorIs(t, restarted.Claim(ctx, key, loser), port.ErrAgentEnrollmentConflict)

	recreatedNamespace := key
	recreatedNamespace.NamespaceUID = "new-namespace-uid"
	require.NoError(t, restarted.Claim(ctx, recreatedNamespace, loser))
	require.ErrorIs(t, restarted.Claim(ctx, key, loser), port.ErrAgentEnrollmentConflict)
	require.NoError(t, mongodb.VerifyInstallation(ctx, database, "installation", false))
}
