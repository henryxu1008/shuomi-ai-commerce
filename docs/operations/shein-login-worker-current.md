# Current standalone SHEIN login worker

Implements the user-confirmed [#301/#863 design §1–§7](../architecture/shein-worker-numeric-persistence-retirement-v1.md), frozen at `94fa4cab`. This is a future operating entry for an explicitly authorized fresh installation. This PR does not install, deploy, stop or inspect any retained instance, data, credential or Profile. #301 remains OPEN; real SHEIN login and user acceptance remain NOT_RUN. The optional capability is not an internal-trial prerequisite.

## Inputs and owners

Use a current active SHEIN Store UUID and opaque Organization ID. Store Center owns its business identity, member grants and merchant identity. One deployment binding supplies a new canonical revision UUID and fresh private credential reference; there is no numeric account table, account CRUD or ID translation. SourceAccountRegistry remains the existing 1688 owner.

Copy [the current JSON example](../../config/shein-login-worker.current.example.json) to a new absolute private file and replace all synthetic values. Use a fresh installation UUID, explicitly assigned Redis endpoint/database, new empty private Profile/artifact roots, current Store database and native identity configuration. Supply absolute `browser_path` and `driver_directory` for an explicitly provisioned browser/Playwright driver; serving does not install/download a driver or fall back to a shared driver cache. Optional `viewport_width`/`viewport_height` keep the existing browser viewport behavior. No global YAML, Redis fallback, legacy key, old Profile or old credential is imported. Changing endpoint/proxy/browser/binding metadata changes its fingerprint; rotate binding revision for a new credential binding. Maximum 100 bindings, config 1 MiB, each credential/token input 64 KiB. Unknown fields and trailing JSON are rejected.

The fresh private login JSON contains only `username` and `password`. Optional `proxy_file` refers to another fresh absolute private file containing the explicit proxy URL. Only the executor reads login material. Producers load binding metadata and their current caller JWT. Keep files and roots restricted to the process owner, SYSTEM and Administrators on Windows, or owner-only permissions on Unix. Keep artifacts private: they can contain page/network diagnostics and are never CLI results or Store audit payloads.

An explicitly authorized fresh Store setup uses the existing schema-owner command with `-shein-login-worker`. Provision the dedicated `shein_login_worker` login role through the normal database owner first; the command does not create or change its password. The optional table is `workbench_store_shein_login_bindings`; ordinary Store startup does not require it. Worker startup verifies this schema and its exact narrow role: Store/member reads, five identity/version metadata UPDATE columns, audit SELECT/INSERT, and the control table SELECT/INSERT/UPDATE. Paid service, official connection, member grant mutation, DDL and DELETE are denied. Schema installation is an owner action, never a serving startup action.

The current caller JWT is verified by the native ZITADEL verifier, then narrowed to the configured issuer, project audience, subject and expiry. Every command/execution/Store commit rechecks the exact active project authorization, current member identity, Store lifecycle permission and current Store member grant. Flags and manifest bindings do not grant access. Optional `auth.role_policy_database` uses the existing read-only `organization_role_policy_reader` configuration and owner for custom enterprise roles; without it, custom-role access fails closed. No new role-policy facts or static operator allowlist are introduced. The existing optional organization suspension checker contract remains deny-only.

The current repository has no production local organization-suspension owner or adapter: the main application's resolver also passes a nil overlay, whose documented meaning is no additional local suspension policy. This worker preserves that current contract and native grant/Store revocation; it does not advertise or invent a local organization-suspension feature. Configured overlay suspension/errors deny access, as tested through the existing port. A future production policy owner would require explicit composition under that owner's approved contract.

The retained Docker/Kubernetes source examples now consume an explicit private current config instead of shared numeric DB/Redis environment bindings. The image provisions its driver at `/app/playwright-driver`; set that exact path and `/app/chrome/chrome-144/chrome` in a new Linux config. The Kubernetes example expects a **new separately provisioned** `shein-login-worker-current` Secret with `current.json`, `client-secret`, `service-token`, and `login.json`, mounted as individual readonly regular files with mode 0400. Use new empty private roots below `/app/tmp/shein-login-v1`; add explicit file mounts for any additional bindings/proxy. The historical release bundle is not a current-application publisher. Do not apply it to retained resources under this PR's source authority.

## Commands

Build the existing binary with `go build -o shein-login-worker.exe ./cmd/shein-login-worker`. Every command uses `-config C:/private/new-shein-worker/current.json`. The six producer actions also require `-organization <current-org> -store <Store-UUID> -token-file <fresh-private-current-JWT-file>`.

| Action | Additional input | Behavior |
| --- | --- | --- |
| `serve` (default) | Optional `-health-port 8086`; 0 disables | Starts only the independent consumer and health-only `/livez` / `/healthz`; prepares only its new marked roots. |
| `activate` | `-request-key <UUID>`; first claim uses expected generation 0 | Live-authorized Store transaction claims this installation/revision. A later activation requires `-expected-generation <confirmed-prior-generation>`. |
| `login` | `-request-key <UUID>`; optional explicit `-force-login` | Enqueues a credential-free attempt; request UUID is its attempt UUID. Producer never launches a browser. |
| `status` | `-attempt <original-UUID>` | Reads only that actor/member/scope's original attempt; no login, Cookie refresh or Store sync. |
| `verify` | `-attempt <original-UUID>`; verification code on stdin | Delivers the code to the original live attempt, with a bounded TTL; never put it in argv. |
| `cancel` | `-attempt <original-UUID>` | Cancels an active attempt. A late result cannot overwrite cancellation or persist a Cookie. |
| `quiesce` | `-request-key <UUID>` | Fences new claims, drains the exact active owner and waits for its actual browser/HTTP closure confirmation. |

For example, after separate installation/activation authorization:

`/livez` is process liveness. `/healthz` is bounded readiness for Redis, the current Store database, any configured role-policy database, native token verification and the scoped authorization API. It verifies the configured service token and performs a readonly lookup for that service identity in one configured organization; an empty grant is healthy and grants no business permission. Dependency loss returns 503; recovery can restore readiness. The source Kubernetes startup/liveness probes use `/livez`, so a provider outage does not itself restart an executor or turn UNKNOWN into closure proof.

```powershell
./shein-login-worker.exe -config C:/private/new-shein-worker/current.json -action serve
./shein-login-worker.exe -config C:/private/new-shein-worker/current.json -action login -organization <org> -store <Store-UUID> -token-file C:/private/new-shein-worker/caller.jwt -request-key <request-UUID>
./shein-login-worker.exe -config C:/private/new-shein-worker/current.json -action status -organization <org> -store <Store-UUID> -token-file C:/private/new-shein-worker/caller.jwt -attempt <same-request-UUID>
```

Successful producers output a bounded JSON result with scope, generation, attempt/status, sync status and general error code. Errors produce a safe JSON failure and nonzero exit; no password, Cookie, verification code, raw provider body or artifact path is returned. `login.status=succeeded` means Cookie/login success only. `sync_status=applied` separately means the Store identity command committed. `blocked` leaves record/service/official connection and other Stores unchanged. A web Cookie never activates an official connection, paid service, publication or observation API.

## Response loss and ownership

Retain every request key and attempt UUID. After UNKNOWN, query/replay the original key; do not generate a fresh key to retry the same intent. Same key/same payload converges; changed payload conflicts. A pending success may resume only its readonly identity lookup and original fixed Store command. Once merchant and expected Store version are fixed, they cannot be recomputed under that key. Store update and immutable audit receipt commit in one local transaction; a unique violation rolls back to its savepoint before the blocked receipt is recorded.

Cookie reuse and pending identity reads load only endpoint/proxy metadata, without reopening the login username/password file. A fixed merchant/version replay needs neither that file nor a Cookie. Only starting a browser reads the login secret. Local Redis cancellation checks run every 300 ms; periodic live authorization/Store ownership checks run every 15 seconds, with fresh checks still required before browser/code/Cookie/identity/Store actions. Quiescing can take up to that recheck interval to begin draining; elapsed time never proves closure or releases ownership.

Execution ownership is Store-scoped across Redis namespaces/installations. Every browser or Cookie-bearing supplier/warehouse request needs the exact generation, active attempt and owner nonce before execution. A new installation/revision activates only after the old row is quiesced, has no active executor, and closure is confirmed; activation compares and increments the prior generation. Replay receipts describe the original control operation, not a new claim or a current health assertion. A fenced activation may be retried using its original request key after closure is confirmed.

Explicit force-login bypasses both the cached Cookie and the ordinary persistent browser session: it uses a new attempt UUID directory beneath the owned binding Profile, refusing to reuse any existing forced-attempt directory. It does not clear/copy the ordinary Profile. Stream reclaim checks Store ownership before changing an attempt; a still-unclosed owner keeps its state and delivery pending, even if the stream has been idle for 45 seconds. An idle quiesce records/replays `quiesced`; a quiesce that originally awaited an owner records/replays `quiescing`, even after later closure or a new generation. That immutable result is not a fresh closure assertion.

The consumer reserves one actual execution slot before reading one stream entry. While all slots are occupied, further requests stay undelivered rather than entering the pending list as unstarted local work; another normally available consumer can read them. This keeps stale recovery from dropping valid requests merely because a verification wait occupied the configured capacity.

The executor installs its exact closure confirmation before claiming ownership. A lost claim response is retried once with the same scope/attempt/nonce; Store recognizes only that identical current owner as an idempotent claim. Before any browser or HTTP work starts, a failed handshake can confirm only its own exact unstarted owner. Other nonces remain fenced, and an unavailable closure result remains UNKNOWN rather than permitting takeover.

TTL, stream ACK, terminal login status, a lost connection, heartbeat absence and stale-job reclaim do not prove a browser has stopped. A crashed/UNKNOWN owner remains fenced. There is no force-takeover, automatic process scan/kill, lock deletion or private Profile adoption. Profile busy fails closed. Any real exclusion/repair of an old executor requires separate authorization to its environment owner; this source implementation does not perform it.

## Evidence and limits

Current tests cover scoped CLI dispatch, native-verifier/live-grant rejection, strict binding inputs, cancellation and atomic Cookie state, fixed-payload response-loss replay, closure fencing, current Store organization/idempotency/identity behavior, and real task-owned PostgreSQL/Redis boundaries. The historical `go test -race ./internal/listingadmin -count=1` directory contains current-contract tests only. Original workflows are unchanged. Exact final HEAD, CI and independent review are recorded in #863; implementation checks are not real-provider or user acceptance.
