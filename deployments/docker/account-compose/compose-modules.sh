#!/bin/sh
# Fixed existing module composition; the same entrypoint handles up/stop/restart.
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
chat=0
public=0
acquisition=0
observations=0
while test "$#" -gt 0; do
 case "$1" in
  --chat) chat=1; shift ;;
  --public) public=1; shift ;;
  --acquisition-plans) acquisition=1; chat=1; shift ;;
  --store-observations) observations=1; acquisition=1; chat=1; shift ;;
  --) shift; break ;;
  *) break ;;
 esac
done
test "$#" -gt 0 || { echo 'Usage: compose-modules.sh [--chat] [--acquisition-plans] [--store-observations] [--public] -- <docker compose command>' >&2; exit 1; }
if test "$public" = 1; then set -- -f docker-compose.public-trial.yml "$@"; fi
if test "$observations" = 1; then set -- -f docker-compose.store-observations.yml -f docker-compose.storage-capacity-store-observations.yml "$@"; fi
if test "$acquisition" = 1; then set -- -f docker-compose.acquisition-plans.yml -f docker-compose.storage-capacity-acquisition-plans.yml "$@"; fi
if test "$chat" = 1; then set -- -f docker-compose.chat.yml -f docker-compose.storage-capacity-chat.yml "$@"; fi
exec docker compose --env-file .env -f docker-compose.yml -f docker-compose.knowledge.yml -f docker-compose.data-services.yml -f docker-compose.modules.yml -f docker-compose.ecoservices.yml -f docker-compose.storage-capacity.yml "$@"
