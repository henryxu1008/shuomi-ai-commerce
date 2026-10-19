# 模型与费率管理 V1

> Status: **DESIGNING / NOT_READY**；Design Basis: **Independent Architecture**。
> Execution: [#830](https://github.com/qq550723504/task-processor/issues/830)。
> Product decision: **PD-MODEL-ROUTE-PRICING-MANAGEMENT-2026-10-19**，用户批准本会话最小方案并要求设计、Figma、独立架构评审先行。
> Source baseline: `main@48e15a7f9f9c3e32e76a742fe1581ea547461e9a`。本文是设计合同，不宣称已实现或运行可用。

## 1. 用户结果、Authority 与范围

平台维护者在已验证的当前企业内，选择一个已接入业务用途，查看当前模型和费率，选择已有合格凭据绑定，维护费率，经本地校验、差异预览后发布一个版本。版本历史可追溯；恢复历史设置也产生新版本。普通企业管理员或成员不因此获得平台定价权限。

复用 [当前 UI/IA Authority](../product/final-ui-ia-authority.md) 的 Figma 文件 `tg48P46SSXl6TBy9lZwg63` / 页面 `31:463`。入口在「我的账户 → 企业空间」的平台维护区，进入「模型与费率」；只对后端确认具有平台维护资格且有当前企业权限的身份显示。不是新的业务一级模块，不在套餐购买/充值页面编辑执行费率。

可编辑设计区域 [7658:2087](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7658-2087)：文本深/浅 `7660:2088` / `7660:2221`，图片 `7660:2143` / `7660:2276`，预览 `7660:2167` / `7660:2300`，回执与历史 `7660:2186` / `7660:2319`，不可用/待核实 `7660:2196` / `7660:2329`。企业空间 `432:4483` 的管理入口 `7663:2225` 仅为平台维护者投影，不改变企业角色权限。复用现有 Console token、Noto Sans SC、Action/说明卡，缺失字段组件为 `7658:2118`；没有完整 UI 栅格图片。

图片专用预览深/浅 `7666:2226` / `7666:2259`、回执 `7666:2245` / `7666:2278`，按 points/image 表达差异，不能跳用文本 Token 费率预览。原型状态链接只打开相应 Figma 设计节点，绝不连接实际发布 API。

继承而非重做：

- [文本供应商中立合同](product-agent-text-provider-neutral-v1.md)、[Product Agent runtime](product-agent-runtime-contract.md)：原 quote、claim、一次发送、完整 usage、UNKNOWN、预算、Review/Apply。
- [企业 Agent Configuration](organization-agent-configuration-v1.md)：企业启用、模板、start snapshots 和 command owner。它不拥有当前模型/价格，不借用这些表保存当前路由。
- [Product Image Agent](product-image-agent-v1.md)：原图片计划、GenerationIntent、reservation、dispatch、proof/recovery、Asset/selection owner。
- [Commercial retail pricing](commercial-retail-pricing-v1.md)：AI 点数充值售价仍归 Commercial catalog，与供应商估算成本和模型消费点数分开。
- [Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Legacy hard-cut](../refactoring/legacy-hard-cut-policy.md)、[AGENTS](../../AGENTS.md)。

此决定只替代上述文本合同中「部署不可变 map 是当前选路/费率来源、无供应商选择 UI」这一有界限制，以及图片 factory 固定 client name/启动费率的选取点。每组织每用途一个当前 route、所有 adapter/usage/硬上界准入继续有效。不是批准任意模型目录、企业 BYOK、新 provider 或新协议。

### 1.1 Must / Should / Out of Scope

**Must**：M1 只有有当前企业访问资格的平台维护者可读写管理数据；M2 不绕过已准入 provider/model/adapter/usage/凭据版本及预算上界；M3 一次发布在一个原 owner 数据库内原子更新、并发有冲突、幂等可取回准确回执；M4 新准备与最后发送复核消费同一权威版本，旧任务不被静默换模/改价；M5 原 in-flight/UNKNOWN/结果/账单/Review 事实保持；M6 页面明确当前/历史/未发布/不可用，配置操作零 provider 调用、零新任务。

**Should**：复用 Console 明暗主题和组件；390px 表单单列可读；错误定位字段并保留未发布编辑内容；历史支持有界分页。

**Out of Scope**：Key 写入/显示、任意 URL/JSON、模型自动发现/在线探测/费率抓取、未准入模型、新协议、自动 failover/重发、跨用途/跨企业原子切换、阶梯/缓存/时段价格、FX、供应商真实账单对账、新账本/运行时/恢复平台、重算历史账单/迁移旧数据、安装/部署/操作保留实例或真实模型。

无新增安全风险豁免；不以 MVP 放宽当前租户、权限、计费和 UNKNOWN 底线。已准入模型的实际健康与价格来源真实性仍由原运营准入负责；本页只验证配置一致性，不证明供应商可用或外部价格正确。发布不是取消已经通过最终发送门禁的请求，属于本轮明确的生效语义。

### 1.2 Threat Model

保护：成员伪造平台角色或企业、切企业后提交旧表单、凭据跨企业引用、任意 endpoint/未准入模型注入、旧校验结果偷换、并发覆盖/重复发布、价格精度/溢出/预算绕过、缓存漂移、切换导致重复发送/错误结算、密钥泄漏。

不新增通用 IAM、批准流、探测器或供应商计价认证系统；复用已批准身份/成员撤销时序、当前安全传输和可信运营者边界。管理员也不能通过本页扩大服务器硬限制。

## 2. 当前实现与根因

| 当前路径 | 可复用行为 | 缺口与最小增量 |
| --- | --- | --- |
| `internal/integration/aicapability/einomodel/route.go` | 按企业/operation、精确 credential/endpoint/version 匹配；既有 adapter allowlist | 构造时复制部署 map，需在原 resolver 注入可信的当前策略 reader |
| `internal/aicapability/governed_text.go`、`model_point_pricing.go` | ModelProfile、估算成本/点数分离、版本 digest、整数/向上取整、最大 quote | 复用原形状，不另建费率计算器/账本 |
| `currentapplication/product_agent.go`、`ai_workbench.go` | 各文本消费者注入、allowlist/预算硬上限 | 原注入点接当前 reader；未接消费者不显示可管理/可执行 |
| `internal/integration/openai/credential_store.go`、`title_credential_provision.go` | `ai_client_credentials`、私密 writer、版本和去敏回执；组织行约束 | 管理页只读去敏绑定元数据；准备新绑定仍由原私密 provisioning，不能覆盖旧任务绑定来切模 |
| `imageagent/generation_provider.go`、`generation_runtime.go`（app worker） | 已准入图片 adapter、metadata 精确比较、一次调用与原恢复 | 固定 client name/启动图片费率改成该图片消费者的当前策略；原 factory/executor 继续使用 |
| `httpapi/ai_workbench_tasks.go` | frozen profile 对比；Review 与 CanStart/CanResume 分开 | 页面发布不能让历史请求静默换模或阻断已生成结果的审核 |

根因是**当前策略没有可持久化、可事务发布的运行事实 owner**。只新增 dropdown 或修改环境变量既不能提供发布回执，也不能保证多进程/重启一致。最小补齐放在现有 AI route/能力 owner 内，文本落原 ProductAgent/Run DB，图片落原 ImageAgent DB；不建设跨数据库控制平面。

## 3. Owner、配置单元与调用接线

### 3.1 一个配置单元

键为 `(verified organization_id, capability_key)`。文本 capability 对应现有 operation：`product_agent_decision`、`product.content.decision`、`listing.materials.decision`、`product.selection.decision`、`shop.operations-diagnosis.decision`、`aiworkbench_chat_plan`。只有原依赖/feature/allowlist 实际注入者可管理；保存配置不启用这些功能。

图片为现有共享 generation factory 的一个消费者单元 `imageagent.generation`，包含现有使用该 factory 的商品图片和 artwork 用途；不虚构按 slot/任务分别选模。UI 明示这一影响范围。不同数据库/用途分别发布，部分成功只返回各自真实回执，无「全部切换成功」。

### 3.2 复用链

| Contract → implementation → injection → consumer | Owner/持久化 |
| --- | --- |
| 当前配置 Read/Validate/Publish/Receipt → AI capability 下的有界配置 service + PostgreSQL repository → currentapplication 对应 DB pool/HTTP module → Console/BFF | 文本 ProductAgent DB；图片 ImageAgent DB；各自事务 |
| Current text policy reader → 原 einomodel RouteResolver / ModelProfile → 原 product_agent、ai_workbench 注入 → quote / Start / Decide / 最后发送 gate | AI capability/route；历史 Invocation 仍原 owner |
| Current image policy reader → 原 generationProviderFactory / metadata → ImageAgent runtime → prepare/confirm/slot revalidate / executor | ImageAgent；GenerationIntent/proof/资源不转移 |
| 去敏 CredentialBindingView → 原 organization-only resolver/store → 同 owner DB → 只返回允许能力的合格引用列表 | `ai_client_credentials`，不新建 secret owner |

部署层保留静态安全/准入描述（provider 身份、model/adapter、用途兼容、endpoint digest、usage mapping、硬 token/bytes/deadline、预算币种/硬上限），这些不是第二套「当前选择/当前价格」。可选绑定由这些已有准入描述与实际组织凭据的交集生成；新增合格绑定须先通过原部署侧准入/provisioning。本设计允许该私密入口按候选准入描述核验，不能仍只允许当前选中那一条，否则切换 happy path 无法完成。不得因此开放通用 credential API。

管理范围的当前 route/费率唯一来自数据库已发布版本。缺行、读失败、凭据禁用/变更或服务未接线时 fail closed；不回落 manifest、成员凭据或全局 Key，不从历史 snapshot 推造当前配置。不使用可陈旧的进程内策略 cache。未纳入本页的其它原消费者继续其批准路径，不通过相互 fallback 连接。

## 4. 最小持久化与事务合同

每个已有 owner DB 安装同形状的两张有界表（由原安装器/schema owner 显式安装；本轮不建表）：

| 记录 | 必要字段/约束 |
| --- | --- |
| `ai_route_config_revisions` | `revision_id`、org、capability、parent revision、schema version、payload digest、不可变去密 payload、actor、published_at、idempotency_key、request_fingerprint；`UNIQUE(org, capability, idempotency_key)`，`UNIQUE(org, capability, revision_id)` |
| `ai_route_config_current` | `PRIMARY KEY(org, capability)`、current revision；复合 FK 指向同 org/capability revision，不能跨作用域引用 |

revision 同时是发布回执/历史，不另建 command ledger、persistent draft、审计系统或任务状态机。发布后业务角色无 UPDATE/DELETE revision 权限；current 只有受限配置 writer 可 CAS。凭据 secret 不复制进 payload/digest/日志；引用使用 credential owner 的非秘密身份、版本及 endpoint digest。

文本 payload 保存原 ModelProfile 所需值：已准入 binding、provider/model/adapter/usage 与限制引用、原币种、input/output micros per million、input/output AI points per million、各价格版本。图片保存原 metadata 所需 binding/provider/model/protocol/限制引用、points per image 与 priceVersion。不新增图片供应商成本推算；该字段明确「尚未接入」。两个费率均使用严格十进制解析为原整数单位，禁 float、NaN、负值、零值、超精度和溢出；复用现有 Validate/MaximumCost/PointTariff 与原预算，不能因配置版本而放宽上界。费率版本及 routing/configuration version 由服务端生成并绑定新 revision，浏览器不能填写或复用旧版本。

### 4.1 Validate 与 Publish

`Validate` 无写入、无 provider 调用：以 fresh identity/当前企业读取当前 revision 与合格 binding，构造完整 payload，复用原确定性 validator，返回规范化配置、base revision、payload digest、差异/任务影响提示。它不是授权票据或供应商健康证据；编辑任一字段/切企业/版本变化后必须重校验预览。

`Publish(expectedCurrentRevision, canonicalPayload, idempotencyKey)`：

1. 每次重新验证当前身份、有效企业/成员及平台资格，拒绝 unknown fields、任意 org/member/endpoint/Key/上界参数。
2. 在所选原 owner DB 事务内先查该作用域幂等记录；相同 key+fingerprint 返回原回执，不同 payload 返回 conflict。fingerprint 覆盖 verified org、capability、actor、expected revision、规范化请求（含目标 binding/version、费率及 rollback 来源）。
3. 在同一 owner 事务内用一个去密投影查询读取并核验精确 credential 身份、企业、Enabled、model/协议、版本和 endpoint digest，重新检查准入描述、权限上下文和预算。使用原 PostgreSQL READ COMMITTED 事务即可；**不锁凭据行、不授凭据 UPDATE 权限**，不调用保存凭据路径、不读写 Commercial 库，外部验证不放进事务。该读取不是提交时凭据仍有效的承诺：并发轮换/禁用可让刚发布版本立即成为 NEEDS_CONFIGURATION；发布回执保持，current 投影与实际执行各自 fresh 核验并如实拒绝失效绑定，不回落另一绑定。无需改变原凭据 writer 的锁/隔离协议。
4. 锁定 current、比较 expected revision。无当前版本的首次发布使用显式 `NONE`，并依靠唯一约束收敛并发。匹配才插入新 revision 并 CAS current，同事务提交。重复幂等键竞争导致唯一冲突时回滚整个事务再读取原回执并比较指纹；不部分提交。
5. COMMIT 成功返回 revision 回执；提交结果无法确认则返回「发布结果待核实」及原 key，先按原 key 读回执，不换 key 重发。已 rollback 的失败无新当前版本；客户端保留相同 key/相同 payload 可安全重试。读回执失败时继续待核实。

回执包含作用域、revision/parent、payload digest、actor/time、差异摘要及请求 key，不包含 Key/完整 endpoint。**历史幂等回执不等于仍然当前**：页面还读取 current revision；若别人随后发布，显示「本次已发布，当前已有更新」并跳到实际当前版本。

rollback 是复制指定历史 payload 后重新执行 Validate/Publish，parent 指向实际当前版本，生成全新 revision/route/price/config identity。旧 binding 已禁用或不再合格时拒绝；旧版本不复活，防止 A→B→A 令旧 quote/确认重新有效。历史不可改写，不做 paid plan/entitlement/余额变更。

### 4.2 API、资源与角色

在当前 HTTP/BFF 组合根注册 feature-local `model-route-pricing` 模块；建议 `/api/model-route-pricing/{text|image}/{capability}` 的 current/history/bindings/validate/publish/receipts，具体路由沿原注册 convention。org 从 verified Effective Organization 取，不接受任意租户参数。BFF 沿用当前认证/CSRF防护和 `X-Expected-User-ID`、`X-Expected-Organization-ID`；切企业后旧页面提交 fail closed。

所有管理读写同时要求注入的当前 authorizer `listingkit.platform_admin` 与原企业访问资格；发布再要求已有 `listingkit.admin.write`（并保留对应 owner 的既有准入）。不可只检查 org-admin/隐藏按钮，也不可调用使用默认全局 allowlist 的静态 helper。权限被撤销后的下一请求/最后业务发送按原 fresh identity 门禁拒绝；不声称撤回已通过原门禁的在途请求。

JSON 请求 ≤16KiB，未知字段拒绝；history 每页 ≤50、游标限定同 org/capability；身份/版本/键沿原长度规则且本模块 key ≤128 bytes；HTTP deadline ≤30s，无网络探测。幂等记录与 revision 同寿命，不因清理失去回执。数据库错误正文和 supplier 响应不回显。

原 credential writer 只管凭据；配置 writer 只写两张配置表并读所需去敏凭据元数据；serving runtime 读取当前配置/历史执行需要的原事实；无 DDL/AutoMigrate/启动 seed。原组合根复用各 pool 生命周期，不为了管理建立第二 worker/队列/缓存。

## 5. 生效、原任务与外部副作用

发布只改变后续准备读到的版本，不创建/启动/续跑任务、不调用模型。quote/ProfileDigest、图片 configuration/price identity 必须纳入 publication revision；准备与最后发送复核都经原 reader 取得当前权威版本。配置改变后用原明确错误/阻断投影提示重新准备与确认，不自动替换 payload 或让旧 key 重新报价。

| 原任务状态 | 发布后的合同 |
| --- | --- |
| 未确认 quote/proposal | 原确认失效；新请求重新准备并明确确认 |
| 已创建、尚未最终发送准入 | 原 frozen 配置不匹配则拒绝发送；不静默换模/改价 |
| 多步运行 | 已完成输出保留；下一模型步骤遇漂移停止，沿当前可用的新准备入口产生明确新确认；无新恢复状态机 |
| 已通过最终发送门禁/在途 | 允许原已准入的一次请求返回；以原 frozen 模型/费率和实际 usage 结算；发布不是远端取消屏障 |
| UNKNOWN/缺 usage/终态持久化不明 | 保留原 invocation/intent、reservation、原 proof/reconcile 责任方；不换模型重发、不当零费用释放 |
| 已生成、待 Human Review | 使用原事实与授权继续 Review/Apply；不为审核再生成；原来源/撤销/Apply 校验仍有效 |
| 已完成/账单 | 模型、价格版本、usage、点数保持原事实，历史不重算 |

切换与最终发送门禁并发的边界以**该次门禁读到并核验的版本**定义：读到新版本必须拒绝旧请求；已按旧版本通过门禁者可继续原一次发送。因此发布响应后可能仍有先前已准入的旧请求发出/完成，不承诺全局网络静止。无需长事务跨 provider HTTP，也不新增分布式锁/Saga。

恢复读取原执行事实/观测证据不能依赖新当前模型可调用，不能将新当前凭据写进旧 intent。切换时旧凭据绑定留存，不通过 upsert 覆盖或删除；配置历史不授权调用旧 route。确需原未决结果查询时仍由既有原请求身份/凭据引用的恢复 owner 负责，缺原凭据时如实 blocked/UNKNOWN，不换新 provider。禁用或轮换凭据仍遵循原独立运维授权；本页无该操作。

## 6. 产品页面与错误状态

页头明确当前企业与「平台维护」，用途选择只列已接线消费者；其它项给出未接入原因且不可发布。当前区展示准确 revision、provider/model、binding 非秘密引用/版本及实际配置准入状态；不显示伪造模型健康、账户余额或供应商成本。

文本编辑：选择已准入绑定；协议、usage mapping、token/bytes/deadline、预算币种与硬上限只读；input/output 估算成本每百万 Token、input/output 产品 AI 点数每百万 Token 分组，币种不混算。图片编辑：一个共享图片消费者、points/image、服务端 price version，**可选择已有合格绑定，协议与限制只读**；供应商成本标尚未接入。充值点数零售价链接到原 Commercial 入口但不在本页编辑。

编辑未生效 →「校验配置」→ old/new diff、单一配置单元、旧任务影响 →「确认发布」→ 回执及当前版本。页面清楚说明校验不调用模型、发布不会自动恢复任务。校验/预览 token 不能绕过发布时重查。待核实时只提供「查询本次发布结果」；新编辑必须先消除原发布歧义。历史恢复走同一预览与确认。

空配置/无合格绑定/未接依赖：真实 unavailable；无权限：denied；字段错误：保留编辑并定位；CAS 冲突：保留编辑、读取当前并重新校验；凭据变化：旧预览失效；读失败不保留旧配置作为「当前」。原型的组织、版本、模型及所有价格/时间均显式标为设计示例，不写入数据。

## 7. 安装边界、Legacy 与验证

新安装先由原 schema installer 安装表/角色与权限，应用 startup 只验证；管理 reader 接线是明确的部署决定。纳入管理的消费者不自动导入旧 map、seed 当前版本、迁移历史 invocation/start snapshot、回填价格、修复现有数据或删除凭据。首次发布可以建立 NONE→revision；此时无 current 的执行保持不可用。准备好的历史 route/binding 描述只是校验依据，不能成为第二 current source。保留实例未来升级/安装/初始发布须另获用户授权，本轮均 NOT_RUN。

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: 原 ModelProfile/图片 metadata、准入校验、credential 组织解析、版本失效、预算/向上取整、quote/claim/一次发送/UNKNOWN。
Current owner: AI capability/route、ProductAgent、ImageAgent；原 Credential/Commercial/Review owners 不变。
Cutover/deletion condition: 对本管理范围，原启动 map/固定 client/费率选取点由同一 reader 替换并退休；不 wrap 旧 Service，不 fallback/双写/第二事实源。
```

未来实现只需在同一 Delivery Batch 做与风险匹配的四组测试；不另造 runner/验收平台：

| 组 | 必要证据（本设计阶段均 **NOT_RUN**） |
| --- | --- |
| V1 权限与准入 | 注入平台用户/角色自定义配置、企业成员/企业切换/撤销、跨企业绑定拒绝、无 Key 泄漏；受支持 model/adapter/用途/usage/exact credential；真实只读凭据角色可发布但不能 UPDATE 凭据；保存不能打开关闭功能 |
| V2 原子发布与费率 | 整数精度/取整/最大预算、并发 CAS/首次发布、同键同载荷/异载荷、事务中断/COMMIT 不明回执、重启/多进程同一 current、rollback 新身份防 ABA、无 manifest fallback；凭据轮换/禁用与发布交错后 current 如实 NEEDS_CONFIGURATION，运行拒绝失效版本 |
| V3 执行与历史 | 上表所有原任务状态；切换在 prepare/最后 gate 前后、多个 step/图片 slot；旧费用结算、UNKNOWN/预留与已生成 Review 保留；不增加第二发送或伪造免费 |
| V4 页面与接线 | 文本/图片 edit→validate→preview→publish→receipt/history；空/无权限/冲突/待核实；两 owner DB/角色 startup verify；fake transport 断言所有管理动作零模型请求；390px 和明暗主题 |

业务规则后续 TDD；事务测试复用现有隔离 PostgreSQL 能力；模型传输用现有 fake/fixture，不要求真实 provider。实现自检/CI、独立最终检查、用户实际试用与生产/模型调用分别记录，不能互相冒充。

## 8. 独立架构评审与停止点

候选完整后由一名独立只读 Reviewer 检查本增量的真实 source/contracts、Figma 和全部上述边界。先分类 finding：BLOCKER 必须对应 AGENTS 的明确类别/受影响 Must/阻塞层；非 Blocker 进入 IMPLEMENTATION_TEST 或 Backlog，不扩大 Threat Model。最多两轮正常评审，后续只复核实际修订增量。

当前状态 **NOT_READY**，等待候选独立评审。达到 IMPLEMENTATION_READY 后冻结本设计；生产代码实施、合并、安装、切模型、真实模型调用及运行验收仍需各自后续授权。滚动 HEAD、CI、评审原文和 disposition 维护在 #830/同一设计 PR，不在此文伪造执行 PASS。
