> #301 / #980 用户确认的[完整退休闭包](../../architecture/legacy-ai-routing-image-policy-closure-retirement-boundary.md)已在同一PR实施：本文涉及旧AI PolicyRouter/Legacy-Shadow/Planner、闲置通用HTTP图片生成adapter、旧本地模板/排版/白底或IP启发式的KEEP/开工要求，仅在该设计§5/§6.1明列范围被替代。有效账本值、图片字节检查、SceneOptions/metadata原规则及通用TestCategoryOf断言明确保留；当前AI/图像审核/JSON/hash/计费/UNKNOWN、采集和独立SHEIN登录/Store同步不变。以下实际起始main完整原Git blob、批准/失败/历史/数字/勾选逐字保留；无compat alias/新运行装配、workflow/完整inventory/既有资源操作或试用前置，#301 OPEN。

> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](../../architecture/legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。仅替代旧 shared/aiidentity ExecutionEnvelope/PersistedExecutionEnvelope、capture/restore/context/分类/来源白名单、Task tenant 辅助和两专属测试的保留/实施含义；五工件闭包已在同一 PR 源码退休。当前 Identity/WithIdentity/FromContext/request roles、authidentity 与现役 ImageAgent/Commercetool 授权/隔离/恢复完整保留；不复建旧无 scope 放行或旧信封，不执行历史 plan 的派工/迁移/运行步骤。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

# ProductEnrich Governed Execution Integrity Design

## Status

Approved direction for resolving the P1/P2 review findings on PR #177.

## Problem

PR #177 introduces provider-neutral routing, invocation records, and durable AI
execution identity, but ProductEnrich still composes those facilities around
legacy execution paths instead of making them one execution boundary. Four
observable inconsistencies follow:

- a tenant outside the active rollout is represented as `policy_denied`, then
  invoked through an unrecorded legacy side path;
- score caches are read before route selection and invocation recording;
- repository tenant scoping differs between GORM and in-memory implementations;
- persisted envelope presence is reimplemented by callers and does not inspect
  every persisted field.

The remaining review findings expose two local omissions: image invocations do
not receive operation-specific prompt metadata, and the product schema migration
Job has no Kubernetes-side deadline.

## Goals

1. Make rollout selection, route selection, provider invocation, cache identity,
   and invocation recording one coherent execution lifecycle.
2. Preserve the legacy named-client-then-default fallback for tenants outside
   the active rollout without reporting an active policy failure.
3. Ensure governed score caches cannot cross tenant, route, model,
   configuration, prompt, or base-score boundaries.
4. Make tenant isolation a repository contract shared by persistent and
   in-memory implementations.
5. Give `PersistedExecutionEnvelope` one authoritative absent/partial/valid
   classification.
6. Correct prompt attribution and bound the schema migration Job.

## Non-goals

- Do not introduce LangGraph or a new agent runtime.
- Do not replace Temporal, RabbitMQ, Redis, or the existing provider adapters.
- Do not redesign unrelated ListingKit, Shein, or Temu execution paths.
- Do not infer identity for legacy tasks.
- Do not retain compatibility with contaminated governed score-cache entries;
  the new namespace intentionally makes them unreachable.

## Design principles

- Rollout exclusion is an expected `legacy` route outcome, not an authorization
  error. `policy_denied` remains a terminal decision that performs no provider
  call.
- One logical execution produces one truthful invocation record for the path
  that actually ran. A legacy provider success must not be preceded by a false
  failed active invocation.
- Cache lookup happens only after an execution plan has been resolved.
- Security and identity invariants are expressed once and tested against every
  implementation.
- ProductEnrich-specific client adapters stay in ProductEnrich; shared AI
  packages contain provider-neutral execution metadata only.

## 1. Governed execution plan

Add a small provider-neutral planning contract to `internal/aicapability`:

```go
type ExecutionPlan struct {
    Mode          RoutingMode
    RouteOutcome  RouteOutcome
    Decision      RouteDecision
    LegacyClients []string
}

type ExecutionPlanner interface {
    Plan(context.Context, RouteRequest) (ExecutionPlan, error)
}
```

The ProductEnrich planner receives an active-tenant selector, the existing
`Router`, and an ordered legacy client list.

- active tenant: call `Router.Decide` and return `Mode=active` with its
  `RouteDecision`;
- non-active tenant: return `Mode=legacy`, `RouteOutcome=legacy`, and the
  configured ordered legacy clients without calling the active router;
- missing identity, invalid request, or a real policy failure: return a
  categorized terminal error and no executable plan.

The rollout allowlist therefore leaves the active model policy. The active
policy resolver no longer uses `policy_denied` to signal rollout exclusion.

After planning, the ProductEnrich adapter resolves the selected active or
legacy client into an internal bound execution containing only
provider-neutral provider/model/configuration metadata plus the callable client.
This binding is resolved before cache lookup but is not added to
`internal/aicapability`, so provider-manager details remain local.

## 2. One invocation lifecycle for text and image

Refactor the ProductEnrich governed text and image adapters around one internal
execution helper. The helper owns:

1. identity validation;
2. plan resolution;
3. active or legacy client resolution;
4. provider call timing;
5. prompt/input/output hashing;
6. exactly one invocation record containing the actual route and outcome.

Legacy resolution is ordered. For text understanding the candidates are
`fast`, then `default`; for vision understanding they are `vision`, then
`default`. Quality scoring uses the configured scorer client followed by
`default`, without duplicating a candidate. A missing named client therefore
preserves the pre-governance behavior.

An active route record uses the provider/model/policy/configuration metadata
from `RouteDecision`. A legacy record uses `RoutingModeLegacy` and
`RouteOutcomeLegacy`; it records the resolved legacy client identity and the
actual provider success or failure. A terminal route failure may be recorded as
a rejected execution, but it must not be represented as a provider invocation.

Text and image adapters remain narrow domain interfaces. The shared helper
accepts callbacks for the provider-specific text or image call rather than
depending on OpenAI or Gemini types.

## 3. Governed scoring cache identity

Scoring resolves and binds an execution plan before reading its cache. The cache
key is a stable hash of a versioned identity containing:

```text
cache schema version
tenant ID
capability and operation
route mode and route outcome
provider ID, model ID, and routing key
policy version and configuration version
prompt key, prompt version, and prompt scope
base score
input hash
```

Legacy plans include the resolved legacy client/configuration identity. This
prevents two tenants or rollout modes from sharing a result even when their raw
text or image input is identical. The base score is included because it is part
of the rendered scoring prompt.

Cache hits still pass through identity validation and plan resolution. They are
recorded as governed execution outcomes with the same route and prompt lineage,
but with no provider usage or cost. Add a provider-neutral `CacheStatus` field
to `InvocationRecord` and its GORM row with values `not_applicable`, `hit`, and
`miss`, so a cache hit is never misreported as a provider call. Existing
unversioned cache entries are not read.

The cache interface accepts a typed `ScoreCacheIdentity` instead of raw text or
URL values. Text and image convenience methods may remain, but they must derive
their key from the complete identity.

## 4. Repository tenant-scope contract

Define one shared tenant-match helper based on verified `aiidentity` context and
the task's persisted execution tenant. Repository behavior is:

- when the context has a verified tenant, reads, lists, and mutations can only
  affect tasks whose `execution_tenant_id` matches;
- cross-tenant access returns the existing not-found error and reveals no task
  existence;
- when no verified tenant exists, existing worker and migration behavior
  remains unscoped; worker execution must still restore and validate the durable
  envelope before governed AI calls.

Both GORM and in-memory Amazon repositories use this contract. A repository
conformance suite runs the same get/list/mutation cases against both
implementations. ProductEnrich and ProductImage repositories retain the same
contract and receive matching conformance coverage where an in-memory
implementation exists.

## 5. Persisted envelope state

`PersistedExecutionEnvelope` exposes an authoritative classifier:

```go
type PersistedEnvelopeState int

const (
    PersistedEnvelopeAbsent PersistedEnvelopeState = iota
    PersistedEnvelopePartial
    PersistedEnvelopePresent
)

func (p PersistedExecutionEnvelope) State() PersistedEnvelopeState
```

`Absent` requires every persisted field, including `ExecutionTraceID`, to be
zero. `Present` requires a supported version and all required identity/source
fields. Every other combination is `Partial` and fails with
`ErrIdentityIntegrity`.

`ExecutionEnvelope` and service guards use this classifier. ProductEnrich and
ProductImage no longer enumerate persisted fields inline.

## 6. Prompt metadata and migration safety

`GovernedImageAnalyzerConfig` gains `PromptKey`, `PromptVersion`, and
`PromptScope`, with the current understanding prompt as its default. The quality
scoring builder supplies `productenrich.llm_scorer.image_scoring` and its
resolved prompt version explicitly.

The product schema migration Job gains `activeDeadlineSeconds: 900`, matching
the driver timeout and the ListingKit migration Job. The workflow/static tests
must assert the deadline so it cannot regress.

## Failure semantics

- missing or malformed identity: fail closed before planning, cache, or provider
  execution;
- active route denial or unavailable capability: no legacy escape unless the
  execution plan was explicitly `legacy` before active routing;
- missing named legacy client: try the next configured candidate;
- all legacy candidates unavailable: return credential-unavailable and record
  the failed legacy execution;
- provider error: retain current retry behavior, then record the final actual
  outcome;
- recorder or cache write failure: log through the existing callbacks without
  changing a successful provider result.

## Verification

Acceptance requires tests proving:

1. a non-active tenant with only the default client succeeds through legacy
   text and vision paths and produces one successful legacy record;
2. a failed legacy provider produces one failed legacy record;
3. a real policy denial performs no provider call;
4. equal input under different tenant, route/configuration, prompt version, or
   base score cannot share a cache entry;
5. a cache hit still validates identity and emits truthful audit metadata;
6. GORM and memory repositories reject cross-tenant get/list/mutation access;
7. a trace-only persisted row is partial and fails integrity validation;
8. vision quality scoring records its quality prompt identity;
9. the product schema migration Job is bounded to 900 seconds;
10. the full Go suite, deployment workflow tests, and migration manifest tests
    remain green.

## Rejected alternatives

**Patch each review line independently.** Rejected because it would preserve
the false `policy_denied` rollout semantics and allow future cache or repository
implementations to bypass governance again.

**Disable score caching whenever governance is active.** Safe but rejected as
the final design because it removes useful behavior and does not define the
identity required for future governed caches.

**Move ProductEnrich client resolution into `internal/aicapability`.** Rejected
because it would pull provider/client-manager details into the provider-neutral
boundary.
