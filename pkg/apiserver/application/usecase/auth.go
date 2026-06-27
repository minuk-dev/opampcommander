package usecase

import (
	"context"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AuthProvisioningUsecase creates or updates the user record on login and re-applies
// RBAC policies so the user picks up default roles and matching bindings. It is
// best-effort: failures are logged internally and never block authentication.
type AuthProvisioningUsecase interface {
	EnsureUserOnLogin(ctx context.Context, provisioning port.LoginProvisioning)
}
