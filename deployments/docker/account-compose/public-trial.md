# Public-domain colleague trial

This opt-in composition reuses the [unified native modules](modules.md) and the
existing ZITADEL / Auth.js identity and Organization contracts. It is for an
explicitly authorized **new empty project**. It does not upgrade a retained local
instance, import its identity/business data, or open production release gates.

Append `docker-compose.public-trial.yml` after the normal module overlays. Set
these non-secret values before the first bootstrap, along with the project,
three unused loopback ports and current-origin plugin directory:

```dotenv
ACCOUNT_PUBLIC_APPLICATION_ORIGIN=https://trial.example.com
ACCOUNT_PUBLIC_IDENTITY_ORIGIN=https://auth-trial.example.com
ACCOUNT_PUBLIC_APPLICATION_HOST=trial.example.com
ACCOUNT_PUBLIC_IDENTITY_HOST=auth-trial.example.com
```

Only exact HTTPS DNS origins on port 443 are supported by this trial layer.
Origins and hostnames must agree. Bootstrap records the two origins in its
existing state volume; restart with the original values. Changing a retained
local installation to public, changing either public origin, or dropping the
public layer is rejected. Keep every original overlay, volume and private file.
The backend also validates the private manifest's public issuer/API as the same
canonical HTTPS DNS origin; its admitted boundary is described in
[`public-trial-identity-origin.md`](../../../docs/architecture/public-trial-identity-origin.md).

Build the backend and UI from the selected Git commit with
[`scripts/build-listingkit-images.ps1`](../../../scripts/build-listingkit-images.ps1)
and keep its commit/tree/image receipt. Build the existing bootstrap/schema-init
targets from the same clean source archive. Use the existing capture-plugin
builder with the exact application origin plus `/capture/1688`; never reuse a
package whose receiver belongs to another installation.

The host Caddy ingress uses `Caddyfile.public-trial`, the same non-secret
environment values, and valid public certificates for both DNS names. The
Compose services keep their existing loopback bindings. After bootstrap, copy
only this project's **public** `trusted-ca:/root-ca.pem` to
`/etc/caddy/trial-upstream-ca.pem` so Caddy verifies the existing localhost
upstream certificates. Do not copy private keys or use certificate verification
bypasses. Caddy preserves the fixed public Host while verifying upstream SNI
`localhost`. Traefik uses its native dynamic-file templating for the two fixed
hosts and retains the same narrow Data Services paths. SQL, Mailpit, Temporal,
storage administration and schema-owner credentials have no public ingress.

Start bootstrap first, install the upstream public CA and Caddy configuration,
then start the same Compose project with all selected overlays. OpenTofu uses
the public issuer and system trust roots, and installs the exact public OIDC
callback. The schema owners, private manifests, Auth.js and tool-package receiver
consume those same configured origins. Runtime does not create schemas.

Private password files are handed to the operator without logging their values.
The existing bootstrap account and native Login V2 remain the login path.
Complete ordinary sign-in and enterprise selection, then use the entries in
`modules.md`. Mailpit remains a private local capture transport: real email/SMS
delivery, provider/model/merchant credentials, Store connections and published
browser artifacts require their original configuration. Missing capabilities
stay disabled/unavailable; startup creates no sample business data or balances.

Record public TLS/issuer/callback checks, service health, unauthenticated access
rejection, retained restart and actual user trial separately. Self-checks do not
constitute user or production acceptance. Stop with the original Compose files,
or stop the authorized VM; preserve every volume. Never use `down -v`.
