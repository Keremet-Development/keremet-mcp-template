package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/fx"

	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/mcpserver"
)

// Сквозной тест: собираем приложение целиком (без запуска портов), ставим перед ним httptest
// и говорим с ним так же, как агент платформы (keremet-agents, app/mcp_client.py):
// initialize → notifications/initialized → tools/list → tools/call → DELETE.

const (
	userToken    = "user-token"
	serviceToken = "svc-token"
)

// fakeSystem — API системы: /me знает один токен, /items возвращает одну запись.
func fakeSystem(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+userToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"id":"42","name":"Иванов Иван","department":"Бухгалтерия"}`)
	})
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+userToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("q") != "договор" {
			_, _ = io.WriteString(w, `{"items":[]}`)
			return
		}
		_, _ = io.WriteString(w, `{"items":[{"id":"7","title":"Договор поставки","status":"active"}]}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// startMCP собирает приложение с тестовыми настройками и отдаёт адрес MCP-эндпоинта.
func startMCP(t *testing.T) string {
	t.Helper()
	cfg := &config.Config{
		Path:         "/mcp",
		AuthMode:     config.AuthUserToken,
		ServiceToken: serviceToken,
		SystemURL:    fakeSystem(t).URL,
		LogLevel:     "info",
	}

	var h *mcpserver.Handler
	app := fx.New(
		Module,
		fx.Replace(cfg, slog.New(slog.NewTextHandler(io.Discard, nil))),
		fx.NopLogger,
		fx.Populate(&h),
	)
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL + "/mcp"
}

// client повторяет заголовки платформы.
type client struct {
	t       *testing.T
	url     string
	token   string
	session string
	nextID  int
}

// call отправляет JSON-RPC и возвращает HTTP-статус и поле result (или error) ответа.
func (c *client) call(method string, params any, notify bool) (int, map[string]any) {
	c.t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		body["params"] = params
	}
	if !notify {
		c.nextID++
		body["id"] = c.nextID
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	if c.session != "" {
		req.Header.Set("Mcp-Session-Id", c.session)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if s := resp.Header.Get("Mcp-Session-Id"); s != "" {
		c.session = s
	}

	raw, _ := io.ReadAll(resp.Body)
	if notify || resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}

	var msg struct {
		Result map[string]any `json:"result"`
		Error  map[string]any `json:"error"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		c.t.Fatalf("%s: ответ не JSON: %s", method, raw)
	}
	if msg.Error != nil {
		c.t.Fatalf("%s: ошибка JSON-RPC %v", method, msg.Error)
	}
	return resp.StatusCode, msg.Result
}

func (c *client) initialize() {
	c.t.Helper()
	status, res := c.call("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1"},
	}, false)
	if status != http.StatusOK || res["protocolVersion"] != "2025-06-18" {
		c.t.Fatalf("initialize: статус %d, результат %v", status, res)
	}
	if status, _ := c.call("notifications/initialized", nil, true); status >= 300 {
		c.t.Fatalf("notifications/initialized: статус %d", status)
	}
}

// toolText — текст ответа инструмента и признак isError, как их читает платформа.
func toolText(res map[string]any) (string, bool) {
	var parts []string
	content, _ := res["content"].([]any)
	for _, c := range content {
		if m, _ := c.(map[string]any); m["type"] == "text" {
			parts = append(parts, m["text"].(string))
		}
	}
	isErr, _ := res["isError"].(bool)
	return strings.Join(parts, "\n"), isErr
}

func TestUserSession(t *testing.T) {
	c := &client{t: t, url: startMCP(t), token: userToken}
	c.initialize()

	_, list := c.call("tools/list", map[string]any{}, false)
	names := map[string]bool{}
	for _, tool := range list["tools"].([]any) {
		names[tool.(map[string]any)["name"].(string)] = true
	}
	if !names["whoami"] || !names["search_items"] {
		t.Fatalf("tools/list: %v", names)
	}

	_, res := c.call("tools/call", map[string]any{"name": "whoami", "arguments": map[string]any{}}, false)
	if text, isErr := toolText(res); isErr || !strings.Contains(text, "Иванов Иван") {
		t.Fatalf("whoami: %q (isError=%v)", text, isErr)
	}

	_, res = c.call("tools/call", map[string]any{
		"name": "search_items", "arguments": map[string]any{"query": "договор"},
	}, false)
	if text, isErr := toolText(res); isErr || !strings.Contains(text, "Договор поставки") {
		t.Fatalf("search_items: %q (isError=%v)", text, isErr)
	}

	_, res = c.call("tools/call", map[string]any{
		"name": "search_items", "arguments": map[string]any{"query": "   "},
	}, false)
	if _, isErr := toolText(res); !isErr {
		t.Fatal("search_items с пустым query должен вернуть ошибку инструмента")
	}

	// Платформа закрывает сессию DELETE-ом; сервер без сессий не должен на этом падать.
	req, _ := http.NewRequest(http.MethodDelete, c.url, nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Fatalf("DELETE: статус %d", resp.StatusCode)
	}
}

func TestServiceTokenOnlyListsTools(t *testing.T) {
	c := &client{t: t, url: startMCP(t), token: serviceToken}
	c.initialize()

	_, list := c.call("tools/list", map[string]any{}, false)
	if len(list["tools"].([]any)) == 0 {
		t.Fatal("служебный ключ должен видеть список инструментов")
	}

	_, res := c.call("tools/call", map[string]any{"name": "whoami", "arguments": map[string]any{}}, false)
	if _, isErr := toolText(res); !isErr {
		t.Fatal("служебный ключ не должен вызывать инструменты в режиме user_token")
	}
}

func TestRejectedToken(t *testing.T) {
	c := &client{t: t, url: startMCP(t), token: "wrong"}
	status, _ := c.call("initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "test", "version": "1"},
	}, false)
	if status != http.StatusUnauthorized {
		t.Fatalf("ожидали 401 (по нему платформа понимает «токен не принят»), получили %d", status)
	}
}
