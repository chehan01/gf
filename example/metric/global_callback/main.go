// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/gogf/gf/contrib/metric/otelmetric/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/gogf/gf/v2/os/gmetric"
)

var (
	counter = gmetric.MustNewCounter(gmetric.CounterConfig{
		MetricConfig: gmetric.MetricConfig{
			Name: "goframe.metric.demo.counter",
			Help: "This is a simple demo for Counter usage",
			Unit: "%",
			Attributes: gmetric.Attributes{
				gmetric.NewAttribute("const_label_a", 1),
			},
			Instrument:        "github.com/gogf/gf/example/metric/basic",
			InstrumentVersion: "v1.0",
		},
	})
	gauge = gmetric.MustNewGauge(gmetric.GaugeConfig{
		MetricConfig: gmetric.MetricConfig{
			Name: "goframe.metric.demo.gauge",
			Help: "This is a simple demo for Gauge usage",
			Unit: "bytes",
			Attributes: gmetric.Attributes{
				gmetric.NewAttribute("const_label_b", 2),
			},
			Instrument:        "github.com/gogf/gf/example/metric/basic",
			InstrumentVersion: "v1.0",
		},
	})
	histogram = gmetric.MustNewHistogram(gmetric.HistogramConfig{
		MetricConfig: gmetric.MetricConfig{
			Name: "goframe.metric.demo.histogram",
			Help: "This is a simple demo for histogram usage",
			Unit: "ms",
			Attributes: gmetric.Attributes{
				gmetric.NewAttribute("const_label_c", 3),
			},
			Instrument:        "github.com/gogf/gf/example/metric/basic",
			InstrumentVersion: "v1.0",
		},
		Buckets: []float64{0, 10, 20, 50, 100, 500, 1000, 2000, 5000, 10000},
	})
)

func main() {
	var ctx = gctx.New()

	// Prometheus exporter to export metrics as Prometheus format.
	exporter, err := prometheus.New(
		prometheus.WithoutCounterSuffixes(),
		prometheus.WithoutUnits(),
	)
	if err != nil {
		g.Log().Fatal(ctx, err)
	}

	gmetric.MustRegisterCallback(func(ctx context.Context, m gmetric.CallbackObserver) error {
		glog.Infof(ctx, `callback triggered1`)
		m.Observe(counter, 100)
		m.Observe(gauge, 300)
		return nil
	}, counter, gauge)

	gmetric.MustRegisterCallback(func(ctx context.Context, m gmetric.CallbackObserver) error {
		glog.Infof(ctx, `callback triggered2`)
		m.Observe(counter, 100)
		m.Observe(gauge, 300)
		return nil
	}, counter, gauge)

	// OpenTelemetry provider.
	provider := otelmetric.MustProvider(metric.WithReader(exporter))
	provider.SetAsGlobal()
	defer provider.Shutdown(ctx)

	// Add value for counter.
	counter.Inc()
	counter.Add(10)

	// Observe value for gauge.
	gauge.Set(100)
	gauge.Inc()
	gauge.Sub(1)

	// Record values for histogram.
	histogram.Record(1)
	histogram.Record(20)
	histogram.Record(30)
	histogram.Record(101)
	histogram.Record(2000)
	histogram.Record(9000)
	histogram.Record(20000)

	// HTTP Server for metrics exporting.
	s := g.Server()
	s.BindHandler("/metrics", ghttp.WrapH(promhttp.Handler()))
	s.SetPort(8000)
	s.Run()
}