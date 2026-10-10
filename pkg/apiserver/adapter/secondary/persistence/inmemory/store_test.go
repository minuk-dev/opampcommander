//nolint:testpackage // Verify the private store clones only the page.
package inmemory

import (
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
