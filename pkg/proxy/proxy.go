/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/pkg/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type BackendsFunc func() []string

type SprayProxy struct {
	backends    BackendsFunc
	insecureTLS bool
	logger      *zap.Logger
}

func NewSprayProxy(insecureTLS bool, logger *zap.Logger, backends ...string) (*SprayProxy, error) {
	backendFn := func() []string {
		return backends
	}

	return &SprayProxy{
		backends:    backendFn,
		insecureTLS: insecureTLS,
		logger:      logger,
	}, nil
}

func (p *SprayProxy) HandleProxy(c *gin.Context) {
	// currently not distinguishing between requests we can parse and those we cannot parse
	metrics.IncInboundCount()
	errors := []error{}
	zapCommonFields := []zapcore.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.Bool("insecure-tls", p.insecureTLS),
	}
	// Read in body from incoming request
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "too large: %v", err)
		p.logger.Error("request body too large", zapCommonFields...)
		return
	}
	body := buf.Bytes()

	client := &http.Client{}
	if p.insecureTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	for _, backend := range p.backends() {
		backendURL, err := url.Parse(backend)
		if err != nil {
			p.logger.Error("failed to parse backend "+err.Error(), zapCommonFields...)
			continue
		}
		copy := c.Copy()
		newURL := copy.Request.URL
		newURL.Host = backendURL.Host
		newURL.Scheme = backendURL.Scheme
		// zap always append and does not override field entries, so we create
		// per backend list of fields
		zapBackendFields := append(zapCommonFields, zap.String("backend", newURL.Host))
		newRequest, err := http.NewRequest(copy.Request.Method, newURL.String(), bytes.NewReader(body))
		if err != nil {
			p.logger.Error("failed to create request: "+err.Error(), zapBackendFields...)
			errors = append(errors, err)
			continue
		}
		newRequest.Header = copy.Request.Header
		// currently not distinguishing between requests we send and requests that return without error
		metrics.IncForwardedCount(backendURL.Host)

		// for response time, we are making it "simpler" and including everything in the client.Do call
		start := time.Now()
		resp, err := client.Do(newRequest)
		responseTime := time.Now().Sub(start)
		metrics.AddForwardedResponseTime(responseTime.Seconds())
		// standartize on what ginzap logs
		zapBackendFields = append(zapBackendFields, zap.Duration("latency", responseTime))
		if err != nil {
			p.logger.Error("proxy error: "+err.Error(), zapBackendFields...)
			errors = append(errors, err)
			continue
		}
		defer resp.Body.Close()
		zapBackendFields = append(zapBackendFields, zap.Int("status", resp.StatusCode))
		p.logger.Info("proxied request", zapBackendFields...)
		if resp.StatusCode >= 400 {
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				p.logger.Info("failed to read response: "+err.Error(), zapBackendFields...)
			} else {
				p.logger.Info("response body: "+string(respBody), zapBackendFields...)
			}
		}

		// // Create a new request with a disconnected context
		// newRequest := copy.Request.Clone(context.Background())
		// // Deep copy the request body since this needs to be read multiple times
		// newRequest.Body = io.NopCloser(bytes.NewReader(body))

		// proxy := httputil.NewSingleHostReverseProxy(backendURL)
		// proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		// 	errors = append(errors, err)
		// 	rw.WriteHeader(http.StatusBadGateway)
		// }
		// if p.insecureTLS {
		// 	proxy.Transport = &http.Transport{
		// 		TLSClientConfig: &tls.Config{
		// 			InsecureSkipVerify: true,
		// 		},
		// 	}
		// }
		// doProxy(backend, proxy, newRequest)
	}
	if len(errors) > 0 {
		// we have a bad gateway/connection somewhere
		c.String(http.StatusBadGateway, "failed to proxy: %v", errors)
		return
	}
	c.String(http.StatusOK, "proxied")
}

func (p *SprayProxy) Backends() []string {
	return p.backends()
}

// InsecureSkipTLSVerify indicates if the proxy is skipping TLS verification.
// This setting is insecure and should not be used in production.
func (p *SprayProxy) InsecureSkipTLSVerify() bool {
	return p.insecureTLS
}

// doProxy proxies the provided request to a backend, with response data to an "empty" response instance.
func doProxy(dest string, proxy *httputil.ReverseProxy, req *http.Request) {
	writer := NewSprayWriter()
	proxy.ServeHTTP(writer, req)
	fmt.Printf("proxied %s to backend %s\n", req.URL, dest)
}
