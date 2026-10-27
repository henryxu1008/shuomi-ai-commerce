> #301 / #977 用户确认的[完整退休闭包](legacy-imageagent-http-runtime-closure-retirement-boundary.md)已在同一主要PR实施：仅替代两root integration与Chain1图片段必须逐字保活的旧KEEP；按新§4.2/§6.1改typed Service及真实scope授权，原review计量/发送/取消/UNKNOWN、Source/Review/Store/草稿及其它断言保持。 此顶注只校准本批指定旧KEEP，以下实际开工main的完整原正文/批准/失败/数字/勾选逐字保留为历史。无新alias/替代旧宿主、workflow/inventory/运行/数据操作或内部试用前置；独立SHEIN登录与Store同步保留，#301 OPEN。

> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

> #953 获批 §6.1 准确例外（2026-10-25）：用户已批准[精确提案6398981963](https://github.com/qq550723504/shuomi-ai-commerce/pull/953#issuecomment-6398981963)。原 TestIntegrationProviderBoundaryDecodesAllowedAndForbiddenGoImportLiterals 仅将期望及报错两行 2→3；原 raw/escaped fixture、解码/遍历/失败处理、生产白名单和其它断言不变，当前正确拒绝 app/core/旧 enrichment、保留 image。原 CI40202695034 的 Architecture/Backend 同根失败已复现；该最小校准不恢复旧授权，不改 workflow/完整 inventory。以下源码阶段说明和冻结设计全文完整保留。

> #953 源码实施阶段：用户已确认冻结设计 §§1–7（含 §6.1）。旧生成链与 `internal/product/enrichment` 已整批退休；当前候选合同及纯规则归 `internal/product/review/proposal`，Review 仅接现成候选，沿原四个 HTTP 审核路由处理。必要自检、准确源码 CI 与最终复核分别记录在 [PR #953](https://github.com/qq550723504/shuomi-ai-commerce/pull/953)，当前尚未交付合并授权；运行/产品验收 NOT_RUN，#301 保持 OPEN。 以下冻结设计、索引和历史证据完整保留。

# #301 旧生成链与审核候选合同完整退休设计（#953）

> 状态：**Independent Architecture / REDESIGNING / SOURCE_NOT_STARTED / NOT_READY**。
> 本次按用户新决定重做 #953，替代旧“保留整个 product/enrichment 包”的设计。当前只修改本设计、architecture README、legacy register 三文档，**尚未授权实施**。
> 调查基线：`ac54a11fed2ffdb72b3a5e8e2bd5c3575500830b`；本分支已同步 main，接单及前当前段存档：[#301 评论 6397602264](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6397602264)。
> 旧设计 HEAD b380860e9 的 CI/复核仅覆盖旧范围，不构成本次扩大闭包的批准。新设计需适用独立复核、准确设计 HEAD 原 CI、用户确认 §1–§7（含 §6.1）及 IMPLEMENTATION_READY / Ready 读回，之后才实施。

## 1. 这次具体做什么

**删旧生成器，保留当前审核，再把审核仍在用的小合同与纯规则搬到正确 owner，最后删除整个 `product/enrichment` 包。** 同时收齐 #953 原定的旧 OpenAI 调用入口及全部专属尾部，一个主要 PR，不机械拆批。

| 处理 | 用户能理解的结果 |
| --- | --- |
| RETIRE 旧生成链 | 删除固定生成器、旧 OpenAI 整商品适配器、未注册的生成 HTTP 入口和未使用的 1688 输入桥；不把它们搬到测试或换名字保活 |
| EXTRACT 当前审核需要的部分 | 候选、证据、内容选择、确定性校验归到 `internal/product/review/proposal`；它不生成、不发送、不保存、不审核批准 |
| KEEP 当前真实路径 | 当前 Agent 生成候选 → 同一审核 owner 校验/保存 → 现有人工审核/显式 Apply；当前 AI、图像、采集、独立 SHEIN 登录与 Store 同步保持 |
| 收齐尾部 | 当前消费者改接新 owner；有效测试改用当前候选接口；转换旧路径检查及准确文档引用，不留 alias、wrapper 或旧根 |

使用者是维护当前 AI/审核/采集路径的开发者；可见结果是源码只有当前调用路径和事实 owner。旧死入口不是正在阻塞用户操作的故障，本批不增加内部试用前置。未来实施范围为 §5.8 **149 个唯一路径工件**（删除路径与抽取目的路径分别计数，不是拆批目标），其中 18 文档；当前只三文档。

Must：旧根/旧 import/旧生成服务完整退出；当前候选、JSON/hash、证据/有效规则、授权/组织隔离、审核事务/显式 Apply/UNKNOWN 保持。Out of Scope：新生成能力、provider、UI/Figma/IA、新表/迁移/兼容读写、预算/计费/恢复/调度协议、依赖升级、部署、真实环境验收。**workflow、完整 inventory（CURRENT/历史/计数/ceilings/检查逻辑）、依赖与保护零改；不操作既有实例、数据、凭据或进程；#301 保持 OPEN。**

## 2. Design Basis 与前批 KEEP 替代

采用 **Independent Architecture**：本次改变内部领域合同归属，跨 Agent/Review/持久化消费者，且超过文件风险提示线；不因此设计全局新平台。当前 Review 仍是提案/决定/Apply 的事实 owner，抽取叶包不是新业务 owner。

- 产品依据：[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy 登记](../refactoring/legacy-register.md)。只 EXTRACT / RETIRE，无旧包兼容类别、alias、双读写、迁移或第二事实源。
- 当前行为依据：[Product Agent 运行合同](product-agent-runtime-contract.md)、[多字段内容合同](product-content-localization-v1.md)、[Knowledge Context](agent-knowledge-context-v1.md)、[供应商中立文本](product-agent-text-provider-neutral-v1.md)、[Google 组合](product-agent-google-interactions-v1.md)、[共享 Chat/Task 组合 §6.7](ai-workbench-chat-business-task-v1.md#67-google-interactions-composition-increment)。
- #953 原“product/enrichment 整包 KEEP”撤回。前批 #923 的同包 KEEP 和 Phase3 目标映射仅由本设计准确替代其**源码 owner/生成入口**，不撤回当前 JSON、证据、校验或审核事实合同。#827/#904 的严格 JSON、版本与读取拒绝规则原样；#868/#921 的 SourceIdentity 四零值槽位/规范化/hash 原样。
- #893/#934 的 OpenAI Manager/Client KEEP 只按 §5.1 雕刻闲置方法/类型/默认选择；真实 Client/Manager/RequestPool/BaseClient、logger、SDK/当前图像和轮转/关闭保持。
- #923 对 `integration/openai.WithTenantID/WithUserID` 两便利函数的 KEEP 被 §5.1 替代；Identity 两字段、WithIdentity/IdentityFromContext、其它 owner 的身份函数不变。
- #383/#386 的隔离固定生成 HTTP 装配是历史交付证据；当前已注册路径是 Agent Candidate intake + 四条审核路由。用户本轮明确授权设计退休旧构造器/生成分支，不把旧测试解释成继续开放旧入口的产品义务。
- 无用户页面或交互变化；[当前标题审核 UI](../engineering/product-title-review-ui.md)明确不做生成入口，适用 Figma 增量 N/A。沿用现有 Eino/go-openai/Google 与标准 JSON/hash，不引入新库或自制替代传输。

旧 issue/测试/历史 PASS 不自动保活旧包；本设计也不授权改写前批历史数字/勾选或扩大真实运行权限。Accepted Risks 不新增。

## 3. 实际调用链与删除闭包

调查是准确基线的静态源码/装配证据，不代表任何实例已启用或真实调用通过。已追踪 Go、独立 debug、TS/BFF/浏览器夹具及检查一并核对。

| 链路 | 实际消费者与决定 |
| --- | --- |
| 当前入口 → Agent → Review | `httpapi/current_application.go` 装配 ProductAgent；`product_agent_application.go` 使用 **review.NewCandidateService**，执行校验/候选提交。它不需要旧 Proposer |
| 当前规则 | `review/content_intake.go`、`content_validation.go` 调用 ValidateCandidate；Agent/Workbench/配置冻结与 Review 的 Candidate、ContentSelection、Proposal 仍在使用，必须先抽取 |
| 旧固定生成 | 唯一生产 NewProposer/NewService 组合在未注册的 **NewProductReviewApplication**；Review.Service.Create 的 nil-candidate 分支再调用 Propose。该构造器无生产根调用，外部仅夹具；整链 RETIRE |
| 旧生成 HTTP | productReviewRoutes 定义五条，当前 productReviewOnlyRoutes/准入过滤掉根 POST；当前运行只取 GET 集合、GET 详情、POST decisions、POST apply。删根 POST 与专属解码/派发后，直接复用准确四条，不加新入口 |
| 旧来源桥 | a1688.ProductEnrichmentRequest 只由其本文件测试调用，连 clone helper 一起 RETIRE；不迁入新 owner。真实 Sourcing.Normalize/ToSnapshot、原 Amazon 浏览器/调试与当前采集保持 |
| 旧 OpenAI | Manager.CompleteText/旧 route/HTTP seam 外部仅旧测试；ProductEnrichmentAdapter 无生产注入。旧便利 chat代理/default选择/身份setter按 §5.1 退出；当前图像审核显式 route 与治理文本不走它们 |
| 当前存储/hash | Review.Record.Original、ContentPayload/Selection，aiworkbench JSONB selection、agentconfig 冻结快照/fingerprint 均使用旧包类型；**换 Go owner，保持 wire 与全部原计算** |
| UI/BFF | 已只有四条窄审核路由，集合 POST 明确 405；不改 TS/页面。浏览器 Go fixture 曾通过旧 POST 造候选，必须改接真实 Candidate owner，BFF/UI 脚本及其未发送/未知结果规则保持 |

扫描中 74 个 Go 文件出现旧根路径，其中**真正 import 声明 68 文件：31 生产、36测试、1负例 fixture**；另六文件是检查用路径字面量，不能当业务消费者。精确路径见 §5.8；实现时 AST 解码 import，连负例 fixture 一并改接，最终不留旧 Go import。历史/禁入字符串可保留，不能设“跳过 testdata”来凑清零。

## 4. 正确 owner、JSON/hash 与持久化边界

### 4.1 纯叶包与依赖方向

抽取目的：**`internal/product/review/proposal`，package proposal**。现有 Review 拥有审核候选事实与确定性准入；它的共享纯合同放在本域叶包，Agent、contentagent、Workbench、配置/存储消费者引用该叶包。不能直接放到 Review 父包：父包已依赖 Agent，会造成 Agent ↔ Review 循环；contentagent 也依赖 AgentConfig/Review，不适合反向承载共享合同。

叶包只消费现有 Catalog/Sourcing 事实、标准库与已用 `golang.org/x/text/language`。**不依赖 Agent、AgentConfig、Review 父包、app/integration/platform、数据库/SDK，不提供生成、Repository、queue、retry、provider 或 runtime 语义。** Sourcing/Catalog 和它们的持久化 adapter 不反向依赖此叶包。正常局部 import 名采用 `reviewproposal` 以免被 `proposal` 局部变量遮蔽；这不是兼容别名。禁止把新路径导入为 `enrichment`，禁止旧目录、旧 type alias、旧工厂 wrapper、re-export、转发接口。

### 4.2 最小抽取合同：形状和行为保持

| 抽取 | 必须保留 |
| --- | --- |
| model.go 十种类型 | Request、PolicySnapshot、Proposal、Candidate、FieldChange、Evidence、QualityScore、ValidationResult、Warning、Rejection；再加 content_selection.go 的 ContentSelection 共十一种。字段名/顺序/类型/tag/零值、时间和 map/slice/pointer语义全部原样；不是重建 DTO |
| 内容选择 | ContentPolicyVersion 原值；NewContentSelection、Absent/Valid/Fields/TitleOnly、语言规范化和字段顺序原样 |
| 纯校验 | ValidateCandidate 的 nil/已取消ctx、cloneRequest→validateRequest→cloneCandidate→validateEvidence→scoreCandidate→稳定诊断→validateProposal顺序不变；原证据 ID 优先级、允许/必需字段、质量阈值、warnings/rejections排序去重、深拷贝/无别名原样 |
| 错误 | 仅五个仍被纯规则使用的 sentinel：ErrInputInvalid/ErrEvidenceInsufficient/ErrCapabilityUnsupported/ErrPolicyRejected/ErrOutputValidation，**原错误字符串保持**。字符串含 enrichment 不等于保留旧 Go owner；provider-only ErrExternalCapabilityUnavailable随生成链退休 |
| 非抽取 | CandidateGenerator、GenerationRequest、Proposer/Dependencies/proposer、NewProposer/isNilCandidateGenerator/stableGenerationError/Propose、旧 prompt/64KiB预算/反射 walker/传输和来源桥全部 RETIRE |

这里的“最小”是当前消费者加其确定性传递闭包；不删仍参与规则的质量/诊断字段，也不因为当前 policy 阈值恰好为零而简化纯函数。不新增规则、取消点、字段容错或标准化步骤。

### 4.3 wire、指纹和事实 owner

| 边界 | 原合同必须保持 |
| --- | --- |
| Review JSON | Record.Original / ContentPayload、CandidateInput、CreateInput、View与严格解码均不改shape。**review.CreateInput仍是当前Base/Record.Input，不按名字误删**。原字段大小写、JSON标签、null/空集合/omitzero/omitempty、拒绝未知/重复/坏Unicode保持 |
| Hash | Review.operation 的 kind=`create_candidate`、payload形状/顺序；proposalDigest 中匿名结构的字段/类型形状与 json.Marshal→sha256；Agent/Workbench/AgentConfig payload和snapshot fingerprint规范化/版本/hash原算法和原输入保持。只换Go路径，不把type名字写入hash、不加version/fallback |
| 持久化 | Review proposal/operation、Catalog publication/head、Sourcing receipt、Workbench任务与AgentConfig冻结快照仍由原仓库/UoW拥有。aiworkbench.ContentSelection 的 pointer/jsonb/serializer标签不变；无schema/表/列/索引/迁移/历史数据访问 |
| 权限/外部副作用 | Candidate提交的fresh identity、组织/source授权与二次事务内校验、ContextProvenance、execution gate、transactionAttempt、replay/CAS/Apply保持。当前治理文本唯一dispatch/no-send/发送后UNKNOWN、ledger/计费/Knowledge租约由原owner负责 |
| 其它当前能力 | 当前AI/图像/规则/采集/PoolManager deadline取消、严格草稿JSON/SourceIdentity四槽位、独立SHEIN登录与Store同步保持，不触发它们运行 |

当前治理文本/组件2分钟最大timeout、route的1..120秒与Knowledge.DispatchPermitLease的5分30秒保持；退旧5分钟Manager测试不取消当前边界。既有资源状态 UNKNOWN，不读不动；无需新outbox/Saga/恢复平台/影子owner或数据迁移。新安装和当前空数据基线不自动授权访问旧实例。

## 5. 完整 EXTRACT / RETIRE / KEEP 矩阵

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior: §4的当前最小候选合同与纯规则。Current owner: Review/proposal纯叶包；Review/Catalog/Sourcing及当前执行器仍持有事实/副作用。Cutover/deletion condition: 新设计复核和用户确认后，同一 #953 原子改接全部当前消费者、收齐尾部并删除旧根；无中间双包并存交付，无兼容 alias。

### 5.1 继续收齐旧 OpenAI 闭包：七生产、十二测试工件

| 工件 | 准确 RETIRE / 混合处置 | KEEP |
| --- | --- | --- |
| `internal/integration/openai/text_completion.go` | 整文件：Manager.UsesOrganizationCredentials/UsesOrganizationOnlyCredentials/ResolveTextRoute/ResolveTextRouteDetails/resolveTextConfiguration/CompleteText、completeTextOnce、boundedTextHTTPClient.Do共8函数；TextCompletionRequest/TextCompletionResult/TextRouteDetails/boundedTextHTTPClient四类型、2限额常量、3错误及专属imports | 当前受控执行器/组件、原供应商协议及SDK不在本文件，全部保持 |
| `internal/integration/openai/product_enrichment_adapter.go` | 整文件：NewProductEnrichmentAdapter、ProductEnrichmentAdapter.Generate、canonicalContextError/isNilTextInvoker/canonicalRequestedFields、buildProductEnrichmentPrompt/buildProductEnrichmentPromptWithMarshal、inspectProductEnrichmentPromptBudget、walker.visit/visitReference、parseProductEnrichmentResponse、budgetError.Error共12函数；8类型、专属常量/变量/interface assertion/imports | 旧 product/enrichment 根按 §5.3 完全退出；当前 Candidate/JSON/规则按 §4 抽取到 Review/proposal |
| `internal/integration/openai/client_manager.go` | GetClient/GetDefaultClient/HasConfigResolver/RegisterClient/defaultClientNameLocked/resolveStaticClient/ListClients共7函数；Manager.defaultClient、ManagerConfig.DefaultClient两字段；NewManager仅去“设置默认客户端”整个专属选择/错误块 | 所有其它类型/字段/函数及NewManager其余字节，特别是Logger注入、named clients、route/version/rotation/cache/Close |
| `internal/integration/openai/contextual_client.go` | contextualChatClient类型及resolve/CreateChatCompletion/Generate/AnalyzeImage/GetDefaultModel五方法 | contextualImageClient整个类型/8方法与imports原样 |
| `internal/integration/openai/client.go` | 仅Client.CreateChatCompletionWithTimeout及失去用途的time import | 其余全部方法/类型/字段/ptr设置/请求池/SDK/日志原样，Generate不是本批旧接口 |
| `internal/integration/openai/identity.go` | 仅WithTenantID/WithUserID | Identity两字段、WithIdentity/IdentityFromContext、strings/context/aiidentity imports原样 |
| `internal/app/worker/imageagent/dependencies.go` | 仅buildImageAgentWorkerAI内ManagerConfig literal删除 `DefaultClient: "default"` 一项 | Clients/Logger原表达式、NewManager错误处理、resolver注入、worker全部其余字节 |

### 5.2 十二个测试工件

| 工件 | 精确转换或退出 |
| --- | --- |
| `internal/integration/openai/text_completion_test.go` | 整退仅旧文本seam的测试和textTestManager/textTestRequest/requireNoErrorText专属fixtures；不得退当前受控执行器/组件的原测试 |
| `internal/integration/openai/text_knowledge_lease_test.go` | 整退旧Manager route与5分钟上限的唯一专属测试；当前2分钟cap、knowledge租约、deadline/取消在原owner保持并验证，无新测试平台 |
| `internal/integration/openai/product_enrichment_adapter_test.go` | 整退旧adapter/prompt预算/wire oracle/反射walker/严格旧JSON的全部专属测试、嵌入fixture与mutator；当前Candidate/Review/Eino有效规则和测试保持 |
| `internal/integration/openai/tenant_manager_test.go` | 退TestManagerReportsResolverWiring（只证明退休报告方法）；TestManagerRoutesChatClientByTenantOrUserContext用原三个context分别调用ResolveEffectiveClientRoute→GetClientWithRoute→同一callChat，保持三组原Auth/Model/数量精确断言；去该fixture一DefaultClient赋值；三WithTenantID(Background,id)改WithIdentity(Background,Identity{TenantID:id})。其余测试/fixtures/helper原样 |
| `internal/integration/openai/effective_configuration_test.go` | 五函数各去一个DefaultClient赋值；两WithTenantID改同一WithIdentity。版本hash不含secret、配置漂移/空版本拒绝、DB override/真正不可用、原fixture/错误/调用次数全部保持 |
| `internal/integration/openai/credential_store_test.go` | TestGormCredentialResolverPrefersUserThenTenantConfig两处、TestOrganizationOnlyCredentialResolverNeverUsesMemberOrGlobalCredential一处WithTenantID改同一WithIdentity；DB fixture/tenant-user优先级/组织only拒绝成员与全局、其它断言逐字保持 |
| `internal/integration/openai/logger_test.go` | 仅TestManagerPropagatesLoggerToResolverCreatedDynamicClient的一WithTenantID改同一WithIdentity，日志隔离/不变输入/原断言全保留 |
| `internal/app/runtime/currentapplication/title_credential_provision_test.go` | 仅TestTitleCredentialProvisionWritesOnlyTheAdmittedOrganizationRow中 `openai.WithTenantID(context.Background(), "org")` 改为 `openai.WithIdentity(context.Background(), openai.Identity{TenantID: "org"})`；原fixture、admitted组织写入范围、禁用后不借成员凭据/全局回退及所有其它断言逐字保持 |
| `internal/app/worker/imageagent/capabilities_test.go` | TestBuildProductionImageCapabilitiesQuotesCurrentRouteAndRejectsStaleAuthorization、TestProductionSourceEditDoesNotResolveReviewRoute各只去一DefaultClient赋值 |
| `internal/app/worker/imageagent/organization_main_temporal_postgres_test.go` | TestOrganizationWorkerRealTemporalUnpricedGenerationDoesNotReserveOrDispatch只去一DefaultClient赋值，全部真实Temporal/PG安全断言原样 |
| `tests/imageagent_review_invocation_integration_test.go` | helper review334Manager只去一DefaultClient赋值，全部审核/计量/发送/重复/UNKNOWN原断言不动 |
| `tests/imageagent_organization_integration_test.go` | TestOrganizationCredentialAdmissionChecksVerifiedScopeBeforeResolver、scope339Fixture.verifyActivity各只去一DefaultClient赋值，全部身份/组织/持久化隔离断言不动 |

九混合测试合计：**12个DefaultClient赋值、10个WithTenantID改接**；连同唯一生产赋值共13个默认选择写入退出。不是全局删除http.DefaultClient或其它包同名函数，不新增测试helper或公共运行seam。


### 5.3 旧包十二路径退出，六源文件及有效测试抽取

| 原路径（均在 internal/product/enrichment） | 未来处置 |
| --- | --- |
| model.go、content_selection.go、validation.go、scoring.go | 只抽 §4 当前合同/完整有效纯闭包到新叶包同名文件，旧路径删除 |
| errors.go | 五 sentinel 到新 errors.go；provider-only 错误退休，旧路径删除 |
| proposer.go | 仅 ValidateCandidate 原函数到新 candidate.go；生成服务/接口/Dependencies/类型检查/稳定provider错误全部删除 |
| ports.go | CandidateGenerator/GenerationRequest整文件退休，无替代端口 |
| content_selection_test.go、boundary_guard_test.go | 有效选择/纯依赖与runtime AST负例原算法随owner到新叶包；只转换package/root/name，禁入与失败关闭保持 |
| candidate_validation_test.go | 当前repair/evidence/policy/nil/取消/输入验证原断言随owner；“MatchesProposer”不保留旧生成器作oracle，改独立字面期望，保留深拷贝/不变输入 |
| proposer_test.go | §6.1列清生成专属测试退出、纯规则原期望迁到新validation_test.go；validRequest等纯fixture到fixtures_test.go，无fake生成端口 |
| immutability_test.go | 原输入/候选/返回值深层无别名断言在ValidateCandidate执行；只退生成回调/生成输入那一层，不删当前纯规则隔离断言 |
| 新 wire_test.go | 仅合成固定输入/独立literal期望验证候选、Proposal、ContentSelection的原JSON形状；当前Review/AgentConfig原测试里核对原hash，不以旧包并行作oracle |

新叶包未来十三路径：六生产、七测试，§5.8逐路径列出。只有十种model类型加ContentSelection，不把整旧service、测试生成器或无消费者桥换包名当抽取。

### 5.4 当前 Review 装配中的专属尾部

| 工件/符号 | 精确退休与保留 |
| --- | --- |
| product_review_application.go | 只退NewProductReviewApplication及专属imports；KEEP InstallProductReviewSchema/productReviewSchemaReady/buildProductReviewCore/当前source授权binder/Issue36LocalTrialSourceWiring/Review仓库与Applied reader |
| review/service.go | 退Service.proposer/NewService/Service.Create；private create只保留非nil候选路径，去`create`operation kind与固定生成request/branch。KEEP NewCandidateService及其nil检查，CreateFromCandidate/WithTransactionAttempt/ValidateCandidate与当前事务内全部行为；注释改成准确当前能力 |
| product_review_handler.go | specs去根POST/create；request只解码decision/apply，未知kind失败关闭，不默认当create。四条method/path/auth/权限、cached-read/live-write/source bind/projector、32KiB/单幂等键/Unicode与严格解码全部保持 |
| product_review_only_routes.go、product_agent_routes.go、issue36_trial_admission.go | 退productReviewOnlyRoutes过滤wrapper；原调用直接用准确四条productReviewRoutes。KEEP buildLocalTrialReviewRoutes、原timeout赋值/当前source/鉴权依赖，不新增默认装配 |
| current_application.go与四组准入测试 | 只去已无目标的根POST过滤分支，直接枚举四条；角色/配置/当前route集合不扩大。四测试为chat_content_route_admission/content_title_admission/current_application_agent_points/own_content_route_admission |
| a1688/product_enrichment_input.go及_test.go | ProductEnrichmentRequest/两个clone helper和仅自用测试整退；原Normalize/ToSnapshot和真实采集不动 |
| product_review_integration_test.go | titleGenerator及GenerationRequest/afterGenerate/f.g计数全部退；当前fixture用既有core+NewCandidateService+四路由。候选seed经真实CreateFromCandidate，读取/decide/apply仍走真实HTTP/仓库。保留有效并发/重放/撤权/事务/transport/严格JSON断言，详§6.1 |
| product_agent_review_integration_test.go | agentReviewFixture直接NewCandidateService；去旧生成器oracle，保留当前候选准入、scope/provenance、同键冲突、零额外写/事务尝试等证据 |
| chain1_local_product_acceptance_test.go | 去chain1TitleGenerator/旧constructor及根POST造数据；用同一当前fixture组合并经Candidate owner入库。#388 Source→Review→ImageAgent→SHEIN其余业务步骤/owner/原lost response断言保留，不直接插批准事实 |
| product_title_review_browser_fixture_test.go | 旧constructor及seed旧POST改当前候选组合/真实Candidate intake；每个原组织/角色/产品/数量/预期title固定，不编造AgentRun或账务receipt。真实BFF四路由/列表/决策/Apply/响应丢失/重建/audit全部保留 |
| collection/issue36测试 | product_review_collection_integration_test.go和issue36_trial_review_routes_test.go的titleCreate seed改同一当前fixture helper；issue36_trial_admission_test.go改四路由，不删集合隔离/分页/坏记录/撤权/取消及trial准入断言 |

测试组合只在已有 _test.go内使用现有core/port/owner；不把旧生产factory移进测试继续转发生成，不新增生产测试hook、provider、HTTP seed端点、全局server→fixture注册表或验收runner。合成nil ContextProvenance/Execution只沿其**已有允许缺省语义**，有上下文/执行事实的原测试继续传原值，不能伪造已完成事件。

### 5.5 其它当前消费者：只换合同来源

§5.8标为“当前消费者改接”的58路径除product_agent_routes的旧过滤调用也按§5.4收齐外，**只换旧import/限定符为reviewproposal**，删除实际失去用途imports；所有字段/tag/字面值/条件/函数体算法/JSON/hash/原测试期望不变。覆盖Agent/AgentConfig/Workbench、app当前Agent/Content/Supply、Eino模型/证据、三种持久化消费者、ContentAgent和Review域。

不得新增临时alias、默认值迁移、双解析/双hash或放宽读取拒绝。旧包中“定义仍被使用”只证明需要抽取这部分行为，不能再次推导整包KEEP。

### 5.6 必需检查与当前能力 KEEP

检查转换限定§6.1：原聚合加**一个完整闭包guard**；原depguard/域纯度/负例/owner矩阵准确转向新owner、收紧已无需要的OpenAI准入，不关检查、不加missing-skip。workflow与完整inventory零diff，原CI照常。

KEEP当前治理Executor/组件、原SDK/图像ImageClient/Client.Generate（当前TextChatCompleter实现义务）、有效Manager named routes/hash/version/rotation/cache/Close、Identity/WithIdentity、当前AI/计量与UNKNOWN。KEEP当前UI/BFF及其集合POST=405，当前ProductReview schema/记录/四路由/权限/事务/Apply、当前来源/原浏览器/规则/共享基础层、独立SHEIN登录/Store同步。范围外代码不借退休名义顺带删。

### 5.7 文档尾部（18工件，当前仅三文档）

- 本设计在原路径原地重写；README/register从本轮main的完整blob前置准确当前说明，保留原索引/历史/数字/勾选。未来实施只前置实际阶段/证据，不回改冻结设计。
- product-agent-runtime-contract：§2旧文本/Adapter/Proposer/Review表行改当前owner；当前纯校验、Quality、repair/tool示例、测试命令及可执行路径引用换新叶包。旧代码证据改ac54基线permalink并标历史，全部Must/数值/UNKNOWN/授权规则保持。
- agent-knowledge-context-v1、product-content-localization-v1、platform-listing-materials-agent-v1、project-boundaries：只校准当前候选/证据/纯规则的owner/路径/符号，不改业务政策或已冻结范围。
- module-target-mapping/current-refactoring-status：校准当前package映射和REV-1当前constructor/候选owner；#383/#386历史交付/原PASS原样可追溯，新增准确退休注，不把历史改写成未发生。
- issue-36-local-trial-runtime：仅旧constructor前提换当前Candidate组合/四路由说明；运行准备/授权/NOT_RUN边界保持，不另加试用操作。
- product-title-review-ui：仅“Publisher/Proposer/POST造提案”fixture说明改为真实Publisher→Candidate intake→四HTTP审核；generation UI仍Out of Scope，页面/BFF不改。
- ai-commerce-agent-platform-strategy：仅加当前实现owner校准注，澄清Enrichment是能力名称而非保留旧package/执行器的授权，原产品决定全文后缀保持。
- product-agent-text-provider-neutral-v1/product-agent-google-interactions-v1及#893/#934/#923三旧退休文档：前置准确本批替代及新合同owner注，原全文blob后缀保持。
- 其余有日期superpowers计划和历史证据不重写；没有当前消费者的旧源链接不包装为现役入口。本批不扩大产品策略、Figma或验收范围。

### 5.8 同一主要 PR 的完整路径清单

该清单是删除/抽取/准确改接的边界，不按工件数拆PR。149路径包含18文档；旧根十二路径与新叶包十三路径分计。路径只是精确许可，未写明的语义变化不因此获准；§5/§6.1的更窄规则优先。

| 工件路径 | 处置 |
| --- | --- |
| `.golangci.yml` | 原检查准确转换，见 §6.1 |
| `docs/architecture/README.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/agent-knowledge-context-v1.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/architecture/issue-36-local-trial-runtime.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/platform-listing-materials-agent-v1.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/architecture/product-agent-google-interactions-v1.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/product-agent-runtime-contract.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/product-agent-text-provider-neutral-v1.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/architecture/product-content-localization-v1.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/architecture/project-boundaries.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/engineering/product-title-review-ui.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/product/ai-commerce-agent-platform-strategy.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/refactoring/current-refactoring-status.md` | 当前 owner 引用校准，见 §5.7 |
| `docs/refactoring/legacy-register.md` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `docs/refactoring/module-target-mapping.md` | 当前 owner 引用校准，见 §5.7 |
| `internal/agent/content_selection_test.go` | 当前消费者改接，见 §5.5 |
| `internal/agent/contracts.go` | 当前消费者改接，见 §5.5 |
| `internal/agent/manual_review_test.go` | 当前消费者改接，见 §5.5 |
| `internal/agentconfig/content_template.go` | 当前消费者改接，见 §5.5 |
| `internal/agentconfig/content_template_test.go` | 当前消费者改接，见 §5.5 |
| `internal/aiworkbench/content_task_test.go` | 当前消费者改接，见 §5.5 |
| `internal/aiworkbench/contracts.go` | 当前消费者改接，见 §5.5 |
| `internal/aiworkbench/projection_test.go` | 当前消费者改接，见 §5.5 |
| `internal/aiworkbench/selection.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/ai_workbench_content_integration_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/chain1_local_product_acceptance_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/chat_content_route_admission_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/content_title_admission_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/current_application.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/current_application_agent_points_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/issue36_trial_admission.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/issue36_trial_admission_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/issue36_trial_review_routes_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/own_content_route_admission_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/own_product_agent_integration_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_application.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_evidence.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_evidence_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_execution_mode.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_integration_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_knowledge.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_agent_review_integration_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_agent_routes.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_content_application.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_review_application.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_review_collection_integration_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_review_contract_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_review_dto.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/product_review_handler.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_review_integration_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_review_only_routes.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/product_title_review_browser_fixture_test.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/app/httpapi/supply_chain_agent_integration_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/httpapi/supply_chain_agents.go` | 当前消费者改接，见 §5.5 |
| `internal/app/localtrial/seed.go` | 当前消费者改接，见 §5.5 |
| `internal/app/runtime/currentapplication/title_credential_provision_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/app/supplychain/effective_product_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/supplymarket/effective_product_test.go` | 当前消费者改接，见 §5.5 |
| `internal/app/worker/imageagent/capabilities_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/app/worker/imageagent/dependencies.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/app/worker/imageagent/organization_main_temporal_postgres_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/agent/eino/runtime_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/agent/einomodel/content_evidence.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/agent/einomodel/content_evidence_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/agent/einomodel/content_model_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/agent/einomodel/model.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/agent/einomodel/skill_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/crawler/a1688/product_enrichment_input.go` | 整退未使用来源桥，见 §5.4 |
| `internal/integration/crawler/a1688/product_enrichment_input_test.go` | 整退未使用来源桥，见 §5.4 |
| `internal/integration/openai/client.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/client_manager.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/contextual_client.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/credential_store_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/effective_configuration_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/identity.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/logger_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/product_enrichment_adapter.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/product_enrichment_adapter_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/tenant_manager_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/text_completion.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/text_completion_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/openai/text_knowledge_lease_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `internal/integration/persistence/agentconfig/content_snapshot_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/persistence/agentconfig/content_template_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/persistence/agentconfig/snapshot.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/persistence/aiworkbench/content_task_test.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/persistence/aiworkbench/store.go` | 当前消费者改接，见 §5.5 |
| `internal/integration/persistence/product/review/content_postgres_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/contentagent/own_source_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/contentagent/validator.go` | 当前消费者改接，见 §5.5 |
| `internal/product/contentagent/validator_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/enrichment/boundary_guard_test.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/candidate_validation_test.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/content_selection.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/content_selection_test.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/errors.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/immutability_test.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/model.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/ports.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/proposer.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/proposer_test.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/scoring.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/enrichment/validation.go` | 旧包路径全部删除，见 §5.3 |
| `internal/product/review/candidate_attempt_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/candidate_execution_gate_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/collection.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_intake.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_intake_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_patch.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_patch_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_record.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_record_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/content_validation.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/execution.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/execution_policy.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/limits_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/lineage.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/lineage_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/model.go` | 当前消费者改接，见 §5.5 |
| `internal/product/review/proposal/boundary_guard_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/candidate.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/candidate_validation_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/content_selection.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/content_selection_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/errors.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/fixtures_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/immutability_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/model.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/scoring.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/validation.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/validation_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/proposal/wire_test.go` | 纯合同/规则与有效测试抽取，见 §5.3/§6.1 |
| `internal/product/review/service.go` | 生成与过滤/夹具尾部，见 §5.4 |
| `internal/product/review/skill_gate_test.go` | 当前消费者改接，见 §5.5 |
| `internal/product/sourcing/boundary_guard_test.go` | 原检查准确转换，见 §6.1 |
| `tests/depguard_config_test.go` | 原检查准确转换，见 §6.1 |
| `tests/imageagent_organization_integration_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `tests/imageagent_review_invocation_integration_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `tests/import_boundaries_test.go` | 原检查准确转换，见 §6.1 |
| `tests/issue388_chain1_acceptance_contract_test.go` | 原检查准确转换，见 §6.1 |
| `tests/listingkit_backend_retirement_test.go` | 旧 OpenAI / 原三文档与依据，见 §5.1/§5.7 |
| `tests/product_review_boundary_test.go` | 原检查准确转换，见 §6.1 |
| `tests/target_architecture_phase2_test.go` | 原检查准确转换，见 §6.1 |
| `tests/target_architecture_phase3_product_test.go` | 原检查准确转换，见 §6.1 |
| `tests/testdata/catalog_adapter_product_domain_violations/nonadapter.go` | 原检查准确转换，见 §6.1 |

## 6. 验证与验收

当前设计：三文档diff、149路径存在性/唯一性/闭包、准确import/consumer/旧装配核对、路径/链接、README/register完整main blob后缀、diff --check和原文档/consumer检查。作者本阶段不运行构造器/应用/业务测试；准确设计HEAD原CI单列，不是实现或运行验收。

未来用户确认/Ready后，同一Writer先用新增完整guard确认真实RED；再按§4抽取并改接、收齐生成/夹具/检查/文档尾部、旧根缺席后GREEN。采用现有合成fixtures与原测试，不加验收工具平台。

| Must | 必须取得的证据 |
| --- | --- |
| 无旧生成/兼容尾部 | 整旧根/old bridge/oldAdapter/text seam缺席；全部tracked Go（含negativefixture/debug）decoded import零旧根；无旧Proposer/NewService/factory/rootPOST/过滤wrapper/旧typealias转发 |
| 当前pure行为 | 原有效选择/证据/质量/诊断/深拷贝测试迁后执行；新leaf无provider/runtime/parent/Agent依赖，Catalog/Sourcing不反向引用 |
| JSON/hash稳定 | 独立合成literal字节/expected hash覆盖当前Proposal/Candidate/Selection零值/null/空集合，当前Review.operation/proposalDigest与AgentConfig/Workbench冻结指纹原期望；旧路径删除后原严格读取拒绝/JSON/SourceIdentity测试仍执行。禁止自比较、更新golden来适配意外diff |
| 当前Review安全 | Candidate intake/transactionAttempt、replay同键同载荷/冲突、授权重查、证据/source不写、锁/事务/CAS/Apply、response loss和取消原有效断言改接执行；当前UI/BFF四路由不增加生成 |
| 当前执行/凭据 | 原组织only/路由版本/hash/secret隔离/logger/图像review/source edit，治理Executor final deadline/no-send/UNKNOWN及Knowledge租约原测试保持 |
| 可编译/交付 | 原current-application、shein-login-worker、1688-local-agent、1688-public-browser-collector与hack/debug的test-amazon/test-1688六入口仅编译到任务TEMP，不执行；准确源码HEAD原PR CI整run与独立最终复核 |
| 权限与历史 | workflow/完整inventory/依赖/保护零diff，未操作既有实例/数据/凭据/进程；#301 OPEN且marker后历史完整 |

作者只运行适用必要原自检；原CI按不变workflow使用它自有合成PG/Temporal测试配置，本批不另开真实实例或探针，不把已有配置当作读取凭据授权。运行/平台/产品验收 **NOT_RUN**；实现者/AI review不能签发用户验收或可上线。

### 6.1 原检查与测试的准确转换边界

1. **聚合**：tests/listingkit_backend_retirement_test.go只新增TestLegacyOpenAIAdapterClosureIsRetired一个完整函数，复用原trackedFiles/AST/decoded import/失败关闭helper；原函数/helpers/imports不变。覆盖§5.1原35函数/13类型/2字段与本批旧根/生成ports/factory/Review.Create/来源桥/根POST/过滤wrapper缺席及当前leaf/current4routes保持，按准确package/receiver，不全局禁Generate/Client/Manager/Create等通名。
2. **纯测试抽取**：proposer_test原NewProposerRejectsNil/TypedNil、ReturnsOnlyStableGenerationErrors、HonorsCancellationThatOccursDuringGeneration、DoesNotGenerateWhenContextCancelsDuringInputPreparation五生成专属测试退。ValidatesContextAndInputBeforeGeneration只留当前ValidateCandidate的nil/cancel/input断言；其余八个证据/验证顺序/坏诊断/稳定Quality/诊断去重排序测试保留原fixture与独立期望，去生成回调而直接验证Candidate。candidate_validation的MatchesProposer改独立literal oracle；immutability保留当前深clone隔离，content_selection及两类AST guard原算法/负例保持，只有owner/name/path转换。wire_test仅独立字节fixture；hash回归加在既有当前owner测试文件，不给新叶包引父域/Agent。
3. **Review旧fixture测试**：§5.4精确工件内去titleGenerator/chain1TitleGenerator/afterGenerate/call counters；NewCandidateService/current四routes组合放原_test helper，seed用真实CreateFromCandidate和真实Reader/Source/Store/auth，不直接SQL插批准事实。保留PostgresLifecycle/CallerOwnsLifecycle/OneDatabaseConnection/BoundedPool/锁等待proof到期/ReplayConflict/双提案竞争/BoundedRead/Capacity/PublisherRace/AtomicFailures/LostResponse/CorruptEvidence/RejectStale/严格JSONUnicode原有效断言。只属“provider在锁外运行”的原测试退（当前Review不再有provider调用）；其中如有当前事务/连接断言则改由同一candidate路径保留。
4. **授权/transport sibling**：RevokedBeforeCreateCommit/GrantDependencyFailureAfterGeneration等有效撤权/依赖失败断言移到当前candidate validation→transaction窗口，利用现有source/auth port测试替身或原DB callback，**不加生产hook**；原组织/source/commit前无写与错误分类保持。旧root POST的通用body deadline/Unicode/strict input换到当前decision/apply，已commit丢响应通过当前候选提交/真实审核动作验证transactionAttempt/replay；不因删除旧URL删掉有效HTTP/事务断言。根POST只验不注册，不伪造旧POST返回成功。所有四路由准入测试与#388/BFF/browser当前安全断言保留。
5. **.golangci.yml仅owner转换**：phase3_product_domain_boundaries、domain_app_httpapi_boundaries、project_boundary_listingkit三规则原enrichment file globs精确换review/proposal对应根/子树；原deny列表/算法/其它roots不变。product_catalog_persistence_boundary保留旧退役根deny，增加新leaf根`$`/子树`/`两deny。不是workflow或inventory修改，不关闭linter/不加exclude/不放宽SDK或provider准入。
6. **depguard_config_test.go四函数**：TestPhase3ProductDomainBoundaryDepguardRuleCoversOnlyTargetSubpackages/TestDomainAppHTTPAPIDepguardPatternCoversBusinessTrees/TestProjectBoundaryListingKitDepguardPatternCoversDomainTrees只同步对应新glob；TestProductCatalogPersistenceDepguardDeniesEverySiblingProductDomain只追加新leaf根/子树的必须拒绝项。原集合/正反例/失败关闭保持；其它函数不变。
7. **Phase2 provider矩阵**：isAllowedIntegrationProviderAdapterImport的openai分支仅删enrichment白名单；TestIntegrationProviderAdaptersEnforceRootSpecificContractMatrix同步退该allowed项，将旧根和新leaf列为openai forbidden，并为其它provider原forbidden矩阵追加新leaf。其余规则/decoded import算法/escaping负例/类型解析不改；不把新leaf加入OpenAI白名单。
8. **Phase3/Sourcing/Review边界**：TestPhase3ProductTargetDependencies的旧root改为实际新leaf并强制存在/有生产文件（不继承missing-skip），其它roots原规则保持；sourcing禁入表保留旧根并追加新leaf。两ProductEnrichmentRuntimeSemantic/HasNoRuntimeSemanticDeclarations测试改新owner名称/root，原声明语义遍历/负例不变。internal/product/sourcing/boundary_guard_test.go仅在TestProductSourcingDependsOnlyOnNeutralProductFacts追加新leaf禁入；tests/product_review_boundary_test.go仅向adapter禁入表追加新leaf，原禁止默认旧factory断言可原样保留（新guard负责完整缺席），其余不改。
9. **Catalog负例**：tests/testdata/catalog_adapter_product_domain_violations/nonadapter.go只把旧enrichment根和/future两import改为review/proposal根和/future；TestProductCatalogPersistenceGuardRejectsEverySiblingProductDomainFixture的want同样改接。通用“非Catalog Product全部拒绝”的算法/其它import/负例不变，不保留旧实际Go import或豁免testdata。
10. **#388 assembly**：TestIssue388Chain1AcceptanceAssemblyContract只把NewProductReviewApplication必需marker换成现有harness实际调用的新candidate-only测试helper marker，并在同一函数显式读取该helper源确认review.NewCandidateService/真实core；原其它owner/禁止直写事实/runner/授权/receipt断言不变，不新建runner或验收步骤。
11. **旧OpenAI测试**仅§5.1的整退/函数退役、12个测试DefaultClient赋值删除、10身份fixture改接、一个tenant路由测试改当前显式route；连生产唯一赋值共13默认赋值退。原有效Auth/Model/count/hash/version/secret/logger/组织only等断言和helpers保留，不机械重写其它代码。
12. **文档/历史/冻结**：§5.7准确当前引用变化外不改政策/Must/历史PASS/旧数字勾选；README/register及五旧依据/产品策略规定完整blob后缀保持。#301更新前新读，只改当前段；从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保留，正文≤65536字符且OPEN。完整inventory、workflow、其它检查逻辑零改；真实CI若再暴露漏项，先查同根sibling/当前Must，给精确最小补丁再取用户例外，不能视本设计为任意删检查授权。

## 7. 评审、确认和停止点

1. 只设计三文档 → 必要自检/准确设计HEAD原CI → 适用独立Architecture Review → 用户确认本§1–§7/§6.1 → Issue/PR IMPLEMENTATION_READY / Ready读回，才准生产实现。原旧29工件设计复核不覆盖本次149工件设计；当前SOURCE_NOT_STARTED/NOT_READY。
2. 一个Writer、原分支/PR整批完成；起始已同步actual main，最终候选前再核对真实main组合，不为了始终最新反复merge。完整提案/纯规则改接是同一交付闭包，不拆成长期双包/兼容过渡。
3. 最多两轮正常Architecture Review；finding先分类并指出当前Must/具体后果/阻塞层级；非Blocker到IMPLEMENTATION_TEST/Backlog，不无穷扩架构。源码完整后一次独立最终交付复核，修复只复核增量，设计review不能替代源码review。
4. GitHub复核启动失败至多唯一重试；**未经用户本轮授权不默认新增AI互验线程**。旧#953窄范围或设计review授权不扩作本次重做设计/未来源码review授权，不新增实现者、runner或治理平台。
5. 原CI准确源码HEAD整run和最终复核满足后，交用户另行授权正常expected-head合并；不绕过/强推/改保护。获准合并后只认实际mergeSHA的event=push/main准确整run completed/SUCCESS；设计CI/PRCI不能代替。
6. #301保持OPEN；当前采集/规则/PoolManager deadline取消/独立SHEIN登录与Store同步保持。不改workflow/完整inventory，不操作既有实例/数据/凭据/进程，不增加内部试用前置。源码、mainCI、运行/产品验收分列，合并/部署/关单另授权。
