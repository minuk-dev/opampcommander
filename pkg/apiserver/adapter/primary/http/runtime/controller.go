// Package runtime provides the Kubernetes-authenticated Commander runtime API.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

const reviewTimeout = 5 * time.Second

var (
	errInvalidConfig = errors.New(
		"runtime API requires audience, installationNamespace, installationName and installationUID",
	)
	errInstallationChanged = errors.New("identity anchor does not match configured installation UID")
)

// Config opts into the native runtime API and pins its installation identity.
type Config struct {
	Enabled               bool   `mapstructure:"enabled"`
	Audience              string `mapstructure:"audience"`
	InstallationNamespace string `mapstructure:"installationNamespace"`
	InstallationName      string `mapstructure:"installationName"`
	InstallationUID       string `mapstructure:"installationUID"`
}

// Controller resolves authorized runtime namespace identities without exposing Agent records.
type Controller struct {
	config Config
	client kubernetes.Interface
}

// NewController validates the configured anchor before serving runtime requests.
// A disabled API does not need Kubernetes credentials.
func NewController(config Config, client kubernetes.Interface) (*Controller, error) {
	if !config.Enabled {
		return nil, nil //nolint:nilnil // Disabled native API intentionally has no controller.
	}

	if config.Audience == "" || config.InstallationNamespace == "" ||
		config.InstallationName == "" || config.InstallationUID == "" {
		return nil, errInvalidConfig
	}

	controller := &Controller{config: config, client: client}

	ctx, cancel := context.WithTimeout(context.Background(), reviewTimeout)
	defer cancel()

	err := controller.checkInstallation(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime installation: %w", err)
	}

	return controller, nil
}

// Register installs routes before the legacy JWT/Casbin middleware is added.
func (c *Controller) Register(engine *gin.Engine) {
	if c != nil {
		engine.GET("/api/runtime/v1/namespaces/:namespace", c.namespace)
	}
}

func (c *Controller) namespace(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), reviewTimeout)
	defer cancel()

	status := c.authorize(requestContext, ctx.Request.Header.Values("Authorization"), ctx.Param("namespace"))
	if status != http.StatusOK {
		if status == http.StatusUnauthorized {
			ctx.Header("WWW-Authenticate", "Bearer")
		}

		ginutil.ProblemResponse(ctx, status, http.StatusText(status))

		return
	}

	err := c.checkInstallation(requestContext)
	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Runtime installation unavailable")

		return
	}

	namespace, err := c.client.CoreV1().Namespaces().Get(requestContext, ctx.Param("namespace"), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ginutil.ProblemResponse(ctx, http.StatusNotFound, "Namespace unavailable")

		return
	}

	if err != nil {
		ginutil.ProblemResponse(ctx, http.StatusServiceUnavailable, "Namespace lookup unavailable")

		return
	}

	if namespace.UID == "" || namespace.DeletionTimestamp != nil ||
		namespace.Labels["opampcommander.io/installation-uid"] != c.config.InstallationUID {
		ginutil.ProblemResponse(ctx, http.StatusForbidden, "Namespace is not assigned to this installation")

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"name": namespace.Name, "namespaceUID": namespace.UID, "installationUID": c.config.InstallationUID,
	})
}

func (c *Controller) authorize(ctx context.Context, headers []string, namespace string) int {
	if len(headers) != 1 {
		return http.StatusUnauthorized
	}

	fields := strings.Fields(headers[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return http.StatusUnauthorized
	}

	//exhaustruct:ignore
	review, err := c.client.AuthenticationV1().TokenReviews().Create(ctx, &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: fields[1], Audiences: []string{c.config.Audience}},
	}, metav1.CreateOptions{})
	if err != nil {
		return http.StatusServiceUnavailable
	}

	identity := review.Status.User
	if !review.Status.Authenticated || review.Status.Error != "" ||
		!slices.Contains(review.Status.Audiences, c.config.Audience) || identity.Username == "" ||
		identity.Username == "system:anonymous" || slices.Contains(identity.Groups, "system:unauthenticated") {
		return http.StatusUnauthorized
	}

	return c.authorizeUser(ctx, identity, namespace)
}

func (c *Controller) authorizeUser(ctx context.Context, identity authenticationv1.UserInfo, namespace string) int {
	extra := make(map[string]authorizationv1.ExtraValue, len(identity.Extra))
	for key, value := range identity.Extra {
		extra[key] = authorizationv1.ExtraValue(value)
	}

	//exhaustruct:ignore
	access, err := c.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authorizationv1.SubjectAccessReview{
		//exhaustruct:ignore
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User: identity.Username, UID: identity.UID, Groups: identity.Groups, Extra: extra,
			//exhaustruct:ignore
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group: "runtime.opampcommander.io", Resource: "namespaces", Verb: "get", Namespace: namespace,
				Name: namespace,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil || access.Status.EvaluationError != "" {
		return http.StatusServiceUnavailable
	}

	if !access.Status.Allowed || access.Status.Denied {
		return http.StatusForbidden
	}

	return http.StatusOK
}

func (c *Controller) checkInstallation(ctx context.Context) error {
	anchor, err := c.client.CoreV1().ConfigMaps(c.config.InstallationNamespace).
		Get(ctx, c.config.InstallationName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get identity anchor: %w", err)
	}

	if string(anchor.UID) != c.config.InstallationUID || anchor.DeletionTimestamp != nil {
		return errInstallationChanged
	}

	return nil
}
