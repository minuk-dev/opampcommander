package main

import (
	"fmt"
	"os"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
)

func main() {
	options := apiserver.CommandOption{}
	cmd := apiserver.NewCommand(options)

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apiserver not executed. err=%+v", err)
	}
}
