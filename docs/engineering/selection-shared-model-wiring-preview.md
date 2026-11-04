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

本轮未读取密码、Key 或 worker 私密配置，未查询实际业务/共享 current/余额，未连接真实模型。
只核对固定源码、原非秘密记录、文件哈希及容器版本/挂载元数据；不是“已验证真实绑定/数据库资格”。
不做旧兼容、迁移、双读/双写、旧 ID 映射或第二 current；不改凭据、不发布配置、不构建/更新实例、不真实采集/AI/付费、不合并。

## 当前基线与目标差异

2026-11-04 本轮 fresh 核对：原身份/后端/UI 均 running/healthy，前后端仍为 a66；当前 manifest SHA256 仍为
`1271E0D89BC8E8F8DE5DF92FFCE94D9B9032291C9D7EC04FEA8FCC6AFB195B97`，与已批准非秘密结构记录对应。
以下旧字段来自该记录，不重新打开含秘密的配置正文。旧文档中的“尚未实现/待最终复核”已被 1e3509 的交付证据替代。

| 配置位置 | 原 a66 已核结构 | 1e3509 目标与边界 |
| --- | --- | --- |
| `modelRoutePricing` | `imageWriter` + 4 条 image binding，无统一 `writer` | 新私密 manifest 只使用当前 `writer` / `bindings`；不是机械改名，不能借原 image DB writer 充当 Run writer |
| `modelRoutePricing.writer` | 当前合法 writer 的角色/凭据/权限未核 | host/port/database 必须与 `productAgent.database` 相同；user=`model_route_config_writer`，与 reader 用户不同；maxConnections≤2。其余连接参数由原受控交接提供，不写入本仓库 |
| `productAgent.database` | 非秘密记录为 `product_agent` | 保持该当前 Run/config owner 和原读取身份；不换数据库承接旧业务、不用 Image 或 Product/Review pool 代替 |
| `productAgent.enabled` | false | 保持 false；不为配置管理设 true。不开文字 executor/Review/Asset 执行池 |
| 两项文字定义 | 本轮未查询真实部署声明/版本 | 按下表准备原用途 anchors 和候选描述；缺身份/版本不造值。声明合法只允许管理，不自动获得执行资格 |
| `modelRoutePricing.bindings` | 原 4 条 image 声明 | 逐条按当前 Definition/ImageAdmission 合同核对后明确纳入；不搬旧 current/revision、不解释旧价格字段、不悄悄清空目录绕过校验。未核项保留为待补，不标 Ready |
| `imageAgent.generation` | 只有 `priceVersion` / `pointsPerImage`，无当前 enabled | 新 manifest 显式 `enabled=false`；使用当前 `unitVersion` / `maximumMicropoints`，旧两字段不进入新文件。图片预算上限须由原 owner 给出现行批准值，不从旧点数自动换算或套用研究/选品 20 点 |
| 完整图片模式 | worker 文件引用存在 | 保持非空 `workerConfigFile` 和原 API/Asset/Product/config owner，以显式完整模式保留历史；关闭态不读该文件、不打开 workerDB、不连接图片 Temporal、不装配/启动图片执行 worker |
| 图片/Asset/审计读取 | 原 manifest/挂载仍在，实际 schema/grant 未核 | 保留 `image_agent_runtime`、`supply_asset_runtime` 及原 `accountAuditUsage` 读取分工；完整模式独立 image audit reader，文字执行关闭时相应 product audit reader继续按原合同。不能用 writer 或管理员账号替代 |
| Chat / 其他执行者 | 本轮没有新的无秘密开关投影 | Chat 如保留只能沿原 `conversationOnly=true` 的存储模式且无 planning policies，或关闭 Enabled；其他现有 worker 是否会消费真实任务仍 UNKNOWN，更新前须单列核对，不能把两个 false 当全系统 AI/采集已停 |
| 平台参考价、企业三类选择、操作模式 | 本轮不读取/改写实际 current | 与运行 manifest 分开。平台价格和企业完整三类选择沿原 owner/CAS 发布；本次不发布，不恢复旧 image current，不用残缺选择快照替代完整 CAS |

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

上述 Run 数值仅说明已批准的两项方案。当前 ProductAgent 共享 Limits，若最终 manifest 还声明其他用途，
须先核对它们的原限制和组合，不可用本表直接覆盖其他用途预算，也不在本轮另造一套限制 owner。

需要由现有非秘密准入投影补齐：LAC准确 Organization ID；绑定是否仍存在/启用、准确 ClientName、CredentialVersion、EndpointIdentityDigest；
两用途 Profile/版本；合法价目/FX/参考版本及完整三类选择的当前 revision。历史记录曾见 `local-content-route-830-20261020`，
本轮没有 fresh 确认，不能据此判定研究/选品当前可用。不能复制图片 Key 给文字，也不换型号/渠道。

### 文件编码必须遵循当前合同

- `generation.maximumMicropoints` 与 `bindings[].image.maximumMicropoints` 是十进制 **JSON 字符串**，unitVersion=`ai-micropoint-v1`；非零声明须同时有合法单位与正上限。
- `bindings[].maximumMicropoints`（文字每调用预算）是 **JSON 整数**；它与 Generation 的 wire 格式不同，不可混用。
- Definition 的 scope 使用 `organizationId` / `capability`；文字 `text.Profile`、`text.AdmittedCredentialVersion`、`text.AdmittedEndpointIdentityDigest` 沿当前 Go JSON 结构。
- Profile 的 `DeadlineBound` 沿原 duration 编码和已批准更小值，不凭空写字符串 `120s`；不伪造 BoundEvidenceVersion、Tariff/FX/current revision 来通过校验。
- 仅内容模式仍只可声明内容 policies；关闭态也不能混入标题、研究或选品等其他用途。两项研究/选品不放在 ContentOnly/ContentTitleOnly 安装模式中。

## 启动所需资源与开放所需资格分开

| 缺项 | 影响层级 | 当前证据 / 处理 |
| --- | --- | --- |
| 同 Run DB 的合法 writer 及密码交接、角色连接额度 | 新版本启动 | UNKNOWN；应用要求 pool≤2且现角色 connlimit足够，启动只Verify，不自动ALTER ROLE。不能提权、创建角色或改凭据补齐 |
| 原 Run reader / writer schema与grant | 新版本启动 | UNKNOWN；沿 `VerifySchema`/`VerifyReader`/`VerifyWriter(false)`：当前 revisions/current、credential metadata view及原约束/窄权限。不能用空库、默认用户或临时DDL绕过 |
| 原完整图片/Asset/配置/审计读取资格 | 原历史与启动 | UNKNOWN；保留原owner和历史，按原Verify检查。缺schema/权限则停止更新，不初始化/迁移/清库，不更换owner以丢掉历史 |
| 准确企业/两项文字定义元数据 | 声明这些候选、后续配置选择 | 本轮未fresh查询；补齐后再在私密新manifest声明，不能用不合法policy换取页面可开 |
| image credential metadata | 图片技术Ready | 关闭态没有worker credential reader，沿原CREDENTIAL_READER_MISSING；不是为了让所有选项Ready而启worker的理由 |
| 参考价/完整共享选择/操作模式版本 | 后续真实研究与选品 | 本次不发布；未就绪只如实显示。配置管理启动不等于模型可执行，技术Ready也不等于真实调用授权 |
| 其他worker/恢复入口、真实队列影响 | 更新前外部副作用边界 | UNKNOWN；需无秘密开关/运行范围投影。不得启动保留队列来做验收，不从关闭图片worker推定其他执行者已停 |

本轮不执行上述真实资源核对或安装。未来若需查schema/grant，只用单独获准的原身份做只读元数据检查；
不要把当前码中的installer视作对保留库的执行授权。查出差异先列明最小处置，不把保留数据迁移/清理当默认步骤。

## 未来配置交付清单

1. 原 owner 提供仅含所需字段的无秘密投影；填清本页 UNKNOWN 与既有凭据交接是否满足“不改凭据”。不需要再次研究精确厂商价格。
2. 形成独立的新私密 manifest 及去密 diff：只包含明确批准的字段改动，productAgent=false、generation=false；原 a66 文件和哈希不变。现在没有生成或写入该真实文件。
3. 在明确获准的离线/元数据检查范围完成当前严格字段和资源校验；不以在 a66 上尝试启动或请求模型来“验证配置”。
4. 通过后才进入[两服务更新方案](market-research-unified-update-plan.md)。模型/参考价发布、构建、实例更新、真实AI分别单独授权，不能因批准本文一并执行。

用户业务链仍是智能市场启用/配置 → 同项目选择2–5项资料 → 比较/费用估算 → 明确确认只读分析 → 人工决定 → 准确子集交接。
本轮不操作这条真实业务链；1e3509已通过的合成回归与真实采集、远端模型、现场历史读取及用户验收区分，后者均NOT_RUN。
