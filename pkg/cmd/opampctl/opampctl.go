package opampctl

import "github.com/spf13/cobra"

type CommandOption struct {
	Endpoint string
}

func NewCommand(options CommandOption) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "opampctl",
		Short: "opampctl",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = options.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&options.Endpoint, "endpoint", "localhost:8080", "opampcommander endpoint")

	return cmd
}

func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	return nil
}

func (opt *CommandOption) Run(_ *cobra.Command, _ []string) error {
	return nil
}
