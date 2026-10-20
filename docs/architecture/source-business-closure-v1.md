# #30 / #307 完整来源业务闭环设计

Refs [#30](https://github.com/qq550723504/task-processor/issues/30)、[#307](https://github.com/qq550723504/task-processor/issues/307)；[#301](https://github.com/qq550723504/task-processor/issues/301) 保持 OPEN。

Status: **PROPOSED / DESIGN_ONLY**。本轮只交付设计及 Issue/PR 记录，不修改生产代码或 workflow，不运行环境、业务测试、来源请求或凭据探测。代码调查基线为 main `9983fc6931046872004e546c6fe5b55a6c58ba02`。用户确认设计且适用独立检查明确达到 **IMPLEMENTATION_READY** 后才能实施；设计 PR 合并、业务实施、运行验收和父 Issue 关闭分别授权。

## 1. 用户结果、产品依据与范围

使用者是具有当前企业采集、我的数据及待适配相应权限的用户。一个 Delivery Batch 完成以下已有业务路径的实际接通和如实反馈，不按采集、保存、图片、测试机械拆 PR：

`新来源输入 → acquisition evidence → SourceEnvelope → SRC/Catalog/Collection 保存 → 我的数据刷新/读取来源与版本 → 用户显式转入待适配 → 人工确认来源图片 → 精确版本的 Product inputs readiness`。

公开 1688 获取 **不要求 SourceAccount、Connection、1688 登录或 SHEIN worker**。源账号登记 `pending_connection` 不是采集准入。用户可以沿已有指定商品、按页、关键词或 Browser Capture 入口；本批不增加来源/采集模式。规则开关原语义不变：成功商品都保存，只有用户已开启且匹配的来源才按原计划转入待适配；手动转入仍须确认。

Authority：

- 用户本轮明确的 #30/#307 设计授权；[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)。
- #30 的 [2026-09-11 决定](https://github.com/qq550723504/task-processor/issues/30#issuecomment-5637305932)：匿名公开获取不依赖账号；#307 正文中“正常授权源账号/店铺”仅适用于实际消费该能力的路径，不能恢复为匿名采集前置。
- [1688 产品决定](../product/pd-1688-acquisition-figma-2026-10-11.md)、[已冻结采集/规则/转入合同](1688-acquisition-figma-v1.md)、[公开浏览器采集](2026-09-26-1688-server-public-browser-acquisition-design.md)、[既有计费](2026-10-03-acquisition-channel-pricing.md)。
- [当前 Figma Authority](../product/final-ui-ia-authority.md)、[我的供应链及人工来源图批准 §7.3](my-supply-chain-shein-v1.md)。复用文件 `tg48P46SSXl6TBy9lZwg63` / 页面 `31:463` 已批准节点：采集 `1905:359`、`3792:1353`、`3792:1390`；我的数据批次 `430:2067`、详情 `3395:389`；转入 `3342:359`；待适配 `3507:1965` / `3507:3143`。本轮实际读取详情/待适配的字体、结构及 tokens，新增 §4.3.1 的独立待确认提案，不修改这些原稿或宣称产品视觉验收。

Must：准确持久化及 lineage、当前 verified Effective Organization/actor/member、原幂等/UNKNOWN、真实警告、来源图默认未批准、明确人工批准与精确 Asset inventory、刷新后仍可读取及诊断。readiness 仅表示 `product.inputs`，不代表平台规则通过、图片远端字节已验证、可上传或已发布。

Out of Scope：1688 登录/账号池、其它来源或采集模式、AI/图片生成、标题优化或自动清除来源警告、官方店铺连接/平台规则/上传、支付赠额、全仓重构、runner/验收平台、历史迁移/ID 映射/兼容、真实环境操作。保留既有内部试用；本批不是新增试用前置。没有新增 Accepted Risk；匿名来源挑战/频率/出口风险沿原已批准范围。

## 2. Design Basis 与准入

业务实施分类为 **Independent Architecture**，范围只限“现有 Product/Preparation → Asset 人工批准 → 无 Agent 的资料诊断”新增装配和用户消费接缝：新增普通用户可达的组合路径、显式 Asset DB 注入及受权读写入口，跨 Product/Asset 持久化边界。不能把整条新用户路径写成纯文档 N/A，也不重复设计已经冻结的 SRC/Catalog/Resource 状态机。

继续复用原 SourceEnvelope/SRC-1、Catalog、Collection、Preparation、SourceApprovalService、Asset repository/inventory 和 `listing/readiness.ProductInputs`。**不新增事实 owner、表、批准身份、状态机、跨库大事务、Saga、IAM 或后台恢复器**。GORM/PostgreSQL 事务、现有 HTTP/Next BFF/Console 组件和 collector/Playwright 继续使用仓库当前实现，不另造底层组件。

本轮生产 Admission=**NOT_READY / 未授权实施**。独立高风险检查只核对 §4–§6 的新装配、权限、原始版本/Asset scope 和副作用边界；已有批准能力引用原证据。用户确认与适用检查完成之前，只能修改本设计、索引和 Issue/PR 元数据。

## 3. 当前入口、owner、调用链与实际缺口

以下为当前基线源码核对，不是本轮运行结果。

| 用户步骤 / 入口 | 当前 contract → implementation → injection → consumer | 判断 |
| --- | --- | --- |
| 在线采集 `/workbench/supply/acquisition` 及 operation 详情 | `/api/v1/workbench/sourcing/1688/acquisitions`、`plans` → `app/productsourcing` → `current_application.go` 显式 Product DB/collector/charge port → 同源 BFF/采集页 | 已有，复用模式、计划、原键核实与真实失败；不重新实现采集 |
| 插件公开页面输入 | Browser Capture payload → sourcing 当前 evidence mapping → 原 publication/UoW | 已有补充入口；不是账号/profile 读取，也不新建另一事实链 |
| 商品与 lineage 保存 | `product/sourcing` InternalProducer/receipt → persistence sourcing + Catalog writer + Collection publication participant → Product DB 同一 UoW | 已有；operation 是恢复输入，SRC 拥有 evidence/receipt，Catalog 独占 ProductSnapshot，Collection 只保存成员/准确来源引用 |
| `/workbench/data/mine` 批次/查看商品 | `/api/v1/workbench/collections/items/:item_id` → `collection.Service.ReadItem` → 当前 Collection + bounded Catalog reader → BFF `product-collection` / `CollectionPage` | 已有精确读，验证 org/key/version/publication；前端 contract 已含 sources/warnings，但当前 ProductFacts 未形成完整的来源说明及缺失项展示 |
| 显式转入/待适配 | 原 Collection 选择 → Preparation transfer/source selector → `composePreparationCore` / `PreparationRoutes` → `/workbench/supply/mine` | 已有无 Store/Asset/Temporal 的窄装配；继续固定原 publication/version，不强制自动转入 |
| 人工确认来源图片 | `asset.SourceApprovalService` → 原 Asset DB receipt/assets/inventory heads；`supplyapp.SourceImageReader` 重新读取授权来源 | 合同已支持 human_source 和 target=`product`；目前 HTTP 接线在 `buildSupplyChainModule`，其构造同时要求 Store、官方应用、starter/worker 等；Preparation-only 不暴露 images approve/inventory。现有确认 UI 在平台 target editor，普通来源资料闭环缺独立消费位置 |
| Product inputs readiness | `listing/readiness.ProductInputs` → `readinessinspect.Project` exact Product/inventory 投影 | 规则及有界 DTO 已有；生产调用主要由 Product Agent Tool 消费，没有同一普通来源页面的无 Agent 诊断入口。不能为读取诊断启动 Agent、收费或要求官方店铺 |

架构问题是**已有正确业务能力与完整平台执行装配耦合，当前来源路径不能单独消费**；不是 Source/Catalog 缺另一套保存逻辑。最小补齐是原 owner 的窄装配、严格 transport 和现有页面内的来源/图片/诊断区域。

另外：[#848/#849](https://github.com/qq550723504/task-processor/pull/849) 归原 collector Writer，准确候选 `19e25089e47d05196d9b61179fba417ac09d4b2c` 尚未合并；其 Issue 记录 Chromium 启动修复完成，但真实探测 502/503，保存/计费验收 NOT_RUN。这里引用原 owner 的记录，没有执行探测，也不接手该修复。真实公开来源可用性不由 fixture/设计或匿名浏览器能启动推导。

## 4. 已确认合同的复用与拟议接线

### 4.1 输入、保存、lineage 与刷新

当前采集/BFF 原路径保持；调用方不能提交可信 Organization、Catalog version、receipt 或 evidence authority。collector 仍是无凭据、无 DB 能力的独立 leaf；网页证据是不可信输入，由 sourcing 校验。

同 Product UoW 原子保存 normalized envelope、SRC receipt、Catalog publication/version/head 和默认 Collection 成员。页面展示 CollectionItem、ProductKey、原 publication/version、source type/platform/id/URL、已有 warning/missing 信息。展示从现有 DTO/owner 精确读取，不用 URL/名称反推身份，不把前端存储当 publication。只补齐所需的安全字段投影，不公开原始凭据或任意 metadata。

当前前端 sources schema 只保留 type/platform/source_id/URL/reference_type；若展示 checksum、captured_at、snapshot/reference identity，须从当前 Catalog SourceRecord 中显式投影并有界校验，不靠浏览器重算或补默认值。SRC envelope/receipt 与 Catalog source lineage 的持久一致性由原 owner 验证；UI 展示摘要不成为第二份 evidence。

读取/刷新先授权 Collection item，再读其固定版本；来源详情/diagnostic 始终携带相同版本绑定。UI 切换 org/actor 后清当前展示并取消请求；迟到响应不能覆盖新 scope。GET 不执行采集、核实收费、转入或批准；原有需 live-write 的读取策略保持，不改成缓存授权。

### 4.2 来源素材窄能力（拟议 `sourceMaterials`）

为普通 current-application 增加默认关闭的显式能力配置，注入 **原 Asset DB 的 `supply_asset_runtime` 角色**；沿 `cmd/supply-asset-init` 的既有 schema owner 初始化，serving 只验证当前 schema/精确权限。Product/Asset DB 身份与角色保持原隔离；不从 URL、相同字符串或同一 `*gorm.DB` 推断 owner。

该能力只在已有 ProductCollections 与 Preparation 装配上启用（本闭环沿现有 AcquisitionPlans）；消费同一个 Preparation core 及其既有 schema/最小表权限，不通过开启完整 SupplyChain 获得无关 upload/operation/official 表写权限。缺少该 core 时明确拒绝，不新装第二套 Product 表或伪装计划能力。

- 复用 `composePreparationCore`、授权 source selector 和 `EffectiveProductReader` 的原始版本分支；窄模式只允许 `EffectiveCatalogVersion == OriginalSnapshotVersion`、空 ApplyReceiptID、target=`product`。不构造 Review producer/Record/official receipt/Store/Temporal/AI。其它当前完整执行模式保持其已批准的更广合同。
- 将原 SourceApprovalService/Asset reader 构造提取为当前 app owner 的窄接线 helper；完整 SupplyChain 与窄模式复用同一业务 owner。沿 `SourceImageReader` 核对 org/actor/member、Preparation source、原 publication/version/图片 reference hash；不包装 legacy service。
- 窄能力复用原 `/api/v1/workbench/supply-preparations/images/approve`、`images/inventory`，继续原 `preparation.PermissionManage`、verified identity、live-write 组织授权和请求 binder；另要求原 Collection owner 当前访问资格。客户端不提供图片 URL、org/actor、可信批准状态或 Asset ID 替身。
- 完整 SupplyChain 存在时复用其 app/Asset pool；没有时组合 Preparation + source-material app。与 acquisition plans/completion 的 Preparation routes 由同一组合点选择，**每条 route 只注册一次**；不能构造两个 Preparation service 来各自成为事实 owner。配置指向不同 Asset owner、缺权限/schema 或重复能力时明确拒绝启动，不 fallback 到内存/旧库。
- 来源批准只接受已保存来源图片的 ID/用途和原 action key，保留原 40 项上限、严格 JSON/大小/deadline/取消。第一版窄消费不接受 generated artwork/manual media/他版本资产转借；它们原完整能力保留，不能因新模式绕过原额外 guard。

### 4.3 资料诊断与页面消费

拟增加同一 supply-preparations source 上的只读 **GET `/sources/:source_id/product-inputs`**。窄模式 target 固定 `product`，版本由服务端授权 source 的原始 ProductSnapshot 得到；拒绝客户端 Catalog key/version、target、org/actor 和任何请求 body/未知 query。沿原 Preparation 及 Collection read/live owner 授权；只查询 Product/Asset，不开放写 port。

直接复用 `readiness.ProductInputs` 与已有 `readinessinspect.Project` 的 bounded exact DTO，不调用 Agent invoker/模型、Tool execution audit/预算或 provider。返回原 publication/version、inventory scope/count/digest、warnings/review reasons、`ready|blocked|not_evaluated`；依赖失败不得伪造成空 inventory 或 ready。`marketplace.status` 固定 `not_evaluated`。

“我的数据 → 查看商品”只补来源信息展示，继续只读，原下载/明确转入动作不变。新增写入口仅放在“我的供应链 → 待适配 → 来源条目”的 **来源素材** 操作，打开独立“来源素材与资料检查”面板。该面板复用原图片选择集/角色与明确确认交互，不挪用原只读详情为写入口，也不进入平台 TargetEditor 的店铺/规则/保存上传路径。

批准前显示候选及缺失，批准后重新读取精确 inventory/diagnostic。ProductInputs 有 source_review_required 时仍 blocked，不自动做标题 Review/Apply、不清除 warning。无权限/未配置/依赖故障如实表达，不能用开关或隐藏按钮代替后端授权。新面板与原平台图片交互复用组件；不新增顶层导航、账号页或平台 Workbench，现有深浅主题保留。

### 4.3.1 Product/Figma 增量提案（待用户确认，不是已批准 Authority）

可编辑提案：[来源素材与资料检查 `8130:2280`](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8130-2280)，位于同文件/页面的独立新区域，名称明确标注 `#859 / 设计提案`。字体复用当前文件的 Noto Sans SC/Console styles，色彩/间距/圆角绑定原 SUMI tokens；全部为可编辑 Frame/Text/组件实例，没有整页截图底图。该图是深色交互示意，不取消既有浅色主题。商品、图片引用、版本、确认记录和状态均明示为设计示例，不是运行数据。

| 提案节点 / 状态 | 用户操作及可观察结果 |
| --- | --- |
| `8131:2291` 未批准 | 显示固定来源版本、候选图片及用途；用户明确选择完整集合后点击“确认采用所选图片”。不提供替换 URL/自报批准或店铺选择 |
| `8131:2309` 确认并重读 | 仅在原 action 回执确认后显示采用集合/作用版本，再读 Product inputs。刷新恢复当前服务端事实；“重新选择图片”是新的明确动作，不自动改变旧回执 |
| `8131:2324` 结果待核实 | 保留原 scope/action/选择，冻结新确认；“核实原确认”只沿原 key/载荷核实/重放。无法确认时保持该状态，不显示成功或换新 key |
| `8131:2330` 批准成功但资料受阻 | 真实显示 source warnings/缺失；允许重新读取诊断，不把图片批准写成整份资料通过 |
| `8131:2336` 无权限/未配置/依赖不可用 | 明确区分权限拒绝与无法读取；禁止确认，切换企业后清除上一身份资料，不以读取失败推导未批准/就绪 |

关闭面板不取消已提交 action，也不清理未确认意图；回到原 scope 后核实原操作。原始资料只读、待适配操作位置、完整选择/确认、结果核实与错误表达是本次请求用户确认的产品增量。**用户确认该增量之前，生产实施仍 NOT_READY；不得将旧详情/待适配节点或这张提案自行标记为已批准的新写入口。** 确认后将准确节点/语义同步到原 Product Authority，沿同一设计 PR 实施，不新开一轮 UI 产品需求。

## 5. 权限、事务、幂等与失败边界

Threat Model 沿当前 verified identity/Effective Organization、actor-private Collection/Preparation、Asset source approval 与 collector 的边界。保护跨 org/actor 访问、伪造来源/批准/版本、重复收费/批准、UNKNOWN 换键重投；不扩大到隐藏账号存在性、通用 IAM、瞬时全局撤权或通用反爬规避。

| 边界 | 原 owner 与不变量 | 失败/响应丢失/重启/并发处置 |
| --- | --- | --- |
| acquisition/plan 与 Resource | 原 operation/plan 的冻结 command、claim/fence、资源 reservation/proof；服务端每成功 publication 原规则消费 1 DATA_ROW，本地/插件沿原定价 | 原 key/command Verify/Read；UNKNOWN 保留 reservation，不重采、不换 key、不通过新 flag 免计费；读回不再次消费 |
| SRC/Catalog/Collection | 同一 Product PostgreSQL UoW，当前合法 source identity，immutable publication/receipt | 失败原子回滚；未知 COMMIT 由原 upstream 持有原 command 核实，Collection/下游不能自己补写 Catalog 或重建 publication |
| Preparation transfer | 原 source/member/revision/批次、live 权限与同 DB transfer receipt | 同 key 同输入重放、异载荷冲突；已转入来源不随批次改名/移动改写，沿原 source selector 当前 owner 检查 |
| Asset source approval | 原 Asset transaction 保存 action payload hash、assets、receipt/inventory head；human_source 的 Agent 字段保持空/零 | UI 保留原 scope/action/完整选择集；按原 Approve 同 key 同载荷重放取得回执，异载荷冲突。未核实保留 UNKNOWN；不能用当前 inventory 冒充原 action 回执。旧 action 重放不得把后来的 inventory head 改回旧集合 |
| Product ↔ Asset | 先由原 source selector 刷新授权并核对不可变 Product 版本，再由 Asset 原 transaction 提交该 scope 的批准 | 不把两库包成一个新事务；批准失败不删除已保存 Product。original/effective version 与平台绑定不同不得借用批准；后续 Product 新版本不自动继承旧 inventory |
| readiness/页面 | exact Product + matching Asset inventory 只读推导，不持久化第二份 ready 状态 | 无批准是明确 blocked；依赖失败是 unavailable/not_evaluated；损坏/跨 scope/错版本拒绝。刷新、重启、迟到响应不能改事实、开始采集或批准 |

撤权、组织/成员不可用、授权依赖故障遵守各现有 owner 的拒绝策略；窄装配不得用旧 operator/viewer 静态角色恢复授权。没有新的 bearer retention、后台 writer、清理器或跨组织缓存。来源 URL/图片只作为原 owner 的安全引用展示，不为了本批放宽 SSRF、URL/重定向、TLS、容量或平台图片读取策略。

## 6. 完整实施、保留/删除矩阵与证据计划

确认后一个主要 PR 连续覆盖下列接缝。范围相关文件预计少于 30，若实际 owner/权限/事务/持久化设计超出本文，先报告具体差异；不为达阈值拆小 PR。没有获准删除新的业务闭包。

| 处置 | 范围 / 当前 owner | 条件 |
| --- | --- | --- |
| KEEP | sourcing/SRC/Catalog/Collection 保存、Resource/Commercial、计划/插件/采集器、SourceAccount、Store、当前身份/权限 | 原合同不变；不为本闭环重写或追加账号前置 |
| EXTRACT（当前代码复用，非 legacy 包装） | app/httpapi 的原 SourceApproval/Asset reader 接线、来源图片选择 UI | 原完整能力与窄模式共用；消费者切到同一 owner 后删除被替代的局部重复接线，不删除业务规则 |
| ADD/EDIT（当前 owner） | 显式 sourceMaterials 配置/注入及精确 DB 权限校验；原 source routes/BFF 的窄选择；product-inputs 只读 transport；现有 Collection/来源面板及原 pending command | 无新 schema/事实、状态机、计费、后台执行；缺配置如实不可用；不能导出完整上架 routes |
| RETIRE / 保持缺席 | 已退休 tenantbridge、旧数字 Task/handoff/HTTP/profile、legacy import/migration/fallback | 原 absence/边界 guard 保留；本轮不因旧名字扫描再造删除批次 |
| KEEP，未获准退休 | 独立 SHEIN 登录 worker、数字账号/Store DB、Redis/Profile/attempt、验证码/取消/health/shutdown、Store 同步及 tenant 查重 | #301 OPEN；既有实例、数据、凭据和部署零操作，不向当前 Organization/Store 接桥 |
| 不变 | 两套 workflow、inventory/ceilings、依赖/锁、保护规则、#849/#692 与其它 owner 工作区 | CI 用原规则；不新增 trial gate，不把本设计作为其它核心能力前置 |

Legacy decision: **EXTRACT | RETIRE**（仅遇到已登记旧 owner 时适用）；Reusable behavior 为当前纯规则/授权/精确读取及 Asset 批准；Current owner 如上；Cutover/deletion condition 是当前消费者切换且无保留职责。旧持久化/运行态不在删除范围。

最小验证按用户操作覆盖整条链，复用现有测试/fixture，不建设 runner：

1. 保留并按受影响增量复用 `TestAcquisitionBusinessChainPublishesExactVersionsAndObservesRevocation`、`TestAcquisitionBusinessChainRecoversLostResponseUsingFrozenCommand`、`TestAcquisitionMountedCurrentApplicationFullChainAndLiveAuthorization`、`TestAcquisitionMountedHTTPResponseLossAndApplicationRestart`；原 SRC/Collection PG 证据与 source warnings/limits 保持。仅新增缺失的当前 app→BFF→页面→人工批准→readiness 组合断言。
2. 先用测试捕获新窄装配/权限/来源回执缺口，再实现：无 Store/Agent/Temporal 时可保存、刷新、转入、批准和读诊断；任何平台上传/执行 route 缺席；和完整装配共存时无重复 route/owner。
3. 跨 org/actor、伪造 URL/版本/来源 ID、撤权或 grant 失败无新写入；批准前 inventory 缺失，人工动作后 exact inventory；同 key 重放、异载荷冲突、丢响应/重启/后续 inventory head 均沿原 owner；source warning 存在时不能冒充 ready。复用 `TestSourceApprovalDoesNotInventAgentIdentityAndBindsEffectiveVersion`、`TestSourceAndAgentInventorySelectionReplacesFullSet` 和 `TestProductInputsRequireScopedApprovedAssetsAndExposeSourceReview`。
4. 未来获准的任务自有新 PostgreSQL/合成 token/grant/公开 evidence fixture，只能证明代码/组合，不证明真实 ZITADEL、1688 页面、保留环境隔离或用户验收。不得复用可能清理固定项目资源的历史 CHAIN-1 脚本去触碰现存资源，也不为此扩展清理/验收平台；没有任务自有资源权限就保持 NOT_RUN。
5. 当前浏览器在批准的独立范围操作正常入口；身份与来源替身、真实页面、用户/平台验收分别报告。源码/PR CI/实际 merge push-main CI/环境/产品验收分列；设计阶段全部业务执行 **NOT_RUN**，没有旧 PASS 自动升级。

真实公开 URL 的采集、成功发布的 DATA_ROW 消费、新环境启动及真实身份/素材操作，必须先明确环境/资源归属和独立授权。本轮仅设计，不继承 #856 临时 PG 权限到新任务，也不执行 #849 的真实来源探测。不能要求先取得真实环境 PASS 才允许在已批准架构下开发隔离代码；也不能用隔离代码 PASS 签发 #30/#307 整体验收。

## 7. 交付、准入与停止点

本轮交付本设计、索引及 #30/#307 当前段；历史正文/验收项保留，#301 不关闭。一个 design PR 后可在同一主要 PR 连续实施本用户结果，前提是用户明确确认设计和实施范围、完成 §2 的适用独立准入。设计阶段不派额外 Writer，不重复触发同 SHA 复核，不扩展全局架构。

首轮独立检查在设计 `57af1f5d3` 提出新面板缺少准确 Product/Figma 增量；已先分类 **BLOCKER（生产实施准入层：核心人工批准操作尚无获准入口合同）**。本轮已补 §4.3.1 的具体提案和状态；设计交付继续，但用户确认前不宣称该产品门槛已清除或 IMPLEMENTATION_READY。未变化的业务合同不重开，后续只核对此增量。

独立检查重点是新增可达路径是否只获得原 Preparation/Collection/Asset 权限、只读诊断是否确实无副作用、Product/Asset exact binding/原 action 重放是否成立、窄能力是否与 full execution 隔离。finding 先按 AGENTS 分类；真实跨租户/错误授权、错误批准/数据损坏、重复计费或核心路径无法完成才重新打开相应设计；其它实现细节进 IMPLEMENTATION_TEST，非当前需求进 BACKLOG。

确认后实施完成进入 In Review，集中做一次最终交付检查；原 user flow 的实际入口/操作步骤、准确代码/CI/review、已保存方式、失败与未开放能力交用户验证。未经准确 HEAD 授权不合并，不关闭 #30/#307/#301，不部署/切换/停机/清理，不读既有数据或凭据，不增加内部试用前置。生产仍 **NOT_READY**；本轮实际运行、真实来源、用户/平台验收 **NOT_RUN**。
