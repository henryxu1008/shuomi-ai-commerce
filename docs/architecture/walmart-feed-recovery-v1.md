# Walmart Feed：明确拒绝的修复重试与有界续读

Refs #733 / PR #1028。

- Design Basis: **Independent Architecture**，只补本文件两处恢复合同，不重做全局架构。
- Admission: **IMPLEMENTATION_READY / FROZEN**。原唯一 Reviewer 对候选 `03037fc3d60baa5f3820d496249e65b3c3265ec2` 的[独立局部设计检查](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1028#issuecomment-6527391507)无设计 BLOCKER；§7 两项 IMPLEMENTATION_TEST 留同批实现收敛，不扩大本阶段范围。
- 用户决定：2026-10-30 明确同意当前唯一 Writer / 原唯一 Reviewer，先 #1027 最终工程检查，再本设计检查，准入后在同一 #1028 实现、自检和最终检查；不新增线程或拆 PR。
- 本文以 #1028 `9c7cb07ce9912f0bbb94b3484b31f0286c7a1919` 为调查基线。当前运行/真实保存/浏览器/产品验收 NOT_RUN，缺失证据 UNKNOWN。

## 1. 用户结果与阶段决定

运营人员处理一次修复 Feed 的授权拒绝后，可以重新检查并明确批准原修复载荷，以新命令再次提交；原键只读回原结果。查询较大 Feed 时，超过一次预算不再永远从第零页开始，下一次沿原“查询平台结果”入口继续读取并显示实际页面证据。

用户已确认本阶段分页先交付进度与可见性。**跨次页面不构成可用于 failed-SKU 修复的完整新观察。** 缺少完整一致证据的后续修复继续受阻，完整恢复要求保持未完成；不以本阶段交付关闭 #733，也不将延期写成 PASS/Accepted Risk。

Must：

- R1：只对新产生、原事实 owner 已持久化的明确 401/403 授权拒绝开放修复重试；不将历史 UNKNOWN 转换成失败。
- R2：同键只回读；新命令必须重新批准相同修复载荷、原 accepted 祖先和准确失败 SKU，重新核权限及现有证据；不重发成功 SKU，不绕过 pending/UNKNOWN、其他 Product 或商家占用。
- R3：每次读取仍有界，页面证据与下一位置同事务；并发、崩溃和迟到结果不能丢已提交进度或覆盖新轮次。
- R4：原生本次观察、历史页面及发布/可售事实分开显示时间与覆盖。页数、summary 相同或 PROCESSED 均不能证明跨次快照一致。

Out of Scope：自动重发、旧 UNKNOWN 恢复/清理、改成功 Offer、任意 SKU/Feed 查询、新轮询器/runner/验收平台、租期延长、真实平台或保留环境操作、跨次完整结果自动签发修复资格。

Threat Model 沿用 [原设计](walmart-us-v1.md) §1：本系统重放、并发、授权漂移和响应/commit 丢失；不扩大到商家其他系统的自动对账。用户未接受新风险。

## 2. Product / Figma / 技术依据

复用 [原 Walmart 设计](walmart-us-v1.md) §2 的最终 UI/IA 与 Figma `tg48P46SSXl6TBy9lZwg63` / `31:463`、现有供应链资料及上传结果“查询平台结果”。只在原结果组件显示读取覆盖、页面时间与继续进度，不新增菜单、平台独立页面或额外人工验收步骤。

复用原 ExecutionKernel、同 Product DB official UoW、Target/Asset/Store 权限合同及 [greenfield 基线](../product/greenfield-no-legacy-migration.md)。

官方依据仍为 [Feed item status](https://developer.walmart.com/us-marketplace/reference/getfeeditemstatus) 与 [Feeds overview](https://developer.walmart.com/us-marketplace/docs/feeds-overview)：includeDetails 的 offset 从 0 开始，每页 limit≤50；PROCESSED 可能仍含逐项 INPROGRESS。本轮只读文档，没有调用平台。未证实跨请求快照/稳定排序版本保证，保持 UNKNOWN；不以相同计数推导该保证。

## 3. 唯一 owner、持久化与调用路径

| 事实 | 唯一 owner / 现有载体 |
| --- | --- |
| 原批准 Target / wire / SKU / 商家 | 原 Target record 与 OfficialIntent |
| 提交、明确拒绝、claim/fence | 原 ExecutionKernel attempt / target fence |
| 原 accepted Feed | 原 immutable OfficialReceipt |
| sellerSKU 归属及当前提交者 | listing_walmart_sku_bindings |
| 原生结果、本地页进度及页面证据 | 原 Submission result repository / listing_walmart_feed_observations |
| 身份、Store grant、凭据 | 原 Store owner；不在 Product DB 重建授权事实 |

实现路径：原 supply Upload/PublicationStatus → Submission domain 合同 → 原 submission/official persistence → currentapplication 原注入 → 原 HTTP/BFF strict DTO → 原 Walmart 结果组件。平台 HTTP 仍只在 integration/walmart；原 state machine 不新增状态。

新增的页进度放在原 observation 的 typed JSON 文档内，不新建表或第二事实源。文档组合原 FeedObservation 和可选 ReadProgress；单次原生完整观察仍是当前合法类型，不加旧 owner wrapper、双格式 fallback 或旧数据迁移。ReadProgress 缺失表示没有续读进度，不能从 items 长度猜位置。

## 4. 修复被明确拒绝后的恢复

### 4.1 先保存明确失败

移除本轮 NewWalmartRejectionEvidence 对 RepairFrom 的临时拒绝前，必须先完成以下窄准入。

只有原 adapter 完整收到的 ErrRejected（401/403）可形成 provider_response / failed_definitive。证据绑定原 intent、payload fingerprint、correlation 与固定拒绝类别；原 Complete 同事务更新 attempt/fence，保留 SKU 归属、原 ReadGeneration 与 accepted 历史，不制造 Feed receipt、不把 owner 指针回滚到旧 attempt。

failure COMMIT 不明只读原 intent/attempt；未取得同一拒绝证据继续 UNKNOWN。崩溃、超时、5xx、截断、无 feedId 或发送结果不明不进入本分支。历史 UNKNOWN 不回填，重授权也不清 fence。

### 4.2 新显式命令的原子准入

在原 Acquire 的 target lock / transaction 内：

1. 读取当前 fence 的 immediate attempt。它必须为已核实 failed_definitive，证据与该 attempt 的原 Walmart intent 精确匹配，且该 intent 带 RepairFrom。
2. 新批准 Target/intent 必须是相同 org、Store、永久商家/环境、Product/source、RepairFrom、SKU 集合、Feed selection 与 wire/payload。凭据/授权可以按原 Store 合同更新，不能因此换商家、SKU namespace 或绕过新权限。改载荷/集合须走原重新准备路径，不属于这次“同载荷被拒重试”。
3. 按原 attempt target/fence epoch 查找当前拒绝之前最后一笔 succeeded 的 accepted attempt；原 RepairFrom 仍为实际失败 SKU 的 accepted 祖先，二者可能不同。原 repository 分别核验准确 receipt，不从标题、时间窗口或任意客户端 feedId 猜祖先。
4. 复用原正常修复政策：最后 accepted Feed 必须有完整终结且允许继续的原生观察；RepairFrom 的最新有效完整原生观察必须允许准确失败 SKU。分页混合观察不能满足此条件。最新观察缺失/partial、已有读取 generation 未完成或仍有 pending/UNKNOWN 均拒绝。
5. 锁原 accepted 祖先及最后 accepted Feed 涉及的 SKU 行，范围最多原 Product 的1000 SKU。仍由 accepted receipt 持有的 SKU 继续以 read_generation/latest_observation_id 核对当前证据；本次选中的失败 SKU 必须仍归当前明确拒绝 attempt、同 Product/source/Store，而不是要求其 owner 被回滚为祖先。对它们按重新核实的祖先证据与当前拒绝链裁决，不把已清空的旧 latest_observation_id 伪装成当前证明。
6. 校验、SKU LastAttemptID 推进、新 attempt 与 fence epoch 一起提交。任一不满足都不发 POST，不新建分布式事务或补偿状态。连续明确拒绝只查看当前拒绝和上述原生 accepted 证据，不靠遍历无界拒绝链。

同键原终态保持可读；新键没有用户重新确认不得自动生成。成功/处理中 SKU、跨商家/跨 Product、原 payload 不同、晚到旧结果及竞争新 claim 均保持原阻挡。已经存在的正常初次提交和 accepted 修复准入不变。

## 5. 有界分页进度合同

### 5.1 最小数据

在原 Submission domain 定义 typed ReadProgress，并与原 FeedObservation 组成同一 persisted JSON 文档：

- Round：本轮第一次 read generation；ReceiptID/FeedID 与 expected-SKU hash 精确绑定。
- StartOffset / NextOffset：实际零基页位置，范围0..expected数量；按真实完整页推进，不从 items 条数猜未读页。
- PreviousWindowID：同一轮上一个已提交真实页面的窗口，原 observation ID；零 item 的真实回执仍是页面，无响应窗口不新增空链。
- Pages：本次窗口各页的 offset、item count、response hash 与原生 observedAt，最多20页；Items 按这些真实页归属，不能把以前页面改写为当前观察时间。

JSON、总响应、错误文本和 SKU 数量沿原2 MiB/1000 SKU预算；超限停止且保持partial，不截断后声称完整。不引入新表、job、租约或独立进度 owner。

### 5.2 Begin / HTTP / Save

1. BeginWalmartRead 在原 target/SKU lock 内读取**当前 latest_observation_id 指向的已提交窗口**作为 anchor，验证 body hash、receipt、expected hash、Round/offset。禁止按最大 generation 任意拾取迟到的非 head 结果。
2. 仍递增原 read_generation；保留原 latest_observation_id 作为中断续读 anchor，而不将其称为新 generation 的完整证据。现有修复准入仍要求 observation generation 与 SKU read_generation 完全相等，所以 in-flight read 不会继续授予旧证据修复权。返回 BaseObservationID 与窗口起点，供 Save 校验。
3. 原用户查询有界读取 offset≤1000、每页≤50。每页重核 source/Store/授权，使用原短生命周期 reader；长读不占 mutation permit。收到完整合法页才加入本次窗口，超时/失败保持最后确认的位置。
4. Save 在原事务中验证当前 generation、BaseObservationID、准确 receipt/expected binding、连续页面与 hash/time/count，append immutable observation，再以原 generation/anchor CAS 推进 SKU head。重复相同 save 读回同一结果，不同内容冲突。若 generation 已落后，页面可保留为历史，但不能推进当前 head。
5. 响应丢失后下一次读从原已提交 anchor 继续；若事务没有提交则重读原 offset（GET可安全重复）。无响应/崩溃不从0重来，也不跳未提交页。
6. 本轮达到末页/short page后，下次同一查询入口开始新 Round 从0刷新。Feed summary 变化不把已提交游标退回0，否则处理中 Feed 仍可能永远读不到后页：同一 HTTP 窗口若遇 summary 差异，就止于最后一致页面，下次从尚未提交的页继续；不同窗口各保留自己的 summary/时间，累计只能是partial。权限/身份变化按原合同拒绝，不携带旧 cursor 穿越企业/Feed。

### 5.3 投影与修复资格分开

ReadWalmartResults 从真实 head/anchor 读取本轮有界窗口链，最多20个非空窗口；严格递增 offset，核 receipt/round/expected/hash，不消费未赢得 CAS 的分支。保留每页时间，按 SKU 展示观测到的原值及覆盖，重复/多余/缺失SKU清楚标partial；不把1000个混合历史项计为全量本次结果。

原 ResultRecord 增加 bounded read-progress / observed-pages 投影，BFF strict schema 与原结果组件同步消费。原 Observation 只表达本次 HTTP 读取窗口；历史页不复制进当前 Observation.Items 或改新 ObservedAt。总投影超限保持partial/覆盖受限，不制造完整视图。

只有从 offset0 开始、同一次有界读取已完整覆盖准确 expected 集合、满足原完整性校验的原生观察才保留原修复资格；跨窗口/跨次累积的 read-progress 永远不能使 AllowsRepair 成功。先前完整证据在新 generation 开始时失效，本轮缺证据就继续阻止修复。

预算实施收敛：接入下一页前按整个 typed 文档试算，超限保留原可提交前缀；可选 publication 信息也不能撑爆已经拟合的窗口。历史投影超限只显示有界部分并标记 `limited`，不能使下一次查询在 Begin 前失败。整个 Publication 共用原 2 MiB 响应预算，保留各 live receipt 的查询身份；覆盖受限时可只投影持久化 `readOffset` 并省略明细/页证明，显示 partial、不授予修复，绝不改写已存原生事实或游标。此为 R3/R4 的 IMPLEMENTATION_TEST 收敛，不增加状态或新事实 owner。

## 6. Legacy 与接口边界

Legacy decision: N/A。只使用当前 owner；不恢复旧 Task/queue/workbench，不增加兼容层、历史映射、双读双写或数据迁移。typed JSON 组合是当前单事实模型的扩展，不建立第二套 native result owner。

不改真实权限/配置；controller/worker 原注入不新增服务。只改原 Walmart 结果端口和 strict DTO，不开放客户端自报 cursor/feedId/企业，不新增控制面或通用 SDK。

## 7. 验证与准入

首轮局部设计检查的两项收敛点分类均为 **IMPLEMENTATION_TEST**，不新增设计 BLOCKER：

- R2/R3：Submission 的 persisted typed document 整体计算 canonical hash；Begin、Save重放、ReadWalmartResults 以及 requireWalmartEffect 的最后 accepted / RepairFrom 两处读取统一解码校验。不能只解析/hash旧的原生 FeedObservation 而忽略进度；补 progress 篡改和未赢 CAS 分支负向。
- R4/现有权限 Must：原 WalmartFeedResult 的显示与异步请求绑定 scope/source/store/receipt 和请求代次；输入改变立即隐藏旧显示、取消旧请求、拒绝晚到响应。原生、历史页及 publication 时间分别显示；只复用原组件回归，不新建验证工具。

复用原 flow / synthetic HTTP transport / task-owned PostgreSQL fixtures，TDD先RED后最小实现：

- 拒绝修复：同键回放；明确拒绝后用户同载荷新命令；连续拒绝；完整原祖先/最后accepted不同的情况；并发仅一次claim；改SKU/payload/merchant/权限或原证据partial仍阻挡；失败COMMIT未知/响应丢失保留原读回。
- 页进度：1000 SKU 慢读取预算耗尽后继续较晚页；重建reader/service后续读；save失败/响应丢失；并发/迟到CAS；summary变化；missing/duplicate/extra SKU；跨次满覆盖仍无修复资格；一次完整读取仍沿原资格；source/Store撤销阻止继续。
- 正常消费者：Go PublicationStatus → 原 repository → BFF decoder → 原结果组件，覆盖进度/原生与历史页时间/partial文案；不增加产品人工验收矩阵。
- #1028 既有初次提交拒绝与preflight、原 other-platform tests按受影响面验证，未变代码有效证据复用，完整CI只给稳定候选。

独立设计检查仅核本文件新增恢复/事实边界，最多两轮正常检查；达到 IMPLEMENTATION_READY 后同一 Writer 实现，具体finding只复核增量，最终同一 Reviewer 核实际 diff/有效证据。

合并、部署、关单、真实平台及保留环境操作仍无授权。技术回归/CI不等于运行、真实保存、浏览器或产品验收。
