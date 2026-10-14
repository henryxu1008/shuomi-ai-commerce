#!/bin/sh
set -eu
umask 077
test "${ACCOUNT_SOURCE_MEDIA_TRIAL:-}" = ISOLATED_TRIAL_ONLY
export MC_CONFIG_DIR=/tmp/source-media-mc
mc alias set local http://127.0.0.1:9103 source_media_local_root "$(cat /private/root-password)" >/dev/null
mc mb --ignore-existing local/source-media-trial >/dev/null
# Anonymous object reads only. Do not use the broader download/list policy.
mc anonymous set-json /init/public-policy.json local/source-media-trial >/dev/null
printf 'Source-media trial bucket ready; no other project storage modified.\n'
