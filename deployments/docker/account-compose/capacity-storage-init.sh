#!/bin/sh
# Trusted install-time MinIO identity; never mounted into the serving container.
set -eu
umask 077
case "${1:-}" in
 knowledge) endpoint=http://knowledge-objects:9000; secrets=/private; bucket=knowledge; prefix=knowledge/ ;;
 ecoservices) endpoint=http://127.0.0.1:9101; secrets=/object-runtime; bucket=ecoservices; prefix=ecoservices/files/ ;;
 *) echo 'unknown capacity storage installation' >&2; exit 1 ;;
esac
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
# The parent initializer configured the existing administrative alias.
mc admin info --json local > "$work/operator-info.json"
jq -er '.info.deploymentID | select(type=="string" and test("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") and .!="00000000-0000-0000-0000-000000000000")' "$work/operator-info.json" > "$work/deployment-id"
if test -e /capacity-identity/deployment-id; then
 cmp -s /capacity-identity/deployment-id "$work/deployment-id" || { echo 'MinIO deployment changed; original capacity identity retained' >&2; exit 1; }
fi
mc admin policy create local "$bucket-capacity-read" /init/capacity-policy.json >/dev/null
mc admin policy attach local "$bucket-capacity-read" --user "$(cat "$secrets/access-key")" >/dev/null
mc alias set capacity-runtime "$endpoint" "$(cat "$secrets/access-key")" "$(cat "$secrets/secret-key")" >/dev/null
mc admin info --json capacity-runtime > "$work/runtime-info.json"
jq -er '.info.deploymentID' "$work/runtime-info.json" > "$work/runtime-id"
cmp -s "$work/deployment-id" "$work/runtime-id" || { echo 'runtime capacity identity verification failed' >&2; exit 1; }
# Verify the same narrow credentials can read the exact metered prefix/versions.
# The application retains sole ownership of empty-namespace admission/registration.
mc ls --json --recursive --versions "capacity-runtime/$bucket/$prefix" > "$work/versions.json"
cp "$work/deployment-id" /capacity-identity/deployment-id.tmp
chmod 600 /capacity-identity/deployment-id.tmp
mv /capacity-identity/deployment-id.tmp /capacity-identity/deployment-id
echo 'Original MinIO capacity identity and read policy ready'
