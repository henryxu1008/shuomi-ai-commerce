# 百炼商品图片接线 V1

Execution: [#774](https://github.com/qq550723504/task-processor/issues/774)。Design Basis: **Independent Architecture**；Admission: **IMPLEMENTATION_READY**（冻结设计33fd719640df2fc5e519892f9f25dd718f493fb8，独立增量复核已完成）。本设计先于正式生产代码。

## 产品结果与依据

2026-10-18，用户要求查清保留实例图片执行未接入的原因，补齐必要接线，验证三个官方图片方法的选择入口、权限和明暗主题，保留已有数据与配置、不重复派工；随后明确“先使用百炼”，最终指定 `qwen-image-3.0`，并授权本会话决定地域与试用点数。服务商选择替代原运行接入对 GRSAI 的假设，但不撤销点数、权限、人工确认、UNKNOWN 恢复或素材保存合同。

继续沿用 [完整图片 V1](product-image-agent-v1.md)、[官方方法 V1](official-skills-and-image-method-v1.md)、[企业智能体配置](organization-agent-configuration-v1.md) 及其 Figma `tg48P46SSXl6TBy9lZwg63` / `31:463` / 图片方法节点 `6861:1826`、`6861:1864`。不新增页面、技能类别或付费调用入口；本增量改变 provider 接线，选择器仍使用已批准设计。

使用者从已有合法商品素材上下文打开整套图片面板，选择标准商品套图、场景化商品展示或卖点详情图编排的准确版本；准备阶段显示百炼实际路由和明确点数，只有原人工确认才能开始生成。未完成配置时显示具体不可用原因，不伪造选择成功或执行可用。

Must：显式百炼路由；真实授权素材/准确方法进入原指纹；计划、报价、生成意图和实际 dispatch 的 provider/model/protocol 完全一致；原权限、企业凭据、点数及成员月限复验；一次派发与原 UNKNOWN/结果恢复；现存数据、配置、凭据、任务与卷保留。

Out of Scope：真实模型调用、另设点数账本或默认价格、跨企业/全局密钥 fallback、个人图片直接执行、新任务调度、异步 provider 协议、平台发布、供应链门控变更、技能创作助手、迁移/清理、通用 provider 框架或新验收工具。无新增 Accepted Risk。用户/生产验收继续 NOT_RUN。

## 当前根因与最小改动

保留实例启动清单没有 `imageAgent`，故完整 HTTP/Worker/Asset 未装配；Image DB 尚未安装当前运行表和企业图片凭据表；缺少私有 worker、图片存储和每张点数价格。原 source acquisition pool 已有必要的标题 proposal SELECT，不扩大该读取权限。

代码在 `generation_provider.go`、`generation_quote.go`、`generation_execution.go`、`generation_fact.go` 与 `image_set.go` 多处直接固定 GRSAI。把供应商身份写死在通用意图/报价校验里造成接线耦合，单改 URL 会被拒绝；逐处放开任意字符串又会削弱原准入。

最小处理是明确支持第二个有限路由配置，统一验证其 provider/model/protocol 元组，由受信配置元数据向原报价及意图传播。现有 GRSAI 路由仍是当前已批准能力，不能把其历史回执改写为百炼；本轮不设计运行数据兼容/迁移，不引入未知 provider 或自动 fallback。

## Owner 与接线

| 合同 / owner | 实现与注入 / 消费者 |
| --- | --- |
| 当前企业 credential owner | 沿原 `image_gpt_image_2` 准确企业配置槽；百炼明确 style、model、HTTPS base 和 credential reference，不能复制标题密钥或依赖宿主 CLI 密钥 |
| ImageSet / Generation | 原 quote、plan、intent、effect 表及 JSON 字段；受信 factory 元数据新增或携带明确路由元组，原 fingerprint、确认、worker revalidate、候选采用逐项匹配；无新表/状态机 |
| 百炼 HTTP 集成 | 新窄 adapter，实现既有 route-bound image generator；复用现有 ProductImage adapter、prompt、素材验证和 SSRF/有界读取能力；不建另一套商品图片工作流 |
| Resource / 成员月限 | 原 orgresource reserve / observe / settle / release 与 UNKNOWN owner；价格仍由维护配置明确提供，不从人民币价格自行换算 |
| 字节和 Asset | 原不可变 artifact store、GenerationSuccess observation、物化及 Product Asset owner；不另存“百炼任务”或正式图片事实 |
| 当前应用装配 | 沿原 FullImageSet HTTP/worker/配置/Asset/Temporal 端口，不启用完整 SupplyChain 来间接打开图片 |

当前实例只有已保存集合/Preparation 来源，没有单商品采集 operation。运行前须确认现有 full image source reader 能消费其合法原来源；若需把通用 Preparation 来源从完整平台 Supply worker 中拆出，只提取既有来源读取/当前 IAM 端口，保留现有 `supply` context、原 source ID/固定版本与来源权限，不创建假采集 operation、重复商品或第二事实源。若无法在这些现有合同内完成，保持该入口 NOT_RUN 并报告，不临时新增 source kind/发布或授权模型。

运行接线明确采用现有 Preparation source reader：从 Product DB 读取 `preparation.SourceSelector` 的原来源，配合现有 `supplychain.OrganizationExecutionAuthorizer` 的来源权限检查和 EffectiveImageProductReader，先显式执行 `sources.WithExecution(executionAuth, collection.ExecutionOwnerAuthority{Authorization: executionAuth})`，再把该 selector 的 `SelectForExecution` 与相同executionAuth注入 `ImageSetSourceRouter.Supply`；不得只传ExecutionAuthorization而保留未装配selector。读取模式用原 `preparation.PermissionRead`，新的准备/执行仍用原 execution-source 权限。平台 rule/target ports 仍只在完整 SupplyChain 已配置时提供，通用 `product` target 不要求 Supply worker。不能新增来源ID、假operation或改变actor/tenant/store权限。

现有本地待补全详情消费同一 ProductImageSetPanel（已有 Figma 方法选择组件），使用真实 `kind=supply` / 原 sourceId。只有当前可读合法来源且具备原 image write 的用户能选择方法；方法候选读取独立于图片模型 readiness，准备按钮仍要求原 AVAILABLE、有效模板、真实素材、模式和报价。缺少凭据时可核对方法准确版本，界面明确“模型配置尚未就绪”，不能生成/准备Run。切企业清空选择；不新增默认方法或持久化设置。

实现细化：现存合法 Preparation 来源没有原图时，read-only description 如实返回空 originals 与原来源绑定；PrepareImageProduct 仍拒绝空原图，不改变生成素材合同。`SourceSelector.WithExecution` 会修改 receiver，装配须复制 selector 后调用，以保留既有浏览器来源端口。

冻结路由元组：`provider=bailian`、`model=qwen-image-3.0`、`protocol=dashscope-image-sync-v1`，分辨率 `1024x1024`、quality `auto`。由受信 factory 同时提供给 ImageGenerationQuote、GenerationProviderMetadata、新 GenerationIntent/fingerprint、operation provider/model 比较和派发前 revalidate。领域只允许该完整元组与当前已批准的 `grsai/gpt-image-2.5/grsai-json-sync-v1`，交叉混配和未知值均拒绝；不按URL猜provider。未改变旧意图的准确字段/回执。

## 百炼协议边界

依据阿里云[千问图像生成与编辑接口](https://help.aliyun.com/zh/model-studio/qwen-image-generation-and-editing-api-reference) 和本地实际原文 `bailian-docs-llm-wiki/raw/model-api-reference/image-generation/qwen-image-api-reference/qwen-image-generation-and-editing-api-reference.md`（2026-10-18核对）。按用户最终选择采用 `qwen-image-3.0` 的 DashScope 同步图像编辑，固定每次 `n=1`、请求 `1024*1024`、无水印，显式关闭 `prompt_extend` 及 `enable_thinking`；不接入异步查询/重试协议，不使用 multipart 或假定 OpenAI edits。候选模型仅此一个；已放弃的万相和图片翻译模型不进入执行路由。

百炼 region 与 credential 必须来自同一获准业务空间。只允许官方 HTTPS host：北京通用域名 `dashscope.aliyuncs.com` 或单段业务空间ID的 `<workspace>.cn-beijing.maas.aliyuncs.com`；当前路线拒绝intl、新加坡及其它地域；不接受用户信息、query、fragment、非默认端口、模糊后缀或转发到另一 host。POST 路径固定 `/api/v1/services/aigc/multimodal-generation/generation`。重定向禁用、transport 不自动重发 POST、timeout 最多5分钟；不使用 CLI 子进程代替应用 provider。

将已授权并冻结 hash 的真实 JPEG/PNG 字节按 data URL 传入 `input.messages[].content`，与冻结 prompt 同请求。参考图最多3张，单张不超过10MiB、宽高1–8000且宽高比不超过10:1；沿当前素材总字节上限取更严格值。提示词按当前完整输入的有界合同校验，超过本地限制时派发前明确拒绝，不静默截断方法或丢参考图。官方推荐4500Token以内；本轮不额外调用模型估算token、不声称字节/字符数等于token数。请求体/响应体有界，不把正文、base64、Authorization 写入日志/错误/API/checkpoint。

响应先有界解码，成功必须同时满足：HTTP 2xx、没有错误code、有效request_id、恰一个choice且finish_reason=stop、assistant消息恰一张image结果、usage.input_image_count等于实际发送参考图数、output_image_count=1且输出尺寸为1024×1024、公开HTTPS输出URL。Qwen3同步协议没有output.finished；若响应额外给出明确false或异步task状态，拒绝当成功。使用Qwen3的output_image_count而非旧模型的image_count；缺失/矛盾/非终态保留UNKNOWN。随后沿原观察回调持久化 request ID、结果摘要与私有结果定位，再使用原下载/媒体/1024尺寸验证和不可变存储。provider 对尺寸允许误差，应用仍按现有实际输出尺寸合同拒绝不符结果，不伪造1024元数据或暗中调用放大模型。

百炼按张计费，token usage 缺失不能伪造0 token；Generation 的固定每张点数与实际 provider usage 两种事实继续分开。适配器向原 receipt 传递可证实的调用/图片数量和 request ID，token 字段无证据则维持未知。价格/credential reference/route reference/配置版本变化使原新派发 revalidate 失败，不能把新凭据用于旧已冻结意图。

## 副作用与恢复

沿原 ImageSet confirm → 原准入回执 → worker → Resource reserve → 原 unique effect claim → provider POST → GenerationSuccess observation → 原下载/物化/采用路径。原 Skill 方法锁和来源/当前成员权限栅栏保持；未选方法继续原合同。

POST 交给 transport 后的超时、连接断开、响应丢失、不完整 JSON、无法证明结果的非2xx均不能自动重新提交；保留原 UNKNOWN 事实/预留与核实入口。明确派发前的参数/权限/预算失败不消耗调用；不能仅凭收到错误字符串宣称 provider 没有执行。成功观察已保存但下载/进程退出失败时只沿原结果 URL 恢复下载/物化，不再次生成。观察回调失败也不能重发 POST。相同键换 payload/route/method 与原 digest 冲突；确认/采用回执仍优先恢复。

不新增自动 provider 对账、跨库事务、补偿系统或新的 UNKNOWN owner。供方没有可证实的读回入口时保留人工核实，不能以“没查到”释放未知成本或自动重发。

## 保留实例与停止条件

本设计和隔离实现不授权重跑 retained initializer。原 Compose/.env/私有文件不覆盖；有配置值后采用新的受保护配置文件与最小 override，保留原回退。当前 Image DB 已有当前 `ai_invocations`（0行），因此不能通过删除表满足原 empty-schema initializer。新增显式、有界的 ledger-only owner 准备入口：仅允许确认同一当前 Image owner、除该当前表外无其它业务表/状态、ledger当前结构验证通过；原ledger逐行保留且不alter/迁移，事务内仅创建缺少的当前ImageSet/Asset/企业图片凭据表。已有完整schema使用Verify路径、不重装；混合/未知表或结构拒绝。该入口不在普通启动执行，也不运行宽迁移器或清理。窄HTTP/worker/Asset角色沿原grant命令，数据库凭据仅私有文件。对象存储与企业凭据录入分别记录具体配置、授权和前后证据，不重装官方技能或修改其它业务DB。缺少这些前置时不能让 current application 启动时补表或自动启用企业。

**按用户“你决定”的授权，使用北京地域、每张1 AI点的内部试用价格 `bailian-qwen-image-3-trial-20261018-v1`。** 该价格是明确的产品试用配置，不代表百炼人民币成本，也不更改标题价格或现有余额。业务空间与现有授权凭据须在原配置入口确认；缺少前置则保持生成关闭，不复制宿主CLI/标题密钥。保留实例实际三方法执行选择/计划/生成/采用如未真实执行，一律 NOT_RUN。不得通过新造商品或伪 Run 提供验收截图。

Legacy decision: N/A；本次消费当前 ImageSet/Generation/Skill/Resource/Asset 合同，不依赖或恢复 legacy task/worker、旧试验 overlay、历史迁移或第二事实源。

## 验证与准入

独立检查只针对本次新增百炼路由、副作用和上述接线边界；两轮停止规则及 finding 分类按 AGENTS，不重审未变化的完整图片架构。达到 IMPLEMENTATION_READY 后才正式实现。

TDD：先以受控 HTTP fixture 捕获百炼 route 被拒、元组错配、n/size/原图/prompt 丢失、重复POST或错误结果观察；再补最小 adapter/元数据传播。覆盖正式host准入、超界零派发、只发一次、scope/配置变更拒绝、成功观察先于下载、响应丢失UNKNOWN、无 token evidence 不造usage、准确 Skill 及原 GRSAI 路由回归。复用已有 ImageSet/Resource/Asset fixture，不新增 runner。

实际实例验证三个兼容官方版本可选且保留 ID/version/digest；个人图片、跨企业、失权/停用版本拒绝；两种主题和不使用方法路径；停在准备/确认前，不调用真实模型、不产生真实点数预留或新 Run。若准备需要写 Run 则本轮不点击准备，只验选择入口与权限。缺少合法当前上下文或配置则如实记录具体阻碍。

本批一个主要 PR，原会话/owned worktree 一个 Writer。设计检查、隔离测试、PR CI、retained runtime 和用户验收分别记录；不自动合并/关闭 Issue，不把旧 #795 CI 冒充本增量证据。

## 独立检查处理

Round1 reviewed 703f8615dd。路由元组未冻结 finding 分类 BLOCKER（核心准备/派发身份可能互拒，实施层）：已在上方冻结准确三元组和所有消费位置。百炼成功终态 finding 分类 IMPLEMENTATION_TEST：按用户最终Qwen3协议明确finish_reason、output_image_count和尺寸成功谓词，fixture覆盖缺失/矛盾/非终态；不要求不存在的Wan finished字段。

用户最终模型变更为qwen-image-3.0、北京及试用每张1点已纳入本候选。Round2只复核上述合同修订及必要Preparation/受保护当前ledger准备/UI只读选择边界，不重复#795未变设计。此历史候选当时尚未达到IMPLEMENTATION_READY，未写生产代码。
Round2 finding处置：selector未WithExecution为BLOCKER（通用Preparation核心路径失败，实施层），明确同原Supply装配一致的WithExecution和owner authority后再注入；北京路线仍容许intl为BLOCKER（偏离当前地域配置，实施层），冻结北京host并拒绝intl/其它region；input_image_count一致性为IMPLEMENTATION_TEST，加入实际发送数量成功谓词及fixture。两轮设计已完成，仅对两项新BLOCKER的最小修订请求增量复核，不新增设计版本链或全局评审。

当前准入：两轮及两项BLOCKER最小增量复核已完成，33fd71964未发现新问题，达到IMPLEMENTATION_READY；复核摘要见[PR #799](https://github.com/qq550723504/task-processor/pull/799#issuecomment-6246256153)。冻结上述范围；非Blocker转实现测试，后续正式实现沿本PR继续。
