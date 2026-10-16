# own675：SHEIN 带入的最小复用安装方案

状态：`PREPARED / NOT_AUTHORIZED_TO_EXECUTE`。本轮只读调查与方案文档；不执行 DDL、授权、凭据配置、店铺连接、业务保存或发布。

执行 Issue：[#735](https://github.com/qq550723504/task-processor/issues/735)，主要 PR：[#736](https://github.com/qq550723504/task-processor/pull/736)。核对基线：`23a301245638d972a25b5e5d7c92b778c0a49452`，2026-10-16。实际执行前重新核对实例和候选，不把本次只读证据当作之后的运行验收。

## 用户结果和采用的路径

用户在 own675 正常登录，使用一个专用试用商品和已保存候选 Rn，先通过既有供应链适配/编辑入口保存 SHEIN Target Rm，再进入独立带入页，显式映射、预览并确认保存 Rm+1。保留返回草稿、商品和批次的上下文，不自动发布。

复用当前完整 SupplyChain 运行组合、CompletionDraft、Store 官方规则、Target 原事务/CAS/命令回执。没有新的业务事实 owner、平台草稿表或持久化协议，不增加离线假规则、fixture、第二套编辑器或兼容路径。

当前组合将 Target 与操作执行、官方发布回执、Asset、Temporal 一起准入。因此“只安装三张 Target 表”不能运行。此处的最小集合指**复用既有运行组合与 owner 安装单元的最低安装集合**，不是声称草稿保存本身需要全部发布表。若以后要将草稿运行与执行运行解耦，属于另一个明确的产品/实现范围，本方案不实施。

依据：[供应链架构](../architecture/my-supply-chain-shein-v1.md)、[运行说明](my-supply-chain-shein-v1.md)、[带入合同](../architecture/completion-shein-import-v1.md)。方案文档的 Design Basis 为 N/A（只读调查与文档）；后续有界增量安装入口可按 Reuse Existing Architecture 填写准入，复用现有安装、权限、事务与精确数据库确认合同，不据本文直接开工或执行。

## 只读现状

- 目标仅为 `task-processor-own675-20261014-c1e22342`。后台/前端现为 `23a301245`，均 healthy；本轮不更新其他实例。
- `product_acquisition` 已有 14 张 public 表，owner 为 `acquisition_owner`，运行角色为 `source_acquisition_runtime`；包含 Collection、Preparation、CompletionDraft 和来源事实，不重建或迁移它们。
- `store_center` 已有 12 张 public 表（含版本表），Store schema 与所需运行授权已经存在；当前店铺数为 0。没有配置官方应用，没有已连接 SHEIN 店铺。
- 后台没有 `supplyChain`；前端 `LISTINGKIT_SUPPLY_CHAIN_ENABLED=false`。Product 无下列新增表；无独立 Supply Asset 库/运行角色。
- own675 网络命名空间内 `127.0.0.1:7233` 当前未监听。原数据库、对象存储、登录、代理、端口、37 个命名卷、图片与容量账本均保持。

## 需要新增的表和最小运行权限

### 现有 Product 数据库：新增 11 张，原 14 张及原授权保留

以下新增表沿 `acquisition_owner` 安装。S/I/U 分别表示 SELECT/INSERT/UPDATE；均不含 DELETE、TRUNCATE、REFERENCES、TRIGGER 或 DDL。运行角色仍是 `source_acquisition_runtime`。

| 表 | 用途 | 新增运行授权 |
| --- | --- | --- |
| `listing_target_records` | 不可变 Target 版本，保存 sheinImport 绑定 | S/I |
| `listing_preparation_targets` | Target 当前版本头 | S/I/U |
| `listing_target_record_commands` | 原保存命令与幂等回执 | S/I |
| `listing_preparation_operations` | 既有 adapt/upload/optimize 操作 owner | S/I/U |
| `listing_preparation_operation_items` | 操作的逐商品结果 | S/I/U |
| `listing_submission_execution_attempts` | 既有执行幂等及 UNKNOWN owner | S/I/U |
| `listing_submission_target_fences` | 既有执行 fencing | S/I/U |
| `listing_submission_official_intents` | 官方副作用意图 | S/I |
| `listing_submission_official_receipts` | 官方执行回执/阶段读取 | S/I |
| `product_title_proposals` | 读取当前有效商品及 Review 投影 | S |
| `product_title_operations` | 既有 Review schema 安装单元的配套表 | 无；保持 owner-only |

运行组合直接要求前 10 张表的权限；既有 `reviewpersistence.InstallSchema` 同时创建最后两张表。方案复用该 owner 安装单元，不为减少一张空表复制或拆写 Review DDL，也不授予 title producer 权限。原 `listing_preparations`、`listing_preparation_sources` 和三个 CompletionDraft 表已经满足权限，不新增重复授权或改写事实。

DDL/约束/索引来自原 owner：`internal/integration/persistence/listing/{record,preparation,submission,official}` 和 `internal/integration/persistence/product/review`。不能使用旧 `internal/app/listingrecordstore/schema.sql`；它不是本次 Target owner。

账号保留原 CONNECT/USAGE，禁止超级用户、角色继承、跨 schema、CREATE/TEMP、序列权限和 grant option。不得运行 `GRANT ALL`、改默认权限，或以宽授权绕过既有启动校验。

### 新独立 Asset 数据库：新增 4 张

建议数据库名 `own675_supply_assets`，位于 own675 原 business PostgreSQL 内，独立 owner `supply_asset_owner`；新运行角色 `supply_asset_runtime`。不复用 Product/Store/商业库作为 Asset owner，不改其他数据库的角色或凭据。

| 表 | `supply_asset_runtime` 权限 |
| --- | --- |
| `product_approved_assets` | S/I |
| `product_approval_receipts` | S/I |
| `product_approved_inventory_heads` | S/I/U |
| `product_approved_inventory_version_heads` | S/I/U |

仅 CONNECT、public USAGE 与上述表级权限；无 DELETE/DDL/TEMP/角色继承，不新增其他数据库或 schema 授权。两个新角色均不持有 SUPERUSER/CREATEDB/CREATEROLE/REPLICATION/BYPASSRLS，运行角色不能是表 owner，维护角色及其凭据不交给服务。连接池设为 4（既有合同上限为 8）。新空 Asset 库可以使用现有 `cmd/supply-asset-init` 的明确空库安装入口；该命令不得指向非空旧库。

Store 不新增表或 ACL，继续使用 `store_center_runtime` 当前窄授权。Source/身份/商业/资金/容量 owner 的 schema、凭据及授权保持不变。

业务授权继续按当前企业、actor/member 和 live Store 访问核对：读草稿使用 `workbench.supply.read`，带入预览及本地保存使用 `workbench.supply.manage`，选择店铺使用 `workbench.store.read`。当前 Store 为空，后续新建/更新/启用店铺分别沿 `workbench.store.create/update/lifecycle` 原权限。发布沿独立的 `workbench.listing.submit` 检查，本方案不为带入追加发布权限，也不改已有角色。

## 需要新增或调整的配置

保留原私有 manifest，生成同候选的副本；只增加以下项。最终值经私密文件交接，不在仓库、Issue、命令输出或聊天里填写密钥。

| 位置 | 最小变更 |
| --- | --- |
| 后台 `supplyChain.assetDatabase` | 新 Asset DB 的 loopback host/port、`supply_asset_runtime` 私有凭据、database、maxConnections=4 |
| 后台 `supplyChain.temporalAddress` / `temporalNamespace` | own675 网络内 `127.0.0.1:7233` / `default` |
| 后台 `storeCenter.officialApplications` | 只增加一个实际 SHEIN 注册应用；`platform: shein`、实际 `type`、appId、version、apiOrigin、callbackURL、appSecretFile、credentialKeyFile、credentialKeyId |
| 后台启动参数 | 指向上述新 manifest；原 identity、Product/Store/商业/图片/容量等字段逐项不变 |
| 前端 | 仅 `LISTINGKIT_SUPPLY_CHAIN_ENABLED: false → true`；保留 Collections/CompletionDraft、登录及其他环境变量 |
| Compose | 仅 own675 新增 `supply-temporal` 服务、专属持久卷及新卷的一次权限准备；使用现有 runtime-network，无新增宿主端口 |
| 启动辅助文件 | 追加本轮 overlay，先等专属 Temporal healthy，再启动同候选后台/前端；保留原配置与辅助文件备份 |

Temporal 复用仓库已固定的官方镜像 `temporalio/temporal@sha256:cf86707827fac99e4d1c4a47dc11b105382d796199c7bd41fb3213fb0471628e`、`server start-dev --headless --ip 127.0.0.1 --db-filename /data/temporal.db`，专属卷建议为 `<own675-project>-supply-temporal`。SQLite 内部 schema 由官方程序管理，不另写业务 DDL；这是内部试用运行方式，不表述为生产安装。不能整份启用 data-services/image-agent overlay 或连接其他实例的 Temporal。

新增 Asset 和官方应用私有文件放入本项目私有配置边界；原凭据不轮换、不搬移、不公开。UI/login 的 32768 请求头上限、原 CA、TLS、认证、图片代理、sourceMedia 实例 pin 和容量 true 保留。不开 `IMAGE_PROXY_ALLOW_PRIVATE`，不放宽平台图片验证；原 localhost 来源图片保留，不因本地可读而声称可供 SHEIN 使用。

应用类型由实际注册应用决定，不能为了匹配候选币种伪造 self/semi/fully-managed。一个真实应用即可满足本轮范围，不要求凑齐三种类型。回调必须与实际注册及现有回调协议匹配；若供应商不接受该本地 HTTPS 回调，不擅自开放公网或绕过验证，保持未连接。

## 执行前仍缺少的条件

1. **有界增量安装入口尚不存在。** `cmd/product-acquisition-init` 只有全新空库初始化和既有 plan/completion 增量入口，没有 supply 增量开关。必须先在该现有入口内补齐精确非空库安装路径及必要测试，才能执行本方案；不伪造可直接运行的命令。
2. 一个真实 SHEIN 应用的上述配置和私有文件，以及用户对随后正常店铺连接/官方规则读取的明确授权。当前没有应用或店铺，安装表本身不能产生官方规则。只读方案阶段不发起这些请求。
3. 店铺通过现有 Store 流程达到当前成员可访问、连接有效、服务启用且未过期。沿既有权益/资源合同；不直接写 active 状态，不自动购买套餐、扣费或变更现有权益。
4. 本方案及准确候选的安装授权。当前用户仅批准明确方案，尚未批准 DDL、grants、新服务/配置和业务试用写入。

既有 `privateDraftTrial` 只有七个 GET、没有规则或保存能力，且与当前 CompletionDraft 配置冲突，不能用它替代本次带入确认。

## 增量入口和执行顺序

增量入口采用既有 owner manifest、精确数据库名确认、`source-acquisition-init` 事务锁及 bounded context 的模式。安装目标仅 `product_acquisition`，要求维护身份为既有 owner、原 14 表和能力仍有效；DDL 与**新增表的精确 grant 增量**在同一 Product 事务内提交。复用 `InstallSupplyChainSchema` 及其 owner 安装器，安装前检查现有两张 Preparation 表；不得调用空库初始化或全局 `GrantRuntimePermissions` 清空后重授原权限。

只接受两个清晰结果：新增 11 表均不存在时安装；全部存在且当前 schema/约束/ACL 完整匹配时核实已完成。部分存在、同名漂移、原表/授权变化均停止并报告，不 DROP/ALTER/修复旧对象。响应丢失后先核实原事务结果，不盲目重建；最终以 `RuntimeCapabilities{Collections:true, CompletionDrafts:true, SupplyChain:true}` 的现有只读检查核对。

后续获准执行时：

1. 重新核对容器/镜像/原配置与角色，私密备份 Product、Store、配置和图片基线；先核对官方应用/回调可准备，避免先改变原 Product ACL 再发现凭据缺失。
2. 暂停 own675 后台/前端的写入口，保留原依赖。创建新的 Asset 空库/两个角色，沿既有工具安装四表和窄权限；准备专属 Temporal 卷及服务。不修改原对象存储。
3. 使用已完成检查的 Product 增量入口，事务安装 11 表和上表 grant；原 14 表及原 ACL 保持。外部新库/新服务与 Product 事务不是一个原子事务，失败停在已知阶段，保留资源，未齐备不启用新配置。
4. 核对配置差异仅为上表项目，启动同候选后台、前端。后台只验证 schema/grants，不在启动中执行安装。既有 supply worker 会运行，但不自动创建 adapt/upload/optimize 操作。
5. 用户经正常 Store 入口创建/连接一间获准的 SHEIN 店铺，再以**新专用试用商品、批次和候选**执行适配、保存 Target R1、带入预览及确认 R2；币种/仓库跟随实际商户模式。原 R1 候选和 CNY/零值样例不为试用改写。
6. 核对来源 Vn、候选 Rn、Target Rm，映射前后值、未选字段、刷新读回和返回上下文；原保存并发/幂等合同不变。缺必填项可显示诊断，不冒充可发布。必要开发/安装自检不能代签用户产品验收。

不调用上传、商品发布、图片上传或 UNKNOWN resolve，不建立 upload 操作、官方意图或回执；不新增扫描/retry。正常官方规则读取与后续用户确认的本地 Target 保存分开记录。全量运行组合仍保留既有显式发布能力，不能将“不自动发布”表述为技术上移除了全部发布入口；本轮不调整现有用户角色或扩大权限。

## 失败、停用与数据保留

- Product 事务失败则本次 DDL/grants 回滚；新 Asset 库、Temporal 卷等已创建资源保留，不顺带删除或重置。
- 新权限超出旧 Completion-only 允许集，**仅恢复旧 manifest 会导致启动校验拒绝**。需先停应用、保留当前版本程序/数据，恢复前端原开关和原 manifest，并只撤回本轮新授权的精确 delta，再核对旧能力的权限准入；原 14 表授权不动。
- 新的 Target、命令、Review、Asset 表及其内容、所有图片、命名卷和备份均保留，不以回退为名 DROP/TRUNCATE 或整库覆盖。新服务是否停止先核对有无执行中的操作；有 UNKNOWN/未完成副作用时交原 owner，不自行重试或清理。
- 一旦产生包含 sheinImport 的记录，继续保留理解该 DTO 的同候选/后续程序，不能回到丢弃元数据的旧镜像。关闭入口不抹掉记录；原镜像保留仅供追溯，不默认作为安全降级方案。

交接必须分别报告：入口补齐/CI、安装实际执行、运行健康、真实官方读取、Target 保存与读回、用户验收。当前全部安装与新业务试用为 `NOT_RUN`。
