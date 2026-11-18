# Chat 通用任务创建与首批内容执行

Status: IMPLEMENTATION_READY。Design Basis: Independent Architecture。独立 Round 1 审核绑定设计 SHA `24c30d11be7adbd5b27c22e80e00c58c27a6d869`，无设计 BLOCKER；以下范围冻结后进入实现。
Execution Issue: [#808](https://github.com/qq550723504/task-processor/issues/808)，Parent #298；一个 Delivery Batch、一个 Writer、一个主要 PR。本文先于生产代码；独立检查明确 IMPLEMENTATION_READY 后才开始实现。

## 1. 产品结果、范围和权威

当前企业有商品与 Agent 使用权限的运营人员，在 Chat 选择准确已保存对象、语言和字段，表达内容整理/本地化目标，确认不可变方案和执行上限，创建唯一 BusinessTask；在任务中心查看原内容运行、建议及原审核/采用回执。

用户于2026-10-18明确批准本草案并要求先独立架构检查，达到 IMPLEMENTATION_READY 后实现，保留数据并注意其它会话。新决定扩展 #576/#588 的单标题创建范围，不撤销现有运行/费用/权限底线。#798/#806 仅是已交付的统一浏览依赖，不提供新的执行许可。

Product/Figma Authority：docs/product/final-ui-ia-authority.md；tg48P46SSXl6TBy9lZwg63，Chat427:2467、任务中心427:3005。本轮重读了Chat当前高保真context及截图，使用原会话/方案确认/任务详情结构与现有Card、Button、ContentSelectionFields、CollectionPicker、主题和资产；不新建工作台或品牌资产，不填原型示例数字。

首批能力为 `product.content.localize` → 现 `product.content.agent/v1.0.0`，支持 title、description、sellingPoints、seoKeywords 及规范目标语言。业务对象支持 `own_product` 和 `acquisition` 两个原内容上下文；自有商品不按其采集/导入/手工来源排除。至少一条非标题字段路径属于Must。

这两个上下文是当前明确支持的对象，不是来源的全集。来源provenance沿Collection/SRC事实保持；不能从未知kind推定采集。目标与执行能力使用显式类型和参数，不把公共任务模型写成“标题/采集专用”。其它能力以后按真实合同接入，不为未来能力预建动态Agent/Tool注册或编排平台。

Out of Scope：图片生成/报价/Skill包/图片面板（原图片会话#774/#805负责）、任意Agent/Tool/脚本、平台写入/发布/改价/删除、AI技能创作、新模型/provider/定价、第二Runtime/Review/计量/恢复owner、旧数据转换/回填/ID映射/双写、新验收runner。新Chat内容流程本批不增加Skill选择器：原内容owner对非标题Skill执行尚无合同，不能借用标题专用准入；模板、知识选择沿现有合同。

现行 #576 标题任务是当前合格owner事实，不是Legacy Register的RETIRE对象。其类型、准确已存回执与读取继续原合同；本批不重新开启共享实例的独立标题执行，也不把现有标题事实转换为内容事实。

## 2. 已核实的现状与根因

基线 main2dcbd80e3ea19448b1ec9c0d3d4dccb5b683e8e5。

| 现有位置 | 现状与本次必要变化 |
| --- | --- |
| internal/aiworkbench/contracts.go | MessageInput/WorkScope仅OperationID/TargetPlatform；需增加准确对象和能力参数选择 |
| app/httpapi/ai_workbench_application.go | planner/execution固定title定义、acquisition binding、title profile；需按明确能力注入原content application |
| integration/aiworkbench/einoplanner | 无工具Eino Planner只允许标题+三个平台；增加受服务器选择约束的内容规划输入 |
| persistence/aiworkbench/schema.sql | 两表kind只允许product.title.optimize、platform只允许三个平台；需当前模型的准确组合CHECK |
| task_store/ProjectTask | 原agent.Request已含ContextKind和ContentSelection，但Workbench确认及投影只按标题；需核对原完整内容集合 |
| product_agent_application.go | 多字段内容运行父装配仍要求全功能Asset依赖；新增有界contentOnly装配，不借full profile开启其它Agent |
| own_product_agent.go | ownSource/authorizeOwnBinding/ownStartBinding已有准确对象/版本、来源及回执语义；直接复用 |
| product_agent_knowledge.go | startTextRequest已有完整ContentSelection、配置/模式/Knowledge快照、profile及Quote；复用该入口 |

## 3. owner与调用方向

aiworkbench仍独占Conversation、消息、PlanningCommand、ExecutionProposal、BusinessTask和task-action回执；Collection/SRC/Catalog持有商品、provenance、准确版本；AgentConfig持有模板/配置/模式快照；原Agent持有Run/幂等/预算/deadline/UNKNOWN；Review/Catalog持有建议决策与采用事务；governed text/资源owner持有调用、费用和预留结算。

路径：新Chat严格选择 → 原aiworkbench.Message/T0 → Eino无工具规划 → 原owner重新授权的Proposal → 显式Confirm → 原Prepare/T1 → 原content Runtime/T2 → 原Review/原TaskProjection → #806任务中心。依赖方向不反转；domain不导入GORM、app或provider SDK。app只连接现有owner的窄操作，不新增中心化业务服务。

为首批提供静态、代码限定的content消费者；不实现运行时任意选择Agent的registry。现标题消费者保持其现行明确类型。执行、任务读取和动作必须按任务的准确Kind选择对应当前consumer，不能拿当前默认Agent解释另一种历史任务。

## 4. 公共选择与不可变方案合同

新内容消息沿原POST messages，使用严格的选择分支：

```json
{
  "content":"把描述与卖点整理成英文",
  "taskSelection":{
    "subject":{"kind":"own_product","id":"<item UUID>","catalogVersion":"3","publicationId":"<effective publication ref>"},
    "capability":"product.content.localize",
    "parameters":{"targetLanguage":"en-US","description":true,"sellingPoints":true}
  },
  "templateId":"<optional exact template>",
  "templateRevision":"<paired exact revision>",
  "knowledgeBaseId":"<optional>"
}
```

subject.kind仅上述两个已有上下文；id须UUID、version规范正整数、publicationId沿原有界标识。capability只能当前装配支持的值；parameters沿现ContentSelection严格schema，至少一个字段、规范语言，不接受价格/库存/属性/图片/任意JSON/隐藏字段。

subject的版本和publication来自原owner选择读取，不是授权证据。Admission在当前scope重新解析原binding，并要求对象/准确版本/原publication一致；不接受浏览器提供ProductKey、来源快照、组织、member、Agent、provider或费用。模型只返回原CLARIFY/READY、assistant_text和goal_summary，不得修改选择、语言、字段、对象或路由。

原title消息分支是现行明确合同，不做推测/转换：仅原OperationID+TargetPlatform形状适用于原title执行profile；新taskSelection不能与该二字段混用。两分支显式校验并用各自准确指纹；content-only profile拒绝新的title分支。旧键读取优先，不能把重试转换成当前新类型。新增JSON字段对原title摘要使用omitempty，保留原命令、Proposal和Task回执的准确字节/摘要。

WorkScope中内容分支只写taskSelection，不同时写第二份OperationID/TargetPlatform。T0冻结选择和准确Planner ModelProfile；规划完成前后从原owner重验同一版本，不自动追到新head。变更则拒绝形成可确认方案。

ExecutionProposal新增准确ContextKind、ContentSelection和ExecutionLimits；继续沿已有OperationID物理列保存本对象的原ContextID（不生成或映射ID），原ProductKey/CatalogVersion/PublicationID来自binding；新content目标固定为原owner的product范围，无需用户选择Marketplace。Kind、AgentID/Version、模式修订/activationEpoch、模板、知识revision、execution profile、全任务Limits均参与不可变digest。

ProposalCard返回准确Kind、授权后subject、语言/字段、原配置/模式、最大Steps/ModelCalls/Tokens/CostMicros/Currency/Runtime。最大额度指整条任务Limits，不把单次模型上限冒充任务总上限。规划消息自身沿原governed quote/用量；UI说明消息规划与确认后的业务执行分别计量，不承诺免费。

确认时当前Limits、profile、对象head、配置/模板/模式必须与方案相等；变化返回STALE/冲突，要求新方案，不静默扩额度。执行预算/报价/预留/结算仍按原owner，不创建第二报价账本。

## 5. 当前持久化与真实数据边界

保持ai_workbench原schema/原runtime角色、七表、FK、键/事务/列级权限，没有新表或跨ownerSQL join。

- commands.work_scope继续jsonb<=1024bytes，新增严格选择序列化；超过原预算在T0写入/dispatch前拒绝，不截断或扩大范围。
- execution_proposals当前模型新增可空context_kind、content_selection(jsonb<=1024)、execution_limits(jsonb<=512)。内容Kind必须满足非空且严格组合，title Kind保持原列/NULL字段/摘要。
- execution_proposals/business_tasks的组合CHECK增加product.content.localize + product目标；content context只为own_product/acquisition、agent准确content定义。原title/三个平台组合不松绑为任意字符串。
- business_tasks不新增Selection第二副本：原execution_request jsonb已保存准确Binding.ContextKind和ContentSelection及Limits；反序列化时必须验证Kind、Agent、policy/prompt、binding、配置/知识refs与Proposal及Task列逐项匹配。Task DTO subject与参数只从此已验证原请求投影。
- 所有新内容摘要包含ContextKind/ContentSelection/Limits；原title新增字段为absent，不重算历史hash，不回填或推断新subject。

修改的是当前合格owner模型的增量定义；不开发数据转换/迁移器/兼容adapter。现有schema安装/VerifySchema只针对准确当前模型：启动发现缺列、错误CHECK/owner/权限就fail closed，不能在HTTP/constructor自动ALTER或宽AutoMigrate。隔离测试只使用本任务新资源。

本批不读改保留业务库、不执行真实DDL。既有Conversation、消息、任务、提案、Run、凭据和费用记录保持在原实例原schema/程序中。后续部署若需当前owner的最小增量DDL，必须单独给出准确清单、保留/备份证据和授权；当前不得为交付自动执行。若真实现存事实无法安全读取/恢复，先报告具体冲突，不增加legacy迁移或删除历史。

## 6. T0/T1/T2与失败、重试、并发

| 边界 | 本次变化与保持 |
| --- | --- |
| T0消息/规划 | 原组织+actor+key锁与Conversation序列事务；冻结严格selection及profile。所有原owner I/O在锁外，PlanPreparer仍纯函数。只一次规划调用；同键不同选择冲突 |
| 规划调用 | 原governed Executor / invocationID / Ledger / member额度和deadline；CLARIFY不生成执行方案；INVALID/UNKNOWN沿现PlanningState，未知不重调 |
| Prepare | 当前权限/对象/config/mode/profile/Limits复核，复用startTextRequest选择完整内容字段；可能按原owner创建准确、键控配置/Knowledge不可变快照，不创建Run/扣点/发布。快照可在后续T1失败后保留，不删除补偿 |
| T1确认 | 原serialized LookupTask/ReplayOrFail/Confirm；同键、同Proposal重放原Task；后续USER消息/归档/不同payload仍按原锁与最新序列拒绝。PreparedRequest须逐项等于冻结Proposal，单库原子Task/回执 |
| T2执行 | 先原store.Lookup准确binding/key，已存在必须完整Request相等；未Claim重新当前授权和准确原binding，再原StartWithClaimAttempt。响应丢失沿原Task/Agent回执核实，不发新key |
| task动作 | Start/Resume/Review沿原task_action_receipts，准确Revision/feedback/payload不变。未提交先拒绝、owner可能已开始则UNKNOWN；重复核实回原owner，不造reconciler |

新内容请求的request.PolicyVersion/PromptVersion/ContentSelection必须是原content合同，不能走startRequestWithProfile的空字段title形式。原owner finishTitleCandidate虽为历史函数名，已按真实policy处理内容；复用其有效行为，不复制Review或政策采用。

对象有新head、撤权、成员变化、模式/config变化分别阻止新的效果；已提交回执按原读取/归属权限核实。Receipt优先于新head与新profile，不能因为当前配置改变而丢掉已存在Task或重派原调用。若当前执行consumer未装配，只读原Task元数据和明确不可执行，不尝试其它Agent/default consumer。

## 7. content-only装配与准入

新增明确可信 `productAgent.contentOnly`（默认false），与contentTitleOnly互斥。只允许ContentTextPolicies、当前有界Run/Review/Point owner和合法独立池；拒绝AssetDatabase、title/materials/selection/diagnosis执行policy。沿已抽取的共同依赖直接buildContentProductApplication，不先buildTitleProductRuntime；只安装原content/Review/必要资源路由，不能把未构造的title runtime或供应链标题优化器接入。

contentTitleOnly保留原标题字段限制和storage-only组合；本次不把它改成宽松开关。新contentOnly不要求title-only字段限制，但保留原完整ContentSelection/配置/模式/预算验证。其Asset缺省不应改变图片/POD/供应链等其他owner自身的独立Asset准入。

新增可信 `aiWorkbench.contentTasks`（默认false），仅在ConversationOnly=false且实际contentOnly runtime已装配、Collections对象读取已注入、同一逻辑RunDB但独立ai_workbench_runtime池/角色、当前org明确allowlist、独立Planner policies有效时可启用。两布尔组合与依赖均在manifest validate → runtime开池 → HTTP组合三层核对。storage-only/旧title full组合维持原语义；不默认启用新profile或借TitleReadiness展示可用。

无工具Planner内容版本为ai-workbench-chat-plan-v2，沿原OperationAIWorkbenchChatPlan和原decision schema，profile明确匹配版本；原title规划v1保留自身版本与摘要。内容执行继续原OperationProductContentDecision、原ContentPrompt/Action version。Planner与execution可用性按对应真实consumer计算，不借标题路由作准入。

现workbench-context增加当前执行Kind/readiness，或等价的严格能力描述，只有实际装配才有content能力。UI只显示该范围。原TitleReadiness字段仍只表达原标题路由，不改名后谎报旧consumer。新token/凭据/provider配置不由浏览器或模型输入。

## 8. 具体接线与实现位置

| 合同 | implementation / injection | consumer |
| --- | --- | --- |
| Subject+Capability+ContentSelection | internal/aiworkbench严格类型、指纹与Proposal校验；persistence当前列和CHECK | 原Message/Confirm/TaskDTO；不放业务规则到app |
| 原对象授权 | ownSource/authorizeOwnBinding；currentAcquisitionBinding/authorizeAcquisitionBinding，content application的原freshIdentity | Planner Admission、buildProposal、Prepare、Start/action及授权详情 |
| 内容请求 | product_agent_knowledge.startTextRequest、原ContentSelection和config.Prepare/Match | workbench content execution；原runtime/Review |
| Planner | integration/aiworkbench/einoplanner内容Prepare + 原governed模型；app注入内容选择/对象校验和当前模式 | 原Service.Message，不新建AgentRun |
| contentOnly/contentTasks | runtime/currentapplication product_agent/ai_workbench、runtime开池与httpapi组合/route admission；原application共同构造 | 原Content runtime与Chat，不触图片组件 |
| Task read/action | 原task_store.ValidatePreparedTask、AuthorizedTaskProjectionReader、ProjectTask及app action consumer | #806 TaskBrowser/TaskDetail/原Review |
| 选择/确认UI | 原ChatPage，用现CollectionPicker/对象source读取、ContentSelectionFields、原template/knowledge字段；Zod严格schema/BFF | 原确认卡片，原挂起消息/确认key恢复与切企业锁 |

现有跨域授权和幂等由各原owner持有；不通过HTTP自调用，也不包装已RETIRE Service。继续使用仓库已有Eino、governed text、Zod/TanStack Query，没有新第三方框架或重复runtime。

## 9. 状态、审核、权限与资源边界

保留当前RUNNING/WAITING_CONFIRMATION/COMPLETED/ERROR/PAUSED五种投影，不新增Task状态机。内容Agent HumanReviewRequired必须原Validation.Valid、无StopReason、完整选中字段集合合法、原context citations和content policy一致；不能沿标题len(Changes)==1判断，也不能只取首字段。Agent自身可审核性与Review/政策采用仍由原owner判定。

Review的pending/accepted仍待处理，applied/rejected沿原已完成结果；没有Review且不可读/配置缺失不伪造完成。四模式沿原内容合同：read_only不采用；auto_approve多字段仍人工；full_access按已批准当前业务授权和原政策proof，不虚构人审。内容语义/译文质量不是deterministic验证结论，沿原Accepted Risk，不新增审批模型。

身份/组织/member授权来自现fresh context；Chat权限不替代Collection/source/Agent/Review/Catalog权限。选择、方案卡、确认、每次新effect重验；切企业/撤权/迟到响应清除敏感详情。原title/内容历史仍按其准确类型授权，不扩企业可见范围。

沿原正文<=8KiB、最多50条/64KiB上下文、Planner输出<=16KiB、work_scope<=1KiB、execution_request<=8KiB、原Agent/Review64KiB生命周期预算。原GET15s、message/confirm/action2min，Agent<=120s/16steps/8modelcalls及当前更小Limits；不得通过新能力增加硬上限。超界写前拒绝，不截断、跨库补偿或静默降级。

## 10. 必要验证与交付

TDD只针对实际变化：subject/版本/语言/字段严格组合、未授权/未知kind拒绝；至少自有商品非标题内容+源商品两条路径；T0指纹和已存title摘要不变；准确新schema/CHECK/有界角色；T1重复/并发确认、响应丢失、归档/后续USER；配置/模式/member/profile/Limits变动；原UNKNOWN不重调；完整多字段候选/Review/采用；各profile不误挂title/Asset或图片；原Chat挂起消息/确认恢复、切企业锁和任务中心链接。

复用原Go/隔离PG、受控Eino HTTP模型fixture、Vitest/Zod和现浏览器fixture，不新建runner/专项验收平台。必要本地目标检查后才形成稳定候选完整CI；最终Reviewer检查完整用户路径和真实diff，修复只增量复核。

交付一个主要PR、正常启动/入口与明确profile/schema前置；隔离受控provider路径不冒充真实生成。默认不合并、不部署、不改真实库/数据/凭据/账务/配置，不调用真实模型；图片实测窗口由原会话保留。用户或指定验证者验收，不由实现者/CI/AI review签发产品验收。

## 11. 评审状态与Legacy

独立 Round 1：IMPLEMENTATION_READY，准确设计 SHA `24c30d11be7adbd5b27c22e80e00c58c27a6d869`，无设计 BLOCKER。Reviewer 为只读 task_center_final_review。Must、Out of Scope和Accepted Risk以上述范围冻结；仅符合AGENTS明确后果的新BLOCKER重开设计。

两项 IMPLEMENTATION_TEST（合并前收敛）：手动 Review 动作沿原内容 route 赋完整 ContentSelection；自有商品 subject 使用 source DTO 的当前有效 catalogVersion/publicationId，不能误用 originalVersion/originalPublicationId。覆盖非标题多字段审核、Apply 后新任务冻结当前有效 head、陈旧 tuple 拒绝。

Legacy decision: N/A（当前合格Workbench/Content owner演进）。没有Legacy Compatibility、Task-first Product、旧Dashboard/平台Workbench、fallback/双写/ID映射或数据迁移；触及登记旧代码仅EXTRACT/RETIRE。

## 12. 实现接线与交接入口

商品选择复用 CollectionPicker 和原 Collection 列表。只读 `GET /api/v1/workbench/chat/subjects/:context_kind/:context_id` 将显式 own_product/acquisition 身份交回原 ownSource/currentAcquisitionBinding，返回当前有效 subject tuple；无副作用、无新持久化，不能接受浏览器 ProductKey。BFF 继续核对当前账号/企业和严格响应，未知 kind 拒绝。

在获准的当前 schema 与凭据/预算已由各原 owner 准备的独立环境，正常启动命令为 `go run ./cmd/current-application -config <absolute-private-manifest.json>`，前端沿原 `web/listingkit-ui` 启动命令。manifest 需显式启用 productCollections、productAgent.contentOnly=true、contentTitleOnly=false、aiWorkbench.contentTasks=true、conversationOnly=false；产品/Review与Run库分离、Chat为同一逻辑Run库的独立 ai_workbench_runtime 池，限定组织 allowlist。内容 policies 仍用原 content prompt/action 版本，独立规划 policy 使用 ai-workbench-chat-plan-v2 / ai-workbench-plan-decision-v1。不能复制测试凭据/定价至真实环境。

操作入口 `/workbench/ai/chat/new`：创建会话 → 从我的数据选商品 → 选语言与字段 → 填需求 → 核对对象版本、内容、执行权限与任务总上限 → 确认 → `/workbench/ai/tasks` 查看原运行/审核。描述与卖点等非标题字段可选；图片及尚未接入的对象类型不伪造可用入口。原 UNKNOWN 和未核实消息继续使用原键恢复。

现有保留实例仍使用其原 schema/程序。本 PR 不切换运行、不执行真实 DDL、不生成新业务数据，也不调用真实 provider。当前模型需要的三列和两个组合 CHECK 尚未授权应用到保留库；直接把新程序指向缺少该模型的库会启动失败。这一部署前置不能通过自动迁移或清空数据规避。受控 HTTP/模型 fixture 与前端组件测试仅是开发证据，浏览器实测、真实 provider 和用户验收须如实保留 NOT_RUN。

## 13. 冻结合同内的实现修正

本节说明原合同的准确消费方式，不改变上面的事实 owner、状态、事务、权限、预算或 UNKNOWN 协议；实际运行授权及结果维护于 Issue #808。

- 原 `FieldChange.Value` 始终为字符串。`selling_points` / `seo_keywords` 的完整 JSON 字符串数组须编码在该字符串内；模型不能直接返回数组类型。固定提示提供可严格解析的完整多字段示例，显式映射 ContentSelection 的 camelCase 与内容字段名。仍使用原严格 JSON 解码，未知字段、重复键、错误类型及不合法引用不做兼容转换。
- 结构失败的本地诊断只记录封闭错误类别、已知合同字段路径、调用及输出摘要；不记录模型正文、未知键名、商品文本或凭据，不改变原失败及用量结算。
- 会话读取与原消息收据读取通过当前 scope + conversation + proposal 查找原 BusinessTask，返回 `confirmedTaskId`。该值来自原 T1 事实，不能以 sessionStorage 的 taskId 替代；浏览器只恢复未结束操作键。当前商品/配置已变化或执行 consumer 不可用不抹去已确认任务链接，详情仍按原权限读取。
- 一提案只确认一个 BusinessTask：原 scoped-key 回放优先；不同确认键在原 Conversation 锁内检查已存在提案任务并拒绝再次创建，不增加新事务、锁、表或恢复 owner。历史冲突事实如被发现须 fail closed 并报告，不自动合并或删除。
