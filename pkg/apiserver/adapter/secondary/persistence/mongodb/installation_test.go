package mongodb_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
)

func TestVerifyInstallation(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	db := startReplicaSetMongo(t).Database("installation")
	ctx := t.Context()
	require.Error(t, mongodb.VerifyInstallation(ctx, db, "first", false))

	results := make(chan error, 1)
	go func() {
		results <- mongodb.VerifyInstallation(ctx, db, "first", true)
	}()

	secondErr := mongodb.VerifyInstallation(ctx, db, "second", true)
	firstErr := <-results

	winner, loser := "first", "second"
	if secondErr == nil {
		winner, loser = loser, winner

		require.ErrorContains(t, firstErr, "does not match")
	} else {
		require.NoError(t, firstErr)
		require.ErrorContains(t, secondErr, "does not match")
	}

	for _, ddlAuto := range []bool{false, true} {
		require.ErrorContains(t, mongodb.VerifyInstallation(ctx, db, loser, ddlAuto), "does not match")
		require.NoError(t, mongodb.VerifyInstallation(ctx, db, winner, ddlAuto))
	}
}
