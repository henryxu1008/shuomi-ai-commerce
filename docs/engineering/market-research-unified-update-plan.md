# 市场调研与智能选品：关闭执行候选的两服务更新方案

状态：**OFFLINE_IMPLEMENTED / READONLY_AUDIT_COMPLETE / RUNTIME_SCHEMA_MISMATCH_CONFIRMED / NOT_BUILT / NOT_PUBLISHED / NOT_DEPLOYMENT_READY**。
本轮为安装方案的design-only整理，应用固定f337521d0c69f8342c9335ba31e8b86ea3236fa4；原a66保持。
具体配置/结构差异及最小新空Run方案见[核查结果](channel-contract-f337-readonly-review.md)。不能使用任意分支HEAD/main、后续文档提交或旧1e镜像代替应用来源。
本轮不构建镜像、不改凭据、不发布配置、不更新实例、不真实采集/AI/付费、不合并关单。
预算仍研究/选品各¥0.30/20点、合计¥0.60/40点，原内容候选A为¥0.30/72000 Token。

本轮只读调查的Design Basis: **N/A**，不改正式业务代码或运行语义。此前离线实现依据为 **Independent Architecture / IMPLEMENTATION_READY（仅离线实施）**，见
[数据服务执行生命周期](../architecture/data-services-execution-lifecycle.md)。此前应用离线实施实际main一次组合d80d7b1e0，
复用#1089的共享通道凭据，保留原研究/选品、完整图片关闭态及供应链/图片修复；不重建公共模型、费用或凭据能力。

## 原 owner 已回复与当前安装阻碍

已在原聊天取得 **ACK1075-INSTALL**。公共 owner 确认现成 f337 ledger CLI 只接受 product_agent/product_agent_owner，**不支持另名新空库**；Run prepare 只验证已有的准确52列ledger，不创建它。
因此本方案的资源、角色、顺序与配置可以审查，安装尚非可执行状态。新的[最小入口设计候选](new-run-ledger-installer-design.md)明确拟议参数、独立工具版本、schema-init制品来源及停止条件；尚未实现或获准构建/安装。

| 本候选范围 | 已收敛结果 |
| --- | --- |
| 新空Run | 同一原business-db内拟product_agent_run_20261104，owner product_agent_owner；不新增试用项目、容器服务、端口或卷；不克隆原库 |
| 角色 | 获准集群安装身份做资源/配置，product_agent_owner做ledger/Run；配置/Skill仍各自NOLOGIN owner；serving/writer不提权。实际操作者/现场权限仍未确认 |
| 工具来源 | ledger及配置工具在Dockerfile.account-compose的schema-init target；Run prepare在f337标准后端。新工具需独立准确SourceRef，不能冒称原f337已有参数 |
| 安装顺序 | 另行获准准备制品和新目标 → ledger → Run/通道 → 配置/Skill → 路由 → 只读核验与另获授权的私密解析；不能跳过缺失ledger入口 |
| 商业资源 | 原商业/资源库及Money/初始报价不在此安装范围，不运行commercial-owner-schema-migrate或整套bootstrap |

本轮不读取安装凭据、不连接或修改数据库、不操作容器，不编译或重跑业务测试；原只读快照保留其时间范围，不冒称本轮现场权限验证。

## 用户现在能核对的结果

- DataServices显式executionEnabled=false或未声明时，资料/配置/任务/结果和研究证据读取保留；不创建/Ready采集provider、不dial其Temporal、不建/启动采集worker。
- 关闭GET使用同owner窄Reader，不恢复ADMITTED/RUNNING；创建/取消在原身份/权限边界内拒绝，不创建任务、扣费或释放原预留。
- 显式启用时，GET仍消费原Service.Read/Results，在原DataKey守卫事务内恢复未完任务；原状态/事务/幂等/UNKNOWN不变。
- 关闭时不登记Amazon费用owner，其他consumer仍沿原工厂和注册表；原reservation和UNKNOWN不自动处理。
- 文字AI、完整图片生成和Chat继续沿已批准关闭分支；页面不冒称能执行。该机制没有更新到原a66。

本轮合成检查已覆盖非空保存结果、Console/DataKey、单连接读取、撤销、关闭命令拒绝及启用两态恢复；
研究→选品比较→人工决定→原准备准确子集交接沿当前共享通道/参考价/账本的合成回归通过。
相关owner/runtime/cmd包检查通过；这是开发自检，不代用户业务验收。最终准确HEAD/CI和独立增量复核记录在PR。

## 已核原a66与回退引用

项目仅task-processor-unified-20261010，服务仅current-application和listingkit-ui。
原应用revision为a66d7d0cde74cded28d77060ac941851cc342a30，tree为705f2aefcd84e144e00a6f86ac777e5fb6267dcd。

| 对象 | 原值及回退目标 | 保留要求 |
| --- | --- | --- |
| 后端镜像 | sha256:2848da261bfc92412fbd2af99719b4963cd6ad45a3bc834a476f844ee2406ffc | 保留可寻址原镜像，不覆盖/删除 |
| UI镜像 | sha256:d106b5dfe863e4222214daa0bee42696f9243a9abf5e92308852aa51c71494b1 | 同上 |
| 原manifest宿主文件 | C:/Users/Henry/.codex/private/market-research-unified-20261102-01/candidate/current-application.private.json | SHA256 1271E0D89BC8E8F8DE5DF92FFCE94D9B9032291C9D7EC04FEA8FCC6AFB195B97，原文件和Windows ACL保留 |
| manifest容器目标 | /private/market-research/current-application.json | readonly bind、create_host_path=false；只一个目标 |
| 两服务实际Compose源 | C:/Users/Henry/.codex/private/image-recent-merged-unified-20261103-01/candidate.compose.private.json | SHA256 CDFB443976983FBA6EC68202C58062220B2EF5DBAB2896E6EA353251142B7187，原文件不覆盖 |
| 身份/网络/其余服务 | 原身份v4.17.1、原保留网络命名空间和其他16个项目容器 | 不重建、重启或更换；执行前核引用漂移 |
| 原挂载与卷 | 后端12项、UI5项；55个项目label卷 | 仅未来获准的后端manifest源可变，其余逐项保持 |

原18个容器状态/来源、配置/卷的保留证据见.local/selection-unified-build-1e3509-20261104-01/；
本轮运行保留及离线检查在.local/data-services-disabled-20261104-01/。不保存/提交完整env、命令正文或秘密值。

## 新候选的具体配置差异与停止点

1. **采集执行**：未来配置明确声明dataServices.executionEnabled=false（新系统默认也是关闭），保留同一DataServices DB/权限、原记录及研究依赖。可保留现有合法执行元数据，但不实例化执行资源；不是删除DataServices或伪造Temporal地址。
2. **AI关闭**：productAgent.enabled=false、完整图片generation.enabled=false、Chat=false及已批准预算保持；沿原Run/Image/Asset/Product配置和历史owner，不装配执行器或通道secret reader。
3. **公共通道合同**：原身份已实测新通道两表和metadata视图缺失、旧ai_client_credentials仍在；这会阻止固定版本的配置管理装配，即使AI关闭。原reader/writer的既有路由表权限符合对应边界，不能用提权解决结构冲突。按新空Run合同准备独立新库和当前schema，保留原库/数据/卷，不补旧兼容；connectionId/connectionProvider等去密差异见核查页，连接未创建，声明仍未准入。
4. **私密候选**：原独立文件current-application-1e3509-a.private.json的hash194B770F10B165768959CFC9B678D99F8F1BFECF4FA65387B87C02CFFF0786EC、Windows ACL和原0600证据保留，未改内容/权限或发布。其严格LoadConfig PASS只绑定1e3509；新代码下的实际私密候选解析尚NOT_RUN。后续只准备/审查新字段差异，不能自行生成含密钥新文件、更新实际绑定或安装新schema。
5. **构建**：新候选镜像NOT_BUILT。未来单独获准后才用scripts/build-listingkit-images.ps1，准确SourceRef、Component All、标准data-services-runtime与UI Dockerfile构建；核commit/tree/imageID/标签、应用层和Config.User，再核最终Compose是否覆盖用户身份。不能沿用旧镜像User核对或给出虚构digest。
6. **真实AI**：当前参考价/企业三类选择/模式及新通道准入未发布，原执行表完整合同本轮未复核；缺失的secret reader角色要由安装方另行准备，但关闭态运行不打开该池。未来执行再准备合法连接/保护材料及执行准入，不因此改原库、改凭据或真实试连。

## 两服务最终允许变更的字段

| 服务/字段 | 未来更新目标 | 当前结论 |
| --- | --- | --- |
| current-application.image | 新固定候选标准构建回执中的完整imageID | NOT_BUILT，不使用旧1e/297镜像 |
| listingkit-ui.image | 与后端同一准确SourceRef的标准UI镜像 | NOT_BUILT，不只替换其中一个来源 |
| 两服务来源标签 | 同一准确SourceRef及tree；其他原标签保留 | 在构建回执与最终Compose结果中比较 |
| 后端唯一manifest源 | 单独获准生成、严格解析及审查的新私密文件 | NOT_PUBLISHED；旧a66文件和1e私密候选均不覆盖 |
| 其余字段 | project、container_name、environment、command/entrypoint、network_mode、healthcheck、restart/pull/logging、shm_size、UI working_dir、其余11项后端挂载/5项UI挂载 | 必须零diff；不新增user/capabilities、不改凭据或重放身份的历史overlay链 |

本轮没有可执行部署overlay或更新命令。未来实际文件须与这份白名单比较最终Compose结果，只点名两服务、no-deps。
如果执行前发现容器/源配置并行变化，先查清，不覆盖他人更新。

## 其他后台任务保留范围

详见[配置页的启动范围](selection-shared-model-wiring-preview.md#其他-worker-与任务队列的实际范围)。
本轮只处理真实采集的装配和间接恢复；没有新增全系统只读/暂停机制。
观察preparationOnly不注入官方应用；Knowledge沿内部对象存储/Tika；Ecoservices非支付分支无支付恢复；
充值两渠道关闭而无provider。其他resource consumer、资源订单和推荐成熟仍可能处理/写原内部事实。
未来更新授权仍须明确包含这些原有启动影响，不能宣称所有后台写入停止；本轮不启动候选来验证或停止原a66。

## 后续获准后的更新与回退顺序

1. fresh核原a66的两镜像/标签、原manifest/Compose hash/ACL、网络、挂载、卷及身份依赖健康。
2. 先确认最小安装入口设计，另行授权现有CLI补丁及必要离线自检；冻结独立工具准确Ref/制品来源，应用仍f337。现成命令不支持另名库时，不先建库或用临时SQL绕过。
3. 构建、资源安装和私密配置分别授权：获准后按准确来源准备工具及f337应用制品，再由实际合法安装方按设计页顺序准备新目标与schema/grants；随后才在配置准备授权内生成/解析新候选并核hash/ACL/模式。任何阶段不自动发布模型选择/价格，不导入旧历史/凭据；文档/main后续推进不自动换应用源。
4. 仅在更新与启动范围获准后，先更新后端并核健康/身份/管理读取，再更新UI；不跑initializer/DDL或整个项目up。
5. 获准的只读检查沿正常身份进入市场调研、选品和配置/历史；未登录入口仍跳登录、BFF拒绝未登录。不点击采集、生成、恢复、确认执行或操作具体商品。
6. 后核只有两服务ID/image和经批manifest绑定改变，其他服务、配置/环境/命令/网络、卷保持。回退实际执行和用户验收仍须各自如实记录。

失败回退：后端失败则停止UI更新，恢复原后端imageID/来源标签及原hash1271文件绑定；
若UI更新后回归，恢复原UIimageID/标签，并将后端恢复为同一a66配对版本及原manifest绑定。
仅换回a66镜像却挂新manifest不是有效回退。每步只点名目标服务、no-deps，身份、数据库、数据和卷不动。
回退后复核原两镜像、配置hash/只读绑定、健康、HTTPS与其他服务/卷清单；不成功则停止并保留错误。
不整库恢复、不覆盖durable事实、不清UNKNOWN/队列、不自动结算或释放预留；出现额外写入/配置发布/schema变更，交原owner单独处置。

## 已构建1e镜像的证据范围

上一轮固定1e3509bda06ad9b8160b0c5a7c5d451de9496c61标准构建已完成：后端6838b857b0936300b69349875ed36d9fdd863bd0a82bc781e65863a0e444491f，UI383a8972f1f6b23f2070e452a5d0baafa95e342bbff4d82042c992a4eca7f5da。
回执.local/selection-unified-build-1e3509-20261104-01/build-1e3509bda06ad9b8160b0c5a7c5d451de9496c61-All.json，SHA256 4F4EFB43CE23983891F01E40FF7CB613B5B543E34000C9B9F7B6D89A75BE31E3。
它们没有本轮采集关闭改动或#1089新合同，不作为新候选更新产物；原a66回退引用也不由这些镜像替代。
