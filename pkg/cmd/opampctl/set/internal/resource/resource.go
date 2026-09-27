// Package resource parses kubectl-style resource arguments for set commands.
package resource

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// MinArgs is the argument count for TYPE/NAME VALUE.
	MinArgs = 2
	// MaxArgs is the argument count for TYPE NAME VALUE.
	MaxArgs = 3
)

// ErrInvalidArguments indicates an invalid set target or value.
var ErrInvalidArguments = errors.New("invalid resource arguments")

// Parse accepts either TYPE/NAME VALUE or TYPE NAME VALUE.
func Parse(args []string, kind string) (string, string, error) {
	var name, value string

	switch len(args) {
	case MinArgs:
		prefix := kind + "/"
		if parsedName, ok := strings.CutPrefix(args[0], prefix); ok {
			name, value = parsedName, args[1]
		}
	case MaxArgs:
		if args[0] == kind {
			name, value = args[1], args[2]
		}
	}

	if name == "" || value == "" {
		return "", "", fmt.Errorf("%w: expected %s/NAME VALUE or %s NAME VALUE",
			ErrInvalidArguments, kind, kind)
	}

	return name, value, nil
}
