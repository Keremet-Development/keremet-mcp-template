// Package tools — инструменты сервера. Пример: whoami (профиль) и search_items (поиск записей).
//
// Новый инструмент: метод с сигнатурой
//
//	func(ctx, *mcp.CallToolRequest, ВходнаяСтруктура) (*mcp.CallToolResult, ВыходнаяСтруктура, error)
//
// и строка mcp.AddTool в Register. Схему аргументов go-sdk строит сам по полям структуры
// (теги json и jsonschema — описание поля для модели).
package tools

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/fx"

	"example.com/mcp-template/internal/mcpserver"
	"example.com/mcp-template/internal/system"
)

// Module добавляет инструменты на MCP-сервер.
var Module = fx.Module("tools",
	fx.Provide(mcpserver.AsToolset(New)),
)

// Tools — инструменты, работающие с системой.
type Tools struct {
	system *system.Client
	log    *slog.Logger
}

// New создаёт набор инструментов.
func New(c *system.Client, log *slog.Logger) *Tools {
	return &Tools{system: c, log: log}
}

// Register добавляет инструменты на сервер (реализует mcpserver.Toolset).
func (t *Tools) Register(s *mcp.Server) {
	t.registerProfile(s)
	t.registerItems(s)
}
