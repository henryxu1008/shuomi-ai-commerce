# 旧 SHEIN 敏感词整商品处理、反馈学习与词表管理完整退休闭包

## 1. 用户结果、范围与 Design Basis

状态：**DESIGN_ONLY / AWAITING_USER_CONFIRMATION**，Refs #301，父目标保持 OPEN。用户授权先设计、确认后实施；整商品处理、验证反馈学习、词表管理、强清理/诊断及专属尾部一个 Delivery Batch / 一个主要 PR 收齐，不按文件或检查机械拆批。使用者为维护当前草稿/预览清理、采集及独立 SHEIN 登录/Store 同步的维护者；结果是将实际文本清理与仅被自身/测试保活的旧管理链分开，给出可执行的完整删除/保留边界，不新增产品入口或试用前置。

**Design Basis: N/A / Ready（当前仅本文、Architecture README、Legacy Register 三文档）。未来实施为 Reuse Existing Architecture / NOT_READY**：复用当前 content 内存清理、publishing/shein 草稿清理与已有品牌/JSON合同；无新生产 EXTRACT、service、fact owner、schema、状态机、权限、事务、预算或 retry/UNKNOWN/recovery。只有两个生产文件 import content，既有窄接口只调用 SanitizeDisplayTextWithContext，保留其完整运行语义。13工件、单一实际清理owner；删除闲置内部协议不改变当前领域/跨系统边界，现有批准架构完整覆盖，不重复全局架构审核。若出现当前外部 wire/持久化消费者或必须改当前边界，停止对应部分并报告，不增加兼容。

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[目标架构](project-target-architecture.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、用户本轮明确范围及 #817/#900 已批准边界。无 UI/IA，Figma N/A；沿用原 Logrus/Go regexp/互斥锁和当前成熟能力，不造新词库框架。

Must：当前文本清理、默认词表/有效规则、品牌允许与来源品牌、大小写/标点、日文删词、当前平台 JSON/草稿/预览、采集、独立 SHEIN 登录/Store同步保持。Should：已有 owner/检查/fixture优先，不引入替代管理API。Out of Scope：workflow、完整inventory/CURRENT/历史/计数/ceilings/parser、依赖/保护零改；不读既有配置/词库/凭据，不操作既有实例、数据、日志或进程，不执行真实平台/浏览器/业务入口，不建运行验收平台，不增加试用前置。#29不重开，#301不关闭；实施、合并、部署、关单分别授权。

## 2. 调查基线、实际消费者与旧 KEEP 替代

基线 main **256da97f46b4d10c1950d6c53d384a84b7ed9166**，tree **2e8b105d277bc1a41a08888264d65a65378ba33d**。原唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，复用独立 worktree/分支 codex/issue301-shein-sensitive-word-closure-retirement-design；[接单/前批当前段完整归档6393124009](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6393124009)。已刷新开放#931依赖/#692 Store设计，无本范围第二Writer或PR；不接管主checkout的采集/Figma工作。

纯AST解析全部 **3267个受追踪Go文件**，含tests/build tags/嵌套模块，不加载或运行业务包。content有两个生产importer：sensitive_content_sanitizer.go只消费CleanupContext；submit_prep_sensitive_adapter.go消费该Context与原内存工厂，返回只有SanitizeDisplayTextWithContext的窄接口。外部测试importer只有processor_authorized_brand_test.go；没有dot/blank import。全文核对方法值/调用与同包引用后，旧入口无外部生产消费者；它们的内部相互调用与自测不构成当前需求。

| 支路 | 真实使用 | 决定 |
| --- | --- | --- |
| NewSensitiveWordServiceInMemory → SanitizeDisplayTextWithContext | 当前publishing窄接口/草稿与预览清理 | KEEP，原构造、logs callback/日志fallback、文本规则和消费者零改 |
| ProcessProductData → 多语言/属性/SKC回写 | 只有HandleValidationErrors及包内旧fixture | RETIRE整链，当前草稿字段遍历已有publishing owner，不迁移旧编排 |
| HandleValidationErrors/消息提词 | 只有声明、内部回灌与包内测试；ExtractSensitiveWordsFromValidationNotes亦无调用 | RETIRE DTO/正则提词/追加/重新处理/返回retry意图；无继任owner，不接到Submission重试 |
| Add/Get/Clear词表管理 | 只有旧反馈链和测试 | RETIRE公开管理方法及分类/写入/计数/元数据更新；默认词表和当前只读删词规则KEEP |
| SanitizeTextWithContext/强清理/TestEmojiFiltering | 无外部生产调用，专属helper只服务旧批处理/强清理 | RETIRE；不把强清理、截断200字节或Product替代文本接到当前display接口 |
| containsJapanese | 同时被旧detectLanguage与当前removeWordFromText调用 | **KEEP原实现**；不能随语言识别支路一起退掉 |

| 原批准依据 | 本轮需要确认的准确替代 | 继续KEEP |
| --- | --- | --- |
| [#817 E3](legacy-shein-listing-closure-retirement-boundary.md) §5/§6 | ProductData窄输入、ValidationResult/SKCValidationMessages及语言/SKC反馈JSON、整商品清理/提词保留 | CleanupContext.Context/AuthorizedBrand/SourceBrand、当前品牌/文本清理与平台Product/LanguageContent/SKC/属性模型及JSON |
| [#900](legacy-workerpool-sensitiveword-recovery-closure-retirement-boundary.md) §4.1/§4.2/§5/§6 | 所有词表管理/反馈方法、LastUpdated/Version与markConfigUpdatedLocked、专属防御复制/并发mutation/四槽JSON要求 | 原内存工厂、SensitiveWordService/config/mutex/logger、两词表map及当前清理/读锁/默认词与有效规则 |

两个旧文档在未来确认/实施后只加准确superseding顶注，完整原Git blob作为后缀保留；旧批准/计数/勾选/失败不改。其它文档中的“保留当前清理/有效词表”仍成立，不把泛称扩为所有旧管理方法永久KEEP，也不扫改其它历史。

## 3. owner 与 contract → implementation → injection → consumer

当前路径：publishing/shein 草稿/预览字段遍历 → sheinSensitiveWordSanitizer窄接口 → newSheinSensitiveWordSanitizer(logs) → content.NewSensitiveWordServiceInMemory → SanitizeDisplayTextWithContext。AuthorizedBrand来自当前authorizedbrand.FromContext/Resolved；SourceBrand仍是单次调用值。当前草稿属性是否为自由文本由publishing.shouldSanitizeDraftAttributeValue已有规则判断，不依赖旧content.shouldSanitizeFreeTextAttributeValue；不新增抽取代码。

content继续拥有默认词表与实际纯清理实现。SanitizeDisplayTextWithContext保留：授权品牌allowlist → 词表删词 → Amazon品牌删词 → SourceBrand删词 → 空白整理；不接入旧强清理。保留原删词大小写/连字符/空格/日文路径、授权名称与英文名称、brand token比较及所有返回语义。publishing两消费者、当前草稿/预览/严格解码/版本/hash/读取拒绝和平台合同零改。

保留的17个函数/方法：

```text
componentLogger
NewSensitiveWordServiceInMemory initDefaultConfig getAllSensitiveWords createDefaultConfig
SanitizeDisplayTextWithContext authorizedBrandAllowlist removeSensitiveWordsExcept
removeAmazonBrandWordsExcept removeContextBrandWordsExcept containsFoldedText
normalizeBrandToken getAmazonBrandWords
containsJapanese removeWordFromText cleanupText deduplicateWords
```

除§5明确的默认元数据赋值退出与一条旧helper说明改写外，保留函数代码body逐字保持。SensitiveWordService三字段/原mutex/只读锁保持，不以管理入口退休为由顺带重写并发或改词表。没有新事实、状态机、权限、服务接线、文件/协程生命周期或外部副作用owner。

## 4. 元数据、JSON、持久化及副作用边界

| 对象 | 当前源码证据 | 退休/保留边界 |
| --- | --- | --- |
| CleanupContext.ProductData | 只被旧ProcessProductData/SKC回写与包内fixture消费；当前publishing只用Context/AuthorizedBrand/SourceBrand | 退该字段和product import；其它三字段/原类型/品牌值保持。无JSON tag/持久化入口，不换成新payload或第二事实源 |
| ValidationResult/SKCValidationMessages | 包内旧提词/JSON round-trip测试；没有HTTP/仓库/平台decoder/当前payload消费者 | 两类型及专属JSON槽位整退，不映射/迁移/兼容。**不涉及当前SHEIN平台Product及其语言/SKC/属性JSON**，同名其它owner DTO不动 |
| SensitiveWordConfig.LastUpdated/Version | 构造时赋值；只在旧mutation更新/旧stats日志与包内测试读取/序列化。生产工厂返回窄接口，无config getter/serializer/注册入口 | 退两字段、last_updated/version标签、默认两赋值、time import和markConfigUpdatedLocked。没有新timestamp/version/hash/revision替代物 |
| SensitiveWordConfig.StaticWords/DynamicWords | 原getAllSensitiveWords/默认词表实际读取，动态map默认空，但两集合合并/去重是当前纯规则 | 原map类型与static_words/dynamic_words标签、默认全部语言词条/顺序及空动态map保持；不是新外部JSON端点或兼容4槽协议 |
| 词表管理 | 当前没有生产Add/Get/Clear调用或配置/词库读取；最后写路径只属于旧管理链 | 退写入/自动语言路由/清空/查询API；有效自定义短语清理用原包测试准备合成map验证，不增加生产设置接口/fixture构造器 |
| 当前并发 | 当前SanitizeDisplay读取原mutex保护的config；原getter返回新去重slice | 保留getAllSensitiveWords读锁与防御复制，原清理/字典不变；退无消费者的Add/Get/Clear并发mutation要求，改原测试验证当前并发只读清理 |

只证明本仓库显式源码，不推断真实旧词库/JSON文件是否存在或已被外部程序读取；其状态UNKNOWN。本轮不读取这些材料，不处理已有覆盖/文件/行/key，不创建删除或回填迁移。#900文件持久化/保存协程已退的禁入继续有效，不增加I/O、fallback、双读/双写或新保存owner。若出现具体当前wire/持久化义务，先报告用户，不以旧测试推定兼容，也不自行接受风险。

独立SHEIN worker的授权/claim/fence/UNKNOWN/凭据/Redis/登录/Store同步不在本链，源文件/JSON/持久化合同全部KEEP。当前AI/图像/采集/HTTP/Retry/身份/SourceIdentity与所有原安全检查不改，不用本设计变更内部试用或生产发布阶段。

## 5. 完整13工件删除/保留矩阵

未来同一PR收齐：**1整退 + 5生产混合 + 1测试混合 + 1原guard混合 + 2旧依据顶注 + 3当前文档 = 13工件**。43函数/方法退出，17实际函数保留。当前阶段只改最后三文档，不提前实施其它10工件。

### 5.1 整文件 RETIRE

`internal/shein/content/validator.go`，六方法全退：extractSensitiveWordsFromValidation、ExtractSensitiveWordsFromValidationNotes、extractWordsFromMessages、splitWords、addWordsByLanguageLocked、addWordsToConfig。没有当前消费者、保留算法无需迁到另一个管理器。

### 5.2 五生产混合文件

| 文件 | 准确RETIRE/允许改动 | 必须保持 |
| --- | --- | --- |
| internal/shein/content/word.go | 12方法：ProcessProductData、HandleValidationErrors、GetStaticSensitiveWords、GetDynamicSensitiveWords、getWordsFromConfig、GetSensitiveWordsByLanguage、AddDynamicSensitiveWords、AddStaticSensitiveWords、AddStaticSensitiveWordsByLanguage、addSensitiveWordsLocked、AddDynamicSensitiveWordsByLanguage、ClearDynamicSensitiveWords及专属注释 | 原工厂/日志、initDefaultConfig、getAllSensitiveWords、原import全部不变 |
| internal/shein/content/processor.go | 13函数：removeSensitiveWords、removeSensitiveWordsAndBrandsWithContext、SanitizeTextWithContext、processMultiLanguageNames、processMultiLanguageNamesWithBrandsAndContext、processMultiLanguageDescs、processSKCData、processProductAttributes、shouldSanitizeFreeTextAttributeValue、cleanTextForSheinPlatform、removeAmazonBrandWords、removeContextBrandWords、TestEmojiFiltering；移除最后用途退出的regexp/unicode/product imports；removeSensitiveWordsExcept中提及旧normalizeSpecialCharacters的说明仅改为保留原文大小写/标点的说明 | §3本文件8函数代码body、原品牌词表/允许列表/SourceBrand语义不变；strings保持，不替换任何算法 |
| internal/shein/content/utils.go | 11方法：classifyWordsByLanguage、detectLanguage、containsChinese、containsKorean、containsCyrillic、containsArabic、filterEmojis、removeAllEmojisAggressively、normalizeSpecialCharacters、countWordsInConfig、logSensitiveWordStats；退出unicode import | containsJapanese/removeWordFromText/cleanupText/deduplicateWords四函数及regexp/strings原样，不能误退当前日文规则 |
| internal/shein/content/config.go | markConfigUpdatedLocked、LastUpdated/Version字段/标签、createDefaultConfig内对应两赋值及time import | SensitiveWordConfig两map及tags、Service三字段、sync/logrus、createDefaultConfig其余完整字典/空map原样；不改初始化/锁协议 |
| internal/shein/content/cleanup_context.go | ProductData字段/product import、ValidationResult/SKCValidationMessages及专属注释 | CleanupContext.Context/AuthorizedBrand/SourceBrand及其原类型/import保持 |

### 5.3 测试、原guard、旧依据与当前文档

| 文件 | 准确处置 |
| --- | --- |
| internal/shein/content/cleanup_context_test.go | 按§6.1转换原六测试；有效当前规则完整保留，专属旧协议测试退出，无新生产seam |
| tests/listingkit_backend_retirement_test.go | 仅§6.1两个原函数调整；第三个NumericContent guard和全部其它检查零改 |
| docs/architecture/legacy-shein-listing-closure-retirement-boundary.md | 未来只加#817 E3准确替代顶注，完整本基线Git blob保留为后缀 |
| docs/architecture/legacy-workerpool-sensitiveword-recovery-closure-retirement-boundary.md | 未来只加#900管理/反馈/元数据KEEP准确替代顶注；原完整历史后缀、数字/计数/勾选保持 |
| docs/architecture/legacy-shein-sensitiveword-closure-retirement-boundary.md | 本文，设计/授权状态与原依据明确分层，不用未来状态冒充当前实施 |
| docs/architecture/README.md | 只加当前设计入口；完整原Git blob后缀保持 |
| docs/refactoring/legacy-register.md | 只加本准确完整闭包说明；完整原Git blob后缀/计数/历史保持 |

**显式零改**：content/logger_port.go、processor_authorized_brand_test.go；publishing/shein的sensitive_content_sanitizer.go、submit_prep_sensitive_adapter.go及当前草稿/预览/品牌/自由文本属性/JSON测试；shein/api/product模型/DTO/JSON、authorizedbrand Resolved/context；tests/import_boundaries_test.go、workflow/完整inventory/依赖/保护及所有独立worker/Store文件。不能以包/语言/敏感词名字批量清扫。

```text
Legacy decision: RETIRE
Reusable behavior: 无新生产EXTRACT；默认词表与当前display清理/品牌/日文/去重留原owner，已有draft字段处理规则复用现owner
Current owner: content内存只读清理；publishing/shein草稿/预览字段遍历；当前平台JSON及各采集/登录/Store owner
Cutover/deletion condition: 用户确认§1–§7/§6.1与未来Reuse Existing Architecture / Ready读回后，同一PR按准确矩阵收齐；无当前消费者/新兼容/运行资源操作
```

## 6. 必要验证与验收边界

设计阶段只做AST/消费者/字段/JSON/引用调查、13工件/相对链接/索引历史后缀、原适用文档检查和稳定设计候选原CI；不制造业务RED，不调用业务constructor、配置加载、旧脚本/平台、真实DB/Redis/浏览器，不开发runner/验收平台。

未来获准后，先在两个原guard中加入精确缺席/声明/字段禁入和正向保留确认RED，再完整退出至GREEN。开发自检只复用content原测试（含race）、publishing与marketplace/shein当前清理/品牌/自由文本/JSON/严格解码相关fixture、原retirement/import/inventory/README检查；不人工重复原CI全套。只编译current-application、独立SHEIN登录、ImageAgent worker、1688-local-agent与hack/debug原Amazon入口，-mod=readonly输出任务临时目录且不执行。最终稳定源码原CI/一次集中交付检查绑定准确HEAD；未变化证据允许明确复用，不能冒充新SHA执行或运行验收。

### 6.1 原检查完整转换矩阵

| 原函数/检查 | 唯一允许转换 | 必须保持 |
| --- | --- | --- |
| TestLegacyWorkerpoolSensitiveWordRecoveryClosureIsRetired | 原缺席表加content/validator.go；retiredFunctions追加shouldSanitizeFreeTextAttributeValue，retiredMethods追加§5全部42个准确SensitiveWordService方法；retained只移除ProcessProductData/HandleValidationErrors/GetSensitiveWordsByLanguage/getWordsFromConfig/classifyWordsByLanguage/addWordsToConfig六旧保活，追加§3其它实际函数正向项；wantConfig只退LastUpdated/Version两项，保留两map/tag精确比较；在原content GenDecl分支精确禁止ValidationResult/SKCValidationMessages类型及CleanupContext.ProductData字段 | 原workerpool/recovery/文件持久化/保存协程/生命周期禁入、完整tracked扫描/import解析/receiver/错误处理、原factory唯一logs参数检查、Service三字段/锁/logger、map/tag反射匹配全保留；不全包禁入或missing-skip |
| TestListingKitTaskBackendSourcesAreRetired | retained token map的cleanup_context.go行仅退出ValidationResult/SKCValidationMessages/两个旧语言消息JSON key；保留type CleanupContext struct/SourceBrand，追加AuthorizedBrand/Context正向tokens | 全部其它路径/函数/字段禁入、当前平台JSON/属性/日志/DB/ptr/模型/原命令等断言与算法零改 |
| TestNumericContentConfigurationClosureIsRetired及import_boundaries的content检查 | **零改** | 原内存工厂调用、窄logs callback传递、CleanupContext存在、禁止旧管理/数字配置/包import等原逻辑继续有效；无需改扫描根或豁免 |
| TestCleanupContextPreservesAuthorizedAndSourceBrandRules | 仅去ProcessProductData(nil)/空payload两个专属检查 | 原SanitizeDisplay授权品牌/来源品牌/大小写/普通文本/nil context断言与fixture原字节保持 |
| TestCleanupValidationResultPreservesLanguageAndSKCJSON | 专属反馈DTO/提词/回灌测试整退 | 当前草稿/平台多语言/SKC/属性/严格JSON测试不动，不改平台同名DTO |
| TestInMemoryCleanupUsesExplicitCustomPhrasesWithoutChangingDefaults | 只将AddStaticSensitiveWordsByLanguage调用换成测试内原config.StaticWords map的合成fixture准备，沿原mutex加解锁；无新增生产helper/入口 | 原默认未加自定义词不删除、加词后短语移除且Office curtain保留的全部有效断言/输入保持，不扩默认词表 |
| TestSensitiveWordMutationsCompleteMetadataSynchronously | 旧mutation/metadata专属测试整退 | 不把删除元数据要求写成通过；原历史#900结果保留，当前只读词表与锁测试继续覆盖 |
| TestSensitiveWordMemoryContractAndDefensiveCopies | 保留原完整四语言默认字典/空动态map断言；只去LastUpdated/Version/time断言，JSON断言只保留两原word map/tag/round-trip。旧管理API调用改为原map合成fixture准备；用当前getAllSensitiveWords验证trim/lower/去重/多语言词存在与返回slice修改不影响存储 | 默认词条/顺序/不新增词、StaticWords/DynamicWords原map类型/tag、有效自定义词清理/规范化/防御复制保持；不新建运行JSON接口 |
| TestSensitiveWordConcurrentAddGetClear | 原测试更名TestSensitiveWordConcurrentCleanupReads；移除已退管理/元数据操作，在启动goroutine前准备原map fixture，原并发测试只反复执行当前SanitizeDisplay和getAll返回副本的独立修改，验证结果与存储不变 | 保留真实并发/race覆盖、当前mutex读锁和默认/品牌清理语义，不加生产同步器/状态或新runner |
| processor_authorized_brand_test.go、publishing当前清理/草稿/预览/自由文本/JSON测试、SSA/import/inventory/其它安全检查 | 零改 | 当前品牌允许/SourceBrand、case/punctuation/原协议及独立SHEIN/Store保留，不跳过必需CI |

§6.1为全部允许的检查转换，不仅新增guard；实施前再按准确main刷新同根文件/函数/DTO引用。真实测试若发现表外尾部，先给具体差异和最小方案，交用户决定后应用，不恢复占位声明、绕过解析、放宽当前安全断言或自行扩大架构。

## 7. 交付与停止点

本阶段仅三文档、一个主要设计PR；#301当前段写N/A/Ready/设计中，历史从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保持，不改旧勾选，正文≤65536字符。最终设计HEAD、原适用CI与真实复核/限制留PR；设计完成交用户确认，不默认继续生产。开发起始已同步main，最终候选前再核对一次实际main，只处理真实组合，不持续追新。

**未经用户本轮授权不默认新增AI互验线程**；只读设计复核如需子代理须另获授权，不因旧批授权自动派工。原GitHub review同SHA运行中不重复触发；启动失败只允许一次重试，仍无正式结论如实记UNKNOWN，不当作finding或PASS，不重开无关全局架构。

确认§1–§7（含§6.1）后，先更新并读回未来Reuse Existing Architecture / Ready，再由原唯一Writer在同一PR完成13工件、原guard RED→GREEN、必要开发自检/原CI及一次集中最终检查。实现者自检/AI复核均不等于用户验收。合并须另获准确HEAD授权、正常expected-head执行；之后核对实际merge SHA的push/main整run completed/SUCCESS，PR CI/其它提交/单Gate不替代。#301持续OPEN，关单另授权；真实平台/登录/Store同步/用户/生产验收NOT_RUN，既有资源操作NOT_AUTHORIZED，不增加内部试用前置。
