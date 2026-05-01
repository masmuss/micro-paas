// Package middleware provides HTTP middleware for the micro-paas application.
package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
)

// InstanceProxy creates middleware that proxies requests to the appropriate container.
type InstanceProxy struct {
	repo   repository.InstanceRepository
	logger *slog.Logger
}

// NewInstanceProxy creates a new InstanceProxy middleware.
func NewInstanceProxy(repo repository.InstanceRepository, logger *slog.Logger) *InstanceProxy {
	return &InstanceProxy{repo: repo, logger: logger.With("component", "instance_proxy")}
}

// Handler is the middleware handler that proxies requests to the appropriate container.
func (p *InstanceProxy) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		subdomain := extractSubdomain(host)
		if subdomain == "" {
			next.ServeHTTP(w, r)
			return
		}
		instance, err := p.repo.GetBySubdomain(r.Context(), subdomain)
		if err != nil {
			p.logger.Error("instance not found", "subdomain", subdomain, "error", err)
			http.NotFound(w, r)
			return
		}
		if instance.Status != model.StatusRunning {
			p.logger.Warn("instance not running", "instance", instance.Name, "status", instance.Status)
			http.Error(w, "Instance not running", http.StatusServiceUnavailable)
			return
		}

		port := instance.Port
		if port == 0 {
			port = 80 // Default if not set
		}

		targetURL, urlParseErr := url.Parse(fmt.Sprintf("http://%s:%d", instance.ContainerID[:12], port))
		if urlParseErr != nil {
			p.logger.Error("invalid target URL", "error", urlParseErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		p.logger.Info("proxying request",
			"method", r.Method,
			"subdomain", subdomain,
			"target", targetURL.String(),
		)

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		proxy.ServeHTTP(w, r)
	})
}

func extractSubdomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}
