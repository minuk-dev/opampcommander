// Package user provides the create user command for opampctl.
package user

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/internal/yamlfile"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrUsernameRequired is returned when neither --username nor --file is given.
var ErrUsernameRequired = errors.New("--username is required (or use --file)")

// CommandOptions contains the options for the create user command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	username      string
	email         string
	password      string
	passwordStdin bool
	formatType    string
	file          string

	// internal state
	client *client.Client
}

// NewCommand creates a new create user command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Create a new user (optionally with a basic-auth password)",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}

	cmd.Flags().StringVar(&options.username, "username", "",
		"Username of the user (required unless --file is used)")
	cmd.Flags().StringVar(&options.email, "email", "", "Email address of the user")
	cmd.Flags().StringVar(&options.password, "password", "",
		"Plaintext password for basic auth. The server stores only a one-way hash.")
	cmd.Flags().BoolVar(&options.passwordStdin, "password-stdin", false,
		"Read the basic-auth password from stdin instead of --password")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a full User YAML definition. When set, individual CLI flags are ignored.")

	return cmd
}

// Prepare prepares the create user command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the create user command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	createRequest, err := opt.buildRequest(cmd.InOrStdin())
	if err != nil {
		return err
	}

	user, err := opt.client.UserService.CreateUser(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	type formattedUser struct {
		UID      string `json:"uid"      short:"UID"      text:"UID"      yaml:"uid"`
		Username string `json:"username" short:"USERNAME" text:"USERNAME" yaml:"username"`
		Email    string `json:"email"    short:"EMAIL"    text:"EMAIL"    yaml:"email"`
	}

	formatted := &formattedUser{
		UID:      user.Metadata.UID,
		Username: user.Spec.Username,
		Email:    user.Spec.Email,
	}

	err = formatter.Format(cmd.OutOrStdout(), formatted, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest(stdin io.Reader) (*v1.User, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.User{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, fmt.Errorf("load user from %s: %w", opt.file, err)
		}

		if req.Kind == "" {
			req.Kind = v1.UserKind
		}

		if req.APIVersion == "" {
			req.APIVersion = v1.APIVersion
		}

		return req, nil
	}

	if opt.username == "" {
		return nil, ErrUsernameRequired
	}

	password, err := opt.resolvePassword(stdin)
	if err != nil {
		return nil, err
	}

	//exhaustruct:ignore
	return &v1.User{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		Spec: v1.UserSpec{
			Email:    opt.email,
			Username: opt.username,
			IsActive: true,
			Password: password,
		},
	}, nil
}

// resolvePassword returns the basic-auth password, reading it from stdin when --password-stdin is set.
func (opt *CommandOptions) resolvePassword(stdin io.Reader) (string, error) {
	if !opt.passwordStdin {
		return opt.password, nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}

	return strings.TrimRight(string(data), "\r\n"), nil
}
