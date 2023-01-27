package metrics

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type fakeResponseWriter struct {
	bytes.Buffer
	statusCode int
	header     http.Header
}

func (f *fakeResponseWriter) Header() http.Header {
	return f.header
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

func TestMetrics(t *testing.T) {
	for _, test := range []struct {
		name         string
		expected     []string
		githubs      int
		forwards     int
		responseTime float64
	}{
		{
			name: "One inbound, two forwards, 50 response time",
			expected: []string{
				`# TYPE ` + inboundRequestsName + ` counter`,
				inboundRequestsName + ` 1`,
				`# TYPE ` + forwardedRequestsName + ` counter`,
				forwardedRequestsName + `{host="host1"} 2`,
				`# TYPE ` + forwardedResponseTimeName + ` histogram`,
				forwardedResponseTimeName + `_sum 50`,
				forwardedResponseTimeName + `_count 1`,
				forwardedResponseTimeName + `_bucket`,
			},
			githubs:      1,
			forwards:     2,
			responseTime: float64(50),
		},
		{
			name: "Two inbound, no forward, no response time",
			expected: []string{
				`# TYPE ` + inboundRequestsName + ` counter`,
				inboundRequestsName + ` 2`,
				// no forwarded requests since it is a vector and we will not set any
				`# TYPE ` + forwardedResponseTimeName + ` histogram`,
				forwardedResponseTimeName + `_sum 0`,
				forwardedResponseTimeName + `_count 0`,
				forwardedResponseTimeName + `_bucket`,
			},
			githubs:      2,
			forwards:     0,
			responseTime: float64(0),
		},
	} {
		registry := prometheus.NewRegistry()
		InitMetrics(registry)

		for i := 0; i < test.githubs; i += 1 {
			IncInboundCount()
		}
		for i := 0; i < test.forwards; i += 1 {
			IncForwardedCount("host1")
		}
		if test.responseTime > 0 {
			AddForwardedResponseTime(test.responseTime)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.PanicOnError})
		rw := &fakeResponseWriter{header: http.Header{}}
		h.ServeHTTP(rw, &http.Request{})

		respStr := rw.String()

		for _, s := range test.expected {
			if !strings.Contains(respStr, s) {
				t.Errorf("testcase %s: expected string %s did not appear in %s", test.name, s, respStr)
			}
		}

	}
}
