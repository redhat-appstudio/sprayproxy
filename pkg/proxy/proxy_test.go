/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
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
	proxy.HandleProxy(ctx)
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	responseBody := w.Body.String()
	if responseBody != "proxied" {
		t.Errorf("expected repsonse %q, got %q", "proxied", responseBody)
	}
}
