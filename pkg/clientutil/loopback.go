package clientutil

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/pkg/browser"

	"github.com/minuk-dev/opampcommander/pkg/client"
)

// loopbackTimeout bounds how long opampctl will wait for the browser callback before giving up.
const loopbackTimeout = 5 * time.Minute

// loopbackReadHeaderTimeout caps how long the loopback server will wait for the request headers.
const loopbackReadHeaderTimeout = 5 * time.Second

// loopbackShutdownTimeout caps how long shutdown will wait for in-flight requests.
const loopbackShutdownTimeout = 2 * time.Second

var (
	// errCallbackEmptyToken is returned when the apiserver callback delivered no token.
	errCallbackEmptyToken = errors.New("authentication callback returned no token")
	// errBrowserCallbackTimeout is returned when no callback arrives before the deadline.
	errBrowserCallbackTimeout = errors.New("timed out waiting for browser callback")
	// errAuthCallback wraps errors reported by the apiserver callback.
	errAuthCallback = errors.New("authentication failed")
	// errLoopbackAddr is returned when the listener address is not a TCP address.
	errLoopbackAddr = errors.New("loopback listener returned non-TCP address")
)

// callbackResult carries the data the OAuth callback delivered to the loopback server.
// Exactly one of (access, errMsg) is non-empty.
type callbackResult struct {
	access  string
	refresh string
	errMsg  string
}

// getTokensByGithubBrowser implements OAuth2 authorization-code flow with a localhost
// loopback redirect (RFC 8252). opampctl spawns an ephemeral http server, asks the apiserver
// for an authcode URL bound to that loopback, opens the browser, and captures the tokens
// the apiserver delivers as query parameters on the redirect.
func getTokensByGithubBrowser(cli *client.Client, writer io.Writer, logger *slog.Logger) (authTokens, error) {
	//exhaustruct:ignore
	listenConfig := &net.ListenConfig{}

	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to bind loopback listener: %w", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()

		return authTokens{}, errLoopbackAddr
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", tcpAddr.Port)
	resultCh := make(chan callbackResult, 1)
	server := newLoopbackServer(resultCh)

	go func() {
		_ = server.Serve(listener)
	}()

	defer shutdownLoopback(server, logger)

	authcodeResp, err := cli.AuthService.GetAuthCodeURL(redirectURI)
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to get auth code URL: %w", err)
	}

	_, _ = fmt.Fprintf(writer,
		"Opening browser for GitHub authentication.\n"+
			"If your browser does not open automatically, paste this URL:\n\n  %s\n\n"+
			"Waiting for authentication callback on %s...\n",
		authcodeResp.URL, redirectURI,
	)

	openErr := browser.OpenURL(authcodeResp.URL)
	if openErr != nil {
		logger.Debug("failed to open browser automatically", slog.String("error", openErr.Error()))
	}

	return waitForCallback(resultCh)
}

// waitForCallback blocks until the loopback handler reports a result or the deadline fires.
func waitForCallback(resultCh <-chan callbackResult) (authTokens, error) {
	select {
	case res := <-resultCh:
		if res.errMsg != "" {
			return authTokens{}, fmt.Errorf("%w: %s", errAuthCallback, res.errMsg)
		}

		if res.access == "" {
			return authTokens{}, errCallbackEmptyToken
		}

		return authTokens{access: res.access, refresh: res.refresh}, nil
	case <-time.After(loopbackTimeout):
		return authTokens{}, fmt.Errorf("%w (%s)", errBrowserCallbackTimeout, loopbackTimeout)
	}
}

// newLoopbackServer wires the /callback handler that captures tokens from query params
// and signals the waiting goroutine via resultCh.
func newLoopbackServer(resultCh chan<- callbackResult) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(writer http.ResponseWriter, req *http.Request) {
		params := req.URL.Query()

		res := callbackResult{
			access:  params.Get("token"),
			refresh: params.Get("refreshToken"),
			errMsg:  params.Get("error"),
		}

		if desc := params.Get("error_description"); desc != "" && res.errMsg != "" {
			res.errMsg = res.errMsg + ": " + desc
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")

		if res.errMsg != "" {
			writer.WriteHeader(http.StatusBadRequest)
			//nolint:gosec // The error message is HTML-escaped by loopbackErrorPage.
			_, _ = io.WriteString(writer, loopbackErrorPage(res.errMsg))
		} else {
			_, _ = io.WriteString(writer, loopbackSuccessPage)
		}

		select {
		case resultCh <- res:
		default:
		}
	})

	//exhaustruct:ignore
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: loopbackReadHeaderTimeout,
	}
}

func shutdownLoopback(server *http.Server, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), loopbackShutdownTimeout)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		logger.Debug("failed to shutdown loopback server", slog.String("error", err.Error()))
	}
}

const loopbackSuccessPage = `<!doctype html>
<html><head><meta charset="utf-8"><title>opampctl — signed in</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:3rem;">
<h2>opampctl signed in</h2>
<p>You can close this tab and return to your terminal.</p>
</body></html>`

func loopbackErrorPage(msg string) string {
	return `<!doctype html>
<html><head><meta charset="utf-8"><title>opampctl — sign-in failed</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:3rem;">
<h2>opampctl sign-in failed</h2>
<pre style="white-space:pre-wrap;">` + html.EscapeString(msg) + `</pre>
</body></html>`
}
