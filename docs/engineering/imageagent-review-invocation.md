> #1033 源码退休替代说明：本文为历史证据，原文完整保留。旧通用抠主体/渲染/模型审核、OpenAI Manager/请求池/全局与成员凭据回退及通用多槽位发布的 KEEP/供应承诺由[当前完整退休边界](../architecture/legacy-imageagent-generic-execution-closure-retirement-boundary.md)替代；仅抽取当前来源编辑、结果构造、单主图批准和策略值规则。当前图组/Artwork、JSON/hash、预算/UNKNOWN/恢复和 Store 观察同步保持；本注记不授权旧历史续跑、真实资源操作或增加试用前置。

# Organization Review invocation contract

Refs #334. This increment consumes the merged #339 organization assembly and
does not change IAM, run ownership or Temporal recovery. Historical baseline
evidence under `evidence/issue334` remains dated evidence, not current behavior.

Independent narrow admission: IMPLEMENTATION_READY. App owns the recording
policy; Integration owns actual adapter request/response observation; the
existing GormInvocationRecorder owns persistence. No second client, ledger,
retry scheduler or customer charge is introduced.

- Organization Review requires a matching verified effective organization,
  actor, restored run/context, recorder and normalized route/quote before I/O.
- One authorized model call uses the existing request-level MaxRetries=0
  override. The BaseClient retry algorithm and legacy consumers are unchanged.
  No model fallback is installed. Tests count actual loopback HTTP requests.
- Each adapter call has a local observer for safe usage and prompt metadata.
  Missing usage and actual cost remain explicitly unknown; an authorized cost
  upper bound is never recorded as actual cost. No prompt/image/raw error enters
  the ledger or degradation log. External references are bounded and validated.
- App writes one invocation after adapter completion using WithoutCancel and a
  two-second recording deadline. This does not extend the provider deadline.
  Missing recorder is rejected before dispatch; a write failure emits a safe
  structured degradation event and preserves the original model result/error.
  Ledger failure never becomes a retryable provider error or model fallback.
- Existing Product Image post-response cancellation semantics remain. The
  organization Activity's existing budget reservation/outcome gates own reentry;
  tests prove no extra provider call for the same action after success, record
  failure and cancellation. No arbitrary crash exactly-once claim is made.
- Existing ledger primary-key conflicts are retained: replay or differing facts
  under one invocation ID cannot overwrite its original record. Model execution
  is never used to repair a record. Logger filtering covers the resolved client
  and pool, including error-body sentinels in failure tests.

Validation uses real HTTP admission, isolated PostgreSQL, Temporal converter/SDK
Activity, Manager, credentials, adapter and recorder. Identity issuance, remote
IAM, Temporal server acceptance, staged prior generation evidence and provider
responses are controlled external boundaries. Real IAM, paid models, production,
generation/approval/publication and real-server restart remain NOT_RUN.
