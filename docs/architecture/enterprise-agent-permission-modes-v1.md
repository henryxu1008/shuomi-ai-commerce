# 企业智能体权限与审批模式 V1

> Status: APPROVED / IMPLEMENTATION_READY (architecture only)
>
> 必要独立 Architecture Review 已完成两轮：私有质检模式记录缺口已修订；第2轮对应增量复核明确无新增 BLOCKER。审查内容 SHA256：635F55BFD2A70B8D834A9037DA206B0AE0E0C7F441748BE70CE9CF47723861D4。本状态标记只记录准入，不改变审查合同；撤销/准确策略采用的相关测试仍属 IMPLEMENTATION_TEST，合并相关生产实现前须通过。
>
> Design Basis: Independent Architecture。当前会话为唯一设计/实现负责人；本轮先完成统一模式设计与独立高风险边界检查。正式生产实现遵守用户既定顺序，原会话先完成店铺同步/worker 与采集保存主流程。
>
> Execution Issue: [#656](https://github.com/qq550723504/task-processor/issues/656)。正式实现及其准确候选/运行交接状态以该 Issue 和主要 PR #657 为准；本设计的准入状态不代表功能已经完成。
>
> Inspected baseline: main f1d77a84fa653813235ddbfeedac59ea6bb0a94c，2026-10-11。
> 本文描述拟实现合同，不表示四种模式已经存在或已经开放。

## 1. 用户结果与产品决定

使用者是当前企业的智能体配置管理员，以及使用企业已授权智能体的成员。管理员在现有“我的智能体 / 能力配置”中选择执行模式；成员在执行确认与任务详情中看到本次准确模式、资源范围、需另授权的动作和不可用原因。标题、图片、指定企业私有质检及后续已准入智能体共用该语义，不建立各自的“完全访问”。

2026-10-11 用户明确要求：默认权限、自动审批、只读、完全访问四种模式；随后确认 **“业务工具全开放，外部发布等操作另授权”**。完全访问只覆盖当前企业内、当前操作者已授权、当前智能体已准入且实际存在的业务能力。真实上架、平台改价/下架、删除和增加付费额度始终需要单独授权。

本决定为局部业务动作增加可配置的事先授权，不授予机器宿主、任意脚本、任意工具、跨企业身份、平台管理员身份或新预算。选择模式不构成真实模型调用、部署、共享数据修改或真实平台写入的执行授权。

直接冲突引用的适用关系：现 organization-agent-configuration-v1 的“既有 Human Review / explicit Apply 不变”与 product-image-agent-v1 的“结果仅人工采用”继续描述现 default/未接入运行；后续本功能正式接入的 auto_approve/full_access 本地动作以本文策略采用合同为准。原人工入口、来源使用许可、owner 内容/版本校验、预算与外部授权不被替换。不得用旧人工-only描述恢复本次已明确授权的本地自动采用禁令，也不能拿本文的设计准入标记冒充现 runtime 已改变。

本次不同时开发此前讨论的五种新智能体，不接通模型/支付/SHEIN/SDS 凭据，不改店铺同步或采集主流程，不建立通用 IAM、审批平台、规划模型或第二套运行时。

Must：四模式真实后端执行，已授权企业/资源边界不提升，本地准确采用与可区分的审批出处，受保护动作不自动执行，现幂等/预算/UNKNOWN 不变。Should：复用现页面和开源组件，不增加常驻审批 Agent。Out of Scope 如上。

Threat Model：防止模型/浏览器自报权限、跨企业/私有交付泄漏、直接入口/worker 绕过、旧批准作用于变更内容、模式切换重发外部效果或增加预算。现有用户选 full_access 的本地自动采用不包含逐项人工审阅；程序校验也不保证 AI 文案/图片业务判断正确。这是所选模式的可见语义，不能记作人工确认或模型保真 PASS；不增加其他未获用户决定的 Accepted Risk。

### 1.1 UI / IA Authority

沿用 [最终 UI / IA](../product/final-ui-ia-authority.md)：
Figma 文件 tg48P46SSXl6TBy9lZwg63，当前非归档页面 31:463。
现有已准入配置流程引用“我的智能体”4703:429、能力配置4703:892、执行确认4722:429、Review4711:429。新增四种模式来自本次用户决定，只补现有配置和确认流程中的模式选择/说明，不新增顶层菜单或独立权限中心。

这些引用不是本轮已重新检查 Figma 的声明。UI 正式实现前读取当前非归档节点，确认位置与既有组件；若节点与当前产品决定冲突，报告具体差异，不能恢复已被用户替换的“一律人工确认本地动作”要求。

### 1.2 四种模式

| 模式 / API 值 | 业务行为 | 需要确认的内容 |
| --- | --- | --- |
| 默认权限 / default | 自动读取、分析、生成提案；普通业务事实变更进入既有 Review/确认路径。 | 本地商品/草稿/正式资产写入须绑定准确内容确认；经济意图按现报价和硬预算确认。 |
| 自动审批 / auto_approve | 在已有资源权限内，后台按代码声明的动作与确定性规则审核本地业务变更；通过才自动采用。 | 未覆盖的动作、校验不能判断或规则要求人工的内容回到既有人工确认；规则拒绝则停止，不能循环申请。 |
| 只读 / read_only | 读取、分析、诊断和生成建议；不得改变 Catalog、平台草稿、正式 Asset、来源授权等业务事实。 | 无“批准后自动写入”的只读例外；需写入必须显式创建另一个有写权限的新运行。 |
| 完全访问 / full_access | 当前范围内已准入业务动作可自动执行和采用，不逐项请求本地操作确认。 | 外部发布/改价/下架、删除、提升预算等单独授权；未开放能力或证据/权限/预算不足仍阻止执行。 |

“自动审批”是实际后台策略判断，不是把所有请求直接标为批准，也不是自动点击确认按钮。首期复用现有确定性校验，不新增付费 LLM 审批者。完全访问仍执行内容校验、版本检查、租户与 live 权限、预算和幂等控制，但不伪装成人工已逐项审阅。

只读允许原运行记录、模型计量/结算、审计及诊断报告/提案作为执行结果保存；这些记录不是商品/草稿/正式资产的改动。只读禁止生成或采用会产生新商品图片资产的图片流程；图片智能体可以查看原计划/结果，并明确展示“当前模式不允许生成/采用图片”。模型推理若已有授权报价/预算可运行；缺少模型配置则显示不可用。

## 2. 当前代码事实与根因

| 现状 | 可复用能力 / 缺口 |
| --- | --- |
| internal/commercetool/definition.go、invocation.go | 已有 RiskRead/Compute/Propose/Write/Publish、side-effect、权限、exact Tool allowlist、幂等与审计；invocableRisk 目前只放行 read/compute/propose。新增模式不能直接把所有 write/publish 解锁。 |
| internal/agentconfig/contracts.go 及 integration/persistence/agentconfig | 企业启用、CAS revision/activation epoch、模板、不可变标题/图片配置快照与命令回执已有 owner；尚无四种模式字段。 |
| internal/agent、integration/agent/eino | Scope/Binding/ConfigurationSnapshotRef、exact 请求指纹、checkpoint/Claim/Resume/UNKNOWN 已存在；不新增运行状态机。 |
| internal/product/review/service.go、execution.go | 标题候选、确定性校验、准确基线、Decide/Apply 同库事务与回执已有；tokenless worker 只能候选 intake，不能人工 Decide/Apply。 |
| internal/imageagent/image_set_service.go、temporal、assetpublication | 图片 prepare/报价确认、exact source/config、每 slot effect/UNKNOWN、结果集合 digest 和 Asset 发布已有；真实 effect 不都经过 Commerce Tool。 |
| internal/agentcustomization/private.go | 已交付私有质检按准确保存草稿生成不可变报告，无模型/平台 mutation；私有交付仍由该 owner 控制，不能因选择模式获得其他企业交付。 |

根因是权限范围与动作审批尚未组成统一、可追踪的执行策略。仅改 UI、仅改 Commerce Tool 风险开关，均无法控制标题 Apply 和图片 workflow 等直接业务入口。不得为此绕开各领域现有 owner。

## 3. Owner、合同与依赖方向

复用 [企业智能体配置](./organization-agent-configuration-v1.md)、
[Agent Runtime](./product-agent-runtime-contract.md)、
[完整图片智能体](./product-image-agent-v1.md)、
[私有交付](./private-agent-delivery-v1.md)、
[Project Boundaries](./project-boundaries.md) 与
[Greenfield](../product/greenfield-no-legacy-migration.md)。

| contract → implementation → injection → consumer | 职责 |
| --- | --- |
| typed mode + pure action decision → 现 internal/agent 的有界合同 → 当前应用/worker 注入 → title/image/quality 与 Commerce Tool adapter | 共用四种语义；只计算 allow / human-required / separate-authorization-required / deny，不拥有事实、队列、凭据或新审批生命周期。 |
| execution-mode subject/command/snapshot → 现 agentconfig 与 persistence adapter → 当前配置服务 → 我的智能体/执行确认 | 独立于 activation 的有界模式记录、准确 revision、配置与运行绑定；不拥有领域写入、私有交付或身份授权。 |
| code-owned action descriptor → 当前 Agent/Tool registration 与真实领域 adapter → 现 assembly → 运行边界 | 声明动作 ID/版本、风险、副作用、是否可自动审批、哪些检查和经济意图必需；不从模型参数采信这些字段。 |
| policy-scoped title decision/apply → 当前 Product Review owner → 现 Product Agent application → proposal completion | 重用现校验/事务/回执，增加明确的“策略授权采用”窄入口；不扩展原 tokenless human API。 |
| policy-scoped image result adoption → 当前 ImageAgent / Product Asset owner → 现 Temporal activity registration → 现结果发布 | 准确集合与 bytes digest、新鲜授权、唯一 effect；仍使用已有 workflow/slot/资产回执。 |
| private delivered-agent subject resolution → 当前 agentcustomization Delivery owner 的窄授权 port + 共用策略 → 现 private-agent consumer | 由实际 DeliveryID 证明企业、definition/version；不经过仅含 title/image 的公共 catalog 或创建伪 ENABLED 记录；质检各模式仍只诊断。 |
| safe projection → 现配置/任务 HTTP + BFF/Console → 现页面 | 模式、范围、策略采用/人工采用、拒绝或待确认原因；不展示假审批或成功状态。 |

实际允许执行 = 现身份/企业/成员/店铺授权 ∩ code-owned Agent allowlist ∩ owner 能力与 readiness ∩ 本次准确策略 ∩ 必需内容/预算/幂等检查。模式只是其中一项。

Commerce Tool 保持现 write/publish 默认拒绝；只有本交付明确接入现 owner、绑定策略和回执的准确动作版本可以开放。直接的标题/图片/私有入口也必须检查同一策略，不能存在未经过工具层的旁路。

## 4. 配置事实、运行冻结与撤销

### 4.1 默认值与写入权

mode 是 enterprise Agent 的有界配置，由现 agentconfig owner 保管，但独立于现 organization_agents activation。使用现 agent.configure 权限、命令幂等和 expected revision CAS。新安装未设置模式时，产品默认值为 default；无记录的模式 revision 为 0，读取不创建配置。HTTP 仅接受四个准确 enum；不接收任意权限、提示词、工具名单或预算。

准确 subject 只有两种：registered_agent，subject_id 是当前 code-owned Agent ID；private_delivery，subject_id 是当前 DeliveryID。公共 subject 由原 registration/activation owner 证明；私有 subject 必须由 agentcustomization 的窄授权 port 从真实不可变 delivery 解析 Organization/definition/version，并检查当前 read/use 或 configure 所需权限。definition/version 不接受浏览器自报。私有交付事实与公开启用关系不互相替代，模式记录不提供启用、交付、可执行定义或 grant。

在现 agentconfig feature module 下拟增加 GET/PUT /api/v1/workbench/agents/execution-modes/:subject_kind/:subject_id，使用独立 subject resolver，不调用仅支持 public catalog 的 agent_id 模板 handler。PUT 是 set-execution-mode，包含 mode、expectedRevision（规范十进制 string；初次为0）和原 Idempotency-Key；响应返回准确 subject/definition/version、mode/revision 与原命令 receipt。GET 要求该真实 subject 的当前 read 权限，PUT 额外要求原 agent.configure；两者均使用原 verified Organization/actor/member。该入口不列举未知/其他企业的私有交付。

默认模式适用于新运行；运行确认展示精确 mode/策略版本/模式 revision，绑定准确 ModeSnapshotRef ID/digest 与原 ConfigurationSnapshotRef。首期不增加任意 per-run 权限提升入口。私有质检沿原 DeliveryID 确认和执行流程消费 mode，不经公共市场启用；不在公开市场泄露私有交付，不建立第二份启用/交付事实。

没有 configure 权限的成员可使用其有 use 权限的企业 Agent，但没有因管理员设 full_access 而获得 listingkit.admin.write、Asset 写权限或其他角色。只读成员只能获得本来可读范围；缺失写权限展示明确原因。

### 4.2 不可变快照与变更

共用 typed ModeSnapshotRef（kind=agent-execution-mode-v1、UUID、SHA256 digest）属于现 agent 的纯执行合同，不复用/篡改原 agent-configuration-v1 kind。它指向 agentconfig owner 的独立不可变模式快照，包含 mode、code-owned policy version、mode revision、准确 subject/definition/version、verified scope/member 与原 context/binding/request key；其完整内容参与 digest 和原请求 fingerprint。标题 Request/配置快照、图片配置/plan/admission 与私有质检原 run/report 分别只持准确 ref，不建第二 Run。无 mode ref 或无法证明绑定不能准入新运行/效果；不将旧记录缺字段解释为新写权限。

运行确认提交 expectedModeRevision，以阻止确认到 Start 期间模式变化后静默扩大执行。Prepare 在现配置 DB 的一个事务内读取准确 mode 并冻结 snapshot；相同作用域/binding/request key 只保存一个意图。同 key 修改 expected revision、subject/definition/version 或内容是冲突；准确回放保留原 ref，不按当前模式重建。标题同库 Claim 与 config mode 锁共用原事务；图片/私有消费者不复制或迁移运行数据库。

模式变更不修改旧快照、不把已有运行升级。对于撤销/收紧，当前 mode revision 与冻结 revision 不同则阻止新副作用并显示“执行策略已变更，需要新运行”；既有只读查询、原准确回执读取和已发生 effect 的核实/结算不受此阻断。变更为更宽模式也不能激活原 UNKNOWN 或等待人工的旧操作。

每次真实副作用前重新核验当前 mode revision、适用公共 activation 或私有 delivery、原 actor/member/source/store 权限和领域前置条件。重放已完成准确回执允许读取历史结果，但回执重放不能触发 executor，也不能跳过读取的当前授权。

“撤销阻止新效果”以各原 owner 的新动作准入为界：真实 dispatch/业务 commit 前的最后一次有界策略读取成功，才可进入该准确 action 的原执行边界；mode 写入提交后才开始该检查的新 action 必须拒绝旧 revision。此前已获准并进入执行的准确在途 action 可能完成，撤销不声称跨 ProductAgentDB/ImageDB/AssetDB/Provider 瞬间取消它。其 ACK 丢失/UNKNOWN 仅核实原 action，不再准入新 effect、生成新 attempt 或扩大结果选择。UI 变更后分别显示阻止后续动作与原在途结果待核实；不增撤销协调器。

ImageRunAdmissionReceipt 保留原整运行计划/报价/预算确认意义，不是每次新 slot generation/Asset 采用的现时 mode grant。当前原准入回执优先 replay 的代码不能代替下一次 activity 的 mode/activation/delivery/live/source 检查。相关 race/ACK-loss 测试在实际 owner 的既有测试环境收敛。

### 4.3 持久化/安装边界

在现 agent_configuration schema/ProductAgentDB 内新增两个有界配置记录；不修改 private delivery 或复制 activation：

| 拟新增记录 | 最小字段与约束 |
| --- | --- |
| execution_modes | PK (organization_id,subject_kind,subject_id)；kind 仅上述两值，ID 原约束≤128 bytes（private 为 UUID）；mode 四 enum；revision positive bigint，created/updated actor/time。没有 activation、grant、budget、prompt、tool list 或运行状态。 |
| execution_mode_snapshots | UUID PK；完整 verified scope/member、subject/definition/version、context kind/id、request key、mode/mode revision（可0）、code policy version、bounded typed payload≤8KiB 与 SHA256；unique (organization_id,actor_id,subject_kind,subject_id,context_kind,context_id,request_key)。insert/read-only，无外键依赖 organization_agents。 |

跨库 private subject 没有伪 FK：每次由原 delivery owner 授权和核对准确 immutable identity。没有 mode 行时的默认0是新产品默认值，不能作为未交付 subject 的默认许可。第一次 PUT/Prepare 并发复用当前 PostgreSQL transaction/advisory-lock 能力，以完整 org/kind/subject 串行化；hash 只用于锁，实际 unique identity/CAS 仍用完整字段。原 config commands 保存 set-execution-mode 的完整 subject 与 before/after mode revision 的 typed receipt/fingerprint，沿原 org/actor/key 唯一键；不建第二命令 ledger。

mode snapshot 和实际 Start/Prepare 的 immutable refs 原样复制/重用，不重新 mint grant；配置暂不可读时新动作 fail closed，原历史读取/已发生效果恢复依各原 owner 授权。Review/图片策略 provenance 进入各自既有决策/操作回执，不建通用 approval_requests 表或 reconciler。

生产实现只定义明确空安装的 schema 和 serving schema/privilege 校验。当前已有数据、volume、业务 schema 不自动迁移/回填/删除；本次不对保留的统一试用实例执行安装或升级。运行新候选的隔离检查用空数据库和独立资源。若交付要开放到现有带数据实例，先报告准确 schema 差异和升级决定，不能通过 fallback/双读/双写或 destructive reinstall 实现。

## 5. 动作审批、确定性规则与授权证据

首期动作与当前 owner 一一绑定，禁止泛化成“RiskWrite 都可以”。code-owned descriptor 同时声明业务事实变更、受保护动作及审批资格；动作版本变更不能复用旧审批。

| 动作 | default | auto_approve | read_only | full_access |
| --- | --- | --- | --- | --- |
| 授权内读取/分析/质检/保存诊断或提案 | 允许 | 允许 | 允许 | 允许 |
| 当前标题候选采用 | 原人工 Review/Apply | 仅既有 title-only policy 通过、基线/来源/证据准确且无待人工项时策略采用 | 拒绝采用 | 当前 owner 检查通过后策略采用 |
| 已准入本地草稿写入 | 原确认 | 准确字段/规则支持自动审批才策略采用；无写合同仍不可用 | 拒绝 | 消费现 owner 合同与准确类型/写权限；待补全草稿可未绑 Store，官方适配草稿须实际 Store/规则 |
| 图片生成 | 保留准确计划/报价与硬预算确认 | 保留准确经济意图，不以模式代替报价确认 | 拒绝生成 | 已确认计划/报价/硬预算内可运行；不得扩大预算或重复生成 UNKNOWN |
| 图片结果采用 | 原准确集合人工批准 | 当前确定性检查不证明商品保真，首期仍需人工集合确认 | 拒绝采用 | 准确集合/来源/bytes/授权/版本等 owner 校验通过后策略采用，记录未经逐项人工审阅 |
| 外部发布、平台改价/下架、删除、提高预算 | 单独授权 | 单独授权 | 拒绝 | 单独授权 |

图片 auto_approve 的限制来自现真实验证能力，不造假“模型审核通过”；full_access 是用户要求的事先授权本地采用模式，其效果不得记为人工确认。原已批准来源/使用许可要求保持；模式不能把未经许可的来源转成已获使用许可。

本地采用入口保存：原 actor/member/org、exact run/config/mode revision、policy/action version、proposal/result digest、base/version/target、策略判定依据与原 idempotency key。Policy adopted 与 Human accepted 在任务和 Review 投影中可区分。不能伪造历史人工 Decision，也不能把“有配置权限”写成“已逐项确认此内容”。

人工确认与策略批准都必须由真实 owner 在提交前重读当前业务版本、source 许可、租户与 write 权限。标题策略采用复用同库 Review/Catalog 的准确 CAS/回执；图片采用复用 existing ImageAgent action/resultDigest/Asset publication 的原恢复路径，不让通用 runtime 直接写 Catalog 或 Asset。

策略采用不是可伪造的 allow 布尔值：标题拟增独立 PolicyAdoptionService 窄 worker port，从准确 run/ref/subject 加上 fresh identity/grant/source 解析 typed decision proof，再由 Review 的原事务验证并保存 policy provenance。现 NewExecutionCandidateService 继续只能 validate/intake；原 HTTP human Decide/Apply 的字段与权限保持，模型/浏览器不能传 policyApproved 或 impersonated human。Apply 可共享纯验证/事务行为，不能把原人工 Decision 写成已点击确认。

图片策略入口同样沿原 owner 构造合法 ImageSetCommand：准确 action、Choices、ExpectedHead、SelectionDigest、run/plan/result/config refs 与 typed policy provenance。选择由冻结计划定位与已完成合格候选生成，不取任意 latest、不自主补生成失败 slot、不把模拟 approve signal 视作人工证明。缺少完整合法集合或计划结果仍 partial 时保留原待确认/失败状态；完整访问不把未生成图伪装成已采用。

本次只保存“需要另授权”结果及原因，不实现新外部授权平台或解锁当前禁止的平台工具。未来外部动作单独授权仍需匹配准确身份、Store、动作、载荷/版本、有效期/预算和幂等身份；选择 full_access 本身不能替代该证明。

## 6. 失败、并发、UNKNOWN 与恢复

- configure CAS/同键异载荷和 snapshot uniqueness 复用现事务；并发模式修改不能覆盖彼此。snapshot 冻结与新运行 Claim 复用现同库准入。
- Title 策略决策与本地 Apply 仍以 Review/Catalog 同库事务与既有操作回执为最终事实；断线读取原回执，不创建另一份“已应用”事实。若沿既有分步状态提交，恢复只沿准确操作身份查询/重用原步骤，不重新生成候选。
- 图片使用原 run/plan revision/slot/attempt/action/result digest；结果丢失、partial 与 UNKNOWN 保留原费用、预留与效果，按当前 effect owner 核实。策略不持有另一重试队列。
- 授权撤销或配置变化阻止新效果；旧已发生效果的核实和可信结算仍由原 owner 恢复，不误释放或新扣费。
- 审批规则不可用或证据不完整返回安全待人工/不可用状态；不能默认批准。规则明确拒绝返回拒绝，不能转成用户“完全访问”绕过。
- 原请求大小、timeout/deadline、资源上限、模型路由/额度与内容校验保持。取消与“未产生副作用”不能相互替代。

## 7. 当前实施顺序与交付边界

遵守用户原顺序：原会话负责店铺内部同步与 worker、采集/添加自有商品 → 标准 Product → 我的数据/待适配/待补全草稿主流程；不重复派工。其最新用户决定允许先保存未绑定店铺的待补全草稿，之后配置实际 Store 再适配/校验/上传。该 owner 正补齐准入，本批次只消费其正式合同，不另建草稿 owner；没有实际 Store/官方规则时不显示官方已适配或可上传。

本会话先交付本设计与关联执行 Issue，进行一次必要的独立高风险边界检查。统一权限功能作为同一个后续 Delivery Batch，由一个 Writer 连续完成：
1. 共用模式/纯策略、配置 CAS/快照与正常 projection。
2. 标题既有 Review/Catalog 的准确策略采用及直接入口防旁路。
3. 图片各真实生成/采用边界与 Temporal 恢复、私有质检 adapter 接入。
4. 现配置/确认/任务页面消费者与一次最终交付检查。

切片是同一个主要分支/PR 的实现与验证边界，不另开四个 Writer 或建立审批平台。只有当前 main 主流程已完成到用户可检查的保存结果、上述设计准入完成且 UI authority 已核实，才开始正式权限生产路径。任何未接入的 Agent 或动作不能显示“四种模式已可执行”。

## 8. 风险匹配的验证与验收

开发先写能够捕获目标越权/旁路/错误采用的失败测试，再补最小实现。复用现 Go、PostgreSQL、Temporal test environment、HTTP/BFF 及 UI 检查，不建设专项 runner。

Must 验证：
- 四种 mode 投影真实且刷新读回；非法 enum、无 configure/use/write 权限、跨企业/私有交付访问拒绝。
- 同键不同 mode 冲突、revision CAS、快照不可变；mode/activation 撤销后直接入口与 worker 不能新增效果。
- readonly title 可产生提案但不能 Apply，readonly image 不能生成或发布 Asset；日志/诊断/已有结算不伪装成商品写入。
- default 保留实际人工确认；auto_approve 有通过/人工/拒绝结果；full_access 可准确本地采用但不可执行受保护外部/删除/预算动作。
- 标题原 base/source/hash/事务/idempotency 和图片 plan/quote/result digest/partial/UNKNOWN 不因模式变化被绕开。
- 已完成回执重放无 executor/provider 调用；旧等待/UNKNOWN 操作不因切 full_access 换键重发。
- 一条受控完整 title 流程与一条受控 image/质检流程走实际 adapters，区分内部 fixture PASS、真实模型/Store/付费调用 NOT_RUN。

当前设计 PR 仅验证文档/引用及独立设计评审，产品运行/用户试用为 NOT_RUN。实现交付必须提供实际运行入口、模式选择步骤、可用动作与不可用原因；实现者不签发用户验收或生产上线通过。

## 9. Legacy decision 与复用

Legacy decision: N/A for this design；production implementation only reuses current owners。
Reusable behavior: current Commerce Tool / Casbin authorization、agentconfig CAS/snapshot、Review/Catalog transaction、ImageAgent/Temporal effect recovery、私有交付隔离。
Current owner: 上述各 owner 不变。
Cutover/deletion condition: 不新增旧内部代码 consumer；遇到登记 RETIRE 的代码时按 EXTRACT/RETIRE 单独记录，不加兼容层。

参考 OpenAI 官方资料：
- [Codex sandboxing](https://learn.chatgpt.com/docs/sandboxing)：技术资源边界与审批策略是独立控制。
- [Agents guardrails and approvals](https://developers.openai.com/api/docs/guides/agents/guardrails-approvals)：审批与副作用边界绑定；Agents SDK/API 不自动继承 Codex Auto-review。

复用当前项目使用的 Casbin、Eino、Temporal、JSON Schema 和领域事务能力，不为了模仿 Codex 重建 OS sandbox 或另装 Agent SDK。OpenAI 资料是实现方法参考，不替代本项目产品决定和事实 owner。
