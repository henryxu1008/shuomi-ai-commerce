# 来源商品白底图与资源结算合同

现行 Product/Image 入口对已授权来源商品生成单张白底图候选，经人工预览和明确批准后进入当前 Asset owner。完整图片集合的操作入口见[商品图片智能体](../operations/product-image-agent.md)。

当前调用使用 Product/Image、ImageAgent、Resource 和选定 provider 的既有 owner。以下是固定 Image owner 的跨库不可变回执、派发顺序、UNKNOWN 恢复和成员限额合同；不开放通用事务代理，不以文档存在证明特定实例已经启用或通过真实平台验收。


本合同约束固定 Image owner 的跨库资源调用；实现与开放状态由当前执行 Issue 单独确认。`image-db:5437/image_agent` 与 commercial DB 是不同容器数据库，`worker/dependencies.go` 分别建连接。`orgresource.TransactionalReservationOwnerStore` 的同事务锁 owner/Bind/terminal-proof 不能直接满足；现生产也没有 ReservationAuthorizer 或已注册的 reservation owner。最小改动采用下述**仅 image_generation_v1** 的显式 receipt-based 入口，保留其他 owner 原共享事务合同。不迁库，不复制 V3 row，不新余额表、Saga、runner 或定时扫描服务。

### A. 唯一事实与固定身份

- ImageAgent V3 attempt 是生成意图、dispatch fence、provider效果和真实token审计的唯一 owner。现 effect row 增加一个受限 generation JSON fact（schema version；immutable intent；reservation binding；dispatch state；immutable outcome；settlement receipt reference），不新建通用 invocation owner。原 staging/publication phase 独立，不能用它覆盖生成效果。
- `intent_id` 固定来自 org/run/planRevision/slot/attempt（不得把变化的 quote hash 放入该主键逃避冲突）。fingerprint 包含 actor/canonical member、exact Catalog version/publication/hash、source bytes digest、prompt/policy version、route/provider/model/credential config identity（不含secret）、统一协议/1K/quality、服务端 price version 与正整数 AI point quantity、resource=ai_point。价格相同但版本不同仍冲突；原attempt不在重放时重新报价。
- 配置 owner 是现运行配置装配的版本化 image price 小合同：明确 provider/model/resolution/operation 与 points；没有配置或不匹配就是 unavailable。配置值不来自浏览器/Figma样例；本片不新增价格管理UI、月度额度、图片资源或token换算。
- 资源 owner 仍只使用现 `saas_organization_resource_operations/reservations/buckets/events/audit_logs`。资源 operation 的不可变请求摘要、receipt/evidence hash 是其扣占依据，不是第二份可变 V3 状态。generation fact 只保存资源receipt身份/摘要，不复制余额作为authority。

### B. 窄入口与现合同的精确替代

在 `internal/integration/orgresource` 显式构造 `ImageGenerationReservationExecutor`，仅注册常量 image_generation_v1 + ai_point；继续消费 ledger 的正quantity、fingerprint、余额/负债规则。它接一个**内部 owner reader**，只按身份读取 V3 已持久化 intent/不可逆 terminal proof；API不收proof/outcome，不能靠caller拼 succeeded。reader通过ImageAgent owner方法读取，其跨模块桥在 integration/app装配，不把商业gorm tx传ImageAgent DB。

窄入口为 `ReserveImageGeneration` / `ReadImageGenerationReservation` / `FinalizeImageGeneration`。普通 `ExecuteReservation`、`ExecuteSettlement` 和 TransactionalReservationOwnerStore 不放松。资源写事务内沿原 bucket/debt顺序、原event/audit/immutable result持久化；提取现账务计算/写入helper复用，禁止第二套balance算法。

所有三个写/封闭动作使用固定 org+resourceOperationID 的**现 operation row**作资源侧串行 fence：在事务内 insert-on-conflict 后 FOR UPDATE，校验固定请求fingerprint；不要跨网络/数据库持有该锁。读取不可变owner证明在锁外完成，入事务后校验proof身份和digest。可变“目前未开始”的远端快照不足以授权释放；只有已CAS成不可逆 no_generation/succeeded 的proof可终结。

`FinalizeImageGeneration` 在尚无 reservation 时同样必须封闭该 operation：只接受 owner 的不可逆 no_generation proof，写现 operation 的 terminal rejected/closed immutable receipt（不改变余额、不伪造零quantity事件）。随后晚到的 Reserve 必须返回相同closed结果而不能预留。若Reserve先提交，则Finalize释放其原reservation。这不是新恢复owner，而是关闭“cancel读到不存在后并发reserve晚提交”的孤立扣占窗口。窄reader理解本owner的closed receipt；普通 reservation replay 语义不变。

当前[持久化与副作用边界](project-boundaries.md#persistence-and-external-effects)保留默认同事务规则；仅此固定 Image owner 以不可变intent/terminal proof + 双方各自CAS + 资源operation fence替代；禁止其他owner自动采用。现 `SettlementService` 的caller不能选择outcome原则保留；no_generation映射release，provider_succeeded映射commit，未知不能结算。此固定owner合同例外已通过本次独立IMPLEMENTATION_READY，随实现同步§3.10，不适用于其他owner。

### C. 唯一派发顺序

1. 当前 actor/org/canonical member live授权、exact source读取/bytes digest、资源消费授权、route与版本化价格核实全部成功；缺依赖在provider前拒绝。V3原slot预算仍独立，不把AI点数当images/tokens单位。
2. V3独占claim后 `PrepareGenerationIntent` 在本库CAS保存完整intent，dispatch=`prepared`。这是尚未派发的证明，**不是**原phase=provider_claimed的推断。只有该新generation fact合同的当前单main-slot使用此路径；旧记录不做兼容迁移/自动放行。
3. 调用资源 Reserve：owner reader核intent，资源事务得到不可变receipt。ACK丢失以原operation/intent查询或原键Reserve重放；不得换operationID。回读校验org/ownerAttempt/resource/quantity/fingerprint/state=reserved后，V3 CAS绑定reservation ID及receipt hash。绑定失败不调provider，重试只补原binding。
4. 再次live授权和route/config核验；若已取消/失权/配置漂移，V3 `prepared → no_generation` CAS不可逆，Finalize原资源operation，不能重发。否则 `BeginGenerationDispatch` 将prepared+exact binding CAS为`dispatch_started`，**只有CAS获胜且成功ACK的调用栈**可做一次POST。CAS ACK不明，即使回读是dispatch_started也不授权发送；保留UNKNOWN。不会发放可被新activity重复消费的持久“许可”。
5. adapter持有同intent observer，在收到typed成功payload后、任何下载/解码前，`RecordGenerationOutcome` CAS写provider_succeeded、provider request id（有则存）、结果引用摘要、单张原结果locator（HTTPS、公网URL校验、最多4096字节、无userinfo/fragment；非法/空/超限则明确invalid_result且仍为生成成功）和可信token（有则Known=true，无则明确unknown）。locator仅在worker私有V3事实中，保留原签名query、不进入HTTP/Account Audit/日志/Temporal DTO。本次官方wire没有usage合同，不填0。成功fact持久化后可以先Finalize commit点数，再继续下载/原bundle/staging；任何后处理失败不释放。ExecuteSlotV3的claimed重放与RecoverEffectV3的unknown提前返回前，在live授权和exact Catalog核验后优先恢复原bundle，缺失才对原locator使用现安全Fetcher/解码限额GET→现Prepare/Preserve；同V3事务核验原success/身份/quote/manifest并单调进入staging_prepared，绝不回到claimed或重新POST。此路径不解析当前价格/凭据、不重新取来源；撤权后财务finalizer仍只补原结算、不能GET。原locator过期/坏图只使结果不可恢复，不反转成功扣点、不重新生成。
6. 已证明provider未调用的前置失败可写no_generation；普通HTTP错误、timeout、EOF、取消、JSON错误、running状态均不凭推测释放。当前没有已核“失败响应一定无生成”的可信映射时保守UNKNOWN。dispatch_started之后只能同一次observer的可信结果推进；late成功可由UNKNOWN单调到succeeded，但不能由no_generation翻转。对已dispatch路径不会由恢复线程制造no_generation。
7. Finalize只读不可逆proof、按原资源identity原子commit/release，V3保存其receipt引用。同键同proof幂等、异proof/quantity/quote冲突。资源ACK丢失只回读/补Finalize，不回provider。经济终局和Asset批准完全分开。

### D. Execute/Recovery/取消并发的消费规则

现 `ExecuteSlotV3` 的“claimed但非首次→只查bundle”，以及 `RecoverEffectV3` 的“providerUnknown直接block”，均在当前固定image owner分支先消费上述generation fact，不只加列。

| 持久状态/故障点 | 原attempt恢复动作；禁止项 |
| --- | --- |
| intent未提交 | 无资源预留、无provider；原请求可按现幂等合同重进 |
| prepared，Reserve/Bind响应丢失 | 原资源operation回读/重放并补绑定；自动恢复不派发。若取消/恢复选择终止，先CAS no_generation，再Finalize封闭operation，即使原Reserve晚到也不能留下扣占 |
| 原执行仍活跃 vs 恢复终止prepared | BeginDispatch与no_generation竞争同一V3 CAS；只允许一个获胜。失去CAS者绝不POST |
| dispatch_started或其ACK不明，无terminal fact | UNKNOWN、保留原预留；无新POST、无虚构query、无定时释放 |
| 成功响应已收到但V3 outcome写失败 | 同调用栈在bounded detached finalization内重试同payload；若仍失败/进程丢失则UNKNOWN held，不能据内存成功无持久proof收费/释放；不重生成 |
| provider_succeeded，settle/下载/bundle失败 | 先按原proof补原结算，再恢复现bundle/已有结果的安全下载；坏URL/坏文件仍不批准。不得让MarkBudgetUnknown或phase迁移清除success/token事实 |
| no_generation，资源尚不存在/ACK丢失 | Finalize封闭原operation或释放已存在reservation，回读确认；不得把“未读到reservation”当已安全结束 |
| 已terminal，workflow取消/用户不采用 | 只恢复对应commit/release回执，不改变provider效果，不重复生成 |

恢复继续由现 Temporal effect recovery workflow/activity负责，有界重试/原身份人工redrive沿现入口，不新scanner。为经济finalization提供只接受已存在V3 identity的内部窄恢复分支：撤权不能触发新reserve+dispatch/读图，但不应阻止既有terminal proof的后台结算/释放；调用者为显式装配的原effect owner，不是租户HTTP可自选outcome。外部GET/Approve仍实时权限。prepared终止仅作资源operation封闭，不需重新给撤权actor分配权限或资金。

### E. 成员 AI 点数月度上限

生产已有 `OrganizationExecutionAuthorizer` 使用 exact active grant、AuthorizationID==MemberID、组织未停用及 image_agent.write；复用它，不新IAM/permission。该permission授予operator/admin，但仍不能省略成员被管理员分配资源的约束。`accountallocation.MetricToken` 是成员token额度，不是AI点数分配；商业purchase/topup权限是购买，不偷换模型消费。Figma 1636:371/431:4158要求管理员分配给成员，统一企业余额并未批准全员无分配共享消耗。

用户已批准“补齐成员月度消费上限，按 Figma”，数值由管理员设置，企业余额不清零。该limit是使用企业AI点数的约束，不是成员钱包/余额。现 `accountallocation.QuotaReader.ReadTokenQuota`、`Service.SetTarget/Snapshot/Consume`强制权益window，DTO同时带MetricToken和WindowStart/End；抽取其中仍有效的成员校验/版本/幂等交互，但不wrap Token Service、不换label/metric、不迁移旧表或使用saas_usage计点。

**事实拆分与月份。** 在 `orgresource` owner 增加成员限额配置 row（org/member主键，monthly_limit、version、active、updated_by/at）及月计数 row（org/member/month_start主键，reserved、consumed）。配置不作为credit，不累加到bucket，不要求所有成员limit之和等于企业余额；每次调用同时满足个人remaining与企业available。未配置或limit=0即不能新消费，不填100等样例。时区复用仓库已核UTC自然月约定（`usage_ledger.go:canonicalUsagePeriodKey`的UTC YYYY-MM及商业read的UTC月[start,end)），只复用时间约定、不接其Token ledger。服务端在V3 intent首次持久化时固定month_start/end和当时配置version，进入fingerprint与原资源reservation；浏览器不能传月，跨月同key回放不改月/价格/member。


**原子扣占与锁序。** 新reserve在commercial同tx：image operation fence → member config行 → 原月counter行 → enterprise bucket → 必要debt；创建counter insert-on-conflict后锁行。核intent配置version等于当前version，current month等于intent月且未过期、active、limit-(consumed+reserved)>=price、企业available>=price，才同时增加成员reserved与企业reserved并写原资源reservation/event/audit。配置version变更或月已过不对原prepared intent重报价/切月，返回明确不可派发，由原no_generation关闭；另发用户新操作才用新值。相同已提交reservation的回读不再以新version拒绝。

Finalize复用operation fence → reservation → 原月counter → bucket → debt；配置更新不锁reservation，因此无反向环。commit成员reserved-=quantity/consumed+=quantity；release仅reserved-=quantity；企业balance沿原算法同tx变动。month/member/quantity取原reservation，不取当前配置/当前月。member counter不足/事实冲突整tx失败，UNKNOWN在原月持续reserved；不能靠次月分配释放。资源receipt需包含member、原月、price/config version与intent fingerprint，V3绑定逐项核对；原terminal proof决定结果，caller不能提交charge outcome。


**UI与验证。** 实施UI前Writer自己加载Figma skill并读取1636:371/431:4158，不以转述当验收；只当前AI点数月度栏与必要设置，明确UTC月界，原Token不冒名、店铺/数据预算不扩。生产准入须已有live image_agent.write、exact member、月limit与企业余额全满足；current HTTP gate待该链完整才开放。针对性TDD含未配置/成员不足/企业不足0provider，reserve与降limit并发，跨月原key和UNKNOWN，撤权后不新消费而原proof可结算，反向org/member篡改、版本/幂等冲突；用原PG/Temporal设施不建runner。Figma2061:362/3395:389仍非单图生成对照全链，不扩八图菜单。

### F. 正式 wire 与当前代码落点

已核官方 [gpt-image统一接口](https://qmy27nhsd9.apifox.cn/452409160e0)：POST /v1/api/generate 支持 gpt-image-2.5、images base64、aspectRatio=1024x1024、quality=auto、replyType=json。因此选定复用现grsai exact-byte materialization与no-replay，**不走未验证image[]或双协议fallback**。仅此route显式构造这些字段，不能混用nano的imageSize或现size/response_format；返回running仅UNKNOWN，本片不加accepted-task/query状态。既有其他合法消费者默认poll/retry语义不变。成功payload要在 `GenerateImageOnce 的输出物化` 前交observer，失败不能丢typed事实；provider输出下载继续原public URL SSRF限制。

- `internal/imageagent/slot_effect_v3.go`、`store/records.go`、`store/slot_effect_v3_{gorm,memory}.go`：generation fact校验、intent/binding/dispatch/outcome/receipt CAS，原row安装与权限只增必要字段，无新表owner。
- `internal/imageagent/temporal/activities_execution_v3.go`、`activities_recovery_v3.go`：消费同attempt状态、禁止replay派发、bounded finalization/原recovery接入。
- `internal/ledger/orgresource` 与 `internal/integration/orgresource`：仅固定image owner receipt合同、现operation fence/closed receipt、复用原reserve/settle账务事务；其他 owner 继续遵守各自既有合同。
- `internal/integration/grsai/client.go`、现ProductImage adapter/port：正式json字段、exact bytes、no-poll单次路径、后处理前typed outcome observer；observer错误仍UNKNOWN，不重POST。
- `internal/app/worker/imageagent/dependencies.go`、现current配置：显式price/授权/owner reader装配及fail-closed；runtime grant按实际SQL最小修改，普通启动只Verify，不操作当前保留实例。
- [当前系统边界](project-boundaries.md#persistence-and-external-effects)：固定 Image owner 的资源回执例外，不扩展到其他 owner。

**配置命令与下限。** Account owner 的 AI 点数配置使用 `ReadMemberAIPointLimits` / `SetMemberAIPointMonthlyLimit`，复用现管理员权限、live active canonical成员、ExpectedVersion+IdempotencyKey；服务端决定current UTC月，管理操作identity绑定首次月份，原key跨月返回原immutable结果而不再次修改。Target允许0，但不得低于current month consumed+reserved；旧月UNKNOWN的reserved不释放、不抹计数，旧月不因新配置迁移。配置持续用于后续月份，新月计数从无记录开始0（不是赠送余额）；当前月提高/降低受下限约束。成员撤权立即阻新派发，不删除配置/原月counter/reservation；恢复结算原消费仍有效。存储错误/串org/member fail closed。
