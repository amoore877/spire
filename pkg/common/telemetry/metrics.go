package telemetry

import (
	"context"
	"errors"
	"time"

	"github.com/armon/go-metrics"
	"github.com/spiffe/spire/pkg/common/util"
)

const timerGranularity = time.Millisecond

// Label is a label/tag for a metric
type Label = metrics.Label

// Sink is an interface for emitting metrics
type Sink = metrics.MetricSink

// Metrics is an interface for all metrics plugins and services
type Metrics interface {
	// A Gauge should retain the last value it is set to
	SetGauge(key []string, val float32)
	SetGaugeWithLabels(key []string, val float32, labels []Label)

	// Should emit a Key/Value pair for each call
	EmitKey(key []string, val float32)

	// Counters should accumulate values
	IncrCounter(key []string, val float32)
	IncrCounterWithLabels(key []string, val float32, labels []Label)

	// Samples are for timing information, where quantiles are used
	AddSample(key []string, val float32)
	AddSampleWithLabels(key []string, val float32, labels []Label)

	// A convenience function for measuring elapsed time with a single line
	MeasureSince(key []string, start time.Time)
	MeasureSinceWithLabels(key []string, start time.Time, labels []Label)
}

type MetricsImpl struct {
	*metrics.Metrics

	c       *MetricsConfig
	runners []sinkRunner
	// Each instance of metrics.Metrics in the slice corresponds to one metrics sink type
	metricsSinks []*metrics.Metrics
}

var _ Metrics = (*MetricsImpl)(nil)

// NewMetrics returns a Metric implementation
func NewMetrics(c *MetricsConfig) (*MetricsImpl, error) {
	if c.Logger == nil {
		return nil, errors.New("logger must be configured")
	}

	impl := &MetricsImpl{c: c}

	for _, f := range sinkRunnerFactories {
		runner, err := f(c)
		if err != nil {
			return nil, err
		}

		if !runner.isConfigured() {
			continue
		}

		fanout := metrics.FanoutSink{}
		fanout = append(fanout, runner.sinks()...)

		conf := metrics.DefaultConfig(c.ServiceName)
		conf.EnableHostname = false
		conf.EnableHostnameLabel = true
		conf.EnableTypePrefix = runner.requiresTypePrefix()

		metricsSink, err := metrics.New(conf, fanout)
		if err != nil {
			return nil, err
		}

		impl.metricsSinks = append(impl.metricsSinks, metricsSink)
		impl.runners = append(impl.runners, runner)
	}

	return impl, nil
}

// ListenAndServe starts the metrics process
func (m *MetricsImpl) ListenAndServe(ctx context.Context) error {
	var tasks []func(context.Context) error
	for _, runner := range m.runners {
		tasks = append(tasks, runner.run)
	}

	return util.RunTasks(ctx, tasks...)
}

func (m *MetricsImpl) SetGauge(key []string, val float32) {
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.SetGauge(keyCopy, val)
	}
}

// SetGaugeWithLabels delegates to embedded metrics, sanitizing labels
func (m *MetricsImpl) SetGaugeWithLabels(key []string, val float32, labels []Label) {
	sanitizedLabels := SanitizeLabels(labels)
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.SetGaugeWithLabels(keyCopy, val, sanitizedLabels)
	}
}

func (m *MetricsImpl) EmitKey(key []string, val float32) {
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.EmitKey(keyCopy, val)
	}
}

func (m *MetricsImpl) IncrCounter(key []string, val float32) {
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.IncrCounter(keyCopy, val)
	}
}

// IncrCounterWithLabels delegates to embedded metrics, sanitizing labels
func (m *MetricsImpl) IncrCounterWithLabels(key []string, val float32, labels []Label) {
	sanitizedLabels := SanitizeLabels(labels)
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.IncrCounterWithLabels(keyCopy, val, sanitizedLabels)
	}
}

func (m *MetricsImpl) AddSample(key []string, val float32) {
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.AddSample(keyCopy, val)
	}
}

// AddSampleWithLabels delegates to embedded metrics, sanitizing labels
func (m *MetricsImpl) AddSampleWithLabels(key []string, val float32, labels []Label) {
	sanitizedLabels := SanitizeLabels(labels)
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.AddSampleWithLabels(keyCopy, val, sanitizedLabels)
	}
}

func (m *MetricsImpl) MeasureSince(key []string, start time.Time) {
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.MeasureSince(keyCopy, start)
	}
}

// MeasureSinceWithLabels delegates to embedded metrics, sanitizing labels
func (m *MetricsImpl) MeasureSinceWithLabels(key []string, start time.Time, labels []Label) {
	sanitizedLabels := SanitizeLabels(labels)
	for _, s := range m.metricsSinks {
		keyCopy := copyKey(key)
		s.MeasureSinceWithLabels(keyCopy, start, sanitizedLabels)
	}
}

// TODO: [rturner3] Remove this if/when go-metrics is fixed to not modify its input "key" slice
func copyKey(key []string) []string {
	return append([]string(nil), key...)
}
