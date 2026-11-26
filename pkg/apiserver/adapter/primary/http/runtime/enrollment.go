package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

const (
	agentEnrollmentPath     = "/api/runtime/v1/namespaces/:namespace/agent-enrollments/:instanceUID"
	agentEnrollmentResource = "agentenrollments"
	maxEnrollmentBody       = 4096
)

type enrollmentScope struct {
	//nolint:tagliatelle // Keep the runtime API's UID field names consistent.
	NamespaceUID string `json:"namespaceUID"`
	//nolint:tagliatelle // Keep the runtime API's UID field names consistent.
	InstallationUID string `json:"installationUID"`
}

func (c *Controller) enrollAgent(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), reviewTimeout)
	defer cancel()

	key, identity, ok := c.enrollmentRequest(requestContext, ctx, "update")
	if !ok {
		return
	}

	if !enrollmentScopeMatches(ctx, key.NamespaceUID, key.InstallationUID) {
		return
	}

	status := c.verifyEnrollmentIdentity(requestContext, ctx.Param("namespace"), identity)
	if status != http.StatusOK {
		ginutil.ProblemResponse(ctx, status, "Enrollment requires a current ServiceAccount in this namespace")

		return
	}

	err := c.enrollments.Claim(requestContext, key, identity.UID)
	if errors.Is(err, port.ErrAgentEnrollmentConflict) {
		ginutil.ProblemResponse(ctx, http.StatusConflict, "Agent enrollment belongs to another ServiceAccount")

		return
	}

	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Agent enrollment storage unavailable")

		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *Controller) agentEnrollment(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), reviewTimeout)
	defer cancel()

	key, _, ok := c.enrollmentRequest(requestContext, ctx, "get")
	if !ok {
		return
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

func (c *Controller) enrollmentRequest(
	requestContext context.Context, ctx *gin.Context, verb string,
) (port.AgentEnrollmentKey, authenticationv1.UserInfo, bool) {
	instanceUID := ctx.Param("instanceUID")

	parsed, err := uuid.Parse(instanceUID)
	if err != nil || parsed == uuid.Nil || parsed.String() != instanceUID {
		ginutil.ProblemResponse(ctx, http.StatusBadRequest, "Agent instance UID must be a canonical nonzero UUID")

		return port.AgentEnrollmentKey{}, authenticationv1.UserInfo{}, false
	}

	//exhaustruct:ignore
	namespace, identity := c.requestNamespace(requestContext, ctx, authorizationv1.ResourceAttributes{
		Group: permissionGroup, Resource: agentEnrollmentResource, Verb: verb,
		Namespace: ctx.Param("namespace"), Name: instanceUID,
	})
	if namespace == nil {
		return port.AgentEnrollmentKey{}, authenticationv1.UserInfo{}, false
	}

	return port.AgentEnrollmentKey{
		InstallationUID: c.config.InstallationUID, NamespaceUID: string(namespace.UID), InstanceUID: instanceUID,
	}, identity, true
}

func enrollmentScopeMatches(ctx *gin.Context, namespaceUID, installationUID string) bool {
	body := http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxEnrollmentBody)
	defer func() { _ = body.Close() }()

	var scope enrollmentScope

	data, err := io.ReadAll(body)
	if err != nil || json.Unmarshal(data, &scope) != nil || scope.NamespaceUID == "" || scope.InstallationUID == "" {
		ginutil.ProblemResponse(ctx, http.StatusBadRequest, "Enrollment requires namespaceUID and installationUID")

		return false
	}

	if scope.NamespaceUID != namespaceUID || scope.InstallationUID != installationUID {
		ginutil.ProblemResponse(ctx, http.StatusConflict, "Enrollment scope has changed")

		return false
	}

	return true
}

func (c *Controller) verifyEnrollmentIdentity(
	ctx context.Context, namespace string, identity authenticationv1.UserInfo,
) int {
	name, ok := strings.CutPrefix(identity.Username, "system:serviceaccount:"+namespace+":")
	if !ok || identity.UID == "" || len(validation.IsDNS1123Subdomain(name)) != 0 {
		return http.StatusForbidden
	}

	account, err := c.client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return http.StatusForbidden
	}

	if err != nil {
		return http.StatusServiceUnavailable
	}

	if account.DeletionTimestamp != nil || string(account.UID) != identity.UID {
		return http.StatusForbidden
	}

	return http.StatusOK
}
