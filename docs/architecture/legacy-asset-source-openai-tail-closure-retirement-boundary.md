> #301 / #983 当前整批实施：**RETIRE / Independent Architecture / USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认冻结[设计18a955e22 §1–§7（含§4.2/§6.1）](https://github.com/qq550723504/shuomi-ai-commerce/blob/18a955e225285c5f1068f7e6ea07e9f8ba1e5a9e/docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md)；同一39路径收齐资产静态recipe、来源Publisher包装和17合同alias/错误转发/旧resolver版本尾部，不机械拆批。有效行为复用现有internal/ai/ToSnapshot/catalog.Publisher，原断言仅按冻结§6.1精确改接；JSON/hash/版本/已保存载荷、当前能力、独立SHEIN登录与Store同步保持，无兼容/新owner。workflow/完整inventory/依赖/保护与既有实例/数据/凭据/进程零改，不增试用前置，#301 OPEN。必要自检、准确源码HEAD/原CI/最终复核留同PR；不等于运行/产品验收，合并/部署/关单另授权。以下冻结正文完整保留。

# 旧资产配方、来源发布包装与 OpenAI 迁移尾部完整退休边界

状态：**DESIGN_ONLY / AWAITING_REVIEW_AND_USER_CONFIRMATION / SOURCE_NOT_STARTED**。父 Issue [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301) 保持 OPEN。调查基线为 main `2505dd76676f452a85e91c07e43fac3eadcd4de9`；以下是未来同一主要 PR 的完整边界，不是已完成的源码退休声明。

## 1. 用户结果、范围与 Design Basis

使用者是维护当前内部试用、图像与来源业务及独立 SHEIN worker 的操作者。结果是清除不再拥有当前操作入口的资产静态配方、来源发布包装，以及 AI 合同迁移时遗留的转发和旧版本绑定；当前来源保存、审核、资产批准、聊天/图像、采集、SHEIN 登录与 Store 同步沿现有入口继续工作。不新增用户步骤或内部试用前置。

用户授权三组一起设计，不机械拆批，确认后再实施。本阶段 **Design Basis: N/A / 文档工作 Ready**，实际只允许本文、架构索引和 Legacy Register 三文档；不改生产、测试、fixture、旧依据或运行资源。未来实现采用 **Independent Architecture / NOT_READY**：本次结束一项公共 Go 合同过渡，并收窄 Manager 路由版本接受集合，需要明确当前 owner、持久化和副作用边界，不能以“改 import”代替设计准入。适用独立复核、用户确认及 Issue/PR 的 IMPLEMENTATION_READY 读回均须先于生产修改。

产品依据为本轮用户决定、[greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及当前来源/AI/资产合同。无 UI、IA、路由或页面变化，Figma N/A。复用 [provider-neutral AI owner](../superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md)、[当前 Product Agent 运行合同](product-agent-runtime-contract.md)、[来源业务闭环](source-business-closure-v1.md)和实际源码；本文只替代 §2 指定的旧 KEEP/迁移要求，不重开全局架构。

Must：三组完整收口，无兼容 alias/re-export/wrapper；catalog.Publisher、全部当前能力及独立 SHEIN 登录/Store 同步保持；原有效业务、安全、组织隔离、幂等、无发送和取消断言保留。Should：复用已有 owner 和现有原测试，不新造框架。Out of Scope：workflow、完整 inventory、依赖/lock、保护规则、UI、新 provider、schema/迁移、运行验收和既有实例/数据/凭据/进程。无新增 Accepted Risk；现有阶段边界不变，运行/provider/用户验收 NOT_RUN，不宣称可上线。

## 2. 前批 KEEP 的准确替代与 Legacy 决策

| 旧依据/对象 | 本批确认后替代的准确边界 | 继续保留 |
| --- | --- | --- |
| #972 `legacy-localagent-source-policy-closure-retirement-boundary.md` 中 asset/recipe KEEP；#981 `legacy-source-normalization-facts-closure-retirement-boundary.md` 的 recipe KEEP | 仅当前 `internal/product/asset/recipe` 整根及专属 fixture 引用退出；不能把“ImageSet/recipe”泛称解释为所有近名类型一起退出 | 当前 imageagent.ImageSlotRecipe、图片提示词/计划/图像规则、资产候选/显式批准/Inventory/readiness、SDS/POD |
| Phase3 Product spec/plan 与当前 product-sourcing handoff/MVP 中 `sourcing.Publisher -> catalog.Publisher` | 只结束 `product/sourcing/publish.go` 的 PublishRequest/Publisher/NewPublisher/Publish 包装，当前文档改指真实 InternalProducer/SRC/Catalog 组合 | catalog.Publisher/PublishRequest/Repository/快照版本，SourceEnvelope/ToSnapshot/PublicationIdentity、InternalProducer 及原授权事务 |
| 2026-08-21 provider-neutral AI spec/plan 的 one-cycle aliases、保留别名与 sentinel 转发要求 | 一次性改接所有包内外实际消费者到已有 internal/ai，删除 openai 的 17 type aliases 与一个错误 re-export；原“本次不删别名”仅为历史阶段决定 | internal/ai 的全部原合同与原 error 对象；OpenAI Client/Manager/Pool/配置/resolver/凭据/HTTP/日志及实际 provider 能力 |
| Manager 的 resolverVersion 注释、旧 CacheKey 版本接受分支及其唯一正向测试 | Manager 只接受它自己返回的有效配置版本；删除闲置 EffectiveClientRouteResolver 接口，不删除同名方法 | EffectiveClientRoute 值、ResolveEffectiveClientRoute/GetClientWithRoute、实际 contextualImageClient、已存在的严格版本/凭据漂移拒绝 |

Legacy decision: **RETIRE**。Reusable behavior: 有效 AI 合同已在 internal/ai，来源转换/保存与资产批准已在当前 owner；本批复用，不新建生产 EXTRACT 包、替代配方器或第二 Publisher。Current owner: internal/ai；integration/openai 的配置解析与有效版本；product/sourcing、catalog 与其现有持久化 adapter；product/asset、imageagent。Cutover/deletion condition: 用户确认、适用复核和准入后，同一主要 PR 按 §5/§6.1 原子改接并删除，保留断言和准确 CI 通过；源码删除不授权数据或环境删除。

历史文档不重写、不改勾选/计数/旧失败证据；只在 §5.2 六份直接依据加准确替代顶注，并保持起始 main 完整原 blob 为逐字后缀。前批广泛 KEEP 不能恢复本文已确认退休的具体对象，也不能授权扩大到未列对象。

## 3. 真实消费者与完整调用闭包

调查解析基线全部 **3114 个 tracked Go 文件**的 import 字面量、alias、selector、声明和引用，包括测试、debug、testdata/构建条件文件，再核对同包调用、receiver、接口、构造/注入与非 Go 引用。只运行标准 Go 源码解析器，不执行仓库应用、不连接既有环境；文件引用计数不冒充运行验收。

| 目标 | 实际消费者/装配 | 完整处置 |
| --- | --- | --- |
| asset/recipe 四生产文件 | 无实际生产 import/构造/注册；唯一外部 import 是违规导入 fixture；本包 resolver_test.go 四测试 | 四源码及专属测试整退，fixture 的嵌套域负例换成 asset/future，仍保留八个违规样本及原算法 |
| sourcing.Publisher 包装 | 包外生产消费者零；本包 publish_test.go 两测试和 tests/issue30_sourcing_contract_test.go 的 fixture helper 使用；不含任何当前 schema/队列/HTTP 注册 | 删 publish.go；两个原测试文件的有效断言改用 ToSnapshot + catalog.Publisher，不创建生产或 test-only Publisher 类 |
| openai 合同别名/re-export | 包内六生产文件和六测试文件有未限定引用；包外生产只有 debug/replay-sale-attribute/main.go 的两个 chat 类型、worker/imageagent/capabilities.go 的 ImageRouteSelection；另一个包外测试使用 route 类型 | §5.2 的 16 个文件同批完成：13 个 OpenAI 文件、两个包外生产文件和一个包外测试文件；只改到已有 ai 合同，不留下中转 |
| 旧 resolver 版本绑定 | 仅 client_manager.go 生成/接受 resolverVersion；tenant_manager_test.go 一个用 CacheKey 作 ConfigurationVersion 的正向调用依赖它。未发现生产传入该裸 CacheKey 的消费者 | 改该测试为 ResolveEffectiveClientRoute 返回值；新增实际旧口径拒绝测试后删旧字段/赋值/条件/注释 |
| EffectiveClientRouteResolver | 只有声明，无生产/测试类型引用、实现断言或注入 | 随 route 专属尾部退出；Manager.ResolveEffectiveClientRoute 仍保留 |

真实保留路径：

```text
当前来源 HTTP / acquisition / Browser Capture / 自有来源 / 数据服务
  -> 已授权 InternalProducer / 原 SRC store 事务
  -> ToSnapshot / 原 Catalog bridge -> catalog.Publisher -> 不可变快照
  -> Collection / Review / 明确资产批准 / 当前 readiness

ImageAgent routed provider -> Manager.ResolveEffectiveClientRoute
  -> quote/authorization 的当前精确版本
  -> GetClientWithRoute 或 contextualImageClient.EditImageWithRoute
  -> 相同版本核验 -> 原 Client/Pool/HTTP
```

`app/productsourcing/application.go` 的 catalogBridge、`app/dataservices/publication.go` 和 Review 事务真实使用 catalog.Publisher，全部零改。Manager 路径的 image/review quote 已使用 EffectiveClientRoute.ConfigurationVersion。当前 `generationProviderFactory` 的 GRSAI/百炼版本由自身已发布配置/metadata 和绑定客户端持有，不经过 Manager 的 resolverVersion 兼容条件；不能把这些有效版本改成 Manager hash。当前文本传输、模型路由、identity、缓存优先级/失效和恢复 owner 也不改。

需要指出的现有问题：一方面代码仍要求保留“迁移一周期”的合同转发，另一方面 Hard-Cut 已无 Compatibility 类别；Manager 又同时接受有效配置身份和旧 resolver 标识，使版本绑定有两种口径。解决方式是结束这两个具体过渡面，保留现有正确 owner，不另建通用路由/凭据层。没有证据表明当前生产请求正在利用旧分支，不将静态发现写成实际事故。

## 4. Owner、JSON/hash/版本与持久化边界

### 4.1 事实、授权和外部副作用 owner

资产静态 recipe 的 ResolveRequest/Template/AssetRecipe/Resolver 没有 repository、schema、注册 codec、HTTP/队列或当前持久化 consumer。其 JSON tag 不构成当前对外义务；不把这些值搬进现役 ImageSlotRecipe，不改 asset.Kind 的字面量/枚举。物理环境是否残留相关记录为 UNKNOWN，本批不探查、读取、删除或迁移。

来源包装不拥有事务或新的持久化事实。当前 SourceEnvelope/SRC receipt/证据、Catalog 快照与版本、资产显式批准各守原 owner；Organization/actor/access 复核、ExpectedBaseVersion、幂等冲突与恢复均保持。测试直连 Catalog 仍标注为受控领域 fixture，不把它包装成授权入口，不替换正式 InternalProducer。当前 SourceIdentity 四零值槽位、原规范化/拒绝全部零改。

OpenAI 保留真实配置/resolver/identity 注入、Client/Pool 限速/排队/deadline/重试、SSRF 防护、引用图限额、流量和计量、无发送释放/UNKNOWN/recovery、当前账务/组织边界。删除旧版本条件只收窄 Manager 的入场口径，不新增 provider 请求、重试、自动接管、fallback 或恢复扫描；其余当前作用域/事务/状态机及 SHEIN worker 原 owner/claim/fence/UNKNOWN 零改。

### 4.2 编码、hash 和版本不变量

1. `internal/ai/contracts.go` 与 contracts_test.go **完整零改**。17 个原 type aliases 的目标原字段/tag/顺序/类型/接口方法不变，客户端签名直接写 ai.*；不复制 type、不重命名 JSON 字段、不增加 alias/转发。17 项为 ChatCompletionMessage、ChatCompletionContentPart、ChatCompletionContentPartImage、ChatCompletionRequest、ChatCompletionChoice、Usage、ChatCompletionResponse、ChatCompleter、TextChatCompleter、ImageGenerateRequest、ImageEditRequest、ImageData、ImageResponse、ImageAsyncSubmitResponse、ImageAsyncQueryResponse、ImageGenerator、ImageRouteSelection。
2. 删除 `openai.ErrAsyncImageGenerationNotSupported` 后原返回值和断言直接使用 **同一个** ai.ErrAsyncImageGenerationNotSupported 对象；保留 errors.Is 和直接比较语义，不重造 errors.New、不改变实际 async unsupported 行为。Go import 的局部命名不属于兼容 re-export；优先现有 ai 名，不保留把新 owner 命名成旧包的假兼容。
3. Manager 的 `effectiveConfigurationVersion` 整个函数、`effective:v1:` 格式、字段串接顺序、规范化、SHA-256 和不把 secret 纳入 hash 的既有规则零改；resolver CacheKey 的产生、凭据引用及当前 GORM CredentialVersion/UpdatedAt 语义零改。空版本/旧 CacheKey/过期版本/错凭据均在返回客户端或发送前以原 ErrClientConfigurationChanged 失败关闭；nil selection 的现有未绑定调用语义不扩不缩。
4. 删除 effectiveClientConfiguration.resolverVersion、其非 static 赋值、matchesLegacyResolverVersion 及其唯一 OR 接受分支；有效条件仍为非空且等于 resolved.route.ConfigurationVersion。缓存仍以该有效版本作 key，resolver/static 优先级、staticClient/dynamicClients 生命周期不变。闲置 EffectiveClientRouteResolver 接口可删，EffectiveClientRoute 值及全部使用中的方法留原 owner。
5. SourceEnvelope、SourceIdentity、PublicationIdentity/ToSnapshot、ProductSnapshot/PublishedSnapshot、当前 hash/SourceKey/publication key、Catalog/SRC schema/版本/既有 payload 全部零改。issue30 fixture 仍先用原 PublicationIdentity 取得同一 ProductKey/PublicationID，再用原 ToSnapshot 构造 catalog.PublishRequest，不引入新的身份算法或 ExpectedBaseVersion 默认语义。
6. 本批不改、重算或补录已有报价、invocation、快照、绑定版本或数据库记录，不新增双版本解析和旧记录迁移。静态追踪表明本次旧 CacheKey 正向 consumer 是测试，当前 Manager quote/执行已用有效版本；若实现取得具体当前外部义务或持久化 consumer 的相反证据，停止对应部分报告用户，不自行增兼容入口。

## 5. 删除、改接与保留矩阵

### 5.1 成员级边界

| Owner / 文件 | RETIRE / 允许改接 | KEEP |
| --- | --- | --- |
| asset/recipe 整根 | BaseAssetRecipes、ResolveForPlatforms、FlattenResolved、StaticResolver/NewStaticResolver/Resolve、newRecipe、ResolveRequest/Template/AssetRecipe/Resolver 及四测试/recordingResolver | asset 父包、当前 ImageSlotRecipe、全部有效图像与批准规则 |
| sourcing/publish.go | PublishRequest、Publisher、NewPublisher、Publisher.Publish 整文件 | 当前 sourcing 其余源码、InternalProducer、PublicationCommand/Scope/Receipt、catalog 同名类型 |
| sourcing/publish_test.go | 两旧包装测试改为当前 ToSnapshot→catalog.Publisher 的组合测试，见 §6.1 | sourcingSnapshotWriter 及原有效断言，不另建 wrapper/工厂 |
| openai/types.go | 17 type aliases、一个 error re-export、旧迁移注释及失去用途的 ai import | ClientConfig、NewClientConfig、PoolConfig 所有字段/tag/default/表达式逐字保留 |
| openai/client_manager.go | §4.2 所列旧版本字段/条件及闲置接口；合同类型改用 ai.* | hash 函数、所有有效 route/config/identity/cache 错误与约束、原 Manager 运行行为 |
| 其余列入的 AI 消费者 | 仅原类型/错误引用及必要 import/gofmt 改为已有 ai.* | 任何 HTTP 请求体/SDK 参数/模型/计费/超时/重试/async/日志和断言不改 |

### 5.2 未来同一主要 PR 的 39 唯一路径 manifest

**整退（6）：**

- `internal/product/asset/recipe/base_recipes.go`
- `internal/product/asset/recipe/model.go`
- `internal/product/asset/recipe/resolve_groups.go`
- `internal/product/asset/recipe/static_resolver.go`
- `internal/product/asset/recipe/resolver_test.go`
- `internal/product/sourcing/publish.go`

**来源原测试改接（2）：**

- `internal/product/sourcing/publish_test.go`
- `tests/issue30_sourcing_contract_test.go`

**OpenAI 生产/原测试及全部包外 consumer（16）：**

- `internal/integration/openai/types.go`
- `internal/integration/openai/client.go`
- `internal/integration/openai/client_manager.go`
- `internal/integration/openai/contextual_client.go`
- `internal/integration/openai/images.go`
- `internal/integration/openai/pool.go`
- `internal/integration/openai/product_image_adapter.go`
- `internal/integration/openai/effective_configuration_test.go`
- `internal/integration/openai/images_test.go`
- `internal/integration/openai/pool_request_test.go`
- `internal/integration/openai/product_image_adapter_test.go`
- `internal/integration/openai/tenant_manager_test.go`
- `internal/integration/openai/types_contract_test.go`
- `hack/debug/replay-sale-attribute/main.go`
- `internal/app/worker/imageagent/capabilities.go`
- `tests/imageagent_review_invocation_integration_test.go`

**原 guard / fixture（3）：**

- `tests/listingkit_backend_retirement_test.go`
- `tests/import_boundaries_test.go`
- `tests/testdata/catalog_adapter_product_domain_violations/nonadapter.go`

**当前设计/导航/owner 说明（6）：**

- 本文 `docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md`
- `docs/architecture/README.md`
- `docs/refactoring/legacy-register.md`
- `docs/architecture/project-boundaries.md`
- `docs/product/product-sourcing-handoff.md`
- `docs/product/product-sourcing-mvp-plan.md`

**旧直接依据只加窄顶注（6，起始 main 原 blob 完整后缀逐字保留）：**

- `docs/architecture/legacy-localagent-source-policy-closure-retirement-boundary.md`
- `docs/architecture/legacy-source-normalization-facts-closure-retirement-boundary.md`
- `docs/superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md`
- `docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md`
- `docs/superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md`
- `docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md`

6+2+16+3+6+6=39。当前三文档不是 39 路径已实施；未来状态只给本文/索引加当前顶注，冻结正文保留。当前 handoff 只替换 canonical publication owner 行和目标流程旧 wrapper 节点，并窄注 recipe KEEP 被本文取代；MVP 只改同一 Canonical facts 行和顶注。project-boundaries 只加准确当前顶注。两索引保持原完整后缀，不重排旧阶段。

### 5.3 完整零改面

表外源码和检查全部保留。尤其 catalog.Publisher/请求/仓库/Review 事务、source_identity/ToSnapshot/PublicationIdentity/当前 SRC 读写、asset/ImageAgent/当前 recipe、当前 AI 合同/文本传输/凭据存储/identity/日志注入/动态 generation factory、采集/原浏览器/debug/PoolManager deadline取消、独立 SHEIN 登录/Store 同步零改。debug/replay-sale-attribute 仅换合同 import/引用，不启动、不重放、不读 prompt 或配置。

workflow、完整 inventory（CURRENT/历史/计数/ceilings/accuracy/parser）、lint/depguard、依赖/lock、保护规则与其它检查零改。原 OpenAI 消费边仍在：debug 保留 NewClient/AdaptLogrus，worker 保留 Manager/ProductImageAdapter；不需要 CURRENT 校准。不做 go mod tidy，也不把独立 SDK/PoC、当前准备能力或其它近名 Publisher/Recipe 顺带删除。

## 6. 同批实施与必要验证

用户确认、适用独立复核和 IMPLEMENTATION_READY 读回后，同一 Writer/主要 PR：刷新准确 main/consumer；在原 guard 追加本批退休断言、在原 effective_configuration_test.go 加旧版本拒绝负例，先取得真实 RED；按矩阵一次性改接所有合同 consumer、测试 owner，删除旧源码/分支并收齐文档；取得 GREEN 和必要当前回归，最终候选前核对一次实际 main 组合。不用长期双包/兼容中间态交付，不为文件数拆 PR。

### 6.1 原检查与有效断言的精确处置

| 原文件 / 函数 | 允许的准确变化 | 必须保留 |
| --- | --- | --- |
| recipe/resolver_test.go 的四测试及 recordingResolver | 随已无运行 consumer 的静态配方整退 | 不能据此删当前图像/asset 批准、readiness、ImageSlotRecipe 的任何测试；旧浅列表数量不是当前产品 requirement |
| sourcing/publish_test.go 两 TestPublisher* | 改名体现 ToSnapshot/Catalog 组合；去 NewPublisher，直接原 ToSnapshot 后构造 catalog.PublishRequest，复用原 catalog.Publisher/sourcingSnapshotWriter；第二测试直接断言 ToSnapshot 身份缺失错误和 writer 未调用 | 原 fixture 内容、精确 tenant/product/publication identity、Version/Title、属性排序及 ErrSourceIdentityRequired/未写入断言；不删任何有效断言，表外测试零改 |
| tests/issue30_sourcing_contract_test.go | 仅 issue30Publisher 返回/构造改用 catalog.Publisher；issue30Request 返回 catalog.PublishRequest，以同一原 PublicationIdentity/ToSnapshot 得到 identity/载荷，并 require.NoError；仅三处语句的四个字段选择器改接：跨组织读取中的 request.ProductKey → request.Identity.ProductKey；other.TenantID → other.Identity.TenantID；版本比较中的 firstRequest.ProductKey / secondRequest.ProductKey → 各自 Identity.ProductKey。PublicationID 选择器不变 | 除左列四个结构选择器外，原三个 TestIssue30* 的所有断言与控制流程逐字保持：重开DB/响应丢失重放、同键异载荷/lineage冲突、并发、取消、组织隔离、跨来源版本/历史、批准actor/幂等/readiness，原 fixture 数据/权限注释/DB helper 不变 |
| tenant_manager_test.go / TestManagerAllowsResolverBackedUnregisteredImageClient | 原带 identity 的同一 ctx 先 ResolveEffectiveClientRoute，断言成功，以返回的 CredentialReference/ConfigurationVersion 调用原 EditImageWithRoute；合同引用改 ai.* | 原 resolver fixture/静态 fallback、GetDefaultModel 空值、未注册 image consumer 可解析、原“缺少 image bytes”错误断言全部保留。其它测试仅机械类型限定，不改行为 |
| effective_configuration_test.go | 原测试只将 ImageRouteSelection 改 ai.*；追加有界新测试，旧 CacheKey 必须 ErrClientConfigurationChanged 且 capture server 收到零请求；当前有效版本仍可成功调用；同 CacheKey 下模型或 endpoint 变更后旧有效版本仍拒绝且零请求 | 原全部测试函数的 fixture/期望/错误/次数断言及 effective:v1 算法、非secret规则、static/DB优先级、空值/漂移/缺配置失败关闭不变；不修改原断言以接受旧 CacheKey |
| types_contract_test.go / TestOpenAIClientImplementsProviderNeutralContracts | 仅移除退休的 openai 错误 re-export 等于 ai.error 的 if 块 | 原两个 Client 满足 ai.ChatCompleter/ai.ImageGenerator 的编译断言保留；实际 async sentinel 正确性由下行原测试保留，不能造新 sentinel |
| OpenAI images_test/pool_request/product_image_adapter 及包外 review integration 测试 | 只将失去 alias 的类型/错误名改到 ai.* 和必要 import；原已使用 ai.* 的部分不动 | 原 errors.Is、usage decode、实际 multipart/JSON/async、SSRF/限额/timeout、once-send/预算/quote/trace 的全部数据和断言不变；其余测试函数零改 |
| import_boundaries_test.go / TestProductCatalogPersistenceGuardRejectsEverySiblingProductDomainFixture + 对应 nonadapter.go | 两处都只把 `task-processor/internal/product/asset/recipe` 换为仍合法负例的 `task-processor/internal/product/asset/future` | 八样本数量/顺序对应、所有其它路径、len/逐项精确断言、findNonCatalogProductDomainImports/AST/parser/失败处理不变；仍检验 asset 嵌套子域，不删负例、不加 missing-skip |
| listingkit_backend_retirement_test.go | 只追加 TestLegacyAssetSourceOpenAITailClosureIsRetired，复用原 trackedFiles/AST/decoded-import/decl/helper；不改任何原函数/表/期望 | 精确根/源缺席；全部 tracked Go 禁入旧 recipe import（含子路径/fixture/debug/转义）；原 openai owner 中17 aliases/error/interface/旧版本字段条件不得重建；sourcing 中旧包装声明不复活；正向保护 catalog.Publisher、ai 原合同、配置声明/当前Manager、SHEIN/Store原入口。不能全局 ban Publisher/Template/Recipe 同名 |

新增 guard 要检查 types.go 中 ClientConfig/NewClientConfig/PoolConfig 的原 AST 形状及 internal/ai 原值合同保持；可复用已有 AST helper，不开发新 runner/验证框架。原永久 guard、迁移安全断言、Architecture/Phase3/Admission/depguard/inventory 和其余检查全部不变。实际发现表外原检查与批准删除冲突时，先查齐同根路径并报告准确最小修订，不先放宽/skip 或推定用户已授权。

必要开发自检限原相关包：internal/ai、integration/openai、product/sourcing、product/catalog、app/worker/imageagent 的受影响原测试；tests 中本批退休/导入边界、Issue30/原review invocation相关测试。按风险复用现有 Manager 路由/组织/漂移与图像未发送测试；不调用真实 provider/浏览器/DB/Redis。入口仅编译 current-application、shein-login-worker、原 replay-sale-attribute debug；需要的其它原必需检查交给稳定候选 CI，不人工重复完整 CI。此处仅拟定验证，不把尚未运行写成 PASS。

## 7. 设计评审、确认与停止点

1. 当前只提交三份设计/导航文档，原准确设计 HEAD 的 CI 与适用独立设计复核分别记主要 PR；设计 CI 不证明源码已退休，不证明用户/运行验收。该三文档设计状态为 In Review；正式实施保持 NOT_READY。
2. 独立设计复核检查真实 consumer、三组完整闭包、旧 KEEP 替代、§4.2 与 §6.1，不重开无关架构。正常最多两轮；finding 先按 AGENTS 分类，只有命中当前 Must 的真实 BLOCKER 才阻塞/重开；非阻塞实现验证问题收敛在同批。未经用户本轮授权不默认新增 AI 互验线程；设计复核和未来源码最终复核的授权分开。
3. 用户确认 §1–§7（含 §4.2/§6.1）、适用设计复核完成并读取 Issue/PR 的 IMPLEMENTATION_READY 之后，才允许同一主要 PR 实施全部 39 路径。发现真实当前消费者、外部义务或持久化边界与本文不符，停止对应部分报告用户，不包装旧 Service、不造兼容 alias/双读/第二 owner。
4. 正式实现后必要自检、准确 HEAD 原 CI、一次集中源码最终交付检查分别留证；范围内修复只复核相关增量，不重复检查未变代码。实现者不签产品验收；运行/平台/provider/用户验收保持 NOT_RUN。
5. 合并须用户准确 HEAD 的另行授权，expected-head 正常操作；合并后另查实际 merge SHA 的完整 event=push/head_branch=main CI。部署/关单/运行操作仍未授权，#301 OPEN，#29 不重开。

本轮明确不读现有凭据/DSN/token，不启动/停止/连接/探测既有实例、浏览器、数据或进程，不创建运行资源，不访问已有 Prompt/Store/轨迹文件。当前能力、catalog.Publisher、独立 SHEIN 登录与 Store 同步保持，不增加内部试用前置。
