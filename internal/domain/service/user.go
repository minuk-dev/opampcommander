package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.UserUsecase = (*UserService)(nil)

// UserService implements the UserUsecase interface.
type UserService struct {
	userPersistencePort port.UserPersistencePort
	logger              *slog.Logger
}

// NewUserService creates a new instance of UserService.
func NewUserService(
	userPersistencePort port.UserPersistencePort,
	logger *slog.Logger,
) *UserService {
	return &UserService{
		userPersistencePort: userPersistencePort,
		logger:              logger,
	}
}

// GetUser implements [port.UserUsecase].
func (s *UserService) GetUser(
	ctx context.Context,
	uid uuid.UUID,
) (*model.User, error) {
	user, err := s.userPersistencePort.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from persistence: %w", err)
	}

	return user, nil
}

// GetUserByEmail implements [port.UserUsecase].
func (s *UserService) GetUserByEmail(
	ctx context.Context,
	email string,
) (*model.User, error) {
	user, err := s.userPersistencePort.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email from persistence: %w", err)
	}

	return user, nil
}

// ListUsers implements [port.UserUsecase].
func (s *UserService) ListUsers(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.User], error) {
	resp, err := s.userPersistencePort.ListUsers(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list users from persistence: %w", err)
	}

	return resp, nil
}

// SaveUser implements [port.UserUsecase].
func (s *UserService) SaveUser(
	ctx context.Context,
	user *model.User,
) error {
	_, err := s.userPersistencePort.PutUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save user to persistence: %w", err)
	}

	return nil
}

// DeleteUser implements [port.UserUsecase].
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
