/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/redhat-appstudio/sprayproxy/pkg/server"
	"github.com/redhat-appstudio/sprayproxy/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogRequestId(t *testing.T) {
	var buff bytes.Buffer
	config := zap.NewProductionConfig()
	// construct core to make it use the buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&buff),
		config.Level,
	)
	logger := zap.New(core)
	server.SetLogger(logger)
	backend := test.NewTestServer()
	defer backend.GetServer().Close()
	testBackend := map[string]string{
		backend.GetServer().URL: "",
	}
	server, err := server.NewServer("localhost", 8080, false, true, false, testBackend)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	buff.Reset()
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("hello"))
	server.Handler().ServeHTTP(w, req)
	log := strings.TrimSuffix(buff.String(), "\n")
	logLines := strings.Split(log, "\n")
	if len(logLines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(logLines))
	}
	var line1, line2 map[string]any
	json.Unmarshal([]byte(logLines[0]), &line1)
	json.Unmarshal([]byte(logLines[1]), &line2)
	if line1["request-id"] == "" || line2["request-id"] == "" {
		t.Errorf("request-id not set: %s, %s", line1["request-id"], line2["request-id"])
	}
	if line1["request-id"] != line2["request-id"] {
		t.Errorf("request-id does not match: %s, %s", line1["request-id"], line2["request-id"])
	}
}

// test original request body is matching the forwarded requests
func TestBackendRequestBody(t *testing.T) {
	backend1 := test.NewTestServer()
	defer backend1.GetServer().Close()
	backend2 := test.NewTestServer()
	defer backend2.GetServer().Close()
	testBackend := map[string]string{
		backend1.GetServer().URL: "",
		backend2.GetServer().URL: "",
	}
	server, err := server.NewServer("localhost", 8080, false, true, true, testBackend)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	form := url.Values{}
	form.Add("payload", `{"foo":"bar"}`)
	reqBody := form.Encode()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	server.Handler().ServeHTTP(w, req)
	if reqBody != backend1.GetReqBody() {
		t.Errorf("first backend, forwarded request does not match, want %q, got %q", reqBody, backend1.GetReqBody())
	}
	if reqBody != backend2.GetReqBody() {
		t.Errorf("first backend, forwarded request does not match, want %q, got %q", reqBody, backend2.GetReqBody())
	}
}

func TestServerProxyEndpoint(t *testing.T) {
	backend := test.NewTestServer()
	defer backend.GetServer().Close()
	testBackend := map[string]string{
		backend.GetServer().URL: "",
	}
	server, err := server.NewServer("localhost", 8080, false, true, false, testBackend)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/proxy", bytes.NewBufferString("hello"))
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected repsonse %q, got %q", "proxied", responseBody)
	}
}
