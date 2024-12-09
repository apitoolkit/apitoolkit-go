package apitoolkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sethvargo/go-envconfig"
	hostMetrics "go.opentelemetry.io/contrib/instrumentation/host"
	runtimeMetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/encoding/gzip"
)

const APITOOLKIT_ENDPOINT = "otelcol.apitoolkit.io:4317"

type Option func(*OConfig)

type OConfig struct {
	TracesEnabled            *bool             `env:"OTEL_TRACES_ENABLED,default=true"`
	ExporterEndpoint         string            `env:"OTEL_EXPORTER_OTLP_ENDPOINT,overwrite"`
	ExporterEndpointInsecure bool              `env:"OTEL_EXPORTER_OTLP_INSECURE"`
	ServiceName              string            `env:"OTEL_SERVICE_NAME,overwrite"`
	ServiceVersion           string            `env:"OTEL_SERVICE_VERSION,overwrite,default=unknown"`
	MetricsEnabled           *bool             `env:"OTEL_METRICS_ENABLED,default=true"`
	MetricsReportingPeriod   string            `env:"OTEL_EXPORTER_OTLP_METRICS_PERIOD,overwrite,default=30s"`
	LogLevel                 string            `env:"OTEL_LOG_LEVEL,overwrite,default=info"`
	Propagators              []string          `env:"OTEL_PROPAGATORS,overwrite,default=tracecontext,baggage"`
	ResourceAttributes       map[string]string `env:"OTEL_RESOURCE_ATTRIBUTES,overwrite,separator=="`
	SpanProcessors           []trace.SpanProcessor
	Sampler                  trace.Sampler
	ResourceOptions          []resource.Option
	Resource                 *resource.Resource
	ErrorHandler             otel.ErrorHandler
}

func ConfigureOpenTelemetry(opts ...Option) (func(), error) {
	c, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	if c.LogLevel == "debug" {
		log.Println("debug logging enabled")
		log.Println("configuration")
		s, _ := json.MarshalIndent(c, "", "\t")
		log.Println(string(s))
	}

	if c.ErrorHandler != nil {
		otel.SetErrorHandler(c.ErrorHandler)
	}

	otelConfig := OtelConfig{
		config: c,
	}

	type setupFunc func(*OConfig) (func() error, error)

	for _, setup := range []setupFunc{setupTracing, setupMetrics} {
		shutdown, err := setup(c)
		if err != nil {
			return otelConfig.Shutdown, fmt.Errorf("setup error: %w", err)
		}
		if shutdown != nil {
			otelConfig.shutdownFuncs = append(otelConfig.shutdownFuncs, shutdown)
		}
	}
	return otelConfig.Shutdown, nil
}

func setupTracing(c *OConfig) (func() error, error) {
	var enabled bool
	if c.TracesEnabled == nil {
		enabled = true
	} else {
		enabled = *c.TracesEnabled
	}
	if !enabled {
		log.Println("tracing is disabled by configuration: enabled set to false")
		return nil, nil
	}

	opts := []trace.TracerProviderOption{
		trace.WithResource(c.Resource),
		trace.WithSampler(c.Sampler),
	}
	for _, sp := range c.SpanProcessors {
		opts = append(opts, trace.WithSpanProcessor(sp))
	}
	spanExporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(c.ExporterEndpoint),
			otlptracegrpc.WithCompressor(gzip.Name),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create span exporter: %v", err)
	}

	bsp := trace.NewBatchSpanProcessor(spanExporter)
	opts = append(opts, trace.WithSpanProcessor(bsp))

	tp := trace.NewTracerProvider(opts...)
	// if err = configurePropagators(c); err != nil {
	// 	return nil, err
	// }
	otel.SetTracerProvider(tp)
	return func() error {
		_ = bsp.Shutdown(context.Background())
		return spanExporter.Shutdown(context.Background())
	}, nil
}

func setupMetrics(c *OConfig) (func() error, error) {
	var enabled bool
	if c.MetricsEnabled == nil {
		enabled = true
	} else {
		enabled = *c.MetricsEnabled
	}
	if !enabled {
		log.Println("metrics are disabled by configuration: enabled set to false")
		return nil, nil
	}
	metricExporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint(c.ExporterEndpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithCompressor(gzip.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %v", err)
	}

	var readerOpts []metric.PeriodicReaderOption
	if c.MetricsReportingPeriod != "" {
		period, err := time.ParseDuration(c.MetricsReportingPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid metric reporting period: %v", err)
		}
		if period <= 0 {
			return nil, fmt.Errorf("invalid metric reporting period: %v", c.MetricsReportingPeriod)
		}
		readerOpts = append(readerOpts, metric.WithInterval(period))
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(c.Resource),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, readerOpts...)))

	if err = runtimeMetrics.Start(runtimeMetrics.WithMeterProvider(meterProvider)); err != nil {
		return nil, fmt.Errorf("failed to start runtime metrics: %v", err)
	}

	if err = hostMetrics.Start(hostMetrics.WithMeterProvider(meterProvider)); err != nil {
		return nil, fmt.Errorf("failed to start host metrics: %v", err)
	}

	otel.SetMeterProvider(meterProvider)
	return func() error {
		return meterProvider.Shutdown(context.Background())
	}, nil
}

func newConfig(opts ...Option) (*OConfig, error) {
	c := &OConfig{
		ResourceAttributes: map[string]string{},
		Sampler:            trace.AlwaysSample(),
		ExporterEndpoint:   APITOOLKIT_ENDPOINT,
	}

	// apply environment variables last to override any vendor or user options
	envError := envconfig.Process(context.Background(), c)
	if envError != nil {
		log.Fatalf("environment error: %v", envError)
		return nil, fmt.Errorf("environment error: %w", envError)
	}
	for _, opt := range opts {
		opt(c)
	}
	var err error
	c.Resource, err = newResource(c)

	if err != nil {
		if errors.Is(err, resource.ErrSchemaURLConflict) {
			log.Printf("schema conflict %v", err)
			// ignore schema conflicts
			err = nil
		}
	}

	return c, err
}

type OtelConfig struct {
	config        *OConfig
	shutdownFuncs []func() error
}

func (otel *OtelConfig) Shutdown() {
	for _, shutdown := range otel.shutdownFuncs {
		if err := shutdown(); err != nil {
			log.Fatalf("failed to stop exporter: %v", err)
		}
	}
}

func newResource(c *OConfig) (*resource.Resource, error) {
	options := []resource.Option{
		resource.WithSchemaURL(semconv.SchemaURL),
	}
	if c.ResourceAttributes != nil {
		attrs := make([]attribute.KeyValue, 0, len(c.ResourceAttributes))
		for k, v := range c.ResourceAttributes {
			if len(v) > 0 {
				attrs = append(attrs, attribute.String(k, v))
			}
		}
		options = append(options, resource.WithAttributes(attrs...))
	}
	options = append(options, c.ResourceOptions...)
	if c.ServiceName != "" {
		options = append(options, resource.WithAttributes(semconv.ServiceNameKey.String(c.ServiceName)))
	}
	if c.ServiceVersion != "" {
		options = append(options, resource.WithAttributes(semconv.ServiceVersionKey.String(c.ServiceVersion)))
	}
	options = append(options, resource.WithHost())
	options = append(options, resource.WithAttributes(
		semconv.TelemetrySDKNameKey.String("apitoolkit-otelconfig"),
		semconv.TelemetrySDKLanguageGo,
	))
	options = append(options, resource.WithFromEnv())
	return resource.New(
		context.Background(),
		options...,
	)
}

func WithServiceName(name string) Option {
	return func(c *OConfig) {
		c.ServiceName = name
	}
}
func WithServiceVersion(version string) Option {
	return func(c *OConfig) {
		c.ServiceVersion = version
	}
}

func WithLogLevel(loglevel string) Option {
	return func(c *OConfig) {
		c.LogLevel = loglevel
	}
}

func WithResourceAttributes(attributes map[string]string) Option {
	return func(c *OConfig) {
		for k, v := range attributes {
			c.ResourceAttributes[k] = v
		}
	}
}

func WithResourceOption(option resource.Option) Option {
	return func(c *OConfig) {
		c.ResourceOptions = append(c.ResourceOptions, option)
	}
}

func WithPropagators(propagators []string) Option {
	return func(c *OConfig) {
		c.Propagators = propagators
	}
}

// Configures a global error handler to be used throughout an OpenTelemetry instrumented project.
// See "go.opentelemetry.io/otel".
func WithErrorHandler(handler otel.ErrorHandler) Option {
	return func(c *OConfig) {
		c.ErrorHandler = handler
	}
}

func WithMetricsReportingPeriod(p time.Duration) Option {
	return func(c *OConfig) {
		c.MetricsReportingPeriod = fmt.Sprint(p)
	}
}

func WithMetricsEnabled(enabled bool) Option {
	return func(c *OConfig) {
		c.MetricsEnabled = &enabled
	}
}

func WithTracesEnabled(enabled bool) Option {
	return func(c *OConfig) {
		c.TracesEnabled = &enabled
	}
}

func WithSpanProcessor(sp ...trace.SpanProcessor) Option {
	return func(c *OConfig) {
		c.SpanProcessors = append(c.SpanProcessors, sp...)
	}
}

func WithSampler(sampler trace.Sampler) Option {
	return func(c *OConfig) {
		c.Sampler = sampler
	}
}
