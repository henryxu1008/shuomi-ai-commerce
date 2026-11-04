# Local Account Center：1e3509 统一模型配置差异

状态：**REVIEWABLE_DRAFT / NOT_PUBLISHABLE**。本轮只读核对及文档准备，不是实际配置文件、安装回执或发布授权。
应用固定为 `1e3509bda06ad9b8160b0c5a7c5d451de9496c61`，tree `074f19ab3dd06a8f9552ba73d13b344055913b6c`。
原独立增量复核与 [CI 41765722589](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41765722589) 已通过；本轮文档提交不改变应用来源，也不把原 CI 冒称新文档 SHA 的直接执行结果。

## 本次交付与依据

使用者为 Local Account Center 配置管理员及指定实例操作者。目标是准备一份符合当前统一 owner 的关闭态运行配置，
使管理员后续可以查看配置、业务人员可以读取原历史；研究/选品和图片新执行继续关闭。
本轮 Design Basis: **N/A（只读现状映射及文档）**，复用已准入的[生命周期合同](../architecture/ai-disabled-application-lifecycle.md)、
[统一模型合同](../architecture/enterprise-shared-model-configuration.md)和[两项文字合同](../architecture/bailian-deepseek-shared-text-admission.md)。
不改变产品、权限、状态、事实 owner 或恢复语义。原工作区、唯一 Writer、PR #1075 保持。

本轮使用原浏览器登录身份，以及原 Run reader / writer 身份完成只读元数据查询。
原凭据仅在本机认证进程内复用，未展示、另存或改写；没有读取 Key 表数据、worker 私密文件、任务载荷、商品、余额或模型响应。
数据库检查使用 `BEGIN READ ONLY`、5秒语句/1秒锁超时，最终 `ROLLBACK`；没有调用发布、安装、恢复或执行入口。
不做旧兼容、迁移、双读/双写、旧 ID 映射或第二 current；不改凭据、不发布配置、不构建/更新实例、不真实采集/AI/付费、不合并。

## 当前基线与目标差异

2026-11-04 本轮 fresh 核对：原身份/后端/UI 均 running/healthy，前后端仍为 a66；当前 manifest SHA256 仍为
`1271E0D89BC8E8F8DE5DF92FFCE94D9B9032291C9D7EC04FEA8FCC6AFB195B97`，与已批准非秘密结构记录对应。
字段来自原 manifest 的白名单投影，凭据值不进入投影或仓库。旧文档中的“尚未实现/待最终复核”已被 1e3509 的交付证据替代。

| 配置位置 | 原 a66 已核结构 | 1e3509 目标与边界 |
| --- | --- | --- |
| `modelRoutePricing` | `imageWriter` + 4 条 image binding，无统一 `writer` | 新私密 manifest 只使用当前 `writer` / `bindings`；不是机械改名，不能借原 image DB writer 充当 Run writer |
| `modelRoutePricing.writer` | 原 `imageWriter` 目标为 `image_agent`；同一原身份已成功只读连接 Run DB | 新 writer 目标为 `127.0.0.1:5433/product_agent`，user=`model_route_config_writer`，pool 保留1（合同上限2）；原凭据无需变更。不是把旧 Image current 搬到 Run DB |
| `productAgent.database` | `127.0.0.1:5433/product_agent`，`product_agent_runtime`，pool4 | 保持该当前 Run/config owner 和原读取身份；不换数据库承接旧业务、不用 Image 或 Product/Review pool 代替 |
| `productAgent.enabled` | false | 保持 false；不为配置管理设 true。不开文字 executor/Review/Asset 执行池 |
| `productAgent.contentOnly` / `contentTitleOnly` | true / false | 若声明研究/选品，必须改为 false / false；保持 enabled=false。原内容声明与共享 Limits 的处理见下文，不能只加两个 policy |
| 两项文字定义 | 部署中无 research/selection policies；两条 LAC DeepSeek 凭据元数据已核 | 按下表及已核版本准备原用途 anchors 和候选描述。声明合法只允许管理，不自动获得执行资格 |
| `modelRoutePricing.bindings` | 原 4 条 image 声明 | 逐条按当前 Definition/ImageAdmission 合同核对后明确纳入；不搬旧 current/revision、不解释旧价格字段、不悄悄清空目录绕过校验。未核项保留为待补，不标 Ready |
| `imageAgent.generation` | 只有 `priceVersion` / `pointsPerImage`，无当前 enabled | 新 manifest 显式 `enabled=false`；使用当前 `unitVersion` / `maximumMicropoints`，旧两字段不进入新文件。图片预算上限须由原 owner 给出现行批准值，不从旧点数自动换算或套用研究/选品 20 点 |
| 完整图片模式 | worker 文件引用存在 | 保持非空 `workerConfigFile` 和原 API/Asset/Product/config owner，以显式完整模式保留历史；关闭态不读该文件、不打开 workerDB、不连接图片 Temporal、不装配/启动图片执行 worker |
| 图片/Asset/审计读取 | 原 manifest/挂载仍在，实际 schema/grant 未核 | 保留 `image_agent_runtime`、`supply_asset_runtime` 及原 `accountAuditUsage` 读取分工；完整模式独立 image audit reader，文字执行关闭时相应 product audit reader继续按原合同。不能用 writer 或管理员账号替代 |
| Chat / 其他执行者 | Chat enabled=false；其余范围见本轮启动清单 | Chat 保持关闭。DataServices 的存在会启动采集 worker，Knowledge 与恢复循环也独立于两个 AI 开关；不能宣称全系统停止执行 |
| 平台参考价、企业三类选择、操作模式 | 已读 current 的版本元数据，未改写；共享选择及参考价未发布 | 与运行 manifest 分开。平台价格和企业完整三类选择沿原 owner/CAS 发布；本次不发布，不恢复旧 image current，不用残缺选择快照替代完整 CAS |

上述是字段级目标，不是可直接复制的 manifest。没有用假密码、`<组织ID>`、虚构 revision 或不存在的 passwordFile/env 插值功能拼出“可运行配置”。
后续实际私密文件另建，原 a66 文件不覆盖；凭据仍由原受控 owner 提供。如果现有环境没有合法 writer 身份，不能靠本次“不改凭据”授权创建账号或设密码。

## 两项文字定义与预算保持

| 项目 | 项目资料分析 | 独立商品对比 |
| --- | --- | --- |
| Agent / capability | `product.research.agent` / `product.research.decision` | `product.selection.agent` / `product.selection.decision` |
| runtime anchor | `productAgent.researchTextPolicies`，并保留 ProductCollections + MarketResearch 消费条件 | `productAgent.selectionTextPolicies`，并保留 ProductCollections 消费条件 |
| prompt / output schema | `market-research-prompt-v1` / `market-research-action-v1` | `product-selection-prompt-v1` / `product-selection-action-v1` |
| 准确模型 | DASHSCOPE / `deepseek-v4.1-flash`，openai-compatible | 同左 |
| 固定调用策略 | `dashscope-deepseek-v4.1-flash-nonthinking-v1`；非思考、非流式，无模型工具/联网/多模态 | 同左 |
| 每项预算 | ¥0.30 / 20 AI 点 | ¥0.30 / 20 AI 点 |
| 调用/Token | 最多2调用，每次≤32000 prompt / 4000 completion；原更小限制优先 | 同左 |
| 单定义限制 | Definition `maximumMicropoints=10000000`（每调用预算）；成本 ceiling≤150000 微元 | 同左 |
| 原 Run 限制 | CNY、CostMicros=300000、最多2调用，总 Token≤72000且不放宽原更小值；原 Steps/byte/deadline/runtime≤120秒不提高 | 同一原 Run 限制 |

合计各执行一次仍 ¥0.60 /40点。候选参考价仍输入 ¥2/百万 Token、输出 ¥8/百万 Token，不用缓存优惠；
原策略 `reference-text-totals-v1`、CNY 恒等 FX，最大两调用参考估算 ¥0.192/19.2点。它是费用估算，不追逐实时精确价或厂商账单硬保证。
原 ceil R/floor A、显式完整零用量、缺基础用量/矛盾/超限 UNKNOWN 及原键不重发均不变。

上述 Run 数值仅说明已批准的两项方案。原 ProductAgent 共享 Limits 实际为 CNY、12步、2调用、139264 Token、
1000000微元、120秒；原内容 Profile 为65536 prompt /4096 completion、45秒，含当前 ModelProfile 已不接受的 `PointTariff` 字段。
因此“增加两项 policy”不是完整差异：研究/选品要求的300000微元、≤72000 Token与原共享上限不同，
原内容声明也须按当前合同重新声明。不能默改内容用途预算、复制旧 Profile，或清空目录来掩盖冲突；本轮仅列出这个待审范围，不写真实配置。

### 本轮 LAC 绑定及配置版本

原浏览器会话的有效企业为 **Local Account Center / `394513833252618242`**，原角色包含 `listingkit_admin` / `platform_admin`，未切换或增加角色。
原 Run reader 只查询 `ai_client_credential_bindings` 的该企业、空 user_id、指定 DeepSeek 型号；`key_present` 只是布尔标记，不证明远端可调用。

| 原 ClientName | 型号 / 状态 | CredentialVersion |
| --- | --- | --- |
| `local-content-route-830-20261020`（沿原候选） | `deepseek-v4.1-flash`，enabled=true、key_present=true | `org-credential:v1:9b2ab8feb2e44d7399f9649a8debe471355bbb0388a5a06b022d507ce7907737` |
| `local-content-title-trial-20261016` | 同型号、同状态；不自行替换原候选 | `org-credential:v1:d27ce7c4d97a4a4613b61a1fcb8909d398998ced1db5f699a8da8c3717606ddd` |

两条均 openai-compatible、timeout45秒；EndpointIdentityDigest 同为
`0ec872526f86b31a2cfa18b91270cde45e9c4929123926de91819e677b3e2797`。
版本按仓库 `CredentialVersion` 规则，以元数据和精确纳秒时间计算，不含 Key。两用途的模型 deadline 须≤45秒，Run 原120秒不提高。

Run DB 的 LAC 内容 current 为 `route-af239e9c-04e4-4846-bb1f-31a07365faa7`；
研究、选品、`ai.models.enterprise` current 均不存在；平台 `ai.models.official-prices` current 也不存在。
原 a66 图片页面仍为旧图片配置界面，当前图片 revision 为 `route-dc1c8e3f-bd29-48bf-87d6-37452dbe995c`。
这些版本属于不同 owner/用途，不迁移、不互作替代；缺少 current 是“尚未发布”，不是授权本轮补写。

### 文件编码必须遵循当前合同

- `generation.maximumMicropoints` 与 `bindings[].image.maximumMicropoints` 是十进制 **JSON 字符串**，unitVersion=`ai-micropoint-v1`；非零声明须同时有合法单位与正上限。
- `bindings[].maximumMicropoints`（文字每调用预算）是 **JSON 整数**；它与 Generation 的 wire 格式不同，不可混用。
- Definition 的 scope 使用 `organizationId` / `capability`；文字 `text.Profile`、`text.AdmittedCredentialVersion`、`text.AdmittedEndpointIdentityDigest` 沿当前 Go JSON 结构。
- Profile 的 `DeadlineBound` 沿原 duration 编码和已批准更小值，不凭空写字符串 `120s`；不伪造 BoundEvidenceVersion、Tariff/FX/current revision 来通过校验。
- 仅内容模式仍只可声明内容 policies；关闭态也不能混入标题、研究或选品等其他用途。两项研究/选品不放在 ContentOnly/ContentTitleOnly 安装模式中。

## 启动所需资源与开放所需资格分开

| 缺项 | 影响层级 | 当前证据 / 处理 |
| --- | --- | --- |
| 同 Run DB 的合法 writer 及连接额度 | 新版本启动 | 原身份只读登录成功，current_user=session_user=`model_route_config_writer`，无super/createDB/createRole/replication/bypassRLS，connlimit=4；快照原writer连接1条。目标pool1无需调整角色或凭据 |
| 原 Run reader / writer 配置schema与grant | 配置管理启动 | revisions/current/元数据view、4个作用域/CAS约束及窄权限与 `VerifyReader/VerifyWriter(false)` 的只读检查项相符；reader无配置写权，writer仅revisions INSERT、current INSERT/UPDATE，无Key读取/凭据写入/public CREATE。未实际运行应用Verify或尝试写入 |
| `ai_route_config_options` | 关闭态统一配置管理 | Run库不存在；1e明确使用 requireOptions=false，不是该路径启动缺项，不能据此安装旧Image options表 |
| `ai_invocations` | 后续真实文字执行 | 当前53列/目标52列：缺 `official_tariff_json`、`billable_usage_json`；多旧 `point_price_version`、input/output points列。不符合1e严格执行合同；enabled=false不装配文字执行器，因此不把它误判成关闭态管理启动阻碍。本轮不迁移/安装/改库 |
| 原完整图片/Asset/配置/审计读取资格 | 原历史与启动 | UNKNOWN；保留原owner和历史，按原Verify检查。缺schema/权限则停止更新，不初始化/迁移/清库，不更换owner以丢掉历史 |
| 准确企业/两项文字定义元数据 | 声明这些候选、后续配置选择 | 企业/原Client/凭据版本/端点/超时已核，部署中两用途policy未声明；仍须形成合乎当前合同的完整定义，处理contentOnly/旧内容字段/共享Limits，不用不合法policy换取页面可开 |
| image credential metadata | 图片技术Ready | 关闭态没有worker credential reader，沿原CREDENTIAL_READER_MISSING；不是为了让所有选项Ready而启worker的理由 |
| 参考价/完整共享选择/操作模式版本 | 后续真实研究与选品 | 本次不发布；未就绪只如实显示。配置管理启动不等于模型可执行，技术Ready也不等于真实调用授权 |
| 其他worker/恢复入口、真实队列影响 | 更新前外部副作用边界 | 已核如下启动清单；DataServices/知识处理/恢复循环并未随AI关闭。运行范围尚须在更新方案中明确接受或调整，不能悄悄停服务或扩开发范围 |

初次连接检查受本机 Windows CRLF 输入影响而认证失败；改为LF后，同一原reader和writer均成功，未更换身份或凭据。
失败记录保留，但不再作为“凭据失效”结论。以上是只读元数据匹配，不能冒称完整启动、现场历史读取或产品验收通过。

### 其他 worker 与任务队列的实际范围

源码依据固定1e的 [currentapplication runtime](../../internal/app/runtime/currentapplication/runtime.go)、
[HTTP装配](../../internal/app/httpapi/current_application.go)及各现有owner；未新增开关、调度器或第二执行owner。
2026-11-04 04:55–04:57 UTC仅对原Temporal `default` 执行 task-queue describe，没有poll、启动workflow、读取历史/载荷或执行恢复。

| 执行者 | 原配置 / 启动条件 | 本轮证据与1e影响 |
| --- | --- | --- |
| DataServices采集 | 配置存在即装配/Start，无独立enabled；Amazon14站点 | `data-services-v1`有workflow/activity poller，近似backlog各0。1e保持此配置仍会启动，两个AI false不控制它；移除DataServices又会破坏当前市场调研依赖，不能当作无影响配置修补 |
| 完整图片 | 原worker文件引用存在，a66在消费 | `image-agent-organization-v1`有workflow/activity poller，近似backlog各0。1e显式generation=false后仅不装配本应用图片worker；不控制其他进程，也不从本次快照推断所有workflow已结束 |
| 店铺观察 | observations存在，preparationOnly=true | `store-observations-current`有workflow/activity poller，近似backlog各0。preparationOnly仍装配worker；现合同要求空观察库且不提供official访问，不等于worker关闭。本轮不读取/清空该业务库 |
| POD / Supply / Fulfillment | 三项配置均不存在 | 本应用不装配对应worker；不宣称其他进程不存在 |
| Knowledge | enabled=true | 独立5秒Sweep、并发2、对象读取/文档解析；不受AI开关控制。不触发Sweep或查询业务队列 |
| Ecoservices / 商业资源 / consumer恢复 / 推荐成熟 | 原Ecoservices启用，Commercial/Money owner存在，Referrals启用 | 固定1e仍装配Ecoservices15秒、资源购买/consumer30秒、推荐成熟1小时循环，启动会立即运行；非Temporal队列。本轮不触发恢复、不查订单/余额/待结算记录 |
| 充值 / Chat / BrowserCollector | walletTopUp缺省（两渠道false）；Chat enabled=false；采集桥配置仍存在 | 不能把“无新支付渠道”写成“所有恢复停止”；充值恢复回调仍可装配。原独立browser-collector容器保持退出，不启动、不控制既有桥会话 |

这些队列0积压只是一个近似快照，不证明无运行中的workflow、未来任务或恢复副作用。
启动所消费的非AI现有任务范围须审查；本轮不停止原a66，也不通过重新启动消费者来验证边界。

## 未来配置交付清单

1. 复用本轮已核writer/LAC版本；仅补未核的图片现行声明/读取资格和现有执行者处理范围。先审查旧内容声明与共享Limits的具体差异，不再研究精确厂商价格，不申请新凭据或角色。
2. 形成独立的新私密 manifest 及去密 diff：只包含明确批准的字段改动，productAgent=false、generation=false；原 a66 文件和哈希不变。现在没有生成或写入该真实文件。
3. 在明确获准的离线/元数据检查范围完成当前严格字段和资源校验；不以在 a66 上尝试启动或请求模型来“验证配置”。
4. 通过后才进入[两服务更新方案](market-research-unified-update-plan.md)。模型/参考价发布、构建、实例更新、真实AI分别单独授权，不能因批准本文一并执行。

用户业务链仍是智能市场启用/配置 → 同项目选择2–5项资料 → 比较/费用估算 → 明确确认只读分析 → 人工决定 → 准确子集交接。
本轮不操作这条真实业务链；1e3509已通过的合成回归与真实采集、远端模型、现场历史读取及用户验收区分，后者均NOT_RUN。
