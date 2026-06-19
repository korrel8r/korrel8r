// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
)

var otelCollectorFlag *string

func startMetrics() (http.Handler, func()) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	promExp := must.Must1(promexporter.New(promexporter.WithRegisterer(reg)))

	opts := []sdkmetric.Option{
		sdkmetric.WithReader(promExp),
	}

	if *otelCollectorFlag != "" {
		otlpExp := must.Must1(otlpmetrichttp.New(
			context.Background(),
			otlpmetrichttp.WithEndpointURL(*otelCollectorFlag),
		))
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlpExp)))
		log.V(1).Info("OTLP metric push exporter", "endpoint", *otelCollectorFlag)
	}

	mp := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	stop := func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Error(err, "MeterProvider shutdown")
		}
	}
	return handler, stop
}
