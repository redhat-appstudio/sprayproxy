/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestServerRootPost(t *testing.T) {
	server, err := NewServer("localhost", 8080, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("hello"))
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestServerHealthz(t *testing.T) {
	server, err := NewServer("localhost", 8080, false)
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
	server, err := NewServer("localhost", 8080, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
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
