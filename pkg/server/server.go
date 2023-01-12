/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/adambkaplan/sprayproxy/pkg/proxy"
)

type SprayProxyServer struct {
	server *gin.Engine
	proxy  *proxy.SprayProxy
	host   string
	port   int
}

func NewServer(host string, port int, insecureSkipTLS bool, backends ...string) (*SprayProxyServer, error) {
	sprayProxy, err := proxy.NewSprayProxy(insecureSkipTLS, backends...)
	if err != nil {
		return nil, err
	}
	r := gin.Default()
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
	fmt.Printf("Running spray proxy on %s\n", address)
	fmt.Printf("Forwarding traffic to %s\n", strings.Join(s.proxy.Backends(), ","))
	if s.proxy.InsecureSkipTLSVerify() {
		fmt.Printf("WARNING: Skipping TLS verification on backends.")
	}
	return s.server.Run(address)
}

// Handler returns the http.Handler interface for the proxy server.
func (s *SprayProxyServer) Handler() http.Handler {
	return s.server
}

func handleHealthz(c *gin.Context) {
	c.String(http.StatusOK, "healthy")
}
