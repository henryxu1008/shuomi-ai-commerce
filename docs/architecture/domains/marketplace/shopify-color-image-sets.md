# 工作台按颜色生成商品图组

执行 [#1637](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1637)。Independent Architecture。用户于 2026-11-25 指定复用通用 Skill v4、使用黑灰白外套原图在隔离环境各生成主图、轮播和详情，并核对现有 Shopify 开发店外套草稿的六个变体。随后同意同一工作台将固定的每色三张改为按现有模板勾选图位及核对每组原图、张数和费用。沿用 [企业商品图片](../ai/enterprise-grsai-product-image.md) 与 [Shopify 草稿已采用图片](shopify-draft-approved-image-transfer.md) 合同。

## 用户流程和责任

- Shopify 隔离工作台的同一商品图片智能体提供「按颜色生成图组」。用户明确选择商品已有的销售规格列及每个值的一张原图。Listing 从当前固定原商品的结构化变体属性和变体图片推导可选值及原图 ID。S、M 共享该颜色的图片图组。通用图片、空绝对图和其他颜色图不能冒充原图。共同挂在多个颜色下的同一 URL 对这些冲突颜色无资格。任何颜色无已归属原图时不准备任务。程序不从 SKU、照片名称、自由文本或模型推断颜色。
- Image Agent 复用现有执行模式、团队 Skill、企业渠道和模板。工作台让用户勾选所选共用原图模板版本已有的图位，一个图位生成一张。通过已有 `selectedTaskIds` 提交，省略字段仍表示采用该模板全部图位；显式空数组、未知或重复 ID 被拒绝。选中图位按模板顺序保留，逐颜色生成相同图位序列。模特图可以使用带上身要求的场景图任务或已有自定义图位。沿用通用商品目标、企业用量渠道、2 至 3 个实际值、每色一个不同原图及不加文字。总张数为颜色数乘已选图位数，同时受现有 32 槽位上限与企业/运行实际预算限制。一次确认固定任务总数、模板及 Skill 版本、分组规格、每色图源、稳定槽位 ID、每图事实、模型和报价。现有旧任务按原配方及摘要读取，无新状态、schema、后端持久化事实或重试机制。
- 准备后，确认区从冻结计划读取每组原图和图位数量，在现有报价区展示最高调用次数及付费方。企业渠道只提供调用次数，界面不推算其未知的实际货币费用。使用 Image Agent 既有请求键、每槽位外部调用栅栏、用量计量、重试和不确定结果恢复。固定颜色存在于槽位配方及正式素材图位；同一颜色各槽位引用同一张原图。生成或采用时重新核对同一企业、同一商品版本与原图资格。对分组任务逐图保留人工采用。已有 Asset 头版本、选择摘要、素材工件及用量来源仍由原 owner 校验。图片的颜色标签以槽位为准；借用其他商品、颜色或历史任务的候选不能附加到本图组。图片模板依旧可用于其他商品。
- Shopify 运行路径沿 #1634。试用只触及已存在、可读回的开发店 DRAFT `10419372589218`，先核对标题、DRAFT、变体原 ID 和六个 SKU。使用 Shopify 既有后台媒体及配图能力补入本轮选用的图，每色 S、M 关联同一张主图。原有图库可保留。无创建请求、无商品标题/价格/库存/可售状态更改。现有工作台销售平台的创建幂等规则不变，不建立正式的 Shopify 既有商品更新服务。店铺主题和前台按颜色切换图库不在范围内。提交结果不明时只刷新查回现有商品。

### 分色不变量

- 每个分色槽位的提示词以冻结的颜色值为唯一销售规格与颜色依据。分色场景不注入原商品汇总颜色或规格，保留描述材质等同款共用且当前合同允许的事实。构造与校验使用同一个提示词函数，旧计划保持原提示词字节。颜色不作为图中文字。
- 一次确认覆盖全部实际颜色，各色图位相同，其组别内位置从 1 连续编号。同一位置的用途、说明和模板参数一致，执行时以完整冻结计划与报价摘要验证整套，用单图执行配方核对单槽位颜色与原图。只核实、恢复同一任务的原调用。分色计划不能进入普通重新生成入口，普通计划也不能通过重新生成换成分色计划。整套通过逐图审核后提交。失败或仍待核实时，保留原任务、槽位和原结算。仅补生、跨平台直传、平台媒体上限和现有草稿更新不属于此增量。
- 采用要求本计划全部槽位及准确图位一致。图片的组别、顺序和颜色从已通过的槽位配方核实。后续保留已批准图片时，已存颜色及图位不能被选择请求抹除或覆盖。通用候选、原图和手工材料不能通过自行填写颜色取得分色生成的证明。

## 验证

原三图切片已有真实 Shopify 草稿的图库与六个 SKU 配图核对记录。此增量用定向回归验证旧九图与新十二图、变更选中图位、失配/缺失图位、总量上限、确认预算、单槽执行和同色原图。在现有隔离实例使用三色原图准备可核对的新计划，验证确认页面以冻结计划为准。真实供应商新调用与 Shopify 写入另有授权时才执行。

## #1642 分色图组单张返工设计候选

**Independent Architecture · NOT_READY。** 依据 [#137 图片业务决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)、[#1642 当前范围](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1642)和[本轮派工](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1642#issuecomment-7162140080)。此节供独立架构复核。上文是 #1637 已交付的整组规则，继续约束初次生成与整组采用。此候选只为明确选中的一个分色图位提出有界返工规则。

### 用户结果与范围

商品运营在刊登工作台打开已采用的黑、灰、白外套图片。白色结构补充图仍带模特，其余 11 张可以继续使用。运营选中该图位，核对原图、生成方法和本次最多一次供应商调用，再比较原始白色参考图、当前正式白色结构图和新候选。选择「采用新图」并保存后，当前商品素材仍为三色各 4 张，只有指定位置的图片发生变化。选择「保留当前图片」、取消返工或遇到生成失败时，现有整套素材仍可查看。图片构图是否合格由运营判断。

- 本切片覆盖现有隔离工作台、自有商品、已采用完整分色集合、企业 GRSai、单商品单图位、现有商品素材目标 `product` 与人工采用。颜色数、图位数、变体关系及原文字策略沿 #1637。单图返工采用独立的新报价和原有批准流程。
- 范围之外包括多选返工、从未采用或部分完成的分色图组返工、换色换原图、换渠道或扩供应商、新语言、自动质量修复、跨商品批次和 Shopify 草稿写入或发布。已有合格图片继续服务当前消费者。旧 UNKNOWN 由 [#1628](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1628) 处理。
- 本切片以「当前完整图组中的一个图位，经一次确认生成一个新候选」为交付单位。采用后的下一轮换图、同一基线同一图位的第二次确认及从任意历史素材拼图列为后续产品决定。重新打开已确认返工仍回到原任务。

### 现状与证据

以下源码以主线 `5edddae5834842e6c6d1aec4e654a2afa1405d93` 为准。历史试用与源码读取均是设计依据。

| 依据 | 已观察事实 | #1642 设计需要补齐 |
| --- | --- | --- |
| [#1637 的三色实测](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1637#issuecomment-7154687005) | 隔离候选 `dde24ba9c71019057449196aefd9a691ee0fa031` 的运行 `5e6be934-bd0c-51ff-998e-35c8f8e44093` 确认一次，12 个槽位均为 attempt 1。取回后工作台整组采用且刷新持久，三色各 4 张共用采用记录。白色结构图仍带模特。Shopify 的六变体核对来自开发店原生后台。 | 用同构的合成集合验证保留 11 张和只换 1 张。 |
| [普通重新生成准入](../../../../internal/imageagent/image_set_service.go#L528)、[分色槽位校验](../../../../internal/imageagent/color_image_set.go#L112)和[工作台](../../../../web/listingkit-ui/src/components/workbench/acquisition/product-image-set-panel.tsx#L300) | 普通返工验证父运行已收束且任务 ID 存在，但显式拒绝含 `ColorImages` 的父计划。分色槽位 ID 由模板任务 ID 和颜色派生。完整分色计划要求所有颜色有同一布局，普通 UI 使用未派生的模板任务 ID。 | 通过父计划里的稳定分色槽位选择一项。完整组和返工子计划分别验证。 |
| [分色采用校验](../../../../internal/imageagent/image_set_selection.go#L97)、[候选谱系](../../../../internal/imageagent/assetpublication/image_set_candidate.go#L163)和[素材选择](../../../../internal/product/asset/image_set_service.go) | 当前分色采用只接受此运行的全套生成候选。普通候选读取能沿已记录的 regeneration 父链核验。Asset 已支持从批准回执选用正式素材，保存新的完整集合。 | 只准一个当前返工候选进入指定位置，其余位置严格引用基线回执的正式图片。 |
| [企业 GRSai 合同](../ai/enterprise-grsai-product-image.md)、[#1628 恢复验收](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1628)和[#1631 隔离记录](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1631) | 原 effect 可按已持久的供应商任务号核实或继续取回。#1637 实测恢复无追加生成。#1631 有三项尚无原任务号的请求待核实。 | 子计划沿同一恢复入口；待核实的返工不会通过父图组的新请求绕过。 |
| [#1641](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1641) 与 [PR #1646](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1646) | Skill 准入和共享数据库容量由原执行处理，影响准备、确认、派发及采用。 | 落地后核对新路径沿其准入和锁边界运行。 |

### 生成计划、身份与报价

1. 工作台在同一商品中通过既有 `regenerate` 入口提交父 Run、一个父 SlotID、看到的旧结果身份和素材版本，以及原有请求键。ImageAgent 按已登录的企业、actor、member、商品操作读取父计划和已知结果。父运行必须已完成采用、原准入回执与该历史 Run 和 Plan 相符、无未结命令或可恢复 effect，原计划 revision、结果摘要及 `ImageSetClosedEffectsDigest` 均吻合。目标要求唯一、已采用且成功物化的原槽位，准确绑定其 attempt、候选 ID、工件哈希和原经济证明。Product Asset 按同企业/商品/source 读基线完整批准回执及当前库存，验证该回执确由父分色运行结果产生，完整覆盖父计划的唯一图位且目标图片匹配。此切片要求基线采用动作同时是该 source version 的库存采用动作和商品 `product` 当前全局 head 的动作。父运行的旧执行期限仍用于核对原事实，子运行使用本次新确认的期限。任一证据缺失、旧页面或无权限时先反馈。
2. 在现有 `ImageSetRegeneration` 中追加可选的分色单图绑定，沿原 `RunID/ClosedEffectsDigest/ResultDigest` 保存父边，增加父 revision、目标 SlotID/attempt/旧候选身份、准确的基线回执 action 与全局 head action/hash。可用父计划和批准回执回查完整排序及其余 11 张，不复制为新候选或新库存快照。绑定及输入选项进入原 `InputDigest` 和规范计划摘要，随原 Plan、Slot 和投影持久化，旧计划和普通重生成的空字段、序列化及摘要保持原样。
3. 子计划保留完整 `ColorImages` 作为分组和来源归属证明，执行清单仅有这一个槽位。以父槽位的 ID、`Color/Group/Order`、用途、角色、原图 ID 和观察指纹为准。其原始位置即使为第 3 项也仍是第 3 项。既有模板精确版本和 Skill 版本经当前配置与权限重新准入，原模板任务 ID 通过冻结模板与分色 ID 算法正向映射，避免从 UUID 反解或使用数组下标。分色单图完整计划和逐槽执行都要校验「恰好一个槽位、同一个父槽位、同一颜色的一个原图」。完整图组的多组布局、逐色连续顺序检查继续适用；普通 regeneration 入口拒绝绕道将分色图变为无色候选。模板版本、原定义、Skill 或原渠道不可用时说明原因，保留当前素材。
4. 在 Listing 的相同企业、商品、原发布版本、有效商品版本和 Apply 回执内核对分色属性及原图资格。准备和确认经 `ImageSetContextReader` 的原来源读取，分色布局沿结构化变体核对，本次参考图只取目标色的一张，重新校验其 ID、散列、类型、尺寸和大小。运行前和采用时在原 live 权限及来源通道重验。父目录清单覆盖多色，子清单只覆盖实际使用的一张。复用现有同商品上下文比较，再逐项绑定选中原图的精确观察值，并分别核实父、子各自的目录摘要。来源版本漂移、跨企业、任意 URL、异色图或更换参考图的请求在原边界拒绝。
5. 复用准备、企业报价、配置快照和 `ConfirmImagePlan`。准备界面显示「白色 · 结构补充图」、白色参考图、模板与方法版本、所选企业 GRSai 模型和「生成 1 张 · 最多提交 1 次 · 保留其他 11 张」。现有企业用量模式由企业直接向 GRSai 支付，硕米不扣点。无供应商费用证据时显示金额待向供应商核对。新报价冻结当前有权使用的原企业渠道、精确凭据/模型配置、单张单调用、原硬限制与 deadline。报价/权限/来源变化要求复核或重新准备，叉开的旧报价不能派发。本候选按原配方报价和确认。
6. 同准备键同载荷读取同一个子计划，同键异载荷冲突。相同确认动作重放原配置准入回执、ImageDB CAS 和稳定 Temporal identity，保持原期限与预算。同一基线同一图位只允许一个已确认的子 Run。跨请求确认的裁决与回查见下节。
7. 现有 effect 身份仍为企业/owner 下的新 Run、计划 revision、子计划沿用的稳定 SlotID、子 attempt。重用槽位 ID 表示用户看到的是同一图位，区分执行和素材身份的是 Run 绑定。执行用原 Temporal 活动、`GenerateQuotedSlot`、固定派发栅栏与现有企业用量 proof，目标为一个生成 POST，调用额度、deadline、字节与 payload 上限沿现有合同。准备及首次确认均读取基线 head，未开始外部调用时再按现有前置授权点核对可核实漂移。采用前的 head 检查只读。Asset 写事务和锁用于最终采用，供应商调用期间不持有。基线在派发之后发生变化时，继续核实本次真实结果和用量，采用时由原 head CAS 阻止旧集合覆盖当前集合。

#### 单图确认的落库与回查

- **当前边界。** 配置库的 `AdmitImageRun` 按配置快照保存原命令和确认期限。ImageAgent 的 `ConfirmImagePlan` 随后经 `CommitProjection` 将同一回执落到 ImageDB。原事务锁定当前子 Run 和投影，核对版本，在 `applyGormProjectionMutation` 更新 `image_agent_v2_runs.admission_json`、状态及版本，并原子写投影、`run.confirmed` 和 `confirm:<ActionID>` 提交记录。现有唯一性限于 Run scope 与该 Run 的提交键。两个不同准备键可各自产生子 Run、配置快照及配置回执，需补充跨子 Run 的裁决。 [配置保存](../../../../internal/integration/persistence/agentconfig/image_configuration.go#L247) · [确认入口](../../../../internal/imageagent/image_set_service.go#L570) · [ImageDB 事务与重放](../../../../internal/imageagent/store/projection.go#L382) · [Run 写入](../../../../internal/imageagent/store/projection.go#L723)
- **唯一键与落点。** 在原 `image_agent_v2_runs` 增一个可空的分色返工确认键，以 `tenant_id` 与此键建唯一索引，未确认子计划及其他 Run 留空。服务端从事务校验过的冻结 Plan 派生规范摘要，输入为商品身份、来源操作和发布身份、原始/有效版本及 Apply 回执、`product` 目标、基线批准 action 和父图位 ID。基线回执 payload hash 继续在冻结绑定内校验。用户、成员、子 Run、重新报价和确认请求键均不能改变此位置的唯一键。只在 `RunMutation.ImageAdmission` 的首次确认分支，同一行 `UPDATE` 写键和原回执。重复键使整个子 Run 的确认事务回滚，投影、确认事件和提交记录一起回滚。键绑定已确认的子 Run，在本阶段持续保留。列为空的旧数据沿原路径读取。内存实现复用现有事务锁模拟同一判定，真实并发用 PostgreSQL 验证。 [原 Run 记录](../../../../internal/imageagent/store/records.go#L5) · [原确认前置校验](../../../../internal/imageagent/store/projection.go#L544)
- **同键重放与竞争回查。** 同 Run 的同一 ActionID、plan revision、plan digest、quote digest 优先读 ImageDB 的原确认，提交 ACK 遗失时先用原 `confirm:<ActionID>` 与原命令指纹回查。不同载荷拒绝。对不同子 Run，只在确认事务返回该唯一索引的确定冲突后，以同一企业及确定键查出胜出 Run，并重新校验冻结身份和当前 viewer 权限。页面有权时打开已确认的返工，无权时只提示该图位已确认。失败请求不能替胜出动作启动。遇到数据库超时或无法核实 ACK 时按原来的子 Run 和确认 ActionID 回查，再决定恢复原确认或继续待核实。数据库故障及其他约束错误不冒充胜出记录。 [原确认重读](../../../../internal/imageagent/store/projection.go#L479) · [确认参数和原动作核对](../../../../internal/imageagent/image_set_service.go#L580)
- **配置回执与派发。** 配置回执只证明该快照的原准入命令，受原 Run、确认 action、计划/报价摘要和 deadline 限制。只有 ImageDB 同事务提交成功并读到本 Run 的确认，`StartManual` 才可在服务端以该 Run 的原 workflow identity 启动。竞争失败方即使已收到配置回执，仍无已确认的 ImageDB 投影。再次拿同一配置回执仍过同一个确认约束，换子 Run 或动作亦不能借用该回执。工作流和活动继续核验原计划及 Run 的确认，派发仍经过原 slot effect 栅栏。按 #1641 候选，首次确认的 Skill/容量边界持有至 ImageDB CAS 返回，失败后的胜出 Run 查询放在其释放后。Asset 版本只读核对放在借用该配置容量之前。唯一索引与 ImageDB 更新共用原事务连接。 [启动顺序](../../../../internal/imageagent/image_set_service.go#L657) · [worker 执行授权](../../../../internal/imageagent/temporal/image_set_execution.go#L25) · [#1641 固定候选](https://github.com/henryxu1008/shuomi-ai-commerce/blob/6a83f42eeebde65f1dacbe66d9fa2c6c2e2fab75/internal/imageagent/image_set_service.go#L616)


### 采用、冲突与刷新

- **选择以基线保存顺序为准。** 按完整基线批准回执顺序构造 N 个选择，其中 N 等于父图组图位数。除目标位置外，N−1 项使用现有 `approved` 类型，逐项引用同一个基线批准 action 及其准确 AssetID，保留原图位、角色、生成谱系与经济证明。目标项使用唯一的子 Run `generated` 候选，完整绑定 run/plan/slot/attempt/candidate/result digest 及父位置。新生成的候选从当前子 Run 读取，旧正式素材继续通过其 Asset 回执读取。此路径禁止普通生成候选跨运行拼接、手动来源或 `GenericHead` 引用，也禁止把原图补入空位、复制图片到多个图位、跳项、移序和自动采用。
- ImageAgent 的完整组校验和单图返工校验共用同一个领域入口，在 HTTP 预览、领域批准、Temporal 与 Asset publisher 使用相同约束。先用冻结 binding 做形状校验，再按基线回执逐项比较顺序和 provenance。要求保留项**直接**引用该回执的对应条目，不能只依赖通用 approved resolver 允许的历史祖先匹配。新候选读取沿 `ImageSetCandidateReader` 校验已知成功、原 effect 及工件/用量证明、可读取性与 Skill 使用准入。已正式采用的 11 项沿当前 Asset 可复用路径读回。浏览器可提交标识和选择意图，服务器从 owner 读取 URL、图位、费用和来源事实。
- Asset 的版本库存 `ApprovalActionID` 与全局 `Head` 各有含义。版本库存确定正式素材所属的回执，全局 head action 与 payload hash 是原事务竞争检查。预览与保存都核对冻结基线和 canonical selection digest。原 `CommitApproval` 一次事务写本次完整 12 项、不可变的新 action 回执并推进 head，同时保留父图组、原 12 项和原采用回执。保留的 11 项可取得本次选择行的新 AssetID，底层素材字节、图位、生成身份和用量证明沿旧项，`SelectionReceipt` 仍指回原批准。采用前可取消或选择「保留当前图片」，无需为同一套旧图再写一张采用回执。
- **当前素材变化。** 发布前发现基线过时，提示「商品素材已更新，请刷新查看当前图片」，保留新图生成事实。并发保存同一基线时只有一个事务能提交。当前 head 的变化禁止自动换基线或合并。发布开始后的原动作核实见下节。
- 刷新从既有请求恢复、父/子 Run、inventory 和原 approval action 读回。成功、生成失败或取消均保留合法读权限下的当前正式集合。子候选已出但用户拒收时，沿既有取消/保留路径结束未采用的子工作，已产生的图片、用量和生成证明留在子任务。取消只允许在原协议给出的时点执行。页面用「重新生成这张」「保留当前图片」「采用新图」「核实原生成」表达操作。在切商品或企业后丢弃迟到响应。预览和返回继续使用工作台已有入口。

#### 原保存回执与确定未提交

- **当前边界。** Temporal 在调用 Asset 前持久化 `approval.publish_started`。Asset 的原事务先占用唯一 `tenant_id/action_id`，同 action 先查不可变回执并比对载荷。首次写入时锁定全局 head 并校验 action/hash，然后在同一事务写回执、批准素材和新 head。重新执行已提交的 action 只返回原回执，head 推进只发生在新 action 事务内。当前 `ReadApprovalCommit` 查无记录会保持 pending。通用 `ErrApprovalConflict` 还可能表示载荷、来源或候选冲突，不能据此断言原保存未提交。 [发布边界](../../../../internal/imageagent/temporal/workflow.go#L1184) · [Asset 事务与原回执重放](../../../../internal/integration/persistence/product/asset/repository.go#L112) · [锁定与对比 head](../../../../internal/integration/persistence/product/asset/image_set.go#L13) · [现有恢复入口](../../../../internal/imageagent/temporal/image_set_approval_recovery.go#L16)
- **原动作的三种结果。** 分色返工的原 pending receipt 增加可选的 canonical request digest，expected head 从已冻结子 Plan 读取。发布活动与恢复入口把同一个 action、企业/商品/source、结果和选择摘要交给 Product Asset 原回执端口。Asset owner 给出下列判断，具体补在 Asset repository 的只读查询及其 ImageAgent/Temporal 消费契约，沿现有 app 装配。只读查询沿既有恢复权限，位于 Skill 采用栅栏之前，或在原选用事务返回且已释放栅栏后执行。保留普通批准的当前协议。
  - **已提交。** 先按原 action 查不可变回执，核对 request/selection digest、商品和来源版本。成功回执优先于 head 现状。即使之后又有人采用另一套素材，仍据原回执收束子 Run。相同 action 的异载荷或回执不一致走冲突，保留原证据。
  - **确定未提交。** 仅针对本切片已有非空基线的旧 head 冲突。Asset 用同一一致性快照读取原 action 回执、基线回执及该企业/商品/目标的全局 head。只有原 action 无回执、基线回执及绑定 hash 有效、当前有效 head 已由另一个提交动作取代基线时，才能返回绑定原 action、request digest 与 expected head 的明确拒绝。理由是旧 action 的成功回放无法恢复旧 head，新写只能以新动作推进。若原事务已持锁且可能赢，单次一致性读取只能看到旧 head，或同时看到已提交的新 head 和原回执。仅比较库存页面或分次读取得到的两个观察值不足以证明。发布活动以该明确拒绝收束现有原命令，在原命令回执路径保存准确失败原因，再释放原 pending 并允许用户返回当前图片。失败回执写入或 ACK 丢失时，从同一原 action 重读证明并完成原命令。原 action 保持已消费，禁止为这次保存另换键或自动改用新基线。
  - **待核实。** 原回执暂缺且原基线仍是 head、读取超时、身份/回执证据冲突或无法取得一致读时，仍保留 `approval.publish_started` 与原 action。工作台显示「正在核实本次保存」，只允许沿原动作核实。需实际重放提交时继续使用原载荷、实时权限和原 Asset CAS。仅 Asset 明确确认未提交后才显示「本次图片未保存」。此查询不写 Asset、不增加批准状态表，也不能授予新的生成或采用。 [原 head 推进](../../../../internal/integration/persistence/product/asset/repository.go#L400) · [发布活动对原请求/选择摘要的核对](../../../../internal/imageagent/temporal/image_set_publication.go#L20) · [现有待核实状态](../../../../internal/imageagent/temporal/workflow.go#L1738)


| 原执行或采用观察 | 可继续的原操作 | 可见结果与计量 |
| --- | --- | --- |
| 待确认或明确未派发 | 读回计划或取消，沿原报价确认 | 未确认时 0 次供应商生成。 |
| 原请求结果待核实，有任务号 | 依 #1628 按原来的子 Run/Slot/attempt 与任务号 GET，随后原路径物化 | 待核实内容继续留在原任务，0 新 POST。 |
| 发送后无任务号、供应商查询不可用或证据冲突 | 保留原请求和原 attempt，沿 #1628 核实 | 显示结果待核实，费用待核对，0 新 POST。 |
| 供应商确认成功但取图或落盘失败 | 复用原结果定位、物化和原结算路径 | 已可证实的出图用量照常记录，当前 12 张保留，新图未可采用。 |
| 已知失败、用户保留旧图或取消 | 先沿原协议收束已经派发的 effect | 保留当前正式集合和原回执。生成事实和已发生费用按原证据展示，采用意愿不触发退款或释放。 |
| 新图物化完成且用量证明已收束 | 比较新旧，人工预览完整集合并决定是否采用 | 新候选在明确保存且提交成功后才替换指定位置。 |

### Owner、装配与评审范围

| 合同 | 实现与装配 | 消费者 |
| --- | --- | --- |
| 商品来源、实际分色选项、当前商品版本和授权原图 | Listing 的既有商品来源 reader → `ImageSetContextReader` → 原 HTTP 与 worker 的来源/执行授权装配 | 准备、确认、执行、审核和刷新 |
| 冻结计划、父谱系、单图配方、调用身份与恢复 | ImageAgent 的 Prepare/Confirm、原投影事务和 Temporal → 原 worker `GenerateQuotedSlot` / `RecoverEffectV3`。配置和 Skill 使用原准入，企业渠道由现有报价与路由装配提供 | 工作台图组、报价和恢复。原生成事实记录供应商可确认的出图量；Resource 继续负责既有平台额度/预留/结算合同 |
| 新旧素材的完整选择、当前 head、正式素材与回执 | `ImageSetCommand` → 原 ImageAgent `ImageSetPublisher` 和 candidate adapter → Product Asset `ImageSetService` / 原 Asset repository。沿 `image_set_application.go` 注入相同来源、选择和回执端口 | 同一工作台，以及商品和后续 Listing 的既有库存读取 |

独立复核聚焦新增的精确基线、单槽谱系、并发确认及混合采用边界。保护范围包括身份/来源替换、无报价或重复付费调用、待核实后换键重发、挑换其他 11 张、旧集合覆盖、保存回执丢失后重复提交。基于现有 runtime 的准备字节/时限、执行 deadline、命令和 40 项集合限制、Skill 停用及实时授权，受影响路径按 #1641 的装配和事务规则核对。Legacy decision 为 N/A。沿当前 Plan/Slot、ImageDB/AssetDB 事务及回执扩展，不加独立素材库、独立计费或恢复 owner，也不迁移原记录。

### 最小验证与开工条件

| 后续验证 | 预期证据 | 当前 |
| --- | --- | --- |
| 三色 12 图合成夹具与原工作台路径 | 三色各 4 张先完成和采用。选择白色结构图，确认前 0 POST，重复确认及刷新后共 1 POST，0 硕米点数。比较原图/旧图/新图。拒收后原 head 及 12 张一致，采用后 12 张顺序与颜色相同、仅指定项变更，旧回执可读，11 项素材哈希与证明一致。 | NOT_RUN |
| 原来源、准入、谱系与失败控制 | 拒绝异租户、异商品/旧版本、错色/错图位、替换超过一张、改动保留项、停用 Skill 或漂移报价。同一确认键并发及丢 ACK 后读原事实。用 PostgreSQL 并发使两个子 Run 均有各自配置回执，验证同位仅一笔 ImageDB 确认和一次供应商 POST（合成计数），败方无确认事件/工作流及用量且只向有权用户回查胜出 Run。模拟原任务待核实及成功但取图失败，恢复沿原 attempt 且 0 新 POST。初次分色及普通返工回归。 | NOT_RUN |
| 原 Asset 并发及 ACK 用例 | 复用现有 PostgreSQL 并发和 Temporal 原 action 恢复测试。竞争提交胜出后，若旧动作确无回执，只有符合 Asset 一致读证明才能结束原 pending。用夹具停在原回执写入但事务未提交的位置，核对尚无回执且 head 未变时继续待核实。原动作提交成功而 ACK 丢失、head 再次前进时仍按原回执收束，保留当前实际 head。覆盖同键异载荷、拒绝结果丢失后按原动作核实及刷新反馈。 | NOT_RUN |
| 获准后的有界实测 | 先确认隔离试用资产和本次费用授权，经工作台只确认一张白色结构图，核对实际 POST、供应商用量证据、结果取回、新旧对照、拒收或采用后的回读及 11 张素材身份。构图质量由运营确认。无 Shopify 写入步骤。 | NOT_RUN，另需真实操作授权 |

### 配方范围与组合准入

| 选项 | 运营能做什么 | 改动与成本 |
| --- | --- | --- |
| **默认采用原配方** | 重抽同一张白色结构图，参考原图与已有候选比较后采用或保留旧图。因为沿用原提示约束，可能仍出现模特。 | 使用父槽位冻结的 brief、用途与基础提示词，以及准确模板与 Skill 版本。沿当前配方校验，原配方已不适用时反馈具体原因。无新输入、模板修改或额外调用。一次确认仍最多一次供应商生成。 |
| **待用户决定是否允许只补一句构图要求** | 可在出图前指出「展示外套结构，画面不出现人物」，模型仍可能偏离要求，最终由运营看图确认。 | 增加该图位的一项可选输入及报价预览。沿既有 brief 的长度和输入约束，将补充说明纳入此图位冻结配方、输入/计划摘要及报价确认显示。核对父事实与颜色来源、同键异内容、刷新预览和单图作用范围。模板、Skill、其他图位和供应商调用上限依旧按原范围。无需额外模型调用，需增加工作台和服务合同的实现与测试。 |

候选复核沿 [#1649 已完成的首轮独立检查](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1649#pullrequestreview-6057569438)。本轮以原配方为默认，新增构图输入等待协调方向用户确认后才纳入。需由协调方对确认唯一键和 Asset 确切未提交的补充合同做增量复核。组合准入等待 #1641 合入且对应主线必需 CI 通过，再按实际合入版本核对 Skill、连接池额度、CAS 及原回执；#1641 的 PR CI 与候选差异供现在核对，组合结果仍为 NOT_RUN。正式 Writer 以 #1642 的 `IMPLEMENTATION_READY` 和后续执行授权为开工依据。
