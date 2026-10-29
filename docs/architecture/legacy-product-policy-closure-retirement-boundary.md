> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

> #301 / #964 当前替代边界（[用户确认设计 §1–§7/§6.1](legacy-source-result-contract-closure-retirement-boundary.md)）：仅替代 internal/model.Product 路径不动的 KEEP；同一 Product值与JSON归Amazon域内result纯叶子，旧全局根退出。旧Product策略机制保持退休，不新建共享或兼容层。 当前能力、独立SHEIN登录/Store同步及持久化事实owner保持，workflow/完整inventory/依赖及既有资源不动，无试用前置，#301 OPEN。以下原完整Git正文/数字/勾选/证据逐字保留。

> #301 / #961 精确替代顶注：按[legacy-publishing-support-policy-closure-retirement-boundary.md §1–§7（含§4.2/§6.1）](legacy-publishing-support-policy-closure-retirement-boundary.md)，旧 publishing/common 与 jsonx 根、十一闲置 SHEIN 策略族及三个混合策略尾部已在源码完整退出。当前八值合同分归已有 offlinepackage / publishing.Price，本地组装纯行为归 draft；本批表列 KEEP 由该设计矩阵精确替代，其它当前能力/JSON/hash算法/版本/事实owner及独立SHEIN登录与Store同步保持。仅未来新多属性数组固定按原始名排序，不重写已有载荷。以下原文、数字、勾选与历史证据逐字保留，不作为恢复旧owner或运行操作的授权。

> #301 / #926 **SUPERSEDED，仅本批旧平台调用闭包**：用户已确认[完整退休设计 §1–§7/§6.1](legacy-platform-api-closure-retirement-boundary.md)。本文涉及旧 Amazon/TEMU API、SHEIN 闲置 Client/manager/DTO、SPAPI/旧平台包装配置的 KEEP/当前 owner 建议，只在该准确删除/保留矩阵内被替代；不移植旧 auth refresh/retry 或平台副作用。当前 SHEIN DTO/完整 JSON/有效规则、SessionTransport/三个读取端点、采集/原 Amazon 浏览器/debug、官方 Store、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留。只抽原仓库两纯函数，不加 compat；workflow/完整 inventory/计数/检查逻辑及既有实例/数据/凭据/进程不动，不增加试用前置。以下原完整正文、历史数字/勾选/批准/证据逐字保留。

> 当前适用边界（#301 / #912）：用户已确认并按[旧 SHEIN 闲置策略与辅助合同完整退休闭包](legacy-shein-policy-contract-closure-retirement-boundary.md) §1–§7（含 §6.1）整批实施，仅替代本文件中旧 SHEIN pricing.PriceManager/计算器与本批辅助合同的广泛 KEEP；当前平台 pricing_repo.go 和 api/product 同名 PriceManager 不在退休范围及对应旧 KEEP 检查；准确删除/保留矩阵以新设计 §2/§5/§6.1 为准，其它当前 API/有效规则/采集/独立 SHEIN 登录与 Store 同步不变。下方原历史正文、数字和证据逐字保留，不提供旧 JSON 迁移或运行/数据操作授权。

# #301 旧商品筛选与价格/库存推断策略完整退休闭包设计

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED** in #909。用户已确认冻结设计 §1–§7（含 §6.1）；正式修改前 #301/主要 PR 已记录并读回 **Design Basis: Reuse Existing Architecture / Ready**。六个旧 Go 工件、原四项 KEEP→整根缺席/decoded import/限定 owner 声明保护、十二旧依据窄顶注及唯一失效链接已整批切换；不新增替代策略。原 guard 已实测 RED→GREEN，准确实现 HEAD/必要自检/原 CI/最终复核证据在主要 PR 维护，不以设计检查冒充实现或运行验收。以下冻结设计矩阵保留为实施依据；合并、部署、既有资源操作和关单未授权，#301 保持 OPEN，#29 不重开。

调查基线 main **`7a220abb22903294b834e8a6c726e07628fac980`**，包含已完成的 #904。[前批 push/main CI39924638080](https://github.com/qq550723504/shuomi-ai-commerce/actions/runs/39924638080) completed/SUCCESS 只证明前批组合，不代替本轮 CI/设计检查或未来实现验证。接单及前批当前段存档见 [#301 / 6362114181](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6362114181)。唯一 Writer 复用原独占 worktree，新分支 `codex/issue301-product-policy-closure-retirement-design`；不接管当前采集 #903/#907/#908 或 Store schema #692。

## 1. 用户结果、完整范围与权限

使用者是维护当前采集、平台规则与内部试用的维护者。交付一个完整源码退休边界：旧 `internal/marketplace/productpolicy` 筛选器、值对象、价格/库存/配送推断、专属测试及保留检查/说明尾部一次收齐。当前没有被本包阻塞的用户操作；这项维护不成为内部试用的新前置，不增加产品功能或新验收工具。

根因是历史消费者已退出后，旧 KEEP 和存在性检查仍把无注入的规则库当作现役能力。#754 保留它时存在 SHEIN validation 消费，#822 继续整体 KEEP，#824 删除旧规则管理/转换接线，#891 仍按当时范围保留纯规则；这些历史边界不构成永远保存一个无当前需求的独立策略 owner 的理由。不能只因“零引用”删除有效能力，也不能只因“纯函数”或测试还在而推定当前 requirement。本设计同时核对当前准入依据、生产调用/注入和原回归，未找到要求继续承接本包旧筛选策略的当前产品操作。

旧策略包括：价格 original→list/initial/final 选择，运费恒 0，stock_min 超过 30 则按 30 比较，可售但无数量推定 31，整链固定用 24 小时检查配送，以及以 Amazon 字样判断配送/自营。它们在基线没有生产消费者，不能据此声称当前业务正在错误计算；本批 RETIRE 这套闲置假定，不修补算法、不把它们移进当前采集或官方平台规则。

设计阶段原授权仅覆盖只读调查和三文档/Issue/PR；本轮用户已确认本文冻结 §1–§7（含 §6.1），授权同一 PR 整批实施六 Go 工件、原 guard 和准确文档尾部，必要自检/提交/推送。配置/schema/依赖/锁/workflow/inventory/保护零改；合并、部署、关单仍分别待授权；不读或操作既有实例、数据、凭据、私密配置、Profile 或进程，不运行应用/浏览器/provider/数据库入口。扩大 Cookie 接口退休提议已撤回，本批不恢复该提议。

## 2. Design Basis、Authority 与不变量

设计阶段 **Design Basis: N/A**：仅文档与静态消费者/元数据检查，不改变正式业务行为。实施拟 **Reuse Existing Architecture**：删除无当前消费者的旧完整策略包，不新增替代 owner、公开领域合同、表/持久化事实、配置、权限/租户、状态机、事务、幂等、retry/UNKNOWN、计费或平台副作用边界。范围不足 30 文件；即使命中数量提示，也不得通过拆批规避实际风险。发现需要改变上述边界时停止相关正式实施，升级 Independent Architecture，不能在实现 PR 临时发明架构。

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)：只允许 EXTRACT/RETIRE，不承接旧数据/ID/服务，不为保活建立兼容、fallback 或第二事实源。
- [项目边界](project-boundaries.md)、[数据服务 v1](data-services-v1.md)、[来源业务闭环](source-business-closure-v1.md)：当前 Product/Sourcing/Collection、平台明确事实/规则和采集 owner 保持；不因旧 model 被依赖而删除该共享 model。
- [#754 筛选包装退休](legacy-temu-filter-mapping-retirement-boundary.md)、[#822 Amazon 刊登退休](legacy-amazon-listing-closure-retirement-boundary.md)、[#824 数字管理尾部](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#891 预览尾部](legacy-listing-preview-download-closure-retirement-boundary.md)：只在 §5/§6.1 明列的 productpolicy 包和其自身检查/文档含义上提议替代旧 KEEP；历史证据和其它保留对象不变。
- [独立 SHEIN 持久化边界](shein-worker-numeric-persistence-retirement-v1.md)：独立登录、Store 同步/归属、fence/UNKNOWN 不变，不增加真实验收动作或试用前置。

Must：整包与专属尾部同批；无新规则库或旧策略接线；保留当前采集、原浏览器/debug、平台 API/有效规则、共享 model、PriceManager/PriceCalculator、独立 SHEIN 登录/Store 同步；workflow/inventory/依赖/保护零改。Should：复用原 tracked-source/AST/guard 和 CI，不开发 checker/runner/验收平台。Out of Scope：其它无默认装配的当前能力、Amazon/1688 爬虫、API 客户端、平台规则/DTO、价格服务/成本公式、当前身份/Store/映射/草稿与运行验收。Accepted Risks：不新增；既有物理状态 UNKNOWN，真实运行/用户验收 NOT_RUN，生产发布 NOT_AUTHORIZED。UI/IA/Figma N/A，没有用户入口或展示合同变化。

Threat Model 仅覆盖误删当前采集/规则/共享值、遗漏旧策略尾部或依赖、把源码删除当物理清理、以新 mapper/启发式规则重建旧 owner。不重开无关平台/身份安全架构，不把未开放功能升级为本次试用门禁。

## 3. 实际消费者：contract → implementation → injection → consumer

| 对象 | 基线源码/装配证据 | 本批决定 |
| --- | --- | --- |
| `marketplace/productpolicy.FilterRule` | 七筛选字段，仅本包 RuleChecker 参数使用；#824 的 listingruntime 转换接线已退出；全追踪 Go 检索无包外 import | 随整包 RETIRE；不恢复数字 FilterRule API/仓库，不新建 DTO |
| `RuleChecker` 与构造、七检查方法 | `CheckAllRules` 只串包内价格/库存/评分/评论/时效/配送；包外没有构造/调用/注入 | 整退该链，不把其中委派拆成后续小批 |
| `GetProductPrice/GetFreight/GetInventory/extractStockFromAvailability` | 只被上述串联及本包专属测试消费；依赖共享 `internal/model.Product` | 退本包算法/默认值；共享 model 及原浏览器/平台独立规则不动 |
| `IsFBAFulfillment/IsAMZSeller` | 只被本包 `CheckFulfillmentType` 使用 | 同批退；不全仓禁止 FBA/AMZ 或同名平台字段 |
| `price_helper_test.go` | 只验证本包 nil→0/不写进程日志和 original 价格回退；测试启动自己的 child test，不是另一个生产消费者 | 随整包退专属旧行为测试；不把旧回退复制到当前规则 |
| 原 `TestListingKitTaskBackendSourcesAreRetired` | 四个 KEEP 文件项强制 RuleChecker/FilterRule/价格库存/配送声明存在；无其它包外正向 Go 消费者 | §6.1 原设施将这四项从 KEEP 转为整根缺席/禁入保护，不移除整项测试或其它 KEEP |
| 原 `TestSharedPackagesDoNotImportAppDomainPlatformOrIntegration` | forbidden 中包含 productpolicy；这是禁止依赖，非要求目录存在 | 原样 KEEP，不能因为源删了就删除负向架构约束 |
| 六当前入口 | `go list -mod=readonly -deps` 分别覆盖 current-application、shein-login-worker、image-agent-temporal-worker、1688-local-agent、1688-batch-import、1688-public-browser-collector；输出不含本包 | 保持这些入口/实际注入；元数据不是运行验收 |
| 其它 `FilterRule`/`GetInventory` 字样 | 原退休负例/历史文档、平台 inventory API、保留 MappingInfo 字段等，不 import 本包；ImageAgent 测试同名方法非本包调用 | KEEP，不做全仓同名字串禁令 |

全仓 Git 追踪 Go 的 import/符号检索包含测试、嵌套模块与 build-tag 文件；六入口元数据为本机 Windows/default tags 的补充证据，不冒充所有平台组合。未来 guard 须沿原解析设施检查 decoded import path，而非仅 grep 字串。设计阶段没有执行本包构造器、业务方法或原策略测试以“证明旧能力可用”。

## 4. 持久化、配置与副作用 owner

| 边界 | 本包实际责任 | 变化/保留 |
| --- | --- | --- |
| 规则输入/输出 | 指针阈值和共享 Product 纯内存读取，返回 error/数值；无 schema、DB、队列、缓存、文件或配置 owner | 仅删除旧策略源码；不删除/改写任何规则、商品、Store 行或列 |
| 价格/库存/时效/配送事实 | 本包无持久化事实权；原启发式推定不应成为新系统事实 | 不迁移/重算/补齐或调用替代策略，不变更当前官方/采集 parser |
| 权限/租户/业务事务 | 不拥有这些边界，也未被当前应用注入 | 原 Organization/授权/UoW/幂等及计费 owner 不动，无新 Unit of Work/outbox/Saga |
| provider/platform/浏览器 | 无 client、HTTP、Mutation/receipt 或浏览器状态 owner | 不产生 provider 请求、登录/授权撤销、重试或平台补偿；API及原浏览器原样 |
| 独立 SHEIN 登录与 Store | 原 worker/Store 持久化和归属控制 owner | 登录、Store 同步、响应丢失/撤权/取消/并发/restart/UNKNOWN/fence 均按原 owner，源码零改，不自动接管 |

无业务状态转换或新增持久化步骤；删除包不触发恢复/停机/清理。既有数据/物理对象是否存在为 UNKNOWN，不能从静态代码推导，也不需要读取它们来完成本批设计。若发现具体当前外部可观察依赖或需承接旧状态，先报告用户，不能默认为“临时兼容”。

## 5. 完整删除、保留与抽取矩阵

### 5.1 确认后整批 RETIRE 的六个准确 Go 工件

| 文件 | 完整边界 |
| --- | --- |
| `internal/marketplace/productpolicy/doc.go` | 旧包 owner 声明 |
| `internal/marketplace/productpolicy/filter_rule.go` | FilterRule 七字段整类型 |
| `internal/marketplace/productpolicy/rule_checker.go` | RuleChecker/NewRuleChecker、CheckPriceRange/CheckInventory/CheckRating/CheckReviewCount/CheckDeliveryTime/CheckFulfillmentType/CheckAllRules |
| `internal/marketplace/productpolicy/price_helper.go` | GetProductPrice/GetFreight/GetInventory/extractStockFromAvailability 及全部旧多语言正则/价格回退 |
| `internal/marketplace/productpolicy/fulfillment.go` | IsFBAFulfillment/IsAMZSeller |
| `internal/marketplace/productpolicy/price_helper_test.go` | 仅上述包自己的 nil/日志与 original 价格测试/child fixture |

5 生产（含 doc.go）+1 专属测试，整根消失，不留 stub/README 充当第二规则 owner。没有专属 schema、migration、YAML/环境键、Prompt、API DTO、CLI、部署或 workflow 可以删；不为扩大文件数追删其它能力。

### 5.2 KEEP 和准确替代旧 KEEP

| 对象 | 本轮边界 |
| --- | --- |
| #754 的本包纯规则/当时 SHEIN caller KEEP，#822 §5.3 本包 KEEP、#891 的有效纯规则排除范围 | 用户确认后，仅对本包六工件及 §6.1 的检查/说明替代为 RETIRE；旧当时的消费者和旧测试证据仍是历史，不重写 |
| #824 的旧管理/转换 import 退出 | 继续保留退出结论；不恢复其类型/接线，也不再编辑该历史设计正文 |
| 当前 Marketplace SHEIN/TEMU goods/rules/workspace、草稿/严格 codec/诊断、PriceHandler、PriceManager/PriceCalculator 与 config 成本纯公式 | 原 owner/版本/输入输出/有效测试全 KEEP；不以本包名称相近推定可删 |
| 原 Amazon/1688 浏览器/提取器/debug、当前 acquisition/data services、localagent/batch/public collector、Sourcing/Collection/Asset | 源码/入口/配置/行为 KEEP，不转接新策略、不实际运行 |
| 共享 `internal/model` Product、listingruntime MappingInfo、两种 SHEIN 属性、Store/Organization 和独立登录/同步 | KEEP；不瘦身字段、不改 JSON/hash/持久化、UNKNOWN 或归属语义 |
| 平台 API/DTO、通用身份/AI/prompt/图像/基础层 | KEEP；不恢复已撤回的 Cookie API 退休范围 |
| workflow/inventory/历史记录/计数/ceilings/解析器/其它检查/依赖/锁/保护 | 全部零改；本包未命中 CURRENT consumer 行，无行需要删除/增加 |

Legacy decision: **RETIRE**。Reusable behavior: 无从该闲置包新抽取的当前需求；当前有效规则继续由原 owner 使用，不新建 mapper/规则库。Current owner: Product/Sourcing/Marketplace 现行平台规则、共享 model、Store 和 SHEIN worker 各不变。Cutover/deletion condition: 设计检查完成且用户确认 → Issue/PR Reuse Existing Architecture/Ready 读回 → 原 guard RED → 六工件/原 guard/文档尾部同批切换 GREEN → 原 CI/准确实现最终检查。不是生产/数据清理授权。

## 6. 必要验证与 §6.1 精确处置

设计阶段仅三文档/链接/索引/消费者/元数据检查和原 CI；不为纯文档制造 RED。实施阶段不修改保留业务逻辑；TDD 使用原 guard 先加整根缺席保护，使六工件仍在时 RED，再在同批删除而 GREEN。无业务算法要新增实现或复制测试；专属旧回退测试随 owner 删除，保留所有其它当前规则测试。

适用开发自检限原 retirement guard、原跨包 import/当前入口依赖保护、受影响文档检查和必要当前入口编译；完整必需 CI 留稳定候选执行，不人工重复全量。CI 中原临时 fixture/安全检查照常，不新增 PostgreSQL/Redis 或实际资源验收；本包没有持久化改动，不借机扩大故障矩阵。

### 6.1 guard、文档、inventory 与冻结边界

| 工件 | 确认后最小处置 | 保留/禁止 |
| --- | --- | --- |
| 本文、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md` | 设计阶段仅新设计+两个入口；确认后只更新同一状态/准确替代范围 | 不改历史正文/勾选/计数，滚动 HEAD/CI/复核/授权留 PR |
| 上表六 Go 工件 | 同一 PR 整根 RETIRE | 不保留兼容 wrapper、alias、迁移或替代策略 |
| `tests/listingkit_backend_retirement_test.go` | 只转换原 TestListingKitTaskBackendSourcesAreRetired 的四个 productpolicy KEEP 项；沿原 tracked sources/AST 设施增加本根全部文件缺席与全追踪 Go decoded import/子路径禁入、限定退休 root/owner 的专属顶层声明保护；保留原当前入口依赖退出检查 | 其它 KEEP/RETIRE、测试 negative fixtures、读取/解析失败即失败和全部安全/worker 合同不动；不全局禁 FilterRule/GetInventory/NewRuleChecker 同名声明，不 skip 丢文件，不新增 runner |
| `tests/target_architecture_phase2_test.go` | **零改**：继续禁止 shared 导入 productpolicy | 负向路径不是存在性要求；不以零调用删除架构检查 |
| 下列十二份直接含旧 productpolicy KEEP 的设计 | 仅各加一条链接到本文的窄 superseded 顶注：本包准确六工件/四 KEEP 检查不再保留，其余规则/API/采集/worker 不变；确认前不改这些旧依据 | 原正文/历史状态/数字/原验收证据保留；不是逐个重写旧设计 |
| `legacy-temu-filter-mapping-retirement-boundary.md` 的唯一 `../../internal/marketplace/productpolicy` 链接 | 实施时将这一链接标签准确标记历史 owner，目标换本文，避免指向删除目录 | 其它链接与该文旧规则/消费者叙述正文保留，由顶注限定其历史性 |
| 两份 2026-09-01 Phase3 specs/plans、#824 管理尾部设计及其它同名方法/负例 | **零改**，登记入口说明历史依据不能恢复本 owner | 不全仓扫词/改历史，不删除 ImageAgent/平台 inventory 同名合同 |
| 全部 workflow/inventory/依赖/锁/ceilings/保护规则 | **零改**；原 inventory CURRENT/历史/计数/parser/accuracy 均无改动 | 不为了 CI 手工删行/降阈值/取消 job；如真实原检查冲突超出此表，先报告精确提案 |

十二份窄顶注文件（均在 `docs/architecture/`）：

```text
legacy-temu-filter-mapping-retirement-boundary.md
legacy-temu-spu-price-query-retirement-boundary.md
legacy-temu-listing-submit-retirement-boundary.md
legacy-temu-sku-skc-retirement-boundary.md
legacy-temu-category-template-retirement-boundary.md
legacy-temu-text-bullet-retirement-boundary.md
legacy-temu-content-handler-retirement-boundary.md
legacy-temu-property-pipeline-retirement-boundary.md
legacy-temu-listing-closure-retirement-boundary.md
legacy-amazon-listing-closure-retirement-boundary.md
legacy-numeric-acquisition-tail-closure-retirement-boundary.md
legacy-listing-preview-download-closure-retirement-boundary.md
```

完整实施矩阵为六 Go +一个原 guard +三当前文档+十二准确旧依据，共 **22 个范围工件**；不要求为计数制造修改。设计阶段只三文档，不预先调整 guard 或旧 KEEP。#301 更新前新读正文；只切换当前段，前批当前段归档，历史后缀自“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”开始逐字保留，正文 ≤65536 字符，不修改历史勾选。#29 关闭状态不动。

## 7. 设计检查、实施与交付停止点

原 GitHub 设计检查集中核对三文档、真实消费者/当前需求、§5/§6.1 的完整边界和 KEEP 替代；不重新评审无关平台架构。按 AGENTS.md 先分类 finding，真实 BLOCKER 才改变已冻结边界；非 Blocker 的实现验证按原范围处理，不反复版本化设计或新建验证平台。

本轮是 Reuse Existing Architecture 的有界退休设计，不新增高风险 owner/状态/副作用边界；无需把现有独立平台方案再做一次全局 Architecture Review。设计检查不等于用户确认，原 CI 成功也不授权生产删除。若 GitHub review 无法启动/长期无结论，准确记录状态；**未经用户本轮授权不默认新增 AI 互验线程**，已运行同 SHA 不重复触发评审。

设计阶段交出准确设计 HEAD、原 CI 与设计检查证据，等待用户确认 §1–§7（含 §6.1）。用户确认后，原唯一 Writer 在同一 PR 记录/读回 Reuse Existing Architecture/Ready，一次完成六工件与全部专属尾部、必要自检/提交/推送。稳定实现候选集中最终交付检查；修复只复核对应增量，不另建 Writer 或机械拆 PR。最终准确实现 HEAD/原 CI/复核交用户决定合并，合并与 Issue 关闭均另行授权；若获合并授权，再核对实际 merge SHA 对应 completed/SUCCESS 的 push/main 整 run。

当前采集、平台 API/有效规则、图像/身份/Store 与独立 SHEIN 登录及 Store 同步不变；不操作既有实例/数据/凭据/进程，不修改 workflow/inventory，不增加内部试用前置。源码交付不等于用户验收/上线/物理清理，#301 继续 OPEN。
