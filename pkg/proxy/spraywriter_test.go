/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"testing"
)

func TestSprayWriterWrite(t *testing.T) {
	writer := NewSprayWriter()
	bytes := bytes.NewBufferString("hello").Bytes()
	written, err := writer.Write(bytes)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if written != len(bytes) {
		t.Errorf("expected written bytes %d, got %d", len(bytes), written)
	}
}
