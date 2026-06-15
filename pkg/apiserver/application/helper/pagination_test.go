package helper_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestPaginateUUIDs(t *testing.T) {
	t.Parallel()

	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = uuid.New()
	}

	t.Run("nil options returns the whole slice in one page", func(t *testing.T) {
		t.Parallel()

		page, err := helper.PaginateUUIDs(ids, nil)
		require.NoError(t, err)
		assert.Equal(t, ids, page.Items)
		assert.Empty(t, page.Continue)
		assert.Equal(t, int64(0), page.RemainingItemCount)
	})

	t.Run("first page with limit", func(t *testing.T) {
		t.Parallel()

		page, err := helper.PaginateUUIDs(ids, &model.ListOptions{Limit: 2})
		require.NoError(t, err)
		assert.Equal(t, ids[:2], page.Items)
		assert.Equal(t, "2", page.Continue)
		assert.Equal(t, int64(3), page.RemainingItemCount)
	})

	t.Run("middle page resumes from continue token", func(t *testing.T) {
		t.Parallel()

		page, err := helper.PaginateUUIDs(ids, &model.ListOptions{Limit: 2, Continue: "2"})
		require.NoError(t, err)
		assert.Equal(t, ids[2:4], page.Items)
		assert.Equal(t, "4", page.Continue)
		assert.Equal(t, int64(1), page.RemainingItemCount)
	})

	t.Run("last page has empty continue token", func(t *testing.T) {
		t.Parallel()

		page, err := helper.PaginateUUIDs(ids, &model.ListOptions{Limit: 2, Continue: "4"})
		require.NoError(t, err)
		assert.Equal(t, ids[4:5], page.Items)
		assert.Empty(t, page.Continue)
		assert.Equal(t, int64(0), page.RemainingItemCount)
	})

	t.Run("offset beyond length returns empty page", func(t *testing.T) {
		t.Parallel()

		page, err := helper.PaginateUUIDs(ids, &model.ListOptions{Continue: "99"})
		require.NoError(t, err)
		assert.Empty(t, page.Items)
		assert.Empty(t, page.Continue)
	})

	t.Run("invalid continue token errors", func(t *testing.T) {
		t.Parallel()

		_, err := helper.PaginateUUIDs(ids, &model.ListOptions{Continue: "abc"})
		require.ErrorIs(t, err, helper.ErrInvalidContinueToken)
	})
}
