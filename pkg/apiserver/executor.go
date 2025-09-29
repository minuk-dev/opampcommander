package apiserver

import (
	"context"
	"log/slog"
	"sync"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/helper"
)

// Executor is a struct that schedules and manages the execution of runners.
// It uses a WaitGroup to wait for all runners to finish before stopping.
type Executor struct {
	wg sync.WaitGroup
}

// NewExecutor creates a new Executor instance.
func NewExecutor(
	lifecycle fx.Lifecycle,
	runners []helper.Runner,
	logger *slog.Logger,
) *Executor {
	executor := &Executor{
		wg: sync.WaitGroup{},
	}
	executorCtx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			for _, runner := range runners {
				executor.wg.Add(1)

				go func(runner helper.Runner) {
					defer executor.wg.Done()

					err := runner.Run(executorCtx)
					if err != nil {
						logger.Error("Runner error",
							slog.String("runner", runner.Name()),
							slog.String("error", err.Error()),
						)
					}
				}(runner)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			// Wait for all runners to finish
			executor.wg.Wait()

			return nil
		},
	})

	return executor
}
