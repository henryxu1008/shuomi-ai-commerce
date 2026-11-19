# Chat → BusinessTask 标题建议试用交接

> 归类说明（2026-11-19）：本页保留标题建议首批试用、2026-10-11 无模型会话模式及后续标题技能交接的阶段记录。下文“当前只开放个人标题技能/只支持标题建议”不概括现行全部 Chat 范围；后续个人文字技能见[通用文字合同](../../architecture/chat-general-text-skills.md)，统一入口见[Chat 创作与项目操作](../../architecture/chat-project-actions.md)。这不证明目标实例已安装、配置或完成真实模型/用户验收。
> `conversationOnly` 是显式受限模式，不是现有完整内容任务实例的覆盖配置；已有数据接线须遵守[保留数据正式设计](../../architecture/domains/workbench/chat-35444-retained-wiring-design.md)及其有界后续准入，不能直接重跑空库安装器。旧 `consumer` 同行凭据及每模型版本回填仅作历史记录，当前按[企业渠道连接](../../architecture/domains/ai/shared-channel-credentials.md#11-新版本的受限维护入口)维护，不复制旧 Key 到 Run/Image 库；用途、模型准入、预算、计费与人工 Review/Apply 仍分别核验。本次仅归类和链接调整，不读取凭据、不运行下文命令、不修改实例或执行模型。

## Chat 内创建个人标题技能

当前创建技能流程依据 [Chat 自动调用与主动选择设计](../../architecture/chat-project-actions.md#24-在同一-chat-中自动调用或主动选择创建技能)。直接在 Chat 输入「帮我创建一个保留品牌和型号的标题技能」，发送后由模型判断是否调用官方创建技能。也可以在「选择店铺」旁打开「选择技能」右侧抽屉，主动选择准确版本后在同一个输入框发送需求；手选不调用意图识别模型。当前只开放个人标题技能，可以包含一个离线 Python 标题处理脚本。

生成结果显示在会话消息中。用户可人工编辑正文与脚本，点击「让 Chat 修改」后在原聊天输入框补充需求，发送时携带当前编辑稿。编辑不调用模型，也不自动保存。点击「保存到我的技能」才按原保存合同写入个人私有草稿；发布、启用和标题处理沿技能库原流程进行。首次与后续生成均不展示点数金额，实际用量到「用量明细」查看。

生成或保存结果不明时，显式核实原请求；刷新、双标签页与重复核实不新建第二个技能。会话、会员、企业与准确技能版本绑定原请求。普通历史读取不执行模型；旧的仅创建会话请求不会因更新页面而自动发送需求。没有原请求凭据时不凭“无用量”认定零消费。

### 有界安装与开启

复用现有试用环境、当前企业连接、有效资费与 owner，保持其他模块配置。`aiWorkbench.chatTurnsEnabled` 默认关闭。已有当前 schema 由原 Workbench owner 显式执行以下操作；它仅增加父消息与原技能草稿的关联字段及限定权限，不改写已有业务行：

```powershell
$env:AI_WORKBENCH_SCHEMA_DSN = '<private owner DSN>'
go run ./cmd/ai-workbench-schema-init --runtime-role ai_workbench_runtime --chat-turn-only --expected-database '<approved database>' --expected-server-address '<approved IP>' --expected-server-port <approved port>
Remove-Item Env:AI_WORKBENCH_SCHEMA_DSN
```

安装先核对当前约束和运行权限，部分安装或约束漂移则停止；运行服务不执行 DDL。新后端仍读取原请求。自动模式通过独立用途 `aiworkbench_chat_intent` 配置 `chatIntentTextPolicies`，准确提示为 `ai-workbench-chat-intent-v1`、结果格式为 `ai-workbench-chat-intent-decision-v1`，并在现有 `modelRoutePricing.bindings` 准入同一用途。原 `aiworkbench_store_route` 配置保留，不能替换成新提示。手选仅要求 Chat 交接能力和原创作模型可用，不要求识别模型配置。

准备好 schema、模型准入和新后端后更新前端，再按批准范围开启 `chatTurnsEnabled`。一旦有新格式请求，故障时保留新 reader 与原请求恢复、暂停新发送并前向修复；不能切回不认识新消息的旧 reader，不能删除关联或用户草稿。关闭新发送不禁止原请求核实。验收分别记录受控浏览器、真实模型与账本结果；开发自检和独立代码复核不代表用户验收。

> 新版本凭据维护已切换到[企业渠道连接](../../architecture/domains/ai/shared-channel-credentials.md#11-新版本的受限维护入口)：一条企业连接可被多个显式准入模型引用。本文旧 `consumer/action/clientName/model` 同行 Key 配置及“回填每模型凭据版本”段落只作历史记录，不是当前执行指引；不向 Run/Image 两库分别保存 Key，不导入保留库。原独立模型准入、调用预算与单一计费不变。

Refs #576、#588。入口为当前 Console 的 `/workbench/ai/chat` 和 `/workbench/ai/tasks`。获准企业的成员从已保存的 1688 商品采集操作开始，在 Chat 中描述标题目标；模型只返回澄清或待确认方案。成员确认后创建 BusinessTask，调用现有 Product Agent 生成建议，最后仍须在现有 Review 中由授权人员接受、编辑和应用。当前不发布到平台，也不自动应用标题。

## 准备当前应用

### 本轮无模型的会话管理

2026-10-11 用户确认暂无模型配置，并选择先开放真实会话管理。按
[`ai-workbench-conversation-runtime.md`](../../architecture/domains/workbench/ai-workbench-conversation-runtime.md)
使用显式 `aiWorkbench.conversationOnly: true` 和 `allowedOrganizationIds`，保留
`productAgent.enabled: false`，不配置 planning policy。仍使用原 `product_agent`
逻辑库中的 `ai_workbench` schema 和独立 `ai_workbench_runtime` 连接池。
模型发送、方案确认和任务操作均返回不可用，且不创建执行收据或调用 provider。

已接入统一 Compose 的内部试用实例，在原 base、knowledge、data-services、modules、
ecoservices 五层之后加 `docker-compose.chat.yml`。私密 env 中显式提供
`CHAT_ALLOWED_ORGANIZATION_IDS='["已授权试用企业ID"]'`，使用包含
`ai-workbench-schema-init` 的 schema-init image。首次启动只安装原空 schema 和
限定 runtime role；发现已有 schema/role 或中断 marker 时停止，不覆盖或重置。
再次启动校验原输入/输出 manifest hash、组织列表与原 runtime password。
Serving 不挂载 owner/admin secret。新增 chat-install-state、chat-runtime 两个 volume；
原数据 volume 必须全部保留。更改组织或 base manifest 需显式重新组装，不能删除 marker 重跑。

正常登录并选择获准企业，进入 `/workbench/ai/chat`：新建会话，修改名称、收藏、
归档；在最近、收藏、归档列表中重新打开，归档详情可恢复。所有事实按当前企业和本人保存。
正常 stop/start 后会话仍在；这轮模型状态显示待配置，不能发送消息。
以下完整规划/执行步骤仍适用于明确取得模型配置及执行授权后的 full mode。

复用 [`product-agent-trial.md`](../product-agent-trial.md) 中的 Product Acquisition、Product Agent、Review、Asset、AI invocation、Commercial、正常登录和私有 manifest。`productAgent.enabled` 及对应企业 allowlist 必须已配置。`aiWorkbench.enabled: true` 额外要求：

- `aiWorkbench.database` 指向与 `productAgent.database` 相同的 PostgreSQL host、port 和 database，用户固定为 `ai_workbench_runtime`，密码只放私有 manifest，连接池不超过 8。
- `planningTextPolicies[organizationId]` 为获准企业单独设置 `RoutePolicy`，其 `Profile` 使用 `PromptVersion=ai-workbench-chat-plan-v1`、`OutputSchemaVersion=ai-workbench-plan-decision-v1`。规划和标题执行分别有自己的组织凭据、准入版本、输入/输出上限、点数费率与价格；不能借用对方的 route。
- 标题 `productAgent.textPolicies` 也须使用当前 `RoutePolicy`。两种文本操作都经 Eino 的受控组件、当前 AI invocation 与成员额度 owner 执行；实际 provider 的计量、价格和付费准入需要部署者核实并单独授权。缺任何一项时保持执行关闭。

从已有全源 Account Audit 配置接入 ProductAgent 时，复用当前 `product_agent` owner 数据库，省略 `accountAuditUsage.product` 专用 reader 配置，保留未启用 ImageAgent 的 `accountAuditUsage.image` 只读目标。审计 product 来源由现有 ProductAgent pool 明确提供；不要同时配置同 namespace 的 reader，也不要关闭整个 Account Audit。启动装配须同时消费 `WithProductAcquisition`、`WithProductAgent` 和 `WithAIWorkbench`，仅修改清单或菜单不会挂载任务接口。

全新安装时，先由数据库 owner 创建限制权限且没有 schema/table 所有权的 `ai_workbench_runtime` 角色，再使用私有、具备建表权限的连接执行：

```powershell
$env:AI_WORKBENCH_SCHEMA_DSN = '<private PostgreSQL DSN>'
go run ./cmd/ai-workbench-schema-init --runtime-role ai_workbench_runtime
Remove-Item Env:AI_WORKBENCH_SCHEMA_DSN
```

命令在一个事务中安装 `ai_workbench` schema 并授予 runtime 对其表的限定 SELECT/INSERT/列级 UPDATE；该 runtime 不能 DELETE/TRUNCATE/DDL。应用启动只验证 schema，不自动安装或迁移。这里的连接与私有 manifest 不提交到仓库。运行前按现有 PostgreSQL 备份/恢复规程覆盖新增 schema；正常 stop/restart 保留 Conversation、PlanningCommand、Proposal、BusinessTask，销毁数据库不是停止操作。

首次配置时，先保持 `productAgent.enabled: false`、`aiWorkbench.enabled: false`，在获准企业的 title/planning 策略中留下空的 `AdmittedCredentialVersion` 和 `AdmittedEndpointIdentityDigest`。通过现有 `cmd/product-agent-credential-provision` 给规划 route 写独立的组织凭据：私有输入 `consumer: "planning"`，其他字段及独立 writer 角色要求与标题凭据相同；标题使用 `consumer: "title"`。该命令验证策略形状及输入 endpoint 后，只返回非敏感的 `credentialVersion` 与 `endpointIdentityDigest`；将两者填入对应策略，再启用服务。轮换或停用会改变准入状态，不会自动改用其他企业、成员或默认凭据。OpenAI 兼容与 Claude 原生组件均需为精确 route 核实计量与计价，不能由协议兼容性推断付费可用。

```powershell
go run ./cmd/current-application -config C:\private\current-application.json
```

Console 的 `LISTINGKIT_SERVICE_API_BASE` 设为当前 application 的 origin 加 `/api/v1`，并保持现有 Auth.js/ZITADEL 配置。进入 `web/listingkit-ui` 后用正常 `pnpm dev` 或既有构建/启动命令。当前应用未启用 Workbench 或缺 schema/策略时，Chat/Task 请求明确不可用；前端入口本身不证明后端、provider 或付费额度已开放。规划与标题两条当前企业路由都就绪时才显示 Chat 规划可用；任一路由失配时保留已有会话读取，新消息不会触发付费规划发送。

## 使用路径与限制

1. 正常登录、选择获准企业，在 1688 采集详情获取已保存商品的 `operation_id`。Chat 新建会话后输入该 ID、目标平台与标题目标，可选当前可读的模板或 KnowledgeBase。Chat 仅保存本人/当前企业会话。
2. 查看模型回复；只有 `READY` 才显示由服务器当前 Product、AgentConfig、Knowledge 和模型准入事实形成的方案。核对商品、平台、模板、知识、模型与最大额度，再显式确认。新消息或归档会使旧方案不再可首次确认。
   Chat 规划权限与标题执行权限分别检查；确认须具备当前企业的执行角色。仅配置为全局管理员不能替代该角色，缺少权限时保留规划和历史读取，禁用确认并显示权限提示。人工 Review / Apply 继续要求其现有企业角色。
   若标题执行路由未就绪，或方案冻结的标题模型配置已不再匹配当前准入配置，确认按钮会禁用；可保留历史阅读，配置恢复后刷新，模型配置变更时重新提出方案。即使按钮可点，服务器仍在确认时重新验证当前事实。
3. 从任务中心打开确认生成的 BusinessTask。若执行在 Agent Claim 前中断，用“启动原任务”沿原请求继续；需要反馈时在原运行上继续。结果未知时不要创建新任务试图重发。
4. 候选通过校验后提交现有人工 Review；由有权人员接受/编辑并显式应用。任务状态来自 Product Agent 与 Review 的当前事实，不能把模型建议当成已应用结果。

会话和任务页面只显示当前账号/企业及当前页的真实数据；无权限的商品细节会隐藏。浏览器会话中保存待决消息和确认操作键，刷新后的显式重试使用原键及原载荷。关闭浏览器不取消已派发的模型调用。当前只支持已保存商品的标题建议；没有平台发布、多 Agent 编排或自动应用。真实 provider、实际浏览器登录路径和用户试用须在目标环境分别验证；隔离数据库与合成模型测试只是开发证据。
