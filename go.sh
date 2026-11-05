#!/usr/bin/env bash
# Go в Docker, ничего не ставим на хост:  ./go.sh test ./...
set -euo pipefail
root="$(cd "$(dirname "$0")" && pwd)"
image=golang:1.25.3-bookworm
cache=mcp-template-go-cache

# Новый том Docker принадлежит root: при первом запуске отдаём его текущему пользователю.
if ! docker volume inspect "$cache" >/dev/null 2>&1; then
  docker run --rm -v "$cache:/cache" "$image" chown "$(id -u):$(id -g)" /cache
fi

exec docker run --rm -u "$(id -u):$(id -g)" -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e HOME=/tmp \
  -v "$cache:/cache" -v "$root:/src" -w /src --network host "$image" go "$@"
