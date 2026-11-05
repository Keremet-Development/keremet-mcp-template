package mcpserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"

	"example.com/mcp-template/internal/auth"
)

// blockDiscoveryCalls не даёт вызывать инструменты служебным ключом в режиме user_token.
// Ключ нужен платформе только чтобы узнать список инструментов; данные — только с правами сотрудника.
func blockDiscoveryCalls(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if _, isCall := req.(*mcp.CallToolRequest); isCall && auth.CallerOf(req).DiscoveryOnly {
			return errorResult("Служебный ключ подходит только для списка инструментов. " +
				"Вызов инструмента нужен от имени сотрудника."), nil
		}
		return next(ctx, method, req)
	}
}

// Исход вызова для метки status.
const (
	statusOK       = "ok"
	statusToolErr  = "tool_error"     // инструмент отработал и вернул ошибку (isError)
	statusRejected = "protocol_error" // SDK отклонил вызов: нет такого инструмента, неверные аргументы
)

// callObserver пишет метрики и журнал по каждому вызову инструмента.
type callObserver struct {
	calls    *prometheus.CounterVec
	duration *prometheus.HistogramVec
	log      *slog.Logger
}

func newCallObserver(reg prometheus.Registerer, log *slog.Logger) (*callObserver, error) {
	o := &callObserver{
		calls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "Вызовы инструментов MCP по инструменту и исходу.",
		}, []string{"tool", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mcp_tool_call_duration_seconds",
			Help:    "Время вызова инструмента MCP.",
			Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		}, []string{"tool", "status"}),
		log: log,
	}

	for _, c := range []prometheus.Collector{o.calls, o.duration} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}

	return o, nil
}

func (o *callObserver) middleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		call, ok := req.(*mcp.CallToolRequest)
		if !ok {
			return next(ctx, method, req)
		}

		start := time.Now()
		res, err := next(ctx, method, req)
		elapsed := time.Since(start)

		// Имя пришло от клиента: в метку — только если SDK принял вызов, иначе меток станет без счёта.
		tool, status := "-", statusRejected
		if r, ok := res.(*mcp.CallToolResult); ok && err == nil {
			tool, status = call.Params.Name, statusOK
			if r.IsError {
				status = statusToolErr
			}
		}

		o.calls.WithLabelValues(tool, status).Inc()
		o.duration.WithLabelValues(tool, status).Observe(elapsed.Seconds())

		// В журнал — кто и что вызвал, но не аргументы и не ответ: там могут быть личные данные.
		attrs := []slog.Attr{
			slog.String("tool", call.Params.Name),
			slog.String("status", status),
			slog.Duration("duration", elapsed),
		}
		if u := auth.CallerOf(req).User; u != nil {
			attrs = append(attrs, slog.String("user_id", u.ID))
		}
		if err != nil {
			attrs = append(attrs, slog.Any("error", err))
		}

		level := slog.LevelInfo
		if status != statusOK {
			level = slog.LevelWarn
		}
		o.log.LogAttrs(ctx, level, "вызов инструмента", attrs...)

		return res, err
	}
}

// errorResult — ошибка инструмента, которую увидит модель (isError: true).
func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
