package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/mcp-template/internal/auth"
	"example.com/mcp-template/internal/system"
)

const (
	defaultLimit = 10
	// maxLimit держит ответ небольшим: платформа обрезает результат инструмента примерно
	// до 6000 символов, и лишние записи только вытесняют полезные.
	maxLimit = 20
)

func (t *Tools) registerItems(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "search_items",
		Title: "Поиск записей",
		Description: "Ищет записи в системе «Пример» по словам из названия. Видны только записи, доступные сотруднику. " +
			"Вызывай, когда спрашивают найти запись, проверить её статус или узнать, есть ли такая. " +
			"Пиши в query 1–3 ключевых слова, а не весь вопрос. " +
			"Ответ: список {id, title, status, updatedAt}. Пустой список — ничего не нашлось: попробуй другие слова, " +
			"не выдумывай записи.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}, t.searchItems)
}

type searchItemsInput struct {
	Query string `json:"query" jsonschema:"ключевые слова из названия записи, например: договор поставки"`
	Limit int    `json:"limit,omitempty" jsonschema:"сколько записей вернуть, от 1 до 20; по умолчанию 10"`
}

type searchItemsOutput struct {
	Items []system.Item `json:"items"`
	// Note — подсказка модели, что делать дальше (уточнить запрос, сменить слова).
	Note string `json:"note,omitempty"`
}

func (t *Tools) searchItems(ctx context.Context, req *mcp.CallToolRequest, in searchItemsInput) (*mcp.CallToolResult, searchItemsOutput, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, searchItemsOutput{}, errors.New("пустой query: укажи ключевые слова из названия записи")
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)

	items, err := t.system.SearchItems(ctx, auth.CallerOf(req).Token, query, limit)
	if errors.Is(err, system.ErrUnauthorized) {
		return nil, searchItemsOutput{}, errors.New("система не приняла токен сотрудника: попросите его войти заново")
	}
	if err != nil {
		t.log.WarnContext(ctx, "search_items: ошибка системы", slog.Any("error", err))
		// Текст ошибки увидит модель: без адресов, ключей и внутренних подробностей.
		return nil, searchItemsOutput{}, errors.New("система «Пример» сейчас недоступна, попробуй позже")
	}

	out := searchItemsOutput{Items: items}
	switch {
	case len(items) == 0:
		out.Items = []system.Item{} // [] вместо null: модели так понятнее
		out.Note = "Ничего не найдено. Попробуй другие или более короткие слова."
	case len(items) == limit:
		out.Note = fmt.Sprintf("Показаны первые %d. Если нужной записи нет — уточни запрос.", limit)
	}

	return nil, out, nil
}
