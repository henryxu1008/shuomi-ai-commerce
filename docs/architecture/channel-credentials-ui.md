# 企业渠道 Key 的 UI 配置与轮换：最小设计候选

日期：2026-11-05（Asia/Singapore）。执行载体：#134 / #128；当前唯一 Writer 仍为原统一模型/凭据负责人 `/root`，原独立工作区。

**PRODUCT_DECISIONS_CONFIRMED / IMPLEMENTATION_READY / IMPLEMENTATION_AUTHORIZED**。Design Basis: **Independent Architecture**，原因是本次把已有受限维护操作开放到经过企业授权的浏览器入口，新增凭据管理权限、秘密请求与应用装配边界。不是纯表单改动。

2026-11-05 实施准入：用户已授权原唯一 Writer 在原工作区与 PR #1096 按已复核设计先补离线/隔离回归，再最小实现权限、Key 管理接口与原页面接线、自检并沿既有 Codex 自动评审复核。D1–D3 冻结；设计提交 `38046a3` 已完成自动范围内评审、0 finding，后续 `a94c76489` 仅文档索引增量及 CI 成功。准入由当前协调方按上述证据登记，不伪称自动回执包含正式 APPROVED vote、运行或产品验收。下文的候选/未授权描述保留原设计阶段语境，当前以本段和 Issue 当前阶段为准；实施不授权真实操作、合并、部署或关单。

2026-11-05 用户已明确回复 **“认可 D1–D3，继续仅设计”**。该确认只确定下述产品范围；不授权提交、推送、PR、评审、实现或真实 Key 操作。权限技术名称、HTTP 形状和装配仍是待适用架构复核的候选，不把产品认可写成 IMPLEMENTATION_READY。

用户随后单独授权 **提交/推送这三份设计 Markdown、创建 design-only PR 并转 Ready、核准确 HEAD 的正常 CI，沿既有 Codex 自动评审做一次范围内架构复核**。同一 Writer 维护范围内文档 finding，不新增人员/线程或并行手工重复评审。该授权替代前段产品确认时的提交/评审限制，**仍不授权实现、真实 Key 操作、实例/数据库/配置变更、合并/部署/关单、镜像构建或模型调用**。架构复核结果维护在 PR，未完成前不标 IMPLEMENTATION_READY；业务、浏览器和产品验收 NOT_RUN。

## 1. 用户结果、依据和本次边界

使用者是获本企业渠道凭据管理权限的管理员。操作场景：在现有「我的账户 → 企业空间 → 模型与费率」配置同一渠道的一份 Key，供本企业多个已准入模型共用，并在同一页面轮换或停用。

依据是用户本次明确要求：“渠道 Key 也要能在 UI 中统一配置和轮换，多个模型复用同一凭据。先……整理最小设计，明确企业隔离、权限和 Key 保护，暂不实现、不新增派工”。该决定开启新的 UI 管理阶段，替代以下旧文档中“本阶段仅命令维护、不提供网页 Key 编辑”的范围限制；用户随后认可 D1–D3，但**新接口/装配尚未复核或实现**。

- [共享渠道凭据](shared-channel-credentials.md)：唯一 Run DB owner、AES-GCM/HKDF、连接版本、CAS、去密回执、发送前最后门禁和原恢复。
- [企业统一模型](enterprise-shared-model-configuration.md)及[Eino 按需接入](eino-model-admission.md)：企业三类选择、可信连接/模型/用途准入及一套计费。
- [全新系统基线](../product/greenfield-no-legacy-migration.md)：不迁移、复制或重解释保留 Key/历史。
- UI/Product Authority 沿[当前账户中心与 UI/IA](../product/final-ui-ia-authority.md)、原「模型与费率」及本次明确产品要求。原账户 Figma `31:463`、模型版本节点 `7658:2087` 只作已有页面依据；本候选不冒称已有渠道表单原型或视觉验收，Figma 编辑 NOT_RUN。

源码依据固定为已读 main `ba2951f724ba64470b8238cd20d0a724d07864e8`；此时无同范围开放 PR，原 #1075 是另一负责人的安装/调研交付。本候选不修改其固定应用/工具版本、实例或安装窗口。未来实现须单独核定准确组合和准入，不能将本设计自动部署到它的保留实例。

本次不做：任意 URL/模型发现、厂商管理 API/真实连通性测试、自动撤销厂商旧 Key、OAuth/AK-SK 等新凭据类型、个人覆盖/跨企业共享、平台全局 Key 池、保护主密钥轮换、新凭据库/KMS/IAM/审计平台、保留库迁移或真实 Key 导入。

## 2. 现有能力与确需补齐的地方

| 当前证据 | 复用或最小补齐 |
| --- | --- |
| `channelcredential.Command`、`channelcredentialstore.Store.Publish` 已有连接唯一键、固定元数据、CAS、同库回执事务与加密 | 继续唯一 owner/两张原表，不复制写入引擎；浏览器只是同一应用服务的新消费者 |
| 原 CLI `product-agent-credential-provision` 的输入含私密 `writerDatabase`，校验绑定定义和受控 endpoint | 抽取/复用有效的连接准入与保存行为；**不能把该私密输入 DTO 整体开放成 HTTP API**。writer/保护材料只由服务器注入 |
| 原模型页面 `usePublish` 用 `useResourcePending` 将完整 `input` 写入 sessionStorage | 复用页面、非秘密状态和原请求键模式；Key 命令必须有只含去密引用的 pending DTO，禁止直接复用完整请求持久化 |
| Store 已有去密 Receipt 查询，Publish 内继续校验完整请求 HMAC；尚无浏览器原回执消费者 | 直接复用同一 owner 的 Receipt，只补应用/HTTP 的企业+actor+requestKey 定位及 connection 核验，不要求浏览器保留或重传 Key；不回传 fingerprint/ciphertext |
| 现有 `workbench.agent.configure` 只授权企业选模型；原合同明确不授 Key 管理权 | 候选增加一个独立窄权限并接入现有角色/权限目录；不把原选模型权限或角色名称自动提升为 Key 管理权 |

Legacy decision: **EXTRACT / RETIRE**。复用当前维护入口的合格校验/加密/事务/回执；退休的只是新 UI 中任何“模型同行存 Key、Image DB 复制 Key、每模型另录 Key”的方案，不恢复旧 `ai_client_credentials`。原受限 CLI 可作为同一 owner 的维护消费者保留，不能出现第二套事实或写入逻辑。

## 3. 已确认的产品决定（D1–D3，仅产品范围）

**D1 — 谁能管理。** 已确认仅明确获得本企业「渠道凭据管理」权限的管理员可以操作。技术权限名 `workbench.ai.credentials.manage`；本轮只在现有权限/角色目录注册，不自动授予已有角色或真实用户。沿已有有效企业、成员与角色授权链实施；只有模型选择权的成员不自动获得该权。平台价格维护身份不因维护价格而自动获本功能的跨企业权限。角色名叫 admin 也必须通过当前有效授权，不以原始角色名称放行。现有自定义角色目录增加独立的渠道凭据管理模块，只有经现有明确分配流程获得它的本企业管理者可操作；不附带模型、价格或通用管理员写权限。

**D2 — 秘密怎样显示与保存。** 已确认 Key 只输入、不回显、不导出，不显示尾号或普通 hash；页面只显示“未配置/已保存/已停用”、版本及去密回执。新录入和轮换均输入一份新 Key，显示保存结果不等于已验证该账号的模型访问资格。Key 不写浏览器持久存储，结果不明用原请求编号查回执。

**D3 — 可配置什么。** 已确认首阶段只配置当前企业服务器已经准入的 API-Key 渠道连接；可以为尚无凭据的已登记连接首次录入，再轮换/停用。同一连接由多个模型明确引用。网页不自建任意厂商/endpoint/型号，不增加真实连通性测试；若需要新连接身份或当前合同未覆盖的凭据类型，先经原可信定义/准入流程处理。

既有轮换合同直接复用：这是**更换本系统保存的 Key**；不声称代替厂商后台撤销旧 Key。通过最后发送门禁的原一次在途调用可完成；门禁前发现版本变化则拒绝旧准备，UNKNOWN/预留不因轮换改变。以上不是新增“立即撤销全部在途请求”的保证。

## 4. 页面入口与完整操作流程

沿原 `/workbench/account/organization/model-route-pricing` 页面增加「渠道凭据」区/选项卡，保留「企业模型」和原平台「参考价格与汇率」。按实际权限显示各区；仅有凭据权限者也能进入其区，不要求同时具有改模型或改价格权限。普通模型配置者继续看到现有就绪状态，不因本功能获得 Key 管理接口。

1. 顶部持续标明当前企业。渠道列表只读去密信息：已准入的厂商/渠道/地域、连接标识、账号备注、已保存/未配置/停用、当前版本，以及引用它的可见模型。账号备注不是已验证的厂商账号身份。首次保存后不可借轮换偷改连接身份。
2. 「录入 Key」或「轮换 Key」打开独立短对话框；固定展示当前连接和“影响此连接全部模型”。表单仅接收 Key；首次录入若原连接尚无账号备注，可收取原合同允许的非秘密账号标签。厂商、地域、endpoint、模型及数据库配置不由表单任意改写。
3. 确认区说明保存会更新整份连接的版本，不自动选择模型、发起真实请求或改变费用；轮换不会替用户撤销厂商旧 Key。确认后发送一次当前版本命令。
4. 成功清空秘密输入并关闭表单，返回同一列表并读取新版本。取消不发送；本地校验失败可继续修改未发送的输入。一旦发出，成功/失败/UNKNOWN 均清空受控 Key 输入状态，保留可修正的非秘密信息。
5. CAS 冲突保留为冲突，重新读取版本并由用户决定，不静默覆盖。结果不明展示「保存结果待核实」及「查询本次结果」，不自动重试或另造 requestKey 重发。
6. 切企业/登出/关闭表单/离开页面清空秘密输入；已发出的请求即使浏览器中止也不当作回滚。去密 pending 引用仍按原用户/企业/连接隔离，切回且仍获权后才能查询原回执，不把旧结果显示到新企业。
7. 「停用」单独确认影响范围，沿原 Store 保留加密 Key/提升 revision，不删除连接、调用或账务；重新启用在首阶段走再次输入 Key 的明确保存，浏览器不取回旧 Key。页面不提供删除/复制 Key 或“自动回退旧版本 Key”。

沿原组件、焦点返回、窄屏与可访问标签，不另建完整设置中心或多步向导。无对应数据时如实显示未配置/不可读取，不补演示数字、假账号或“连接成功”。

## 5. 身份、授权与企业隔离

所有入口使用原 ZITADEL 会话、verified identity、effective organization/member、有效期和 live 企业访问/权限核对；后端拒绝 body 中的 actor/org/writer/保护材料，不能以客户端选择、URL 或 Expected-* header 替代权威身份。Expected-* 仅检测页面上下文漂移。

读取/写入始终以 `(effectiveOrganizationID, connectionID)` 定位；同名连接在其他企业独立。receipt 再限定当前 actor 和原 requestKey；跨企业/跨 actor 不返回去密回执。撤权、退出企业、过期会话或权限依赖不可用即拒绝，不因页面原来打开过而继续授权。

复用 `AuthPolicyCurrentIdentity`、live 组织策略和 `AllowedOrganization`；新权限进入既有角色模块/身份投影/route descriptor 校验。BFF 和后端均设门禁，隐藏按钮不是授权。自定义角色只通过当前合格角色管理流程获得能力，凭据管理不授予模型执行、平台价格维护、角色管理或其他企业访问权。

## 6. 唯一 owner、数据库与应用注入

路径候选：现有页面 → 原会话 BFF 的窄 DTO/同源门禁 → 当前 HTTP 模块的企业凭据子路径 → 当前 channelcredential 应用服务 → **原 Run DB 的 connections + receipts 同一事务**。

服务器由可信 definitions 先产生当前企业可管理连接集合，逐项读原 metadata，不提供任意全表/跨企业查询。provider/channel/region/受控 origin 从可信定义解析；文本定义只含摘要时由私密 `Definition.connectionEndpoint` 明确 URL，或使用同连接可信图片定义中的准确 endpoint，须匹配所有文本摘要/适配器及图片 endpoint，不能反推或猜测。新输入账号标签只作为原有限元数据。沿原 `ValidateChannelCredentialProvision` 的有效绑定/endpoint 不变量准入，不能直接接受私密 provision DTO 的 writerDatabase、BaseURL 或组织身份。

复用 `title_credential_writer` 的原窄权限与保护服务；当前它只从 CLI 私密输入注入，未来需要服务器端受控的管理 writer 装配，**浏览器不提供 DSN/角色/保护 Key**。普通 `product_agent_runtime`、`model_route_config_writer` 不新增 ciphertext 或 Key 写权限；handler 只持有窄管理接口，不暴露 secret resolver。

管理写服务与执行 reader 的装配分开，保护材料仍一个受控来源。关闭 AI 时可保存管理配置，但不因此打开执行 secret-reader 池、创建 provider/worker 或调用模型。缺 writer/保护材料时只读提示不可保存，不降为明文或给普通 DB pool 提权。若准确候选依赖 #1075 尚未合入的关闭态装配，先标明组合依赖，不复制该负责人的代码或靠启用 AI 绕过。

不新增表、凭据副本、Key 同步或第二计费。CLI 与 UI 共用同一当前服务/存储协议；保护主密钥仍只由部署维护，网页不能读取或轮换它。

## 7. Key 保护与有限 HTTP 合同（当前源码接线，真实环境未启用）

| 候选入口 | 内容与边界 |
| --- | --- |
| `GET /api/v1/model-route-pricing/connections` | 当前企业已准入连接与去密状态，原列表上界；不返回 Key、密文、保护材料或请求 HMAC |
| `POST /api/v1/model-route-pricing/connections/:id/commands` | `set-key` 或 `disable`、expectedRevision、原 Idempotency-Key；仅 set-key 输入 apiKey。actor/org、连接准入元数据和 writer 均由服务器确定 |
| `GET /api/v1/model-route-pricing/connections/:id/receipts/:key` | 当前企业+actor+原 key，并核 connection 的去密回执；不要求再次输入 Key、不携带秘密 fingerprint |

上述路径在同一原 HTTP/BFF 模块接线；`expectedRevision` 与回执版本使用十进制整数字符串，不经 JavaScript 浮点转换。DTO 有界严格解析，复用当前 16 KiB 写入请求边界、Secret 4096 字节及原命令校验；未知/重复字段拒绝，固定 metadata 响应白名单。同源/CSRF 检查、受信后端 origin、无重定向/自动重发与现有超时继续复用；真实环境使用现有 HTTPS 身份入口。本源码交付不代表保留实例已更新或获得真实保存授权。

Key 只经受信 HTTPS 请求短暂进入 BFF/后端和当前保护服务；沿现有 AES-GCM，AAD 绑定企业/连接全身份/revision，HKDF/HMAC 继续原完整命令幂等。不新造加密算法，不把 Key/hash 放入 URL、日志、错误、分析事件、审计正文、快照、点数账本或回执。禁止响应回显、显示旧 Key、导出或让前端接收密文。停用所需的内部解密/重封装仍封闭在原 Store 内。

页面响应 `no-store`，Key 不进 localStorage/sessionStorage/IndexedDB/持久 Query cache。原 pending 模式只可存 `{requestKey, userID, organizationID, connectionID, expectedRevision, action}` 这类去密引用，schema 明确拒绝 apiKey；待发送 body、错误和调试对象不进入通用日志或表单持久化。浏览器只在受控内存中保留输入，清理不等于能保证 JavaScript 运行时所有字符串副本被物理擦除，不作此保证。

## 8. CAS、UNKNOWN 与轮换效果

继续原连接 current+去密 receipt 同库事务、原锁/expectedRevision 和包含秘密的服务端 HMAC。重复同 key/同完整命令回原 receipt，不同 Key/载荷冲突；接口不得在遇到并发冲突后替用户改 expectedRevision。

提交回执丢失：浏览器清空 Key，仅查询原 requestKey。未找到回执不证明未提交；保持待核实，不自动重发、不换 key、不凭当前 revision 猜测哪次命令成功。原用户主动再次输入的相同命令也须遵守原 key/载荷协议；不能为了恢复表单另造秘密持久化或恢复服务。

轮换影响 `(org, connectionID)` 的全部模型，无须逐模型另录 Key或重写默认选择。A→B→A 仍有新 revision；旧准备按原最后门禁拒绝。已通过门禁的在途原调用、历史回执、UNKNOWN/R、ObservedOverage、原键恢复和可信观测后的本地结算全部按原 owner，不自动换 Key 重发、释放预留或清历史。

保存 Key 不自动启用模型、不校验真实账号模型权限、不产生模型调用或新计费。后续正常调用继续原价格/FX、用途准入、预算/共享连接限额、单一 Invocation/orgresource 账本；配置成功仅表示本地安全保存。

## 9. 最小未来验证与停止条件

设计阶段只核文档和源码；当前已获实施授权，沿现有 Go/PostgreSQL/HTTP/BFF/Vitest 执行离线/隔离回归，不执行真实浏览器或业务操作：

- 两企业同 connectionID、伪造 org/actor、缺独立权限、只有选模型权、撤权/企业切换、越界回执读取均拒绝；真实角色不在测试中改动。
- 同连接多模型共享 Key；轮换/CAS/同 key 异秘密、A→B→A、停用及原门禁/UNKNOWN/费用恢复保持，原已有有效证据按未变代码复用。
- 合成 Key 不能出现在任何响应/去密回执/日志/浏览器持久存储；刷新、关闭、组织切换和写 ACK 丢失只留去密引用并找回原 receipt，不重复发布。
- trusted origin、严格请求/响应、原 DTO 大小、超时/断开、错误不回显；不调用真实 provider，不新增全量故障平台。
- 管理态 AI 关闭、缺 writer/保护材料仍安全拒绝；普通 reader/writer 权限不扩大，保存不启用执行或扣点。

新的权限/浏览器秘密边界已在本轮适用架构复核中检查，**当前为 IMPLEMENTATION_READY，D1–D3 冻结，实施增量沿同一 PR 的既有 Codex 自动评审复核**。若确需改变唯一 owner、schema、旧恢复合同、预算/计费或供应商授权，停下报告，不在实现里临时加兼容、网关或通用安全平台。

## 10. 当前交付与后续授权

当前 Delivery Batch 包含上述冻结设计及已授权的最小权限、管理接口、原页面实现和必要隔离回归，始终维护同一 PR #1096，不另派 Writer/Reviewer。缺少的真实 writer/保护材料/现场资格保持 UNKNOWN；运行、真实保存/恢复、模型调用、浏览器与产品验收 NOT_RUN。#47 UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT 保持。

D1–D3 产品决定已确认；**实施、必要自检、提交/推送、原 PR 维护及既有 Codex 自动评审现已获单独授权**，不把实施授权扩展为真实环境或产品验收。后续真实 Key 录入、配置发布、部署/安装和真实调用不由本次授权覆盖。不重复请求相同产品决定，不自动推进实现或现场阶段。原共享凭据/多厂商与 #1075 的历史证据和当前实例保持，不把新 UI 写成已交付。
