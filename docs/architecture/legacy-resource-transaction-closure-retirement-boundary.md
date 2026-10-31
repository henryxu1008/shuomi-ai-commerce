# 旧资源通用事务与 Store 同库扣费链完整退休边界

## §1 用户结果、Design Basis 与授权阶段

本批只清理两套未装配的旧执行链：通用资源预留/结算，以及 Store 与资源强制同库的扣费执行器。当前图像仍借用旧仓库的一段事务结算行为，先抽到当前资源持久化 owner，再删除旧编排及专属尾部。当前 AI/图像扣费、余额/月限、Store 开通续费和观察同步保持原入口。

- Execution Issue: [#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301)，保持 **OPEN**；一个 Writer、一个主要 PR，不机械拆批。
- 使用者：维护当前图像与资源扣费、Store 服务的开发者。交付结果是源码不再依赖闲置的同库/任意 owner 注册协议；不声称当前用户业务被阻塞，不新增产品功能、试用门禁或验收平台。
- Design Basis: **Independent Architecture**。当前 **DESIGN_ONLY / REVIEW_PENDING / SOURCE_NOT_STARTED / NOT_READY**；用户当前仅授权完整设计，适用独立复核与用户确认完成前不修改生产实现。
- 调查基线：main `27039ab105ad0bac4cc274b5d59a96b703808a09`（含 #1039、#1040）。Writer 会话 `01a133ef-757c-73e3-b8f6-76958c642397`，分支 `codex/issue301-resource-execution-retirement`。已核对当前 Issue、open PR 与会话；#1038 的 SHEIN 履约归原 owner，不接管。
- 本轮设计提交仅新增本文并向 `legacy-register.md`、`project-boundaries.md`、`docs/architecture/README.md` 追加顶注/必需导航；§5 其余路径是**待确认实施范围**，不是已经实施。

Product Authority：[全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[当前统一基础方案](unified-base-prepaid-resources-v1.md)、[成员资源与 Store 当前合同](member-resource-allocation-v1.md)、[原资源/Store 设计](../superpowers/specs/2026-09-03-shuomi-resource-ledger-store-design.md)以及用户本轮“先做完整设计、确认后实施”的决定。没有 UI/IA/交互改动，Figma 增量 **N/A**；不重开上述当前产品决定。

Must：保留有效行为和当前业务合同；不留旧 facade/alias/fallback/第二事实源；保持当前权限、事务、JSON/hash、UNKNOWN 和恢复；完整收齐专属尾部。Out of Scope：新计费/价格/状态机、余额算法重构、迁移、旧历史续跑、部署、关单与真实资源操作。没有新增 Accepted Risk，也不把现有风险擅自接受为 PASS。

## §2 前批 KEEP 替代与 Legacy 决策

| 旧要求/依据 | 本批准确替代边界 | 继续有效的部分 |
| --- | --- | --- |
| 原组织资源通用注册与同事务 owner 协议；`project-boundaries.md` §3.10 | 仅退休 §5 的 ReservationService/SettlementService、GormReservationRepository 与任意 owner 注册/锁 proof 编排；没有当前 runtime 注册方 | 当前固定 image_generation_v1、model_invocation_v1、ConsumerCharge 及各自原事务/身份/财务边界 |
| `member-resource-allocation-v1.md` §2 的 “Preserve existing generic/same-DB contracts for their current consumers” | 已查明无当前通用/同库执行消费者；当前图像借用的事务行为按 §4 抽取后，该保活不再适用 | 该文档的独立 Store/Commercial DB、成员资金来源、receipt/CAS、UNKNOWN/恢复合同全部保留 |
| `2026-09-25-issue487-main-image-execution-design.md` 对通用合同与算术复用的 KEEP | 只替代旧通用注册协议及对旧仓库方法的借用；不改变固定图像 owner 的发送/持久化/结算规则 | 当前主图、图组/Artwork、预算、结果不可用仍计费、原月归属与恢复 |
| `store-center-current-application-v1.md` 的旧同库 executor 路径；#988 对当前 Resource 整体 KEEP | 仅退休未装配的 `integration/orgresource.StoreServiceExecutor`；不退休整个 orgresource 包，也不退休 Store 服务合同/仓库锁方法 | 当前 `app/storecenter.ServiceLifecycleExecutor`、Store 服务日期/审计、本地事务与 ConsumerCharge、Store 观察同步 |

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior：当前 ReservationState/错误/事件值、有效资源类型校验、图像事务结算及两项共享测试断言。Current owner：`ledger/orgresource` 当前资源值合同与购买校验；`integration/orgresource` 图像事务结算；Store 当前 app/native owner。Cutover/deletion condition：§5/§6.1 完整闭包退出、当前不变量与原 CI/适用最终复核有准确 HEAD 证据；合并另授权。当前 owner 不得经 wrapper 继续依赖旧注册仓库。

## §3 实际消费者与 owner

下表来自基线的声明/构造器/方法调用和测试引用核对；“只剩测试装配”不等于整个包都闲置。

| contract → implementation → injection → consumer | 当前结论 |
| --- | --- |
| ReservationService/SettlementService → GormReservationRepository → test owner map/authorizer → reservation/settlement 专属测试及混合 PostgreSQL 测试 | 没有生产构造器或任意 owner 注册方。通用 request/replay/result/snapshot、OwnerAttemptState 与 proof/binding 专属协议均退出 |
| Store ServiceLifecycleExecutor → integration/orgresource.StoreServiceExecutor → 共享 GORM tx → 旧 Store 测试 | 没有当前生产装配；同名 app 执行器是不同的当前实现，不能按简单名称全局禁入 |
| `generation_runtime.go` → NewGormImageGenerationRepository → FinalizeImageGeneration → `(&GormReservationRepository{}).calculateSettlement(...)` | **真实生产消费者**。只借用 commit/release 计算及 debt-first 写入，不使用通用注册/Reserve/Settle |
| `current_store_center.go` → `app/storecenter.NewServiceLifecycleExecutor` → ServiceLifecycleApplication → 当前 HTTP | 当前 Store 开通、续费、重新激活使用独立 Store intent/proof 与 Resource ConsumerCharge，原样保留 |
| 当前 image/model invocation、ConsumerCharge、DataAcquisition、member allocation、Store HTTP → ReservationState/七错误；image event → OperationReserve | 这些值是当前合同，不随旧声明文件删除 |
| `ledger/orgresource/purchase.go` → validResourceType | 唯一保留的生产函数消费者；校验体归该购买 owner，不保留旧 reservation.go |
| 当前 welcome/trial grant 测试 → assertBucket/assertDebt → 旧专属测试文件 | 两个实际共享 helper 原样抽到现有 gorm_repository_test.go；其它旧 fixture 不为它们保活 |

架构问题：当前图像通过空的旧仓库实例调用**会写债务的事务 helper**；若把它当纯函数移到领域层，或仅删旧包，会破坏 owner/事务。旧 same-DB Store 编排也不能用来表示当前跨库原子性。本批只解开这两个耦合，复用现有 GORM、transactionRunner、原本地事务/receipt/CAS，不新建框架。

调查范围包含 internal/cmd/tests/tools/scripts/部署与文档引用；没有发现生产以旧构造器注入的第二条链。发现清单外消费者、共享 helper 或原检查要求时，先报告准确差异和最小方案，不用假调用保活、不直接删安全检查。

## §4 抽取、持久化和副作用边界

### §4.1 最小抽取与切换

1. 新增 `ledger/orgresource/reservation_state.go`，仅承接 `ReservationState` 和四值 reserved/committed/released/reconciliation_required、原七个 error 单例（ErrReservationOwnerNotRegistered、ErrOwnerScopeMismatch、ErrOwnerNotReservable、ErrInsufficientBalance、ErrOwnerNotTerminal、ErrInvalidOwnerProof、ErrReservationNotFound）、`OperationReserve="reserve"`。原名称、类型、值和错误文本不改，不是旧服务 alias；所有当前调用点无需改名。OwnerAttemptState/六值、通用 DTO/服务/接口、OperationSettleReservation 不抽取。
2. `validResourceType` 原函数体移入现有 `purchase.go`；仍只接受原三种资源类型，不新增能力或抽象。
3. 新增 `integration/orgresource/image_generation_settlement.go`，只承接当前图像需要的私有 `imageSettlementDecision` 及 commit/release 两值、`calculatedImageSettlement` 六个余额/credit 字段和 `calculateImageGenerationSettlement(ctx, tx, reservation, bucket, decision)`。从原 calculateSettlement 去掉无用 receiver 及只供旧 snapshot 使用的 result.decision；**算术、溢出拒绝、error、applyPositiveCredit 调用/时间语义原样保持**。不抽取通用注册或创建 tx 的能力，不接受任意 owner，不导出新服务。
4. 当前 image_generation_repository.go 仅将旧 Decision 引用及空旧仓库方法调用改接上述私有当前 helper。Event.Reason 仍为原 **commit/release**，不能错改成 receipt 的 committed/released；其它 receipt/state/事件/审计字段保持。判断仍由原持久化 GenerationFact 决定，caller 不得选择成功或释放。
5. helper 仍在调用者原 tx 内执行；ctx/tx 不新建、不分离提交，不新增查询、重试或恢复入口。当前 Finalize 的 operation fence → reservation 锁 → 原月 counter → bucket → debt 原调用/锁顺序保持，不借本批重排锁或优化协议。applyPositiveCredit 继续由持久化 owner 拥有；bucket 已锁后原 debt-first 更新和回滚保持。
6. 共享 assertBucket/assertDebt **函数体原样**进入当前 gorm_repository_test.go；错误仍失败关闭。其它专属 fake owner/seed/runner/Store fixture 随旧文件删除，当前测试不能再引用旧接口。

### §4.2 JSON/hash、物理账本与历史取舍

| 边界 | 处理 |
| --- | --- |
| 当前 JSON/hash/版本 | 当前图像 intent/fingerprint/proof/receipt、model reserve/finalize 两 hash、ConsumerCharge payload/资金来源、Store service request fingerprint/receipt 均不改；不新增 encoder/decoder/版本或零槽位。新 private decision 仅保持原事件字符串，不参与新协议 |
| 共享物理事实 | schema.go、全部 row/关联/index/约束、Goose/安装/grants/权限检查零改，包括当前仍使用的 saas_organization_resource_reservations/operations/events/audit/buckets/debts；不因通用入口退休删表/列或改新安装 schema |
| 当前资金不变量 | 原组织/member/月份/price/quantity/身份绑定、余额/月限、防重复扣费、净 credit 与债务优先、UNKNOWN 保留、无发送才释放、replay/response-loss/recovery 不变；保存过的当前数据不重算 |
| 旧通用协议历史 | 旧 Reserve/Settle request/replay/snapshot/fingerprint、同库 Store immutable result 不再承诺由退休入口续跑或翻译成当前合同；不新增 alias/迁移/双读/dual-write/fallback。其物理记录留存不变，不扫描、读取或处理真实记录 |
| 当前 Store 与资源 | Store 日期/版本/连接/审计在原 Store DB 本地 tx，资源扣费在原 Resource DB；当前 receipt/CAS/terminal proof/recovery 原样保持。LockServiceState/ApplyServiceState 仍有当前 service_charge 消费，禁止连带删除 |
| 外部副作用 | 不调用 provider、SHEIN/ERPNext、DB/对象存储/浏览器等实际环境；正式代码只移除闲置路径，原发送/UNKNOWN owner 不改变。#985 已退休的独立 SHEIN 登录/身份写回不恢复 |

如发现当前外部可观察兼容义务，停止并给出具体证据交用户决定；旧测试或历史文档不能自行授权兼容。此设计与代码交付都不授权真实数据迁移/清理。

## §5 完整删除/抽取/保留矩阵

实施范围共 **27 路径**：9 生产、9 测试、9 文档；是允许上限与准确矩阵，不为凑数制造修改。设计阶段仅 D1–D3/D9 四文档。清单外变化按 §3/§6.1 先报告，不机械新开小 PR。

### §5.1 生产：9 路径

| ID | 路径 | 决策 |
| --- | --- | --- |
| P1 | `internal/ledger/orgresource/reservation.go` | EXTRACT 当前值/validResourceType 后整退旧文件和通用服务 |
| P2 | `internal/ledger/orgresource/settlement.go` | EXTRACT 三当前错误后整退；旧公共 Decision/DTO/服务退出 |
| P3 | `internal/integration/orgresource/reservation_repository.go` | RETIRE 全部通用注册/锁 owner/预留编排 |
| P4 | `internal/integration/orgresource/settlement_repository.go` | EXTRACT 当前图像事务行为后整退通用结算/proof/snapshot |
| P5 | `internal/integration/orgresource/store_service_executor.go` | RETIRE 旧同库 executor/专属接口/runner/failure snapshot |
| P6 | `internal/ledger/orgresource/reservation_state.go`（新增） | §4.1 当前值合同，不含旧编排 |
| P7 | `internal/ledger/orgresource/purchase.go` | 只原样承接 validResourceType |
| P8 | `internal/integration/orgresource/image_generation_repository.go` | 只切换私有 decision/helper，其它原逻辑保持 |
| P9 | `internal/integration/orgresource/image_generation_settlement.go`（新增） | §4.1 同 tx 私有图像结算行为 |

### §5.2 测试：9 路径

| ID | 路径 | 决策 |
| --- | --- | --- |
| T1 | `internal/ledger/orgresource/reservation_test.go` | RETIRE 旧通用服务保活及专属替身 |
| T2 | `internal/ledger/orgresource/settlement_test.go` | RETIRE 旧通用服务/DTO 保活及替身 |
| T3 | `internal/integration/orgresource/reservation_repository_test.go` | 抽出 assertBucket，整退其它旧 owner fixture/测试 |
| T4 | `internal/integration/orgresource/settlement_repository_test.go` | 抽出 assertDebt，整退其它旧结算 fixture/测试；有效资金安全验证见 §6.1 |
| T5 | `internal/integration/orgresource/store_service_executor_test.go` | RETIRE 旧 same-DB 路径及专属 helper，不删当前 app Store 测试 |
| T6 | `internal/integration/orgresource/gorm_repository_postgres_test.go` | 仅去首函数内三段旧 Reserve/Settle/同库 Store 尾部及唯一旧 import；保留 welcome 并发与完整权限测试 |
| T7 | `internal/integration/orgresource/gorm_repository_test.go` | 只原样承接两共享 helper，原当前测试/断言保持 |
| T8 | `internal/integration/orgresource/image_generation_repository_test.go` | 原测试保持；将仍有效的释放债务/溢出/事务失败回滚安全验证改接真实当前图像终结入口 |
| T9 | `tests/legacy_resource_transaction_retirement_test.go`（新增） | 全闭包禁入、最小当前值/事务接线、当前 Store 装配必需及八份既有文档冻结原后缀验证；不改其它 guard 算法 |

### §5.3 文档：9 路径

| ID | 路径 | 处置 |
| --- | --- | --- |
| D1 | `docs/architecture/legacy-resource-transaction-closure-retirement-boundary.md` | 本文，唯一完整设计 |
| D2 | `docs/refactoring/legacy-register.md` | 仅当前顶注/导航；历史计数/勾选/正文不改 |
| D3 | `docs/architecture/project-boundaries.md` | 仅窄顶注准确替代通用/同库旧合同；其它 owner 边界和旧正文保持 |
| D4 | `docs/architecture/member-resource-allocation-v1.md` | 仅顶注替代 §2 generic/same-DB 保活，当前 receipt/独立库/权限/资金合同保持 |
| D5 | `docs/architecture/store-center-current-application-v1.md` | 仅顶注旧同库 executor 路径失效；不改当前 Store 产品要求 |
| D6 | `docs/engineering/2026-09-25-issue487-main-image-execution-design.md` | 仅顶注替代通用注册/旧仓库借用；当前图像合同保持 |
| D7 | `docs/architecture/legacy-subscription-metering-closure-retirement-boundary.md` | 仅顶注说明 #988 Resource KEEP 只对本批专属对象被替代 |
| D8 | `docs/superpowers/specs/2026-09-03-shuomi-resource-ledger-store-design.md` | 仅顶注替代旧通用 owner 和 §8 同库 Store 编排承诺；当前资金与其它历史原文完整保留 |
| D9 | `docs/architecture/README.md` | 追加新设计的显式索引，原导航/历史完整保留；原文档索引检查不变 |

D2–D9 一律从基线完整 Git blob 追加顶注，原内容作为**逐字后缀**保留（只在验证工作区时允许 CRLF→LF 归一化）；不对历史做全局 replace。滚动 HEAD/CI/评审状态写 PR，不用提交改历史状态。

### §5.4 KEEP / 零差异

当前 Resource schema/transactionRunner/debt.go、模型调用结算/两处 JSON hash、ConsumerCharge/成员分配/月限/余额/购买/欢迎与试用 grant、当前 Store 原服务合同/仓库/LockServiceState/ApplyServiceState、app/storecenter/current_store_center、产品采集/DataService、全部现役图像/AI/provider/Temporal/恢复、身份权限及 Store 观察同步保持。P7/P8 只限上述窄抽取，不改当前业务语义。

workflow（含两套原 workflow）、完整 inventory（CURRENT/历史/计数/ceilings/算法）、依赖、保护规则零改。无当前 inventory 行因本批退出；不删整个 ledger/orgresource 或 integration/orgresource，不恢复此前退休对象。

## §6 验证与验收方法

设计交付只做引用/清单/后缀/差异检查和原 CI，不运行 provider、数据库或容器，不读 .env/现有凭据。设计 review/CI 不代表源码、运行或产品验收。

确认后实施：先让 T9 在旧链存在时真实 RED，再在同一 PR 依 §4 抽取/删除后 GREEN；保留并运行与变更直接相关的原资源/图像/当前 Store 测试，必要 race、当前命令仅编译；原 CI 留稳定候选执行。无关全量重跑、完整故障矩阵/验收平台不是前置。测试只能使用任务自有临时资源及合成 fixture；不得读取既有实例、凭据或修改其数据/进程。

### §6.1 原检查与有效安全断言处置

| 检查/函数 | 准确允许变化 | 必须保持 |
| --- | --- | --- |
| T1/T2 的旧服务全部测试与专属替身 | 随无生产装配的协议整退，不造兼容替身 | 当前授权/组织/范围/幂等验证由现有具体 owner 原测试继续覆盖 |
| T3/T4/T5 旧通用/同库仓库测试 | 抽取 assertBucket/assertDebt 后整文件退出；其它专属 helper 退出 | 不删除当前图像/ConsumerCharge/Store 的安全断言；有效旧资金后果按下一行在当前入口验证 |
| T8 `TestImagePointsReserveBothLimitsAndSettleOriginalMonth` | 原函数断言不变，可复用其合成 fixture形成新当前入口测试 | UNKNOWN 不结算、不重发、原月/撤权、audit/同一 receipt replay、跨组织隔离保持。增加当前 Finalize 的 no-effect 释放 debt-first、consumed/available 溢出拒绝、event/audit 失败时债务/余额/月计数/reservation/receipt 一起回滚；不得只测新 helper 自己或复活 testOwnerStore |
| T6 `TestPostgresWelcomeGrantConcurrentSourceReplayMintsOnce` | 只删从 `AutoMigrate(&testOwnerAttemptRow{})` 开始到旧 `assertStoreServiceRow(...)` 为止的三段专属尾部及唯一 storecenter import | 原 welcome 12 并发/source claim/一次 mint/四表计数/余额断言逐字保留；原 context、容器隔离及错误处理保持 |
| T6 `TestPostgresResourceRuntimeMonthlyLimitPermissions` 及 init | **零改** | 真实 owner/runtime、撤权、越权拒绝、月限/表权限及全部断言保持；不加 skip、不放宽 grant |
| T7 全部原 Test 函数 | **零改**，只追加两原 helper | Welcome replay/eligibility变化、response-loss、rollback/debt-first、负余额/事件约束等原断言完整保持。trial_grant_repository_test.go 和 transaction_test.go 零改 |
| T9 新 guard | 旧十文件明确缺席；保留根遍历中禁入旧独有声明/构造器/注册协议，失败关闭；当前装配/值/helper 与历史后缀必需 | 按完整包限定旧 StoreServiceExecutor，不能误禁当前 app 的 ServiceLifecycleExecutor；不以文件不存在 skip，不扫描历史文档中的符号作为生产依赖 |
| 其余所有原测试/AST/lint/CI/inventory | **零改** | 特别保留 #932 deadline 回归、#1039 两 hash 和冻结后缀检查、当前 Store 跨库扣费/恢复/撤权/删除保留付费证明，以及 model/consumer current tests |

不是删除仍有效的资金安全检查。旧通用 owner 状态枚举、同库 Store 原子事务、旧 DTO/fingerprint 本身不再是当前 requirement；其保活退出。对于当前资金仍成立的后果，转接当前 owner验证；不能把退休 generic 并发 PASS 当成当前跨库证明。若原 CI 暴露遗漏保活/check，不先改成 skip/放宽断言；按准确补丁报告用户。

## §7 准入、评审、实施顺序与停止条件

1. 本轮四文档提交设计 PR，记录起始 SHA、scope/owner/未授权项，核对原 CI 和适用独立设计复核。**未经用户本轮授权不默认新增 AI 互验线程**；设计与源码最终复核授权分开、绑定准确 HEAD。GitHub 同 SHA 复核运行时不重复触发；启动失败/不可用不冒称 review PASS。
2. 发现 finding 先按 AGENTS.md 分类，指出当前 Must/后果/阻塞层级；最多两轮正常架构 review，不重开无关全局架构。有效证据只复用未变化部分。
3. 适用设计 review 收敛、原 CI 和用户确认 §1–§7（含 §4.2/§6.1）后，才能登记 USER_CONFIRMED / IMPLEMENTATION_READY 并实施；设计 CI 绿色本身不授权生产变更。
4. 同一主要 PR 内先值/购买校验与事务 helper抽取、当前接线，再退旧五生产文件和专属五测试/混合尾部，收齐原安全验证和历史顶注；小切片只作提交/验证边界，不机械拆批。
5. 稳定实现后核对 main 组合，运行适用原 CI，做一次完整最终交付检查。修复后只复核增量；合并另取最终准确 HEAD 授权，expected-head 正常合并，再确认实际 merge SHA 的完整 push/main run；PR CI/单 Gate 不代替 main CI。
6. #301 保持 OPEN。当前能力保留不等于运行/产品验收；不部署、不停进程、不读现有凭据/缓存/真实记录，不增加内部试用前置，不关闭父 Issue。

停止条件：发现清单外当前消费者或兼容义务；需改变余额/权限/事务/JSON/hash/恢复或物理 schema；需 workflow/inventory/保护/依赖变化；有效安全断言无法保持；需操作既有资源。报告根因、具体最小差异和适用授权，不自行迁移、兼容、删表、放宽检查或扩大设计。
