package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func newTraceProvider(
	lifecycle fx.Lifecycle,
	traceConfig config.TraceSettings,
	_ *slog.Logger,
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
		return nil, errors.New("invalid trace sampler type")
	}

	traceCtx, cancel := context.WithCancel(context.Background())
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			cancel()

			return nil
		},
	})

	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	switch traceConfig.Protocol {
	case config.TraceProtocolHTTP:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(traceConfig.Endpoint),
		}
		if traceConfig.Compression {
			switch traceConfig.CompressionAlgorithm {
			case config.TraceCompressionAlgorithmGzip:
				opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
			default:
				return nil, errors.New("invalid trace compression algorithm")
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

		exporter, err = otlptracehttp.New(traceCtx, opts...)
	case config.TraceProtocolGRPC:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(traceConfig.Endpoint),
		}
		if traceConfig.Compression {
			switch traceConfig.CompressionAlgorithm {
			case config.TraceCompressionAlgorithmGzip:
				opts = append(opts, otlptracegrpc.WithCompressor(string(traceConfig.CompressionAlgorithm)))
			default:
				return nil, errors.New("invalid trace compression algorithm")
			}
		}

		if traceConfig.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		if len(traceConfig.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(traceConfig.Headers))
		}

		exporter, err = otlptracegrpc.New(traceCtx, opts...)
	default:
		return nil, errors.New("invalid trace protocol")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewSimpleSpanProcessor(exporter)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	return traceProvider, nil
}
