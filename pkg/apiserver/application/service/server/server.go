// Package server provides the implementation of the ServerManageUsecase interface.
package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ applicationport.ServerManageUsecase = (*Service)(nil)

// Service is a struct that implements the ServerManageUsecase interface.
type Service struct {
	logger        *slog.Logger
	serverUsecase agentport.ServerUsecase
}

// New creates a new instance of the Service struct.
func New(
	serverUsecase agentport.ServerUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:        logger,
		serverUsecase: serverUsecase,
	}
}

// ListServers lists all alive servers.
func (s *Service) ListServers(ctx context.Context) (*v1.ListResponse[v1.Server], error) {
	servers, err := s.serverUsecase.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	return v1.NewServerListResponse(
		lo.Map(servers, func(server *agentmodel.Server, _ int) v1.Server {
			return v1.Server{
				ID:              server.ID,
				LastHeartbeatAt: v1.NewTime(server.LastHeartbeatAt),
				Conditions:      mapConditionsToAPI(server.Conditions),
			}
		}),
		v1.ListMeta{
			RemainingItemCount: 0,
			Continue:           "",
		},
	), nil
}

// mapConditionsToAPI converts domain server conditions to API conditions.
func mapConditionsToAPI(conditions []model.Condition) []v1.ServerCondition {
	if len(conditions) == 0 {
		return nil
	}

	apiConditions := make([]v1.ServerCondition, len(conditions))
	for i, condition := range conditions {
		apiConditions[i] = v1.ServerCondition{
			Type:               v1.ServerConditionType(condition.Type),
			LastTransitionTime: v1.NewTime(condition.LastTransitionTime),
			Status:             v1.ServerConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return apiConditions
}
