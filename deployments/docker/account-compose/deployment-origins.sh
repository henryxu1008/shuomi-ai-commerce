#!/bin/sh
# Shared deployment validation; authentication stays with ZITADEL and Auth.js.
account_resolve_origins() {
  account_public_app=${ACCOUNT_PUBLIC_APPLICATION_ORIGIN:-}
  account_public_identity=${ACCOUNT_PUBLIC_IDENTITY_ORIGIN:-}
  account_identity_host=
  case "${ACCOUNT_PUBLIC_TRIAL:-}" in
    '')
      test -z "$account_public_app$account_public_identity" || { echo 'public origins require explicit public trial opt-in' >&2; return 1; }
      return 0 ;;
    1) ;;
    *) echo 'invalid public trial opt-in' >&2; return 1 ;;
  esac
  for account_origin in "$account_public_app" "$account_public_identity"; do
    test "${#account_origin}" -le 261 && printf '%s\n' "$account_origin" | grep -Eq '^https://([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$' || {
      echo 'public trial requires two exact HTTPS DNS origins on port 443, without paths or credentials' >&2; return 1;
    }
  done
  test "$account_public_app" != "$account_public_identity" || { echo 'application and identity origins must be distinct' >&2; return 1; }
  account_identity_host=${account_public_identity#https://}
  if test -n "${ACCOUNT_PUBLIC_APPLICATION_HOST:-}"; then
    test "https://$ACCOUNT_PUBLIC_APPLICATION_HOST" = "$account_public_app" || { echo 'application hostname must match the configured public origin' >&2; return 1; }
  fi
  if test -n "${ACCOUNT_PUBLIC_IDENTITY_HOST:-}"; then
    test "$ACCOUNT_PUBLIC_IDENTITY_HOST" = "$account_identity_host" || { echo 'identity hostname must match the configured issuer origin' >&2; return 1; }
  fi
}

account_pin_origins() {
  account_resolve_origins || return 1
  account_origin_pin="$1/.public-origins"
  if test -f "$account_origin_pin"; then
    test "$(cat "$account_origin_pin")" = "$(printf '%s\n%s' "$account_public_app" "$account_public_identity")" || {
      echo 'retained public origins changed; use the original configuration and keep all data' >&2; return 1;
    }
  elif test -n "$account_public_app"; then
    test "$2" = fresh || { echo 'public trial requires a new empty project' >&2; return 1; }
    (umask 077; printf '%s\n%s\n' "$account_public_app" "$account_public_identity" > "$account_origin_pin")
  fi
}
