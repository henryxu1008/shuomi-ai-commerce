# #301 旧商品来源归一化与中间事实合同完整退休设计

> Design Basis: **Independent Architecture**。当前 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；仅获准先设计，适用独立设计复核及用户确认完成后才可记录 IMPLEMENTATION_READY。本文和两个索引的文档工作本身为 N/A / Ready，不改变生产行为。
> 起始 main：`1ec41a4712f7f6d9688a0bc371ad54f5a527abea`（#980 已合并、准确 push/main CI 已成功）；调查日期：2026-10-28。#301 保持 OPEN。

## 1. 用户结果、范围与权限

使用者是维护当前来源采集、商品保存读取、SHEIN 草稿及资产审批的开发者。本批结果是删除已无生产接线的旧来源归一化和中间合同，让维护者沿真实 owner 定位行为。它不新增业务入口，不声称当前用户操作被这些闲置文件阻断，也不增加内部试用前置。

四组同一 Delivery Batch、一个主要 PR：①旧 `catalog.Normalize` 及全部专属 helper；②canonical 中的旧来源置信度/审核判断；③商品与资产旧 Facts DTO；④旧 SourceRequest/SourceResult 及测试、注释、旧 KEEP 和直接文档尾部。不按文件或检查机械拆批。

当前唯一 writer 复用已附属独立 worktree `issue301-browser-tail-946/task-processor`，分支 `codex/issue301-source-facts-retirement-design`；已新读 #301、检查 open PR 和工作区，开工只 fetch 一次 main，无同范围 writer。当前仅修改本文、architecture/README、Legacy Register 三份 Markdown；未改生产源码、测试、配置或依赖。

Out of Scope：当前来源获取/批量/原浏览器与 debug、SourceEnvelope/SourceIdentity、ToSnapshot/保存读取刷新、Catalog/Review/Collection、当前 canonical 值合同与 SHEIN 草稿投影、资产候选/批准/recipe、AI/图像/点数/计费、PoolManager deadline/取消、SDS/POD、独立 SHEIN 登录与 Store 同步。workflow、完整 inventory、依赖、保护规则不改；不操作既有实例、数据、凭据、文件或进程，不读现有 DSN/token，不创建运行资源。源码实施、合并、部署和关单未授权。

## 2. Design Basis 与前批 KEEP 的准确替代

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[来源业务闭环](source-business-closure-v1.md)。适用 Figma/Product Authority 为 [final-ui-ia-authority](../product/final-ui-ia-authority.md)；本批没有 UI/IA/页面/交互变化，无 Figma 增量。

采用 Independent Architecture：涉及三个当前领域包的闲置公共合同退出，并明确替代前批 KEEP 和有效身份测试的入口。删除公共声明不能仅以“grep 零命中”代替合同/持久化核对。没有新增事实 owner、schema、状态机、权限、事务或外部副作用。

| 旧依据 / KEEP | 本批确认后才替代的准确含义 | 仍保留 |
| --- | --- | --- |
| #867 数字采集尾链：SourceRequest 剩余字段/variant/identity 保留；#868 来源值合同：SourceRequest 曾是 canonical identity 的实际 producer | 仅 SourceRequest、NormalizeSourceRequest、SourceRequest.Identity、VariantSourceRequest 及其专属测试退出；最后适配消费者已退出 | SourceIdentity 六个 Source* 字段、四 JSON 零值槽位、规范化/指纹/SourceKey、非零拒绝与持久化算法原样 |
| #901 Amazon 来源适配层、#921 任务身份合同的 SourceRequest 广泛 KEEP | 仅替代本次没有生产消费者的 sourcing/source_request.go 与 source_result.go；不恢复旧 adapter | 原 Amazon 浏览器/当前 Amazon capture、1688 公共浏览器/扩展/批量、当前 source owner 原样 |
| Phase3 Product 的 `catalog.Normalize(*canonical.Product)` 及旧 enrichment normalization 规划 | 旧入口及 17 helper、四项推断/评分函数退出；不搬到新包保活 | 当前 sourcing.Normalize/ToSnapshot、ProductSnapshot/Publisher/Repository、Review 与草稿读取合同 |
| 旧 1688 lineage/read-model 与 Commerce Tool foundation 中的 catalog.ProductFacts / asset.Facts | 仅这两组无接线的 Go DTO 和其 HasIdentity/HasAssets 退出 | 当前 SourceEnvelope/ProductCandidate/AssetCandidate、原 SRC receipt/lineage、Catalog/资产 owner、现役 Tool/前端 DTO |

此前 KEEP 不是错误回写；只有用户确认本文并完成准入后，上表才替代准确对象的保留含义。历史正文、批准、计数、勾选和失败证据保留。`source-business-closure-v1.md` 中前端 ProductFacts 等近名表述不等于这里的 Go 类型，不作全局文本清理。

Legacy decision: **RETIRE**。
Reusable behavior: 当前有效来源规范化、确定性转换、身份拒绝、保存/读取/hash 和资产批准已有现役 owner；原样复用，不新增 EXTRACT 代码或替代归一化框架。
Current owner: `product/sourcing` 的当前 envelope/publication；`product/catalog` 的 snapshot/publisher/repository；`product/asset` 的候选/批准与当前 SHEIN draft 投影。
Cutover/deletion condition: 用户确认设计、适用设计复核完成、实施前消费者核对仍成立、§6.1 原有效断言与必要 CI 通过；源码退出不授权数据/资源删除。

## 3. 实际消费者与完整调用闭包

基线全部 3122 个 tracked Go 文件已用标准 Go AST 解码 import、alias 和 selector，覆盖 `_test.go`、debug、testdata 与构建条件文件；再核对包内函数引用、方法 receiver、构造/接口/注册和非 Go 文档。分析仅解析源码，未运行仓库应用或访问既有环境。不是仅按包 import 数判断成员是否存活。

| 对象 | 当前实际引用 | 结论 |
| --- | --- | --- |
| catalog.Normalize | 外部 Go 消费者为零；包内只有 normalize_test.go 三个测试调用。18 个函数全部在 normalize.go 内闭合，无其它当前同包调用 | 整文件及专属测试 RETIRE |
| canonical.HasSourceType / BuildFieldTrace / ProductNeedsReview / ShouldReviewLLMOnlySourceFact | 无外部生产或测试 selector；HasSourceType 仅被同文件其余旧 helper 调用；没有接口/注册/装配入口 | 四函数及唯一 strings import RETIRE；13 个值类型与六个 Source* 常量逐字保留 |
| catalog.ProductFacts / VariantFacts / FactWarning / ProductFacts.HasIdentity | 无生产或测试消费者；仅本文件结构相互引用 | 整 facts.go RETIRE |
| asset.Facts / ItemFacts / FactWarning / Facts.HasAssets | 无生产消费者；仅 facts_test.go 的 TestFactsReportsWhetherSourceAssetsExist | 两文件整退，不误删 SourceWarning 或其它包的同名 FactWarning |
| SourceRequest / NormalizeSourceRequest / Identity / VariantSourceRequest | 无生产消费者；仅 source_request_test.go 三测试、source_identity_test.go 的一个测试和一个旧 wrapper 比较尾部 | 旧输入整退；身份有效断言按 §6.1 改接当前 NormalizeSourceIdentity |
| SourceResult | 除声明外没有任何 Go 消费者，无 schema/codec/接口装配 | source_result.go 整退 |

真实保留路径：

```text
当前 acquisition / 公共浏览器与扩展 / 自有来源 / POD / SupplyMarket
  -> sourcing.SourceEnvelope + 当前显式身份/授权
  -> sourcing.NormalizePublicationEnvelope / Normalize / ToSnapshot
  -> SRC 与 Catalog 原事务、回执及版本化 ProductSnapshot
  -> Collection / Review / 当前 Agent、Tool 与 Listing 消费者

已保存 ProductSnapshot -> catalog.ProjectCanonical
  -> marketplace/shein/draft.Builder -> 现役严格草稿与平台纯规则
```

`ToSnapshot` 的真实生产调用包括 `integration/persistence/product/sourcing/repository.go`、`product/pod/template.go`、`product/supplymarket/publication.go`。`ProjectCanonical` 由 `marketplace/shein/draft/builder.go` 直接使用；canonical 的 Product/Image/Attribute/Dimensions/Weight 等仍由 draft/offlinepackage 使用，所以不整退该包、不换 owner、不改字段。

当前没有任何生产实例创建这四组旧对象；无“把旧分支接入当前启动”的切换步骤。静态消费者不存在并不证明物理数据库/文件不存在；本批不探测也不处置它们。

## 4. Owner、JSON/hash 与副作用边界

### 4.1 唯一事实 owner 与当前安全不变量

当前来源事实由 SourceEnvelope、SRC receipt/evidence 与 Catalog 已保存快照持有，事务/锁、幂等键、ExpectedBaseVersion、读取刷新/历史版本和组织隔离留原 owner。旧 Facts/SourceRequest/SourceResult 没有独立 repository、schema/迁移、codec、消息注册或持久化写入，退休不新增替代事实源，不做补录/迁移/清理。

asset.Facts 的“Items 非空”不是当前批准或 readiness 门禁；保留 AssetCandidate、显式 Approval、ImageSet/recipe 和现役无发送/UNKNOWN/recovery 所有者。canonical 的固定置信度 0.6/0.9 与字符串空值推断没有当前接线，不替换为新评分器，不改变当前 Review/风险/批准规则，也不宣称当前结果已可发布。

权限/租户、当前身份与计费/预算、provider 一次发送、超时/取消、UNKNOWN、恢复、资金、物理资源和 UI 均不变。复用当前 Must/Threat Model/Accepted Risks，不新增通用治理、审核或人工流程作为交付前置。

### 4.2 编码、hash、四零值槽位与值合同

1. `SourceIdentity` 的六个 Source* 字段，以及 Platform、Region、ProductID、StoreID 四个零值槽位的原字段/tag/类型/布局全部零改。原规范化、指纹算法、SourceKey、输入限额、四槽位非零拒绝、损坏记录拒绝和 publication identity 均不变；删除 SourceRequest 不赋予旧字段新的身份含义。
2. `SourceEnvelope`、ProductCandidate/Variant/AssetCandidate、SourceWarning/Trace、ProductSnapshot、PublishedSnapshot 和 canonical 的全部 13 类型、六个常量原声明保持。原 JSON/tag、nil/empty、map/数组排序、decimal/string/number 表达、错误 identity、版本和已保存载荷不改。
3. 保留 `sourcing.Normalize`/NormalizePublicationEnvelope/ToSnapshot、catalog.ProjectCanonical 全函数体及真实 consumer；保留原 EnvelopeJSON/SnapshotJSON/InputHash/EnvelopeHash/SnapshotHash 与发布身份。只有无生产消费者的 catalog.Normalize 退出，不将它误认为 sourcing.Normalize，也不重写转换规则。
4. 两组旧 Facts 和 SourceRequest/SourceResult 无当前 HTTP/JSON/队列协议；静态审查未发现具体、当前、外部可观察兼容义务。若实施出现相反的实际证据，停下报告用户；不得加 alias、wrapper、转发、fallback、双读/双写或第二 owner。
5. 身份测试改接当前 owner 只改变旧 wrapper 的输入构造，不能改变现有四槽位、key、拒绝、指纹、保存读取或 hash 期望。实施用原有合成 fixture/测试核对不变源码及编码/hash；不读真实记录、不建 runner/黄金数据平台、不借本批扩大验收矩阵。

## 5. 删除 / 保留矩阵与完整路径

### 5.1 四组源码精确处置

| 文件/声明 | RETIRE / 调整 | KEEP |
| --- | --- | --- |
| catalog/normalize.go | Normalize、buildAttributes、buildVariants、buildImages、buildPrice、buildSpecifications、buildDimensions、buildWeight、buildPackage、buildTrace、buildReviewState、collectReviewReasons、collectSources、uniqueSourceRecords、sortedAttributeKeys、uniqueStrings、cloneStrings、firstNonEmpty 共 18 函数 | sourcing 的当前 Normalize/ToSnapshot、catalog 的 Snapshot/投影/仓库函数不动；不跨包禁用通用同名函数 |
| catalog/canonical/types.go | 仅 HasSourceType、BuildFieldTrace、ProductNeedsReview、ShouldReviewLLMOnlySourceFact 四函数及失去用途的 strings import | 所有 13 type 和六个常量原样；FieldTrace 的持久化值不因旧评分器退出而改写 |
| catalog/facts.go | ProductFacts、VariantFacts、FactWarning、ProductFacts.HasIdentity | ProductSnapshot/SourceRecord/Warning、前端当前 ProductFacts 不动 |
| asset/facts.go | Facts、ItemFacts、FactWarning、Facts.HasAssets | AssetCandidate/Asset/Approval/recipe/Repository 及共享警告合同不动 |
| sourcing/source_request.go | SourceRequest、NormalizeSourceRequest、SourceRequest.Identity、VariantSourceRequest | 当前 SourceIdentity/SourceEnvelope 与生产 producer 不动 |
| sourcing/source_result.go | SourceResult | 当前采集 Result/回执/错误、SourceEnvelope 及来源证据不动 |
| canonical/doc.go | 仅注释改为当前 Catalog 投影/草稿共享值 owner，去旧 enrichment/ListingKit runtime 说明；不加入运行逻辑 | package 名称/路径不变 |
| sourcing/source_identity_test.go | 仅 §6.1 两个函数的准确旧入口转换 | 原有效断言和其它函数逐字保持 |

### 5.2 未来同一 PR 的 26 路径完整 manifest

**整退源码/专属测试（8）：**

- `internal/product/catalog/normalize.go`
- `internal/product/catalog/normalize_test.go`
- `internal/product/catalog/facts.go`
- `internal/product/asset/facts.go`
- `internal/product/asset/facts_test.go`
- `internal/product/sourcing/source_request.go`
- `internal/product/sourcing/source_request_test.go`
- `internal/product/sourcing/source_result.go`

**混合源码/测试/包注释（3）：**

- `internal/product/catalog/canonical/types.go`
- `internal/product/catalog/canonical/doc.go`
- `internal/product/sourcing/source_identity_test.go`

**原 guard 追加一函数并转换一项旧 KEEP（1）：**

- `tests/listingkit_backend_retirement_test.go`

**当前设计/索引/边界文档（4）：**

- `docs/architecture/legacy-source-normalization-facts-closure-retirement-boundary.md`
- `docs/architecture/README.md`
- `docs/refactoring/legacy-register.md`
- `docs/architecture/project-boundaries.md`

**历史文档只加准确替代顶注（10；起始 main 完整原 blob 后缀逐字保留）：**

- `docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md`
- `docs/architecture/legacy-numeric-value-contract-closure-retirement-boundary.md`
- `docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md`
- `docs/architecture/legacy-task-identity-mapping-closure-retirement-boundary.md`
- `docs/superpowers/specs/2026-08-06-1688-source-lineage-design.md`
- `docs/superpowers/plans/2026-08-06-1688-source-lineage.md`
- `docs/superpowers/specs/2026-08-07-source-lineage-read-model-design.md`
- `docs/superpowers/specs/2026-08-31-commerce-tool-foundation-design.md`
- `docs/superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md`
- `docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md`

当前只改设计三文档，其余 23 路径未改；26 是未来完整边界，不是当前已退休声明。两个索引和 project-boundaries 也只追加当前准确顶注，保留起始 main 原文；本文冻结后未来实施状态只加顶注，不回写冻结设计。

### 5.3 零改保留面

除上表外的全部生产源码、测试和检查保持。特别是 SourceIdentity/SourceEnvelope/normalize/publication、catalog_asset_handoff、Catalog snapshot/projection/publisher/repository、当前 Review/Collection/Asset 的有效规则和安全断言；canonical 所有值声明/JSON；当前采集/原浏览器/debug/PoolManager deadline取消、SDS/POD、独立 SHEIN 登录/Store 同步源码零改。

完整 inventory（含 CURRENT、历史、计数、ceilings、检查）与所有 workflow、lint、依赖/lock、保护规则零改。本批目标没有 logger/OpenAI 边，不需要 CURRENT 校准。不做 go mod tidy，不把其它零 import 的独立 SDK/PoC 或当前准备能力顺带退休；product-sourcing-inventory 与 phase2-runtime-inventory 完整零改。

## 6. 实施顺序与验证

设计及适用复核完成、用户确认后，在同一主要 PR：①重新核对实际 main/消费者并冻结原保留声明/函数/断言；②在原聚合测试追加本批 guard，并按 §6.1 将一项旧必读要求转换为明确源缺席，旧对象仍在时取得真实 RED；③按矩阵整退四组、准确改接身份测试、收齐包注释和文档尾部；④guard GREEN，必要原测试/入口仅编译，稳定准确 HEAD 的原 CI 和一次源码最终复核。不建新验证工具/runner/运行宿主，不为阶段证据拆 PR。

### 6.1 原检查与断言的精确处置

| 原文件 / 函数 | 本批允许转换 | 必须保留的有效规则 |
| --- | --- | --- |
| catalog/normalize_test.go 三 TestNormalize* 及 attributeNames | 随仅供旧 Normalize 使用的旧实现测试整退 | 它们约束旧输入/固定 Review 原因；当前 ToSnapshot/Publication/Canonical fixture 的确定性、字段保真和 hash 原测试零改，不将旧人工评分搬入当前路径 |
| asset/facts_test.go / TestFactsReportsWhetherSourceAssetsExist | 随旧 Facts/HasAssets 退出 | Items 非空不是现役批准；当前资产候选/批准/Repository/SHEIN/图像有效测试零改 |
| sourcing/source_request_test.go 三测试 | 随旧请求正常化/继承/Identity wrapper 整退 | 其中有当前意义的身份正常化/key/四槽位断言已由下两行当前 owner 覆盖；不改四槽位非零拒绝与持久化测试 |
| source_identity_test.go / TestSourceIdentityDoesNotDeriveOrBackfillRetiredAliases | 仅移除末尾 `request := (SourceRequest{...}).Identity()` 及 `if request != id` 比较块；这是旧 wrapper 与当前身份等价的旧实现断言 | 前面完整 table/循环、缺值不得互填、Normalize(SourceEnvelope) 拒绝和当前 id.SourceKey/四槽位检查逐字保留；不得删除该函数或缩减任何 table entry |
| source_identity_test.go / TestSourceRequestIdentityUsesCanonicalKey | 函数改名为 TestCanonicalSourceIdentityUsesCanonicalKey；只将 id 输入构造换为 `NormalizeSourceIdentity(SourceIdentity{SourceType: " CRAWLER ", SourcePlatform: " Amazon ", SourceID: " B001 "})` | 原函数从第一个 SourceType 断言起至末尾的所有断言/错误字面量逐字保持：SourceType、SourcePlatform、SourceID、四零值槽位、SourceKey 全保留；其它测试函数零改 |
| 原聚合 guard / TestNumericAcquisitionAndPlatformTailClosureIsRetired | 仅从其 mandatory os.ReadFile 表删除 `internal/product/sourcing/source_request.go: {"TenantID"}` 一项，改为同函数明确 `os.Lstat` + `os.IsNotExist` 检查该完整源文件必须缺席；不能在读取失败时忽略或 skip | 原 TenantID 禁入由整源缺席作更强替代；type_amazon.go/type_platform.go 两项必读表与 token、所有其它路径禁入、trackedFiles/decoded import/parser、错误处理及原逻辑逐字保持 |
| tests/listingkit_backend_retirement_test.go 新 guard | 仅追加 TestLegacySourceNormalizationFactsClosureIsRetired，复用原 trackedFiles/AST/decl/import/error helper；除上行准确旧 KEEP 转换外，不改其它原函数/表项/期望/算法 | 精确八路径缺席；各 owner 内旧声明/alias/转发禁入；canonical 13 值/六常量、SourceIdentity 十字段与当前 ToSnapshot/ProjectCanonical 必须存在；保留原解码 import/缺失及解析错误失败，不加 skip，不全局 ban Facts/Normalize/SourceRequest 同名 |

新增 guard 对 canonical 原 type/常量形状作准确正向约束；对 SourceIdentity 四槽位及当前合同检查只读原文件，不更改它们。新 guard 保护本批删除边界；除上表一项旧必读转换，原永久 guard 全部保留，尤其 #868 身份拒绝、#964 capturedata、#968/#972/#977/#980 当前合同，Phase3/import/Architecture/Admission/depguard/inventory 逻辑均不变。

本项来自独立设计复核的唯一 finding，分类 IMPLEMENTATION_TEST：整退 source_request.go 却保留其必读要求，会使原检查必然失败；准确转换保留并加强原 TenantID 禁入含义，不改变产品/安全边界，不是新全局架构 BLOCKER。已逐项核对八个退休路径在 tracked tests/check 脚本中的强制读取，仅此一项；生产/测试尚未修改，当前补齐的是设计矩阵。

未来必要自检：上述 guard；当前 catalog/sourcing/asset 包原测试及 SHEIN draft/offlinepackage/validator 原回归；原架构/准入/import 检查；current-application、image-agent-temporal-worker、shein-login-worker、1688-public-browser-collector 及现役 Amazon/1688 debug 六入口仅编译。debug 按已有嵌套模块/单文件入口核对，不改 go.mod，不启动。原 PR CI 绑定准确 HEAD 完整 run；部署、真实 provider/浏览器/运行产品验收保持 NOT_RUN。

实施若暴露表外旧 KEEP/同根检查，先查 sibling paths，提出准确最小例外，再取得用户决定；不能 missing-skip、删有效断言、放宽错误/AST/安全检查、改 ceilings/workflow 或默认重跑。出现当前 Must 的真实 BLOCKER 按项目分类上报，非 Blocker 不重开全局架构。

## 7. 设计交付、独立复核与停止点

当前交付仅三个文档和 Issue/PR 准确状态。独立设计复核应检查 §1–§7、26 路径完整闭包、真实包内/外消费者、前批 KEEP 替代、JSON/hash/四零值槽位、13 值合同与 §6.1；不跑应用/运行资源，不新增验收平台。原 CI/作者检查不等于独立设计结论，也不等于用户验收。

按 finding 分类与最多两轮正常设计评审收敛；只有当前 Must 的真实 BLOCKER 重开设计。未经用户本轮授权不默认新增 AI 互验线程；若 GitHub 自动复核启动失败，记录实际状态，不当 PASS、不反复触发。任何替代独立子代理复核须本轮明确授权；设计和源码最终复核授权分开。

适用独立设计复核通过后，交用户确认本文及 §6.1，再记录 USER_CONFIRMED / IMPLEMENTATION_READY 并允许 Writer 改正式源码。源码完成后一次独立最终交付检查，具体 finding 修复只复核增量。正常合并另获准确 HEAD 授权，合并后只认实际 merge SHA 的 push/main 整 run 终态。

#301 始终 OPEN；不自行关闭、部署或接管当前业务。不操作既有实例、数据、凭据、文件或进程，不增加内部试用前置。独立 SHEIN 登录与 Store 同步完整保留。
