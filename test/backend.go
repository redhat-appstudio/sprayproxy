/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

type testBackend struct {
	server  *httptest.Server
	reqBody string
	err     error
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
	b.reqBody = buf.String()
	rw.WriteHeader(http.StatusOK)
}

func (b *testBackend) GetServer() *httptest.Server {
	return b.server
}

func (b *testBackend) GetError() error {
	return b.err
}

func (b *testBackend) GetReqBody() string {
	return b.reqBody
}

func NewTestServer() *testBackend {
	testServer := &testBackend{}
	mux := http.NewServeMux()
	mux.Handle("/", testServer)
	testServer.server = httptest.NewServer(mux)
	return testServer
}
