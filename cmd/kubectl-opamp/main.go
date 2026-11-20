// Command kubectl-opamp is the kubectl plugin for Commander.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/kubeplugin"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	err := kubeplugin.NewCommand().ExecuteContext(ctx)

	stop()

	if err != nil {
		os.Exit(1)
	}
}
