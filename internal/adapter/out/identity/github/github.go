// Package github provides a GitHub-specific identity provider adapter
// that resolves GitHub user identities and organization memberships
// for use with the RBAC system.
package github

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v72/github"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.IdentityProviderUsecase = (*IdentityProvider)(nil)

// GitHubClientFactory creates a GitHub API client from an OAuth access token.
// This abstraction enables testing without real HTTP calls.
type GitHubClientFactory func(ctx context.Context, accessToken string) GitHubClient

// GitHubClient abstracts the GitHub API calls needed by the identity provider.
type GitHubClient interface {
	// GetUser returns the authenticated user.
	GetUser(ctx context.Context) (*github.User, error)
	// ListEmails returns the authenticated user's email addresses.
	ListEmails(ctx context.Context) ([]*github.UserEmail, error)
	// ListOrgs returns the organizations the authenticated user belongs to.
	ListOrgs(ctx context.Context) ([]*github.Organization, error)
}

// IdentityProvider implements port.IdentityProviderUsecase for GitHub.
type IdentityProvider struct {
	clientFactory GitHubClientFactory
	logger        *slog.Logger
}

// NewIdentityProvider creates a new GitHub identity provider.
func NewIdentityProvider(
	clientFactory GitHubClientFactory,
	logger *slog.Logger,
) *IdentityProvider {
	return &IdentityProvider{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// ProviderName implements [port.IdentityProviderUsecase].
func (p *IdentityProvider) ProviderName() string {
	return model.IdentityProviderGitHub
}

// ResolveIdentity implements [port.IdentityProviderUsecase].
func (p *IdentityProvider) ResolveIdentity(
	ctx context.Context,
	accessToken string,
) (*model.ExternalIdentity, error) {
	client := p.clientFactory(ctx, accessToken)

	user, err := client.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	emails, err := client.ListEmails(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub emails: %w", err)
	}

	primaryEmail := findPrimaryEmail(emails)
	if primaryEmail == "" {
		return nil, fmt.Errorf("no primary verified email found for GitHub user %s", user.GetLogin())
	}

	orgs, err := client.ListOrgs(ctx)
	if err != nil {
		p.logger.Warn("failed to list GitHub orgs, continuing without org info",
			slog.String("error", err.Error()),
		)

		orgs = nil
	}

	groups := make([]string, 0, len(orgs))
	for _, org := range orgs {
		if org.GetLogin() != "" {
			groups = append(groups, org.GetLogin())
		}
	}

	rawAttrs := map[string]string{
		"login":      user.GetLogin(),
		"avatar_url": user.GetAvatarURL(),
		"html_url":   user.GetHTMLURL(),
	}

	return &model.ExternalIdentity{
		Provider:       model.IdentityProviderGitHub,
		ProviderUserID: fmt.Sprintf("%d", user.GetID()),
		Email:          primaryEmail,
		DisplayName:    user.GetLogin(),
		AvatarURL:      user.GetAvatarURL(),
		Groups:         groups,
		RawAttributes:  rawAttrs,
	}, nil
}

// ListOrganizations implements [port.IdentityProviderUsecase].
func (p *IdentityProvider) ListOrganizations(
	ctx context.Context,
	accessToken string,
) ([]string, error) {
	client := p.clientFactory(ctx, accessToken)

	orgs, err := client.ListOrgs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub organizations: %w", err)
	}

	orgNames := make([]string, 0, len(orgs))
	for _, org := range orgs {
		if org.GetLogin() != "" {
			orgNames = append(orgNames, org.GetLogin())
		}
	}

	return orgNames, nil
}

func findPrimaryEmail(emails []*github.UserEmail) string {
	for _, email := range emails {
		if email != nil && email.GetPrimary() && email.GetVerified() && email.GetEmail() != "" {
			return email.GetEmail()
		}
	}

	return ""
}
