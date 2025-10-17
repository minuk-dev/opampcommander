# OAuth HTTP Tracing Implementation

## Overview
Added OpenTelemetry tracing support for HTTP calls made during OAuth authentication flows, including GitHub OAuth and device authorization flows.

## Changes Made

### 1. Updated `internal/security/security.go`
- Added imports for `net/http`, `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, and `go.opentelemetry.io/otel/trace`
- Added `tracerProvider` and `httpClient` fields to the `Service` struct
- Modified `New()` constructor to:
  - Accept `trace.TracerProvider` as a parameter (injected by fx DI container)
  - Create an HTTP client wrapped with `otelhttp.NewTransport()` for automatic tracing
  - Store both the tracer provider and HTTP client in the service

### 2. Updated `internal/security/oauth2.go`
- Added import for `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- Modified `Exchange()` function to:
  - Inject the traced HTTP client into the context using `context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)`
  - Wrap the GitHub API client's transport with `otelhttp.NewTransport()` for tracing GitHub API calls
- Modified `DeviceAuth()` function to:
  - Inject the traced HTTP client into the context
- Modified `ExchangeDeviceAuth()` function to:
  - Inject the traced HTTP client into the context
  - Wrap the GitHub API client's transport with `otelhttp.NewTransport()` for tracing GitHub API calls

## How It Works

### HTTP Call Tracing Flow

1. **OAuth Token Exchange**: When exchanging OAuth codes for tokens, the HTTP client with OpenTelemetry instrumentation is injected into the context. This ensures all HTTP requests to the OAuth provider (e.g., GitHub) are traced.

2. **GitHub API Calls**: After obtaining the OAuth token, when making API calls to GitHub (e.g., to fetch user emails), the HTTP client's transport is wrapped with `otelhttp.NewTransport()` to trace these API calls as well.

3. **Trace Propagation**: The OpenTelemetry instrumentation automatically:
   - Creates spans for each HTTP request
   - Propagates trace context through HTTP headers
   - Records HTTP method, URL, status code, and other relevant metadata
   - Captures errors and exceptions

### Dependency Injection

The `trace.TracerProvider` is automatically injected by the fx dependency injection container:
- Observability service creates the TracerProvider
- `ExposeObservabilityComponents()` exposes it to the DI container
- `security.New()` receives it as a parameter
- If TracerProvider is nil, the service falls back to using standard HTTP client without tracing

## Benefits

1. **Full Observability**: All OAuth-related HTTP calls are now traced, making it easier to:
   - Debug authentication issues
   - Monitor OAuth flow performance
   - Track API call latencies to external services (GitHub, etc.)
   - Identify bottlenecks in the authentication process

2. **Distributed Tracing**: Traces can be correlated across services if GitHub or other OAuth providers support trace context propagation

3. **No Breaking Changes**: The implementation is backward compatible:
   - If TracerProvider is not available, standard HTTP client is used
   - Existing functionality remains unchanged
   - No changes required to calling code

## Testing

The implementation compiles successfully and maintains backward compatibility. To test the tracing:

1. Ensure OpenTelemetry trace exporter is configured in the application settings
2. Perform OAuth authentication (GitHub OAuth or device flow)
3. Check your tracing backend (e.g., Jaeger, Zipkin) for spans related to:
   - OAuth token exchange requests
   - GitHub API calls (e.g., ListEmails)

## Example Trace Spans

When performing GitHub OAuth authentication, you should see spans like:
- `HTTP POST` to `https://github.com/login/oauth/access_token` (token exchange)
- `HTTP GET` to `https://api.github.com/user/emails` (fetch user emails)

Each span will include metadata such as:
- HTTP method and URL
- Request and response headers
- Status codes
- Duration
- Any errors encountered
