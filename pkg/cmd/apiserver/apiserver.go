package apiserver

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/app"
)

type CommandOption struct {
	// flags
	dbHost string

	// internal
	app *app.Server
}

// NewCommand creates a new apiserver command.
func NewCommand(opt CommandOption) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "apiserver",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opt.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = opt.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opt.dbHost, "db-host", "localhost:2379", "etcd host")

	return cmd
}

func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	opt.app = app.NewServer(app.ServerSettings{
		EtcdHosts: []string{opt.dbHost},
	})

	return nil
}

func (opt *CommandOption) Run(_ *cobra.Command, _ []string) error {
	opt.app.Run()

	return nil
}
