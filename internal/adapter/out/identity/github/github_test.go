package github_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	gogithub "github.com/google/go-github/v72/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/identity/github"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

var (
	errGitHubAPI = errors.New("github api error")
)

// mockGitHubClient implements github.Client for testing.
type mockGitHubClient struct {
	user   *gogithub.User
	emails []*gogithub.UserEmail
	orgs   []*gogithub.Organization

	getUserErr   error
	listEmailErr error
	listOrgsErr  error
}

func (m *mockGitHubClient) GetUser(_ context.Context) (*gogithub.User, error) {
	return m.user, m.getUserErr
}

func (m *mockGitHubClient) ListEmails(_ context.Context) ([]*gogithub.UserEmail, error) {
	return m.emails, m.listEmailErr
}

func (m *mockGitHubClient) ListOrgs(_ context.Context) ([]*gogithub.Organization, error) {
	return m.orgs, m.listOrgsErr
}

func ptr[T any](v T) *T {
	return &v
}

func newTestProvider(client github.Client) *github.IdentityProvider {
	factory := func(_ context.Context, _ string) github.Client {
		return client
	}

	return github.NewIdentityProvider(factory, slog.Default())
}

func TestIdentityProvider_ProviderName(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(nil)
	assert.Equal(t, usermodel.IdentityProviderGitHub, provider.ProviderName())
}

func TestIdentityProvider_ResolveIdentity(t *testing.T) {
	t.Parallel()

	t.Run("Successfully resolves identity with orgs", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			user: &gogithub.User{
				ID:        ptr(int64(12345)),
				Login:     ptr("testuser"),
				AvatarURL: ptr("https://avatar.url/testuser"),
				HTMLURL:   ptr("https://github.com/testuser"),
			},
			emails: []*gogithub.UserEmail{
				{Email: ptr("test@example.com"), Primary: ptr(true), Verified: ptr(true)},
				{Email: ptr("secondary@example.com"), Primary: ptr(false), Verified: ptr(true)},
			},
			orgs: []*gogithub.Organization{
				{Login: ptr("my-org")},
				{Login: ptr("another-org")},
			},
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.NoError(t, err)
		require.NotNil(t, identity)
		assert.Equal(t, usermodel.IdentityProviderGitHub, identity.Provider)
		assert.Equal(t, "12345", identity.ProviderUserID)
		assert.Equal(t, "test@example.com", identity.Email)
		assert.Equal(t, "testuser", identity.DisplayName)
		assert.Equal(t, "https://avatar.url/testuser", identity.AvatarURL)
		assert.ElementsMatch(t, []string{"my-org", "another-org"}, identity.Groups)
		assert.Equal(t, "testuser", identity.RawAttributes["login"])
	})

	t.Run("Successfully resolves identity without orgs", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			user: &gogithub.User{
				ID:    ptr(int64(99)),
				Login: ptr("solo-user"),
			},
			emails: []*gogithub.UserEmail{
				{Email: ptr("solo@example.com"), Primary: ptr(true), Verified: ptr(true)},
			},
			orgs: []*gogithub.Organization{},
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.NoError(t, err)
		require.NotNil(t, identity)
		assert.Empty(t, identity.Groups)
	})

	t.Run("Continues when ListOrgs fails", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			user: &gogithub.User{
				ID:    ptr(int64(100)),
				Login: ptr("org-fail-user"),
			},
			emails: []*gogithub.UserEmail{
				{Email: ptr("user@example.com"), Primary: ptr(true), Verified: ptr(true)},
			},
			listOrgsErr: errGitHubAPI,
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.NoError(t, err)
		require.NotNil(t, identity)
		assert.Empty(t, identity.Groups)
		assert.Equal(t, "user@example.com", identity.Email)
	})

	t.Run("Fails when GetUser fails", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			getUserErr: errGitHubAPI,
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.Error(t, err)
		assert.Nil(t, identity)
		assert.Contains(t, err.Error(), "failed to get GitHub user")
	})

	t.Run("Fails when ListEmails fails", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			user: &gogithub.User{
				ID:    ptr(int64(1)),
				Login: ptr("user"),
			},
			listEmailErr: errGitHubAPI,
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.Error(t, err)
		assert.Nil(t, identity)
		assert.Contains(t, err.Error(), "failed to list GitHub emails")
	})

	t.Run("Fails when no primary verified email", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			user: &gogithub.User{
				ID:    ptr(int64(1)),
				Login: ptr("no-email-user"),
			},
			emails: []*gogithub.UserEmail{
				{Email: ptr("unverified@example.com"), Primary: ptr(true), Verified: ptr(false)},
				{Email: ptr("secondary@example.com"), Primary: ptr(false), Verified: ptr(true)},
			},
		}

		provider := newTestProvider(client)

		identity, err := provider.ResolveIdentity(t.Context(), "test-token")

		require.Error(t, err)
		assert.Nil(t, identity)
		assert.Contains(t, err.Error(), "no primary verified email found")
	})
}

func TestIdentityProvider_ListOrganizations(t *testing.T) {
	t.Parallel()

	t.Run("Successfully lists orgs", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			orgs: []*gogithub.Organization{
				{Login: ptr("alpha-org")},
				{Login: ptr("beta-org")},
				{Login: ptr("gamma-org")},
			},
		}

		provider := newTestProvider(client)

		orgs, err := provider.ListOrganizations(t.Context(), "test-token")

		require.NoError(t, err)
		assert.Equal(t, []string{"alpha-org", "beta-org", "gamma-org"}, orgs)
	})

	t.Run("Returns empty for user with no orgs", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			orgs: []*gogithub.Organization{},
		}

		provider := newTestProvider(client)

		orgs, err := provider.ListOrganizations(t.Context(), "test-token")

		require.NoError(t, err)
		assert.Empty(t, orgs)
	})

	t.Run("Skips orgs with empty login", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			orgs: []*gogithub.Organization{
				{Login: ptr("real-org")},
				{Login: ptr("")},
				{Login: nil},
			},
		}

		provider := newTestProvider(client)

		orgs, err := provider.ListOrganizations(t.Context(), "test-token")

		require.NoError(t, err)
		assert.Equal(t, []string{"real-org"}, orgs)
	})

	t.Run("Fails when GitHub API fails", func(t *testing.T) {
		t.Parallel()

		client := &mockGitHubClient{
			listOrgsErr: errGitHubAPI,
		}

		provider := newTestProvider(client)

		orgs, err := provider.ListOrganizations(t.Context(), "test-token")

		require.Error(t, err)
		assert.Nil(t, orgs)
		assert.Contains(t, err.Error(), "failed to list GitHub organizations")
	})
}
