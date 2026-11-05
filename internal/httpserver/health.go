package httpserver

import (
	"net/http"
	"sync/atomic"
)

// Health отвечает на пробы оркестратора.
//
//	/healthz — процесс жив (перезапуск не нужен);
//	/readyz  — готов принимать запросы: запущен и ещё не останавливается.
//
// Хотите, чтобы экземпляр выводился из балансировки, пока система недоступна, — добавьте в Ready
// быструю проверку системы (например, GET {SYSTEM_URL}/health с таймаутом 2–3 с). Только не в /healthz:
// перезапуск сервера не починит чужую систему.
type Health struct {
	ready atomic.Bool
}

// NewHealth создаёт пробы; готовность — false до запуска сервера.
func NewHealth() *Health { return &Health{} }

func (h *Health) markReady()    { h.ready.Store(true) }
func (h *Health) markNotReady() { h.ready.Store(false) }

// Live — всегда 200, пока процесс отвечает.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// Ready — 200 после запуска, 503 до запуска и во время остановки.
func (h *Health) Ready(w http.ResponseWriter, _ *http.Request) {
	if !h.ready.Load() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok"))
}
