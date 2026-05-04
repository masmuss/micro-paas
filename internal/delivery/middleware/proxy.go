// Package middleware provides HTTP middleware for the micro-paas application.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
)

// InstanceProxy creates middleware that proxies requests to the appropriate container.
type InstanceProxy struct {
	repo   repository.InstanceRepository
	logger *slog.Logger
	cfg    *config.Config
}

// NewInstanceProxy creates a new InstanceProxy middleware.
func NewInstanceProxy(repo repository.InstanceRepository, logger *slog.Logger, cfg *config.Config) *InstanceProxy {
	return &InstanceProxy{
		repo:   repo,
		logger: logger.With("component", "instance_proxy"),
		cfg:    cfg,
	}
}

// Handler is the middleware handler that proxies requests to the appropriate container.
func (p *InstanceProxy) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always bypass API requests
		if strings.HasPrefix(r.URL.Path, "/api") {
			next.ServeHTTP(w, r)
			return
		}

		host := strings.ToLower(r.Host)
		// Remove port if present (e.g., localhost:8080 -> localhost)
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		subdomain := p.extractSubdomain(r.Context(), host)
		if subdomain == "" {
			next.ServeHTTP(w, r)
			return
		}

		instance, err := p.repo.GetBySubdomain(r.Context(), subdomain)
		if err != nil {
			p.logger.ErrorContext(r.Context(), "instance not found", "subdomain", subdomain, "error", err)
			http.NotFound(w, r)
			return
		}

		if instance.Status != model.StatusRunning {
			p.logger.WarnContext(
				r.Context(),
				"instance not running",
				"instance",
				instance.Name,
				"status",
				instance.Status,
			)
			http.Error(w, "Instance not running", http.StatusServiceUnavailable)
			return
		}

		port := instance.Port
		if port == 0 {
			port = 80 // Default if not set
		}

		targetURL, urlParseErr := url.Parse(fmt.Sprintf("http://%s:%d", instance.Name, port))
		if urlParseErr != nil {
			p.logger.ErrorContext(r.Context(), "invalid target URL", "error", urlParseErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		p.logger.InfoContext(
			r.Context(),
			"proxying request",
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

func (p *InstanceProxy) extractSubdomain(ctx context.Context, host string) string {
	p.logger.InfoContext(ctx, "extracting subdomain", "host", host, "main_domain", p.cfg.MainDomain)

	// If it's just localhost or an IP, no subdomain
	if host == "localhost" || net.ParseIP(host) != nil {
		return ""
	}

	if host == p.cfg.MainDomain {
		return ""
	}

	if strings.HasSuffix(host, "."+p.cfg.MainDomain) {
		return strings.TrimSuffix(host, "."+p.cfg.MainDomain)
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// For localhost testing: web.localhost (len 2) -> "web"
	// For production: app.example.com (len 3) -> "app"
	return parts[0]
}
