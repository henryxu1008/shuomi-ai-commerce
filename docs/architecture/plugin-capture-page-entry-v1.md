# Product capture: source-page primary entry

Execution: [#924](https://github.com/qq550723504/shuomi-ai-commerce/issues/924).
Design Basis: **Independent Architecture**. Admission: **NOT_READY**, pending independent design review before production changes.

## Product outcome and authority

The user approved on 2026-10-24: a source-page capture button as the primary entry,
with shortcut/popup as secondary entries, design review first, then development,
tests and one main PR. No merge, installation/runtime update, real capture/save,
deployment or server collection. Preserve retained data, requests, reservations,
audit, configuration, loading directory and extension identity.

Authorities: [ingestion](plugin-capture-ingestion-v1.md),
[existing entry](plugin-capture-entry-flow-v1.md),
[source quote read](plugin-capture-source-quotes-v1.md),
[listing fields](../product/plugin-capture-listing-fields-v1.md),
[final UI/IA](../product/final-ui-ia-authority.md), Figma
`tg48P46SSXl6TBy9lZwg63 / 31:463`, and this explicit user entry decision.
Application navigation, receiver, account/enterprise confirmation and save pages
are reused in full. The prior design's no-source-button/exact-popup-only internal
entry restriction is superseded only by this reviewed source-entry adapter.

User path: supported product detail → **硕米 · 采集并在应用确认** → existing
extraction and fixed receiver → existing application review/confirm/save →
original result. A small fixed panel uses a branded native button, isolated styles,
visible focus and a live status region. It does not cover the site's purchase controls
by deliberately attaching inside them; it remains separate from site-owned controls.
Shortcut and popup remain available. No drag-position persistence or new UI framework.

## Owners and unchanged contracts

Content script: URL gate, one panel in the top document, display-only state and
explicit user intent. Background: trusted browser sender/document verification and
entry scheduling. Existing Controller/Handoff: one ephemeral payload/key/nonce and
own-created application tab. Application/SRC/Catalog/Collection: consent, identity,
organization/member authorization, submission, durable facts and result authority.

Contract → implementation → injection → consumer:
strict `page.*` messages → background entry adapter → Controller.captureAndHandoff
and original extractor/Handoff → fixed Amazon/1688 receiver → original confirmation
and save. Return uses existing own-tab/window activation, without a URL/navigation.
No new external message, backend port, schema, storage, transaction, retry/UNKNOWN
owner, provider/billing action or restart recovery. Legacy decision: **N/A**;
these are current owners, with no legacy wrapper/compatibility path.

## Installation and activation boundary

Add a static ISOLATED content script at `document_idle`, `all_frames:false`, only
for product-shaped paths on the existing six exact HTTPS hosts. Reuse strict
`capturePageSource` at runtime: unsupported product IDs/paths mount nothing.
Permissions remain exactly `activeTab` and `scripting` and the same six hosts;
no wildcards, additional permission names, `web_accessible_resources` or page bridge.
Static registration provides automatic UI presence without a toolbar activation;
the existing host grants provide access, not a fabricated activeTab user gesture.

Mount/status reads never extract product facts, prepare an operation, open a tab
or POST. Extraction remains user-triggered. The script checks current URL before
each action and on focus/navigation signals; unsupported navigation removes the
panel. It never traverses variants/pages, logs in, solves challenges or fetches
remote source content. A singleton avoids duplicate panels in one document.

## Internal page messaging and data exposure

Keep exact own-popup validation intact. Add a separate narrow `page.*` branch;
do not relax the popup handler or expose commands with `window.postMessage`, DOM
custom events, page-world injection or `onMessageExternal`.

- `page.status`: exact `{type}`; read only.
- `page.capture`, `page.return`, `page.newCapture`: exact `{type, revision}`.
  `revision` is an opaque ephemeral entry revision, never a handoff/operation key.
  New-capture confirmation is presented before `page.newCapture` is sent.
- Browser sender must be the same extension, valid tab ID, top frame, active
  document with a nonempty documentId, exact approved HTTPS source origin and
  strictly supported canonical product URL. Sender tab and document source must
  agree; IDs/URLs/destinations are never supplied by the message.
- Before responding/acting, execute a document-bound URL check in that sender's
  documentId and compare canonical source. Extraction targets that documentId,
  and retains the original post-extraction document/source recheck. A replaced
  document or changed product fails without handing off.
- Only bounded allowlisted primitive input is accepted. Page replies contain only
  `revision`, `loading`, `captured`, `handed`, `sameProduct` and safe error code.
  No raw payload/title, operation key/nonce, receipt ID, receiver/recovery URL,
  account/organization state or arbitrary error enters the source-page DOM.
- Listeners run in ISOLATED context and only accept trusted user click events.
  Site `element.click()`/dispatched events cannot trigger capture/reset/return.
  Background still validates sender and document; DOM is never authority.

## Entry revision, repetition and explicit new capture

Use the existing single Controller rather than a per-tab registry. Background
keeps a small in-memory revision alongside it; it changes on controller reset and
on reserving a new empty capture entry from any of the three entry surfaces.
Revision comparison and synchronous Controller reservation/reset happen together
after asynchronous document checks, with no intervening await. A stale page
action returns `ENTRY_CHANGED` and a fresh narrow projection, with no mutation.
The page requires another explicit user action; it never replays automatically.

| Current state | Page behavior |
| --- | --- |
| Empty | Primary “采集并在应用确认”; user click calls existing combined entry |
| In flight | Busy projection, controls disabled; existing cross-entry locks prevent duplicates |
| Handed, same product | Primary “返回应用核实”; activate original app tab/window |
| Handed, another product | Explicit “已有另一件商品的采集” and “返回已有采集”; never label it as capture of this product |
| Captured/handed, user wants another capture | Secondary “采集当前商品（新操作）” opens explicit confirmation; cancel has no effect; confirmed revision-matching action resets only when idle and immediately reserves existing combined entry |
| Stale action or confirmation | Reject without resetting/creating a key; update visible state and close stale confirmation |
| Unknown transport, unavailable original tab, expired worker | Visible safe guidance to original application; no automatic retry/new tab/new key/save; transport error disables mutation until explicit status refresh |

`page.capture` requires empty state; it cannot silently return another product if
the UI became stale. `page.return` requires an existing handed operation. A new
entry from shortcut/popup invalidates a pending page confirmation. Both already
use the same Controller locks and keep their existing behavior. Old application
pages/requests are not modified when an explicit new capture starts.

On response loss retain uncertainty, disable mutation and offer only status
refresh; do not interpret transport loss as failed save. A page that observes a
different entry revision refreshes its projection without executing its rejected
intent. Worker loss remains the documented ephemeral limitation; the UI explicitly
directs possible prior submissions to the original application, without inventing
recovered identity/persistence. The application remains the only save authority.

## Threat model and validation

Must protect against arbitrary webpage messages/forged sender or frame, stale
documents, raw evidence/key disclosure, synthesized DOM-triggered capture,
cross-product misleading labels, duplicate entry and stale confirmation resetting
a newer operation. Source content remains untrusted; this does not certify source
authenticity or protect against the source site hiding/removing/spoofing UI.
Browser/extension compromise, site challenge bypass and durable worker recovery
are outside scope. No new Accepted Risk is used to weaken existing requirements.

TDD in existing Vitest/jsdom/browser API mocks: no mount on invalid paths/frames,
single mount and safe accessible rendering, no extraction on load/status, trusted
versus synthetic click dispatch, strict page-message/sender/document gates, narrow
replies, both platform receivers, stale document/action/confirmation rejection,
in-flight cross-entry coalescing, same/other-product projection, explicit new flow,
original-tab failure/UNKNOWN and unchanged external own-tab checks. Frozen wire
fixtures/parsers/save contracts remain unchanged.

Run extension tests/typecheck/lint/build and existing isolated transport fixture
where applicable, adapting it to click the real injected button, not adding a runner.
Use disposable task profiles and fixture responses only; no retained browser/profile,
real source requests, database or application runtime. Final independent review
checks the full diff/call path and actual evidence. CI/fixtures are development
evidence, not user acceptance; retained runtime and production verification NOT_RUN.

Native APIs: [content scripts](https://developer.chrome.com/docs/extensions/develop/concepts/content-scripts),
[messaging](https://developer.chrome.com/docs/extensions/develop/concepts/messaging),
[scripting](https://developer.chrome.com/docs/extensions/reference/api/scripting).
Reuse native extension/DOM APIs and existing esbuild/jsdom/Vitest; no new dependency.
