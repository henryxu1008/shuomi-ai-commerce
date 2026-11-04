# Local Account Center：1e3509 统一模型配置差异

状态：**STATIC_FIELDS_VALIDATED / ACTUAL_BIND_PERMISSION_BLOCKED / NOT_PUBLISHED**。独立候选已生成、0600内存副本的字段解析通过；本轮实际只读bind复核发现Linux mode=0777，不满足固定1e私密文件门禁。本文不含实际配置正文或发布授权。
应用固定为 `1e3509bda06ad9b8160b0c5a7c5d451de9496c61`，tree `074f19ab3dd06a8f9552ba73d13b344055913b6c`。
2026-11-04用户已明确接受候选A：原内容也收紧到¥0.30/72000 Token。该预算决定已批准，不再待确认；
它不授权改凭据、发布配置、构建/更新实例或真实调用。本页其他运行缺项仍按各自层级处理。
原独立增量复核与 [CI 41765722589](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41765722589) 已通过；本轮文档提交不改变应用来源，也不把原 CI 冒称新文档 SHA 的直接执行结果。
前一文档提交 `09830cf5527b3ebc9c58091106535e25168b7035` 的 [CI 41782829529](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41782829529)
已 completed/SUCCESS，Required CI Gate `126424403536` SUCCESS；架构、准入、分类、前后端、CodeHealth均SUCCESS，条件Windows/Release/Capture为SKIPPED。
本页随后收敛的草案不是该旧SHA直接执行过的配置，也没有被应用到实例。

## 本次交付与依据

使用者为 Local Account Center 配置管理员及指定实例操作者。目标是准备一份符合当前统一 owner 的关闭态运行配置，
使管理员后续可以查看配置、业务人员可以读取原历史；研究/选品和图片新执行继续关闭。
本轮 Design Basis: **N/A（只读hash/ACL/bind及启动合同复核、文档）**，复用已准入的[生命周期合同](../architecture/ai-disabled-application-lifecycle.md)、
[统一模型合同](../architecture/enterprise-shared-model-configuration.md)和[两项文字合同](../architecture/bailian-deepseek-shared-text-admission.md)。
不改变产品、权限、状态、事实 owner 或恢复语义。原工作区、唯一 Writer、PR #1075 保持。

此前阶段使用原浏览器及原数据库身份完成只读元数据查询，数据库检查为 `BEGIN READ ONLY`、5秒语句/1秒锁超时、`ROLLBACK`。
本轮按用户明确授权，将原manifest凭据原值复用到新的私密候选；未展示、改写凭据或写入仓库/去密回执。
没有读取Key表数据、worker私密文件、任务载荷、商品、余额或模型响应，没有调用发布、安装、恢复或执行入口。
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
| `modelRoutePricing.bindings` | 原 4 条旧 image 声明 | 本次新目录显式仅纳入下文3条当前文字Definition；旧4条图片声明逐条列入“不进入新目录”，原a66文件及Image current/history不改。完整图片读取沿原owner，不复制旧options或换算旧点价；不是清空整个目录绕过校验 |
| `imageAgent.generation` | 只有 `priceVersion` / `pointsPerImage`，无当前 enabled | 新差异为 `{"enabled":false}`，其余单位/上限不声明，使用合同已有的零元数据关闭态；旧两字段不进入新文件。不制造正图片预算或套用文字20点；本次也无新ImageAdmission需要预算 |
| 完整图片模式 | worker 文件引用存在 | 保持非空 `workerConfigFile` 和原 API/Asset/Product/config owner，以显式完整模式保留历史；关闭态不读该文件、不打开 workerDB、不连接图片 Temporal、不装配/启动图片执行 worker |
| 图片/Asset/审计读取 | 本轮5个原角色均READ ONLY登录成功，元数据结果见下表 | 保留 `image_agent_runtime`、`supply_asset_runtime` 及原 `accountAuditUsage` 读取分工；完整模式独立 image audit reader，文字执行关闭时相应 product audit reader继续按原合同。不能用 writer 或管理员账号替代；实际历史GET仍NOT_RUN |
| Chat / 其他执行者 | Chat enabled=false；其余范围见本轮启动清单 | Chat 保持关闭。DataServices 的存在会启动采集 worker，Knowledge 与恢复循环也独立于两个 AI 开关；不能宣称全系统停止执行 |
| 平台参考价、企业三类选择、操作模式 | 已读 current 的版本元数据，未改写；共享选择及参考价未发布 | 与运行 manifest 分开。平台价格和企业完整三类选择沿原 owner/CAS 发布；本次不发布，不恢复旧 image current，不用残缺选择快照替代完整 CAS |

上述是字段级目标，不是可直接复制的 manifest。没有用假密码、`<组织ID>`、虚构 revision 或不存在的 passwordFile/env 插值功能拼出“可运行配置”。
独立私密文件已生成，原a66文件没有覆盖；复用原受控凭据，不创建账号、设密码或修改任何grant。

### 已生成文件与离线解析回执

候选宿主路径：`C:/Users/Henry/.codex/private/market-research-unified-20261104-config-a-01/current-application-1e3509-a.private.json`。
SHA256：`194B770F10B165768959CFC9B678D99F8F1BFECF4FA65387B87C02CFFF0786EC`。
新目录先限制ACL，再写入凭据；允许主体仅Henry和SYSTEM。原文件SHA256仍为上文1271…，原后端仍只读挂载原文件，未加载该候选。
本轮fresh复核hash仍完全匹配，宿主文件无链接跳转、父ACL受保护；本轮没有修改文件内容、ACL或POSIX mode。

固定1e的 `internal` tree为 `8653591a9667dc803777e0d711984bba1d841fb7`，与本工作区解析时的生产源码完全相同。
临时解析测试只调用现有 `currentapplication.LoadConfig`，不调用`Run`；在network=none、uid/gid=65534、只读根的一次性进程中运行，
输入经stdin进入私有tmpfs的0600内存文件。未启动应用、身份预检、监听器、DB连接、worker、恢复循环或provider；未构建应用镜像。

| 离线检查 | 结果 | 范围 |
| --- | --- | --- |
| 固定源码LoadConfig：唯一JSON键、未知字段、尾随JSON及完整静态validate | PASS | 针对上述候选hash，Linux路径语义；不证明实际挂载/依赖启动 |
| 已批准A、完整图片零声明、单Run writer和3条文字定义 | PASS | 不包含模型使用时资格或实际价目发布 |
| 注入未知字段、重复字段、尾随JSON的负例 | 均被拒绝 | 使用同一个现有严格入口，无宽松解析/fallback |
| 原凭据与其他配置语义对比 | PASS | 密码/令牌/密钥相关值及引用未变；writer仅改变目标库，限定变化外全部相同 |
| 原a66与候选状态 | 原容器健康、原hash/挂载不变；候选未发布 | 1e正式启动、现场历史GET和用户验收NOT_RUN |

### 实际只读挂载：内容可读，但Linux私密文件模式不合格

本轮在一次性工具进程中，将候选只读bind到未来后端的同一目标 `/private/market-research/current-application.json`。
原后端PID1已核为UID/GID 0:0；固定Dockerfile无USER切换，预期相同。检查只复用这个已存在的身份，
network=none、根只读、capabilities全删除、no-new-privileges，不执行应用ENTRYPOINT，不挂其他私密目录/业务卷。
1e应用镜像仍未构建，所以最终镜像Config.User及Compose覆盖须在获准构建后再次核对；本次不冒称已验证1e镜像。

| 复核项 | 实际结果 | 结论 |
| --- | --- | --- |
| 宿主hash / Windows ACL | 194B…0786EC，Henry/SYSTEM，仅原身份；父ACL受保护 | PASS，未改权限或凭据 |
| bind路径/只读标志 | 同上目标，实际mount flags为ro | 仅工具进程试挂载；原a66挂载不变，候选未发布 |
| 实际读取 | SHA256与宿主完全一致，CapEff=0 | READ_PASS，没有显示文件内容 |
| Linux文件属性 | 普通文件，owner 0:0，mode **0777** | 不符合私密文件要求 |
| 固定1e LoadConfig文件门禁 | `info.Mode().Perm() & 0077 != 0` 在0777上成立 | **按实际属性与固定源码确定不合格**；该guard在打开JSON前拒绝，不是字段解析失败 |
| 新镜像/正式应用加载 | NOT_BUILT / NOT_RUN | 没有运行1e应用来试错 |

Windows ACL私密和Linux POSIX mode是两个检查。此前严格解析是在0600内存副本上运行，证明字段合同成立，
没有证明宿主文件直接bind后的mode合格。这里保留原PASS的准确范围，不将其扩大成真实候选可启动。
DrvFS的`test -w`在只读bind上仍返回true，本轮未据此试写；只读状态以实际mount flags的ro为证据。
真实读取的hash和mode记录于 `.local/selection-config-mount-review-1e3509-20261104-01/mount-review.metadata.json`，无秘密值。

```text
Finding: 新候选在实际只读bind下呈现0777，固定1e LoadConfig会拒绝。
Product requirement affected: AI关闭后完整应用仍须能够启动、管理配置并读取历史。
Classification: BLOCKER（未来更新/试用启动；核心happy path无法完成），不影响当前原a66。
Reason: Linux文件门禁先于JSON打开/解析，宿主Windows ACL不能替代该检查。
Action: 本轮只读复核不修改。最小后续修正仅将这一个未发布候选的POSIX mode收紧到0600，
        同时核Windows ACL、候选/原文件hash不变；再在同一实际bind下调用固定1e LoadConfig，
        不能使用0600内存副本代替，也不调用Run。不改原a66、不发布、不构建或更新服务。
```

该修正是文件权限元数据变更，超出本轮只读复核；未自行chmod、放宽ACL、换身份或跳过门禁。

首次真实严格解析发现，原**已关闭Chat**的 `aiWorkbench.planningTextPolicies.394513833252618242.Profile.PointTariff`
也会被全文件严格解码拒绝。最终候选只额外移除这个退休字段，Chat仍enabled=false，其余Chat配置/凭据不变；
不把旧点价映射成OfficialTariff，不增加Chat定义或放宽解析。初始拒绝与最终通过区分保留，不把失败记作PASS。

去密回执为 `.local/selection-private-a-parse-1e3509-20261104-01/redacted-diff.json`，
仅包含源/候选hash、实际变化的字段路径、白名单目标值和凭据未变结论；不包含任何秘密值。
同目录保存严格解析结果与一次性测试源码/产物的hash，不作为正式工具或生产入口提交。

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
原内容声明也须按当前合同重新声明。以下将差异收敛为准确候选及其产品影响；不写真实配置，不默改原内容用途预算。

### 三用途声明的准确候选

本轮候选均沿 LAC 原 `local-content-route-830-20261020` 和下节已核凭据/端点版本，不新增凭据记录、不复制Key。
研究、选品分别新增对应 policies；原内容保留其当前 owner 和 prompt/schema，以当前 ModelProfile重新声明，
删除不再接受的 `PointTariff`，不把它换算成新的客户点价，也不复制旧 current/revision。

| 字段 | 原内容的当前合同候选 | 项目资料分析候选 | 独立商品对比候选 |
| --- | --- | --- | --- |
| policy map / capability | `contentTextPolicies` / `product.content.decision` | `researchTextPolicies` / `product.research.decision` | `selectionTextPolicies` / `product.selection.decision` |
| prompt | `product-content-agent-v1` | `market-research-prompt-v1` | `product-selection-prompt-v1` |
| output schema | `product-content-action-v1` | `market-research-action-v1` | `product-selection-action-v1` |
| prompt / completion上限 | 保留65536 /4096 | 32000 /4000 | 32000 /4000 |
| 输入/输出字节上限 | 保留65536 /16384 | 32768 /16384（不高于原用途界限） | 32768 /16384 |
| `DeadlineBound` | 保留45000000000纳秒（45秒） | 45000000000 | 45000000000 |
| 每调用参考成本上界 | 163840微元，即¥0.16384 | 96000微元，即¥0.096 | 96000微元，即¥0.096 |
| Definition每调用点预算 | 保留0（未声明独立点上限；不是零扣费） | `maximumMicropoints=10000000` | `maximumMicropoints=10000000` |

三项共同的 `ProviderID=DASHSCOPE`、`AdapterKind=openai-compatible`、`ModelID=deepseek-v4.1-flash`、
`AdapterPolicyVersion=dashscope-deepseek-v4.1-flash-nonthinking-v1`、Currency=CNY不变；
输入/输出参考费率仍2000000 /8000000微元每百万Token，`UsageMappingVersion=openai-compatible-token-usage-v1`。
不把该serializer标识与价目的 `reference-text-totals-v1` 用量策略混成同一字段。

同一绑定覆盖这三个文字消费者时，三个显式 Definition 必须使用相同的 manufacturer / region / serviceType / channel / priceId / referenceVersion：
候选为 `deepseek` / `cn-beijing` / `standard-sync` / `bailian-beijing` /
`lac-deepseek-reference-text-v1` / `lac-deepseek-text-reference-v1`。
Profile的候选RoutingPolicyVersion为 `lac-deepseek-text-admission-20261104-v1`、CostPricingVersion为 `reference-text-totals-v1`。
这些**新增标识均为未发布草案**，不是已有价目或current；未来发布时须随实际版本一起审查。
参考策略不要求BoundEvidenceVersion，留空；不伪造精确费用证明。部署Profile不嵌入执行OfficialTariff，由唯一共享owner在使用时读取并冻结。

不能只补研究/选品而让原内容 anchor 缺少同一ReferenceVersion等元数据，否则原SharedCatalog会按 `CONSUMER_BINDING_MISMATCH` / `OPEN_CONSUMER_MISSING` 拒绝，
不是两项分析通过就可跳过内容声明。原4条图片定义本次不进入新目录，不能用文字元数据或20点预算改装成ImageAdmission。

### 共享 Limits：已批准候选A

固定1e的 [ProductAgentConfig](../../internal/app/runtime/currentapplication/product_agent.go)只有一套Limits。
研究/选品prepare直接继承parent Limits，仅把ModelCalls收紧到2；
[配置装配](../../internal/app/runtime/currentapplication/model_route_pricing.go)还会覆盖显式Definition的MaximumCostMicros，研究/选品使用 `CostMicros/2`，内容使用完整 `CostMicros`。
因此在binding里手填150000不能独立于原¥1共享上限生效。这里是现有装配的真实限制，本轮不新增按用途预算owner或修改固定应用。

| 共享字段 | 原a66 | 已批准A：三用途共享当前上限 |
| --- | --- | --- |
| enabled / contentOnly / contentTitleOnly | false /true /false | false /false /false |
| currency / steps / modelCalls / runtimeSeconds | CNY /12 /2 /120 | 原样保留 |
| tokens | 139264 | 72000 |
| costMicros | 1000000 | 300000 |
| 派生Definition成本上限 | 内容1000000 | 内容300000；研究/选品各150000每调用 |

候选A满足两项每Run¥0.30、最多2调用、每调用32000/4000、20点；参考最大两调用¥0.192/19.2点。
原内容的单次Profile仍可装入新上限（69632 Token、¥0.16384），但两次最大调用需要139264 Token、¥0.32768，
超过候选A共享上限。因此A**会收紧原内容Run的预算和Token包络**，不能称为原内容行为完全不变。
用户已明确接受这一影响；批准的是目标差异，实际配置仍未发布。

候选B保留原内容¥1/139264共享Limits，则研究/选品Run展示和冻结的上限也会继承这两个原值，
不符合本次已批¥0.30/≤72000方案；不能靠实际估算较低或20点Definition预算掩盖。B不进入两项功能的配置发布候选。
如果既要求内容原Run上限完全不变、又要求两项准确独立上限，固定1e没有纯配置解法，须另定源码范围；本轮不实施。

当前技术及产品候选均明确为A，不再将原内容预算列为阻碍。候选B不保留为待选方案，也不新增按用途预算设计。
本轮仍只交付差异，不发布manifest、不声称真实配置已启动。无需因此继续研究精确厂商价格。
本轮非运行草案及整数算术记录保留于 `.local/selection-config-convergence-1e3509-20261104-01/`，无DB连接参数或凭据值；未做实际启动/远端资格验证。

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

- 本次Generation只声明enabled=false，单位/上限均省略，无ImageAdmission。未来如声明正预算，`generation.maximumMicropoints` 与 `bindings[].image.maximumMicropoints` 必须为十进制 **JSON 字符串**，unitVersion=`ai-micropoint-v1`；不能仅填半组元数据。
- `bindings[].maximumMicropoints`（文字每调用预算）是 **JSON 整数**；它与 Generation 的 wire 格式不同，不可混用。
- Definition 的 scope 使用 `organizationId` / `capability`；文字 `text.Profile`、`text.AdmittedCredentialVersion`、`text.AdmittedEndpointIdentityDigest` 沿当前 Go JSON 结构。
- Profile 的 `DeadlineBound` 沿原 duration 编码和已批准更小值，不凭空写字符串 `120s`；不伪造 BoundEvidenceVersion、Tariff/FX/current revision 来通过校验。
- 仅内容模式仍只可声明内容 policies；关闭态也不能混入标题、研究或选品等其他用途。两项研究/选品不放在 ContentOnly/ContentTitleOnly 安装模式中。

### 本次关闭态图片目录差异

按已准入生命周期合同，零Generation元数据可以表示“没有图片执行资格”，不是必须先补正数图片预算。
固定1e的 `Config.validate` 仅在Generation存在非零元数据时校验单位/上限；`buildFullImageApplication` 的关闭分支
保留Image/Asset/Product/Run配置读取和历史接口，不依赖目录中的ImageAdmission、报价或worker凭据。
因此本次供审差异为：保留workerConfigFile的原非空引用和完整模式，Generation=`{"enabled":false}`，
新bindings仅含3条文字定义；旧4条图片定义不搬入新Run目录。没有新增图片点数预算或把旧点数换成微点。

| 原部署图片声明 | 新关闭态目录差异 | 原数据/原a66处理 |
| --- | --- | --- |
| LAC / `image_gpt_image_2` / bailian qwen-image-3.0-pro | 不进入本次新bindings | 原凭据、Image current、历史和原manifest不改 |
| LAC / `image_gpt_image_2_vip` / grsai gpt-image-2-vip | 同上，不伪造完整用量准入 | 同上 |
| A组织394513989062623234 / `image_gpt_image_2`，部署标记not-provisioned | 同上，不将占位版本当已准入 | 不查询该组织业务记录，不操作其配置 |
| B组织394513989213618178 / `image_gpt_image_2`，部署标记not-provisioned | 同上 | 同上 |

这是显式列出的**新文件目录范围差异**，尚未发布，不是从原配置中删除记录，不清理数据库、不覆盖旧revision、不改变原历史状态。
新管理界面没有该4项已准入生图候选，默认/未准入的图片类别如实不可用；未来启用图片仍须另给合法ImageAdmission/预算/原reader和真实调用授权。
不能为保持下拉列表外观而装配图片worker、捏造正预算，或把新目录缺项伪装成可生成。
企业三类选择仍沿完整快照/CAS；未发布的当前选择不被本次目录差异补写或迁移。

## 启动所需资源与开放所需资格分开

| 缺项 | 影响层级 | 当前证据 / 处理 |
| --- | --- | --- |
| 同 Run DB 的合法 writer 及连接额度 | 新版本启动 | 原身份只读登录成功，current_user=session_user=`model_route_config_writer`，无super/createDB/createRole/replication/bypassRLS，connlimit=4；快照原writer连接1条。目标pool1无需调整角色或凭据 |
| 原 Run reader / writer 配置schema与grant | 配置管理启动 | revisions/current/元数据view、4个作用域/CAS约束及窄权限与 `VerifyReader/VerifyWriter(false)` 的只读检查项相符；reader无配置写权，writer仅revisions INSERT、current INSERT/UPDATE，无Key读取/凭据写入/public CREATE。未实际运行应用Verify或尝试写入 |
| `ai_route_config_options` | 关闭态统一配置管理 | Run库不存在；1e明确使用 requireOptions=false，不是该路径启动缺项，不能据此安装旧Image options表 |
| `ai_invocations` | 后续真实文字执行 | 当前53列/目标52列：缺 `official_tariff_json`、`billable_usage_json`；多旧 `point_price_version`、input/output points列。不符合1e严格执行合同；enabled=false不装配文字执行器，因此不把它误判成关闭态管理启动阻碍。本轮不迁移/安装/改库 |
| 原完整图片/Asset/配置/审计读取资格 | 原历史与启动 | 本轮原5角色READ ONLY核对结果见下表；权限/必要列/约束已匹配相应检查项，未读取历史业务行、未真正装配1e，不能称完整启动或历史验收PASS |
| 准确企业/三项文字定义元数据 | 声明候选、后续配置选择 | 企业/原Client/凭据版本/端点/超时已核；三用途定义与A预算已写入独立候选并通过严格解析，尚未应用到实际部署，研究/选品未获执行资格 |
| 图片候选及credential reader | 图片技术Ready | 本次不声明新ImageAdmission，类别保持未准入；无worker credential reader不阻碍关闭态历史读取。未来声明候选时仍可能CREDENTIAL_READER_MISSING，不为显示Ready而启worker |
| 参考价/完整共享选择/操作模式版本 | 后续真实研究与选品 | 本次不发布；未就绪只如实显示。配置管理启动不等于模型可执行，技术Ready也不等于真实调用授权 |
| 其他worker/恢复入口、真实队列影响 | 更新前外部副作用边界 | 候选保留/关闭范围已在下表明确；保留依赖不授权消费任务或执行恢复。不能悄悄停原服务，也不能将候选的启动副作用解释为已批准 |

初次连接检查受本机 Windows CRLF 输入影响而认证失败；改为LF后，同一原reader和writer均成功，未更换身份或凭据。
失败记录保留，但不再作为“凭据失效”结论。以上是只读元数据匹配，不能冒称完整启动、现场历史读取或产品验收通过。

本轮额外核对仍只复用原manifest身份，current_user=session_user，BEGIN READ ONLY、5秒语句/1秒锁超时、ROLLBACK。
Image/Asset直接复用固定1e现有验证器中的只读catalog查询；没有调用grant/installer、加载worker文件或读取Key/历史/商品/队列载荷。

| 原身份 / 库 | 本轮元数据结果 | 明确未验证 |
| --- | --- | --- |
| `image_agent_runtime` / image_agent | 原权限查询required_privileges=true、forbidden_privileges=false | 实际完整图片历史GET、冻结快照和URL可读性 |
| `supply_asset_runtime` / image_agent | 4张当前审批/库存表所需列、4个主键及2个唯一索引匹配，运行权限查询true | 实际库存/批准回执业务行读取 |
| `product_agent_runtime` / product_agent | agent_configuration的8张表存在且可SELECT，template_revisions.image_parameters存在 | 配置载荷/模式/历史快照具体内容，Skill等后续实际装配组合 |
| `account_audit_image_reader` / image_agent | 原身份登录、无super/createDB/createRole/replication/bypassRLS；ai_invocations可SELECT，无INSERT/UPDATE/DELETE/TRUNCATE | 具体审计事件和页面读取 |
| `account_audit_product_reader` / product_agent | 同上；不因文字执行关闭而替换为writer | 同上；真实文字执行schema不因此获准 |

原五角色pool配置及owner不变。本轮非秘密SQL/结果仅保留 `.local/selection-final-config-a-1e3509-20261104-01/`，
其中审查JSON含已批准A和零图片声明，不是可加载manifest，也未包含DB连接凭据。

### 更新前缺项已按影响收敛

| 项目 | 当前结论 | 最小后续动作 / 所属阶段 |
| --- | --- | --- |
| 内容预算决定 | RESOLVED：A已由用户批准 | 不再重复请求同一决定 |
| writer、Run配置/图片API/Asset/审计元数据 | MATCHED_METADATA | 真正更新前核漂移，实际启动/历史GET在获准更新后按原入口检查；不把未执行结果写PASS |
| 当前运行文件 | CREATED / STATIC_FIELDS_PASS / ACTUAL_BIND_MODE_FAIL / NOT_PUBLISHED | hash/宿主ACL/实际读取已核；0777私密模式为更新启动BLOCKER。需单独授权只收紧候选mode并在原bind下复核，不覆盖原a66、不正式启动试错 |
| 图片预算/旧4声明 | 本次不开放图片执行，采用零声明及显式目录排除；不是待查精确图片价格 | 作为本次新配置差异一并审查；未来生成准入另行处理，不拖住关闭态管理/历史方案 |
| 保留worker和恢复器的启动 | 范围已明确，启动/消费NOT_AUTHORIZED | 后续更新授权须明确覆盖真实启动影响；若要求完全无后台消费，固定1e+现依赖配置不足，本轮不造开关或改代码 |
| 平台参考价、共享三类选择、操作模式发布 | NOT_PUBLISHED | 不属于本次关闭态启动前置，未来沿原owner完整CAS；不可测试保存/先发布一版 |
| ai_invocations执行合同差异 | CONFIRMED_INCOMPATIBLE_FOR_TEXT_EXECUTION | 只影响未来真实文字执行；不迁移旧数据、不改库，不把它升级为关闭态启动阻碍 |
| 构建/更新/用户试用 | NOT_AUTHORIZED / NOT_RUN | 仅在实际配置审查及相应授权后进入两服务方案，应用SourceRef仍为1e |

### 其他 worker 与任务队列的实际范围

源码依据固定1e的 [currentapplication runtime](../../internal/app/runtime/currentapplication/runtime.go)、
[HTTP装配](../../internal/app/httpapi/current_application.go)及各现有owner；未新增开关、调度器或第二执行owner。
2026-11-04 04:55–04:57 UTC仅对原Temporal `default` 执行 task-queue describe，没有poll、启动workflow、读取历史/载荷或执行恢复。

本阶段原a66的全部现有服务、worker、队列、数据和卷原样保留；不停止、不暂停队列、不重放任务。
以下是**未来1e配置候选的保留范围**，不是本轮启动或任务执行授权；没有新增worker及新的数据来源。

| 执行者 | 原配置 / 启动条件 | 候选保留范围与启动影响 |
| --- | --- | --- |
| DataServices采集 | 配置存在即装配/Start，无独立enabled；Amazon14站点 | 保留原配置和采集队列依赖，维持市场调研资料读取/配置接线；不扩站点、不换SHEIN源。原队列有workflow/activity poller、快照backlog各0；未来1e仍会消费，移除又破坏研究依赖，不能伪造enabled=false或当无影响修补 |
| 完整图片 | 原worker文件引用存在，a66在消费 | `image-agent-organization-v1`有workflow/activity poller，近似backlog各0。1e显式generation=false后仅不装配本应用图片worker；不控制其他进程，也不从本次快照推断所有workflow已结束 |
| 店铺观察 | observations存在，preparationOnly=true | 保留准备模式，不打开official应用访问；队列`store-observations-current`原有workflow/activity poller、快照backlog各0。准备模式仍装配worker并要求空观察库，不等于worker关闭；本轮不读取/清空业务库 |
| POD / Supply / Fulfillment | 三项配置均不存在 | 本应用不装配对应worker；不宣称其他进程不存在 |
| Knowledge | enabled=true | 保留原资料服务及其5秒Sweep、并发2、对象读取/文档解析；不受AI开关控制。启动会处理存量任务，不把它冒称纯只读；本轮不触发Sweep或查询业务队列 |
| Ecoservices | enabled=true、nonPaymentOnly=true；newPayments/newMerchantApplications均false | 保留非支付资格模式；按`buildEcoservices`该分支返回nil恢复回调，**不装配Ecoservices15秒支付恢复循环**。更正此前仅凭enabled推断有该循环的说法，不打开支付/商户申请，也不读其秘密配置 |
| 商业资源 / consumer恢复 / 推荐成熟 | Commercial/Money owner存在，Referrals启用 | 保留原owner/依赖，不删池绕过启动；固定1e仍有资源购买/consumer30秒、推荐成熟1小时循环，首次立即运行，可能写原业务事实。只描述装配，不推定实际待处理数量，不调用恢复或读取订单/余额 |
| 充值 / Chat / BrowserCollector | walletTopUp缺省（两渠道false）；Chat enabled=false；采集桥配置仍存在 | 不能把“无新支付渠道”写成“所有恢复停止”；充值恢复回调仍可装配。原独立browser-collector容器保持退出，不启动、不控制既有桥会话 |

启动时机也属于更新范围：资源/consumer/充值恢复和推荐成熟在HTTP server装配时启动goroutine并安排首次立即执行；
Knowledge处理在Listen前启动；DataServices与观察Temporal worker在Listen后、Serve前Start。
因此不能将“尚未打开页面/尚未人工提交任务”当作“没有后台副作用”。文字执行关闭时不装配modelPointRecovery，
Ecoservices非支付分支没有15秒支付恢复；它们不能抵消其他已保留消费者的启动影响。

这些队列0积压只是上一轮近似快照，不证明无运行中的workflow、未来任务或恢复副作用；本轮不将它重新标成实时零积压。
保留的DataServices/观察/知识/商业恢复会在候选启动时运行，**当前禁止真实采集/付费及不更新实例的边界没有解除**。
因此后续两服务更新必须单独明确该启动范围；若要求候选启动后全系统绝无后台消费/写入，固定1e与当前依赖配置不满足，不能只靠两个AI false承诺。
本轮不停止原a66、不修改消费者源码，也不通过重新启动来验证边界。

## 未来配置交付清单

1. 已完成：按A生成独立候选、补清关闭Chat退休字段、0600内存副本上的固定1e字段解析及去密diff；原a66文件/挂载/凭据值不变。实际bind内容可读但mode0777为启动BLOCKER，不能把前一PASS当成该项通过。
2. 单独获准后先最小修正候选POSIX mode并直接bind重验；之后再审hash/ACL/原元数据漂移、最终镜像User和保留worker启动范围。不重复预算/精确价格调查，不改代码/原a66或创造权限绕路。
3. 只有获得相应授权才进入[两服务更新方案](market-research-unified-update-plan.md)。本轮不挂载/发布该文件，不构建应用镜像或更新实例；模型/参考价发布、真实AI也分别单独授权。

用户业务链仍是智能市场启用/配置 → 同项目选择2–5项资料 → 比较/费用估算 → 明确确认只读分析 → 人工决定 → 准确子集交接。
本轮不操作这条真实业务链；1e3509已通过的合成回归与真实采集、远端模型、现场历史读取及用户验收区分，后者均NOT_RUN。
