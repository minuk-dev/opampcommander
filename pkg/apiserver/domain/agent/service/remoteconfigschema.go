//nolint:dupl // Resource CRUD adapters intentionally mirror the other aggregates.
package agentservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.RemoteConfigSchemaUsecase = (*RemoteConfigSchemaService)(nil)

// RemoteConfigSchemaService provides operations for managing remote config schemas,
// including the creation/update lifecycle rules (identity validation, uniqueness,
// stamping, and immutable-field preservation).
type RemoteConfigSchemaService struct {
	persistence agentport.RemoteConfigSchemaPersistencePort
	clock       clock.Clock
}

// NewRemoteConfigSchemaService creates a new RemoteConfigSchemaService.
func NewRemoteConfigSchemaService(
	persistence agentport.RemoteConfigSchemaPersistencePort,
) *RemoteConfigSchemaService {
	return &RemoteConfigSchemaService{
		persistence: persistence,
		clock:       clock.NewRealClock(),
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *RemoteConfigSchemaService) SetClock(c clock.Clock) {
	s.clock = c
}

// GetRemoteConfigSchema implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) GetRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.RemoteConfigSchema, error) {
	resource, err := s.persistence.GetRemoteConfigSchema(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote config schema: %w", err)
	}

	return resource, nil
}

// ListRemoteConfigSchemas implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) ListRemoteConfigSchemas(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.RemoteConfigSchema], error) {
	resourceResp, err := s.persistence.ListRemoteConfigSchemas(ctx, namespace, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote config schemas: %w", err)
	}

	return &model.ListResponse[*agentmodel.RemoteConfigSchema]{
		Items:              resourceResp.Items,
		RemainingItemCount: resourceResp.RemainingItemCount,
		Continue:           resourceResp.Continue,
	}, nil
}

// SaveRemoteConfigSchema implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) SaveRemoteConfigSchema(
	ctx context.Context,
	schema *agentmodel.RemoteConfigSchema,
) (*agentmodel.RemoteConfigSchema, error) {
	resource, err := s.persistence.PutRemoteConfigSchema(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to save remote config schema: %w", err)
	}

	return resource, nil
}

// CreateRemoteConfigSchema implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) CreateRemoteConfigSchema(
	ctx context.Context,
	schema *agentmodel.RemoteConfigSchema,
	actor string,
) (*agentmodel.RemoteConfigSchema, error) {
	if schema.Metadata.Name == "" {
		return nil, fmt.Errorf("%w: remote config schema name must not be empty", model.ErrInvalidArgument)
	}

	// Reject creating over an existing schema instead of silently upserting it.
	_, err := s.persistence.GetRemoteConfigSchema(ctx, schema.Metadata.Namespace, schema.Metadata.Name, nil)
	switch {
	case err == nil:
		return nil, fmt.Errorf("%w: remote config schema %q in namespace %q",
			model.ErrResourceAlreadyExist, schema.Metadata.Name, schema.Metadata.Namespace)
	case !errors.Is(err, model.ErrResourceNotExist):
		return nil, fmt.Errorf("check existing remote config schema: %w", err)
	}

	schema.MarkAsCreated(s.clock.Now(), actor)

	created, err := s.persistence.PutRemoteConfigSchema(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote config schema: %w", err)
	}

	return created, nil
}

// UpdateRemoteConfigSchema implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) UpdateRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	schema *agentmodel.RemoteConfigSchema,
) (*agentmodel.RemoteConfigSchema, error) {
	existing, err := s.persistence.GetRemoteConfigSchema(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote config schema for update: %w", err)
	}

	existing.ApplyUpdate(schema)

	updated, err := s.persistence.PutRemoteConfigSchema(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update remote config schema: %w", err)
	}

	return updated, nil
}

// DeleteRemoteConfigSchema implements [agentport.RemoteConfigSchemaUsecase].
func (s *RemoteConfigSchemaService) DeleteRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetRemoteConfigSchema(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get remote config schema for deletion: %w", err)
	}

	resource.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutRemoteConfigSchema(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to mark remote config schema as deleted: %w", err)
	}

	return nil
}
