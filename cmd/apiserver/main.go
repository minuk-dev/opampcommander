package main

import (
	"fmt"
	"os"

	"github.com/minuk-dev/minuk-apiserver/pkg/cmd/apiserver"
)

func main() {
	options := apiserver.APIServerCommandOption{}
	cmd := apiserver.NewAPIServerCommand(options)
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apiserver not executed. err=%+v", err)
	}
}
