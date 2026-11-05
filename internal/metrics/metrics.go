// Package metrics — реестр Prometheus и отдельный HTTP-сервер для /metrics.
//
// Метрики на отдельном порту (METRICS_ADDR, по умолчанию :9090), чтобы их не было видно
// снаружи вместе с MCP. Пустой METRICS_ADDR — сервер метрик не запускается.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/fx"
)

// Module даёт реестр метрик (*prometheus.Registry и prometheus.Registerer) и запускает /metrics.
var Module = fx.Module("metrics",
	fx.Provide(
		NewRegistry,
		func(r *prometheus.Registry) prometheus.Registerer { return r },
	),
	fx.Invoke(serve),
)

// NewRegistry создаёт свой реестр (не глобальный), чтобы тесты и несколько экземпляров
// приложения в одном процессе не мешали друг другу.
func NewRegistry() *prometheus.Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return r
}
