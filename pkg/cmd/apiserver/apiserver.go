package apiserver

import (
	"github.com/minuk-dev/minuk-apiserver/pkg/app"
	"github.com/spf13/cobra"
)

type APIServerCommandOption struct {
	// flags

	// internal
	app *app.Server
}

func NewAPIServerCommand(o APIServerCommandOption) *cobra.Command {
	return &cobra.Command{
		Use:   "apiserver",
		Short: "apiserver",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = o.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func (o *APIServerCommandOption) Prepare(cmd *cobra.Command, args []string) error {
	o.app = app.NewServer()
	return nil
}

func (o *APIServerCommandOption) Run(cmd *cobra.Command, args []string) error {
	return o.app.Run()
}
