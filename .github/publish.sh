#!/usr/bin/env bash
# Публикует собранный образ в ghcr.io: publish.sh <локальный образ> <ghcr.io/…/имя>
#   ветка main  → :main, :sha-<7 знаков коммита>
#   тег vX.Y.Z  → :X.Y.Z, :latest
# Вход — токеном задания (TOKEN), в отдельном DOCKER_CONFIG: на общем раннере пароль не остаётся.
set -euo pipefail
src=$1 dst=$2

export DOCKER_CONFIG="$RUNNER_TEMP/docker-$GITHUB_RUN_ID-$GITHUB_JOB"
mkdir -p "$DOCKER_CONFIG"
trap 'rm -rf "$DOCKER_CONFIG"' EXIT
echo "$TOKEN" | docker login ghcr.io -u "$GITHUB_ACTOR" --password-stdin

tags=("sha-${GITHUB_SHA::7}")
case "$GITHUB_REF" in
  refs/heads/main) tags+=(main) ;;
  refs/tags/v*)    tags+=("${GITHUB_REF#refs/tags/v}" latest) ;;
esac

# Первая публикация создаёт пакет, и ghcr.io выдаёт права на него не сразу: первый push может получить
# denied. Поэтому пробуем ещё раз после паузы.
push() {
  local n
  for n in 1 2 3; do
    docker push "$1" && return 0
    [ "$n" = 3 ] || { echo "push не прошёл, повтор через $((n * 15)) с"; sleep $((n * 15)); }
  done
  return 1
}

for t in "${tags[@]}"; do
  docker tag "$src" "$dst:$t"
  push "$dst:$t"
  docker image rm "$dst:$t" >/dev/null
  echo "опубликован $dst:$t"
done
