# #301 旧 OpenAI 调用适配整链退休设计

> 状态：**Independent Architecture / DESIGNING / SOURCE_NOT_STARTED / NOT_READY**。
> 用户已授权先设计三组完整闭包；当前 PR **只改本设计、architecture README、legacy register 三文档**。
> 调查基线：`d044142bd9a557eac32199eacc835cd7b101736a`（含 #948、#951），接单与前状态存档：#301 评论 6397009772。
> 不以设计 CI 代替源码复核；适用独立设计复核、用户确认 §1–§7（含 §6.1）及 Issue/PR `IMPLEMENTATION_READY / Ready` 读回后，才在同一主要 PR 实施。

## 1. 用户结果、范围与准入

给维护当前 AI、图像与商品路径的开发者，移除已经退出生产装配的第二套文本调用面和便利入口，保留正在使用的事实 owner、有效规则和安全边界。死入口不被包装成正在阻塞用户操作的故障，也不成为内部试用的新前置。

一句话范围：**旧文本传输 + 未装配整商品富化适配器 + 旧客户端/身份便利入口，一次收齐专属类型、默认选择、调用、测试与文档尾部**，不机械拆批。预计同一 **28 工件**：7 生产、11 测试、1 原聚合、6 旧依据和3当前文档；拟 RETIRE **35 生产函数、13 类型、2 混合字段**。没有新增生产 EXTRACT：仍有效的能力已由现役 owner 承接，不复制旧 prompt、64 KiB 富化预算或旧 HTTP seam。

设计采用 Independent Architecture：需要明确退休旧 provider 调用合同与公开内部 ManagerConfig 默认选择面，校准前批 KEEP，同时证明当前组织凭据/精确路由/发送后 UNKNOWN 与共享图像合同不被改变。不是以文件数拆批或要求重做全局架构。

**Must**：完整退出而无兼容层；当前 Eino/受控文本、图像/审核/source edit、组织隔离、JSON/有效规则、采集/PoolManager deadline取消、独立 SHEIN 登录与 Store 同步保持。**Out of Scope**：任何新 AI 调用、供应商接入、UI/Figma/IA、规则调整、ledger/凭据/schema/迁移、重试/恢复、镜像构建/部署、既有实例/数据/凭据/进程操作。workflow、完整 inventory（CURRENT/历史/计数/ceilings/逻辑）、依赖和保护零改。#301 OPEN，无新增试用前置，无新增 Accepted Risk。

## 2. Design Basis 与原 KEEP 的准确替代

- [Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[登记](../refactoring/legacy-register.md)：只有 EXTRACT / RETIRE，不迁移旧数据、不加 adapter/fallback/双读写/第二事实源。
- [当前 Agent 文本供应商合同](product-agent-text-provider-neutral-v1.md)、[#588 组合依据](product-agent-google-interactions-v1.md#2026-10-05-当前-588-组合依据)、[共享 Chat/Task 组合 §6.7](ai-workbench-chat-business-task-v1.md#67-google-interactions-composition-increment)：当前文本由 `integration/agent/einomodel` 和 `integration/aicapability/einomodel` 承接；Manager.CompleteText 已无生产消费者。旧 Google facade 已退休，本批再收齐剩余兼容协议 Manager 文本 seam，不退当前 `integration/googleinteractions`。
- [Product Agent 运行合同](product-agent-runtime-contract.md) §2 仍列旧文本传输与 ProductEnrichmentAdapter 两行。本批准确替代这两行的“当前能力”含义；旧实现改为基线 Git permalink 历史证据，当前 owner 改指受控执行器/组件和现役 Candidate 校验。其它行、Must、状态、权限和原数字不变。
- [#893 Logger 设计](legacy-logger-facade-closure-retirement-boundary.md)和[#934 adapter/helper 设计](legacy-idle-adapter-helper-closure-retirement-boundary.md)对 OpenAI Manager/Client 的保留，仅在 §5 的闲置方法/类型/默认选择范围被替代；真实 Client/Manager/RequestPool/BaseClient、Logger 注入、SDK、当前轮转/队列/关闭保持。
- [#923 Prompt 设计](legacy-tenant-prompt-closure-retirement-boundary.md)对其它 owner 的 TenantID/WithTenantID 等不变要求，仅准确替代 `integration/openai.WithTenantID/WithUserID` 两便利函数；`Identity` 两字段、`WithIdentity/IdentityFromContext` 与共享 `aiidentity` 全合同原样。

用户已批准的当前产品需求和上列现役合同优先；旧源码/旧测试不能单独要求旧入口继续可用。产品/Figma authority 为现有产品决定；本批无 UI/IA/交互变化，Figma 增量 N/A。外部 SDK 继续复用现有 Eino/go-openai/Google 实现，不引入新库或手写替代传输；无需为源码退休额外查验或调用供应商。

## 3. 实际消费者：合同 → 实现 → 注入 → 使用者

| 链 | 实际状态与决定 | 当前 owner / 保留证据 |
| --- | --- | --- |
| Manager.CompleteText / ResolveTextRoute / 旧 bounded HTTP | 生产根只有本文件内部调用；外部只剩两旧测试文件。RETIRE整链，不改造成新入口 | `app/httpapi/product_agent_title_runtime.go` 注入 `governed.Executor` 到 `texteino.NewAgentTextModel`，并绑定 fresh identity / ledger / admission；不是旧 Manager 文本 seam |
| NewProductEnrichmentAdapter → Generate → prompt/预算 walker/严格 JSON | 无生产构造/注入；类型断言只证明满足 CandidateGenerator。RETIRE整个 adapter 和专属 DTO/helper/test | `product/enrichment` 仍被 agent、review、contentagent、aiworkbench、agentconfig、acquisition/persistence 消费；该**领域包不退**。当前候选/证据/策略/JSON校验仍在原 owner |
| GetClient/GetDefaultClient → contextualChatClient | 唯一构造该聊天代理的两入口均无生产调用；默认选择只服务 GetDefaultClient。RETIRE代理、七Manager方法、默认字段/选择块与赋值 | 当前 ImageAgent Review 在 `worker/imageagent/capabilities.go` 先 ResolveEffectiveClientRoute，再 GetClientWithRoute；返回真实 Client，不经该代理 |
| Client.CreateChatCompletionWithTimeout | 无生产调用，专属 time import 失去用途。RETIRE | Client.CreateChatCompletion、Generate/AnalyzeImage/GetDefaultModel、pool/SDK当前合同保持；不把删除包装方法变成删除真实超时/队列 |
| WithTenantID/WithUserID | 无生产调用，WithTenantID只剩合成测试。RETIRE | 当前 WithIdentity/IdentityFromContext 原样；测试改为同一 Identity 构造，不改变 tenant/user 优先级或授权 |
| named image / source edit / organization resolver | 有真实 worker 消费。KEEP | GetImageClient/contextualImageClient、GetClientWithRoute、ResolveEffectiveClientRoute、resolveClient/resolveClientWithSelection、当前 hash/version/cache、GormCredentialResolver 全链 |

前述消费调查覆盖已追踪 Go 源、测试和独立 hack/debug 模块，排除注释/文档/测试自调用冒充生产装配。未来实施前复核同根 sibling 和最新 main 的真实组合，不因未引用就顺带删除其它新产品代码。

## 4. Owner、JSON、持久化及失败边界

### 4.1 共享合同保持

`internal/ai` 的 TextChatCompleter/ChatCompleter 仍服务图像审核；Client.Generate 等方法虽无独立业务调用，仍是当前接口的实现义务，**不退、不收窄接口、不调整 JSON/alias**。不把“文本链退休”扩大为删除当前整个 openai 包、所有静态配置或全部 chat transport。

Manager 的 `clients/dynamicClients/configResolver/logger/mu`、ManagerConfig 的 `Clients/ConfigResolver/Logger(json:"-")`、NewManager 的 nil/空Clients拒绝、同一 clone/logger/每个 named client 构造保持。只去掉默认选中指针、DefaultClient字段和专属选择/错误分支；它不是当前 named image/审核路由的选择依据。`normalizeClientName` 的空名→default、实际显式路由选择、resolver/static优先级及静态Client配置仍被当前图像链使用，原样保留，不新增任何旧设计fallback。

### 4.2 持久化与 wire 所有权

| 对象 | 本批决定 | 不改变的 owner / 边界 |
| --- | --- | --- |
| 旧 TextCompletionRequest/Result/TextRouteDetails、TextInvocationRequest/TextInvoker | 只服务未装配的旧请求/响应桥；不是当前 HTTP/DB/缓存公开合同，RETIRE | 当前 Eino/受控请求、原始usage存在性、路由/配置版本、no-send/UNKNOWN合同仍由原执行器/组件拥有 |
| ProductEnrichmentAdapter 专属预算类型/64 KiB constants/错误/反射walker | 退出旧 prompt格式/解析政策；不移入现役 Agent，也不宣称旧值是当前规范 | Candidate/GenerationRequest/PolicySnapshot、Catalog/Review/来源证据、当前 JSON/纯校验原样，不删除已有候选/历史 |
| Manager.defaultClient / ManagerConfig.DefaultClient | 内存指针及构造参数，无当前序列化/DB写入/恢复消费者；RETIRE | 保留每个显式 named client、客户端路由/有效配置hash/动态缓存与组织凭据行；不扫内存或已有记录 |
| Identity 与 credential resolver | 仅退两便利setter，KEEP完整当前合同 | 组织/成员凭据及配置版本、tenant/user优先级、组织only拒绝成员/全局借用保持；不访问实际数据库/secret |
| Invocation/Commercial/Knowledge/Agent/Review、SHEIN/Store | 当前 durable business owner，KEEP | claim/fence/UNKNOWN、账务/预留/结算、显式Apply、Store固定同步/独立登录无改动 |

无 schema/迁移/SQL/Redis/文件持久化修改，无跨库新一致性问题，无 outbox/Saga/新事务/状态机/计费/恢复平台。旧函数删源码不执行 HTTP、Close、登录、同步或数据删除；既有实例/记录/凭据/进程状态 UNKNOWN 且不读不动。

### 4.3 安全与失败语义

当前 `governed.Executor` 的授权、请求排队、最终gate、唯一dispatch、响应丢失与UNKNOWN由原 owner 负责，未发送的 durable事实和已交接后禁止重发保持。当前组件独立限制请求/响应及2分钟最大timeout，route验证1..120秒；Knowledge.DispatchPermitLease仍5分30秒。旧Manager seam的5分钟比较测试只约束旧入口，退出不取消当前更窄约束或知识租约 owner；不复制旧transport证明来替代当前执行证据。

当前原测试 `TestExecutorFinalGateRechecksDeadlineAfterSlowAuthorization`、`TestExecutorPreservesUnknownClaimInsteadOfInventingNoDispatch`、`TestPinnedComponentsCannotRetryOrRedirectAfterHandoff`、`TestOrganizationRouteNeverBorrowsMemberCredential` 等在原 owner 原样保持。无新增 Threat Model/Accepted Risk，无额外真实provider探针或试用门禁。

## 5. 完整删除 / 保留矩阵（28 工件，同一主要 PR）

Legacy decision: **RETIRE**。Reusable behavior: 由既有当前owner承接的有效文本/图像/凭据/候选行为原样复用。Current owner: §3–§4。Cutover/deletion condition: 适用独立设计复核+用户确认+IMPLEMENTATION_READY读回后，完整实施与§6/§7交付；不新增EXTRACT/helper/兼容层。

### 5.1 七个生产工件：35函数、13类型、2混合字段

| 工件 | 准确 RETIRE / 混合处置 | KEEP |
| --- | --- | --- |
| `internal/integration/openai/text_completion.go` | 整文件：Manager.UsesOrganizationCredentials/UsesOrganizationOnlyCredentials/ResolveTextRoute/ResolveTextRouteDetails/resolveTextConfiguration/CompleteText、completeTextOnce、boundedTextHTTPClient.Do共8函数；TextCompletionRequest/TextCompletionResult/TextRouteDetails/boundedTextHTTPClient四类型、2限额常量、3错误及专属imports | 当前受控执行器/组件、原供应商协议及SDK不在本文件，全部保持 |
| `internal/integration/openai/product_enrichment_adapter.go` | 整文件：NewProductEnrichmentAdapter、ProductEnrichmentAdapter.Generate、canonicalContextError/isNilTextInvoker/canonicalRequestedFields、buildProductEnrichmentPrompt/buildProductEnrichmentPromptWithMarshal、inspectProductEnrichmentPromptBudget、walker.visit/visitReference、parseProductEnrichmentResponse、budgetError.Error共12函数；8类型、专属常量/变量/interface assertion/imports | product/enrichment整个领域及现役Candidate/JSON/规则不动 |
| `internal/integration/openai/client_manager.go` | GetClient/GetDefaultClient/HasConfigResolver/RegisterClient/defaultClientNameLocked/resolveStaticClient/ListClients共7函数；Manager.defaultClient、ManagerConfig.DefaultClient两字段；NewManager仅去“设置默认客户端”整个专属选择/错误块 | 所有其它类型/字段/函数及NewManager其余字节，特别是Logger注入、named clients、route/version/rotation/cache/Close |
| `internal/integration/openai/contextual_client.go` | contextualChatClient类型及resolve/CreateChatCompletion/Generate/AnalyzeImage/GetDefaultModel五方法 | contextualImageClient整个类型/8方法与imports原样 |
| `internal/integration/openai/client.go` | 仅Client.CreateChatCompletionWithTimeout及失去用途的time import | 其余全部方法/类型/字段/ptr设置/请求池/SDK/日志原样，Generate不是本批旧接口 |
| `internal/integration/openai/identity.go` | 仅WithTenantID/WithUserID | Identity两字段、WithIdentity/IdentityFromContext、strings/context/aiidentity imports原样 |
| `internal/app/worker/imageagent/dependencies.go` | 仅buildImageAgentWorkerAI内ManagerConfig literal删除 `DefaultClient: "default"` 一项 | Clients/Logger原表达式、NewManager错误处理、resolver注入、worker全部其余字节 |

### 5.2 十一个测试工件

| 工件 | 精确转换或退出 |
| --- | --- |
| `internal/integration/openai/text_completion_test.go` | 整退仅旧文本seam的测试和textTestManager/textTestRequest/requireNoErrorText专属fixtures；不得退当前受控执行器/组件的原测试 |
| `internal/integration/openai/text_knowledge_lease_test.go` | 整退旧Manager route与5分钟上限的唯一专属测试；当前2分钟cap、knowledge租约、deadline/取消在原owner保持并验证，无新测试平台 |
| `internal/integration/openai/product_enrichment_adapter_test.go` | 整退旧adapter/prompt预算/wire oracle/反射walker/严格旧JSON的全部专属测试、嵌入fixture与mutator；当前Candidate/Review/Eino有效规则和测试保持 |
| `internal/integration/openai/tenant_manager_test.go` | 退TestManagerReportsResolverWiring（只证明退休报告方法）；TestManagerRoutesChatClientByTenantOrUserContext用原三个context分别调用ResolveEffectiveClientRoute→GetClientWithRoute→同一callChat，保持三组原Auth/Model/数量精确断言；去该fixture一DefaultClient赋值；三WithTenantID(Background,id)改WithIdentity(Background,Identity{TenantID:id})。其余测试/fixtures/helper原样 |
| `internal/integration/openai/effective_configuration_test.go` | 五函数各去一个DefaultClient赋值；两WithTenantID改同一WithIdentity。版本hash不含secret、配置漂移/空版本拒绝、DB override/真正不可用、原fixture/错误/调用次数全部保持 |
| `internal/integration/openai/credential_store_test.go` | TestGormCredentialResolverPrefersUserThenTenantConfig两处、TestOrganizationOnlyCredentialResolverNeverUsesMemberOrGlobalCredential一处WithTenantID改同一WithIdentity；DB fixture/tenant-user优先级/组织only拒绝成员与全局、其它断言逐字保持 |
| `internal/integration/openai/logger_test.go` | 仅TestManagerPropagatesLoggerToResolverCreatedDynamicClient的一WithTenantID改同一WithIdentity，日志隔离/不变输入/原断言全保留 |
| `internal/app/worker/imageagent/capabilities_test.go` | TestBuildProductionImageCapabilitiesQuotesCurrentRouteAndRejectsStaleAuthorization、TestProductionSourceEditDoesNotResolveReviewRoute各只去一DefaultClient赋值 |
| `internal/app/worker/imageagent/organization_main_temporal_postgres_test.go` | TestOrganizationWorkerRealTemporalUnpricedGenerationDoesNotReserveOrDispatch只去一DefaultClient赋值，全部真实Temporal/PG安全断言原样 |
| `tests/imageagent_review_invocation_integration_test.go` | helper review334Manager只去一DefaultClient赋值，全部审核/计量/发送/重复/UNKNOWN原断言不动 |
| `tests/imageagent_organization_integration_test.go` | TestOrganizationCredentialAdmissionChecksVerifiedScopeBeforeResolver、scope339Fixture.verifyActivity各只去一DefaultClient赋值，全部身份/组织/持久化隔离断言不动 |

八混合测试合计：**12个DefaultClient赋值、9个WithTenantID改接**；连同唯一生产赋值共13个默认选择写入退出。不是全局删除http.DefaultClient或其它包同名函数，不新增测试helper或公共运行seam。

### 5.3 原聚合与九文档

| 工件 | 准确变动 |
| --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 只新增TestLegacyOpenAIAdapterClosureIsRetired；所有原函数/helper/import/断言逐字不变，见§6.1 |
| `docs/architecture/product-agent-runtime-contract.md` | 只改§2旧“文本传输”与“ProductEnrichmentAdapter”两表行：旧源链接改为基线d044142bd的GitHub blob permalink并标RETIRE历史证据，当前owner指受控执行器/组件与现役Candidate校验；其它字节不变。避免保留指向已删源文件的当前相对链接，不篡改其它历史段落/数字 |
| `docs/architecture/product-agent-text-provider-neutral-v1.md` | 只前置当前执行器替代Manager文本seam的窄注；原完整blob后缀/协议安全规范保持 |
| `docs/architecture/product-agent-google-interactions-v1.md` | 只前置本批退剩余兼容协议Manager入口的窄注；当前Google SDK/受控组合及原全文后缀保持 |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 只前置#893准确闲置方法/默认选择替代，当前logger/Manager/Client保持，原完整blob后缀 |
| `docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md` | 只前置#934对当前OpenAI core KEEP的准确雕刻，不退RequestPool/BaseClient/Logger/真实重试，原完整blob后缀 |
| `docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md` | 只前置#923其它owner KEEP中本包两便利函数的窄替代，字段/完整Identity/JSON保留，原完整blob后缀 |
| `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md` | 当前设计，未来实施仅前置准确阶段/证据，原确认设计完整后缀 |
| `docs/architecture/README.md` | 当前设计入口/阶段；原完整索引后缀 |
| `docs/refactoring/legacy-register.md` | 当前三组完整范围/owner/阶段；原完整历史/计数/勾选后缀 |

旧Phase3计划只作有日期历史，不是保活旧adapter的当前authority；已声明旧Manager无消费者的Chat/Task组合文档无需再改。范围外源码/文档/测试不借本设计顺带清理。

## 6. 验证、验收与原门禁

**当前设计阶段**仅三文档：核对§1–§7/§6.1、35函数/13类型/2字段、28工件、消费者/前批KEEP、路径链接、原README/register完整blob后缀、diff --check及原文档/consumer检查。作者不跑业务测试/构造器/编译应用；未改原workflow的准确设计HEAD CI另列，不等于真实环境或用户验收。

未来用户确认和准入读回后：同一Writer/分支/PR形成原新增guard的真实RED→整批RETIRE与原测试改接→GREEN，运行与变更风险直接相关的原测试/导入/文档检查及原PR CI；不新增runner/故障矩阵/验证工具的验证工具。

| 当前Must | 适用证据 |
| --- | --- |
| 完整闭包 / 无旧调用入口 | 七生产工件的函数/receiver/type/field/构造/赋值/import及全部已追踪引用，2整生产文件/3整测试文件缺席；不使用空占位或missing-skip |
| 当前安全与实际行为保持 | 所有存活生产函数除NewManager准确选择块、buildImageAgentWorkerAI一字段外原字节；当前文本/route/SDK、image/审核/source edit、身份/有效规则/JSON与SHEIN/Store owner保持；§5.2有效断言改接后仍执行 |
| 组织/配置/日志隔离 | 原tenant_manager/effective_configuration/credential_store/logger、worker capabilities和两imageagent集成测试的适用原测试；当前Eino executor/component/route原测试保持，fixture与真实provider执行分列 |
| 当前源码仍能被消费 | 原current-application、shein-login-worker、1688-local-agent、1688-public-browser-collector及hack/debug的test-amazon/test-1688六入口仅编译到任务TEMP，不执行产物 |
| 原门禁 / 历史保持 | 原aggregate除新增一函数逐字保持；八文档完整基线blob后缀、runtime合同两表行外逐字保持；workflow/完整inventory/依赖/保护零diff；准确源码HEAD原PR CI整run结果 |

复用现有测试/合成fixture即可，不需要作者另启PostgreSQL/Redis/Temporal/浏览器。现有真实PG/Temporal测试由原CI按原workflow执行，不把CI改成已有共享实例或真实凭据。运行、平台、产品验收NOT_RUN，部署/合并/关单另授权。

### 6.1 原检查与测试精确边界

1. 原聚合只新增 **TestLegacyOpenAIAdapterClosureIsRetired** 一个函数，复用现有trackedFiles、AST/decoded import、失败关闭；原imports/helpers/其它函数完全不变。按精确包/receiver禁止35函数、13类型、2字段、DefaultClient literal/旧chat代理构造及专属引用；保护实际ManagerConfig.Logger/json:"-"、named client/logger/route/version/cache、contextualImageClient、当前AI/组织only/UNKNOWN/图像/审核/source edit及采集/SHEIN/Store。不得全局按Manager/Client/Generate/DefaultClient/WithTenantID等通用名字误杀。
2. 原11测试文件只准§5.2列出的整退/函数退役、12默认赋值删除、9身份改接以及一个原tenant路由测试改接当前精确route API。原callChat/newCaptureChatServer等helper全保留，不新建测试接口；其它fixtures/断言/函数/import不变，只有实际失去用途的专属import可退出。
3. **完整inventory零改**：包和全部当前外部消费者仍有生产入口，现役worker/core-config/AI clients等incoming import保持，不需CURRENT校准。历史/计数/ceilings/检查逻辑、Code Health与其它原检查零改。
4. 六旧文档中运行合同只准两行准确校正；其它五旧文档和三当前文档按§5.3前置/完整blob后缀处理。历史数字/旧勾选不改；#301只更新当前段，既定#726 marker后完整历史逐字保持且正文≤65536字符。
5. 若原CI暴露遗漏，先查同根sibling与当前Must，提供具体冲突/精确最小补丁，再请用户准确例外；不能把本设计当任意删旧检查/安全断言、放宽算法、改workflow/inventory或扩范围的授权。

## 7. 评审、确认与停止点

1. 设计阶段先完成三文档、必要自检、准确设计HEAD原CI和适用独立Architecture Review；复核需亲读真实消费者/当前owner/§4持久化与默认选择、§5闭包/§6.1，不只采信作者。当前正式源码NOT_READY。
2. 无未解决BLOCKER、适用设计复核与用户确认§1–§7（含§6.1）后，Issue/PR记录并读回IMPLEMENTATION_READY / Ready，再同一PR整批实施28工件。最多两轮正常架构复核；finding先按AGENTS分类，明确当前Must/具体后果/阻塞层级。非Blocker转IMPLEMENTATION_TEST或Backlog，不循环重开架构。
3. 完整源码后一次集中独立最终交付复核；设计复核不替代源码复核，修复只复核对应增量。GitHub启动失败至多唯一重试；**未经用户本轮授权不默认新增AI互验线程**，旧批/设计授权不扩作源码授权，不新增实现者/runner/平台。
4. 开始及最终候选前核对actual main真实增量，不为始终最新反复merge-main。源码准确HEAD的原CI整run与最终复核满足后，交用户另行授权普通expected-head合并；不得管理员绕过/强推/改保护。获准合并后只认实际merge SHA的event=push/main准确整run completed/SUCCESS。
5. #301始终OPEN。当前采集/有效规则、PoolManager deadline取消、独立SHEIN登录/Store同步保持；不操作既有实例/数据/凭据/进程，不改workflow/完整inventory，不增加试用前置。源码完成、main CI、运行/产品验收分列，不自行宣称可上线或用户验收通过；部署/关单另授权。
