package ioc

import (
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"log"
	"time"
)

// InitZipkinTracer 初始化 zipkin tracer
func InitZipkinTracer() *trace.TracerProvider {
	// 创建资源信息
	res, err := newResource()
	if err != nil {
		log.Fatal(err)
	}

	// 初始化传播器
	otel.SetTextMapPropagator(newPropagator())

	// 初始化 tracer provider
	tp, err := newTracerProvider(res)
	if err != nil {
		log.Fatal(err)
	}

	return tp
}

// newTracerProvider 创建 tracer provider
func newTracerProvider(res *resource.Resource) (*trace.TracerProvider, error) {
	// 从配置读取 zipkin 断点地址
	zipkinEndpoint := viper.GetString("trace.zipkin.endpoint")

	// 创建 Zipkin 导出器
	exporter, err := zipkin.New(zipkinEndpoint)
	if err != nil {
		return nil, err
	}

	// 创建 tracer provider
	return trace.NewTracerProvider(
		trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	), nil
}

// newPropagator 创建上下文传播器
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newResource 创建 OpenTelemetry 资源
func newResource() (*resource.Resource, error) {
	serviceName := viper.GetString("trace.zipkin.serviceName")
	serviceVersion := "v0.0.1"

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
}
