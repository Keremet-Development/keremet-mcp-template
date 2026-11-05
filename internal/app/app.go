// Package app собирает приложение из модулей. Новый модуль подключается здесь одной строкой.
package app

import (
	"go.uber.org/fx"

	"example.com/mcp-template/internal/auth"
	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/httpserver"
	"example.com/mcp-template/internal/logger"
	"example.com/mcp-template/internal/mcpserver"
	"example.com/mcp-template/internal/metrics"
	"example.com/mcp-template/internal/system"
	"example.com/mcp-template/internal/tools"
)

// Module — приложение целиком.
var Module = fx.Options(
	// Основа.
	config.Module,
	logger.Module,
	logger.WithFxLogger,
	metrics.Module,

	// Система и вход.
	system.Module,
	auth.Module,

	// MCP и инструменты.
	mcpserver.Module,
	tools.Module,

	// HTTP: MCP-эндпоинт и пробы.
	httpserver.Module,
)
