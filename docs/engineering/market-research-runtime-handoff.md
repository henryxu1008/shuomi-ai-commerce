# 产品研究的运行交接

## 一级菜单业务首页修正（2026-11-03，已更新指定实例）

用户指出一级入口仍是项目列表，要求参考其他一级菜单的业务流程首页。已用原正常Edge会话查看智能市场、店铺中心实际首页，复用现有 `PrimaryIntroduction` 完整组件、主题和响应式样式。新一级页面包含业务简介、三项入口、四步流程、业务协同/使用说明：市场调研“目标 → 资料 → 证据 → 样本汇总”，智能选品“候选 → 可选辅助分析 → 人工决定 → 核对转入”。所属视图合法 `step` 继续消费原 ProjectPage；详情、候选选择、pending 和原权限/恢复 owner 不变。

Design Basis: Reuse Existing Architecture，明确用户修正与已查看的当前首页为局部 Product Authority。首页只使用现有工作台上下文决定入口是否可用，不读取项目/候选业务数据或执行命令；无配置/资料权限或企业切换时不广告旧可用入口，未配置模型不阻止事实比较和人工决定。入口保留合法 mode/search，不引入新业务状态、事实 owner、schema 或执行/计量能力。

准确源码 `a2d12ec5462adbf1592e1e0747ec376fb3cc875c`、tree `83213bdf51af5617d0c0f051193e17ccc53b05e4`，标准干净Git archive构建UI镜像 `sha256:857f7ce8ff8cbcd374c6630d015c17b8a54898711831ef516caf4ab849a6b446` 已完成，仍维护原PR #1075。新8项回归先6 RED / 2 PASS再8 PASS；相关4文件85项通过，最后仅把可选能力位显式转换为boolean后重新8项通过，其他77项为未变化代码证据复用。typecheck、scoped eslint及生产构建通过；原独立Reviewer绑定 `a2d` 增量检查PASS，并独立8项通过，无新增finding。最新GitHub Actions无已观察run，保持UNKNOWN。

**首页更新 SUCCESS / In Review，待用户实际试用。** 用户已明确回复“允许”，授权准确 `a2d` 修正版仅更新 `task-processor-unified-20261010` 的前端并短暂重启。当前UI容器 `cae36d3c68be41fb21893c29d9f4a0898d333e65beff8c100ca3ab1eb2533e40` healthy，实际镜像为上述 `857f7ce8…`，容器/镜像来源标签与源码和tree一致。回退覆盖准确保留更新前 `2aea` / `b3fa12c3…`，不回退到更早版本。

Operational Design Basis: Reuse Existing Architecture。仅替换UI镜像和来源标签；其余17个容器的ID/image/运行或停止状态一致，后端仍 `ad3ed151` / `6174e299…` healthy，原两个停用服务继续停用。55个项目标签卷清单保留；UI command/entrypoint/networkMode/mounts/ports与基线一致，原私密Compose配置输入SHA256及42个环境变量名称集合相同，未读取或输出变量值。无schema、角色、原配置来源、业务数据或卷修改，无初始化、删除或prune。

正常原管理员Edge会话已打开两个一级原路由：各有业务简介、3项入口、4步业务流程和2项协同/使用说明，根首页没有“还没有项目”替代流程。已点击市场调研“管理研究项目”和智能选品“进入分析与人工评审”，分别进入原 `step=projects` / `step=analysis` 列表，面包屑、三级选中态、mode/search及先选择项目提示正确；点击一级菜单可返回对应业务首页。当前项目仍为空，未创建记录或执行采集/模型/转入。内部原UI实际端口3001未登录两个一级页307、项目BFF401，正常浏览器HTTPS访问完成。

本次只读现场核对和截图记录在本地首页交付目录的 rollout 下；全页截图曾超时，改用可见视口截图完成两页证据，不把截图故障当功能结果。原opt-in合成页面用例本轮未执行，移动端与完整业务使用未重跑；真实平台、模型、付费和用户产品验收仍NOT_RUN，开发/运行核对不代用户验收。后续其他版本/服务更新仍需具体授权，不合并、不关闭Issue。

## 三级业务导航交付（2026-11-03，前一版保留记录）

用户已确认“市场调研 / 智能选品”为一级菜单，二级展示业务阶段，三级进入具体功能，名称去掉“中心”，随后明确允许将已构建的准确 `2aea` 版本仅更新统一实例前端并短暂重启。运行 UI 源码为 `2aea4d1a6cf7a22776c341def70d4058e04a9fe4`、tree `68f2bbd30f75a891ca77614cf2eff29b1543ee95`，仍为原 PR #1075。路径沿用 `/workbench/data/research`、`/workbench/supply/selection` 和本人项目详情；完整层级见[当前产品增量](../product/market-research-selection-centers-v1.md#当前导航增量2026-11-03)。

二级为流程展开分组；三级通过白名单 `step` 选择已有功能区域。有项目时保留同项目的有效查询条件和准确 URL 样本，无项目时先选择或创建研究项目再继续。只切换步骤不新建项目、调用模型、提交人工决定或转入；项目与研究请求、原 visit 不因该切换重复，面板中的已选候选、草稿理由和原 pending 保持。跨身份 / 企业 / 成员 / 权限变化仍沿原准入与 key 隔离。

相关6个测试文件在前一候选106项通过；最终两文件增量新增真实 ProjectPage → ResearchPanel 的延迟 GET 回归，先在旧码确认丢失选择 RED，再用相同身份的稳定 scope 修复，ProjectPage 14项全部通过，typecheck / scoped eslint 通过。独立 Reviewer 对原导航增量及最终两文件修复分别检查，`2aea` 对应 finding 已关闭，无剩余 finding。未变化代码的测试证据明确复用，不冒称在新 SHA 全量重跑。最新 GitHub Actions run 尚未观察到，状态 UNKNOWN。

**导航更新 SUCCESS / In Review，待用户试用；完整业务验收 NOT_RUN。** 通过仓库标准脚本从准确 `2aea` Git archive 构建的 UI 镜像为 `sha256:b3fa12c3e0d4be4bbfa212fc276fa64b9c48ede0760ba0999fc738ed40ea2652`，干净 Next 编译、类型及标准镜像检查全部通过；已仅替换 `task-processor-unified-20261010` 的 `listingkit-ui`，新容器 `d970d39d5435fe5ee5e2c2be5fa78cfff5ae20253e28492dc43e5f91e5dd40e2` 为 healthy，容器/镜像来源标签一致。前一 `1ce` 候选虽曾构建，但对应真实父页选择丢失 finding 未关闭，不作为交付镜像。

Operational Design Basis: Reuse Existing Architecture。仅覆盖镜像和来源标签，复用原私密 Compose 与本次 UI 覆盖文件；其余17个容器 ID、image 和运行/停止状态相同，含原两个停用服务；后端仍为 `ad3ed151` / `6174e299…`。项目标签下55个命名卷的清单原样保留，不做 prune 或删除；UI entrypoint、command、networkMode、mounts、ports 与基线一致，原配置输入文件 SHA256 与42个环境变量名称集合相同，未读取或输出变量值。原 UI `6bbe5bb48` / `efa4d769…` 及其来源标签的回退覆盖保留。

正常管理员 Edge 会话已通过原 HTTPS 入口打开新页面：两个页面标题/导航均去掉“中心”，两入口实际为 `console-nav-level-1`；已展开研究的3个业务分组和选品的4个业务分组，能看到9项三级功能。点击“取得与关联资料”“核对转入商品”分别进入原列表的准确 step，面包屑、选中态和“选择或新建研究项目后继续”提示一致；当前仍无项目，未创建资料或研究命令。截图和非秘密运行/保留回执保存在本次本地交付目录。

内部 UI 原实际端口3001的未登录检查：两个页面307，项目BFF401。第一次内部检查误指共享网络的3000端口得到404，不作为UI鉴权证据；纠正端口后通过。Windows公共HTTPS curl因证书吊销检查离线失败，没有降低TLS验证；正常浏览器HTTPS读取成功，不能把该curl失败写成通过。未执行真实采集、模型、付费或研究业务。后续其他版本/服务更新仍需具体授权，不合并、不关闭Issue；用户试用不由实现者代签。

本批范围见 [冻结设计](../architecture/market-research-selection.md)、[#1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072) 和 [PR #1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)。实例启用范围和检查结果以下方当前交接为准；健康检查不代替真实业务验收。合并、真实平台/模型/付费调用和数据删除仍未授权，实际用户试用不由实现者代签。

**当前代码批次**为[已确认的两个中心 V1](../product/market-research-selection-centers-v1.md)，新增 AI 沿[冻结证据分析合同](../architecture/market-research-evidence-analysis-v1.md)实现。2026-11-03 用户明确允许从已审源码 `ad3ed151d6dfc4d9c4862944391476dfe91a0dd2` 构建镜像并仅更新 `task-processor-unified-20261010` 前后端；又明确允许启动已停止的 Docker Desktop 并继续更新。现已完成两服务更新，两个新入口可见。此决定替代本新增批次原先“不部署”的阶段限制，不授权后续自动更新、其他服务升级或真实调用。

## 正常入口与可用路径

1. 正常登录自己的企业，从一级菜单「市场调研」`/workbench/data/research` 创建或打开本人 PRODUCT_RESEARCH 项目，进入 `/workbench/data/research/{projectID}`。一级菜单「智能选品」`/workbench/supply/selection` 及其同一项目详情共享原项目、候选、证据与人工决定。二级按业务阶段展开，三级进入对应功能区域；没有项目时先选择项目。原「AI 工作台 → 项目中心」研究区继续可用。
2. 项目目标下方的「研究选品」区域先打开原数据市场 `/workbench/data/market`。当前成员须具备数据市场使用与相应商品资料查看权限；不足时显示明确的权限拒绝，由企业管理员核对，不自动授予角色或权限。真正采集仍要求单独授权、原最大 DATA_ROW 用量确认及原执行通道；本轮未执行。只读页面不触发采集或模型。
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

后端仍运行 `ad3ed151d6dfc4d9c4862944391476dfe91a0dd2`，tree `1eaccf3770a93c480c573b7fadf3d89fd35f887c`，镜像为 `sha256:6174e2993c69ff4a54c146b48df205767a7f506757a2aa2d0cddcbe6db68cc62`。同日后续数据市场错误反馈修复仅更新 UI：源码 `6bbe5bb48695fa9441bd0db8342aeea2ce09b182`，tree `92166bb9e80d055204a0bf0f7c639d8fc81be945`，镜像 `sha256:efa4d769a69cf5410806f34fdd06dcb0424db95b611fc0c4fbf040db082a1a78`；原 UI `6a8a94ca…` 保留为回退。上述均为标准干净源码构建、Docker可寻址的镜像引用，容器来源标签与对应镜像同步，两个服务healthy。后续文档提交不改变运行的应用源码。

本次 Operational Design Basis 为 **Reuse Existing Architecture**：只更换两服务镜像及相应来源标签，未修改正式产品代码、schema、身份、权限、私密 manifest、环境、命令、entrypoint、网络或挂载。其余16个容器ID与镜像保留；Docker恢复后仅按原ID启动原有依赖，原已停用的 browser-collector / qualification-objects 保持停用。原十个应用命名卷保留，未执行 initializer、迁移、down -v、prune 或数据删除。

Product、Project 和 Agent 三库更新前均已生成私密 custom-format 备份，并以 pg_restore --list 验证归档可读；本次没有另建恢复库或宣称实际恢复验证。原配置和镜像回退材料继续保留。回退只按原两服务配置恢复旧镜像并保留当前durable records，不自动覆盖数据库备份或删除卷。私密备份/Compose不提交仓库或公开讨论。

HTTPS首页200、两个中心未登录307；原BFF项目/研究分析接口未登录401，内部原生研究分析接口未登录401。公开HTTPS入口不路由原生/api/v1路径，其404不作为原生鉴权结果。服务端data_acquisition_jobs、data_service_commands、research_commands、research_opportunities仍均为0。本实例原productAgent.enabled=false、researchTextPolicies未配置，保持不变；模型调用没有启用。

**NOT_RUN**：真实 Amazon/1688、真实模型/付费、实际保存证据→人工决定→转入的业务使用，以及用户产品验收。需要真实资料时另行明确站点、查询与最大DATA_ROW；模型另行明确配置、预算和调用授权。此次入口检查不代签以上结果，不合并或关闭Issue。完整构建、备份、保留检查和失败历史记于原PR #1075。

### 同日后续：数据市场502与权限反馈修复

用户另行明确允许先查清并修复数据市场502/权限链路，继续在现有实例推进，同时禁止自行提权和真实采集、AI、付费调用。Design Basis: **N/A**，仅修正当前统一错误合同的有界反馈，不改变授权、租户、状态、持久化或外部副作用语义。当前后端审计确认配置、抓取历史、定制历史三个只读请求因缺少 `workbench.data-market.use` / `workbench.collection.read` 被拒绝；原BFF只接受Data Services嵌套错误，将统一鉴权层的顶层PERMISSION_DENIED误报502。

UI修复复用现有严格Workbench错误解析器并校验code/status，只重包合格的统一错误；不把后端错误正文透传给页面，写入的5xx/畸形/矛盾结果仍保留DATA_UNKNOWN及原键恢复。成功响应、服务器Bearer/企业上下文和原生授权逻辑未变。相同处理也适用于原API管理及专员BFF消费者。

正常保留会话现场核实三个请求现在均为 **403 / PERMISSION_DENIED**；页面说明缺少权限及需管理员核对的两项权限，采集与定制按钮仍禁用，历史读取失败不显示成空历史成功。当前权限摘要仍仅有原店铺查看/浏览器使用权限，未修改任何角色或授权。仅UI容器替换，其余17个容器ID/image及环境、命令、entrypoint、网络、挂载比对不变；四类采集/研究记录仍0。源代码独立增量检查与运行证据分开记录于PR；此修复不代表当前账号已具备采集权限或真实业务验收完成。

## 首批指定统一实例交接历史（2026-11-02）

用户批准的目标 `task-processor-unified-20261010` 已进行定向启用：仅替换后端和 UI，两者运行本 PR 已独立审查/CI 的应用源码 `d8b85805e82d8d6c8fe60c95ecdc88958c823390`；后续授权/交接文档提交不代表另一次应用逻辑变更。原网络、十个原挂载卷、环境变量、身份与 Browser Kernel 配置保留。新增配置只包含 `marketResearch.database` 及该实例原先已有但当前 manifest 未装配的 Data Services 显式子树；没有启动采集或模型请求。

入口：[统一实例项目中心](https://localhost:35444/workbench/ai/projects)。使用原有账户正常登录，创建/打开类型为「选品分析」的项目，在项目目标下方查看「研究选品」。不要在本次未授权阶段发起真实采集或 AI 调用。

实际启用证据：Product 数据库备份成功，并在新建的临时核验数据库恢复出 33 张表；核验库随后删除，原 Product 库保留。3 张研究表/受限角色/五类 Preparation CHECK 在同一有界事务提交；新角色实际登录成功，不能 UPDATE 命令表或读取原 Collection 表。两服务健康，正常入口可达，未登录研究 BFF 与原生后端研究/数据任务接口返回401。其余基础设施容器保持原ID，未删除/重建任何原卷。

当前没有已保存的服务端 Data Services 采集任务/证据；因此真实研究证据 → 人工决定 → 转入的运行验收仍为 **NOT_RUN**，不能用已存在的插件 `amazon_data` 代替服务端证据或伪造计费记录。此前隔离合成测试证据仍有效，实际用户试用由用户独立确认。

运维备份、固定镜像回执、候选/回退配置与运行记录保存在操作员专属受限目录，私密路径由 #1072/PR 仅作非秘密交接，不向公开讨论复制配置、凭据或数据库内容。镜像回退保留新 durable records 和所有卷，不自动还原整个数据库或收紧已消费的 Amazon CHECK。构建镜像使用 Docker 接受的 OCI index ID 固定；linux/amd64 平台 config ID 另行记录，不能把不可寻址的 config ID 当作创建容器的镜像引用。
