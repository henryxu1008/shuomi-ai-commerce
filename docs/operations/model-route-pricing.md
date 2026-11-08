# 企业模型与官网费率

2026-11-04 阶段更新：[企业共享合同](../architecture/enterprise-shared-model-configuration.md#当前阶段决定参考价与估算2026-11-04)
已将本阶段收敛为平台维护参考价/估算参数，沿原配置版本用于报价与预算估算，并如实标为估算。
下文原“逐次适用官网价/完整分项/厂商完整费用硬上界才可执行”是历史实现合同，不能恢复为当前业务前置。
文字参考估算和按类别执行准入已在#1078工程接入；Flare 图片参考策略与原直连接线本轮已补离线实现，准确实现 HEAD 的 CI/最终工程复核见 PR #1078；真实配置尚未发布。
当前[Flare最小设计候选](../architecture/openai-flare-image-generation.md#本次设计候选2026-11-04)明确估算预留、基础用量结算和原UNKNOWN责任，
D1“缺输入细分时使用较高输入参考费率”已获用户认可、原 Reviewer 架构复核通过；实现不授权发布或启用，真实运行/产品验收 NOT_RUN。
原单一配置/账本、预算与点数上限、权限/租户、幂等及不确定调用保护保留；
不授权发布真实配置、修改凭据/实例、调用模型/平台或改变实际扣款/恢复语义。

当前依据：[企业统一模型配置](../architecture/enterprise-shared-model-configuration.md)、PR #1078。此前 #830 按用途、两个数据库分别发布及单计划 VIP 的方案已被本次产品决定替代；历史实现和执行证据仍按原 SHA/范围保留。

## 用户入口

### 企业渠道 Key 管理（PR #1096）

同一“模型与费率”页增加“渠道凭据”，可录入、轮换和停用本企业服务器已准入的连接。一份凭据供该连接列出的多个模型共用。只有单独获得 `workbench.ai.credentials.manage` 的企业管理者能进入；原模型配置、平台价格或 `admin` 角色名称不自动获得此权限。该权限通过现有企业自定义角色的“渠道凭据管理”模块明确授予，本交付不修改真实用户权限。

Key 仅输入、不回显或导出，不进入浏览器持久存储。提交后清空输入；保存不明时用原请求编号“查询本次结果”，未找到回执不证明未提交，不自动重发。轮换影响同一连接全部模型；停用保留记录、历史回执、在途调用与 UNKNOWN/费用预留。重新启用须明确输入 Key。保存不选模型、不测试真实连接、不调用模型或扣点，也不代替厂商撤销旧 Key。

服务器私密配置可选择 `channelCredentials.managementWriter`：仍使用同一 Run DB 的 `title_credential_writer`、独立连接池且 `maxConnections` 不超过 2，沿原保护材料和同一 credential Store/回执事务；不扩大普通 runtime、模型配置 writer 或执行 secret-reader 的权限。只验证已有 schema/角色，不安装或修复保留库、不改变集群角色连接限额。管理未配置时展示只读状态，不伪造可保存。

文本 binding 若只有 endpoint 摘要，需在可信 `modelRoutePricing.bindings` 中明确 `connectionEndpoint`；同连接也可复用已有图片定义的准确 endpoint，但须匹配全部文本摘要、适配器及图片 endpoint，不能猜测或由网页提交。关闭 AI 时管理 writer 可独立装配，不打开执行 secret-reader 或创建 provider/worker。浏览器 API 只接受 action/expectedRevision、首次账号备注和待保存 Key，绝不接收 writerDatabase、企业身份或任意 URL。版本为精确十进制字符串。

当前源码交付不代表任何保留实例已配置此入口；真实写入资格/保护材料保持 UNKNOWN，运行、真实保存/恢复、浏览器及产品验收 NOT_RUN。实际安装、服务器配置与真实 Key 操作须另行授权。

### 企业模型与参考费率

我的账户 → 企业空间 → 模型与费率（`/workbench/account/organization/model-route-pricing`）。企业拥有 `workbench.agent.configure` 的成员在同一处选择默认渠道和文本、图片生成、多模态理解三类共用模型，所有成员共用。保存时按类别/准确绑定版本校验完整选择快照，在原CAS事务一次保存；执行时只对实际消费类别fresh核资格和预算。未开放生图不阻止合法文字选择保存，也不因此获得执行资格；不创建第二current或pending平台。

企业选择不包含费率或密钥。平台维护者在同一页面的“官网价格与汇率”维护原币价目、官网来源、核验及生效/失效时间、地区、服务类型及必要的图片尺寸/质量。非人民币价目使用已核验发布的 CFETS 中间价；人民币使用恒等汇率。价格由平台权限控制，发布还需要 `listingkit.admin.write`；企业配置权限不能写官网价格或汇率。

收费只有一条路径：原平台维护的参考价、获准策略的计量映射和冻结汇率，经原算费与资源账本折算点数，1点=人民币0.01元。页面标明参考/估算，不承诺供应商逐次账单精度；不叠加代理零售价或服务费，不填写独立AI点价或每张固定扣点。图片 D1 仅在准确 Flare 参考策略中生效，其他现有策略不由文字/图片候选自动放宽。

1 AI 点 = 1,000,000 微点。原资源 owner 的余额、债务、成员月限/月计数、预留、实际结算、差额释放、事件及回执明确使用 `ai-micropoint-v1`；购买/试用商品源数量仍为整点，只在原可信 grant 边界检查溢出并转换一次。重复 source/key 不再次增加。其它资源和人民币分的数量单位不改变。

点数通过十进制字符串传输：图片报价和原始 proof 使用整数微点字符串及明确单位，页面按百万比例精确显示 AI 点；余额、成员月限、事件与审计使用服务端格式化的精确点数字符串。前端不以 `Number` 或浮点计算费用。

发送前按获准策略的预计量ceil预留，终态按该策略所需可信用量floor一次；既有精确策略保持各自准入合同。结算可低于预留或为零，差额沿原debt-first事务释放。缺基础用量、矛盾或金额超R继续UNKNOWN/保留R，不自动补扣或扣尽R；缺精细分项按对应已准入策略处理，不伪造零。原finalize key/proof、RecoverDue和输出恢复责任不变，不因改价或改模型重发。

业务页只读显示企业已配置图片模型和预留费用估算，不另选VIP或修改点价。原冻结计划、请求键、回执和 UNKNOWN 保留；不跨配置继续采用旧模型绑定。历史结果找回不借新模型生成。

## 新空库准备（仅说明，不是环境操作授权）

复用原 ProductAgent/Run DB 的 revision/current、原 CAS 和回执表作为唯一配置事实 owner。私密 manifest 使用 `modelRoutePricing.writer`，数据库必须匹配原 ProductAgent/Run owner；固定 `model_route_config_writer` 非 owner/非 DDL 角色，writer pool ≤ 2。不再配置文字/图片两份 writer/current，不同步两库，不从旧 per-operation、Image current、member 或进程默认配置兜底。

沿原安装器明确安装原配置关系、去密 credential metadata view 和窄权限；运行只验证，不 DDL/seed/发布。可信 `bindings` 是准确厂商/协议/地区/服务类型/费用预算/原更小token、字节、期限的准入描述，含准确reference/credential/endpoint、策略适用证据版本及price ID，不能充当第二份用户收费价目。文本描述的 shape 准入不要求填一个伪造官网价格；正式执行的完整 profile 仍必须持有 use-time 官网价/FX。

同一共享 reader 注入标题、内容、平台资料、选品、诊断、Chat 规划及图片 API/worker。各用途保留原 prompt/schema、调用次数和更小限制。渠道 Key 统一由当前 Run DB credential owner 管理，图片侧沿窄读取接口使用同一连接；图片副作用事实仍由原 Image owner 管理，不另存 Key、不扩大真实权限。

`imageAgent.generation` 仅包含 `enabled`、`unitVersion: ai-micropoint-v1` 和显式 `maximumMicropoints` 准入上限，不是价格。旧 `priceVersion`/`pointsPerImage` 不会产生可执行报价；API/worker 同一上限的一致性检查保留。缺共享current、适用参考价/FX、使用协议或该策略估算参数时拒绝新执行；响应缺该策略基础用量则UNKNOWN。原终态恢复不依赖新current，不将厂商完整硬界调查恢复为Flare前置。

已有单位缺失、混合或不完整的资源库不升级、不修表、不换算、不重新解释；明确拒绝。本文和实现均不授权安装、迁移、真实启停、真实配置发布、授予或结算保留数据。

## 当前未准入项

- 默认准确型号仍为文本/多模态 `gpt-5.4-mini-2026-03-17`、生图 `gpt-image-2.5-flare-2026-09-08`；界面可显示其未准入身份，不把常量写成实际 current。
- Flare 参考图片策略、适用估算参数及原直连装配按已冻结 f35 设计实现；仍须准确实现 HEAD 的 CI 与独立工程最终复核。不改默认/参数/预算，不把参考量称为厂商硬界或按张固定收费。
- C2-2已有准确SHA的限定消费链路与工程复核证据；现场绑定/准备和真实使用不能从源码或文字可用推定，运行/产品验收NOT_RUN。
- 文本包含已实现的受控OpenAI-compatible参考策略`reference-text-totals-v1`及原已准入映射；基础P/C/T须显式非负且一致，缓存细分不作参考策略前置。Google/Claude native缺相应mapper时仍未就绪，协议或价目存在不等于能结算。
- GRSAI URL-only 响应没有已准入的完整计费用量；现有传输组件不等于收费准入。新工作保持关闭，历史已记录结果按原事实处理。
- Qwen 已有 native 单图响应可验证输出张数和 1024×1024；仍须准确组织绑定、官方适用地区/型号/图片参数价目、FX、原上限及完整用量同时成立。离线合成测试不代表实际配置/真实运行通过。

发布前按原键保留原输入；ACK 丢失只查询同一回执，未找到或服务不可用都不授权换键重发。历史选择也须重新预览并重新准入。前端/离线自检、CI、独立工程复核和用户使用验收分开；运行、真实保存/恢复、浏览器、模型效果和产品验收保留 NOT_RUN，现场缺证据 UNKNOWN。

## Flare 图片参考估算的配置与恢复

平台原图片价目 `image.referenceEstimate` 保存预计输入/图片输出 Token、原操作范围、1024/auto 参数域、来源数量/像素/字节与提示词字节上限。只对 `openai-direct` / `gpt-image-2.5-flare-2026-09-08` / `reference-images-totals-v1` 生效。参数、参考价与 FX 经原发布版本和企业选择冻结，不填写 `UsageBound` 或虚构厂商硬界。缺参数不可执行；本说明不提供真实默认估算或发布授权。

原 R 向上、A 向下取整，均沿唯一微点算术与原资源账本。原页面费用预算投影同一 R/A；原 per-image 固定点价不会进入该策略。完整输入细分按对应参考价，缺细分按较高输入价；明确零基础计量可结算零，缺失不等于零。A>R 保留 UNKNOWN/R 和原证据，停止同企业同冻结绑定的新发送，不自动补扣、截断或重发；原键历史和终态找回继续由原 owner 处理。

base64 结果沿原不可变恢复包保存，不能把存储 ACK 丢失当成未调用或退款依据；坏图在可信用量满足 A<=R 时仍正常结算。仅恢复原事实/原对象，不调用 provider 重新生成。实际价目/估算、FX、凭据与发布状态 UNKNOWN；运行、真实保存/恢复、浏览器及产品验收 NOT_RUN。


## Eino 多厂商按需接入

沿[冻结用途设计](../architecture/eino-model-admission.md)，仍在原企业三类模型页选择，不增加个人模型设置或网页 Key 输入。
模型和协议在受限配置的原 Definition 中声明，连接只保存企业渠道身份和秘密；同一合格连接可被多个模型引用。原BoundedAdmission按企业+ConnectionID共享10并发、5次/秒、burst15；不按模型引用增发额度，缺连接身份不回退为模型名限流。

| 用途 | AdapterKind | AdapterPolicyVersion | UsageMappingVersion |
| --- | --- | --- | --- |
| Gemini 文本 | google-generate-content | eino-gemini-v0.1.36-text-v1 | gemini-generate-content-usage-v1 |
| Gemini 单图意见 | google-generate-content | eino-gemini-v0.1.36-single-image-v1 | gemini-generate-content-usage-v1 |
| Qwen 文本 | qwen-compatible | eino-qwen-v0.1.9-text-v1 | qwen-text-v1 |
| Qwen 单图意见 | qwen-compatible | eino-qwen-v0.1.9-single-image-v1 | qwen-text-v1 |
| Claude 原生文本 | claude-native | eino-claude-v0.1.26-text-v1 | claude-text-v1 |
| 合格 OpenAI-compatible 单图意见 | openai-compatible | eino-openai-v0.1.13-single-image-v1 | openai-text-v1 |

上述新增用途只消费同一已发布 reference-text-totals-v1 参考费率，原 Mini 单图准确策略及 Flare/Qwen 生图仍走各自已准入用途和同一微点账本。
Claude 基础输入总量由普通输入、缓存读取与缓存创建的显式计数合并；缺任一组成不能按SDK默认零结算，继续UNKNOWN/保留预留。Google由显式总量与输入推导含thought的输出；无有效原始证据不按SDK归零。
Gemini 的可信 ModelProfile.ThinkingMode 必须显式为 low、minimal 或 budget-zero，并只给声明支持该参数的准确型号使用；例如当前3.8使用low，支持关闭思考的2.5用途可选budget-zero，不按名称自动猜测或回退。最终wire只允许对应thinkingLevel或零thinkingBudget之一，maxOutputTokens包含thought且仍受原上限约束，见[Google官方协议](https://ai.google.dev/gemini-api/docs/generate-content/thinking?hl=en)。
单图用途显式提供原 ModelProfile.ImagePromptTokenEstimate 参考估算，须留在原 MaximumPromptTokens/费用/点数上限内；Mini 保留原 3001/高细节策略。
共同 SingleImageInput 只携带准确来源/批准身份、内容摘要、MIME、字节与尺寸；不再携带 Mini 专属 detail/计量字段。来源图不冒充批准素材。

每个已开放文本消费者必须有对应的准确 operation profile，保留其 prompt/output 合同；只配置标题不能让其他文本用途假装已准备好。
存在渠道/厂商型号差异时，Definition.manufacturerModel 必须显式指向同一原价目中的厂商型号；Profile.ModelID 仍是发送给渠道的准确型号。
本批显式别名用于已消费的文本/单图用途；Flare/Qwen生图保持自身原准确型号合同，不借别名开放新生图型号。
原冻结 tariff 的 model 保留厂商计价身份，wireModel 绑定渠道型号；报价、最后门禁与结算复用同一组合，不自动去前缀、发现型号或开启备用模型。
模型厂商来源当前支持 OpenAI、Google、Qwen、DeepSeek 和 Anthropic 的已声明官方域；其他实际需要的协议/认证仍需沿原 owner 最小补齐，不能伪造可用目录或免费价格。

本段只描述源码交付和配置合同，不代表保留实例已更新、配置已发布或真实调用已验收；正式新安装仍用原初始化入口，不迁移保留库或建设第二安装器。
未准备好依赖继续显示原因；UNKNOWN、原预留及当前合同内原键恢复不因换型号/换 Key 自动改变。真实调用、效果、浏览器及产品验收保持 NOT_RUN。

## 原单图调用的明确拒绝、未计费恢复

沿[已批准恢复设计](../architecture/own-intake-provider-unbilled-recovery.md)，受控命令只恢复配置指定的原调用与原预留。它没有模型调用、商品保存、schema 初始化或通用退款入口。

入口为 `go run ./cmd/model-invocation-recovery -config <本机私密配置绝对路径> -action inspect`。默认 `inspect` 使用只读连接，输出原调用、预留、绑定摘要、证明就绪状态和已确认结算状态。配置文件必须通过现有私密文件权限检查，严格 JSON 限 16 KiB；不得提交到 Git 或打印数据库密码。

配置固定 `organizationId`、`memberId`、`invocationId`、`reservationId`、`expectedDispatchFingerprint`、`operatorId`、`approvalEvidenceId`、`observedAt` 和 `evidence`，数据库仅使用 `invocationDatabase` 的 `product_agent_runtime` 与 `resourceDatabase` 的 `commercial_owner_runtime`。不提供退款数量或模型凭据。受保护操作者先核验原请求关联、单次发送、明确拒绝及未计费来源，再登记来源摘要；摘要本身不能替代来源证明。HTTP 429、汇总零用量、账单已支付或单条控制台错误均不足以单独准入。

资格检查复用原模型路由 owner 的只读表授权合同，保留运行角色对已发布模型配置的合法读取。不要为通过检查而撤销正常路由读取权限，也不要临时扩大角色权限；路由写入、授予权限和无关关系访问仍须拒绝。准备命令与只读 inspect 使用同一资格边界。

只有另行获准真实账本恢复且证据齐备后，才把只读核对得到的 `proofDigest` 固定到 `expectedProofDigest`，显式设置 `resolveEnabled: true` 并使用 `-action resolve`。操作键由原调用派生，重复命令、并发恢复和原 owner 的扫描器复用同一原预留与 A=0 结算。原计费 owner 仍按原月份、债务优先及原事务处理；失败或响应丢失后检查同一身份，不创建新调用、不重发模型。缺证明保持原 UNKNOWN 和预留。

已有合格 52 列原调用库需要单独批准的显式 schema 能力准备 `go run ./cmd/product-agent-runtime-prepare -target invocation-provider-rejection`。该目标只增加可空证明列及原 runtime 的精确列权限，不写原调用或账本，也不由服务启动自动执行。新空库沿原准备流程使用当前完整合同。部署必须配套当前合同的程序；已经写入证明后不得直接回滚到不认识该列与终态的程序。

本次只完成离线实现和独立隔离验证。试用库准备、真实恢复和后续模型调用均未执行；原请求缺少供应商明确未计费证明，继续保持不可释放。
