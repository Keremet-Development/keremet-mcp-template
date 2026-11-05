// Package logger даёт *slog.Logger: JSON в stdout, уровень из LOG_LEVEL.
package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"example.com/mcp-template/internal/config"
)

// Module даёт *slog.Logger и направляет в него журнал самого fx.
var Module = fx.Module("logger",
	fx.Provide(New),
)

// WithFxLogger пишет события fx (запуск модулей, ошибки сборки) в тот же slog.
// Подключается в app рядом с остальными модулями.
var WithFxLogger = fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
	l := &fxevent.SlogLogger{Logger: log}
	l.UseLogLevel(slog.LevelDebug) // штатные события fx — только в debug
	return l
})

// New создаёт логгер и делает его логгером по умолчанию.
func New(cfg *config.Config) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("LOG_LEVEL=%q: %w", cfg.LogLevel, err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	return log, nil
}
