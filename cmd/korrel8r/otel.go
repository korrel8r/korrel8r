// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
)

var (
	// Metric output
	otelCollectorFlag = rootCmd.PersistentFlags().String("otel-collector", "", "URL of OTLP collector endpoint for pushing metrics (e.g. http://localhost:4318/v1/metrics)")
	metricFileFlag    = rootCmd.PersistentFlags().String("metric-file", "", "Write metrics to the given file at the end of execution")

	metricsHandler http.Handler
	metricsStop    func()
)

func startMetrics() {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Expose prometheus metric scrape point
	var opts []sdkmetric.Option
	promExp := must.Must1(promexporter.New(promexporter.WithRegisterer(reg)))
	opts = append(opts, sdkmetric.WithReader(promExp))

	// Push metrics to OTEL collector
	if *otelCollectorFlag != "" {
		otlpExp := must.Must1(otlpmetrichttp.New(
			context.Background(),
			otlpmetrichttp.WithEndpointURL(*otelCollectorFlag),
		))
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlpExp)))
		log.V(1).Info("OTLP metric push exporter", "endpoint", *otelCollectorFlag)
	}

	// Dump metrics to file on exit
	fileStop := func(context.Context) {}
	if *metricFileFlag != "" {
		file := must.Must1(os.Create(*metricFileFlag))
		fileExp := must.Must1(stdoutmetric.New(
			stdoutmetric.WithWriter(file),
			stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithoutTimestamps(),
		))
		fileReader := sdkmetric.NewManualReader()
		opts = append(opts, sdkmetric.WithReader(fileReader))
		log.V(1).Info("Metric file exporter", "file", *metricFileFlag)

		fileStop = func(ctx context.Context) {
			rm := &metricdata.ResourceMetrics{}
			if err := fileReader.Collect(ctx, rm); err != nil {
				log.Error(err, "Metric file export")
			} else if err := fileExp.Export(ctx, rm); err != nil {
				log.Error(err, "Metric file export")
			}
			_ = file.Close()
		}
	}

	mp := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)

	metricsHandler = promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	metricsStop = func() {
		ctx := context.Background()
		fileStop(ctx)
		if err := mp.Shutdown(ctx); err != nil {
			log.Error(err, "MeterProvider shutdown")
		}
	}
}
