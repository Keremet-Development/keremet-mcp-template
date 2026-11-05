// Package auth решает, кто обращается к серверу, и пропускает к MCP только тех, кого пустили.
//
// Платформа присылает заголовок Authorization: Bearer <токен>. Что в нём — зависит от способа
// входа, выбранного в консоли платформы (AUTH_MODE):
//
//	user_token    — токен сотрудника; проверяем его в системе (GET /me) и работаем с его правами.
//	                Служебный ключ (SERVICE_TOKEN) тоже принимаем, но только для списка инструментов:
//	                так платформа узнаёт о новых инструментах без сотрудника.
//	service_token — постоянный служебный ключ; сравниваем с SERVICE_TOKEN.
//	none          — без проверки.
//
// Отказ — всегда HTTP 401: по нему платформа понимает, что токен не принят.
package auth

import (
	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/mcp-template/internal/system"
)

// callerKey — ключ, под которым Caller лежит в TokenInfo.Extra.
const callerKey = "caller"

// Caller — кто вызывает инструмент.
type Caller struct {
	// User — сотрудник, чей токен прислала платформа. nil — вызов не от сотрудника
	// (служебный ключ или сервер без входа).
	User *system.User
	// Token — токен сотрудника для API системы. Пусто — сотрудника нет.
	Token string
	// DiscoveryOnly — служебный ключ в режиме user_token: можно получить список инструментов,
	// но не вызывать их (см. mcpserver.blockDiscoveryCalls).
	DiscoveryOnly bool
}

// CallerOf достаёт Caller из запроса инструмента. Без входа (AUTH_MODE=none) — пустой Caller.
func CallerOf(req mcp.Request) Caller {
	extra := req.GetExtra()
	if extra == nil || extra.TokenInfo == nil {
		return Caller{}
	}
	c, _ := extra.TokenInfo.Extra[callerKey].(Caller)
	return c
}

// tokenInfo упаковывает Caller так, как его передаёт go-sdk: middleware кладёт TokenInfo
// в контекст HTTP-запроса, а SDK — в req.Extra.TokenInfo каждого вызова.
func tokenInfo(c Caller) *sdkauth.TokenInfo {
	info := &sdkauth.TokenInfo{Extra: map[string]any{callerKey: c}}
	if c.User != nil {
		info.UserID = c.User.ID
	}
	return info
}

// RequestAs собирает вызов инструмента от имени c. Для тестов инструментов.
func RequestAs(c Caller) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{Extra: &mcp.RequestExtra{TokenInfo: tokenInfo(c)}}
}
