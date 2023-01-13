/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleProxy(t *testing.T) {
	proxy, err := NewSprayProxy(false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "localhost:8080", bytes.NewBufferString("hello"))
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected repsonse %q, got %q", "proxied", responseBody)
	}
}

type testBackend struct {
	server *httptest.Server
	err    error
}

func (b *testBackend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(req.Body)
	defer req.Body.Close()
	if err != nil {
		b.err = err
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func TestHandleProxyMultiBackend(t *testing.T) {
	backend1 := newTestServer()
	defer backend1.server.Close()
	backend2 := newTestServer()
	defer backend2.server.Close()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "localhost:8080", bytes.NewBufferString("hello world!"))
	proxy, err := NewSprayProxy(false, backend1.server.URL, backend2.server.URL)
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

	if backend1.err != nil {
		t.Errorf("backend 1 error: %v", backend1.err)
	}
	if backend2.err != nil {
		t.Errorf("backend 2 error: %v", backend2.err)
	}
}

func newTestServer() *testBackend {
	testServer := &testBackend{}
	mux := http.NewServeMux()
	mux.Handle("/proxy", testServer)
	testServer.server = httptest.NewServer(mux)
	return testServer
}
