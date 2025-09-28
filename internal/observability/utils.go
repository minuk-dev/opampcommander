package observability

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	traceapi "go.opentelemetry.io/otel/trace"
)

var (
	// It's very week constraint.
	// If it has any problem, we should use own tracer key logic.
	// ref. https://github.com/open-telemetry/opentelemetry-go-contrib/blob/91447f2ff738d1909cfd85258df3e5624d7c2502/instrumentation/github.com/gin-gonic/gin/otelgin/gin.go#L24
	tracerKey = "otel-go-contrib-tracer"
)

var (
	// ErrNilContext is returned when a nil context is provided.
	ErrNilContext = errors.New("nil context")
	// ErrNoGinContext is returned when the provided context is not a Gin context.
	ErrNoGinContext = errors.New("no gin context")
	// ErrNoTracerInContext is returned when there is no tracer in the context.
	ErrNoTracerInContext = errors.New("no tracer in context")
	// ErrInvalidTracerInContext is returned when the tracer in the context is not valid.
	ErrInvalidTracerInContext = errors.New("invalid tracer in context")
)

// GetTracer retrieves the tracer from the Gin context.
// It returns an error if the context is nil, not a Gin context, or does not contain a valid tracer.
func GetTracer(ctx context.Context) (traceapi.Tracer, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	ginCtx := ctx.(*gin.Context)
	if ginCtx == nil {
		return nil, ErrNoGinContext
	}

	tracerAny, exists := ginCtx.Get(tracerKey)
	if !exists {
		return nil, ErrNoTracerInContext
	}

	tracer, ok := tracerAny.(traceapi.Tracer)
	if !ok {
		return nil, ErrInvalidTracerInContext
	}

	return tracer, nil
}
