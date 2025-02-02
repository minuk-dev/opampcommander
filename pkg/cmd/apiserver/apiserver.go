package apiserver

import (
	"fmt"

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
	// test
	return nil
}

func (opt *CommandOption) Run(_ *cobra.Command, _ []string) error {
	err := opt.app.Run()
	if err != nil {
		return fmt.Errorf("apiserver run failed: %w", err)
	}

	return nil
}
