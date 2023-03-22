/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import "net/http"

type SprayWriter struct {
	http.ResponseWriter
	response *http.Response
}

func NewSprayWriter() *SprayWriter {
	return &SprayWriter{
		response: &http.Response{
			Header: http.Header{},
		},
	}
}

func (w *SprayWriter) Header() http.Header {
	return w.response.Header
}

func (w *SprayWriter) Write(body []byte) (int, error) {
	if w.response.StatusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return len(body), nil
}

func (w *SprayWriter) WriteHeader(statusCode int) {
	w.response.StatusCode = statusCode
}
