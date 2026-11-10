# 四张空 Store 观测表的最小合同对齐

Execution: [#1231](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1231)；解除 #1158 / PR #1227 原 35444 更新的一个启动阻塞。

Design Basis: **Independent Architecture**。Admission: **IMPLEMENTATION_READY / FROZEN**。R1 准确候选与实施义务见 §10；生产实现和副本/原实例操作仍未授权。

实施窗口（2026-11-10 后续用户明确决定）：已授权在原 PR #1232 实现本冻结方案，并在隔离副本验证数据/权限保留、并发重试、新版启动及回退、IT-1 校验接线。该决定替代本文设计阶段“生产实现/副本演练未授权”的阶段限制；原 35444 配置/数据/服务、应用模型/发布/原预留操作仍禁止，合并与原实例更新未授权。实施入口及实际证据写 #1231 / #1232，不把设计记录中的 NOT_RUN 当作新阶段执行结果。

## 1. 用户结果、权限与设计依据

本方案让本地试用维护者在后续获准阶段，通过原 Store schema-owner 的显式维护入口，将四张**仍为空**的观测表对齐到当前程序要求，保留已有 Store、商品、R1、平台草稿、图片、权限及原 AI 点预留。它只解除一个启动合同阻塞，不代表完整平台草稿链路已经开通。

用户 2026-11-10 明确授权本轮：最小设计、独立架构评审、复用 owner 和备份还原入口、明确保留/副本验证/回退。**本轮不写生产代码，不执行副本 DDL 或启动演练，不改原 35444 配置/数据/服务，不调用应用模型，不发布，不释放原 15.445282 点预留。** 文档、Issue/PR 与独立设计评审不作为上述真实操作许可。后续实施、副本演练、合并和原实例更新分别按用户授权执行。

依据按当前 [AGENTS](../../AGENTS.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[owner 边界](project-boundaries.md)、[观测安装](store-observations-installation.md)、[准备模式](store-observations-preparation-runtime.md)、[现有 Store 核心对齐](store-core-schema-alignment.md) 和 [镜像构建](../operations/application-image-build.md)。用户最新决定优先；上述旧文档的 Figma 权威措辞不恢复。无 UI/IA 改动。

全新系统基线明确允许当前新系统自己的正常 schema 演进；这里没有旧业务导入、ID 映射、数据转换、fallback、双读双写或第二事实源。Legacy decision: **EXTRACT 当前合格 owner、事务、定义比较和备份还原能力**。

## 2. 当前证据与真正根因

调查起始 main 为 `55b95607cd93263ee727295e15f0ded7170dc4c4`。2026-11-10 使用原业务库的显式 repeatable-read/read-only 事务、10 秒 statement timeout，仅读取 Store catalog/聚合计数；审计连接不是未来维护身份授权。

- 原 `store_center` 中 `store_observations.commands / syncs / records / heads` 均为 0 行；四表 owner 均为 `store_center_owner`。Store 本身已有记录，不能走全新空 Store 初始化。
- 既有 Goose applied 历史准确为 `0, 2026092801, 2026101101, 2026101601`。观测安装本身当前不登记独立 Goose 版本。
- 原观测 CHECK 为当前运行后台基线 `0f41a88ecb0be733885a926ea22cd76de552df35` 的定义；`kind` 只有 products/orders，payload 分支缺少当前新增订单类型。
- 当前后台仍 healthy；私有 manifest 摘要 `3e9e6897f7fc48ea438d8af9d71862d5b79add74e28144ede44b0cef3679d204`，observations 为 `preparationOnly:true`，officialApplications 数量 0。本轮未读出密码、token 或密钥值。
- 新程序 `buildCurrentStoreObservations → observationstore.NewRepository → VerifyRuntime` 会要求新增 payload 分支；现有安装入口拒绝已存在 schema，因此不能靠换镜像或重跑安装补齐。

本机只读证据：`C:/Users/Henry/.codex/private/empty-observations-contract-design-20261110/readonly-baseline.json`。它是时点证据，不能代替执行事务内重新核对。上轮商品/R1/图片/预留读回及 #1225 副本还原结果只作既有证据，不冒充本候选新演练。

架构缺口是**当前观测合同演进没有对应的显式 owner 维护入口**。现有 serving 的关键字检查也不足以证明完整前/后合同；本次维护必须精确比较完整定义，不能把关键字出现或应用健康当作对齐成功。保持 serving 无自动 DDL，不展开全项目 schema verifier 重构。

## 3. Scope、Must 与 Threat Model

范围仅四张空观测表，实际替换四项 CHECK：

| 对象 | 原基线 | 唯一目标 |
| --- | --- | --- |
| `syncs.syncs_kind_check` | products/orders | 当前 canonical products/orders/supply-orders |
| `heads.heads_kind_check` | 同上 | 同上 |
| `records.records_kind_check` | 同上 | 同上 |
| `records.store_observation_record_variant` | 原商品/SHEIN 订单组合 | 当前 canonical 商品及 SHEIN/Walmart/TikTok/TEMU/供货订单组合、互斥与绑定条件 |

`commands` 不修改，但同样锁住、验空与核对。目标是原 `internal/integration/persistence/storeobservations/schema.sql` 的完整定义，**不是只加几个关键词，也不是放宽成任意 JSON**。

Must：四表全空；准确 owner/库/结构/权限；只改四 CHECK；保留原关系、表 OID、列/默认值、PK/FK、索引、其他 CHECK、RLS 状态、schema/表/列 ACL 和 grant options、角色/成员及原 Goose 历史；单事务全成全败；并发/提交结果不明可用原入口核实；原有业务和跨库事实不变。CHECK 对象自身的 OID 因替换变化是预期差异。

Out of Scope：非空观测数据升级/搬迁、任意中间版本修复、缺表补装、删表/删 schema/CASCADE、重建或重命名、角色/GRANT/REVOKE/凭据变更、自动 Down、其他 Store 业务表变更、通用迁移框架；不启用供货/平台草稿、官方店铺/模型/worker 新能力，不运行发布或预留恢复。

Threat Model：防误库/误身份、runtime 调用维护能力、未知或弱化的 schema、权限漂移、验空后并发写入、双重执行、取消/失败/提交失联导致半套合同。可信 DBA 的同时恶意改角色/系统 catalog 不在本轮模型；维护窗口禁止其他 owner 维护操作，不声称表锁能锁住集群角色或其他数据库。没有新增 Accepted Risk，也不扩大当前租户/外部平台模型。

## 4. Owner 与最小调用链

| 责任 | 复用位置 / 后续最小补齐 |
| --- | --- |
| 显式命令准入 | 原 `cmd/store-center-schema-init` 增加互斥模式 `--align-empty-observations-contract --confirm-database <准确库名>`，沿严格私有 owner JSON 配置；拒绝与所有现有模式混用和额外参数 |
| 维护装配/事务 | 原 `internal/app/schema/storecenter` 增加有界入口，复用 `InstallCurrentStoreObservations` 的现有 PostgreSQL/GORM 事务和 advisory lock 模式 |
| 观测事实与 DDL | `internal/integration/persistence/storeobservations` 唯一拥有当前观测定义；提供 pinned `*sql.Tx`/同事务查询的窄前/后合同校验和四 CHECK 对齐，不让 CLI 导入另一份 schema |
| Store 边界 | 原 `internal/storecenter` 的当前 core/owner/runtime capability catalog 核对按本实例已安装能力复用，不触发 fresh 安装器或借 owner 身份充当 runtime |
| PostgreSQL 定义比较 | 复用 #1225 的 `internal/pkg/pgschema`（基于已有 go-pgquery）；只容许其明确的 PostgreSQL 还原重写等价，缺失、弱化、不同常量/运算符/类型都拒绝 |
| 消费者 | serving 仍只验证原 Store 模块；新 owner 模式不作为 HTTP/BFF/后台自动修复入口，不新增注入配置、表或公共 API |

复用时必须落实的窄接线：现有 `VerifyCurrentObservationsInstallBoundary(installed=true) → verifyStoreMaintenancePermissions → verifyRetainedCapabilityContracts` 最后对 `core_alignment_observations.json` 做 raw `reflect.DeepEqual`，只认 B，且逻辑还原的等价表达式也可能不相等；`installed=false` 又不接受原观测权限。因此新入口不能整条调用该 helper 来验证 A，也不能把参数改成 false 跳过权限。本轮实施应在原 owner 内有界拆出/复用现有 owner/core 与有效权限检查 primitive，再由观测 owner 的完整 A/B + `pgschema` 比较承担本入口的观测 catalog 核对。原安装/核心维护等其他消费者保留其完整边界，不加全局 skip 开关、不全局放宽 raw contract，也不复制第二套权限 allowlist。此项列为 §10 IT-1，实施与合并前必须通过隔离 PG 验证。

不新增 Goose 行或新的版本表。观测表原来由独立 owner 安装、完整物理合同已经足够判定本次两种状态；给 Store core 追加版本会牵连现有多个精确历史消费者，本轮无此必要。原 Goose provider/核心对齐/历史完全保留；采用已有观测安装的 GORM 事务原语，不造日志、任务、恢复 worker 或迁移平台。

前/后合同 A/B 是检查分类，不持久化新状态机。当前唯一 B 来自 canonical schema.sql；实施时四项目标表达式应由该 owner 的共享定义供 fresh DDL/对齐消费，并有现有 PG 测试验证 fresh 与对齐完整定义一致。A 是一个固定已核实前置合同，不是可运行 fallback：准确引用 `0f41a88…:internal/integration/persistence/storeobservations/schema.sql`，LF 文件 SHA256 `0fe49cd71de39f44eccc79757e44ebc8985822fbb761e7de2340aec8eba80437`。B 对应 `55b95607…` 同路径，SHA256 `8f064bb4108274be965792617032855cb7a34d73ca29d3918806492c9829a078`。这些 pin 是开发依据；不从本机 Git/临时文件在运行时加载 DDL。除四项 CHECK 外，两者完整观测合同必须相同。

## 5. 准入、事务与并发

后续执行必须另获准确环境操作授权并先准备备份。CLI 使用原 private owner config：literal loopback、固定 `store_center_owner`、1–2 连接、非系统库、准确 confirm-database；真实 `session_user=current_user` 且为 owner，不接受集群审计管理员、runtime 或 SET ROLE 冒充。owner/runtime 不得有危险集群权限、意外继承/成员关系。原 runtime allow/deny、数据库/schema/列权限及 grant option 仍由当前边界校验，不通过补授权解决。

先可提供只读状态核对，但它不是写事务准入。正式执行使用单个 pinned PostgreSQL **READ COMMITTED** 事务、总 deadline 30 秒、lock_timeout 5 秒、statement_timeout 25 秒：

1. 与原观测安装器使用同一 `current_database()||':store-observations-install'` transaction advisory lock。该锁串行维护命令；业务并不依赖它。
2. 固定顺序取得 `commands → heads → records → syncs` 四表 `ACCESS EXCLUSIVE` 锁；缺表即拒绝，不补表。全部检查均在锁后重新读取，直到 commit 持锁，阻断验空后的 INSERT。READ COMMITTED 保证等锁后的后一个调用看到前一个已提交合同，不沿用锁前快照。
3. 同事务核对准确库/session/owner、四表及 schema 完整 catalog、原 Goose 历史和 Store core、角色与 runtime 有效权限。原实例目前四条 applied 历史；本有界模式不支持缺失、混合或未知前缀，不顺带安装 browser/OAuth 等模块。
4. 四表均无行。检查四 CHECK 必须是完整 A 或完整 B；列、默认值、PK/FK、索引（含有效性）、其余约束（validated/deferrable/inherited 等）、非预期表/触发器/RLS/规则/列 ACL 或对象权限有漂移均拒绝。不能只比较表数/约束名称或对修改前后拍照就认可错误权限。
5. A 时仅对固定四约束按固定顺序 `DROP CONSTRAINT / ADD CONSTRAINT` 为完整 B，立即有效且已验证，不留 NOT VALID。不得 DROP TABLE、UPDATE/DELETE/TRUNCATE、重写 JSON 或更改权限。B 时只核对并返回已完成，不执行 DDL。
6. 同事务重新精确核对完整 B、四表空、保留 catalog/owner/ACL/Goose 历史不变，只有四 CHECK 定义/OID 为允许差异。失败抛错，由原事务回滚全部 DDL。所有 verifier 必须使用同一事务连接，不能经 `db.DB()` 开另一池证明未提交结果。
7. commit 后原入口用只读完整 B 核对报告结果；不输出私有配置。没有应用业务调用或其他库写入。

未知约束、A/B 混合、提前“补过一项”、NOT VALID、额外/缺失对象或权限漂移均保留现状并拒绝，不自动修复。A/B 需要精确枚举完整原合同，不用模糊 LIKE/字符串清理规避 #1225 的严格比较边界。外部依赖不被删除或重建；不在本轮放宽现有 schema 边界。

## 6. 重入、UNKNOWN 与失败责任

| 可见状态/事件 | 结果与唯一恢复入口 |
| --- | --- |
| 完整 A、四表空、其余准入一致 | 单事务执行四 CHECK；无新的业务命令、版本行或第二回执 owner |
| 完整 B、四表空、其余准入一致 | 同入口只读核实已完成，零 DDL/数据/权限/历史写入 |
| 四表任一非空，即使 B | 明确拒绝本“空表维护”模式；不能清空数据或把它当安全重试。可继续既有只读诊断，不宣称本操作成功 |
| 两个维护进程同时进入 | 原 advisory/table locks 串行；第二个锁后看 B 并返回已完成。无重复版本记录可产生 |
| 锁超时/DDL/后置核对失败、取消、进程提交前退出 | PostgreSQL 同事务回滚；原 A、表和权限仍在，不需补偿或后台修复 |
| COMMIT 附近断连/成功回执丢失 | 保留准确库/候选/合同 pin，结果 UNKNOWN；重入原入口：完整 B 核实完成，完整 A 可重试，混合/漂移/非空拒绝并交原 owner，只凭退出码不推断结果 |
| commit 后新版应用其他模块启动失败 | 不擅自反改约束、删 Goose 历史或清库；按下节已验证回退方案处理，停止继续业务开放 |

幂等身份为准确数据库 + 固定 owner + 固定 A→B 合同；持久化事实就是原四项 CHECK。此处 DDL 由一个数据库原子提交，无外部不可重复副作用，不需要新增 UUID 命令账本、outbox、Saga 或 Temporal 恢复协议。

## 7. 数据、权限与备份保留

后续实施/副本演练复用现有 `pg_dump / pg_dumpall --roles-only / pg_restore`、私密数据/schema/ACL 指纹、对象文件 manifest、标准 source image builder。原 `issue1220-retained-drill-20261109` 与 `issue1205-unified-rollout-20261109-7dec3b955/backup-product.py` 是可复用的既有入口/证据，不在本轮新建 runner。**这些脚本有历史 SHA/容器 pin、写入/还原分支，不能原样针对原实例执行**；后续只在准确任务副本参数下复用已有备份还原步骤，不能 import 会执行旧真实操作的脚本。

维护前在另行批准的窗口停止本实例相关写入口/worker并核实没有在途副作用；保留原 Temporal namespace、任务与队列，不新开 worker 消费原队列。普通用户写入不是恶意管理员模型，四表事务锁仍必须落实；不能用“已停服务”代替锁与验空。此轮仅规定后续步骤，不停止原服务。

| 对象 | 保留要求 |
| --- | --- |
| Store | 当前 Store、分组、成员授权、连接/凭据密文、业务回执、原 Goose 行、数据库/schema/default/表/列 ACL、角色成员关系；四表只变四 CHECK |
| Product | 原测试商品 `own-27ef901c-30e0-57f9-81d9-56032b0efa61`、collection item `99b8f948…`、Source `41cb802a…`、R1 `f7853e11…`、0/空/空、全部历史与原键不变；原实例目前没有平台 Target 表，不能导入隔离 Target 伪装保留原草稿 |
| 平台草稿 | 其他隔离环境已有 Target/记录/卷保持不动；本维修不安装 listing 执行表、不转入/创建/保存草稿。若执行前原实例后来出现真实 Target，则纳入备份/前后只读摘要，不删除、不重建 |
| 图片 / Asset | 原公共主图与地址、源媒体对象卷及秘密卷、image/Asset 库、候选/批准/历史、容量账本；不移动/重传/自动批准图片，不借隔离白 M 合成图替换原图 |
| 资金/AI | commercial 原 reservation `0d63cf7a-2874-4229-a415-ff812ce1cd4d` 仍应 reserved 15.445282 点；保留 ProductAgent 原调用/凭据与其他库事实，既不零结算也不释放、不重试模型 |
| 运行 | 保留实际 Compose 层、当前后端 manual-only 配置覆盖、secret/CA/登录挂载、网络/端口和现有镜像；容器旧 revision 标签不当成真实构建证据 |

实际维护前更新私密备份和基线，不复用旧快照覆盖新业务。至少包含 Store 完整库/权限/角色；涉及完整实例启动/回退演练时复用既有全依赖库备份和原对象快照。备份可读取原库，但恢复和 DDL 只能落到准确任务副本；运行副本不挂原可写卷，不接原业务 Temporal，不调用模型或官方平台。

## 8. 后续有界副本验证（本轮 NOT_RUN）

用户后续授权实现/副本演练后，先 TDD 捕获 A 在新版本下拒绝且新维护入口缺失，再做最小实现；复用现有 Store PostgreSQL/CLI 测试，不建立新验收系统。验证只覆盖本次 Must：

1. 在隔离数据库还原本次准确备份及权限，核对完整 schema、数据/历史、角色/ACL、商品/R1/图片/预留与原快照相等；复用 #1225 比较器允许的 restore 重写，不把损坏/放宽当等价。
2. 执行 A→B，四 CHECK 完整等于 canonical；所有其他表/数据/owner/ACL/Goose 行/OID 与允许差异一致。B 再入无写；两个命令并发只生效一次。验证未知/混合/缺 CHECK/弱化/未验证/权限漂移/误库误身份/任一表非空均拒绝。
3. 一条持锁写事务与维护竞争：写先提交则维护看到非空并拒绝；维护先锁则写不能穿过验空/DDL间隙。事务中定向失败/取消回滚，成功后丢回执按原命令核实；复用现有测试原语，不开发网络故障平台。
4. 用实际目标程序、原 preparation-only 配置副本和隔离依赖启动，走本机健康/未登录拒绝及允许的只读页；不启用供货、官方应用、平台保存/发布或模型。保存日志/准确镜像 receipt 和只读 preservation 摘要。实现自检与用户产品验收区分。
5. 按下节还原到演练前备份，启动准确回退候选并复核保留；失败与未跑分别记 FAIL/NOT_RUN，不以原 #1225 或新 CI 成功替代本候选直接执行。

实现按当前 Delivery Batch 一个主要 PR，最终集中独立复核；本次独立架构评审不自动要求重复全局审核。对应真实事务、锁与权限验证是 IMPLEMENTATION_TEST，实施/合并前收敛；不将理论 hardening 自动升级为新平台需求。

## 9. 回退方式与停止点

本入口**不提供反向 DDL/自动 Down，不删除任何版本或业务数据**。事务失败由 PostgreSQL 原子回滚。已提交 B 后，优先保留现状诊断；不得把旧前后端镜像直接接到新草稿数据上继续编辑。

后续副本演练在同一无业务写维护窗口内：保存 A 的完整 Store 备份/原权限/原运行配置 → 对齐 B/新候选启动 → 停止任务副本 → 用现有 `pg_restore` 将**该任务 Store 副本**恢复到同一 A 快照和权限 → 启动准确回退后台/UI副本 → 核对原数据、权限、R1/图/预留未变。其他库无变化则不还原它们，避免无关覆盖；如后续还有另行获准的 Product 安装，必须另按其 owner 的备份/回退计划处理，不能称本事务覆盖跨库升级。

原后台实际业务基线 `0f41a88…`，原 UI `7dec3b955…`。#1225 旧演练回退候选 `d840121…` 的业务基线是 7dec，**不能冒充当前 0f41 的回退证据**。逻辑还原可能触发未经修复的旧 schema 文本比较误判；后续回退候选需保留 0f41 当前业务语义及已批准 #1225 schema 规范化修复，标准 Dockerfile/source builder 生成准确 receipt，并在本副本真实启动验证；不能临时放宽 guard 或改标签伪造来源。

未来原实例若获准执行，仍须刷新精确备份、明确暂停写入和准确候选后再做。若维护后任何库出现新业务写入、Store schema/ACL 或平台任务变化，停止整库还原建议，报告具体新增事实，由原 owner形成不丢数据的处理；不得以回退为由覆盖新商品/草稿、退还额度或重放发布。

交付停止点是**方案评审完成、实施准入是否达到及明确 IT**。原实例启动、真实用户登录/使用、实际回退、新平台草稿能力和产品验收均未执行；任何一项不因设计 READY 而变成 PASS 或获得授权。

## 10. 独立评审记录

独立 Architecture Review R1 已完成关键路径核对，准确候选 **`139a9ff01275f0e6f60708e194fbc79560be264e`**，结论 **IMPLEMENTATION_READY，0 BLOCKER**。Reviewer 实际核对真实源码、完整两文件文档差异、A/B schema 源 hash/四 CHECK 差异、原 owner/GORM/advisory key、pinned queryer、runtime/preparation 装配及既有还原证据；未改生产代码或操作原实例。本段及 §4 文字只记录该结论和明确其接线义务，不改变冻结的权限、事务、状态或副作用边界；最终文字增量核对另记 PR/Issue。

| Finding | 当前 Must / 后果 | Classification / 层级 | Action |
| --- | --- | --- | --- |
| IT-1 不能整条复用仅认 B 的 raw retained-capability helper | 安全准入完整 A/B、还原等价、有效权限；直接套旧链会让核心路径无法执行，不能靠跳过权限解决 | IMPLEMENTATION_TEST，实施/合并前 | §4 明确窄 primitive 复用；A/B 完整合同交观测 owner + pgschema，原其他消费者边界不变；实施时验证 A 接线、还原后 A、B 只读重入和权限漂移拒绝 |

R1 报告位于 `C:/Users/Henry/.codex/private/empty-observations-contract-design-20261110/architecture-review-r1.md`。§8 的事务、锁、并发、恢复、精确保留和副本回退仍是后续必须实际完成的实现验证，当前全部 **NOT_RUN**。架构 READY 不等于原实例 EXECUTION_READY、实现完成或用户产品验收。
