package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"go.uber.org/fx"

	"example.com/mcp-template/internal/config"
	"example.com/mcp-template/internal/system"
)

// Module даёт *Guard. Проверка сотрудников идёт через *system.Client.
var Module = fx.Module("auth",
	fx.Provide(
		func(c *system.Client) UserChecker { return c },
		NewGuard,
	),
)

// UserChecker проверяет токен сотрудника в системе. Его реализует *system.Client.
type UserChecker interface {
	Me(ctx context.Context, token string) (*system.User, error)
}

// Guard пропускает к MCP только принятые токены.
type Guard struct {
	wrap func(http.Handler) http.Handler
}

// NewGuard собирает проверку под AUTH_MODE.
//
// Разбор заголовка Authorization и ответ 401 делает go-sdk (auth.RequireBearerToken),
// здесь — только решение «кто это и пускать ли» (verify).
func NewGuard(cfg *config.Config, users UserChecker, log *slog.Logger) *Guard {
	if cfg.AuthMode == config.AuthNone {
		return &Guard{wrap: func(next http.Handler) http.Handler { return next }}
	}

	v := &verifier{mode: cfg.AuthMode, serviceToken: cfg.ServiceToken, users: users, log: log}

	return &Guard{wrap: sdkauth.RequireBearerToken(v.verify, &sdkauth.RequireBearerTokenOptions{
		// Срок жизни токена знает только система, сервер его не проверяет.
		AllowMissingExpiration: true,
	})}
}

// Wrap оборачивает обработчик проверкой токена.
func (g *Guard) Wrap(next http.Handler) http.Handler { return g.wrap(next) }

type verifier struct {
	mode         config.AuthMode
	serviceToken string
	users        UserChecker
	log          *slog.Logger
}

// verify вызывается на каждый HTTP-запрос к MCP. Ошибка с sdkauth.ErrInvalidToken — ответ 401,
// любая другая (система недоступна) — ответ 500: это не «токен не принят».
func (v *verifier) verify(ctx context.Context, token string, _ *http.Request) (*sdkauth.TokenInfo, error) {
	if v.mode == config.AuthServiceToken {
		if !v.isServiceToken(token) {
			return nil, fmt.Errorf("%w: неверный служебный ключ", sdkauth.ErrInvalidToken)
		}
		return tokenInfo(Caller{}), nil
	}

	// Режим user_token.
	if v.isServiceToken(token) {
		return tokenInfo(Caller{DiscoveryOnly: true}), nil
	}

	// Каждый HTTP-запрос — отдельная проверка в системе. Если /me дорогой, добавьте здесь
	// короткий кэш (ключ — хэш токена, срок — минута).
	user, err := v.users.Me(ctx, token)
	switch {
	case errors.Is(err, system.ErrUnauthorized):
		return nil, fmt.Errorf("%w: система не приняла токен сотрудника", sdkauth.ErrInvalidToken)
	case err != nil:
		v.log.ErrorContext(ctx, "проверка токена сотрудника", slog.Any("error", err))
		return nil, errors.New("не удалось проверить токен: система недоступна")
	}

	return tokenInfo(Caller{User: user, Token: token}), nil
}

// isServiceToken сравнивает за постоянное время, чтобы ключ нельзя было подобрать по задержке ответа.
func (v *verifier) isServiceToken(token string) bool {
	return v.serviceToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(v.serviceToken)) == 1
}
