/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"fmt"
	"net/http"
	"strings"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/redhat-appstudio/sprayproxy/pkg/logger"
	"github.com/redhat-appstudio/sprayproxy/pkg/proxy"
)

var zapLogger *zap.Logger

type SprayProxyServer struct {
	server *gin.Engine
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

func NewServer(host string, port int, insecureSkipTLS bool, backends ...string) (*SprayProxyServer, error) {
	sprayProxy, err := proxy.NewSprayProxy(insecureSkipTLS, zapLogger, backends...)
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
	r.GET("/healthz", handleHealthz)
	return &SprayProxyServer{
		server: r,
		proxy:  sprayProxy,
		host:   host,
		port:   port,
	}, nil
}

// Run launches the proxy server with the pre-configured hostname and address.
func (s *SprayProxyServer) Run() error {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	zapLogger.Info(fmt.Sprintf("Running spray proxy on %s", address))
	zapLogger.Info(fmt.Sprintf("Forwarding traffic to %s", strings.Join(s.proxy.Backends(), ",")))
	if s.proxy.InsecureSkipTLSVerify() {
		zapLogger.Warn("Skipping TLS verification on backends")
	}
	defer zapLogger.Sync()
	return s.server.Run(address)
}

// Handler returns the http.Handler interface for the proxy server.
func (s *SprayProxyServer) Handler() http.Handler {
	return s.server
}

func handleHealthz(c *gin.Context) {
	c.String(http.StatusOK, "healthy")
}
