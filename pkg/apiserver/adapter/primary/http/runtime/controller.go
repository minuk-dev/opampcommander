// Package runtime serves installation-scoped runtime reads.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

const lookupTimeout = 5 * time.Second

var (
	errInvalidInstallation = errors.New(
		"installation requires installationNamespace, installationName and installationUID",
	)
	errInstallationChanged = errors.New("identity anchor does not match configured installation UID")
)

// Config pins the Kubernetes installation identity.
type Config struct {
	InstallationNamespace string `mapstructure:"installationNamespace"`
	InstallationName      string `mapstructure:"installationName"`
	InstallationUID       string `mapstructure:"installationUID"`
}

// Configured reports whether installation identity settings were supplied.
func (c Config) Configured() bool {
	return c.InstallationNamespace != "" || c.InstallationName != "" || c.InstallationUID != ""
}

// Controller resolves installation-owned namespaces and Agent bindings.
type Controller struct {
	config      Config
	client      kubernetes.Interface
	enrollments port.AgentEnrollmentStore
}

// NewController checks the live installation anchor before serving runtime requests.
func NewController(
	config Config, client kubernetes.Interface, enrollments port.AgentEnrollmentStore,
) (*Controller, error) {
	err := VerifyInstallation(config, client)
	if err != nil {
		return nil, fmt.Errorf("runtime installation: %w", err)
	}

	return &Controller{config: config, client: client, enrollments: enrollments}, nil
}

// Register installs the internal runtime read endpoints.
func (c *Controller) Register(engine *gin.Engine) {
	if c == nil {
		return
	}

	engine.GET("/api/runtime/v1/namespaces/:namespace", c.namespace)

	if c.enrollments != nil {
		engine.GET(agentEnrollmentPath, c.agentEnrollment)
	}
}

func (c *Controller) namespace(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), lookupTimeout)
	defer cancel()

	namespace := c.requestNamespace(requestContext, ctx)
	if namespace == nil {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"name": namespace.Name, "namespaceUID": namespace.UID, "installationUID": c.config.InstallationUID,
	})
}

func (c *Controller) requestNamespace(requestContext context.Context, ctx *gin.Context) *corev1.Namespace {
	err := checkInstallation(requestContext, c.config, c.client)
	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Runtime installation unavailable")

		return nil
	}

	namespace, err := c.client.CoreV1().Namespaces().Get(requestContext, ctx.Param("namespace"), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ginutil.ProblemResponse(ctx, http.StatusNotFound, "Namespace unavailable")

		return nil
	}

	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Namespace lookup unavailable")

		return nil
	}

	if namespace.UID == "" || namespace.DeletionTimestamp != nil ||
		namespace.Labels["opampcommander.io/installation-uid"] != c.config.InstallationUID {
		ginutil.ProblemResponse(ctx, http.StatusForbidden, "Namespace is not assigned to this installation")

		return nil
	}

	return namespace
}

// VerifyInstallation checks that the configured identity matches the live anchor.
func VerifyInstallation(config Config, client kubernetes.Interface) error {
	if config.InstallationNamespace == "" || config.InstallationName == "" || config.InstallationUID == "" {
		return errInvalidInstallation
	}

	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	return checkInstallation(ctx, config, client)
}

func checkInstallation(ctx context.Context, config Config, client kubernetes.Interface) error {
	anchor, err := client.CoreV1().ConfigMaps(config.InstallationNamespace).
		Get(ctx, config.InstallationName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get identity anchor: %w", err)
	}

	if string(anchor.UID) != config.InstallationUID || anchor.DeletionTimestamp != nil {
		return errInstallationChanged
	}

	return nil
}
