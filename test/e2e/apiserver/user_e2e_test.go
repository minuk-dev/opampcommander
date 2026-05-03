//go:build e2e

package apiserver_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestE2E_UsersMe_RequiresAuthentication(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_usersme_test")
	defer apiServer.Stop()

	apiServer.WaitForReady()

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		unauthClient := client.New(apiServer.Endpoint)

		_, err := unauthClient.UserService.GetMyProfile(t.Context())
		require.Error(t, err)

		var respErr *client.ResponseError
		require.ErrorAs(t, err, &respErr)
		assert.Equal(t, http.StatusUnauthorized, respErr.StatusCode)
	})

	t.Run("authenticated admin returns profile with email and roles", func(t *testing.T) {
		authedClient := apiServer.Client()

		profile, err := authedClient.UserService.GetMyProfile(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "test@test.com", profile.User.Spec.Email)
		assert.True(t, profile.User.Spec.IsActive)
		assert.NotEmpty(t, profile.Roles, "profile should include at least the default role")
	})
}
