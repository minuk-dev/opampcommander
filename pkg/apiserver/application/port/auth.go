package port

import (
	"context"

	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
)

// Identity provider names used when provisioning a user on login. They alias the
// domain values so primary adapters can specify the provider without importing the domain.
const (
	IdentityProviderGitHub = usermodel.IdentityProviderGitHub
	IdentityProviderBasic  = usermodel.IdentityProviderBasic
)

// LoginProvisioning carries the identity attributes resolved during a successful
// login that the application layer uses to create or update the user's record.
type LoginProvisioning struct {
	// Provider is the identity provider that authenticated the user (e.g. "github", "basic").
	Provider string
	// Username is the provider-side username; for providers that have no separate
	// username it is the email.
	Username string
	// Email is the user's email, used as the stable identity key.
	Email string
	// Groups are provider groups (e.g. GitHub orgs) synced to the user's labels.
	Groups []string
}

// AuthProvisioningUsecase creates or updates the user record on login and re-applies
// RBAC policies so the user picks up default roles and matching bindings. It is
// best-effort: failures are logged internally and never block authentication.
type AuthProvisioningUsecase interface {
	EnsureUserOnLogin(ctx context.Context, provisioning LoginProvisioning)
}
