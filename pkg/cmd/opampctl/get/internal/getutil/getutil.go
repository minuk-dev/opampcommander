// Package getutil preserves per-target failures for get subcommands.
package getutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var errEmptyIdentity = errors.New("identity must not be empty")

// Collect retrieves each requested identity in order. Failures go to stderr and
// are returned alongside successful values, which callers format on stdout.
func Collect[T any](cmd *cobra.Command, kind string, ids []string, get func(string) (*T, error)) ([]T, error) {
	// Lookup failures are operational errors; Cobra's usage would corrupt structured stdout.
	cmd.SilenceUsage = true
	items := make([]T, 0, len(ids))

	var errs []error

	for _, id := range ids {
		var (
			item *T
			err  error
		)
		if strings.TrimSpace(id) == "" {
			err = errEmptyIdentity
		} else {
			item, err = get(id)
		}

		if err != nil {
			err = fmt.Errorf("failed to get %s %q: %w", kind, id, err)
			cmd.PrintErrln(err)
			errs = append(errs, err)

			continue
		}

		items = append(items, *item)
	}

	return items, errors.Join(errs...)
}
