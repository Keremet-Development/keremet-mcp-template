# Скиллы для Go (services/gateway, services/control)

Скопированы без изменений из ваших проектов (base-template, sed-mcp, consumer): там они закреплены
на коммитах апстрима. Лицензии — в `THIRD_PARTY_LICENSES/`.

| Скиллы | Источник | Лицензия |
|---|---|---|
| golang-* | samber/cc-skills-golang (как в base-template и consumer, коммит bb1d43d9…) | MIT |
| clickhouse-best-practices | ClickHouse/agent-skills (как в consumer, коммит 69a8354b…) | Apache-2.0 |

## Что важнее скиллов
Правила проекта (`CLAUDE.md`) главнее: шлюз пишется на uber fx, понятно и без лишних зависимостей —
`samber/*` и прочие библиотеки, которые советуют скиллы, не добавляем без необходимости.
