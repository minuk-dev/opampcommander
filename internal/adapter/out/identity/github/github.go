// Package github provides a GitHub-specific identity provider adapter
// that resolves GitHub user identities and organization memberships
// for use with the RBAC system.
package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/go-github/v72/github"

	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.IdentityProviderUsecase = (*IdentityProvider)(nil)

// ErrNoPrimaryEmail is returned when no primary verified email is found.
var ErrNoPrimaryEmail = errors.New("no primary verified email found")

// ClientFactory creates a GitHub API client from an OAuth access token.
// This abstraction enables testing without real HTTP calls.
type ClientFactory func(ctx context.Context, accessToken string) Client

// Client abstracts the GitHub API calls needed by the identity provider.
type Client interface {
	// GetUser returns the authenticated user.
	GetUser(ctx context.Context) (*github.User, error)
	// ListEmails returns the authenticated user's email addresses.
	ListEmails(ctx context.Context) ([]*github.UserEmail, error)
	// ListOrgs returns the organizations the authenticated user belongs to.
	ListOrgs(ctx context.Context) ([]*github.Organization, error)
}

// IdentityProvider implements userport.IdentityProviderUsecase for GitHub.
type IdentityProvider struct {
	clientFactory ClientFactory
	logger        *slog.Logger
}

// NewIdentityProvider creates a new GitHub identity provider.
func NewIdentityProvider(
	clientFactory ClientFactory,
	logger *slog.Logger,
) *IdentityProvider {
	return &IdentityProvider{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// ProviderName implements [userport.IdentityProviderUsecase].
func (p *IdentityProvider) ProviderName() string {
	return usermodel.IdentityProviderGitHub
}

// ResolveIdentity implements [userport.IdentityProviderUsecase].
func (p *IdentityProvider) ResolveIdentity(
	ctx context.Context,
	accessToken string,
) (*usermodel.ExternalIdentity, error) {
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
		return nil, fmt.Errorf("%w for GitHub user %s", ErrNoPrimaryEmail, user.GetLogin())
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

	return &usermodel.ExternalIdentity{
		Provider:       usermodel.IdentityProviderGitHub,
		ProviderUserID: strconv.FormatInt(user.GetID(), 10),
		Email:          primaryEmail,
		DisplayName:    user.GetLogin(),
		AvatarURL:      user.GetAvatarURL(),
		Groups:         groups,
		RawAttributes:  rawAttrs,
	}, nil
}

// ListOrganizations implements [userport.IdentityProviderUsecase].
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
