package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"

	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/system"
)

// fakeUsers — система, в которой известен один токен сотрудника.
type fakeUsers struct{ down bool }

func (f fakeUsers) Me(_ context.Context, token string) (*system.User, error) {
	switch {
	case f.down:
		return nil, errors.New("connection refused")
	case token == "user-token":
		return &system.User{ID: "42", Name: "Иванов Иван"}, nil
	default:
		return nil, system.ErrUnauthorized
	}
}

func TestGuard(t *testing.T) {
	tests := []struct {
		name       string
		mode       config.AuthMode
		service    string
		users      fakeUsers
		header     string
		wantStatus int
		wantCaller Caller
	}{
		{name: "сотрудник принят", mode: config.AuthUserToken, header: "Bearer user-token",
			wantStatus: 200, wantCaller: Caller{User: &system.User{ID: "42", Name: "Иванов Иван"}, Token: "user-token"}},
		{name: "чужой токен", mode: config.AuthUserToken, header: "Bearer other", wantStatus: 401},
		{name: "без заголовка", mode: config.AuthUserToken, wantStatus: 401},
		{name: "не Bearer", mode: config.AuthUserToken, header: "Basic dXNlcg==", wantStatus: 401},
		{name: "служебный ключ в user_token — только список", mode: config.AuthUserToken, service: "svc",
			header: "Bearer svc", wantStatus: 200, wantCaller: Caller{DiscoveryOnly: true}},
		{name: "система недоступна — не 401", mode: config.AuthUserToken, users: fakeUsers{down: true},
			header: "Bearer user-token", wantStatus: 500},
		{name: "служебный ключ верный", mode: config.AuthServiceToken, service: "svc",
			header: "Bearer svc", wantStatus: 200, wantCaller: Caller{}},
		{name: "служебный ключ неверный", mode: config.AuthServiceToken, service: "svc",
			header: "Bearer svc2", wantStatus: 401},
		{name: "без входа", mode: config.AuthNone, wantStatus: 200, wantCaller: Caller{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{AuthMode: tt.mode, ServiceToken: tt.service}
			guard := NewGuard(cfg, tt.users, slog.New(slog.NewTextHandler(io.Discard, nil)))

			var got Caller
			h := guard.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if info := sdkauth.TokenInfoFromContext(r.Context()); info != nil {
					got, _ = info.Extra[callerKey].(Caller)
				}
			}))

			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус %d, ожидали %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != 200 {
				return
			}
			if got.Token != tt.wantCaller.Token || got.DiscoveryOnly != tt.wantCaller.DiscoveryOnly ||
				(got.User == nil) != (tt.wantCaller.User == nil) ||
				(got.User != nil && *got.User != *tt.wantCaller.User) {
				t.Fatalf("caller %+v, ожидали %+v", got, tt.wantCaller)
			}
		})
	}
}
