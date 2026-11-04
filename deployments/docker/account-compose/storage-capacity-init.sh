#!/bin/sh
# Final private manifest composition only: no SQL, IAM, objects or prices written.
set -eu
umask 077
test -s /base/current-application.json
for path in /knowledge-identity/deployment-id /ecoservices-identity/deployment-id; do
 test -s "$path" || { echo 'trusted capacity identity missing' >&2; exit 1; }
done
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
jq -e '
 .commercialOwnerDatabase != null and .moneyOwnerDatabase != null and
 .knowledge.enabled == true and .knowledge.storage.bucket == "knowledge" and
 .knowledge.storage.endpoint == "http://knowledge-objects:9000" and
 .ecoservices.enabled == true and .ecoservices.storage.bucket == "ecoservices" and
 .ecoservices.storage.endpoint == "http://127.0.0.1:9101" and
 .sourceMedia == null and .imageAgent == null and .storageCapacity != true
' /base/current-application.json >/dev/null
jq --rawfile knowledge /knowledge-identity/deployment-id --rawfile ecosystem /ecoservices-identity/deployment-id '
 def pin: rtrimstr("\n") | if test("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") and .!="00000000-0000-0000-0000-000000000000" then . else error("invalid trusted capacity identity") end;
 .storageCapacity = true |
 .knowledge.storage.capacityIdentityProvider = "minio-deployment" |
 .knowledge.storage.capacityMinIODeploymentID = ($knowledge | pin) |
 .ecoservices.storage.capacityIdentityProvider = "minio-deployment" |
 .ecoservices.storage.capacityMinIODeploymentID = ($ecosystem | pin)
' /base/current-application.json > "$work/current-application.json"
if test -e /runtime/current-application.json; then
 cmp -s "$work/current-application.json" /runtime/current-application.json || { echo 'Capacity inputs changed; original private manifest retained for inspection' >&2; exit 1; }
 echo 'Original enterprise capacity configuration retained'
 exit 0
fi
cp "$work/current-application.json" /runtime/current-application.json.tmp
chmod 600 /runtime/current-application.json.tmp
mv /runtime/current-application.json.tmp /runtime/current-application.json
echo 'Enterprise storage capacity composed; original runtime performs namespace admission'
