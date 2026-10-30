# #301 旧认证路由推断、权限尾部与 S3 对象操作完整退休设计

## §1 这批具体做什么

维护者以后只维护当前路由的显式身份/组织/权限合同，以及当前对象存储的不可覆盖合同。本批把以下两个闭包放在同一主要 PR，不按文件数或内部模块机械拆批：

| 退休闭包 | 当前能力继续归谁 |
| --- | --- |
| 根据旧 Module/URL/HTTP method 猜认证或 permission；已退出 Prompt 的写权限、授权条目；两个闲置全局管理员便利函数 | `httproute.Descriptor` 的显式合同 → 当前 HTTP 认证/组织解析 → 当前 `authz` 实例 |
| 旧可覆盖上传、批量上传、带 metadata 上传、Delete/Exists、时间戳 key 与猜类型 helper；只为这些方法存在的 S3 Logger、五处注入及空传参尾部 | 当前 `integration/s3` immutable put/inspect/bounded read/copy/public URL → 当前领域存储 port |

**本轮交付仅三文档，Design Basis: N/A / Ready。未来正式实施采用 Independent Architecture / DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。** 本设计必须完成适用独立复核、原设计 CI 和用户确认 §1–§7（含 §4.2、§6.1）后才可标为 IMPLEMENTATION_READY；现在不修改生产、测试、配置、workflow 或 inventory。

调查基线 main `ed129d0d73c79a6125bb4bd2fbc82c6be033c6a2`；#1017 的实际 push/main CI41043650614 attempt1 整 run SUCCESS。接单：[当前 #301 设计授权与单 Writer 记录](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6521373146)。分支 `codex/issue301-auth-storage-tail-design`，复用本会话独立工作区，一个 Writer、一个主要 PR；接单时 open PR 为空，已核对活跃聊天和 worktree，不接管其它 owner。

使用者与当前价值：维护现有采集、图像、知识库、生态/供应文件和 Store 观察同步的贡献者。交付是可准确确认的删除边界，解决已经退出的功能仍被当作认证依据、旧可覆盖/吞错对象接口仍可被新消费者误用的问题；没有声称当前业务 happy path 被闲置代码阻塞，不新建产品能力或验收前置。

Must：现役认证/租户/组织/角色/撤权/权限、凭据保护、当前采集/图像/JSON/hash/版本、资金/事务/幂等/UNKNOWN/恢复和 Store 观察同步不变。Should：只维护真实合同，原样复用成熟 Gin/Casbin/AWS SDK 与既有 owner。Out of Scope：workflow、完整 inventory/CURRENT/历史/计数/ceilings、保护规则、依赖与工具升级；既有实例、数据、凭据、配置/token/profile/文件或进程；部署、真实平台操作、Issue 关闭、新内部试用前置。无新增 Accepted Risk；#301 始终 OPEN。#985 已退独立 SHEIN 登录/身份回写链不恢复。

## §2 Design Basis、Authority 与前批 KEEP 替代

依据用户本轮明确范围、[AGENTS](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[项目边界](project-boundaries.md)。当前存储依据 [图像恢复与持久化](legacy-imageagent-manual-runtime-closure-retirement-boundary.md)、[容量合同](enterprise-storage-capacity-v1.md)、[生态文件](ecosystem-services-v1.md)，当前身份依据 `httproute` / `authidentity` / `authz` / `workbenchcontext` 与现有原测试。本轮无 UI/IA/交互增量，Product/Figma Authority **N/A**；不引入新的 Threat Model。

采用 Independent Architecture 的原因是退出旧授权推断、缩减内部对象副作用合同与贯穿当前工厂签名，完整未来范围38路径，须独立核对实际当前路径；不是新增 IAM、存储平台、事实 owner、状态机或恢复协议。设计阶段只改文档不等于已经得到源码实施准入。

Legacy decision: RETIRE。
Reusable behavior: 当前显式路由解释、Verifier/Casbin/组织解析、S3 immutable/严格读与领域存储 owner 原样保留；有效测试接回现役实例。无需新增生产 EXTRACT、wrapper、alias 或 fallback。
Current owner: §3 的调用链；现役事实与生命周期由原 owner 持有。
Cutover/deletion condition: 本设计获确认、所有当前消费者和有效断言保持、§6.1 精确转换、必要开发自检及准确源码候选原 CI/最终复核完成；正常合并另授准确 HEAD，运行与关单分别授权。

| 前批 KEEP / 历史依据 | 本批确认后仅替代 | 继续有效 |
| --- | --- | --- |
| [#1017](legacy-http-auth-support-closure-retirement-boundary.md) §4.1 保留 `RouteRequiresZitadelAuth` 当前检测 | 保留其显式 VerifiedIdentity 检测，退出 Module/旧 URL 推断；不是删除当前认证函数或留空 protected branch | OPTIONS、当前 public/identity-only/verified roles、组织准入、headers、错误/审计、deadline/取消 |
| [#885](legacy-sds-runtime-closure-retirement-boundary.md) 保留 SHEIN health、旧 SMS 和其它 Module/permission | 这些已退注册器不再签发认证/权限；旧 admin Module/method permission 推断同退 | 当前 SMS 的显式 public+原 HMAC、Store/当前业务 permission 与组织授权 |
| [#923 Prompt 退休](legacy-tenant-prompt-closure-retirement-boundary.md)、[#985 登录链退休](legacy-shein-login-identity-sync-closure-retirement-boundary.md) | 收齐失去业务消费者的 Prompt permission、policy 和路由识别尾部；不恢复任何旧管理 API | 当前 AI/prompt 使用、官方 Store 授权及 Store 观察同步 |
| [#893 日志门面退休](legacy-logger-facade-closure-retirement-boundary.md) 的 S3 Logger/method-set/SSA 保留 | 仅 S3 专属 Logger/字段/注入/生产 SSA rule；OpenAI/GRSAI、Logrus/当前全局日志不退 | constructor inventory、SSA/typed-nil/decoded parser 算法、其它 provider logging 断言 |
| [旧认证设计](../superpowers/specs/2026-08-25-zitadel-authruntime-design.md)、[旧抽取计划](../superpowers/plans/2026-08-25-zitadel-authruntime-extraction.md) | 不再要求已退出 Module/URL/全局管理员 helper 的旧行为 | 当前中立身份与 provider verification，当前安全错误和取消 |
| [Phase 2 迁包计划](../superpowers/plans/2026-08-30-internal-target-architecture-phase2.md) 的 S3 全接口/Logger | 不保活旧 Upload/Delete/Exists/key/helper/Logger；现役 SDK client 与 immutable path 仍在原包 | AWS/COS 资格、签名、精确缺失分类、校验、URL、容量与原安全边界 |

§5 的历史文档仅追加准确 supersession 顶注，原正文、批准/失败证据、计数、勾选完整保留。其它历史材料不能恢复本批已取消要求，也不能由本表推导其它删除权限。

## §3 实际消费者、注入与 owner

### §3.1 认证与权限

当前链：`cmd/current-application → currentapplication → NewCurrentApplication/WithOptions → 当前 builder/modules 的 httproute.Descriptor → routeAuthHandlersWithDependencies → workbenchAuthenticationMiddleware(Verifier) → 原组织解析 → NewRouteRoleMiddlewareWithAuthorizer[AndResponder] → 同一 authorizer instance → handler`。

`RouteRequiresZitadelAuth` 的唯一生产调用在 `internal/app/httpapi/server_auth.go`；`routeRequiredPermission` 仅被同文件当前 role middleware 使用。保留这两个现役函数及 middleware，删除它们的旧推断部分。生产注册器不再产生 `listing-kit`、`listing-kit-admin`、`listing-kit-platform-admin`、`listing-kit-prompts`、`shein-login`、`product-sourcing`、`local-agent`、`crawler-1688`、`image-agent`、`images` 这些旧 Module 名的路由；`images` 等同名字面值在商品 JSON/模块目录仍可能有效，绝不做全仓词禁令。

当前生产例子：ProductAgent/AIWorkbench/图组使用显式 VerifiedIdentity/Permission/组织策略；内核目录、身份管理等使用 CurrentIdentity 或 CurrentIdentityWithVerifiedRoles；当前 SMS `zitadel-sms` 和 TikTok lifecycle callback 的完整显式 public 描述由 `current_application.go` 精确准入。这些描述和准入不改。旧 Module 名在当前 server 测试中仍可作为反例/无权威标签存在，但须服从显式合同。

`PermissionListingKitPromptWrite` 只剩 `authz/listingkit.go` 定义、三条固定 policy、两段配置管理员赋权和旧 route case；无当前后端/前端 handler 消费。`IsListingKitPlatformAdmin` 无生产/测试调用；`IsListingKitTenantAdmin` 仅本包四断言测试消费。`DefaultListingKitAuthorizer` 仍被管理员模块目录等当前代码调用，实例 `IsTenantAdmin` 被 Store/观察、审核和资金准入调用，**两者 KEEP**。

### §3.2 S3 对象能力

现役五个生产 constructor 保持一处一调用：

| 生产入口 | 当前存储 port / 事实 owner | 本批只退出的专属尾部 |
| --- | --- | --- |
| `cmd/current-application/knowledge.go` | `NewKnowledgeStore → Knowledge` 私有不可覆盖对象 | Logger option/adapter、prepareKnowledge 及 NewKnowledge hook 的空 logger 参数 |
| `cmd/current-application/ecoservices.go` | `NewEcoservicesStore → Ecoservices` 文件事实/支付仍原 owner | 同上，prepareEcoservices/NewEcoservices 的空参数 |
| `cmd/current-application/supply_market.go` | `NewSupplyMarketFiles → SupplyMarket` 私有文件 | 同上，prepareSupplyMarket/NewMarketStorage 的空参数 |
| `app/runtime/currentapplication/source_media_storage.go` | `SourceMediaStorage → collection/productsourcing` 原来源素材事实 | Logger option、NewSourceMediaStorage 空参数/仅此参数的 nil 哨兵、runtime/fixture 传参 |
| `app/worker/imageagent/dependencies.go` | `workerArtifactStore → ImageAgent objectstore`、当前图组材料读取 | Logger option/component entry、纯对象工厂的空参数以及 BuildArtifactStore hook/调用/fixture |

`S3ObjectAPI` 继续只封装 AWS SDK 原 request boundary；Put/Head/Get/Copy 仍有真实消费者。Uploader 的 Upload 仅被 UploadMultiple 自身调用，UploadMultiple/UploadWithMetadata/Delete/Exists/GenerateUniqueKey 无生产消费者；generateKey/detectContentType 仅服务旧 batch。唯一真实日志调用均在这些退休方法内，immutable/read/copy 没有消费 S3 Logger；因此不是把现役日志静默换 noop。

当前 `UploaderOptions` 五处创建不变，去掉 Logger field 后不能用 wrapper/空 logger adapter 维持旧形状。三个 CLI 工厂的 Dependencies function type 和四条 worker/source-media 工厂传参链收齐，不能留只转发日志的空链。`cmd/current-application/main.go` 的函数值绑定不用变化；仍被真实 AI、generation/recovery、runtime/平台日志使用的 logger 参数和字段原样保留。

`S3ObjectAPI.DeleteObject` 可在旧 Delete 退出后移除；这不改变 AWS SDK、本地存储权限、知识库现有集成测试直接使用 SDK 验证拒绝删除的安全断言，也不新增/执行任何对象清理。

### §3.3 所有权与持久化边界

身份、Organization grant、permission、原 token/凭据的 owner 不变；本批不读取其真实内容。当前 DB/事务/账本/容量/metadata/对象 intent、对象 key、hash、JSON、版本、receipt/UNKNOWN/recovery 均在原 owner。没有新表、Goose/schema/grants/migration、双读双写或第二事实源。删除旧时间戳 key API 不改任何当前内容寻址 key，更不处理已经存在的对象。

## §4 目标合同与安全边界

### §4.1 认证不再从旧 Module/URL 猜测

1. `RouteRequiresZitadelAuth` 只解释当前 `AuthPolicyVerifiedIdentity`；旧10 Module 名、SHEIN health 例外和旧 SMS module/path 例外整批退出。显式 public 原样保留；CurrentIdentity 两模式与组织策略仍由原 server 处理，不把函数改为“一律不认证”。
2. `routeRequiredPermission` 只返回现有 `strings.TrimSpace(route.Permission)`，删除旧 admin/prompts Module+method 回退。不新增默认 permission，也不根据 path 再猜。
3. `server_auth.go`、Descriptor、当前注册器/准入及原 role middleware 的实现保持。OPTIONS/public、需组织即认证、身份清理/verified roles、组织解析/role 先后、权限/撤权、error responder、audit、headers、body/request deadline/cancel 均不改变。不存在用删除 fallback 为缺省描述新开公开业务路由的授权；遇见实际当前 producer 仍依赖旧推断，按§5.2先报告，不临时加兼容分支或扩大路由改造。
4. 移除 Prompt 常量、三条固定 policy 和配置 role/subject 各一段赋权；原其余 policies、platform-admin 配置、当前菜单/模块和 default/instance authorizer 均保持。旧 permission 字符串即使由调用者提交也不能获得授权。
5. 两个未被当前生产调用的全局管理员便利函数整退。原四条租户管理员断言接到现有 `NewListingKitAuthorizer(nil,nil).IsTenantAdmin`，输入/预期不变。平台管理员与租户管理员语义、实例隔离绝不合并。

### §4.2 对象副作用、JSON/hash 与完整尾部

- 整退 Uploader 八方法：Upload、UploadMultiple、UploadWithMetadata、Delete、Exists、generateKey、GenerateUniqueKey、detectContentType；移除 S3ObjectAPI.DeleteObject、Uploader.logger、UploaderOptions.Logger、整个 logger.go 及专属注入。旧覆盖上传和 Exists 吞错不再有当前调用面，不给它们另建安全包装。
- KEEP：NewClient/NewKnowledgeClient、NewUploaderWithOptions 及 nil/typed-nil/bucket 校验；PutImmutable、InspectObject、ReadObject、CopyImmutable、PublicURL/resolveObjectURL 和全部有效纯 helper。SDK 替身仍满足剩余四方法；AWS/COS headers、不可覆盖资格、checksum/size/type/maxBytes、typed not-found、copy source escaping、路径/URL 规则和原错误传播不改。
- 工厂日志尾部完整退出：三个 prepare/New hook 的 Logger 参数；NewSourceMediaStorage 的 Logger 参数与只因它存在的 nil 校验；buildImageAgentImmutableObjects、buildImageAgentDurableArtifactStore/BuildArtifactStore、NewImageSetMaterialReader 的 Logger 参数及所有列明调用/fixture。原 runtime 全局 logger、AI/generation/recovery/能力日志保持，不能据此删掉这些真实消费者的 logger。
- 只缩减这些内部 Go 技术合同，未变化字段/返回值/存储选项原样；不改变当前 wire payload、canonical JSON/hash/版本、账户/组织身份、对象 key/digest/大小、数据保存、DB pool/事务或资金归属。NewSourceMediaStorage 原资格/URL/模式检查保持；logger 不再是存储能力，不能再创建装饰性的 nil-log 准入门槛。
- 不引入 provider mutation、自动重试、删除器、补偿或新的 UNKNOWN owner；原 immutable retry/readback/失败分类与取消恢复保持。CI/fixture 不是运行验收，真实对象/实例/凭据/权限/数据清理均未授权。

## §5 完整删除/保留矩阵

未来同一批 **38路径 = 13生产 + 15测试 + 10文档**；目前只改 D1–D3 三文档。仅 logger.go 整文件退出，其余按精确声明/字段/调用混合退休，不整包删除 authz、S3、runtime 或 worker。

### §5.1 生产路径（未来）

| ID | 路径 | RETIRE / 保留 |
| --- | --- | --- |
| P1 | `internal/authz/httpapi/route_authorization.go` | 仅两个函数的旧 Module/path/method 推断；保留显式合同、三个 role middleware/responder、组织授权及错误处理 |
| P2 | `internal/authz/listingkit.go` | Prompt permission及3固定/2配置赋权、两个全局 helper；其余 Casbin/default/实例/模块/权限原样 |
| P3 | `internal/integration/s3/uploader.go` | §4.2八方法/三个成员或seam与独占 time/filepath imports；KEEP 当前 immutable/读/copy/URL/构造/错误全部实现，只校准 PublicURL 的旧 Upload 注释 |
| P4 | `internal/integration/s3/logger.go` | 整文件 RETIRE，不留 noop/adapter/type alias |
| P5 | `cmd/current-application/knowledge.go` | 仅 Logger option/adapter及工厂 logger 参数/独占 import；真实存储/DB/parser/service不变 |
| P6 | `cmd/current-application/ecoservices.go` | 同上；支付/加密/DB/NonPaymentOnly不变 |
| P7 | `cmd/current-application/supply_market.go` | 同上；私有文件资格和当前 factory不变 |
| P8 | `internal/app/runtime/currentapplication/source_media_storage.go` | 仅旧 Logger option/adapter、参数/nil哨兵/import；其余 ValidateSourceMediaStorage/存储合同保持 |
| P9 | `internal/app/worker/imageagent/dependencies.go` | 仅对象工厂三签名/BuildArtifactStore type+调用、componentLogger block/option；保留其它logger、AI/role/资格/时序/lease/DB/恢复 |
| P10 | `internal/app/runtime/currentapplication/runtime.go` | 仅 NewKnowledge/NewEcoservices/NewMarketStorage hook的末尾 logger 参数及3调用，和 NewSourceMediaStorage 一调用；其它生命周期、日志、owner pool/关闭/准入不变 |
| P11 | `internal/app/worker/imageagent/image_set_runtime.go` | 仅 buildImageAgentDurableArtifactStore 一调用去 logger；当前 generation/recovery仍用原 logger |
| P12 | `internal/app/httpapi/current_application.go` | 仅 NewImageSetMaterialReader 一调用去 logger；所有 route/owner/权限/存储准入不动 |
| P13 | `internal/app/httpapi/image_set_application.go` | 仅同一材料 reader 一调用去 logger；其余计划/预算/权限/worker/current owner不动 |

### §5.2 文档路径与清单外发现

| ID | 路径 | 允许范围 |
| --- | --- | --- |
| D1 | `docs/architecture/legacy-auth-route-s3-operation-closure-retirement-boundary.md` | 本设计/冻结基线与准确批准例外；滚动SHA/CI/评审留PR |
| D2 | `docs/architecture/README.md` | 本批当前段及正式backtick文件索引；原正文后缀逐字保持 |
| D3 | `docs/refactoring/legacy-register.md` | 本批当前段；原历史/数字/勾选后缀逐字保持 |
| D4 | `docs/architecture/project-boundaries.md` | 未来仅追加当前auth/S3边界顶注；原正文后缀保持 |
| D5 | `docs/architecture/legacy-http-auth-support-closure-retirement-boundary.md` | 未来仅顶注替代旧推断KEEP；当前显式检测/Verifier/transport保留 |
| D6 | `docs/architecture/legacy-sds-runtime-closure-retirement-boundary.md` | 未来仅顶注替代旧SHEIN/旧SMS/Module权限推断KEEP，其它原禁令保持 |
| D7 | `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 未来仅顶注替代S3 logger/生产rule的精确KEEP；其它provider/算法/检查保持 |
| D8 | `docs/superpowers/specs/2026-08-25-zitadel-authruntime-design.md` | 未来仅顶注限定退出的旧推断/便利函数；原文保持历史 |
| D9 | `docs/superpowers/plans/2026-08-25-zitadel-authruntime-extraction.md` | 同上，不恢复旧兼容实施步骤或历史子代理要求 |
| D10 | `docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md` | 未来仅顶注替代旧S3全接口/Logger保活；历史计划/勾选/计数完整保持 |

若出现清单外真实消费者、失败断言或当前外部/持久化兼容义务，先停止该部分，提供实际路径/根因/最小准确补丁与边界影响供用户决定；不能先删再倒推设计、加fallback或偷偷改检查。实际source或业务安全冲突按AGENTS分类，仅新BLOCKER重开架构；不据纯理论场景追加平台或无限复核。

## §6 验证与验收方法

设计期仅检查三文档差异、索引/相对链接/状态、原文后缀、矩阵准确性、实际消费者和原设计CI；无生产测试、运行/数据库/对象操作。源码实施获得确认后，先使聚焦闭包guard及旧推断/旧Prompt赋权反例真实RED，再完成最小删除/当前owner接回使其GREEN。无需新runner或故障注入平台。

未来必要开发自检限定受影响 authz/httpapi、authz、S3、HTTP transport/currentruntime/worker 与精确原guard；相关认证/取消/并发适用原race。当前命令编译，原 integration/minio/Temporal fixture 编译；未执行的环境用例保持NOT_RUN。不连接既有实例，不额外建设临时PG/MinIO/Temporal平台；原CI按既有配置执行，不能为了过关改workflow/inventory。

验收关注用户可见行为仍可完成：当前登录/组织权限和Store观察准入不变；现有来源素材/图组/知识等存储继续走不可覆盖/严格读取，UI与保存JSON/hash无变化。源码/CI/另一AI复核不等于产品验收，真实运行与产品验收仍由用户另授；#301不关单。

### §6.1 原检查与 fixture 的精确转换（未来）

仅以下15测试文件可变；未列函数/断言/算法零改。机械参数删除只清理随之未使用的 import；不放宽 flags、期望错误、pool/事务/资金/取消/UNKNOWN 或安全值，不加 missing-root skip。

| ID | 测试路径 | 准确允许的转换；必须保持 |
| --- | --- | --- |
| T1 | `internal/authz/httpapi/route_authorization_test.go` | TestRoutePolicyPreservesVerifiedIdentityAndExplicitPublicAuthority 的 image-agent case显式设置VerifiedIdentity；追加无Module/path推断及显式Permission优先/无回退反例。TestRoleMiddlewareUsesVerifiedContextAndOriginalPermissionPolicy 原伪造headers/未授权/当前admin输入及403/204全部原样 |
| T2 | `internal/authz/listingkit_test.go` | TestListingKitAuthorizerRestrictsPromptWritesToTenantAndPlatformAdmins 转为旧Prompt permission对固定/配置role/user均无grant；原两个拒绝保留，原旧正向不再保活。TestIsListingKitTenantAdminIncludesTenantAndPlatformAdmins 的四断言改调现役实例IsTenantAdmin，值/预期原样。其余platform/tenant/admin/当前业务矩阵不动 |
| T3 | `internal/integration/s3/uploader_test.go` | RETIRE TestUploadMultipleRejectsEmptyInputWithoutPanic、TestUploaderUsesConfiguredStructuredLogger、TestExistsPreservesLegacyPlainHeadAndErrorSemantics 及仅第二项用的recordedLog/recordingLogger/methods；删 fakeS3API/nilSliceS3API 的DeleteObject方法。nil/typed-nil/bucket、URL、typed404/错误、checksum/COS policy/headers/copy转义等其余全部原样 |
| T4 | `tests/integration_provider_logger_contract_test.go` | 仅 TestIntegrationProviderLoggerMethodSetsMatch 删除S3一项及失效import；OpenAI/GRSAI完整method-set/签名、所有其它函数/算法原样 |
| T5 | `tests/provider_logger_ssa_test.go` | 仅 productionProviderLoggerRules 删除真实task-processor/internal/integration/s3的一条rule；其它生产rule、SSA算法/加载/typed-nil/控制流失败关闭不变。TestProviderLoggerSSAMustBoundUsesSecondS3OptionsArgument、example/root/s3合成model及fixtureProviderLoggerRules保持，作为ConfigArgument=1的纯分析器正负覆盖，并非真实S3当前合同或生产兼容层 |
| T6 | `tests/issue398_acquisition_admission_test.go` | 仅 issue398AcquisitionAPIViolations 的source_media_storage.go current API准确集合删除AdaptLogrus一项；其它允许项、ceiling=8、root/OS/build-tag/decoded处理、负例/失败关闭原样 |
| T7 | `tests/listingkit_backend_retirement_test.go` | 仅新增一项 TestLegacyAuthRouteS3OperationClosureIsRetired：精确检查旧八方法/两个全局helper/Prompt声明赋权/Logger file-field-seam缺席，当前显式检测与immutable/read/copy/URL存在，实根解析失败关闭。复用既有AST/file index，不新增宽泛字符串禁令；原所有guard保持 |
| T8 | `internal/app/httpapi/server_test.go` | 仅新增当前显式认证不依赖旧Module名的transport用例，验证currentIdentity/verified/组织/角色与public；已有包括TestCurrentSMSWebhookDoesNotRelaxProtectedImageRoutes、header清理/组织不可伪造、顺序/双server/审计/撤权/stream/deadline全部原样 |
| T9 | `internal/app/runtime/currentapplication/source_media_storage_test.go` | 仅 testStandaloneSourceMediaStorage 的一调用去logger参数/import；原两测试的签名header/If-None-Match、单次put、原key/hash/授权/受限读取/取消断言原样 |
| T10 | `internal/app/runtime/currentapplication/ecoservices_test.go` | 仅 TestEcoservicesAndNotificationRuntimePoolsStayIndependent、TestEcoservicesRuntimeReleasesEachOwnerOnceBeforeConstructingAliasedFeature 中NewEcoservices/NewKnowledge三个fixture签名去末logger；原独立pool/释放一次/错误/金融准入不变 |
| T11 | `internal/app/runtime/currentapplication/supply_chain_test.go` | 仅 TestSupplyAndEcoservicesRuntimePoolsStayIndependent 的NewEcoservices一个fixture签名去logger；原pool/拒绝/关闭全部保持 |
| T12 | `internal/app/runtime/currentapplication/supply_market_runtime_test.go` | 仅 TestPODRuntimeAssemblyFailureClosesCanonicalPools 的NewMarketStorage一个fixture签名去logger；原POD/owner/关闭断言保持 |
| T13 | `internal/app/worker/imageagent/dependencies_test.go` | 仅六处BuildArtifactStore fixture签名及TestBuildImageAgentDurableArtifactStoreUsesConfiguredS3ClientPath的一调用去logger。对应函数：TestResolveOrganizationWorkerComposesGovernedSingleMainSlotAndLiveAuthorizer、TestTrialOrganizationWorkerVerifiesDedicatedRoleBeforeProviderConstruction、TestResolveImageAgentTemporalDependenciesForOrganizationFailsBeforeDatabaseOnInvalidPolicy、TestResolveImageAgentTemporalDependenciesForOrganizationFailsBeforeDatabaseWhenArtifactModeIsAbsent、TestResolveImageAgentTemporalDependenciesRejectsCredentialsBeforeDatabaseOpen、TestResolveImageAgentTemporalDependenciesRejectsOperationTimeoutOutsidePublicationLeaseBeforeStoreOrDatabase；原六函数的全部有效准入/原锁/凭据/时序/取消断言不变 |
| T14 | `internal/app/worker/imageagent/minio_compat_test.go` | 仅 TestIsolatedTrialMinIOPreservesRecoversAndPublishesImmutableAsset 一调用去末nil日志参数；原不可覆盖/恢复/发布/SDK效果与env未提供时原skip不变，本地不执行真实环境 |
| T15 | `internal/app/worker/imageagent/organization_main_temporal_postgres_test.go` | 仅 TestOrganizationWorkerRealTemporalUnpricedGenerationDoesNotReserveOrDispatch 的BuildArtifactStore一个fixture签名去logger；无预占/无dispatch/恢复及原全部安全断言不变，仅编译或原CI自有环境执行 |

显式零改的检查还包括：`tests/s3_integration_architecture_test.go` 的五构造器精确inventory与legacy consumer冻结、全部provider_logger_ssa_*算法/变异/合成控制流（除T5生产rule）、SDK import guard、architecture/import/原退休guard、Code Health逻辑、完整inventory/CURRENT/历史/ceilings/accuracy/parser、workflow、依赖与保护。currentapplication/runtime.go三个hook签名仅技术尾部，不删掉现役pool/当前真实logger。所有同根消费者已在矩阵列出，实际遗漏仍遵守§5.2先报告。

## §7 复核、确认与停止点

本轮只形成三文档设计提案，独立设计复核聚焦：实际当前route显式合同是否完整、不因退出推断而公开业务路径、旧permission赋权是否收齐、S3 immutable/read/copy与日志尾部是否完整、空传参链及§6.1是否精确、原持久化/资金/UNKNOWN/Store边界是否保持。按AGENTS先分类finding，最多两轮正常设计复核，只有真实BLOCKER重开架构；不追加全局架构或验收工具。

**设计与源码最终复核分别绑定准确HEAD；未经用户本轮授权不默认新增AI互验线程。** 先使用原GitHub适用复核；未完成时不重复触发，同一失败最多一次重试。需要独立子代理时须本轮授权，只读核对真实diff/消费者/已有证据，不写代码、不跑测试构建、不操作既有资源、不重复GitHub评审；若原适用复核先完成且无待处理问题则不新增。

原设计CI与适用设计复核完成后，交用户确认本设计§1–§7（含§4.2、§6.1）和38路径完整闭包；确认前禁止生产/测试实施。源码阶段一次完整交付检查，准确HEAD/原CI run-attempt/复核/授权维护在同一PR；不把未变化代码已有证据冒充新SHA实测。合并、实际merge SHA完整push/main CI、部署、真实运行验收、Issue关闭分别授权/记证；#301保持OPEN，不增加内部试用前置。
