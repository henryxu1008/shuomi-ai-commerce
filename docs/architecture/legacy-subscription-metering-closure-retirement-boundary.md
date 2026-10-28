# 旧订阅、额度与影子计量完整退休设计

这批清理三件事：**旧套餐/订阅管理与购买、旧 Token/Store 数量额度、旧 OpenMeter 影子计量。当前预付资源、AI 点数/月限、Token 审计、图像与 Store 观察同步继续使用现有 owner。** 旧包不能靠换名继续存在。

状态：**DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_REVIEW_AND_USER_CONFIRMATION**。父目标 [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301) 保持 OPEN。只读调查基线 main `0aab8d5ebe03140e05c2a9c9716d3665da0a0ace`（#985 已合并且准确 main CI 成功）。本文是待确认方案；不签发运行、部署或数据操作授权。

## 1. 用户结果、产品依据与 Design Basis

使用者是当前采集、图像、账户资源和 Store 用户及维护者。结果是源码不再提供旧套餐等级、订阅有效期、Token 分配或订阅 Store 数量额度，不再构建其影子计量 PoC；当前入口不因此增加新操作或内部试用前置。旧表、订单、IAM、凭据和进程均不处理。

本轮文档工作 **Design Basis: N/A / Ready**，仅本文、架构索引和 Legacy Register 三文档。未来源码采用 **Independent Architecture / NOT_READY**：结束旧持久化/恢复 owner，改接跨 AI/Resource/Store 的有效安全断言，并超过 30 文件。完成适用独立设计复核、用户确认 §1–§7（特别 §4.2/§6.1）、Issue/PR 的 IMPLEMENTATION_READY 读回后才实施。当前“可以”仅授权设计，不授权删除源码。

适用 Authority：

- [统一基础方案与预付资源 V1](unified-base-prepaid-resources-v1.md) §1、§3–§5、§9：当前产品是统一基础方案、Store 服务期和 AI/数据预付资源；旧订阅购买要求已被替代。**Token 是审计量，不是 AI 点数，旧 Token 分配不是必须抽取的当前能力。**
- [正式价格 V1](commercial-retail-pricing-v1.md)、[模型与费率管理](model-route-pricing-management-v1.md)、[企业存储容量](enterprise-storage-capacity-v1.md)：继续由当前 offer/route 与 owner 定价；本批不改变费率、资源单位或存储计量。
- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[当前架构边界](project-boundaries.md)。无兼容类别，无旧系统权益/账务承接。
- Product/Figma Authority 为上述当前产品决定及 [Final UI/IA Authority](../product/final-ui-ia-authority.md)。不改页面、导航、交互或可见字段，无新增 Figma 设计；现有 UI 的基础方案/资源 JSON 保持。

Must：完整移除被替代的套餐、Token allocation、Store quota、影子投影及专属尾部；保留当前钱包、真实资源购买/预留/消费/UNKNOWN、AI 点数和成员月限、Token 原始审计、图像预算/审批/资产批准、采集、当前身份/组织隔离、Store 官方连接与观察同步。当前 JSON/hash/版本、既有载荷和现有价格不改。

Should：保留现有 AI invocation/dispatch、Resource owner 和当前只读入口；不新建计费、预算、恢复、迁移或验收平台。

Out of Scope：新套餐、新费用、图片审核另行定价、模型费率重设、旧数据迁移/清理、历史续跑、真实 provider/数据库/IAM/对象存储访问、现有凭据或进程操作、workflow/inventory/保护规则变更。独立 SHEIN 登录及其身份回写已由 #985 退休，不恢复；保留的是 Store 观察同步和官方授权。

Accepted Risks：无新增风险接受。运行停用、真实资源状态、产品/生产验收 NOT_RUN；源码退休不证明旧实例已停止。生产发布 NOT_READY，不把本设计或运行停用变成内部试用新前置。

## 2. 旧 KEEP 替代与 Legacy 决策

| 前批/旧依据 | 本批拟确认后的准确替代 | 继续有效 |
| --- | --- | --- |
| #347/#350 旧 CommercialOverview 的套餐/entitlement/usage 读合同；旧 commercial-wallet、self-service-subscription-purchase | 当前 serving 已用 unified commercial overview；旧只读 Service/HTTP/builder、套餐购买/激活适配和未接线恢复循环退出 | 当前 `commercial/billing` 的资源/服务/钱包订单；当前总览的 availability、组织授权及隐私边界 |
| #478 前期订阅套餐和 StoreQuotaLedger KEEP | 按已批准统一基础方案退休 store_count 的 allocation/bucket/lease/补偿及 fresh schema；不造无限 quota | native Store 幂等/权限/事务/审计/真实服务期与当前 Resource 服务扣费 |
| #479 旧 accountallocation 与 #480 旧 Token 分配/审计 | 旧 Token 额度、窗口和分配表/API/schema 整退；不把它们迁到新包 | 当前 `ledger/orgresource` AI 点数月限、资源分配与审计，以及当前身份目录/成员授权 |
| PAY-041/PAY-042 旧多 metric usage ledger、legacy counter mirror、Studio→ListingKit entitlement 迁移 | 所有仅为旧产品服务的 reserve/commit/reverse/reconcile/outbox、启动迁移和旧测试退出 | 当前 Resource/money、AI native invocation、storagecapacity 各自事实及事务/恢复 |
| #958/#977 等曾保留 Dub/OpenMeter 准备边界 | 本批拟结束 OpenMeter 影子计量的仓库源码/脚本，不恢复投递或把 SDK 接到当前资源 | Dub 与当前推荐/分润、其它独立集成不在本批；OpenMeter 历史评估结论仍留档 |
| #934/#937 的 Retry/DB 安全 KEEP，#985 之外的当前共享能力 | 只退出旧 ledger 的消费者；不因少一个 caller 退休共享能力 | 当前 Retry、数据库、日志、AI recorder、Temporal/WithWorker canary、采集/图像/Store |
| issue357 / RUN-1 历史商业验收与旧 Compose SQL | 有效来源账号/身份/零写入断言改接当前事实；旧套餐与 Token seed、角色/配置使用点退出 | 当前来源账号隔离、租户拒绝、取消、当前角色最小权限和资源/Store 只读断言 |

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior: 当前审计、预算、组织隔离、防重复外部发送、public 事实读取和零写入断言。Current owner: AI native invocation / 原 ImageAgent run-slot budget、`ledger/orgresource` 与 `integration/orgresource`、当前 unified commercial read、Store native owner。Cutover/deletion condition: §5/§6.1 的整链退出、当前能力与原安全断言有准确 HEAD 证据、适用 CI 和最终独立检查完成；合并另授权。新代码不得依赖旧五根包，不留 alias、wrapper、fallback、双写或第二事实源。

## 3. 实际入口、消费者与根因

调查使用 tracked 源码的 Go AST import/alias/selector、同包调用、构建条件/测试/fixture 及非 Go 引用；不以零外部 import 自动判 RETIRE，也不运行应用探测实例。

```text
当前：current-application
  -> unified commercial overview -> Resource balances + Store service summary
  -> member AI point limits / native member resources -> ledger/orgresource
  -> Product Agent invocation -> native priced proof -> Resource point settlement
  -> managed image-set/artwork -> GenerationExecution -> current Resource/image owners
  -> Store official connection / observations -> current platform adapters

旧：listingsubscription Service / Handler / old read builder / purchased adapter
  -> plans / subscriptions / entitlements / counters / usage ledger / store quota
  -> legacy mirror / Studio entitlement migration / outbox -> OpenMeter PoC
  -> old accountallocation / schema / fixture / samples / scripts

混合尾部：standalone image worker buildImageAgentWorkerAI
  -> native invocation recorder + old AIInvocationUsageAdapter
  -> recordedReview -> old Token reserve/release
```

| 对象 | 基线实际证据 | 处置 |
| --- | --- | --- |
| `listingsubscription` | 生产 import 只来自 3 个旧 HTTP 装配文件、旧 schema runtime、standalone image worker、旧 accountallocation persistence、旧 read HTTP 子包和 Workbench schema | 整根退休；不是把 23 个生产 Go 文件整体搬到另一个包 |
| `NewService/NewServiceWithLedger/NewRuntimeService` 与 `NewGormStoreQuotaLedger` | 构造只在旧包/测试使用；当前 Store 不注入旧 quota，当前 billing 没有 EnableSubscriptionPurchases 调用 | 构造/旧门面、独立 Handler、Store quota 与旧购买激活链整退 |
| old commercial/accountallocation builders | 当前默认应用走 `buildUnifiedCommercialRead` 和当前 member resource/point modules；旧 builder 留在 fixture/历史验收 | 删除旧 production builder；有效测试改接当前入口，不能删除安全 gate |
| current AI/Resource | Product Agent 的 SetUsageSettler 注入 `GormModelInvocationRepository`；managed image-set 从 image_set_application 直接调用 NewManagedImageSetTemporalDependencies | 当前点数价格、native proof、余额/月限、预留/结算与恢复完整保留 |
| image worker 的旧 Token 装配 | dependencies.go仍统一注入旧adapter；但organizationMainSlotExecutor三个review方法直接拒绝，当前managed image-set不走该旧分支 | §4.1整退闲置recordedReview/Token装配，只抽取仍被当前credential resolver消费的日志脱敏；现役AI dispatch/点数合同不改 |
| public-schema CI gate | `TestCommercialReadVerifiedPublicSchemaPostgres` 只装配旧 read module 和旧 saas_* 事实 | 保留函数名/原 CI 命令，改接当前 unified read 的两组件及真实安全断言 |
| current readers | Resource ReadBalances 的两个表、Store ReadServiceSummary 的 ORM 当前查询未显式 public 绑定 | 在原两个 read owner 窄修复 PostgreSQL public 绑定；不改变余额/服务规则或全局连接 search_path |
| OpenMeter | adapter 无生产 consumer；usage outbox builder 无生产 consumer，SDK/契约 PoC 与三个脚本独立存在 | 作为待确认产品退休范围收齐；不把“目前未接线”写成当前已获删除授权 |

架构问题有两处：旧包混放已被产品决定取消的计费事实和当前审计行为；原 CI 仍保护被替代投影，未覆盖当前默认商业读。设计的解决是取消旧事实 owner、把有效安全约束接到现有当前路径，避免保活旧抽象或仅改检查文字。

已核对活跃 #979、#966、#956、#692；无同批退休 Writer。成员权限/Store schema 等共享 owner 只在本矩阵内改；不编辑他人分支或未合入能力。最终候选同步实际 main 一次，处理真实重叠；无关 main 增量可复用未变证据，不持续同步制造重审。

## 4. Owner、持久化与副作用

### 4.1 有效行为回到当前 owner，闲置审核链退休

当前 `organization_main_admission.go` 明确封闭旧 Extract/Render/Review：QuoteStagedReview、ReviewStagedSlotQuoted、ReviewStagedSlot 均直接返回原 BudgetQuoteUnavailable / ProviderNotDispatched。current managed image-set/artwork 从 image_set_application 直接构造 NewManagedImageSetTemporalDependencies，仅执行现有 GenerationExecution。`recordedReview` 虽仍有装配及专属测试，已不拥有一条当前可执行产品路径。这与 [当前图片智能体](product-image-agent-v1.md) 的 source-edit/显式生成路径一致，不能为它补新计费或发送协议。

本批因此直接退休 recordedReview、OrganizationReviewOptions/ReviewPricing、稳定旧review identity/报价/Token reserve-release/observer/SDK override 及其专属测试尾部。原 organization 审核封闭接缝和其“无报价、无额度预留、无 provider 调用”的真实断言保持。其余 v2/v3 注册与无副作用 WithWorker canary、当前 image-set/artwork/组织生成与审批/UNKNOWN 不退出，也不把已经封闭的审核重新导向未治理 provider。

**唯一生产 EXTRACT** 是仍由 `organizationCredentialAdmission` 使用的 `safeReviewTransportLogger` 及 Debug/Info/Warn/Error 四方法：从即将删除的 review_invocation.go 移到已有 organization_scope.go；字段、事件名、固定消息和丢弃传输文字/任意SDK字段的语义保持。无新包、无alias、无日志字段扩展。OrganizationExecutionAuthorizer 与当前 custom-role/status/fresh-member 检查保持。原有效敏感日志/身份拒绝断言留在当前凭据/授权测试 owner；已有等价当前证据复用，不为退出的审核重建测试框架。

BuildOrganizationImageCapabilities / routed provider 只去旧 reviewGovernance 字段、参数、录制回调和仅为它创建的配置分支；原实际 provider 端口、纯规则、组织credential resolver保持。`BuildOrganizationWorkerCapabilities` 去仅服务旧录制分支的 recorder 要求，不创建可执行的新review入口；组织执行器三个 closed 方法原样保留。保留的普通 provider Review 端口仍限它既有调用者/预算协议，不改变其已有重试或价格，更不能把旧 organizationReview 绕到此端口。

worker BuildAI移除旧CommercialDatabase Token pool、无用commercialDB参数与AIInvocationUsageAdapter注入；同时删除仅供旧录制链使用的 imageCapabilityRuntime.InvocationRecorder、BuildAI的recorder返回值/创建和对应测试stub接线。现役AI recorder本体及其他真正消费者不改。organization模式继续显式使用CommercialOwnerDatabase的原Resource runtime role/权限校验；旧两池同DB比较随已退出Token pool删除，当前Resource owner自身的角色/配置/权限和关闭责任保留。无数据库alias、隐式凭据获取或新运行前置。

当前AI native invocation schema/Token原值、InvocationDispatchClaimer/ClaimInvocation、terminal CAS/恢复、InvocationUsageSettler/InvocationUsageReservation、SetUsageSettler与Product Agent的GormModelInvocationRepository注入均有实际当前消费者，**源码和收费合同原样保留**。本批不修改这些声明来支持旧review，不扩operation白名单、不新建审核价格、不新加免费调用，不用nil/no-op结算适配器保活旧链。真实模型点数、run/slot预算、图像GenerationExecution的资源事务/UNKNOWN和受控日志继续由当前owner负责。

### 4.2 Schema、JSON/hash、恢复和 public 读取

- 旧 plans/modules/plan_modules/subscriptions/entitlements/counters/counter adjustments、usage events/buckets/outbox、subscription audit、purchased activation fence/decision、Store quota allocation/bucket、account_member_token_* 及专属 Goose version 的 DDL/安装/grant/verify 源码随 owner 退出。全新安装不再创建它们；现有表、行、版本表、角色和权限一律不操作，不提供 Drop/迁移/回填或兼容 reader。
- 移除 Studio→ListingKit entitlement 迁移、legacy counter mirror、old usage reconcile 和恢复者；不转交新 scanner、不自动接管旧 UNKNOWN。当前 money/Resource/AI/Store 的既有事务、锁、幂等/不同载荷冲突、revocation、不可逆外部副作用和恢复继续由原 owner 负责。
- `workbench/schema` 留当前 Store/audit/Resource 初始化，只删旧 quota 两调用；listingkit-schema-migrate 保留当前 Workbench 操作，退出仅供旧商业 schema 的 commercial scope/依赖和拒绝路径的错误说明。当前显式 commercial-owner schema CLI、money owner、storagecapacity、source account 等各自 installer/roles 保留。已退出 scope 由原 flag/validation 明确拒绝，不重定向到另一 owner。
- 旧订阅购买服务、orders/effect-admission/recovery 和专属 port/type 从混合 billing 源码清除。共用 Offer/Quote/Order/row、现有资源/服务/存储/钱包购买、实际订单查询/分页、server-owned quote、整数金额、订单与 resource/money proof 保留；只为已有当前 JSON/既有 shared row layout服务的字段按当前 wire 合同保留，不能再签发 SUBSCRIPTION_PURCHASE/PLAN、激活或查询入口。原字段/hash/版本不因删除一个 handler 重算。
- 当前商品/来源/草稿/资产/模型原始 Token、Resource points、Store/binding/observations 的 JSON/hash/版本、SourceIdentity 四零值槽位和已有载荷不改。历史旧订阅订单没有新恢复/迁移承诺；不清数据，不把历史证据改为已完成当前验收。
- 仅对 Resource.ReadBalances 两张表与 Store.ReadServiceSummary 单表的 PostgreSQL 查询显式绑定 public，在原 read owner 内完成；SQLite 原表名路径保持，仅适配已支持 dialect，不对 PostgreSQL missing/denied 退回 search_path。所有组织谓词、聚合公式/UTC窗口、数字编码/unknown/not_recorded 语义不改，不设置全局 search_path 或提升 runtime 角色。
- 这两个 read 各为原单 SQL 一致快照，不宣称跨两数据库原子 snapshot。沿原 request ctx/取消/timeout，不写入/seed/修复或加写锁。当前 API 某组件失败按 unified 合同返回该组件 unavailable/null；不会用 shadow 值或伪造 0。401/403、先授权后读、健康组件独立 availability 保持。旧投影统一503与旧多查询事务形状是被替代实现细节，按 §6.1准确转换，不能据此移除 public/撤权/取消/零写入断言。
- 删除仓库旧样例键、SQL 挂载或 init 声明不执行它们。真实 `.env`、DSN、PAT/token、Profile、卷、已有生成 config、既有表和进程不读取、不改写。共享 runtime secret目录仍承载当前 Resource/money 凭据，不能整目录退休或重命名；仅删旧 Token reader 键的仓库声明。

## 5. 完整删除、抽取与保留矩阵

本轮未来源码候选清单为下方 77 个整退文件、69 个有界混合工件、1 个新增完整 guard，共 147 路径；不是必须制造 143 个 diff，也不是以文件数拆批。只改实际闭包，清单外新消费者、schema/费用/权限变化先报告，不在实现中自创方案。设计阶段仍只三文档。

### 5.1 整退集

五根包 `internal/listingsubscription`、`internal/accountallocation`、`internal/integration/persistence/accountallocation`、`internal/app/schema/accountallocation`、`internal/integration/openmeter` 整根退出；以下按基线 tracked 文件展开，另收齐旧 builder、订阅购买/恢复、OpenMeter 脚本和旧 SQL：

```text
deployments/docker/referrals-compose/terraform/commercial-schema.sql
internal/accountallocation/httpapi/handler.go
internal/accountallocation/httpapi/handler_test.go
internal/accountallocation/service_test.go
internal/accountallocation/types.go
internal/app/httpapi/account_resource_allocation_module.go
internal/app/httpapi/commercial_read_module.go
internal/app/httpapi/commercial_subscription_adapter.go
internal/app/httpapi/subscription_purchase_recovery.go
internal/app/httpapi/subscription_purchase_recovery_test.go
internal/app/schema/accountallocation/migrations.go
internal/app/worker/imageagent/review_invocation.go
internal/commercial/billing/subscription_service.go
internal/integration/openmeter/client.go
internal/integration/openmeter/client_test.go
internal/integration/openmeter/doc.go
internal/integration/openmeter/entitlement_contract_test.go
internal/integration/openmeter/errors.go
internal/integration/openmeter/errors_nonwindows.go
internal/integration/openmeter/errors_test.go
internal/integration/openmeter/errors_windows.go
internal/integration/openmeter/metering_contract_test.go
internal/integration/openmeter/poc_env_test.go
internal/integration/openmeter/poc_fixture_test.go
internal/integration/openmeter/replay_contract_test.go
internal/integration/openmeter/storage_contract_test.go
internal/integration/openmeter/usage_event.go
internal/integration/openmeter/usage_event_test.go
internal/integration/persistence/accountallocation/repository.go
internal/integration/persistence/accountallocation/repository_test.go
internal/integration/persistence/commercialbilling/subscription_purchase_postgres_test.go
internal/integration/persistence/commercialbilling/subscription_repository_test.go
internal/integration/persistence/commercialbilling/subscription_service_test.go
internal/listingsubscription/ai_usage.go
internal/listingsubscription/ai_usage_test.go
internal/listingsubscription/commercial_read.go
internal/listingsubscription/commercial_read_gorm.go
internal/listingsubscription/commercial_read_schema.go
internal/listingsubscription/commercial_read_schema_test.go
internal/listingsubscription/commercial_read_test.go
internal/listingsubscription/commercial_token_quota.go
internal/listingsubscription/commercial_token_quota_test.go
internal/listingsubscription/gorm_default_catalog_upgrade_test.go
internal/listingsubscription/gorm_repository.go
internal/listingsubscription/gorm_retired_module.go
internal/listingsubscription/gorm_usage_ledger_test.go
internal/listingsubscription/handler.go
internal/listingsubscription/handler_test.go
internal/listingsubscription/httpapi/commercial.go
internal/listingsubscription/httpapi/commercial_test.go
internal/listingsubscription/mem_repository.go
internal/listingsubscription/purchased_activation.go
internal/listingsubscription/purchased_activation_gorm.go
internal/listingsubscription/purchased_activation_test.go
internal/listingsubscription/service.go
internal/listingsubscription/service_test.go
internal/listingsubscription/service_usage_ledger_get_test.go
internal/listingsubscription/service_usage_ledger_test.go
internal/listingsubscription/store_quota.go
internal/listingsubscription/store_quota_gorm.go
internal/listingsubscription/store_quota_gorm_test.go
internal/listingsubscription/store_quota_runtime.go
internal/listingsubscription/store_quota_test.go
internal/listingsubscription/tenant_display_name.go
internal/listingsubscription/types.go
internal/listingsubscription/usage_ledger.go
internal/listingsubscription/usage_ledger_gorm.go
internal/listingsubscription/usage_ledger_mem.go
internal/listingsubscription/usage_ledger_mem_test.go
internal/listingsubscription/usage_ledger_reconciliation.go
internal/listingsubscription/usage_ledger_reconciliation_test.go
internal/listingsubscription/usage_ledger_test.go
internal/listingsubscription/usage_outbox.go
scripts/lib/openmeter-poc.ps1
scripts/openmeter-poc.Tests.ps1
scripts/run-openmeter-poc.ps1
tests/imageagent_review_invocation_integration_test.go
```

### 5.2 混合工件的允许改动

| 分组 | 允许的准确改动 | 保留条件 |
| --- | --- | --- |
| worker dependencies/capabilities/organization_scope 与原测试 | §4.1去旧Token pool/adapter和闲置review录制，只抽取有效安全日志；旧专属review测试退出 | 当前运行模式/canary/凭据组织边界、预算/quote、点数 owner、取消/UNKNOWN/资产与日志脱敏 |
| config.go/loader_builder/config_env、private manifest 原测试 | 只删 CommercialDatabase 字段/加载、该块 7 个 env key及其别名/专属旧断言；字段不存在由新缺席 guard 保护 | CommercialOwnerDatabase/MoneyOwner/当前 source/Store 等配置与原 no-implicit-alias 负例 |
| billing contracts/service/resource_offers/httpapi/persistence repository | 删除 Subscription* 专属 interfaces、Service 五依赖字段/构造探测、购买/激活/reconcile methods、旧 product/kind 注册和仅此用途 helper；通用当前 helper留原 owner | 当前三资源、服务/存储/钱包订单与列表、JSON/hash/整数金额/quote/fingerprint/授权/恢复全部保持 |
| Workbench schema、旧 schema runtime 与测试 | §4.2 删除旧商业安装 scope与Store quota调用，保留明确当前 owner initializer | 当前 Store/audit/Resource/money/来源 schema、锁/幂等/拒绝边界；无Drop或改老版本记录 |
| Compose bootstrap/init/samples、K8s两worker样例和secret样例 | 删除旧 Token配置/角色/seed/query以及旧SQL mount/init调用；共享镜像/服务/卷/密钥目录/端口等无关项不变；image trial不再检查旧paid_pilot/Token表 | 当前 Resource/money/Store/来源/图像准备、原受控fixture与不自动修复策略；不读真实生成文件、不运行脚本 |
| issue357与现有浏览器/PGfixture | 删除旧套餐/Tokenseed，商业读改接当前Resource/Store；current/source-account模式及IAM/撤权/隔离/取消原场景保留 | 不建设新runner，不重新把RUN-1作为试用门禁，不改既有环境 |
| Makefile、scripts/test-fast.ps1 | 原 fast测试包列表只把已退出 ./internal/listingsubscription 替换为已有 ./internal/integration/orgresource | 其它fast包/参数和全部workflow/Required Gate不改 |
| go.mod/go.sum、integration README | 仅清除已退出OpenMeter SDK直接require/专属checksum与README投影项；共享/其它依赖保留，无版本升级或顺带tidy | Dub/现用Eino/OpenAI/HTTP/S3/Goose/pgx/GORM与当前工具/PoC保持 |
| 当前 unified read、Resource balances、Store summary及原测试 | §4.2 public事实绑定与§6.1原gate改接 | 当前payload、组件availability、组织scope、零写入、取消和现用price/Resource合同 |
| tests/边界/设计检查 | 仅§6.1范围内转换当前owner与退出登记，并新增一个完整absence/import/fixture guard | 其余永久禁令/AST解码/报错失败关闭、安全断言保持，不加skip/短路放行 |

```text
Makefile
deployments/docker/account-compose/README.md
deployments/docker/account-compose/bootstrap.sh
deployments/docker/account-compose/business-db-init.sh
deployments/docker/account-compose/image-trial-init.sh
deployments/docker/account-compose/init.sh
deployments/docker/acquisition-compose/bootstrap.sh
deployments/docker/acquisition-compose/docker-compose.yml
deployments/docker/acquisition-compose/init.sh
deployments/docker/referrals-compose/bootstrap.sh
deployments/docker/referrals-compose/init.sh
deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-deployment.yaml
deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-v3-deployment.yaml
deployments/kubernetes/listingkit-workbench/base/secret.example.yaml
go.mod
go.sum
internal/app/httpapi/account_audit_browser_fixture_test.go
internal/app/httpapi/chain1_local_product_acceptance_test.go
internal/app/httpapi/commercial_read_browser_fixture_test.go
internal/app/httpapi/commercial_read_integration_test.go
internal/app/httpapi/commercial_read_schema_binding_postgres_test.go
internal/app/httpapi/current_application_permissions_postgres_test.go
internal/app/httpapi/current_application_startup_context_postgres_test.go
internal/app/httpapi/current_store_center_postgres_test.go
internal/app/httpapi/issue357_runtime_test.go
internal/app/httpapi/issue357_validation_test.go
internal/app/httpapi/knowledge_application_crash_fixture_test.go
internal/app/httpapi/membership_integration_test.go
internal/app/httpapi/referral_registration_browser_fixture_test.go
internal/app/httpapi/unified_commercial_read.go
internal/app/runtime/currentapplication/config_test.go
internal/app/runtime/currentapplication/unified_config_test.go
internal/app/runtime/listingkitschemamigrate/runtime.go
internal/app/runtime/listingkitschemamigrate/runtime_test.go
internal/app/worker/imageagent/capabilities.go
internal/app/worker/imageagent/dependencies.go
internal/app/worker/imageagent/dependencies_test.go
internal/app/worker/imageagent/organization_main_admission_postgres_test.go
internal/app/worker/imageagent/organization_main_admission_test.go
internal/app/worker/imageagent/organization_main_temporal_postgres_test.go
internal/app/worker/imageagent/organization_scope.go
internal/app/worker/imageagent/organization_scope_test.go
internal/commercial/billing/contracts.go
internal/commercial/billing/httpapi/module.go
internal/commercial/billing/httpapi/module_test.go
internal/commercial/billing/resource_offers.go
internal/commercial/billing/service.go
internal/core/config/config.go
internal/core/config/config_env_test.go
internal/core/config/loader_builder.go
internal/integration/README.md
internal/integration/orgresource/balance_read.go
internal/integration/orgresource/balance_read_test.go
internal/integration/persistence/commercialbilling/repository.go
internal/storecenter/service_summary.go
internal/storecenter/service_summary_test.go
internal/workbench/schema/runtime.go
internal/workbench/schema/runtime_test.go
scripts/issue357-runtime.mjs
scripts/issue357/compose.mjs
scripts/issue357/contract.mjs
scripts/test-fast.ps1
tests/architecture_docs_test.go
tests/commercial_schema_contract_test.go
tests/imageagent_organization_integration_test.go
tests/import_boundaries_test.go
tests/listingkit_image_workflow_boundary_test.go
tests/openmeter_poc_boundary_test.go
web/listingkit-ui/scripts/commercial-ui-acceptance.mjs
```

新增唯一完整 guard：`tests/legacy_subscription_metering_retirement_test.go`。它覆盖五根与专属尾部缺席、retired import/alias/移动声明和旧构建/配置/SQL/PoC入口禁入；保留有效 current owner，不造新的检查框架。

### 5.3 必须保留的实际能力

- 当前身份/ZITADEL/membership/authz、SourceIdentity与采集、原Amazon/1688浏览器及deadline/取消、SDS/POD、图像与资产审批/发布。
- 当前 `unified_commercial_read.go` / 基础方案JSON、`ledger/orgresource` 全部余额/点数/月限/资源分配、`integration/orgresource` 写入/恢复；本批不改其计费状态机或价格。
- current Product Agent的native priced invocation/SetUsageSettler、GormModelInvocationRepository及原恢复；current generation、storagecapacity与money/Resource购买链。
- Store观察同步、官方连接、native Store成员授权/幂等/审计/服务期与Store-Resource扣费。#985已退出独立SHEIN登录不恢复。
- 原 `cmd/commercial-owner-schema-migrate/main.go` 与 main_test、`integration/orgresource/runtime_permissions_test.go` 对旧商业角色的拒绝/隔离负例；只是存在旧角色字符串，不属于删除消费者。
- `tests/account_compose_database_name_test.go`、`tests/commercial_readiness_workflow_test.go`、前端 commercial-billing.test.ts对旧 subscription product 的拒绝负例，全部workflow、inventory（CURRENT/历史/计数/ceilings）、依赖安全/保护规则保持。Makefile精确包替换不改变workflow源码。

### 5.4 直接文档尾部

未来实施允许对下列旧文档仅追加最前方 SUPERSEDED/RETIRED 源级注记，指向本文与统一基础方案；其原正文、失败/验收记录、数字/勾选逐字保留：

- `docs/architecture/commercial-wallet-billing-contract.md`
- `docs/architecture/self-service-subscription-purchase-contract.md`
- `docs/architecture/store-center-current-application-v1.md`
- `docs/architecture/openmeter-shadow-metering-poc-report.md`
- `docs/superpowers/specs/2026-08-13-openmeter-shadow-metering-poc-design.md`
- `docs/superpowers/plans/2026-08-13-openmeter-shadow-metering-poc.md`
- `docs/engineering/issue347-commercial-read-contract.md`
- `docs/engineering/issue350-console-commercial-ui.md`
- `docs/superpowers/specs/2026-08-14-pay-042-listingkit-generation-usage-cutover.md`

当前 authority `unified-base-prepaid-resources-v1.md`可追加本批源码owner落点与历史链接，不改既有产品/费率/JSON合同；当前 `account-compose/README.md`只准确删除旧Token准备要求并保留现用步骤。本文/架构索引/Register更新当前段，下方历史全文不改。无关历史文档不追求文字“零命中”。

## 6. 实施顺序与验证

设计确认及 IMPLEMENTATION_READY 读回后，同一 Writer/分支/主要 PR连续完成：

1. 用原安全检查和新增完整 guard对真实旧工件先RED；保持原organization审核封闭/无provider和当前AI/Resource的Claim/terminal/UNKNOWN证据，必要日志脱敏断言改接当前凭据owner；public gate先对当前真实reader证明shadow回归RED。禁止改检查先绿再找理由。
2. §4.1 最小日志脱敏抽取、闲置审核退出和当前public读取修复先形成可验证行为；原point/Resource/Store/图像路径留当前owner，不新增legacy财务合同。
3. 五根包、旧subscription purchase/恢复、schema/config/样例/SQL/PoC/fixture专属闭包一次退出；原guard与直接文档同时转换。
4. 只做必要原测试：相关root/worker/AI recorder与point/Resource/Store/billing测试；原PG public绑定gate、审核持久化/并发回归和现有fixture；必要命令仅编译 current-application、image-agent-temporal-worker、listingkit-schema-migrate、commercial-owner-schema-migrate、store-center-schema-init、1688-public-browser-collector。Pester/Node仅检查已改脚本的原fake/stub测试，不运行init/真实provider/CLI。
5. 稳定候选运行原完整PR CI并按准确HEAD核对。静态设计不执行runtime；未来若原必要PG测试需要资源，只允许任务临时PostgreSQL与合成fixture，并由实施授权一并明确。没有真实执行就保留NOT_RUN，开发自检和AI评审均不等于产品验收。

### 6.1 原检查与准入校准矩阵

以下是待用户确认的精确转换范围；没有通过删安全断言、放宽门禁或missing-skip的授权。

| 原检查/工件 | 本批允许转换 | 不可丢失的有效断言 |
| --- | --- | --- |
| `TestCommercialReadVerifiedPublicSchemaPostgres`（原ci.yml命令不改）及原commercial-read PG/browser/integration fixture | 同一函数改接当前unified商业读、Resource/Store事实；旧subscription/Token seed/field期待退出 | public与同名schema shadow分别/一起/新连接/四真实连接；撤SELECT、真实表缺失、只剩shadow均不能读shadow；组织A/B、ctx取消/超时、连接更换、读取前后行值/xmin零写入；当前组件失败unavailable/null而其它健康组件独立，凭据/表细节不泄露 |
| `tests/imageagent_review_invocation_integration_test.go` 与原worker organization/review测试 | 旧recordedReview专属故障矩阵/Token释放/SDK override测试随整链退出；仅实际复用的日志脱敏与组织credential拒绝断言抽到已有organization_scope/organization integration测试 | 原organization_main_admission的三个review闭口/无预留无发送断言保持；当前nativeAI Claim/terminal/取消/UNKNOWN、防重复与点数价格/授权测试原样保留并引用准确证据，不把旧专属测试删除记成当前功能PASS |
| `dependencies_test.go` / organization admission / real Temporal unpriced原测试 | 去旧CommercialDatabase fixture和Token表count，统计当前native dispatch/Resource reservation/余额事实；原测试仍覆盖真实保留运行模式 | role与owner权限preflight、所有失败连接清理、原quote/缺价不dispatch、真实generation不重复、撤权/截止/预算/canary保持 |
| `current_application_permissions/startup_context`、account_audit/member/source/Knowledge/chain1/currentStore fixture原测试 | 仅删旧Token/订阅安装/角色行、原commercial projection改接当前模型 | 原当前角色完整allow/deny、跨组织、public/schema权限、搜索路径、ctx/清理、未授权不查询、来源保存/读取拒绝；不得整体跳过PG场景 |
| `workbench/schema/runtime_test.go`、schema runtime/旧commercial scope原测试、commercial_schema_contract原测试 | 当前表集合移除旧表、旧quota非空列检查改验当前同类组织列与原Store/Resource约束；显式证明旧DDL/SQL路径不存在；旧scope改为拒绝 | 当前owned表精确集合/可重复/不触碰非owner表/原约束/ nil拒绝，当前迁移锁/版本/权限不变 |
| `billing/httpapi/module_test.go`、当前资源/钱包合同测试 | 旧订阅购买专属三测试退休，订单过滤原测试改为拒绝旧kind/product；当前共用helper只去旧激活分支 | 当前订单/quote/金额/幂等/分页/请求边界、跨企业、真实资源/资金UNKNOWN及撤权不变；专属订阅恢复3测试文件随退出，不冒充当前产品PASS |
| `tests/import_boundaries_test.go` 的 businessHTTPPackages 与 AppHTTPAPIListingKitSupportImports allowlist | 去五根/旧HTTP的精确allow项，完整guard永久禁入，保留原算法/转义解码/错误处理 | 其它domain与root/legacy禁令，不能把缺根当异常跳过；负例中的retired字面量继续允许作为负例 |
| `tests/listingkit_image_workflow_boundary_test.go` 的 listingKitImageBoundaryRules | 去旧subscription迁移例外和旧包保活分类；迁移根不存在由完整guard强制 | 原retired mutation负例、当前图像owner及全部其它规则/扫描失败关闭 |
| `tests/architecture_docs_test.go` 两个旧商业/自助订阅contract测试 | 当前owner锁定改读已批准unified产品authority与本文；旧自助购买文档改验SUPERSEDED/历史保留 | server-owned quote、Idempotency-Key、RECONCILIATION_REQUIRED、租户权限、正向授信/浏览器价格禁令等当前有效底线留在现有当前合同/测试 |
| `tests/openmeter_poc_boundary_test.go` | SDK必须隔离转换为本批adapter/脚本缺席、SDK禁止重入当前runtime；保留原runtime/deployment隔离算法及当前负例 | 不新增PoC运行、外部依赖或prod服务；不把专属契约测试删除记为当前metering成功 |
| `config_env_test.go` / manifest原测试 | 旧字段加载改为缺席；原显式current owner与禁止implicit alias检查继续 | 所有当前环境绑定、no-secret-log、权限与owner配置不变 |
| Makefile / scripts/test-fast.ps1 | §5.2唯一包目标替换 | 原命令参数、其它包、全部workflow和Required Gate不变 |

其它当前安全检查保持。实现发现矩阵遗漏时先查同根兄弟路径，提出具体最小修订；不能用“整批退休”自行降低Must或扩大到新费用/schema/权限事实。此矩阵允许在列明测试工件内改接fixture与owner，不要求为了已授权对象的纯机械编译修订反复申请逐行确认。

## 7. 复核、授权和停止点

- 当前交付只有三文档；实现、合并、运行、旧表/凭据/进程处理、#301关闭均未授权。设计准备完成后按准确设计HEAD核对原CI与适用Architecture Review，再交用户确认本方案（包括旧审核录制链退出与安全日志抽取、旧订阅购买闭包、schema/样例与检查转换）。
- 独立检查读取真实消费者、最小EXTRACT与正确owner、§4.2/§6.1、当前图像/资源/Store保留。若当前Must无法完成、租户越权、重复计费/不可安全恢复的provider effect、数据损坏或错误owner成立，按AGENTS分类BLOCKER并指出对应类别；纯理论扩展按IMPLEMENTATION_TEST/BACKLOG处置。
- 原GitHub复核启动失败只作状态证据，至多一次必要重试，不伪造PASS。未经用户本轮授权不默认新增AI互验线程；设计与源码最终复核授权分开，旧#985授权不覆盖本批。没有结论则保持AWAITING_REVIEW/NOT_READY。
- 完成用户确认与准入后，单实现者同一主要PR连续实施，不按文件/提交/内部模块机械拆批。高风险边界和最终交付检查按现有AGENTS执行，未变代码证据如实复用，修复只复核相关增量。
- 最终交接分别给出源级退休、必要自检、准确PR HEAD/CI、独立检查及NOT_RUN；实际合并使用最终准确expected-head且须另获授权，再核对实际merge SHA的整run main CI。不会把green CI或另一个AI当产品验收，不因此关闭#301。
