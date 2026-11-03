# Local Account Center 智能选品：共享文字模型接线与差异预览

日期：2026-11-03（Asia/Singapore）。状态：**REVIEW_ONLY / NOT_PUBLISHABLE / NOT_RUN**。

用户已选择百炼 `deepseek-v4.1-flash`，仅准备两项操作的配置接线和差异预览。每项渠道成本上限 ¥0.30、用户消费上限20 AI点，两项各一次合计 ¥0.60/40点。该确认不是调用、凭据写入、配置发布、服务更新、迁移或合并授权。

Design Basis: **N/A**，本轮只读对齐、非执行预览与文档；不修改生产代码/配置/schema。后续正式消费者接线须引用届时实际交付的共享模型合同，不以本预览替代开工准入或运行许可。

## 1. 与正在重构的模型设计对齐

已读取用户指定的[检查可关闭的 Issue](codex://threads/01a17b25-eb0a-7123-bf5b-ea5c4b914071)最近10轮及其原工作区当前设计，并核对[PR #1078](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1078)。引用基线为已提交HEAD `0bd9d243af86710dc2e2b1329b7e3e6b0b5503ac` 的[企业统一模型配置与官方价折算](https://github.com/henryxu1008/shuomi-ai-commerce/blob/0bd9d243af86710dc2e2b1329b7e3e6b0b5503ac/docs/architecture/enterprise-shared-model-configuration.md)；读取的设计文件SHA256为 `5C252EC552322FFFA020E81A01EA817415B93D1459E1CD90BC88DD9C9CCE47E9`。

该PR的限定安全基础为IMPLEMENTATION_READY/FROZEN，整批三类默认、全部消费者、页面与单位贯通尚未交付；原Writer仍在开发。该工作区中的未提交账本修改不是本预览的批准实现。本会话不改写该工作区、不另派Writer/Reviewer，也不发布另一套共享模型或计费实现。

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

公共reader、统一目录/价目/资源单位由#1078原owner负责；#1075继续拥有在途research消费者及两项业务入口，不复制上游源码或另造共享实现。#1078当前基线尚未包含#1075的research消费者，未来组合须核对实际映射，不将登记说明冒称已接线。shared reader未就绪时不能fallback到旧per-operation current。

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
| 两项共同读text及全体已开放text消费者资格 | NOT_IMPLEMENTED_AS_COMBINED_DELIVERY | #1078实际共享reader交付后，由原消费者owner检查准确operation/prompt/schema和预算；不并行复制实现 |
| 三类整体发布 | NOT_READY | 原合同要求text/image-generation/multimodal全部合格；Flare费用上界缺口仍由#1078保留，不只发布text，不造另外两类binding，不绕过原Must |
| 新单位与保留实例 | NOT_READY_FOR_RUNTIME_CHANGE | 新schema/单位全链及正式运行交付未完成；旧实例不可自动迁移/换量纲，后续由用户决定适用安装/切换范围 |
| 真实调用与商品资料 | NOT_AUTHORIZED / NOT_RUN | 后续单独明确准确资料和费用后才可执行；本轮不选择商品或创建研究项目 |

以上缺项阻止发布/运行，不阻止本次非执行预览交付。整体发布/实际保存/完整业务验收NOT_RUN，不将上游局部基础Ready或旧CI转换成这两项已可运行的结论。

## 6. 本次交付与停止点

产物仅为本预览、运行交接修正以及本地去敏计算记录。未写正式接线代码、凭据、业务/模型配置或账本，未改变服务、实例、数据、卷、会话或角色；未调用模型、平台、付费接口或发送消息到另一线程。沿原PR #1075维护文档，不合并、不关闭Issue。

后续先复用#1078的实际交付与原owner，取得明确的去敏binding和统一目录/用量依据，再把本预览转为可核验的真实配置差异。三类/单位/资料或授权缺失时保持未开放，不恢复已撤回的独立点价和旧路由。
