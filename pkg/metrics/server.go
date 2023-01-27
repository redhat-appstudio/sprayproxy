package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	host    string
	port    int
	crtFile string
	keyfile string
	srv     *http.Server
}

// NewServer creates the http.Server struct
func NewServer(host string, port int, crt, key string) (*MetricsServer, error) {
	if port <= 0 {
		return nil, errors.New("invalid port for metrics server")
	}

	bindAddr := fmt.Sprintf("%s:%d", host, port)
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	ms := &MetricsServer{
		host:    host,
		port:    port,
		crtFile: crt,
		keyfile: key,
		srv: &http.Server{
			Addr:    bindAddr,
			Handler: router,
		},
	}

	return ms, nil
}

// StopServer stops the metrics server
func (s *MetricsServer) StopServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		fmt.Printf("Problem shutting down HTTP server: %v", err)
	}
}

// RunServer starts the metrics server.
func (s *MetricsServer) RunServer(stopCh <-chan struct{}) {
	go func() {
		var err error
		if len(s.crtFile) > 0 && len(s.keyfile) > 0 {
			err = s.srv.ListenAndServeTLS(s.crtFile, s.keyfile)
		} else {
			err = s.srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("error starting metrics server: %v", err)
		}
	}()
	<-stopCh
	if err := s.srv.Close(); err != nil {
		fmt.Printf("error closing metrics server: %v", err)
	}
}
