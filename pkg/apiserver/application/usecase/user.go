package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// UserManageUsecase is a use case that handles user management operations.
type UserManageUsecase interface {
	GetUser(ctx context.Context, uid uuid.UUID, options *port.GetOptions) (*v1.User, error)
	GetUserByEmail(ctx context.Context, email string) (*v1.User, error)
	ListUsers(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.User], error)
	CreateUser(ctx context.Context, user *v1.User) (*v1.User, error)
	DeleteUser(ctx context.Context, uid uuid.UUID) error
	GetMyProfile(ctx context.Context, email string) (*v1.UserProfileResponse, error)
}
