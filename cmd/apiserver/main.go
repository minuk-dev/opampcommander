// Package main provides the entry point for the opampcommander apiserver.
package main

import (
	"fmt"
	"os"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
)

// @title OpAMP Commander API Server
// @version 1.0
// @description	This is the API server for OpAMP Commander, providing endpoints for managing OpAMP agents.
// @termsOfService http://swagger.io/terms/
func main() {
	//exhaustruct:ignore
	options := apiserver.CommandOption{}
	cmd := apiserver.NewCommand(options)

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apiserver not executed. err=%+v", err)
	}
}
