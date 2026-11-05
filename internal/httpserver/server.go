// Package httpserver — основной HTTP-сервер: MCP-эндпоинт, /healthz и /readyz.
package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/fx"

	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/mcpserver"
)

// ReadHeaderTimeout защищает от медленных клиентов, которые держат соединение, не досылая заголовки.
const ReadHeaderTimeout = 10 * time.Second

// Module запускает основной сервер.
var Module = fx.Module("httpserver",
	fx.Provide(NewHealth),
	fx.Invoke(serve),
)

func serve(
	lc fx.Lifecycle,
	sd fx.Shutdowner,
	cfg *config.Config,
	log *slog.Logger,
	mcpHandler *mcpserver.Handler,
	health *Health,
) {
	mux := http.NewServeMux()
	// Без метода: Streamable HTTP — это POST (запросы) и DELETE (закрытие сессии).
	mux.Handle(mcpHandler.Path, mcpHandler)
	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: ReadHeaderTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	log = log.With(slog.String("server", "api"))
	Bind(lc, sd, log, srv, cfg.ShutdownTimeout, health.markNotReady)
	// Хуки запуска идут по порядку добавления: «готов» — только когда порт уже занят.
	lc.Append(fx.StartHook(health.markReady))
}
