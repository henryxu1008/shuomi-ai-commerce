# AI 工作台：固定会话工作范围与任务筛选

状态：**DESIGN_CANDIDATE（用户已确认） / NOT_IMPLEMENTATION_READY**。执行范围：[Issue #298](https://github.com/henryxu1008/shuomi-ai-commerce/issues/298) 的 G298-C1/C2。Design Basis：**Independent Architecture**，因为新增了执行上下文的持久化、跨域权限读取和冻结请求的恢复约束。用户于 2026-11-05 回复“认可设计及上述边界（推荐）”，确认本候选、D4 及首期报告边界。尚未进行独立架构复核或实现。

基线：main `05e5b14f7f0d3aa80cdccd8dc222d01916c019db`，tree `33095375daeffc73c02457c8521cf72fe5c0df8e`。原工作区的 `0d5fc5db7f4597651b9a9f7f7282700cd602a204` 与 main 同树。复用当前负责人，一个 Delivery Batch，后续最多一个主要 PR。本次授权只整理本地设计及其 Issue 索引，不创建 PR、不派工或评审。

## 1. 用户结果与产品边界

有权操作商品内容的成员，在新建 Chat 时选择**通用业务、一个项目或一家店铺**，再选择现有合格商品、语言和字段。规划卡显示准确工作范围与原预算；用户确认后得到唯一 BusinessTask，可以在任务中心按同一范围找到任务及其原运行、提案和采用结果。

- **D1，已确认：** 一条会话固定一个范围，换范围新建会话。项目已绑定的店铺必须明示。模型只读取获准的项目目标和必要店铺信息，不自动读取关联材料、订单或经营数据。选择店铺只说明工作目标，不改变商品归属，不代表已上架或已获平台写权限。
- **D2，已确认：** 首期筛选覆盖有准确范围的 BusinessTask 及其明确关联的运行和提案。独立记录继续可读并标为“未记录工作范围”；不按名称、时间或当前项目关系猜历史归属。全部独立来源的范围支持留后续，不能计为已交付。
- **D3，已确认（#36）：** 商品内容逐字段查看和编辑，整份接受或拒绝，完整选中集合一次采用。沿[现有内容审核合同](product-content-localization.md)，不增加部分批准或字段级状态。人工决定、政策采用和外部动作权限各按原合同。

产品依据：[D1/D2 决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/298#issuecomment-6649958948)、[D3 决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/36#issuecomment-6649971251)、[最终 UI / IA](../product/final-ui-ia-authority.md)。沿现可见 Figma 的 [Chat 427:2467](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=427-2467)、[任务中心 427:3005](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=427-3005) 及原项目入口。本轮引用原已批准依据，没有重新读取或修改 Figma；新增选择、范围说明和筛选控件是待确认的局部候选，不冒称原稿已有。

**Must：** 真实范围、确定的同键身份和输入、实时授权、原有界预算、唯一 Task/Run、原恢复、分页正确，以及未记录范围如实表达。**Should：** 复用原页面和控件，按一次有用的内容任务路径交付。**不做：** 通用上下文服务、项目自动编排/关联、所有来源统一分配范围、范围权限体系、第二账本/恢复器、旧库迁移或发布上架。无新 Accepted Risk。

## 2. 现状与最小改变

| 已核实的现状 | 本批最小改变 |
| --- | --- |
| [Conversation / WorkScope](../../internal/aiworkbench/contracts.go) 只有会话元数据及商品选择、模板、知识引用；现 `commands.work_scope` 上限 1 KiB，项目目标本身可达 4 KiB | 将固定业务范围作为 Conversation 的元数据；原商品选择仍在 WorkScope。冻结的有界工作上下文只存于原规划命令，Proposal 和执行请求引用它，不把正文塞进 1 KiB WorkScope、512 字节 GoalSummary 或 Knowledge 引用 |
| [项目详情读取](../../internal/aiworkbench/projectcenter/service.go) 会解析手工关联，Project 的 Store 字段与 Task 链接并非 Chat 执行范围 | 在原 Project owner 增加只读的元数据窄投影，复用本人/企业/当前权限及既有仓储；不调用会展开关联的详情查询，不自动写 Project 的关联关系 |
| [Store member reader](../../internal/storecenter/member_access.go) 已按当前企业和成员授权读 Store，原事实另含连接等敏感字段 | 复用 member-scoped reader，立刻投影本用途白名单，不把完整 StoreSnapshot、连接信息或已授权账户内容交给 Workbench/模型 |
| [ListTasks](../../internal/integration/persistence/aiworkbench/list.go) 按创建时间和 ID 取页，当前游标只有 Task ID；[TaskBrowser](../../web/listingkit-ui/src/components/workbench/ai-workbench/task-browser.tsx) 无真实范围筛选 | 在原 Workbench 查询内先按准确范围过滤、再沿既有排序取页；游标绑定同一筛选。原 Run/Review 详情沿确切 Task 关联读取，不新建全局聚合源 |
| 原 Eino Planner、内容模型已有 exact input/profile、Quote、最后发送门禁和原账本；Planner 的模型摘要不是可信范围 | 在现有 [Planner](../../internal/integration/aiworkbench/einoplanner/planner.go) 和 [内容模型](../../internal/integration/agent/einomodel/model.go) 输入分别显式消费同一冻结上下文，沿原 quote/input hash、Claim、send gate 和 Invocation 结算 |

## 3. Contract → implementation → injection → consumer

| 合同 / 事实 owner | 实现与注入 | 当前消费者 |
| --- | --- | --- |
| 固定范围、命令快照、Proposal 和 Task：`internal/aiworkbench` | 原 domain 类型、原 ai_workbench 逻辑库/schema/store、原 Workbench HTTP/BFF | 新会话选择、会话/方案/任务详情、Task 列表 |
| Project 目标、创建者、生命周期与绑定店铺：`projectcenter`；Store 身份、版本与成员权限：`storecenter` | 当前 application 注入两个有界只读端口；Project 读窄元数据，Store 消费原 member-scoped reader；不可用时相应范围 fail closed | 选择候选、冻结前、形成方案/确认前、范围数据展示及新效果门禁 |
| 内容请求、Run、配置、预算、模型调用：原 Agent/AgentConfig/Eino/AI Capability/Resource owners | Workbench 只传经服务器铸造的上下文引用；原内容配置/Run/request fingerprint 和模型准备端口注入该引用的授权读取；现 Knowledge/Skill 端口和权限保持独立 | 同一个内容 Runtime 和模型适配器，沿原调用/计量，不增加 Agent 产品或调用次数 |
| 任务状态、Review、Catalog：原各 owner | [AuthorizedTaskProjectionReader](../../internal/aiworkbench/task_reader.go)、原 TaskDetail/ProposalSession、原独立人工 Apply 或合法政策采用 | 按实际 Task→Run→Review 关系查看，采用仍归 Review/Catalog；Target、Asset、Submission 不改 |

端口仅供当前业务组合，不经由本机 HTTP 自调用。Project 与 Store 不向 Workbench 转移写权限，Workbench 也不拿它们的库连接。唯一新增的冻结事实是 Workbench 自己的**用户执行输入**，不是对源事实的第二份可编辑副本。Legacy：不承接旧 owner，不新增兼容路径；现有合格入口和原合同复用。

## 4. 固定范围和准确输入

### 4.1 新建会话

原 Create 仅新增严格的 `scopeSelection`，形式为 `{kind:"general"}`、`{kind:"project",projectId}` 或 `{kind:"store",storeId}`。三种形状互斥；不接收 actor/org/member、Project/Store 事实、授权标志或自由文本范围。ID 沿对应 owner 的有界格式，未知、重复键及额外字段拒绝。项目的 Store 由服务端读取，不能在 project 分支同时传另一个 Store。客户端用最近一次获准预览的范围确认摘要确认所见的绑定；预览与保存不一致时明确重新确认，不静默选店铺。

持久化 `BusinessScope{version, kind, projectId?, storeId?}`：项目分支可带其创建时准确绑定的 Store，通用显式记录为 general。创建者/企业沿现 Conversation scope；范围只在 Create 写入，原改名/收藏/归档不改它。Project 关联的店铺后来换绑、添加或移除时，原会话不换范围；可读历史保持，若要按新范围发起规划则新建会话。

范围发现复用原有权列表/窄投影，只返回 ID、可见名称与明示的绑定店铺；选项有界分页，项目首期每页最多 20，Store 保持原较小限制。不展开关联列表。无 Project reader 或 Store reader 时只关闭相关选项；带店铺的 Project 不能静默退化成无店铺。通用与会话只读可独立于模型是否配置，范围选择不启用 Chat Planner。

### 4.2 T0 冻结工作上下文

每条新消息仍提交原商品/语言/字段/模板/知识选择，不再次接受任意范围。服务器先由当前会话取固定范围，再在原 Workbench 事务**外**读取当前授权及以下白名单：

| 用途 | 固定快照包含 | 不包含 |
| --- | --- | --- |
| 绑定及版本校验 | 精确 org/actor/member、Conversation、范围版本/种类、Project ID/observed revision、Store ID/observed version、准确 Project→Store 关系和实际使用字段摘要 | 浏览器提交的权限快照、令牌、跨企业资源、完整源对象 |
| 人员确认和模型上下文 | Project 标题/目标，Store 显示名、原 Platform/Region（作为目标市场背景，不推断平台字段/规则） | 私有附件、全部链接对象、知识库自动绑定、账号 ID/凭据/连接引用、订单/经营/资金数据 |

Project 目标原上限 4096 UTF-8 字节，标题及 Store 各字段沿原 owner 上限；冻结 JSON（含编码后的真实长度）最多 16 KiB，字段和结构严格校验。保留原 `work_scope` ≤1 KiB、`execution_request` ≤8 KiB、消息 ≤8 KiB、最多 50 条且原历史正文 ≤64 KiB，以及更紧的实际模型 profile/请求限制。候选对**新有范围规划**使用不超过 64 KiB 的序列化规划输入包，历史和新增范围数据共同计入；这不是声称现代码已有整个输入包的该限制。对实际编码后的输入重做原 Quote 校验，计入 token、时间及点数预算；超界拒绝，不裁剪目标、改默认模型、增加解释调用或放大预算。

从现 `PlanningPort.Admission` 得到的纯准备闭包携带准确范围与有界快照；T0 在现 scoped key lock → Conversation row lock 下复查范围身份、归档和原键，再冻结原消息前缀/选择/快照/Planner profile/InvocationID/InputHash。快照随原用户消息和命令原子提交。准备闭包不做跨 owner I/O、网络调用或建资源；不可在锁内取 Project、Store 或模型。纯计算可以拒绝输入超界，原键已存在时仍先走精确回执。

作用域元数据与供模型消费的字段分开：模型只收被批准的有界内容和必要上下文，身份/引用摘要由宿主用于绑定，不作为模型自行选权限/资源的指令。Project 文字和显示名是非可信数据，不当系统提示。模型不得改范围、选店铺、添加工具、改变商品选择、语言、平台目标或已确认费用。模型生成的 GoalSummary 继续只是目标摘要，不用作范围证据。

### 4.3 绑定与版本

- `conversations` 保存固定范围的严格形状；`commands` 保存一次规划所用的有界快照和 canonical digest。新范围快照用带版本的确定性 JSON，严格类型/字段检查后重编码并核摘要，不从序列化字段顺序或 DB 格式推断内容变化。
- `execution_proposals` 只保存 `WorkbenchContextRef{contract, conversationId, commandKey, digest}`（有界，不含正文），并纳入 proposal digest；必须匹配原消息、sequence、会话、scope、member 和合法命令。Task 通过现 Proposal FK 和 execution request 内同一引用读取，使用同一份快照；Task 不复制可变 Project 目标，不另建快照表。新查询在 Workbench 同库按 Conversation 不可变范围筛选，禁止跨库 JOIN。
- 原 Agent 的 `ContextSnapshotRef` 仍是 Knowledge 引用。可比较的 WorkbenchContextRef 结构由 `agent` 合同承载，Workbench 仍拥有所指的快照；Agent 不导入 Workbench 或跨库读取。该引用随原内容 Request→配置绑定/序列化摘要→ModelInput→Quote/Decide 贯穿，原 Review ExecutionProvenance 仅保留必要的同一引用与准确 Run/RequestKey。带范围 profile 与合法直接内容 profile 分别精确校验；后者不能当作缺少/损坏 scope 时的 fallback。只读取与原 Task、Run、当前成员和准确商品绑定一致的原命令，不能凭命令 ID 自行扩权。
- 候选版本：范围/引用 `workbench-scope-v1`，有范围 Planner `ai-workbench-chat-plan-v3`，有范围内容 prompt/profile `product-content-scoped-agent-v1`，继续原内容 action/Review policy。使用原 profile/prompt/定义准入和版本绑定，不把新输入套入 v2 Planner 或未声明的内容 profile；新会话包括“通用业务”也走同一严格新合约。Agent Definition 与配置清单若因该组合必须递增，正式实现前固定准确版本，并按原准入证明所有现合法直接消费者不受误伤。不把当前候选标识写成已注册或准入。
- 新空库显式 installer、原 Workbench 非 owner runtime 角色；精确检查新列、约束及列级写权限（固定范围只能创建写入，运行端不改身份），原安装与 readiness 失败关闭。保留实例/库/会话/已提交回执由其原版本和原 owner 保持；不迁移、不回填范围、不让旧请求跨本新合同执行、不双读双写。若要求新版访问保留历史或跨版本恢复，停下报告，须单独产品决定和设计；不是本次实现义务。

## 5. 权限、范围变化与发布边界

| 边界 | 需要的当前授权与行为 |
| --- | --- |
| 发现/创建范围 | 现 Chat 新建权限；Project 必须是当前企业本人私有且可读、未归档，不以管理员角色跨创建者；Store 必须当前 member-scoped 可读且非删除记录。只读 metadata 不授予 Store 操作、付费服务或连接授权；也不把平台连接健康当范围读取前置 |
| 消息、形成 Proposal、T1 确认 | 现 Chat Use/确认权限与原商品/Agent/知识/模式/预算所有门禁；重读准确 Project/Store 及相关字段，校验仍为固定会话范围。未确认方案所用资料/绑定变动，要求重新规划并明确确认，原方案不可静默追新。范围授权拒绝在新效果前失败，不能自动换成通用 |
| 已确认任务的启动/继续/发送 | 原冻结输入和同一 Task/Run 身份；按原每次新效果门禁重验当前身份/member、scope 读取、原对象/配置/mode/预算。在 Planner 与内容模型的原发送 gate 内再核当前范围权限和精确快照；失败不发送、不换路由。Project 目标改动不替换已确认的任务输入；归档 Project 不是取消已确认任务。既有挂起工作继续需当前授权。业务读取撤权不被 old snapshot 代替 |
| 读取、回执与结果 | Chat/Task/Run/Review 继续各自原读取权限，并对有范围用途的消息、方案及建议加窄范围披露检查；Project/Store 不可读时掩蔽相关名称、目标、ID/链接、模型生成内容及统计，原有权读取的去敏操作状态可保留。会话标题可能由该范围生成，列表同样掩蔽。用原冻结 request/profile/provenance 判定范围用途，缺损引用视为不可用，不能按直接记录显示；原 Run/Review 详情、列表、历史及读回用同一检查，不在 UI 单独补一层就放行 |
| 采用与已采用事实 | 新 Apply/政策采用仍按原 Review/Catalog 及四模式；新效果须满足当前相关范围和业务授权。采用后标准商品事实沿原 Catalog 权限，不挂载 Project/Store 私有快照、不写新归属/发布许可，不追溯改变原商品读取合同 |

对 Project 比较当前有效性、固定 Store 关系及**实际消费字段摘要**，不以无关手工关联修改强制作废方案；对 Store 比较身份及实际消费字段。observed revision/version 始终保留追溯，不能只比较名称。Project/Store 缺失/不可达和撤权用现错误体系，私有/跨企业对象不泄露存在性；缺依赖不是成功空。保留 #36 的权限与完整集合原子采用，不增加 Task/Review 业务状态。

**Report 边界：** [原 TITLE_REVIEW 来源](../../internal/app/reportcenter/sources.go)会持久化审核快照，且[现 Report 合同](ai-workbench-reports.md)的已保存报告不以来源在线为前置。不能把有范围的候选伪装成这个已有来源，否则即使 Task/Review 撤权检查正确，另存的派生内容仍可能在后续读回。首期有范围的提案在原任务/审核入口查看；原报告选择器不开放这类新来源，原服务端 capture 明确拒绝带本新范围用途的候选。既有三类报告、原回执、独立历史读取及下载均保持；不新增 Report 来源、另一个快照或跨库授权协议。若未来需要导出这类候选，先明确持久化披露权限及撤权语义。

本批 Threat Model 限于伪造/串用范围、越权读取私有数据、提示注入、快照错配、同键换输入、撤权后新发送/采用、重复计费及筛选/游标泄露。不把日志用作输入副本，只保留原安全事件、版本、摘要和关联 ID，不记录冻结目标正文、原模型载荷或真实 Key。

**已确认边界 D4：** 项目资料变化使未确认方案失效；已经确认的 Task 保留原输入，归档项目不自动取消它，但每个新效果必须仍有当前权限。已通过最后准入的一次在途调用按原结果与账本核实，撤权后不允许后续新发送、展示受保护输出或借失败自动重发。不新增跨库事务或实时取消承诺；若现有最后准入/读取接线无法兑现此边界，须在准入时报告，不降级为只做 UI 检查。

## 6. 原键恢复与并发

两种摘要不能混用。客户端原请求 fingerprint 由操作、严格规范请求和原资源 ID 组成，Create 含明确范围选择/确认摘要；Message 含原会话 ID 和准确消息/选择，范围身份由该会话绑定，**不**把重试时读到的最新 Project/Store 数据重算成 wire fingerprint。服务端快照 digest/InputHash 单独固定首次实际用的输入；确认和 Agent request 必须绑定该引用。跨企业、跨会话、同键不同内容/范围，继续按原权限与幂等冲突拒绝。

| 状态 / 情形 | 可做的事 | 必须拒绝的事 |
| --- | --- | --- |
| Create 同键、同选择响应丢失 | 保留原 Create 路由当前权限，先核原已提交 Conversation receipt，再按当前资源权限投影；只读 GET 仍按原读取权限 | 用只读权限调用新建动作，先读最新项目再生成第二会话，或用旧回执泄露撤权范围 |
| T0 已提交、规划终态存在 | 读原命令/原终态并做披露校验；无新准备/调用。若只 GET 会话或任务，这是只读投影，不宣称替代精确命令回执 | 把旧终态按当前目标重算，因撤权/新版本丢弃或重复发送 |
| 原键仍 READY_TO_DISPATCH | 保留现“重试同一消息”的明确操作。仅当原 owner/账本允许首次发送、命令未失效、同一快照/实际 profile/InputHash/预算和最后准入都通过时，才可完成**原一次**首次派发；从原消息前缀与原快照重建 | 将原键 POST 标为纯读取、刷新自动 POST、换 scope/输入/模型、增加一次解释或付费重试 |
| 快照变化/权限失效且原结果不明 | 先按原 invocation/claim/send/receipt 核实。只有原合同确认 no-send 才按相应失败路径；已有发送或证明不足保持原 PENDING/UNKNOWN、原预留 | 用新一次读失败/撤权/404 推断旧发送没发生，伪造用量零、释放预留或换键补发 |
| T1 确认已提交、响应丢失 | 原 current receipt auth → scoped LookupTask → 原相同 Task；预检失败仍用原 serialized ReplayOrFail 路径处理并发提交。保存后由原新效果入口显式启动或核实 | 旧方案重新准备/再建 Task/触发第二 T2；同 Proposal 换键不产生第二任务 |
| T2 尚无 Run，或已有 Run/动作回执 | 原 Lookup 先于新效果；无 Claim 时仅按原确认的同一 Request、scope/ref/权限和原 Start 规则，已有/可能 Claim 时按原 run/key/fence、状态和回执核实 | 用最新 scope 替换请求，重复 Claim，丢失确认后自动换键 Start/Resume/Apply |
| 已明确拒绝或不确定后回到页面 | 浏览器维持原 scope/key/payload 与原锁；身份/范围变化取消迟到响应。允许明确核实/刷新，不能从单次 GET 假定没有提交 | 把“已接受”“可读结果”“已采用”混成同一状态，或通过切换筛选绕过未决写锁 |

保持 T0 原 scoped key→会话锁、T1 原 task key→会话锁及 latest USER/归档规则，保留单 Proposal 单 Task；不新建锁服务、补偿器或跨域事务。跨 owner 读取在锁外，写前在已有门禁按当前事实再验，持久化字段仍来自冻结快照。T1 原子提交的是本库确认意图；它本身不是外部发送许可。后续效果由各 owner 的最后门禁负责。

## 7. 工作范围筛选、分页及真实空状态

| 筛选 | 精确匹配规则 |
| --- | --- |
| 全部 | 原多来源可读范围；每来源原独立页/状态，区分任务、运行和提案 |
| 通用业务 | 只匹配显式 general 的 Task，缺少 scope 不匹配 |
| 指定项目 | 固定 scope 的准确 Project ID，按当前本人和项目读取权查 |
| 指定店铺 | 固定 scope 的准确 Store ID，包括当时明确绑定同一店铺的 Project 会话任务；不因项目后来改绑重分配历史任务 |
| 未记录工作范围 | 原可读但没有准确范围事实的独立记录。未找到关联不等于证实没有关联；无法确定时明确显示未记录/未核定，不赋予默认范围 |

**有范围视图**由 Task owner 提供分页：同库按当前 org/actor、严格 scope 谓词筛选，再按 `(created_at DESC, id DESC)` 取 `limit+1`，上限 50。范围选择先用原资源读权限核实；每个结果仍沿现授权投影，已拒绝的记录不泄露详情、计数或跨权限游标。关联的 Run/Review 在 Task 行与详情沿原精确 ID 展示，不为了筛选遍历各原生运行/Review 全集。现任务/原生来源独立分页保留；无法证明关联的独立行不拼接到此查询，也不在某页里按相似名字去重。

- `after` 为本新合同的有界、不透明展示游标（候选：base64url 编码的严格版本 JSON，≤512 字节），包含规范查询摘要、首屏上界 Task ID 和本页最后可见 Task ID。摘要绑定 current org/actor + 规范范围查询；服务器按同一合法筛选及当前披露权限重核两个锚点，再使用实际行的排序元组，不信客户端提交的时间或权限。首屏上界来自该筛选首个可见 Task；位于上界之外的新任务刷新后读入。这不是跨来源快照。游标不作为授权凭据，不引入签名 Key、游标表或全量遍历。
- 换企业、账号、范围或权限后清除页面、游标、选择和晚到数据；原 query key/authorizationKey 范围继续保持。错误/过期/不匹配游标要求从当前筛选首屏重读，不能回退到“全部”。不把不可读行的 ID、排序锚点或数量放入 DTO/游标。局部遮蔽时，只允许从合法可见锚点继续，并明确结果不完整；若当前有候选但无安全锚点，显示该段不可用、停止该源分页，不返回伪造的“已查完 / 0 项”，也不为跳过隐藏窗口遍历所有历史。范围整体失权时清理该范围结果。
- 状态和文本搜索继续按原**已加载且可判断**范围；这次增加的是服务端工作范围筛选，不新增全库状态/文本搜索或全企业统计。详情用原 Run/Review 当前授权读取；仅具有能力/模块已装配不等于具有实际历史或执行许可。
- UI 沿原新建会话和 TaskBrowser：选择页展示固定范围与项目绑定店铺；方案卡明确显示实际工作范围和原预算；任务列表可切上述筛选并标注覆盖范围；关联内容进入原详情。原挂起/UNKNOWN 操作的切换锁继续，不能换筛选来绕开原操作核实。

## 8. 最小实施义务与准入

后续实施获授权后，复用原 HTTP/BFF、隔离 PostgreSQL、原合成 transport 和已有模型/Task/Review 回归。这里只列验证义务，**本轮均未运行**。

| 义务 | 风险匹配的证据 |
| --- | --- |
| IT1 固定范围与当前授权 | 通用/项目及其显式店铺/直接店铺；无管理权限的可读角色不被误提权，跨企业/非创建者/无店铺 grant 拒绝；变更项目绑定、缺 reader、撤权和迟到响应不泄漏或自动降级。Project 读窄投影不读取关联材料 |
| IT2 冻结及同键恢复 | 超界、digest/profile 不匹配、同键换范围/载荷拒绝；原消息/命令已提交但丢响应不二次写；项目修改发生于方案/确认间需重新确认，原终态照其权限可读。未派发与可能已发送分开，原 Claim/UNKNOWN/预留不误判 |
| IT3 原任务和费用边界 | 同 Proposal 单 Task，T1 ACK 丢失/归档优先级/未 Claim 的原显式启动；同一范围引用贯穿实际 Planner 与内容 Quote/Decide 的准确输入，撤权发生在最后发送准入前阻止发送，未获得 Scope 不触发默认模型。复用现预算、账本和有限修复，不加模型调用 |
| IT4 列表与关联 | 明确范围先过滤再分页，包含相同时间的稳定 ID 顺序、新任务刷新、范围/跨企业游标误用及权限变动；只显示精确关联 Run/Review；未知与通用不混淆，未记录或不可读不冒称完整零，原 Task 操作锁与直接详情/原 owner 读取保持 |
| IT5 原采用与装配 | 真实注入链消费快照和当前授权；原完整字段集合、编辑失效、CAS、四模式保持，原直接读取不绕范围门禁。有范围的候选不能借旧 TITLE_REVIEW 来源另存，既有报告按原合同可读/恢复。新空库安装和启动验证 fail closed，不接保留库、不隐式 DDL、不增加 serving 特权 |

实施只动范围合同和必要直接消费点；不建新测试 runner、调度/审计平台或重复评审。按阶段使用原准确候选 CI 和既有证据，不把测试替身/CI写成真实产品验收。架构敏感边界是本批实际新增的固定/冻结上下文、权限、读回与分页；正式生产修改前仍须由适用独立复核确认达到 `IMPLEMENTATION_READY`。若发现不能在现 owner/原账本和现权限门禁内实现，或必须增加安装兼容、权限或恢复风险，先报告具体阻碍，不能在实现中临时发明合同。

**用户已确认本候选、D4 和首期报告边界。** 有范围的提案首期只在原任务和审核入口查看，不作为报告来源；既有报告不变。D1/D2/D3 已有明确依据。确认仅覆盖设计，不授权实现、评审、提交/推送、开 PR、合并或关单。#298/#36 继续开放，G298-C1/C2 未实现。实例、数据、会话、真实 Key 和原 UNKNOWN/预留保留；不部署、不读改保留库、不安装或启停、不执行真实操作、不调用模型或业务平台、不发邮件短信。#47 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT、#1075 原负责人继续。缺证据 UNKNOWN，本轮运行、真实保存/恢复、浏览器及产品验收 NOT_RUN。
