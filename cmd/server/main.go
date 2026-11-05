// Команда server — MCP-сервер, через который агенты платформы работают с вашей системой.
// Настройки — переменные окружения (см. README).
package main

import (
	"go.uber.org/fx"

	"example.com/mcp-template/internal/app"
)

func main() {
	// Run запускает модули, ждёт SIGINT/SIGTERM и аккуратно всё останавливает.
	fx.New(app.Module).Run()
}
