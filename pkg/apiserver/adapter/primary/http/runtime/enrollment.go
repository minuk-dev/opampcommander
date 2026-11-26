package runtime

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

const agentEnrollmentPath = "/api/runtime/v1/namespaces/:namespace/agent-enrollments/:instanceUID"

func (c *Controller) agentEnrollment(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), lookupTimeout)
	defer cancel()

	instanceUID := ctx.Param("instanceUID")

	parsed, err := uuid.Parse(instanceUID)
	if err != nil || parsed == uuid.Nil || parsed.String() != instanceUID {
		ginutil.ProblemResponse(ctx, http.StatusBadRequest, "Agent instance UID must be a canonical nonzero UUID")

		return
	}

	namespace := c.requestNamespace(requestContext, ctx)
	if namespace == nil {
		return
	}

	key := port.AgentEnrollmentKey{
		InstallationUID: c.config.InstallationUID, NamespaceUID: string(namespace.UID), InstanceUID: instanceUID,
	}

	owner, err := c.enrollments.Lookup(requestContext, key)
	if errors.Is(err, port.ErrAgentEnrollmentNotFound) {
		ginutil.ProblemResponse(ctx, http.StatusNotFound, "Agent enrollment not found")

		return
	}

	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Agent enrollment storage unavailable")

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"instanceUID": key.InstanceUID, "namespaceUID": key.NamespaceUID, "installationUID": key.InstallationUID,
		"serviceAccountUID": owner,
	})
}
