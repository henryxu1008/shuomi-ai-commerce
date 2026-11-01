# 在当前 Store 安装既有订单观察模块

Status: IMPLEMENTATION_READY — Independent Architecture。独立评审绑定设计候选 `57ddbce5f6ea9a62682893ffe3edec1a093600ad`，2026-11-01 无设计 BLOCKER。当前合法19表/固定runtime-role事务内准入及原子性/重复/并发/原数据保留归 IMPLEMENTATION_TEST，须在试用安装前收敛；不代表已安装、用户验收或真实平台联调通过。

执行 Issue [#1037](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1037)，同一 Writer、分支和主要 PR #1038。

## 用户结果与依据

2026-11-01 用户要求用新合成原单完整走美国站自发货页面。`TRY-US-UI-20261101-001` 已完成关联、1件现货预留、装箱及合成运单上传；物流查询被未安装的 Store Observations 阻塞。当前安装器在事务内发现 `workbench_stores` 非空而拒绝，未建表或授权。用户随后明确选择：**先设计并评审现有试用库的模块安装支持，再开发**，保留当前实例和数据，不另建环境、不操作真实平台、不合并。

适用产品决定：[全新系统产品基线](../product/greenfield-no-legacy-migration.md)允许当前系统自身正常 schema 演进，不授权历史系统兼容/迁移；本次只在用户授权的新独立试用 `ordertrial-20261101-01` 安装现有模块。`task-processor-unified-20261010`、`orderfit-20261031` 及其他保留资源不在运行授权内。

复用 [Store observations v1](store-center-platform-observations-v1.md)、[准备运行接线](store-observations-preparation-runtime.md)、[仓库履约](order-procurement-fulfillment-v1.md)和[当前 UI Authority](../product/final-ui-ia-authority.md)。不新增页面、菜单、交互或权限。此前 preparation-runtime 的“禁止非空 Store 安装”仍约束原 fresh/preparation 入口；本用户决定仅为下述**另行显式选择的当前 Store 安装入口**提供新依据，不能据此弱化旧入口。

Scope：已有当前 Store、尚无任何观察 schema 时，精确确认数据库，原 owner 一次性安装既有四张观察表及原最小 grants，验证成功后人工部署本试用的 full-mode 配置。解除消费者订单的正常同步/详情/物流查询阻塞，并续办原单交接。

Out of Scope：通用模块管理/迁移平台、其他模块安装、旧 schema/旧观察数据升级、backfill、导入/ID映射、自动修复权限、自动启动时建表、任意库安装、保留实例/生产部署、删除或重建数据、真实平台/承运商/付款、合并与 Issue 关闭。

## 事实 owner 与调用链

Store 的原表/连接/成员/审计/凭据和原 schema owner 不变。SHEIN 仍拥有平台观察，`store_observations.commands/syncs/records/heads` 仍是唯一观察持久化；ERPNext SO/预留/DN 仍是实际履约与库存事实，轨迹不提交 DN、不改变库存。

`store-center-schema-init --observations-current-store --confirm-database <exact-name> --config <private-owner-json>` → 已有私密文件/loopback/角色校验 → 当前 Store 合同与 owner/数据库/权限 preflight → app/schema/storecenter 的有界安装事务 → 原 `storeobservations.Install` / `GrantRuntime` → 独立 runtime-role `--observations-verify` → 仅试用环境 full-mode 配置与已有 worker → 原平台订单记录页面。

新增标志只选择安装行为，不能与 fresh/general migration、alignment、browser、grants-only、verify、其他平台/模块模式组合；缺少/错误数据库确认在连接或变更前拒绝。原 `--observations-only` 和 fresh 安装继续拒绝非空 Store；原 grants-only 不能安装缺失 schema。安装不调用 Store Migrate/AutoMigrate，不增加 Goose 历史或第二张安装状态表。

## 准入、事务和不变量

1. 仅私有配置中的 `store_center_owner`、literal loopback、现有数据库，确认名须同时匹配配置和 `current_database()`；排除 postgres/template 库。session_user=current_user，public search_path，owner/runtime 均不得有超级、建库/建角色、replication、bypassRLS 或跨角色继承权限。复用现有 owner/catalog 检查，不使用 SET ROLE 或新凭据。
2. 复用当前 core schema/ownership/约束与 runtime 最小权限检查；与本次无关的可选平台表不成为新增安装前置。当前维护边界只接纳当前核心与其原合法表；已有未知/漂移/不支持的可选模块组合拒绝，不自动对齐 core、修改 grants 或补历史步骤。
3. `store_observations` 和退休的 `shein_observations` 都必须不存在。已有 schema 无论空/非空、正确/漂移、缺少 grants 均拒绝此次安装，不覆盖、不补齐、不恢复权限。原显式只读 verify/grants-only 仍用于其各自已批准场景。
4. 使用现有 PostgreSQL/GORM 事务；在同一连接内设置 lock_timeout=5s、statement_timeout=25s，取得数据库内此模块安装的 transaction advisory lock，重新执行准确 owner/core/absence preflight。复用现有严格 schema SQL 创建四表与原有 grants。不存在业务行写入、删除或历史变更。最后核对当前 core 及原 runtime grant 边界允许新增观察能力，其余不变；失败全部 rollback。
5. 使用与 serving 相同的 catalog verifier，针对固定 runtime role 查询权限，不读取它的密码、不赋予 owner/runtime 互相继承。观察表对象归原 owner。事务只增加原四表、schema USAGE及既有 SELECT/INSERT、三张可变表UPDATE；无 DELETE/TRUNCATE/CREATE/GRANT OPTION。未改变 core 的任何行、表定义、默认值、约束、索引、ACL 或 Goose 历史。
6. 配置发布与数据库安装分开。安装命令不启动 worker、不修改配置、不调用 provider。仅在原 runtime-role verify 通过后发布 full-mode 观察配置，保留其余设置及回滚副本；正常启动继续原严格 preflight 与真实 IAM/Store 检查。不用 preparationOnly 绕过已有正式应用或假造可用能力。

## 重复、失败、并发和恢复

幂等身份为已确认数据库的 `store_observations` namespace，载荷为仓库既有四表合同及最小 grants，不接受用户自定义 schema/payload。并发命令由事务锁串行；首个提交后，后者检测 schema 已存在而无变更失败。与其他 DDL 竞争由 PostgreSQL/超时返回失败，不覆盖对象。

连接丢失/超时/取消/进程退出发生在 commit 前，数据库事务回滚；若 commit 回执丢失，状态标为 **UNKNOWN**，不声称未执行、不更换库或重建数据。重进命令先检测 schema：不存在才能重新申请安装；存在则本入口拒绝并提示独立 `--observations-verify` 和原 owner/catalog 检查。只读检查成功表示可部署已有模块；检查失败保留现状，不能自动 grants-only、drop 或迁移。没有新自动 retry/reconciler。

安装失败不发布配置；安装已提交、配置尚未发布时保持观察能力关闭，不删除表。启用后若需回滚，先停止本试用观察 worker，恢复原 disabled 配置，并按既有准备接线文档的 owner 操作撤销**仅新增**观察 runtime grants，以满足原 disabled 严格 preflight；保留四表和观察数据。重新开启须显式原 grants-only/verify，不再次安装。本批不对其他环境执行回滚/重启，也不把镜像回退说成完整数据库回退。

## 权限、Threat Model 与外部副作用

当前 Must：精确目标与 owner；业务数据/原权限不受损；最小 grants 与事务原子性；重复或 UNKNOWN 不破坏已存在观察；实际页面仍按企业/成员/店铺和用途授权。威胁模型是误选目标、错误角色、schema/ACL 漂移、失败/并发安装；不新增任意恶意 DBA 对抗系统或全局运维平台。

安装只产生当前模块数据库 DDL/GRANT，不产生平台请求、商家发货、供应商付款或 ERP 出库。恢复页面流程时继续原操作 ID，物流读取只允许已保存订单与其包裹；合成服务仅响应本次明确原单/包裹，继续禁止向外代理。合成轨迹须标明模拟，上传受理不表示真实揽收；真实平台待联调。

Legacy decision: N/A（本补充仅复用当前 owner，不修改 legacy 路径）。复用当前 owner 的 strict current-schema/catalog preflight、既有观察四表安装与最小 grants、CLI 私密配置检查；原 fresh 路径继续合法独立存在，非空当前库只可选本显式入口。原 legacy sheinsync 的 RETIRE 决定保持，退休 shein_observations 仍拒绝，不做 compatibility/fallback。

## 必要验证与交付

复用现有 PostgreSQL integration fixtures（不用 serving 试用库跑回归），TDD 验证：有合成 Store 行时本模式成功且原业务/合同/ACL/history保持；原 fresh 模式仍拒绝；错误/缺失确认、错误 owner、core 漂移、已有/退休 schema 拒绝且无变更；grant失败回滚；重复/并发不重建、不补权限。验证范围限于本安装事务，不另建验收框架。

独立架构评审达到 IMPLEMENTATION_READY 后才开发。最终独立增量检查、现有必需 CI 与标准构建完成后，仅更新此新试用所需安装工具/接线。运行前对该试用 Store 做私密备份，保留原 manifest/镜像；记录原 Store事实与元数据、原 schema/ACL/history，运行后对比，并用 runtime-role verify 验证。

恢复同一 `TRY-US-UI-20261101-001`：正常页面同步订单 → 详情内查询原包裹合成轨迹 → 回到原 DN 记录1件模拟交接 → 核对99→98现货、reserved=0、DN提交和原回执；不得重复运单上传或更换原单绕过状态。留实际入口、截图、操作步骤及限制；开发自检、独立评审和 CI 不等同用户验收/生产上线。
