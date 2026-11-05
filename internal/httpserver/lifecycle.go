package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"
)

// Bind запускает srv при старте приложения и аккуратно останавливает при выходе.
//
// Порт занимается сразу в OnStart: если он занят, приложение не стартует (а не пишет ошибку в журнал
// и работает дальше без сервера). Если сервер упал позже — останавливаем всё приложение с кодом 1.
// beforeStop (может быть nil) вызывается первым при остановке, например чтобы /readyz стал 503.
func Bind(lc fx.Lifecycle, sd fx.Shutdowner, log *slog.Logger, srv *http.Server, timeout time.Duration, beforeStop func()) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var lcfg net.ListenConfig
			ln, err := lcfg.Listen(ctx, "tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("слушать %s: %w", srv.Addr, err)
			}

			log.Info("http-сервер запущен", slog.String("addr", ln.Addr().String()))

			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("http-сервер упал", slog.Any("error", err))
					_ = sd.Shutdown(fx.ExitCode(1))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if beforeStop != nil {
				beforeStop()
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			log.Info("http-сервер останавливается")
			// Shutdown перестаёт принимать новые соединения и ждёт текущие запросы.
			return srv.Shutdown(ctx)
		},
	})
}
