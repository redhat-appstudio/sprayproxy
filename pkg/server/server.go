/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/redhat-appstudio/sprayproxy/pkg/apis/proxy"
	"github.com/redhat-appstudio/sprayproxy/pkg/logger"
)

var zapLogger *zap.Logger

type SprayProxyServer struct {
	router *gin.Engine
	proxy  *proxy.SprayProxy
	host   string
	port   int
}

func init() {
	zapLogger = logger.Get()
}

func SetLogger(logger *zap.Logger) {
	zapLogger = logger
}

func NewServer(host string, port int, insecureSkipTLS, insecureSkipWebhookVerify, enableDynamicBackends bool, backends map[string]string) (*SprayProxyServer, error) {
	sprayProxy, err := proxy.NewSprayProxy(insecureSkipTLS, insecureSkipWebhookVerify, enableDynamicBackends, zapLogger, backends)
	if err != nil {
		return nil, err
	}
	// comment/uncomment to switch between debug and release mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// by default gin will trust all request headers that contain alternative client IP
	// https://pkg.go.dev/github.com/gin-gonic/gin#Engine.SetTrustedProxies
	r.SetTrustedProxies(nil)
	// https://github.com/gin-gonic/gin/issues/3336#issuecomment-1272582870
	r.TrustedPlatform = "X-Forwarded-For"
	// set middleware before routes, otherwise it does not work (gin bug).
	// The addRequestId middleware must be set before the logging middleware.
	r.Use(addRequestId())
	r.Use(ginzap.GinzapWithConfig(zapLogger, &ginzap.Config{
		Context: ginzap.Fn(func(c *gin.Context) []zapcore.Field {
			return []zapcore.Field{
				zap.String("request-id", c.GetString("requestId")),
			}
		}),
	}))
	r.Use(ginzap.RecoveryWithZap(zapLogger, true))
	r.GET("/", handleHealthz)
	r.POST("/", sprayProxy.HandleProxy)
	r.GET("/proxy", handleHealthz)
	r.POST("/proxy", sprayProxy.HandleProxyEndpoint)
	if enableDynamicBackends {
		r.GET("/backends", sprayProxy.GetBackends)
		r.POST("/backends", sprayProxy.RegisterBackend)
		r.DELETE("/backends", sprayProxy.UnregisterBackend)
	}
	r.GET("/healthz", handleHealthz)
	return &SprayProxyServer{
		router: r,
		proxy:  sprayProxy,
		host:   host,
		port:   port,
	}, nil
}

// Run launches the proxy server with the pre-configured hostname and address.
func (s *SprayProxyServer) Run(stopCh <-chan struct{}) {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	zapLogger.Info(fmt.Sprintf("Starting sprayproxy on %s", address))
	zapLogger.Info(fmt.Sprintf("Forwarding traffic to %s", strings.Join(s.proxy.Backends(), ",")))
	if s.proxy.InsecureSkipTLSVerify() {
		zapLogger.Warn("Skipping TLS verification on backends")
	}
	defer zapLogger.Sync()
	// gin.Engine does not support graceful shutdown, so we explicitly leverage http.Server
	srv := &http.Server{
		Addr:    address,
		Handler: s.router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal(fmt.Sprintf("Running sprayproxy error %v", err))
		}
	}()
	<-stopCh
	zapLogger.Info("Shutting down sprayproxy")
	// ensure graceful shutdown
	// the gin-gonic example https://gin-gonic.com/docs/examples/graceful-restart-or-stop/
	// is catching ctx.Done(), but that always blocks until the timeout expires even when
	// the server is idle, which will slowdown pod restarts
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error(fmt.Sprintf("Shutdown sprayproxy error %v", err))
	}
}

// Handler returns the http.Handler interface for the proxy server.
func (s *SprayProxyServer) Handler() http.Handler {
	return s.router
}

func handleHealthz(c *gin.Context) {
	c.String(http.StatusOK, "healthy")
}
