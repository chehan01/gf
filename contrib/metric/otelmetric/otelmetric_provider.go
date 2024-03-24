// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package otelmetric

import (
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gmetric"
)

// localProvider implements interface gmetric.Provider.
type localProvider struct {
	*metric.MeterProvider
}

// newProvider creates and returns an object that implements gmetric.Provider.
// DO NOT set this as global provider internally.
func newProvider(options ...metric.Option) (gmetric.Provider, error) {
	// TODO global logger set for otel.
	// otel.SetLogger()

	var (
		err          error
		metrics      = gmetric.GetAllMetrics()
		builtinViews = createViewsForBuiltInMetrics()
		callbacks    = gmetric.GetRegisteredCallbacks()
	)
	options = append(options, metric.WithView(builtinViews...))
	provider := &localProvider{
		// MeterProvider is the core object that can create otel metrics.
		MeterProvider: metric.NewMeterProvider(options...),
	}

	if err = provider.initializeMetrics(metrics); err != nil {
		return nil, err
	}

	if err = provider.initializeCallback(callbacks); err != nil {
		return nil, err
	}

	// builtin metrics: golang.
	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
		runtime.WithMeterProvider(provider),
	)
	if err != nil {
		return nil, gerror.WrapCode(
			gcode.CodeInternalError, err, `start built-in runtime metrics failed`,
		)
	}

	return provider, nil
}

// SetAsGlobal sets current provider as global meter provider for current process,
// which makes the following metrics creating on this Provider, especially the metrics created in runtime.
func (l *localProvider) SetAsGlobal() {
	gmetric.SetGlobalProvider(l)
	otel.SetMeterProvider(l)
}

// MeterPerformer creates and returns a MeterPerformer.
// A Performer can produce types of Metric performer.
func (l *localProvider) MeterPerformer(option gmetric.MeterOption) gmetric.MeterPerformer {
	return newMeterPerformer(l.MeterProvider, option)
}

// createViewsForBuiltInMetrics creates and returns views for builtin metrics.
func createViewsForBuiltInMetrics() []metric.View {
	var views = make([]metric.View, 0)
	views = append(views, metric.NewView(
		metric.Instrument{
			Name: "process.runtime.go.gc.pause_ns",
			Scope: instrumentation.Scope{
				Name:    runtime.ScopeName,
				Version: runtime.Version(),
			},
		},
		metric.Stream{
			Aggregation: metric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{
					500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000,
				},
			},
		},
	))
	views = append(views, metric.NewView(
		metric.Instrument{
			Name: "runtime.uptime",
			Scope: instrumentation.Scope{
				Name:    runtime.ScopeName,
				Version: runtime.Version(),
			},
		},
		metric.Stream{
			Name: "process.runtime.uptime",
		},
	))
	return views
}

// initializeMetrics initializes all metrics in provider creating.
// The initialization replaces the underlying metric performer using noop-performer with truly performer
// that implements operations for types of metric.
func (l *localProvider) initializeMetrics(metrics []gmetric.Metric) error {
	var meterPerformer gmetric.MeterPerformer
	for _, m := range metrics {
		meterPerformer = l.MeterPerformer(gmetric.MeterOption{
			Instrument:        m.Info().Instrument().Name(),
			InstrumentVersion: m.Info().Instrument().Version(),
		})
		if initializer, ok := m.(gmetric.MetricInitializer); ok {
			if err := initializer.Init(meterPerformer); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *localProvider) initializeCallback(callbackItems []gmetric.CallbackItem) error {
	var meterPerformer gmetric.MeterPerformer
	for _, callbackItem := range callbackItems {
		if callbackItem.MeterPerformer != nil {
			continue
		}
		if len(callbackItem.Metrics) == 0 {
			continue
		}
		meterPerformer = l.MeterPerformer(gmetric.MeterOption{
			Instrument:        callbackItem.Metrics[0].(gmetric.Metric).Info().Instrument().Name(),
			InstrumentVersion: callbackItem.Metrics[0].(gmetric.Metric).Info().Instrument().Version(),
		})
		callbackItem.MeterPerformer = meterPerformer
		if err := meterPerformer.RegisterCallback(
			callbackItem.Callback, callbackItem.Metrics...,
		); err != nil {
			return err
		}
	}
	return nil
}