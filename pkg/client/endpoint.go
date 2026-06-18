//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListEndpointURL is the path to list all endpoints.
	ListEndpointURL = "/api/v1/namespaces/{namespace}/endpoints"
	// GetEndpointURL is the path to get an endpoint by name.
	GetEndpointURL = "/api/v1/namespaces/{namespace}/endpoints/{id}"
	// CreateEndpointURL is the path to create a new endpoint.
	CreateEndpointURL = "/api/v1/namespaces/{namespace}/endpoints"
	// UpdateEndpointURL is the path to update an existing endpoint.
	UpdateEndpointURL = "/api/v1/namespaces/{namespace}/endpoints/{id}"
	// DeleteEndpointURL is the path to delete an endpoint.
	DeleteEndpointURL = "/api/v1/namespaces/{namespace}/endpoints/{id}"
)

// EndpointService provides methods to interact with endpoints.
type EndpointService struct {
	service *service
}

// NewEndpointService creates a new EndpointService.
func NewEndpointService(service *service) *EndpointService {
	return &EndpointService{
		service: service,
	}
}

// GetEndpoint retrieves an endpoint by its namespace and name.
func (s *EndpointService) GetEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.Endpoint, error) {
	getSettings := newGetSettings(opts)

	var endpoint v1.Endpoint

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&endpoint).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)
	getSettings.applyTo(req)

	res, err := req.Get(GetEndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get endpoint(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &endpoint, nil
}

// EndpointListResponse represents a list of endpoints with metadata.
type EndpointListResponse = v1.ListResponse[v1.Endpoint]

// ListEndpoints lists all endpoints in a namespace.
func (s *EndpointService) ListEndpoints(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*EndpointListResponse, error) {
	listSettings := newListSettings(opts)

	var listResponse EndpointListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)
	listSettings.applyTo(req)

	res, err := req.Get(ListEndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list endpoints(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &listResponse, nil
}

// CreateEndpoint creates a new endpoint.
func (s *EndpointService) CreateEndpoint(
	ctx context.Context,
	namespace string,
	createRequest *v1.Endpoint,
) (*v1.Endpoint, error) {
	var result v1.Endpoint

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateEndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create endpoint(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// UpdateEndpoint updates an existing endpoint.
func (s *EndpointService) UpdateEndpoint(
	ctx context.Context,
	updateRequest *v1.Endpoint,
) (*v1.Endpoint, error) {
	var result v1.Endpoint

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateEndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update endpoint(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// DeleteEndpoint deletes an endpoint by namespace and name.
func (s *EndpointService) DeleteEndpoint(
	ctx context.Context,
	namespace string,
	name string,
) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteEndpointURL)
	if err != nil {
		return fmt.Errorf("failed to delete endpoint(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete endpoint(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return nil
}
