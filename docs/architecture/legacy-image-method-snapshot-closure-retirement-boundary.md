# 旧 ImageAgent 方法策略与历史快照兼容链完整退休设计

> #301 · Independent Architecture · DESIGN_ONLY / DESIGN_REVIEW_PENDING / SOURCE_NOT_STARTED / NOT_READY。当前仅交付本文、架构索引和 Legacy Register 三文档；生产源码尚未获准。用户确认本设计并完成适用独立复核后，在同一主要 PR 整批实施，不机械拆批。

## 1. 用户结果、范围与 Design Basis

使用者是维护当前图组、Artwork、采集和 Store 观察同步的开发者。交付结果是当前 ImageAgent 只接受当前方法与计划合同，退出旧版本、旧文案生成和历史宽松校验，避免同一 owner 继续维护多套内部语义。没有新用户入口、平台、运行操作或试用前置。

- 用户已同意先设计四组完整闭包：图片方法 v1/v2/v3、旧详情文案及专属尾部、历史计划宽松校验、四笔事故专用 `image-reconcile-774`。这不是源码实施或旧快照承诺撤回的最终批准。
- 调查 main：`32fedd54283092135dc80ce31697c2e0ef8b0477`（#1001 已合并，准确 push/main CI 成功）。[接单与范围](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6500498384)。原线程/独立 worktree 单 Writer，分支 `codex/issue301-image-method-snapshot-design`；只有无关 UI PR #1003，无重复派工。
- **未来源码 Design Basis: Independent Architecture**：改变旧计划读取、恢复和公开方法策略的支持边界，不能按机械死代码删除。当前三文档为 N/A / Ready；源码保持 NOT_READY，直到适用独立设计结论和用户确认 §1–§7（特别是 §4.2、§6.1）。
- 不做：新图像能力、模板/Skill 发布、翻译或 provider 调用、新状态机/表/迁移/协议版本、兼容 alias/fallback、双读写、第二事实 owner、配置别名/浏览器清理、并行预算/投影旧问题修复或验收平台。
- GitHub workflow、完整 inventory、依赖、保护规则零改；不读取或操作既有实例、数据、凭据、进程、对象、真实历史或队列；不增加内部试用前置。#301 保持 OPEN，合并/部署/运行验收/关单另行授权。

## 2. Authority、前批 KEEP 替代与保留能力

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[图像产品架构](product-image-agent-v1.md)、[当前图片文案与语言合同](image-text-language-confirmation-v1.md)。不扩大 Product/Threat Model，不新增 Accepted Risk 来掩盖当前 Must 缺陷。

当前 Figma Authority 沿用语言确认设计的[深色 7811:2248](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7811-2248)与[浅色 7811:2273](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7811-2273)。不改变当前页面布局、入口、人工文案确认或操作顺序；只移除不再支持的旧版本状态展示及分支，无新 Figma 增量。

| 依据 | 本设计提议准确替代，待用户确认 | 继续保留 |
| --- | --- | --- |
| #1001 [历史执行协议退休](legacy-imageagent-history-protocol-closure-retirement-boundary.md) | “当前 GetVersion=1”不保证旧方法/旧计划还能读取或执行；本批另按领域合同拒绝 §4.2 对象 | 14 个 ID / 15 个原读取位置/current1、当前 Temporal V3、Update/Resume、预算 false 事实、绝对 deadline/取消/finalization/恢复 |
| [官方方法 V1](official-skills-and-image-method-v1.md)、[视觉 brief V2](image-skill-visual-brief-v2.md) | 方法 policy v1/v2 的历史有效性、只读展示、旧准备请求重放和旧结果新增采用承诺 | Skill 唯一 owner、准确不可变包/ref/digest、视觉资源、授权/可用性 fence、当前 v4 组合器；Skill 数字版本及 `skill-method-v1` 不退出 |
| [字面文案 V3](image-detail-visible-copy-v3.md) | v3 自动从商品事实生成标签、旧 prompt 与验证/恢复分支 | 当前显式无文字或已人工确认的文案、事实核对、目标语言、当前 prompt 与原安全限制 |
| [文案语言确认](image-text-language-confirmation-v1.md) | 旧 `Text=nil` 请求和计划的原样重放/展示要求，包括 Method=nil 的旧图组 | 当前 `Text.Mode=none/reviewed`、审核版本/原始资料 lineage、已冻结当前输入、幂等与回执 |
| [#774 四笔对账](bailian-image-no-generation-reconciliation-v1.md) | 仅四个固定身份的事故命令源码与 README 活跃入口；旧计划展示不再作为本批承诺 | 通用 no_generation 证明、原 UNKNOWN、成功优先、防重发送/预留、经济回执、审计/事件 owner；已有记录和图片一行不改 |

Must：当前身份/租户/member/actor、授权撤销、同键异载荷、准确 source/Skill/模板/quote、单 pending、持久化 ACK、CAS/ledger/fence、provider 与资金/发布幂等、UNKNOWN 与取消恢复不降低。当前图组/Artwork、采集单主图、Store 观察/官方授权及 JSON/hash 保留；#985 已退休的 SHEIN 登录/身份回写不恢复。

Legacy decision: **RETIRE + SELECTIVE EXTRACTION**。复用当前严格计划规则、`validateReviewedImageText` 和 `imageTextPrompt`，留在现有 ImageAgent owner；不包装旧方法、增加新 parser 或兼容层。

## 3. 实际消费者、完整尾部与 owner

| contract → implementation → injection → consumer | 事实与根因 |
| --- | --- |
| PrepareImageSet → `ImageMethodPolicyVersion` → ImageSet/worker | 新建方法固定 v4，但 `ImageMethodBinding.Valid/Executable` 仍区别对待 v1/v2/v3；Skill adapter 还为 v1 返回空材料。版本分支已经渗入准备、派发、采用和恢复 |
| 原请求 ID → PrepareImageSet 已有投影分支 → PreparedImageSetFromProjection/Get | 仍按已存旧 policy 重算请求 digest，且只在“新记录”分支要求 Text。只改新建入口或删常量会漏掉旧幂等重放、GET 和已准入确认早返回 |
| `Text=nil` / 旧方法 → `usesLiteralDetailCopy` → 构造/校验/恢复 | 五个旧 helper、`visualImagePrompt`、构造器里被当前文案覆盖的 JSON/视觉 prompt，以及 generation_input/asset_catalog/store 恢复条件共同构成闭包；当前文案规则直接改接原 owner |
| ValidateSubmittedPlan / AgainstCatalog → canonical validators → Service、Temporal、Repository、Asset publication | 旧 ValidatePlan 允许空 pending、超 32 slots、非 JSON-safe 修订；旧 catalog 校验放过 size 首图缺尺寸。初始/替换 revision 的业务检查仍独立有效，不能将它们一并删掉 |
| Run DTO → TS schema → 图片面板 | 枚举仍接受 v1/v2/v3，Text optional，面板有 v1 禁生图提示、v3 字面文案及旧方案补选文案状态。必须随领域合同转换，而非留下可用假象 |
| `hack/debug/image-reconcile-774/main.go` → 四固定指纹与证据文件 → 原 owner | 是独立事故命令，非通用 UNKNOWN owner。#774 记录过该四笔已释放的证据；本批仅据公开记录设计，不重新核验真实账务，不运行命令，不把“未读取当前环境”当作排空证明 |

owner 不变：组织 Repository 拥有 run/plan/slot/projection/commit；Skill 拥有包与授权；Product Review 拥有已确认文案；GenerationFact/V3 ledger 拥有效果事实和恢复；Asset/Resource/钱包及其原事务和回执保持。

## 4. 唯一当前合同与持久化边界

### 4.1 当前值、校验与 hash

1. `ImageMethodBinding` 非空时只接受原 v4 字符串。Method=nil 仍允许；Skill 的版本号、完整包、`skill-method-v1`、PLATFORM/ORGANIZATION 限制和 fence 不变。删三旧常量和不再有不同语义的 Executable 方法，派发 guard 保持既有错误分类与 nil Method 行为。
2. **ImageSet 必须有当前有效 Text**，包括显式 `none`；旧 Text=nil 即使 Method=nil 也退出。当前 `reviewed` 的准确审核副本、语言、TaskIDs、普通标点、事实/evidence 与 prompt 全部按原规则。Artwork 和采集单主图不因此需要 ImageSet.Text。
3. 五个旧字面文案 helper 和 `visualImagePrompt` 整退。asset_catalog/generation_input 直接调用现有 `validateReviewedImageText`；该 helper 对非 ImageSet 仍允许 nil set，对 ImageSet 的 nil Text 明确拒绝。构造器只保留当前 `imageTextPrompt`，退被覆盖的局部 `visualProductFacts`/payload/旧 prompt 及专属调用，不改最终当前 prompt 字节、顺序、引号、语言或字段。
4. 原严格规则归 `ValidatePlan`：JSON-safe 修订、最多 32 slots、显式 pending；Artwork 原分支也必须执行同样当前约束，且其直接 validator 不再接受空 pending。`ValidatePlanAgainstCatalog` 收入原 size 首选源必须有真实尺寸的规则。删除两个冗余 Submitted 门面并改接所有生产调用；`ValidateInitialSubmittedPlan` / `ValidateReplacementSubmittedPlan` 保留初始 1/0 和单步修订业务含义。不为零修订/空计划加跳过校验；原缺投影或初始化未完成的错误合同保持。
5. 两个真实 Repository 的 GetProjection 已调用 `ValidateProjectionSnapshot → ValidatePlan`，复用该现有链封闭旧读取，不在 Service.Get 重复增加 validator 或改变身份顺序，不把原始读取改成迁移/过滤。Prepare 的 Text 在已有/新记录之前统一要求；旧 policy 的 input digest 重算分支退出。已有请求路径仍校验原身份与当前计划，原同键异载荷、source revalidate 及准确当前回放不变。PreparedImageSetFromProjection 对缺 Set 显式失败，不能解引用 panic。
6. Confirm 在原身份/修订/digest检查后、任何已准入回执早返回或 StartManual 前经 PlanDigest 校验当前计划。**当前 v4 的原 admission 恢复仍先于新 source/Skill/quote 准入**，不因本批把撤销后的安全关闭路径堵死。`ValidateImageSetAdmission` 在原 pending-copy 前拒绝空 slot.Status；保留原非空当前执行状态的 plan-definition pending-copy 和准确回执比较，不能借复制把已退休空状态修成有效计划。Asset publication、直接 tools、worker 和结果恢复沿现有 validator 调用链拒绝旧计划；不复制领域验证到各适配器。
7. `ImageSetQuoteDigest` 保留当前 Route+Text 与无 Route+Text 的两种原编码，退出仅 Method/仅 quotes 的旧 Text=nil 分支；计算前拒绝非当前 set，不能递归调用依赖 QuoteDigest 的完整 plan validator。PlanDigest 改用 canonical ValidatePlan，**原 JSON 声明/tag/顺序、nil/空值、omitempty、当前 SHA256/指纹/版本和值完全不变**。当前无 Method、有/无 Route、none/reviewed 四种组合必须证明字节与 hash 等价。
8. `catalog-v1` 仍用于当前 Artwork 的空素材目录，catalog-v2 仍用于商品上下文；两种 hash、Normalize、ProductContext 零值及 SourceIdentity 四零值槽位全部保留。不把旧方法 policy 与 Skill 数字版本、ImageSet schema、Temporal V3 或所有“历史”一词混同。

### 4.2 必须由用户明确确认的 hard-cut

| 对象 | 提议行为与准确限制 |
| --- | --- |
| 方法 v1/v2/v3、任意 Text=nil 的旧 ImageSet | 不再承诺业务详情展示、原准备请求恢复、确认/续跑、重新生成、新采用或按旧 recipe 物化恢复；即使 Temporal markers 全为 current1、属于组织模式或已有 admission 也同样适用。不能仅凭 #1001 的旧授权宣称本决定已批准 |
| 空 pending、超上限、非安全修订、size 首图缺尺寸的历史计划 | canonical 当前验证拒绝，不自动补字段、裁剪 slots、换图、改 revision 或重算已存 digest。已保存字节/记录不动；拒绝沿既有 ErrValidation/ErrCommandBlocked 等错误 owner，不新建 legacy 错误协议或迁移服务 |
| 当前合法的不可变旧来源版本/准确 Skill/审核版本 | 仍是当前 v4 的有效冻结事实；继续正常读取、采用、幂等和恢复。不能按创建时间、版本号或名称里的 historical 退休当前合同 |
| 读取与列表 | Service 的业务读取不再输出不受支持的计划；列表/分页沿原授权、source 核对及错误处理，不能偷偷丢弃旧项、伪造 totals 或返回部分成功。原始事实/审计/回执的 owner 读取不删，不为“恢复显示旧图”重建旧 parser |
| 已保存资产、成功/UNKNOWN/经济事实 | 不删除、不回写、不改钱/点数/预留/receipt；通用事实查询、成功优先、UNKNOWN 不重发、准确 no_generation 证明与结算关闭保持。读取旧原始事实不授予旧计划执行许可；不承诺旧计划通过当前 UI 继续采用/恢复 |
| 四笔事故工具 | 只删固定四身份命令与专属装配；#774 及既有事件/证据文件不删除，通用 generation rejection/reconciliation、资源释放与防重实现不删。本次不证明实际资金或队列已全部结清，更不自行运行旧工具 |
| 运行/rollout | 实际旧计划和任务存在性 UNKNOWN；源码切换可能使已排队旧任务校验失败或继续重试，不保证自动 terminalize、排空或撤回共享 Activities。当前新合同恢复保留，不操作真实 Temporal/数据库/对象/进程，不增加内部试用前置 |

若发现新的具体当前外部义务，先提供证据报告用户；不得由 Reviewer/实现者恢复兼容分支。本次设计确认只批准源码边界，不批准真实环境清理、迁移、部署或旧事实处置。

## 5. 完整删除 / 保留 / 抽取矩阵

**本次设计提交只有三文档。未来源码同一 PR 覆盖以下 3 个生产整退文件、16 个生产有界文件及全部专属检查/文档尾部。** 测试文件按 §6.1 的同根 fixture/调用改接边界授权，不为凑文件数改未受影响测试；最终实际路径与准确 HEAD 写 PR。

| 文件 | 决定与边界 |
| --- | --- |
| `internal/imageagent/image_detail_copy.go` | RETIRE 五个旧字面文案 helper 全文件；有效当前分支改接原 image_text owner |
| `internal/imageagent/image_visual_prompt.go` | RETIRE 唯一旧视觉 prompt helper，最后消费者随构造器整退 |
| `hack/debug/image-reconcile-774/main.go` | RETIRE 四固定身份事故命令及局部证据/文件/配置装配；不执行、不动外部文件 |
| `internal/imageagent/image_method.go` | 三旧常量、Executable 与旧空材料分支退出；v4/准确 Skill/组合器/权限 fence 保留 |
| `internal/integration/skillimage/context.go` | 只删旧 policy 返回空材料分支；当前授权、规范包/ref/digest、持锁和 release 保留 |
| `internal/imageagent/image_text.go` | 原 helper 对 ImageSet nil Text 拒绝、校准旧注释；当前正文算法、语言/标点/复制及 DTO 不变 |
| `internal/imageagent/validation.go` | 当前严格边界归 canonical ValidatePlan，删冗余 Submitted 门面；初始/替换单步与原业务校验保留 |
| `internal/imageagent/artwork.go` | 只退出空 pending 接受；当前 Artwork 字段/纯 hash/无原图/quote/来源证明保留 |
| `internal/imageagent/asset_catalog.go` | 直接调用当前文案校验、归入首选尺寸规则、删 Submitted 门面；目录/URL/源权限及两个 hash 保留 |
| `internal/imageagent/generation_input.go` | 当前文案直接校验，删 literal 条件；原 product/AssetCatalog 一致性与输入指纹保持 |
| `internal/imageagent/image_set.go` | 当前 Text 必需、旧标签 fallback 与旧 quote 编码退出、PlanDigest 改接；当前 DTO/clone/schema/quote及两当前 hash 编码保持 |
| `internal/imageagent/image_set_service.go` | Prepare/GetPrepared/Confirm 原请求和回执快捷路径覆盖当前校验；整退旧 digest重算、旧构造器/局部 payload 与 prompt；当前 receipt 恢复顺序/source授权不变 |
| `internal/imageagent/service.go` | 仅规范校验调用改接；Get/Repository 原校验和身份顺序复用，不改命令/授权/事务 owner |
| `internal/imageagent/image_set_admission.go` | 仅在原 pending-copy 前拒绝空 slot.Status；原非空执行状态的定义复制、准确回执/digest/身份与时限比较保持 |
| `internal/imageagent/image_set_result.go` | 仅 Submitted 门面调用改接当前 canonical；partial/settled/UNKNOWN/结果 hash 保留 |
| `internal/imageagent/temporal/worker.go` | 仅两处 Submitted 门面调用改接；SDK client/Update/Start/队列和组织隔离原样 |
| `internal/imageagent/store/generation_output_recovery.go` | 仅删旧 v3 literal 条件，当前 Text 的不可变商品事实加载/验证保留；原锁/成功事实/settlement/恢复事务不改 |
| `web/listingkit-ui/src/lib/contracts/product-image-set.ts` | Run plan.Method 仅 v4、plan.Text 必需；原请求字段形状、严格 decoder/版本/来源/权限/closure/release断言不变 |
| `web/listingkit-ui/src/components/workbench/acquisition/product-image-set-panel.tsx` | 退 v1 提示/按钮条件、v3文案展示、旧无Text补选分支；当前确认文案/当前再生成审核及绑定版本/并发恢复/权限保持 |
| `internal/imageagent/image_detail_copy_test.go` | RETIRE 仅旧标签/prompt/历史容忍的实现测试；有用当前安全断言由原文案测试保留，见 §6.1 |
| `tests/legacy_imageagent_manual_runtime_retirement_test.go` | 追加本批完整缺席/当前保留 guard，原 #994/#996/#998/#1001 函数和断言零改 |
| `hack/debug/README.md` | 移除事故命令唯一活跃入口、说明源码退休；其它工具及说明不改 |
| 本文、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md` | 当前设计/索引/登记；README/register 的起点正文、历史数字与勾选完整保留 |
| `docs/architecture/official-skills-and-image-method-v1.md`、`image-skill-visual-brief-v2.md`、`image-detail-visible-copy-v3.md`、`image-text-language-confirmation-v1.md`、`bailian-image-no-generation-reconciliation-v1.md` | 源码阶段各追加准确旧承诺失效/当前保留注记，原正文/证据/勾选不改；本次设计阶段不改这五文件 |

实际已有入口/依赖核对：事故命令未列入 workflow/consumer inventory；上述生产包仍有当前消费者，未发现需删 CURRENT inventory 行。未来若真实发现清单外生产消费者、检查算法变化或 inventory 矛盾，报告具体证据与最小提案；不能用漏项继续默认保留旧链。

## 6. 实施顺序与必要验证

获准后同一批连续执行：先对旧 request/policy/nil Text/宽松计划写真实 RED，再将当前文案和严格规则归原 canonical owner，切断全部 read/confirm/dispatch/publication/recovery 快捷路径，最后删除三个生产文件与专属尾部。普通测试、合成 fixture 和成熟 SDK 足够，不建设新 runner/故障矩阵平台。

- 负例：三旧 policy、无 Method 的 nil Text、已有相同请求/原 admission 的旧计划、空 pending/超32/非安全revision/缺首图尺寸；必须在新增 admission、reserve、provider、启动或发布前拒绝，无正文/跨租户泄露，不伪造 no_generation/released。
- 正例：当前 v4 和无 Method；none/reviewed；有/无 Route；当前原请求/原回执恢复、撤权后安全关闭、同键冲突与来源谱系、Artwork/采集单主图、当前恢复/UNKNOWN。保留 nil/空值和当前 hash 输入；对冻结有效输入逐字节/独立 SHA 断言，不改 hash 算法或掩盖旧记录不再支持。
- 两个 #1001 SDK prefix 是任务自有合成数据。已只读确认 current prefix 的七 slot 均 pending、revision1、合法幂等键；**两 fixture 文件和原 SDK marker 时点/错误/回放断言不改**。不读取真实历史。
- 必要原测试：受影响的 ImageAgent、Skill adapter、worker/HTTP/Asset publication 与前端合同/API/面板；沿原测试按当前风险选择 race 和现有合成/任务自有资源测试。只编译原 current-application、image-agent-temporal-worker、1688-public-browser-collector、1688-batch-import，不启动。
- 当前三文档仅验证链接、范围/矩阵、历史后缀和 diff；不运行源码测试、数据库、worker或 canary。原 CI 留给稳定准确候选，按原分类如实区分 SUCCESS/SKIPPED。设计 CI 不等于源码、产品、运行或 rollout 验收。

### 6.1 原检查转换与 fixture 校准边界

| 原检查族 | 允许的准确转换 | 必须保留 |
| --- | --- | --- |
| `image_text_language_test.go` 的 `TestHistoricalPrepareWithoutTextReplaysFrozenBytes` | 旧成功重放改为拒绝；覆盖有/无 Method、旧策略、相同请求与无新调用/写入；测试旧载荷可用普通合成 JSON/独立原hash构造，不在生产保留兼容计算器 | 当前语言、copy owner、普通标点、原请求/原结果字节、同键冲突和明确 no-text 选择断言 |
| `model_test.go` 的四组历史宽松正向断言 | 空pending/超32/非安全revision/首图无尺寸改为 canonical 拒绝；保留原错误与边界值、合法32/11 slots、首图顺序、初始/单步替换与长幂等键规则 | 其它授权源、重复/非法角色、Unicode/字符长度、无 panic 与 quote/slot不变量 |
| `image_method_test.go`、`integration/skillimage/context_test.go` | 旧方法空材料/只读成功改负例；当前原Skill资源/fence测试使用v4，不退 Skill 数字版本 | 授权撤销、准确digest/ref、资源缺失/超界、release、无泄露/无额外provider |
| `image_detail_copy_test.go` | 退出旧生成/旧兼容语义；现有效语言/控制字符/事实/无文字安全断言映射到原 `image_text_language_test.go`/`image_method_test.go`，不能靠新策略降低要求 | 无伪造文案、不泄露内部字段、正确标点与事实验证 |
| ImageSet 当前 fixture（下列限定包） | 仅补当前显式 Text（无文案时 none）、pending、真实合成尺寸/当前4绑定；重算该合成输入的原 quote/hash；Submitted 调用改名；不得把正常 current source历史当legacy | 原成功/失败/权限/并发/幂等/事务/错误分类/receipt/success/UNKNOWN/cleanup断言；不能把应拒绝的用例“修成合法” |
| `image_set_test.go` 原无 ModeSnapshotRef JSON/hash测试 | 其共享 fixture 改为当前 Text；缺省字段/顺序/omitempty/独立SHA断言保持，不新增 ModeSnapshotRef 或修改生产序列化以迁就测试 | 当前hash算法、两个当前 quote编码、JSON/指纹；旧载荷被拒绝不是改写数据 |
| `store/generation_execution_test.go`、`generation_output_recovery_test.go`、worker/tools/Asset publication 方法测试 | 当前成功fixture从旧2/3改4并接当前Text；旧方法作为负例保留；成功观察、下载物化、再次尝试拒绝等当前断言完整保持 | 不凭旧方法/校验失败释放UNKNOWN、不重新POST、不丢成功与settlement证据 |
| `image_set_admission_test.go`（Temporal/store 的原同族测试） | 追加空状态不能经回执 pending-copy 恢复的负例；当前非空执行状态的定义复制与原回执恢复正例保留 | 原组织/member、CAS、撤权后安全收尾、原deadline和回执防重，不要求旧空状态继续可用 |
| 前端面板及 API/proxy 既有测试 | projection helper补当前Text/4；旧policy/nilText响应改为decoder拒绝/无可执行动作，旧v1/v3专属展示正向测试退出；当前历史来源/版本/late response测试保持 | scope/run/source身份绑定、撤权/只读/并发响应/草稿保留、人工文案与当前再生成审核、严格响应大小与HTTP错误 |
| 现有 root guard 与 code health | 只追加本批缺席/当前能力/不改workflow与inventory guard；原四函数、算法、ceilings、历史计数不改 | 不加 skip、忽略、缺文件成功或放宽禁令 |

允许同根测试 fixture/调用改接的**现有测试包边界**：`internal/imageagent` 及其 `store/temporal/tools/assetpublication`；`internal/integration/skillimage`；`internal/app/{httpapi,imageagent,pod,productsourcing,worker/imageagent}`。只允许上述语义变动，不包括改生产函数、增加外部调用或改变有效安全预期；不以同包其它功能为由顺手重构。直接已发现文件包括 `image_set_*`、`image_text_language_test.go`、`model_test.go`、`artwork_test.go`、`repository_contract_test.go`、`projection_test.go`、`source_context_test.go`、worker generation tests，以及它们原共享 fixture。实现自检确定的实际改动路径全部在 PR 列出；未受影响文件不改。

前端限 `product-image-set-panel.test.tsx`、`src/lib/api/product-image-set.test.ts`、`src/lib/server/product-image-set-proxy.test.ts` 的对应 fixture/调用改接。原安全断言和其它检查逻辑不改；新的清单外生产消费者或必须修改有效安全预期时先报告。此边界允许同根 fixture 收齐，不靠逐个函数审批机械拆批。

## 7. 交付、独立复核与停止点

1. 当前三文档和准确设计 HEAD 交 PR，原 CI 与适用独立设计结论分别记录；还需用户明确确认 §4.2 旧读/恢复承诺及 §6.1检查转换，才可 SOURCE IMPLEMENTATION_READY。
2. 独立检查看真实源码/消费者、19生产矩阵、旧快捷路径、hash/JSON/Artwork/资金及当前恢复保留，不重复重开无关架构。finding 先分类，正常设计复核最多两轮；真实 Must 缺陷处理，非阻塞增强不扩为平台或试用前置。
3. 已有适用 GitHub 复核运行时不重复触发；启动失败可以唯一重试，空 reviews/Completed 不冒充通过。**未经用户本轮授权不默认新增 AI 互验线程；设计和源码最终复核授权分开。**
4. 获准后同一 Writer/主要PR整批实现、必要TDD/原测试/编译/原CI；源码稳定后一次最终独立交付检查，修复仅复查真实增量。开发检查/AI复核不签发用户产品验收。
5. 当前无合并/部署/实例/数据/凭据/进程/真实历史操作或关单授权。未来合并须准确expected-head授权和正常门禁；合并后另确认实际merge SHA的push/main整run。
6. #301始终OPEN，编辑前新读；当前段最小更新。历史后缀从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保持，勾选不动，正文≤65536字符。源码设计/CI不把运行存在性UNKNOWN、真实验收NOT_RUN改成PASS。
