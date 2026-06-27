package security_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func newBasicAuthService(t *testing.T, pepper string, repo *inmemory.UserRepository) *security.Service {
	t.Helper()

	//exhaustruct:ignore
	cfg := &security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: pepper},
		//exhaustruct:ignore
		JWTSettings: security.JWTSettings{
			SigningKey: "test-signing-key",
			Issuer:     "test",
			Expiration: time.Minute,
		},
		AdminSettings: security.AdminSettings{Username: "admin", Password: "adminpass", Email: "admin@x"},
	}

	return security.New(slog.Default(), cfg, http.DefaultClient, security.NewPasswordHasher(cfg), repo)
}

// Regression: when no pepper is configured, a failed/non-admin basic login must surface as
// ErrInvalidUsernameOrPassword (→ HTTP 401), not ErrBasicAuthDisabled (→ HTTP 500).
func TestBasicAuth_DisabledPepper_FailedLoginIsInvalidCredentials(t *testing.T) {
	t.Parallel()

	svc := newBasicAuthService(t, "", inmemory.NewUserRepository())

	_, err := svc.BasicAuth(context.Background(), "someone", "whatever")
	require.ErrorIs(t, err, security.ErrInvalidUsernameOrPassword)
	require.NotErrorIs(t, err, security.ErrBasicAuthDisabled)
}

func TestBasicAuth_DBUser(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewUserRepository()
	svc := newBasicAuthService(t, "test-pepper", repo)

	hasher := security.NewPasswordHasher(&security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: "test-pepper"},
	})

	hash, err := hasher.Hash("s3cret")
	require.NoError(t, err)

	user := usermodel.NewUser("bob@example.com", "bob")
	user.SetPasswordHash(hash)
	_, err = repo.PutUser(context.Background(), user)
	require.NoError(t, err)

	result, err := svc.BasicAuth(context.Background(), "bob", "s3cret")
	require.NoError(t, err)
	assert.Equal(t, "bob@example.com", result.Email)
	assert.NotEmpty(t, result.Token)

	_, err = svc.BasicAuth(context.Background(), "bob", "wrong")
	require.ErrorIs(t, err, security.ErrInvalidUsernameOrPassword)
}

func TestBasicAuth_InactiveDBUserRejected(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewUserRepository()
	svc := newBasicAuthService(t, "test-pepper", repo)

	hasher := security.NewPasswordHasher(&security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: "test-pepper"},
	})

	hash, err := hasher.Hash("s3cret")
	require.NoError(t, err)

	user := usermodel.NewUser("carol@example.com", "carol")
	user.SetPasswordHash(hash)
	user.Spec.IsActive = false
	_, err = repo.PutUser(context.Background(), user)
	require.NoError(t, err)

	_, err = svc.BasicAuth(context.Background(), "carol", "s3cret")
	require.ErrorIs(t, err, security.ErrInvalidUsernameOrPassword)
}
