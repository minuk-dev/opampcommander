// Package deleteutil provides shared helpers for the 'opampctl delete' subcommands,
// so each resource command only supplies its per-identifier delete call and shares
// one consistent success/failure summary.
package deleteutil

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// Run deletes each identifier via del and prints a standard summary. kind is the
// singular resource noun used in messages (e.g. "agent", "agentgroup"). It never
// returns an error itself: per-identifier failures are reported on stderr so a
// partial batch still deletes what it can.
func Run(cmd *cobra.Command, kind string, ids []string, del func(id string) error) {
	type result struct {
		id  string
		err error
	}

	results := lo.Map(ids, func(id string, _ int) result {
		return result{id: id, err: del(id)}
	})

	deleted := lo.FilterMap(results, func(r result, _ int) (string, bool) {
		return r.id, r.err == nil
	})

	cmd.Printf("Successfully deleted %d %s(s): %s\n", len(deleted), kind, strings.Join(deleted, ", "))

	failed := lo.Filter(results, func(r result, _ int) bool {
		return r.err != nil
	})
	if len(failed) > 0 {
		messages := lo.Map(failed, func(r result, _ int) string {
			return fmt.Sprintf("%s: %v", r.id, r.err)
		})
		cmd.PrintErrf("Failed to delete %d %s(s): %s\n", len(failed), kind, strings.Join(messages, "; "))
	}
}
