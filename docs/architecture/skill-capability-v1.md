# Skill V1：三层归属与现有商品内容 Agent 接入

> Design Basis：Independent Architecture；执行 Issue：[#688](https://github.com/qq550723504/task-processor/issues/688)，产品父项 #137。
> 状态：APPROVED / IMPLEMENTATION_READY。2026-10-14 第 1 轮独立技术边界检查无未处理 BLOCKER；用户同日批准第 2 节局部产品位置。实现与用户验收另行记录。
> 2026-10-17 局部入口调整：用户批准「智能市场 → 技能库」，含官方模板、我的技能和团队技能；执行 [#756](https://github.com/qq550723504/task-processor/issues/756)，Design Basis 为 Reuse Existing Architecture。此决定仅替代原入口限制，下面的归属、授权、版本、持久化与执行合同保持有效。
> 调查基线：main `8fc90f237c5794904771a62c7f3b85d2ecbd6b76`，2026-10-14。滚动实现/评审状态写 Issue/主要 PR。

## 1. 用户结果、依据和阶段边界

已登录用户可以从允许复制的官方模板创建个人 Skill，或用表单/Markdown 自建；保存有界且不可变的版本，在当前获授权企业的现有商品内容操作中显式选择，查看建议、所用版本及原 Review/Catalog 采用结果。组织维护者可以批准个人提交的指定版本并生成组织副本。

首个方法为 `product-title-normalization`：根据已保存商品的事实整理/翻译标题，缺信息返回待确认项，不添加没有依据的品牌、材质、认证、销量或利润。本轮只支持 `product.content.agent` 的 title-only 选择；沿 #661 已合入的内容职责及四模式使用，不再建标题 Agent、工具或运行时。未选择 Skill 的原路径保持原合同。

产品授权来源：#688 当前正文及用户 2026-10-14 指派“开始处理#688”。任务允许设计、准入后实现、必要隔离验证与主要 PR；不接管 #137 活跃交付，不取得合并、安装、共享实例变更、真实模型费用、平台写入或关闭权限。

适用依据：[最终 UI/IA](../product/final-ui-ia-authority.md)、[项目边界](./project-boundaries.md)、[身份与组织](./auth-and-tenancy.md)、[企业 Agent 配置](./organization-agent-configuration-v1.md)、[商品内容](./product-content-localization-v1.md)、[原运行](./product-agent-runtime-contract.md)、[原知识](./agent-knowledge-context-v1.md)、[四模式](./enterprise-agent-permission-modes-v1.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[派工规则](../engineering/issue-driven-development.md)。#688 仅替代现配置/内容设计中“用户方法文本不可编辑”的对应非目标；不开放 AgentDefinition、工具、模型路由或任意系统提示词编辑。

Must：三层归属、动作分离、不可变准确版本、个人明确同意的组织复制、当前企业准入、原预算/指纹/UNKNOWN、原 Review/Catalog 保存读回、无越权材料或假成功。Should：在已有合格能力可直接复用时 AI 整理草稿；不是首版前置。Out of Scope：公开/付费市场、跨企业可撤销分发、多 Skill 调度、脚本/任意网络、第二 IAM/账本/任务系统、迁移/双读双写、验收平台。没有新增 Accepted Risk；原 full_access 未经逐项人工审核的既有产品边界继续适用。

## 2. Figma 核验与局部产品缺口

2026-10-14 使用当前文件 `tg48P46SSXl6TBy9lZwg63`、页面 `31:463`「硕米官网」进行只读核验：`4703:429` 我的智能体、`4703:892` 能力配置、`4722:429` 执行确认、`432:534` 账户资料均可见且非归档，已读取高保真 context 与截图。按可见节点名称检索未发现 Skill/技能节点；这只说明本次结构检索结果，不把不存在的控件声明为批准设计。

局部映射：**用户于 2026-10-17 明确批准以下技能库入口调整**，替代 2026-10-14 仅在账户资料/智能体配置增加 Dialog、不新增二级菜单的阶段方案。当前 Figma 继续约束周边布局/主题/组件；批准该局部增量不表示 Figma 原先已有对应控件。历史位置批准和验收证据保留在 #688。

| 位置 | 拟定增量 | 用户操作 |
| --- | --- | --- |
| 智能市场 → 技能库 → 官方模板 | `/workbench/agents/skills/official`；技能库根路径默认显示同一分类 | 已登录身份读取当前获准平台模板及准确版本，按原许可复制到本人或有效当前企业；空目录/服务不可用如实显示 |
| 智能市场 → 技能库 → 我的技能 | `/workbench/agents/skills/mine` | 不要求有效企业，维护本人技能、格式检查、版本/复制/导出及共享申请；无企业时不显示可执行模型测试 |
| 智能市场 → 技能库 → 团队技能 | `/workbench/agents/skills/team` | 必须有效当前企业，按原权限查看组织版本、创建私有草稿、发布/启停、管理个人技能准入及审批个人提交 |
| 原账户资料 `432:534` / 商品内容能力配置 `4703:892` | 保留“我的技能”/“团队技能”快捷管理 Dialog | 与技能库复用同一管理组件及 BFF/API，无新数据 owner；不改动其他活跃会话负责的智能体卡片 |
| 原商品内容操作 / 执行确认 `4722:429` | “使用的方法/技能”可选框，显示来源、准确版本和不可用原因 | 一次选一个，明确不选；title-only 才可选；选中失效后阻止继续，须用户重新选择并确认 |
| 原建议/历史 | 版本元数据及按权限读取正文入口 | 显示实际 Skill ID/归属/宿主版本/摘要，不伪造模型质量或审批 |

沿现 ConsolePage、Card/Dialog/表单、CSS Modules 和设计 tokens；一级导航不变，在智能市场下增加二级「技能库」和上述三个分类。技能库使用页内管理视图，原快捷入口继续使用 Dialog，两种展示复用同一管理逻辑。不是公开/付费技能交易市场，不增加个人 Workspace、搜索推荐平台或第二技能服务。

新页面复用 `requireAuthenticatedUserId` 的 server identity gate，并将准确 expected user 交给客户端；官方/个人读取不依赖有效企业，团队必须当前 live organization。Console 已有 identity-only 页面白名单精确增加技能库根路径、`/official`、`/mine`；`/team` 及未知子路由不放宽原企业访问要求，不把整个 `/agents` 前缀作为个人入口。登录失效或身份变化立即清除原管理子树；企业切换/撤权/上下文失败时不保留旧团队内容和请求，个人/官方仍按 identity-only 原合同读取。页面与导航不产生任何后端授权，正文/复制/审批/准入仍由原 service/BFF 复核。管理组件按用户、企业、分类及权限变化重建，不混用旧版本、待确认命令或审批预览。

本次复用范围为 UI → 原 `skillRequest` → 原 `/api/workbench/skills/{official|personal|organization}` → 原 `internal/skill` service/RunDB；不改变 API/schema/owner、状态机、事务、CAS、幂等/UNKNOWN、模型预算或执行选择。没有迁移、回填、复制现有业务库数据或共享实例更新。开发验证覆盖导航/面包屑、三分类、无企业个人与官方访问、团队错误/权限/切换隔离、原管理操作及准确服务不可用状态；真实保留实例验收不在本次授权内。

局部产品位置已由用户本人确认，记录在 #688 及主要 PR；不是独立技术 Reviewer 代替产品批准，也不是功能或用户验收。

## 3. 当前源码事实和开源复用

| 已核验入口 | 当前事实 / 最小增量 |
| --- | --- |
| `internal/agentconfig/contracts.go`；`internal/integration/persistence/agentconfig/{snapshot,guard}.go` | 企业模板/配置和 Claim 共用 RunDB；不能将其冒充个人 Skill owner。配置 payload 上限 8192 字节，不放技能正文 |
| `internal/agent/contracts.go`；`internal/integration/agent/eino/{runtime,steps}.go` | Request 可比较，完整 fingerprint，单 Claim/Commit/checkpoint；增加有界可比较 Skill ref，零值省略，原 ref 不变 |
| `internal/integration/agent/einomodel/{model,content_model,knowledge}.go` | prepare 同时用于 Quote/Decide，原 governed executor 的 beforeSend gate；方法材料在 prepare 进入同一报价输入 |
| `internal/app/httpapi/product_agent_knowledge.go` | 原 Start 区分首次/准确重放，先准备配置/知识、再完整 prompt 预检；接入显式 Skill selection，不借 Knowledge ID 偷存方法正文 |
| `internal/authz/listingkit.go`；`internal/httproute/descriptor.go` | 当前角色权限与 identity-only / verified-global-role / live-organization 三种准入可复用；没有新 IAM |
| `web/listingkit-ui/.../account/account-page.tsx`；`.../agents/agent-page.tsx` | 账户资料个人查询和企业 Agent 页面已有不同上下文 gate；请求子树按用户/组织变化清理 |

外部依据：[Agent Skills Specification](https://agentskills.io/specification) 规定 SKILL.md 的 YAML 前置信息与 Markdown 正文，并允许可选附属材料；本产品只支持下述文本子集，不声称完整执行规范。锁定 `github.com/cloudwego/eino v0.9.21` 的 [Skill filesystem backend](https://github.com/cloudwego/eino/blob/v0.9.21/adk/middlewares/skill/filesystem_backend.go) 使用现有 `gopkg.in/yaml.v3`，其 frontmatter splitter 是私有函数，导出的 backend 依赖文件扫描且按名称加载。当前 `compose.Graph` 不消费 ADK middleware；不能通过同名查找取代准确 Skill ID/版本，或为了调用私有解析函数替换 Runtime。

复用已安装 yaml.v3 的严格 decoder、现 goldmark 的 Markdown AST/link 检查、标准路径与摘要库。只补产品需要的有界文件封装/字段约束；不自建 YAML/Markdown 引擎，不增加解析服务或升级 Eino。v0.9.21 的 model/context/fork 能力均不接入。

## 4. 单一 Skill owner 与调用链

新增有界 domain `internal/skill`，只拥有 Skill 定义、版本、复制来源和共享申请；不拥有身份/组织、模型调用、Agent 运行、知识或商品。

| Contract → implementation → injection → consumer | 责任 |
| --- | --- |
| skill 定义/版本/命令/材料 reader → `internal/integration/persistence/skill` → current application 的原 RunDB → Skill service/httpapi | 一个事实源，同库有界内容、管理命令回执、显式组织复制 |
| skill ActionAuthorizer → 现验证身份/权限 adapter → 原 auth assembly → 每个 Skill service 动作 | 原身份与当前组织权限；不相信客户端 owner/creator/role |
| agentconfig 的个人技能准入合同 → 同 RunDB `agent_configuration` → 现配置模块 → prepare/Claim/dispatch/采用 gate | 企业对内容 Agent 的准入策略，不复制 Skill 正文或 owner |
| agent 可比较 SkillVersionRef → 原 configuration snapshot/fingerprint/checkpoint → 原 Eino/typed governed model → 原 Review/Catalog | 仅携带准确引用与原选择；运行和商品事实仍由原 owner 保存 |
| skill UseGate / reader → 同池事务 adapter → 原 beforeSend 和 Review gate → governed executor / 原采用 | 新效果前核对版本状态/当前权限，冻住准确有界材料；不新建 worker/recovery ledger |

正常路径：本人/组织创建或获准模板复制 → 校验/保存版本 → 个人启用或组织维护者发布/启用 → 原商品内容操作明确 title-only + Skill version → 当前企业准入/原配置与四模式冻结 → 原报价/确认/Claim → Eino/原工具/governed model → 原候选校验/Review → 原人工或策略采用 → Catalog 版本及准确原回执读回。

所有 production adapters 显式注入；未安装新 schema 或缺适用模型配置时 Skill 功能/真实生成显示不可用。没有 Skill selection 的原内容、图片、上架资料等路径不访问 Skill storage，不因该可选能力缺失被阻断；选中但不可用不得回退。

## 5. 归属、动作与 HTTP 边界

Skill owner 是 `{kind: PLATFORM|ORGANIZATION|USER, id}`；平台 id 为固定宿主值，组织 id 来自 live effective organization，个人 id 来自验证 user。owner 和 creator 独立、创建后不可修改；同名跨 owner 不冲突，不按名字执行。

| 类型 | 目录/正文 | 草稿/新版本 | 发布与启停 | 复制/导出 |
| --- | --- | --- | --- | --- |
| 个人 | 本人；组织只能凭指定共享申请读取被提交的准确版本 | 本人 | 本人；不授予企业执行权 | 本人；组织复制需该版本明确 Consent |
| 组织 | 已共享版本需本组织当前 Agent read/use 权限；私有草稿只给创建人及获授权组织维护者 | Agent use 成员可维护自己组织草稿；configure 维护本组织版本 | 现 Agent configure，单独检查动作 | 另验版本的 allowCopy/allowExport；不因 use 权限自动允许 |
| 平台 | 仅已发布且符合当前开放条件的版本；平台草稿只给维护者 | identity-only verified global `listingkit.platform_admin` | 同一当前全局权限，仅平台资源 | 按官方版本明确 allowCopy/allowExport |

平台全局角色不增加个人/组织正文读取例外。组织 admin 不读取其他人的个人 Skill。继承的角色名称是现 authz 代码，不恢复 root ListingKit 依赖。权限提供方不可用、grant 撤销或 owner 不匹配时拒绝。

路由按身份范围分开：`/api/v1/workbench/skills/personal/*` 使用 CurrentIdentity + OrganizationAccessPolicyNone；`.../skills/organization/*` 使用 CurrentIdentity + LiveWrite（包括正文/目录和管理动作）；`.../skills/official/*` 的公开版本读取使用验证当前身份及宿主开放条件，管理使用 CurrentIdentityWithVerifiedRoles + 当前 platform-admin permission。path 中 ID 和 organization selector 不产生授权。共享申请的本人取消通过 personal 路由，本组织审阅/批准通过 organization 路由。

body 采用现 strict JSON，有界解析，不接受 owner/creator/官方标识/角色；创建归属由选用的路由确定。列表过滤在 repository 查询层完成，再由 service 校验，不先获取全目录后交给模型/浏览器。cursor 绑定用户/owner/filter；版本/文件/导出/共享均复验原资源；统一不存在响应避免泄漏他人资源存在性。读写均 private/no-store，正文不进普通日志或公共审计。

## 6. 持久化、版本与组织复制

同原 RunDB 新增 `skill_capability` schema，由明确 NOLOGIN owner 持有，serving role 只有范围内 DML；构造器/健康读取只 Verify，不建表。最小表为：

- `skills`：UUID、immutable owner/creator、head_version、CAS revision、创建/更新时间。
- `versions`：Skill ID + 宿主管理递增版本、完整规范文本包、摘要、操作兼容标识、可见范围、生命周期、复制/导出标记和来源三元组。正文/资源/摘要 append-only；状态列独立变更。
- `commands`：owner scope + actor + Idempotency-Key 唯一、operation/input digest、准确结果回执/最小审计，不存全文或凭据。
- `share_requests`：源 Skill/版本/摘要、原 owner、target organization、明确组织复制维护 Consent、请求/撤回/批准记录及结果副本。不是发布队列或执行任务。

企业准入由现 agentconfig owner 新增窄 `agent_configuration.skill_admissions` 表，只含 organization + registered content Agent、allow_personal、revision、操作者/时间。缺记录时个人技能不准入；正式写入需 configure。本组织已发布版本及已开放官方版本仍需明确选择和使用权，不形成自动启用。策略变更与原 organization_agents 锁同事务处理；无需重写既有模板/事实。

每次保存先校验并创建不可变新版本，初始 DRAFT 且私有；变更须生成新版本，不能覆盖已启用内容。publish 校验兼容/资源/动作权限并转 READY（组织同时明确共享范围）；enable 将 READY/DISABLED 转 ENABLED，disable 将 ENABLED 转 DISABLED。安全 revoke 将版本置 REVOKED，终态不再启用/复制；保留历史，不删除正文。编辑头、publish/enable/disable/revoke 各用 expected revision + 命令幂等；同键异 operation/payload/version 冲突。新版本不静默取代正在使用或组织绑定的旧版本。

个人分享按指定版本创建请求并明确允许目标组织复制及维护；组织维护者仅看该版本和 Consent。批准时在同一 Skill 事务中锁请求与源版本，重新检查 Consent 未撤回、源未安全撤销、目标组织/当前 configure 及包可复制，再创建 ORGANIZATION owner 的私有副本版本并记录 source ID/version/digest，最后原子写回请求结果和命令回执。重复批准只读同一回执；不同键也不能再次批准同一请求生成第二副本。发布/启用副本仍为单独组织动作。

个人原件与组织副本独立演进；原件后续更新/停用/离职不改写副本。尚未批准的请求可由本人撤回；批准后 Consent 是已完成复制的证据，不伪装成可撤销跨组织分发。离职者没有原组织访问权，组织保留副本。官方复制同样验证准确版本和复制许可，创建当前个人/组织新资源，不修改原 owner。copy/export 禁止自动展开知识/商品/外部附件；V1 包只含用户明确上传的有界文本和包内引用，现 Knowledge 另按原选择和授权加载。

这些表和配置增量只由明确 installer 在同 RunDB 一次事务安装并核对 owner/形状/约束及范围 grants，缺失/部分/外来 schema fail closed。不回填旧记录、映射旧 ID、访问或迁移业务数据。共享实例安装由用户另行授权，不能借本设计执行。

## 7. 有界导入与材料

表单、Markdown 及文件 bundle 使用同一个校验器。支持根 SKILL.md、`references/*.md|*.txt` 和 `assets/*.md|*.txt|*.json` 文本模板；每包最多 16 文件、每文件最多 16 KiB、规范包总字节最多 48 KiB、HTTP envelope 最多 96 KiB，列表单页最多 50 条，管理请求 deadline 10 秒。限制是首版资源边界，不要求恰好 16 个文件才能用；完整模型 envelope 仍受当前更严格的实际 token/bytes/cost/deadline 限制，超界明确报错，不能截断。

SKILL.md 只接受 name、description、license、metadata（至多 16 个有界 string 对）；name/description 遵循所引用规范，目录 bundle 有根名时必须匹配。metadata.author/version 仅展示输入信息，不作身份/官方标识/宿主版本。未知执行字段、allowed-tools、context/fork、agent、model、full_access、命令/动态加载入口均明确 UNSUPPORTED_FIELD，不解释为权限。YAML 重复键、多文档、复杂标签、别名/递归结构拒绝；复用 yaml.v3 node/strict decoding，不自写 YAML 语法。

bundle 是有界文件列表，不接 zip/tar 解压；拒绝重复/大小写冲突路径、绝对路径、反斜线、drive/UNC、空段、`.`/`..`、编码穿越、symlink，以及不在许可目录/扩展名的文件。Markdown 用现 goldmark AST 检查链接；仅允许当前包存在的规范相对文件/锚点，拒绝远程 URL、file/data/javascript、跨包引用、HTML/image 动态材料和未包含资源。JSON 模板只作有界文本，不执行或注册 Schema/Tool。

一次选中后把 SKILL.md 和全部受支持附属文件作为有界实际方法输入；不实行模型自主渐进加载，避免报价后读文件。方法内容放在原 prompt 的独立 untrusted method section，由 code-owned skill policy 指明只能帮助表达，不能覆盖 system/schema/商品事实/权限/预算。技能正文不进 Agent checkpoint、配置 payload、Review record 或普通日志；这些位置仅持 ref。格式检查不宣称识别全部恶意指令，不建立另一人工审核平台。

## 8. 冻结、报价、认领及撤销竞态

客户端 start 只提交 `{skillId, version, digest}`；由服务端读取资源判定 publisher/owner/操作兼容，不能上传正文代替版本。新增可比较 `agent.SkillVersionRef` 包含 kind、ID、owner kind/id、宿主 version、package digest、skill policy version；Request/ModelInput 使用零值 omitzero。只允许准确 registered content Agent + title-only 字段；其他消费者和无支持 reader 的选中请求拒绝。

首次：当前个人/组织权限与企业 policy → 读取并校验完整已启用版本 → ref 放入 agentconfig StartCommand.Request → 配置输入 fingerprint、冻结 snapshot、mode intent digest → 原完整模型 Quote/preflight → 原用户确认 → 原 Store.Claim。技能全部字节、顺序规范包/摘要及 code-owned method policy 进入原 TextInputIdentity prompt、quote InputHash 和预算。原 profile/operation/工具白名单/费用归属不改；不新增 model provider route。未选 Skill 时原 system/prompt/序列化/指纹保持原合同。

配置 payload 不存正文，8 KiB 上限不扩大；技能 ref 增量包含在原限额。版本内容始终 immutable，引用只在包内且整个包入报价，因此冻结材料可从准确版本重建，无需第二份 materialization 表或跨库 blob 事务。恢复/重放加载同 ref，不读 head/latest、不换 key/版本/模型、不重新调用；匹配完整 request。原 UNKNOWN 仍由 Agent/governed/ledger owner 处理。

认领：扩展现同 RunDB config Guard，在其原事务/Claim 前锁原 organization agent/policy 和 Skill/version，验证相同 ref、当前身份、启用和企业准入。没有新 effects 的准确已存在回执可按原权限读回，不能把回执读取当新执行准入。当前已撤销后新 Claim/Resume 拒绝。

dispatch/新候选保存/人工或策略采用：UseGate 先重验当前身份/业务权限、准确版本/owner/digest/ENABLED 和企业 policy；由同 RunDB 窄事务持有 policy/版本共享锁，返回 release closure。关闭版本/修改准入取得相同记录的排他锁。固定锁顺序为 organization_agents/准入 → Skill UUID → version → share request/command；不同 Skill 多源复制按 UUID 排序，拒绝无界锁等待。

beforeSend gate 在原 governed executor 的发送准入获得该锁，保持到同步调用返回/UNKNOWN 记账完成后释放（原 hard deadline 最多两分钟；rollback/进程断连自动释放）。revoke 若先提交，调用拒绝；若调用先准入，revoke 等待该有界效果结束，不把已在途调用描述为撤销成功或自动重做。复用现 Knowledge permit，若两者都有：先 Skill gate，再原 Knowledge gate，失败释放已取资源，原 finally 顺序释放；不为 Skill 建新 lease/reaper/运行状态机。上游成员/角色撤销沿原 live check，不能撤回已准入的外部请求。

Review 采用在取得 Skill gate 后执行原 ProductDB UoW，保持共享锁直到其事务结果确定，再释放；这只是防版本/准入在检查后变更，不声称 RunDB 与 ProductDB 原子事务。最小 gate port 在准确 Preflight replay 后、Store.Run 前取得，覆盖 candidate intake、人工变更和策略采用，事务内重核冻结 provenance；不能在现 ExecutionPolicyResolver 中检查后立即释放。人工 Reviewer 的业务权限与原执行人的 Skill 使用权分开：复用现 live origin lookup 核验原主体/member/个人版本使用和当前企业准入，允许现有获授权管理员审核，但不向其返回他人个人正文或赋予个人管理权。Catalog/Review/receipt 仍同原 ProductDB 事务；已提交产品回执只核实和读回，Skill 当前撤销不触发补偿、重调模型或重复商品写入。若没有获得 gate、任何 reader/验证失败或无法确定结果，拒绝新效果并保留原 owner 的 UNKNOWN/回执结果。

## 9. 结果出处、缓存和验证

Review intake/record/view 增加可省略的 Skill provenance ref（同冻结 run ref），校验真实 run/request/配置及 candidate digest；它不是 canonical evidence，不能进入商品来源 EvidenceIDs 或事实。历史可显示已有运行的 Skill 归属、版本和摘要，正文/文件读取仍走当前资源权限；无正文权限显示受限状态。组织复制来源元数据不授予读取个人原件的能力。

前端 query/body/selection 按 user + effective organization + operation + 原配置/模式界限绑定；用户/组织切换时取消旧请求、清理选择及相关缓存、拒绝晚到响应。个人编辑草稿可保留在当前本人表单状态，不能将组织运行内容带入新的企业上下文；localStorage/sessionStorage 不存私有材料或模型上下文。

开发按相关 TDD 与已有测试验证，不新增 runner：归属动作和个人无企业 CRUD；有界导入负例；不可变版本/CAS/同键异内容；组织复制 Consent/同请求并发/撤回竞态；准确 ref 在配置/run/quote/恢复中的身份；禁个人策略及 revoke 与 Claim/dispatch/采用顺序；default/read_only/auto/full 原行为；完整 title-only Console/BFF/API/fixture 模型/Review/Catalog 采用读回。PostgreSQL 使用独立 UUID 数据库和任务资源，不使用保留实例。

独立技术检查仅本次新增 ownership/持久化/准入/撤销边界，最多两轮正常架构检查；finding 按 AGENTS.md 分类，明确 Must、具体后果及层级。完成实现后一次集中最终用户路径检查，修复只复核对应增量。开发自检、CI 和 AI review 不是用户验收。

交接正常入口/启动和 schema 安装说明、三层管理与启停、数据保存位置、准确历史及不可用原因。实现、合并、保留实例安装、真实模型费用、用户验收均单列；当前均 NOT_RUN。没有模型/凭据/组织配置时不伪造建议成功。

Legacy decision：N/A（当前新增有界资源，消费现 owner）；不依赖 root ListingKit/tenantbridge/compatibility、旧 Task-first/Enrich worker，不做全仓退休或数据兼容。

## 10. 开工门槛和交付批次

一个 Writer、一个独立分支 `codex/issue688-skill-v1` 和一个主要 PR。首先交本设计与局部 Figma 候选供检查；随后在准入后按可验证提交边界完成 Skill 管理/版本/复制 → 原冻结/受控运行 → UI/原采用读回，完整用户结果一次交接，不把内部层各拆为 PR。

2026-10-14 第 1 轮独立检查绑定设计候选 e3873bb9c973fd875d129ff943df93d5d4bbbdf8，明确 IMPLEMENTATION_READY for technical boundaries，无 BLOCKER。两项 IMPLEMENTATION_TEST：实际 Review gate 覆盖原事务及准确 replay；人工 Reviewer 与 live origin 权限分离。第 8 节记录现合同澄清，未更换 owner/状态机/恢复协议；在实现测试及合并前收敛，不为此重开架构或创建验收平台。用户已批准局部产品映射，当前 Issue 可记录 Ready 并由同一 Writer 开始正式 TDD。源码基线已包含 #656/#661 当前合入结果，#137 后续 Agent/安装整合有其他 owner，本批不修改其活跃分支、部署或数据。
