# #301 旧 HTTP 备用装配、白名单认证与辅助尾部完整退休设计

## §1 这批具体清理什么

维护者今后只维护实际在用的 HTTP 与身份装配。本批一次收齐三组，不按文件数拆 PR：

| 退出的旧链 | 当前继续使用的能力 |
| --- | --- |
| 备用 HTTP 装配、未启用的旧 health/启动计时/Options、独立 SourceAccount/审计/SHEIN record 启动门面、旧 Store/SMS 装配 | current-application 统一应用；当前来源账号、采集、审计、草稿、Store 观察与当前 SMS |
| 仅旧装配调用的租户/用户/角色白名单中间件，以及专属全局配置 | 当前 Zitadel Verifier → 当前身份 → 组织解析 → permission；当前平台管理员配置 |
| #1010 后失去最后消费者的 goroutine 整包、简单 HTTP factory、短 AI timeout | 实际 worker、Chromium 安装 HTTP、Download 两个 timeout、当前 deadline/取消 |

**当前仅三文档设计：Design Basis: N/A / Ready。未来源码：Independent Architecture / DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。** 独立设计复核及用户确认 §1–§7（含 §4.2/§6.1）后，才可标记 IMPLEMENTATION_READY 并在同一主要 PR 整批实施。现在不修改生产代码、测试或配置输入。

调查起点 main `be7f6592a7f9792c1119a98b774624a31a8593e7`；[#1010](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1010) 对应 push/main CI41023989892 attempt1 整 run SUCCESS。接单与边界见 [#301 当前接单](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6518684671)。单 Writer、当前独立工作区、一个主要分支/PR；#1014 的内核发行目录保持原 owner，不接管。后续 main 若推进，仅核对真实组合及实际消费者，不能据旧快照删除新消费者。

设计候选前已同步实际 main `6229cd96c80bfa93cdd91d06c8556ca7f0bb7644`（原 owner 合入 #1014）。组合核对：新增内核目录只调用现有 WithOptions/module 与当前 identity/verified-roles，没有调用本批退出声明；其 BrowserKernel optional feature、metadata/persistence owner、manifest、权限、workflow 和新原检查全部保留。同步不改变本批91路径提案；本 PR 对实际 main 仍只有三文档差异，上游源码/CI调整不是本批 authored 修改。

Must：当前采集、图像、身份/组织/权限、Store 观察同步、JSON/hash/版本、凭据保护、事务、deadline/取消和现有 retry/UNKNOWN/恢复事实 owner 保留。Out of Scope：workflow、完整 inventory、历史记录/计数/ceilings、保护规则、工具与依赖版本；既有实例、数据、凭据、配置/token/profile/轨迹文件或进程；部署、运行验收、Issue 关闭、新内部试用前置。#301 保持 OPEN。不恢复已由 #985 退休的独立 SHEIN 登录与身份回写 worker。

## §2 Design Basis 与旧 KEEP 替代

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)。没有 UI/IA/Figma 增量，不新增 Threat Model 或 Accepted Risk。安全关注仅为本批真实接线退出是否造成认证旁路、丢失当前行为或错误关闭借用连接，不建设新 IAM、runner 或验收平台。

Legacy decision: RETIRE（优先复用现有 owner，不新造兼容层）。
Reusable behavior: 当前 verifier、project roles、模块 registry、路由准入、transport、Store/来源/草稿/采集行为直接保留；仍有效的测试接回这些 owner。
Current owner: §3 的现役调用链与 §4 的 owner。
Cutover/deletion condition: 用户确认本设计、原检查有效转换、必要自检、准确源码 HEAD 的原 CI 与最终复核完成；合并仍另授准确 HEAD。

| 前批 KEEP / 依据 | 本次待确认替代的范围 | 继续有效 |
| --- | --- | --- |
| [#914](legacy-idle-support-contract-closure-retirement-boundary.md)、[#937](legacy-idle-support-facade-closure-retirement-boundary.md)、[#939](legacy-global-config-logging-tail-closure-retirement-boundary.md)、[#969](legacy-support-control-tail-closure-retirement-boundary.md) | goroutine 八声明、NewSimpleWithTimeout、AIShortTimeout/WithAIShortTimeout 不再为已退出的消费者保留 | 当前 logging/httpclient、两个下载 timeout、真实 worker/取消 |
| [#1007](legacy-config-projection-dub-closure-retirement-boundary.md) 与原 mandatory-auth 设计 | 当时保留的四个 Primary allowlist 输入及仅服务旧 SMS builder 的全局输入不再接受；不是把授权开关默认值改为 false | username 拒绝陷阱、当前平台管理员、Verifier/身份/组织/permission |
| [模块映射 SA1 / DRAFT-S1](../refactoring/module-target-mapping.md)、[当前试用边界](issue-36-local-trial-runtime.md) | NewSourceAccountApplication / NewSheinRecordApplication 的独立 HTTP 生命周期门面；独立审计/会员/采集便利构造器同退 | 当前 module/core/routes、schema、记录/资产/Store 共同 DB 假设、默认未开放的能力仍未开放 |
| [#1010](legacy-restored-debug-closure-retirement-boundary.md) | 已退出消费者的最后专属 helper，不重新抽取闲置库 | 当前 extension/公开采集器/Chromium 安装、当前业务路径 |

前批文档保留为历史。只有本设计明确列出的 KEEP 被替代，不能据此扩大删除其它“deadcode”。当前原生身份 CLI `internal/zitadelprovision/cmd` 是实际 package main；测试辅助包和当前领域合同都不因生产静态根未覆盖而退休。

## §3 真实消费者与完整闭包

调查方法：Git 跟踪源码引用、root production import graph，以及仓库已安装 pinned deadcode v0.48.0 对 `./cmd/... ./scripts/...` 的只读分析。静态输出只作为线索，逐一确认调用、注入和测试；没有执行应用、浏览器、provider 或 DB，也没有安装/升级工具。下列“无消费者”只指起点源码生产路径，不声称检查过运行实例。

**当前正式链保持：**

`cmd/current-application.NewApplication` → `NewCurrentApplication` / `NewCurrentApplicationWithOptions` → `buildCurrentApplication` → 现役 module builders → `buildRuntimeBundleFromModules` → `validateCurrentApplicationRoutesInternal` → `buildCurrentApplicationHTTPServer` → `buildHTTPServerFromRoutesAtWithAuthDependencies` / `mountRoutesWithAuthDependencies`。

`buildWorkbenchContextModule` 注入当前 `routeAuthDependencies`，其 Verifier、organizationResolver、authorizer、roleMiddleware、auditRecorder/auditNow 继续使用。整个 `kernel/module.Module/Registry` 与实际模块保持。**NewCurrentApplication 本身有 cmd 调用，必须保留。**

**退出链：**

- `buildHTTPServerBundleFromModules` → 旧 `buildRouteAuthorization` → `runtimeBundle.buildServerBundle` → 三个 server/mount 转发门面。没有正式入口调用。旧 httpModule/newCoreHTTPModule、启动计时、Options、旧 Store factory 和全局 SMS builder 同属闲置装配，不拿测试调用充当产品依赖。
- 七个便利构造器：`NewCurrentApplicationWithBrowserCapture`、`NewCurrentApplicationWithMembership`、`NewCurrentApplicationWithAcquisition`、`NewCurrentApplicationWithAcquisitionAndReferrals`、`NewSourceAccountApplication`、`NewAccountAuditApplication`、`NewSheinRecordApplication`。只有测试/历史文档使用；保留各自当前业务 module/core/routes，测试直接装配当前 owner，不增加生产替代 constructor。
- 七个 route-validator 门面：`validateCurrentApplicationRoutes`、`ForAcquisition`、`ForSourcing`、`WithFeatures`、`WithAccountProfile`、`WithBrowser`、`WithBrowserFeatures`（后三至七项均以 validateCurrentApplicationRoutes 为前缀）。仅测试还调用；测试向现有 Internal 传入与原门面逐项相同的 flags，不改准入算法、optional features 或白名单。
- `buildRouteAuthorization` 是外部唯一调用 `zitadel.NewMiddleware` 的生产声明，但自身不可达。allowlist 和全局 SMS 字段只经旧 core config 加载后供上述旧 builder 使用；当前 SMS 读取现有私密 JSON 文件，并非这份全局配置。
- goroutine 整根无生产 importer；NewSimpleWithTimeout 的最后旧 IP 定位调用和短 AI timeout 的旧 SHEIN 调用已退出。当前安装器使用有效 httpclient 与 Download timeout，保持原入口与数值。

## §4 Owner、JSON 与副作用边界

| 能力/事实 | 当前 owner，保持不变 | 本批效果 |
| --- | --- | --- |
| 身份/token/claims | Zitadel Verifier、ParseRolesForProject、authidentity、当前 workbench context | 移除闲置 allowlist Gin 包装；不读真实凭据，不替代认证核心 |
| 组织/角色/平台管理员 | 当前组织解析、authz.ListingKitAuthorizer、当前 manifest 的 ListingKitAuthorization | 同一当前组织/permission；不从旧 allowlist 派生组织授权 |
| SourceAccount / 审计 / 草稿 | 原当前 domain/repository/schema/core/module | 只退独立启动门面；不动 JSON、规范化、hash 算法/版本、四个 SourceIdentity 零值槽位及读取拒绝 |
| Store 与观察 | buildCurrentStoreCenterModule、官方连接、member-scoped repo、当前 observation owner | 不使用旧 buildStoreCenterModule 的 cfg.Database；实际 Store 同步与官方授权继续 |
| 费用/账本/恢复 | buildCurrentResourceCharges / WithOwners 及现有持久化 owner | 只退 buildProductResourceCharges 一层转发，资金/扣费/UNKNOWN/恢复不改 |
| SMS | buildZitadelSMSModule → 私密配置 → 原 TencentSender/Service | 旧全局配置不再成为输入，现役 HMAC、过期值和凭据隔离不改 |
| 生命周期/transport | 当前 runtime 持有 DB/server，module 借用；当前 HTTP server | 不增加 DB owner、listener、shutdown loop、状态机或外部写入 |

没有 schema/migration/table 变化；没有读/写/删除既有行。没有迁移、双读/双写、fallback、alias、全局 auth 缓存、第二事实源或后台补写。现有业务事务、幂等键、retry/响应丢失/UNKNOWN/recovery 不经过本批新协议；全部沿当前 owner。

### §4.1 认证与 HTTP 不变量

1. 当前 protected branch 统一追加**既有** `workbenchAuthenticationMiddleware(dependencies.workbenchVerifier)`；移除 `identityMiddleware` 字段及旧 else-if，不能留下“需要认证但没 middleware”的空分支。保留 OPTIONS/public 判断、`RouteRequiresZitadelAuth` 当前检测合同和所有实际路由准入。无 token→原401，nil verifier→原依赖失败，坏 claims/上游失败→原分类；不放宽断言。
2. 当前 identity-only 的租户/组织/role 清除、verified-roles、组织 target/live/cached 解析、撤权/取消/审计、permission 顺序和不可信头清除均保持。旧 listing-kits 示例路由不再因测试挂回当前 runtime；原“旧白名单拒绝”改验证旧路由不被准入/挂载，当前受保护路由仍需真实当前认证。
3. 保留当前 loopback:0、ReadHeaderTimeout=5s、Read/Write=32s、Idle=60s；仅当前已准入长操作的125s write deadline 保持。no-store/nosniff、route deadline、先到 parent deadline、cancel、recovery、无响应缓冲均保持。不为旧 isolated wrapper 的任意37s参数或 wildcard constructor 继续保活。
4. 原测试只能把 fixture 接回当前 module/routes + server/auth dependencies；需要测试假的 module 时使用 test-only stub，不能在生产保留 httpModule、兼容 constructor 或 alias。共享 loopback identity fixture 补当前 verifier 必需的合法 exp/project claims，旧缺失/过期/畸形案例仍独立拒绝，不能让 stub 绕过待验证的认证。
5. 只删除构造器，不能删除 current SourceAccount/SHEIN record/审计的合法业务验证。全新 schema 显式准备、无隐式 DDL、不关闭借用 pool、组织隔离、Store/Asset/Product 同 DB、已有草稿不可变与默认未开放边界仍有效。

### §4.2 旧配置输入的准确 hard-cut

删除四个 allowlist 字段及旧全局 `SMS/ListingKitZitadelSMSConfig`、known bindings、defaults、BuildConfig 赋值/专属 expiry parser；删除 current CoreConfig 的装饰性 `AuthorizationRequired:true`。当前 manifest 的 JSON/schema/版本及平台管理员数组不变。

两条正式 loader 继续先走现有 `rejectRetiredProductRuntimeConfig`。仅加入四个精确 YAML key 与 `listingkit.zitadel.sms` 精确 subtree 的存在性拒绝（根、任意子键、空 map/list/string、false、null 都不得静默忽略）。用现有 Viper 的 InConfig/AllKeys 检查这些新增项；不把全局所有 map 改写或扩大拒绝根。22 个环境名按下表精确拒绝，包括赋空值与 Primary/AutomaticEnv 并存；报错只带键名，不含值。此前已拒绝的三个无 TASK_PROCESSOR 前缀 allowlist 名仍拒绝，其算法与其它旧 alias 拒绝不变。

| YAML（listingkit.zitadel 下） | Primary 环境名 | AutomaticEnv 环境名 |
| --- | --- | --- |
| `listingkit.zitadel.authorizationRequired` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHZ_REQUIRED` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHORIZATIONREQUIRED` |
| `listingkit.zitadel.allowedTenantIDs` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWEDTENANTIDS` |
| `listingkit.zitadel.allowedUserIDs` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWEDUSERIDS` |
| `listingkit.zitadel.allowedRoles` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWEDROLES` |
| `listingkit.zitadel.sms.signingKey` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_SIGNING_KEY` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_SIGNINGKEY` |
| `listingkit.zitadel.sms.tencentSecretID` | `TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_ID` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_TENCENTSECRETID` |
| `listingkit.zitadel.sms.tencentSecretKey` | `TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_KEY` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_TENCENTSECRETKEY` |
| `listingkit.zitadel.sms.tencentAppID` | `TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_APP_ID` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_TENCENTAPPID` |
| `listingkit.zitadel.sms.tencentSignName` | `TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SIGN_NAME` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_TENCENTSIGNNAME` |
| `listingkit.zitadel.sms.tencentTemplateID` | `TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_TEMPLATE_ID` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_TENCENTTEMPLATEID` |
| `listingkit.zitadel.sms.phoneVerificationExpiryMinutes` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONE_VERIFICATION_EXPIRY_MINUTES` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONEVERIFICATIONEXPIRYMINUTES` |

KEEP：issuerURL/authorizationAPIURL/clientID/clientSecret/projectID、tenantDirectoryToken/memberInvitationToken、`listingkit.platformAdminUsers/platformAdminRoles`、当前私密 manifest/身份文件、`ZITADEL_SMS_CONFIG_FILE` 及其 JSON/权限/HMAC；imageagent.admission.allowedTenantIDs 和当前组织 admission；所有其它环境输入。不得拒绝整个 LISTINGKIT/ZITADEL 前缀。username 字段和 LegacyUsernameAllowlistConfigured 继续仅作拒绝陷阱；只更新拒绝 Hint 到当前身份/组织权限与平台管理员 owner，不能恢复用户名授权。

当前配置 checklist 删除这 11 个 key 的“可用输入”指引，明确不读取或改动现有配置/凭据、不要求任何实例操作。不是新增旧数据迁移流程或内部试用前置。

## §5 精确工件矩阵与删除条件

完整未来范围 **91 路径 = 32 生产 + 53 测试 + 6 文档**。当前设计提交只有新设计和两个导航，共3文档。生产文件不能整包误删；下面的“仅”是边界，不是授权其它 deadcode 清理。

### §5.1 生产矩阵

| ID | 路径 | 决策 | 精确声明/保留边界 |
| --- | --- | --- | --- |
| P1 | `internal/app/httpapi/http_module.go` | RETIRE 整文件 | httpModule / Name / Enabled / Register；不动 kernel/module |
| P2 | `internal/app/httpapi/http_modules.go` | RETIRE 整文件 | newCoreHTTPModule 及未接入当前应用的 /health；不新增替代健康端点 |
| P3 | `internal/app/httpapi/module_runtime.go` | RETIRE 整文件 | buildHTTPServerBundleFromModules / buildRegisteredRoutesForModules |
| P4 | `internal/app/httpapi/application_http_server.go` | RETIRE 整文件 | buildIsolatedApplicationHTTPServer；有效 transport 断言改接当前 server |
| P5 | `internal/app/httpapi/options.go` | RETIRE 整文件 | Options / defaultShutdownTimeout / shutdownTimeout；不动真实 cmd shutdown |
| P6 | `internal/app/httpapi/startup_timing.go` | RETIRE 整文件 | startupTimer / newStartupTimer / phase / total / log |
| P7 | `internal/app/httpapi/storecenter_module.go` | RETIRE 整文件 | storeCenterBuildResult / storeCenterModuleBuilder / storeCenterFactories / defaultStoreCenterFactories / buildDefaultStoreCenterModule / buildStoreCenterModule / storeCenterStartupError / newStoreCenterStartupError / Error / Unwrap / unavailableConnectionStatusProvider.Status |
| P8 | `internal/app/httpapi/zitadel_sms_composition.go` | RETIRE 整文件 | buildZitadelSMSService；现役 zitadel_sms.go 不删除 |
| P9 | `internal/authruntime/zitadel/middleware.go` | RETIRE 整文件 | middleware.Handle / bearerToken / authorizeIdentity |
| P10 | `internal/pkg/goroutine/goroutine.go` | RETIRE 整根 | Task / Result / ProcessFunc / Processor / NewProcessor / ProcessParallel / worker / executeTask；不另建 workerpool |
| P11 | `internal/app/httpapi/runtime_bundle.go` | 混合 RETIRE | 仅 authDependencies 字段、buildServerBundle；KEEP routes 和 buildRuntimeBundleFromModules 原算法 |
| P12 | `internal/app/httpapi/server.go` | 混合 RETIRE | 仅 buildHTTPServerFromRoutes / buildHTTPServerFromRoutesAt / mountRoutes；KEEP WithAuthDependencies 版本、serverAddress、requestContextTimeout |
| P13 | `internal/app/httpapi/server_auth.go` | 混合 RETIRE + 复用当前认证 | 仅 routeAuthorization alias、buildRouteAuthorization、routeAuthHandlers、withWorkbench、identityMiddleware 字段/旧分支；按 §4.1 统一使用既有 verifier middleware |
| P14 | `internal/app/httpapi/current_application.go` | 混合 RETIRE | 仅七个 validateCurrentApplicationRoutes* 便利门面，见 §3；KEEP Internal 算法、实际 routeFeatures/current server/业务装配/恢复 |
| P15 | `internal/app/httpapi/browser_capture_application.go` | 混合 RETIRE | 仅 NewCurrentApplicationWithBrowserCapture；KEEP 模块/路由/服务、WithBrowserCapture 当前选项 |
| P16 | `internal/app/httpapi/organization_membership_module.go` | 混合 RETIRE | 仅 NewCurrentApplicationWithMembership；KEEP MembershipDependencies、buildMembershipModule、当前邀请/授权 |
| P17 | `internal/app/httpapi/product_acquisition_application.go` | 混合 RETIRE | 仅 NewCurrentApplicationWithAcquisition / NewCurrentApplicationWithAcquisitionAndReferrals；KEEP 实际采集/费用/保存/来源绑定 |
| P18 | `internal/app/httpapi/source_account_application.go` | 混合 RETIRE | 仅 NewSourceAccountApplication；KEEP buildSourceAccountModule(ctx,db,authorizer)、SA1 当前 owner |
| P19 | `internal/app/httpapi/account_audit.go` | 混合 RETIRE | 仅 NewAccountAuditApplication；KEEP buildAccountAuditModule、当前审计读/权限/summary |
| P20 | `internal/app/httpapi/shein_records_application.go` | 混合 RETIRE | 仅 NewSheinRecordApplication；KEEP buildLocalTrialListingRoutes / sheinRecordStoreReader.GetStoreReference、当前本地草稿合同 |
| P21 | `internal/app/httpapi/consumer_resource_charges.go` | 混合 RETIRE | 仅 buildProductResourceCharges 转发；KEEP buildCurrentResourceCharges / WithOwners、所有费用/恢复事实 |
| P22 | `internal/authruntime/zitadel/config.go` | 混合 RETIRE | 仅 AuthorizationConfig / middleware / NewMiddleware / newMiddleware；KEEP Config、IntrospectionResponse、discoveryDocument、normalizeConfig |
| P23 | `internal/authruntime/zitadel/parsing.go` | 混合 RETIRE | 仅 ParseRoles / StringSliceToSet / firstNonEmptyValue / valueInSet；KEEP ParseRolesForProject / parseRoles / addRoleClaimValue |
| P24 | `internal/core/config/type_listingkit.go` | 混合 RETIRE | 仅四个旧 allowlist 字段及 SMS / ListingKitZitadelSMSConfig；KEEP issuer/client/project、当前 token 和 username 拒绝槽位 |
| P25 | `internal/core/config/config.go` | 混合 RETIRE | 仅四旧字段和七 SMS key 的 knownEnvBindings 项；保留其它 Primary/AutomaticEnv、loader 和秘密处理 |
| P26 | `internal/core/config/defaults.go` | 混合 RETIRE | 仅上述 11 项默认注入；保留其它默认值 |
| P27 | `internal/core/config/loader_builder.go` | 混合 RETIRE | 仅四字段局部变量/赋值、SMS 构造、phoneVerificationExpiryMinutes；KEEP username 配置检测和其它 builder |
| P28 | `internal/core/config/retired_product_runtime.go` | 复用现有拒绝 owner | 只追加 §4.2 精确键/环境名，SMS 精确 subtree 的存在性检测；不扩大其它根或改既有检查算法 |
| P29 | `internal/core/config/validator_listingkit.go` | 仅提示校准 | 只把旧 username 拒绝 Hint 改指当前身份/组织权限和平台管理员配置；拒绝条件/Field/Message 不变 |
| P30 | `internal/app/runtime/currentapplication/config.go` | 混合 RETIRE | 仅 CoreConfig() 的 AuthorizationRequired:true 装饰性赋值；KEEP 当前 manifest JSON/平台管理员/身份边界 |
| P31 | `internal/pkg/httpclient/httpclient.go` | 混合 RETIRE | 仅 NewSimpleWithTimeout；KEEP Config / Client / New / Do、当前下载/TLS/超时/取消 |
| P32 | `internal/pkg/timeout/timeout.go` | 混合 RETIRE | 仅 AIShortTimeout / WithAIShortTimeout 及失去使用的 context import；KEEP DownloadTimeout=10m / DownloadLongTimeout=30m |

### §5.2 文档矩阵与范围纪律

| 文档 | 未来允许的准确调整 |
| --- | --- |
| 本设计、architecture/README、refactoring/legacy-register | 当前设计/实施阶段及准确证据；原历史后缀、数字、计数、勾选保留 |
| refactoring/module-target-mapping.md | 在当前段明确 supersede SA1/DRAFT-S1 的两个 standalone constructor；保留 schema/domain/BFF/JSON 和历史验收，不把 NOT_RUN 改 PASS |
| architecture/issue-36-local-trial-runtime.md | 当前段注记 standalone 门面退出；现有 buildLocalTrialListingRoutes / 私有试用组合与共同 DB 边界保持，不能声称已部署或默认开放 |
| development/listingkit-config-migration-checklist.md | 把旧 allowlist/global SMS 迁移操作指引改为退休说明和当前 owner 入口；不要求读取凭据/改实例，不重写其它历史证据 |

必须在真正删除前按最终 main 再核对实际消费者。发现清单外的真实消费者、当前调用、public/JSON/持久化合同、检查转换或 inventory 不一致时，先报告具体引用、最小闭包与差异；不自行扩大、不跳过检查、不用弃用壳代替退休。准确例外由用户决定，再留同一 PR，不机械派生小批。

## §6 验证与交付边界

设计阶段只验证三文档 diff、引用、矩阵/当前消费者映射和原文保留；不为纯文档伪造 RED。原仓库 CI 按现有 workflow 自然运行，不改 CI/检查来迁就设计。

未来实施在本主要分支连续完成三个相互闭合的实现/提交边界：① 当前认证与配置输入 hard-cut 及有效用例；② 旧装配/constructor/测试改接；③ helper、文档和原 guard 收口。不是三个 PR。业务边界测试先 RED→最小实现 GREEN，旧实现细节只退不用制造无意义业务测试。只执行与改动直接相关的现有包/测试与必要编译，完整 CI 留稳定候选。临时 fixture 限既有测试的 httptest/合成配置和任务自有临时文件/内存或 SQLite，不连接既有资源；真实 PostgreSQL/外部依赖若未获任务自有执行授权则 NOT_RUN，由原适用 CI 覆盖，不冒充运行验收。

### §6.1 原检查转换清单

**A. 下表 T1–T42 是旧符号引用所在的精确函数/fixture helper。** 仅把旧 constructor/server/mount/module/validator 的调用改接 §3 当前 owner；原参数、功能开关、场景、身份/组织、存储输入、业务返回值与所有有效断言保持。旧七 validator 的 flags 按原转发逐项展开给 Internal。不能把整文件测试删掉、改成 stub 成功、t.Skip、missing-skip 或缩短超时掩盖失败。未列函数只允许因同文件已移除引用而必要的 import 整理；其它运行/断言原样保持。下面 B/C 是这些函数中的明确语义转换，不是另授权全文件改写。

| ID | 原路径 | 允许改接的原函数/测试 helper |
| --- | --- | --- |
| T1 | `internal/app/httpapi/agent_configuration_integration_test.go` | `TestAgentConfigurationUsesCurrentEnterpriseWithoutTextRuntime` |
| T2 | `internal/app/httpapi/account_read_test.go` | `TestAccountCurrentIdentitySkipsLegacyAndOrganization` |
| T3 | `internal/app/httpapi/account_audit_test.go` | `TestAccountAuditHTTPProjectsImagePointsUnderLiveOrgPermission`、`TestAccountAuditHTTPUsesFreshOrgReadPermission` |
| T4 | `internal/app/httpapi/account_audit_summary_integration_test.go` | `TestAccountAuditSummaryCommittedOwnersPostgres` |
| T5 | `internal/app/httpapi/chain1_local_product_acceptance_test.go` | `TestChain1LocalProductAcceptance` |
| T6 | `internal/app/httpapi/browser_capture_application_test.go` | `TestBrowserCaptureHTTPOptionalCompositionPreservesExistingContracts` |
| T7 | `internal/app/httpapi/application_http_server_test.go` | `TestBuildIsolatedApplicationHTTPServerPreservesShellContract` |
| T8 | `internal/app/httpapi/ai_workbench_integration_test.go` | `testAIWorkbenchOwners` |
| T9 | `internal/app/httpapi/ai_workbench_conversation_only_test.go` | `testConversationOnlyScopedCRUDAndNoExecution` |
| T10 | `internal/app/httpapi/ai_workbench_content_integration_test.go` | `testChatContentThroughOwners` |
| T11 | `internal/app/httpapi/current_application_test.go` | `TestCurrentApplicationAuditMembershipRouteCombinations`、`TestCurrentApplicationReferralRouteAdmission`、`TestCurrentApplicationAuditRequiresExactDescriptor` |
| T12 | `internal/app/httpapi/commercial_resources_test.go` | `TestCommercialResourcesLiveAuthorization` |
| T13 | `internal/app/httpapi/commercial_read_integration_test.go` | `TestCommercialHTTPPostgresBFFClientZeroWrites` |
| T14 | `internal/app/httpapi/current_official_knowledge_test.go` | `TestOfficialKnowledgeUsesLiveAccessWithoutEnterpriseStorage` |
| T15 | `internal/app/httpapi/http_module_test.go` | `TestCoreHTTPModuleRegistersHealthRoute`、`TestHTTPModuleRegisterRejectsNilRegistrar`、`TestBuildHTTPServerBundleFromModulesSkipsDisabledModules`、`TestBuildHTTPServerBundleFromModulesSkipsNilModules`、`TestBuildRuntimeBundleFromModulesCollectsRoutes` |
| T16 | `internal/app/httpapi/image_set_module_test.go` | `TestFullImageReadOnlyGrantCanReadRequirementsButCannotMutate` |
| T17 | `internal/app/httpapi/membership_integration_test.go` | `newMembershipFixture` |
| T18 | `internal/app/httpapi/legacy_product_retirement_test.go` | `TestProductionRuntimeRegistryHasNoLegacyProductModules` |
| T19 | `internal/app/httpapi/member_resource_audit_test.go` | `TestAccountAuditHTTPReadsNativeMemberResourceFactsWithFreshAuthorization` |
| T20 | `internal/app/httpapi/own_product_agent_integration_test.go` | `testOwnContentThroughOwners` |
| T21 | `internal/app/httpapi/product_agent_integration_test.go` | `testProductAgentOwners` |
| T22 | `internal/app/httpapi/server_test.go` | `TestProductionAuthShapeUsesIndependentServerAndWorkbenchAuthentication`、`TestWorkbenchOrganizationPolicyFailsClosedWithoutAuthenticationMiddleware`、`TestBuildHTTPServerFromRoutesAtBindsLoopback`、`TestHTTPServersKeepRouteAuthorizationIsolated`、`TestCurrentSMSWebhookDoesNotRelaxProtectedImageRoutes` |
| T23 | `internal/app/httpapi/retail_catalog_admission_test.go` | `TestCurrentApplicationRejectsPublicPriceBoundaryDrift` |
| T24 | `internal/app/httpapi/product_title_review_browser_fixture_test.go` | `TestProductTitleReviewBrowserFixture` |
| T25 | `internal/app/httpapi/product_selection_integration_test.go` | `testSelectionHTTPThroughNativeOwners` |
| T26 | `internal/app/httpapi/product_review_integration_test.go` | `candidateReviewFixtureApplication` |
| T27 | `internal/app/httpapi/shein_records_integration_test.go` | `recordApplication` |
| T28 | `internal/app/httpapi/shein_records_failure_test.go` | `TestSheinRecordDefaultProductionHasNoRoute`、`TestSheinRecordCancellationAndIsolatedSource`、`TestSheinRecordAuthorizationDeadlineReturnsHTTPTimeout` |
| T29 | `internal/app/httpapi/shein_diagnostic_integration_test.go` | `diagnosticApplication` |
| T30 | `internal/app/httpapi/shein_diagnostic_failure_test.go` | `TestSheinDiagnosticAuthorizationAndSlowBodyDeadlinePostgres` |
| T31 | `internal/app/httpapi/shein_diagnostic_browser_fixture_test.go` | `TestSheinDiagnosticBrowserFixture` |
| T32 | `internal/app/httpapi/shared_regression_postgres_test.go` | `sharedRegressionApplication` |
| T33 | `internal/app/httpapi/source_account_browser_fixture_test.go` | `startSourceAccountFixtureApplication` |
| T34 | `internal/app/httpapi/source_account_application_test.go` | `TestNewSourceAccountApplicationRequiresExplicitSchemaWithoutDDL`、`TestNewSourceAccountApplicationRejectsNilDatabase` |
| T35 | `internal/app/httpapi/source_account_application_postgres_test.go` | `TestSourceAccountApplicationPostgresAcceptance`、`TestSourceAccountConfiguredUserPostgresAcceptance` |
| T36 | `internal/app/httpapi/shop_diagnosis_integration_test.go` | `testDiagnosisHTTPThroughNativeOwners` |
| T37 | `internal/app/httpapi/storecenter_module_test.go` | `TestStoreCenterBuildIsInertWhenWorkbenchIsDisabled`、`TestStoreCenterBuildFailsClosedWithoutDatabase`、`TestStoreCenterBuildSharesOneDatabaseAcrossPersistenceAndReturnsOneCloser`、`TestStoreCenterBuildCleansUpAndReturnsSafeErrorsAfterOpen`、`TestStoreCenterRuntimeConnectionProviderReportsUnavailable` |
| T38 | `internal/app/httpapi/unified_commercial_read_test.go` | `TestUnifiedOverviewUsesIndependentAvailabilityAndTrustedScope`、`TestResourceEventsUseBoundedFiltersAndLiveOrganization` |
| T39 | `internal/app/httpapi/zitadel_auth_test_support_test.go` | `package/global`、`TestMain` |
| T40 | `internal/app/httpapi/workbench_context_module_test.go` | `TestWorkbenchRuntimeBundleConsumesInjectedAuthenticationDependencies` |
| T41 | `internal/app/httpapi/zitadel_sms_composition_test.go` | `TestBuildZitadelSMSServicePassesPhoneExpiryConfiguration` |
| T42 | `internal/app/httpapi/listing_materials_integration_test.go` | `testMaterialsNativeOwners` |

**B. T 表中的旧实现细节与当前安全断言分开处理。**

- http_module_test：`TestHTTPModuleRegisterRejectsNilRegistrar` 仅证明旧结构体错误，随退；`TestCoreHTTPModuleRegistersHealthRoute` 的未在当前入口挂载旧 health 随退，不新增端点。disabled/nil module、注册失败与 routes 汇集的有效行为仍测试 buildRuntimeBundleFromModules，用 test-only module，原 enabled/disabled/404/顺序断言保留。
- application_http_server_test：旧 isolated 参数37s/39s 与 wildcard旧默认退出；用实际 current server 验证当前32s/loopback及所有 context、较早 deadline、cancel、no-store/nosniff、404断言。server_test 的同类 wildcard 断言只退旧便利构造，不能取消实际 serverAddress/loopback 检查。
- storecenter_module_test：五个旧 factory/open/close/error-wrapper/unavailable provider 用例与独占 helper 随旧 builder 退休；其“构造不隐式开库、不关闭借用连接、当前观察不可伪造可用”要求由当前 Store/SourceAccount owner 的现有验证及下表窄用例承接。不把旧 cfg.Database 自动打开、旧错误类或 fake unavailable provider 抽到当前包。
- zitadel_sms_composition_test 的旧 builder 用例随退；0/1/60有效、-1/61/畸形拒绝接回现有当前 SMS 私密配置测试，不能让全局 key 回流。
- server_test：当前认证/组织顺序、nil verifier/resolver、稳定错误、隔离两server issuer/authorizer、SMS public不放松 protected image 全部保留；旧示例路径用合法当前 descriptor/实际路由准入验证，删除旧 allowlist 拒绝语义，不删 protected handler 不执行断言。workbench_context_module_test 的注入用例改调实际 current server，精确注入同一 verifier/resolver，保留只调用一次等断言。
- SourceAccount/审计/记录及各种 integration/browser fixture 仍由真实当前 module/core/routes提供业务，原无DDL/borrowed DB可用、JSON/hash/组织/授权超时/响应头/不可变/无外部副作用断言保留。记录 reader 仍取原 builder 返回值；不重新开放默认记录写入口。旧单独 server 的生命周期是 test-only fixture，不新增生产 facade。
- zitadel_auth_test_support_test 的 TestMain 仅换当前 verifier + authorizer 依赖构造，并补当前合法 exp（及对应 project claims）；loopback provider、每请求 bearer 和失败用例保持。所有上层 fixture 继续验证实际当前中间件，不以旧 global allowlist 或预置可信 header代替。

**C. 其它精确测试路径：**

| ID | 路径 | 允许转换与必须保留 |
| --- | --- | --- |
| T43 | `internal/authruntime/zitadel/middleware_test.go` | RETIRE 整文件；有效 discovery/introspection/claims 用例原断言迁入 verifier_test；HTTP 身份头/请求组织不可信用例迁入 server_test。roundTripFunc 供 verifier/self_service 继续复用，移到 verifier_test；其余独占 helper 随退。 |
| T44 | `internal/authruntime/zitadel/verifier_test.go` | 接收上行当前行为用例；ParseRoles 用例改调 ParseRolesForProject，输入/顺序/去重不变；normalize/default HTTP 5s 直接验证 NewVerifier 的既有配置；现有 verifier 安全用例全部保持。 |
| T45 | `internal/core/config/config_test.go` | 仅 TestConfigBuildIncludesListingKitConfig 去掉四退休字段的正向组装输入/断言；保留平台管理员、issuer/client/project、username 拒绝、imageagent admission 全部原断言。 |
| T46 | `internal/core/config/config_env_test.go` | 精确转换 §6.1 C 表所列旧配置用例，并追加 §4.2 两 loader 的存在即拒绝及当前输入保留；其它函数原样保持。 |
| T47 | `internal/core/config/validator_listingkit_test.go` | 仅 requireLegacyUsernameAllowlistRejection 对新 Hint 的匹配；所有旧 username 输入拒绝/值不泄漏/Field/Message/默认与当前配置断言保持。 |
| T48 | `internal/app/runtime/currentapplication/config_test.go` | 仅 TestListingKitAuthorizationManifestMapsOnlyPlatformAdminAllowlist 去掉装饰性 AuthorizationRequired 字段判断；保留 ProjectID/管理员/manifest 原精确值与当前安全断言。 |
| T49 | `internal/app/httpapi/zitadel_sms_test.go` | 仅 TestZitadelSMSPrivateConfig 复用已有临时私密文件 fixture 接收原全局配置测试中仍有效的 0/1/60/-1/61/畸形过期值约束；私密文件、HMAC、失败安全、取消和旧路由404等原断言全保留。 |
| T50 | `internal/app/httpapi/current_store_center_test.go` | 仅 TestCurrentStoreOptionalAssembly 与同文件窄用例核对当前 native records pool 缺失即失败、不替开旧 cfg.Database、不关闭调用者持有连接；当前观察/授权/路由 drift 断言全保留。 |
| T51 | `tests/listingkit_backend_retirement_test.go` | 仅 §6.1 D 指定两旧 guard + 一项本批闭包 guard；所有其它函数/计数/算法不变。 |
| T52 | `tests/import_boundaries_test.go` | 仅 §6.1 D 指定两个 HTTP wrapper guard；其余包括身份中立、package import allowlists 均保持。 |
| T53 | `tests/issue388_chain1_acceptance_contract_test.go` | 仅 TestIssue388Chain1AcceptanceAssemblyContract 将必需 NewSheinRecordApplication 改为实际 buildLocalTrialListingRoutes + 当前 server 装配，增加旧构造器禁入；其余 owner/原 workflow/无伪造批准/JSON/契约断言不变。 |

config_env_test 允许调整的旧函数仅：
`TestNewViper_BindsLegacyEnvironmentAliases` 中三个旧 allowlist 行的 Primary 正向保活（username 行保持，三项原 retired alias 拒绝继续由原拒绝测试覆盖）；
`TestNewViper_BindsListingKitEnvironmentVariables` 中四旧 allowlist 正向输入/断言；
`TestBuildConfigReadsTencentSMSWebhookCredentials`、`TestBuildConfigRejectsMalformedPhoneVerificationExpiry`、`TestBuildConfigPreservesValidPhoneVerificationExpiry` 的旧全局正向组装语义；
`TestLoadConfigFromFile_AssemblesListingKitAndZitadelConfig` 的旧 key/env 输入/四字段断言；
`TestLoadConfigFromFile_CannotDisableListingKitAuthentication` 转为两 loader 拒绝 false/0旧开关且无 config返回。追加 §4.2 keys/22env 的 presence/空值/无值泄漏与当前输入不受影响测试。其余 Primary/alias/username/database/当前自动环境/旧配置拒绝/安全测试原样保持。

middleware_test 的有效 discovery缓存/坏响应/过大响应/subject/resourceowner/claims 用例迁给 verifier；身份头/请求组织不可信迁给当前 HTTP authentication。只退旧全局 allowlist 匹配、StringSliceToSet 与旧 Gin 包装配置细节；ParseRoles 改调同实现的 ParseRolesForProject，原输入和输出顺序/去重不变。当前 Verifier 原失败分类、项目边界、expiry、大小、deadline/cancel 和 SelfService 用例不删。

**D. 三个 aggregate test 文件的准确改动：**

| 原函数 | 仅允许的转换 | 不变 |
| --- | --- | --- |
| TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired | goroutine 在 retired/KEEP表中的入口转换成 Lstat 整根缺席；移除八声明保活与 owner存在例外；timeout KEEP只删AI两项；httpclient KEEP只删NewSimpleWithTimeout；本批新 guard明确这些缺席 | kernel/module/httpapi/current config/logging/httpclient其余声明、下载值、遍历/错误失败关闭、其它 owner 与原禁令 |
| TestLegacyLoggingHTTPMigrationSupportClosureIsRetired | 仅删 NewSimpleWithTimeout 的正向源码片段行，改为声明缺席 | 当前 HTTP/logger/TLS/security 与其它精确片段/AST/算法 |
| TestAppHTTPAPISupportHTTPModuleWrappersStayRetired / TestAppHTTPAPIFeatureHTTPModuleWrappersStayRetired | 原必须读取 http_modules.go 改为显式 Lstat 文件缺席；旧声明/禁词继续用保留 internal/app/httpapi 索引检查 | 原禁入集合、其它 route_handler_types 检查/失败处理；不 early-return 跳过、无 root缺失 skip |
| TestIssue388Chain1AcceptanceAssemblyContract | 只把旧独立 SHEIN constructor 的必需字符串改为真实 buildLocalTrialListingRoutes/当前 server 装配，显式禁回旧constructor | 全部业务/批准/worker/原工具契约及禁止fixture伪造持久事实断言 |

允许在原 listingkit_backend_retirement_test.go 添加一项聚焦本批的闭包 guard：精确声明/整源缺席、32生产路径混合KEEP、当前认证必需调用与22env/配置拒绝合同；复用既有 AST/文件索引工具，不新造检查平台或宽泛词禁令。其它检查函数、Code Health逻辑、CURRENT inventory、历史/计数/ceilings、workflow、保护规则、依赖零改。若实际 CI暴露遗漏，按 §5.2 报告准确例外，不能偷偷放宽。

### §6.2 候选证据与状态

未来自检重点为 `internal/app/httpapi`、`internal/authruntime/zitadel`、`internal/core/config`、`internal/app/runtime/currentapplication`、现有 kernel/module/httpclient/browserinstall 包和上述原 guard。对认证隔离/取消适用 race；当前命令、公开采集器与 schema/worker入口只编译，不启动进程/下载/provider/真实DB。已有 integration fixture 的测试逻辑必须编译，环境未运行即 NOT_RUN；原 workflow 保持并完成准确候选 CI，不把局部PASS冒充完整run。

源码/测试 diff 完成后，一次集中最终交付复核；准确 HEAD、原 CI run/attempt/状态、独立结论及授权分别写同一 PR。PR CI不等于merge/main CI，源码证据不等于部署或产品验收。用户另授合并后，只认实际 merge SHA 的 push/main 完整 run completed/SUCCESS；SKIPPED/NOT_RUN/UNKNOWN与SUCCESS区分。#301保持OPEN。

## §7 复核、确认与停止点

当前交付是可复核的三文档设计提案。适用独立设计复核重点：当前认证无空分支、旧配置拒绝不触及当前身份、前批KEEP替代准确、fixture保留有效断言、无隐式持久化/外部操作、删除闭包完整。最多两轮正常架构复核，finding 先按AGENTS分类；只有真实BLOCKER重开架构，IMPLEMENTATION_TEST在本批实现验证，BACKLOG不追加试用门禁。

**设计与源码最终复核分别绑定准确 HEAD；未经用户本轮授权不默认新增 AI 互验线程。** GitHub适用复核如先完成且无待处理问题，复用该结论，不重复启动子代理。若需独立子代理，只读核对具体diff/实际消费者和现有证据，不写代码、不跑测试、不操作既有资源、不重复触发GitHub评审。

达到设计复核可接受、原设计候选CI通过后，交用户确认 §1–§7（含 §4.2/§6.1）及完整闭包，再更新IMPLEMENTATION_READY。用户目前“可以”只覆盖设计，不授权源码实施/合并/部署/关单。清单外消费者或真实边界冲突按 §5.2 提供准确提案；不扩全局架构、不机械拆批。#301 始终 OPEN。
