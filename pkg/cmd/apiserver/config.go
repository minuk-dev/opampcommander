package apiserver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const redactedValue = "<redacted>"

var errUnsupportedConfigKind = errors.New("unsupported config value kind")

// newConfigCommand creates the `config` command group.
func newConfigCommand(opt *CommandOption) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect the apiserver configuration",
	}

	cmd.AddCommand(newConfigViewCommand(opt))

	return cmd
}

// newConfigViewCommand creates the `config view` command.
func newConfigViewCommand(opt *CommandOption) *cobra.Command {
	var showSecrets bool

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Print the effective configuration",
		Long: "Print the configuration the apiserver would run with, given the same flags, " +
			"config file and environment variables. Secret values are redacted unless --show-secrets is set.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opt.Init(cmd, args)
			if err != nil {
				return fmt.Errorf("failed to initialize command: %w", err)
			}

			return opt.writeConfig(cmd.OutOrStdout(), showSecrets)
		},
	}

	cmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "print secret values instead of redacting them")

	return cmd
}

// writeConfig writes the effective configuration as YAML, preceded by comments
// naming the config file and the environment variables that were applied.
func (opt *CommandOption) writeConfig(out io.Writer, showSecrets bool) error {
	node, err := configNode(reflect.ValueOf(*opt), showSecrets)
	if err != nil {
		return err
	}

	node.HeadComment = strings.Join(opt.sourceComments(), "\n")

	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2) //nolint:mnd

	err = encoder.Encode(node)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// sourceComments describes where the configuration came from.
func (opt *CommandOption) sourceComments() []string {
	var comments []string

	switch {
	case opt.configReadErr == nil:
		comments = append(comments, "config file: "+opt.viper.ConfigFileUsed())
	case opt.configFilename != "":
		comments = append(comments, fmt.Sprintf("config file: %s (not loaded: %v)", opt.configFilename, opt.configReadErr))
	default:
		comments = append(comments, "config file: none")
	}

	envs := opt.appliedEnvs()
	if len(envs) == 0 {
		return append(comments, "environment variables: none")
	}

	comments = append(comments, "environment variables:")
	for _, env := range envs {
		comments = append(comments, "  - "+env)
	}

	return comments
}

// appliedEnvs returns the set environment variables that map to a config key.
func (opt *CommandOption) appliedEnvs() []string {
	var envs []string

	for _, key := range opt.viper.AllKeys() {
		env := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if _, ok := os.LookupEnv(env); ok {
			envs = append(envs, env)
		}
	}

	slices.Sort(envs)

	return slices.Compact(envs)
}

// configNode converts the exported, mapstructure-named fields of a config
// struct into an ordered YAML mapping.
func configNode(value reflect.Value, showSecrets bool) (*yaml.Node, error) {
	//exhaustruct:ignore
	node := &yaml.Node{Kind: yaml.MappingNode}

	for index := range value.NumField() {
		field := value.Type().Field(index)
		if !field.IsExported() {
			continue
		}

		fieldValue := value.Field(index)

		var (
			valueNode *yaml.Node
			err       error
		)

		switch {
		case field.Tag.Get("secret") == "true" && !showSecrets && !fieldValue.IsZero():
			valueNode, err = scalarNode(redactedValue)
		case field.Type.Kind() == reflect.Struct:
			valueNode, err = configNode(fieldValue, showSecrets)
		default:
			valueNode, err = scalarNode(fieldValue.Interface())
		}

		if err != nil {
			return nil, fmt.Errorf("%s: %w", field.Name, err)
		}

		//exhaustruct:ignore
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: configKey(field)}
		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}

func scalarNode(value any) (*yaml.Node, error) {
	if duration, ok := value.(time.Duration); ok {
		value = duration.String()
	}

	switch reflect.ValueOf(value).Kind() { //nolint:exhaustive
	case reflect.Func, reflect.Chan, reflect.Pointer, reflect.Interface, reflect.UnsafePointer:
		return nil, fmt.Errorf("%w: %T", errUnsupportedConfigKind, value)
	}

	var node yaml.Node

	err := node.Encode(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value: %w", err)
	}

	return &node, nil
}

// configKey returns the config key of a field: its mapstructure name, or the
// lower-camel field name (which viper matches case-insensitively).
func configKey(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
	if name != "" {
		return name
	}

	return strings.ToLower(field.Name[:1]) + field.Name[1:]
}
