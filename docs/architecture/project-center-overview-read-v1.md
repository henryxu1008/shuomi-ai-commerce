# Project Center overview read V1

Status: IMPLEMENTATION_READY (readonly independent review, 2026-10-11).
Design Basis: Independent Architecture, readonly contract supplement to
`ai-workbench-project-center-v1.md` (IMPLEMENTATION_READY). Execution #619,
original owner #624 / merged #626, follow-up to the now merged delivery PR #631.

## Outcome and authority

The current member can open Project Center, see real private-project overview
figures, filter by business scope / authorized associated store / project type,
and open the original project. User decision on 2026-10-11: supplement the real
summary/filter read design and completely integrate, as for preferred applications.
Figma Product Authority: `docs/product/final-ui-ia-authority.md`, file
`tg48P46SSXl6TBy9lZwg63`, visible page `31:463`; active dark `667:359`
(content `667:458`), light `667:630` (content `667:729`). Read live before design.
Recent `579:3445` and archived `579:4069` were also read live; neither has four
overview figures. Reuse existing recent/archived/templates entries and commands. Example numbers,
overall percentages, reports and suggested next steps are not product facts.

Four figures describe **all active private projects in the current organization**,
independent of pagination/search/filter. Active count; number of distinct linked
BusinessTasks in WAITING_CONFIRMATION; projects whose own updated_at is today;
store-associated / general project counts. Waiting is explicitly labelled linked
tasks. Today uses the browser IANA timezone and server time; visit and task activity
do not update project metadata. Existing task ratio remains labelled completed
linked tasks / total linked tasks. No guessed project progress.

Out of scope: project sharing, new task states or owners, AI execution, providers,
payments, templates redesign, schema/role changes, CRUD changes, automatic resource
association, task cancellation, production deployment or business-data writes.

## Contract, owners and flow

New bodyless GET `/api/v1/workbench/projects/overview?mode=active&timeZone=Asia%2FSingapore`
accepts only one each of mode (active/recent/archived) and timeZone (valid IANA,
max 128 bytes). Browser BFF `/api/workbench/projects/overview` validates the same
strict bounded contract. Missing timezone uses UTC and is identified in response.

Response: activeProjects, todayUpdatedProjects, storeProjects, generalProjects,
timeZone, today (YYYY-MM-DD), waitingTasks (number or null), stores (authorized
id/name pairs, max 100), storesAvailable (boolean). Null waiting never means zero.
Store options cover the selected list mode, without search/type/scope narrowing;
recent options only include previously visited active projects. List already
supports storeId: reuse it. Fixed sorting remains updated_at DESC / stable ID;
recent remains last_visited_at DESC / stable ID. UI states these existing orders.

Project Center Repository reads only its own projects/references/visits. A readonly
repeatable-read transaction obtains counts, distinct active linked task IDs (limit
201) and distinct store IDs for selected mode (limit 101), then ends before any
resource I/O. All joins/predicates require organization + actor. Snapshot carries
its Scope; Service rejects mismatched or malformed results. No cross-owner SQL.

Service -> existing ReferenceReader -> original source permission/member check ->
original taskView / Store reader. Canonical task projection remains authoritative;
raw persisted task state is not used as a substitute. Only distinct task IDs are
resolved, so one task linked to multiple projects is counted once. Up to 200 task
reads in a separate two-second context: incomplete, denied, missing, timed out or
over-budget projections produce waitingTasks=null. Store projections have their
own two-second context and max 100 reads. Unavailable stores disclose no ID/name;
storesAvailable=false signals incomplete options rather than an empty complete
catalog. No transcripts, knowledge contents, results or task IDs in this response.

Recheck the original Project authorization before returning the response. Local
aggregate counts remain usable when reference sources are unavailable. A
project authorization failure or local snapshot failure fails the entire request.
HTTP timeout remains 10s, response limit 256 KiB, Cache-Control private,no-store.

Injection remains the existing Project Center Service and source reader. New route
uses CurrentIdentity, LiveWrite organization binding and workbench.project.read.
It grants no resource permissions and creates no new auth/session/cache authority.

## Invariants and UI

Private creator + current organization scope, original source permission checks,
reference redaction and current identity capability binding are unchanged. Read
transaction does not alter revisions, receipts, audit or visits. GET has no writes.
Existing CRUD optimistic revision/idempotency/UNKNOWN recovery remains unchanged.
There are no new facts, states, compensation, retries or external side effects.
Legacy decision: RETIRE; no legacy owner, adapter, fallback or migration is added.

Reuse Console shell/tokens, native buttons/inputs and CSS module. Active overview
matches the Figma header, four metrics, business filter panel and two-column cards,
with responsive stacking and light/dark tokens. Search and reset update the existing
list query; store selection is enabled only for associated-store scope. Store ID
clears when leaving that scope. Filters reset pagination. Keep other menu entries
working and preserve detail/create/template commands. No duplicate root title/tabbar.

Overview loads separately from list; failure clears prior figures and offers reload.
Aborted scope or session-epoch changes cannot install old figures/store options.
Zero is shown only for successful complete facts, unavailable is explicit. No
fixture/sample data in the normal runtime. Browser trial remains readonly, opening
new-project dialog then cancelling is permitted; do not open project detail because
its existing explicit visit command writes metadata.

## Verification and delivery

TDD: complete/denied/budgeted task projections, invalid timezone/DST day bounds,
scope mismatch; repository exact counts beyond page one, private tenant isolation,
deduplication, inactive-reference/archive/recent exclusion and runtime-role reads.
HTTP/BFF strict query/body/no-cache admission. UI real metrics, store query/reset,
error clearing, late aborted response isolation, existing CRUD recovery regression.
Run only impacted Go tests, database integration, frontend tests/types/build.

One readonly architecture review before implementation; one independent final
diff/path review after it. Restore retained local runtime with merged #631, this follow-up
and official knowledge PR #633 exact sources; record distinct composition SHA and
image IDs. Recreate only backend/UI, preserve every named volume and other module.
Give the user the normal HTTPS entry and honest limitations. PR remains In Review;
developer checks and AI review are not user acceptance or merge authorization.

Review: `/root/native_auth_review`, baseline `913565ef5df7ab33b07e3528a7f03b22a803d707`,
proposal SHA256 `445D82020B563DFFC0006DAB785169595D7F14F96D655D27BD1B4101FBC161B9`.
No BLOCKER; IMPLEMENTATION_READY. R1 IMPLEMENTATION_TEST: exact private joins,
dedup/archive/recent and IANA/DST dates. R2 IMPLEMENTATION_TEST: honest budget/source
unavailability, response-entry authorization recheck and late-scope response isolation.
These checks remain required for this increment; review is not product acceptance.
