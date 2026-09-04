// Package selectorflags registers the server-side filtering flags shared by the
// `opampctl get` commands and turns them into client list options.
//
// The three flags mirror the query parameters every list endpoint accepts, so a
// filter is answered by the datastore rather than by fetching the collection and
// discarding most of it locally.
package selectorflags

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// ErrUnsupportedLocally is returned when a filtering flag cannot be honoured by a
// listing path that filters in the process rather than in the datastore.
var ErrUnsupportedLocally = errors.New("filter not supported by this listing")

// Flags holds the parsed values of the shared filtering flags.
//
// The zero value registers and applies nothing, so a command that embeds it
// without calling Register still works.
type Flags struct {
	// label is the raw -l/--selector expression.
	label string
	// field is the raw --field-selector expression.
	field string
	// name is the raw --name prefix.
	name string
}

// Register adds -l/--selector, --field-selector and --name to cmd.
//
// Which fields a resource supports is deliberately not repeated here: the server
// owns that list and rejects an unsupported field with an error naming it and
// listing the supported ones, so the help text cannot go stale.
func (f *Flags) Register(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.label, "selector", "l", "",
		"Filter by labels, e.g. -l 'env=prod,tier notin (canary,dev),!deprecated'")
	cmd.Flags().StringVar(&f.field, "field-selector", "",
		"Filter by resource fields, e.g. --field-selector metadata.namespace=prod")
	cmd.Flags().StringVar(&f.name, "name", "", "Filter by a case-sensitive name prefix")
}

// LabelSelector returns the parsed label selector, for the few listing paths that
// have no server-side equivalent and must filter what they fetched.
func (f *Flags) LabelSelector() (selector.LabelSelector, error) {
	parsed, err := selector.ParseLabels(f.label)
	if err != nil {
		return nil, fmt.Errorf("invalid --selector: %w", err)
	}

	return parsed, nil
}

// ListOptions validates the flags and renders them as client list options.
//
// Parsing happens here so a malformed expression is reported before a request is
// made; the expressions themselves are sent verbatim, and the server remains the
// authority on which fields a resource supports.
func (f *Flags) ListOptions() ([]client.ListOption, error) {
	_, err := f.LabelSelector()
	if err != nil {
		return nil, err
	}

	_, err = selector.ParseFields(f.field)
	if err != nil {
		return nil, fmt.Errorf("invalid --field-selector: %w", err)
	}

	var opts []client.ListOption

	if f.label != "" {
		opts = append(opts, client.WithLabelSelector(f.label))
	}

	if f.field != "" {
		opts = append(opts, client.WithFieldSelector(f.field))
	}

	if f.name != "" {
		opts = append(opts, client.WithName(f.name))
	}

	return opts, nil
}

// ConstrainsField reports whether the field selector already constrains the
// given field. A command uses it to drop a default filter that would otherwise
// be ANDed with an explicit, contradicting one and silently return nothing.
//
// A malformed expression constrains nothing; [Flags.ListOptions] reports it.
func (f *Flags) ConstrainsField(field string) bool {
	parsed, err := selector.ParseFields(f.field)
	if err != nil {
		return false
	}

	return slices.Contains(parsed.Fields(), field)
}

// LocalFilter evaluates the filtering flags in the process, for the few listing
// paths that have no server-side equivalent — a group's member agents, for
// instance.
type LocalFilter struct {
	labels selector.LabelSelector
	name   string
}

// LocalFilter parses the flags for local evaluation.
//
// It fails when --field-selector is set: a locally filtered path has no field
// projection to evaluate, and dropping the flag would hand back a list the
// caller would mistake for a narrowed one.
func (f *Flags) LocalFilter() (LocalFilter, error) {
	//exhaustruct:ignore
	var zero LocalFilter

	if f.field != "" {
		return zero, fmt.Errorf("%w: --field-selector is not supported by this listing", ErrUnsupportedLocally)
	}

	labels, err := f.LabelSelector()
	if err != nil {
		return zero, err
	}

	return LocalFilter{labels: labels, name: f.name}, nil
}

// Matches reports whether a resource with the given name and labels satisfies
// the filter.
func (lf LocalFilter) Matches(name string, labels map[string]string) bool {
	if lf.name != "" && !strings.HasPrefix(name, lf.name) {
		return false
	}

	return lf.labels.Matches(labels)
}
