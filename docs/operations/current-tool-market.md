# Current application Tool Market composition

Refs #619, #613. Reuses the frozen [Tool Market design](../architecture/tool-market-v1.md)
and [module handoff](../engineering/tool-market-runtime-handoff.md).

Use a new isolated installation. Product acquisition and collections continue to
use one current Product pool; the prepared Asset owner is shared by future actual
consumers. Tool Market uses a dedicated `tool_market` database and the existing
`tool_market` schema. Explicitly install with `toolmarketpersistence.InstallSchema`
or its embedded SQL under a `tool_market_owner` NOLOGIN schema owner. Never migrate
or replace retained instances. The serving role is `tool_market_runtime`, with
CONNECT, schema USAGE, SELECT/INSERT/UPDATE on activations, requests and commands,
and SELECT/INSERT on events and attachments. It has no schema CREATE or owner membership.
The complete Figma form uses the current fresh schema with scenario/contact/consent
columns and the private attachments table; see [#730 Design Basis](../architecture/tool-customization-figma-v1.md).
Grant only SELECT/INSERT on attachments, with no UPDATE/DELETE/TRUNCATE. VerifySchema
rejects an installation missing these columns or this table. Do not upgrade or reset
a retained database by running the initializer; this delivery validates a new isolated database.

The private native manifest adds:

```json
{
  "toolMarket": {
    "database": {"host":"127.0.0.1","port":5433,"user":"tool_market_runtime","password":"<private runtime credential>","database":"tool_market","maxConnections":4},
    "captureAppURL":"https://localhost:31544/capture/1688",
    "packageDirectory":"/private/tool-release"
  }
}
```

`packageDirectory` is optional. It contains the existing build script's ZIP and
`release.json`. The recorded full capture URL must match `captureAppURL` exactly;
the SHA verifies the ZIP and its compiled receiver. An invalid or missing record
leaves downloads unavailable. The nonfixture extension builder accepts the fixed
HTTPS application receiver, including a trusted localhost installation with an
explicit non-default port. Use the actual installation URL; image admission still
rejects private/local hosts. Do not relabel a fixture as a release. Online/local capture
availability derives from the actually injected acquisition and receiver modules;
enterprise activation does not authorize their use or prove provider success.

Serving opens the restricted pool, verifies schema, uses current Organization
resolver/Casbin, and admits the exact domain descriptors through the app module.
The handler injects `ReadAuthorize=admission.AuthorizeRead` for GET, downloads
and action visibility (CachedRead), and `Authorize=admission.Authorize` for writes
and transaction guards (LiveWrite). LiveWrite never falls back to cached grants.
The existing module catalog enables `tools` and `tools-custom` and uses
`authz.ToolMarketModulePermissions` before `NewListingKitAuthorizer` creates its
module enforcer. Native enterprise roles obtain read or read/customize through
current `RoleModules`; the permission projection is not a grant. Manage stays
limited to `listingkit_admin`. Retired viewer/operator roles must not be restored
through `platformAdminRoles`.
Schema maintenance credentials stay in initializer-only volumes; serving mounts
only its manifest and runtime credentials. No startup DDL or recovery owner is added.

Set UI `LISTINGKIT_TOOL_MARKET_ENABLED=true` only when this module is injected.
The three entries are `/workbench/tools/official`, `/workbench/tools/mine`, and
`/workbench/tools/custom`. The exact `/workbench/tools/custom/review` route uses
independent verified platform identity and does not require customer membership.
Enterprise siblings retain their Organization gate. Seven future tools stay closed.

Customization children are `/workbench/tools/custom/new` and
`/workbench/tools/custom/progress`. The form saves scenario, contact name, phone/WeChat,
explicit contact consent (initially unchecked), and up to three private reference
files (2MiB each; PDF, PNG, JPEG or UTF-8 text/CSV). Files are saved in the same
transaction as the request and can be downloaded in enterprise and specialist details.
Contact consent is recorded with the submitting actor/time and server version;
the application does not automatically contact anyone. No payment or installation fact is created.
Large pending payloads survive navigation in bounded memory; after reload or eviction,
the remaining session marker blocks a new submission if its bytes are unavailable.
Use the progress page to inspect saved requests and ask the platform to verify the original
operation; do not clear browser storage or submit a new key to recover an unknown result.

Local trial: administrator enables acquisition → member reads the enterprise list;
member assigned the `tools-custom` module submits a synthetic requirement →
platform specialist records progress →
enterprise reads the saved events. This is local development verification, not user
acceptance, payment, tool installation or provider execution. Browser/tool execution
not actually performed is recorded NOT_RUN. Retain named volumes with stop/start;
use the same base and Issue36 overlay. Native localTrial excludes real acquisition,
so the new Product-serving manifest omits that synthetic Review module; old trials
and data remain intact. No merge, Issue close, production or real-data authority.
