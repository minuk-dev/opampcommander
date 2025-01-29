package apiserver

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/minuk-apiserver/pkg/app"
)

type CommandOption struct {
	// flags

	// internal
	app *app.Server
}

func NewCommand(o CommandOption) *cobra.Command {
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

func (o *CommandOption) Prepare(_ *cobra.Command, args []string) error {
	o.app = app.NewServer(app.ServerSettings{})

	return nil
}

func (o *CommandOption) Run(_ *cobra.Command, args []string) error {
	return o.app.Run()
}
