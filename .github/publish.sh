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

for t in "${tags[@]}"; do
  docker tag "$src" "$dst:$t"
  docker push "$dst:$t"
  docker image rm "$dst:$t" >/dev/null
  echo "опубликован $dst:$t"
done
