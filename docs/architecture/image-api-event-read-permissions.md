# Image API event-read permission correction

Design Basis: Independent Architecture. Admission: IMPLEMENTATION_READY.

The original independent visual_brief_arch_review approved design commit
47e2a3155a757a992bbaaaba7ce6deb4c46ef6fc on 2026-10-19, before production
implementation. No BLOCKER; the bounded PostgreSQL cases below are
IMPLEMENTATION_TEST, not new architecture. No merge or runtime authority implied.

## Product outcome and authority

Issue #774: an operator can confirm an existing image plan through the current API
database role, without losing the original confirmation, configuration admission
receipt, audit, exact skill/template versions, or retained data/configuration.

The human decision of 2026-10-19 authorizes the minimum API event-read correction,
the formal grant inventory, startup verification, and real PostgreSQL permission
tests; Figma and Issue must precede production implementation. Later real trials
retain the existing total cap of 3 calls / 3 AI points / CNY1 and stop on UNKNOWN,
denial, abnormal output, insufficient balance, or over-budget. This development
decision is not merge, retained-instance installation, or deployment authority.

Figma Product Authority: [confirmation preservation, dark/light](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7748-2242),
derived from the existing confirmation/UNKNOWN controls. This patch changes no
UI copy or interaction: retain the request, verify it, and continue only the
original command under the existing authority. The earlier mixed error-state
draft 7724:2238 remains historical and does not authorize new UNKNOWN handling.

References: [current image architecture](product-image-agent-v1.md),
[shared event-cursor amendment](bailian-image-no-generation-reconciliation-v1.md#shared-event-cursor-blocker-amendment-2026-10-18-implementation_ready),
and [greenfield product baseline](../product/greenfield-no-legacy-migration.md).

## Evidence and smallest boundary

The retained run 0fd17ffe-280f-53c6-8f26-825eac9c0b7f accepted the original prepare
key and parameters. Its single confirm b785bac4-d2b4-46df-bec1-cd5b0fce6dd8
returned UNKNOWN. PostgreSQL recorded `permission denied for table
image_agent_v2_events` while allocating the next event cursor. Configuration
admission 739b9f32-f306-4745-aa6a-257a467b0b29 committed; the ImageDB projection
transaction rolled back. No model dispatch or AI point reservation occurred.

The existing allocator legitimately reads the shared event log after the approved
cursor correction. The API grant inventory still supplies only INSERT, and its
startup allowlist rejects SELECT. The current positive PostgreSQL test already
calls this confirmation path, but skips without its opt-in DSN. Fix the grant
and startup contract together and make that existing test execute in CI with
the repository's existing testcontainers/PostgreSQL dependency when no isolated
DSN is supplied. No new runner, workflow, dependency, or acceptance platform.

Scope: add SELECT to this one existing API table grant and required/allowed
startup privileges; strengthen the existing narrow-role PostgreSQL test.
The allocator and business service remain unchanged. Column-specific SELECT
would require changing the existing SELECT-all allocator and a separate column
read policy; this patch keeps its existing owner and bounded scoped query.

Out of scope: new roles/tables/schema, role memberships or owner credentials,
automatic permission repair, new event APIs, authorization/RLS redesign,
state/transaction/UNKNOWN/retry/cursor/settlement changes, schema/data migration,
manual receipt edits, quota/billing/provider changes, or real-instance writes.

## Contract, implementation, injection, consumer

- ImageDB owns runs, projection snapshots/commits and the shared immutable event
  log. ProductAgent's existing configuration owner owns immutable admission.
- `OrganizationRuntimeRole` remains the pre-provisioned `image_agent_runtime`.
  Only `image_agent_v2_events` changes from INSERT to SELECT,INSERT. Its UPDATE,
  DELETE, TRUNCATE, REFERENCES and TRIGGER remain denied, as do unrelated tables,
  schema creation, role membership and superuser privileges.
- `GrantOrganizationRuntimePermissions` and the existing explicit maintenance
  command consume that inventory. Startup `VerifyOrganizationRuntimePermissions`
  requires and permits the same SELECT; missing required or extra forbidden
  grants still fail closed. Startup performs no GRANT or DDL.
- The existing current-app API repository consumes this role. Confirm/Cancel
  and other projection commits keep the existing run lock, scoped event cursor,
  projection CAS, commit fingerprint and immutable receipt transaction.
- Worker privileges, credential readers, model/price configuration and ordinary
  member product permissions remain unchanged. Database grants do not replace
  the service's organization/actor/member/run authorization.

## Persistence, failure, idempotency and recovery

No new persistence boundary or compensation owner. Configuration admission may
commit before ImageDB confirmation; this is the current approved recovery path.
Read `generationAdmitted=false` or no commit cannot erase that receipt or prove
the whole command had no effect. An original confirmation replay uses the same
action, plan/quote digests, configuration receipt and immutable deadline. A
different action/payload remains rejected. No deadline or budget extension.

Permission failure still rolls back the existing ImageDB transaction and returns
UNKNOWN, retaining the request. This change does not reinterpret failures,
automatically release points, unlock requests, retry, start workflows, or call a
provider. Existing run lock/CAS and service authorization handle concurrency and
revocation. The current query's scope, LIMIT, request deadline and pool bound
(at most eight connections) remain unchanged.

Real installation must later use a reviewed exact build, the existing explicit
grant command and startup verifier under separate operational authority. Do not
grant SELECT to the old serving binary: its old startup allowlist forbids it.
Keep original requests, admission/deadline, audit and backups. No rollback may
rewrite receipts or data; any later rollback of code/privileges is separately
coordinated, not an automatic data repair.

## Must verification and delivery

1. RED: existing real PostgreSQL narrow-role confirmation reproduces the denied
   event read under the original grant inventory; no provider is involved.
2. GREEN: the same API role initializes, reads and commits the original confirmed
   projection; exact replay has one commit/event, and a scoped cross-tenant read
   remains rejected. Cover an intervening evidence cursor using existing owner
   behavior, preserving monotonic event and projection identity.
3. Startup rejects missing event SELECT and event mutation overgrants; restoring
   precisely the declared grant restores admission. Actual SQL event UPDATE,
   DELETE and TRUNCATE remain forbidden, alongside existing denied operations.
4. Reuse existing isolated PostgreSQL tests and required CI. No shared instance
   is used for development tests, and skipped tests are not reported as PASS.
5. Original independent Reviewer checks this bounded privilege change before
   production edits and the final actual diff/test evidence. Implementation and
   CI are not user acceptance; retained-instance continuation remains NOT_RUN.

Legacy decision: N/A. Only current ImageAgent owners and their approved shared
event cursor are used; no retired abstraction or compatibility path is added.
