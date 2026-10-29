# 1688 / Amazon browser capture

A Manifest V3 extension for explicit 1688 or Amazon product capture, with an optional
Amazon same-family variant scope. Chrome and Edge use the same build. This directory owns capture
and local unpublished drafts plus ephemeral transport; it cannot publish a Product or authorize an import.

## Durable drafts (execution #989)

Current recovery authority: [durable draft design](../../docs/architecture/plugin-capture-durable-drafts-v1.md).
The service worker uses `idb` / IndexedDB as the single durable owner. A single
capture is committed before handoff; every Amazon family observation is committed
before the progress count advances. The popup lists retained local drafts and
offers explicit restore or confirmed local-only cleanup. Stop, new operation,
worker reconstruction and page closure do not delete committed facts. Recovery
starts paused and never resumes source reads or business saves automatically.

The app reads metadata after its trusted account/organization/member context is
ready, then reads the draft, verifies its frozen body and original key, and asks
for save confirmation. An unknown save stays available for original-key checking.
Amazon permits a manual identical-key/body retry only after an explicit original
lookup reports absence; the existing backend transaction remains the authority.
Known 1688 pre-admission rejection can be retried explicitly without changing
the original operation. Saved reports retain the local copy until explicit cleanup.

The candidate adds `unlimitedStorage` but retains an 8 MiB logical draft budget
and all existing capture limits. Quota failure retains the last committed data;
corrupt drafts stay stored and are isolated from healthy recovery. Clearing the
extension's browser data, removing it or disk failure can still lose local drafts.
No historical in-memory capture is migrated or claimed recovered. Development
tests and fixture-browser checks are not acceptance on the user's retained Edge.
Installing/updating/reloading a candidate still requires separate authorization.

The following older entry notes remain useful for supported sites and explicit
user actions. Their memory-only loss, no-replacement-tab and no-retry descriptions
are superseded by #989's design and current recovery implementation.

## Direct entry and return to the original application

[Execution #955](https://github.com/qq550723504/shuomi-ai-commerce/issues/955) adds
the reviewed [optional Amazon family scope](../../docs/architecture/plugin-capture-amazon-variants-v1.md).
On an Amazon detail page, **同时采集其他颜色／尺寸（本次按钮）** is off by default.
Choose it before the source-page capture button, or inside the existing new-operation
confirmation. It applies only to that action; idle shortcut/popup capture remains selected-only.
A normal extension tab shows observed/discovered counts and reuses one inactive reader tab;
the original source tab stays in place. Only explicitly mapped same-family ASINs are read.
Errors pause with **继续采集**, **仅使用已采数据** and **取消本次采集**; limits allow
partial use or cancel. Completion sends one aggregate to the existing application for one
confirmation/save. Every variant keeps its own observed price/currency and specification image;
the anchor supplies common product text/gallery. Even N/N remains a partial-family observation.
While active, all entries return to the same progress page; after handoff they return to the
original application. Keep the progress tab open: refreshing/closing it interrupts unhanded
facts, and browser restart/extension reload does not restore the batch. No automatic retry or
business save is added. Installing or trying this candidate remains a separately authorized step.

Amazon verification and sign-in pages pause with a specific reason. Use **查看需处理的商品页**,
handle the source site's request yourself, return that reader to the original pending specification,
then explicitly choose **继续采集**. A ready matching child is read in place without another navigation;
an unresolved check remains paused. The extension does not handle verification, automatically retry,
raise collection limits, or change the existing one-time handoff/save. Keep the progress page open.

[Execution #924](https://github.com/qq550723504/shuomi-ai-commerce/issues/924) adds
the reviewed [source-page primary entry](../../docs/architecture/plugin-capture-page-entry-v1.md).
Supported Amazon/1688 detail pages show **硕米 · 商品采集** with **采集并在应用确认**.
Loading the page only mounts this control and reads limited status; extraction
starts on a real user click. The isolated content script uses the existing exact
HTTPS site grants, without additional hosts or permissions. It passes no product
payload, destination, tab ID or operation key through page messages.
Automatic button display requires the browser to allow the extension on that
site; a withheld site grant cannot be replaced by the content script. The popup
and shortcut remain alternative entries under the browser's normal permissions.

After handoff the primary button becomes **返回应用核实**. Another product's page
explicitly shows **返回已有采集**; it does not pretend the old operation belongs
to the new product. **采集当前商品（新操作）** requires confirmation and rejects a
stale confirmation after another entry changes the operation. Busy actions do not
duplicate capture. A lost connection disables mutation and offers explicit status
refresh; the original application remains the save/result authority. Shortcut and
popup are secondary entries into the same Controller/Handoff.

[Execution #920](https://github.com/qq550723504/shuomi-ai-commerce/issues/920) follows
the reviewed [entry design](../../docs/architecture/plugin-capture-entry-flow-v1.md).
On the current supported product page, the named browser shortcut (suggested
`Alt+Shift+P`) captures and immediately opens the fixed application confirmation
page. The popup's **采集并在应用确认** button runs that same operation. Saving still
requires confirmation in the existing application under its current account and
enterprise; neither entry submits a backend save.

The popup shows the shortcut actually assigned by the browser. A collision or
user removal is shown as unassigned; it does not silently claim the suggested key
works. Users can use the popup button or assign a key through their browser's
extension shortcut settings. The command is browser-scoped, not global.

Repeated entry after handoff and **返回原应用核实操作** activate the original
own-created application tab/window, without navigation, a new recovery tab, key,
capture or save. This preserves Amazon's original-tab session marker. A closed
tab is reported honestly, without opening a replacement. **开始新的采集** retains its explicit confirmation and is
refused while capture, delivery or return is in flight. Keep an original possibly
submitted operation's page/key; a transport failure is not proof of a failed save.

The reviewed [#927 session-return increment](../../docs/architecture/plugin-capture-session-return-v1.md)
retains only one original-tab location in native `chrome.storage.session`, using
the additional `storage` permission. A worker recreated within the same browser
session can return through the source button, shortcut or popup even after its
payload is lost. The record contains only revision, canonical product source,
pending/bound stage and original tab ID; content scripts cannot read the store.
It contains no payload, account, operation key, handoff nonce or saved result.
All entries wait for initialization, and pending/invalid/unreadable location does
not silently become an initial capture. Popup new-operation confirmation is also
bound to the revision that was displayed when it was opened.

The receiver and application save/original-key checks stay unchanged. Returning
does not restore a transport payload, retry or save. Browser restart and extension
disable/reload/update clear session storage, so those cases and pre-fix lost
locations still require the original application page. No tab discovery,
replacement, local/sync storage or backend recovery service is introduced.

These source changes do not update an existing installed extension or runtime.

Field/quote increment: [execution #916](https://github.com/qq550723504/shuomi-ai-commerce/issues/916), based on the reviewed [source quote read design](../../docs/architecture/plugin-capture-source-quotes-v1.md) and existing [ingestion design](../../docs/architecture/plugin-capture-ingestion-v1.md). The original image relationship increment is #906.
New 1688 captures use the exact `captureVersion=2 / schemaVersion=2 / browser_acquisition/v2 / src2b-browser-capture-v2 / 1688-browser-dom/v3` combination. Each variant carries its own `images: [{url, role}]`. Only an explicit image on the SKU entry or its exact property value (`skuProps[].value[].imageUrl`) establishes that relationship. A product album never supplies a guessed SKU image. Other DOM structures remain unsupported in this increment.

The 1688 receiver is `/capture/1688`. The same extension also selects Amazon's exact US/CA/UK/DE/JP product URLs and sends them only to the compiled sibling `/capture/amazon` at the same application origin; there is no second shell or runtime destination. Amazon's strict source/observation contract and receiver/save backend originated in [execution #905](https://github.com/qq550723504/shuomi-ai-commerce/issues/905). Amazon observes the selected dimensions, rendered product overview/detail attributes, availability text and current product gallery. Only the main image is explicitly linked to the selected variant; gallery images remain at product level. Default capture does not traverse; the explicit family option above adds bounded mapped-child reads without claiming a complete family, inferring inventory quantities or guessing currency from a site. Both source validators run before document-bound handoff.

1688 also reads existing `skuRangePrices` and `currentPrices` models when higher-priority `offerPriceRanges` has no amount. It retains decimal strings and reads safe, already-loaded description text while skipping executable, form, hidden and frame subtrees. It does not fetch external descriptions. The original confirmation/save path and wire versions are unchanged.

The listing-material increment also reads Amazon's explicit `colorImages.initial` metadata using static Acorn/JSON parsing. Only a group bound to the observed MAIN image can upgrade exact visible thumbnail URLs using `hiRes`, then the original `large`; URLs are never manufactured. Conflicts or unsupported metadata preserve known images with a missing-fact reason. The main/thumbnail duplicate is merged, and gallery/detail images never become implicit variant images. Current-product technical parameters, safe A+ paragraphs and product-description images reuse the existing attributes/description/images fields.

1688 supplements the known context attributes with two-cell parameter rows and images inside the already-loaded description container. Original packaging/shipping `unitWeight` values retain their decimal text; an absent unit is labelled explicitly rather than assumed or converted. Supplier services, videos, hidden galleries, external description fetches and other SKU families remain outside this increment. See [listing-material design](../../docs/architecture/plugin-capture-listing-materials-v1.md).

The existing 1688 receiver is `/capture/1688`. After explicit confirmation under the current user and organization, the original save chain returns an operation receipt. Open **我的数据 → the saved batch → 商品详情 → 规格图片** to expand each source variant; **下载原始资料** keeps the exact version and nested image relationships. Images are source URL candidates, with the existing asset approval requirements.

In that same detail, **来源报价（参考）** reads the existing SRC facts for the selected publication/version. Amounts retain their original strings; unknown currency and minimum quantity are labelled explicitly. Quotes are reference information, separate from the merchant's selling price and stock. A failed supplementary quote read leaves known product data visible. Empty quotes mean no product-level quote was recorded, not that the source page has no price. Variant quote facts use the same human-readable format; the JSON download retains the original attributes and fixed-version quote projection. This change does not recapture or backfill older records.

The v1 wire fixture, canonical bytes and command hash remain frozen for historical recovery. New captures never fall back to v1. Capture records without a frozen command or with a mixed descriptor/contract/parser combination fail closed. No schema, image table or migration is added.

Local Go/PostgreSQL, UI, and browser transport fixtures are development evidence. Installing into the user's browser, real source collection/save, retained-instance changes and user acceptance require their separate authorized run; they have not been performed by this increment.

Original contract: [SRC-2B2 R1](https://github.com/qq550723504/task-processor/issues/399#issuecomment-5642868657),
[recovery clarification](https://github.com/qq550723504/task-processor/issues/399#issuecomment-5642876140),
and the [src2b-acquisition-v1 owner contract](https://github.com/qq550723504/task-processor/issues/398#issuecomment-5637525452).

## Directory and package naming

The source directory is `extensions/product-capture`; the private npm package is
`@shuomi/product-capture`. The Tool Market build produces
`shuomi-product-capture.zip` with its matching `release.json`. The trusted build
record and archive belong to the same release; receiver URLs and runtime
configuration still use their existing values.

This source rename does not update an installed extension. Keep an existing
installation's unpacked/loading directory and identity; do not load the new source
path as a second extension or remove browser data. An installation update must be
handled separately after pending handoffs and original-key UNKNOWN recovery have
been resolved. No new manifest key, extension ID or automatic reinstall is added.

## Build and local checks

Requires Node 24 and npm. Install with `npm ci`, then run `npm test`,
`npm run typecheck`, and `npm run lint`. Tests include the built manifest's
permission and destination boundary.

A release build requires an explicit application URL. There is no default
production destination. For example, in PowerShell, replace the example host
with the current application's approved HTTPS origin:

```powershell
$env:CAPTURE_APP_URL = 'https://app.example.com/capture/1688'
npm run build
```

A retained local installation can use its already trusted
`https://localhost:<explicit non-default port>/capture/1688` receiver. Set that
exact URL at build time; `/capture/amazon` is derived from that same origin, never configured separately. Certificate trust remains the user's normal browser
requirement. This uses the real application and release build, with the same
permissions and exact sender origin/port/path checks. It permits no runtime
destination input or private IP receiver. Captured images remain public HTTPS
URLs; the local application exception does not apply to image admission.

Load the resulting `dist` directory as an unpacked extension. The application
at `/capture/1688` and the Amazon application from #905 at `/capture/amazon` must implement the receiving contract below. A build alone
does not establish that the receiving application is available.

Task fixtures use a separate, visibly labelled artifact:

```powershell
$env:CAPTURE_APP_URL = 'http://127.0.0.1:4399/capture/1688'
npm run build -- --fixture
node scripts/port-isolation-check.mjs
node scripts/port-isolation-check.mjs --edge
```

The Windows harness starts an installed Chrome/Edge in its own new profile
under `artifacts`. It uses the official CDP `Extensions.loadUnpacked` testing
method with the test-only extension debugging flag. Automated runs use headless
mode; omit `--cdp` for the manual headed loader. It clicks the actual source-page
button with trusted browser input, rejects a synthetic click, tests original-tab
return, explicitly confirms a new Amazon fixture capture, checks the other-tab
rejection, interrupts the task's workers, and reloads the
receiver with the same recovery key. The flag is never part of the extension
or a user's regular browser configuration. The debugger uses an OS-assigned
dynamic port recorded in this launch's atomically allocated profile; all controls
come from the same launch's Playwright pipe. No fixed-port
target discovery is used. The regression wrapper occupies the old port 4398
with a task-owned decoy and verifies zero queries reach it while capture still
works. Receiver port 4399 must be free; an occupied receiver fails startup
without connecting to or stopping its owner.

The canonical 1688 document response is substituted with `tests/fixtures/product.html`,
and Amazon with `tests/fixtures/amazon-inline-product.html`;
other page network requests are blocked except the task's loopback receiver.
The receiver is explicitly a **transport fixture**, with no account, BFF,
database or publication code. It never proves backend acceptance or real
1688 availability. Browser receipts/screenshots stay under ignored `artifacts`.
Browsers and the loopback server close in `finally`; the task owner can remove
only its closed disposable profiles after preserving the receipts.

## Capture boundary

- Permissions remain `activeTab`, `scripting` and the existing return-location `storage` permission, plus exact HTTPS host grants for `detail.1688.com`, `www.amazon.com`, `www.amazon.ca`, `www.amazon.co.uk`, `www.amazon.de`, and `www.amazon.co.jp`. No subdomain wildcard or `<all_urls>`, cookies, tabs, history, webRequest, clipboard or downloads permissions.
- A static main-frame ISOLATED script mounts only the source-page control on supported product paths. Product extraction follows an explicit capture click/shortcut/popup action and validates the product path. No batch or background traversal is introduced.
- The named shortcut fixes its triggering tab; the popup selects its active tab once. Injection
  targets its main frame in the isolated world. The same document ID and
  canonical source are checked again after extraction.
- Accepted 1688 pages have the exact `detail.1688.com/offer/<ID>.html` shape. IDs
  remain decimal strings, 1–20 digits without leading zero. Tracking query
  and fragment are discarded. The backend independently derives identity.
- For 1688, only the known static `context.result` product structure and the identified
  rendered description node are read. Amazon uses its dedicated selected-product DOM and statically bound image-literal module. No page scripts execute on behalf of
  the extension; no global objects, forms, cookies, storage or other tabs are
  inspected. Unknown structures and challenge/login pages fail explicitly.
- Product fields are projected onto the strict v2 wire: no raw HTML,
  arbitrary metadata, actor, organization, role, Catalog identity or trusted
  asset status. Image URLs are candidates only, never downloaded by capture.
- Missing data stays null/empty with warnings. Amounts and SKU identifiers
  preserve source string precision. Images with query strings, credentials,
  non-HTTPS or private/local literal hosts are rejected.
- The static block and encoded payload are bounded to 2 MiB; a field to
  8 KiB; a collection to 256 items; aggregate fact items to 1024. Extraction
  also has bounded node/depth work and an 8-second deadline.

`tests/fixtures/capture-v2.json` is the new deterministic lower-case wire golden
for the controlled product fixture at a fixed timestamp. `capture-v1.json` remains unchanged as a historical wire oracle. Its content hash
covers the ordered public field projection only: sourceURL, offerID, title,
description, attributes, variants, priceFacts, images. This is a **capture
claim**, not an authenticity signature or the backend's canonical intent
digest. The server owns validation and the sole evidence-to-SourceEnvelope
mapper; SRC-1 and Catalog own durable publication and exact read-back.

## Application handoff contract

An explicit source-page button, shortcut or combined popup action captures and opens the current app,
where the user reviews the summary and confirms saving. The extension freezes one capture and UUID key, creates only its own app tab,
and includes `extensionId`, `handoffId`, `idempotencyKey` in the fragment.
No payload, credentials or identity claims are put in the URL.

The app uses external `chrome.runtime.sendMessage` with strict messages:

```typescript
// Request. All identifiers must match this handoff.
{ version: 1, type: "capture.read", handoffId, idempotencyKey }
// Response.
{ version: 1, type: "capture.payload", handoffId, idempotencyKey, payload }
// Optional status projection from the app after its real backend operation.
{ version: 1, type: "capture.status", handoffId, idempotencyKey,
  outcome: "processing" | "published" | "failed" | "outcome_unknown",
  operationId? }
```

Every message checks the browser-provided exact origin, URL scheme/host/port/
path, top-level frame and own-created tab ID. Other extensions are not
admitted. All fields are required except the optional operation ID; a
published status requires that ID. A late processing message cannot erase
unknown, and a different receipt ID cannot replace the original. Status is
only an application projection, never publication authority.

The app can retry an early `capture.read` at most three times, 250 ms apart,
while `tabs.create` finishes. Unbound tabs fail closed. There are no automatic
write retries. Before any POST, the app must replace the initial fragment
with `#operationKey=<original UUID>`, show the currently verified user and
Effective Organization, and obtain explicit confirmation using the existing
BFF context-drift protections. Login, organization change, revocation and
authorization checks belong to that app/backend owner, not this extension.

The browser ingress is
`/api/v1/workbench/sourcing/1688/browser-captures`, with `/verify`,
`/by-key/:key` and `/:operation_id`. These endpoints and the Web receiver
are owned by Product Sourcing and the application, not this transport directory. Public `{source}`
acquisition is not a substitute for uploading BrowserCapturePayload.

One handoff lives only in worker memory, for at most five minutes. Chrome may
terminate an idle worker much earlier. There is no keepalive or storage;
after loss/expiry, capture reads fail. If a request may already have been
submitted, the app's original key recovery URL remains the recovery entry.
The backend must reauthorize that scope and read the original staging command
through the existing SRC Verify/Read path. The original 1688 ByKey path may resume a previously frozen command under its existing guarded Claim/Finish protocol; it never captures a new page or replaces the command.
A missing receipt does not authorize an automatic new POST/key. The extension
cannot recover a forgotten key after restart; it tells the user to use the
original app page. Beginning another capture requires an explicit new action.

## Release packaging limitation

The release builder includes the new `page.js` asset in `shuomi-product-capture.zip`.
The current Tool Market `LoadPackage` still pins six assets, the older single-platform
name and one receiver; it already rejects the current dual-platform manifest.
That existing download-server admission is outside #924's source-entry scope and
must be resolved before distributing this new ZIP through Tool Market. An unpacked
source build and transport fixture do not prove the download chain works. This PR
does not update the retained package, server configuration or installed extension.

## Verification still required for delivery

Extension unit tests and the browser transport fixture are separate from the
real application chain. Final acceptance requires the actual Web/BFF → browser
validator → existing mapper → SRC-1 → Catalog flow and exact read-back, including
scope changes, revocation, response loss and by-key read-only recovery. A fixture
success cannot satisfy these requirements. Real 1688 login/network acceptance
needs separate authorization; do not use real accounts for this harness.

## Legacy decision

Legacy decision: **EXTRACT**. Reusable behavior is the known 1688
productTitle/gallery/Root SKU/price field interpretation. The current owner
here is the untrusted browser capture adapter. No old services, DTOs, profile,
queues, fallback or handoff are imported or kept alive. Existing legacy
consumers are outside this slice and follow their own retirement owner.
