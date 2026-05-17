package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ensureDefaultNamespace creates the "default" namespace if it does not exist.
func ensureDefaultNamespace(
	ctx context.Context,
	namespaceUsecase agentport.NamespaceUsecase,
	clk clock.Clock,
	logger *slog.Logger,
) error {
	_, err := namespaceUsecase.GetNamespace(
		ctx, agentmodel.DefaultNamespaceName, nil,
	)
	if err == nil {
		return nil
	}

	if !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check default namespace: %w", err)
	}

	logger.Info("creating default namespace")

	ns := agentmodel.NewNamespace(agentmodel.DefaultNamespaceName)
	ns.MarkAsCreated(clk.Now(), "system")

	_, err = namespaceUsecase.SaveNamespace(ctx, ns)
	if err != nil {
		return fmt.Errorf("create default namespace: %w", err)
	}

	return nil
}

// registerDefaultNamespaceHook registers a lifecycle hook to ensure
// the default namespace exists on startup.
func registerDefaultNamespaceHook(
	lifecycle fx.Lifecycle,
	namespaceUsecase agentport.NamespaceUsecase,
	logger *slog.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return ensureDefaultNamespace(
				ctx,
				namespaceUsecase,
				clock.NewRealClock(),
				logger,
			)
		},
		OnStop: nil,
	})
}
