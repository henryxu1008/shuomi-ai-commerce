# 图片事实检查与非执行性建议：C2-1 / C3-1

状态：IMPLEMENTATION_READY / IMPLEMENTING。Design Basis：Independent Architecture。
本文是 #134 / #128 的冻结设计基线，不是运行或产品验收。
产品范围及 D2 已获产品认可；准确设计 HEAD e498c0c1e87203341c25df79ed58c5cefd228999 的
[R1 独立架构复核](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1076#issuecomment-6577193191)
已达 IMPLEMENTATION_READY。实现沿同一 PR 和唯一 Writer 推进，不修改保留环境或数据库。

## 1. 用户结果与产品依据

商品运营在现有商品智能体/图片入口查看已有图片事实、缺项及下一步建议；
在现有平台资料页查看当前已保存 Target 的真实规则和已批准素材的位置建议。
建议不会准备任务、生成、保存、批准或发布。

依据：

- [C2 产品决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6575806435)：C2-1 本阶段，C2-2 后续。
- [C3 产品决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6575617237)：C3-1 本阶段，C3-2 后续。
- [设计候选及现状证据](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6576072576)。
- [候选/D2 认可及 design-only Draft PR 授权](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6576601045)；
  该认可不包含独立架构复核、实现、合并、部署或操作保留运行。
- [Commerce Tool](commerce-tool-foundation.md)、[Agent 运行合同](product-agent-runtime-contract.md)、
  [平台资料 Agent](platform-listing-materials-agent.md)、[图片 Agent](product-image-agent.md)。
- [当前系统边界](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
- [UI/IA Authority](../product/final-ui-ia-authority.md)：Figma tg48P46SSXl6TBy9lZwg63，非归档页面 31:463。
  复用已有商品智能体、资料结果页和图片入口，不新增页面/导航、独立工作台或审核流程。
  本文复用既有 Authority 记录，没有重新读取 Figma 或宣称视觉验收；布局/交互若实质改变须再核适用节点。

### 1.1 Must

1. 只使用原 owner 已授权、实际读取且准确绑定的 Product/source/Asset/Target/规则事实。
2. 区分已有事实、确定冲突、未知与不适用；事实未知不伪装成不合格或通过。
3. 输出有依据的图片检查及非执行性动作建议，具备实际 Tool/观察和现有用户结果消费者。
4. 通用图片不增加 Store 前置；原生图片不能绕过 Store/source/Target head/平台规则准入。
5. 保留原 Registry、权限、有限预算、审计、确定性重验及保存/批准/发布边界；不提高调用或费用上限。
6. 选定集合仍须满足原 E2 组合证明；不把不同 Agent/binding/SHA 的局部 PASS 拼成全体通过。

Should：复用原 reader、纯校验、错误分类和 UI/BFF 组件，只补薄适配与必要投影。

Out of Scope：C2-2 额外视觉模型；C3-2 独立候选类目查询；图片下载/采集、生成/修图、
Prepare/Run/Confirm/Approve、自动 Apply/publish、额外 planner/reviewer、模型/价格变更、
通用规则/诊断/编排引擎、新事实源/审批/报告/运行 owner、历史迁移/兼容、部署和真实操作。
C2-2 继续 DEFERRED / NOT_IMPLEMENTED；C3-2 的[单独冻结设计](listing-candidate-category-rules.md)
已完成自身 R1 并获后续实现授权，在 #1077 维护证据；仍在本文与 #1076 冻结范围之外。
两者不作 #1076 本阶段 Must，不记该 PR 的实现 PASS、风险接受或永久取消。
C3-2 的规则覆盖与版本执行资格依据自身后续批准，不继承本文 R1 / D2。
没有新增 Accepted Risk；权限隔离、证据、资金和副作用安全底线不变。

## 2. 当前能力与最小差异

| 项 | 当前能力 | 本候选的差异 |
| --- | --- | --- |
| Asset | exact reader、批准库存、角色/尺寸/位置及作用域；product.asset.inspect v1 | 复用事实和 owner 验证，增加纯事实检查/建议的薄 compute 投影，不改旧 Tool v1 的含义。 |
| Product Agent | 原 PoC 注册 canonical/source/asset/readiness；审核写字段受限 | 新准确注册显式加入图片事实工具，增加只读建议结果展示；不把图片动作写入标题 Review。 |
| 原生资料 | materials inspect v2 已披露准确 Product/Target/rules/批准资产/当前诊断 | 复用现数据与图片选择/摆放，增加有证据状态的建议投影和精确新版本，不塞入不同 Binding 的通用 Tool。 |
| C3-1 | QueryRules/RuleReader、Store 授权、saved Target 规则和原换类目流程 | 以现链路完成本阶段合同/证据映射，不为延期 C3-2 新建查询 Tool。 |
| UI | 商品智能体结果；materials before/after/Proposal.Images/unresolved | 补结构化 checks/suggestions 展示及原图片入口提示，现有字符串缺项列表不冒充完整动作合同。 |

当前没有一套可以直接包成无副作用 Tool 的完整图片准备入口。
PrepareImageConfiguration / InitializeRun 会写快照和任务；只能复用其 owner 中必要纯行为。
原 goods 图片要求/校验继续归 Marketplace，不复制常量、图片计划或平台 SDK。

## 3. Owner、接线与消费者

| 合同 → 实现 → 注入 → 消费 | 责任和边界 |
| --- | --- |
| 图片事实/纯建议 → 当前 Product Asset owner 的窄纯计算 → App 中注入现 Catalog/ApprovedInventory reader → 同一 Commerce Registry 的薄图片 Tool | DTO/纯计算不导入 Agent framework、DB 或 provider；原库存/批准为唯一事实 owner。 |
| 原生规则/位置检查 → goods 原图片约束与 diagnostic → Supply/materials 注入原 Store/规则/Target reader → materials inspect/固定 Validator | 仅当前 SHEIN 美国站 saved Target 消费，不外推所有平台；建议没有规则事实所有权。 |
| 记录的 Tool 输出 → 原 Agent Run/control/checkpoint → BFF 有界只读投影 → 原 Product Agent panel / materials page | 沿原 Run 存储，不新建建议表/报告；页面展示原观察，不用当前值补齐历史。 |
| 用户后续选择操作 → 原素材/ImageAgent/Target 入口 | 原入口重验参数、权限、价格/预算与原键；建议及导航没有执行效力。 |

通用侧只接受原 Product Agent 的服务端 Binding，不新增模型自由参数或把 selector 塞进
GoalSummary/ContextID。保留原明确的 target_platform 作用域，但没有 Store/Target 时不评价官方发布规则。
原生侧保持 listing-target-record/SHEIN 的准确 Binding，模型仍只获得其 inspect Tool；
不修改 Action/Gateway 以承接 C3-2，不自动给 contentagent 或所有 Agent 增加 allowlist。

## 4. 事实与建议合同

### 4.1 事实输入

- Product/source：ProductKey、publication、有效 Catalog version、适用 apply receipt/来源谱系；
  可信组织/成员由当前身份与 reader 提供，不来自模型。
- Asset：原批准库存的 AssetID、角色、已存真实宽高/位置及准确库存/批准身份；
  原始候选、生成候选与批准资产不互换。没有对应证据的字段为未知。
- 原生上下文：Target ID/revision、Store/site/category、准确规则版本/摘要及原 image observations；
  商品/来源/库存作用域和 Target 关系由原 reader 校验，不能复用另一版本库存。
- 通用上下文：只做现事实存在性/合法性及 owner 已定义的技术约束；
  没有具体平台规则时 platform checks 为 not_evaluated，不写“符合平台要求”。

URL、文件名、图片用途标签不能证明商品内容、构图、文案、保真、侵权或审美。
本阶段不调用下载/看图/采集来补事实，不把建议材料提升为批准状态。

### 4.2 只读 ImageAdvice v1

候选 schema 标识为 image-advice-v1。字段语义：

| 字段 | 语义与来源 |
| --- | --- |
| binding | 该次准确 Product/source/版本/平台；原生侧另含 Target/revision/Store/site/category/规则摘要；模型不提供可信值。 |
| evidence | 实际 recorded ToolRef/CallID 与观察摘要；Advice 的稳定语义摘要包括 schema/绑定/事实/规则与检查建议，不含可变读取时间。 |
| facts | 仅已披露资产身份、批准状态/角色/真实尺寸状态与允许的位置引用；不含凭据/内部权限或任意来源 URL。 |
| checks | 稳定 code、资产/位置和事实/规则引用、known_satisfied/known_conflict/unknown/not_applicable、简短理由；这些是视图状态，不修改原公共 DiagnosticResult 或业务状态机。 |
| suggestions | 固定 kind：verify_fact、select_approved_asset、check_placement、open_existing_images；对应 checks/evidence/合法资产位置，附非执行性理由。 |
| unresolved | 未评价/未知及原因；权限/依赖故障、未读取与确定缺失分开，不伪造完整性。 |

checks.code 的候选闭合集合为 dimensions_known、dimensions_unknown、dimensions_conflict、
placement_conflict、required_image_missing、approved_assets_unavailable、rule_unavailable、
platform_not_evaluated、owner_image_issue。后者仅保留原图片诊断引用，不从原未分类消息推断冲突/合规。
known_satisfied 只证明对应的明确检查，不代表图像内容、全部发布条件或产品通过。

select/check 只能引用本次确实披露的资产和允许位置；open_existing_images 是服务端已知入口种类，
不是模型 URL、actionID、任务参数、批准令牌或预算承诺。不能创建新资产或改变归属。
角色/尺寸/位置仅为图片资料，不成为商品内容/材质/颜色等文本事实的证据。
标题/内容 Validator 的字段与来源限制不因新增 Tool 而放宽，模型不能以图片标签推断商品属性。

只有 owner 明确返回当前作用域已授权的空库存，才能表达“没有已批准素材”；
未准备好应表达该状态而非伪造完整空集合。权限、作用域不匹配、not_found/依赖错误继续原错误路径，
不能为了给建议将错误吞成成功读取。

已有选择/摆放的正式候选仍只有 materials.Proposal.Images + EvidenceIDs，
由原 ApplyProposal → goods.AssessOfficial → 固定 Validator → Target owner 重验。
建议不加入 writable Candidate.Changes，不新增 DomainProposalKind，也不让标题 Review 承接图片保存。

### 4.3 两处必须保留的区别

1. goods/images.go 的 image_dimensions 可能表示尺寸已知不满足，或尚未核实真实尺寸。
   Advice 结合真实输入区分 conflict/unknown；原发布门禁仍保持原结果，未知不能放行。
2. Supply ImageSetTargetRules 的计划位置校验使用 1024×1024 占位输入。
   不能将它当原图尺寸/图像事实，也不能调用整个 PrepareImageSet 来获得无副作用分析。

### 4.4 资源和 Schema

- image-advice-v1 的七项顶层字段全部必需，受控对象 additionalProperties=false。
  通用 binding 不含原生 Target 字段；有原生上下文时其完整绑定字段必须成组存在，禁止半绑定。
  identity/版本沿原合法格式且字符串最长 128 字符，code/kind 为封闭枚举，理由最长 1024 字符，
  不提供任意扩展 map/URL 或执行载荷；checks/suggestions 引用必须在本次 facts/规则/检查集合中解析。
- 薄 Tool 输入复用原 Asset inspect 的严格 product_key/catalog_version/target_platform 形状；
  Binding/identity 由服务端映射，publication/原生上下文不让模型注入。
- 新 Tool 与建议输出各不超过 512KiB；总 Run/state 及 BFF 继续各自原上限，不提高原限制。
  原 reader 的输入字节/数量界限先执行；每项检查/建议引用均在已读集合内，字符串采用既有 identity/text 上限。
- 数组最多为原输入允许数量与 10000 的较小值，且受总字节界限；不先无限 hydrate/normalize。
  输出无法完整容纳时返回明确边界错误/未覆盖，不静默截断后宣称完整。超限回归为实现义务。
- 通用薄 Tool 复用 Asset 的 3s 总时限，fresh principal/preflight 和 executor 共用 deadline；
  原生 inspect 仍用其原时限。错误沿原 taxonomy，不把权限/不可用错误变成空库存成功。

## 5. 权限、预算、状态与持久化

原 Registry 处理 exact Definition/版本/schema、scope/resource 授权、审计、timeout/retry。
通用侧保留原 LocalAgentWrite 与来源/库存读准入，不因“纯读取”自动降低权限。
原生侧保留原 member/source/Store/head/规则/库存检查及最终 fresh authorization。

纯 compute 无模型调用、Child AI Invocation 或图片资源占用，声明 unmetered + SideEffectNone。
外层 Agent 的 governed 模型决策仍真实计量；已有 ModelCalls/token/cost/deadline 不提高，
没有额外 planner/reviewer 或隐藏 repair。新增可选 Tool 不保证预算内总能完成：不足按原合同停止/中断，
不能借建议开第二次调用或图片预算。冻结为两次的内容流程保持原 canonical→propose/interrupt，contentagent 不新增本工具。

不新增数据库/表、配置快照或 durable business fact。纯建议随原 recorded Tool output 保存在原 Run；
commit/checkpoint 仍原原子边界，失败不产生可采用记录。页面不另存建议作为事实。
原 UNKNOWN、pending invocation、预留/核实和唯一恢复 owner 不变；纯分析重试不得转为执行原图片请求。
相同键不同 payload、响应丢失、取消和恢复继续原 Request fingerprint/Claim/CAS；
没有新 Saga、outbox 或 reconciler 的需求。

授权撤销/跨企业/来源或 head 漂移由原 fresh reader 拒绝，不输出缓存旧资料作为当前成功；
历史读取仍走原历史授权和准确记录，不能把历史允许读当作可执行/可采用。

## 6. 准确版本与历史效力（D2 已产品确认、R1 已冻结）

下表为已冻结的实现版本，不是对保留环境的运行切换授权：

| 合同 | 候选版本/处理 |
| --- | --- |
| 新图片薄 Tool | product.image.analyze@v1.0.0，Owner=product.asset，Risk=compute，SideEffectNone，unmetered。 |
| Product Agent 新准入 | product.title.agent@v2.0.0，原四只读工具加新 Tool；PromptVersion=product-title-agent-v2 / product-title-agent-knowledge-v2，保留原 title-review-v1 领域校验策略。准确注册/配置快照同源，不原位扩大 v1 定义。参数和业务写字段不扩展。 |
| 原生 inspect/Agent | listing.materials.inspect@v3.0.0 / listing.materials.agent@v3.0.0，listing-materials-policy-v3 / listing-materials-prompt-v3；仍单 inspect Tool。ProposalKind/action/config 参数语义保持原 v1，不追加执行型动作。 |
| 当前 content/image Agent | 不改变其 registration、配置、allowlist、预算或执行合同。 |

D2 产品决定：新执行/claim/resume/adopt 只接本次新定义；旧版本历史和已提交回执只按准确原格式只读，
不跨版本续跑/采用、不迁移、不补新建议、不重写历史。保留 #34 既有 D1 的 v1 只读效力；
D2 扩及本次被替换的旧准确定义，本轮已取得上方单独明确认可，不能假定 D1 自动覆盖所有后继版本。
这是版本执行资格/历史效力的合同变化，影响旧版本 resume/adopt；不能以“没有新增状态机”声称它不影响恢复。
原 Claim/CAS、原键/预算和 UNKNOWN 核实机制保持；D2 已获产品认可并经 R1 独立架构复核。
现存未终结运行的处置不自行执行；若必须跨版本继续或新增兼容读取，停止并报告，不能加 fallback。

读取旧运行时若没有 image-advice-v1 输出，UI 明确该项未提供，不使用新算法/当前规则补造旧建议。
新运行的历史展示只解析原 recorded payload；损坏/缺失保持不可用和原证据，不刷新、重算或付费重试。
展示/采信建议前检查准确 ToolRef、CallID、recorded audit、完整 schema 与该 Run Binding/摘要一致；
record_failed、缺失或不匹配输出不能成为新候选/采用证据。
准入与 UI/store 对版本的检查必须同源；本设计不授权部署、操作任何保留运行或变更其状态。

## 7. 用户入口与实际消费

通用侧在现商品智能体结果组件增加只读“图片事实与建议”结果，沿现步骤/记录找回；
资料侧沿原 materials 页的依据、差异与待核对区域，保留“尚未保存”、原模式/权限与保存按钮的含义。
两侧 DTO/BFF 都消费有实际观察来源的同一版本结果，不以未验证的模型 unresolved 字符串充当可信检查。

需要补素材/生成时只提示原图片入口，不在打开页面/链接时调用 prepare、confirm、save、approve。
服务端从当前合法上下文选择入口；当前来源已不可用或企业/记录不匹配时不自动带入旧操作。
真正执行仍由原入口确认素材、位置、参数、权限、价格与预算，并用原键核实/恢复。
本阶段的非执行性提示不会自动采用或替用户选择新操作。

## 8. E2 组合证据与必要验证

C3-1 原读取/校验与素材摆放可做合同映射；C2-1 的投影、Tool/实际消费者和 UI 仍确需实现。
C2-2/C3-2 不作 E2 前置，其余 Must 不减少。E2 继续要求同一 Registry 合同、真实领域 owner、
governed C1 提案、确定性重验/有界 repair 和原人工审核交接，不把所有工具开放给每个 Agent。

| 证据 | 可复用 | 尚缺 |
| --- | --- | --- |
| Registry/source/canonical/asset/readiness、C1 | #133/#272/#502/#1069 的原代码/复核/准确 CI，按原 SHA/范围 | 不直接证明新增 C2-1。 |
| C3-1 和原素材放置/重验 | #666/#1064、materials inspection/validation、图片 placement 的局部依据 | 按本阶段 saved Target 范围对齐，不外推所有平台或保留实例。 |
| C2-1 新增边界 | 原 Asset/Marketplace/Agent/UI 测试可扩展 | 未知与真实冲突、证据/版本/权限、薄 Tool/实际消费、无执行副作用与版本 D2 的增量验证。 |
| E2 一致组合 | 原 Fake Agent、真实 owner integration 和有界 repair/human-review 基础 | 扩展同一隔离场景的实际 Registry/owner/模型协议：新图片观察→候选→repair→固定重验→Review，以及原生当前规则→候选→原 Target 门禁。准确 HEAD 的执行结果与独立最终复核维护在 PR #1076，不代替产品验收。 |

实现阶段仅扩现 tests：
- 事实未知/冲突、错误资产/位置、过期规则/head、越权与超限；原写入/批准/预算断言保持。
- Fake transport 的实际 Tool/观察准入、预算下 propose 或 interrupt、原 repair 后固定 Validator 重验；
  合法消费者分别绑定正确对象，不伪造一个全权限 Agent，不拼多份报告为单次组合通过。
- 原 UI tests 确认建议显示且不 prepare/generate/save/approve；新准入/旧历史不能跨版本执行。

参考现有 materials inspection、validation、public_diagnostic、proposal/commit、goods image_requirements、
Agent budget/engine、HTTP integration 和 product-agent-panel/materials-page 测试。不新建 runner/人工验收 session。
原代码未变证据可以复用，但不得标成新 SHA 直接执行 PASS；历史失败/未知/跳过保留。

## 9. 准入与交付停止点

冻结设计已完成 R1 独立复核；用户随后明确授权同一 Writer 实现及原 Reviewer 增量/最终工程复核。
R1 的 IT1 在原 Supply/materials 读取路径保留“未准备好”和“明确空库存”的区别，
经离线回归收敛；不新增库存 owner 或状态机。实现、自检、准确 HEAD CI 和最终复核状态见 PR #1076。
工程最终复核/交付仍须完成，不因本设计 Ready、开发测试或 CI 而自动授权合并、部署或关闭父 Issue。

文档完成、CI 或独立架构复核均不代表产品通过。运行、真实保存/恢复、浏览器、视觉/模型效果和产品验收 NOT_RUN；
缺失证据 UNKNOWN。#47 当前1.0.2 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT 保持。
不部署、不操作保留实例/数据/会话、不真实调用模型/平台或发送邮件短信；Issue 关闭和合并另有权限边界。
