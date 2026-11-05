// Package config читает настройки сервера из переменных окружения.
//
// Все переменные перечислены в README и в .env.example. Отдельной библиотеки нет:
// os.Getenv и значения по умолчанию.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"
)

// Module даёт *Config остальным модулям.
var Module = fx.Module("config", fx.Provide(Load))

// AuthMode — как платформа входит в сервер. Совпадает со «способом входа» в консоли платформы.
type AuthMode string

const (
	// AuthUserToken — платформа передаёт токен сотрудника, сервер проверяет его в своей системе
	// и работает с правами этого сотрудника.
	AuthUserToken AuthMode = "user_token"
	// AuthServiceToken — платформа передаёт постоянный служебный ключ (SERVICE_TOKEN).
	AuthServiceToken AuthMode = "service_token"
	// AuthNone — без входа. Только для внутренней сети и открытых данных.
	AuthNone AuthMode = "none"
)

// Config — все настройки сервера.
type Config struct {
	// Addr — где слушать MCP и /healthz, /readyz. Например ":8080".
	Addr string
	// MetricsAddr — где отдавать /metrics, например ":9090". Пусто — метрики не отдаются.
	MetricsAddr string
	// Path — путь MCP-эндпоинта. Этот адрес вписывают в консоли платформы.
	Path string

	AuthMode AuthMode
	// ServiceToken — служебный ключ платформы. В режиме service_token обязателен;
	// в режиме user_token необязателен: с ним платформа заранее узнаёт список инструментов.
	ServiceToken string

	// SystemURL — адрес API вашей системы, например https://crm.example.lan/api.
	SystemURL string
	// SystemToken — ключ сервера к API системы для режимов service_token и none
	// (когда токена сотрудника нет). В режиме user_token не используется.
	SystemToken string
	// SystemTimeout — сколько ждать ответа системы.
	SystemTimeout time.Duration

	// LogLevel — debug, info, warn или error.
	LogLevel string
	// ShutdownTimeout — сколько ждать завершения текущих запросов при остановке.
	ShutdownTimeout time.Duration
}

// Load читает окружение и проверяет, что настройки согласованы.
func Load() (*Config, error) {
	cfg := &Config{
		Addr:            env("ADDR", ":8080"),
		MetricsAddr:     env("METRICS_ADDR", ":9090"),
		Path:            env("MCP_PATH", "/mcp"),
		AuthMode:        AuthMode(env("AUTH_MODE", string(AuthUserToken))),
		ServiceToken:    os.Getenv("SERVICE_TOKEN"),
		SystemURL:       os.Getenv("SYSTEM_URL"),
		SystemToken:     os.Getenv("SYSTEM_TOKEN"),
		LogLevel:        env("LOG_LEVEL", "info"),
		SystemTimeout:   15 * time.Second,
		ShutdownTimeout: 20 * time.Second,
	}

	var err error
	if cfg.SystemTimeout, err = duration("SYSTEM_TIMEOUT", cfg.SystemTimeout); err != nil {
		return nil, err
	}
	if cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ловит ошибки настройки при запуске, а не на первом запросе.
func (c *Config) Validate() error {
	if c.SystemURL == "" {
		return errors.New("SYSTEM_URL не задан: укажите адрес API вашей системы")
	}

	switch c.AuthMode {
	case AuthUserToken, AuthNone:
		return nil
	case AuthServiceToken:
		if c.ServiceToken == "" {
			return errors.New("AUTH_MODE=service_token, но SERVICE_TOKEN пуст")
		}
		return nil
	default:
		return fmt.Errorf("AUTH_MODE=%q: ожидается user_token, service_token или none", c.AuthMode)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: ожидается длительность вида 15s: %w", key, v, err)
	}
	return d, nil
}
