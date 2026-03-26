// Package permission provides application services for permission management.
package permission

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
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.PermissionManageUsecase = (*Service)(nil)

// Service is a struct that implements the PermissionManageUsecase interface.
type Service struct {
	permissionUsecase domainport.PermissionUsecase
	mapper            *helper.Mapper
	logger            *slog.Logger
}

// New creates a new instance of the Service struct.
func New(permissionUsecase domainport.PermissionUsecase, logger *slog.Logger) *Service {
	return &Service{
		permissionUsecase: permissionUsecase,
		mapper:            helper.NewMapper(),
		logger:            logger,
	}
}

// GetPermission implements [applicationport.PermissionManageUsecase].
func (s *Service) GetPermission(ctx context.Context, uid uuid.UUID) (*v1.Permission, error) {
	permission, err := s.permissionUsecase.GetPermission(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return s.mapper.MapPermissionToAPI(permission), nil
}

// ListPermissions implements [applicationport.PermissionManageUsecase].
func (s *Service) ListPermissions(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Permission], error) {
	response, err := s.permissionUsecase.ListPermissions(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}

	return &v1.ListResponse[v1.Permission]{
		Kind:       v1.PermissionKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(permission *model.Permission, _ int) v1.Permission {
			return *s.mapper.MapPermissionToAPI(permission)
		}),
	}, nil
}
