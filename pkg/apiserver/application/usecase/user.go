package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// UserManageUsecase manages user accounts and lets the authenticated caller
// read their own profile. It backs the /api/v1/users controller.
type UserManageUsecase interface {
	// GetUser returns the user with the given UID, or model.ErrResourceNotExist if
	// absent.
	GetUser(ctx context.Context, uid uuid.UUID, options *port.GetOptions) (*v1.User, error)
	// GetUserByEmail returns the user with the given email, or
	// model.ErrResourceNotExist if absent.
	GetUserByEmail(ctx context.Context, email string) (*v1.User, error)
	// ListUsers returns a paged list of users.
	ListUsers(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.User], error)
	// CreateUser persists a new user.
	CreateUser(ctx context.Context, user *v1.User) (*v1.User, error)
	// DeleteUser removes the user with the given UID.
	DeleteUser(ctx context.Context, uid uuid.UUID) error
	// GetMyProfile returns the profile of the authenticated caller identified by
	// email.
	GetMyProfile(ctx context.Context, email string) (*v1.UserProfileResponse, error)
}
