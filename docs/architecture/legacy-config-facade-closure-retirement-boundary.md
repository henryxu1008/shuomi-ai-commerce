# #301 旧配置加载与校验门面完整退休闭包设计

Refs [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)。#301 保持 OPEN，#29 不重开。

- Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已确认冻结设计 `f88f791a1c7d4f55750050795b7aff77e05f5183` §1–§7（含 §6.1），同一主要 PR #917 整批实施；准确源码 HEAD、必要自检、CI、最终复核与权限留 PR，不把源码完成当运行/用户验收。
- Design Basis：设计阶段三文档 **N/A**；本轮实施 **Reuse Existing Architecture / Ready**。[用户确认/接单6379330796](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6379330796)及 Issue/PR Ready 记录读回先于生产修改。冻结 §1–§7/§6.1 语义与矩阵不扩展；本文不是合并、运行或关单授权。
- 基线：main **12da6cc2856a693f2c58e8a28d6c2faf942d7de3**，tree **cf6f46689c5142ad807944652dec4a9bd995bea3**。#915 的 matching push/main CI39982851824 整 run SUCCESS 只作前批基线。实施开工同步实际 main `11dddb74a06fae3f341c6fb2910842d285e15d6d` 的 #918 采集报价增量，不改本批配置/校验链；共享索引新增行保留。
- 唯一 Writer 沿原独占 worktree，分支 `codex/issue301-config-facade-closure-design`；[接单及前批当前段逐字归档6369445835](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6369445835)。一个主要 PR 覆盖全部闭包，不按文件、方法或测试拆批。

## 1. 用户结果、范围与产品依据

使用者为维护当前配置、采集、HTTP、图像、登录及 Store 源码的维护者。交付结果是退出没有当前入口的旧宿主文件加载包装、重复 Validator 对象及 Config 校验门面；有效校验在现有函数 owner 下继续执行。没有当前核心用户操作因此阻塞，不新增配置产品、校验平台或内部试用前置。

本批完整范围：两条旧加载入口及其 Amazon 专属 helper；Validator/NewValidator 与全部方法；Config 三个校验门面、专属日志包装；四项有效聚合调用的选择性抽取；当前加载链与四个原测试文件的调用尾部、原 guard 和两份直接冲突旧 KEEP。设计阶段仅本文、Architecture README 和 Legacy Register；源码实施范围见 §5/§6.1。

Authority 为本轮用户明确范围、[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、当前配置与字段校验 owner。UI/IA/Figma N/A；历史 KEEP、导出声明、原测试和 `WithFallback` 名称均不自动构成当前产品要求。

Must：完整闭包、有效规则及当前全部能力保持；原 OpenAI/Browser/Amazon/Platforms/ListingKit/Workbench 六组校验不减，聚合顺序、nil/无错误结果、错误分组/字段/hint/格式不变；不触碰既有资源，不新增试用前置。Should：复用现有函数、Viper、标准库与原测试，无新配置框架或验证工具。Out of Scope：配置 schema/default/env/样例变化、放宽或严格化规则、全局 nil-safe 改造、凭据迁移/兼容/fallback/双读双写、依赖整理、provider/运行验收及其它退休。Accepted Risks 无新增；真实环境状态 UNKNOWN，运行/产品验收 NOT_RUN。

Threat Model 限漏接有效校验、第一处错误即短路、聚合顺序/格式漂移、误删同名现役 Validator/Config、将旧 helper 的文件/日志/nil 合同带入当前 owner，以及测试失去原拒绝能力。不扩大为新配置安全平台或输入策略。

## 2. 根因、Design Basis 与前批 KEEP 替代

旧宿主 API 退出后，两条文件加载门面只剩声明和专属内部调用。`ValidateConfig` 仍构造 Validator，后者只把四个当前规则按顺序调用；Config 又增加一层同义方法。根因是旧装配外壳与有效规则混在同包，早期广泛 KEEP 和原测试调用令其继续存在。处理为 **RETIRE 外壳 + EXTRACT 仍有效的四项聚合调用到既有 ValidateConfig**，不删除有效规则、不造新 Validator/facade/service。

| 前批依据 | 经本轮确认后仅替代的准确 KEEP | 仍保留 |
| --- | --- | --- |
| [#836 配置闭包](legacy-runtime-config-closure-retirement-boundary.md) §5.2 | loader.go 的旧路径检查/日志及 Amazon 专用入口；NewValidator 暂留；Config 方法和对象构造形式 | NewDefaultConfig/LoadFromBytes、有效加载/环境优先级、字段规则、错误聚合/格式、身份/Workbench拒绝 |
| [#878 宿主配置闭包](legacy-host-config-featureflag-tracing-retirement-boundary.md) §3/§5.3 | 旧 Config/Validator 方法的具体调用形式及同文件闲置门面 | 当前 Viper/文件/bytes/WithoutValidation/BuildConfig、scoped .env、认证关闭拒绝、初始化保护及所有有效校验语义 |

上述旧文档在实施时仅增加准确 superseding 顶注，原正文完整后缀保留；不批量撤销旧 KEEP 或改历史计数。#915 保留的日志/版本、HTTP/TLS/retry、JPEG/PNG/WebP 生产解码注册不在本次删除矩阵。

拟实施为 **Reuse Existing Architecture**：仍是 core/config 的同一 Config、同一六组规则和同一错误出口；没有新业务能力、公共领域合同、配置 owner、事实/schema、权限/租户、状态机、事务、retry/UNKNOWN/recovery 或外部副作用边界。十四个范围工件，无新增高风险边界；复用现有成熟组件，不另造开源能力已有的替代品。若真实消费者或持久化/外部可观察合同要求改变上述边界，停止对应删除并报告，必要时升级 Independent Architecture，不在实现中增加兼容。

## 3. 实际消费者：contract → implementation → injection → consumer

核对全部 tracked Go、导入别名/selector、根模块及 `hack/debug` 独立模块、包内引用、原测试/脚本/workflow 与直接旧文档。core/config 包仍被当前路径广泛消费，但没有包外调用两旧入口或构造 NewValidator；包级保留不等于所有旧声明均需保留，以下准确调用链才是依据。

| 对象 | 实际调用/尾部 | 决定 |
| --- | --- | --- |
| LoadConfigWithFallback | 只有 loader.go 自身声明，没有生产/debug/test 调用；内部 Stat→LoadConfigFromFile→日志，不存在实际 fallback 分支 | RETIRE 完整入口及专属 imports/comment；不改当前文件 loader |
| LoadAmazonCrawlerAPIConfigWithFallback | 只有声明；内部调用 LoadConfigFromFileWithoutValidation 和唯一 validateAmazonCrawlerAPIConfig | 两者同退；原 Amazon browser/采集/debug 不通过此入口 |
| validateAmazonCrawlerAPIConfig | 只被上一旧入口调用；聚合 Browser/Amazon 两规则，另有仅其拥有的 nil 错误 | helper RETIRE；实际规则函数保留，不把其局部 nil 合同推广为新全局需求 |
| Validator/NewValidator | 类型/构造仅在自身文件，唯一外部于文件的构造调用为同包 ValidateConfig | 对象完整 RETIRE；四项聚合行为先抽回原函数 owner |
| Validator.Validate/ValidateAndLog/ValidateWithError | Validate 被 ValidateConfig 和本对象两闲置方法调用；两个日志/错误方法无外部 caller | 方法与专属 Logrus imports 同退，不增加新对象 |
| Config.Validate | 四处调用：config_test.go 的两个原测试共三处、validator_dependencies_test.go 一处 | 原断言保留，同批切到 ValidateConfig；不把测试调用误称为零消费者 |
| Config.ValidateWithError | config.go/loadWithViper 一个生产调用；validator_listingkit_test.go 和 validator_workbench_test.go 各一处 | 三处切到 ValidateConfigWithError，当前入口不减校验 |
| Config.ValidateAndLog/ValidateConfigAndLog | 前者无 caller，后者只被前者调用 | 完整日志包装 RETIRE；当前日志 owner/依赖关系不动 |
| 当前加载/运行入口 | 原 browser/debug 使用 LoadConfig/LoadConfigFromFile；ImageAgent/schema/current image worker等使用原加载入口或 WithoutValidation；1688 local agent 用 NewDefaultConfig；currentapplication manifest 与 sheinlogin 自有 LoadConfig 在各自 owner | 全部 KEEP，不改 runtime/CLI/debug/injection |

当前已校验路径为 `文件或 bytes → newViper/BuildConfig → loadWithViper → ValidateConfigWithError → ValidateConfig → 六组规则`；`WithoutValidation` 沿原明确用途保持，不偷偷补校验或复活旧部分校验入口。独立 SHEIN 仍为 `cmd/shein-login-worker → app/runtime/sheinloginworker → sheinlogin/bootstrap/worker`，持久化/Store 同步归原 sheinlogin/storecenter，不经旧 Validator 对象。

## 4. owner、持久化与副作用边界

| 边界 | 原 owner/不变量 |
| --- | --- |
| Config/Viper/加载 | core/config 保有类型、schema、defaults、known env、scoped .env 和原文件选择；本批无键值/格式/优先级变化，无新来源或第二 Config |
| 聚合校验 | ValidateConfig 原 owner；将四个调用直接放入原函数，依序 OpenAI→Browser→Amazon→Platforms，再保留原 ListingKit→Workbench；累加全部错误，不短路，var nil slice/append 行为保持 |
| 规则与错误 | 各 Validate*Config、ValidationError、ValidateConfigWithError、formatValidationErrors/classifyValidationError/moduleFromField/validationModuleRank 原实现保持；不改错误字段、hint、排序、General 分支或输出字符串 |
| 旧加载门面 | 曾有 Stat/文件读/日志行为但没有当前 caller；删除不授权读取实际路径、env、日志或凭据，也不新增检查入口 |
| 当前业务 | 身份/组织授权、计费、采集、图像/资产、schema 初始化、Store、独立 SHEIN 登录/同步、fence/UNKNOWN 均保原 owner；无表/迁移/缓存/文件/进程操作 |

原缺字段/disabled/deleted/错误身份/撤权/超时/失败/响应丢失/重试/重启/取消/并发处理不因本批改变；无新状态、事务、恢复或自动接管。无当前凭据或配置搬运，不能从源码缺席推断真实实例/历史配置已退出。若发现当前外部可观察义务，报告用户，不增加临时 fallback、适配器或迁移。

## 5. 删除、保留与选择性抽取矩阵

### 5.1 四个生产工件

| 文件 | 精确 RETIRE / EXTRACT / 调用转换 | KEEP |
| --- | --- | --- |
| internal/core/config/loader.go | RETIRE LoadConfigWithFallback、LoadAmazonCrawlerAPIConfigWithFallback、validateAmazonCrawlerAPIConfig 与专属 os/strings/logrus imports/comment | LoadFromBytes、NewDefaultConfig 完整实现/值及 bytes/fmt imports；不改其它当前入口 |
| internal/core/config/validator_validator.go | 整文件 RETIRE：Validator、NewValidator、Validator.Validate/ValidateAndLog/ValidateWithError 及专属 imports | 四个有效规则函数在原文件保留；仅四项聚合行为按下一行抽回原 owner |
| internal/core/config/validator.go | RETIRE Config.Validate/ValidateAndLog/ValidateWithError、ValidateConfigAndLog、专属 logrus import；EXTRACT Validator.Validate 的四个顺序 append 调用到既有 ValidateConfig，改用同一 c.OpenAI/Browser/Amazon/Platforms 地址 | ValidateConfig 原后两项 ListingKit/Workbench 调用与顺序、全错误聚合；ValidateConfigWithError 及全部格式/分类/rank helper 完整实现不变 |
| internal/core/config/config.go | 仅 loadWithViper 中 cfg.ValidateWithError()→ValidateConfigWithError(cfg) | rejectRetiredProductRuntimeConfig→BuildConfig→校验→错误/返回流程原样；其余文件/bytes/WithoutValidation/env/路径/默认/日志/Config 声明逐字不动 |

不搬包、不建新函数 owner/接口/注册器；有效函数已存在。不是全仓 `Validator`、`Config`、`Validate`、`ValidateWithError` 禁词，别的领域、SDK、当前 manifest 或独立 worker 的同名声明/调用不变。没有重做浏览器爬虫、平台 API、登录或 Store 同步。

### 5.2 原调用尾部与直接旧依据

四个原测试文件只作 §6.1 指定的六处调用替换及一个聚合行为回归，另有 config.go 一处生产调用转换；原业务/安全断言、fixture 和测试数量不以删除门面为由减少。旧加载入口没有原专属测试文件，不制造空壳或迁移用替身。

仅两份旧依据 `legacy-runtime-config-closure-retirement-boundary.md`、`legacy-host-config-featureflag-tracing-retirement-boundary.md` 在实施时加窄顶注，准确替代 §2 对象，原全部正文逐字后缀保留。当前设计/README/Legacy Register 只维护本批状态和入口，历史计数不改。

## 6. 实施顺序与必要验证

设计获用户确认后，在同一 PR 先写 Reuse Existing Architecture / Ready 并读回；扩展 §6.1 原退休 guard，在旧声明仍存在时确认准确 RED。先在原测试文件保留六组聚合行为的回归，确认旧实现的有效合同，再抽回四项聚合、删除外壳/两旧入口、切换全部现役和原测试调用、补旧 KEEP 顶注至 GREEN。不伪造“原规则有缺陷”的 RED，不复制字段规则或开发新测试框架。

必要开发自检：原退休 guard、原 Architecture README 四项检查；四个受影响原测试中的配置/身份/Workbench/默认及原 validator_format_test.go、字段/依赖校验适用用例；现役独立 SHEIN/Store 适用原合成 fixture。只跑直接相关选定测试，完整 CI 留稳定候选，不重复未变代码已绑定的有效证据。

开发只使用任务自有临时目录、合成配置/env、原内存/fake/loopback fixture，不新建 PG/Redis/容器/浏览器/provider 环境。不能让 LoadConfig/.env 搜索读取现有配置/凭据；加载类用例仅在其原 fixture 能隔离的任务自有 cwd/合成输入下执行，不能保证者本地 NOT_RUN，交原 CI 隔离环境；不因此扩建 runner、修改表外测试或增加试用前置。不执行旧入口/constructor、当前程序、schema/CLI 或真实请求。

保留七入口仅编译：current-application、image-agent-temporal-worker、shein-login-worker、product-listing-api-schema-migrate、1688-local-agent、1688-batch-import、hack/debug 独立 module test-amazon；不执行产物。准入、开发自检、PR CI、最终只读复核、运行/产品验收和未来 merge/main CI 分开记，不把编译/源码缺席签成可上线。

### 6.1 原检查转换与十四工件准确边界

确认后同批实施 **14 工件**：四生产文件 + 四原测试文件 + 一个原退休 guard + 两旧 KEEP 文档 + 本文/Architecture README/Legacy Register 三当前文档。设计阶段仅后三文档；用户确认及 Ready 读回后，已按下表同批完成源码与测试转换，原规则和表外检查不变。

| 工件/原函数 | 唯一允许变更 | 必须保持 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go / TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired | 增加 validator_validator.go fail-closed Lstat 缺席；沿原 owner-qualified TypeSpec/FuncDecl/ValueSpec 逻辑，仅对 core/config 增加 §5.1 十二项退休声明/方法；正向保留当前配置入口、六组规则、ValidateConfig/ValidateConfigWithError、ValidationError/格式 helper | 原 kernel/HTTP/config/日志/timeout/goroutine/jsonx/appenv/httpclient/imagex 全部旧禁入/保留、字段/decoder imports、解析/读取失败不 skip；只此函数变化，其余全文件逐字不变 |
| internal/core/config/config_test.go / TestCurrentConfigDoesNotRequireRetiredTaskInputs、TestConfigValidation | 三处 cfg/validConfig/invalidConfig.Validate()→ValidateConfig(同一对象) | 原有效/非法配置、已退休输入不要求、结果为空/非空及错误数量断言与其它测试逐字不变 |
| internal/core/config/validator_dependencies_test.go / TestValidateConfig_CatchesDependencyErrors | 一处 cfg.Validate()→ValidateConfig(cfg)；同文件新增一个 TestValidateConfigPreservesValidationGroupOrder | 原 OpenAI client完整性、SP-API凭据拒绝和聚合拒绝断言/fixture不减。新回归用同一合成非法配置触发六组现有规则，按已存在的错误字段定位各组，验证六组均存在且顺序 OpenAI/Browser/Amazon/1688/obsolete allowlist/Workbench 保持；不重写规则、固定 map 内部迭代顺序或包含实际凭据 |
| internal/core/config/validator_listingkit_test.go / TestValidateConfigRejectsLegacyListingKitUsernameAllowlistWithoutExposingValues | 一处 cfg.ValidateWithError()→ValidateConfigWithError(cfg) | 拒绝 obsolete username allowlist 与不暴露合成敏感值断言、原 YAML/env fail-closed 测试、fixture和其它内容逐字不变 |
| internal/core/config/validator_workbench_test.go / TestConfigValidateWithErrorIncludesWorkbenchZitadelConfig | 一处 cfg.ValidateWithError()→ValidateConfigWithError(cfg)，不改测试名 | 原 enabled/disabled、完整输入、issuer/authorization URL/client secret/projectID 及错误断言、fixture和其它内容逐字不变 |
| §5.2 两旧依据 | 仅添加准确 superseding 顶注/本文链接 | 原全部正文精确后缀，不改其它 KEEP/历史/勾选/计数 |

根 guard 禁入集合按准确 owner/receiver 区分：`LoadConfigWithFallback`、`LoadAmazonCrawlerAPIConfigWithFallback`、`validateAmazonCrawlerAPIConfig`、`Validator`、`NewValidator`、`Validator.Validate`、`Validator.ValidateAndLog`、`Validator.ValidateWithError`、`Config.Validate`、`Config.ValidateAndLog`、`Config.ValidateWithError`、`ValidateConfigAndLog`。不禁止包外同名对象或删除检查换绿。原其它检查、validator_contract_boundary_test.go、validator_format_test.go 与字段规则源/测试均零改。

**workflow/完整 inventory/CURRENT/历史/计数/ceilings/依赖/锁/保护规则及 inventory 检查逻辑零改**。core/config 仍从 config.go 消费 platform/logging、从类型消费 integration/openai，四条 CURRENT 关系没有失效，不申请校准行。若原 CI 实证出现表外孤立尾部或冻结 KEEP 冲突，先列具体失败、真实消费者及准确最小修订，取得用户决定后再应用；不能借完整闭包扩大到其它配置/检查。

## 7. 交接、复核与停止条件

### 7.1 设计准入

当前仅设计/两个索引。完成适用原文档检查、准确设计 HEAD 原 CI 与一次适用独立只读设计复核，记录真实审核 SHA/结论/finding 分类，交用户确认 §1–§7（含 §6.1）；用户确认并 Ready 读回前，不改源码/测试/样例，不以文档 N/A 授权生产修改。UI/IA/Figma 不涉及，不另开全局产品架构。

### 7.2 实施与最终交付

确认后单 Writer 在同一主要 PR 连续实现、必要自检、提交/推送及范围内 finding/CI 修复，不机械拆批。先分类 finding；只有真实 BLOCKER 重开设计，非阻塞完善按原规则收敛。最终一次只读检查完整实际 diff、调用/兄弟路径、规则/格式/身份拒绝和 §6.1/历史边界；修复只复核增量，不重复未变证据。不以 CI/AI 代替用户验收。

### 7.3 评审与权限

同一 HEAD 的 GitHub 复核运行时不重复触发；仅明确启动失败后一次受控重试，空 reviews/Completed 标签或摘要布尔值不冒充实际 PASS。**未经用户本轮授权不默认新增 AI 互验线程**；#915 设计/源码子代理授权均不覆盖本轮。备用设计/源码最终复核须列准确 HEAD、只读范围和原失败证据，先获用户决定，不自行启动 Reviewer/Writer。

设计确认、源码实施、合并、部署及运行/产品验收各自记录。本轮用户已确认冻结设计并授权范围内源码/测试实施、隔离必要自检、提交/推送与同一 PR 维护；合并/部署/关单与既有实例/数据/凭据/进程操作 NOT_AUTHORIZED，源码最终复核子代理须另获授权；运行/产品验收 NOT_RUN。#301 保持 OPEN，无新试用前置。未来获准确 HEAD 正常合并授权后，仅以实际 merge SHA 的 event=push/head_branch=main 整 run completed/SUCCESS确认，不能用PR CI、单独Gate或其它main提交替代。
