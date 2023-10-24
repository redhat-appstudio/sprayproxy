/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
)

// create GitHub webhook like HTTP request including signature
func newProxyRequest() *http.Request {
	form := url.Values{}
	form.Add("payload", `{"foo":"bar"}`)
	formBody := form.Encode()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(formBody))
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("x-hub-signature-256", "sha256=c92b37ae0a1bcf9373c8b968d3c973891349b3fd993e23e6febc6a43dc7517fd")
	return req
}

func TestServerRootPost(t *testing.T) {
	// override default logger with a nop one
	zapLogger = zap.NewNop()
	t.Setenv("GH_APP_WEBHOOK_SECRET", "testSecret")
	server, err := NewServer("localhost", 8080, false, false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	req := newProxyRequest()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestServerHealthz(t *testing.T) {
	// override default logger with a nop one
	zapLogger = zap.NewNop()
	server, err := NewServer("localhost", 8080, false, true, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", bytes.NewBufferString("hello"))
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	zapLogger = zap.NewNop()
	port := 8080
	server, err := NewServer("localhost", port, false, true, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := gin.New()
	// create a slow handler, so we can signal shutdown while query is handled
	r.GET("/", func(c *gin.Context) {
		time.Sleep(time.Second * 2)
		c.String(http.StatusOK, "foo")
	})
	server.router = r
	stopCh := make(chan struct{})
	go server.Run(stopCh)

	if err := blockUntilServerStarted(port); err != nil {
		t.Fatalf("error while waiting for server: %v", err)
	}

	respCh := make(chan *http.Response, 1)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err != nil {
			t.Fatalf("error making client request: %v", err)
		}
		respCh <- resp
	}()
	// Allow client to perform query before we signal shutdown. Must be shorter than
	// the sleep in handler, so server shutdown is initiated while handling a request.
	time.Sleep(time.Second)
	close(stopCh)
	resp := <-respCh
	if (*resp).StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, (*resp).StatusCode)
	}
}

func blockUntilServerStarted(port int) error {
	return wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (bool, error) {
		if _, err := http.Get(fmt.Sprintf("http://localhost:%d/", port)); err != nil {
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

func TestGetRegisteredServer(t *testing.T) {
	// override default logger with a nop one
	zapLogger = zap.NewNop()
	Data := map[string]interface{}{
		"url": "https://test.com",
	}
	data, _ := json.Marshal(Data)
	t.Run("Get Backend request when enable-dynamic-backends is unset", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodGet, "/backends", bytes.NewReader(data))
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("Get Backend request when enable-dynamic-backends is set", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, true, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodGet, "/backends", bytes.NewReader(data))
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestServerRegister(t *testing.T) {
	// override default logger with a nop one
	zapLogger = zap.NewNop()
	Data := map[string]interface{}{
		"url": "https://test.com",
	}
	data, _ := json.Marshal(Data)
	t.Run("Register request when enable-dynamic-backends is unset", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodPost, "/backends", bytes.NewReader(data))
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("Register request when enable-dynamic-backends is set", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, true, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodPost, "/backends", bytes.NewReader(data))
		server.Handler().ServeHTTP(w, req)
		if !(w.Code == http.StatusOK || w.Code == http.StatusFound) {
			t.Errorf("expected status code %d or %d, got %d", http.StatusOK, http.StatusFound, w.Code)
		}
	})
}

func TestServerUnregister(t *testing.T) {
	// override default logger with a nop one
	zapLogger = zap.NewNop()
	Data := map[string]interface{}{
		"url": "https://test.com",
	}
	data, _ := json.Marshal(Data)
	t.Run("Unregister request when enable-dynamic-backends is unset", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodDelete, "/backends", bytes.NewBuffer(data))
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("Unregister request when enable-dynamic-backends is set", func(t *testing.T) {
		w := httptest.NewRecorder()
		server, err := NewServer("localhost", 8080, false, true, true, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, _ := http.NewRequest(http.MethodDelete, "/backends", bytes.NewBuffer(data))
		server.Handler().ServeHTTP(w, req)
		if !(w.Code == http.StatusOK || w.Code == http.StatusNotFound) {
			t.Errorf("expected status code %d or %d, got %d", http.StatusOK, http.StatusNotFound, w.Code)
		}
	})

}

func TestServerAccessLog(t *testing.T) {
	var buff bytes.Buffer
	config := zap.NewProductionConfig()
	// construct core to make it use the buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&buff),
		config.Level,
	)
	logger := zap.New(core)
	zapLogger = logger
	server, err := NewServer("localhost", 8080, false, true, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	t.Run("log 200 response for proxy endpoint health check", func(t *testing.T) {
		buff.Reset()
		req, _ := http.NewRequest(http.MethodGet, "/proxy", nil)
		server.Handler().ServeHTTP(w, req)
		expected := `"msg":"/proxy"`
		log := buff.String()
		if !strings.Contains(log, expected) {
			t.Errorf("expected string %q did not appear in %q", expected, log)
		}
	})
	t.Run("log 200 response", func(t *testing.T) {
		buff.Reset()
		req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
		server.Handler().ServeHTTP(w, req)
		expected := `"msg":"/healthz"`
		log := buff.String()
		if !strings.Contains(log, expected) {
			t.Errorf("expected string %q did not appear in %q", expected, log)
		}
	})
	t.Run("log 404 response", func(t *testing.T) {
		buff.Reset()
		req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
		server.Handler().ServeHTTP(w, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `"msg":"/nonexistent"`
		log := buff.String()
		if !strings.Contains(log, expected) {
			t.Errorf("expected string %q did not appear in %q", expected, log)
		}
	})
}
