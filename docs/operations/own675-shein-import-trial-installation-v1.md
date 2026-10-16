# own675：SHEIN 带入的最小复用安装方案

状态：`STORE_CORE_ALIGNED / REMAINING_INSTALLATION_NOT_AUTHORIZED / APPLICATION_NOT_PROVIDED`。2026-10-16 用户已单独授权并完成 own675 Store 核心两表四项 CHECK 对齐、Goose 版本登记及重入核实；其余安装仍未执行。本阶段只整理真实应用/回调、私密配置和剩余接线的准备条件，不改实例、生成密钥或调用真实平台。用户暂时无法提供真实应用资料，先交付准备清单。

同日最终同步发现 main 的 PR #734 已将两张 Walmart 执行表纳入现有官方安装器和 SupplyChain 必需权限。用户明确批准代码与方案跟随当前 owner，从原 11 表调整为下列 13 表；不拆写旧版安装器或增加缺表 fallback。原 11 表测试证据保留，当前 13 表的代码、增量复核与 CI 另在 PR #736 绑定。此决定不构成 Product/Asset/Temporal 及官方配置的执行授权。

**Store 核心差异已经处理，尚未启用 Store 或供应链。** 按 [Store 核心对齐合同](../architecture/own675-store-core-alignment-v1.md)，四项准确 CHECK 已 validated，原 Goose 表仅新增 `2026101601`，原表/业务内容/授权保留，原命令只读重入核实通过。执行证据维护在 Issue #735 / PR #736；不再把这一步列作待安装，也不以它代表真实应用、店铺或完整 SHEIN 试用就绪。

执行 Issue：[#735](https://github.com/qq550723504/task-processor/issues/735)，主要 PR：[#736](https://github.com/qq550723504/task-processor/pull/736)。本次合同/代码核对基线：`deaf7453bddd74d6fa1deece043d3832a5e51720`，2026-10-16；实例后台/前端仍为 `23a301245`。实际执行前重新核对实例和准确候选，不把本次只读证据当作之后的运行验收。

## 用户结果和采用的路径

用户在 own675 正常登录，使用一个专用试用商品和已保存候选 Rn，先通过既有供应链适配/编辑入口保存 SHEIN Target Rm，再进入独立带入页，显式映射、预览并确认保存 Rm+1。保留返回草稿、商品和批次的上下文，不自动发布。

复用当前完整 SupplyChain 运行组合、CompletionDraft、Store 官方规则、Target 原事务/CAS/命令回执。没有新的业务事实 owner、平台草稿表或持久化协议，不增加离线假规则、fixture、第二套编辑器或兼容路径。

当前组合将 Target 与操作执行、官方发布回执、Asset、Temporal 一起准入。因此“只安装三张 Target 表”不能运行。此处的最小集合指**复用既有运行组合与 owner 安装单元的最低安装集合**，不是声称草稿保存本身需要全部发布表。若以后要将草稿运行与执行运行解耦，属于另一个明确的产品/实现范围，本方案不实施。

依据：[供应链架构](../architecture/my-supply-chain-shein-v1.md)、[运行说明](my-supply-chain-shein-v1.md)、[带入合同](../architecture/completion-shein-import-v1.md)。方案最初为只读文档；当前有界入口阶段的 Design Basis 为 Reuse Existing Architecture，准入在 Issue #735 记录，复用现有安装、权限、事务与精确数据库确认合同。此实现授权不构成环境安装授权。

## 只读现状

- 目标仅为 `task-processor-own675-20261014-c1e22342`。后台/前端现为 `23a301245`，均 healthy；本轮不更新其他实例。
- `product_acquisition` 已有 14 张 public 表，owner 为 `acquisition_owner`，运行角色为 `source_acquisition_runtime`；包含 Collection、Preparation、CompletionDraft 和来源事实，不重建或迁移它们。
- `store_center` 仍为原 12 张 public 表（含版本表），均归 `store_center_owner`；已完成上述四项 CHECK 与唯一 Goose 新版本，历史为 `0/2026092801/2026101101/2026101601`。店铺和 merchant bindings 均为 0，原窄 ACL 保留。后台 `storeCenter.enabled=false`，没有配置官方应用，没有已连接 SHEIN 店铺。
- 后台没有 `supplyChain`；前端 `LISTINGKIT_SUPPLY_CHAIN_ENABLED=false`。Product 无下列新增表；无独立 Supply Asset 库/运行角色。
- own675 网络命名空间内 `127.0.0.1:7233` 当前未监听。原数据库、对象存储、登录、代理、端口、37 个命名卷、图片与容量账本均保持。

## 真实 SHEIN 应用与回调的准备条件

本节是已有配置合同的准备说明（Design Basis: N/A，纯文档，不改变业务语义）。公开官方文档只能确认通用接入要求，不能代替该应用的开发者后台登记、权限或商家授权证据。本次没有登录平台、创建授权 attempt、访问 API 或读取未提供的凭据文件。

### 平台侧需要准备什么

| 项目 | 所需资料 / 条件 | 当前状态 |
| --- | --- | --- |
| 真实开发者应用 | 已注册且启用的 appId、实际业务类型；本轮只需一个应用。仓库类型为 `self_operated` / `semi_managed` / `fully_managed`，必须与真实应用、商家模式匹配 | 未提供，不代选类型 |
| 应用修订 | 本地 `version` 用于绑定配置修订及应用类型；另记录平台有无版本/审批限制。两者不可直接视为同一字段 | 待准备；不生成虚构平台注册资料 |
| 官方 API 环境 | 与实际应用环境匹配的生产 origin；当前适配器只接受 `https://openapi.sheincorp.com` 或 `https://openapi.sheincorp.cn`，授权入口沿既有 `openapi-sem.sheincorp.com` | 实际应用环境待核实；测试域名不在当前合同内 |
| 回调登记 | 平台登记的完整 HTTPS URL、平台是否接受 localhost/自定义端口，以及授权浏览器能否访问该地址 | 未提供 / 未核实，不能承诺本地地址被平台接受 |
| 接口权限 | 授权换钥、店铺信息及下面列出的站点/规则/类目/属性/品牌/仓库读取；核对应用后台实际开放情况 | 未核实，不申请全量接口或发布权限来替代核对 |
| 出口 IP 白名单 | 确认 own675 后台实际对外出口是否稳定、与应用白名单一致；该地址不是容器 loopback，也不是浏览器回调地址 | 未核实；本轮不探测公网 IP 或改网络 |
| 可授权商家 | 店铺归属、实际经营模式、US 站点/币种/仓库，以及能够在官方页面授权的商家账号 | 未提供；own675 当前店铺 0 |

官方换钥文档列明接口访问权限和线上服务 IP 白名单，并说明临时 `tempToken` 需在取得后 5 分钟内换取商家凭据。因此 token 应在实际连接时由商家新授权产生，不能作为提前交付的静态配置。见 [SHEIN 官方换钥接口](https://open.sheincorp.com/documents/apidoc/detail/3001520-1000012)；应用/商家最终状态仍须在后续获准连接时核实。

代码中当前路径涉及的接口准备范围如下；这里只列清权限，不调用接口：

| 现有能力 | 官方路径 |
| --- | --- |
| 授权换钥、读取商家身份 | `/open-api/auth/get-by-token`、`/open-api/openapi-business-backend/query-store-info` |
| 站点与规范、类目 | `/open-api/goods/query-site-list`、`/open-api/goods/query-publish-fill-in-standard`、`/open-api/goods/query-category-tree` |
| 属性、关联规则、品牌 | `/open-api/goods/query-attribute-template`、`/open-api/goods/get-associated-attribute-rules`、`/open-api/goods/query-brand-list` |
| 仓库、可发品条件读取 | `/open-api/msc/warehouse/list`、`/open-api/goods/product/check-publish-permission` |

最后一项是现有规则读取，不是发品操作。本轮不准备订单/观察同步或商品发布调用，也不改变仓库现有权限边界。

### 回调接线与浏览器条件

代码中的独立页为 `/workbench/stores/shein/callback`，own675 的候选完整地址是 **`https://localhost:39744/workbench/stores/shein/callback`**。本地代理的 application 路由将该路径交给前端；原 HTTPS 入口可正常校验证书，但尚未做真实授权后的回调验证。平台是否允许该登记值、是否需要其他域名均为 `NOT_VERIFIED`。不根据 localhost 推定已被允许，也不根据资料缺失推定必须开放公网。

- `callbackURL` 必须为完整 HTTPS 地址，无 userinfo、query 或 fragment，与实际登记和访问入口一致。它是 SHEIN 商家连接回调，不是 ZITADEL 登录回调或 webhook，不替换已有登录配置。
- 从同一浏览器标签页、同一 origin 的正常 Store 入口发起并返回；`localhost`、`127.0.0.1`、不同端口/域名不能混用。当前恢复上下文依赖该标签页的 sessionStorage，并核对原 actor、企业、attempt 和有效期。
- 当前适配器只处理唯一的 `appid`、`tempToken`、`state` 查询参数；前端捕获后立即清除地址栏参数，仅在内存保留。确认账号/企业后，用户点击“完成官方连接”才经本地 `/api/workbench/stores/<storeId>/connection/complete` 完成；不手工拼 token URL 或将完整回调写进日志/Issue。
- 现有客户端要求换钥响应回显一致的 appid/state。官方应用若实际返回协议不符，停在具体差异，不加宽松 fallback。刷新导致一次性参数丢失或结果不确定时沿原 attempt 查询/恢复，不自动重放换钥。
- 若平台不接受本地地址，先明确最小域名/TLS/访问方案及对原登录 origin 的影响，再单独授权；本阶段不建 tunnel、开放端口、改证书信任或更换原访问入口。

依据：[应用配置解析](../../internal/app/runtime/currentapplication/official_store_connection.go)、[官方协议适配器](../../internal/integration/shein/official_client.go)、[回调页](../../web/listingkit-ui/src/app/workbench/stores/shein/callback/page.tsx)、[回调处理](../../web/listingkit-ui/src/components/workbench/stores/shein-authorization-callback.tsx)。

### 私密配置及文件清单（只列字段，不生成或写入）

| 配置 / 文件 | 来源与内容 | 最小保管要求 |
| --- | --- | --- |
| `appSecretFile` | 平台为该 appId 签发的 App Secret 原文 | 私密独立文件；不放仓库、聊天、镜像、前端变量或命令参数 |
| `credentialKeyFile` + `credentialKeyId` | 本地生成的 32 字节随机 AES-256 密钥，以标准 Base64 保存；keyId 为稳定、非敏感标识 | 与 App Secret 分开；用于保护保存的商家凭据，须与其备份一同保留，不任意换钥或覆盖 |
| Asset owner / runtime 凭据 | 后续为新库两个角色分别生成，维护配置只给安装器，runtime 配置只给后台 | 不复用或轮换原 Product/Store 凭据；owner 配置不挂入 serving |
| 新私有后台 manifest | 原 manifest 副本，仅增加下面配置差异及上述文件绝对路径 | 原文件/授权/其他配置保留，不上传原文或凭据 |

官方应用两个文件须为不同的绝对路径、普通非空文件，各不超过 1024 字节；当前读取器拒绝符号链接。Windows 宿主文件遵循现有私有 ACL 检查，Linux 服务内文件不允许 group/other 权限，按运行 uid 可读且只读挂载（如 `0400` 或 `0600`）。Docker/Compose 默认 `0444` 文件不能直接假定通过检查；后续准备时需核实服务内实际 uid/mode。App Secret 由平台签发，本地 AES key 由本系统生成，两者不能互换。现有文件合同还要求 App Secret 为有效 UTF-8、去除首尾空白后不少于 16 字节且不超过 512 字节；这只是仓库准入检查，不代替平台规范。

`openKeyId`、商家 `secretKey`、临时 token/state 不作为提前手填的后台配置；授权后沿现有 Store owner 加密保存和读取。当前未生成任何新密钥、数据库密码、私有 manifest 或 secret 卷。后续只需经私密文件交接实际资料，无需在聊天发送秘密。

## 需要新增的表和最小运行权限

### 现有 Product 数据库：新增 13 张，原 14 张及原授权保留

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
| `listing_walmart_sku_bindings` | 当前官方 owner 安装单元的 Walmart SKU 执行绑定 | S/I/U |
| `listing_walmart_feed_observations` | 当前官方 owner 安装单元的 Walmart feed 观察记录 | S/I |
| `product_title_proposals` | 读取当前有效商品及 Review 投影 | S |
| `product_title_operations` | 既有 Review schema 安装单元的配套表 | 无；保持 owner-only |

运行组合直接要求 12 张表的权限；既有 `reviewpersistence.InstallSchema` 同时创建最后两张表，其中 `product_title_operations` 保持 owner-only。方案复用该 owner 安装单元，不为减少一张空表复制或拆写 Review DDL，也不授予 title producer 权限。Walmart 两表是当前通用 SupplyChain 启动的既有依赖，不因此配置 Walmart 应用、店铺或执行平台调用。原 `listing_preparations`、`listing_preparation_sources` 和三个 CompletionDraft 表已经满足权限，不新增重复授权或改写事实。

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

Product 增量入口不操作 Store 表或 ACL。Store 已完成的独立阶段仅调整两张既有表四项 CHECK 和原 Goose 元数据，继续保留 `store_center_runtime` 窄授权；本清单不要求再次改 Store schema。不开启 Observations/Cockpit/TikTok/Walmart Store capability，不创建/授权其可选表。Source/身份/商业/资金/容量 owner 的 schema、凭据及授权保持不变。

业务授权继续按当前企业、actor/member 和 live Store 访问核对：读草稿使用 `workbench.supply.read`，带入预览及本地保存使用 `workbench.supply.manage`，选择店铺使用 `workbench.store.read`。当前 Store 为空，后续新建/更新/启用店铺分别沿 `workbench.store.create/update/lifecycle` 原权限。发布沿独立的 `workbench.listing.submit` 检查，本方案不为带入追加发布权限，也不改已有角色。

## 需要新增或调整的配置

保留原私有 manifest，生成同候选的副本；只增加以下项。最终值经私密文件交接，不在仓库、Issue、命令输出或聊天里填写密钥。

| 位置 | 最小变更 |
| --- | --- |
| 后台 `supplyChain.assetDatabase` | 新 Asset DB 的 loopback host/port、`supply_asset_runtime` 私有凭据、database、maxConnections=4 |
| 后台 `supplyChain.temporalAddress` / `temporalNamespace` | own675 网络内 `127.0.0.1:7233` / `default` |
| 后台 `storeCenter.enabled` | `false → true`；原 Store DB 连接及窄 runtime 身份不变，schema 对齐成功不会自动启用该项 |
| 后台 `storeCenter.officialApplications` | 只增加一个实际 SHEIN 注册应用；`platform: shein`、实际 `type`、appId、version、apiOrigin、callbackURL、appSecretFile、credentialKeyFile、credentialKeyId |
| 应用文件挂载 | 新 App Secret 与 AES key 只读挂入后台的两个独立绝对路径，运行 uid/mode 满足上述私有文件合同；不挂维护配置或泄露至前端 |
| 后台启动参数 | 指向上述新 manifest；原 identity、Product/Store/商业/图片/容量等字段逐项不变 |
| 前端 | 仅 `LISTINGKIT_SUPPLY_CHAIN_ENABLED: false → true`；保留 Collections/CompletionDraft、登录及其他环境变量 |
| 后台/前端镜像 | 更新为重新核对、另获授权的同一 PR #736 准确候选；当前实例仍为旧 `23a301245`，Store 维护未替换应用镜像 |
| Compose | 仅 own675 新增 `supply-temporal` 服务、专属持久卷及新卷的一次权限准备；使用现有 runtime-network，无新增宿主端口 |
| 启动辅助文件 | 追加本轮 overlay，先等专属 Temporal healthy，再启动同候选后台/前端；保留原配置与辅助文件备份 |

Temporal 复用仓库已固定的官方镜像 `temporalio/temporal@sha256:cf86707827fac99e4d1c4a47dc11b105382d796199c7bd41fb3213fb0471628e`、`server start-dev --headless --ip 127.0.0.1 --db-filename /data/temporal.db`，专属卷建议为 `<own675-project>-supply-temporal`。SQLite 内部 schema 由官方程序管理，不另写业务 DDL；这是内部试用运行方式，不表述为生产安装。不能整份启用 data-services/image-agent overlay 或连接其他实例的 Temporal。

新增 Asset 和官方应用私有文件放入本项目私有配置边界；原凭据不轮换、不搬移、不公开。UI/login 的 32768 请求头上限、原 CA、TLS、认证、图片代理、sourceMedia 实例 pin 和容量 true 保留。不开 `IMAGE_PROXY_ALLOW_PRIVATE`，不放宽平台图片验证；原 localhost 来源图片保留，不因本地可读而声称可供 SHEIN 使用。

应用类型由实际注册应用决定，不能为了匹配候选币种伪造 self/semi/fully-managed。一个真实应用即可满足本轮范围，不要求凑齐三种类型。回调必须与实际注册及现有回调协议匹配；若供应商不接受该本地 HTTPS 回调，不擅自开放公网或绕过验证，保持未连接。

## 执行前仍缺少的条件

1. **当前 13 表增量入口已补齐。** `cmd/product-acquisition-init --enable-supply-chain` 复用现有 owner 安装器，隔离 PostgreSQL 的非空基线保留、精确权限、并发重入、旧 11 表不完整状态拒绝及回滚测试已通过。代码、独立检查与 CI 的最终绑定记录在 PR #736；尚未对 own675 执行该入口。
2. 一个真实 SHEIN 应用的上述配置和私有文件，以及用户对随后正常店铺连接/官方规则读取的明确授权。当前没有应用或店铺，安装表本身不能产生官方规则。只读方案阶段不发起这些请求。
3. 店铺通过现有 Store 流程达到当前成员可访问、连接有效、服务启用且未过期。沿既有权益/资源合同；不直接写 active 状态，不自动购买套餐、扣费或变更现有权益。
4. 剩余方案及准确候选的安装授权。用户仅单独批准且完成 Store 核心阶段；Product 13 表/权限增量、Asset 新库/角色/四表、Temporal 新服务/卷、应用密钥与 manifest/前后端更新均未执行，仍未获准执行；业务试用写入也未授权。
5. Store 核心四项 CHECK 和 Goose `2026101601` 已完成、重入核实通过。后续维护前只读核对仍满足 B+版本，不把版本记录当作应用启用或店铺连接证明，不重复 fresh 初始化。

既有 `privateDraftTrial` 只有七个 GET、没有规则或保存能力，且与当前 CompletionDraft 配置冲突，不能用它替代本次带入确认。

## 增量入口和执行顺序

### 已批准实现范围的入口合同

入口为 `product-acquisition-init --config <私有owner-manifest> --enable-supply-chain --confirm-database <准确Product数据库名>`。`--enable-supply-chain` 与 `--confirm-empty-database`、其他 feature 增量开关互斥；拒绝额外位置参数。保留原空库与 plan/completion 入口语义。增量 manifest 明确 `collections:true,supplyCompletionDrafts:true,supplyChain:true`，本阶段不组合其他可选 capability，避免把未授权功能的安装混入。

维护连接沿现有私有文件、literal loopback、准确数据库名和 30 秒 context；不接受 runtime 身份。数据库中现有 14 张 public 基线表必须归同一当前维护 owner、满足现有 owner schema/keys 与当前 runtime allow/deny 合同，不能用其他同名表/角色继承或凭据轮换取得准入。

同一个 Product 事务与既有 advisory lock 串行首次安装和重入。首次只对新 13 表执行现有 owner 安装器，新增授权只覆盖上表的 12 张 runtime 表，既有 14 表 ACL、表 owner 和数据不改。全部 13 表已存在时只读核实 owner、schema、约束和完整权限，不再次安装或补授权。缺失部分表（包括仅存在原 11 表）或任一不匹配即拒绝。事务失败或超时不留下部分 DDL/GRANT，提交响应不确定时原命令重入核实已完成结果，不建新的安装事实源。

维护核对复用现有 runtime 权限查询中的同一 allow/deny 清单，以固定 `source_acquisition_runtime` 为 catalog 检查对象；普通运行连接仍检查实际 session/current user，维护检查不能变成 serving 准入。没有 `SET ROLE`、运行凭据读取、role 创建/继承、全库 REVOKE 或默认授权变更。新表的 Review 校验归既有 Review owner，不能只凭表名把漂移误认成完成。

验证：CLI 互斥/准确确认/严格 manifest；隔离 PostgreSQL 中的真实非空基线保留、精确新增权限、重入及并发、非 owner/部分安装/结构或权限漂移拒绝、事务后段失败回滚与有界取消。复用现有测试依赖，不建安装/验收平台；不访问 own675、对象存储或真实 SHEIN。

增量入口采用既有 owner manifest、精确数据库名确认、`source-acquisition-init` 事务锁及 bounded context 的模式。安装目标仅 `product_acquisition`，要求维护身份为既有 owner、原 14 表和能力仍有效；DDL 与**新增表的精确 grant 增量**在同一 Product 事务内提交。与 `InstallSupplyChainSchema` 共用只安装新表的内部 helper 和现有 owner 安装器；不重跑原 Preparation 安装器，先只读核对其 schema 和来源分页索引。不得调用空库初始化或全局 `GrantRuntimePermissions` 清空后重授原权限。

只接受两个清晰结果：新增 13 表均不存在时安装；全部存在且当前 schema/约束/ACL 完整匹配时核实已完成。部分存在、同名漂移、原表/授权变化均停止并报告，不 DROP/ALTER/修复旧对象。响应丢失后先核实原事务结果，不盲目重建；最终以 `RuntimeCapabilities{Collections:true, CompletionDrafts:true, SupplyChain:true}` 的现有只读检查核对。

### 命令准备（当前不执行）

从同候选构建 `product-acquisition-init` 后，使用独立私有维护 manifest。字段沿现有严格解析器，不使用后台完整 manifest，也不写入 runtime 凭据：

```json
{
  "schemaVersion": 1,
  "database": {
    "host": "127.0.0.1",
    "port": 5432,
    "user": "acquisition_owner",
    "password": "<私密维护凭据>",
    "database": "product_acquisition"
  },
  "collections": true,
  "supplyCompletionDrafts": true,
  "supplyChain": true
}
```

端口必须取目标网络中实际的 PostgreSQL loopback 端口；上述值是格式说明，不是 own675 的连接配置。私有文件权限遵循既有入口检查。随后获准执行时使用准确目标名：

```text
product-acquisition-init --config <私有owner-manifest绝对路径> --enable-supply-chain --confirm-database product_acquisition
```

成功首次执行提交 13 表和精确权限增量；相同命令重入仅核实。拒绝时保留现状，先查明缺失/漂移原因，不改成空库命令或放宽授权。

本轮开发证据包括 CLI 互斥与位置参数检查、PostgreSQL 17.2 的非空基线保留/并发重入/结构及列级权限漂移/后段回滚/取消，以及原供应链、图片和带 integration 标签的数据服务安装回归。原 14 表的行、owner、表级/列级 ACL、列定义、约束和索引以隔离测试前后快照比较；不读取或重写原图片对象。未将隔离测试结果记为 own675 安装或用户验收。

后续获准执行时：

1. 重新核对容器/镜像/原配置与角色，私密备份 Product、Store、配置和图片基线；先核对官方应用/回调可准备，避免先改变原 Product ACL 再发现凭据缺失。
2. 暂停 own675 后台/前端的写入口，保留原依赖。只读核对已完成的 Store B+版本；原配置字段及重入合同见 [Store 核心对齐方案](../architecture/own675-store-core-alignment-v1.md)，不重复改库或登记版本。维护运行环境须包含 `Asia/Shanghai` 时区库；此前最小 Alpine helper 因缺 zoneinfo 在建连阶段拒绝、未入 DDL，改用原有带时区库的应用镜像承载同一安装器后成功，失败证据保留。随后按剩余准确授权创建新的 Asset 空库/两个角色，沿既有工具安装四表和窄权限；准备专属 Temporal 卷及服务。不修改原对象存储，各阶段记录实际完成状态，不冒充整实例原子操作。
3. 使用已完成检查的 Product 增量入口，事务安装 13 表和上表 grant；原 14 表及原 ACL 保持。外部新库/新服务与 Product 事务不是一个原子事务，失败停在已知阶段，保留资源，未齐备不启用新配置。
4. 核对配置差异仅为上表项目，启动同候选后台、前端。后台只验证 schema/grants，不在启动中执行安装。既有 supply worker 会运行，但不自动创建 adapt/upload/optimize 操作。
5. 用户经正常 Store 入口创建/连接一间获准的 SHEIN 店铺，再以**新专用试用商品、批次和候选**执行适配、保存 Target R1、带入预览及确认 R2；币种/仓库跟随实际商户模式。原 R1 候选和 CNY/零值样例不为试用改写。
6. 核对来源 Vn、候选 Rn、Target Rm，映射前后值、未选字段、刷新读回和返回上下文；原保存并发/幂等合同不变。缺必填项可显示诊断，不冒充可发布。必要开发/安装自检不能代签用户产品验收。

不调用上传、商品发布、图片上传或 UNKNOWN resolve，不建立 upload 操作、官方意图或回执；不新增扫描/retry。正常官方规则读取与后续用户确认的本地 Target 保存分开记录。全量运行组合仍保留既有显式发布能力，不能将“不自动发布”表述为技术上移除了全部发布入口；本轮不调整现有用户角色或扩大权限。

## 失败、停用与数据保留

- Product 事务失败则本次 DDL/grants 回滚；新 Asset 库、Temporal 卷等已创建资源保留，不顺带删除或重置。
- 新权限超出旧 Completion-only 允许集，**仅恢复旧 manifest 会导致启动校验拒绝**。需先停应用、保留当前版本程序/数据，恢复前端原开关和原 manifest，并只撤回本轮新授权的精确 delta，再核对旧能力的权限准入；原 14 表授权不动。
- 新的 Target、命令、Review、Asset 表及其内容、所有图片、命名卷和备份均保留，不以回退为名 DROP/TRUNCATE 或整库覆盖。新服务是否停止先核对有无执行中的操作；有 UNKNOWN/未完成副作用时交原 owner，不自行重试或清理。
- 一旦产生包含 sheinImport 的记录，继续保留理解该 DTO 的同候选/后续程序，不能回到丢弃元数据的旧镜像。关闭入口不抹掉记录；原镜像保留仅供追溯，不默认作为安全降级方案。

交接必须分别报告：入口补齐/CI、安装实际执行、运行健康、真实官方读取、Target 保存与读回、用户验收。当前仅 Store 核心对齐及重入已完成，原后台/前端仍 healthy；其余安装、真实官方读取、带入保存与用户试用均为 `NOT_RUN`。本准备清单不授权上述后续执行。
