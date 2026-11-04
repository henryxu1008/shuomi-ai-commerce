# f337：新通道合同的运行配置与数据库只读核查

2026-11-04；执行 Issue #1073，原工作区与 PR #1075。
应用 SourceRef 固定 **f337521d0c69f8342c9335ba31e8b86ea3236fa4**，tree **01ca1f2cd3102563bdafaca8551fe15709e6068b**；后续文档提交不改变该应用来源。

**结论：已确认当前保留库不满足 #1089 的新通道元数据合同，不能直接用于新版启动。**
这是实际结构缺项，不是模型参考价问题，也不是需要把原读取身份提升成管理员。
新程序的实际 LoadConfig、Run、镜像构建及实例更新仍为 **NOT_RUN**；启动拒绝结论来自固定源码与实际系统目录的对照，没有启动程序试错。

Design Basis: **N/A**。本次仅原身份只读调查、去密差异和方案整理；不改变产品、权限、状态、持久化或副作用语义。
依据为已冻结的[共享通道凭据合同](../architecture/shared-channel-credentials.md)、[共享模型配置](../architecture/enterprise-shared-model-configuration.md)、[数据服务执行生命周期](../architecture/data-services-execution-lifecycle.md)和[全新系统基线](../product/greenfield-no-legacy-migration.md)。
不改凭据/数据库，不发布配置，不构建或更新实例，不执行真实采集、AI 或付费调用，不合并关单。

## 本次实际检查了什么

仅复用未发布私密候选中的两个原身份：`product_agent_runtime` 和 `model_route_config_writer`，分别连接原 `product_agent` 库。
两次连接均确认 `current_user=session_user`，端点为原 `127.0.0.1:5433`，数据库 OID 为 16413。
使用现有 business-db 容器内的 psql 客户端，容器 OS 用户 postgres；数据库身份始终是上述原角色，未用数据库管理员、owner、SET ROLE 或新身份。

连接默认只读并执行 `BEGIN READ ONLY`，语句/锁超时为 5 秒/1 秒，结束执行 `ROLLBACK`。
查询只涉及系统目录、对象列/约束、角色及有效权限；未读取业务行、Key、密文、配置 current/revision 内容或用量。
数据库密码仅在原进程内用于认证并随即恢复环境，不进入参数、日志或新文件；没有创建辅助容器、临时库或验证程序。

本地忽略证据目录：`.local/channel-contract-readonly-f337-20261104-01/`。
`catalog.sql` 为固定目录查询；`run-reader.jsonl`、`writer.jsonl` 各 61 条元数据，退出码均 0，stderr 均为空。
`configuration.nonsecret.json` 仅为白名单字段投影；before/after/preservation/https 文件记录保留检查，不含私密正文。

## 实际结构与权限差异

| 对象 | 实际观察 | 固定合同要求与影响 |
| --- | --- | --- |
| `ai_channel_connections` | 不存在 | 新连接当前事实表；必须为 product_agent_owner 拥有的永久普通表 |
| `ai_channel_connection_receipts` | 不存在 | 新连接命令回执表；标准新空 Run 安装范围之一 |
| `ai_channel_credential_bindings` | 不存在 | 配置管理只读去密视图；必须为准确投影且与连接表同一 owner |
| `ai_client_credentials` | 仍存在，owner=product_agent_owner | 新投影校验明确要求旧表不存在；不能增加一个同名新视图就绕过 |
| `ai_client_credential_bindings` | 旧视图仍存在，owner=business_cluster_admin | 不能作为新视图的替代，也不转换旧凭据版本 |
| `ai_route_config_revisions/current` | 均存在，owner=business_cluster_admin | 本次检查的列和四项主键/唯一键/外键满足相应声明；不能据此推定新通道投影合格 |
| `ai_route_config_options` | 不存在 | 标准路由安装范围缺项；当前 prepareSharedModelServices 传 false，不单独作为本关闭态管理路径的启动阻碍；未来图片 options 路径另核 |
| `ai_invocations` | 表存在，owner=product_agent_owner | 本次仅确认存在/归属，没有复核完整列、业务内容或用量；不把既往差异写成本轮新实测 |

固定调用路径为 `prepareSharedModelServices → VerifyReader/VerifyWriter(false) → VerifySchema → VerifyMetadataProjection`。
最后一步即使 AI 全部关闭也检查新视图、连接表、owner、准确视图定义及旧表不存在。
实测 `viewExists=false / tableExists=false / retiredTableAbsent=false`，已经足以证明当前库不满足这项前置。
应用启动只验证，不安装、修复或转换；本轮也未调用这些应用验证函数连接真实库。

| 身份 | 实际观察 | 结论 |
| --- | --- | --- |
| product_agent_runtime | LOGIN；无超级/建库/建角色/复制/BYPASSRLS、成员关系、public CREATE 或本库 CREATE/TEMP；路由两表只有 SELECT，没有写入权限 | 与本次已有路由读取边界一致；候选池 4 保持，未改身份 |
| model_route_config_writer | 同样无上述高权限/成员关系/DDL；连接上限 4，候选池 1；revisions 为 SELECT/INSERT，current 为 SELECT/INSERT/UPDATE，均无 DELETE/TRUNCATE | 原 writer 的已有路由权限不是阻碍；新通道对象尚不存在，其去密读取及禁读密文的最终准入尚不能确认 |
| title_credential_writer | 角色存在、LOGIN、无上述高权限和成员关系；未用其登录或执行 | 仍是未来通道受限维护身份，不能用配置 writer 替代 |
| ai_channel_secret_reader | 角色不存在 | 全套标准 Run preparer 要求预备该角色；但 AI 全关的运行装配不打开该池，此项不等于关闭态必须配置密码/保护材料 |

缺失对象的权限记录是 NULL，不能解释成“已确认缺少某条 GRANT”或“权限检查已通过”。
没有用 owner 登录，也没有修改角色、连接上限、成员关系或 ACL。

## 可审查的配置差异：仅方案，未写入私密文件

现有未发布候选仍为 `current-application-1e3509-a.private.json`，SHA256 **194B770F10B165768959CFC9B678D99F8F1BFECF4FA65387B87C02CFFF0786EC**。
它只有绑定旧 1e 版本的严格解析证据；本次只读投影不构成 f337 严格解析 PASS。
下面所有目标值都是拟议差异，不是已发布的默认选择或已创建的连接。

| 路径/范围 | 实际候选 | 最小目标及生效阶段 |
| --- | --- | --- |
| `dataServices.executionEnabled` | 未声明，f337 的有效默认值为 false | 显式 false，便于审查；不改变当前有效关闭语义 |
| productAgent / image generation / Chat | false / false / false；contentOnly、contentTitleOnly 均 false | 保持；不打开执行器、provider 或 secret reader |
| `productAgent.database` 与 `modelRoutePricing.writer` 的数据库目标 | 同一原 product_agent，身份分别为 product_agent_runtime/model_route_config_writer，池为 4/1 | 未来经批准改为同一个新空 Run DB；host/port、原身份和凭据保持，目标数据库名需实际安装方案给出，不伪造为已存在 |
| LAC 三条文字 binding 的 `connectionId` | 均未声明 | 拟统一 `lac-bailian-cn-beijing`；只在本组织三用途间共享连接，未创建 |
| 同三条的 `connectionProvider` | 均未声明 | 拟 `bailian`；以后须与维护入口提交并返回的连接 provider 完全一致；不是把模型 ProviderID DASHSCOPE 改名 |
| 三类 RoutePolicy 的 `AdmittedCredentialVersion` | 仍为已退休 org-credential:v1 引用 | 新候选清空旧准入引用，保持未配置；不机械改前缀、不伪造 channel-credential-v1 版本。实际新版本由公共 owner 根据连接元数据准入 |
| `text.profile.ConnectionID/CredentialVersion` | 无连接 ID，旧版本声明 | 不再复制一套连接版本事实；Catalog.Qualify 根据已声明连接及其当前元数据填充合格 profile |
| endpoint 准入、模型、channel/region | deepseek-v4.1-flash / DASHSCOPE / bailian-beijing / cn-beijing / standard-sync | 保留现有准确 endpoint 约束、用途 prompt/schema、模型准入；不能从连接有效推定任意型号可用 |
| 参考价与三类默认选择 | 原参考版本 lac-deepseek-text-reference-v1、price lac-deepseek-reference-text-v1；未发布 | 保留参考方案；不因本次查询写入平台价格、企业选择或执行模式 |
| `channelCredentials` | 未声明 | 本关闭态继续不声明；真实执行阶段再单独准备受限 reader 和保护材料，本轮不生成/复制 Key 或密码 |

LAC 组织为 **394513833252618242**；三用途为 product.content.decision、product.research.decision、product.selection.decision。
连接 ID/provider 只是合法候选值，没有相应连接事实，当前不能宣称模型 AVAILABLE。

预算不变：研究/选品各 **¥0.30、20 点**，各一次合计 **¥0.60、40 点**；原内容仍为 **¥0.30 / 72000 Token**。
共享 Limits 为 CNY、12 步、2 调用、72000 Token、300000 微元、120 秒；参考估算仍输入 2 元/百万、输出 8 元/百万。
本次投影确认研究/选品每调用 maximumMicropoints=10000000；原内容该字段为默认 0（未声明独立每调用点数上限），不表示免费。
不把此前文档中的“每条文字声明均 10000000”当成已批准或实际值，也不借此给内容增加预算决定。

## 最小处理方案与尚未授权的操作

1. **原 a66、原库、配置、数据和卷全部保留。** 不在原库添加替代视图、删旧表、转换旧 Key 或修改旧 schema。
   这是保留运行实例，不是让新版兼容旧库。旧格式历史留在原版本；新空 Run 不承诺显示或续跑旧 Run/凭据历史。
2. **新空 Run 资源方案已收敛为候选。** 在现有 business-db 服务内拟 product_agent_run_20261104，owner 为 product_agent_owner，不要求另建试用实例；未创建、预留或授权实际操作。
   原 reader/writer 已可认证且边界受限，优先复用合格原身份/凭据；缺少的新 secret-reader 角色由具备合法权限的安装方另行准备，不能由本会话自行提权。
3. **复用当前 installer，先补齐正式入口缺口。** 原公共owner已明确ACK1075-INSTALL，确认现成f337 ledger CLI固定库名、不支持另名新库；[最小入口设计](new-run-ledger-installer-design.md)随后获用户认可，原 PR 已完成最小离线补丁/合成自检，未构建或安装，不临时建表。未来获准并安装合格52列ai_invocations后，再沿原product-agent-runtime-prepare的run目标准备四表组及投影，以及原配置/Skill和路由入口。
   preparer 不创建角色或密码，也不创建/修补 ai_invocations；仅建一个空库再直接执行 preparer 并不足够。其它新 Run 消费者需要的当前 schema/权限沿原 owner 安装，具体清单须安装前列清，本轮未盘点为完整安装 PASS。
   role、对象 owner、四表组完整性、准确投影、route reader/writer 与 secret-reader 隔离逐项按原合同验收；不改公共合同或新增迁移/兼容设施。
4. **资源方案落实后，再单独准备私密候选并按 f337 严格解析。** 仅改变获批数据库目标及上述新通道字段，继续关闭采集和全部 AI。严格解析不等于实际 Run，也不授权创建连接、发布价格/模型选择、构建或更新两服务。
5. **真实 AI 仍是后续阶段。** 连接维护、保护材料、三类选择/参考价发布、完整执行计量及预算准入分别沿现有 owner；当前不导入原凭据、不做平台或模型试连。其它保留 worker 的启动影响仍见[两服务方案](market-research-unified-update-plan.md#其他后台任务保留范围)。

直接相关的运行说明曾仍写旧三表和 model-bound 凭据字段，已同步纠正到当前合同；不沿过时说明对保留库执行命令。

## 原实例保留与本次停止点

前后均为 **18 个原容器、55 个项目卷**，容器 ID、image、来源、启动时间、user/network，以及 environment/command/mounts/HostConfig 哈希零差异。
原 manifest、未发布候选及两服务 Compose 的内容哈希、长度、Windows ACL 均未变化；没有重新挂载或修改 Linux 模式，原 0600 证据保留其此前验证范围。
后端/UI 仍为 **a66d7d0c**，两服务、zitadel-api 和 zitadel-login 均 healthy。
系统 TLS 信任下 HTTPS 首页 200，未登录访问研究和选品入口均 307；不跟随登录、不登录或点击业务操作，用户登录后业务验收 **NOT_RUN**。

本轮交付为 **READONLY_AUDIT_COMPLETE / RUNTIME_SCHEMA_MISMATCH_CONFIRMED / NOT_DEPLOYMENT_READY**。
没有修改数据库、凭据、私密配置或运行服务；f337 实际严格解析/启动、新镜像构建、配置发布、更新和真实调用均 **NOT_RUN**。
后续已取得原owner明确回执，消息处理不再是阻碍。当前缺项是现有ledger CLI的新空目标模式、独立工具准确来源，以及实际安装操作者/授权；详见[design-only候选](new-run-ledger-installer-design.md)。本轮仅整理方案，不读取安装凭据、不连接或操作数据库/容器、不实现/构建/安装，不把原快照改写为本轮执行PASS。
