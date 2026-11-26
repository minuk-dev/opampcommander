package runtime_test

import (
	"cmp"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubetesting "k8s.io/client-go/testing"

	runtimeapi "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/runtime"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

const enrollmentInstance = "abc115aa-730b-4c1d-8638-e2b64eb27b34"

type enrollmentStore struct {
	key               port.AgentEnrollmentKey
	serviceAccountUID string
	lookup            bool
	err               error
}

func (s *enrollmentStore) Claim(_ context.Context, key port.AgentEnrollmentKey, serviceAccountUID string) error {
	s.key = key
	s.serviceAccountUID = serviceAccountUID

	return s.err
}

func (s *enrollmentStore) Lookup(_ context.Context, key port.AgentEnrollmentKey) (string, error) {
	s.key = key
	s.lookup = true

	return "collector-uid", s.err
}

func TestAgentEnrollment(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name            string
		method          string
		username        string
		accountUID      string
		namespaceUID    string
		installationUID string
		deny            bool
		storeError      error
		wantStatus      int
		wantClaim       bool
		wantLookup      bool
	}{
		{name: "allowed", wantStatus: http.StatusNoContent, wantClaim: true},
		{name: "forbidden", deny: true, wantStatus: http.StatusForbidden},
		{name: "not a service account", username: "alice", wantStatus: http.StatusForbidden},
		{name: "foreign service account", username: "system:serviceaccount:other:collector",
			wantStatus: http.StatusForbidden},
		{name: "recreated service account", accountUID: "previous-account-uid", wantStatus: http.StatusForbidden},
		{name: "stale namespace", namespaceUID: "previous-namespace-uid", wantStatus: http.StatusConflict},
		{name: "stale installation", installationUID: "previous-installation-uid", wantStatus: http.StatusConflict},
		{name: "claimed by another service account", storeError: port.ErrAgentEnrollmentConflict,
			wantStatus: http.StatusConflict, wantClaim: true},
		{name: "storage unavailable", storeError: io.ErrUnexpectedEOF,
			wantStatus: http.StatusServiceUnavailable, wantClaim: true},
		{name: "lookup", method: http.MethodGet, wantStatus: http.StatusOK, wantLookup: true},
		{name: "permission denied for lookup", method: http.MethodGet, deny: true, wantStatus: http.StatusForbidden},
		{name: "no enrollment", method: http.MethodGet, storeError: port.ErrAgentEnrollmentNotFound,
			wantStatus: http.StatusNotFound, wantLookup: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			method := cmp.Or(tt.method, http.MethodPut)

			verb := "update"
			if method == http.MethodGet {
				verb = "get"
			}

			config, client := fixture()
			_, err := client.CoreV1().ServiceAccounts("tenant").Create(t.Context(), &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "collector", Namespace: "tenant", UID: "collector-uid"},
			}, metav1.CreateOptions{})
			require.NoError(t, err)

			identity := authenticationv1.UserInfo{
				Username: cmp.Or(tt.username, "system:serviceaccount:tenant:collector"),
				UID:      cmp.Or(tt.accountUID, "collector-uid"),
			}

			client.PrependReactor("create", "tokenreviews", func(kubetesting.Action) (bool, runtime.Object, error) {
				status := authenticationv1.TokenReviewStatus{
					Authenticated: true, Audiences: []string{config.Audience}, User: identity,
				}

				return true, &authenticationv1.TokenReview{Status: status}, nil
			})

			reviewAccess := func(action kubetesting.Action) (bool, runtime.Object, error) {
				create, ok := action.(kubetesting.CreateAction)
				require.True(t, ok)
				review, ok := create.GetObject().(*authorizationv1.SubjectAccessReview)
				require.True(t, ok)
				require.Equal(t, identity.Username, review.Spec.User)
				require.Equal(t, identity.UID, review.Spec.UID)
				require.Equal(t, &authorizationv1.ResourceAttributes{
					Group: "runtime.opampcommander.io", Resource: "agentenrollments", Verb: verb,
					Name: enrollmentInstance, Namespace: "tenant",
				}, review.Spec.ResourceAttributes)

				return true, &authorizationv1.SubjectAccessReview{
					Status: authorizationv1.SubjectAccessReviewStatus{Allowed: !tt.deny},
				}, nil
			}
			client.PrependReactor("create", "subjectaccessreviews", reviewAccess)

			store := &enrollmentStore{err: tt.storeError}
			controller, err := runtimeapi.NewController(config, client, store)
			require.NoError(t, err)

			engine := gin.New()
			controller.Register(engine)

			request := httptest.NewRequestWithContext(t.Context(), method,
				"/api/runtime/v1/namespaces/tenant/agent-enrollments/"+enrollmentInstance,
				strings.NewReader(`{"namespaceUID":"`+cmp.Or(tt.namespaceUID, "namespace-uid")+
					`","installationUID":"`+cmp.Or(tt.installationUID, config.InstallationUID)+`"}`))
			request.Header.Set("Authorization", "Bearer credential")
			request.Header.Set("X-Remote-User", "system:admin")

			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, tt.wantStatus, response.Code, response.Body.String())
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			require.NotContains(t, response.Body.String(), io.ErrUnexpectedEOF.Error())

			if tt.wantClaim || tt.wantLookup {
				require.Equal(t, port.AgentEnrollmentKey{
					InstallationUID: config.InstallationUID, NamespaceUID: "namespace-uid", InstanceUID: enrollmentInstance,
				}, store.key)
			}

			require.Equal(t, tt.wantLookup, store.lookup)
			require.Equal(t, tt.wantClaim, store.serviceAccountUID == "collector-uid")

			if tt.wantStatus == http.StatusOK {
				require.JSONEq(t, `{"instanceUID":"`+enrollmentInstance+`","namespaceUID":"namespace-uid",`+
					`"installationUID":"installation-uid","serviceAccountUID":"collector-uid"}`, response.Body.String())
			}
		})
	}
}
