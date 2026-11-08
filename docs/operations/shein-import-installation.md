# SHEIN 带入：安装与配置边界

用户在正常供应链入口，以已保存的候选和准确 Target 版本进行显式映射、预览与本地保存；不自动发布。依据为[供应链合同](../architecture/my-supply-chain-shein.md)、[带入合同](../architecture/completion-shein-import.md)及[正常操作](my-supply-chain-shein.md)。

安装按当前 owner 单元进行，不拆写 DDL、伪造规则或新增缺表 fallback。新装与已有当前系统的能力增量是不同命令模式；空库 flag 不能指向非空库。保留实例的维护、凭据、平台连接与数据操作须有其明确授权。

## 准备条件

- 选定准确 Git 版本及对应后台/前端镜像，明确 Product、Store、Asset 和 Temporal 的数据库、owner、runtime 身份及网络。
- 真实 SHEIN 应用的类型、appId、version、API origin、HTTPS callbackURL、开发者权限、出口白名单和商家授权以真实登记为准，不以示例值替代。
- App Secret 与 credential key 通过独立私密只读文件注入，不提交仓库或交给浏览器；维护凭据与运行身份分离。
- 回调使用当前 `/workbench/stores/shein/callback`。它与 ZITADEL 登录回调不同；发起/返回保持同一 origin、tab、actor/org/attempt 及有效期。参数读取后清除地址栏，用户明确确认连接，不记录 token URL。
- 后台 SupplyChain 组合需要当前 Product、Asset、Store 与 Temporal 接线；只安装 Target 三表无法满足完整组合。是否拆分运行组合须另有明确产品和设计依据。

## 当前 owner 安装单元与最小权限

### Product 的 SupplyChain 依赖

以下表沿 `acquisition_owner` 安装。S/I/U 分别表示 SELECT/INSERT/UPDATE；均不含 DELETE、TRUNCATE、REFERENCES、TRIGGER 或 DDL。运行角色仍是 `source_acquisition_runtime`。

| 表 | 用途 | 新增运行授权 |
| --- | --- | --- |
| `listing_target_records` | 不可变 Target 版本，保存 sheinImport 绑定 | S/I |
| `listing_preparation_targets` | Target 当前版本头 | S/I/U |
| `listing_target_record_commands` | 原保存命令与幂等回执 | S/I |
| `listing_preparation_operations` | 既有 adapt/upload/optimize 操作 owner | S/I/U |
| `listing_preparation_operation_items` | 操作的逐商品结果 | S/I/U |
| `listing_preparation_upload_plans` | 不可变上传计划及进度 | S/I/U |
| `listing_preparation_upload_plan_items` | 原计划成员与逐项关联 | S/I/U |
| `listing_submission_execution_attempts` | 既有执行幂等及 UNKNOWN owner | S/I/U |
| `listing_submission_target_fences` | 既有执行 fencing | S/I/U |
| `listing_submission_official_intents` | 官方副作用意图 | S/I |
| `listing_submission_official_receipts` | 官方执行回执/阶段读取 | S/I |
| `listing_walmart_sku_bindings` | 当前官方 owner 安装单元的 Walmart SKU 执行绑定 | S/I/U |
| `listing_walmart_feed_observations` | 当前官方 owner 安装单元的 Walmart feed 观察记录 | S/I |
| `product_title_proposals` | 读取当前有效商品及 Review 投影 | S |
| `product_title_operations` | 既有 Review schema 安装单元的配套表 | 无；保持 owner-only |

运行组合直接要求 14 张表的权限；既有 `reviewpersistence.InstallSchema` 同时创建最后两张表，其中 `product_title_operations` 保持 owner-only。方案复用该 owner 安装单元，不为减少一张空表复制或拆写 Review DDL，也不授予 title producer 权限。Walmart 两表是当前通用 SupplyChain 启动的既有依赖，不因此配置 Walmart 应用、店铺或执行平台调用。原 `listing_preparations`、`listing_preparation_sources` 和三个 CompletionDraft 表已经满足权限，不新增重复授权或改写事实。

DDL/约束/索引来自原 owner：`internal/integration/persistence/listing/{record,preparation,submission,official}` 和 `internal/integration/persistence/product/review`。不能使用旧 `internal/app/listingrecordstore/schema.sql`；它不是本次 Target owner。

账号保留原 CONNECT/USAGE，禁止超级用户、角色继承、跨 schema、CREATE/TEMP、序列权限和 grant option。不得运行 `GRANT ALL`、改默认权限，或以宽授权绕过既有启动校验。

### 独立 Asset 数据库

Asset 数据库使用目标环境明确的独立名称和 `supply_asset_owner`；新运行角色 `supply_asset_runtime`。不复用 Product/Store/商业库作为 Asset owner，不改其他数据库的角色或凭据。

| 表 | `supply_asset_runtime` 权限 |
| --- | --- |
| `product_approved_assets` | S/I |
| `product_approval_receipts` | S/I |
| `product_approved_inventory_heads` | S/I/U |
| `product_approved_inventory_version_heads` | S/I/U |

仅 CONNECT、public USAGE 与上述表级权限；无 DELETE/DDL/TEMP/角色继承，不新增其他数据库或 schema 授权。两个新角色均不持有 SUPERUSER/CREATEDB/CREATEROLE/REPLICATION/BYPASSRLS，运行角色不能是表 owner，维护角色及其凭据不交给服务。连接池设为 4（既有合同上限为 8）。新空 Asset 库可以使用现有 `cmd/supply-asset-init` 的明确空库安装入口；该命令不得指向非空旧库。

Product 增量入口不操作 Store 表或 ACL。Store 使用当前 owner schema 和 `store_center_runtime` 窄授权，不借 Product 安装修改 Store。不开启 Observations/Cockpit/TikTok/Walmart Store capability，不创建/授权其可选表。Source/身份/商业/资金/容量 owner 的 schema、凭据及授权保持不变。

业务授权继续按当前企业、actor/member 和 live Store 访问核对：读草稿使用 `workbench.supply.read`，带入预览及本地保存使用 `workbench.supply.manage`，选择店铺使用 `workbench.store.read`。新建/更新/启用店铺分别沿 `workbench.store.create/update/lifecycle` 原权限。发布沿独立的 `workbench.listing.submit` 检查，本方案不为带入追加发布权限，也不改已有角色。


## Product 增量安装入口

入口为 `product-acquisition-init --config <私有owner-manifest> --enable-supply-chain --confirm-database <准确Product数据库名>`。`--enable-supply-chain` 与 `--confirm-empty-database`、其他 feature 增量开关互斥；拒绝额外位置参数。保留原空库与 plan/completion 入口语义。增量 manifest 明确 `collections:true,supplyCompletionDrafts:true,supplyChain:true`。已安装的当前采集计划、数据市场和图片读取能力分别用 `acquisitionPlans`、`supplyMarket`、`imageSets` 准确声明；最终精确权限验证使用完整组合，不能省略声明以隐藏额外权限。

这些声明仅保留和验证既有能力：缺表、缺权限、错误 owner 或漂移都拒绝，不顺带安装或补权。`dataServices` 声明既有独立数据服务角色，复用其维护权限/schema 校验；`researchSupplierReview` 声明已安装的研究资料表，核对 owner 和基础 Research schema，但不替代 Research 运行身份的完整准入验证。两者不增加 Product runtime 权限；本入口不修改其表、列授权或数据。POD 与离线记录的重叠安装组合本期继续拒绝。

准备私有维护 manifest 时，按目标环境当前已启用能力填写；维护文件的 database 使用原 Product owner，绝不复制服务 manifest 的 runtime 凭据来执行安装。本次修改不授权运行实例安装、改配置或部署。

空库安装继续使用原显式入口；SupplyChain、SupplyMarket 与 ImageSets 组合共享 Review owner，只安装一次，仍使用原 schema 和最小权限。

维护连接沿现有私有文件、literal loopback、准确数据库名和 30 秒 context；不接受 runtime 身份。数据库中现有 14 张 public 基线表必须归同一当前维护 owner、满足现有 owner schema/keys 与当前 runtime allow/deny 合同，不能用其他同名表/角色继承或凭据轮换取得准入。

同一个 Product 事务与既有 advisory lock 串行首次安装和重入。15 张依赖表分为 13 张 Supply 执行表和 2 张共享 Product Review 表。13 张 Supply 表全缺时，只安装缺失的当前能力：Review 两表也全缺则调用既有 Review 安装器；Review 已成组存在则先验证同一维护 owner 和当前 Review schema，复用而不重建、修复或改写数据/owner/索引。新增授权仍限现有权限矩阵：13 张 Supply 表的原精确权限，以及 Review proposal 的 SELECT（已有时保留）；不授予 Review 写权限或 operation 读写权限，不改变既有 Review 授权及原 14 表 ACL。全部 15 表已存在时只读核实 owner、schema、约束和完整权限，不再次安装或补授权。Supply 部分安装（包括缺少上传计划两表的部分集）、Review 仅有一表/漂移/异 owner，或已安装 Supply 缺少 Review，均拒绝。事务失败或超时不留下部分 DDL/GRANT，提交响应不确定时原命令重入核实已完成结果，不建新的安装事实源。

维护核对复用现有 runtime 权限查询中的同一 allow/deny 清单，以固定 `source_acquisition_runtime` 为 catalog 检查对象；普通运行连接仍检查实际 session/current user，维护检查不能变成 serving 准入。完整权限在同一事务提交前核对，允许原本已有的 proposal SELECT；原权限缺失或额外表/列权限、grant option 等漂移仍拒绝并回滚，不通过撤权/重授来修复。没有 `SET ROLE`、运行凭据读取、role 创建/继承、全库 REVOKE 或默认授权变更。Review 校验归既有 Review owner，不能只凭表名把漂移误认成完成。


## 配置与核对

使用当前严格私有 manifest：`supplyChain.assetDatabase`、`temporalAddress` / `temporalNamespace`、`storeCenter.enabled` 及真实 `storeCenter.officialApplications`；前端使用对应的 `LISTINGKIT_SUPPLY_CHAIN_ENABLED` 开关。数据库地址、应用 origin 和回调必须取目标环境真实值。

服务启动只验证 schema 与窄权限，不执行安装。Product、Asset、新 Temporal 服务与官方授权不是一个原子事务；按已知阶段记录结果，依赖未齐备不启用入口。开发环境组合见 [Account Compose](../../deployments/docker/account-compose/README.md)。

核对原来源、候选和 Target 版本，预览选中/未选字段、确认保存、刷新读回和返回上下文。缺少必填项可以展示诊断，不据此宣称可发布。平台规则读取、本地保存、图片上传与正式发布分别授权和取证。

## 失败与停用

- Product 事务失败回滚其 DDL / grant；其他已创建资源保留，不顺带删除库、卷或对象。
- 部分安装、错误 owner、schema / ACL 漂移均停止，不放宽权限或切换空库命令。响应丢失使用原命令核实原事务结果。
- 停用入口不删除 Target、回执、图片或资源事实；有 UNKNOWN 或未完成副作用交原 owner 处理，不自行重试。
- 若撤回增量运行配置，先核对新权限与旧能力 allow/deny 的关系；只恢复旧 manifest 不保证准入。保留理解当前 DTO 的程序及数据，不默认降级到会丢弃元数据的旧镜像。
