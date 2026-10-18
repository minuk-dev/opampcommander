package ginutil

import (
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/api"
)

// ReadMergePatch requires the merge-patch media type before reading a patch body.
func ReadMergePatch(ctx *gin.Context) ([]byte, bool) {
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/merge-patch+json" {
		ctx.Header("Accept-Patch", "application/merge-patch+json")
		ctx.AbortWithStatusJSON(http.StatusUnsupportedMediaType, &api.ErrorModel{
			Type: "about:blank", Title: "Unsupported Media Type", Status: http.StatusUnsupportedMediaType,
			Detail: "Use Content-Type: application/merge-patch+json", Instance: ctx.Request.URL.String(), Errors: nil,
		})

		return nil, false
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		HandleValidationError(ctx, "body", "", fmt.Errorf("read patch: %w", err), false)

		return nil, false
	}

	return body, true
}
