# Local Account Center 智能选品：共享文字模型接线与差异预览

日期：2026-11-03（Asia/Singapore）。配置预览状态：**REVIEW_ONLY / NOT_PUBLISHABLE**；真实配置与调用 **NOT_RUN**。

## 最新决定：全新系统，不承接旧数据（2026-11-03）

用户在同意限定设计后进一步明确“旧的不用保留”。因此，下方保留旧配置/历史/余额/预留及旧53列演进的方案**撤回，不实施**；不再作为DeepSeek新系统接线前置，也不记为完成。当前只补[DeepSeek共享文字准入设计](../architecture/bailian-deepseek-shared-text-admission.md)，沿原owner新空安装/当前模型合同，不设计迁移、兼容或旧账本续跑。原系统基线本就允许新系统自身正常Schema演进，此次无需恢复旧系统迁移项目。

现有记录/公开原价调查仅作准入候选证据；新空目标的组织凭据和共享current须原owner合法准备，不复制旧Key、旧配置或历史事实。现有统一实例不更新、不清空、不重建，实际资源处置须先有明确清单及单独授权。预算不变，当前设计/评审不授权生产改码、配置发布、真实调用或合并。以下保留实例方案按其当时范围保留，本节为最新Scope。

新准入设计已完成R1及三项合同澄清增量复核，无设计BLOCKER；状态DESIGN_REVIEW_PASS / OWNER_CONTRACT_PENDING / NOT_IMPLEMENTATION_READY。实现测试尚未运行，真实usage/计价时点资格未验证，不将设计通过写成可调用。

## 共享准入核实与保留统一实例方案（2026-11-03，13:08 UTC）

本轮用户授权只读核实、非执行差异和实施方案，Design Basis: **N/A**。没有正式业务代码修改。本节更新下方“原版厂商价格身份未对应”的早期结论：已找到原版官方发布价表；真实绑定计量、公共价目准入及保留数据的切换仍未完成，不签发IMPLEMENTATION_READY或可发布配置。

### 现有凭据和共享准入

再次使用原 `product_agent_runtime`、`product_agent` 库及READ ONLY事务，固定LAC组织级筛选。两条记录、准确凭据版本和端点摘要均与前次一致；首选审查仍为 `local-content-route-830-20261020`。SQL仅在服务端核对官方workspace端点格式，返回 `cn-beijing-workspace-compatible`，不返回原URL/Key。原内容current仍为 `route-af239e9c-04e4-4846-bb1f-31a07365faa7`；没有本企业共享/研究/选品current，也没有 `platform-ai-models / ai.models.official-prices` current。没有GRANT、SET ROLE、管理员连接或provider请求。

| 准入条件 | 当前证据 | 结论与最小后续 |
| --- | --- | --- |
| 凭据事实 | 原型号、OpenAI兼容、enabled/key_present、45秒及北京端点格式已核 | 可作为同企业去密候选；Key远端有效性/余额不由存在标记证明 |
| 受控请求 | 原固定非思考策略准确限定DASHSCOPE / deepseek-v4.1-flash；原受控组件非流式、无模型工具 | 继续纯文本和各用途原prompt/schema/权限；不增加搜索、视觉或修复调用 |
| 共享descriptor | `ReferenceVersion`、`BoundEvidenceVersion`、正式`PriceID`未取得 | 由原Catalog提供，绑定同一credentialVersion/endpoint摘要/地域/协议/参数/成本证据；不以凭据版本或旧revision代替 |
| 已开放用途 | 原SharedCatalog对企业全部已开放文本scope核同一候选，缺项返回 `OPEN_CONSUMER_MISSING` | 研究/选品保留各自较小边界；原内容等已开放用途也需兼容，不能只补两项就宣布共享text合格 |
| 厂商来源 | 原 `officialPriceHost` 只支持openai/google/qwen/cfets，没有deepseek；#1078当前已提交aac555f3该字段规则相同 | **正式价格发布阻碍**：原公共owner须先批准并最小补齐DeepSeek官方来源/准确身份，不伪装成qwen或渠道零售价 |
| 发布原子性 | 原SharedModelService要求text/image-generation/multimodal三类全部ready | 本轮两项text准备不授权text单独发布；#1078的Flare/完整视觉链仍未交付完整三类开放 |

依据：[原SharedCatalog](../../internal/integration/aicapability/routeconfig/shared_catalog.go)、[原共享发布](../../internal/aicapability/routeconfig/shared.go)、[原价目owner](../../internal/aicapability/routeconfig/platform_prices.go)、[固定策略](../../internal/integration/aicapability/einomodel/deepseek.go)。#1078当前aac555f3的C2-2在途实现只作依赖现状，不合并或接管；本分支已批准的341278d5有限基础保持其原范围。

### 原版准确价目与预算

价格均为人民币/百万Token；这里只保存公开原价证据，没有PriceID、发布版本、生效窗口或API载荷。

| 口径 | 闲时：非缓存输入 / 缓存输入 / 输出 | 忙时：非缓存输入 / 缓存输入 / 输出 | 用途 |
| --- | --- | --- | --- |
| DeepSeek原版V4.1 Flash厂商官网 | 1 / 0.02 / 4 | 2 / 0.04 / 8 | 用户消费点数的原价依据 |
| 百炼原型号、北京地域 | 1 / 0.1 / 4 | 2 / 0.2 / 8 | 独立渠道成本上限，不叠加用户收费 |

原版来源为[2026-09-10官方发布](https://api-docs.deepseek.com/zh-cn/news/news260910/)中的[原始价格图](https://api-docs.deepseek.com/zh-cn/img/v4.1_260910_price_cn.jpeg)，已实际查看；[官方更新日志2026-10-17](https://api-docs.deepseek.com/zh-cn/updates/)说明升级1017时调用方式和价格未变，与[当前厂商价表](https://api-docs.deepseek.com/zh-cn/quick_start/pricing/)相符。这补足原版公开价格依据，但不证明百炼当前实际权重等于1017，原凭据型号保持不变。百炼来源为[原型号详情](https://help.aliyun.com/zh/model-studio/deepseek-v4-1-flash)，本地技能索引 `models/groups/deepseek.json` 与 `raw/model-user-guide/support/model-studio-model-list/model-list-text-generation/deepseek-v4-1-flash.md` 相符。

厂商高峰为东八区非中国法定节假日的周一至周五09–12、14–18，其余闲时；百炼闲时为每日22–次日08，其余忙时。两种日历独立，不能用一个忙闲标记同时定成本和零售。人民币沿原owner的 `cny-identity-v1`，不新增外汇条目。

两调用、每调用≤32000输入/4000输出、无缓存的厂商高峰算例：`2 × (32000 × 2 + 4000 × 8) / 1000000 = ¥0.192 = 19.2点`；闲时为¥0.096/9.6点。百炼北京按其忙时上界同为¥0.192成本；两个操作各一次最坏算例共¥0.384/38.4点，仍在已确认每项¥0.30/20点、总¥0.60/40点内。实际序列化输入、原更小Token/字节/期限及费用上限继续收紧，不扩额度。

原价格owner已有 `EffectiveFrom/EffectiveUntil` 和每次Resolve，但没有周期日历/节假日合同。后续先由公共owner确定如何复用连续有效窗口表达准确价目及跨窗口发送前检查；不能把忙时价整天用于实际结算，也不擅自引入定时器、节假日平台或另一个收费引擎。19.2点是保守上界算例，不是固定扣费。

### 完整用量逐字段映射

| 百炼OpenAI Chat字段 | 原mapper/计费含义 | 准入条件 |
| --- | --- | --- |
| `usage.prompt_tokens = P` | 完整输入，含缓存；InputText=P | 必须实际存在且非负，不靠SDK默认零值 |
| `usage.prompt_tokens_details.cached_tokens = C` | CachedInputText=C；非缓存输入=P−C | 必须存在且0≤C≤P；缺失/null不当0，不把全部输入按非缓存收费 |
| `usage.completion_tokens = O` | OutputText=O | 必须存在、非负；回答和思考已含在其中，不重复加算 |
| `usage.total_tokens = T` | 一致性校验，不作为第四项收费 | 必须存在且T=P+O |
| `completion_tokens_details.reasoning_tokens` | 可选输出子集 | 有值时0≤reasoning≤O；本候选仍强制非思考 |
| 搜索、显式缓存创建、图像/音频等其他收费项 | 本轮纯文本合同没有这些收费维度 | 不申请这些能力；若准确响应或正式合同出现额外收费维度，不能以三项完整为由忽略，须原owner明确拒绝或补齐后再准入 |

当前mapper为[ObserveOfficialOpenAITextUsage](../../internal/integration/aicapability/einomodel/official_usage.go)：保留字段存在性、拒重复/矛盾/越界，用 `openai-text-v1`；旧profile的token观测标签不是其替代。计费沿[原公式](../../internal/aicapability/official_model_pricing.go)：按P−C、C、O分别乘适用原价，预留ceil、完整实际floor；缺完整usage保留原预留/UNKNOWN和原key，不改价、换Key或重发。

[百炼通用Chat参考](https://help.aliyun.com/zh/model-studio/qwen-api-via-openai-chat-completions)及[缓存说明](https://help.aliyun.com/zh/model-studio/context-cache)能证明通用字段语义；但通用例子是Qwen，并非本候选响应。[百炼DeepSeek参考](https://help.aliyun.com/zh/model-studio/deepseek-api)的V4 Pro示例甚至出现 `prompt_tokens_details=None` 或未列缓存字段，不能据此证明V4.1 Flash缓存为零或始终有完整分项。现有53列Invocation也没有完整分项事实。本轮不读取历史提示词/响应、不调用模型，故本凭据的真实完整usage资格仍 **NOT_VERIFIED**。最小解除证据是原owner可追溯的准确型号/北京端点/固定策略非流式响应合同，覆盖缓存命中与未命中的存在性及所有收费维度；无此证据继续关闭。

### 保留task-processor-unified-20261010的实施顺序与停止点

现场再次确认18个原Compose服务、55个带项目标签的卷、52个已挂载不同命名卷（集合可不同，实际以挂载清单为准）。后端/UI仍a66d7d0c、原5cde984e/25ae8979容器及2848da26/d106b5df镜像；业务PG仍72359f29，数据挂载仍 `task-processor-unified-20261010-business-db → /var/lib/postgresql/data`。仅保存镜像/标签/挂载元数据，没有启动、停止、复制卷或备份恢复执行。配置内容/数据内容和财务余额未核验，不以清单冒称恢复验收通过。

| 顺序 | 具体交付 | 配置、数据和卷的处理 | 停止条件 |
| --- | --- | --- | --- |
| 1，本轮已完成 | 既有凭据核实、原版公开价目、字段映射、原挂载清单及本方案 | 原配置/凭据/数据/卷均不写入，AI仍关闭 | REVIEW_ONLY / NOT_PUBLISHABLE |
| 2，后续代码范围须另行明确 | 原公共owner最小补齐DeepSeek来源/价目有效窗口，取得真实协议/完整usage和BoundEvidence；保持各已开放文本scope兼容 | 仍只在原开发分支离线验证；不复制Key、不改现场current，不接管#1078在途视觉开发 | 相关Design Basis及公共owner批准、稳定接口/CI证据；任一缺失不进入配置发布 |
| 3，保留数据的设计决定 | 明确要保留的是原配置、历史、余额、在途/UNKNOWN及其找回语义；原owner提供同一事实源的有界演进设计 | 当前只读实证仍为旧53列，目标52列；资源库的实际单位本轮未查，保持UNKNOWN。不得把原表改几个字段或直接乘百万当方案 | **当前没有原地演进准入**。若要求保留数据且在同一实例继续新版调用，须新的产品决定及Independent Architecture达到IMPLEMENTATION_READY；本方案不是迁移授权 |
| 4，设计批准后才能准备 | 复用既有私密备份/恢复能力，保存原配置基线、准确镜像、全部相关owner数据库/卷及回退依据，在获准副本验证原历史/余额/预留/原key不丢失不重复 | 不删除原卷、不覆盖原配置；不得只备份Invocation而遗漏Run、原资源、图片及配置事实。不能只换Agent库再让新消费者写旧bucket，也不能清空/改名旧表伪装全新安装 | 完整受影响owner一致性、原预留/UNKNOWN处置及数据处置设计未就绪则停止；副本验证不是替换用户交付实例 |
| 5，另行授权实际更新 | 冻结准确源码/镜像及经审查必要配置差异；用仓库标准构建入口，随后仅更新现有项目的后端/UI | 沿原项目、网络、身份和全部原挂载，其他服务不随依赖重建；凭据保持原记录，不自动seed模型或费率 | 用户明确更新授权、保留数据方案及应用启动验证均满足才执行；本轮NOT_RUN |
| 6，另行授权配置与试跑 | 原页面正常权限维护官网价、原三类共享选择CAS；保持两项read_only和原预算，先只读核对，再申请限定真实调用 | 不自行提权、不借旧内容current或复制Key；三类任一不合格不部分发布，真实采集/AI/付费仍需各自明确授权 | 当前全部NOT_RUN；保存成功不等于真实模型/业务验收 |

关键冲突依据：[冻结设计§5.2–6](../architecture/enterprise-shared-model-configuration.md#52-新-schema-的最小增量与安装顺序候选)、[新系统产品基线](../product/greenfield-no-legacy-migration.md)、[原Invocation启动校验](../../internal/aicapability/store/model_point_fact.go)、[运行说明](../operations/model-route-pricing.md)。当前批准合同只支持新空库，现有53列缺 `official_tariff_json/billable_usage_json` 已直接不兼容；“原样保留”不自动授权改变旧量纲或历史事实。若不作新的产品决定，方案必须停在第2步及离线准备，不能承诺直接覆盖镜像后可用。

本轮发现按当前Must分类，均不阻止本次只读交付：DeepSeek来源未受理为 **BLOCKER / 后续正式配置发布**（准确官网价的核心路径无法完成）；本绑定完整usage缺证为 **BLOCKER / 后续真实收费开放**（不能按完整用量正确收费）；保留53列数据而启动52列消费者为 **BLOCKER / 后续实例切换**（当前rollout无法安全完成）。周期窗口、缓存/无缓存/额外收费维度和跨边界断言属于原owner后续 **IMPLEMENTATION_TEST**，不因理论场景另建平台。以上分类不授权本轮改代码、补造资格或接受风险。

回退也须尊重事实：未产生新写入时可以按获准方案恢复准确原镜像/配置；一旦已有新格式/余额/预留写入，不得简单回退旧镜像或覆盖旧备份丢弃新事实，先关闭新执行，由原owner按已批准恢复安排处理。本轮不设计或实现第二账本、双读写、fallback、通用迁移器或新的恢复机制。

本地证据 `.local/selection-bailian-admission-plan-20261103-01/` 被Git忽略，含去密复核、字段白名单挂载清单、原公开价格图和非执行摘要。首次清单采集曾在进程内接收完整inspect对象再过滤；未输出/保存其环境变量或凭据，后续已改为inspect字段白名单，保存文件仅含安全元数据。应用测试、备份恢复、安装/演进、配置发布、实例更新及真实调用均 **NOT_RUN**。

## 已取得的受限元数据与候选差异（2026-11-03，12:46 UTC）

**本企业组织级凭据记录已确认；新版绑定准入和运行条件仍未完成。** 用户“继续”沿只读范围核实。Design Basis: N/A，只读现有去密投影、配置元数据及表结构，更新本预览；没有凭据、权限、配置、schema或实例写入。

### 读取范围与权限证据

正常页面再次确认当前企业LAC `394513833252618242`、原平台维护身份。实际运行库名称 `product_agent` 来自原非秘密备份回执；使用既有 `product_agent_runtime`，`session_user=current_user`，superuser/create_role/create_db/bypass_rls/replication均false。该账户已经拥有去密视图及原配置表SELECT，本轮没有GRANT/SET ROLE或管理员连接。每次事务均 `BEGIN READ ONLY`，后续读取另加本地3秒statement timeout。

查询固定 `tenant_id='394513833252618242' AND user_id=''`，仅筛选原型号和1017；只返回允许的元数据、端点摘要/主机格式推导的地域提示及Key存在布尔值。视图列不含api_key，不返回完整端点或Key值。首次误用了测试夹具角色agent_runtime（现场不存在）；正式运行账户访问postgres系统库CONNECT被拒绝后未改授权，依据非秘密回执只连接其自身运行库。以上失败和后续成功分别保留。

### 已有记录与来源

| 组织级记录 | 型号/协议 | 状态 | 地域提示与超时 | 凭据版本 |
| --- | --- | --- | --- | --- |
| `local-content-route-830-20261020` | `deepseek-v4.1-flash` / `openai-compatible` | enabled=true，key_present=true | cn-beijing；45秒 | `org-credential:v1:9b2ab8feb2e44d7399f9649a8debe471355bbb0388a5a06b022d507ce7907737` |
| `local-content-title-trial-20261016` | 同上 | enabled=true，key_present=true | cn-beijing；45秒 | `org-credential:v1:d27ce7c4d97a4a4613b61a1fcb8909d398998ced1db5f699a8da8c3717606ddd` |

两条端点摘要同为 `0ec872526f86b31a2cfa18b91270cde45e9c4929123926de91819e677b3e2797`；没有推断二者使用相同Key。版本按原owner公开的非秘密公式计算，首条版本另与原内容current所存版本一致。该范围内没有1017记录。enabled/key_present只证明本地记录状态，远端有效性、额度及实际用量未测试。

只读current查询发现原内容用途 `product.content.decision` 的 `route-af239e9c-04e4-4846-bb1f-31a07365faa7` 正引用首条记录；没有本企业研究、选品或 `ai.models.enterprise` current行。该原内容profile的provider为DASHSCOPE，adapter为openai-compatible，固定非思考策略已是 `dashscope-deepseek-v4.1-flash-nonthinking-v1`；prompt/schema仍是内容用途自身版本，不能直接用作研究/选品。较早试用记录保留为来源，不作失败fallback。

### 本次已可填写与仍须核准的差异

| 边界 | 已核实来源 | 非执行候选/剩余条件 |
| --- | --- | --- |
| 凭据候选 | 首条是本企业组织级记录，且被原内容current引用 | 优先审查复用 `local-content-route-830-20261020`；仅引用现有记录，不创建、复制或改写Key |
| 凭据版本/端点 | 已有准确版本和摘要；北京主机格式提示 | 使用上述版本/摘要；地域和服务适用范围仍须原目录正式核准 |
| 协议/模式 | DASHSCOPE、OpenAI兼容、原固定非思考策略 | 两项继续使用该固定策略及各自准确prompt/schema/read_only/权限，不复制内容配置 |
| 用量/字节/期限 | 原内容profile上界65536/4096Token、输入65536字节、输出16384字节、deadline45秒 | 两项仍≤32000/4000Token、最多两调用，并保留各业务原更小字节/期限/费用约束；不把凭据超时自动设成操作期限 |
| 共享绑定reference | 只有原内容revision和凭据版本；无新共享current | 新ReferenceVersion、BoundEvidenceVersion、Price ID仍待原owner资格；不用旧revision或credentialVersion顶替。正式发布前须由该owner重读CAS，不生成发布请求 |
| 厂商价目 | 模型元数据仍为用户原名称；没有1017记录 | 原名称保持，厂商当前1017价格的准确适用身份仍未对应；下节公开价目只能作条件参考 |
| 计费资格 | 原内容UsageMappingVersion为 `openai-compatible-token-usage-v1` | 原token观测标记不等于新版 `openai-text-v1` 完整分项已验证；不能只改版本字符串或用历史总Token补造缓存用量 |

### 新确认的运行阻碍

现场 `public.ai_invocations` 有 **53列**，含 `point_price_version`，没有 `official_tariff_json`、`billable_usage_json`。仓库当前唯一owner模型为 **52列**，[启动校验](../../internal/aicapability/store/model_point_fact.go)要求准确列集合并拒绝旧结构。这已是直接元数据证据，不再只是旧/新单位不明；本轮未运行安装器、ALTER/迁移、重算消费或新增第二账本。

因此，当前可推进的是以首条现有记录形成准确候选差异。正式运行还需要：新共享目录reference/边界证据、准确型号的官网价适用性、完整可信usage资格，以及另行确定现有实例/数据保存与新合同的适用方案。旧内容选择仅作来源证据，不转成新current或回退源。预算保持每项¥0.30成本/20点、两项各一次总¥0.60/40点；没有因找到凭据而获得发布、更新实例或调用授权。

新证据在被Git忽略的 `.local/selection-bailian-owner-metadata-readonly-20261103-01/`，不是配置API载荷。以下12:12 UTC页面核查记录保留其当时的UNKNOWN；本节补足了凭据存在性、版本和地域提示，不把它们继续写成未知。

## 本次现场核查与配置差异（2026-11-03，只读）

Design Basis: **N/A**。本轮仅核实正常管理页、既有去敏GET、镜像来源标签、官方文档及原代码映射，更新本预览；没有修改业务代码、凭据、配置、schema或实例。本节保留2026-11-03 12:12:52 UTC（东八区20:12:52）的页面核查快照；其中凭据版本、端点摘要和地域的UNKNOWN已由上方12:46 UTC受限元数据证据补足，其余未完成条件继续有效。

### 现场事实与边界

当前企业为Local Account Center `394513833252618242`，正常登录维护身份为 `394513839896395778`。管理页只有图片用途，已发布的是原图片配置；这不能证明文字凭据不存在，也不能用图片Key替代文字绑定。

| 已有只读入口 | 实际结果 | 能证明什么 |
| --- | --- | --- |
| `GET /api/model-route-pricing` | 200，仅 `image / imageagent.generation` | 当前管理能力列表未开放文字用途 |
| `GET /api/model-route-pricing/text/product.research.decision/bindings` | 400 / `INVALID` | 现场BFF的用途白名单未包含研究操作；不是“查询成功且无绑定” |
| `GET /api/model-route-pricing/text/product.selection.decision/bindings` | 503 / `UNAVAILABLE` | 当前企业catalog未开放该文字scope，尚未进入绑定读取；不是“不存在密钥”或模型不可用的实测结论 |

三次均为原浏览器会话、当前身份/企业期望头的GET，只接收白名单去敏字段；没有读取cookie/token/secret、绕过权限、调用provider或写配置。现场 `a66d7d0c` 的BFF白名单未含研究操作；原生handler在catalog不支持本企业scope时即返回UNAVAILABLE，尚未调用绑定owner。该源码与正常页面/GET结果一致，未尝试管理员SQL补证。

现场两个服务的OCI来源标签均为 `a66d7d0cde74cded28d77060ac941851cc342a30`，running/healthy：后端镜像 `sha256:2848da261bfc92412fbd2af99719b4963cd6ad45a3bc834a476f844ee2406ffc`、UI镜像 `sha256:d106b5dfe863e4222214daa0bee42696f9243a9abf5e92308852aa51c71494b1`。这是本轮观察，不是本轮部署；与早前e75/ad3记录不同，也不是本线程f872共享模型/微点源码已部署的证明。保留库不能自动换单位。

### 可审查差异：待填项不伪造为可发布值

| 配置边界 | 当前核实结果 | 拟议目标，尚未写入 |
| --- | --- | --- |
| 组织与操作 | 已确认LAC；文字用途未开放 | 两项继续共用该企业 `text`，各自operation/prompt/schema/权限保留 |
| 绑定与版本 | 文字binding ID、binding版本、credential版本、endpoint摘要、地域均UNKNOWN | 只引用原owner提供的本企业去敏记录；不创建/复制Key，不借图片或成员绑定 |
| 准确型号 | 用户指定 `deepseek-v4.1-flash`；现场无文字绑定可证明其版本 | 先保留该名称；不静默替换 `deepseek-v4.1-flash-1017` 或厂商的 `deepseek-flash` |
| 请求协议 | 仓库有DASHSCOPE/OpenAI兼容文字适配；现场对应绑定UNKNOWN | 候选为Chat Completions、固定 `enable_thinking=false`、原两调用/更小限制；不用图片协议、原生DashScope响应或Responses响应冒充同一mapper |
| 官网价/FX | 下表公开原价已核；对本企业准确型号的适用身份、版本和有效期尚未核准 | 厂商价经原平台owner冻结，CNY按原identity换算；不自造price/FX版本，不恢复200/800独立点价 |
| 内部渠道成本 | 百炼公开价随地域/时段不同；本企业地域UNKNOWN | 作为单独成本预算，不能叠加成第二份用户收费 |
| 消费/调用预算 | 每项成本¥0.30、消费20点；两项各一次总¥0.60/40点已确认 | 保持两调用、每调用输入≤32000/输出≤4000Token及原更小上限；未发布或执行 |
| 共享current/CAS | 新版企业共享revision未取得；现场仍是旧分用途合同 | expectedRevision保持UNKNOWN，不能假设no-revision或用旧图片revision顶替；三类资格和正式运行适用范围仍须满足 |

### 价格：厂商用户计价与百炼渠道成本分别核实

以下单位均为CNY/每百万Token，按2026-11-03官方页面原价记录，不含优惠/免费额度，不是可执行报价。

| 价目及用途 | 非缓存输入：忙/闲 | 输出：忙/闲 | 缓存输入：忙/闲 | 时段依据 |
| --- | --- | --- | --- | --- |
| 百炼北京（渠道成本候选） | 2 / 1 | 8 / 4 | 0.2 / 0.1 | 东八区22:00至次日08:00为闲时，其余忙时 |
| 百炼新加坡国际（渠道成本候选） | 2.188 / 1.094 | 8.75 / 4.375 | 0.219 / 0.109 | 同上；缓存为模型页展示值，不能自行从折扣描述推导更细精度作为已发布费率 |
| DeepSeek官网 `deepseek-flash`（仅准确版本匹配后作为用户计价候选） | 2 / 1 | 8 / 4 | 0.04 / 0.02 | 东八区工作日非中国法定节假日09–12、14–18忙时，其余闲时 |

来源：[百炼准确型号及地区价格](https://help.aliyun.com/zh/model-studio/deepseek-v4-1-flash)、[百炼价格表](https://help.aliyun.com/zh/model-studio/model-pricing)、[厂商型号与价格](https://api-docs.deepseek.com/zh-cn/quick_start/pricing/)。厂商页将 `deepseek-flash` 标为DeepSeek-V4.1-Flash-1017；百炼将用户原名称与1017分别列出，现有证据不能证明两者准确版本相同。即使无缓存单价相同，也不能据此签发版本/价目适用性。百炼缓存价与厂商缓存价、两方峰谷日历不得混用。

整数条件算例仍保持每项两调用、32000/4000Token：厂商高峰无缓存为¥0.192=19.2点，两个¥0.384=38.4点；北京渠道成本¥0.192/项，新加坡国际¥0.210032/项（两个¥0.420064），均低于原成本预算。**若把新加坡渠道价误作用户点价，会变成21.0032点/项，超过20点；不得用它替换厂商用户计价。** 版本/地域/时段未知时，这些只是算例；没有对现有余额、原预留或历史消费重算。

### 用量映射：字段对齐与真实资格分开

[百炼Chat Completions响应合同](https://help.aliyun.com/zh/model-studio/qwen-api-via-openai-chat-completions)与[缓存说明](https://help.aliyun.com/zh/model-studio/context-cache)可对齐现有 `ObserveOfficialOpenAITextUsage` / `openai-text-v1`：

| 原响应字段 | 现有映射与检查 |
| --- | --- |
| `usage.prompt_tokens` | 总输入；非负且有界 |
| `usage.completion_tokens` | 总输出；不再另加reasoning tokens |
| `usage.total_tokens` | 验证等于输入+输出，不作为第三项收费 |
| `usage.prompt_tokens_details.cached_tokens` | 输入中的缓存子集；必须存在且在0到总输入之间，缺失不能当0或全价 |
| `usage.completion_tokens_details.reasoning_tokens`（可选） | 若出现必须不大于总输出；候选仍固定非思考，不能靠默认模式执行 |

源码：[严格用量解析](../../internal/integration/aicapability/einomodel/official_usage.go)、[缺缓存/矛盾/重复字段的合成断言](../../internal/integration/aicapability/einomodel/official_usage_test.go)、[固定非思考准入](../../internal/integration/aicapability/einomodel/deepseek.go)。本轮仅阅读，不新建mapper或运行模型。已有DeepSeek适配测试证明非思考请求控制，不等于本企业真实缓存分项已验证；文档字段一致也不等于准确binding已合格。真实原始响应/用量证据本轮未取得，真实AI/付费调用未授权。

### 剩余阻碍与当前交付结论

1. 当前正常管理链未开放可用的文字绑定读取，无法取得本企业准确binding/版本/地域。需原受限owner提供去敏记录或获准后接通相应管理入口；不通过读取secret/管理员DB补证。
2. 原别名与厂商当前1017的准确身份/适用价格尚未对应；如将来选择1017，原固定非思考准入也须按准确型号核对，不能只换字符串。当前不请求/执行换模。
3. 需按准确协议、型号和地域补可信完整usage证据，再核准原mapper/保守上界。真实调用没有授权，缺失/矛盾仍交原owner保持UNKNOWN，不补造用量或退款。
4. 仓库离线工程与现场a66d合同不同；新版三类完整资格、单位/实例适用安排和运行授权未满足。即便文字条件补齐，也不能部分发布text或自动换量纲。

**结论：公开价格与字段合同已核实；本企业可执行文字绑定和价目适用性未核实，仍NOT_PUBLISHABLE。** 证据在被Git忽略的 `.local/selection-bailian-binding-readonly-20261103-01/`，仅含去敏观察和整数算例，不是API请求体。下方为此前设计/工程交付记录。

Issue #1073已记录后续离线开发授权及 **Reuse Existing Architecture / Ready**。先前 `25477d81f` 的读取器准入修正已通过原CI；现已在本开发分支组合#1078稳定提交 `341278d5743cb9f9c4b2e257d72ae8a5c0ed1aa5`，两项消费者共同使用原SharedTextReader及官方价/微点运行链。研究准入描述和夹具完成组合适配，两项真实HTTP/配置/原账本/原key路径的本地合成回归通过；没有另建公共模型或计费能力。详见[当前离线交接](market-research-runtime-handoff.md#共享模型稳定依赖组合2026-11-03离线实现完成待独立检查)。下方N/A和“未写正式代码”仅适用于原预览阶段，不否定后续授权；预算与禁止发布/真实调用等限制保持。

用户已选择百炼 `deepseek-v4.1-flash`，仅准备两项操作的配置接线和差异预览。每项渠道成本上限 ¥0.30、用户消费上限20 AI点，两项各一次合计 ¥0.60/40点。该确认不是调用、凭据写入、配置发布、服务更新、迁移或合并授权。

Design Basis: **N/A**，本轮只读对齐、非执行预览与文档；不修改生产代码/配置/schema。后续正式消费者接线须引用届时实际交付的共享模型合同，不以本预览替代开工准入或运行许可。

## 1. 与正在重构的模型设计对齐

已读取用户指定的[检查可关闭的 Issue](codex://threads/01a17b25-eb0a-7123-bf5b-ea5c4b914071)最近10轮及其原工作区当前设计，并核对[PR #1078](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1078)。引用基线为已提交HEAD `0bd9d243af86710dc2e2b1329b7e3e6b0b5503ac` 的[企业统一模型配置与官方价折算](https://github.com/henryxu1008/shuomi-ai-commerce/blob/0bd9d243af86710dc2e2b1329b7e3e6b0b5503ac/docs/architecture/enterprise-shared-model-configuration.md)；读取的设计文件SHA256为 `5C252EC552322FFFA020E81A01EA817415B93D1459E1CD90BC88DD9C9CCE47E9`。

后续核对#1078稳定提交 `341278d57` 已完成限定安全基础工程交付，其CI41620567748和原独立工程检查通过；共享text profile、官方价/微点及原页面接口可作为本轮依赖。#1075已组合这份准确提交并验证研究/选品消费者；不消费上游未提交代码、不另建共享模型或计费。完整三类默认、Flare及完整视觉消费者仍未开放，原Writer在原PR继续后续工作，不将有限基础交付写成完整产品验收。

- 企业统一选择 `text` / `image-generation` / `multimodal` 三类模型；各Agent继续拥有自己的业务输入、prompt/schema、权限、模式、预算、run及UNKNOWN恢复。
- 同一企业的两项选品操作读取同一个 `text` 类别选择；不分别维护两条当前模型，不新建Agent专属模型选择、独立费率或第二余额。
- 平台维护官网价/FX，企业选择合格binding。用户扣点仅由厂商官网适用价、冻结FX及完整实际用量推导，经原资源账本预留/结算；渠道成本只约束内部成本预算，不再叠加扣点。
- **撤回上一版每百万输入200点/输出800点的独立资费草案**。20点是单项消费上限，不能作为固定收费或独立Token费率；不逐调用向上取整到整数点。

默认OpenAI/Mini/Flare的全局产品决定保留。本次用户的最新DeepSeek选择只作为Local Account Center的共享文字类别候选，不改变其他企业/新安装默认，也不改图片或多模态选择。若该文字候选将来发布，会影响本企业所有已接通共享text reader的消费者，需在差异预览中明确告知；不能承诺只影响这两个Agent。

## 2. 当前 → 目标差异

| 边界 | 当前保留实例/旧草案 | 本次目标预览 | 本次实际写入 |
| --- | --- | --- | --- |
| 企业 | Local Account Center，原核对ID `394513833252618242` | 固定该企业；正式身份仍由服务器核实 | 无 |
| 启用/业务模式 | 两项已启用，`read_only`，执行设置版本1 | 保留，不升级权限 | 无 |
| 模型选择 | 现场模型管理仅展示图片用途，文字绑定UNKNOWN；旧草案按操作分别配置 | 同一企业 `text` 选择：百炼 / `deepseek-v4.1-flash` 候选，由两项复用 | 无 |
| 凭据 | 文字凭据/地域/准确版本未核实 | 只引用原owner的组织级去敏binding及准确版本；不读写secret、不借图片/成员/全局Key | 无 |
| 用户计价 | 旧预览存在200/800独立点价 | 该草案撤回；厂商DeepSeek官网适用价 + 冻结FX → 微点 → 原资源账本 | 无 |
| 渠道成本 | 百炼地域/峰谷价格参考 | 单项成本≤¥0.30，与用户点数上限分别验证；不重复收费 | 无 |
| 单位/资源 | 现有运行仍e75 UI / ad3后端，原页面显示整点资源 | 目标单位 `ai-micropoint-v1`，1点=1,000,000微点；只做目标数学预览 | 无转换、DDL、授予、月限或余额修改 |
| 发布 | 无本企业可证明的三类共享current/revision | 引用共享配置owner的校验、整体CAS发布、原键回执；本文件不是API请求体 | 无 |

不生成可直接套用的旧manifest、per-operation路由覆盖、SQL或Compose覆盖。当前保留库不能被猜测为新单位或自动演进成新版库；后续安装/切换及数据处置须另行明确决定。

## 3. 两项操作的目标调用接线

| 操作/准确定义 | 读取的共享类别 | 保留的业务合同 | 原业务入口 |
| --- | --- | --- | --- |
| 项目资料分析 `product.research.agent@v1.0.0`，operation `product.research.decision` | `ai.models.enterprise` 的 `text` binding及类别摘要 | `market-research-prompt-v1` / `market-research-action-v1`，同一项目2–5份准确资料、项目历史、人工决定和原研究权限 | `/workbench/supply/selection?step=analysis` |
| 独立商品对比 `product.selection.agent@v1.0.0`，operation `product.selection.decision` | 同一个 `text` binding及类别摘要 | `product-selection-prompt-v1` / `product-selection-action-v1`，独立商品引用/context/历史和原事实读取权限 | `/workbench/agents/mine/product.selection.agent` |

目标链：原application注入共享配置reader → 按operation固定映射到text → 组合本操作prompt/schema及更小硬限制 → 原Prepare保存准确选择/凭据/官方价/FX/单位/输入/模式快照 → 原最后发送门禁 → 原调用计量与资源预留/结算。

公共reader、统一目录/价目/资源单位由#1078原owner负责；#1075继续拥有research消费者及两项业务入口。本轮已组合其稳定提交，覆盖research运行声明/私有shape准入、两项共享text与原官方价/微点链，并运行合成HTTP/配置/账本/恢复回归。shared reader缺失或不合格时不能fallback到旧per-operation current。真实Local绑定资格和完整三类发布仍未满足，合成通过不代替真实配置/调用授权。

## 4. 预算与官网价参考

| 限制 | 单项 | 本轮两项各1次 | 含义 |
| --- | --- | --- | --- |
| 渠道成本 | ¥0.30，即300000货币微元 | ¥0.60，即600000货币微元 | 百炼实际渠道成本预算，不作为第二份用户扣款 |
| 用户消费 | 20 AI点，即20000000微点 | 40 AI点，即40000000微点 | 按1点=¥0.01，对应官网价折算金额上限¥0.20/¥0.40；不是20/40元，也不是固定收费 |
| 模型调用 | 最多2次/项 | 仅两项各1次业务分析的计划范围 | 不追加修复调用或新运行；UNKNOWN由原owner核实 |
| 用量草案 | 每次调用输入≤32000、输出≤4000Token | 最大合计输入128000、输出16000Token | 输入/输出字节、事实披露及原更小限制继续有效；准确serializer/上界尚待核准 |

渠道成本≤¥0.30和用户消费≤20点必须同时满足，不能把¥0.30自动换成可扣30点。两项总量是这次仅各1次的计划约束，不新增跨Agent预算账本或自动调度机制。有效期、时长及其他未核实限制不得擅自扩大。

[DeepSeek厂商官网](https://api-docs.deepseek.com/zh-cn/quick_start/pricing/)当前列出 `deepseek-flash` 对应DeepSeek-V4.1-Flash-1017：每百万Token高峰缓存输入0.04元、非缓存输入2元、输出8元，空闲时减半；其高峰时段和节假日规则与百炼渠道价目不同。[百炼模型页](https://help.aliyun.com/zh/model-studio/deepseek-v4-1-flash)另列原别名及 `-1017` 快照；不能由同系列名称推定两渠道准确版本一致。本次保持用户指定别名，不静默换成快照，准确版本及厂商价目映射仍UNKNOWN。

仅为条件算例：假设准确版本、CNY价目、时段和用量映射已获准，采用厂商高峰非缓存输入2元/输出8元每百万Token，每项两调用的官网金额上界为 `2 × (32000×2 + 4000×8) / 1000000 = ¥0.192`，即19.2点/19200000微点；两项为¥0.384、38.4点/38400000微点。它们低于20/40点上限，不把预留当实际收费。

[百炼渠道价格](https://help.aliyun.com/zh/model-studio/model-pricing)在表列北京/全球高峰下同一用量约¥0.192/项，新加坡国际高峰约¥0.210032/项；两项约¥0.384或¥0.420064，低于¥0.60渠道成本预算。实际地域和渠道条款仍UNKNOWN，不能把这些参考值视为可执行报价。

官网价和渠道价必须各自使用准确适用时段/版本，不能将一方的峰谷日历套给另一方。准备时冻结适用价目/有效期，报价不能跨已知价格边界。CNY使用显式单位换算/FX版本，不能编造汇率版本。

原资源owner按保守上界最终ceil微点预留，完整实际费用最终floor微点结算并按原债务规则释放差额。上界计算可按无缓存保守预留；真实用量缺缓存字段不等于零缓存，缺失/矛盾维持UNKNOWN，不按全价或零价补扣。原余额、原24点预留和既有消费不因本预览转换、释放或重算。

## 5. 不可发布项及解除条件

| 项目 | 当前状态 | 需要的具体证据/责任 |
| --- | --- | --- |
| 文字binding、地域、组织归属、凭据版本、共享current/expected revision | UNKNOWN | 原受限凭据/共享配置owner给出合格去敏投影；不读取secret或借别企业绑定 |
| DeepSeek准确型号/快照与厂商官网价身份 | UNKNOWN | 实际百炼binding能证明对应厂商准确版本；不从两个别名相似推断 |
| 官网价/渠道价有效时段、缓存分项及serializer/token上界 | NOT_VERIFIED | 原可信目录及usage mapper核准；#1078现有Mini的`openai-text-v1`解析不能自行宣称适用于百炼DeepSeek |
| 两项共同读text及全体已开放text消费者资格 | OFFLINE_WIRING_VERIFIED；真实资格UNKNOWN | 组合稳定SharedTextReader，保留准确operation/prompt/schema和更小预算；真实企业绑定/全体消费者资格仍由原目录核验，不通过合成夹具代签 |
| 三类整体发布 | NOT_READY | 原合同要求text/image-generation/multimodal全部合格；Flare费用上界缺口仍由#1078保留，不只发布text，不造另外两类binding，不绕过原Must |
| 新单位与保留实例 | NOT_READY_FOR_RUNTIME_CHANGE | 新schema/单位全链的限定工程基础已交付；未授权实例更新，原库不可自动迁移/换量纲，后续由用户决定适用安装/切换范围 |
| 真实调用与商品资料 | NOT_AUTHORIZED / NOT_RUN | 后续单独明确准确资料和费用后才可执行；本轮不选择商品或创建研究项目 |

以上缺项阻止发布/运行，不阻止本次非执行预览交付。整体发布/实际保存/完整业务验收NOT_RUN，不将上游局部基础Ready或旧CI转换成这两项已可运行的结论。

## 6. 本次交付与停止点

产物仅为本预览、运行交接修正以及本地去敏计算记录。未写正式接线代码、凭据、业务/模型配置或账本，未改变服务、实例、数据、卷、会话或角色；未调用模型、平台、付费接口或发送消息到另一线程。沿原PR #1075维护文档，不合并、不关闭Issue。

后续先复用#1078的实际交付与原owner，取得明确的去敏binding和统一目录/用量依据，再把本预览转为可核验的真实配置差异。三类/单位/资料或授权缺失时保持未开放，不恢复已撤回的独立点价和旧路由。
