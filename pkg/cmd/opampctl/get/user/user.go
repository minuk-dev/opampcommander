// Package user provides the command to get user information.
package user

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrUnsupportedFormatType is returned when an unsupported format type is specified.
	ErrUnsupportedFormatType = errors.New("unsupported format type")
)

// CommandOptions contains the options for the user command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	includeDeleted bool

	// internal
	client *client.Client
}

// NewCommand creates a new user command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Get user information",
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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")
	cmd.Flags().BoolVar(&options.includeDeleted, "include-deleted", false, "Include soft-deleted users")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return opt.List(cmd)
	}

	return opt.Get(cmd, args)
}

// ItemForCLI is a struct that represents a user item for display.
type ItemForCLI struct {
	UID      string `short:"UID"      text:"UID"`
	Email    string `short:"EMAIL"    text:"EMAIL"`
	Username string `short:"USERNAME" text:"USERNAME"`
	IsActive bool   `short:"ACTIVE"   text:"ACTIVE"`
}

// List retrieves the list of users.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	listOpts := []client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}

	users, err := clientutil.ListUserFully(cmd.Context(), opt.client, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	switch formatType := formatter.FormatType(opt.formatType); formatType {
	case formatter.SHORT, formatter.TEXT:
		displayedUsers := lo.Map(users, func(user v1.User, _ int) ItemForCLI {
			return toItemForCLI(user)
		})

		err = formatter.Format(cmd.OutOrStdout(), displayedUsers, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), users, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrUnsupportedFormatType)
	}

	if err != nil {
		return fmt.Errorf("failed to format users: %w", err)
	}

	return nil
}

// Get retrieves user information for the given user UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	type userWithErr struct {
		User *v1.User
		Err  error
	}

	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	results := lo.Map(ids, func(id string, _ int) userWithErr {
		user, err := opt.client.UserService.GetUser(cmd.Context(), id, getOpts...)

		return userWithErr{
			User: user,
			Err:  err,
		}
	})

	users := lo.Filter(results, func(result userWithErr, _ int) bool {
		return result.Err == nil
	})
	if len(users) == 0 {
		cmd.Println("No users found or all specified users could not be retrieved.")

		return nil
	}

	displayedUsers := lo.Map(users, func(result userWithErr, _ int) ItemForCLI {
		return toItemForCLI(*result.User)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedUsers, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format users: %w", err)
	}

	return nil
}

func toItemForCLI(user v1.User) ItemForCLI {
	return ItemForCLI{
		UID:      user.Metadata.UID,
		Email:    user.Spec.Email,
		Username: user.Spec.Username,
		IsActive: user.Spec.IsActive,
	}
}
