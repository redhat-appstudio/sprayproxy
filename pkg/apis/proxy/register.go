package proxy

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redhat-appstudio/sprayproxy/pkg/apis/proxy/v1alpha1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GetBackends gives the list of backend servers available to be proxied
func (p *SprayProxy) GetBackends(c *gin.Context) {
	backendUrls := ""
	for backend := range p.backends {
		backendUrls += backend + ", "
	}
	backendUrls = strings.TrimSuffix(backendUrls, ", ")
	c.String(http.StatusOK, "Backend urls: "+backendUrls)
}

// RegisterBackend registers the backend server to be proxied
func (p *SprayProxy) RegisterBackend(c *gin.Context) {
	zapCommonFields := []zapcore.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.Bool("dynamic-backends", p.enableDynamicBackends),
	}
	var newUrl v1alpha1.Backend
	if err := c.ShouldBindJSON(&newUrl); err != nil {
		c.String(http.StatusBadRequest, "please provide a valid json body")
		p.logger.Info("backend server register request to proxy is rejected, invalid json body", zapCommonFields...)
		return
	}
	zapCommonFields = append(zapCommonFields, zap.String("backend", newUrl.URL))
	if _, ok := p.backends[newUrl.URL]; !ok {
		if p.backends == nil {
			p.backends = map[string]string{}
		}
		p.backends[newUrl.URL] = ""
		c.String(http.StatusOK, "registered the backend server")
		p.logger.Info("server registered", zapCommonFields...)
		return
	}
	c.String(http.StatusFound, "backend server already registered")
	p.logger.Info("server already registered", zapCommonFields...)
}

// UnregisterBackend removes the backend server from the list of backend
// so that it should not be proxied anymore
func (p *SprayProxy) UnregisterBackend(c *gin.Context) {
	zapCommonFields := []zapcore.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.Bool("dynamic-backends", p.enableDynamicBackends),
	}
	var unregisterUrl v1alpha1.Backend
	if err := c.ShouldBindJSON(&unregisterUrl); err != nil {
		c.String(http.StatusBadRequest, "please provide a valid json body")
		p.logger.Info("unregister request is rejected, invalid json body", zapCommonFields...)
		return
	}
	zapCommonFields = append(zapCommonFields, zap.String("backend", unregisterUrl.URL))
	if _, ok := p.backends[unregisterUrl.URL]; !ok {
		c.String(http.StatusNotFound, "backend server not found in the list")
		p.logger.Info("server not registered")
		return
	}
	delete(p.backends, unregisterUrl.URL)
	c.String(http.StatusOK, "backend server unregistered")
	p.logger.Info("server unregistered", zapCommonFields...)
}
