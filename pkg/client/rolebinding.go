//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListRoleBindingURL is the path to list all role bindings within a namespace.
	ListRoleBindingURL = "/api/v1/namespaces/{namespace}/rolebindings"
	// GetRoleBindingURL is the path to get a role binding by namespace and name.
	GetRoleBindingURL = "/api/v1/namespaces/{namespace}/rolebindings/{id}"
	// CreateRoleBindingURL is the path to create a new role binding.
	CreateRoleBindingURL = "/api/v1/namespaces/{namespace}/rolebindings"
	// UpdateRoleBindingURL is the path to update an existing role binding.
	UpdateRoleBindingURL = "/api/v1/namespaces/{namespace}/rolebindings/{id}"
	// DeleteRoleBindingURL is the path to delete a role binding.
	DeleteRoleBindingURL = "/api/v1/namespaces/{namespace}/rolebindings/{id}"
)

// RoleBindingService provides methods to interact with role bindings.
type RoleBindingService struct {
	service *service
}

// NewRoleBindingService creates a new RoleBindingService.
func NewRoleBindingService(service *service) *RoleBindingService {
	return &RoleBindingService{
		service: service,
	}
}

// GetRoleBinding retrieves a role binding by its namespace and name.
func (s *RoleBindingService) GetRoleBinding(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.RoleBinding, error) {
	getSettings := newGetSettings(opts)

	var roleBinding v1.RoleBinding

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&roleBinding).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)
	getSettings.applyTo(req)

	res, err := req.Get(GetRoleBindingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get role binding(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get role binding(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &roleBinding, nil
}

// RoleBindingListResponse represents a list of role bindings with metadata.
type RoleBindingListResponse = v1.ListResponse[v1.RoleBinding]

// ListRoleBindings lists all role bindings in a namespace.
func (s *RoleBindingService) ListRoleBindings(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*RoleBindingListResponse, error) {
	listSettings := newListSettings(opts)

	var listResponse RoleBindingListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)
	listSettings.applyTo(req)

	res, err := req.Get(ListRoleBindingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list role bindings(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &listResponse, nil
}

// CreateRoleBinding creates a new role binding.
func (s *RoleBindingService) CreateRoleBinding(
	ctx context.Context,
	namespace string,
	createRequest *v1.RoleBinding,
) (*v1.RoleBinding, error) {
	var result v1.RoleBinding

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateRoleBindingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create role binding(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create role binding(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// UpdateRoleBinding updates an existing role binding.
func (s *RoleBindingService) UpdateRoleBinding(
	ctx context.Context,
	updateRequest *v1.RoleBinding,
) (*v1.RoleBinding, error) {
	var result v1.RoleBinding

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateRoleBindingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update role binding(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update role binding(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// DeleteRoleBinding deletes a role binding by its namespace and name.
func (s *RoleBindingService) DeleteRoleBinding(ctx context.Context, namespace string, name string) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteRoleBindingURL)
	if err != nil {
		return fmt.Errorf("failed to delete role binding(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete role binding(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
