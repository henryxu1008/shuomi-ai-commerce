# Product capture entry and original-tab return

Execution: [#920](https://github.com/qq550723504/shuomi-ai-commerce/issues/920). Design Basis: **Reuse Existing Architecture**. Status: **DESIGN_REVIEW_PENDING**; no production implementation before the requested review.

## Product basis and scope

The user approved on 2026-10-24: a direct keyboard entry, combined capture/handoff, and return to the original application tab; design review first, then one implementation PR. Merge, installation/runtime update, real capture/save, deployment and server collection are not authorized. Existing data, configuration, extension ID/loading directory, original requests, reservations and audit remain untouched.

Authorities: [current ingestion](plugin-capture-ingestion-v1.md), [source quote read](plugin-capture-source-quotes-v1.md), [listing fields](../product/plugin-capture-listing-fields-v1.md), [final UI/IA](../product/final-ui-ia-authority.md), Figma `tg48P46SSXl6TBy9lZwg63 / 31:463`, and the user's current entry decision. Application routes and navigation stay unchanged. The existing Amazon/1688 receiver and confirmation/save/recovery pages are reused in full.

User path: supported product page → named browser shortcut (or one popup primary button) → capture and immediately hand off → existing application account/enterprise confirmation → existing save → original receipt. The popup remains a secondary way to use the same path. Starting another capture retains the existing explicit new-capture confirmation.

No persistent source-page button, side panel, application-driven capture command, broader host grant, new dependency or new storage is introduced. This is a local extension entry change, not a solution to restrictions of an automation tool or a blocked source website.

## Current behavior and owner

`background.ts` accepts only exact own `popup.html` messages and has separate capture/handoff commands. `Controller` coalesces capture and delivery individually. `Handoff` owns one ephemeral payload/key/nonce and one own-created application tab. It validates the exact external sender, frame, receiver origin/path, key and nonce. The worker can be terminated before the five-minute maximum.

The application owns consent, scope, backend submission and result. Amazon's original-tab session marker binds actor/organization/member/source/key/digest; opening a replacement recovery tab does not establish that marker. SRC, Catalog and Collection remain the durable owners. Extension status is only a projection of the application result.

Legacy decision: **N/A**. These are current owners. Retire the split internal `popup.capture`/`popup.handoff` dispatch after the combined entry is connected; do not add a compatibility route. Historical payload/wire and backend original-key recovery are unchanged.

## Entry and reuse

Add a named `capture-current-product` command, browser-scoped, with suggested `Alt+Shift+P`. It is not `_execute_action`, which can merely open the popup. Read the actual command assignment with `commands.getAll()` for accurate popup help; an empty assignment is shown as unassigned, not as a working shortcut. Do not overwrite user assignments or use a global shortcut.

The command handler captures the event's tab identity; a missing/invalid event tab fails explicitly. The popup resolves its current active tab once. Extraction uses that fixed tab ID, the existing platform/source validator, isolated main-frame extractor, and same-document/source recheck. It never selects a different active tab after an await. No other tab is inspected to obtain source data.

Both entries call one combined Controller operation that coalesces the complete capture → handoff operation. It calls the existing `capture()` and `handoff()` implementations, retaining one payload, UUID pair and own receiver tab. The new UI has one primary action, `采集并在应用确认`; the summary/missing facts move through the existing receiver without an additional popup confirmation. This operation never POSTs or confirms saving.

Internal popup messages remain strict, own-extension and exact-popup only. The named browser command is a trusted extension event, not a new externally callable message. The existing external `capture.read`/`capture.status` whitelist and all sender validation stay byte-for-byte in meaning. Manifest permissions stay exactly `activeTab`/`scripting`, with the same six exact source hosts, fixed sibling receiver URLs, CSP, name, package and identity policy.

## Repetition, return and failure

| Condition | Entry behavior | Identity / effect |
| --- | --- | --- |
| Empty | Capture fixed source tab, immediately hand off | One new capture and UUID pair; no save |
| Capture/delivery in progress | Join the same in-flight operation | No second extraction/tab/key |
| Already handed off, including unknown/terminal projection | Return to the own-created original application tab | No extraction, new key, rebind, save or retry |
| Explicit new-capture confirmation | Reset only when no entry is in flight | Existing new-capture semantics; old application tab/request remains |
| Original tab cannot be activated / creation previously failed | Clear failure guidance, retain original key and state | No replacement tab, no re-capture, no automatic POST |
| Worker expired/restarted | Existing ephemeral limitations; guide to original app page | No invented recovered identity or automatic startup retry |

Returning activates only the stored own-created tab ID (`tabs.update(id, {active:true})`) and its window (`windows.update(windowId, {focused:true})`). It never supplies a URL, reloads, copies session markers, changes tab ownership or rebinds Handoff. If the tab has navigated elsewhere, focusing it is navigation assistance only, not proof of recovery/success; the original application remains the result authority. No arbitrary tab ID is accepted from a popup/webpage. This consumes the existing tab handle, not a new recovery protocol or persistent owner.

The recovery control becomes a button using this same return operation, replacing the `target=_blank` recovery link. Application links/status and original `#operationKey` are not changed. A closed original tab or worker loss is not repaired through new storage or a widened backend endpoint.

Reset must reject while the combined entry, extraction, delivery or return is busy; it must not replace a Controller whose asynchronous work is completing. The existing explicit confirmation remains available after the current action settles. Late external messages for an older key cannot mutate the new flow under the unchanged Handoff checks.

Failures use a small allowlisted set of user messages for unsupported/missing source tab, extraction/page-change failure, unavailable handoff, expired capture, busy action and unavailable original tab. No raw page/server exception is displayed. A shortcut failure leaves an action badge/title directing the user to the existing popup for the precise guidance; this adds no notification permission. It never says that a failed handoff means a failed save. The popup must distinguish transport loss from backend outcome and continue directing a possibly submitted operation to its original application page.

## Boundaries and validation

No new backend/HTTP/wire/domain contract, schema, persistence, transaction, tenant/member authorization, billing/provider call, server worker, retry/UNKNOWN owner, storage or host permission. The original document validation, bounded payloads, single own-created receiver tab, five-minute maximum and strict external sender constraints remain. Existing source data is untrusted; only the application authorizes/publishes.

TDD uses the existing Vitest/browser API mocks and fixture assets. Necessary checks: both platform receiver paths; named-command/one-button entry; fixed triggering tab despite active-tab drift; concurrent popup/command coalescing; repeated entry returns without new key/extraction/tab; original-tab/window activation uses no URL; closed tab/window failure preserves key and does not replace/retry; busy reset rejects; explicit settled reset and stale-message rejection; correct shortcut availability; unchanged manifest grants and external messaging. Adapt existing smoke selectors to the one-button/recovery-button flow without creating a new runner. Run relevant extension tests, type/lint/build and final PR CI; no real product navigation/recapture, retained browser profile, runtime or business DB is used.

Official APIs: [commands](https://developer.chrome.com/docs/extensions/reference/api/commands), [activeTab gestures](https://developer.chrome.com/docs/extensions/develop/concepts/activeTab), [tabs](https://developer.chrome.com/docs/extensions/reference/api/tabs), [windows](https://developer.chrome.com/docs/extensions/reference/api/windows), [Edge API support](https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/api-support). Reuse native APIs; no framework/library is needed for the bounded entry adapter.
