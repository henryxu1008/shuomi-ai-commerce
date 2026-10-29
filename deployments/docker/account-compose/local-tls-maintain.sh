#!/bin/sh
# Operator-only, explicit TLS maintenance. Never invokes bootstrap or deploys artifacts.
set -eu
umask 077

die() { printf '%s\n' "$*" >&2; exit 1; }
usage() { die 'usage: local-tls-maintain.sh init ABSOLUTE_NEW_STATE PORT | issue STATE | refresh-crl STATE | revoke STATE HEX_SERIAL'; }
[ "$#" -ge 2 ] || usage
action=$1
requested=$2
case "$action:$#" in init:3|issue:2|refresh-crl:2|revoke:3) ;; *) usage ;; esac
case "$requested" in /*) ;; *) die 'state must be an absolute path' ;; esac
case "$requested" in *'
'*|*''*) die 'state path must not contain line breaks' ;; esac
[ "$(uname -s)" = Linux ] || die 'Linux private filesystem permissions are required'
command -v openssl >/dev/null 2>&1 || die 'OpenSSL 3 is required'

valid_port() {
  case "$1" in ''|*[!0-9]*|0*) return 1 ;; esac
  [ "${#1}" -le 5 ] && [ "$1" -le 65535 ]
}
valid_serial() {
  case "$1" in ''|*[!0-9A-F]*) return 1 ;; esac
  [ "${#1}" -le 40 ]
}
private_dir() {
  [ -d "$1" ] && [ ! -L "$1" ] || die 'private state directory missing or linked'
  [ "$(stat -c '%u:%a' "$1")" = "$(id -u):700" ] || die 'private state directory must be owned by current uid with mode 0700'
}

if [ "$action" = init ]; then
  port=$3
  valid_port "$port" || die 'port must be a decimal value from 1 to 65535 without leading zeros'
  [ ! -e "$requested" ] && [ ! -L "$requested" ] || die 'init requires a new state path; existing state is never replaced'
  mkdir -m 700 "$requested"
else
  private_dir "$requested"
fi
state=$(cd "$requested" && pwd -P)
export LOCAL_TLS_STATE="$state"

if [ "$action" != init ]; then
  # Known state paths only; do not search for keys or discover other issuers.
  for entry in private issued releases history public openssl.cnf index.txt index.txt.attr index.txt.old index.txt.attr.old serial serial.old crlnumber crlnumber.old .ready port private/root-ca.key public/root-ca.pem public/root.crl; do
    [ ! -L "$state/$entry" ] || die 'linked state artifact refused'
  done
  for entry in private issued releases history; do private_dir "$state/$entry"; done
  [ -f "$state/.ready" ] && [ -s "$state/public/root.crl" ] || die 'state initialization incomplete'
  [ "$(stat -c '%u:%a' "$state/private/root-ca.key")" = "$(id -u):600" ] || die 'CA key permissions must be 0600 for current uid'
  port=$(cat "$state/port")
  valid_port "$port" || die 'invalid retained port'
fi

if [ "$action" = revoke ]; then
  serial=$3
  valid_serial "$serial" || die 'serial must be uppercase hexadecimal, at most 40 characters'
  [ -f "$state/issued/$serial.pem" ] && [ ! -L "$state/issued/$serial.pem" ] || die 'serial is not an issued certificate of this CA'
fi

op="$state/.operation"
mkdir -m 700 "$op" 2>/dev/null || die 'unfinished operation exists; preserve evidence and obtain explicit recovery instructions, do not retry'
step=0
trap 'status=$?; if [ "$status" -ne 0 ]; then printf "STOPPED exit=%s step=%s\n" "$status" "$step" > "$op/result"; fi' 0
trap 'exit 130' INT
trap 'exit 143' TERM HUP

run() {
  step=$((step + 1))
  status=0
  openssl "$@" > "$op/$step.stdout" 2> "$op/$step.stderr" || status=$?
  printf '%s\n' "$status" > "$op/$step.exit"
  [ "$status" -eq 0 ] || exit "$status"
}

refresh_crl() {
  run x509 -in "$state/public/root-ca.pem" -checkend 86400 -noout
  [ ! -e "$state/public/root.crl" ] || cp "$state/public/root.crl" "$op/previous-root.crl"
  run ca -config "$state/openssl.cnf" -gencrl -out "$op/root.crl.pem"
  run crl -in "$op/root.crl.pem" -noout -no-CAfile -no-CApath -no-CAstore -verify -CAfile "$state/public/root-ca.pem"
  # OpenSSL crl can print verify failure yet return zero. Unknown output is not success.
  [ "$(cat "$op/$step.stderr")" = 'verify OK' ] && [ ! -s "$op/$step.stdout" ] || die 'CRL signature verification did not explicitly succeed; publication stopped'
  run crl -in "$op/root.crl.pem" -outform DER -out "$op/root.crl.der"
  [ -s "$op/root.crl.der" ] || die 'empty CRL output refused'
  [ "$(stat -c %d "$op")" = "$(stat -c %d "$state/public")" ] || die 'atomic publication requires the same filesystem'
  chmod 644 "$op/root.crl.der"
  mv "$op/root.crl.der" "$state/public/root.crl"
}

case "$action" in
  init)
    for entry in private issued releases history; do mkdir -m 700 "$state/$entry"; done
    mkdir -m 755 "$state/public"
    : > "$state/index.txt"
    printf 'unique_subject = no\n' > "$state/index.txt.attr"
    printf '1000\n' > "$state/serial"
    printf '1000\n' > "$state/crlnumber"
    printf '%s\n' "$port" > "$state/port"
    cat > "$state/openssl.cnf" <<'EOF'
[ca]
default_ca = local_ca
[local_ca]
dir = $ENV::LOCAL_TLS_STATE
database = $dir/index.txt
new_certs_dir = $dir/issued
certificate = $dir/public/root-ca.pem
private_key = $dir/private/root-ca.key
serial = $dir/serial
crlnumber = $dir/crlnumber
default_md = sha256
default_days = 30
default_crl_days = 7
unique_subject = no
copy_extensions = none
policy = server_policy
crl_extensions = crl_ext
[server_policy]
commonName = supplied
[req]
distinguished_name = req_dn
prompt = no
[req_dn]
CN = localhost
[ca_ext]
basicConstraints = critical,CA:true,pathlen:0
keyUsage = critical,keyCertSign,cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always
[crl_ext]
authorityKeyIdentifier = keyid:always
[server_ext]
basicConstraints = critical,CA:false
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always
subjectAltName = DNS:localhost,IP:127.0.0.1,IP:::1
EOF
    printf 'crlDistributionPoints = URI:http://localhost:%s/.well-known/local-tls/root.crl\n' "$port" >> "$state/openssl.cnf"
    run req -config "$state/openssl.cnf" -x509 -newkey rsa:2048 -nodes -sha256 -days 3650 -subj '/CN=Task Processor Local TLS CA' -extensions ca_ext -keyout "$state/private/root-ca.key" -out "$state/public/root-ca.pem"
    refresh_crl
    chmod 644 "$state/public/root-ca.pem"
    : > "$state/.ready"
    ;;
  issue)
    serial=$(cat "$state/serial")
    valid_serial "$serial" || die 'invalid CA serial state'
    release="$state/releases/$serial"
    mkdir -m 700 "$release"
    # Require enough root lifetime for the complete leaf validity interval.
    run x509 -in "$state/public/root-ca.pem" -checkend 2678400 -noout
    run req -config "$state/openssl.cnf" -newkey rsa:2048 -nodes -sha256 -subj '/CN=localhost' -keyout "$release/localhost.key" -out "$op/localhost.csr"
    run ca -config "$state/openssl.cnf" -batch -notext -extensions server_ext -in "$op/localhost.csr" -out "$release/localhost.crt"
    for scope in 'hostname localhost' 'ip 127.0.0.1' 'ip ::1'; do
      # Values are fixed constants; no operator-provided flags are expanded here.
      set -- $scope
      run verify -no-CAfile -no-CApath -no-CAstore -CAfile "$state/public/root-ca.pem" -CRLfile "$state/public/root.crl" -crl_check -purpose sslserver "-verify_$1" "$2" "$release/localhost.crt"
    done
    : > "$release/complete"
    ;;
  refresh-crl) refresh_crl ;;
  revoke)
    run ca -config "$state/openssl.cnf" -revoke "$state/issued/$serial.pem" -crl_reason cessationOfOperation
    refresh_crl
    ;;
esac

printf 'COMPLETE action=%s steps=%s\n' "$action" "$step" > "$op/result"
archive="$state/history/$(date -u +%Y%m%dT%H%M%SZ)-$$-$action"
[ ! -e "$archive" ] || die 'history destination already exists; preserve operation evidence'
mv "$op" "$archive"
trap - 0 INT TERM HUP
printf 'complete: %s; artifacts prepared only; no trust or service changes\n' "$action"
