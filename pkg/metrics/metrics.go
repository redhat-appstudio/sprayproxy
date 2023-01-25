package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

const (
	separator     = "_"
	requestsTotal = "requests_total"

	subsystem = "sprayproxy"

	inbound                   = "http" + separator + "inbound"
	inboundRequestsName       = subsystem + separator + inbound + separator + requestsTotal
	forwarded                 = "htp" + separator + "forwarded"
	forwardedRequestsName     = subsystem + separator + forwarded + separator + requestsTotal
	responseTime              = "http" + separator + "response" + separator + "time"
	forwardedResponseTimeName = subsystem + separator + responseTime + separator + "duration_seconds"
	hostLabel                 = "host"

	MetricsPort = 6000
)

var (
	initCalled        = false
	lock              = sync.Mutex{}
	inboundRequests   prometheus.Counter
	forwardedRequests *prometheus.CounterVec
	responseTimes     prometheus.Histogram
)

func InitMetrics(registry *prometheus.Registry) {
	lock.Lock()
	defer lock.Unlock()
	if initCalled && registry == nil {
		return
	}
	initCalled = true
	if registry == nil {
		prometheus.MustRegister(createMetrics()...)
		return
	}
	registry.MustRegister(createMetrics()...)
}

func createMetrics() []prometheus.Collector {
	inboundRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: inboundRequestsName,
		Help: "Counts incoming requests to the proxy.",
	})
	forwardedRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: forwardedRequestsName,
		Help: "Counts forwarded attempts to backend server(s).",
	},
		[]string{hostLabel})
	responseTimes = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: forwardedResponseTimeName,
		Help: "Forwarded request duration in seconds.",
		// Create buckets of 0.005, 0.05, 0.5, 5, and +Infinity
		Buckets: prometheus.ExponentialBuckets(0.005, 10, 4),
	})
	return []prometheus.Collector{
		inboundRequests,
		forwardedRequests,
		responseTimes,
	}
}

func IncInboundCount() {
	if inboundRequests != nil {
		inboundRequests.Inc()
	}
}

func IncForwardedCount(hostname string) {
	if forwardedRequests != nil {
		forwardedRequests.With(prometheus.Labels{hostLabel: hostname}).Inc()
	}
}

func AddForwardedResponseTime(seconds float64) {
	if responseTimes != nil {
		responseTimes.Observe(seconds)
	}
}
