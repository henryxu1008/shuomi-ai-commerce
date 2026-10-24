# Product capture: original-tab return across worker recreation

Execution: [#927](https://github.com/qq550723504/shuomi-ai-commerce/issues/927).
Design Basis: **Independent Architecture**. Admission: **IMPLEMENTATION_READY**
after the existing `amazon_fix_review` independently reviewed design
`ef185ffff61c573c15f0cbed6187f77a3d2fe8cf`, with no design BLOCKER. The #927
review record precedes production changes. Shared initialization, interrupted
writes, bounded handoff timing and return-only semantics are IMPLEMENTATION_TEST
within this frozen design; final code/evidence review remains required.

## Product authority and outcome

The user's 2026-10-24 decision approves native `chrome.storage.session` for
original-tab location only, same-browser-session worker recreation, and one
implementation PR after independent design review. Source-page button, shortcut
and popup share this behavior. No merge, retained instance/plugin update, real
capture/save, deployment or server collection is authorized. Existing business
data, original requests, reservations/audit, configuration, unpacked directory
and extension identity remain untouched.

Authorities: [current entry](plugin-capture-entry-flow-v1.md),
[source-page entry](plugin-capture-page-entry-v1.md),
[ingestion](plugin-capture-ingestion-v1.md),
[source quote read](plugin-capture-source-quotes-v1.md),
[final UI/IA](../product/final-ui-ia-authority.md), Figma
`tg48P46SSXl6TBy9lZwg63 / 31:463`, and this explicit user decision. Existing
capture buttons, receiver routes, account/enterprise confirmation and original
operation checks are retained. No Figma navigation/layout change is needed.

This decision supersedes only the prior no-session-storage/no-worker-return
limitation. Full payload recovery, browser restart and extension disable/reload/
update recovery remain outside scope. The existing ephemeral transport and the
application's publication authority remain unchanged.

## Root cause and responsibility

`background.ts` creates a fresh Controller and entry revision whenever its MV3
worker starts. Handoff's payload, nonce/key, own-created app tab and outcome all
live in that worker. The page can outlive it, so a previously handed page sees an
empty new Controller/revision and correctly refuses a stale action. Popup and
shortcut must also stop treating this situation as an initial capture.

The five-minute Handoff TTL limits transport reads; it does not clear the stored
tab ID or the Controller's captured/handed flags. It cannot explain that state
becoming empty. The retained observation is consistent with worker recreation;
its exact browser lifecycle event was not directly observed.

Owners remain: source adapter for trusted intent/document gating;
Controller for coalesced capture/handoff/return; Handoff for ephemeral transport
and strict external sender/key/nonce checks; application/SRC/Catalog/Collection
for authorization, save, original-key reads and durable results. A new small
ReturnSession component owns only one navigation projection, not business facts,
submission state or a retry/recovery executor. Legacy decision: **N/A**.

## Session record and permissions

Use one namespaced session item with a strict bounded schema:

- `version: 1`;
- `revision`: current opaque entry UUID, retained across worker recreation;
- `sourceURL`: strictly validated canonical supported product URL;
- `phase: pending | bound`;
- `tabId`: a nonnegative integer, required only for bound and absent for pending.

No payload/title/images/prices, account/organization/member, credential, original
operation key, Handoff nonce, recovery URL, operation/publication ID or saved
outcome is stored. Receiver origin/path remain compiled configuration. Source
replies remain narrow primitives, with no raw session record or private identity.

Add only the native `storage` permission to the existing manifest. Restrict the
session area to `TRUSTED_CONTEXTS`; source content scripts cannot read it. The
six source hosts, CSP, external receiver matches, commands, extension name/key
policy and loading directory remain unchanged. No `tabs`, `sessions`, broad host
grant, source-page storage, new dependency or background keepalive.

## Initialization, transitions and interrupted writes

Register browser listeners synchronously, then make status and all three entry
adapters wait on one initialization promise. Hydrate the navigation record and
its revision before deciding capture versus return. No storage read error or
malformed record is interpreted as empty. Such errors remain visibly uncertain
and block new capture/reset, rather than deleting unknown state or retrying.

The existing Controller still owns in-flight coalescing. Session writes and
explicit reset must remain inside its/entry's single busy boundary, including
the awaits; no late bind/write can overwrite a newer entry. Do not persist or
reconstruct promises/locks, Handoff state, payload availability or saved outcome.

| Event | Session effect | Allowed next behavior |
| --- | --- | --- |
| No record, initial explicit capture | Existing fixed-document extraction | No capture on mount/status |
| About to open existing fixed receiver | Await pending record before tabs.create | A failed/unknown write does not open a tab |
| tabs.create returns own valid ID | Await bound record before exposing completion | Same ID is handed to existing Handoff binding |
| Worker recreation with bound record | Hydrate same revision/source/tab ID | Return-only projection, without claiming payload/save |
| Worker recreation with pending record, or tab creation/bind failure | Retain uncertain projection | No automatic create/extract/new key/POST |
| Explicit confirmed new capture/reset while idle | Await deletion/replacement of this navigation item; then replace Controller/revision | Old application tab, key, request and data remain |
| Read/write/reset failure | Latch unavailable/uncertain; retain whatever existing evidence is readable | No mutation retry or automatic reset |

Bind the existing Handoff immediately after tabs.create returns, before awaiting
the bound navigation write. The original receiver's bounded reads must not wait
for a storage write; no external message or additional retry is introduced.

If tabs.create succeeded but its ID was not recorded before termination, the
pending record deliberately cannot recover that identity. It must not scan for a
matching tab or open a replacement. This bounded UNKNOWN follows the existing
no-replay contract, rather than inventing a new cross-boundary recovery protocol.

## Return-only state and the three entry surfaces

Expose navigation capability separately, e.g. `returnState: none | available |
uncertain`, alongside the existing captured/handed/loading projection. Restored
location is not a reconstructed payload and never sets captured/handed or
published to true. Same-product is computed from the validated stored source and
sent only as a boolean. The source adapter keeps trusted clicks, documentId/URL
checks and opaque revision comparison intact.

- Source primary: available location means “返回应用核实” (or existing other-product
  return wording), even when the recreated worker has no payload. Old uncertainty
  may clear for navigation only after a valid same-entry location is observed;
  it never unlocks automatic capture. Uncertain location disables primary capture
  and retains explicit new-operation guidance.
- Shortcut and popup primary: available location invokes the same existing
  Controller return operation, never fresh extraction; uncertain state reports
  guidance. Popup exposes its existing original-operation button from navigation
  capability, not from availability of a private recovery URL.
- New capture remains the explicit existing confirmation path. All reset/new
  actions use the shared busy/revision boundary. A stale revision/confirmation
  is rejected without executing its intent or replaying it automatically.

Controller's original-tab return keeps its current coalescing and receives the
navigation target through dependency injection. It reuses the current
`tabs.update(id, {active:true})` and current-window focus code. It never supplies
a URL, reloads, searches tabs, copies markers, rebinds Handoff or creates a new
tab/key. A moved tab uses its current window. A closed/unavailable tab reports
failure without replacement. If the original tab navigated away, focusing it is
navigation assistance only, never proof that an operation was recovered/saved;
this retains the existing approved return boundary without new host permission.

The original application remains the only place to check the save result, using
its existing account/enterprise/member checks, original operation key, and
Amazon original-tab session marker/digest. Returning neither submits a request
nor automatically clicks an application check/save button. An already-loaded
receiver may retain its payload and continue its existing user-confirmed save;
the recreated extension cannot reread/replay a lost payload. External messages
keep the same strict ephemeral transport rules and do not bootstrap a new
Handoff from the navigation record.

## Scope, threat model and verification

Must: same-session worker loss retains only original location; no stale action
replay/duplicate capture, tab, key or POST; no raw session data to source DOM;
all entry surfaces obey initialization/uncertainty; strict own sender/document
and application authorization boundaries retained. Storage/partial-binding,
concurrent entry/reset and closed-tab cases are implementation tests within this
design, not reasons to create a general lifecycle/verification framework.

The #927 initial-entry correction reuses these owners: before a trusted initial
source-page capture, read the existing `page.status` once inside the same local
busy boundary. Only a previously pristine view that is still readable, idle,
uncaptured, unhanded and navigation-free may send its single `page.capture` with
the newly read revision. Changed, known or uncertain operations only update the
view and require another user choice. Background revision/document checks stay
strict; a race after that read is rejected without mutation retry. Return and
new-operation confirmations retain their original revision binding. No empty
entry revision or payload is added to session storage.

Outside scope: browser/extension compromise, hostile source UI, challenge bypass,
payload durability, automatic save recovery/retry, browser restart/extension
reload/update identity recovery, reconstructing pre-fix lost location, arbitrary
tab discovery, backend/schema/tenant/billing/provider changes, marketplace
package admission, Go module/directory rename and any retained environment action.
No new accepted risk weakens current business/security requirements.

Use existing Vitest/browser API mocks and disposable dual-platform transport
fixture. TDD first demonstrates recreated background losing return. Cover both
receiver platforms; all three entry surfaces; stable revision and separate
payload/navigation states; shared startup/concurrent actions; exact narrow
session contents/access level; failure before/after tab binding; read/write
failure; stale new confirmation; explicit new operation; moved/closed original
tab; no new extraction/tab/key/save on return; unchanged payload TTL and external
sender validation. Adapt the existing fixture for worker termination and return;
no new runner, retained profile, real source request, database or runtime.

Run related extension tests/typecheck/lint/build, applicable documentation guard
and stable-head PR CI. Existing independent Reviewer checks this architecture
before implementation and the complete final diff/evidence at delivery. These
are development/review evidence, not user acceptance. Runtime/plugin update,
real recapture/save and production acceptance remain NOT_RUN/NOT_AUTHORIZED.

Native references: [MV3 lifecycle](https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle),
[storage.session/access/initialization](https://developer.chrome.com/docs/extensions/reference/api/storage),
[tabs](https://developer.chrome.com/docs/extensions/reference/api/tabs),
[Edge API support](https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/api-support).
