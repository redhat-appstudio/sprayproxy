/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestHandleProxy(t *testing.T) {
	proxy, err := NewSprayProxy(false, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString("hello"))
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected repsonse %q, got %q", "proxied", responseBody)
	}
}

func TestHandleProxyMultiBackend(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	backend2 := test.NewTestServer()
	defer backend2.GetServer().Close()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString("hello world!"))
	proxy, err := NewSprayProxy(false, zap.NewNop(), backend1.GetServer().URL, backend2.GetServer().URL)
	if err != nil {
		t.Fatalf("failed to set up proxy: %v", err)
	}
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected response %q, got %q", "proxied", responseBody)
	}

	if backend1.GetError() != nil {
		t.Errorf("backend 1 error: %v", backend1.GetError())
	}
	if backend2.GetError() != nil {
		t.Errorf("backend 2 error: %v", backend2.GetError())
	}
}

func TestProxyLog(t *testing.T) {
	var buff bytes.Buffer
	config := zap.NewProductionConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&buff),
		config.Level,
	)
	logger := zap.New(core)
	backend := test.NewTestServer()
	defer backend.GetServer().Close()
	proxy, err := NewSprayProxy(false, logger, backend.GetServer().URL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString("hello"))
	proxy.HandleProxy(ctx)
	expected := `"msg":"proxied request"`
	log := buff.String()
	if !strings.Contains(log, expected) {
		t.Errorf("expected string %q did not appear in %q", expected, log)
	}
}
