# TikTok 发送前未执行的终止与显式重试：设计候选

- Execution Issue: [#669](https://github.com/henryxu1008/shuomi-ai-commerce/issues/669)，同一 Draft PR #1031。
- Design Basis: **Independent Architecture，局部恢复合同增量**。
- Admission Status: **DESIGN_CANDIDATE / NOT_IMPLEMENTATION_READY**。
- Authority：2026-10-31 用户仅授权设计 no-send 证明、原 claim/fence 的 CAS 终止、原键重放与新尝试规则；**暂不实现、不新增 Reviewer**。本文件不是实现、评审通过或上线授权。
- 原 [TikTok 首版设计](tiktok-shop-us-v1.md)及[官方合同依据](tiktok-shop-us-contract-evidence.md)其余范围保持冻结；本候选只处理 #670 的发送前失败恢复 finding，不重新设计整个 Submission。
- 唯一设计/写入负责人沿用 #669 当前会话与独立工作区；保留实例、数据和会话，不部署、不真实启停或保存配置，不调用真实平台/模型，不发送邮件短信。

## 1. 用户结果、边界与设计决定

美国跨境 TikTok 店铺运营人员明确确认上传后，若当前 worker 在源资料、图片或店铺授权复核中失败，而且**尚未进入平台 mutation 调用**，这一次应有准确的“未发送”结果。处理缺项后，用户可以沿原上传入口重新确认；原操作回执保持原结果。

本候选建议同时采用两项决定，均待用户确认及适用独立架构复核：

1. 仅在**仍存活的原执行者**持有 Submission 创建的一次性发送门禁、且它能不可逆地禁止该执行者后续发送时，允许提交 `not_sent`。没有这种证明的进程中断、重启、超时和已有 UNKNOWN 继续保守关闭重发。
2. `not_sent` 是原键的永久终态；不自动重试。只有用户重新确认产生的新 Operation/item command，重新检查全部当前准入后，才能竞争新的 fence。成功前置回执仍优先复用；UNKNOWN 仍阻止新命令发送。

本次不做：从历史 UNKNOWN 推测未发送、进程重启自动解锁、通用 dispatch/outbox/Saga/reconciler、后台重试、人工填写成功、其他平台接入本能力、旧数据迁移、保留库升级或真实平台验收。

产品/Figma Authority 复用 #669 已确认的美国跨境店铺与现有供应链确认上传/操作结果页面，以及 [final-ui-ia-authority](../product/final-ui-ia-authority.md)。只增加原结果中的准确状态和说明，不建新页面、按钮或人工恢复工作台；若实现需要新的操作流程，先报告，不在实现中追加。

## 2. 当前根因与为什么不能只放开 fence

基于 PR #1031 设计前的生产代码核对：

- [ExecutionKernel.Acquire](../../internal/listing/submission/execution_kernel_service.go) 提交 `claimed` 后直接把 `SendPermit` 交给调用者；[Kernel 状态合同](../../internal/listing/submission/execution_kernel.go)只允许 provider 结果终结已取得的 claim。
- [TikTok stage](../../internal/app/supplychain/tiktok_upload.go)在取得 permit 后还会复核来源/资产/规则、取得授权 merchant、读取并核对图片字节、解析载荷；这些步骤未调用 mutation 也可能失败。当前统一记 UNKNOWN，target fence 不能再取得。
- 直接增加 `Abort(claim)` 不足：若原调用者仍持有可用 permit，数据库释放后它可能迟到发送，与新尝试重复。没有结果/没有 request ID/进程已经消失都不是 no-send 证明。
- 图片和 global-create 当前子键绑定 record/target，却不绑定外层 item command。若仅把失败写成永久 `not_sent`，即使外层换键也仍会回放同一子键，重试路径不能完成。
- [Operation consumer](../../internal/app/supplychain/operations.go)遇到普通依赖错误会重试原 durable command；no-send 必须作为确定终结结果进入原 item receipt，不能只返回一个触发通用重试的 error。

因此最小改动同时需要：**一次性发送门禁、同库终止事实、明确新命令的子键与消费规则**。这是恢复合同改变，不能把它当作普通条件判断补丁。

## 3. 唯一 owner、调用链与不变量

| 事实或职责 | 当前 owner / 本候选的最小消费方式 |
| --- | --- |
| live 身份、企业/成员/店铺、资料/资产、规则及 service 授权 | 原 IAM、Store、Product/Asset、Target application；所有现有准入保留 |
| 原 intent、attempt、claim token、fence epoch、no-send 终态 | 唯一 `internal/listing/submission`；使用 Product 库原 Submission 两表事务 |
| 本次进程是否已经交出发送能力 | Submission 创建的私有一次性门禁；不是可持久化/恢复的第二业务事实源 |
| TikTok 图片/global-create/publish 的实际调用 | 原 Supply application + Store server-only mutation handle；不能在门禁开始前调用 mutation |
| 原 Operation/item 的确定结果与显式新确认 | 原 `listing/preparation` 和 `OperationActivities`；不增加重试 owner |

调用链：原 Console/BFF → 当前授权的 Operation → item command → TikTok stage → 原 OfficialIntent → Submission 准备 claim/门禁 → 保留的发送前复核 → **no-send CAS 终止或一次性进入发送，两者互斥** → 原 mutation/原 completion UoW → 原 item 结果。

必须保持：同键不同载荷冲突；一个 target 同时一个占用；provider/key/receipt 身份不变；任何旧 epoch 晚到写入拒绝；成功步骤不可重建；UNKNOWN 不按新键自动绕过；跨企业/成员不可读不可发；凭据 server-only；不改变收费、权益或商业 owner。

## 4. no-send 证明：先封住本执行者，再终结数据库占用

### 4.1 一次性、不可恢复的门禁

拟在现有 Kernel 内提供仅面向 `TikTokImageAction`、`TikTokGlobalCreateAction`、`TikTokPublishAction` 的准备入口。它复用原 reservation、claim token hash、target lock 和 intent 验证，取得 `claimed`，但**不给 Supply 裸 SendPermit**，只返回私有构造的准备句柄。

该句柄持有一个不可从外部重建的共享门禁。门禁只有三个进程内状态：可准备、已进入发送、已关闭。所有句柄拷贝共享同一门禁指针；不能因 Go 值拷贝复制两份“尚未发送”状态。零值、JSON、读取 attempt、重放 intent、重启及持久化 token 都不能重建句柄。复用标准同步原语与现有 claim 能力，不建设 session/runner。

- **进入发送**：门禁原子地从可准备改为已进入发送，只能一次，然后才交出原 SendPermit 给唯一 mutation 调用路径。此后任何 no-send 证明请求均拒绝。
- **终止未发送**：门禁原子地从可准备改为已关闭，先不可逆地阻止后续取 permit，再由 Submission 内部形成 no-send 证明并执行 CAS。数据库失败也不能把门禁改回可发送。
- 两者并发只能一个成功；必须测试门禁的同指针复制、迟到任务与重复方法调用。不能靠“当前函数通常是串行的”代替互斥。

现有普通 `Acquire` 的裸 permit 不能转换为这个准备句柄，也不能调用 no-send 终止。新 TikTok stage 统一使用准备入口，不保留其旧直接 Acquire 分支或 fallback。其他当前平台仍消费原合同，不自动推广新能力。

**证明的可信边界**是受信任的服务端 Submission/Supply 实现与原 capability，不是客户端的 `notSent=true`。当前系统并不防御恶意修改后的服务端二进制；本候选不扩大该 Threat Model。私有构造和唯一调用路径必须由实际代码/原测试证明，而非仅以字符串或调用者声明担保。

### 4.2 哪些错误可以进入该证明

保留 Acquire 后的源/Target head、Product/Asset、规则、当前权限和 Store binding 复核。图片字节读取/散列核对、所有分支的确定载荷解析也在门禁开始前完成；这些步骤的失败可请求 no-send 终止，但**只有终止事务确定成功才回报 `not_sent`**。

原因只取有限安全代码，例如 `preflight_unavailable`、`facts_changed`、`access_recheck_failed`、`image_content_changed`、`payload_invalid`；不保存 provider 正文、URL、token 或原异常文本。

保守的分界是**进入 merchant 的 UploadImage/CreateGlobal/CreateLocal/PublishUS mutation 方法之前**。其内部还可能做只读授权检查；一旦已进入该调用包装器，即使错误看起来发生在网络写入前，也不再签发 no-send。这避免本次扩成 transport 内部 dispatch 检测项目。该界限覆盖原 finding 中 stage 的 facts/merchant 获取失败及相邻的图片/载荷预检。

真正发送时，原 Store handle 的即时授权、有效期和单次 consume 校验继续执行。若预检后权限漂移，handle 拒绝；不能为了 no-send 删除最终检查。门禁已经开始时继续保守 UNKNOWN。

no-send 仅证明**本阶段受保护的 mutation 未进入**，不声称没有执行过规则/身份读取，不证明零 API 成本，也不撤回先前成功图片/global 步骤的副作用或退还任何计费。

### 4.3 拟议的最小 Port 与装配

以下是待确认的合同名称，不是已存在 API：`PrepareTikTokExecution(command)` 返回原 attempt 和准备句柄；`BeginTikTokSend(handle)` 一次性交出 SendPermit；`StopTikTokBeforeSend(handle, reason)` 关闭门禁并提交原 claim 的 no-send 终止。`ReadIntent/Get` 只读，任何重放都不返回可发送/可终止句柄。

准备入口只在原 reservation/target 事务确定成功、原 lease 仍有效时签发句柄；事务结果不明、replay 或过期均无句柄。Begin 还检查原有界 context/lease，不能以进入门禁延长许可。Supply 的原窄 Kernel Port 消费这些能力；current application 仍注入同一 ExecutionKernel/官方效果 Repository，不增加第二装配或 provider owner。普通 Complete/UNKNOWN/official completion 保留原路径。

终止和读回复用现有有界收口 context/ExecutionTimeout；客户端取消不使本执行者重新获得发送能力。超时或数据库不可用就停在原不确定路径，不新增无限重试、后台任务或恢复端点。

## 5. 原两表内的最小终止合同

建议新增一个明确终态 **`not_sent`** 和证据种类 **`local_no_send`**，不伪装成 provider 拒绝、manual cancellation 或成功。仍在原 `listing_submission_execution_attempts` / `listing_submission_target_fences` 保存；不新增表、列、dispatch 日志或第二套状态机。

no-send 证据只由 Kernel 的已关闭门禁生成。它绑定原 organization、attempt/intent、target/action、payload fingerprint、claim owner/token、fence epoch 与安全原因；不提供可由 HTTP 或任意 caller 填写的公共 evidence 构造入口。Repository 校验原 claim token hash 和完整身份，不凭一个 reason 放行。

单个原 Product DB 事务：

1. 复用原 attempt/target 的锁顺序与事务工具，锁定原 attempt 和当前 target fence。
2. 核对 organization、attempt、intent、target/action、payload、owner、token hash、epoch，以及 fence.current_attempt_id/current_status 与 attempt 一致；仅原 TikTok intent/action 可用。
3. 在锁内复用 Repository 原 `mutationTime` 与微秒规范化/单调校验，复核 `status=claimed` 且原 lease 尚有效；不信任调用者时间，也不另改整个 Kernel 的时钟合同。已 UNKNOWN、已进入其他终态、过期或被后继 epoch 替代时拒绝，不做 UNKNOWN → not_sent。
4. 同事务写 attempt `status=not_sent`、`unknown_reason=NULL`、`evidence_kind=local_no_send`、`evidence_outcome=not_sent`、allowlisted reason、finished/observed/updated 时间；reference 为原 attempt ID，fingerprint 由 Submission 规范化身份与原因生成。`evidence_authorized_by=NULL`，因为它不是人工授权结果。
5. 同事务只把**原** fence.current_status 改为 `not_sent`，保留 current_attempt_id 和 epoch；不删除 fence、不预先递增 epoch、不插入官方成功回执、不改 Target/Product/Asset。

原两表 CHECK、Go 读写/重放校验和 schema verifier 必须一致扩展：`local_no_send` 只能对应 TikTok `not_sent`，无 provider response/readback，finished/observed 值合法且结束时间在 lease 内。原 `Complete` 和 `ResolveUnknown` 不接受这种 evidence，不允许据此签发成功或释放 UNKNOWN。

相同 proof 的重复终止：若原 attempt 已保存匹配的 no-send 证据，仅返回原事实。即使已有后继 fence，也不得重写当前 fence；若 tuple/reason 冲突则拒绝。首次终止响应丢失时，本执行者门禁保持关闭，只读回原 attempt；读不清就报告不确定，不直接放开或发送。

## 6. 原键重放、新尝试和阶段回执

### 6.1 原键永远不重新发送

同一 organization + intent key 保留唯一原 attempt。相同键/相同载荷回放原 `not_sent`，无新 permit；相同键/不同载荷冲突。通用 worker 的网络重试、同 Operation 重投递和页面刷新都不能把它变成新尝试。

原 Operation 的已终结 item 优先返回原结果。进入 Upload/stage 时也必须先检查**本命令的准确 stage keys**的原终态，再考虑其他命令的最新 target 成功回执；不能在后继已成功后把老命令的 not_sent 改说成老命令成功。所有读回仍先做当前身份/源/店铺访问检查，原 receipt 不绕过撤权。

### 6.2 新确认才产生新子键

新 Operation 的 item command 沿原 `ItemCommandID(operationID, sourceID, action)` 取得；不新增前端隐形 retry 或后台 key 生成器。

拟将 TikTok 图片/global-create 子键统一用现有 `collection.StableID` 从 **organization + 本次 item command key + stage + Store/target subject + record ID** 派生；publish 仍绑定本次 item command。编码必须复用现有无歧义稳定 ID 实现。相同命令重投递各子键不变；重新确认才改变未成功步骤的子键。provider idempotency key 继续由原 intent key 生成，不另设可覆盖的 provider key。

顺序是：读回本命令原终态 → 对没有原终态的步骤核对并复用相容的成功回执 → 当前 freshness/权限和明确批准 → 准备新 intent/claim。图片/global-create 已成功时继续使用原不可变回执，不重新上传或创建；只有未执行步骤竞争新 fence。

新 claim 仍使用原 target 唯一锁。在锁内核验原占用：`not_sent` 必须有合法匹配证据；claimed/UNKNOWN 拒绝，新键不构成证据；succeeded 走原回执复用；其他已有终态保持各自原合同，不扩大其重试权限。允许接续时才创建新 attempt 并把 epoch 从 n 改为 n+1。并发新确认仍最多一个取得当前占用。

新尝试不沿用旧的可变授权快照。当前 source/record、资产、规则、grant/connection revision、服务、用户确认的载荷与阶段前置回执全部重新核对；发生漂移时先重新校验资料与用户确认。no-send 只是“原阶段未产生 effect”的事实，不授予任何新业务权限。

### 6.3 现有消费者必须接通

`UploadResult/UploadAttempt` 如实投影 `not_sent`、原 attempt ID 和安全说明；沿原严格 DTO/BFF 增加该值。原 `OperationActivities` 把确定的 not_sent 作为本 item 的终结失败结果（`ItemFailed`），保存 result reference 与“本次未发送，处理缺项后可重新确认”的说明；不返回会触发依赖重试的普通 error，不改原 Operation 状态机。

如果 no-send 事务/读回仍不确定，继续走原 UNKNOWN/原键查询路径；不能把“终止调用失败”投影成确定未发送。失去当前权限时仅允许内部原 claim 做必要收口，不能因此向用户额外暴露原数据或授权新发送。

## 7. 失败、中断与竞态结果

| 边界 | 确定结果 / 下一步 |
| --- | --- |
| 准备 claim 尚未提交就失败 | 无新发送能力；沿原键查询，不猜已提交/未提交 |
| 原执行者仍活着，门禁未开始，预检失败且 no-send CAS 提交成功 | not_sent；原键只回放，用户可重新确认新命令 |
| BeginSend 与 no-send 终止并发 | 同一门禁只有一方成功；先关闭则不可能取得 permit，先开始则不再允许 no-send |
| no-send CAS 丢失响应 | 本进程不发送；读原 attempt。读到确切 not_sent 才报告未发送，否则保持不确定和原占用 |
| 进程在取得 claim 后崩溃，未提交 no-send | 不能恢复门禁或以“没有请求日志”证明未发送；原 lease 到期后 UNKNOWN，不自动新发 |
| 门禁开始后、实际 mutation 调用前崩溃/取消 | 保守 UNKNOWN；这是有意保留的不确定窗口，本候选不增加 durable dispatch phase 来恢复它 |
| 平台调用超时、5xx、响应丢失、结果/本地 completion 不确定 | 原 UNKNOWN、原键/目标占用和既有 qualified readback 合同；不转 not_sent |
| 过期 claim、旧 epoch、错误 owner/token 或跨企业证明晚到 | 拒绝；不得改变新 attempt/fence。合法旧 not_sent 的重复读回也不能覆盖新 fence |
| 图片成功，global-create not_sent | 原图片回执保留；新确认仅竞争未成功的 global-create，再按原回执链发布 |
| global-create 成功，publish not_sent | 原全球商品/SKU 回执保留；新确认仅竞争 publish，不重新创建 global 商品 |
| 终止事实成功，Operation item 收口前进程中断 | 重投递同一 item/stage key 只读回原 not_sent，再完成原 item；没有新 permit |

新方案保证的是**可证明的当前进程未发送失败**能够终结，不保证所有“实际上没发送”的崩溃都能恢复。旧 UNKNOWN 不因本设计被重分类。

## 8. 验证方法、投入与实现停止点

只复用现有 Kernel/PG/Upload/Operation/DTO/BFF/UI 测试，不新增 runner、验收 session 或专项平台。未来获准实现时先 RED，再最小 GREEN：

| 不变量 | 原设施中的必要验证 |
| --- | --- |
| 门禁不能既发送又终止 | Kernel 单测/并发：BeginSend vs Abort、共享门禁的句柄复制、重复/迟到调用、零值/反序列化/原键重放无句柄 |
| no-send 绑定原身份与当前占用 | 原临时 PG：错误 org/token/owner/payload/epoch、过期/UNKNOWN 拒绝，no-send 事务中断前后及响应丢失读回 |
| 原终态与新 fence 原子且不能回退 | 原 PG：旧 proof 幂等回放不改后继，两个新命令竞争，一个获得 n+1；读写/SQL CHECK/verifier 一致 |
| 无变相新发送 | 原 TikTok Upload：facts/merchant/图片/载荷失败在门禁开始前零 mutation；真实调用错误仍 UNKNOWN |
| 原键稳定，新确认才能接续 | 原 Operation→item→worker：原 item 重投递不发，新 Operation 的子键变化，old not_sent 不被后继成功覆盖 |
| 阶段不重复 | 复用原两种 listing method 和图片/global/publish fixture，保留成功步骤、准确 Shop/SKU/载荷；UNKNOWN 前置步骤继续阻断 |
| 权限与消费链完整 | 原撤权/Store handle、严格 DTO/BFF 与 UI：not_sent 是失败而非成功/UNKNOWN，当前失权不能新发，显示原 attempt/安全原因 |

原 `TestTikTokLocalDraftAndActualSendImageDrift` 的历史 UNKNOWN/PASS 仍按原 SHA 保留。未来在新门禁尚未开始、no-send CAS 已确定提交的分支，才将新代码预期明确改为 not_sent；门禁开始后的漂移/错误仍保持 UNKNOWN。不能只改断言制造通过。

实现代价包括现有 Submission 域/Kernel、两表 CHECK 与 verifier、TikTok stage/子键、Operation/result DTO/BFF 和原测试；不是只加一个状态值。仍留在同一 #669 / #1031、单 Writer，不新建服务、表、调度器或恢复操作台。复用原 PostgreSQL/GORM 事务、锁、claim token 校验、稳定 ID 与标准同步组件，无成熟组件缺口需要引入通用框架。

Legacy decision: **N/A**，仅当前 owner 的新合同；遵守 [greenfield baseline](../product/greenfield-no-legacy-migration.md)。未来只在获准的任务独立新空测试库中修改/校验 schema；本设计不提供旧表迁移、UNKNOWN backfill、双读双写或旧 TikTok 发送入口兼容。保留实例继续原版本/原数据，不触碰；如后续要求接入不匹配的新 schema，先报告并取得单独方案/授权，不隐式升级。

当前交付仅本设计和索引/引用。**用户尚未确认这两项设计决定，独立 Architecture Review NOT_RUN，生产代码/正式 schema 修改 NOT_RUN；不得标为 IMPLEMENTATION_READY 或据此继续实现。** #1031 既有三项修复及其 CI 是各自原 SHA 的工程证据，不是本候选恢复协议已经实现或验证的证据。运行、真实保存/恢复/刷新/刊登、浏览器与产品验收保持 NOT_RUN，现场缺失证据 UNKNOWN。
