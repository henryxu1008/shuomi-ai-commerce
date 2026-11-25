# 商品图片智能体 V1 — 完整图片能力设计

> Classification and chronology (2026-11-20): This is the frozen full product image-set architecture, grouped with ImageAgent execution. It does not transfer Catalog source identity, Resource settlement, Product Asset approval, or Marketplace release authority. The separate [operations handoff](../../../operations/product-image-agent.md) remains at its original path. [Issue #612](https://github.com/henryxu1008/shuomi-ai-commerce/issues/612) closed for engineering scope and [PR #616](https://github.com/henryxu1008/shuomi-ai-commerce/pull/616) merged; design-stage and isolated results below retain their own time and scope. They establish neither current provider use or charges, current deployment, real platform use, nor product acceptance or authorization.

> Status: **IMPLEMENTATION_READY**。2026-10-09 独立 Architecture Review R2 已解除唯一设计 BLOCKER；用户随后确认本线程统一完成本批必要共享接口与启动接线，正式开工依赖已满足。
> Execution: [#612](https://github.com/qq550723504/task-processor/issues/612)；Parent: #137。
> 当前调查基线：main `d95807f13475d27c0b8691f044cca6320a540d06`，2026-10-09。
> Design Basis: **Independent Architecture**。评审证据与实施状态记在 #612 / 主要 PR #616；后续范围内实现沿本冻结合同推进。

> [#1641 图片 Skill 事务与连接容量设计](#9-图片任务并行时的-skill-使用事务与容量1641) 保留现行停用保证，实施准入为 **NOT_READY**，待独立增量复核。

## 1. 用户结果与当前决定

使用者是当前硕米内部试用企业中的商品运营人员。用户为商品配置整套主图与详情图规则，启动生成，查看逐图结果和整套进度，人工审核并保存可被现有商品详情和供应链消费的正式图片集合。

2026-10-09 用户明确：“要完整的，不要单图的，单图的只是验证模型能力而已。”本任务不再以 #487 的单白底图页面挂载作为交付终点。#487/#503/#525 已交付的派发、统一点数、恢复和批准能力保留为复用依据，不重写其历史完成或验收状态。

用户随后接受智能体粒度方案：**一个可复用图片智能体；每个商品一个独立 Run/Plan；每次图片生成一个 Slot/effect。** 这是已确认的产品粒度；具体领域合同见已完成独立架构准入的§7。

同日确认两项内容决定：**共用视觉素材仅共用原始素材，主图与详情图两组仍独立生成**；详情任务采用商品全貌、核心卖点、使用场景、结构功能、细节特写、规格尺寸、使用步骤、包装配件。每项可勾选；缺少真实依据时提示补充或取消该项，不编造规格、配件或功效。

用户进一步指出平台与类目图片要求的差异。完整计划必须区分通用内容模板与目标平台发布规则；不把“标准 8+8”写成各平台、各类目的必需张数或通用发布合同。

一个智能体定义可以服务多个商品；商品之间不共享 run、来源身份、预算、审批或恢复状态。单张图不会变成另一个智能体定义。整套协调不表示一次 provider 请求生成所有图片，也不授权额外规划模型调用。

## 2. Product / Figma Authority

- [当前 UI / IA Authority](../../../product/final-ui-ia-authority.md)，Figma `tg48P46SSXl6TBy9lZwg63` / page `31:463`。
- 2026-10-09 本 Writer 只读实查当前可见设计：`1998:539`、`1998:545` 完整 AI 重构图片设置；`2061:362` 背景；`2144:361` 语言；`4262:1294` 图片维护；`4908:429` 审核单图处理；`5198:429` 预览；`3395:389` 商品详情。
- `1998:545` 已取得当前截图，明确官方 8+8、自定义指令、共用视觉素材/分别生成、主图/详情图、背景和语言。当前可见状态只展开八个官方主图任务；详情八项以本会话用户明确决定为依据，不冒称已观察到对应 Figma 状态。
- “官方方案”是当前产品内容方案的标签，不能暗示其八项内容已由目标电商平台认证。共用视觉素材旁“减少重复生成”与当前用户决定冲突：仅共用原始素材，两组独立调用与计费；正式 UI 必须据此修正文案。
- 背景设置为统一自然语言背景要求，默认白色；语言为英语、中文、西班牙语、法语、德语、日语。
- Figma 的 token 计费文案与 #487 最新统一 AI 点数决定冲突时，沿领域计费决定：图片按实际生成张数与配置价格扣统一 `orgresource.ai_point`，真实 token 仅审计，不暗改成 token 消费。
- Figma 中“数量不限”不能取消当前请求大小、执行预算、时间和资源边界；自定义任务的有界提交规则须在完整设计中明确，不能先实现无限派发。

### 2.1 平台、类目规则与内容任务的关系

规则按 **平台 + 站点 + 末级类目/商品类型 + 适用商品属性/变体 + 店铺应用模式** 解析。不同类目可能共用规则，但不能因为同平台就假定要求相同。主图、轮播附图、规格/变体图、详情内容等位置及其名称由平台合同定义；当前 UI 的“主图/详情图”分组不得直接等同于所有平台的位置类型。

- 平台硬约束：需要哪些图片位置、允许/要求的数量、类型、尺寸/比例、格式/大小，以及有明确官方依据的背景、文字等内容限制。
- 内容任务模板：已确认主图八项与详情八项是可选内容意图，根据商品证据及目标规则筛选和映射；不是强制生成十六张。
- 企业风格/用户指令：背景、语言、构图与其他偏好，只在适用硬约束内生效。冲突在确认计划前显示，不通过提示词偷偷覆盖平台要求。

调用路径：精确目标与来源 → 当前 Marketplace/Store 规则 reader → 固定规则引用/摘要及适用条件 → 图片计划/报价确认 → 逐图生成及本地可确定校验 → 显式批准 → Listing 发布前沿原 owner 再校验。规则变化使原发布校验失效，不删除既有生成资产或自行触发付费重生成。未绑定目标的通用图片仍可作为通用素材生成与审核，但不声称已通过某个平台发布规则。

当前只读核对结果：

| 当前代码 | 已有能力与本轮接缝 |
| --- | --- |
| `internal/app/listing/rules.go` / `RuleReader.ReadTargetRules` | 读取当前店铺应用绑定、末级类目、类目填写规范、商品类型属性及关联规则；图片 consumer 应复用这条授权读取路径，不增加另一套规则抓取服务。 |
| `internal/integration/shein/official_goods.go` / `QueryProductFillStandards` | 官方只读查询携带 `category_id`；`picture_config_list` 返回该类目图片要求标志。它并未提供全部尺寸、背景、文字规则，不能把未知字段当不限。 |
| `internal/marketplace/shein/goods/images.go` | 当前 Marketplace owner 验证 SPU/SKC/SKU/站点详情位置、主图/方图/附图需要与允许条件、顺序、数量及真实尺寸；尺寸/类型约束有当前发布合同代码。图片计划应复用并版本绑定这些约束，不在 ImageAgent 再抄一套常量。 |
| `internal/marketplace/imagepolicy` / `internal/app/httpapi/image_agent_policy.go` | 内部质量阈值与场景默认策略；不是官方类目发布规则，存在该 profile 不代表该平台/类目已可发布。 |

本轮需要 Marketplace owner 提供计划可消费的窄图片要求投影，并与最终发布校验共享同一约束来源。官方动态标志与当前已核对的官方文档规则分别标明依据/版本；没有已核对规则或实际 adapter 的目标显示规则不可用，不伪造全平台支持，也不扩大成通用规则管理平台。

## 3. 已实现能力与真实缺口

| 已核对当前入口 | 可复用行为 / 限制 |
| --- | --- |
| `internal/imageagent/model.go` 的 Run/Plan/Slot | 已有商品运行、计划、多图片 slot 和有界并发；无须创建第二图片运行时。 |
| `internal/imageagent/generation_execution.go` | 当前正式经济准入只接受 Main、单来源、product/zz/default/general 与 render_source_white_background；不能仅复制十六次请求来实现完整能力。 |
| `internal/app/worker/imageagent/generation_provider.go` | 当前 GRSAI 同步 adapter、exact bytes、单次 dispatch、typed 成功观察与价格/凭据版本绑定可复用；能力工厂目前只构造白底行为。 |
| [固定 Image owner 资源回执](source-white-background.md) | 固定 image owner 的跨库 immutable intent/terminal proof、资源 operation fence、成员 UTC 月限与 UNKNOWN 处理已独立准入；完整图片扩展需要覆盖实际新增边界。 |
| `internal/imageagent/validation.go` / tools executor | 内部 SlotRole 与面向用户的主图/详情图分组不是同一概念；不能把所有轮播图都标为 Main 后全局放松原不变量。 |
| `internal/agentconfig` 与其准入文档 | 企业启用、精确模板/配置版本存在，但正式模板仅支持 title-config-v1；图片参数不能冒名写入标题模板。 |
| `web/.../acquisition/acquisition-page.tsx` | exact 商品详情与 ProductAgentPanel 已有；主图 hook 尚无视觉消费者。 |
| 当前 Supply / Asset 合同 | 来源批准、完整集合与 exact inventory 消费存在；完整生成图片必须进入该 owner，不能保存第二套 Listing 私有批准事实。 |

## 4. 建议粒度与完整操作路径

```text
企业图片智能体与版本化模板
  → 商品的 exact 来源/资料与有效配置
  → 一个商品级 Plan（主图、详情图、素材、顺序、背景、语言）
  → 用户确认实际计划与预算后开始
  → 现 Temporal 对各图片 Slot 执行 / 记录 / 恢复
  → 整套结果列表与逐图原图对照
  → 用户逐图处理，显式确定完整采用集合
  → Product Asset 不可变批准 / inventory
  → 商品详情与 Listing 按 exact inventory 消费
```

多商品批量入口复用当前用户业务任务/批次消费者，每件商品保持独立运行；不新增“管理所有智能体的智能体”。单图重新生成是整套运行中的显式修改意图，必须有新的准确版本/图片身份和预算确认；它不是 UNKNOWN 重试。旧成功结果及其费用事实保留。

## 5. Owner 与装配边界

| Contract → implementation → injection → consumer | 责任 |
| --- | --- |
| Catalog/Sourcing narrow source reader → 当前 source owner → 明确消费者注入 → 图片 Start / source resolver | 来源、actor 与准确商品版本；无 latest fallback。 |
| ImageAgent Plan/Run/Slot contracts → 当前 ImageAgent/Temporal → 当前 worker 能力注册 → 图片 consumer | 商品级协调、图片级唯一 effect 和恢复。 |
| 图片参数/企业启用合同 → 当前 agentconfig owner 的有界增量 → 注册元数据与模板 consumer | 配置不提供权限，不复制执行状态，不把标题模板变图片模板。 |
| 固定 image_generation_v1 intent/proof → 当前 orgresource integration/ledger → 原 owner reader → 每个实际图片 dispatch | 原统一点数/月限、准确价格和每次副作用结算。 |
| 图片候选/完整集合批准合同 → 当前 Product Asset owner → current consumer → Listing / 商品展示 | 未明确采用不进入 ApprovedAsset，旧批准/原 action 回执保留。 |
| safe projection → 当前 BFF/Console → 页面内消费者 | 真实 loaded/denied/unavailable/partial/unknown，不伪造分数或计量。 |

公共 runtime/启动装配由会话 `01a11f74-2565-78e0-9118-4fe1ff53e726` 唯一负责。本 Writer 提供明确能力/消费者/生命周期需求；未协调前不改共享导航、启动装配、Supply 或 agentconfig 公共写入合同。

## 6. 保持的安全、计费与恢复不变量

- 所有来源、配置、运行、slot、结果和批准均绑定 verified Organization、actor、canonical member、exact Catalog/source/config 版本；每次外部派发及读取/批准重验适用 live 权限。
- 每个真实图片 effect 使用原稳定 run/planRevision/slot/attempt 身份；同键异载荷拒绝。多图不共享一个可重复消费的 provider 许可。
- 复用现固定 image owner 经济事实。生成成功仍按该次价格扣点，后处理失败或用户不采用不释放；可信未生成才释放；UNKNOWN 保留原预留并核实，不能由重试按钮换键重发。
- 每次修改/显式重新生成需要新的准确意图；任何未核实旧 effect 不会被伪装成无效果或取消成功。
- 每张图片程序校验有效才可成为候选；程序校验不保证商品保真。保留商品事实、内容安全、SSRF、文件/资产边界，不增加 Extract 或默认模型 Review，不填假评分。
- 批准动作必须绑定所采用的准确候选集合、顺序/角色和当前版本；ACK 丢失沿原 action/payload/immutable receipt 核实。未保存成功不显示正式集合已应用。
- 部分生成失败不删除成功候选和已发生费用。授权撤销阻止新生成/读图/批准，原 terminal proof 的后台结算沿原 owner 恢复。

## 7. 执行合同候选

下文是本轮技术方案，供独立评审；新增名字是拟实现合同，不冒称 main 已存在。

### 7.1 商品级计划与图片级配方

一个 Run 固定一个商品来源版本与一个目标上下文；同一商品投向不同平台/站点/类目时创建各自明确的计划，复用同一个 ImageAgent 定义。重新生成是该商品后续的显式操作，可以创建新的 Run；不是给每张图创建 Agent。页面按商品展示本次整套运行与后续修改历史，不建立第二份商品图片任务 owner。

在现 Plan/Slot 内增加 typed `product-image-set-v1` 配方与集合元数据。`Placement.Group=carousel|detail`、组内顺序及可选 canonical variant identity 表达展示位置；现 `SlotRole` 表达内容用途，二者独立。新集合合同不要求全局恰好一个 `SlotRoleMain`；所选平台需要的首图/变体图位置由 Marketplace 要求验证。详情单独生成不插入假主图。原已准入单白底合同的不变量不得因一次全局宽松修改而被绕过。

| 固定输入 | 合同 |
| --- | --- |
| 商品/来源 | verified Organization、actor、canonical member；采集/collection selector receipt、OriginalPublicationID/Version、EffectiveCatalogVersion、CatalogManifest hash。读取由现 source owner 授权，不接受用户 URL 或自报 tenant。 |
| Agent 配置 | `product.image.agent` 的 code-owned definition/version；exact `image-config-v1` template ref、配置 snapshot ID/digest、activation epoch。模板只提供参数，不提供权限/凭据/Tool allowlist。 |
| 目标 | `generic` 或实际已接入的 platform/site/store/category/productType/variant/application binding；规则 owner snapshot/ref/digest、文档约束版本、图片位置要求。generic 内部 inventory target 继续使用现 `product` 维度。 |
| 集合配置 | content-template/custom 模式、main/detail 所选任务、shared-originals/separate-originals、对应授权来源 identity、背景、语言。shared-originals 为两组解析同一原始来源集合，但生成调用和结果身份各自独立。 |
| 单图配方 | 有界 task ID、自定义 brief、所需事实引用、用途/展示位置、所选原始图片有序 identity+实际 bytes digest、prompt version、最终请求内容 digest、provider/model/protocol/输出参数。 |
| 经济确认 | 每 slot 单张生成报价与价格版本；计划真实张数/调用数、总 AI 点数上限、当前成员月限观察与有效硬预算。token 仅在可信 provider 回执存在时作为审计观察。 |

内容任务需要事实时，计划构建器沿 exact Product evidence 引用值；尺寸、材料、配件、步骤、功效缺失返回具体待补全项。自然语言指令和图片中的文字均为不可信内容，不能变成身份、路由、授权或 provider 参数。V1 不新增规划 LLM；确定性构造配方，产品保真与不可程序判断的内容由现人工审核承担。

边界沿现提交上限：每 Run 最多 32 个生成 Slot；默认并发 4、部署硬上限 10，只有实际能力开放时允许配置。自定义任务提供增删，不展示“无限执行”。单 slot 最多 8 张原始参考图，每张不超过现部署 `MaxSourceBytes`，合集 bytes 不超过 16MiB；超过则计划待调整。background ≤1024 UTF-8 bytes、brief ≤2048 bytes，模板序列化 ≤64KiB、完整计划/投影沿既有 2MiB 上限；不得截断输入或把图片 bytes 放进 Temporal history/数据库 JSON。

### 7.2 企业配置与跨库准入

现 `agent_configuration` 仍是唯一企业启用、默认模板、版本和 start-config snapshot owner，继续位于 ProductAgentDB。新增图片 typed schema discriminator；title-config-v1 与 image-config-v1 的参数、验证及数据库 CHECK 分开，不把图片参数塞进标题字段或任意 JSON map。图片不带 KnowledgeBase、可编辑 AgentDefinition、模型路由或自定义凭据。图片 catalog 与 worker 构造都读同一个 code-owned registration，readiness 来自真实依赖，无付费探测。

拟增 `PrepareImageConfiguration` / `LoadImageConfiguration` 窄 port：在 config owner 的一个事务中校验企业启用、exact template、同作用域 request key 与输入指纹，存不可变图片配置 snapshot，返回 exact ref。metadata snapshot ≤8KiB，只固定 typed 参数/计划摘要与外部引用，不复制 provider secret、图片 bytes、财务状态或 Run 状态。新 image template payload 以独立有界 typed 列存入原 template revision owner；empty-install schema 更新，不迁移旧数据或运行时 AutoMigrate。

ImageAgent 在自己的本地事务初始化 Run、Plan、授权 Catalog、配置/规则/报价引用及初始投影。配置 snapshot 创建后 ImageAgent 初始化失败可留下不可变未消费 snapshot；原 key 查询/重放得到同一 snapshot，Run ID 从原组织/actor/source operation/request key 稳定派生。不能因响应丢失生成第二 Run，不能假称两个数据库共用事务。

图片企业停用沿当前已批准配置产品语义：**停止新的整套任务准入，已经准入的有界执行可完成**。不把一次跨库 live 读取声称为可线性排序的逐 Slot 停用门禁，不新增逐图片许可平台。配置 owner 拟增 `AdmitImageRun`：锁当前企业 Agent 行并核对 ENABLED/原 epoch、snapshot scope与hard ceilings，在同一原配置事务写入一次 `ImageRunAdmissionReceipt`，**该事务成功提交是整套任务准入点**。disable 先提交则新准入失败；原准入先提交则该 Run 可在原预算/deadline 内继续。UI 在停用确认中沿现语义说明这点；新的重生成必须新的整套准入。

准入回执是原 start snapshot 的 append-only/write-once关联记录，固定 snapshot ID/digest、Organization/actor/member、稳定 RunID/confirm actionID、source/input/plan/quote digests、definition/模板版本及原 admittedAt/deadline/硬预算；原 immutable snapshot payload/digest不改写，不复制 ImageAgent 状态。ImageDB 确认CAS与Temporal启动只消费原确切回执；配置事务ACK未知时先按原action读取回执，绝不换key或刷新原预算/deadline。配置准入成功、ImageDB确认/启动失败按原回执重放；已经cancelled/completed的Run不复活，过期回执不产生新生成、点数预留或 provider POST，但允许原稳定 workflow 按原期限保存超期结果。已有工作流阻止/失败/等待审核的进度不通过确认动作复活；已有成功与 UNKNOWN 仍由原 Slot/fact owner 判断，不能根据启动 ACK 未知统一标为未派发。确认CAS同时比较当前Run/plan/action摘要，不由配置回执覆盖已改变的ImageDB状态。此处仍是两个本地事务，不冒称跨库原子。

每次实际派发仍核原 actor/member 的 live 身份、image write 与当前 source权限、原 recipe/config引用、provider凭据/价格及当前资源月限；企业配置停用本身不撤销已获得整套准入的执行，也不删除候选或代替领域权限。领域授权/来源/凭据等重验失败走原 no-generation proof/预留释放；已取得唯一dispatch CAS的工作沿原 proof/recovery收束，不授权第二 POST。结果读取与 Asset批准保持原领域 live 权限，配置停用不新增审批门禁。这里不声称与外部IAM原子提交。

标题 Agent 现有同库 Claim/disable 语义保持不变；不能用图片的跨库端口替换标题已批准的事务入口。模板更新/归档保留已固定内容和历史读取；归档阻止新准备，企业停用/epoch控制新的整套准入；已准入Run的派发继续受原领域权限与冻结预算约束。

### 7.3 准备、确认与整套预算

`PrepareImageSet` 只做授权读取、规则解析、原始图片有界读取/真实尺寸/hash、确定性配方与无外部效果报价；经原 repository 初始化 `awaiting_plan_approval` Run。每张报价固定 provider/model/protocol、输出参数、priceVersion 与点数；总点数为选定 Slot 报价求和，不预扣全额，不把美元 costMicros 当 AI 点数。当前钱包/月限仅为观察，其他工作可能消耗额度，不能承诺预览即预留。

用户看到所选任务、两组原始素材、目标位置/尺寸、实际生成张数和总点数后，`ConfirmImagePlan(runID, expectedRevision, planDigest, quoteDigest, actionID)` 明确确认。服务重验 source/目标规则及价格，先取得§7.2原配置 owner事务准入回执，再在原 ImageAgent projection/command ingress 原子记录原action、准入回执引用与确认摘要，以原稳定workflow identity启动已有Temporal；启动/ACK不确定只查询/继续原action。原 admittedAt固定整套deadline，延迟初始化/重启不延长时间或预算。修改尚未确认的配置重新准备明确的新计划/request key并取消旧待确认计划；不静默消费旧报价，不向已确认Run塞新参数。报价或输入漂移要求重新准备确认，不派发。

每次 Slot dispatch 继续使用原 `GenerationIntent` → resource reservation → dispatch CAS → terminal proof/finalization。native images=1、model calls=1、operation=1；整套 budget 绑定 N 张/N 次，无自动 repair/额外 Review 调用。重生成另做一次明确预算确认，不能藏在本次次数内。价格版本不一致或当前资源/月限不足时，不派发该项及后续未开始项；已派发项完成/核实，成功候选及费用保留，页面显示实际已扣/预留/未派发项。共享素材没有共享生成折扣。

原成员 UTC 月限在每次原资源预留时核对。已预留 effect 的恢复沿 intent 固定原月份及 limit/price，不把旧 UNKNOWN 迁入新月份；跨月未派发的新 effect 在执行时核对当月实际限额，但仍受确认总点数上限约束。超时/取消不把已派发当作未生成。

### 7.4 Provider 适配与输入指纹

复用现 GRSAI HTTPS transport、`MaxAttempts=1`、`gpt-image-2.5`、`replyType=json`、成功观察先于下载、现 staging/immutable artifact/materialization/recovery。在 `internal/product/image` 定义窄 source-edit request/capability，复用现 integration adapter 编码；支持各用途 source edit，不走旧 Extract 或默认模型 Reviewer，不按假 Main 角色绕路。

2026-10-09 只读核对 [GRSAI 官方生成接口](https://qmy27nhsd9.apifox.cn/452409160e0)：images 接受 base64/URL，支持 prompt 与 aspectRatio；gpt-image-2.5 quality 为 auto，接口未给出参考图上限或可靠 token usage。V1 发送 worker 已验证并复制的 exact source bytes；不发送可变来源 URL，不切换异步接口或擅自改模型。最多 8 张是本地边界，不冒称 provider 支持量已实测。真实 provider 能力仍 NOT_RUN。

扩展原 GenerationIntent 的版本化 `InputDigest`，绑定有序来源 identity/bytes digest、完整最终 prompt、code prompt version、配方/placement、配置 snapshot、exact 目标/规则与报价/输出参数。`SourceDigest` 为有序 source bundle hash，worker 用已复制输入形成最终 request fingerprint，再创建 intent；保留原 intent identity 与 proof reader，不新增财务账本。相同 identity/key 不同 payload 在 intent 创建、worker、resource proof reader 和恢复入口统一拒绝。密钥只用原安全 credential reference/version，不进入指纹明文。

准备时输出目标参数必须在现已配置 renderer 支持范围内。若目标要求不能由当前 renderer 合法满足，显示具体能力缺口，不伪造尺寸、不放大图片冒充原生清晰度、不默换更贵模型。生成后用真实 bytes 验类型/大小/像素/目标尺寸及可确定硬约束；校验失败不是退款证据，仍按已发生成功 effect 结算。对于无法程序确定的背景、文字、商品保真，不把技术校验写成平台最终批准。

### 7.5 运行、部分失败、取消与重生成

复用现 Run 和逐 Slot effect 状态，不建立批次账本或第二任务状态机。新增产品集合的明确动作；service、Temporal 更新校验和 Asset publisher 使用相同资格函数，不能只在按钮层保护。

| 触发/观察 | 持久结果与允许行为 |
| --- | --- |
| 准备成功 | awaiting_plan_approval；无预留、无 POST；确认/修改/取消。 |
| 确认成功 | 原确认 action 持久后 planning/executing；刷新读取原 Run，不重 Start。 |
| 有成功、其余运行中 | 保存逐 Slot immutable candidates/proof；展示已成功结果，待后续项收束。 |
| 所有 effect 已结算，存在已知失败/未派发，至少一个可用候选 | 允许选择成功结果与已有正式/原始图片保存完整集合；不强制所有 Slot 成功，不把失败改为 accepted。Run 可进入 awaiting_final_approval，保留逐图失败原因。 |
| 任一 UNKNOWN / 成功后物化或财务恢复未完成 | Run 保持 blocked 与全部 RecoverableEffects；原 effect 核实/恢复，只 GET/materialize/finalize，不 POST。不开放整个 Run 的终结批准，也不使它因保存成功图而消失。成功候选可查看。 |
| 已知失败、无可用候选 | blocked；明确新生成请求或取消，费用按各原 proof；不显示生成成功。 |
| 用户取消 | 停止后续派发；等待已派发项确定/按原核实入口保留 UNKNOWN，再沿现 cancellation 协议收束；不删成功图，不退已发生成功费用。 |
| 用户对已知终态图片重新生成 | 新 action/request key、新 Run（可只含所选任务）、原商品/source/目标与 parent candidate/slot ref；重新确认报价。原成功/失败及费用不覆盖。 |
| 用户选择未知图片重新生成 | 拒绝；只核实原请求，不能借 parent ref 或新 key 抹掉该项 UNKNOWN。 |

重生成 consumer 从原 exact Run/slot/attempt 核对 parent ref；必须是已知终态且原费用已结算（provider 已成功但图片永久不可用时也要先结算）。新 Run 的未改图片不复制成新候选，不再次调用模型；原正式库存或本次已知成功候选用原身份明确选择。选择集合可引用本商品同 source version 的已知终态 parent/new Run 候选，不能将旧对象改写成新 plan identity。页面对比原始素材、旧结果和新结果，用户决定采用哪个。

V1 不新增“所有商品自动重试”、自动质量 repair 或跨商品 Agent。原来源操作/已准入 collection 消费者可逐商品启动上述路径；没有当前消费者的批量入口显示不可用，不借 legacy Task 桥补齐。

### 7.6 完整资产选择与回执

Product Asset 继续唯一持有 ApprovedAsset、immutable approval receipt 与当前 inventory head。拟增同 owner 的 typed `SelectImageSet` 命令/服务，复用原 repository transaction 与 action-key/payload hash；ImageAgent 的 `ApproveSelectedResults` 在所有 effect 收束后消费该服务，而不是先把全部候选批准、再过滤展示。

请求只接受 actionID、expected exact inventory action/hash（初次不存在需明确）、product/source binding、exact result/selection digest 与 ordered typed choices。choice 为 human-source identity、exact current-approved receipt identity 或 image-agent run/plan/slot/attempt/candidate identity；客户端不提供 URL/尺寸/计费结果。来源 reader、当前批准回执 reader及候选 reader各沿自己的 live 权限读取，候选必须 bytes/技术校验及原经济结算已完成，DurableAsset identity/hash 与 effect精确一致。完整集合最大 40 项，非空、无重复来源选择、同图身份不得重复批准进一套集合。

生成候选使用拟增 `ImageCandidateSelectionReader`：读当前同 Organization/actor/product/source 的确切 Run projection 与已知终态 effect，返回封装事实及结果摘要；只接受当前服务已固定的 typed target/rule context。generic 素材采用到具体平台时是新的显式 target approval action，保持原生成 provenance，按该 target 的要求核对；不伪造原 Run target、不静默跨平台读旧 inventory。不同 source/effective version 沿原 source selector/Apply receipt 做明确版本绑定，缺证据不 fallback。

新集合使用独立版本化 set result/selection digest：绑定 source/target/config/规则及plan revision、ordered Slot identities与真实状态（含已知失败/未派发）、可选候选的origin attempt与durable object/hash/settlement、最终ordered choices/位置及expected inventory head。旧ResultDigestV3要求全部Accepted且一个Main的合同不全局放宽。新配方/位置/摘要字段在原normalized Run/Plan/Slot行、projection/event的同一事务保存，禁止只写投影导致重启丢失合同。

在原 ApprovedAsset payload增加presentation group/order/optional variant identity，纳入canonical serializer/selection hash；origin identity仍是原Run/plan/slot/attempt或source provenance，不把展示位置当provider身份。重选沿现SelectionReceipt与新的selection AssetID保存原provenance/批准出处，不重复插入tenant下已占用的原生成AssetID；调整顺序/角色是新的显式选择action，旧记录与回执不可变。标题Apply后沿原exact source/effective binding再选择，不能宽松跨版本消费。Marketplace官方位置（SPU/SKC/SKU/type/sort等）作为独立typed要求/映射事实纳入目标摘要；不是carousel/detail的别名。同一批准资产可被Listing引用到多个合法位置，不复制批准库存行或额外生成。

一次事务校验 expected inventory head，写本次完整选择、immutable receipt，并替换 head；并发旧 head 返回冲突，不能由最后一次旧页面保存悄悄覆盖新集合。不自动合并所有历史批准。原图、先前正式图片与本次候选的最终采用状态都在保存前显示；用户必须明确提交全套选择。未选项不入库存；仅详情生成不要求假主图，是否能发布由实际目标必需位置决定。

跨ImageAgent/Asset终结沿现approval saga：持久原action/选择摘要 → Asset原子提交/读原receipt → ImageAgent completed。同key replay先按原payload/action返回原immutable receipt，不因当前head变化拒绝原成功回放。出版开始后，失败/超时或ACK未知不得被cancel/supersede；`ReadApprovalCommit=not_found`不是未提交证明，原事务可能仍在途后提交。只通过原action/hash回执恢复并收束；允许取消须有原owner确切no-commit结果，或在任何出版调用前已确定未开始，不能靠异常类型或一次查询空值推断。沿现command/effect-owner恢复，不新增关闭平台。必须修复`failedPendingActionCanBeSuperseded`的同根因路径并验证Asset提交/ACK丢失后cancel不越过原批准。

Asset已提交但ACK/Run finalization丢失只读原immutable receipt，不能沿当前head推断原action；同key异集合拒绝。在ACK核实前UI显示保存结果待核实，不显示已应用。Run completed表示本次生成/人工选择协议收束，不代表平台已上传或所有候选都被采用。

### 7.7 UI / HTTP consumer

入口沿当前采集商品详情与已准入 Supply 图片编辑，不新增独立 AI 工作台。设置沿 Figma 1998:545 的官方内容方案/自定义、主图/详情任务、共用原始素材/分别选择、背景、语言；目标存在时显示实际平台/类目要求及冲突。准备计划与点数确认、逐图进度、原始/结果对照、勾选采用与完整集合保存是完整功能的必要消费者，应用现 Console 表单/错误/确认模式，不增加人工验收 session。

拟有 feature-local `/images` source consumer：GET 原始候选/当前库存/可用配置；POST prepare；GET exact run；POST confirm、cancel、regenerate、approve-selection；原 recover/resume 沿现 effect/action 核实入口。所有 mutation 使用 verified Effective Organization/member、image read/write 与原 source/collection 权限；BFF 传 `X-Expected-User-ID` / `X-Expected-Organization-ID`、Idempotency-Key，schema strict，拒绝未知字段、请求未读体/大体。prepare/selection 请求 ≤128KiB，现通用投影≤2MiB，30s HTTP deadline；外部 generator deadline仍原≤5min、原 finalization grace。POST 不允许浏览器任意 source URL、member/month、price、model或任意规则 JSON。

页面保留 exact source operation/request/action 与 request fingerprint，原 scope 独立隔离；切企业/商品后迟到响应丢弃。刷新查原任务/原批准 action，只有用户明确新生成才换请求键。state 来自 owner：loading/empty/denied/unavailable、等待确认、逐图执行、部分失败、UNKNOWN、待审核、保存待核实、正式集合；无假进度、假分数/假扣点。人工文件替换复用当前 source media/collection 与 Asset 选择，不增加可绕过来源批准的图片上传 owner。

### 7.8 持久化、恢复与共享装配清单

| Owner / 本地事务 | 本轮有界增量与恢复 |
| --- | --- |
| agent_configuration 原 schema | typed image template revision/metadata start snapshot及write-once关联的ImageRunAdmissionReceipt；企业行锁与snapshot准入原事务/命令幂等；准入ACK未知读原action，原payload/deadline不改。标题原CHECK/Claim不得宽松替代。 |
| ImageAgent 原 DB/Repository | typed set recipe/元数据、配置/规则/报价refs，Prepare/Confirm ingress receipt、selected approval digest；Run/Plan/catalog/projection/event本地原子事务。无新运行库/队列。 |
| GenerationFact / SlotEffect 原 owner | InputDigest/source bundle、逐图报价；仍原意图、唯一dispatch CAS与proof reader，restart/retry只核实原 effect。 |
| orgresource 原数据库 | 原 reserve/finalize/cancel operation fence，统一 AI 点数和成员月限；只消费固定 image_generation_v1 exact proof，不增加消费账本。 |
| Product Asset 原 DB/UoW | 位置payload、exact expected-head CAS、selected完整集合与原审批回执；失ACK读原action，不跟latest。 |
| Store/Marketplace/Listing | 当前授权只读规则 port → 窄图片要求投影，共享发布约束版本与plan检查；无第二规则库，无新平台写/凭据路径。 |
| 当前 app/worker composition | 注入配置 reader、source/collection reader、规则 port、现资源/Artifact/Asset依赖、typed renderer，前后端均以实际依赖投影可用性。 |

Formal write ownership：2026-10-09 用户确认本线程统一完成本批必要修改。会话01a11f6e-a012-7110-a953-3f7d43dbe6ad为唯一Writer，覆盖ImageAgent/图片domain/feature-local consumer、agentconfig typed合同、Asset选择、Marketplace图片要求、Supply消费者与必要runtime/启动接线；替代原会话01a11f74-2565-78e0-9118-4fe1ff53e726在本批范围内的分配。沿同一分支/主要PR，不覆盖其他工作，不新增部署或真实数据/付费provider权限。

### 7.9 Must / Threat Model 与验证

Must 是整套参数→实际计划/报价确认→逐图运行→逐图可检查→明确采用完整集合→现商品/Listing读取；包括来源/租户/权限、预算上限/成员月限、每 effect 单次外部请求、UNKNOWN不重发、 immutable选择及版本绑定、refresh/restart可核实。未接入平台不宣称可发布，不能以单白底fixture取代完整消费者。

Threat Model沿已有#487/#503及当前Source/Asset/Store边界：不可信浏览器/图片文字/自然语言、跨企业或actor ID/receipt伪造、权限撤销/配置与来源漂移、同key异载荷、并发与ACK丢失、provider模糊结果、不可逆计费和不安全远端文件。现文件/SSRF/凭据保护复用；不新增IAM、全局Saga、规则平台、安全扫描、验收runner或理论side-channel需求。没有新 Accepted Risk；Out of Scope沿§1/Issue，不调用付费provider，不做真实采集/平台写/部署。

验证只覆盖改变的用户行为与高风险边界：typed配置隔离/字节边界、官方规则动态标志与共享尺寸约束、source/target/recipe指纹冲突、未确认0POST、N独立生成与计量、价格/额度不足保留成功项、UNKNOWN读取恢复0POST、known部分失败可选择成功集合、未选失败不变accepted、明确重生成不重算未改图片、并发选择CAS/原actionACK核实、切scope迟到响应和完整页面路径。先写能捕获目标缺陷的相关测试再实现；复用现PG/Temporal/provider fixture/BFF/component工具，不建新验证平台。

本次架构独立检查审核实际新合同与代码接缝，最多两轮；达到IMPLEMENTATION_READY后冻结，只对新BLOCKER重开设计。最终独立检查一次核完整用户路径与实际diff/运行；TDD/CI/Reviewer不能自行签发产品验收。真实provider、真实平台和用户使用效果如实保留NOT_RUN至获准实际执行。

## 8. Legacy 与验证范围

### 8.1 实际启动阻碍：通用集合与平台装配解耦（IMPLEMENTATION_READY）

2026-10-10 在最终运行准备中发现可执行 BLOCKER：§4 允许无平台目标的通用素材生成与审核，但完整 ImageAgent 启动强制 Supply，Supply 又强制正式 SHEIN 应用。`TestFullImageGenericManifestNeedsCurrentOwnersWithoutOfficialPlatform` 在无 Supply/Store、具有当前 Product/config/resource/Image/Temporal 的 manifest 上实际 RED，失败原因为强制 Supply。受影响 Must 是通用商品整套参数→生成→人工选择→Asset 保存；阻塞层级为完整模式试用和正式交付，不涉及平台规则降低。

最小组合修正只消费当前合格 owner：

- `imageAgent.assetDatabase` 显式声明与 `imageAgent.database` 同一物理 Image/Asset owner 的独立 `supply_asset_runtime` pool，最多8连接；API、worker、Asset 角色仍分离。已有 Asset 初始化/授权/启动校验不变。
- Supply 是可选消费者。有 Supply 时其 Asset `DatabaseConfig` 必须与 image 的 Asset 配置完整一致，装配共用这一原窄 Asset pool；不开第二 owner/pool。无 Supply 时只打开 image 的该 Asset pool，不构造 Supply worker、官方应用或平台 adapter。
- Acquisition 通用来源直接复用当前有界 Product snapshot、Review AppliedPublicationLookup、作用域 publication reader、现 OrganizationExecutionAuthorizer 与原 source/actor/member 检查。配置 owner、ImageAgent、orgresource、Asset 调用及恢复协议不变；不从另一来源 fallback。
- 只有真实 Supply 已装配时才注入 Supply source 与官方 target/rule ports。无该 port 时平台来源、平台规则与批准明确拒绝；不能用 generic 冒充平台，不能削弱 Supply/Store 的原正式应用准入。
- SourceMedia 与同根手动替换依赖当前 Product/Collections、current IAM 和既有 immutable storage，不以 Supply/正式应用为前置。保持现文件、权限、SSRF、不可覆盖与真实字节读回要求。
- 不新增数据库、schema、状态、财务、准入回执、外部副作用、IAM 或通用启动/验收框架。不包装历史单图 overlay；此前无完整模式实例，使用当前明确 manifest，不新增旧配置兼容路径。

contract → implementation → injection → consumer：当前 source/Review/组织执行授权与 Asset contracts → HTTP 有界原 owner 构造 → full ImageAgent 注入；有 Supply 时注入其已批准 ports → 采集通用面板/原 Supply 面板与 worker。前后端仅按实际规则配置开放平台选择；generic 仍无发布合规声称。

验证限于本阻碍：先保留上述 RED，再验证无官方应用的 generic 配置/独立 Asset 角色与生命周期；错误角色、不同物理 owner、与其它 owner 混池拒绝；有 Supply 的单 Asset pool 与原 worker 保留；未接入平台来源/规则拒绝；当前来源授权/Asset 选择及 SourceMedia 不变量使用已有测试。其余冻结的状态/计费/UNKNOWN/恢复证据不重跑全局矩阵。2026-10-10 本增量经对应独立检查明确达到 IMPLEMENTATION_READY；仅对该启动 BLOCKER 进入正式装配修复，原冻结合同保持有效。

Legacy decision: **EXTRACT | RETIRE**。抽取当前合格计划/逐图 effect/文件校验/资源计费/回执行为到正确 owner；不 Wrap 已退休 root ListingKit、Task-first 或旧 Service，不增加双读/双写/第二事实源、旧数据迁移或 compatibility。

2026-10-09已按AGENTS完成两轮正常独立架构评审，技术设计收束为IMPLEMENTATION_READY。R1配置准入BLOCKER已由§7.2整套事务准入回执解除；R1的批准ACK/cancel、set digest/持久化、官方位置映射、Asset CAS/replay及多来源proof接缝为IMPLEMENTATION_TEST，正式实现与必要测试内收敛，不重开全局设计。正式实现使用本Delivery Batch一个Writer/分支/主要PR；复用现PG/Temporal/provider fixtures，不建设runner/验收平台。评审候选合同SHA256为417e65c9fdd0f1705d5e0a174fe63efc8647b78f36f532ffcc47035dcdb7179c；本次状态标记和R2提出的非阻塞停用文案清理不新增合同边界。

最终交付检查包括完整用户路径、准确diff/当前运行组合、权限/错误副作用/数据保存。架构独立review已完成；正式实现进度与开发检查证据记录在#612/PR#616，设计准入不代表完整用户链或产品验收。最终交付检查、付费provider、真实平台与用户验收仍**NOT_RUN**。无合并、关单、共享/生产部署或真实数据授权。

### 8.2 受控试用的生成物实际字节读取（Reuse Existing Architecture / Ready）

2026-10-10 用户明确选择“先用受控响应跑通完整流程”。既有 IsolatedTrialGeneratedURLPolicy 只允许配置 localhost 下经验证的精确生成物 key，明确不允许任意网络下载；generic material 的公网探测会拒绝该生成物，阻塞试用的预览/保存。对应独立复核分类为 IMPLEMENTATION_TEST，现 owner/合同足以覆盖，不需重开全局架构。

复用当前 Image/Asset 候选核验、原不可变批准回执、canonical published-key 与 ImmutableObjectStore.ReadObject。MaterialTargetResolver 对已经 service 核实的生成候选/已批准生成物走独立只读 owner port：source tenant、原 Run/revision/slot/attempt/hash 与规范 published key、配置 PublicBase 精确匹配后，从当前 immutable owner 读取真实字节，继续原大小、完整解码、hash/尺寸校验。已批准素材可能由同企业其它成员生成，原 URL/身份必须来自已核实批准回执，不以当前 actor 重写原 owner key。API 预览和 worker 保存注入同一 port；没有 port 或证据拒绝，不回落到 URL 下载。原图仍走公共 HTTPS/SSRF 读取；手动素材保留已有 immutable storage 读取。没有任意 key/URL 的用户入口，也没有 localhost 下载豁免。

仅补当前用户完整受控试用所需读取接线；不改变 schema、canonical owner、权限政策、批准事务、生成/计费状态、UNKNOWN、恢复或 provider POST。复用现 S3 uploader/worker object-store bridge；不得增加存储框架或验收 runner。验证为原 local generated URL 拒绝 RED，精确 key 读取/跨企业及身份/hash/key漂移拒绝、原图私网拒绝、真实字节损坏拒绝的有界 GREEN，以及 API/worker 同 port 和实际受控完整流程。独立评审已给上述最小边界，Issue 记录为 Reuse Existing Architecture / Ready 后实施；真实模型质量和用户验收保留 NOT_RUN。
### 8.3 受控启动发现的配置和报价接线修复

Design Basis: Reuse Existing Architecture / Ready。完整 GRSAI 模式只消费当前企业凭据 owner；通用 worker 配置的全局 OpenAI 密钥校验不属于这条运行路径。专用配置加载仅检查当前 Image worker 数据库目标和有界角色，Generation/Admission/immutable storage 的完整校验仍由现有完整模式组合承担。计划报价读取同一已批准 worker 只读凭据能力，再返回已绑定的配置元数据；不增加 API 数据库凭据权限，不向客户端返回密钥，不创建新 owner、schema、状态、恢复或计费规则。验证：无全局 OpenAI key 的专用配置可加载；错 owner/角色仍拒绝；用真实隔离角色复现旧报价失败，并跑通原计划确认路径。

### 8.4 当前 Product 只读消费者的启动阻碍（IMPLEMENTATION_READY）

受控实例实际启动失败于 `build full image Product context: review unavailable`。§8.1 所需快照/AppliedPublicationLookup 被误装配为完整 Review 写服务，且 acquisition 的 capability grants 只有 Supply 能获得 `product_title_proposals` SELECT。这是核心通用完整路径无法启动的 BLOCKER；不代表需要开放标题写操作或平台功能。

最小修正复用已批准当前 Product owner 的两个只读合同：有界 Catalog snapshot reader 和 Review repository 的 AppliedPublicationLookup，不构造 Review producer/service，也不要求读取 `product_title_operations`。现 acquisition RuntimeCapabilities 增加显式 ImageSets 标志，仅在完整图片消费者存在时准入现有 `product_title_proposals` SELECT；所有共用 Product pool 的现消费者按一致 capability 校验。Supply 的原权限不变；ImageSets 不授予 Review INSERT/UPDATE、operations SELECT 或平台表权限。初始化仍只对明确空专用数据库安装已有 Review schema；无请求时 DDL、无新表/owner/事实/状态/迁移。

原 live source 权限、actor/member、Apply lineage、bounded bytes 与租户检查保持；同一个冻结完整图片 Must 所需只读接线，没有新产品范围。先以真实隔离 narrow role 复现启动失败，再检查原始快照与现 Apply lineage 可读、额外写权限和未准入读取拒绝，最后完成实际页面流程。本增量经对应高风险边界检查后才修改正式接线，不重开全局架构。

2026-10-10 现独立 Reviewer 对 §8.3/§8.4 的对应高风险边界检查明确达到 IMPLEMENTATION_READY，无新 BLOCKER。检查文档 SHA256 为 7B615DACCB6B7ED5DA5BAAB1474AB96DDB9CCEF5BB62F468DD3FBD4AAFF69D5B；本状态标记不改变合同。仅准入上述最小实现，权限测试、受控完整流程和用户验收分别取得实际证据。

### 8.5 刷新后的部分重生成模板读取（Reuse Existing Architecture / Ready）

评审 finding 4235681299 分类 IMPLEMENTATION_TEST，受影响 Must 为刷新后原任务的明确部分重生成。仅补已有只读合同：Run HTTP projection 从其绑定的现配置快照读取精确 TemplateRef，按原企业/actor/member/Run/context/ref 核验；浏览器读取该不可变模板版本，部分重生成使用原任务模板和对应任务 ID、原图配方，不被当前默认模板或模板 head 覆盖。新整套计划仍沿当前显式模板选择。不改变模板事实、准入/启停、Run 状态、生成/计费/UNKNOWN 或恢复；缺少原模板读取则拒绝重生成，原审核仍可用。验证为旧自定义/旧版本与当前默认不一致时的页面 RED→GREEN，以及实际刷新后的单项新计划。

## 9. 图片任务并行时的 Skill 使用事务与容量（#1641）

**Independent Architecture / NOT_READY，待独立增量复核。** 运营同时处理多件商品时，报价、确认、生图、配置读取和结果采用应有明确的等待期限。本批唯一方案是缩短纯读取事务、为必要的嵌套连接预留容量，并保留现行 Skill 停用保证。依据为 [#1641](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1641)、本页 §6、§7.2、§7.6、[官方标题与图片方法产品决定](../../../product/final-ui-ia-authority.md#官方标题与图片方法2026-10-18-用户批准)及 [已完成的复现与方向复核](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1646#pullrequestreview-6056394884)。Legacy decision 为 N/A。范围包括商品图组的配置、Skill 使用、现有 ImageAgent worker 装配及原请求的容量反馈。真实供应商、发布、页面改版、通用调度及标题/Chat Skill 语义在范围外。

### 9.1 根因与复现

[PR #1631](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1631) 记录了 5 个活动 Skill/image 操作占满 5 个连接、灰色计划停在派发前。试用将原池上限调到 8。源码基线为 `5edddae5834842e6c6d1aec4e654a2afa1405d93`。当前应用把同一个 `configurationDB` 交给 Skill、配置和 worker 的模式/来源重验。图片 API、worker 的效果库角色、商用结算和 Asset 另有各自的连接池。 [装配](../../../../internal/app/httpapi/image_set_application.go) · [运行入口](../../../operations/product-image-agent.md)

`AcquireUseForTarget` 用一个连接开启事务，对 `organization_agents` 中的图片 Agent 行和 `skill_capability.skills` 中的 Skill 行取得 `FOR SHARE`，并通过该锁保护精确版本读取。生成和图位报价在外围函数退出时才释放。进入 `GenerateQuotedSlot` 后，来源、用量、授权、配置、派发与获取供应商结果可能跨越多个 owner。生成授权复核会进入 `modeBoundImageContexts` 等原消费者。准备和确认也会在外层 Skill 事务持有期间对配置池另开事务。采用路径为不同 Skill ref 逐个借连接，跨来源组合存在单请求耗尽的风险，尚未在本轮执行该路径。 [Skill 准入](../../../../internal/integration/persistence/skill/use.go) · [生成](../../../../internal/imageagent/generation_execution.go) · [图片集消费者](../../../../internal/imageagent/image_set_service.go) · [采用](../../../../internal/imageagent/assetpublication/image_set_candidate.go)

隔离复现使用 PostgreSQL 16.16，应用池 `MaxOpen=2 / MaxIdle=2`，只用于取证的独立会话上限 1。用原 `GenerationExecution`、`skillimage.Context`、PostgreSQL Skill/AgentConfiguration Store 和现有 Run/GenerationFact/Resource 测试夹具。两次场景均由不同 Run 驱动，首个 Run 通过原派发 CAS 进入进程内模拟供应商，第二个 Run 在真实 Skill 准入后的测试注入点调用真实 `ReadAgent`。一个场景卡在 `QuoteSlot → PrepareProvider`，另一个卡在 `GenerateQuotedSlot → RevalidateProvider`。测试以渠道通知派发和进入查询，以 `DB.Stats` 判断等待成立。独立配置读取期限设为 350ms。

| 场景 | 持有和等待 | 取消后 |
| --- | --- | --- |
| 慢供应商下 A 生图、B 图位报价 | 2/2 个连接在用，2 个 PG 会话事务内空闲。两个 `FOR UPDATE NOWAIT` 探针分别在 Agent、Skill 行返回 `55P03`。B 和独立查询令 `WaitCount` 增 2，独立读取超时。 | 取消 B 后在用数降为 1，配置读取成功。取消 A 后在用数为 0，A 保留 `UNKNOWN`，相同 attempt 重入后模拟发送计数仍为 A=1、B=0。 |
| 慢供应商下 A 生图、B 生图复核 | 同上。B 已准备效果和资源预留，卡在测试注入的配置复核，未取得发送许可。 | B 为 `GenerationNoEffect`，预留 `released`。A 为 `UNKNOWN`，归还连接并重入后发送计数仍为 A=1、B=0。 |

测试命令为 `go test ./internal/imageagent/store -run '^TestIssue1641SharedSkillPoolSpike$' -count=1 -v`，两个子用例通过。效果和资源夹具使用临时 SQLite，来源字节、IAM 和供应商为测试替身。配置读取注入到真实方法的回调接缝，未经完整 HTTP 装配、Temporal 或真实供应商网络。这证明真实 Skill 事务占用配置池及两条生图相关消费者的等待条件。整套报价、采用、异步轮询、实际流量容量和修复后的表现尚待执行。临时夹具、命令与日志摘要在 Draft PR 记录，源码不进入仓库。

### 9.2 事务生命周期和原派发许可

**当前停用合同。** [已批准的 #795 图片方法设计 §5 步骤 7–8](https://github.com/henryxu1008/shuomi-ai-commerce/blob/70c0622059966cb1cadc9613dce065f24323fbe4/docs/architecture/official-skills-and-image-method-v1.md#5-合同--实现--注入--消费者)要求新效果在预留和派发前持 Skill 读锁，直到原供应商调用返回（含 UNKNOWN）。新整套确认持锁到配置准入写入和 ImageDB CAS 返回。新采用持锁到原 Asset 提交返回。原准确回执与原效果核实仍可以沿各自权限恢复。[#301 退休后基线](https://github.com/henryxu1008/shuomi-ai-commerce/blob/a4b3dfc9dc78b70bc5a8bee98e79424faa469cdd/docs/architecture/official-skills-and-image-method-v1.md#5-合同--实现--注入--消费者)仍含上述步骤，[文档归并](https://github.com/henryxu1008/shuomi-ai-commerce/commit/0dca66876ac2243b92a95c82663dd75025530cae)保留行为门禁，[#774 关闭核对](https://github.com/henryxu1008/shuomi-ai-commerce/issues/774#issuecomment-6552651245)确认原撤销栅栏保留。本轮未找到更改这两处顺序的后续产品决定。

现代码的新确认 `defer release()` 还覆盖 ImageDB CAS 之后的 `StartManual`，长于步骤 8 写明的范围。本批确认在 CAS 返回后释放，再向 Temporal 发出原启动请求。原 `GenerateQuotedSlot` 持锁跨提交、同步等待或异步轮询及原结果处理，本批保留该释放点。 [新确认](../../../../internal/imageagent/image_set_service.go) · [生成](../../../../internal/imageagent/generation_execution.go)

事实归属沿原 owner。Skill 保存准确不可变版本、归属、摘要、可用状态和使用锁。AgentConfiguration 保存精确快照和一次性整套确认回执。ImageAgent Store 持有原 run/revision/slot/attempt 效果 CAS、供应商任务 ID 和 proof。商用 Resource 保存点数、月限与结算。企业自有渠道沿原用量合同，供应商侧费用继续由原供应商计量。Asset 保有采用事务及原动作回执。应用组装依赖与运行上限。无跨库写入合并、第二许可表或新恢复 owner。

| 消费者 | 事务范围及同时使用的配置连接上限 |
| --- | --- |
| 图位 `GenerationExecution.QuoteSlot` | 原 live 身份、企业、成员和来源核对后，短事务只读当前有效的精确 Skill ref/policy、摘要和有界材料。完成校验后显式释放，再构造 provider 报价。只会顺序使用 1 条配置连接。材料留在本次内存，原报价指纹仍绑定准确 ref。 |
| HTTP `prepare/regenerate → ResolveSelectionForTarget → PrepareImageSet` | HTTP 新选择解析拥有者及短 Skill 事务已在 service 调用前发生。先检查原准备回执。新准备在读取 Skill 前取得 1 个名额。`PrepareImageSet` 读取准确材料后释放再做其余工作。原 `PrepareImageConfiguration` 在快照事务内以可信 principal 再核 ref、摘要、归属和 ENABLED，顺序为原 mode-subject → Agent → Skill。注入 Skill 存储的图片目标事务内检查，沿现有标题事务内检查接缝，图片仍用 ImageTarget。完整准备回执按原键读回。若只有配置快照、尚未写入原图组 Run，继续形成新准备时同事务仍复核精确 ref。缺接线、scope 或版本不符则拒绝。应用内单次最多 1 条配置连接。 |
| HTTP `confirm → ConfirmImagePlan` | 先按原企业、成员、action、plan、quote、input 检查 ImageDB 既有确认。仍待首次写入 ImageDB 的带 Skill 路径在 `ReadImageRunAdmission` 前一次取得 2 个名额，无 Skill 只计 1 个。准确已有配置回执走原 CAS/启动恢复，无回执才按当前来源、模式、报价和 Skill 进入新确认。保持外层 Skill 锁及 `AdmitImageRun` 的独立配置写事务，直到原 ImageDB CAS 返回。ACK 未明时只读原回执。归还事务与容量后再调用 `StartManual`。 |
| Worker `ExecuteSlotV3 → GenerateQuotedSlot` | 带 Skill 的图位在进入活动、任何配置读取或预算预留前一次取得 2 个名额。该活动中的短预报价沿同一额度，无 Skill 图位计 1 个。保持原 Skill 事务到整个生成执行返回，覆盖原预留、CAS、供应商返回和 UNKNOWN。第二条用于原 `generationResourceAuthorizer`、派发前授权及异步查询中串行的配置/模式复核，包括 `CheckModeSnapshot` 的原事务。路由/凭据照旧复核。已有 effect 先按原入口核实，重入无法取得新的发送许可。 |
| 新预览、采用 | HTTP `preview/approve`、worker `PrepareImageSetPolicyAdoption` 的预览，以及 `PublishApprovedImageSet → ImageSetPublisher → ImageSetService.Select` 的人工和策略采用均进入预算。原 Asset 回执先读，纯预览在一次短事务内核对全部准确 ref 后释放。新采用先收集和复核每个不可变 lineage、proof、当前来源、成员和模式，再用一个事务按 Agent → Skill UUID → version 批量核对所有去重 ref，持有到原 Asset commit 返回。单次最多 1 条配置连接，持锁后无逐 ref 新事务。 |

新确认的配置回执必须先于 ImageDB CAS 持久化。用承载 Skill 行锁的事务来提交配置回执会先释放锁。外层锁加原独立配置写入仍需两条连接。现 `AdmitImageRun` 的 mode-subject → Agent `FOR SHARE` 与外层 Agent → Skill `FOR SHARE` 在 PostgreSQL 已有排队停用写者用例。生成同样保留两条预算。`modeBoundImageContexts` 通过 `LoadImageConfiguration → CheckImageExecutionMode → CheckModeSnapshot` 串行借第二条。把 mode-subject 锁直接移入已取得 Agent/Skill 锁的事务会倒置配置侧锁序，本批沿现有调用保留嵌套。采用写入继续核原 expected head、动作和来源。两库 ACK 未明沿原回执核实。 [配置事务](../../../../internal/integration/persistence/agentconfig/image_configuration.go) · [锁序测试](../../../../internal/integration/persistence/agentconfig/image_skill_lock_postgres_test.go) · [模式复核](../../../../internal/app/httpapi/image_agent_execution_mode.go) · [资源授权](../../../../internal/app/worker/imageagent/generation_runtime.go)

**保留原效果和原动作。** 请求等待、取消及重复进入均使用冻结的请求键、action、Run、plan/slot/attempt、provider task ID、资源身份和原绝对 deadline。只有原 GenerationFact CAS 的明确胜者能走原发送；超时或 ACK 丢失沿原 proof 和 UNKNOWN 核实。保留已有点数预留、结算、原供应商计量、确切 Asset 回执和 GET-only 恢复。任何容量拒绝都不能覆盖已存在的 effect 或把 UNKNOWN 写成未发送。 [原 CAS](../../../../internal/imageagent/store/generation_fact.go) · [原供应商任务核实](../../../../internal/imageagent/generation_provider_usage_execution.go)

**本批范围外待决。** 派发 CAS ACK 后但首个 POST 前释放，会允许停用先成功、原胜者随后才首次发送。配置回执提交后但首个 ImageDB CAS 前释放，会允许停用先成功、首次确认随后才写入 ImageDB。两者均继续搁置。企业关闭图片 Agent 时，带 Skill 的图位与采用仍经 `checkUseForTarget` 校验 ENABLED，无 Skill 路径沿原入口。两条路径各自适用的 `CheckImageExecutionMode` 仍检查当前 activation/epoch，这与 §7.2 的整套准入表述需另行核定。本批保留实际分支，与 Skill 停用顺序分别处理。 [Skill 合同 §8](skill-capability.md#8-冻结报价认领及撤销竞态) · [配置模式代码](../../../../internal/integration/persistence/agentconfig/image_mode.go)

### 9.3 活动并发、等待预算和恢复

完整图片 worker 内嵌在每个启用生图的 `current-application` 进程，每个进程创建一个 org worker。当前 `sdkworker.Options{}` 未显式限制该池容量；单 run 默认 4 图位。历史隔离的 pool8 API 启停日志各见一个 organization worker。2026-11-25 18:53（SGT）在同一试用 Temporal `default/image-agent-organization-v1` 的只读 activity queue 查询见 1 个近期 poller。历史所有进程是否一度重叠、其他环境的副本数与物理连接预算均未证实。每进程上限覆盖该进程内多个 run。多进程须把实例数及滚动更新重叠计入，且分别核对每个池和后端物理数据库可用连接，不能用 task queue 每秒速率代替并发。 [运行入口](../../../operations/product-image-agent.md) · [Worker 装配](../../../../internal/imageagent/temporal/worker.go) · [Temporal WorkerOptions v1.43.0](https://pkg.go.dev/go.temporal.io/sdk@v1.43.0/worker)

**装配与入场。** `buildFullImageApplication` 在完整执行分支从 `configurationDB.DB().Stats().MaxOpenConnections` 取实际 `P`。这个 pool 来自 `productAgent.database`，与 ImageDB worker pool 的上限分别读取。每个实际配置 pool 建立一个现有 `x/sync/semaphore.Weighted`，图片操作合计可取 `P−2` 个名额。注入同一个容量检查到 HTTP 新准备入口、`ConfirmImagePlan` 新写入路、HTTP 选择服务入口，以及 `ImageAgentTemporalDependencies → Activities` 的图位、策略预览、采用入口。应用同时把活动上限经 `NewOrganizationImageSetWorker → NewWorker` 传给 SDK。每个外层入口按上表用一次 `TryAcquire(1/2)` 原子取得全部名额，内部调用使用外层额度。worker 的 `RecoverEffectV3` 原任务查询也按最多 1 条进同一门，原 effect 只读回执和无需配置池的结算照原入口。没有名额时不开始配置 SQL、Skill 查找、`BEGIN` 或新效果写入。归还发生在相关 SQL/事务已结束后，错误和取消同样执行归还。 [组装与选择服务](../../../../internal/app/httpapi/image_set_application.go) · [HTTP 入口](../../../../internal/app/httpapi/image_set_module.go) · [策略预览](../../../../internal/app/httpapi/image_set_policy_adoption.go) · [worker 依赖](../../../../internal/app/runtime/image_agent_temporal_runtime.go) · [任务核实活动](../../../../internal/imageagent/temporal/activities_recovery_v3.go)

| 配置池上限 | 图片容量 / SDK `MaxConcurrentActivityExecutionSize` | 可执行的组合 |
| --- | --- | --- |
| 5 | 3 / 1 | 一个双连接的新确认可进入。第二个双连接确认在取得第一条连接前收到繁忙反馈。一个带 Skill 的活动可以与一个单连接报价/预览并行。该活动进行期间，需同时占两条的新确认等原操作再次提交。可以有两个 Run，在同一 worker 排队执行图位。 |
| 8 | 6 / 2 | 两个带 Skill 的活动和一个双连接新确认可以同进，合计占用最多 6。也可调度等额的预览或采用。保留 2 条供配置读取和原核实/收尾。 |

同样的算法给 4 连池保留 2 个图片名额与 1 个活动。池更小时，预算不足的双连接新操作沿原入口返回容量反馈，原只读和核实入口继续。首试限定经过核对的单 worker 进程。多进程或轮换有重叠时须以实际各池与物理 PostgreSQL 预算核算。余量用于阻止本批活动占尽池，普通配置 GET、Skill 候选目录、标题/Chat Skill、POD/其他 Agent 的配置读取和管理操作仍共享余量。目录 `CandidatesForTarget` 串行读取，无外层 Skill 锁。标题 dispatch/adoption 及 Skill 管理持锁入口属于现有其他消费者。本批不修改其停用合同或并发；首试验证记录实际重叠。配置回执、采用和预览的工作流入口均计入图片容量，纯控制与读回沿原路径。 [配置池](../../../operations/product-image-agent.md#配置与启动) · [目录读取](../../../../internal/integration/persistence/skill/directory.go)

**满额与期限。** 新 HTTP 操作容量不足时立即使用已有 503 `IMAGE_UNAVAILABLE` 响应。仅确定在操作入口就拒绝时添加 `Retry-After: 1`，由现 BFF 在已校验的图片错误响应中透传此提示。现图片面板对该明确分支保留原动作与参数，沿“继续原确认/继续原操作”重试。当前通用 `IMAGE_UNAVAILABLE` 会清除新 intent，需为明确的容量反馈补这个有界分支。无需自动发新命令，也无需把明确拒绝表述为结果未知。已读到 ImageDB 准入的确认照原回执核实与原工作流启动。原配置准入刚提交但尚未写入 ImageDB 的同 action 请求，即使满额，也保留原确认编号等待恢复。 [错误映射](../../../../internal/app/httpapi/image_set_module.go) · [BFF 错误响应](../../../../web/listingkit-ui/src/lib/server/workbench-proxy.ts) · [原请求面板](../../../../web/listingkit-ui/src/components/workbench/acquisition/product-image-set-panel.tsx)

SQL 借连接/锁等待的首次试用上限为 1 秒，取阶段上限和调用上下文剩余期限的较小值。对持续持锁的事务，短 checkout context 只用于取得连接，事务用保有原 provider/HTTP deadline 的上下文，行锁语句用同事务 `lock_timeout` 限时。不能用一秒的事务 context，否则会在原供应商调用期间提前回滚。HTTP 保留现 32 秒 BFF 窗口，采用继续用原 30 秒上限。Temporal 以现 task queue 和 SDK activity 上限排队，无本地无限等等。活动被派发时若容量不足，走现有有界 activity retry，沿原 attempt 在任何新预留/CAS 前退回。对新图位 activity 使用到原 `DeadlineAt` 与 `LifecycleDeadlineAt` 较早者的剩余时间，加原 60 秒收尾窗口作为 `ScheduleToCloseTimeout`，按既有 workflow patch/replay 机制保留历史分支。出队、活动重试和原 CAS 前复核同一绝对 deadline，provider、心跳/取消、原结果核实与结算仍沿原限制。仅有原未发送证明时才映射为未发送，队列或重试超时本身不能证明旧 attempt 从未发送。 [Temporal Activity 超时](https://docs.temporal.io/encyclopedia/detecting-activity-failures) · [图位 Activity](../../../../internal/imageagent/temporal/slot_workflow.go) · [原尝试执行](../../../../internal/imageagent/temporal/activities_execution_v3.go)

### 9.4 实施验证与准入

- 复用 §9.1 RED 及既有 Skill 目标/归属、配置写者排队、回执、冻结输入、结算和 Asset lineage 测试。两连接 PostgreSQL 夹具只验证释放短读后同池报价/配置前进、配置快照同事务复核，以及带两条额度的嵌套操作能推进并在满额拒绝时不借第一条。用屏障保持慢操作占住第一条，断言短报价在借唯一剩余连接读取材料后归还，再在同一池完成依赖读取。此夹具用于连接层因果，完整 HTTP 和 worker 使用 5/8 配置验证。
- 使用现 HTTP、Temporal 和模拟 provider 夹具，把两个 Run 分别置于 5 和 8 连接预算。用屏障核对饱和时查询可读、双连接确认满额反馈及同键重试、8 的实际活动并行、5 的等待状态、多个 Skill ref 采用最多一条配置连接。分别卡住配置回执后的 ImageDB CAS、原供应商返回和 Asset 提交，验证停用等待、取消后归还额度、停用后拒绝新动作。纯准备读后在快照事务前停用或改变 ref，验证拒绝新快照。
- 用既有并发回执和恢复夹具补缺口：原确认/采用 ACK 丢失、排队跨 deadline、活动满额重试、跨 scope 和取消不会新增 attempt、刷新报价或重发。用原 task 查询、结算和相同 attempt 断言保存的 proof/费用不变。在窗口到期后若已有 effect，验证核实或 UNKNOWN 状态继续由原 owner 处理。真实环境试用与用户验收各沿原授权。达到 `IMPLEMENTATION_READY` 由协调方独立核对并在 #1641 正文记录。
