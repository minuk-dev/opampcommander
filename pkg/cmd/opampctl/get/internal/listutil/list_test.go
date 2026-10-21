package listutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/listutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

type item struct {
	Name string `json:"name" short:"Name" text:"Name" yaml:"name"`
}

func display(items []item) any { return items }

func TestRunPages(t *testing.T) {
	t.Parallel()

	for _, format := range []formatter.FormatType{formatter.JSON, formatter.YAML, formatter.SHORT, formatter.TEXT} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			var out, diagnostics bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())
			cmd.SetOut(&out)
			cmd.SetErr(&diagnostics)

			calls := 0
			fetch := func(ctx context.Context, namespace string, opts ...client.ListOption) (*v1.ListResponse[item], error) {
				calls++

				assert.Equal(t, "team", namespace)

				if calls > 1 {
					assert.Contains(t, out.String(), "first", "output precedes the next request")
				}
				// Exercise option forwarding through the real client.
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "2", r.URL.Query().Get("limit"))

					if calls == 1 {
						assert.Empty(t, r.URL.Query().Get("continue"))
					} else {
						assert.Equal(t, "next", r.URL.Query().Get("continue"))
					}

					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"items":[]}`)
				}))
				_, err := client.New(server.URL).AgentService.ListAgents(ctx, namespace, opts...)
				server.Close()

				if err != nil {
					return nil, fmt.Errorf("list agents: %w", err)
				}

				name, remaining := "first", int64(1)
				if calls == 2 {
					name, remaining = "last", 0
				}

				return &v1.ListResponse[item]{Items: []item{{Name: name}}, Metadata: v1.ListMeta{
					Continue: "next", RemainingItemCount: remaining,
				}}, nil
			}
			require.NoError(t, listutil.Run(cmd, nil, listutil.Flags{PageSize: 2, Stream: true},
				"team", false, format, fetch, display))
			require.Equal(t, 2, calls)
			assert.Empty(t, diagnostics.String())

			switch format {
			case formatter.JSON:
				var items []item
				require.NoError(t, json.Unmarshal(out.Bytes(), &items))
				assert.Equal(t, []item{{Name: "first"}, {Name: "last"}}, items)
			case formatter.YAML:
				decoder := yaml.NewDecoder(&out)

				for _, name := range []string{"first", "last"} {
					var page []item
					require.NoError(t, decoder.Decode(&page))
					assert.Equal(t, []item{{Name: name}}, page)
				}

				var extra any
				require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
			case formatter.SHORT, formatter.TEXT:
				assert.Contains(t, out.String(), "last")
			}
		})
	}
}

func TestRunBoundedResumeAcrossNamespaces(t *testing.T) {
	t.Parallel()

	var namespaceCalls []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespaceCalls = append(namespaceCalls, r.URL.Query().Get("continue"))
		assert.Equal(t, "1", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("continue") == "" {
			_, _ = io.WriteString(w, `{"items":[{"metadata":{"name":"one"}}],
 "metadata":{"continue":"ns-two","remainingItemCount":1}}`)
		} else {
			_, _ = io.WriteString(w, `{"items":[{"metadata":{"name":"two"}}],
 "metadata":{"continue":"end","remainingItemCount":0}}`)
		}
	}))
	defer server.Close()

	cli := client.New(server.URL)

	var visited []string

	fetch := func(_ context.Context, namespace string, _ ...client.ListOption) (*v1.ListResponse[item], error) {
		visited = append(visited, namespace)

		return &v1.ListResponse[item]{Items: []item{{Name: namespace}}}, nil
	}

	var out, diagnostics bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	require.NoError(t, listutil.Run(cmd, cli, listutil.Flags{}, "", true, formatter.JSON, fetch, display))
	assert.Equal(t, []string{"one"}, visited)

	const prefix = "--continue="

	_, rest, _ := strings.Cut(diagnostics.String(), prefix)
	resume, _, _ := strings.Cut(rest, " using")
	token, err := strconv.Unquote(resume)
	require.NoError(t, err)
	out.Reset()
	diagnostics.Reset()
	require.NoError(t, listutil.Run(cmd, cli, listutil.Flags{Continue: token}, "", true, formatter.JSON, fetch, display))
	assert.Equal(t, []string{"one", "two"}, visited)
	assert.Equal(t, []string{"", "ns-two"}, namespaceCalls)
	assert.Empty(t, diagnostics.String())
}

func TestRunStopsWithRetryCursor(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name          string
		cancel, stuck bool
	}{
		{name: "request failure"}, {name: "cancellation", cancel: true}, {name: "stuck cursor", stuck: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			cmd := &cobra.Command{}
			cmd.SetContext(ctx)

			var out, diagnostics bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&diagnostics)

			calls := 0
			failure := io.ErrUnexpectedEOF
			fetch := func(context.Context, string, ...client.ListOption) (*v1.ListResponse[item], error) {
				calls++
				if calls == 2 && !test.stuck {
					return nil, failure
				}

				if test.cancel {
					cancel()
				}

				return &v1.ListResponse[item]{Items: []item{{Name: "first"}}, Metadata: v1.ListMeta{
					Continue: "next", RemainingItemCount: 1,
				}}, nil
			}
			err := listutil.Run(cmd, nil, listutil.Flags{Stream: true}, "team", false, formatter.JSON, fetch, display)
			require.Error(t, err)

			if test.cancel {
				require.ErrorIs(t, err, context.Canceled)
				assert.Equal(t, 1, calls)
			}

			assert.Contains(t, diagnostics.String(), `--continue="next"`)
			assert.Contains(t, diagnostics.String(), "stdout may be partial")
			assert.Contains(t, out.String(), "first")
		})
	}
}

func TestRunRejectsInvalidControlsBeforeFetching(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name          string
		flags         listutil.Flags
		allNamespaces bool
	}{
		{name: "negative size", flags: listutil.Flags{PageSize: -1}},
		{name: "oversized page", flags: listutil.Flags{PageSize: 1001}},
		{name: "invalid namespace cursor", flags: listutil.Flags{Continue: "not-a-cursor"}, allNamespaces: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())
			cmd.SetErr(io.Discard)

			fetch := func(context.Context, string, ...client.ListOption) (*v1.ListResponse[item], error) {
				t.Fatal("invalid controls must not send a request")

				return nil, io.ErrUnexpectedEOF
			}
			err := listutil.Run(cmd, nil, test.flags, "team", test.allNamespaces, formatter.JSON, fetch, display)
			require.ErrorIs(t, err, listutil.ErrPagination)
		})
	}
}

//nolint:paralleltest // Heap measurements must exclude allocations from parallel tests.
func TestRunLargeInventoryUsesBoundedMemory(t *testing.T) {
	// This test samples the live heap in isolation; parallel cases would add unrelated allocations.
	const total, size = 300000, 100

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	runtime.GC()

	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	count := 0
	fetch := func(context.Context, string, ...client.ListOption) (*v1.ListResponse[item], error) {
		if count%50000 == 0 {
			runtime.GC()

			var current runtime.MemStats
			runtime.ReadMemStats(&current)
			t.Logf("items=%d retained heap delta=%d bytes", count, int64(current.HeapAlloc)-int64(baseline.HeapAlloc))
			require.Less(t, int64(current.HeapAlloc)-int64(baseline.HeapAlloc), int64(16<<20),
				"live memory must not grow with inventory")
		}

		items := make([]item, size)
		for index := range items {
			items[index].Name = fmt.Sprintf("%d-%s", count+index, strings.Repeat("x", 1024))
		}

		count += size

		return &v1.ListResponse[item]{Items: items, Metadata: v1.ListMeta{
			Continue: strconv.Itoa(count), RemainingItemCount: int64(total - count),
		}}, nil
	}
	require.NoError(t, listutil.Run(cmd, nil, listutil.Flags{Stream: true}, "team", false, formatter.JSON, fetch, display))
	require.Equal(t, total, count)
}
