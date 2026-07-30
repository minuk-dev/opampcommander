//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListRemoteConfigSchemaURL is the path to list all remote config schemas.
	ListRemoteConfigSchemaURL = "/api/v1/namespaces/{namespace}/remoteconfigschemas"
	// GetRemoteConfigSchemaURL is the path to get a remote config schema by name.
	GetRemoteConfigSchemaURL = "/api/v1/namespaces/{namespace}/remoteconfigschemas/{id}"
	// CreateRemoteConfigSchemaURL is the path to create a new remote config schema.
	CreateRemoteConfigSchemaURL = "/api/v1/namespaces/{namespace}/remoteconfigschemas"
	// UpdateRemoteConfigSchemaURL is the path to update an existing remote config schema.
	UpdateRemoteConfigSchemaURL = "/api/v1/namespaces/{namespace}/remoteconfigschemas/{id}"
	// DeleteRemoteConfigSchemaURL is the path to delete a remote config schema.
	DeleteRemoteConfigSchemaURL = "/api/v1/namespaces/{namespace}/remoteconfigschemas/{id}"
)

// RemoteConfigSchemaService provides methods to interact with remote config schemas.
type RemoteConfigSchemaService struct {
	service *service
}

// NewRemoteConfigSchemaService creates a new RemoteConfigSchemaService.
func NewRemoteConfigSchemaService(service *service) *RemoteConfigSchemaService {
	return &RemoteConfigSchemaService{
		service: service,
	}
}

// GetRemoteConfigSchema retrieves a remote config schema by its namespace and name.
func (s *RemoteConfigSchemaService) GetRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.RemoteConfigSchema, error) {
	getSettings := newGetSettings(opts)

	var schema v1.RemoteConfigSchema

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&schema).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)
	getSettings.applyTo(req)

	res, err := req.Get(GetRemoteConfigSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote config schema(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get remote config schema(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &schema, nil
}

// RemoteConfigSchemaListResponse represents a list of remote config schemas with metadata.
type RemoteConfigSchemaListResponse = v1.ListResponse[v1.RemoteConfigSchema]

// ListRemoteConfigSchemas lists all remote config schemas in a namespace.
func (s *RemoteConfigSchemaService) ListRemoteConfigSchemas(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*RemoteConfigSchemaListResponse, error) {
	listSettings := newListSettings(opts)

	var listResponse RemoteConfigSchemaListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)
	listSettings.applyTo(req)

	res, err := req.Get(ListRemoteConfigSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote config schemas(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list remote config schemas(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &listResponse, nil
}

// CreateRemoteConfigSchema creates a new remote config schema.
func (s *RemoteConfigSchemaService) CreateRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	createRequest *v1.RemoteConfigSchema,
) (*v1.RemoteConfigSchema, error) {
	var result v1.RemoteConfigSchema

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateRemoteConfigSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote config schema(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create remote config schema(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// UpdateRemoteConfigSchema updates an existing remote config schema.
func (s *RemoteConfigSchemaService) UpdateRemoteConfigSchema(
	ctx context.Context,
	updateRequest *v1.RemoteConfigSchema,
) (*v1.RemoteConfigSchema, error) {
	var result v1.RemoteConfigSchema

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateRemoteConfigSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update remote config schema(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update remote config schema(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// DeleteRemoteConfigSchema deletes a remote config schema by namespace and name.
func (s *RemoteConfigSchemaService) DeleteRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteRemoteConfigSchemaURL)
	if err != nil {
		return fmt.Errorf("failed to delete remote config schema(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete remote config schema(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return nil
}
