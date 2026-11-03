# 产品研究的运行交接

本批范围见 [冻结设计](../architecture/market-research-selection.md)、[#1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072) 和 [PR #1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)。实例启用范围和检查结果以下方当前交接为准；健康检查不代替真实业务验收。合并、真实平台/模型/付费调用和数据删除仍未授权，实际用户试用不由实现者代签。

**当前代码批次**为[已确认的两个中心 V1](../product/market-research-selection-centers-v1.md)，新增 AI 沿[冻结证据分析合同](../architecture/market-research-evidence-analysis-v1.md)实现。2026-11-03 用户明确允许从已审源码 `ad3ed151d6dfc4d9c4862944391476dfe91a0dd2` 构建镜像并仅更新 `task-processor-unified-20261010` 前后端；又明确允许启动已停止的 Docker Desktop 并继续更新。现已完成两服务更新，两个新入口可见。此决定替代本新增批次原先“不部署”的阶段限制，不授权后续自动更新、其他服务升级或真实调用。

## 正常入口与可用路径

1. 正常登录自己的企业，从「数据服务 → 市场调研中心」`/workbench/data/research` 创建或打开本人 PRODUCT_RESEARCH 项目，进入 `/workbench/data/research/{projectID}`。「供应市场 → 智能选品中心」`/workbench/supply/selection` 及其同一项目详情共享原项目、候选、证据与人工决定。原「AI 工作台 → 项目中心」研究区继续可用；指定实例现已包含两个新中心。
2. 项目目标下方的「研究选品」区域先打开原数据市场 `/workbench/data/market`。真正采集仍要求单独授权、原权限、原最大 DATA_ROW 用量确认及原执行通道；本轮未执行。只读页面不触发采集或模型。
3. 「关联已保存资料」复用我的数据批次选择器，仅允许自己当前可读的服务端 `amazon_data` 结果。旧插件结果也使用该 kind，因此关联时由原 Data Services owner 核实；没有原服务端 item 的资料明确拒绝，页面排除该项并允许继续，不留下待核实写入。浏览器桥并未因此启用。展示站点/ASIN、原时间、parser、结构化证据 checksum、准确 publication/version、查询样本范围、已观察价格/币种与缺失项。无价为未知；不同币种不计算统一排名。来源撤权或不可读后遮蔽内容，保留可解除的私有关系槽。
4. 两中心选择 2–5 个可读候选，「读取所选样本」核对保存事实、同币种/同页面报价口径范围和缺项；切到选品中心核实原候选 revision、选定观察、digest 和商品引用。可选证据分析使用独立 `product.research.agent` / `product.research.decision`；须在原私密 manifest 显式配置 `productAgent.researchTextPolicies`，沿原凭据、模型路由和 Agent 激活/模式配置开放，不能借用其他智能体配置。准备后单独确认最多两次调用及 token/费用上限才开始；建议和引用保存在原 Agent 运行中，历史按项目读取。模型不可用不阻止只读事实与人工决定。原项目详情仍使用既有商品事实比较；全市场需求/趋势/销量、权重和利润没有开放。
5. 「提交人工审阅」后填写理由，再批准测试或否决；归档项目只读。新关联同站点/ASIN 归组但不覆盖已选证据；明确切换观察会回到待研究，旧决定/理由/证据身份保留在最近历史中。页面展示最近 20 项观察及当前选定项、最近 20 条人工历史；数据不会因窗口隐藏而删除。
6. 勾选来源可读且已批准的候选，按批次进入原我的数据弹窗。研究预填锁定原 item revision/publication/version，重新读取不一致则停止。显式清除预填后属于新的普通商品选择。确认发送非空 ItemIDs，空子集不能退化为整批；原独立整批操作有明确范围。跨批分别确认，不承诺全部成功。
7. 转入仍由 Preparation 执行，成功后沿原回执进入 `/workbench/supply/mine?preparation={id}` 查看待适配。研究页不保存“已转入”布尔状态。转入响应未知沿原 Supply key 核实；研究写响应未知先「核实原操作」，没有回执才允许明确使用原键重试。挂起意图按 actor/org/project 保存于浏览器会话存储，不自动重发。

浏览器桥、1688 货源匹配、利润与销量预测、监测、自动采购/上架未开放。服务端实际支持的站点和字段仍需获准的真实环境验证，不能从 fixture 或 CI 推断。

## 显式新空安装与受限 serving pool

只在另行获准的新空 Product 安装使用现有 SRC/Catalog/Collection/Data Services/Preparation installer；通用 installer 不作为已有业务库升级入口，不迁移旧数据、不让 serving 自动 DDL。指定统一实例的有界维护只适用[本次明确决定](../architecture/market-research-selection.md#11-指定实例启用决定2026-11-02)，先备份并精确核对当前约束，再在同一事务扩展；不重跑历史 initializer。Amazon Preparation 安装合同现在包含 `amazon_data`，拒绝 `sds_template`、`custom_dataset` 等其他来源。

研究新增三表 `research_opportunities` / `research_evidence_links` / `research_commands`，位于同一物理 Product 数据库。预先建立 `product_research_runtime` 普通角色，不能为 superuser、角色/数据库创建者、复制角色或有角色成员关系，不能拥有 public CREATE 权限。沿既有私密配置途径提供安装 DSN，不把密码写进 shell 历史或仓库：

```powershell
# MARKET_RESEARCH_SCHEMA_DSN 由操作员按原安全方式注入；本轮未对保留实例执行。
go run ./cmd/market-research-schema-init --runtime-role product_research_runtime
```

installer 只安装本 owner 三表并授予准确权限：三表 SELECT/INSERT；机会行仅更新 selected_link_id/state/reason/revision，关联行仅更新 removed。命令表无 UPDATE/DELETE，其他 owner 表无读取授权。serving 启动验证表约束和真实角色权限。来源与项目读取经原 Service 注入，不给研究 pool 其他 owner 的任意表权限。

在当前私密 JSON manifest 添加 `marketResearch.database`（字段 host/port/user/password/database/maxConnections，user 固定为上述角色，maxConnections 1–4）；host/port/database 与 `productAcquisitionDatabase` 一致，必须使用独立连接池。依赖已有 `projectCenter`、`dataServices`、`productCollections: true` 和 Product pool；Project 保持独立数据库。沿 [Data Services 运行交接](data-services-runtime-handoff.md) 配置显式 browser/driver/Temporal 和原 Resource，但本轮不执行实际任务。缺依赖、错误角色或 schema 均拒绝启动。

```powershell
go run ./cmd/current-application -config C:\private\current-application.json
# 前端仍使用现有同源服务 API 配置，正常启动 web/listingkit-ui：
pnpm --dir web/listingkit-ui dev --hostname 127.0.0.1 --port 3000
```

后端 context 返回 `marketResearchAvailable` 时项目页展示区域；没有安装则不伪造可用。已有选品 AI 与供应链继续使用各自显式模块、权限、配置和恢复 owner，不由研究开关自动开启。研究接口是 `/api/v1/workbench/research/projects/{projectID}`、`POST .../commands`、`GET .../commands/{key}`，前端同源 BFF 保留 verified actor/org 与原键，拒绝浏览器指定事实身份或 scope。

停止沿原 `-shutdown-file` 或正常进程终止执行，等待 serving/worker 结束再释放连接池；不会删除数据库或保存结果。重启读取原本地回执和当前来源，不能因重启重新采集、再次扣费或换键转入。数据保留/备份/恢复沿各数据库现有 owner，不新增研究专属恢复器。

## 当前指定统一实例交接（2026-11-03）

实际入口：[市场调研中心](https://localhost:35444/workbench/data/research)、[智能选品中心](https://localhost:35444/workbench/supply/selection)。原有正常 Edge 会话已分别打开两个页面，导航和项目列表正常显示；当前账号下还没有研究项目。未创建演示项目或伪造证据，用户可以先检查入口和项目表单。

运行源码固定 `ad3ed151d6dfc4d9c4862944391476dfe91a0dd2`，tree `1eaccf3770a93c480c573b7fadf3d89fd35f887c`；标准干净源码构建的后端镜像为 `sha256:6174e2993c69ff4a54c146b48df205767a7f506757a2aa2d0cddcbe6db68cc62`，UI 为 `sha256:6a8a94ca5134bee81b13deeae14695061510ce77d353336dc49f2e8044a7ec85`，均为 Docker 可寻址的镜像引用。容器来源标签已与实际镜像同步，两个服务 healthy。文档提交不改变本次运行的应用源码。

本次 Operational Design Basis 为 **Reuse Existing Architecture**：只更换两服务镜像及相应来源标签，未修改正式产品代码、schema、身份、权限、私密 manifest、环境、命令、entrypoint、网络或挂载。其余16个容器ID与镜像保留；Docker恢复后仅按原ID启动原有依赖，原已停用的 browser-collector / qualification-objects 保持停用。原十个应用命名卷保留，未执行 initializer、迁移、down -v、prune 或数据删除。

Product、Project 和 Agent 三库更新前均已生成私密 custom-format 备份，并以 pg_restore --list 验证归档可读；本次没有另建恢复库或宣称实际恢复验证。原配置和镜像回退材料继续保留。回退只按原两服务配置恢复旧镜像并保留当前durable records，不自动覆盖数据库备份或删除卷。私密备份/Compose不提交仓库或公开讨论。

HTTPS首页200、两个中心未登录307；原BFF项目/研究分析接口未登录401，内部原生研究分析接口未登录401。公开HTTPS入口不路由原生/api/v1路径，其404不作为原生鉴权结果。服务端data_acquisition_jobs、data_service_commands、research_commands、research_opportunities仍均为0。本实例原productAgent.enabled=false、researchTextPolicies未配置，保持不变；模型调用没有启用。

**NOT_RUN**：真实 Amazon/1688、真实模型/付费、实际保存证据→人工决定→转入的业务使用，以及用户产品验收。需要真实资料时另行明确站点、查询与最大DATA_ROW；模型另行明确配置、预算和调用授权。此次入口检查不代签以上结果，不合并或关闭Issue。完整构建、备份、保留检查和失败历史记于原PR #1075。

## 首批指定统一实例交接历史（2026-11-02）

用户批准的目标 `task-processor-unified-20261010` 已进行定向启用：仅替换后端和 UI，两者运行本 PR 已独立审查/CI 的应用源码 `d8b85805e82d8d6c8fe60c95ecdc88958c823390`；后续授权/交接文档提交不代表另一次应用逻辑变更。原网络、十个原挂载卷、环境变量、身份与 Browser Kernel 配置保留。新增配置只包含 `marketResearch.database` 及该实例原先已有但当前 manifest 未装配的 Data Services 显式子树；没有启动采集或模型请求。

入口：[统一实例项目中心](https://localhost:35444/workbench/ai/projects)。使用原有账户正常登录，创建/打开类型为「选品分析」的项目，在项目目标下方查看「研究选品」。不要在本次未授权阶段发起真实采集或 AI 调用。

实际启用证据：Product 数据库备份成功，并在新建的临时核验数据库恢复出 33 张表；核验库随后删除，原 Product 库保留。3 张研究表/受限角色/五类 Preparation CHECK 在同一有界事务提交；新角色实际登录成功，不能 UPDATE 命令表或读取原 Collection 表。两服务健康，正常入口可达，未登录研究 BFF 与原生后端研究/数据任务接口返回401。其余基础设施容器保持原ID，未删除/重建任何原卷。

当前没有已保存的服务端 Data Services 采集任务/证据；因此真实研究证据 → 人工决定 → 转入的运行验收仍为 **NOT_RUN**，不能用已存在的插件 `amazon_data` 代替服务端证据或伪造计费记录。此前隔离合成测试证据仍有效，实际用户试用由用户独立确认。

运维备份、固定镜像回执、候选/回退配置与运行记录保存在操作员专属受限目录，私密路径由 #1072/PR 仅作非秘密交接，不向公开讨论复制配置、凭据或数据库内容。镜像回退保留新 durable records 和所有卷，不自动还原整个数据库或收紧已消费的 Amazon CHECK。构建镜像使用 Docker 接受的 OCI index ID 固定；linux/amd64 平台 config ID 另行记录，不能把不可寻址的 config ID 当作创建容器的镜像引用。
