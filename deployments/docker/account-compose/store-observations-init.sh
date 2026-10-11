#!/bin/sh
# Explicit authorized addition to an existing current, empty Store installation.
set -eu
umask 077
fail() { echo "$1" >&2; exit 1; }
test -s /base/current-application.json || fail 'Original serving manifest required'
test -s /secrets/store-owner/store-owner-password || fail 'Private Store owner credential required'
jq -e '.storeCenter.enabled == true and .storeCenter.database.host == "127.0.0.1" and .storeCenter.database.port == 5433 and .storeCenter.database.user == "store_center_runtime" and .storeCenter.database.database == "store_center" and (.storeCenter.observations == null) and ((.storeCenter.officialApplications // []) | length == 0) and (.identity.tenantDirectoryToken | type == "string" and length > 0) and .dataServices.temporalAddress == "127.0.0.1:7233" and .dataServices.temporalNamespace == "default"' /base/current-application.json >/dev/null || fail 'Explicit current Store at 127.0.0.1:5433 and internal Temporal configuration required; official apps must be absent'
source_hash=$(sha256sum /base/current-application.json | cut -d ' ' -f 1)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
jq '.storeCenter.observations={temporalAddress:"127.0.0.1:7233",temporalNamespace:"default",preparationOnly:true}' /base/current-application.json > "$work/runtime.json"
output_hash=$(sha256sum "$work/runtime.json" | cut -d ' ' -f 1)
jq '.storeCenter.database | .maxConnections=1 | .maxIdleConnections=0' /base/current-application.json > "$work/serving.json"
verify_serving() {
 store-center-schema-init -config "$work/serving.json" -observations-verify || fail 'Observation serving privileges unavailable; explicit owner observations-grants-only restoration required after rollback'
}
if test -e /runtime/.complete; then
 test "$(cat /runtime/.complete)" = store-observations-preparation-v1 || fail 'Unknown observation initializer marker'
 test "$(cat /runtime/source.sha256)" = "$source_hash" || fail 'Original manifest changed; explicit operator decision required'
 test "$(cat /runtime/runtime.sha256)" = "$output_hash" || fail 'Expected observation manifest changed'
 test "$(sha256sum /runtime/current-application.json | cut -d ' ' -f 1)" = "$output_hash" || fail 'Private observation manifest changed'
 verify_serving
 echo 'Existing observation preparation manifest and serving privileges verified; no schema operation'
 exit 0
fi
test ! -e /runtime/.initializing && test ! -e /runtime/current-application.json && test ! -e /runtime/source.sha256 && test ! -e /runtime/runtime.sha256 || fail 'Interrupted observation installation; inspect before retrying'
mkdir /runtime/.initializing || fail 'Observation installation already in progress'
jq -n --rawfile password /secrets/store-owner/store-owner-password \
 '{host:"127.0.0.1",port:5433,user:"store_center_owner",password:($password|rtrimstr("\n")),database:"store_center",maxConnections:2,maxIdleConnections:1}' > "$work/owner.json"
store-center-schema-init -config "$work/owner.json" -observations-only
verify_serving
# The owner command must complete before a serving manifest is published.
cp "$work/runtime.json" /runtime/current-application.json.tmp
chmod 600 /runtime/current-application.json.tmp
mv /runtime/current-application.json.tmp /runtime/current-application.json
printf '%s\n' "$source_hash" > /runtime/source.sha256
printf '%s\n' "$output_hash" > /runtime/runtime.sha256
printf '%s\n' store-observations-preparation-v1 > /runtime/.complete.tmp
mv /runtime/.complete.tmp /runtime/.complete
rmdir /runtime/.initializing
echo 'Store observation module and worker configured for preparation; SHEIN execution remains closed'
