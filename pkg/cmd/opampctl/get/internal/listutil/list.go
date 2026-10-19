// Package listutil provides bounded inventory output and resumable namespace traversal.
package listutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

const defaultPageSize = 100

// ErrPagination indicates invalid list controls or a non-advancing server cursor.
var ErrPagination = errors.New("invalid pagination")

// Flags controls bounded lists. The zero value uses one page of 100 items.
type Flags struct {
	PageSize int
	Continue string
	Stream   bool
}

// Register adds the inventory pagination controls.
func (flags *Flags) Register(cmd *cobra.Command) {
	cmd.Flags().IntVar(&flags.PageSize, "page-size", defaultPageSize,
		"Items per request (1-1000); default output is one page")
	cmd.Flags().StringVar(&flags.Continue, "continue", "",
		"Resume cursor printed on stderr; reuse the same namespace and filters")
	cmd.Flags().BoolVar(&flags.Stream, "stream", false,
		"Stream all pages with bounded memory (YAML: one page per document)")
}

type cursor struct {
	Namespace  string `json:"namespace"`
	Namespaces string `json:"namespaces"`
	Resource   string `json:"resource"`
}

func encodeCursor(state cursor, allNamespaces bool) string {
	if !allNamespaces {
		return state.Resource
	}

	if state == (cursor{}) {
		return ""
	}

	// A struct of strings is always JSON-encodable.
	data, _ := json.Marshal(state) //nolint:errchkjson // cursor contains only strings.

	return base64.RawURLEncoding.EncodeToString(data)
}

// Run writes each page before fetching the next. Errors leave partial stdout and
// report a retry cursor on stderr; a failed page may be replayed on resume.
//
//nolint:cyclop,funlen,gocognit // Keep the cursor state transitions beside their output/error handling.
func Run[T any](
	cmd *cobra.Command, cli *client.Client, flags Flags, namespace string, allNamespaces bool,
	format formatter.FormatType,
	fetch func(context.Context, string, ...client.ListOption) (*v1.ListResponse[T], error),
	display func([]T) any,
) (resultErr error) {
	err := flags.validate(cmd, format)
	if err != nil {
		return err
	}

	state, err := newCursor(namespace, flags.Continue, allNamespaces)
	if err != nil {
		return err
	}

	defer func() {
		if resultErr != nil {
			cmd.PrintErrf("List interrupted; stdout may be partial. Retry with --continue=%q (may replay the last page).\n",
				encodeCursor(state, allNamespaces))
		}
	}()

	writer := cmd.OutOrStdout()

	jsonStream := flags.Stream && format == formatter.JSON
	if jsonStream {
		_, err = io.WriteString(writer, "[")
		if err != nil {
			return fmt.Errorf("write array: %w", err)
		}
	}

	first := true
	wrotePage := false

	for {
		err = cmd.Context().Err()
		if err != nil {
			return fmt.Errorf("list canceled: %w", err)
		}

		if allNamespaces && state.Namespace == "" {
			found, err := state.loadNamespace(cmd.Context(), cli)
			if err != nil {
				return err
			}

			if !found {
				break
			}
		}

		resp, err := fetch(cmd.Context(), state.Namespace,
			client.WithLimit(flags.PageSize), client.WithContinueToken(state.Resource))
		if err != nil {
			return fmt.Errorf("list namespace %q: %w", state.Namespace, err)
		}

		if resp.Metadata.RemainingItemCount != 0 &&
			(resp.Metadata.Continue == "" || resp.Metadata.Continue == state.Resource) {
			return fmt.Errorf("%w: resource cursor did not advance", ErrPagination)
		}

		if resp.Items == nil {
			resp.Items = []T{}
		}

		err = writePage(writer, display(resp.Items), format, flags.Stream, first, len(resp.Items))
		if err != nil {
			return err
		}

		first = first && len(resp.Items) == 0
		wrotePage = true
		state.Resource = resp.Metadata.Continue

		more := resp.Metadata.RemainingItemCount != 0
		if !more && allNamespaces && state.Namespaces != "" {
			state.Namespace = ""
			state.Resource = ""
			more = true
		}

		if !more {
			break
		}

		if !flags.Stream {
			cmd.PrintErrf("More results; resume with --continue=%q using the same filters.\n",
				encodeCursor(state, allNamespaces))

			return nil
		}
	}

	if jsonStream {
		_, err = io.WriteString(writer, "]\n")
		if err != nil {
			return fmt.Errorf("close array: %w", err)
		}
	} else if !wrotePage {
		return writePage(writer, display([]T{}), format, flags.Stream, true, 0)
	}

	return nil
}

func writePage(writer io.Writer, items any, format formatter.FormatType, stream, first bool, count int) error {
	if stream && format == formatter.JSON {
		if count == 0 {
			return nil
		}

		data, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("encode page: %w", err)
		}

		separator := ""
		if !first {
			separator = ","
		}

		_, err = fmt.Fprintf(writer, "%s%s", separator, data[1:len(data)-1])
		if err != nil {
			return fmt.Errorf("write page: %w", err)
		}

		return nil
	}

	if stream && format == formatter.YAML {
		_, err := io.WriteString(writer, "---\n")
		if err != nil {
			return fmt.Errorf("write document: %w", err)
		}
	}

	err := formatter.Format(writer, items, format)
	if err != nil {
		return fmt.Errorf("format page: %w", err)
	}

	return nil
}

func newCursor(namespace, token string, allNamespaces bool) (cursor, error) {
	if !allNamespaces {
		return cursor{Namespace: namespace, Namespaces: "", Resource: token}, nil
	}

	state := cursor{}
	if token == "" {
		return state, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return state, fmt.Errorf("%w: invalid all-namespaces cursor: %w", ErrPagination, err)
	}

	err = json.Unmarshal(data, &state)
	if err != nil || (state.Namespace == "" && state.Namespaces == "") {
		return state, fmt.Errorf("%w: invalid all-namespaces cursor", ErrPagination)
	}

	return state, nil
}

func (state *cursor) loadNamespace(ctx context.Context, cli *client.Client) (bool, error) {
	resp, err := cli.NamespaceService.ListNamespaces(ctx,
		client.WithLimit(1), client.WithContinueToken(state.Namespaces))
	if err != nil {
		return false, fmt.Errorf("list namespaces: %w", err)
	}

	if len(resp.Items) == 0 {
		if resp.Metadata.RemainingItemCount != 0 {
			return false, fmt.Errorf("%w: empty namespace page with remaining items", ErrPagination)
		}

		return false, nil
	}

	if resp.Metadata.RemainingItemCount != 0 &&
		(resp.Metadata.Continue == "" || resp.Metadata.Continue == state.Namespaces) {
		return false, fmt.Errorf("%w: namespace cursor did not advance", ErrPagination)
	}

	if resp.Items[0].Metadata.Name == "" {
		return false, fmt.Errorf("%w: empty namespace name", ErrPagination)
	}

	state.Namespace = resp.Items[0].Metadata.Name

	state.Namespaces = ""
	if resp.Metadata.RemainingItemCount != 0 {
		state.Namespaces = resp.Metadata.Continue
	}

	return true, nil
}

func (flags *Flags) validate(cmd *cobra.Command, format formatter.FormatType) error {
	if flags.PageSize == 0 && !cmd.Flags().Changed("page-size") {
		flags.PageSize = defaultPageSize
	}

	if flags.PageSize < 1 || flags.PageSize > 1000 {
		return fmt.Errorf("%w: --page-size must be between 1 and 1000", ErrPagination)
	}

	if format != formatter.SHORT && format != formatter.TEXT && format != formatter.JSON && format != formatter.YAML {
		return fmt.Errorf("%w: unsupported output format %q", ErrPagination, format)
	}

	return nil
}
