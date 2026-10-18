# 百炼确定未生成与四笔预留对账增量

Execution: [#774](https://github.com/qq550723504/task-processor/issues/774).
Design Basis: Independent Architecture. Admission: REVIEW_PENDING. Original image owner only; reuse existing independent reviewer; no production edits before IMPLEMENTATION_READY.

## Product outcome and authority

2026-10-18 human instruction: 查清百炼明确403为何UNKNOWN，逐笔核实当前4笔UNKNOWN并按实际结果处理，保留已有数据/配置，不重复派工。
结果：确定未生成的请求不长期占用成员点数/企业余额/存储；真正不确定的请求仍保留原UNKNOWN，不重放模型。此授权只覆盖四笔原运行的对账，不授权新付费生成、合并或服务部署。
沿用 product-image-agent-v1.md、bailian-product-image-wiring-v1.md、enterprise-storage-capacity-v1.md 的 GenerationFact / Image V3 / OrgResource / Storage owner。当前Figma tg48P46SSXl6TBy9lZwg63 /31:463/6861:1826、6861:1864；无新页面或交互，仅沿原任务失败/恢复展示。
Must：精确企业/actor/member/run/revision/slot/attempt/intent/fingerprint；无成功证据冲突；确定拒绝后0点消费、恰一次退回原预留；保留全部旧请求、诊断、图片、成功消费、配置和月限。UNKNOWN不能靠空审计/无结果/异常文案释放。
Out of scope：通用对账平台、provider重试/生成/查询请求、账务直接SQL、余额补发/退款、删除或重写历史、凭据/计费开关/模板变化、自动迁移、增加验收runner。无新增Accepted Risk。

## Actual defect and four cases

Bailian client all non-2xx -> opaque unknown; Diagnostic explicitly lacks economic authority. GenerationExecution after BeginGenerationDispatch treats every absent Success as unknown, including local capability preflight rejection; GenerationRecovery cannot terminalize unknown. Dispatch-start marker means permission to attempt, not proof that HTTP/model accepted.
- 02fd4737-f227-5f3c-a6d8-657686d3141c / main-identity /1: original qwen-image-3.0, original member394513850164051970, 1 point. Historical report records original SourceEdit prompt canonical check rejected before HTTP. Must reproduce exact frozen input and original validator before any release.
- 96a22650-2526-5b83-831c-eb56a32730c6 / carousel-usage_scene /1: same local rejection condition, independently reproduce.
- 09ffffa4-0c8e-5160-9ffa-a1370a799274 / custom-cecfeded-f6b0-4d51-bb1e-8c8e23b8619a /1: original qwen-image-3.0, HTTP403 AllocationQuota.FreeTierOnly request e78918de-1b27-9652-994f-3270179d5da6, retained timestamp10:24:59Z.
- c235f484-3ecc-541e-88a5-68d166f3c9f6 / same custom slot /1: qwen-image-3.0-pro, member395658276227776514, HTTP403 AllocationQuota.FreeTierOnly request070f62f5-244a-9703-85c5-a5f0c0d039e0, retained timestamp12:46:30Z.
All four currently generation=unknown/effect=provider_outcome_unknown, no success/settlement. Facts and original reservation IDs frozen in local BEFORE-FOUR-UNKNOWN.jsonl. Evidence is not sufficient until each exact lineage is checked.

## Contract -> implementation -> injection -> consumer

1. Add narrow typed no-generation evidence; no inference from generic ErrInputInvalid or 403 alone. Provider evidence admits only Beijing bailian/dashscope-image-sync-v1, 403 + exact AllocationQuota.FreeTierOnly, complete bounded strict JSON with valid consistent request ID and no output/task/success fields. Official https://help.aliyun.com/zh/model-studio/rate-limit explicitly says free tier controls whether requests are accepted. Other HTTP failures, malformed/truncated/conflicting bodies, timeout/disconnect remain unknown. SourceEdit local proof originates only from validation before backend invocation; downstream generic validation errors do not qualify.
2. Raw provider/capability evidence crosses existing worker factory/tools seam into a bound Generation rejection proof. Bind original fingerprint and provider/model/protocol/route/config. Never put prompts, URLs, keys, raw bodies or caller-selected points into proof. Current caller cannot submit proof through HTTP.
3. GenerationFact remains canonical economic owner. Extend its existing JSON with optional bounded immutable NoGeneration evidence; no new table and no parallel fact store. New guarded operation records dispatch_started/unknown -> existing no_generation only with checked evidence and no Success/Settlement. Exact replay returns same fact; conflicting evidence, success, scope, identity, or settlement rejects. Existing successful/no_generation proof digest stays byte-identical when new evidence absent; evidence-bearing new terminal digest includes evidence.
4. Persist fact before calling existing OrgResource FinalizeImageGeneration. Reuse original image-finalize:intent operation and receipt, original member/month/price/quantity, existing bucket + member-month locked transaction; no balance grant, no direct SQL. Bind original receipt back to fact. Existing Storage close after confirmed no_generation releases only that task capacity; no object deletion. Crash at each step resumes through original GenerationRecovery; no provider/credential requirement or new reserve.
5. Image V3/slot/run projections must close consistently as known no-generation/released via original repository/policy/CAS/event owner after verifying terminal fact and receipt. Reuse provider_not_dispatched failure semantics (which already accepts rejected-before-effect), not publication/success. Historical UNKNOWN -> known no-generation transition requires terminal proof from GenerationFact. Preserve old diagnostic history. Original same attempt must never POST again, even after its budget is released; existing-fact fast path returns terminal no-effect rather than reclassifying it unknown.
6. Future execution: typed proof is recorded immediately; absent proof remains unchanged UNKNOWN. Success observation wins over a conflicting later rejection/error; output download/validation failures after success keep charged original recovery.
7. Historical application: a small owner-scoped operator command, dry-run by default, accepts a bounded approved evidence manifest for at most these four identities and expected row/fact fingerprints. It has only the existing ImageFact/OrgResource/Storage/repository ports; no provider, downloader, schema, initializer, credential resolver, reserve or price writer. Before apply, verify frozen original code/input local rejection or exact preserved provider response, closed original slot execution/no competing dispatch, current facts/no-success/no-artifacts/no-settlement, and original reservations. A textual operator note or log hash by itself is insufficient. Command is necessary to process this incident through owners, not a new runner/framework.
8. Runtime operation requires peer to release current window and fresh private backups. Candidate command may be prepared/reviewed and used only for the human-authorized four-record operation; this does not authorize merging/deploying replacement services. Private DB credentials are existing operator credentials, never printed/copied into repository.

## State, transaction, recovery, boundaries

Each identity remains original owner. Image row lock + run/version CAS serializes proof; Commercial operation lock serializes reserve/finalize and closes late reserve; immutable receipt drives binding; Storage release repeats idempotently. No cross-DB atomic fiction: durable terminal fact -> economic receipt -> fact binding -> capacity close -> effect/projection closure, each resumable with same proof/key and no POST.
Unknown without proof stays unknown/reserved. Simultaneous success/proof allows one consistent terminal outcome; never overwrite success to refund. Fact/proof conflict fails closed. Repeated reconcile cannot refund twice, change price/member/month, create calls or consume another reservation. A running original slot blocks manual apply.
Historical four operations retain local before/after backups and exact proof receipts; all other success rows, original data/credentials/config/Skill versions and grants must be unchanged. Accepted requests cannot be inferred rejected from an empty dashboard.

## Legacy decision

EXTRACT existing single-source GenerationFact, OrgResource/Storage settlement and V3 policy/repository behavior under current owners. RETIRE the blanket classification for proved refusal only. No legacy compatibility adapter, old task owner, migration, dual write or second billing source. Checking frozen historical inputs is incident evidence, not restoring retired runtime.

## Validation and delivery

TDD: strict provider403/no-effect acceptance vs unknown bodies/network/errors; local preflight0backend calls vs post-call validation; bound proof/identity/conflicting success; owner persistence + resource settlement failure/replay; original success digest unchanged; actual PG point/member/capacity release exactly once; original-attempt recovery0POST/0reserve.
Use existing fixture/PG/Temporal tests, no new validation framework. Final independent check covers actual diff and four-record evidence. Scope does not require broad full-CI repetition during development.
Delivery is one primary fix PR from latestmain, no self-merge/deploy; exact four-record operator result (proved released / still unknown with specific missing evidence), retained snapshots and normal task readback. User acceptance remains pending.

