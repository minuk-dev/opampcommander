// Package main is the entry point for the opampctl command line tool.
package main

import (
	"fmt"
	"os"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl"
)

func main() {
	//exhaustruct:ignore
	options := opampctl.CommandOption{}
	cmd := opampctl.NewCommand(options)

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "opampctl not executed. err=%+v", err)
	}
}
