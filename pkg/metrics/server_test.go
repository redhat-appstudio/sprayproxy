package metrics

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	mr "math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	portOffset uint32 = 0
	crtFile    string
	keyFile    string
)

func TestMain(m *testing.M) {
	var err error

	mr.Seed(time.Now().UnixNano())

	keyFile, crtFile, err = generateTempCertificates()
	if err != nil {
		panic(err)
	}

	// sets the default http client to skip certificate check.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	code := m.Run()
	os.Remove(keyFile)
	os.Remove(crtFile)
	os.Exit(code)
}

func generateTempCertificates() (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key)
	if err != nil {
		return "", "", err
	}

	cert, err := ioutil.TempFile("", "testcert-")
	if err != nil {
		return "", "", err
	}
	defer cert.Close()
	pem.Encode(cert, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	keyPath, err := ioutil.TempFile("", "testkey-")
	if err != nil {
		return "", "", err
	}
	defer keyPath.Close()
	pem.Encode(keyPath, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return keyPath.Name(), cert.Name(), nil
}

func blockUntilServerStarted(port int) error {
	return wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (bool, error) {
		if _, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port)); err != nil {
			// in case error is "connection refused", server is not up (yet)
			// it is possible that it is still being started
			// in that case we need to try more
			if utilnet.IsConnectionRefused(err) {
				return false, nil
			}

			// in case of a different error, return immediately
			return true, err
		}

		// no error, stop polling the server, continue with the test logic
		return true, nil
	})
}

func runMetricsServer(t *testing.T) (int, chan<- struct{}) {
	var port int = MetricsPort + int(atomic.AddUint32(&portOffset, 1))

	ch := make(chan struct{})
	server, err := NewServer("", port, crtFile, keyFile)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	go server.RunServer(ch)

	if err := blockUntilServerStarted(port); err != nil {
		t.Fatalf("error while waiting for metrics server: %v", err)
	}

	return port, ch
}

func TestRunServer(t *testing.T) {
	port, ch := runMetricsServer(t)
	defer close(ch)

	resp, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("error while querying metrics server: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Server response status is %q instead of 200", resp.Status)
	}
}

func testServerForExpected(t *testing.T, testName string, port int, expected []metric) {
	resp, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("error requesting metrics server: %v in test %q", err, testName)
	}
	var p expfmt.TextParser
	mf, err := p.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Fatalf("error parsing server response: %v in test %q", err, testName)
	}

	for _, e := range expected {
		if mf[e.name] == nil {
			t.Fatalf("expected metric %v not found in server response: in test %q", e.name, testName)
		}
		v := *(mf[e.name].GetMetric()[0].GetCounter().Value)
		if v != e.value {
			t.Fatalf("metric value %v differs from expected %v: in test %q", v, e.value, testName)
		}
	}
}

type metric struct {
	name  string
	value float64
}

func TestMetricQueries(t *testing.T) {
	for _, test := range []struct {
		name     string
		expected []metric
		githubs  int
		forwards int
	}{
		{
			name: "One inbound, two forwards",
			expected: []metric{
				{
					name:  inboundRequestsName,
					value: 1,
				},
				{
					name:  forwardedRequestsName,
					value: 2,
				},
			},
			githubs:  1,
			forwards: 2,
		},
		{
			name: "Two githubs, zero forwards",
			expected: []metric{
				{
					name:  inboundRequestsName,
					value: 2,
				},
				// no forwards means no entries for the one since it is a CounterVec
			},
			githubs:  2,
			forwards: 0,
		},
	} {
		if inboundRequests != nil {
			prometheus.Unregister(inboundRequests)
		}
		if forwardedRequests != nil {
			prometheus.Unregister(forwardedRequests)
		}
		if responseTimes != nil {
			prometheus.Unregister(responseTimes)
		}
		initCalled = false
		InitMetrics(nil)

		for i := 0; i < test.githubs; i += 1 {
			IncInboundCount()
		}
		for i := 0; i < test.forwards; i += 1 {
			IncForwardedCount("host")
		}

		port, ch := runMetricsServer(t)
		testServerForExpected(t, test.name, port, test.expected)
		close(ch)
	}
}
