package usecase

import (
	"context"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AuthProvisioningUsecase creates or updates the user record on login and
// re-applies RBAC policies so the user picks up default roles and matching
// bindings. It is best-effort: failures are logged internally and never
// block authentication. It backs the auth/login HTTP controllers under
// /api/v1/auth/* (basic and github), called on a successful login rather than
// as a REST CRUD operation.
type AuthProvisioningUsecase interface {
	// EnsureUserOnLogin creates or updates the user described by provisioning
	// and re-applies RBAC. It is best-effort and returns nothing: callers
	// must not block login on it.
	EnsureUserOnLogin(ctx context.Context, provisioning port.LoginProvisioning)
}
