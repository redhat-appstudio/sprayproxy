package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/test"
	"go.uber.org/zap"
)

func TestGetBackend(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	testBackend := map[string]string{
		backend1.GetServer().URL: "",
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/backends", bytes.NewBufferString("hello"))
	proxy, err := NewSprayProxy(false, true, true, zap.NewNop(), testBackend)
	if err != nil {
		t.Fatalf("failed to set up proxy: %v", err)
	}
	proxy.GetBackends(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	expected := backend1.GetServer().URL
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, expected) {
		t.Errorf("expected string %q did not appear in %q", expected, responseBody)
	}

	if backend1.GetError() != nil {
		t.Errorf("backend 1 error: %v", backend1.GetError())
	}
}

func TestRegisterBackend(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	testBackend := map[string]string{
		backend1.GetServer().URL: "",
	}
	body, _ := json.Marshal(testBackend)
	proxy, err := NewSprayProxy(false, true, true, zap.NewNop(), testBackend)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	t.Run("log 400 response for invalid json body", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		body := []byte(`{"invalid_json"}`)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewBuffer(body))
		proxy.RegisterBackend(ctx)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
		responseBody := w.Body.String()
		if responseBody != "please provide a valid json body" {
			t.Errorf("expected response %q, got %q", "please provide a valid json body", responseBody)
		}
	})

	t.Run("log 200 response while register backend server", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewBuffer(body))
		proxy.RegisterBackend(ctx)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}
		responseBody := w.Body.String()
		if responseBody != "registered the backend server" {
			t.Errorf("expected response %q, got %q", "registered the backend server", responseBody)
		}
	})

	t.Run("log 302 response while backend server already registered", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewReader(body))
		proxy.RegisterBackend(ctx)

		// re-register the backend
		w1 := httptest.NewRecorder()
		ctx1, _ := gin.CreateTestContext(w1)
		ctx1.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewReader(body))
		proxy.RegisterBackend(ctx1)

		if w1.Code != http.StatusFound {
			t.Errorf("expected status code %d, got %d", http.StatusFound, w.Code)
		}
		responseBody := w1.Body.String()
		if responseBody != "backend server already registered" {
			t.Errorf("expected response %q, got %q", "backend server already registered", responseBody)
		}
	})
}

func TestUnRegisterBackend(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	testBackend := map[string]string{
		backend1.GetServer().URL: "",
	}
	body, _ := json.Marshal(testBackend)
	proxy, err := NewSprayProxy(false, true, true, zap.NewNop(), testBackend)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	t.Run("log 400 response for invalid json body", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		body := []byte(`{"invalid_json"}`)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewBuffer(body))
		proxy.UnregisterBackend(ctx)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
		responseBody := w.Body.String()
		if responseBody != "please provide a valid json body" {
			t.Errorf("expected response %q, got %q", "please provide a valid json body", responseBody)
		}
	})

	t.Run("log 404 response while unregister backend server not registered", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodDelete, "/backends", bytes.NewReader(body))
		proxy.UnregisterBackend(ctx)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, w.Code)
		}
		responseBody := w.Body.String()
		if responseBody != "backend server not found in the list" {
			t.Errorf("expected response %q, got %q", "backend server not found in the list", responseBody)
		}
	})
	t.Run("log 200 response while unregister the registered backend server", func(t *testing.T) {
		// Register the backend server
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/backends", bytes.NewBuffer(body))
		proxy.RegisterBackend(ctx)

		// Unregister the backend server
		w1 := httptest.NewRecorder()
		ctx1, _ := gin.CreateTestContext(w1)
		ctx1.Request = httptest.NewRequest(http.MethodDelete, "/backends", bytes.NewReader(body))
		proxy.UnregisterBackend(ctx1)
		if w1.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}
		responseBody := w1.Body.String()
		if responseBody != "backend server unregistered" {
			t.Errorf("expected response %q, got %q", "backend server unregistered", responseBody)
		}
	})
}
