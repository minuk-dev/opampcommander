package github_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	github "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/auth/github"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	// memguard starts a single process-lifetime daemon (the Coffer rekeying goroutine) the
	// first time an enclave is created; it never exits by design, so it is not a leak.
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/awnumar/memguard/core.NewCoffer.func1"),
	)
}

const (
	testSigningKey = "test-signing-key"
	testIssuer     = "test"
	loopbackURI    = "http://127.0.0.1:9999/callback"
	// malformedURI passes state parsing (it is only stored, never parsed there) but fails
	// url.Parse in the redirect helpers, exercising their error branches.
	malformedURI = "http://[::1"
)

// errBoom is a transport-level failure used to drive the OAuth2 calls into their error paths.
var errBoom = errors.New("boom")

// roundTripFunc adapts a function to an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// jsonResponse builds a minimal JSON HTTP response bound to the request.
func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

// successClient serves canned OAuth2 token, device-code, and GitHub API responses so the
// whole login flow completes in-process without touching the network.
func successClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/login/oauth/access_token":
			return jsonResponse(req, http.StatusOK,
				`{"access_token":"gho_test","token_type":"bearer","scope":"user:email,read:org"}`), nil
		case "/login/device/code":
			return jsonResponse(req, http.StatusOK,
				`{"device_code":"dc","user_code":"UC-1234",`+
					`"verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`), nil
		case "/user/emails":
			return jsonResponse(req, http.StatusOK,
				`[{"email":"user@example.com","primary":true,"verified":true}]`), nil
		case "/user/orgs":
			return jsonResponse(req, http.StatusOK, `[{"login":"acme"}]`), nil
		default:
			return jsonResponse(req, http.StatusNotFound, `{"message":"not found"}`), nil
		}
	})}
}

// errorClient fails every request at the transport level.
func errorClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errBoom
	})}
}

// provisioningSpy records EnsureUserOnLogin calls.
type provisioningSpy struct {
	calls []applicationport.LoginProvisioning
}

var _ usecase.AuthProvisioningUsecase = (*provisioningSpy)(nil)

func (s *provisioningSpy) EnsureUserOnLogin(_ context.Context, provisioning applicationport.LoginProvisioning) {
	s.calls = append(s.calls, provisioning)
}

// newService builds a real security.Service wired to the given HTTP client.
func newService(t *testing.T, httpClient *http.Client, allowedHosts []string) *security.Service {
	t.Helper()

	//exhaustruct:ignore
	cfg := &security.Config{
		//exhaustruct:ignore
		JWTSettings: security.JWTSettings{
			SigningKey:        testSigningKey,
			Issuer:            testIssuer,
			Expiration:        time.Minute,
			RefreshExpiration: time.Hour,
		},
		//exhaustruct:ignore
		OAuthSettings: &security.OAuthSettings{
			ClientID:             "client-id",
			Secret:               "client-secret",
			CallbackURL:          "http://localhost/callback",
			AllowedRedirectHosts: allowedHosts,
		},
	}

	return security.New(slog.Default(), cfg, httpClient, security.NewPasswordHasher(cfg), inmemory.NewUserRepository())
}

// newController builds a controller and returns it alongside the provisioning spy.
func newController(
	t *testing.T, httpClient *http.Client, allowedHosts []string,
) (*github.Controller, *provisioningSpy) {
	t.Helper()

	spy := &provisioningSpy{}
	controller := github.NewController(slog.Default(), newService(t, httpClient, allowedHosts), spy)

	return controller, spy
}

// newRouter registers the controller routes on a bare gin engine.
func newRouter(controller *github.Controller) *gin.Engine {
	router := gin.New()
	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}

// craftState signs an OAuth2 state JWT with the test signing key, embedding the given
// CLI loopback redirect. Extracted from the real state-signing flow so tests can reach the
// callback branches deterministically.
func craftState(t *testing.T, cliRedirect string) string {
	t.Helper()

	now := time.Now()
	claims := security.OAuthStateClaims{
		CLIRedirect: cliRedirect,
		RegisteredClaims: jwt.RegisteredClaims{
			//exhaustruct:ignore
			Issuer:    testIssuer,
			Subject:   "oauth2_state",
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSigningKey))
	require.NoError(t, err)

	return signed
}

func doGET(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)

	return recorder
}

func TestController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller, _ := newController(t, successClient(), nil)
	require.NotNil(t, controller)

	routes := controller.RoutesInfo()
	require.Len(t, routes, 6)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /auth/github",
		"GET /auth/github/callback",
		"GET /api/v1/auth/github",
		"GET /api/v1/auth/github/authcode",
		"GET /api/v1/auth/github/device",
		"GET /api/v1/auth/github/device/exchange",
	} {
		assert.Contains(t, got, want)
	}
}

func TestController_HTTPAuth(t *testing.T) {
	t.Parallel()

	controller, _ := newController(t, successClient(), nil)

	recorder := doGET(t, newRouter(controller), "/auth/github")

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Location"), "github.com/login/oauth/authorize")
	assert.Contains(t, recorder.Header().Get("Location"), "state=")
}

func TestController_APIAuth(t *testing.T) {
	t.Parallel()

	controller, _ := newController(t, successClient(), nil)

	recorder := doGET(t, newRouter(controller), "/api/v1/auth/github")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, gjson.Get(recorder.Body.String(), "url").String(), "github.com/login/oauth/authorize")
}

func TestController_AuthCodeURL(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 when redirect_uri is missing", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller), "/api/v1/auth/github/authcode")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "redirect_uri is required", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("returns 400 when redirect_uri is not allowed", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/authcode?redirect_uri="+url.QueryEscape("https://evil.example.com/cb"))

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "invalid redirect_uri", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("returns 400 when redirect_uri cannot be parsed", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		// "http://%zz" has an invalid percent-escape in the host, so url.Parse fails.
		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/authcode?redirect_uri="+url.QueryEscape("http://%zz"))

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "invalid redirect_uri", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("returns the auth URL for a loopback redirect", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/authcode?redirect_uri="+url.QueryEscape("http://127.0.0.1:5000/cb"))

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, gjson.Get(recorder.Body.String(), "url").String(), "github.com/login/oauth/authorize")
	})

	t.Run("accepts an allowlisted non-loopback host", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), []string{"web.example.com"})

		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/authcode?redirect_uri="+url.QueryEscape("https://web.example.com/cb"))

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "url").String())
	})
}

func TestController_Callback(t *testing.T) {
	t.Parallel()

	t.Run("returns 500 on an invalid state", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller), "/auth/github/callback?state=not-a-jwt&code=abc")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to generate state", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})

	t.Run("returns tokens and provisions the user on success", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, "")+"&code=abc")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "token").String())

		require.Len(t, spy.calls, 1)
		assert.Equal(t, applicationport.IdentityProviderGitHub, spy.calls[0].Provider)
		assert.Equal(t, "user@example.com", spy.calls[0].Email)
		assert.Equal(t, []string{"acme"}, spy.calls[0].Groups)
	})

	t.Run("redirects to the CLI loopback on success", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, loopbackURI)+"&code=abc")

		require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
		location := recorder.Header().Get("Location")
		assert.True(t, strings.HasPrefix(location, "http://127.0.0.1:9999/callback"))
		assert.Contains(t, location, "token=")
		require.Len(t, spy.calls, 1)
	})

	t.Run("returns 500 when the exchange fails without a CLI redirect", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, errorClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, "")+"&code=abc")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to exchange code for token", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})

	t.Run("redirects the error to the CLI loopback when the exchange fails", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, errorClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, loopbackURI)+"&code=abc")

		require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
		location := recorder.Header().Get("Location")
		assert.True(t, strings.HasPrefix(location, "http://127.0.0.1:9999/callback"))
		assert.Contains(t, location, "error_description=")
	})

	t.Run("returns 500 when the success loopback URI is malformed", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, malformedURI)+"&code=abc")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "invalid cliRedirect URI", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("returns 500 when the error loopback URI is malformed", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, errorClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/auth/github/callback?state="+craftState(t, malformedURI)+"&code=abc")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to exchange code for token", gjson.Get(recorder.Body.String(), "error").String())
	})
}

func TestController_GetDeviceAuth(t *testing.T) {
	t.Parallel()

	t.Run("returns the device authorization response", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller), "/api/v1/auth/github/device")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "dc", gjson.Get(recorder.Body.String(), "deviceCode").String())
		assert.Equal(t, "UC-1234", gjson.Get(recorder.Body.String(), "userCode").String())
	})

	t.Run("returns 500 when device authorization fails", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, errorClient(), nil)

		recorder := doGET(t, newRouter(controller), "/api/v1/auth/github/device")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to initiate device authorization", gjson.Get(recorder.Body.String(), "error").String())
	})
}

func TestController_ExchangeDeviceAuth(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 on an invalid expiry", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/device/exchange?device_code=dc&expiry=not-a-time")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "invalid expiry format", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})

	t.Run("returns tokens and provisions the user without an expiry", func(t *testing.T) {
		t.Parallel()

		controller, spy := newController(t, successClient(), nil)

		recorder := doGET(t, newRouter(controller), "/api/v1/auth/github/device/exchange?device_code=dc")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "token").String())

		require.Len(t, spy.calls, 1)
		assert.Equal(t, applicationport.IdentityProviderGitHub, spy.calls[0].Provider)
		assert.Equal(t, "user@example.com", spy.calls[0].Email)
	})

	t.Run("returns tokens with a valid expiry", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, successClient(), nil)

		expiry := url.QueryEscape(time.Now().Add(time.Hour).Format(time.RFC3339))
		recorder := doGET(t, newRouter(controller),
			"/api/v1/auth/github/device/exchange?device_code=dc&expiry="+expiry)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "token").String())
	})

	t.Run("returns 500 when the device exchange fails", func(t *testing.T) {
		t.Parallel()

		controller, _ := newController(t, errorClient(), nil)

		recorder := doGET(t, newRouter(controller), "/api/v1/auth/github/device/exchange?device_code=dc")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to exchange device code for token", gjson.Get(recorder.Body.String(), "error").String())
	})
}
