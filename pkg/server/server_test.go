/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	server, err := NewServer("localhost", 8080, false, false)
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
	server, err := NewServer("localhost", 8080, false, true)
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
	server, err := NewServer("localhost", 8080, false, true)
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
