package mongodb_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

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
	owner, err := restarted.Lookup(ctx, key)
	require.NoError(t, err)
	require.Equal(t, winner, owner)

	filter := bson.M{"_id": bson.D{
		{Key: "installationUID", Value: key.InstallationUID},
		{Key: "namespaceUID", Value: key.NamespaceUID},
		{Key: "instanceUID", Value: key.InstanceUID},
	}}
	require.NoError(t, database.Collection("agent_enrollments").FindOne(ctx, filter).Err())

	legacyKey := key
	legacyKey.InstanceUID = "existing-instance"
	filter = bson.M{"_id": bson.D{
		{Key: "installationUID", Value: legacyKey.InstallationUID},
		{Key: "namespaceUID", Value: legacyKey.NamespaceUID},
		{Key: "instanceUID", Value: legacyKey.InstanceUID},
	}}
	legacy := database.Collection("runtime_metadata", options.Collection().SetWriteConcern(writeconcern.Majority()))
	_, err = legacy.InsertOne(ctx, bson.M{
		"_id": filter["_id"], "serviceAccountUID": "legacy-owner",
	})
	require.NoError(t, err)
	owner, err = restarted.Lookup(ctx, legacyKey)
	require.NoError(t, err)
	require.Equal(t, "legacy-owner", owner)
	require.ErrorIs(t, restarted.Claim(ctx, legacyKey, "another-owner"), port.ErrAgentEnrollmentConflict)
	require.NoError(t, restarted.Claim(ctx, legacyKey, "legacy-owner"))

	recreatedNamespace := key
	recreatedNamespace.NamespaceUID = "new-namespace-uid"
	_, err = restarted.Lookup(ctx, recreatedNamespace)
	require.ErrorIs(t, err, port.ErrAgentEnrollmentNotFound)
	require.NoError(t, restarted.Claim(ctx, recreatedNamespace, loser))
	require.ErrorIs(t, restarted.Claim(ctx, key, loser), port.ErrAgentEnrollmentConflict)
	require.NoError(t, mongodb.VerifyInstallation(ctx, database, "installation", false))
}
