package tools

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/mcp-template/internal/auth"
	"example.com/mcp-template/internal/system"
)

func (t *Tools) registerProfile(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "whoami",
		Title: "Текущий сотрудник",
		Description: "Кто задаёт вопрос: ФИО, подразделение, почта сотрудника в системе «Пример». " +
			"Вызывай, когда в вопросе есть «я», «мой», «мне» и нужно знать, кто это, " +
			"или чтобы подставить сотрудника в другой инструмент. Аргументов нет. " +
			"Не вызывай для поиска других сотрудников.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}, t.whoami)
}

type whoamiInput struct{}

// whoami отдаёт профиль, который проверка токена уже получила из системы: повторно в систему не ходим.
func (t *Tools) whoami(_ context.Context, req *mcp.CallToolRequest, _ whoamiInput) (*mcp.CallToolResult, system.User, error) {
	user := auth.CallerOf(req).User
	if user == nil {
		return nil, system.User{}, errors.New("сервер работает без входа сотрудника: профиль неизвестен")
	}
	return nil, *user, nil
}
