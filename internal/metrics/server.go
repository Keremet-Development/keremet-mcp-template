package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"

	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/httpserver"
)

func serve(lc fx.Lifecycle, sd fx.Shutdowner, cfg *config.Config, reg *prometheus.Registry, log *slog.Logger) {
	if cfg.MetricsAddr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{Addr: cfg.MetricsAddr, Handler: mux, ReadHeaderTimeout: httpserver.ReadHeaderTimeout}
	httpserver.Bind(lc, sd, log.With(slog.String("server", "metrics")), srv, cfg.ShutdownTimeout, nil)
}
