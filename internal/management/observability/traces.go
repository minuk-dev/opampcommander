package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

var (
	// ErrUnsupportedCompressionAlgorithm is returned when an unsupported compression algorithm is specified.
	ErrUnsupportedCompressionAlgorithm = errors.New("unsupported compression algorithm")
	// ErrUnsupportedProtocol is returned when an unsupported trace protocol is specified.
	ErrUnsupportedProtocol = errors.New("unsupported trace protocol")
	// ErrInvalidTraceSampler is returned when an invalid trace sampler is specified.
	ErrInvalidTraceSampler = errors.New("invalid trace sampler")
)

func newTraceProvider(
	serviceName string,
	shutdownlistener *helper.ShutdownListener,
	traceConfig config.TraceSettings,
	_ *slog.Logger,
) (*sdktrace.TracerProvider, error) {
	sampler, err := toSampler(traceConfig.Sampler, traceConfig.SamplerRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace sampler: %w", err)
	}

	traceCtx, cancel := context.WithCancel(context.Background())

	resource, err := resource.New(
		traceCtx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	exporter, err := newTraceExporter(traceCtx, traceConfig)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	shutdownlistener.Register(bsp)
	shutdownlistener.Register(exporter)
	shutdownlistener.Register(traceProvider)
	shutdownlistener.RegisterFunc(func(context.Context) error {
		cancel()

		return nil
	})

	return traceProvider, nil
}

func newTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (*otlptrace.Exporter, error) {
	switch traceConfig.Protocol {
	case config.TraceProtocolHTTP:
		return newHTTPTraceExporter(traceCtx, traceConfig)
	case config.TraceProtocolGRPC:
		return newGRPCTraceExporter(traceCtx, traceConfig)
	default:
		return nil, ErrUnsupportedProtocol
	}
}

func newHTTPTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(traceConfig.Endpoint),
	}
	if traceConfig.Compression {
		switch traceConfig.CompressionAlgorithm {
		case config.TraceCompressionAlgorithmGzip:
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		default:
			return nil, ErrUnsupportedCompressionAlgorithm
		}
	} else {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
	}

	if traceConfig.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(traceConfig.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(traceConfig.Headers))
	}

	exporter, err := otlptracehttp.New(traceCtx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP trace exporter: %w", err)
	}

	return exporter, nil
}

func newGRPCTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(traceConfig.Endpoint),
	}
	if traceConfig.Compression {
		switch traceConfig.CompressionAlgorithm {
		case config.TraceCompressionAlgorithmGzip:
			opts = append(opts, otlptracegrpc.WithCompressor(string(traceConfig.CompressionAlgorithm)))
		default:
			return nil, ErrUnsupportedCompressionAlgorithm
		}
	}

	if traceConfig.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if len(traceConfig.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(traceConfig.Headers))
	}

	exporter, err := otlptracegrpc.New(traceCtx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC trace exporter: %w", err)
	}

	return exporter, nil
}

//nolint:ireturn // Sampler is an interface that needs to be returned
func toSampler(samplerType config.TraceSampler, ratio float64) (sdktrace.Sampler, error) {
	switch samplerType {
	case config.TraceSamplerAlways:
		return sdktrace.AlwaysSample(), nil
	case config.TraceSamplerNever:
		return sdktrace.NeverSample(), nil
	case config.TraceSamplerProbability:
		return sdktrace.TraceIDRatioBased(ratio), nil
	default:
		return nil, ErrInvalidTraceSampler
	}
}
