# Login cookie isolation — #706

This fixes the reproduced Auth.js cookie collision between independent local
applications on the same hostname. Browser cookies are not scoped by port.
It adds no database migration, IAM state, secret or provider configuration.

## Configuration and cutover

- Keep the original `AUTH_URL`, public base URL, issuer, client ID/secret and
  `AUTH_SECRET`. Auth.js's existing `NEXTAUTH_URL` alias is also respected.
  Existing public-origin resolution is reused when no native Auth.js URL is set;
  its localhost default remains for development. Configure an accurate public
  URL for every deployed instance, as for existing redirects and logout.
- Native session/callback/CSRF/PKCE/state/nonce/challenge cookies now have a
  stable `shuomi.<hash>` namespace derived from canonical public origin and
  OIDC application. The hash contains no secrets or user/Organization identity.
- All public/server Auth.js consumers use the same names. Native security
  flags, TTL, JWT/PKCE checks, chunking and clearing stay enabled.
- Global default Auth.js cookies are retired inputs. They are not copied,
  migrated, tried as fallbacks or cleared across localhost. After the approved
  UI update, an old session needs one ordinary sign-in under the new names.
  Repeated sign-in against unchanged global names is not the fix.

## Retained runtime handoff

The original instance owner alone performs any approved merge-SHA UI update.
Use the existing normal image/Compose flow, keep the original configuration,
secrets, volumes, backend/worker and saved draft. No initializer, account reset,
credential rotation, IAM write, business-table DDL, or data restore is needed.
Do not change another application's cookies/config to manufacture a green run.

After the exact candidate has merge/deployment approval, check normal login in
the existing browser with foreign default cookies left intact, then load the
current session and existing saved recovery draft. Inspect only cookie-name/
attribute metadata if needed; do not expose tokens or cookie values. Keep
controlled fixture evidence, retained-instance verification and user acceptance
separate. Source CI alone is not proof that this UI update is installed.

## Diagnosis limits

Original logs showed `JWTSessionError / no matching decryption secret` before
new credential entry, PKCE `InvalidCheck`, and later recurring JWT errors.
Original credential fingerprints matched cold backups. Another application in
the same browser/host returned native global Auth.js cookie names. The pinned
SDK regression reproduces session/PKCE failures after cross-port overwrites and
passes after namespace isolation, including chunked sessions and one-use checks.
The exact historical foreign session/PKCE cookie was not retained, so do not
claim its value, originating tab, or unique provenance has been reconstructed.
