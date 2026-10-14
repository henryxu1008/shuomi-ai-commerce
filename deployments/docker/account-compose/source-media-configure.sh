#!/bin/sh
set -eu
umask 077
test "${ACCOUNT_SOURCE_MEDIA_TRIAL:-}" = ISOLATED_TRIAL_ONLY
port=${ACCOUNT_APPLICATION_PORT:?ACCOUNT_APPLICATION_PORT is required}
case "$port" in ''|*[!0-9]*|0*) exit 1 ;; esac
test "$port" -ge 1 && test "$port" -le 65535
base="https://localhost:$port/source-media-assets/source-media-trial"
test -s /runtime/current-application.json
test -s /private/root-password
# Use the existing initialized owners; never run Product/schema initialization.
jq -e '.productCollections == true and .productAcquisitionDatabase != null and .commercialOwnerDatabase != null and .moneyOwnerDatabase != null' /runtime/current-application.json >/dev/null
jq --arg base "$base" -e '.sourceMedia == null or (.sourceMediaIsolatedTrial == true and .sourceMedia.publicBase == $base and .sourceMedia.s3.bucket == "source-media-trial" and .sourceMedia.s3.endpoint == "http://127.0.0.1:9103")' /runtime/current-application.json >/dev/null
test -e /runtime/current-application.before-source-media.json || cp /runtime/current-application.json /runtime/current-application.before-source-media.json
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
jq --arg base "$base" --rawfile password /private/root-password \
  '.storageCapacity = true | .sourceMediaIsolatedTrial = true | .sourceMedia = {enabled:true, provider:"s3", publicBase:$base, s3:{region:"us-east-1",endpoint:"http://127.0.0.1:9103",bucket:"source-media-trial",accessKeyID:"source_media_local_root",secretAccessKey:($password|rtrimstr("\n")),usePathStyle:true,artifactMode:"aws"}}' \
  /runtime/current-application.json > "$work/current.json"
chmod 600 "$work/current.json"
mv "$work/current.json" /runtime/current-application.json
printf 'Source-media trial configured with original owners and capacity enabled.\n'
