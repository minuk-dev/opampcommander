// Package user provides application services for user management.
package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ applicationport.UserManageUsecase = (*Service)(nil)

// Service is a struct that implements the UserManageUsecase interface.
type Service struct {
	userUsecase userport.UserUsecase
	mapper      *helper.Mapper
	logger      *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	userUsecase userport.UserUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		userUsecase: userUsecase,
		mapper:      helper.NewMapper(),
		logger:      logger,
	}
}

// GetUser implements [applicationport.UserManageUsecase].
func (s *Service) GetUser(ctx context.Context, uid uuid.UUID) (*v1.User, error) {
	user, err := s.userUsecase.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.mapper.MapUserToAPI(user), nil
}

// GetUserByEmail implements [applicationport.UserManageUsecase].
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*v1.User, error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return s.mapper.MapUserToAPI(user), nil
}

// ListUsers implements [applicationport.UserManageUsecase].
func (s *Service) ListUsers(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.User], error) {
	response, err := s.userUsecase.ListUsers(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return &v1.ListResponse[v1.User]{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(user *usermodel.User, _ int) v1.User {
			return *s.mapper.MapUserToAPI(user)
		}),
	}, nil
}

// CreateUser implements [applicationport.UserManageUsecase].
func (s *Service) CreateUser(ctx context.Context, apiUser *v1.User) (*v1.User, error) {
	domainUser := usermodel.NewUser(apiUser.Spec.Email, apiUser.Spec.Username)

	for key, value := range apiUser.Metadata.Labels {
		domainUser.SetLabel(key, value)
	}

	err := s.userUsecase.SaveUser(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.mapper.MapUserToAPI(domainUser), nil
}

// DeleteUser implements [applicationport.UserManageUsecase].
func (s *Service) DeleteUser(ctx context.Context, uid uuid.UUID) error {
	err := s.userUsecase.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetUserProfile implements [applicationport.UserManageUsecase].
func (s *Service) GetUserProfile(ctx context.Context, email string) (*v1.UserProfileResponse, error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &v1.UserProfileResponse{
		User: *s.mapper.MapUserToAPI(user),
	}, nil
}
