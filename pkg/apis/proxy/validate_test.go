/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const (
	body          = `{"foo":"bar"}`
	secret        = "testSecret"
	invalidSecret = "invalidTestSecret"
)

// create GitHub webwook like HTTP request including signature
func newProxyRequest() *http.Request {
	form := url.Values{}
	form.Add("payload", body)
	formBody := form.Encode()
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/proxy", bytes.NewBufferString(formBody))
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	// signature generated by generateSignature(formBody, secret))
	req.Header.Add("x-hub-signature-256", "sha256=c92b37ae0a1bcf9373c8b968d3c973891349b3fd993e23e6febc6a43dc7517fd")
	return req
}

func expectErrorMessage(t *testing.T, msg string, err error) {
	if err == nil || err.Error() != msg {
		t.Errorf("Expected %q, got %q", msg, err)
	}
}

func generateSignature(body, secret string) string {
	const signaturePrefix = "sha256="
	// the signature is in hex form thus *2
	dst := make([]byte, sha256.Size*2)
	computed := hmac.New(sha256.New, []byte(secret))
	computed.Write([]byte(body))
	hex.Encode(dst, computed.Sum(nil))
	return signaturePrefix + string(dst)
}

func TestMissingWebhookSignature(t *testing.T) {
	r := newProxyRequest()
	t.Run("missing x-hub-signature", func(t *testing.T) {
		r.Header.Del("x-hub-signature-256")
		err := validateWebhookSignature(r, secret)
		expectErrorMessage(t, "validateWebhookSignature: missing signature", err)
	})
	t.Run("empty x-hub-signature", func(t *testing.T) {
		r.Header.Set("x-hub-signature-256", "")
		err := validateWebhookSignature(r, secret)
		expectErrorMessage(t, "validateWebhookSignature: missing signature", err)
	})
}

func TestInvalidWebhookSignature(t *testing.T) {
	r := newProxyRequest()
	err := validateWebhookSignature(r, invalidSecret)
	expectErrorMessage(t, "validateWebhookSignature: payload signature check failed", err)
}

func TestValidWebhookSignature(t *testing.T) {
	r := newProxyRequest()
	if err := validateWebhookSignature(r, secret); err != nil {
		t.Errorf("Unexpected error %q", err)
	}
}
