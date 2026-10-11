# Platform customer support through Chatwoot official APIs

Status: IMPLEMENTATION_READY. Execution Issue: #650. On 2026-10-11 the user selected “官方 API 仅由服务器调用，Chatwoot 管理客服后台，平台提供简洁聊天窗” and explicitly stopped specialized reproduction and additional reviews. This current decision replaces the website-widget proposal and its extra review requirements. It does not accept mixed personal histories or authorize shared/production deployment.

## Outcome and scope

Signed-in users open the existing 联系客服 button, send questions/images, receive human replies and reopen personal history. Chatwoot CE's unchanged official APIs and native staff dashboard own all support facts. A small platform panel and authenticated BFF are the only additions; no app support tables, mirrors, AI, webhook pipeline or new framework.

UI authority: docs/product/final-ui-ia-authority.md, Figma tg48P46SSXl6TBy9lZwg63/page31:463/button427:2546/frame427:2467, plus the user's explicit approval of a simple platform chat panel. Reuse existing Button/Textarea and shell styles. No new navigation page.

## Owners and contracts

ZITADEL/Auth.js owns canonical identity and app access. Chatwoot owns contacts, conversations, messages, staff assignment, attachments and status. No second support store.

- Private server config: CHATWOOT_ORIGIN, CHATWOOT_ACCOUNT_ID, CHATWOOT_INBOX_ID and CHATWOOT_API_TOKEN_FILE (or mutually exclusive server-only CHATWOOT_API_TOKEN). Use a dedicated account/API inbox and integration operator; no Website inbox in that account. Never expose provider customer APIs through platform ingress. Staff dashboard is operator-only. Browser receives no provider credentials/origin/source ID/signed file URL.
- GET /api/support/session: existing serverAuth/current issuer/client/access-token checks; returns only server-derived subject. No provider mutation.
- Subject: platform: plus SHA-256(JSON[configured issuer, canonical tenant ID, canonical user ID]). No browser-selected identity, email/phone or exported business facts.
- Contact lookup: official search followed by exact identifier === subject. Creation supplies that identifier, pseudonymous name, API inbox and server-derived source ID. Never update/merge by email or phone. Concurrent unique-identifier conflict re-reads the exact same contact.
- Conversation creation stores custom_attributes.platform_support_subject = subject. Every list/read/send/download requires configured account/inbox, exact contact identity/contact ID and conversation subject. A client conversation ID alone never proves ownership; contact profile merging does not transfer transcript ownership.
- GET /api/support/conversations lists bounded owned threads without creating facts. GET /api/support/conversations/:id/messages?before=… normalizes public incoming/outgoing messages, excluding private notes and provider metadata.
- POST /api/support/conversations accepts subject, optional owned conversation ID, text and one PNG/JPEG/WebP image up to 5 MiB. Exact same-origin check, bounded body/deadline. Creates contact/thread only when necessary and sends one official incoming message with private=false. No automatic retry after unknown response; retain draft and tell the user to check history before sending again. Empty contact/thread may remain, never reported as a sent message.
- Attachment URLs are authenticated same-origin BFF routes. Verify conversation/message ownership, resolve attachment from that message, then relay bounded bytes only from the configured provider's /rails/active_storage/ path. No arbitrary URL proxy or public signed download URL.
- Closing/route/user/context/organization change clears panel and aborts requests. Focus revalidates before showing cached history; periodic reads reauthenticate and reject changed subject. No widget SDK, iframe, visitor cookie or pubsub credential.

No new durable app state machine/transaction. Contact/source creation uses provider identity constraints. No message mutation replay after unknown response; Chatwoot retains the authoritative result. UI says messages can be answered later and does not invent staff availability/SLA.

## Injection and deployment

Runtime config → root availability prop → existing Workbench SupportLauncher → authenticated BFF → private official Application API → native dashboard. Opt-in overlay connects existing runtime-network and Rails via a dedicated support API network; UI keeps its existing network namespace. DB/Redis remain on provider network. Rails publishes only a loopback staff port; shared environments require separate authorization.

Pin chatwoot/chatwoot:v4.18.0-ce. PostgreSQL/pgvector, Redis and attachments retain named volumes; explicit db:chatwoot_prepare init. Document stop/start and coordinated backups. SMTP, paid channels, AI and public signup remain off.

Legacy decision: N/A. Replace the widget draft completely; no fallback/second path. Commit d4c0c3d4d preserves historical widget behavior and its failed isolation evidence. That earlier review is not evidence for this API delivery.

## Validation and handoff

Existing Vitest covers normal auth/ownership/DTO/message behavior, followed by typecheck/scoped lint and one stable production build. Normal two-user switching, image upload, human reply, reopen, offline message and restart use synthetic local users only. No specialized reproduction, additional reviewer or new acceptance tools. Actual app-ZITADEL-user acceptance remains NOT_RUN until executed; developer checks are not user acceptance.

One writer/worktree/main PR. No implicit merge, Issue closure, shared/production deployment or real-data authority. Handoff includes usable local trial URLs, configuration and material limits.

Official references: [API channel](https://www.chatwoot.com/hc/user-guide/articles/1677839703-how-to-create-an-api-channel-inbox), [create contact](https://developers.chatwoot.com/api-reference/contacts/create-contact), [search contacts](https://developers.chatwoot.com/api-reference/contacts/search-contacts), [contact conversations](https://developers.chatwoot.com/api-reference/contacts/contact-conversations), [create conversation](https://developers.chatwoot.com/api-reference/conversations/create-new-conversation), [send](https://developers.chatwoot.com/api-reference/messages/create-new-message), [read](https://developers.chatwoot.com/api-reference/messages/get-messages).
