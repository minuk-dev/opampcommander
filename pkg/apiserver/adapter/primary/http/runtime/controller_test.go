package runtime_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"

	runtimeapi "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/runtime"
)

func fixture() (runtimeapi.Config, *fake.Clientset) {
	return runtimeapi.Config{
		Enabled: true, Audience: "commander", InstallationNamespace: "system",
		InstallationName: "anchor", InstallationUID: "installation-uid",
	}, fake.NewClientset(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "anchor", Namespace: "system", UID: "installation-uid"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant", UID: "namespace-uid",
			Labels: map[string]string{"opampcommander.io/installation-uid": "installation-uid"}}},
	)
}

func TestNewController(t *testing.T) {
	t.Parallel()

	disabled, err := runtimeapi.NewController(runtimeapi.Config{}, nil, nil)
	require.NoError(t, err)
	require.Nil(t, disabled)

	for _, testCase := range []struct {
		name   string
		change func(*runtimeapi.Config)
	}{
		{"missing audience", func(c *runtimeapi.Config) { c.Audience = "" }},
		{"missing namespace", func(c *runtimeapi.Config) { c.InstallationNamespace = "" }},
		{"missing name", func(c *runtimeapi.Config) { c.InstallationName = "" }},
		{"missing UID", func(c *runtimeapi.Config) { c.InstallationUID = "" }},
		{"replaced anchor", func(c *runtimeapi.Config) { c.InstallationUID = "previous-uid" }},
		{"missing anchor", func(c *runtimeapi.Config) { c.InstallationName = "absent" }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config, client := fixture()
			testCase.change(&config)
			_, err := runtimeapi.NewController(config, client, nil)
			require.Error(t, err)
		})
	}
}

func TestNamespaceAuthorization(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		headers     []string
		changeToken func(*authenticationv1.TokenReviewStatus)
		changeSAR   func(*authorizationv1.SubjectAccessReviewStatus)
		failAPI     string
		want        int
	}{
		{name: "allowed", headers: []string{"Bearer credential"}, want: http.StatusOK},
		{name: "missing token", want: http.StatusUnauthorized},
		{name: "basic token", headers: []string{"Basic credential"}, want: http.StatusUnauthorized},
		{name: "duplicate headers", headers: []string{"Bearer credential", "Bearer second"}, want: http.StatusUnauthorized},
		{name: "revoked token", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.Authenticated = false }},
		{name: "wrong audience", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.Audiences = []string{"kubernetes"} }},
		{name: "no audience", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.Audiences = nil }},
		{name: "token error", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.Error = "credential rejected" }},
		{name: "anonymous", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.User.Username = "system:anonymous" }},
		{name: "anonymous group", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.User.Groups = []string{"system:unauthenticated"} }},
		{name: "missing username", headers: []string{"Bearer credential"}, want: http.StatusUnauthorized,
			changeToken: func(s *authenticationv1.TokenReviewStatus) { s.User.Username = "" }},
		{name: "no permission", headers: []string{"Bearer credential"}, want: http.StatusForbidden,
			changeSAR: func(s *authorizationv1.SubjectAccessReviewStatus) { s.Allowed = false }},
		{name: "explicit deny", headers: []string{"Bearer credential"}, want: http.StatusForbidden,
			changeSAR: func(s *authorizationv1.SubjectAccessReviewStatus) { s.Denied = true }},
		{name: "evaluation error", headers: []string{"Bearer credential"}, want: http.StatusServiceUnavailable,
			changeSAR: func(s *authorizationv1.SubjectAccessReviewStatus) { s.EvaluationError = "backend unavailable" }},
		{name: "token API unavailable", headers: []string{"Bearer credential"},
			failAPI: "tokenreviews", want: http.StatusServiceUnavailable},
		{name: "authorization API unavailable", headers: []string{"Bearer credential"},
			failAPI: "subjectaccessreviews", want: http.StatusServiceUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config, client := fixture()
			tokenStatus := authenticationv1.TokenReviewStatus{
				Authenticated: true, Audiences: []string{"commander"},
				User: authenticationv1.UserInfo{Username: "alice", UID: "user-uid", Groups: []string{"team"},
					Extra: map[string]authenticationv1.ExtraValue{"scope": {"reader"}}},
			}
			accessStatus := authorizationv1.SubjectAccessReviewStatus{Allowed: true}

			if testCase.changeToken != nil {
				testCase.changeToken(&tokenStatus)
			}

			if testCase.changeSAR != nil {
				testCase.changeSAR(&accessStatus)
			}

			client.PrependReactor("create", "tokenreviews", func(action kubetesting.Action) (bool, runtime.Object, error) {
				create, ok := action.(kubetesting.CreateAction)
				require.True(t, ok)
				request, ok := create.GetObject().(*authenticationv1.TokenReview)
				require.True(t, ok)
				require.Equal(t, authenticationv1.TokenReviewSpec{Token: "credential", Audiences: []string{"commander"}},
					request.Spec)

				return true, &authenticationv1.TokenReview{Status: tokenStatus}, nil
			})
			client.PrependReactor("create", "subjectaccessreviews",
				func(action kubetesting.Action) (bool, runtime.Object, error) {
					create, ok := action.(kubetesting.CreateAction)
					require.True(t, ok)
					request, ok := create.GetObject().(*authorizationv1.SubjectAccessReview)
					require.True(t, ok)
					require.Equal(t, authorizationv1.SubjectAccessReviewSpec{
						User: "alice", UID: "user-uid", Groups: []string{"team"},
						Extra: map[string]authorizationv1.ExtraValue{"scope": {"reader"}},
						ResourceAttributes: &authorizationv1.ResourceAttributes{
							Group: "runtime.opampcommander.io", Resource: "namespaces", Verb: "get", Namespace: "tenant", Name: "tenant",
						},
					}, request.Spec)

					return true, &authorizationv1.SubjectAccessReview{Status: accessStatus}, nil
				})

			if testCase.failAPI != "" {
				client.PrependReactor("create", testCase.failAPI, func(kubetesting.Action) (bool, runtime.Object, error) {
					return true, nil, io.ErrUnexpectedEOF
				})
			}

			controller, err := runtimeapi.NewController(config, client, nil)
			require.NoError(t, err)

			engine := gin.New()
			controller.Register(engine)

			// Changing an upstream decision must affect the same controller on its next request.
			for range 2 {
				response := request(t, engine, testCase.headers)
				require.Equal(t, testCase.want, response.Code)
				require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
				require.NotContains(t, response.Body.String(), "credential")
				require.NotContains(t, response.Body.String(), io.ErrUnexpectedEOF.Error())

				if response.Code == http.StatusOK {
					require.JSONEq(t, `{"name":"tenant","namespaceUID":"namespace-uid","installationUID":"installation-uid"}`,
						response.Body.String())

					tokenStatus.Authenticated = false
					testCase.want = http.StatusUnauthorized
				}
			}
		})
	}
}

func TestNamespaceIdentityChanges(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		resource string
		object   runtime.Object
		want     int
	}{
		{"replaced anchor", "configmaps", &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{UID: "new"}},
			http.StatusServiceUnavailable},
		{"deleting anchor", "configmaps", &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{UID: "installation-uid",
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time}}}, http.StatusServiceUnavailable},
		{"unassigned", "namespaces", &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{UID: "namespace-uid"}},
			http.StatusForbidden},
		{"foreign installation", "namespaces", &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{UID: "namespace-uid",
			Labels: map[string]string{"opampcommander.io/installation-uid": "other"}}}, http.StatusForbidden},
		{"deleting namespace", "namespaces", &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{UID: "namespace-uid",
			Labels:            map[string]string{"opampcommander.io/installation-uid": "installation-uid"},
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time}}}, http.StatusForbidden},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config, client := fixture()
			controller, err := runtimeapi.NewController(config, client, nil)
			require.NoError(t, err)
			client.PrependReactor("create", "tokenreviews", func(kubetesting.Action) (bool, runtime.Object, error) {
				return true, &authenticationv1.TokenReview{Status: authenticationv1.TokenReviewStatus{
					Authenticated: true, Audiences: []string{"commander"}, User: authenticationv1.UserInfo{Username: "alice"},
				}}, nil
			})
			client.PrependReactor("create", "subjectaccessreviews", func(kubetesting.Action) (bool, runtime.Object, error) {
				return true, &authorizationv1.SubjectAccessReview{
					Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true},
				}, nil
			})

			engine := gin.New()
			controller.Register(engine)
			require.Equal(t, http.StatusOK, request(t, engine, []string{"Bearer credential"}).Code)

			client.PrependReactor("get", testCase.resource, func(kubetesting.Action) (bool, runtime.Object, error) {
				return true, testCase.object, nil
			})
			require.Equal(t, testCase.want, request(t, engine, []string{"Bearer credential"}).Code)
		})
	}
}

func request(t *testing.T, handler http.Handler, headers []string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/runtime/v1/namespaces/tenant", nil)
	for _, value := range headers {
		req.Header.Add("Authorization", value)
	}

	req.Header.Set("X-Remote-User", "system:admin")
	req.Header.Set("X-Remote-Group", "system:masters")
	req.Header.Set("Impersonate-User", "system:admin")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	return response
}
