# 百炼 DeepSeek 共享文字：参考价估算与离线业务接线

状态：**IMPLEMENTATION_READY（本批离线实现）**。当前公共合同由 #1078 原 owner 在 `5cc8248e5d659e3004c2c6686c582528c8fdd0d4` 确认；#1075 原 Writer 是本批唯一代码写入者。用户明确撤下价格精度、峰闲时点和供应商完整最坏费用证明前置，按参考价推进可用业务。旧研究/咨询路线 CANCELLED，精度证明 DEFERRED，不作为新设计或验收要求；旧版本仅由 Git 历史追溯。

## 1. 用户结果与依据

市场研究人员在同一项目关联已保存资料、选择 2–5 项重点商品、确认本次分析及费用估算、查看只读建议，再记录人工决定/理由并明确转入现有商品准备流程。项目资料分析与独立商品对比仍是两个准确操作，不自动互相启用，不引入自动多 Agent 调度。

依据：[当前产品决定](../product/final-ui-ia-authority.md)、[研究与选品产品范围](../product/market-research-selection-centers-v1.md)、[研究证据分析合同](market-research-evidence-analysis-v1.md)、[公共参考配置合同](enterprise-shared-model-configuration.md#本批文字流程的接口与分工)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。复用既有已批准的领域、配置和资源事务架构；不改变事实 owner、schema、授权、运行状态机或 UNKNOWN 恢复责任。

本批只做原工作区/PR #1075 的离线实现与合成回归。凭据、配置发布、保留实例、真实采集/AI/付费、合并和用户试用均未授权。SHEIN 美国仍是后续数据源设计重点，本批不声称已接通该来源。

## 2. 唯一 owner 与调用路径

- 平台参考价格/FX、企业三类模型选择沿原 routeconfig Repository、CAS、current 和回执；一个完整选择快照，不新增 pending 或第二 current。
- 保存时以原 Catalog.Options 校验类别、准确绑定 ID/版本、组织与合法默认渠道。未就绪的类别可以保留选择，不能因此执行；删除绑定的旧 current/回执保持明确不可用的只读投影。
- 使用时 SharedTextReader → SharedModelService.ReadCategory → SharedCatalog 对本次实际使用的类别 fresh 检查当前凭据身份、协议、参考配置和全部已开放用途。未开放的生图不作文字流程前置。
- 每项操作保留独立定义、prompt/schema、权限、启用和模式。固定 `DASHSCOPE` / `deepseek-v4.1-flash` / `openai-compatible`，不借型号别名或其他操作的就绪状态。
- 原配置准备冻结 ModelProfile/参考版本，原受控 Executor → Invocation 源事实 → 原资源微点账本结算；不另建扣费或历史 owner。

## 3. 明确的估算策略

使用 `reference-text-totals-v1`，绑定原 `OfficialTariff.UsageVersion` 与 `BillableModelUsage.MappingVersion`；`PriceVersion`、`FXVersion` 和原 profile 身份随原快照冻结。不新增 Invocation SQL 列。

本策略必须观测到显式、非负、加总一致的 prompt/completion/total，且不超过原调用限制。Complete 仅表示这个基础计量策略所需字段齐备，不表示全部供应商收费分项齐备。缓存细分不作为前置；本策略不应用缓存折扣，也不把未观测的缓存解释为厂商实际零缓存。图像计价维度不能借用本策略。

参考策略不需要 BillingTimeUTC、峰闲窗口或厂商完整费用证明。原窗口组和响应计价时钟扩展退出当前源码；不填假 BoundEvidenceVersion，也不把空凭据/身份当有效绑定。平台参考价与渠道成本分别使用冻结配置，不宣称供应商逐次账单精确一致。

报价/结果的 `referencePriceVersion` 直接从冻结策略/价目派生；页面明确显示费用估算及版本，不另持久化收费投影。其他已准入计量策略保持原语义，不由本策略自动获得资格。

## 4. 预算、发送和结算不变量

| 范围 | 保持的上限 |
| --- | --- |
| 项目资料分析 | 每项 ¥0.30 渠道费用估算预算、20 AI 点 |
| 独立商品对比 | 每项 ¥0.30 渠道费用估算预算、20 AI 点 |
| 两项各运行一次 | 合计 ¥0.60、40 AI 点 |
| 每项执行 | 最多两次模型调用；每次最多 32000 prompt / 4000 completion；原更小字节/步骤/期限上限继续生效 |

候选参考费率为文字输入 ¥2/百万 Token、输出 ¥8/百万 Token，不应用缓存优惠。这是本阶段配置草案，不是实时厂商报价；允许平台后续发布新参考版本。按两次最大 Token 量计算，参考点数预留最多 19.2 点，低于已批准的 20 点；渠道费用估算为 ¥0.192，低于 ¥0.30。原按实际序列化请求收紧预留的行为保持，不强迫使用最大量。数值预算未提高，也不承诺供应商实际账单硬限。

原文本 Definition 的 `MaximumMicropoints` 保存每调用点数预算；本批两项各设 10000000 微点，配合原最多两次调用守住每项 20 点。它是保守预算分配，不是固定扣费或必须预留 10 点；实际 R 仍按原请求收紧。SharedCatalog 使用当次冻结参考价计算最大 R，超过该元数据上限即不可执行；研究/选品采用参考策略但未声明该上限时也不就绪。元数据/参考版本进入原 ProfileVersion/digest，变更使旧确认失效；渠道成本和原更小限制另行保留。当前生产两项仅经原 SharedTextReader → ReadCategory → SharedCatalog；缺 reader 的原入口不可执行，没有静态绕路。

固定非思考策略 `dashscope-deepseek-v4.1-flash-nonthinking-v1`：enable_thinking=false、非流式、无模型工具/联网搜索/视觉/音频；准确响应 model、请求/响应大小、deadline 和一次发送保护保持。

沿原整数公式 ceil R / floor A，A 是本系统按基础用量和参考配置结算的量。相同键/相同请求只使用原回执；异载荷冲突。完全缺失/矛盾的基础计量、超原限制、发送结果不明继续 UNKNOWN、保留 R、不重发，不默认扣尽 R。显式完整且一致的零用量只在此参考策略走原 A=0 结算并释放剩余 R，成功/observed-invalid/原终态恢复都使用同一证明；缺字段、错策略、矛盾或未 claim 的默认零值不能释放预留。可验证用量但内容无效仍沿原 observed-invalid 结算，不冒充成功建议。撤权、跨组织/成员和配置漂移继续由原 owner 拒绝。

## 5. 持久化与验证

按全新系统、当前空业务数据和原 installer/Verify 边界实施；不做旧数据迁移、旧 Service 包装、兼容 fallback、双读/双写、余额重算或资源清理。保留实例的配置、数据和卷本批没有变更。

验证复用原测试：参考策略基础字段/算术和未知结果、原共享选择/CAS/回执/撤权、受控 DeepSeek wire、原 Invocation/资源账本，以及任务专属 PostgreSQL 的完整 HTTP 业务链和现有界面交接测试。不新增 runner 或验证平台。

离线链覆盖：同项目合成资料 → 原共享参考配置 → 两次只读分析 → 人工理由/批准测试 → 原 Preparation HTTP 精确子集转入 → 同键重放不重复转入/调用/扣点。Project/provenance 上游和 provider 响应为合成资料，不作为真实来源覆盖、真实远端资格或用户验收证明。开发自检、原 Reviewer 增量范围、准确提交及 CI 记在 #1075；运行更新和产品验收 NOT_RUN。
