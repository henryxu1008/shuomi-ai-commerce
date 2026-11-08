# #1159 OpenAI Mini 单图接入

关联 [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158) 与 [#1159](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1159)。源码基线 `40c2e716530b93b77cf02542e8b6b39ead7d6469`。本设计继承 [#1159 独立架构](own-product-ai-intake.md)、[Mini 单图计量](product-image-visual-analysis.md) 和[企业共享配置](enterprise-shared-model-configuration.md)，增量审查 OpenAI 直连、配置顺序和双预算。评审状态见文末。

## 用户结果和适用范围

试用企业的成员可从「添加自有商品」提交说明和一张明确选中的图库图，查看报价并确认，得到可编辑候选。保留一轮一次模型调用、人工采纳、原 Collection/Catalog 保存、同键回执和 UNKNOWN 处理。图像归属由 SourceMedia 验证。试用验收仅用明确标记的一件商品，核对黑 M、黑 L、白 M、图片归属与库存 0/空。设计仅覆盖 OpenAI 全球标准同步直连、指定快照和 #1159 用途。

## 配置方式

1. 平台用原 `productAgent.ownIntakePolicies[试用企业]` 和 `modelRoutePricing.bindings` 声明仅该企业、该 operation 的定义。操作为 `product.own.intake.generate`，连接固定为 `openai / openai-direct / global / standard-sync / https://api.openai.com/v1`，准确快照 `gpt-5.4-mini-2026-03-17`，组件 `openai-compatible`、策略 `openai-mini-vision-v1`、用量映射 `openai-text-v1`。绑定原 #1159 prompt/schema、ConnectionID、价格 ID、定义引用版本和 endpoint 摘要，并显式填真实 `BoundEvidenceVersion`。严格 `openai-text-v1` 在 SharedCatalog 和 SharedModelService 都校验该字段。版本指向经复核的 Mini 单图依据与本用途上界证据，覆盖准确模型、detail=high、3,001 图像 token 和 24k/4k，不使用占位值。价目与 FX 在原 profile 中各自冻结。用已批准的单图协议限制，固定 `detail=high`、`max_completion_tokens=4000`、reasoning_effort=none、service_tier=default、store=false、n=1、非流式、无工具。调用使用原 SDK/guard。原二进制、像素和总 wire 的更小上限一起生效，Base64 只计 wire 字节。
2. 用同一 Run DB 的现有加密连接机制登记企业项目 Key，固定写入角色 `title_credential_writer`、执行角色 `ai_channel_secret_reader`，用原保护材料，池各 ≤2。可信 definition 显式提供 `connectionEndpoint` 及摘要，登记后取得实际回执/版本，执行绑定从当前凭据元数据取得准确版本。企业管理入口要求 `workbench.ai.credentials.manage`。只读检查发现本地试用实例的 3 个既有百炼绑定均缺 `connectionEndpoint`，直接开启管理 writer 会因全量定义校验失败。首轮采用现有 `cmd/product-agent-credential-provision` 私有受限入口登记该企业 OpenAI 连接，管理 writer 暂不装配。企业 UI 入口只有已声明连接均通过现有管理定义校验后方可启用。目录和价目解析仅使用元数据，当前 #1159 Prepare 复用执行 resolver，会在服务端调用同企业 secret reader 核准当前连接并取得 Key，派发前重新核验。Key 留在受限服务端流程。检查 schema、角色与保护材料，真实凭据权限/有效性另验。
3. 平台在原「官网价格与汇率」发布同一快照及渠道的单一适用 USD 价目和 USD→CNY 汇率。价目单位每百万 token，输入/缓存输入/输出分别为 US$0.75/0.075/4.50，字段对应 `750000/75000/4500000` USD 微元。图像 token 已在输入总量，沿 `openai-text-v1`。汇率按 [中国外汇交易中心中间价](https://www.chinamoney.com.cn/chinese/bkccpr/) 归一为 CNY/USD，填写 `cnyMicrosPerCurrencyUnit`、公告日期、来源、核验时间和生效/到期。发布价目与 FX 使用既有 CAS/current，要求唯一生效版本。
4. 企业以 `workbench.agent.configure` 通过「模型与费率」选择多模态绑定，仍保存原三类模型快照。其它两类可选已声明但未就绪的选项，其就绪不作为本操作前置。此后仅为试用企业启用 `product.own.intake.agent`，保留当前成员的 Agent 使用、本地写入和 Collection 管理权限检查。隔离启动须验证原 Asset owner 可作为独立 ProductAgent 池复用、连接容量与仅目标能力可执行。试用实例上一轮读到的 OpenAI 凭据、有效价目及 FX 为零，当前仍须有真实配置证据。

## 双预算如何生效

- 首轮定义最大输入 24,000 token，其中含一张 high 图的 3,001 token（2,500 patches × 1.2，加 1 token 余量），因此其余 20,999 包含系统提示、用户资料和请求封装。最大输出 4,000，一次调用。标准价、按全价输入时，(24,000×0.75 + 4,000×4.50)/1,000,000 = US$0.036。压力验算 FX 8.00 得 ¥0.288 和 28,800,000 微点（28.8 点）。¥8.00 是设计验算值。按这档限额理论临界约 8.333333 CNY/USD，实际准入用定点计算及适用舍入，避免用近似临界值放行。[OpenAI 标准价](https://developers.openai.com/api/docs/models/gpt-5.4-mini) · [图片 token](https://developers.openai.com/api/docs/guides/images-vision?api-mode=chat)
- **渠道成本门槛**依赖执行 profile 中独立的 CNY 费用字段与 `productAgent.costMicros=300000`。私有 profile 使用从适用 USD 标准价折算并向上取整的输入/输出每百万 token 的 CNY 微元值。在实际生效的价目与 FX 下核对两字段的保守性。试用以 6,000,000/36,000,000 CNY 微元每百万 token 作渠道保守费率，在 FX≤8.00 且价目为官方标准费率时给出最坏 ¥0.288。现有代码分别校验两套预算，未校验渠道费率是否覆盖实时 USD 价目与 FX。增加仅此操作和准确 Mini 直连策略生效的共享路由准入检查：按当前发布的 USD 输入/输出单价和 FX 向上取整到 CNY 微元，逐项不得超过绑定的 CNY 费率，且 profile 最大成本不超过 min(现有通用预算, 300,000)。`QuoteText` 使用实际封装输入上界（含图）和最大输出，`ownIntakeQuote` 使用冻结的 Run 限额。这使页面 CNY 报价覆盖按同一输入输出上界折算的 AI 点，汇率或费率漂移超包络即拒绝新报价与发出旧请求。预算不容纳时压低 token 限额并重新验证。
- **点数门槛**显式设 Definition.`maximumMicropoints=30000000`。SharedCatalog 使用实际生效 USD 价目和 FX，按 profile 两项最大 token 计算保守点数并向上取整。现有代码对非零上限执行此检查，0 会跳过。此操作的准入检查要求显式正值且不超过 30,000,000，隔离与拟发布的配置值取 30,000,000。实际执行沿原 Invocation 的准确报价 P/C 上界计提点数。两项上界不会大于已通过 SharedCatalog 的 profile 最大量，预留失败时停止发送。资金 owner 仍验证余额和成员月限。
- **报价、确认和发送**。Prepare 读取当前多模态资格、价格、FX 和凭据版本，使用同一冻结 profile 报价。Confirm 核原 input hash、用途、图片和冻结报价。Executor 在请求 claim 前和最终网络 handoff 前重读并比对当前资格与冻结 profile，任一版本/连接漂移、价目或 FX 过期、预算校验失败均拒绝派发。guard 核准确 wire 及一次实际发送；重复确认查原键。确认之后的已发未知保持原预留，沿现有原回执恢复。实际供应商账单单独验证。
- 不新增表、汇率抓取、第二报价账本、权限系统或事实 owner。接线前若适用费率或 FX 变化，重新验证 CNY 成本字段和点数上限。试用固定成本档以 8.00 做保守报价。价目或 FX 超出逐项覆盖范围时，现有 fresh route 读取必须返回未就绪，缩小 token 档位或按准入条件重算。该增量限于指定 Mini 直连与商品整理用途。

## 验证和停止点

在隔离环境用合成凭据和受控 HTTP 响应跑正常报价/确认/发送，检查实际 SDK 载荷。分别覆盖 300,000 CNY 微元、30,000,000 微点临界值与超限；价目过期、汇率跨边界（包括 8.10 时为 ¥0.2916 / 29.16 点，旧两项检查能通过但低于真实折算的旧报价必须被拒）、报价后 Key/价目/FX 变更、余额不足、同键重复确认和缺用量。验证当前拟部署配置只有目标操作可用。真实凭据登记、价格发布、部署、真实调用和试用验收分别进入后续授权阶段。

### 独立评审结论

独立复核绑定方案 blob 9c976ebf170820eaadce737edb13550651042958 及源码基线 40c2e716530b93b77cf02542e8b6b39ead7d6469。离线实现为 IMPLEMENTATION_READY，剩余设计 BLOCKER 为 0。

- 可信连接定义、现有私有受限登记、平台价目和汇率发布、企业多模态选择可复用。只读检查确认试用 Run DB 的连接表与去密视图存在，title_credential_writer、ai_channel_secret_reader、model_route_config_writer 均存在、可登录、无所列管理属性，角色连接上限各为 2。具体凭据材料、角色完整 ACL 与隔离启动仍待验证。
- 24k/4k 档位在标准价与 FX 8.00 下为 US$0.036、¥0.288 和 28.8 AI 点。现有两项独立检查不足以保证人民币报价覆盖按后来生效 FX 折算的点数，方案明确须补仅对此用途的费率覆盖、点数显式正值和双绝对上限检查，并复用原报价与最终发送前的重新准入。
- 剩余 IMPLEMENTATION_TEST 为隔离环境的真实 SDK 序列化和受控传输、报价及发送前的预算边界和版本漂移、仅目标用途的启动与角色池预算。平台价目/汇率发布、凭据登记、部署、模型调用与试用验收均为 NOT_RUN。
