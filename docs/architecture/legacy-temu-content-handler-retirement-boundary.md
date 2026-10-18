# 旧 TEMU 名称、描述与品牌清理编排退休边界设计

## 1. 状态、用户结果与范围

- 日期：2026-10-18；Refs #29 / #301。用户授权先做名称校验、描述评分/增强、品牌清理编排退休设计，确认后再实施。
- **DESIGN_ONLY / AWAITING_USER_CONFIRMATION；Design Basis: N/A**：当前仅只读调查及本文/架构索引，不改生产、测试、guard、模板或 workflow。未来实施不是 N/A，拟采用 §2 的 Reuse Existing Architecture，当前未开工。
- 取证 actual main：`2dcbd80e3ea19448b1ec9c0d3d4dccb5b683e8e5`，含 #807 已退休图片 API/prompt。唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，复用附属独占 worktree issue29-temu-image-retirement/task-processor；新分支 codex/issue29-temu-content-handler-retirement-design。已核对接单/关联 PR，无同范围 owner，不接管 #809/#805/#692。

使用者为当前内部试用、采集/Store/文案维护者。交付是三个旧 handler、专属 optimizer/描述 helper/结果 DTO 的消费者、有效规则、进程内与持久化/平台副作用 owner、测试及删除矩阵。当前没有用户操作被此维护阻塞，**不增加内部试用前置**。无新页面或交互，Figma N/A。

候选为 internal/temu/product 的 **7 个生产文件/1,104行、2 个旧包装测试/72行**。product 包不整删：PriceHandler 原样保留；AI/SKU、有效 API/DTO、当前图像与此前全部保留能力保持。无新增 EXTRACT 生产代码；原有有效行为复用当前 owner。

非目标：退休 PriceHandler、AIContentRewriter/文案 prompt、其它规则/平台 API/DTO、context/shared/model；修改现有纯规则、语言/长度限制、品牌事实或当前 Product/Marketplace 资料；补完旧 API TODO、重接 pipeline、迁移评分/default文案、增加验证/治理工具或新入口。不得执行 constructor/main、下载/上传/模型/API/Preview/DryRun，读取实例配置/凭据/DB/Redis/已有覆盖记录/业务数据，或部署、重放、迁移、清理、关单。

## 2. Design Basis、authority 与不变量

未来实施拟 **Reuse Existing Architecture**。依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md) §3.5：旧 owner 仅 EXTRACT / RETIRE，不新增 wrapper/fallback/双读双写/旧 ID 映射/第二事实源。
- [#749 保存/提交退休](legacy-temu-listing-submit-retirement-boundary.md) 与 [#768 SPU/价格查询退休](legacy-temu-spu-price-query-retirement-boundary.md)：混合 product 包和旧名称/品牌/描述当批保留，不是永久保留权；用户确认本文后，只窄替代 §5 指定的旧对象，不重写历史证据。
- [当前 publishing owner](../../internal/marketplace/temu/publishing/README.md)：名称/描述清洗、品牌文本、提交格式已有独立纯值 owner，保留实现与现有合同，不 Wrap 旧 handler。
- [TEMU 当前平台设计](temu-platform-v1.md) M3/M4 与 §8：当前 Product/ApprovedAsset/明确资料和官方规则/Submission owner 保持；不能从旧文案优化器推断商品类型、品牌事实或平台接受。既有 publishing 纯规则的保留不等于所有当前官方模式均强制调用这些规则；不将其 ASCII/长度策略接入新路径。
- [#807 保留边界](legacy-temu-image-api-prompt-retirement-boundary.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：当前图像、通用 Prompt、身份/店铺/观察/独立 worker 的事实、权限、receipt/UNKNOWN 与持久化 owner 不变。

主要路径是旧 pipeline.TaskContext → TemuTaskContext → 三 handler/optimizer → 既有 publishing 纯规则/旧进程内 DTO；没有当前命令装配。只退出旧调用者及专属内存值，原四项纯规则与当前有效消费者保留。不改变保留合同的身份/tenant/权限、事实/schema、状态机/事务、幂等/retry/UNKNOWN/recovery、AI预算/计量或平台/provider 副作用 owner。少于30范围文件、生产删除低于2,500行，不以拆分规避风险提示线；若必须改变保留边界，停对应项补适用 Independent Architecture 准入。

Must：PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、AI/SKU/AIBatchInput、AIContentRewriter及文案prompt/key/catalog、当前图像/有效API/DTO/平台规则/共享合同与基础层、DimensionInfo/TextRenderer、两种SHEIN属性、Reader/Cache/Mapping、四TemplateState字段、当前采集/Store观察、独立SHEIN登录worker/Store sync与此前全部保留能力保持。两套及其它workflow、必需/安全检查、依赖/保护/ceilings不变；通用Prompt存储/管理和已有覆盖记录不动。

Should：原退休guard和当前清单准确反映范围。Out of Scope 如 §1；无新增Accepted Risk。Threat Model限于误删有效规则/同名合同、把旧TODO/nil/默认文案/评分当真实平台验证或产品事实、误把内存修改当持久化迁移；不扩成文案审核平台、IAM、事实验证状态机或运行验收矩阵。

## 3. 实际消费者与架构问题

基线刷新了九旧文件、全部构造器/receiver/结果类型/两个包级helper引用。当前31个 cmd 的静态依赖并集中 temu/product 为0；目标owner无包外生产引用/构造/注册。未运行 main/constructor。该结论不证明实例、外部调用方或历史数据不存在；新真实consumer/具体兼容义务必须停相关项报告。

| contract → implementation → injection → consumer | 现有行为 | 决定 |
| --- | --- | --- |
| ProductNameValidator → ProductNameOptimizer / publishing | 清洗旧GoodsName，按旧关键词重排/删词/补Chair、200字节优化和500字节截断；只更新旧TemuProduct内存。旧构造器无外部装配 | RETIRE receiver/optimizer/handler/专属方法；保留 SanitizeProductName / NormalizeProductSubmissionName |
| ProductDescriptionValidator → publishing / 内部评分 | 原描述先SanitizeProductDescription；核心enhanceDescription直接返回原值，再按长度/quality词给旧分数；空值拼接默认营销文案 | RETIRE旧编排/DescriptionValidationResult/评分/default文案，不新增质量事实或复制模板 |
| description_validator_rules/enhance/score | 规则/关键词增强/详细评分/报告入口只见定义，未接到HandleTemu主路径；helper内部互调，isSupportedCharacter/isCommonWord仅此闭包使用 | RETIRE三文件与专属helper；不删除shared/mathx、model或pipeline合同 |
| BrandClearHandler → publishing | 从旧AmazonProduct.Brand取值，改旧GoodsName/GoodsDesc/BulletPoints、清空旧GoodsBrandProperties并设NotSelectBrand；不持有当前品牌事实owner | RETIRE旧mutator/receiver；保留RemoveBrandFromText纯文本规则及当前品牌/来源事实 |
| ValidateProductNameAPI / ValidateDescriptionAPI | 只TODO日志后返回nil，没有HTTP、平台endpoint调用或真实校验 | 随receiver RETIRE；不能写成有效平台验证或新增真实API来“补完” |
| retained PriceHandler / AIContentRewriter | PriceHandler仍有logger/ProfitRuleAPI/运行值合同；AIContentRewriter在源码消费KTemuContentRewriterSystem | 原样保留；源码consumer不代表真实模型执行/用户可用性已验收 |

发现的架构问题：旧优化器会推断并添加Chair，旧默认描述带未证实营销文案；两个API验证方法成功返回却没有验证；描述长度/评分辅助链和主路径不一致。这些不属于当前产品/平台规则权威。无当前装配或事故证据，分类 **NOT_APPLICABLE（当前内部试用阻塞）**；用RETIRE处置，不修旧链、不接新验证服务、不追加试用前置。若真实当前消费者出现，不能用本分类跳过Must，须按 §7 停止/报告。

## 4. 有效规则、测试与持久化 owner

| 行为 / 边界 | 当前 owner / 证据 | 处置 |
| --- | --- | --- |
| 名称清洗与违规类别 | publishing/name_validation_policy.go 的SanitizeProductName / ProductNameSanitizationResult；原name_validation_policy_test.go覆盖装饰/高ASCII/中文、允许标点和商标转换 | 实现/合同不动。旧输入Widget® 中文!→Widget (R)、三违规的精确fixture迁入该既有测试文件，不保留旧wrapper |
| 提交名称格式化 | publishing/submission_name_policy.go / NormalizeProductSubmissionName；原测试准确包含旧cleanSpaces两例及Unicode空白，NormalizeParenthesesSpacing原测试保持 | 重复wrapper测试RETIRE。normalizeOptimizedName只加日志后委托同一函数，不保留包装器/日志测试，不新增规则 |
| 品牌纯文本移除 | publishing/brand_text_policy.go / RemoveBrandFromText；原测试准确覆盖旧品牌例、Unicode空白，另有大写/possessive/空品牌语义 | 旧brand_clear_handler_test.go两例已覆盖，RETIRE重复wrapper测试；纯规则与原测试全保留 |
| 描述清洗 | publishing/description_sanitization_policy.go / ProductDescriptionSanitizationResult；原测试覆盖空白/ASCII处理、10000长度/截断/违规 | 原样保留；不把未接线的2000长度/关键词/评分规则混入现有10000合同或新官方goods路径 |
| 旧TemuTaskContext.TemuProduct / DescriptionValidationResult | 原legacy内存指针/派生结果；目标七文件无DB/queue/远端写入，没有新增durable owner | 只删旧mutator/专属DTO，不改context/平台Product/Extension/Brand字段，不读写历史行或迁移数据 |
| 当前Product/ApprovedAsset/TargetRecord与平台/Store/worker | 当前产品/官方Submission/effect/receipt/UNKNOWN/Store与独立SHEIN owner保持 | 不删除/更改品牌事实、资产、缓存、实例或远端商品，不新增恢复责任方 |
| logger、shared/mathx、pipeline、model及Prompt | constructor会取原global logger，但本轮不执行；PriceHandler仍消费core/logger。文案prompt仍由保留AI消费 | 原基础层/配置/日志初始化及provider准入不变；不删logger consumer行、文案key/YAML/catalog或通用Prompt能力 |

本批**无新增EXTRACT生产代码**。保留已存在owner的正确行为，不把旧优化/营销推断/评分或未执行辅助规则复制成闲置工具；也不据此新接纯函数或宣称当前平台调用已完成。

## 5. 精确删除与保留矩阵（待用户确认）

以下路径相对 internal/temu/product/，全部是整文件；不退休目录本身。

| 文件 / 对象 | 决定 | 边界 |
| --- | --- | --- |
| name_validator.go（128行） | RETIRE | ProductNameValidator/NewProductNameValidator及全部receiver方法，含TODO API、日志/优化编排；原纯规则不动 |
| name_optimizer.go（303行） | RETIRE | ProductNameOptimizer/NewProductNameOptimizer及全部重排、去重、补词、长度/建议helper；不新增替代optimizer |
| description_validator.go（170行） | RETIRE | ProductDescriptionValidator/NewProductDescriptionValidator、DescriptionValidationResult、全部receiver/default/评分/API TODO |
| description_validator_rules.go（121行） | RETIRE | 描述辅助规则/修复与专属isSupportedCharacter；不删其它包同名函数/当前纯规则 |
| description_validator_enhance.go（81行） | RETIRE | 关键词增强/提取与专属isCommonWord；不删model/AmazonContext/shared合同 |
| description_validator_score.go（192行） | RETIRE | 详细评分/相关性/报告；shared/mathx原样保留 |
| brand_clear_handler.go（109行） | RETIRE | BrandClearHandler/NewBrandClearHandler及全部receiver/旧内存品牌字段修改 |
| name_validator_test.go（47行）、brand_clear_handler_test.go（25行） | RETIRE旧wrapper测试 | 按§4映射六个输入例/五个Test函数；一项精确sanitizer fixture进入既有owner测试，其余已覆盖或仅包装委托，不删业务规则 |
| price_handler.go及product目录 | 保留 | PriceHandler/ProfitRuleAPI与全部方法/随机库存/价格/运行值语义不变；不整包清扫 |
| publishing四纯规则/结果合同/原测试、当前productpolicy/TEMU goods/rules/model | 保留 | 仅允许§4精确fixture保留到name_validation_policy_test.go；不改生产规则、语言/长度策略或新接线 |
| AIContentRewriter、AI/SKU、content_rewriter prompt及其它Prompt；平台API/DTO/context和此前全部保留能力 | 保留 | 不按“文案”或同名type/helper跟删共享owner；独立SHEIN登录/Store sync保持 |

Legacy decision: **RETIRE** 三旧handler/optimizer/专属helper/DTO及重复wrapper测试。
Reusable behavior: 四项纯值行为采用现有publishing owner；保留必要原例的准确断言，无新增业务抽取。
Current owner: TEMU publishing/当前goods/rules/model、Product/ApprovedAsset、原官方effect/receipt/UNKNOWN/Store、PriceHandler/AI及独立SHEIN身份持久化。
Cutover/deletion condition: 用户确认§5/§6/§6.1/§7 → Issue Reuse Existing Architecture / Ready登记读回 → 刷新消费者/测试映射 → 原guard RED后精确删除/GREEN → 准确实施HEAD原CI/最终检查；无运行切换、数据迁移或兼容路径。

## 6. 验证与验收边界

当前只做源码/静态依赖/owner与旧测试映射，交本文及索引；文档diff/相对链接和原PR CI/最终独立检查绑定设计HEAD。生产/测试/guard/模板不改；真实平台/模型/运行/用户验收均 **NOT_RUN**。设计阶段不制造TDD RED。

用户确认后的必要验证：

1. 开工同步actual main，刷新九旧文件的构造/直接实例/注册/类型/helper、四纯规则/PriceHandler/AI/prompt消费者及31命令依赖。新真实consumer或当前契约冲突，停对应项报告。
2. 在原退休guard加九精确文件缺席和旧owner包级声明禁令，旧文件还在时确认RED。有效fixture用既有publishing测试文件承接，行为本来已存在时允许先GREEN，不伪造“业务修复RED”；再删旧文件并确认原guard GREEN。
3. 运行现有publishing测试（含精确fixture）、retained PriceHandler/product、SKU/AI/rules及受影响context/API/共享包回归、原consumer/import/命令与独立worker检查、current-application与shein-login-worker build；不运行main/provider/平台。
4. 完整原CI留稳定实施候选，最终集中检查实际diff/消费者及原例映射；不人工重复无关全量或新建专项runner/故障平台。#29/#301其余债务保持OPEN；源码/PR CI/main CI与运行/用户验收分开。

### 6.1 原检查、清单与文档的精确处置

| 原位置 | 未来实施最小变更 | 不得削弱 / 扩大 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原严格缺席列表增加九文件；在原product mixed walk内按局部AST只禁止9个包级owner/ctor/DTO名：四receiver/四New构造器/DescriptionValidationResult，覆盖type alias/constructor var。原保留map补四publishing纯规则与对应当前结果/既有测试的精确存在断言 | product目录/PriceHandler保留；不全局禁止isCommonWord/isSupportedCharacter、其它包同名type/field/selector/local；原五mixed包、SKU fixtures、#807及更早退休、安全/身份/采集/Store断言保持，不新建通用扫描器 |
| publishing/name_validation_policy_test.go / 原四类测试 | 最小承接Widget® 中文!的准确输出/三违规fixture；其它原测试不删不改语义，旧wrapper两文件退出 | 不重新实现规则或复制整套wrapper测试，不强造RED/测试框架，不把旧quality/default当验收 |
| tests/import_boundaries_test.go / phase2-runtime-inventory.md | **零变更**：七文件无OpenAI例外；core/logger→temu/product仍由PriceHandler消费，必须保留。已有OpenAI→temu/product退出是#768历史，不重复删行 | 不改root/loader/deny/ignore/ceiling、dated清单、共享logger/AI准入或其它consumer |
| Legacy Register、code-health product行、本文/索引、publishing README及#749/#768直接冲突说明 | 用户确认后窄标记九文件source-retired；product/PriceHandler/纯规则/AI仍保留，当前目标不再点名旧测试，原声明数/报告/批准保留为历史 | 设计阶段只交本文/索引；不把整个product/所有text或同名DTO退休，不重写历史证据 |
| 两套及其它workflow、依赖/保护 | **零变更** | 必需/安全检查、原编译目标、触发/并发/权限与全部ceilings保持，不加试用/发布前置 |

## 7. 确认、实施与停止条件

1. 本PR仅设计，等用户明确确认§5/§6/§6.1/§7后才在同一主要PR实施；当前N/A不授权改生产/测试/guard。
2. 确认后在执行Issue登记并读回Reuse Existing Architecture / Ready、冻结设计HEAD、用户结果、调用路径/owner/不变量/验证，再按矩阵精确退休；不以实现倒推Design Basis。
3. 新真实consumer、具体当前外部可观察兼容义务、有效行为缺owner或必须改变保留的身份/tenant/事实/状态/事务/规则/管理/恢复/副作用边界时，停受影响部分报告并补适用准入；不临时造adapter、fallback、双写或清理数据。
4. 实施候选In Review并交准确HEAD原CI/最终独立检查，finding先分类；只复核修复相关增量，非Blocker不自动扩大范围/重开全局架构。实现者/CI/AI不签发用户验收。
5. 合并仍需准确最终HEAD另授权，正常expected-head且不绕过保护。#29/#301保持OPEN；无部署/实例/凭据/已有覆盖行/业务数据授权，不增加内部试用前置。
