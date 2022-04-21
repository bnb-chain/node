package metrics

import (
	metricsPkg "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

// Metrics contains Metrics exposed by this package.
type Metrics struct {
	ErrNumOfChannels metricsPkg.Counter
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
func PrometheusMetrics() *Metrics {
	return &Metrics{
		ErrNumOfChannels: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Subsystem: "oracle",
			Name:      "err_num_of_channels",
			Help:      "The error numbers of channel happened from boot",
		}, []string{"channel_id"}),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		ErrNumOfChannels: discard.NewCounter(),
	}
}
