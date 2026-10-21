// Package main is the entry point for the opampctl command line tool.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl"
)

func main() {
	//exhaustruct:ignore
	options := opampctl.CommandOption{}
	cmd := opampctl.NewCommand(options)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	err := cmd.ExecuteContext(ctx)

	stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "opampctl not executed. err=%+v\n", err)
		os.Exit(1)
	}
}
