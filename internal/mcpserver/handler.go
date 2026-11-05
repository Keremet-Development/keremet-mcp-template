package mcpserver

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/mcp-template/internal/auth"
	"example.com/mcp-template/internal/config"
)

// Handler — HTTP-обработчик MCP-эндпоинта (по умолчанию /mcp) вместе с проверкой токена.
type Handler struct {
	Path string
	http.Handler
}

// NewHandler отдаёт сервер по Streamable HTTP.
//
// Stateless: сервер не хранит сессий в памяти, поэтому любой экземпляр за балансировщиком
// ответит на любой запрос, а перезапуск не рвёт работу агентов.
// JSONResponse: ответ — обычный JSON, а не поток SSE; платформа понимает оба.
func NewHandler(cfg *config.Config, s *mcp.Server, guard *auth.Guard) *Handler {
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return &Handler{Path: cfg.Path, Handler: guard.Wrap(h)}
}
