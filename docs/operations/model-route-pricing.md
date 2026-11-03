# 企业模型与官网费率

当前依据：[企业统一模型配置](../architecture/enterprise-shared-model-configuration.md)、PR #1078。此前 #830 按用途、两个数据库分别发布及单计划 VIP 的方案已被本次产品决定替代；历史实现和执行证据仍按原 SHA/范围保留。

## 用户入口

我的账户 → 企业空间 → 模型与费率（`/workbench/account/organization/model-route-pricing`）。企业拥有 `workbench.agent.configure` 的成员在同一处选择默认渠道和文本、图片生成、多模态理解三类共用模型，所有成员共用。三类必须全部满足技术准入，才会在原配置事务内一次发布；缺少任一项时不创建 current、revision、部分配置或 pending 状态。

企业选择不包含费率或密钥。平台维护者在同一页面的“官网价格与汇率”维护原币价目、官网来源、核验及生效/失效时间、地区、服务类型及必要的图片尺寸/质量。非人民币价目使用已核验发布的 CFETS 中间价；人民币使用恒等汇率。价格由平台权限控制，发布还需要 `listingkit.admin.write`；企业配置权限不能写官网价格或汇率。

收费只有一条路径：适用厂商官网价 × 完整计费用量 × 冻结汇率 ÷ ¥0.01/点。不额外加价，不使用代理零售价作为厂商官网价，不填写“输入/输出 AI 点价”或“每张固定扣点”。按张收费的官方型号按官方原币/张计算金额，不伪装为 token 计费。

1 AI 点 = 1,000,000 微点。原资源 owner 的余额、债务、成员月限/月计数、预留、实际结算、差额释放、事件及回执明确使用 `ai-micropoint-v1`；购买/试用商品源数量仍为整点，只在原可信 grant 边界检查溢出并转换一次。重复 source/key 不再次增加。其它资源和人民币分的数量单位不改变。

点数通过十进制字符串传输：图片报价和原始 proof 使用整数微点字符串及明确单位，页面按百万比例精确显示 AI 点；余额、成员月限、事件与审计使用服务端格式化的精确点数字符串。前端不以 `Number` 或浮点计算费用。

发送前 ceil 预留保守上界，终态完整实际用量统一 floor 一次；实际收费可以低于预留，也可以在正用量下为零。差额经原 debt-first 入账处理，不能直接增加 available。缺用量、映射不完整、分项矛盾或超界时保留原预留。源终态、原 finalize key/proof、RecoverDue 和输出恢复责任不变，不因改价或改模型重发。

业务页只读显示企业已配置图片模型和预留上界，不另选 VIP 或修改点价。原冻结计划、请求键、回执和 UNKNOWN 保留；不跨配置继续采用旧模型绑定。历史结果找回不借新模型生成。

## 新空库准备（仅说明，不是环境操作授权）

复用原 ProductAgent/Run DB 的 revision/current、原 CAS 和回执表作为唯一配置事实 owner。私密 manifest 使用 `modelRoutePricing.writer`，数据库必须匹配原 ProductAgent/Run owner；固定 `model_route_config_writer` 非 owner/非 DDL 角色，writer pool ≤ 2。不再配置文字/图片两份 writer/current，不同步两库，不从旧 per-operation、Image current、member 或进程默认配置兜底。

沿原安装器明确安装原配置关系、去密 credential metadata view 和窄权限；运行只验证，不 DDL/seed/发布。可信 `bindings` 是准确厂商/协议/地区/服务类型/原币成本预算/原更小 token、字节、期限和费用上界的准入描述，含准确 reference/credential/endpoint/bound evidence 版本及官网 price ID，不能充当第二份用户收费价目。文本描述的 shape 准入不要求填一个伪造官网价格；正式执行的完整 profile 仍必须持有 use-time 官网价/FX。

同一共享 reader 注入标题、内容、平台资料、选品、诊断、Chat 规划及图片 API/worker。各用途保留原 prompt/schema、调用次数和更小限制。图片密钥和副作用事实仍由原 Image owner 管理，共享配置只借其去密元数据，不能借文本密钥或扩大真实权限。

`imageAgent.generation` 仅包含 `enabled`、`unitVersion: ai-micropoint-v1` 和显式 `maximumMicropoints` 准入上限，不是价格。旧 `priceVersion`/`pointsPerImage` 不会产生可执行报价；API/worker 同一上限的一致性检查保留。缺共享 current 或原币价目/FX/使用协议/完整用量/上界时拒绝新执行，原终态恢复不依赖新 current。

已有单位缺失、混合或不完整的资源库不升级、不修表、不换算、不重新解释；明确拒绝。本文和实现均不授权安装、迁移、真实启停、真实配置发布、授予或结算保留数据。

## 当前未准入项

- 默认准确型号仍为文本/多模态 `gpt-5.4-mini-2026-03-17`、生图 `gpt-image-2.5-flare-2026-09-08`；界面可显示其未准入身份，不把常量写成实际 current。
- Flare 的完整发送前保守费用上界及当前直连生图适配仍未满足；不替换默认、不用平均费用或每张点价代替上界。
- C2-2 图片理解的完整消费链路仍未完成，不能把文本可用或保存选择写成视觉能力已开放。
- 文本正式准入只接受当前受控 transport 已实现的 OpenAI-compatible / `openai-text-v1` 完整分项组合；Google/Claude native 缺对应计费用量 mapper，保持明确未就绪。支持目标不变，协议存在或价目有效不等于能结算。
- GRSAI URL-only 响应没有已准入的完整计费用量；现有传输组件不等于收费准入。新工作保持关闭，历史已记录结果按原事实处理。
- Qwen 已有 native 单图响应可验证输出张数和 1024×1024；仍须准确组织绑定、官方适用地区/型号/图片参数价目、FX、原上限及完整用量同时成立。离线合成测试不代表实际配置/真实运行通过。

发布前按原键保留原输入；ACK 丢失只查询同一回执，未找到或服务不可用都不授权换键重发。历史选择也须重新预览并重新准入。前端/离线自检、CI、独立工程复核和用户使用验收分开；运行、真实保存/恢复、浏览器、模型效果和产品验收保留 NOT_RUN，现场缺证据 UNKNOWN。
