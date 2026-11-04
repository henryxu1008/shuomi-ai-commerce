# 另名新空 Run 的 ledger 安装入口：最小设计候选

状态：**DESIGN_CANDIDATE / ENTRY_GAP_CONFIRMED / NOT_IMPLEMENTED / NOT_INSTALL_READY**。
执行载体仍为 Issue #1073 / PR #1075，原工作区、原唯一负责人；本轮只整理设计，不新增派工、实现 PR 或评审。
用户可审查的结果：明确新版关闭 AI 时所需的新空 Run 如何准备、缺少哪一处安装入口、哪些身份可执行，以及将来每一步需要的单独授权。
本候选不授权代码开发、工具构建、读取凭据、建库、安装、配置发布、实例更新或真实调用，不合并关单。

## 依据与已确认结论

原公共 owner 在[原聊天](codex://threads/01a17b25-eb0a-7123-bf5b-ea5c4b914071)的回合 `01a1a6aa-ecad-7bb2-a0b4-324f87ba1682` 明确回复 **ACK1075-INSTALL**，只读核对了准确应用来源。
确认现成 ledger CLI 只接受 `product_agent / product_agent_owner`；另名库和管理员连接均不能绕过固定检查。
Run prepare 先验证已有 ledger，再创建 Run 四表组及通道投影，不负责创建 ledger。当前为 **52 列**，以唯一 schema owner 为准。
因此，原草案中的另名库不能直接安装；本轮定稿的是可审查的最小设计候选，完整可执行安装清单尚未具备。

本轮文档工作的 Design Basis: **N/A**。拟议实现按 **Reuse Existing Architecture** 整理：引用[共享通道新空 Run 合同](../architecture/shared-channel-credentials.md#11-新版本的受限维护入口)、[共享模型配置](../architecture/enterprise-shared-model-configuration.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
当前 owner 仍为 `cmd/account-audit-ledger-schema-init → internal/aicapability/store`；不改变领域 owner、52 列 schema、计费/预算、授权或既有恢复合同。参数和版本策略待用户确认；实施仍未授权，不标为 IMPLEMENTATION_READY。
无新 UI/IA，沿原研究/选品 Product Authority；不增加凭据、计费或安装平台。

## 应用、工具及制品来源分开固定

| 对象 | 来源与交付策略 | 本轮状态 |
| --- | --- | --- |
| 应用后端与 UI | **f337521d0c69f8342c9335ba31e8b86ea3236fa4**，tree **01ca1f2cd3102563bdafaca8551fe15709e6068b**；应用构建沿[标准入口](../operations/application-image-build.md) | 不构建、不更新；文档 HEAD/main 不替代应用来源 |
| Run prepare | 同一 f337 标准后端制品中的 `/app/product-agent-runtime-prepare`；源码来自 Dockerfile.product-listing-api | 准确 f337 应用制品尚未构建，不能借旧镜像冒称已具备 |
| ledger / 配置安装工具 | 现有 `deployments/docker/Dockerfile.account-compose` 的 **schema-init** target；ledger 位于 `/usr/local/bin/account-audit-ledger-schema-init` | 原 f337 不含拟议新参数；新工具准确 SourceRef、tree、image ID/digest 均待将来批准和实际交付 |
| 拟议工具补丁基线 | 优先以 f337 为基线，仅修改既有 ledger CLI 及对应必要测试/说明；`internal/aicapability/store`、52 列定义、依赖及其它 installer 保持该基线 | 本轮只提出版本策略，不创建分支、代码、二进制或镜像 |

新工具不得自动跟随最新 main/#1093，避免工具使用另一版 schema 而应用仍为 f337。若未来不得不采用其它基线，先说明真实差异并取得决定。
schema-init 镜像的默认 ENTRYPOINT 是整套 `account-schema-init`，未来必须显式选择具体工具，不能默认启动整套 bootstrap。
若选现有容器制品路线，未来先另行批准工具制品及 f337 应用制品构建，再准备新库和执行各入口；构建不启动应用，不自动获得安装/更新权限。旧 1e 镜像不作为 f337 工具证据。

## 新空 Run 资源与角色

| 资源 | 本候选 |
| --- | --- |
| 项目 / 数据库服务 | 原 task-processor-unified-20261010 / business-db；不新增试用实例、容器服务或开放端口 |
| 新目标 | 拟 `product_agent_run_20261104`，host/port 沿原目标；新库 owner 为 product_agent_owner。该名只在此前原 reader 的只读快照中确认未存在，没有预留或创建 |
| 存储 | 沿原 business-db 持久卷，未来新库会新增文件；不删除、重建或克隆原库/卷。新库从空模板准备，不从 product_agent 复制 schema/数据 |
| 保留 | 原 a66、product_agent、镜像、manifest、其余数据库、18 个原容器及 55 个项目卷；本轮不操作数据库或容器 |

| 操作/角色 | 合法合同与边界 |
| --- | --- |
| 新库、角色及基础授权 | 获准的 business_cluster_admin 或具备所需权限的受委托安装身份；仅新目标操作，不把管理员放入应用配置 |
| ledger | 新模式仍要求直接以 product_agent_owner 执行，且数据库 owner 同名；不能用管理员连接代替准确身份检查 |
| Run / 通道准备 | product_agent_owner，或有合法 SET LOCAL ROLE 能力的获准管理员；沿原 preparer，不修改 owner 属性/成员关系 |
| Agent 配置 / Skill | 原 bootstrap 使用 business_cluster_admin，或合格委托安装者；必须具备必要 CREATE ROLE/SCHEMA/SET ROLE 能力。schema 分属 NOLOGIN 的 agent_configuration_owner / skill_capability_owner |
| product_agent_runtime | 原 serving 身份及池 4，按现有 installer 的精确 grants；不授 DDL/DELETE/owner/grant option |
| model_route_config_writer | 原 writer、池 1、全局连接上限 4；只维护路由配置并读去密投影，不读 Key 或维护通道 |
| title_credential_writer | 只维护新连接及其回执，不写 Run/Tool/Invocation；本轮不执行维护命令 |
| ai_channel_secret_reader | 标准 Run 准备的角色前置，目标 LOGIN、连接上限 2、无成员关系/DDL/写入，只读连接密文；AI 关闭时不在 manifest 声明或打开该池 |

已有只读角色存在性/权限证据仅保留其快照范围，不能外推为新库权限或安装身份认证。此前快照中 secret reader 缺失。
**实际安装操作者、使用安装凭据的授权、现场安装权限和新角色认证材料仍未落实。** 本轮不读取或验证任何安装凭据，不提权，不改变原 LOGIN owner、连接上限、密码或成员关系。
新库应先限定 PUBLIC 的 CONNECT/TEMP 和 public CREATE，再给所需身份精确 CONNECT/USAGE；仅作用于新库。角色的只读边界来自 grants，不通过扩大权限或修改原角色默认属性实现。

这里的“资源”是新 Run 的数据库/角色/制品。原商业资源库继续沿 commercial_schema_owner / commercial_owner_runtime，**本候选不新建或改造 Money/商业/资源 schema**。
`commercial-owner-schema-migrate` 默认还安装初始报价，不能拿来代替新 Run 的窄安装步骤，也不作为本候选前置。

## 既有 ledger CLI 的最小新增模式（未实现）

候选只新增一个显式参数 **`-confirm-empty-database <准确库名>`**，仅支持 `-namespace product` 的另名新空目标；不增加任意 SQL、schema 或 owner 参数。
未给此参数时，现有 image/product 固定入口保持原行为；该候选不提供历史数据迁移或兼容模式。

1. 库名必须非空、无首尾空白、符合当前受限标识符范围（小写字母开头，字母/数字/下划线，最长 63 字符），且不能为原 `product_agent`。
2. 私密 DSN 仍沿原输入方式；实际 current_database 必须与显式确认值逐字一致，current_user 和数据库 owner 均须为 product_agent_owner，并沿原 Run 合同校验该 owner 没有 superuser/CREATEDB/CREATEROLE/replication/BYPASSRLS 超额能力。不匹配时在 DDL 前拒绝，不提升角色属性。
3. 使用 public 的固定目标，在同一有界事务中核空库：非系统 schema 中没有业务表、分区表、视图、物化视图、外部表或序列；其它业务 schema、既有 ledger 或旧格式对象均拒绝。
4. 只在准入后复用原 AutoMigrateInvocationLedger；随后沿当前 owner 核对准确 52 列/归属再提交。不复制列定义或建立第二模型，不在 serving 启动时执行 DDL。
5. 新空模式重复执行遇到已有对象同样拒绝，不自动降级为旧入口、不修补已有 schema；合法安装方按顺序执行，不并行跑同一目标的安装动作。
6. 事务失败沿原 PostgreSQL 回滚；退出码/错误不泄露 DSN、密码或 Key，不新增回执表/迁移状态/恢复服务。响应或提交结果不明时停下，只读核目标对象、准确合同和 owner；不盲重跑、不自动 DROP/清库。

下面仅是**待用户确认、尚未实现的命令形状，不可执行**：

```text
account-audit-ledger-schema-init -namespace product -dsn-file <受限的新库-owner-DSN文件> -confirm-empty-database product_agent_run_20261104
```

## 未来安装顺序与配置差异

所有步骤均 **NOT_RUN**；工具/应用制品须先有准确来源和另行构建授权，现场操作须先有实际安装身份与授权。

| 顺序 | 入口 | 身份 / 停止点 |
| --- | --- | --- |
| 0 | PostgreSQL 现有资源管理命令，准备唯一新目标、缺失角色及基础权限 | 获准集群安装方；不得重跑 business-db-init、native-modules-init 或 account-schema-init 整套脚本 |
| 1 | 上述拟议 ledger 新空模式 | 新工具准确 Ref/制品尚未存在；本步未实现或未授权时，后续安装不得开始 |
| 2 | `/app/product-agent-runtime-prepare --target run` | f337 制品，PRODUCT_AGENT_SCHEMA_DSN 私密注入到准确新库；先验 ledger，再建原四表组、去密投影及窄 grants |
| 3 | `agent-configuration-schema-init --runtime-role product_agent_runtime` | 准确工具制品，AGENT_CONFIGURATION_SCHEMA_DSN 指新库；按原 owner 完整安装配置与 Skill，不导入/启用企业业务记录 |
| 4 | `agent-configuration-schema-init --runtime-role product_agent_runtime --model-route-pricing-only --model-route-writer-role model_route_config_writer` | 同一新目标；不传 writer-connections，保留原全局上限。Run prepare 在路由授权之前，避免先扩大其角色权限闭包 |
| 5 | 原 reader/writer 只读核新目标元数据；另获授权后准备私密候选并按 f337 严格解析 | 解析不等于启动。未满足新 schema、凭据保护或准确制品绑定则停止，不拿原 1e 证据代替 |

配置的完整字段差异仍见[配置审查页](selection-shared-model-wiring-preview.md#相对未发布1e配置基线的具体差异)：

- ProductAgent 与 modelRoutePricing.writer 的 database 同指拟议新库，其 host/port/user/password/pool 保持；其余数据库不变。
- dataServices.executionEnabled 显式 false；ProductAgent、完整图片生成、Chat 继续关闭，不声明/解密通道 Key。
- LAC 三用途拟显式共享 connectionId=lac-bailian-cn-beijing、connectionProvider=bailian；清空退休 org-credential 准入，保持未配置，不伪造当前版本。
- 不发布参考价、企业三类模型选择、启用或执行模式；空 Run 不带旧启用、模板、选择或历史，未来沿原管理员入口显式配置。
- 预算研究/选品各 ¥0.30/20 点、合计 ¥0.60/40 点；原内容 ¥0.30/72000 Token。参考估算和原共享 Limits 不变，不研究精确厂商价。

## 最小未来验证、授权与停止条件

本轮仅文档差异与本地引用检查，**不编译工具、不重跑业务/隔离测试**。未来补丁获准后，只计划复用既有 Go 测试和任务隔离 PostgreSQL：

| 目标 | 必要计划 |
| --- | --- |
| 准确目标准入 | 错/缺库名、非 product 模式、原库名、DSN 与确认名不符、错执行身份/数据库 owner 均在 DDL 前失败 |
| 空库与旧格式拒绝 | 非空、既有 ledger、旧 schema 拒绝且对象/ACL 不变；新模式不执行既有表升级 |
| 合法新空成功 | 复用原模型产生准确 52 列与固定 owner；仅目标 ledger，不产生调用/计费行；原库/角色属性/ACL 不变 |
| 失败与结果不明 | 事务失败不留下部分安装；结果不明只读核对并保留不确定性，不盲重试或删除 |
| 默认入口 | 原 image/product 固定模式的未变化合同继续通过原必要回归；不以新模式覆盖或放宽 |

当前真正待用户确认的是：此受限参数方案，以及“应用固定 f337、安装工具另有准确版本”的策略。
设计认可不包含后续授权：**入口补丁开发/自检/交付 → 准确制品构建 → 实际资源/安装及凭据使用 → 私密配置准备/解析 → 两服务更新与后台启动范围 → 真实调用**，各阶段按明确范围分别决定。
工具准确 Ref/制品、实际安装操作者/权限/认证材料仍待落实；不把角色合同确认写成现场可安装。
本候选不改变 #47 UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT，也不把其它 PR/CI 或开发自检当成本轮现场或产品验收。
