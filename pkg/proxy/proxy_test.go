/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestProxyDefaultTimeoutNoEnv(t *testing.T) {
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedTmout := "15s"
	gotTmout := proxy.fwdReqTmout.String()
	if expectedTmout != gotTmout {
		t.Errorf("expected timeout %q, got %q", expectedTmout, gotTmout)
	}
}

func TestProxyDefaultTimeoutBadEnv(t *testing.T) {
	// "foo" is not a time.Duration value and should be ignored
	t.Setenv("SPRAYPROXY_FORWARDING_REQUEST_TIMEOUT", "foo")
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedTmout := "15s"
	gotTmout := proxy.fwdReqTmout.String()
	if expectedTmout != gotTmout {
		t.Errorf("expected timeout %q, got %q", expectedTmout, gotTmout)
	}
}

func TestProxyCustomTimeout(t *testing.T) {
	t.Setenv("SPRAYPROXY_FORWARDING_REQUEST_TIMEOUT", "90s")
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedTmout := "1m30s"
	gotTmout := proxy.fwdReqTmout.String()
	if expectedTmout != gotTmout {
		t.Errorf("expected timeout %q, got %q", expectedTmout, gotTmout)
	}
}

func TestProxyNoWebhookSecret(t *testing.T) {
	// removing the env var is not strictly required, making it explicit
	os.Unsetenv(envWebhookSecret)
	_, err := NewSprayProxy(false, false, zap.NewNop())
	expectedError := "no webhook secret"
	if err == nil || err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err)
	}
}

func TestProxyWebhookSecret(t *testing.T) {
	secret := "testSecret"
	t.Setenv("GH_APP_WEBHOOK_SECRET", secret)
	p, err := NewSprayProxy(false, false, zap.NewNop())
	if err != nil {
		t.Errorf("Unexpected error %q", err)
	}
	if p.webhookSecret != secret {
		t.Errorf("Expected secret %q, got %q", secret, p.webhookSecret)
	}
}

func TestHandleProxyEndpoint(t *testing.T) {
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = newProxyRequest()
	proxy.HandleProxyEndpoint(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected response %q, got %q", "proxied", responseBody)
	}
}

func TestHandleProxy(t *testing.T) {
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = newProxyRequest()
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected response %q, got %q", "proxied", responseBody)
	}
}

func TestHandleProxyMultiBackend(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	backend2 := test.NewTestServer()
	defer backend2.GetServer().Close()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = newProxyRequest()
	proxy, err := NewSprayProxy(false, true, zap.NewNop(), backend1.GetServer().URL, backend2.GetServer().URL)
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

func TestLargePayloadOnLimit(t *testing.T) {
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBuffer(make([]byte, maxReqSize)))
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected response %q, got %q", "proxied", responseBody)
	}
}

func TestLargePayloadAboveLimit(t *testing.T) {
	proxy, err := NewSprayProxy(false, true, zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBuffer(make([]byte, maxReqSize+1)))
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status code %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
	expectedBody := "request body too large"
	responseBody := w.Body.String()
	if responseBody != expectedBody {
		t.Errorf("expected response %q, got %q", expectedBody, responseBody)
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
	proxy, err := NewSprayProxy(false, true, logger, backend.GetServer().URL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	t.Run("proxy log for root endpoint", func(t *testing.T) {
		buff.Reset()
		ctx.Request = newProxyRequest()
		proxy.HandleProxy(ctx)
		expected := `"msg":"proxied request"`
		log := buff.String()
		if !strings.Contains(log, expected) {
			t.Errorf("expected string %q did not appear in %q", expected, log)
		}
	})
	t.Run("proxy log for /proxy endpoint", func(t *testing.T) {
		buff.Reset()
		ctx.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080/proxy/apis", bytes.NewBuffer(make([]byte, maxReqSize)))
		proxy.HandleProxyEndpoint(ctx)
		expected := `"msg":"proxied request"`
		unexpectedBackend := backend.GetServer().URL + "/proxy/apis"
		log := buff.String()
		if !strings.Contains(log, expected) {
			t.Errorf("expected string %q did not appear in %q", expected, log)
		}

		if strings.Contains(log, unexpectedBackend) {
			t.Errorf("proxy forwarded request to unexpected backend: %q, has /proxy path prefix", unexpectedBackend)
		}
	})

}
