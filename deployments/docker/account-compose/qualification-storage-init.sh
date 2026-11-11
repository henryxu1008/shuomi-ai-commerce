#!/bin/sh
set -eu
umask 077
export MC_CONFIG_DIR=/tmp/qualification-mc
printf '%s\n%s\n' qualification_local_root "$(cat /qualification-private/root-password)" | mc alias set local http://127.0.0.1:9000 >/dev/null
mc mb --ignore-existing local/qualifications >/dev/null
mc anonymous set none local/qualifications >/dev/null
mc admin policy create local qualification-runtime /init/qualification-storage-policy.json >/dev/null
printf '%s\n%s\n' "$(cat /runtime/qualification-access-key)" "$(cat /runtime/qualification-secret-key)" | mc admin user add local >/dev/null
mc admin policy attach local qualification-runtime --user "$(cat /runtime/qualification-access-key)" >/dev/null
echo 'Private qualification object storage ready'
