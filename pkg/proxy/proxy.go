/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type BackendsFunc func() []string

type SprayProxy struct {
	backends     BackendsFunc
	inesecureTLS bool
}

func NewSprayProxy(insecureTLS bool, backends ...string) (*SprayProxy, error) {
	backendFn := func() []string {
		return backends
	}

	return &SprayProxy{
		backends:     backendFn,
		inesecureTLS: insecureTLS,
	}, nil
}

func (p *SprayProxy) HandleProxy(c *gin.Context) {
	for _, backend := range p.backends() {
		url, err := url.Parse(backend)
		if err != nil {
			continue
		}
		copy := c.Copy()
		// Create a new request with a disconnected context
		newRequest := copy.Request.WithContext(context.Background())
		proxy := httputil.NewSingleHostReverseProxy(url)
		if p.inesecureTLS {
			proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}
		go doProxy(backend, proxy, newRequest)
	}
	c.String(http.StatusOK, "proxied")
}

// doProxy proxies the provided request to a backend, with response data to an "empty" response instance.
func doProxy(dest string, proxy *httputil.ReverseProxy, req *http.Request) {
	writer := NewSprayWriter()
	proxy.ServeHTTP(writer, req)
	fmt.Printf("proxied %s to backend %s\n", req.URL, dest)
}
