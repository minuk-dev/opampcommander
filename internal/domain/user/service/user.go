package userservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.UserUsecase = (*UserService)(nil)

// UserService implements the UserUsecase interface.
type UserService struct {
	userPersistencePort userport.UserPersistencePort
	logger              *slog.Logger
}

// NewUserService creates a new instance of UserService.
func NewUserService(
	userPersistencePort userport.UserPersistencePort,
	logger *slog.Logger,
) *UserService {
	return &UserService{
		userPersistencePort: userPersistencePort,
		logger:              logger,
	}
}

// GetUser implements [userport.UserUsecase].
func (s *UserService) GetUser(
	ctx context.Context,
	uid uuid.UUID,
	options *model.GetOptions,
) (*usermodel.User, error) {
	user, err := s.userPersistencePort.GetUser(ctx, uid, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from persistence: %w", err)
	}

	return user, nil
}

// GetUserByEmail implements [userport.UserUsecase].
func (s *UserService) GetUserByEmail(
	ctx context.Context,
	email string,
) (*usermodel.User, error) {
	user, err := s.userPersistencePort.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email from persistence: %w", err)
	}

	return user, nil
}

// ListUsers implements [userport.UserUsecase].
func (s *UserService) ListUsers(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.User], error) {
	resp, err := s.userPersistencePort.ListUsers(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list users from persistence: %w", err)
	}

	return resp, nil
}

// SaveUser implements [userport.UserUsecase].
func (s *UserService) SaveUser(
	ctx context.Context,
	user *usermodel.User,
) error {
	_, err := s.userPersistencePort.PutUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save user to persistence: %w", err)
	}

	return nil
}

// DeleteUser implements [userport.UserUsecase].
func (s *UserService) DeleteUser(
	ctx context.Context,
	uid uuid.UUID,
) error {
	err := s.userPersistencePort.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user from persistence: %w", err)
	}

	return nil
}
