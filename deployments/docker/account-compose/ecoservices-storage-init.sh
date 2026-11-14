#!/bin/sh
set -eu
umask 077
export MC_CONFIG_DIR=/tmp/ecoservices-mc
printf '%s\n%s\n' ecoservices_local_root "$(cat /ecoservices-private/root-password)" | mc alias set local http://127.0.0.1:9101 >/dev/null
mc mb --ignore-existing local/ecoservices >/dev/null
mc anonymous set none local/ecoservices >/dev/null
mc admin policy create local ecoservices-runtime /init/ecoservices-storage-policy.json >/dev/null
printf '%s\n%s\n' "$(cat /object-runtime/access-key)" "$(cat /object-runtime/secret-key)" | mc admin user add local >/dev/null
mc admin policy attach local ecoservices-runtime --user "$(cat /object-runtime/access-key)" >/dev/null
if test "${ACCOUNT_STORAGE_CAPACITY_ENABLED:-}" = 1; then
 /bin/sh /init/capacity-storage-init.sh ecoservices
fi
echo 'Private ecosystem qualification object storage ready'
