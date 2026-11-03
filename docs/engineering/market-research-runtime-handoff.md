# 产品研究的运行交接

## 现有文字凭据元数据已取得（2026-11-03，只读）

沿原受限运行账户 `product_agent_runtime`、已授予SELECT的去密投影和READ ONLY事务，固定LAC组织级筛选，确认两条deepseek-v4.1-flash / openai-compatible记录。首选审查候选为原内容current正在引用的 `local-content-route-830-20261020`，credentialVersion为 `org-credential:v1:9b2ab8feb2e44d7399f9649a8debe471355bbb0388a5a06b022d507ce7907737`，与原current一致；enabled/key_present为true，端点主机格式提示北京。未返回Key值、完整端点或其他企业/成员记录，没有提权、改授权或借用管理员。较早试用记录不作为fallback。

原内容current和固定非思考策略已确认，但研究/选品/企业共享current未见；新共享reference/边界证据、厂商准确版本/价目和完整usage资格仍未核准。现场Invocation为53列旧结构，缺官网价/完整计费用量字段；新owner为52列且严格拒绝旧结构，因此实际运行还需要另行明确实例/数据与新合同适用安排，不能只保存模型名或自动迁移。完整元数据、非执行差异和读取边界见[候选差异](selection-shared-model-wiring-preview.md#已取得的受限元数据与候选差异2026-11-031246-utc)。

Design Basis: N/A；原预算¥0.30/20点每项、总¥0.60/40点保持。仅查询元数据并更新文档，未改Key/配置/schema/实例/余额，未执行真实采集/AI/付费或合并。下方“未取得记录”的观察仅属于此前页面核查阶段，不否定本次新增证据。

## Local 百炼绑定与价格只读核查（2026-11-03，未改实例）

正常LAC管理页及三个既有去敏GET已核：能力列表200且仅图片，研究绑定400/INVALID、独立商品对比绑定503/UNAVAILABLE。因此没有取得文字binding ID/版本、凭据版本、endpoint/部署地域，不能写成“密钥不存在”。未读取secret、使用管理员DB或借用图片Key。当前两服务OCI来源标签均 `a66d7d0cde74cded28d77060ac941851cc342a30`，镜像2848da261bfc/d106b5dfe863、running/healthy；与下方旧运行记录不同，本轮只观察，不是本线程f872工程代码已更新实例。

已核官方型号/价格和usage字段；百炼地区原价与DeepSeek厂商价的缓存单价/峰谷日历不同。厂商当前 `deepseek-flash` 对应1017，不能推定用户指定的百炼原名称已经是同版本。候选仍保留原名称、固定非思考、原两调用和预算；现有严格openai-text-v1字段可对齐，但本企业真实完整用量资格未验证。完整的当前→目标差异、公开价格、整数预算和阻碍见[可审查预览](selection-shared-model-wiring-preview.md#本次现场核查与配置差异2026-11-03只读)。

Design Basis: N/A（只读核对及文档）。单项成本¥0.30/20点、总¥0.60/40点不变；无凭据/配置/服务/单位/余额变更、真实采集/AI/付费或合并。配置仍NOT_PUBLISHABLE；下方工程检查、运行观察和历史授权继续分别绑定其准确版本。

## 共享模型稳定依赖组合（2026-11-03，离线实现完成，待独立检查）

已核对原PR #1075的 `247a0b662` / CI41597564207成功；#1078准确依赖 `341278d5743cb9f9c4b2e257d72ae8a5c0ed1aa5` / CI41620567748成功，有限安全基础已独立工程批准。本开发分支已组合该已提交依赖及其main `84206e6e`，未操作GitHub合并或上游在途工作区。Design Basis: **Reuse Existing Architecture**，依据Issue #1073当前Ready记录、研究/选品冻结合同及#1078限定安全基础；无新增公共合同、计费或恢复owner。

项目资料分析与独立商品对比现共同消费原 `SharedTextReader` / `SharedCategoryReader`，保留各自operation/prompt/schema、权限/read_only、原快照/历史/恢复和两调用预算。研究的私有准入描述改用原shape校验，执行仍必须取得共享owner提供的有效官网价快照；不以私有描述或旧静态策略作为执行价格/当前选择。研究夹具已适配 `OfficialTariff`，未恢复旧点价。运行装配保留研究独立受限池，导航保留main原计划参数和研究项目上下文。

合成HTTP已实际贯通两项准备→本地模拟provider两调用→原Invocation/资源账本→历史/原key找回及原撤权/证据保护检查。使用任务专属新PostgreSQL17.6，仅复用已有二进制，不连接保留实例或改其配置。相关领域/模型/配置/运行包通过，HTTP修正后28项通过、无skip；研究配置/模式/完整快照回归已实际运行。入口/导航92项通过（2文件），UI typecheck通过。原研究准入测试先RED→GREEN，依赖组合编译/HTTP失败及后续测试夹具预留断言修正均保留原日志；完整新候选CI与独立实际增量检查另记于PR，不把旧CI当新HEAD成功。

预算保持每项渠道成本≤¥0.30、消费≤20点，两项各一次总≤¥0.60/40点。合成32k/4k、两调用的官网价上界为19.2点，原执行器还会收紧实际输入上界；各合成run的两条完整用量实际合计24000微点，原bucket消费一致、剩余预留0，渠道成本另核为240货币微元。这些是显式测试价格/用量，不是实际百炼报价、当前配置或收费。

**离线消费者组合已完成，不代表真实模型/配置可用。** Local DeepSeek准确binding/官网价/usage资格仍未核实，Flare/完整三类发布及完整视觉消费者仍未开放。没有改凭据、发布配置、更新实例、真实采集/AI/付费、迁移或合并；保留库不能自动换单位。实际运行与用户验收NOT_RUN，原实例不变。准确最终源码/CI/独立检查见[PR #1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)。

## 读取器准入切片记录（2026-11-03，25477d81f，未更新实例）

用户已授权沿原工作区和PR #1075推进离线开发与合成回归。Design Basis: **Reuse Existing Architecture**，Issue #1073已先于改码明确Ready的范围：复用现有 `RoutePolicyReader`，修复两项application把“无静态策略”误判为“没有模型读取器”的构建判断。应用源码 `25477d81ffa4891615a4c7246e11d7d46cf1d3dd` 已提交；两个消费者允许只注入权威reader，并保留各自operation、prompt/schema和原执行链。读取失败不回退到静态策略；没有任何模型配置时仍保留事实查看路径。

已一次同步main `c6f53e1a928862e556bbf0a1883c2af03dbedfb7`；保留研究与类目规则两种引用并验证互斥。工厂回归先复现两项缺陷RED，再最小修正GREEN；相关研究/选品/模型/DeepSeek适配与组合回归79项通过（含子测试），7项PostgreSQL端到端测试因未提供任务专属DSN而SKIPPED。SQL mock及本地模拟provider属于合成回归，没有连接保留实例或真实平台。日志在被Git忽略的 `.local/selection-offline-wiring-20261103-01/`；新代码的最终独立检查与PR CI单独记录于PR，不由旧证据代签。

**仅现有读取器的消费者接线完成；完整企业共享text、官网价与微点运行组合仍NOT_READY。** #1078稳定基线仍为 `0bd9d243af86710dc2e2b1329b7e3e6b0b5503ac`，完整执行profile/价格/单位装配仍由其原owner交付；未复制其未提交代码或新增公共接口/计费能力。同根因的其他消费者由该公共批次处理，本切片仅覆盖已授权的两项操作。预算保留每项¥0.30渠道成本和20点消费、两项各一次总¥0.60/40点；合成profile所需的旧类型占位不是新点价或微点实现。

没有改凭据、发布配置、更新实例、真实采集/AI/付费或GitHub合并。保留实例的启用/read_only及旧单位数据继续保持；本轮只交付离线代码，不宣称用户已经能运行付费分析。待#1078准确稳定提交提供原生运行装配后，沿已有开发授权接入剩余部分；真实配置与调用仍需另行授权。

## 百炼配置接线预览与统一模型重构对齐（2026-11-03）

用户已选择Local Account Center / 百炼 `deepseek-v4.1-flash`，仅准备两项操作的配置接线及差异预览；每项渠道成本上限¥0.30、消费上限20点，两项各1次合计¥0.60/40点。**未授权**写凭据、发布模型配置、更新服务、真实采集/AI/付费调用或合并。完整可审阅结果见[共享文字模型接线与差异预览](selection-shared-model-wiring-preview.md)。

已核对用户指定“检查可关闭的 Issue”线程及PR #1078的当前冻结设计/开发状态：两项操作应共同读取企业text类别，保留各自prompt/schema/权限/预算；用户收费按厂商官网适用价、冻结FX和完整实际用量折算微点，由原资源owner唯一预留/结算。**下节200/800独立点数资费草案已撤回**，不再作为后续配置依据；20/40点仅为消费上限。旧per-operation当前模型/费率也不作为新共享reader的fallback。

本次DeepSeek是Local企业共享文字模型候选，未改变全局OpenAI/Mini/Flare默认决定；将来发布共享text选择会影响本企业其他已接通文字消费者，预览已列出影响，不能声称仅影响两个Agent。三类整体发布、Flare上界、准确文字binding/版本/官网价与百炼usage资格、上游完整微点交付和保留实例适用范围仍未就绪。预览保持REVIEW_ONLY/NOT_PUBLISHABLE，无真实配置/API发布请求或运行变更。

## 文字模型与预算只读核对（2026-11-03，候选与预算草案）

用户选择“先核实项目已接入的文字模型与可用绑定，再给出候选和预算”，仅授权方案核对，不授权改凭据、更新服务、真实AI或付费调用。本节 Design Basis: **N/A**，仅记录只读现状与未发布方案，不实施或改动生产配置。此前两项已启用/read_only版本1的事实继续有效。

正常管理员浏览器在Local Account Center的“我的账户 → 企业空间 → 模型与费率”确认当前身份为平台维护者，已接线用途仅图片生成，未展示两项文字分析用途。当前不能从管理入口取得可用的组织文字绑定；底层文字凭据是否存在、适用地域、版本和有效性仍为 **UNKNOWN**，未读取密钥或用管理员数据库连接补证。运行后端ad3源码已包含两项独立Research/Selection策略、当前模型/费率reader、受限凭据供给及固定适配；缺口是本实例对应用途尚未开放，不能把图片配置当作文字配置。正式当前模型/费率仍由原已发布数据库版本负责，部署准入及硬上界不成为第二当前选择来源。

只读资源观察（2026-11-03 06:31 UTC）：企业可用AI点数 **9934**，另有预留24、已消费42；Local Account Operator本月上限10000，已消费0、预留0、剩余10000（2026-11-01至12-01 UTC）。不改余额、原预留或成员上限；供应商账户余额未核实。正常页面记录在本地 `.local/selection-model-budget-readonly-20261103-01/`。

| 项目已有适配的候选 | 核对结论 | 官方公开价格参考与限制 |
| --- | --- | --- |
| 优先核实：DASHSCOPE / `deepseek-v4.1-flash` | 已有固定非思考策略 `dashscope-deepseek-v4.1-flash-nonthinking-v1` 和兼容接口适配；本企业文字凭据/绑定未核实 | [百炼价格](https://help.aliyun.com/zh/model-studio/model-pricing)：每百万输入/输出Token忙时，北京及表列全球范围为2/8元，新加坡国际范围为2.188/8.75元；地域、实际账号与适用价须在配置前核实，不假设免费额度、缓存折扣或包月抵扣 |
| 备选：Google / `gemini-3.8-flash` | 已有原生Interactions适配；本企业凭据和真实业务资料准入未核实 | [Gemini价格](https://ai.google.dev/gemini-api/docs/pricing#gemini-3.8-flash)：截至2026-12-31，标准付费输入/输出含思考分别0.75/3.75美元每百万Token；2027年起分别1.50/7.50美元。保持美元计算，不做隐式换汇，不将Free Tier合成验证扩为真实资料许可 |

**预算草案，未授权、未发布**：每项业务分析最多2次模型调用，每次最多32000输入/4000输出Token，原事实披露与输入字节门禁继续适用；超限停止，不删改事实来凑预算。按此上限，百炼单项分析的公开单价估算分别为0.192元（2/8）或0.210032元（2.188/8.75）；两项各1次约0.384至0.420064元。可提出单项金额硬上限 **0.30元**、本轮仅两项各1次合计 **0.60元** 的执行范围，最终以准确已发布费率和准备页报价为准；未知结果沿原owner核实，不追加新运行或自动重发。Google同一Token上限为0.078美元/项，两项0.156美元，不混用CNY预算。

历史记录：此轮曾提出每百万输入200点、输出800点、逐调用取整的独立资费草案，未发布。**该草案现已因用户指定的#1078共享模型/官网价/微点设计撤回**，不作为后续开发或配置依据，不复用原标题费率或调用授权。新消费上限为20点/项、40点总量，实际费用沿厂商官网价折算并保留微点；原资源余额和成员月限不因预览改变。

后续顺序：经明确决定核实/准备本企业两个准确文字用途的去敏绑定及原运行依赖 → 按原配置owner形成模型、费率及硬上限差异预览 → 获准后发布配置/进行必要的有界服务更新，刷新能力就绪 → 用户明确商品/项目资料和费用后，先独立对比、再项目资料分析各1次。项目分析当前仍缺项目与合格资料；不以独立商品样本、图片Key或SHEIN未开放来源替代。需要新schema、缺少合格凭据或其他范围变化时先报告具体缺项，不静默初始化/提权/借用其它用途。

本次没有凭据secret读写、模型配置发布、额度修改、构建/服务更新、真实平台/模型请求、商品操作或合并。上述估算不是已生效预算、供应商账单、质量或完整业务验收。

## Local Account Center 两项操作启用与只读设置（2026-11-03，已执行）

用户明确授权在 `Local Account Center` 启用项目资料分析与独立商品对比，分别保存“只读”，核对刷新和入口切换；保持AI关闭，不采集、不生成建议、不改模型凭据、不选择或操作具体商品、不合并。本次 Operational Design Basis: **Reuse Existing Architecture**，仅使用现有企业启用、执行模式保存和正常工作台导航，不修改应用代码、模型配置、权限合同或服务镜像。

| 操作 | 当前企业启用 | 执行设置 | 整页刷新 | 使用入口核对 |
| --- | --- | --- | --- | --- |
| 项目资料分析 `product.research.agent` | 已启用 | `read_only`，配置版本1 | 启用与设置保留 | “查看研究事实”实际进入 `/workbench/supply/selection?step=analysis`，项目仍为空，未创建项目 |
| 独立商品对比 `product.selection.agent` | 已启用 | `read_only`，配置版本1 | 启用与设置保留 | “查看商品事实”实际进入原配置详情内比较页面，未选择批次或商品，“准备分析”禁用 |

操作前发现浏览器在另一个企业的代管视图，先通过正常企业选择器切回明确授权的 `Local Account Center`，确认企业后才提交启用和设置。未在另一个企业提交配置命令。我的智能体整页刷新后显示一张智能选品职责卡片，两个操作均已启用；切换本次操作后，能力弹窗的准确操作、管理链接和事实查看链接对应切换。现有3项其他职责继续保留，职责计数由3变4，不将两个操作错误计成两个职责。

两项文本生成能力仍显示“需要配置”；服务端AI开关及模型凭据未修改，未准备或确认任何分析。只读模式本身不是免费或禁止模型调用的开关，本次另行遵守AI关闭与禁止调用的边界。**NOT_RUN**：具体商品事实读取、真实采集、AI建议生成、费用/扣费验证、完整业务使用及用户产品验收。应用仍为下文e75前端/ad3后端，无构建、服务更新或合并。

正常浏览器页面/刷新截图与核对记录保存在本地 `.local/selection-enable-readonly-20261103-01/`，最终停留在本企业“我的智能体”。本次复用现有行为，无生产代码修改或新增测试工具。发现一处非阻塞显示问题：项目资料分析的事实读取能力有说明和可用状态，但标题漏显，记为 `BACKLOG`；未扩大本次配置操作范围修代码。

## 当前指定实例：智能选品操作入口已更新（2026-11-03）

**前端更新 SUCCESS / In Review，待用户试用。** 用户明确授权构建这版前端并仅更新 `task-processor-unified-20261010`，保留配置、数据和卷。实际运行应用源码为 `e75b7a3d963c78376d2c74367d05f8d38eb2a913`，tree `0c890708b3f2b05df7ee416478287b3abfc75fb6`；后续交接文档提交不改变运行应用源码。下文较早版本均为各阶段记录，当前运行以本节为准。

入口：[智能体市场](https://localhost:35444/workbench/agents/market) → 选品 → 智能选品智能体。两个操作分别进入[项目资料分析配置](https://localhost:35444/workbench/agents/mine/product.research.agent)和[独立商品对比配置](https://localhost:35444/workbench/agents/mine/product.selection.agent)，详情及能力配置继续使用各自原操作 ID、权限与就绪状态。项目分析的“打开项目资料分析”进入原 `/workbench/supply/selection?step=analysis`，独立商品对比保留详情内原比较及本人历史。实际启用与配置保存仍须明确操作；本次没有提交这些命令。

标准 `scripts/build-listingkit-images.ps1 -SourceRef e75b7a3d963c78376d2c74367d05f8d38eb2a913 -Component UI` 构建成功，生产编译、类型检查与单应用 COPY 层检查通过。UI 镜像为 `sha256:548ce6b674127d4c795a70d5fa5ff46848fa316d4ed3fa9f836cb86b33821cd6`，容器 `dd34465f15e9972e8983519204a1d815080dada7d0ff6166c49039cc166b8ab6` healthy，镜像/容器来源标签与准确源码、tree 一致。只对 `listingkit-ui` 执行原 Compose 输入加镜像/来源标签覆盖的 `up -d --no-deps --no-build --pull never`。Compose 对既有其他服务的 orphan 提示未触发清理；未使用 remove-orphans。

Operational Design Basis: **Reuse Existing Architecture**。更新前后其余17个容器的ID、镜像、状态与挂载一致；后端仍为 `ad3ed151d6dfc4d9c4862944391476dfe91a0dd2` / `sha256:6174e2993c69ff4a54c146b48df205767a7f506757a2aa2d0cddcbe6db68cc62`，保持 healthy。原已停止的 browser-collector / qualification-objects 继续停止，55个项目标签卷清单一致。UI command、entrypoint、网络、端口与挂载、原两个 Compose 输入文件哈希及42个环境变量名称集合一致；没有读取或输出变量值。未改私密配置、角色、schema、业务记录或卷，未初始化、迁移或 prune。准确回退覆盖保留更新前 a2d / `sha256:857f7ce8ff8cbcd374c6630d015c17b8a54898711831ef516caf4ab849a6b446`，回退不恢复或删除数据库。

前端更新时，原正常管理员 Edge 会话只读核对：选品分类显示一张职责卡片和两个准确操作链接；当时两个配置详情均为尚未启用、文本模型需要配置。项目使用入口实际进入原项目选择页，当时没有研究项目；独立比较仍显示原比较面板和历史，没有选择或读取具体商品事实。“我的智能体”当时保留原3项已启用职责，未启用的选品未伪造为已启用；后续获准启用与只读配置结果见本页首节。两个业务首页已显示“先了解市场，再决定做什么商品”和“把重点商品比清楚，再决定下一步”，包含业务流程并说明当前仅接通 Amazon、其他来源未开放；此前54fb文案已包含在本次e75镜像中，无需另部署旧候选。

内部实际UI端口3001只读检查：healthz 200，市场/两个配置详情/研究首页/选品分析页未登录307，项目BFF未登录401。构建回执、更新前元数据、候选/回退覆盖、保留检查、健康和只读页面记录/截图保存在本地 `.local/selection-agent-navigation-20261103-01/`；不提交私密配置或凭据。

前端更新检查当时 **NOT_RUN**：实际启用/配置保存、启用后的选品操作切换、真实采集、AI/付费调用、完整业务使用及用户验收。启用、配置保存和切换此后已按本页首节明确授权执行，其余项目继续NOT_RUN。59项开发测试和准确e75独立源码复核证据沿原PR #1075复用，不冒充未执行操作的现场验收。完整市场调研智能体及SHEIN美国来源仍未开放。不合并、不关闭Issue。

## 现有操作的启用、配置与使用入口（2026-11-03，更新前源码记录）

用户允许先打通现有操作。智能市场选品类在“智能选品智能体”卡片中分别提供“项目资料分析”“独立商品对比”链接，进入原企业配置详情后明确启用。我的智能体在同一职责中选择本次操作，配置/状态/使用入口跟随该准确操作；详情及能力弹窗可切换两个操作，原配置和历史不合并。

更新后的操作步骤：打开 `/workbench/agents/market` → 选品 → 智能选品智能体 → 选择操作 → 在原详情检查权限/就绪并明确启用 → 返回我的智能体选择同一操作并进入使用。项目资料分析继续到 `/workbench/supply/selection?step=analysis` 选择本人有权访问的项目，独立商品对比继续在 `/workbench/agents/mine/product.selection.agent` 使用原流程。查看资料/历史不代调用AI；未启用、缺模型或预算时按原原因阻止执行。

源码交付阶段只改前端消费者与交付说明，未改变实际企业配置/角色、数据、模型、后端或保留实例。当时统一实例仍是a2d前端/ad3后端，没有新镜像或部署；后续经用户明确授权的e75更新见本页首节。准确源码、RED/GREEN、必要检查和独立增量复核在同一PR #1075记录，不能把开发测试当作用户试用。

完整市场调研定义与SHEIN美国来源仍未准入，不在目录新增可启用空壳；原工作页/一级入口暂保留直至替代路径完整。真实平台、AI、付费、用户业务验收NOT_RUN，不合并、不关闭Issue。

## 业务定位与通俗文案修正（2026-11-03，先前源码记录）

用户指出Amazon被写成整个市场调研的范围，流程说明又偏技术术语。当前已完成独立于新渠道实施的展示修正：首页从目标市场、商品类别和调研问题介绍用途，Amazon仅在资料来源说明/选择环节作为当前已接通渠道。其他来源明确未开放，未将改文案写成多平台输入能力；来源选择器与后端research.validEvidence仍保持当前Amazon合同。

准确源码 `54fb5898362aaba47689e0748f8f19579f724da1`、tree `bf6aa0fe2fcfd255165be02251e4a83cbf4d6e65`。新标题为“先了解市场，再决定做什么商品”和“把重点商品比清楚，再决定下一步”。首页、菜单、项目与主要操作说明改用调研项目、商品资料、重点商品、价格与特点、选择理由、后续商品准备等业务用语；来源/版本等追溯信息保留在资料详情。N/A（纯显示文案/呈现）：原路由、step、DOM定位、来源准入、权限、handler、默认AI分析目标、费用上限、同键重试和未知结果处理不变。2–5件、币种/价格条件、资料不足、来源不可用和费用确认继续准确表达。

必要既有7文件102项测试、typecheck/scoped eslint通过，纯文案未为形式新增RED测试。原独立Reviewer绑定 `54fb` 对14文件增量检查PASS，无新增finding，未重复全量测试或重审原架构。标准Git archive UI构建通过，镜像 `sha256:9551cc00fcfdf990de23edfe02e49838c26c5b148bb6c63217946dd75448ef9a`；仅UI候选/回退覆盖已准备，回退准确指向当前a2d/857f，不指向更早版本。新GitHub CI尚无已观察run，状态UNKNOWN。

此文案源码交付时尚未更新实例，当时运行仍为下文 `a2d` / `857f7ce8…` / `cae36d3…` healthy。文案现已随本页首节e75镜像更新，并完成两个首页只读核对；原opt-in合成浏览器用例、移动端及用户验收仍NOT_RUN。后续来源规划、SHEIN优先和美国选择沿当前产品决定推进；多平台正式实现仍须明确来源范围和适用设计依据，不能由删文案或放宽校验代替。

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
