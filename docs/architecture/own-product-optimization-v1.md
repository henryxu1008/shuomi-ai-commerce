# 自有商品无店铺依赖的内容优化与 Apply

Execution: #710。Design Basis: **Independent Architecture**。
Admission: **IMPLEMENTATION_READY / FROZEN**；2026-10-15 第一轮独立设计检查通过。

## 用户结果与产品依据

2026-10-15 用户明确要求：补齐自有商品不绑定店铺也能真实优化并 Apply，
先明确设计、复用现有能力；完成后使用已保存测试商品检查优选申请至提交前，
核对供货信息页与 Figma，保留已有数据，不实际提交申请。

当前缺口是正式独立内容执行入口仅接受 acquisition operation；另一条 collection
执行入口依赖 SHEIN Target/店铺。优选仅接受自有来源和真实 Review Apply 是正确边界，
不得以手工改 optimized、伪造采集回执或放宽申请规则消除缺口。

Figma/Product Authority：沿 [最终 UI/IA](../product/final-ui-ia-authority.md)，
文件 `tg48P46SSXl6TBy9lZwg63`：自有商品库 `3500:1100`、自有录入 `3502:405`，
我的智能体/配置 `4703:429` / `4703:892`、执行确认 `4722:429`，
优选选择/供货 `997:764` / `1004:892`。
既有独立自有详情由 [own-product-entry-v1](own-product-entry-v1.md) 承接。
本次在该详情增加内容优化入口和同层独立操作页，复用既有 Agent 面板、执行确认与
任务中心审核页；没有新顶层模块，不声称 Figma 已存在一个新的优化页 frame。

Must：当前企业本人 own 商品、准确版本、真实受治理模型候选、现 Review 接受/Apply、
Catalog 新版本与回执、刷新后真实优选资格；全路径不要求店铺或平台账号。
首个真实试用使用既有内容智能体的 title-only 选择，明确显示仅优化标题。
用户随后指定 DASHSCOPE、预算人民币 100 元；本轮按累计不超过 100 元执行，
具体模型/路由须与现有运行配置核对，继续原费用和积分准入，不通过 CLI 绕过产品调用。
既有内容智能体已支持的其他文本字段仍沿原准入，不在本次扩大字段/权限/模型能力。
实际部署为 title-only 时不得显示其他字段可用。

Out of Scope：批量优化/申请、新 Agent/Registry/队列/调度器/账本、图片/价格/库存/SKU
修改、平台适配/发布、真实优选提交、新 Store、迁移/回填/清理、自动初始化或提高预算。
没有新增 Accepted Risk；沿既有 actor-private、可信版本和模型不可信的 Threat Model。

## 复用 owner 与依赖

- Collection/Source/Catalog 继续持本人保存资料、来源发布和不可变版本。
- `internal/product/contentagent` 增加有界自有来源绑定计算，消费现 Collection 读取、
  Catalog reader 和 Review lineage；不持数据库、运行状态、凭据或业务正文副本。
- 应用仅注入已有 Collection service 与上述 reader；现 Product Agent / Eino / RunDB
  持运行、配置快照、模式、checkpoint、CAS 和调用记录。
- 原 provider-neutral governed text port、Commercial quote/dispatch/settlement 持模型、
  路由、调用上界、额度、费用与 UNKNOWN；不新接 SDK 或写入任何新 Key。
- 原 Product Review 持候选、决定、Apply 与原子 Catalog/receipt。原 SupplyMarket
  EffectiveProductReader 读取真实 Apply，不新增“优化完成”字段或事实 owner。

引用冻结合同：[内容智能体](product-content-localization-v1.md)、
[权限模式](enterprise-agent-permission-modes-v1.md)、
[供应商可替换](product-agent-text-provider-neutral-v1.md)、
[Review](../../internal/product/review/README.md)、
[优选资格](supply-market-choice-eligibility.md)。

#715 的 `contentTitleOnly` + retained Chat 共存、现执行表与受限权限准备由原唯一 Writer
负责。本任务不复制这些实现、不新增安装 owner、不先行改库或覆盖其待确认配置。
源码开发可基于当前 main 独立进行；最终组合纳入 #715 的真实增量后验证相关接线。
真实运行前仍需原运行方案、准确源码/CI/镜像、模型 route/预算/凭据和相应环境授权。
未具备时源码结果与真实模型/页面试用分别记录，不用 fixture 冒充实际通过。

## 来源合同与调用路径

`本人自有详情 → 优化页 → 服务端读取本人 Collection item → 精确原始 Catalog
与当前合法 Apply lineage → own_product Binding → 原配置/模式/模板/预算确认
→ 原 content Agent runtime → 原 Review 提案 → 接受/显式 Apply（或既有合法策略采用）
→ 原 Catalog 新版本/回执 → 优选选择页重新读取资格 → 供货页 → 停在提交前`。

新增上下文 `ContextKind=own_product`，`ContextID=真实 Collection item UUID`，
`TargetPlatform=product`；ProductKey/CatalogVersion/PublicationID 全由 owner 读取。
它不是 acquisition/Store/BusinessTask，也不创建伪造 operation 回执。
RunDB 现唯一键已含 org/actor/context_kind/context_id/request_key，无新 schema。

新路由在当前 Collection item 下：

- `GET /api/v1/workbench/collections/items/:item_id/content-agent/source`
- `POST .../content-agent/runs`
- `GET .../content-agent/runs/:request_key`
- `POST .../content-agent/runs/:request_key/resume`
- `POST .../content-agent/runs/:request_key/review`

源预览只返回本人 item、商品标识、准确原始/有效 publication+version 和标题；
沿现有 bounded snapshot 和安全投影，不泄露原始模型、secret 或任意内部 trace。
新 start 使用现 content start 结构，额外必填 ownSource 的准确
`catalogVersion/publicationId`；只接收 `targetPlatform=product`。
不接收 org/actor/member/roles、任意 productKey、整个快照、批准或 Apply 声明。
其余 acquisition 路径不接受 ownSource；resume/read/review 不接受新的来源选择。
严格未知/重复字段、UUID/UTF-8、请求大小、2 分钟执行和原超时合同不变。

服务端首先通过原 fresh Identity/Collection service `ReadItem` 核对本人、企业、成员、
collection.read 与 own/非归档。新执行还需原 local.agent.write、workbench.agent.use、
已启用内容 Agent、准确模式/配置/模板、route/预算及知识/Skill 现准入。
来源计算核对 Collection 原始 publication/version 与完整 Catalog snapshot，
当前 head 只能是该原始版本或从它出发的真实 Review Apply lineage。
新键比较预览的 exact effective publication/version；确认后资料改变返回冲突，
不静默换 latest 或重新解释人工确认。无图商品可以优化文字，申请投影约束继续独立。

每次 read/resume/工具读取/候选送审重新检查 item 和准确 binding；路由读取固定
`own_product` namespace（使用现 `LookupContext`），不能借同 UUID/key 读取采集运行。
历史运行只能消费其已冻结版本；同一原始来源上的后续合法 Apply 不重绑旧运行。
人工编辑改变 Collection source 后，旧运行不得被用作新来源执行，冲突需用户重选。
本人归档阻止新执行，历史 Review 提案与 Apply 继续遵守原 Review/Source 的权限、
不可变来源与 stale-base 合同；本次不引入另一套归档/审批状态机或撤销协议。

工具来源适配从现 acquisition-only 分支提取当前正确行为：先核对精确上下文，
再复用 `ResolveAppliedSource` 产出同一 canonical/source evidence；不制造新的证据 ID。
content validator 继续校验模型实际观察过的证据及冻结字段/语言。
所有采集、店铺绑定 collection、title/content sibling paths 保留其原准入。

## 事务、幂等与失败

配置/模式、Agent run 及 provider 调用仍在各自已有 owner；不尝试跨 RunDB/ProductDB
做新事务。Review Apply 继续在原 Product UoW 原子提交 Catalog、Review 和 receipt。
未修改字段、原始 Source/Collection 引用保持不变；Apply 后通过 Catalog 当前 head
和真实 lineage 计算优化资格，既不把手工编辑当优化，也不复制更新另一份 Product。

start 的幂等仍为原 org+actor+context kind/id+request key。先读准确 namespace 的
持久回执；已存在时比较原冻结 binding/字段/模式/模板/Skill 等原指纹后返回，
不得因当前 head 前进而再次调用模型。相同键不同载荷冲突；无法确认存储时失败。
浏览器沿现 URL 中的 scope/key/完整原 start 选择恢复，并保存 ownSource 原精确值，
不在刷新后拿新 head 冒充“原请求核实启动”。

收到 Running/发送后超时/响应丢失/usage 缺失继续原 UNKNOWN owner；读原键，
不自动新键或再次 dispatch。Apply 响应丢失沿原 Review 操作键读回，stale base 冲突
不覆盖人工编辑。组织/身份切换清空未发送输入，旧 scope 的回执不跨企业显示。
read_only 只保存建议，不能使优选可申请；default 人工接受/Apply，auto/full 严格复用
已批准策略采用，UI 如实显示实际模式与是否逐项审核，不伪称已人工 Apply。

## UI、运行与验收

自有详情只增加当前权限下的“优化商品内容”入口；优化页复用 ConsolePage、
ProductAgentPanel、字段/语言、模板、Skill/Knowledge opt-in、模式/预算确认与
task-center Review。缺依赖显示真实阻碍，不显示伪造可用/完成状态。
最近运行链接根据真实 context kind 回到 own 优化页；采集历史仍回采集页。
确认结果/Apply 回执后显式返回优选申请并重新读取，不复用旧 React Query 资格。
供货页按当前 Figma 分区实查；单件申请为用户已确认决定，资质/披露的领域要求保留。

开发采用 TDD，必要证据：
1. 当前 own 被 acquisition-only 拒绝的 RED；正确 own、非本人/跨企业/归档/非 own、
   伪造 publication、stale preflight/手工版本、真实多跳 Apply 和 namespace 混淆。
2. 复用真实 Run/配置/模式/Source/Catalog/Review 的现隔离 PostgreSQL 测试，外部模型
   受控；候选→接受→Apply→真实优选资格，非文本字段/原始来源保持，重放不二次模型。
3. 页面来源/确认/恢复、企业切换、缺配置/失败禁用、最近运行入口、真实 Apply 后刷新。
4. 仅改动相关 Go/前端检查与稳定候选 CI；完整路径后一次独立最终交付检查。

实际试用继续使用保留的测试 item `7b9ba0c8-34c6-53e8-88d8-2825dec050c0`，
初始资料版本 2；原 A 商品及其他数据/卷/密钥/持久任务不变。
只有原 runtime/route/预算配置具备并获准后真实生成、审核和 Apply；记录实际模型
调用/版本/Apply ID/资格响应。随后实际检查供货页字段、校验和返回选择，留图比对
Figma，停在提交前，**不上传伪资质、不提交优选申请、不发布、不删除测试记录**。
源码/受控模型/PR CI、真实模型、运行更新及用户验收分开报告。

预计约 20–28 个相关文件，生产代码约 700–1100 行；复用原运行/审核/恢复，
不建设新的 runner 或通用框架。Legacy decision: **N/A**，不消费 RETIRE owner。

## 独立设计检查

2026-10-15，既有独立 Reviewer `/root/choice_contract_review` 对 main
`1bfae4f1039ef7341000247031109bbdc4f27dbb` 与设计 SHA256
`EB113423A433AACA870C6D4F25C161C9C6AC8F6D3E241305C1E274FD795CB8C7`
作第一轮只读检查，明确 **IMPLEMENTATION_READY，无当前 Must 设计 BLOCKER**。
生产代码在该结论后才开工。准确来源/本人隔离、原键恢复/namespace、原四模式与
最近运行导航三组义务为 IMPLEMENTATION_TEST，必须在本批实现/验证中收敛；
不重建设计、Store 或运行平台。源码准入不是实际模型/运行更新/用户验收。
