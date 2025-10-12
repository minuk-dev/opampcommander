package lifecycle

import (
	"context"
	"log/slog"
	"sync"

	"go.uber.org/fx"
)

// Runner is an interface that defines a runner that can be started with a context.
type Runner interface {
	// Name returns the name of the runner.
	// It's only for debugging & logging purposes.
	Name() string

	// Run starts the runner with the given context.
	Run(ctx context.Context) error
}

// Executor is a struct that schedules and manages the execution of runners.
// It uses a WaitGroup to wait for all runners to finish before stopping.
type Executor struct {
	wg sync.WaitGroup
}

// NewExecutor creates a new Executor instance.
func NewExecutor(
	lifecycle fx.Lifecycle,
	runners []Runner,
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

				go func(runner Runner) {
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
