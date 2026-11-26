package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	runtimeapi "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/runtime"
)

func fixture() (runtimeapi.Config, *fake.Clientset) {
	return runtimeapi.Config{
		InstallationNamespace: "system", InstallationName: "anchor", InstallationUID: "installation-uid",
	}, fake.NewClientset(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "anchor", Namespace: "system", UID: "installation-uid"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant", UID: "namespace-uid",
			Labels: map[string]string{"opampcommander.io/installation-uid": "installation-uid"}}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foreign", UID: "foreign-uid"}},
	)
}

func TestNewController(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		change func(*runtimeapi.Config)
	}{
		{"missing namespace", func(c *runtimeapi.Config) { c.InstallationNamespace = "" }},
		{"missing name", func(c *runtimeapi.Config) { c.InstallationName = "" }},
		{"missing UID", func(c *runtimeapi.Config) { c.InstallationUID = "" }},
		{"replaced anchor", func(c *runtimeapi.Config) { c.InstallationUID = "previous-uid" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, client := fixture()
			tt.change(&config)
			_, err := runtimeapi.NewController(config, client, nil)
			require.Error(t, err)
		})
	}
}

func TestNamespaceOwnership(t *testing.T) {
	t.Parallel()

	config, client := fixture()
	controller, err := runtimeapi.NewController(config, client, nil)
	require.NoError(t, err)

	engine := gin.New()
	controller.Register(engine)

	for _, tt := range []struct {
		namespace string
		status    int
	}{
		{"tenant", http.StatusOK}, {"foreign", http.StatusForbidden}, {"absent", http.StatusNotFound},
	} {
		path := "/api/runtime/v1/namespaces/" + tt.namespace
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		require.Equal(t, tt.status, response.Code)
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))

		if tt.status == http.StatusOK {
			require.JSONEq(t, `{"name":"tenant","namespaceUID":"namespace-uid","installationUID":"installation-uid"}`,
				response.Body.String())
		}
	}

	require.NoError(t, client.CoreV1().ConfigMaps("system").Delete(t.Context(), "anchor", metav1.DeleteOptions{}))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/runtime/v1/namespaces/tenant", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
