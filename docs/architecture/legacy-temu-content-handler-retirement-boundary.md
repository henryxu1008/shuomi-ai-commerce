# 旧 TEMU 名称、描述与品牌清理编排退休边界设计

## 1. 状态、用户结果与范围

- 日期：2026-10-18；Refs #29 / #301。用户已确认 #810 冻结设计 `13db7447139aa4767ebf89588c2d7fe7b16901d1` 的 §5、§6、§6.1 和 §7，并授权同一 PR 实施。
- **USER_CONFIRMED / SOURCE_RETIRED；Design Basis: Reuse Existing Architecture**：生产修改前已在 #29/#301 登记并读回 Ready；§5 文件与孤立声明已源码退休，保留规则沿原 owner。原设计 N/A/CI/review 为历史，不替代实施证据；准确实施 HEAD、CI 和最终独立检查维护于 #810，合并另授权。
- 取证 actual main：`2dcbd80e3ea19448b1ec9c0d3d4dccb5b683e8e5`，含 #807 已退休图片 API/prompt。唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，复用附属独占 worktree issue29-temu-image-retirement/task-processor；新分支 codex/issue29-temu-content-handler-retirement-design。已核对接单/关联 PR，无同范围 owner，不接管 #809/#805/#692。

使用者为当前内部试用、采集/Store/文案维护者。交付是三个旧 handler、专属 optimizer/描述 helper/结果 DTO 的消费者、有效规则、进程内与持久化/平台副作用 owner、测试及删除矩阵。当前没有用户操作被此维护阻塞，**不增加内部试用前置**。无新页面或交互，Figma N/A。

候选包含旧七handler/helper及其最后消费者闭包：**10个整生产文件/1,250行、6个旧测试/206行**，另对publishing/name_format_policy.go（基线22行）做精确删减。后者仅保留仍被NormalizeProductSubmissionName使用的leftParenthesisSpacingPattern原声明/regexp import，移除孤立函数及两专属regexp；不整删product或publishing目录。PriceHandler、AI/SKU、有效API/DTO、当前图像与全部保留能力保持，无新增EXTRACT生产代码。

非目标：退休 PriceHandler、AIContentRewriter/文案 prompt、其它规则/平台 API/DTO、context/shared/model；修改仍有效规则的行为、语言/长度要求、品牌事实或当前 Product/Marketplace 资料；为已孤立旧sanitizer新接消费者；补完旧 API TODO、重接 pipeline、迁移评分/default文案、增加验证/治理工具或新入口。不得执行 constructor/main、下载/上传/模型/API/Preview/DryRun，读取实例配置/凭据/DB/Redis/已有覆盖记录/业务数据，或部署、重放、迁移、清理、关单。

## 2. Design Basis、authority 与不变量

本次实施采用 **Reuse Existing Architecture**。依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md) §3.5：旧 owner 仅 EXTRACT / RETIRE，不新增 wrapper/fallback/双读双写/旧 ID 映射/第二事实源。
- [#749 保存/提交退休](legacy-temu-listing-submit-retirement-boundary.md) 与 [#768 SPU/价格查询退休](legacy-temu-spu-price-query-retirement-boundary.md)：混合 product 包和旧名称/品牌/描述当批保留，不是永久保留权；用户确认本文后，只窄替代 §5 指定的旧对象，不重写历史证据。
- [当前 publishing owner](../../internal/marketplace/temu/publishing/README.md)：已抽取到该namespace不等于仍是当前requirement。按当前消费者/需求核对：保留仍被bullet_points_validator消费的NormalizeProductSubmissionName和必需leftParenthesisSpacingPattern；三个仅供本轮旧handler的policy/result/test与已孤立NormalizeParenthesesSpacing不以旧测试维持存活。
- [TEMU 当前平台设计](temu-platform-v1.md) M3/M4 与 §8：当前 Product/ApprovedAsset/明确资料和官方规则/Submission owner 保持；不能从旧文案优化器推断商品类型、品牌事实或平台接受。当前goods/rules按mode/category/官方snapshot校验，不要求旧ASCII剥离、统一10000字符或自动去品牌；这些旧policy不接入新路径。
- [#807 保留边界](legacy-temu-image-api-prompt-retirement-boundary.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：当前图像、通用 Prompt、身份/店铺/观察/独立 worker 的事实、权限、receipt/UNKNOWN 与持久化 owner 不变。

主要路径是旧pipeline.TaskContext → TemuTaskContext → 三handler/optimizer → 三个专供它们的publishing policy/旧内存DTO；没有当前命令装配。退出caller及最后消费者闭包；仍有保留consumer的提交格式化和全部当前有效规则保持。不改变保留合同的身份/tenant/权限、事实/schema、状态机/事务、幂等/retry/UNKNOWN/recovery、AI预算/计量或平台/provider 副作用 owner。少于30范围文件、生产删除低于2,500行，不以拆分规避风险提示线；若必须改变保留边界，停对应项补适用 Independent Architecture 准入。

Must：PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、AI/SKU/AIBatchInput、AIContentRewriter及文案prompt/key/catalog、当前图像/有效API/DTO/平台规则/共享合同与基础层、DimensionInfo/TextRenderer、两种SHEIN属性、Reader/Cache/Mapping、四TemplateState字段、当前采集/Store观察、独立SHEIN登录worker/Store sync与此前全部保留能力保持。两套及其它workflow、必需/安全检查、依赖/保护/ceilings不变；通用Prompt存储/管理和已有覆盖记录不动。

Should：原退休guard和当前清单准确反映范围。Out of Scope 如 §1；无新增Accepted Risk。Threat Model限于误删有效规则/同名合同、把旧TODO/nil/默认文案/评分当真实平台验证或产品事实、误把内存修改当持久化迁移；不扩成文案审核平台、IAM、事实验证状态机或运行验收矩阵。

## 3. 实际消费者与架构问题

基线刷新了全部旧文件、publishing最后消费者及构造器/receiver/结果/共享regexp引用。当前31个 cmd 的静态依赖并集中 temu/product 为0；目标owner无包外生产引用/构造/注册。未运行 main/constructor。该结论不证明实例、外部调用方或历史数据不存在；新真实consumer/具体兼容义务必须停相关项报告。

| contract → implementation → injection → consumer | 现有行为 | 决定 |
| --- | --- | --- |
| ProductNameValidator → ProductNameOptimizer / publishing | 清洗旧GoodsName，按旧关键词重排/删词/补Chair、200字节优化和500字节截断；只更新旧TemuProduct内存。旧构造器无外部装配 | RETIRE旧receiver/optimizer及失去最后consumer的SanitizeProductName；保留仍被要点校验消费的NormalizeProductSubmissionName |
| ProductDescriptionValidator → publishing / 内部评分 | 原描述先SanitizeProductDescription；核心enhanceDescription直接返回原值，再按长度/quality词给旧分数；空值拼接默认营销文案 | RETIRE旧编排/DescriptionValidationResult/评分/default文案及其专属SanitizeProductDescription闭包，不新增质量事实或复制模板 |
| description_validator_rules/enhance/score | 规则/关键词增强/详细评分/报告入口只见定义，未接到HandleTemu主路径；helper内部互调，isSupportedCharacter/isCommonWord仅此闭包使用 | RETIRE三文件与专属helper；不删除shared/mathx、model或pipeline合同 |
| BrandClearHandler → publishing | 从旧AmazonProduct.Brand取值，改旧GoodsName/GoodsDesc/BulletPoints、清空旧GoodsBrandProperties并设NotSelectBrand；不持有当前品牌事实owner | RETIRE旧mutator/receiver与只供它使用的TEMU RemoveBrandFromText；当前品牌/来源事实和其它平台同名实现保持 |
| ValidateProductNameAPI / ValidateDescriptionAPI | 只TODO日志后返回nil，没有HTTP、平台endpoint调用或真实校验 | 随receiver RETIRE；不能写成有效平台验证或新增真实API来“补完” |
| retained PriceHandler / AIContentRewriter | PriceHandler仍有logger/ProfitRuleAPI/运行值合同；AIContentRewriter在源码消费KTemuContentRewriterSystem | 原样保留；源码consumer不代表真实模型执行/用户可用性已验收 |

发现的架构问题：旧优化器会推断并添加Chair，旧默认描述带未证实营销文案；两个API验证方法成功返回却没有验证；描述长度/评分辅助链和主路径不一致。这些不属于当前产品/平台规则权威。无当前装配或事故证据，分类 **NOT_APPLICABLE（当前内部试用阻塞）**；用RETIRE处置，不修旧链、不接新验证服务、不追加试用前置。若真实当前消费者出现，不能用本分类跳过Must，须按 §7 停止/报告。

## 4. 最后消费者、有效行为与持久化 owner

首轮25c241c63的P2指出三policy会失去最后生产consumer。分类 **BACKLOG（当前内部试用非Blocker）**：不命中任何AGENTS列举的BLOCKER后果；设计阶段一次纳入同根因闭包，避免机械派生下一小批。原稿“已抽取即保留”不充分，修订已在冻结13db74471完成复核并获用户明确确认，实施授权仅覆盖本矩阵。

| 行为 / 边界 | 当前消费者与requirement证据 | 处置 |
| --- | --- | --- |
| SanitizeProductName / ProductNameSanitizationResult | 仅旧name_validator调用；其余仅自身测试。当前official goods按自身mode/category规则，不要求旧中文/高ASCII删除或装饰字符分类 | RETIRE整policy/regexp/结果/专属测试，不将Widget® 中文!旧fixture迁到保留owner以维持旧需求 |
| SanitizeProductDescription / ProductDescriptionSanitizationResult | 仅旧description_validator调用；10000长度、ASCII清洗/truncate只供旧链，无当前官方需求/消费者 | RETIRE整policy/const/helper/结果/专属测试，不把它接到当前描述路径或改写当前限制 |
| TEMU RemoveBrandFromText | 唯一生产caller旧BrandClearHandler将退出；SHEIN TextCleaner等同名方法不是本函数消费者，也不纳入删除 | RETIRE本policy及测试；当前品牌/来源事实和其它平台行为不变 |
| NormalizeProductSubmissionName | internal/temu/rules/bullet_points_validator.go:normalizeBulletPoint明确调用；publishing/submission_name_policy.go拥有括号/标点/空白规范化 | 原实现、supported helper与原测试保留，不重新写一套formatter。旧包装中的有效格式行为已由这里承接 |
| NormalizeParenthesesSpacing / name_format_policy.go | exported函数只有自身测试；rightParenthesisSpacingPattern只供它使用，whitespacePattern在上述描述policy退出后也无保留consumer。leftParenthesisSpacingPattern仍被NormalizeProductSubmissionName引用 | RETIRE孤立函数/两regexp及name_format_policy_test.go；同一mixed文件精确保留left regexp原声明与regexp import，不搬家、不留别名/空壳方法 |
| 原格式测试 | submission_name_policy_test已有旧cleanSpaces两例及Unicode空白；旧name_format三个准确输入/输出在此既有测试表承接，原例已存在则不重复 | 六旧测试文件退出；只把有效格式例保留给已有真实consumer的formatter，不迁移sanitizer/brand删除/评分断言。保留原formatter其它测试 |
| 旧TemuTaskContext.TemuProduct / DescriptionValidationResult | 原legacy内存指针/派生结果；七handler/helper无DB/queue/远端写入，没有durable owner | 只删旧mutator/专属DTO，不改context/平台Product/Extension/Brand字段，不读写历史行或迁移数据 |
| 当前Product/ApprovedAsset/TargetRecord与官方平台/Store/worker | 当前产品/官方Submission/effect/receipt/UNKNOWN/Store与独立SHEIN owner保持 | 不删除/更改品牌事实、资产、缓存、实例或远端商品，不新增恢复责任方 |
| logger、shared/mathx、pipeline、model及Prompt | PriceHandler仍消费core/logger；AIContentRewriter仍消费文案prompt；目标constructor未执行 | 原基础层/日志/配置/provider准入不变；logger行、文案key/YAML/catalog和通用Prompt原样保留 |

本批**无新增EXTRACT生产代码**。保留NormalizeProductSubmissionName及其确有消费的共享定义；其它三个policy、孤立格式函数、旧优化/营销推断/评分不因namespace或测试存在被认作有效requirement。保留当前official goods/rules/productpolicy/其它纯规则，不新接任何caller。“无当前需求”仅指本仓库已批准合同与源码消费者中未找到要求，不是对外部平台全部规范的断言；当前官方规则快照校验不变。

## 5. 精确删除与保留矩阵（用户已确认并源码退休）

前九文件相对internal/temu/product/；后续publishing行的生产及同目录测试均相对internal/marketplace/temu/publishing/。本节10整生产文件、6整测试文件已删除，1 mixed生产文件仅删除孤立声明；下列行数保留为冻结设计的删除前取证。

| 文件 / 对象 | 决定 | 边界 |
| --- | --- | --- |
| name_validator.go（128行） | RETIRE整文件 | ProductNameValidator/NewProductNameValidator及全部receiver/API TODO/日志编排；最后consumer policy见下 |
| name_optimizer.go（303行） | RETIRE整文件 | ProductNameOptimizer/NewProductNameOptimizer及全部重排/去重/补词/长度/建议helper，不新增替代optimizer |
| description_validator.go（170行） | RETIRE整文件 | ProductDescriptionValidator/NewProductDescriptionValidator、DescriptionValidationResult及全部receiver/default/评分/API TODO |
| description_validator_rules.go（121行） | RETIRE整文件 | 描述辅助规则/修复与专属isSupportedCharacter，不删其它包同名函数 |
| description_validator_enhance.go（81行） | RETIRE整文件 | 关键词增强/提取与专属isCommonWord，不删model/AmazonContext/shared合同 |
| description_validator_score.go（192行） | RETIRE整文件 | 详细评分/相关性/报告；shared/mathx不动 |
| brand_clear_handler.go（109行） | RETIRE整文件 | BrandClearHandler/NewBrandClearHandler及全部receiver/旧内存品牌字段修改 |
| name_validator_test.go（47行）、brand_clear_handler_test.go（25行） | RETIRE旧wrapper测试 | 有效格式由既有formatter测试承接；旧sanitizer/brand删除的测试不再是保留authority，不搬到新owner |
| publishing/name_validation_policy.go（67行）、name_validation_policy_test.go（33行） | RETIRE整文件 | SanitizeProductName / ProductNameSanitizationResult / temuProductNameAllowedPattern及仅供其使用的测试 |
| publishing/description_sanitization_policy.go（48行）、description_sanitization_policy_test.go（33行） | RETIRE整文件 | SanitizeProductDescription / ProductDescriptionSanitizationResult / temuProductDescriptionMaxLength / truncateProductDescription及测试 |
| publishing/brand_text_policy.go（31行）、brand_text_policy_test.go（45行） | RETIRE整文件 | 仅TEMU包级RemoveBrandFromText及专属测试；其它平台同名方法不删 |
| publishing/name_format_policy.go（22行，mixed）、name_format_policy_test.go（23行） | RETIRE孤立部分/整测试文件 | 删NormalizeParenthesesSpacing/rightParenthesisSpacingPattern/whitespacePattern及不再用的strings import；保留leftParenthesisSpacingPattern原声明/regexp import，供当前formatter使用 |
| publishing/submission_name_policy.go及_test.go | 保留 | 生产实现/原regex/声明不改；仅在原测试表承接旧name_format三例，不建框架或复制formatter |
| price_handler.go、product/context/API/平台DTO与publishing其余规则 | 保留 | PriceHandler/ProfitRuleAPI全部方法/价格/随机库存/运行值、当前goods/rules/model/productpolicy均不变，不整包清扫 |
| AI/SKU、AIContentRewriter及prompt/key/catalog、当前图像和此前全部保留能力 | 保留 | 独立SHEIN登录/Store sync、通用Prompt及已有覆盖行不动；不按“文案”或同名helper跟删其它owner |

上表publishing/为internal/marketplace/temu/publishing/；core seven + 三policy整文件共1,250生产行，mixed删减只按准确声明，不按全文件计算删除量。

Legacy decision: **RETIRE** 三旧handler/optimizer/helper/DTO、失去最后consumer的三policy/result、孤立格式函数和六测试文件。
Reusable behavior: 现有NormalizeProductSubmissionName及left regexp沿原owner保留，原格式例由既有consumer测试承接，无新增业务抽取/接线。
Current owner: 当前TEMU publishing formatter/其它有效规则、current goods/rules/model、Product/ApprovedAsset、原官方effect/receipt/UNKNOWN/Store、PriceHandler/AI及独立SHEIN。
Cutover/deletion condition: 用户明确确认本次修订§5/§6/§6.1/§7 → Issue Reuse Existing Architecture / Ready登记读回 → 刷新全部消费者/测试映射 → 原guard RED后精确退休/GREEN → 准确实施HEAD原CI/最终检查；无运行切换、迁移或兼容。

## 6. 验证与验收边界

设计阶段只做源码/静态依赖/owner与最后消费者/旧测试映射，交本文及索引；原设计CI/最终检查绑定设计HEAD。用户确认后按本节实施，原guard先准确捕获16文件及15包级声明的RED，再精确退休；三个原格式例在既有formatter测试表承接。稳定实施候选的GREEN/相关回归、准确HEAD原CI及最终检查在#810记录。真实平台/模型/运行/用户验收均 **NOT_RUN**，不把源码检查当作用户验收。

必要验证：

1. 开工同步actual main，刷新全部拟删文件/声明的构造/实例/注册/类型/helper、formatter/left regexp/PriceHandler/AI/prompt消费者及31命令依赖。新真实consumer或当前契约冲突，停对应项报告。
2. 在原退休guard加16整文件缺席、product及publishing精确包级声明禁令/保留断言，旧文件/孤立声明还在时确认RED。原格式三例由既有submission_name_policy_test承接，行为已存在时先GREEN，不伪造业务修复RED；再精确退休并确认原guard GREEN。
3. 运行现有publishing测试（含精确fixture）、retained PriceHandler/product、SKU/AI/rules及受影响context/API/共享包回归、原consumer/import/命令与独立worker检查、current-application与shein-login-worker build；不运行main/provider/平台。
4. 完整原CI留稳定实施候选，最终集中检查实际diff/消费者及原例映射；不人工重复无关全量或新建专项runner/故障平台。#29/#301其余债务保持OPEN；源码/PR CI/main CI与运行/用户验收分开。

### 6.1 原检查、清单与文档的精确处置

| 原位置 | 本次实施最小变更 | 不得削弱 / 扩大 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原严格缺席列表加16整文件；原product mixed walk以局部AST只禁9包级owner/ctor/DTO名（四receiver/四New/DescriptionValidationResult），publishing局部AST禁SanitizeProductName/ProductNameSanitizationResult、SanitizeProductDescription/ProductDescriptionSanitizationResult、TEMU RemoveBrandFromText、NormalizeParenthesesSpacing六声明，覆盖alias/var重建 | 补/保留PriceHandler、NormalizeProductSubmissionName、left regexp与当前测试精确存在断言；不全局禁generic helper/其它包同名type、field、selector、local。原五mixed包、SKU fixtures、#807及更早退休/安全/身份/采集/Store断言保持，不建通用扫描器 |
| publishing/submission_name_policy_test.go | 原表承接name_format_policy_test三准确格式例（重复则复用），保留已有Unicode/括号/标点测试；另外六旧测试整退 | 不迁移孤立sanitizer/brand/评分fixture，不重写规则/formatter，不强造RED或新测试框架 |
| tests/import_boundaries_test.go / phase2-runtime-inventory.md | **零变更**：core/logger→temu/product仍由PriceHandler消费。目标无OpenAI例外；原OpenAI→temu/product退出是#768历史 | 不改root/loader/deny/ignore/ceilings、dated清单、共享logger/AI准入或其它consumer |
| Legacy Register、code-health product行、本文/索引、publishing README及#749/#768直接冲突说明 | 用户确认后窄标记caller及最后consumer闭包source-retired；明确只保留有消费者的formatter/共享定义及其它有效规则。原声明数/报告/批准保留为历史，不能恢复旧policy | 设计阶段仅本文/索引；不将整个product/publishing或所有text/同名DTO退休，不重写历史证据 |
| 两套及其它workflow、依赖/保护 | **零变更** | 必需/安全检查、原编译目标、触发/并发/权限与全部ceilings保持，不加试用/发布前置 |

## 7. 确认、实施与停止条件

1. 用户已明确确认冻结13db74471的§5/§6/§6.1/§7并授权同一主要PR实施；原设计N/A只覆盖此前设计阶段。
2. 已在执行Issue登记并读回Reuse Existing Architecture / Ready、冻结设计HEAD、用户结果、调用路径/owner/不变量/验证，然后按矩阵精确退休；未以实现倒推Design Basis。
3. 新真实consumer、具体当前外部可观察兼容义务、有效行为缺owner或必须改变保留的身份/tenant/事实/状态/事务/规则/管理/恢复/副作用边界时，停受影响部分报告并补适用准入；不临时造adapter、fallback、双写或清理数据。
4. 实施候选In Review并交准确HEAD原CI/最终独立检查，finding先分类；只复核修复相关增量，非Blocker不自动扩大范围/重开全局架构。实现者/CI/AI不签发用户验收。
5. 合并仍需准确最终HEAD另授权，正常expected-head且不绕过保护。#29/#301保持OPEN；无部署/实例/凭据/已有覆盖行/业务数据授权，不增加内部试用前置。
