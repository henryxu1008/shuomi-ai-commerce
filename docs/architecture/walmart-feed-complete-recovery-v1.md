# Walmart Feed：完整结果恢复的最小设计候选

Refs #733。2026-11-01，调查基线 `c0210ff34f44c371b5a1993aed89f439cfbdc1ff`。

- Design Basis: **Independent Architecture / DESIGN_ONLY / DESIGN_CANDIDATE / NOT_READY**。
- 本次只交付设计与 Draft PR；暂不实现、不新增评审派工。本文不是 `IMPLEMENTATION_READY`，不授权修改生产路径、schema、状态、修复资格或保留环境。
- 当前唯一 Writer、原工作区及一个主要 PR 保持不变。原 [Feed 恢复合同](walmart-feed-recovery-v1.md) 的 `IMPLEMENTATION_READY / FROZEN` 只覆盖 #1028 已批准的明确拒绝重试和有界续读，不自动覆盖本候选。
- **实际依赖未满足：单响应 50/1000 条上限存在官方文档冲突，适用于 MP_ITEM / MP_ITEM_MATCH 的跨请求结果一致性依据也未证实。** §4.4 只新增有条件的单响应候选，不认定它已可用。完整恢复要求仍未完成，#733 保持开放；不将延期或缺失证明记为 PASS、BACKLOG 或 Accepted Risk。
- 本轮补充授权仅覆盖文档冲突、待澄清问题和单响应候选；咨询 **NOT_SENT**，答复/适用性 **UNKNOWN**。不发送咨询、不实现、不新增评审派工；不以设计补充授权探测真实 API。

## 1. 用户结果与范围

运营人员已取得 accepted Feed 后，沿现有“查询平台结果”查看大 Feed；只有失败 SKU 的证据完整、属于可证明的一致结果且仍满足当前权限与原提交归属时，才允许重新明确批准精确失败子集。用户无需重新输入 Feed ID、拼接页面或判断可以重发哪些 SKU。

本轮输出是这个剩余能力的最小设计候选和依赖说明，不宣称现在已能完成上述恢复。#1028 的分页覆盖、历史页可见性和明确授权拒绝后的原合同继续有效。

| 当前 Must | 本候选要求 |
| --- | --- |
| M1 不把混合历史页当作完整结果 | 分开证明本地覆盖、平台一致性及终态可修复性；缺任意一项均不扩大修复资格 |
| M2 不重发成功或结果不明 SKU | 只消费原 owner 已保存的准确原 Feed、原载荷、失败子集和当前有效证据；新尝试重新批准 |
| M3 并发、授权漂移或中断不破坏归属 | 沿原 target / SKU / attempt-fence 事务，重新核当前 head、generation、身份、权限与 Store 绑定 |
| M4 读取与响应有界 | 复用原预算、CAS、原键回读及历史投影，不延长租期或放大调用/资源上限 |

Out of Scope：实现、真实平台查询或发送、自动重试/轮询、旧 UNKNOWN 判定或账务释放、其他 Feed 类型、手工导入/勾选通过、新菜单/工作台、通用快照/对账/恢复框架、第二事实 owner、保留库迁移。

Threat Model 沿用 [Walmart US V1](walmart-us-v1.md) 与原恢复合同：本系统的并发、重放、响应/commit 丢失及授权/来源漂移；不承诺自动对账商家其他系统的操作。没有新增 Accepted Risk。

## 2. 产品与技术依据及真实缺口

复用原 Walmart 文档 §2 的当前 Product / Figma Authority（最终 UI/IA，Figma `tg48P46SSXl6TBy9lZwg63` / `31:463`）和供应链资料页的原结果入口。本候选不新增页面、不声称执行了视觉验收。采用[全新系统基线](../product/greenfield-no-legacy-migration.md)，不新增历史兼容或迁移义务。

2026-11-01 只读官方文档，未调用业务平台：

- [Feed item status API Reference](https://developer.walmart.com/us-marketplace/reference/getfeeditemstatus)：`includeDetails` 配合 `offset`，参考页写每页 `limit <= 50`；但指南写最大 1000，冲突见 §2.1，不能把 50 当作所有官方资料一致的上限。本次查阅未找到适用的跨请求快照标识、排序版本或结果不可变保证。该保证为 **UNKNOWN / 未证实**，不是断言平台永远不支持。
- [Feeds overview](https://developer.walmart.com/us-marketplace/docs/feeds-overview)：`PROCESSED` 不能单独排除逐项仍在处理；整 Feed `ERROR` 与逐项失败分别处理，沿原状态政策。
- [Get Feed Error Report](https://developer.walmart.com/us-marketplace/reference/getfeederrorreport)：当前列出的适用类型为 `FITMENT_ACES` / `FITMENT_PIES`，不能当作本任务 `MP_ITEM` / `MP_ITEM_MATCH` 的完整结果来源。

当前 `OfficialClient.FeedPage` 只保留原返回身份、计数、逐项结果、响应摘要和观测时间，没有已验证的上游 revision/snapshot 证明。`CompleteFor` 校验准确 SKU 集合和计数；本地 generation、连续 offset、同 summary、相同页 hash 或第二遍读到相同内容都不能自行创造平台一致性保证。

**架构语义缺口：本地“枚举完整”不等于上游“同一结果快照”。** 原合同已禁止跨次页授予资格，本候选不能只去掉 `Partial` 或合并 Items 来完成剩余 Must。本文不追认原生观察具备新的平台保证，也不修改原生单窗口路径；若要将新的快照要求追溯扩到该路径，须先明确范围，不能在本设计中暗改冻结合同。

### 2.1 50/1000 条的文档冲突

2026-11-01 只读核对，以下资料均描述 Feed item status，未执行其中任何请求示例：

| 官方资料 | `includeDetails=true` 的 `limit` 说明 | 当前结论 |
| --- | --- | --- |
| [US Marketplace API Reference](https://developer.walmart.com/us-marketplace/reference/getfeeditemstatus) | 默认 50，上限 50 | 与下列指南冲突，不能直接推断支持大响应 |
| [US Marketplace 使用指南](https://developer.walmart.com/us-marketplace/docs/feed-item-status-api) | 默认 50，上限 1000 | 是单响应候选的依据线索，不是适用性或运行验证 |
| [Global Marketplace 使用指南](https://developer.walmart.com/global-marketplace/docs/feed-item-status-api) | 默认 50，上限 1000 | 不自动授权切换 API 版本、请求头、凭据或准入合同 |

不能以页面较新、默认值相同、HTTP 200 示例或另一产品线文档来裁决冲突。当前请求实际采用的 US 接口、两种 Feed 类型及相应版本/准入条件能否返回超过 50 条，仍为 **UNKNOWN**。即使某次返回超过 50 条，也不能单凭样例证明完整性、无截断或一致性合同。

当前仓库不仅在 `OfficialClient.FeedPage` 请求和响应校验中固定 50，还在 Supply 读取结束条件及 `WalmartReadDocument` 的页计数、offset、续读和摘要校验中消费该合同。未来如确认支持更大单响应，必须先定稿这些相邻路径的单响应表示与准入；不能只改 URL 的 `limit`，也不能直接放宽现有校验。§6 的现行分页合同在此候选中不变。

### 2.2 待向 Walmart 澄清的问题（草案，未发送）

以下问题用于取得可追溯的官方说明，不附商家凭据、真实 Feed ID、业务载荷或用户数据。本轮不建立工单、不发送邮件/短信、不调用业务 API，不代填缺失的响应证据。

1. **适用接口与上限**：对当前 US Marketplace 的 `MP_ITEM` 和 `MP_ITEM_MATCH`，Feed item status 的详细结果上限究竟是 50 还是 1000？请分别说明适用 API 版本、请求头、环境、卖家/项目准入及生效条件，并指出指南与 API Reference 应以哪份更正后的合同为准。若 1000 仅适用其他版本，不视为当前接线已支持。
2. **完整范围与截断**：如支持 1000，对于原 accepted Feed 的准确 SKU 数 `N <= 1000`，从 offset 0 请求 N 条是否能在一个响应中取得该 Feed 的全部 N 项？超过支持上限、服务端截断/钳制或数据过大时，是明确错误还是较短列表，如何辨认？`itemsReceived`、返回项数以及 offset/limit 的含义和完整性信号是什么？
3. **同一响应的结果语义**：Feed 状态、summary 计数及逐项结果是否描述同一完整观察，处理中状态转换时能否出现混合计数/明细？完整性及终态应如何判定，如何处理已 PROCESSED 但逐项仍 INPROGRESS 的审核情况？此问题只核必要的响应合同，不擅自扩成平台数据库线性化快照要求。
4. **如必须分页**：对这两种 Feed 类型，是否存在已支持的结果版本、不可变结果或稳定分页保证，能够把不同请求的所有页绑定为同一完整结果？适用的标识、排序、有效期和失效/换版行为是什么？逐项终态是否可能在同一 Feed ID 下继续变化？单有游标或相同计数不视为已回答一致性问题。
5. **如上述路径均不成立**：是否有正式支持这两种 Feed 类型、能绑定原 Feed 及准确完整 SKU 范围的结果来源？当前 errorReport 文档只列 FITMENT_ACES/PIES，不能当作已有答案。任何新接口、报告下载、订阅或版本切换均须另行核范围与权限，不由咨询答复自动授权采用。

五项答复目前均为 **UNKNOWN**；咨询 **NOT_SENT**。没有获得官方澄清或放行，完整恢复继续受阻。未来若需发出咨询，须先取得单独发送授权；本文件本身不是发送指令。

## 3. 唯一 owner 与最小调用链

| 事实/操作 | 当前唯一 owner 与消费位置 |
| --- | --- |
| 原批准来源、Target record、wire、预期 SKU 与选定 Store | 原 Supply Target / OfficialIntent；保存准确版本，不从当前页面重建旧载荷 |
| accepted Feed 身份与提交回执 | 原 immutable OfficialReceipt / ExecutionKernel attempt |
| 读取 generation、原生观察、页进度、head | Submission result repository；原 `listing_walmart_feed_observations` |
| target fence、SKU 当前归属与新 claim | 原 Submission UoW；原 `listing_walmart_sku_bindings` 和同 Product DB 事务 |
| 当前 member / 企业 / Store grant / 凭据 | 原身份和 Store owner；不在结果 owner 建第二套授权事实 |

候选调用链沿现有接线：原 Supply `PublicationStatus` → 当前 scope / Store reader → `integration/walmart` 官方 adapter → Submission typed observation → 原 official result repository 的 Begin / Save / Read → 原 BFF strict DTO → `walmart-feed-result` 组件。新修复仍由原显式批准入口进入 Submission Acquire，不从 UI 页进度直接推导 claim。

新证据如获准实现，也只由原结果 owner 验证并绑定原 observation；不得在 UI、adapter 或旁路工具自行签发资格。远程读取在 Product DB 事务外，校验和 CAS 保存沿原 UoW。当前没有新增持久化格式；先取得实际来源合同，再确定原 typed observation 所需的最小字段，不能预建一套虚构的 provider 协议。

## 4. 三项独立准入：覆盖、一致性、可修复性

### 4.1 本地覆盖只能说明读到了哪些内容

保留原进度链的 receipt、round、generation、previous-window、offset、每页 hash/时间和整份 typed document digest。页面必须来自同一原 accepted Feed 和准确预期 SKU 集合；不重复、不遗漏、不混入额外 SKU，计数与原结果相符。跨次页面保留原观测时间，不能改成新时间或复制为当前原生观察。

这些是必要的本地完整性校验，不是跨请求平台一致性证明。覆盖达到 100% 但缺下一节依据时仍为历史覆盖，只读展示且无新增修复资格。原生整 Feed ERROR 的证据及原单窗口政策继续按冻结合同处理。

### 4.2 新跨次完整结果必须有实际来源证明

任何拟用于跨次修复的完整结果，须先有可验证且适用于当前 Feed 类型的官方来源合同，能将**所有被消费的内容绑定为同一个完整结果版本**。合同必须明确版本/不可变语义、分页/排序、终态、完整范围以及读取失败或版本变化时的含义；不能只凭一个未说明语义的 ETag、时间戳或本系统生成的 ID。

原结果 owner 消费的证明至少同时绑定：

- org、原 Store/merchant/environment 绑定及原 accepted receipt/Feed；
- 原 intent/target 准确版本、wire digest 和完整 expected SKU 集合摘要；
- 实际来源类型及其经核实的一致性合同/版本标识、页面或整体内容摘要和原观测时间；
- 原 observation 身份、generation/head 以及被消费的全部页范围。

这里只固定必须证明的事实，不宣称存在某个 Walmart 响应字段、下载接口或可执行的快照算法。当前公开依据不足，**该准入不满足**；不能把旧进度链补一个本地证书便放行。取得新来源依据前不实现字段、schema、collector 或 RepairFrom 分支；若唯一可行方案要求新接口/下载/导入或放大预算，先报告并重新取得范围决定。

### 4.3 完整且一致仍不等于允许重发

继续按原 `AllowsRepair` 政策区分整 Feed ERROR、逐项 DATA_ERROR、SUCCESS、INPROGRESS、SYSTEM_ERROR、TIMEOUT_ERROR 和 UNKNOWN。准确失败子集须非空且属于原 SKU；处理/系统故障/超时/未知不凭本候选转成确定拒绝，成功项不混入修复。

原 accepted 祖先、latest accepted、原授权拒绝的 `RepairFrom`、当前批准 payload、source/target/Store 版本和 tenant 授权全部保留。新尝试必须显式重新批准，不复用旧 key 发送、不释放未知占用或改写历史回执。

### 4.4 有条件的单响应候选（暂不实施）

**候选理由是工程推论，不是已验证的平台能力**：若 §2.2 问题 1–3 获得适用于当前接口及两种 Feed 类型的肯定完整合同，原系统最多 1000 个 SKU 的一个 Feed 可能在一个原生响应中取得全部结果，从而避开跨响应拼接。单次返回本身不自动证明完整、一致或终态；仍须先满足已澄清的响应合同及下述校验。本轮未获得该答复，不采用该路径，也不宣称它已解决 M1/M2。

候选最小路径：

1. 原 owner 从当前授权 scope、accepted receipt、原批准载荷及 expected SKU 集合派生 N，不接受客户端随意指定 Feed、条数或范围。拟议为 offset 0、请求准确 N 条的一次详细结果读取，N 不超过原总 SKU 上限 1000；不从旧窗口追加历史项。
2. 原 adapter 先执行有界读取，再核原 Feed 身份、响应完整性、准确 SKU 集合（无缺项/重复/额外项）、summary 与明细相符，以及原终态政策。只有所有条件满足才成为原生完整观察；整 Feed ERROR 仍沿原合同，不伪造 N 条失败记录。
3. 继续由原 Submission result owner 保存原回执/observation 身份、响应摘要、时间及 generation/head；沿原 CAS/UoW 找回与消费，不新建表、schema owner、恢复状态机或旁路证明服务。原资料/Store 权限、accepted 祖先、target/SKU fence 与新 claim 前的最后准入保持不变。
4. 拟议的单响应条数可能超过现行单页 50，属于**待澄清、待准入的合同差异**，不是本轮已批准变更。若后续采用，须先在本设计定稿原 typed observation 的表示/校验及相邻读取路径，再经适用复核和单独实现授权。新单响应不回填或重解释旧页文档，不把历史跨次覆盖重新签发为完整结果。
5. 保持原响应/文档/投影 2 MiB、总 SKU 1000、总请求 15s、远程阶段 10s 及既有调用上限，不追加探测、重试、轮询或放大费用/资源预算。一次结果若缺项、截断、计数矛盾、超过字节或时间预算，按原错误/部分结果如实记录且无新增修复资格；不接第二个响应拼接补齐后冒充此单响应。
6. 完整结果也只允许原政策中的准确失败子集经过新的显式批准进入后续提交；SUCCESS、INPROGRESS、SYSTEM_ERROR、TIMEOUT_ERROR、UNKNOWN 仍不因本候选变成可重发失败。不执行真实提交、保存或恢复。

如官方确认仍只支持 50，单响应候选不能覆盖已有的大 Feed；继续要求 §4.2 的适用跨请求一致性依据。拆小未来 Feed、重复扫描或手工确认都不能代替恢复当前 accepted Feed 所需的证据。两条来源路径若均缺依据，本候选保持 NOT_READY，#733 不收口。

## 5. 持久化、并发与失败语义

1. **读取前**：沿原入口重新核 member/org、资料及 Store 权限、原 accepted receipt、目标和准确预期 SKU。调用方不能提供任意 cursor/Feed 来跨租户或越过当前 head。
2. **读取中**：保留原请求 deadline 和页面边界；每页重新核原 scope。新来源证明如果失效、缺失或与原绑定不符，停止并保留可提交的原部分进度，不签发完整性。取消/中断也不能由“已读部分”推导未读 SKU 失败。
3. **保存时**：复用原 Begin/Save operation 身份、target/SKU 锁与 generation/head CAS。旧轮次/迟到/并行失败分支不得覆盖赢家，不拼接未提交页面，不自动倒退或复活旧完整观察。
4. **commit 结果不明**：按原 operation/receipt 查询提交结果；同键同载荷只回读，同键异载荷拒绝。不能盲目重抓平台、换 observation ID 或重复提交来猜测事务结果。
5. **进入新 claim 前**：原 `requireWalmartEffect` 在原 target/SKU 事务中重新核当前 head/generation、证据摘要、原 accepted 归属、失败子集、当前授权和 Store 绑定。读取权限检查不能替代该最后准入。原子推进 claim/fence/SKU 指针；已被新 generation、target 或 attempt 取代的证据不能使用。
6. **重启/响应丢失**：仍由原回执和观察 owner 找回持久化事实，不新增后台恢复责任方。跨窗口页仍走原只读投影；不触发发送、不终结 UNKNOWN、不延长租期。

若后续真实来源要求不同的状态、事务、持久化 owner 或外部副作用，停止并升级本候选；不得把这些变化夹在实现中。原权限撤销与 Store 漂移处理不降低，凭据、原始响应敏感内容和任意下载地址不得进入结果 UI/日志。

## 6. 资源、投影与用户操作

现行已批准的分页合同保持原值：响应/typed 文档与结果投影 **2 MiB**，准确预期 SKU **最多 1000**，每页 **最多 50**、一次遍历 **最多 20 页**，原 PublicationStatus 总请求 **15s**、远程阶段 **10s**。§4.4 中大于 50 条的单响应仅为有条件候选，未替换此合同；所有字节、总 SKU、时间、调用与费用上限均不提高。按整个文档试算后再接纳下一页；超限保留可提交前缀及原有限投影，不截去错误后伪装完整。新来源若不能在原预算内证明完整，保持受阻，不自动增大预算、拆提交或延长租期。

原结果入口继续区分当前原生观察、历史页面、时间、覆盖和 publication/可售事实。只在已存在的结果区域说明“一致性依据未具备，不能据此修复”；不把 coverage、publication、查无结果或平台回执成功当作整体刊登/修复成功。不新增手工批准一致性、上传报告或绕过门禁的入口。

## 7. 后续实现验证映射（本轮未执行）

只在设计准入及实现授权后，复用现有离线测试，不建验收工具或新测试平台：

| 待验证不变量 | 复用测试位置与有界补齐 |
| --- | --- |
| 本地完整不替代来源证明；终态与精确失败子集 | `internal/marketplace/walmart/result_consistency_test.go`、`repair_test.go`；100%覆盖但无证明/跨版本/同计数换SKU均拒绝；有真实来源合同后再定义正向合成fixture |
| 单响应候选不能只靠调大 limit 放行 | 仍复用原 adapter/flow、typed observation、result budget 与原 PG 测试；待合同准入后补 50/51/1000 条、服务端钳制/截断、集合/计数矛盾、超 2MiB/超时、原键重放和旧窗口不获资格的离线用例；本轮不编写或执行这些新回归 |
| generation/head/摘要与同键语义 | `internal/integration/persistence/listing/submission/walmart_effects_postgres_test.go`；沿原临时PG回归覆盖迟到、并发、篡改、commit未知回读和新claim最后准入，不接保留库 |
| 权限/Store漂移及当前读取接线 | `internal/app/supplychain/walmart_flow_test.go`、原 official adapter 测试；撤权、换商家、缺来源依据时不消费旧证明 |
| 原预算与原生行为不倒退 | `internal/listing/submission/walmart_result_budget_test.go`、原续读/大历史测试；2MiB边界、超时、空页和跨窗口仅可见不授予资格，原单窗口政策保持 |
| 消费端不把覆盖写成恢复通过 | 原 BFF/contract 测试及 `web/listingkit-ui/src/components/workbench/supply/walmart-feed-result.test.tsx`；缺证明保持禁用，strict DTO不接受客户端伪证据 |

这些是拟议测试义务，不是已发生的 RED/GREEN/故障验收。本轮仅适用文档自检；候选能力的专项业务回归、人工临时 PG、运行、真实保存/恢复、浏览器及产品验收均 **NOT_RUN**。自动 CI 中已有检查按 PR 的准确 HEAD/作业单独记录，不以旧 HEAD 的成功冒充新候选已执行。

## 8. Legacy 与停止条件

Legacy decision: **RETIRE**，已退休 Task/root ListingKit/平台独立 Workbench 继续退休，不恢复、不包装。Reusable behavior：当前 Submission owner 的准确回执、预算、显式批准、权限、fence/CAS 与 SKU 唯一归属。Current owner：当前 Submission / Store / Supply owner 不变，不经旧 owner 调用。Cutover/deletion condition：本轮无新增切换或删除，不新增 fallback、双读双写或旧数据迁移，不删除运行数据或源码。

进入 `IMPLEMENTATION_READY` 前必须同时满足：

1. 找到适用于当前 Feed 类型、可核查的完整一致结果来源合同：或澄清 §2.2 问题 1–3 并定稿 §4.4 的单响应合同与相邻校验，或满足 §4.2 的跨请求来源证明；均须说明真实字段/语义/适用性及原预算内的可消费方式。当前两条路径均未获准入，缺失继续 UNKNOWN。
2. 用户明确剩余能力的最小范围及任何必要变化；不以另一轮相同分页、手工确认或扩大预算替代证明。
3. 在获得单独任务授权后，由原唯一 Reviewer 做范围内架构检查，消除当前 Must 的真实阻碍；本轮不派工、不触发评审，不把文档 CI 当架构认可。
4. 设计获得准入后仍须单独取得实现授权，先补离线失败回归再最小实现；未变化的冻结设计和原有效证据复用。

因此本候选可供审阅，但当前不能承诺“只差代码”。源证明依赖尚未解除；#733 保持开放。保留实例、数据和会话，不合并、不部署、不真实启停/配置保存/采集/保存/恢复，不调用真实模型或平台，不发送邮件短信；历史 FAIL/UNKNOWN 不改写，本轮未执行项保持 NOT_RUN。
