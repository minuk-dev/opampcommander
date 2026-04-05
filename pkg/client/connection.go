package client

import (
	"context"
	"fmt"
	"strconv"

	uuid "github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// API paths for connection resources.
const (
	// ListConnectionsPath is the path to list connections in a namespace.
	ListConnectionsPath = "/api/v1/namespaces/{namespace}/connections"
	// GetConnectionPath is the path to get a connection by ID in a namespace.
	GetConnectionPath = "/api/v1/namespaces/{namespace}/connections/{id}"
)

// ConnectionService provides methods to interact with connection resources.
type ConnectionService struct {
	service *service
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(service *service) *ConnectionService {
	return &ConnectionService{
		service: service,
	}
}

// GetConnection retrieves a connection by its namespace and ID.
func (s *ConnectionService) GetConnection(
	ctx context.Context,
	namespace string,
	id uuid.UUID,
) (*v1.Connection, error) {
	var result v1.Connection

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", id.String()).
		SetResult(&result).
		Get(GetConnectionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get connection: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get connection: %w", ErrEmptyResponse)
	}

	return &result, nil
}

// ListConnections lists connections in a namespace.
func (s *ConnectionService) ListConnections(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*v1.ConnectionListResponse, error) {
	var (
		listSettings ListSettings
		result       v1.ConnectionListResponse
	)

	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	req := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetResult(&result)

	if listSettings.limit != nil && *listSettings.limit > 0 {
		req.SetQueryParam("limit", strconv.Itoa(*listSettings.limit))
	}

	if listSettings.continueToken != nil && *listSettings.continueToken != "" {
		req.SetQueryParam("continue", *listSettings.continueToken)
	}

	if listSettings.includeDeleted != nil && *listSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	response, err := req.Get(ListConnectionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}
