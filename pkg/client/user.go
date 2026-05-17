package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListUserURL is the path to list all users.
	ListUserURL = "/api/v1/users"
	// GetUserURL is the path to get a user by ID.
	GetUserURL = "/api/v1/users/{id}"
	// CreateUserURL is the path to create a new user.
	CreateUserURL = "/api/v1/users"
	// DeleteUserURL is the path to delete a user.
	DeleteUserURL = "/api/v1/users/{id}"
	// GetUserProfileURL is the path to get the current user's profile.
	GetUserProfileURL = "/api/v1/users/me"
)

// UserService provides methods to interact with users.
type UserService struct {
	service *service
}

// NewUserService creates a new UserService.
func NewUserService(service *service) *UserService {
	return &UserService{
		service: service,
	}
}

// UserListResponse represents a list of users with metadata.
type UserListResponse = v1.ListResponse[v1.User]

// ListUsers lists all users.
func (s *UserService) ListUsers(
	ctx context.Context,
	opts ...ListOption,
) (*UserListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.User](
		ctx,
		s.service,
		ListUserURL,
		ListSettings{
			limit:          listSettings.limit,
			continueToken:  listSettings.continueToken,
			includeDeleted: listSettings.includeDeleted,
		},
	)
}

// GetUser retrieves a user by its UID.
func (s *UserService) GetUser(ctx context.Context, uid string, opts ...GetOption) (*v1.User, error) {
	return getResource[v1.User](ctx, s.service, GetUserURL, uid, opts...)
}

// CreateUser creates a new user.
func (s *UserService) CreateUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	return createResource[v1.User, v1.User](ctx, s.service, CreateUserURL, user)
}

// DeleteUser deletes a user by its UID.
func (s *UserService) DeleteUser(ctx context.Context, uid string) error {
	return deleteResource(ctx, s.service, DeleteUserURL, uid)
}

// GetMyProfile retrieves the current user's profile with roles and permissions.
func (s *UserService) GetMyProfile(ctx context.Context) (*v1.UserProfileResponse, error) {
	var result v1.UserProfileResponse

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&result).
		Get(GetUserProfileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get user profile: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get user profile: %w", ErrEmptyResponse)
	}

	return &result, nil
}
