// Package mcpserver собирает MCP-сервер из наборов инструментов и отдаёт его по Streamable HTTP.
//
// Новый набор инструментов подключается в своём модуле одной строкой:
//
//	fx.Provide(mcpserver.AsToolset(NewItemTools))
package mcpserver

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

// Module даёт *mcp.Server и *Handler.
var Module = fx.Module("mcpserver",
	fx.Provide(New, NewHandler),
)

// Название и версия сервера видны платформе при подключении (initialize).
// ПОМЕНЯЙТЕ на название своей системы.
const (
	serverName    = "example-mcp"
	serverTitle   = "Пример системы"
	serverVersion = "0.1.0"
)

// instructions — общая подсказка модели про сервер целиком. Коротко: что за система
// и что важно знать до вызова любого инструмента.
const instructions = `Сервер работает с системой «Пример» от имени вошедшего сотрудника: видно только то, что видно ему.
Даты — в формате ГГГГ-ММ-ДД.`

// Toolset — группа связанных инструментов, которая сама добавляет себя на сервер.
type Toolset interface {
	Register(s *mcp.Server)
}

// AsToolset кладёт результат конструктора в группу "mcp_tools", откуда New забирает все наборы.
func AsToolset(constructor any) any {
	return fx.Annotate(constructor, fx.As(new(Toolset)), fx.ResultTags(`group:"mcp_tools"`))
}

// Params — зависимости сервера.
type Params struct {
	fx.In

	Logger     *slog.Logger
	Registerer prometheus.Registerer
	Toolsets   []Toolset `group:"mcp_tools"`
}

// New создаёт MCP-сервер со всеми инструментами.
func New(p Params) (*mcp.Server, error) {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   serverTitle,
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Instructions: instructions,
		Logger:       p.Logger,
	})

	for _, ts := range p.Toolsets {
		ts.Register(s)
	}

	calls, err := newCallObserver(p.Registerer, p.Logger)
	if err != nil {
		return nil, err
	}

	// Порядок: сначала отсекаем служебный ключ, потом считаем то, что дошло до инструментов.
	s.AddReceivingMiddleware(blockDiscoveryCalls, calls.middleware)

	return s, nil
}
