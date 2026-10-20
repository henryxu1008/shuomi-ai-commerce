# own675 Store 核心合同对齐方案

状态：`IMPLEMENTED / EXECUTION_NOT_AUTHORIZED`。2026-10-16 方案 fa41947c3 已独立复核 PASS，用户随后批准现有命令入口及必要隔离测试；本轮不执行 own675 DDL/DML/GRANT、不更新实例或配置、不调用平台、不发布。

执行 Issue：[#735](https://github.com/qq550723504/task-processor/issues/735)；主要 PR：[#736](https://github.com/qq550723504/task-processor/pull/736)。代码依据为 `066a404be`（已包含 main `05ae7947f`）；下面实例证据是先前方案阶段的只读核对，不是安装或运行验收。

## 用户结果、依据与范围

解决 own675 后续安全运行当前 Store/SHEIN 草稿路径时的核心 schema 阻碍。推荐由原 Store owner 对当前系统自己的两张表做四项 CHECK 对齐，通过既有 Goose 记录一个新的 schema 版本；业务表、行、图片、配置、授权和原版本记录保留。不重建数据库或表，不复制/转换业务数据，不创建旧系统兼容路径。

Design Basis：**Reuse Existing Architecture**。引用 [Store 当前应用 owner](store-center-current-application-v1.md)、[当前 Store 目录](store-center-figma-v1.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[实例安装计划](../operations/own675-shein-import-trial-installation-v1.md)，以及现有 `internal/storecenter`、`internal/app/schema/storecenter` 和 `internal/platform/database/migration`。基线明确允许“新系统自身正常的 Schema 演进”，不需要将当前 owner 内的 CHECK 演进误写成旧系统历史迁移例外；此解释不扩大实际操作授权。

本方案不改变 Store 业务状态机、租户/成员权限、官方连接、CAS/幂等、权益/资金或 provider UNKNOWN owner。没有 UI 变更，现有 Figma authority 不变。Legacy decision: N/A；只消费当前合格 Store owner，无 legacy service、双读/双写或第二事实源。

## 方案阶段只读证据与根因

查询仅针对 `task-processor-own675-20261014-c1e22342-business-db-1` 的 `store_center`，使用显式 `BEGIN READ ONLY` 和 5 秒 statement timeout。未读取凭据或业务行内容；仅查询 catalog、Goose 版本及聚合计数。只读审计连接使用现有集群审计身份，不把它作为未来安装身份的准入证据。

- PostgreSQL 17.2。public 共 12 表：11 张 Store 业务表和 `goose_store_center_version`，全部归 `store_center_owner`；11 表列名/顺序与当前 core 相同。
- Store、商家绑定的行数均为 0；这不授权删表、重建或放宽未来非空数据的检查。未将这次列名核对称为完整 schema PASS。
- Goose 已有 `0`、`2026092801`、`2026101101` 三条 applied 记录。现有表无未验证约束、无 invalid/not-ready 索引、无 RLS。
- `store_center_owner` 与 `store_center_runtime` 均无 SUPERUSER/CREATEDB/CREATEROLE/REPLICATION/BYPASSRLS；runtime 无角色继承，原表级 ACL、数据库 CONNECT/public USAGE 保留。
- 本轮没有更改后台/前端镜像；仍为既有 `23a301245`。健康不表示新版本 Store 模块已在实例上完成准入。

先前只从 main #734 识别出两项 Walmart 启动差异；本次实际 catalog 进一步确认基线尚未包含 TikTok identity 约束。因此最小范围为以下四项，而非仅修两项启动报错。

| 既有表 / 约束 | own675 实际 | 当前 owner 目标 | 拟最小变更 |
| --- | --- | --- | --- |
| `workbench_stores.store_platform` | 仅 `shein,temu` | `shein,temu,tiktokshop,walmart` | 同事务替换 CHECK |
| `workbench_stores.store_tiktok_identity` | 缺失 | `platform<>'tiktokshop' OR (region='US' AND external_store_id<>'')` | 新增并验证 CHECK |
| `workbench_stores.store_walmart_market` | 缺失 | `platform<>'walmart' OR region='US'` | 新增并验证 CHECK |
| `workbench_store_merchant_bindings.store_merchant_binding_shape` | SHEIN 要非空 open key；TEMU 要空 open key | SHEIN 不变；TEMU/Walmart 要空 open key，其余原边界不变 | 同事务替换 CHECK |

两个实际架构约束必须保留在后续实现中：

1. `Migrate` 的两个既有 Goose ID 已被应用，但 `InstallCoreSchemaTx` 的创建回调随当前模型演进。旧版本号不能证明当前物理合同；再次 `Up` 不会重跑已应用的创建回调，也不能修复本差异。不得删除/伪造旧版本记录强制重跑。
2. `VerifyCurrentSchemaForCapabilities` 对 Store 核心 CHECK 无条件核对，但对 merchant shape 当前仅核对名称/类型/validated。仅补前两项报错可能掩盖 merchant 差异。未来 owner 的对齐前/后核实须精确覆盖本表四项定义，不把“同名存在”当完成。

## 已批准的最小实现边界

- 领域 DDL 和确切前/后 schema 定义由 `internal/storecenter` 唯一维护，fresh 创建与对齐后校验共用当前定义；不在 CLI 复制一套业务 schema。
- `internal/app/schema/storecenter` 复用已安装的 `github.com/pressly/goose/v3`（当前 3.27.3）、`NewWithVersionTable`、PostgreSQL session locker 和 `RunTx`。追加 Store schema version `2026101601`，不得改写旧 ID 的已应用历史。
- 现有 `cmd/store-center-schema-init` 增加 `--align-core-contract --confirm-database <准确库名>`。与 fresh/observations/Cockpit/TikTok/Walmart/grants-only 模式互斥，不接受位置参数；保留已有模式语义。默认 fresh 路径的新版本只核实 B 并登记，不得隐式对齐保留库；已有非 full-current schema 必须走显式 core-only 入口。
- 沿既有私有 owner config、literal `127.0.0.1`、固定 `store_center_owner`、连接池上限 2 和 30 秒 context；另要求实际数据库名与准确确认一致，拒绝系统库、runtime 或其他维护身份。不能因审计账号能读 catalog 就允许其取得安装权。
- 只消费已存在的 12 表和原 Goose 历史。缺表、旧 applied 版本缺失或未知版本均拒绝，不调用空库 owner 安装器、`Migrate` 的全功能入口或通用补表/补授权。
- 当前 SHEIN 范围的 Store capabilities 均 false（Observations、Cockpit、TikTokShop、WalmartUS）；不创建 5 张 TikTok 或 2 张 Walmart OAuth 表，不授予其权限，不启用应用/店铺。直接调用默认 `VerifyCurrentSchema` 会要求全部平台表，不适用于本 core-only 入口。

允许的持久化变化只有四项 CHECK 和**原 Goose 表追加一条新的 applied 版本记录**。12 表的既有业务行、已有 Goose 行、owner/ACL、列/默认值、索引、PK/FK、其他约束和数据库/schema/default ACL 均不变；不新增表、角色、成员关系、授权或凭据。Goose 是原 schema 元数据 owner，不增加安装 receipt 表、业务 operation 或新事实源。

## 前置状态、事务与重入

将 schema 简称为 A/B 仅用于说明，不持久化另一套状态机：A 是本次确认的两平台基线，除上表四项外全部满足既有 current core 合同；B 是四项均满足当前定义的完整 core。两者必须同时核实原 12 表、列类型/空值、PK/FK/索引、owner 和保留授权；不得只比较表数量或 Goose 版本。

| schema / 新版本记录 | 行为 |
| --- | --- |
| A / 未应用新版本 | 在新 Goose `RunTx` 内核实后，只做四项 CHECK 并追加版本 |
| B / 未应用新版本 | 精确核实 B 和原历史后，可作为新版本的无 DDL `RunTx`，仅由 Goose 追加版本；fresh owner 也复用此后置合同 |
| B / 新版本已应用 | 只读核实完成，无 DDL、业务 DML、GRANT 或重复版本记录 |
| A / 新版本已应用、四项混合、定义弱化/未验证、缺表或任意非预期漂移 | 拒绝并保留现状，不自动修复或覆盖历史 |

后续获准实现的调用链：CLI 精确准入 → 原 Store schema owner → 原 Goose provider/session locker → 单个 `RunTx(*sql.Tx)` → 同一事务内前置核实、锁表、CHECK 对齐、后置核实 → Goose 版本记录 → commit。原版本必须已经 applied；provider 注册保留原 ID 的认知，不允许缺失历史导致重新执行创建回调。

Goose 在单个 migration 内将 `RunTx` 和版本记录放在同一事务；本轮核对了已固定依赖源码，不自行实现事务日志或分布式锁。两个并发入口沿同一个 Store version table/Goose locker 串行，第二个重新读取已完成事实。进程外其他 owner DDL 不属于可自动合并的变更，核对发现即拒绝。

真正的 schema 核实和全部 DDL 必须在 pinned `*sql.Tx` 内完成。当前 GORM verifier 经 `db.DB()` 取连接池，不能直接拿来证明尚未提交的 DDL；未来最小实现只在 Store owner 内复用查询，并让 query receiver 支持现有 `*sql.DB` / `*sql.Tx`，不扩成通用校验框架，不从事务内另开 pool 查询。

维护前须另获准暂停 own675 的写入口；不能把 Goose advisory lock 当成业务写入锁。事务按固定顺序锁定两张目标表，持有必要的表锁直至 commit；原 PK/FK 和业务版本不变，不改既有业务行。锁等待设短 timeout（建议 5 秒），总 context 30 秒；CHECK 验证超时、冲突或不合格行都回滚并报告，不延伸到后台验证/补偿平台。其余 9 张业务表不被写入，Goose 表只追加上述元数据；完整保留对比在已暂停写入的窗口完成。

## 权限、数据与失败恢复

维护登录必须是原 `store_center_owner`，拥有原 12 表（含版本表）。事务内维护核实对固定 `store_center_runtime` 做 catalog 检查，复用当前 Store capability allow/deny 清单及原 ACL（含列 ACL/grant option）快照；不复制第二份权限定义，不用 runtime 凭据或 `SET ROLE`。若现有查询将权限对象写成 `current_user`，最小实现仅在原 owner 内分离“实际 session 身份”和“固定 runtime catalog 对象”，serving 仍必须校验真实 session/current user，不能把 owner 作为运行身份。ACL 出现漂移先停止，不能全局 REVOKE/重授。应用登录、企业/成员授权、连接凭据及商业能力完全沿原 owner。

| 失败位置 | 持久化结果与责任 |
| --- | --- |
| 私有文件/确认/owner/历史/schema/ACL 前置不符 | 无改动，维护入口报告拒绝；不生成成功版本 |
| 等锁、CHECK 验证、取消/超时、事务内后置核实或版本记录失败 | 同一事务回滚四项 DDL 和新版本行；旧表/业务事实/授权保留，释放连接和锁后可重试原准确候选 |
| commit 响应丢失或进程在提交附近退出 | 不猜成功或重置记录；原命令重入，B+新版本核实成功，A+无新版本才重新执行；混合/漂移停止 |
| commit 后应用启动或其他组件安装失败 | Store 对齐作为已知独立完成阶段保留；停止后续启用，保留所有表/数据/版本/图片/配置/授权，不能整库还原覆盖新事实 |

PostgreSQL 的 Goose 行自增序列可能在失败后留空号；不能重置序列来伪造完全未执行。幂等身份使用原 database + 新 Goose version 和精确 schema 合同，不依赖连续行 ID。

不提供自动 Down 或删除版本记录；新核心平台枚举已变化，回旧镜像不等于 schema 已安全回退。若未来确需反向调整，另行核对所有当前行及实际消费者并批准准确方案；本轮优先保留当前状态、由原 Store owner 修复前进，不删业务记录使旧约束通过。Store、Product、Asset/Temporal 不在同一事务；本合同不冒充整实例原子升级。

## 已实现的命令与当前交接

从本 PR 候选构建现有 `cmd/store-center-schema-init`；后续实际执行另获准后，命令为：

```text
store-center-schema-init --config <私有Store-owner配置绝对路径> --align-core-contract --confirm-database store_center
```

配置继续使用原严格 JSON 格式：`host`、`port`、`user`、`password`、`database`、`maxConnections`、`maxIdleConnections`。host 必须为 literal `127.0.0.1`，user 固定 `store_center_owner`；database 必须与准确确认一致且不是系统库，连接池 1–2。实际端口和密码只放入私有配置文件，不放在命令行、仓库或日志。没有新的应用配置项、runtime 授权或平台启用开关。

实现归原 Store owner 和 Store Goose runner。只读核对使用 pinned repeatable-read 事务避免并发提交造成混合读；新版本的核对和四项 DDL 使用 Goose 的同一个 `RunTx`，固定顺序锁两张表并保留 5 秒 lock timeout。新版本记录与 DDL 原子提交。原表列/默认值、原约束定义、PK/FK/索引及 runtime 的原 allow/deny 边界均核对；额外、弱化或未验证的约束不被自动修复。fresh 创建与对齐共用四项当前 CHECK 表达式，默认 fresh 命令不隐式对齐 retained core。

必要验证复用现有隔离 PostgreSQL 测试依赖和 Store owner：用真实非空 SHEIN/TEMU、成员授权、商家绑定、配置/连接记录及 Goose 历史确认保留；覆盖 A→B、B 无版本登记、B 已应用只读重入/并发、混合状态/弱约束/缺史/非 owner/授权漂移拒绝、Goose 版本插入失败回滚、取消和成功响应丢失后的重入。检查四项定义及 validated，原行/ACL/PK/FK/索引不变，只有一个新 Goose 版本；capabilities=false 的 serving schema/权限核对通过。实际网络 commit 响应丢失未另建故障框架，Goose 原子事务与成功后重入分别验证。fresh、observations 及 TikTok/Walmart 显式 capability 权限回归沿用原测试；不增加安装 runner、故障平台或人工验收系统。

原方案调查只读完成；本轮仅获准入口实现和隔离 PostgreSQL 测试，未执行 own675 DDL/版本登记/权限变更、未暂停或重启任何服务、未运行真实 SHEIN/TikTok/Walmart 操作。表、数据、图片、配置和授权保留。实例安装与产品试用均 `NOT_RUN`，实际环境操作仍须另行授权。独立检查及准确提交证据维护在 Issue/PR，不以开发自检或另一 AI 检查代签用户验收。
