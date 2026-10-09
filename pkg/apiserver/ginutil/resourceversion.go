package ginutil

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RequireResourceVersion validates a public mutation's revision precondition.
func RequireResourceVersion(ctx *gin.Context, version int64) bool {
	if version <= 0 {
		HandleValidationError(ctx, "resourceVersion", strconv.FormatInt(version, 10),
			fmt.Errorf("%w: re-read the resource before updating or deleting", ErrRequiredParam), false)

		return false
	}

	return true
}

// ParseResourceVersion reads the mandatory delete precondition.
func ParseResourceVersion(ctx *gin.Context) (int64, bool) {
	version, err := ParseInt64(ctx, "resourceVersion", 0)
	if err != nil {
		HandleValidationError(ctx, "resourceVersion", ctx.Query("resourceVersion"), err, false)

		return 0, false
	}

	return version, RequireResourceVersion(ctx, version)
}
