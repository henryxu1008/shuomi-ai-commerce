# Platform customer support with Chatwoot CE

Status: BLOCKED. Execution Issue: #650. The integration is a local draft and must not be activated, merged or deployed. On 2026-10-11 isolated synthetic users failed the required separation of personal support histories. This is a current Must failure (identity ownership/access-control bypass), not an accepted risk. Further specialized verification and review were stopped at the user's request. Resume activation only after an approved provider/integration basis satisfies the existing personal-history requirement; do not infer approval to open the feature from ordinary integration approval.

Historical admission: independent read-only architecture review on 2026-10-11 (`/root/support_review`, base `9adb2cec9b49d1b5f187ac1452f92803c2d8364f`, reviewed design SHA-256 `16e91d47eddf24a96aa06c9050fdcb39d574359402e43f3bbc2666ad200b45dc`) reached IMPLEMENTATION_READY before production edits. The later observed Must failure reopens that admission; the earlier conclusion is not final delivery approval.

## Product basis and outcome

On 2026-10-11 the user confirmed the audience: our platform's users asking how to use the platform, reporting errors, and contacting a human. The user accepted the recommendation of self-hosted Chatwoot Community Edition with human support first. Deliver one path: a signed-in user opens the existing **联系客服** button, sends a question or screenshot, receives a human reply, and can reopen their own history. When no agent is available, Chatwoot retains the message for a later reply; no invented availability or reply-time promises.

UI authority: [final-ui-ia-authority.md](../product/final-ui-ia-authority.md), Figma file `tg48P46SSXl6TBy9lZwg63`, page `31:463`, existing support button `427:2546` in workbench frame `427:2467`. These exact nodes were inspected on 2026-10-11. Preserve the current shell/button style; the accepted vendor's native widget supplies the chat UI. No new console page, sidebar item, or custom agent dashboard.

Must:

- Authenticated personal identity from the current ZITADEL/Auth.js owner; no browser-selected user or enterprise authority.
- Human messages, screenshots, offline messages, and history use Chatwoot's native widget/dashboard.
- A user cannot acquire another user's transcript through our support identity endpoint or a forged iframe message.
- No app/OIDC tokens, enterprise grants, current URL/query, order/product facts, or automatic diagnostic dumps are exported to Chatwoot.
- Configuration/authentication/handshake failures stop the connection and present a retryable failure. Unconfigured support remains disabled.
- Durable conversation and attachment storage, isolated startup instructions, and data-preserving restart/backup guidance.

Out of scope: AI responses, RAG, billing/provider-paid calls, store-buyer support, public anonymous marketing chat, enterprise-shared transcripts, automated platform actions, new app tables, webhook/event sync, SSO for agents, custom Chatwoot branding, shared/production deployment, real-user acceptance, and a new support workflow/state-machine owner.

Accepted Risks: no additional accepted risk is asserted by this design. Chatwoot's native visitor bearer token is a provider credential, not an app session; destroying our iframe does not revoke previously captured vendor credentials. Support is a personal conversation, not an enterprise-authorized business API. Production policy requiring immediate provider-token revocation would require a separate product decision and a supported provider mechanism before rollout.

## Owners, contracts and injection

| Fact / capability | Owner |
| --- | --- |
| App login/session, subject and authorization | Existing ZITADEL/Auth.js server session |
| Enterprise membership and business facts | Existing owners, neither read nor modified here |
| Contact, conversation, message, agent assignment, offline state, attachment | Chatwoot CE |
| Config, pseudonymous identity signing and iframe lifecycle | Narrow Next BFF integration; no durable support facts |

Contract → implementation → injection → consumer:

1. Runtime-only `CHATWOOT_ORIGIN`, public `CHATWOOT_WEBSITE_TOKEN`, and private `CHATWOOT_IDENTITY_SECRET_FILE` (or server-only `CHATWOOT_IDENTITY_SECRET`) configure the existing Next BFF. Secret never becomes a `NEXT_PUBLIC_*` value or build argument.
2. `GET /api/support/session`, with existing `serverAuth`, verifies current canonical identity, server access token, session error, session issuer/client matching current auth configuration and existing app authorization, then returns only configured origin, public website token and pseudonymous subject. `Cache-Control: no-store` on every response. GET has no provider side effect. Root request-time configuration availability flows through the existing application/shell props; an unconfigured launcher makes no extra browser requests.
3. The launcher creates a fresh, initially hidden cross-origin `/widget?website_token=…` iframe. Never persist or reuse `cw_conversation` or copy a visitor token into an app cookie/storage. The iframe uses `referrerPolicy=no-referrer` and a restricted sandbox, with scripts/forms/same-origin/downloads only.
4. Receive only `chatwoot-widget:` JSON messages whose `origin` equals the configured origin and `source` equals this exact iframe window. A native `loaded` event supplies a temporary visitor token. Ignore unrelated/duplicate/oversized events.
5. `POST /api/support/session` accepts only a bounded `{subject, widgetAuthToken}` body after exact same-origin/Fetch-Metadata validation. Reauthenticate with the same existing owner, recompute the subject, reject a stale/different subject, then PATCH the configured Chatwoot `/api/v1/widget/contact/set_user?website_token=…` with the visitor token and server-derived `identifier`, `name`, `identifier_hash`. Browser never receives the HMAC. Disable redirects and use a bounded request/response deadline and size. No admin API token or broad provider API CORS is needed.
6. After provider success and a valid response, send native `config-set` and `toggle-open` to the exact provider origin. Chatwoot fetches the identified contact/history using its original iframe token. If the provider replaces the token (a stale/already identified visitor source), fail closed and destroy the frame; reopening always creates a fresh source. Do not put bearer credentials in iframe URLs.
7. Closing, unmounting, route change, user/context change, organization switching and session-validation failure destroy the frame and abort outstanding requests. Validate the current session at least every 30 seconds while open and on window focus; discard a changed subject rather than rebinding an existing visible frame. Retry explicitly opens a fresh source.

The subject is `platform:` plus SHA-256 of a JSON tuple containing configured identity issuer, canonical tenant and canonical user ID. All identity fields come from the server session. SHA-256 avoids sending raw IDs; Chatwoot's identity HMAC is SHA-256 over that exact identifier with the inbox identity secret. The issuer/tenant namespace prevents different identity realms/tenants with the same user ID colliding. Do not include username/email in the merge identity. Send a stable pseudonymous display name rather than PII; users may voluntarily type context in the conversation.

## Provider protocol and security rationale

Pin `chatwoot/chatwoot:v4.18.0-ce` (tag existence verified with Docker manifest on 2026-10-11). Use the mature provider unchanged. Recheck this bridge against upstream when upgrading.

Primary implementation references at that tag:

- [Widget controller](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/controllers/widgets_controller.rb): a fresh iframe creates an anonymous contact/source/token; allowed domains enforce `frame-ancestors`.
- [Native widget](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/javascript/widget/App.vue): `loaded`, `config-set`, `toggle-open`; config-set fetches current contact and history.
- [Widget contact API](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/javascript/widget/api/contacts.js) and [controller](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/controllers/api/v1/widget/contacts_controller.rb): PATCH contact/set_user with identifier and HMAC. A fresh source remains the same token after identification; an already identified different contact creates a new token.
- [Contact identify](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/actions/contact_identify_action.rb) and [verified conversation scope](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/controllers/api/v1/widget/base_controller.rb): merges the same canonical identifier and scopes histories to HMAC-verified sources in this inbox.
- [Upstream parent SDK](https://github.com/chatwoot/chatwoot/blob/v4.18.0/app/javascript/sdk/IFrameHelper.js): runs remotely supplied code in the app origin, accepts messages without origin/source checks, uses wildcard target origin and copies bearer tokens into parent cookies. This integration reuses the native UI/API while replacing only that small parent bridge.

The proposed setup requires **Enforce identity validation** on the dedicated website inbox and allowed domains restricted to the app origin. These settings were enabled in the isolated fixture but did not establish the required personal-history separation. Do not treat the narrow BFF's checks as proof of the provider's complete identity boundary. Our frame remains inaccessible until successful server identification, but this alone is insufficient to approve activation.

Threats addressed: client subject substitution, identity namespace collision, forged iframe events, app-token disclosure, stale identity/enterprise context in a reused widget, cross-site identity mutation, arbitrary upstream destinations, secret exposure, hanging/oversized provider calls. A compromised Chatwoot process cannot read the app DOM or app cookies; it owns its own origin and the voluntary support data. Existing app XSS/session security remains with existing owners. No new IAM, admission framework, outbox or recovery worker is needed.

## State, persistence and failure behavior

Only ephemeral launcher states: disabled / closed / connecting / open / failed. These are UI lifecycle states, not a durable domain state machine. No message submission is proxied or retried by our BFF; Chatwoot owns message persistence/delivery.

Identity PATCH has a repeatable effect for the same source/identifier. A lost response can leave an identified empty provider source; destroy the frame and let a user retry with a fresh source that merges into the same contact. No automatic repeated external mutation after an unknown response. Closing/cancellation may leave an empty source but never synthesizes a message or reports one sent. A separate verified source per open also avoids accidentally carrying another logged-in user's visitor token.

The isolated Compose stack adapts [upstream Docker deployment](https://developers.chatwoot.com/self-hosted/deployment/docker): CE Rails + Sidekiq, Postgres/pgvector, Redis and an attachment volume. Postgres and Redis have no published ports; Rails publishes a loopback port only. Init is an explicit one-shot `rails db:chatwoot_prepare`. Do not enable SMTP, paid channels or AI by default. Registration is used only for the initial isolated administrator then disabled. Production needs its separately authorized HTTPS reverse proxy, domain/TLS, agent accounts and backup destination. No local CA trust changes or HTTPS bypass.

An opt-in app Compose overlay reads the existing per-inbox secret from a private host file into the BFF only. It must not alter other service settings/volumes. Existing/shared trial stacks are untouched. `stop` / `start` retains named volumes; docs include DB and attachments backup/restore and never advise `down -v` for retained state.

Legacy decision: N/A. Reuse the approved current console/auth owner despite the historical `listingkit-ui` directory name. No retired backend service, task, compatibility adapter or second fact source.

## Validation and handoff

Before implementation: independent read-only architecture review, findings classified per AGENTS; execution Issue records `IMPLEMENTATION_READY`.

TDD: endpoint tests for missing/expired/unauthorized identity, stale subject, exact origin, malformed/oversized bodies, configuration and secret failures, identity HMAC/namespace, bounded provider errors and token replacement; launcher tests for origin AND source, hidden-before-identification, failure/timeout, close/unmount/context cleanup and revalidation. Use existing Vitest, then typecheck and scoped lint. No unrelated test platform.

Isolated integration: boot the pinned CE image with synthetic credentials/users only; verify the real native handshake, HMAC-verified own history, another subject's separation, human reply, screenshot attachment and offline message. Actual app-ZITADEL-user browser trial remains NOT_RUN unless independently executed; fixtures, CI or provider health are not user acceptance.

Final independent review checks the actual diff and available runtime evidence once the full path exists. One principal PR with Issue reference, usable startup/configuration instructions, precise PASS/FAIL/NOT_RUN and known limits. No implicit merge, shared/production deployment, Issue closure or actual-user acceptance authority.

Current evidence: relevant endpoint/launcher tests (38 tests), typecheck, scoped lint and Next production build passed; isolated native question, image attachment, human reply and reopen passed. Personal-history separation failed. Offline behavior, data-preserving restart and actual app-ZITADEL-user acceptance are not claimed as passed. The isolated runtime is stopped with named volumes retained. No formal delivery approval or production activation exists.
