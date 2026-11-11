# 内容 Chat 与完整商品智能体共存：设计候选

状态：DRAFT，独立 Architecture Review 进行中，不是 IMPLEMENTATION_READY。基线 `6873bb5846154f667d530452947a9d6cdbb0d177`。本文件是有界设计调查，不授权修改正式业务路径、保留实例、价目或凭据。执行 Issue #1242，关联产品目标 #298。

## 结果与范围

获准企业成员在 Chat 选择现有商品、提出内容目标、查看方案并确认后，沿当前内容 Agent、任务、人工审核和采用路径工作；原完整 Product Agent 与其其他已装配能力继续可用。会话管理可用不等于模型消息可用。明确不做一般聊天、模型调用、发布、账务变更、权限提升、数据迁移、视觉改版或修改任何其他能力。部署、真实配置及任何付费调用需另行明确决定。

依据：用户 2026-11-11 指定保留 35444 原能力、原预留和数据，选择已有企业 OpenAI 连接，仅授权先处理共存与隔离验证；`ai-workbench-work-scope.md`、`chat-task-object-goal-design-brief.md`、`shared-channel-credentials.md`、当前权限和资源合同。Figma 非 UI 权威。

## 根因和拟议最小变更

当前完整 ProductAgent 已构造独立内容 application，共享已存在的配置、Run、Review、资源和授权 owner；当前组合却在 manifest 校验、HTTP composition、Workbench 构造三层硬性要求 `contentTasks` 与 `contentOnly` 绑定。内容 application 非空、已注入商品集合且准确企业受限时，候选仅允许完整模式也选用它；`contentOnly` 原合同不变。明确保留 content-only 禁止标题路径、conversation-only 原组合限制和各模式 fail-closed，不让配置自动降级或借其他 Agent。须逐层检查装配顺序、`workbenchScopes` 只挂在内容 application、共享指针和直接商品内容/标题的既有行为。是否能够无新增状态/事实达成须由实现前独立评审核对，不能凭候选宣称已覆盖。

## Owner、权限和效果

Chat 会话/消息/命令/方案/任务由原 `ai_workbench` owner 保存；内容调用和 Run、Review、商品、企业连接、模型选择、价目与 AI 点仍分别由当前 owner 唯一负责。用户、企业、成员、Chat allowlist、商品权限、范围权限、配置和内容执行权限按原路径 fresh 检查；不得从完整 Agent 或另一企业借权限、凭据或 readiness。完整 ProductAgent 的其他 application 和授权不被替换。

规划使用单独 `aiworkbench_chat_plan` 精确用途，执行使用 `product.content.decision`；二者独立合格后才可发送或确认。原企业共享连接可在明确用途 binding 中引用，但连接本身不代表型号、价格或调用授权。选择、模型、连接版本、价目/汇率、限额和适用策略经原路径冻结、发送前重核；不得伪造版本、复用另一企业或用占位价发布。Chat 无能力时真实展示不可用，不将单纯建会话等同消息成功。

## 状态、事务、失败和恢复

不新建状态机、调用 ledger、预算或事实 owner；沿现有 Conversation/Command/Proposal/Task、幂等键及原单次 claim/发送 gate。相同键不同载荷冲突；缺授权、凭据、价目、预算或范围输入失败关闭。凭据轮换与已冻结引用不一致时拒绝新发送。调用结果 UNKNOWN 与原预留保留，绝不换键重发、推定零费用或释放既有预留；已提交事务的回执和本地恢复沿原责任方。验证必须覆盖两个 Agent 的各自能力、共享依赖、失败时不创建意外副作用以及范围授权。

## schema 与目标实例

当前原实例的旧空 Chat schema 缺少 `business_scope`、`frozen_scope`、`scope_digest` 等当前 schema 所需字段；serving 校验拒绝且不会自动 DDL。现有安装器是新安装入口，不得假设 `CREATE TABLE IF NOT EXISTS` 能升级已有表。零记录只读检查不授权删除、迁移或修改原 schema；本候选不提出自动迁移、兼容或第二事实源。隔离测试使用独立新空库。目标环境的可行安装与回滚方式、已有依赖和备份，以及是否改变原数据库，必须有具体方案并在用户决定后单独处理；当前不声明现场可开通。

## 验证与准入

本候选需独立 Architecture Review、执行 Issue 的 Design Basis / scope 及 `IMPLEMENTATION_READY` 后，才可修改正式生产路径。准入前允许明确标为非正式、不得合入的隔离 Spike。必要验证应在任务专用 PostgreSQL、合成凭据及本地模型中运行：完整 Agent 与内容 Chat 同时装配；内容路径按范围完成到 Review/Apply；原标题及其他已装配入口的装配/行为不回归；不同企业和缺组件拒绝；轮换、预算、权限及同键恢复沿原合同；不接原实例或真实 provider。验证仅证明被测组合，不证明真实价格、真实凭据权限或用户验收。

开通方案需列出准确企业、模型、用途 binding、当前官网参考价/有效汇率、每次请求上界和建议预算、schema 方案、回滚方式及待用户决定的实际变更/真实调用；既有 15.445282 AI 点预留不动。

## 非正式隔离 Spike（2026-11-11）

使用 `go test -overlay` 的临时覆盖文件，不修改或提交生产代码，不构成可交付实现。目标基线 `6873bb5846154f667d530452947a9d6cdbb0d177`。专用 PostgreSQL 和回环合成模型下，仅放宽应用构造器内 content-only 条件，并在现有夹具构造完整 Agent：三个内容 Chat 路径均通过，断言完整标题 runtime / asset invoker 存在；覆盖方案、任务、审核和采用等原测试路径。该测试绕过顶层正式 composition，不证明真实启动或其他 Agent 全路径无回归。

单独配置 Spike 先加入完整模式+内容任务测试，原代码按预期失败；临时仅以内容策略存在代替 content-only 强制后，新增用例和现有模式/仅会话配置用例通过。尚未实现或验证完整顶层 composition、所有负向权限和保留实例 schema 路径。Spike 不用于正式构建或部署，不能替代独立架构评审/Issue Ready 或最终交付检查。
