//nolint:testpackage // Verify the private store clones only the page.
package inmemory

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestStoreListClonesOnlyPage(t *testing.T) {
	t.Parallel()

	clones := 0

	store := newStore[int](func(v int) int {
		clones++

		return v
	}, nil, nil, hasNoLabels)
	for i := range 100_001 {
		store.put(i, i)
	}

	clones = 0
	response, err := store.list(&model.ListOptions{Limit: 50}, nil)
	require.NoError(t, err)
	require.Len(t, response.Items, 50)
	require.Equal(t, int64(99_951), response.RemainingItemCount)
	require.Equal(t, 50, clones)

	for i, value := range response.Items {
		require.Equal(t, i, value)
	}

	next, err := store.list(&model.ListOptions{Limit: 50, Continue: response.Continue}, nil)
	require.NoError(t, err)
	require.Equal(t, 100, clones)
	require.Equal(t, 50, next.Items[0])
}

func TestStoreListPageOrdering(t *testing.T) {
	t.Parallel()

	for _, limit := range []int64{1, 50, 1_000} {
		t.Run(strconv.FormatInt(limit, 10), func(t *testing.T) {
			t.Parallel()

			clones := 0

			store := newStore[int](func(v int) int {
				clones++

				return v
			}, nil, nil, hasNoLabels)
			for i := range 1_001 {
				store.put(i, i)
			}

			clones = 0
			response, err := store.list(&model.ListOptions{Limit: limit, Continue: "100"}, func(v int) bool {
				return v%2 == 0
			})
			require.NoError(t, err)

			const matches = 451 // Even values from 100 through 1,000.

			size := int(min(limit, matches))
			require.Len(t, response.Items, size)
			require.Equal(t, size, clones)
			require.Equal(t, int64(matches-size), response.RemainingItemCount)

			for i, value := range response.Items {
				require.Equal(t, 100+2*i, value)
			}

			require.Equal(t, strconv.Itoa(response.Items[size-1]+1), response.Continue)
		})
	}
}

func BenchmarkStoreListLargePage(b *testing.B) {
	for _, size := range []int{10_000, 40_000, 160_000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			store := newStore[int](func(v int) int { return v }, nil, nil, hasNoLabels)
			for i := range size {
				store.put(i, i)
			}

			b.ReportAllocs()

			for b.Loop() {
				_, err := store.list(&model.ListOptions{Limit: int64(size)}, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
