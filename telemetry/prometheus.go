package telemetry

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	exportmetric "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func InitMetrics(serviceName string) error {
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8181"
	}

	res, err := GetResource(serviceName)
	if err != nil {
		return err
	}

	config := prometheus.Config{}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			exportmetric.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(res),
	)
	metricsExporter, err := prometheus.New(config, c)
	if err != nil {
		return fmt.Errorf("failed to setup prom exporter: %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metricsExporter)
		if lerr := http.ListenAndServe(":"+metricsPort, mux); lerr != nil {
			log.Fatalf("failed to setup prom http endpoint: %v", lerr)
		}
	}()

	global.SetMeterProvider(metricsExporter.MeterProvider())
	if err := runtime.Start(); err != nil {
		log.Fatalf("failed to setup runtime monitor: %v", err)
	}

	return nil
}
