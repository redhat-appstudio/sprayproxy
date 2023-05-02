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
	server, err := server.NewServer("localhost", 8080, false, true, backend.GetServer().URL)
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
