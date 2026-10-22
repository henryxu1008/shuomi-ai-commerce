# 硕米最终 UI / IA Authority

> 状态：Active product authority  
> 校准日期：2026-09-05；ListingKit 退休定位澄清：2026-09-26（不代表全部节点或功能状态重新验收）  
> Figma Authority：页面 `31:463`「硕米官网」  
> 适用范围：最终产品信息架构、导航层级、页面归属、用户可见命名、交互语义与产品投影

## 1. Authority 决策

硕米最终成品的 **UI / IA / 页面命名 / 用户交互语义**，以 Figma 页面 `31:463`「硕米官网」中 **当前可见、非归档** 的设计为准。

该 Figma 不再只作为视觉参考，而是最终产品的 Product Projection Authority。

它回答：

- 最终用户看到哪些一级/二级/三级模块；
- 一个领域能力最终投影到哪个产品页面；
- 用户使用什么业务语言理解任务、店铺、商品、智能体与企业资源；
- Human Review、异常、执行进度等状态如何进入用户体验。

它不回答：

- 当前代码是否已经实现；
- 某能力是否已经通过 production gate；
- canonical facts、状态机、权限、幂等、审计等领域 ownership；
- Agent 是否可以绕过 deterministic validator 或人工授权。

因此，**最终原型 ≠ 当前 release capability**。

### 1.1 ListingKit 不是目标产品组成（2026-09-26）

用户重申：ListingKit 需要退休，当前产品以 Figma 设计为准。**旧 ListingKit 产品投影、Task-first 工作台及 root 混合架构不能作为长期子产品、默认入口或永久执行引擎保留。** 这不只是“不让它决定导航”，也不能在能力图中换成“内部引擎”继续保留同一旧 owner。

有效行为按 [Legacy Policy](../refactoring/legacy-hard-cut-policy.md)、[Register](../refactoring/legacy-register.md) 归当前 Product / Listing / Marketplace / Integration / App 等 owner，调用方切换后退休原路径；不增加 wrapper、fallback 或双事实源。Listing 领域事实继续存在，不等于保留 ListingKit。

`web/listingkit-ui` 中已经符合 Figma 和当前领域合同的 Console / BFF / 共享组件继续复用；旧 Task-first 页面逐项退出。真实目录、命令或配置名称尚未修改只表示工程现状，不是产品保留决定，也不授权仅凭名称删除合格实现。本文不宣称物理退休已完成，不授权业务迁移、部署或操作真实数据。

## 2. 当前最终一级信息架构

以当前可见、非归档设计为准，一级模块为：

1. 运营驾驶舱
2. AI工作台
3. 供应市场
4. 智能市场
5. 工具市场
6. 生态服务
7. 数据服务
8. 店铺中心
9. 套餐与权益
10. 我的账户

### 2026-10-22 当前用户决定：技能库归入 AI工作台

执行 [#756](https://github.com/qq550723504/shuomi-ai-commerce/issues/756)。用户明确批准技能库移至「AI工作台」，与知识库并列；继续保留「官方模板 / 我的技能 / 团队技能」三个并列选项卡。智能市场导航及介绍页移除技能库入口，账户资料摘要、经营画像、认证信息移除重复「我的技能」管理入口。智能体配置和商品处理流程中的业务技能选择能力继续保留。

本决定替代下方2026-10-17历史增量中「智能市场归属」和「账户管理快捷入口保留」两项约束；其他领域合同继续有效。仅调整Console产品投影，不改技能数据、API、权限、事实owner、准确技能版本/ref/digest、状态机、执行、幂等、UNKNOWN、恢复或结算。

现有 /workbench/agents/skills 根页、/official、/mine、/team 和 /official/use 地址及query保持；导航归属不要求重命名URL，也不新增重定向或兼容路径。侧栏、面包屑及页面介绍统一归属AI工作台。

Figma先于生产代码核对及更新：原位复用[官方模板深色6676:3254](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6676-3254)/浅色6676:3344，介绍6734:1427/1666、个人6676:3273/3475、团队6676:3295/3614及只读6676:3320/3754。Sidebar组件6662:488的AI工作台选中字体及唯一入口说明已补齐；保留主题令牌、Noto Sans SC、分类选项卡6767:1638及原型节点。账户资料原设计没有技能管理入口。Design Basis为Reuse Existing Architecture，复用Skill V1及原Console，不新增架构边界。

本轮授权仅设计、Issue、独立工作区开发、自检与PR维护；当前实例更新、真实模型调用、授权或资金操作、合并与Issue关闭需相应明确授权。原请求、准确版本、旧UNKNOWN的5点预留、审计及数据配置保留，未以本次导航调整修改真实运行事实。

### 2026-10-17 历史用户批准增量：技能库

执行 [#756](https://github.com/qq550723504/task-processor/issues/756)，复用 [Skill V1](../architecture/skill-capability-v1.md) 合同。用户明确批准在「智能市场」下增加「技能库」，包含「官方模板」「我的技能」「团队技能」。三个分类路径分别为 `/workbench/agents/skills/official`、`/mine`、`/team`；侧栏、页内导航和面包屑保持一致。

最初由用户批准局部 Product Authority，替代 #688 原「仅账户 Dialog、不新增二级菜单」的阶段入口约束；后续已补齐 Figma 节点。个人技能及官方模板按验证身份访问，团队技能按有效当前企业及既有权限管理；原账户/智能体配置的管理快捷入口及业务执行选择保留。官方目录没有模板或服务未就绪时如实展示，不新增公开/付费交易市场。

#### 技能库着陆页（2026-10-17 后续批准）

用户明确批准按 [浅色介绍页 `6734:1666`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6734-1666) / [深色介绍页 `6734:1427`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6734-1427) 实现，保留已有数据、配置及会话，不重复派工。这一决定将根路径 `/workbench/agents/skills` 从默认跳转官方模板改为静态介绍页，复用现有 `PrimaryIntroduction`、Card、Console 主题与响应式布局。

介绍页包含用途说明、三个真实分类入口、方法复用说明、四步流程「选择来源 → 完善方法 → 保存并启用 → 业务中使用」及个人提交团队审核/执行权限边界说明。入口只导航，不读取技能目录、创建模板、启用版本、提交共享或调用模型；不显示虚构模板、数量或成功状态。根页继续验证登录身份且不要求企业，分类页与原管理/执行路径的授权、事实 owner、状态、持久化和配置保持不变。Design Basis: Reuse Existing Architecture；当前执行范围与验证写 #756。

#### 技能库分类页（2026-10-17 后续批准）

用户要求三个分类页沿用着陆页风格，先更新 Figma 与 #756 再开发。原节点已原位更新：[官方模板 深色 `6676:3254`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6676-3254) / 浅色 `6676:3344`、[我的技能 深色 `6676:3273`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6676-3273) / 浅色 `6676:3475`、[团队技能 深色 `6676:3295`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6676-3295) / 浅色 `6676:3614`，团队只读为深色 `6676:3320` / 浅色 `6676:3754`。主题、字体、按钮和分类链接复用当前 Console；可编辑分类导航、列表项、空目录与说明卡见组件板 `6767:1633`。

分类标题和用途 → 三分类导航 → 原权限工具栏 / 技能列表 → 分类使用边界；团队准入独立成区。图中的技能和企业均为标明的排版示例，不写入业务数据。正式页面使用实际名称、版本、许可、企业和策略；不新增搜索、计数、模板或后端能力。采用响应式内容流，明暗主题与 390px 可读；原 Dialog 保留。Design Basis: Reuse Existing Architecture，Skill V1 owner、API、身份/企业门禁、权限、Consent、CAS、幂等/UNKNOWN、持久化和执行语义均不变。本轮仅隔离开发，保留当前运行实例及其数据、配置与会话。

#### 分类选项卡修正（2026-10-17 用户反馈）

用户反馈三分类切换“倒像流程”。三个分类是并列入口，分类页导航改为同一排紧凑选项卡：仅显示「官方模板 / 我的技能 / 团队技能」，去掉编号、卡片和卡片内用途说明，以主题底色、文字和底部下划线标出当前分类。用途说明保留在页面标题下；手机仍为同一排三个分类，不堆叠成步骤。技能库着陆页的介绍与流程不在本次修改范围。

Figma 已先于代码原位更新上述八个分类 frame，保留原型入口和组件实例；分类组件 `6767:1638` 统一为 120×44 的选项卡。高保真导航依据为 [深色 `6767:1728`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6767-1728) / [浅色 `6767:1916`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6767-1916)。实现沿用原 Next Link 路由导航和 `aria-current`，不引入不对应页面行为的 ARIA tab 角色。

Design Basis: N/A，仅调整既有分类导航的展示；路由、领域合同、权限、身份/企业边界、状态机、事实 owner、数据与配置、持久化和外部副作用均不改变。执行与验证记录继续维护于 #756 和本轮主要 PR；当前保留实例不随源码修正自动更新。

#### 官方标题与图片方法（2026-10-18 用户批准）

执行 [#774](https://github.com/qq550723504/task-processor/issues/774)，先交付标题与图片各三份官方方法，技能创作助手列为后续增量。官方目录沿用上述深浅原节点/选项卡，分组展示适用能力、准确版本和使用示例；已安装目录与真实未安装状态分别表达。未安装维护者界面为深色 `6860:1532` / 浅色 `6860:1719`，普通用户不获安装权限。

图片方法选择与计划确认的可编辑面板：深色 [6861:1826](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=6861-1826) / 浅色 `6861:1864`；失效版本为 `6861:1845` / `6861:1883`。用户明确选择一个可用官方/团队版本或不使用；方法不增加已选任务或改写规则/报价，确认摘要显示准确方法来源/版本。个人图片定制先经原团队审核形成团队副本，不借用标题个人准入。设计示例不是运行数据，未开放的创作助手没有可用入口。

图片为新的 Skill 执行消费者，Design Basis: Independent Architecture，具体owner、指纹/撤销栅栏、安装和准入见 [官方标题与图片技能 V1](../architecture/official-skills-and-image-method-v1.md)。设计检查达到IMPLEMENTATION_READY之前不修改正式生产路径；不操作当前实例、原数据或配置。

当前设计中，历史「商品中心」已归档；不建立独立顶层 `Product Center` / `Listing Center` 作为最终 IA 前提。

`ProductSnapshot`、Platform Draft、Listing、ApprovedAsset 等仍然可以并且应该作为稳定领域事实存在，但 **领域对象不等于顶层导航菜单**。

## 3. 关键当前设计节点

### 官方知识库依据展示（2026-10-22 用户批准增量）

执行 #884，用户明确要求「官方知识库不要使用github地址」。原官方知识库 `682:359`
及现有目录/正文页面保持，原出处卡片改为纯文本资料依据名称，结合原维护方、准确版本与digest
表达资料身份；不创建外链、内链占位、资料站或新导航。API与内容不保留GitHub来源地址，
不能仅在界面隐藏。V2内容版本/公开读取合同见
[产品内依据V2](../architecture/official-knowledge-provenance-v2.md)，独立准入完成前不修改正式路径。
该局部用户决定替代旧外部出处链接要求；不授权更新当前实例、部署或实际仓库更名。

### 来源素材与资料检查（2026-10-21 用户批准增量）

执行 #30 / #307，沿 [完整来源闭环设计 §1–§7](../architecture/source-business-closure-v1.md)，用户明确确认 §4.3.1 的 [可编辑面板 `8130:2280`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8130-2280)。「我的数据 → 查看商品」继续只读并补来源/固定版本/警告；仅「我的供应链 → 待适配」来源条目增加「来源素材」，打开独立面板，不增加顶层导航，不要求店铺、AI 或上传能力。

固定来源版本的完整图片选择及用途 → 明确确认 → 原 action 回执 → 重读精确 inventory 与 Product inputs。未知结果保留原 key/选择并核实原确认；关闭面板不取消已提交动作。图片批准与资料受阻分别显示，来源警告不自动清除，Product inputs ready 不代表平台可发布。节点中的商品/图片/回执均为设计示例，不是运行数据；原详情 `3395:389` 与待适配 `3507:1965` 终稿未修改，既有深浅主题保留。

Design Basis: Independent Architecture / IMPLEMENTATION_READY，仅批准上述来源素材窄接线与普通用户只读诊断；复用当前 Preparation/Collection/Asset owner、权限、原始版本和幂等/UNKNOWN，不授权模型、平台调用或既有实例、数据、凭据操作，不增加内部试用前置。实现与准确 CI/复核证据维护在 #859；#301 保持 OPEN，独立 SHEIN 登录及 Store 同步保留。

### 模型与费率管理（2026-10-19 用户批准设计增量）

执行 [#830](https://github.com/qq550723504/task-processor/issues/830)，「我的账户 → 企业空间」`432:4483` 增加仅平台维护者可见的「模型与费率」管理入口 `7663:2225`；仍须具备当前企业访问资格，不授予普通企业管理员定价权，不新增一级业务导航。选择一个企业的一项已接线文本用途或共享图片消费者，选择已有合格绑定并维护执行费率；不是 Key/任意 endpoint 输入页，也不改变套餐/点数充值售价。

可编辑设计总区 [7658:2087](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7658-2087)，文本深/浅 `7660:2088` / `7660:2221`，图片 `7660:2143` / `7660:2276`，差异预览 `7660:2167` / `7660:2300`，回执与历史 `7660:2186` / `7660:2319`，不可用/待核实 `7660:2196` / `7660:2329`；字段组件 `7658:2118` 复用 Console 主题与字体。编辑→本地校验→任务影响和差异预览→确认发布→准确回执；旧回执与实际当前版本分开展示。原型链接只跳设计状态，不执行发布。

图片独立预览/回执深色 `7666:2226` / `7666:2245`，浅色 `7666:2259` / `7666:2278`，使用每张图片点数而非文本费率。

模型、企业、费率、版本和时间都是明示的排版示例，不能作为真实已配置/可用状态。所有管理动作零模型调用、零任务启动；图片供应商成本未接入则如实展示。旧未发送任务遇配置漂移阻断并重新准备；已通过最终发送门禁/UNKNOWN、费用、已有输出与 Review 保留。Design Basis: Independent Architecture，合同与准入见 [模型与费率管理 V1](../architecture/model-route-pricing-management-v1.md)。本轮只设计和评审，不修改代码、切换当前配置、安装或调用真实模型。

### SHEIN 已上传商品的平台状态（2026-10-19 用户批准设计增量）

执行 [#605](https://github.com/qq550723504/task-processor/issues/605)，仍在「我的供应链 → 我的选品/自有商品 → 已上传 → 查看商品」详情内查看。原资料快照 `5543:429` / `5543:4285` 原位加入「平台状态」入口，不新增顶层页面或第六阶段；上传受理、原版本审核、首次上架请求和当前在架/库存分别表达。

三模式浅色/深色节点：自运营 `7232:2272` / `7232:2491`、半托管 `7232:2311` / `7232:2530`、全托管 `7232:2346` / `7232:2565`。自运营及符合原计划的半托管须明确确认，摘要见 `7232:2451` / `7232:2670`；未来定时半托管等待平台，全托管无人工上架按钮。原请求 UNKNOWN 但当前已在架为 `7232:2381` / `7232:2600`；请求成功与库存分别回读为 `7232:2416` / `7232:2635`；驳回/版本缺失/部分失败见 `7232:2710`。

所有新视图的店铺、标识、日期和库存均标为设计示例。模式示例切换、成功回读示例仅用于原型审阅，正式产品只显示服务端实际模式/计划，不能用于切换授权。用户点击「查询平台结果」只读取；「确认首次上架」才是外部写操作，结果待核实期间不提供再次发送。平台已上架不代表全部 SKU 有货或保证结账。

Design Basis: Independent Architecture，详见 [SHEIN publication closure V1](../architecture/shein-publication-closure-v1.md)。本轮只更新设计并完成独立评审；原型不证明实现、店铺 API 准入或用户验收，实施与真实店铺操作均未授权。

以下节点用于固定当前产品语义：

- `393:321`：工作台状态 / 运营驾驶舱展开
- `427:2467`：二级页面 / AI工作台 / 硕米Chat
- `427:3005`：二级页面 / AI工作台 / 任务中心
- `427:3543`：AI工作台 / 知识库总览及「官方知识库 / 我的知识库」三级菜单
- `682:359`：AI工作台 / 知识库 / 官方知识库；设计入口不代表官方内容已实现
- `4703:572`：已批准知识库 v1 企业语义修订，「我的知识库」归当前企业所有，依据 `agent-knowledge-context-v1.md`，不表示个人私有资料空间
- `428:323`：二级页面 / 智能市场 / 智能体市场
- `1560:359`：店铺中心 / 我的店铺 / 优化版
- `429:5323`：店铺中心 / 店铺商品
- `432:4483`：二级页面 / 我的账户 / 企业空间

若后续 Figma 明确更新为新的可见、非归档终稿，应同步更新本 Authority 文档与 GitHub Roadmap，而不是继续引用旧归档 Frame。

### 店铺中心当前路径决定（2026-10-11）

用户明确要求完整对齐 `/workbench/stores` 当前Figma路径，并确认以下语义优先于旧原型实例：使用官方SHEIN API授权，不恢复子账户密码/验证码；API操作不需要固定IP、代理配置、网络检测，移除这些操作和网络正常状态；负责人仅记录业务归属，不改变已有店铺访问授权；低库存阈值由用户设置，无默认业务阈值，库存缺失仍为未知。

主画面继续用`1560:359/716`的概览、筛选、横向卡片，结合`1583:363`分组管理与`1868:570`批量归组。平台、站点、类型、统计、期限来自当前owner，原型的加拿大店铺、示例数量及正常状态不证明实际可用。完整设计/准入见[#641](https://github.com/qq550723504/task-processor/issues/641)及[店铺中心Figma路径设计](../architecture/store-center-figma-v1.md)；这项产品决定不是实现或用户验收结论。

### 3.1 账户中心 v1 阶段边界（历史记录，2026-09-19，已被当前 Issue 范围取代）

该阶段继续使用 Figma `31:463` 约束“我的账户”的导航命名、页面层级、布局和交互语义，交付范围曾限定为真实本人资料、当前企业与导航、成员及源账号操作记录、个人推广码/链接和关系统计。该范围已由本 Issue 的 Account Center — Figma Parity Delivery Batch 取代，仅保留作历史证据。

当时未开放的组件不得据此推断当前范围；本批次应以 3.2 及 Issue #408 最新产品决定为准，仍然不得通过示例数字或前端假保存伪造事实。现有账户页面继续显式表达 loading、error、empty、denied、unavailable 和只读状态，且不得删除已确认的源账号操作记录。

相关设计组件分类以 `docs/engineering/issue348-account-ui.md` 的 Profile `432:534`、Enterprise `432:4483`、Management `1631:490`/`1631:497`/`1631:504`、Resources `1634:359`/`1636:371` 映射为当前 v1 的视觉与语义约束；成员、操作记录和个人推广组件按上述 v1/未来边界记录在 Issue #408。若无 Figma 编辑权限，不通过其他渠道修改原型，仅在 Issue 中保留待标注项。

### 3.2 Account Center — Figma Parity Delivery Batch（2026-09-20）

Issue #408 的最新产品范围取代 3.1 的缩减版 v1 阶段边界。本批次以 Figma `31:463` 当前可见、非归档账户中心为完成目标，连续交付账户总览、账户设置/经营画像/认证信息、成员与角色权限、资源/套餐/额度与成员资源分配、企业通用审计、推广收益与提现。Figma 仍只决定 UI/IA、命名、布局和交互语义；真实事实、权限、幂等、账本和提现状态必须由当前 owner/API/persistence 提供，不得用原型示例数字替代。

本批次按 M1–M5 推进，一个主要分支和一个主要 PR；M3 使用 Token set-target allocation、企业 entitlement window 和 version/idempotency；M5 使用个人 referral、10% minor-unit immutable ledger、30 日结算、退款 adjustment、¥100 人工提现和 version/idempotency 状态机。2026-09-27 用户决定将新收益结算期由 14 日改为 30 日，并完善推广行为规范及现有处理边界，具体按 [结算与推广规范合同](../architecture/referral-settlement-and-conduct.md)；不因此增加冻结、罚款或封禁机制。支付事实仍以 commercial/payment owner 的真实 settled cash payment、refund 和 chargeback 为唯一来源，不得由 referral 或前端推造。当前已接入的 M1 经营画像由账户中心持久化；身份认证状态仍以 ZITADEL 与当前组织授权事实为准。

M5 提现申请还必须消费现有 canonical payout-method owner 的有效收款方式事实；渠道枚举本身不构成收款方式。当前仓库尚无该 owner，因此申请提现接口在 owner 接入前 fail closed，不能把任意 `ALIPAY` / `BANK_TRANSFER` 值当作已验证收款方式。

## 4. AI工作台的产品对象

2026-10-18 用户批准 [#798](https://github.com/qq550723504/task-processor/issues/798)：按既有任务中心 Figma 组织统一列表与详情，标题/内容审核收回待确认流程，撤去 AI 工作台并列二级“标题审核”入口。复用现有 Task/Review owner 与授权，未接入的经营诊断/运营计划、全局或今日统计、店铺筛选和步骤百分比不随布局开放。详见 [统一任务中心](../engineering/unified-task-center.md)。

同日后续明确决定：任务不限定采集商品和标题优化，先明确设计再实现，保留已有数据与其它会话。当前续作沿同一任务中心读取原目录 `canReadRuns` 所允许的多来源、多能力运行，包括已有内容、图片、上架资料、选品、经营诊断、POD 等现有消费者；不因 Chat planner 未启用而隐藏这些读取入口。BusinessTask、原生运行、待确认提案保留原身份和 owner，不互相伪装；采集仅是一种来源，未知上下文不能默认路由到采集。该增量只消费现有授权读取，不启用新的模型/执行能力或修改 Chat 创建合同。范围与阶段语义见 [原生运行投影](../architecture/task-center-native-run-projection-v1.md)，先于生产代码明确设计准入。

AI工作台当前产品结构至少包括：

- 硕米Chat
- 任务中心
- 项目中心
- 知识库
- 我的报告

其中「任务中心」是 **用户可见的业务任务中心**，不是内部 Workflow / Queue / Temporal Task Dashboard。

推荐对象关系：

```text
BusinessTask                 # 用户可理解、可跟踪、可决策的业务任务
  -> AgentRun                # 单次 Agent 运行
    -> AgentStep
      -> ToolCall / ModelCall
        -> Temporal / Queue / Internal Task
```

因此：

- BusinessTask 可以是长期产品对象；
- AgentRun 只拥有单次 Agent 推理/执行状态；
- Temporal / Queue / internal task 仍是执行基础设施；
- 不允许把内部 task ID、workflow 状态或 retry owner 直接提升为主要产品导航。

任务中心当前用户语义包括：

- 执行中
- 待确认 / 待你处理
- 已完成
- 异常任务
- 已暂停
- 工作范围
- 硕米建议
- Human Review / 用户显式决策

## 5. Product / Listing 的产品投影

共享 Product / Listing 领域模型继续保留，但最终 UI 不要求独立 Listing Center。

当前推荐产品路径为：

```text
供应市场
  -> 获取 / 接入 / 选择供给
AI工作台
  -> 分析 / 生成 / 优化 / 决策 / 执行
店铺中心
  -> 我的店铺
  -> 店铺商品
  -> 订单履约
```

Marketplace 平台（SHEIN / TEMU / Amazon）应作为共享 marketplace/listing capability 与店铺商品投影的维度扩展，不复制平台专属产品孤岛，也不为了共享领域模型重新建立顶层 Listing Center。

## 6. Multi-Agent 的最终方向与当前执行顺序

Figma「智能市场 / 智能体市场」明确表达了最终多专业智能体产品方向。

因此：

- Multi-Agent / 多专业智能体是最终产品方向；
- 当前 Roadmap 的「不先做 Multi-Agent」仅表示工程执行顺序；
- 首期仍应先证明单个 bounded Agent + Commerce Tools + Human Review 的安全性和收益；
- 不因为最终产品有多个智能体，就提前建立第二套 Runtime、Tool Contract、权限、事实源或状态机。

## 7. Authority 层级

发生冲突时按以下职责判断：

1. **Figma `31:463` 当前可见、非归档终稿 + 本文档**  
   决定最终 UI / IA / 页面命名 / 用户交互语义 / Product Projection。
2. **产品战略与 Architecture Specs**  
   决定业务规则、canonical facts、领域 ownership、安全边界、Tool/Agent contract、权限、幂等与审计；不能恢复 §1.1 已明确退休的旧产品或架构。
3. **`docs/refactoring/current-refactoring-status.md`**  
   提供注明基线的 implementation / 成熟度记录；今天的代码、接线和验收以准确 HEAD 的 Issue / PR / runtime 证据为准，不把历史状态当实时状态。
4. **GitHub Roadmap Authority（#137）与具体执行 Issue**  
   决定当前工程执行顺序、任务范围、依赖与操作权限。

冲突处理规则：

- 用户可见模块名称、导航归属与交互语义：优先遵循最终 Figma；
- canonical facts、状态机、安全、权限与副作用控制：优先遵循领域/架构 contract；
- Figma 中的「已开放」「可用」等产品状态不得覆盖 production capability gate；
- repository implemented 不等于 production-ready；
- 旧归档 Frame、旧 Listing Workspace/Task-first 文档不得覆盖当前最终 IA；
- 历史 ListingKit 战略或执行计划不得以“内部执行引擎”为理由覆盖既有 EXTRACT / RETIRE 决定。

## 8. 产品定位

硕米智能引擎的最终定位为：

> **以 AI 工作台和专业智能体为核心，以供应链、店铺、数据、工具和生态能力为上下文的 AI 电商经营平台。**

Product、Asset、Listing、Marketplace、Store、Resource 等是支撑该产品的当前领域能力；它们不反向决定最终导航结构。**ListingKit 不再列为目标架构中的领域能力或长期执行引擎**：其有效行为由当前 owner 承接，旧产品投影和混合装配按 EXTRACT → RETIRE 收口。
