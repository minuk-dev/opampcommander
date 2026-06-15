package helper

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// ErrInvalidContinueToken is returned when a continue token cannot be parsed as
// a valid offset into an in-memory slice page.
var ErrInvalidContinueToken = errors.New("invalid continue token")

// UUIDPage is one page of a paginated UUID slice, plus the cursor needed to
// resume after it.
type UUIDPage struct {
	// Items are the UUIDs on this page.
	Items []uuid.UUID
	// Continue is the token to pass to fetch the next page, or empty when this
	// is the last page.
	Continue string
	// RemainingItemCount is how many items follow this page.
	RemainingItemCount int64
}

// PaginateUUIDs slices an ordered UUID collection into a single page using the
// offset-based cursor carried in options.Continue. It is the building block for
// paginating an aggregate's associated-agent list (e.g. a host's agents), where
// the full ordered set is already in memory.
//
// The continue token is the start offset of the next page (an integer). A zero
// or negative Limit returns the whole remainder in one page.
func PaginateUUIDs(ids []uuid.UUID, options *model.ListOptions) (UUIDPage, error) {
	total := int64(len(ids))

	var start int64

	if options != nil && options.Continue != "" {
		parsed, err := strconv.ParseInt(options.Continue, 10, 64)
		if err != nil || parsed < 0 {
			return UUIDPage{Items: nil, Continue: "", RemainingItemCount: 0},
				fmt.Errorf("%w: %q", ErrInvalidContinueToken, options.Continue)
		}

		start = parsed
	}

	if start > total {
		start = total
	}

	end := total
	if options != nil && options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}

	continueToken := ""
	if end < total {
		continueToken = strconv.FormatInt(end, 10)
	}

	return UUIDPage{
		Items:              ids[start:end],
		Continue:           continueToken,
		RemainingItemCount: total - end,
	}, nil
}
