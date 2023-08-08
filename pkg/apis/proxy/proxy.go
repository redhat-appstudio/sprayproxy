/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/pkg/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// GitHub webhook validation secret
	envWebhookSecret = "GH_APP_WEBHOOK_SECRET"
)

type SprayProxy struct {
	backends              map[string]string
	insecureTLS           bool
	insecureWebhook       bool
	enableDynamicBackends bool
	webhookSecret         string
	logger                *zap.Logger
	fwdReqTmout           time.Duration
	maxReqSize            int
}

func NewSprayProxy(insecureTLS, insecureWebhook, enableDynamicBackends bool, logger *zap.Logger, backends map[string]string) (*SprayProxy, error) {

	var webhookSecret string
	if !insecureWebhook {
		if secret := os.Getenv(envWebhookSecret); secret == "" {
			// if validation is enabled, but no secret found
			logger.Error("webhook validation enabled, but no secret found")
			return nil, errors.New("no webhook secret")
		} else {
			webhookSecret = secret
		}
	}

	// forwarding request timeout of 15s, can be overriden by SPRAYPROXY_FORWARDING_REQUEST_TIMEOUT env var
	fwdReqTmout := 15 * time.Second
	if duration, err := time.ParseDuration(os.Getenv("SPRAYPROXY_FORWARDING_REQUEST_TIMEOUT")); err == nil {
		fwdReqTmout = duration
	}
	logger.Info(fmt.Sprintf("proxy forwarding request timeout set to %s", fwdReqTmout.String()))

	// GitHub limits webhook request size to 25MB. Use that as default.
	maxReqSize := 1024 * 1024 * 25
	if maxReqSizeFromEnv, err := strconv.Atoi(os.Getenv("SPRAYPROXY_MAX_REQUEST_SIZE")); err == nil {
		maxReqSize = maxReqSizeFromEnv
	}
	logger.Info(fmt.Sprintf("proxy max request size set to %d bytes (%.2fMB)", maxReqSize, float64(maxReqSize)/(1<<20)))

	return &SprayProxy{
		backends:              backends,
		insecureTLS:           insecureTLS,
		insecureWebhook:       insecureWebhook,
		enableDynamicBackends: enableDynamicBackends,
		webhookSecret:         webhookSecret,
		logger:                logger,
		fwdReqTmout:           fwdReqTmout,
		maxReqSize:            maxReqSize,
	}, nil
}

func (p *SprayProxy) HandleProxy(c *gin.Context) {
	handleProxyCommon(p, c)
}

func (p *SprayProxy) HandleProxyEndpoint(c *gin.Context) {
	// if server post on non root endpoint e.g /proxy
	// remove /proxy from the copied backend URL
	c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/proxy")
	handleProxyCommon(p, c)
}

func (p *SprayProxy) Backends() []string {
	backends := []string{}
	for b, _ := range p.backends {
		backends = append(backends, b)
	}
	return backends
}

// InsecureSkipTLSVerify indicates if the proxy is skipping TLS verification.
// This setting is insecure and should not be used in production.
func (p *SprayProxy) InsecureSkipTLSVerify() bool {
	return p.insecureTLS
}

// handleProxyCommon handles the core proxying functionality
func handleProxyCommon(p *SprayProxy, c *gin.Context) {
	// currently not distinguishing between requests we can parse and those we cannot parse
	metrics.IncInboundCount()
	errors := []error{}
	zapCommonFields := []zapcore.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.Bool("insecure-tls", p.insecureTLS),
		zap.Bool("insecure-webhook", p.insecureWebhook),
		zap.String("request-id", c.GetString("requestId")),
	}

	// Body from incoming request can only be read once, store it in a buf for re-use
	buf := &bytes.Buffer{}
	// Verify request size. If larger than limit, subsequent read will fail.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(p.maxReqSize))
	defer c.Request.Body.Close()
	_, err := buf.ReadFrom(c.Request.Body)
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "request body too large")
		p.logger.Error(err.Error(), zapCommonFields...)
		return
	}
	body := buf.Bytes()

	// validate incoming request
	if !p.insecureWebhook {
		// restore request body
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if err := validateWebhookSignature(c.Request, p.webhookSecret); err != nil {
			// we do not want to expose internal information, so returning generic failure message
			c.String(http.StatusBadRequest, "bad request")
			p.logger.Error(fmt.Sprintf("bad request: %v", err), zapCommonFields...)
			return
		}
	}

	client := &http.Client{
		// set forwarding request timeout
		Timeout: p.fwdReqTmout,
	}
	if p.insecureTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	for backend, _ := range p.backends {
		fwdErr := ""
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

		// for response time, we are making it "simpler" and including everything in the client.Do call
		start := time.Now()
		resp, err := client.Do(newRequest)
		responseTime := time.Now().Sub(start)
		// standartize on what ginzap logs
		zapBackendFields = append(zapBackendFields, zap.Duration("latency", responseTime))
		if err != nil {
			fwdErr = "non-http-error"
			metrics.IncForwardedCount(backendURL.Host, fwdErr)
			p.logger.Error("proxy error: "+err.Error(), zapBackendFields...)
			errors = append(errors, err)
			continue
		}
		defer resp.Body.Close()
		zapBackendFields = append(zapBackendFields, zap.Int("status", resp.StatusCode))
		p.logger.Info("proxied request", zapBackendFields...)
		if resp.StatusCode >= 400 {
			fwdErr = "http-error"
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				p.logger.Info("failed to read response: "+err.Error(), zapBackendFields...)
			} else {
				p.logger.Info("response body: "+string(respBody), zapBackendFields...)
			}
		}
		metrics.IncForwardedCount(backendURL.Host, fwdErr)
		metrics.AddForwardedResponseTime(responseTime.Seconds())

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
		c.String(http.StatusBadGateway, "failed to proxy")
		return
	}
	c.String(http.StatusOK, "proxied")
}

// doProxy proxies the provided request to a backend, with response data to an "empty" response instance.
func doProxy(dest string, proxy *httputil.ReverseProxy, req *http.Request) {
	writer := NewSprayWriter()
	proxy.ServeHTTP(writer, req)
	fmt.Printf("proxied %s to backend %s\n", req.URL, dest)
}
