// Package system — клиент API вашей системы (CRM, учёт, справочник...).
//
// ЗАГЛУШКА: адреса /me и /items и поля ответов придуманы для примера. Замените их на настоящие
// вызовы своей системы. Остальной код сервера от этого не меняется.
package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/fx"

	"example.com/mcp-template/internal/config"
)

// Module даёт *Client.
var Module = fx.Module("system", fx.Provide(New))

// ErrUnauthorized — система не приняла токен (ответила 401 или 403).
var ErrUnauthorized = errors.New("система не приняла токен")

// maxBody ограничивает ответ системы: огромный ответ всё равно не поместится в контекст модели.
const maxBody = 4 << 20

// Client ходит в API системы.
type Client struct {
	baseURL string
	// fallbackToken — ключ сервера к системе, когда токена сотрудника нет (режимы service_token, none).
	fallbackToken string
	http          *http.Client
}

// New создаёт клиент по настройкам.
func New(cfg *config.Config) *Client {
	fallback := cfg.SystemToken
	if cfg.AuthMode == config.AuthUserToken {
		// С токенами сотрудников сервер не должен иметь своего доступа «за всех»:
		// иначе ошибка в коде инструмента даст сотруднику чужие данные.
		fallback = ""
	}

	return &Client{
		baseURL:       strings.TrimRight(cfg.SystemURL, "/"),
		fallbackToken: fallback,
		http:          &http.Client{Timeout: cfg.SystemTimeout},
	}
}

// get делает GET {baseURL}{path}?{query} и разбирает JSON-ответ в out.
// token — токен сотрудника; пустой — вызов идёт с ключом сервера (если он задан).
func (c *Client) get(ctx context.Context, token, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	if token == "" {
		token = c.fallbackToken
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("система недоступна: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		// Тело ошибки не возвращаем: в нём могут быть внутренние подробности системы.
		return fmt.Errorf("система ответила %d на GET %s", resp.StatusCode, path)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out); err != nil {
		return fmt.Errorf("разбор ответа GET %s: %w", path, err)
	}

	return nil
}
