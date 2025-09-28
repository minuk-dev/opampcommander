package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

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

//nolint:ireturn
func newTraceProvider(
	lifecycle fx.Lifecycle,
	traceConfig config.TraceSettings,
	logger *slog.Logger,
) (traceapi.TracerProvider, error) {
	var sampler sdktrace.Sampler
	switch traceConfig.Sampler {
	case config.TraceSamplerAlways:
		sampler = sdktrace.AlwaysSample()
	case config.TraceSamplerNever:
		sampler = sdktrace.NeverSample()
	case config.TraceSamplerProbability:
		sampler = sdktrace.TraceIDRatioBased(traceConfig.SamplerRatio)
	default:
		return nil, ErrInvalidTraceSampler
	}

	traceCtx, cancel := context.WithCancel(context.Background())

	exporter, err := newTraceExporter(traceCtx, traceConfig)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			err := exporter.Shutdown(ctx)
			if err != nil {
				logger.Warn("failed to shutdown trace exporter", slog.String("error", err.Error()))
			}
			cancel()

			return nil
		},
	})

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	return traceProvider, nil
}

//nolint:ireturn
func newTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (sdktrace.SpanExporter, error) {
	switch traceConfig.Protocol {
	case config.TraceProtocolHTTP:
		return newHTTPTraceExporter(traceCtx, traceConfig)
	case config.TraceProtocolGRPC:
		return newGRPCTraceExporter(traceCtx, traceConfig)
	default:
		return nil, ErrUnsupportedProtocol
	}
}

//nolint:ireturn
func newHTTPTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (sdktrace.SpanExporter, error) {
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

//nolint:ireturn
func newGRPCTraceExporter(
	traceCtx context.Context,
	traceConfig config.TraceSettings,
) (sdktrace.SpanExporter, error) {
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
