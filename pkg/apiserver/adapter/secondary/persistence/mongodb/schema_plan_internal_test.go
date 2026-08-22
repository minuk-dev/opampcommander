package mongodb // the index plan is unexported, and its sharing is the thing under test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TestManagedIndexesIsNotSharedState guards against the index plan going back to a
// package-level variable.
//
// The driver mutates the option builders it is handed. From IndexView.CreateMany:
//
//	if model.Options == nil {
//		model.Options = options.Index()
//	}
//	model.Options.SetName(name)
//
// IndexModel.Options is a pointer, so that SetName writes through to whatever builder
// the caller supplied. A plan held in a package-level variable is therefore shared
// mutable state, and two concurrent EnsureSchema calls race on it — which showed up
// as `-race` failures scattered across unrelated MongoDB tests whenever enough of
// them started their containers at once.
//
// This is asserted structurally rather than by racing two EnsureSchema calls: the
// real window is narrow and buried behind container I/O, so a concurrency test here
// would pass whether or not the plan is shared.
func TestManagedIndexesIsNotSharedState(t *testing.T) {
	t.Parallel()

	first := managedIndexes()
	second := managedIndexes()

	require.Len(t, second, len(first))

	seen := make(map[*mongo.IndexModel]struct{})
	withOptions := 0

	for i := range first {
		require.Equal(t, first[i].collectionName, second[i].collectionName)
		require.Len(t, second[i].indexes, len(first[i].indexes))

		for j := range first[i].indexes {
			firstModel := &first[i].indexes[j]
			secondModel := &second[i].indexes[j]

			assert.NotSame(t, firstModel, secondModel)

			if firstModel.Options == nil {
				continue
			}

			withOptions++

			assert.NotSame(t, firstModel.Options, secondModel.Options,
				"%s index %d shares an option builder between calls; the driver names indexes "+
					"in place, so a shared builder is a data race between two EnsureSchema calls",
				first[i].collectionName, j)

			seen[firstModel] = struct{}{}
		}
	}

	assert.Positive(t, withOptions,
		"the plan carries no option builders at all, so this test would pass vacuously")
}

// TestManagedIndexesSurvivesDriverStyleMutation demonstrates the failure mode the
// separate-plan-per-call rule prevents: naming one plan's indexes, the way the driver
// does, must leave another caller's plan untouched.
func TestManagedIndexesSurvivesDriverStyleMutation(t *testing.T) {
	t.Parallel()

	mine := managedIndexes()
	before := optionCounts(mine)

	// A concurrent EnsureSchema hands its own plan to the driver, which names every
	// index in place.
	theirs := managedIndexes()
	for _, collection := range theirs {
		for i := range collection.indexes {
			if collection.indexes[i].Options != nil {
				collection.indexes[i].Options.SetName("name-the-driver-picked")
			}
		}
	}

	assert.Equal(t, before, optionCounts(mine),
		"another caller's driver call changed the options in this plan, so the two are the same objects")
}

// optionCounts records how many option functions each builder in the plan carries.
// The driver appends one when it names an index, so a growth here is the mutation
// leaking between callers.
func optionCounts(plan []collectionAndIndexes) []int {
	counts := make([]int, 0)

	for _, collection := range plan {
		for _, model := range collection.indexes {
			if model.Options == nil {
				continue
			}

			counts = append(counts, len(model.Options.Opts))
		}
	}

	return counts
}
