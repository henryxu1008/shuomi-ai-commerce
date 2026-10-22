# #301 旧 SDS 登录、商品同步与专属尾部完整退休设计

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已明确确认冻结设计 f83ddb2ecd494adea13badfa84586b3a3d07c172 §1–§7（含§6.1），在同一 #885 整批实施，inventory 只删除准确四条CURRENT行。[用户确认/Ready准入](https://github.com/qq550723504/task-processor/pull/885#issuecomment-6335862745)及 #301/PR Ready 读回先于生产修改；设计阶段N/A，本轮 **Design Basis: Reuse Existing Architecture / Ready**。唯一Writer已按§5/§6.1完整退出81专属工件、15混合源及原guard/直接文档尾部，无新EXTRACT/替代owner，不机械拆批。原设计全审/授权只读增量PASS及设计CI保持历史证据；准确实现HEAD、RED/GREEN、必要验证、原CI及最终交付检查写PR，不冒充运行/用户验收。#301 OPEN，#29不重开。

调查基线main26179374cb4d7e0142c06f70022125149746499c，设计候选同步124a4de006326bf5930b1a3b4533254dd3591968；本轮开工同步main3b50940f3b4ee9a5bd1f4491712ca43a355c0c33，仅#886六品牌文档，无当前闭包/保留owner变化。实际根Go、debug、前端和config消费者刷新符合冻结设计。除用户明确允许的下述三对象与六样例键尾部例外外，原矩阵/停止条件保持。

原CI发现旧SDS退出后的三个孤立尾部，归类IMPLEMENTATION_TEST；[精确提案](https://github.com/qq550723504/task-processor/pull/885#issuecomment-6337518484)及[用户确认/Ready接续](https://github.com/qq550723504/task-processor/pull/885#issuecomment-6337551715)先于增量生产修改。最小例外仅删除api-error.ts、response-schema.ts与upstream-identity.ts中三行AuthenticatedUpstreamRequest类型；原共享身份运行行为和全部测试保持，原guard与本文/当前索引登记窄校准。原CI失败及准确新HEAD验证留PR，不改Code Health检查/配置、inventory四行之外内容或workflow。

原候选评审另发现两部署样例六个旧SDS键和父debug索引尾部，归类IMPLEMENTATION_TEST；用户明确允许[准确样例例外](https://github.com/qq550723504/task-processor/pull/885#issuecomment-6337809461)，并在[Ready接续](https://github.com/qq550723504/task-processor/pull/885#issuecomment-6337851157)及正文读回后实施。仅在这六键上取代#847/#850旧共享键值KEEP：ConfigMap一键/两专属注释、Secret example五键退出，所有其它键值保持；debug只删旧test-sds命令，部署README只加SDS退休头注，原历史body逐字保持。无实际配置/凭据/集群操作，不render/apply。

## 1. 用户结果、完整范围与权限

使用者是维护当前应用、采集、图像、Store 和独立 SHEIN worker 的维护者。本批源码结果是完整退出无当前注入的旧 SDS 登录、客户端、模板/设计/同步、HTTP 和专属配置/工具/UI 尾部，避免残留的老入口继续表示已不可用的旧设计。没有已证实的当前核心操作被这些残留阻塞，不扩建产品、治理或试用前置。

本设计覆盖一个完整闭包：61 个旧包 Go 工件、20 个专属尾部工件、15 个混合配置/启动/路由策略/样例源的精确尾部，以及原检查/直接文档的相应处置；不按包、文件数或审核机械拆 PR。保留当前 SDS/POD 业务，不把“退出旧 SDS 实现”写成“退出 SDS 能力”。唯一实现者在同一分支/PR 连续交付，本文确认后再进入源码阶段。

调查阶段只读 Git 追踪的源码/样例键结构、文档和依赖元数据；本轮按用户确认和Ready准入修改准确追踪源/测试/配置/文档。不运行旧 constructor/自测、登录浏览器、旧调试命令、脚本、应用/worker/dry-run、HTTP/provider、Redis/SQL、Docker/Compose/kubectl。不读取实际运行配置、环境凭据、token/私密文件、Profile/缓存/日志/旧报告，不操作既有实例、数据、凭据或进程。不读取/复制源码内嵌的敏感默认值到设计/日志/新测试；旧工具源码随闭包删除，不授权真实账号改密、rotation、历史重写或凭据清理。

设计阶段仅三文档；本轮源码实施已按§5/§6.1整批完成，inventory仅删除用户明确允许的四条CURRENT行，历史/计数/ceilings/检查逻辑不改。workflow/其余依赖/锁/保护零改；源码确认不授权合并、关单、部署或真实环境操作。

## 2. Design Basis、Authority 与保留不变量

设计阶段 **N/A**；本轮用户确认/Ready读回后以 **Reuse Existing Architecture / Ready** 实施，复用：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)和[Issue 驱动规则](../engineering/issue-driven-development.md)。仅 EXTRACT/RETIRE，不引入旧 owner facade、fallback、迁移、旧 ID 映射、双读/双写、状态同步或第二事实源。
- [供应市场及当前 SDS/POD](supply-market-v1.md) §3/§4/§7/§8 已明确旧 design/client 自动重试、补偿删除、结果替代与本地登录 bootstrap 不适用；当前 Product POD/Submission/Asset 及 integration/sds 窄 transport 是现有 owner。当前 [POD Design Agent](pod-design-agent-v1.md) 及能力准入保持，不重新实施或验收这些功能。
- [独立 SHEIN 身份与持久化边界](shein-worker-numeric-persistence-retirement-v1.md)、[#868 值合同边界](legacy-numeric-value-contract-closure-retirement-boundary.md)、[#880 基础层退休边界](legacy-redis-proxy-shared-closure-retirement-boundary.md)中的有效保留合同。SHEIN 登录/Store 同步、UNKNOWN 不接管、现有执行 fence 不变。

这批删除超过 30 个文件并跨 Go/前端/配置，但没有新产品路径、事实 owner、公共领域合同、schema/迁移、权限/租户、计费、状态机/事务、retry/UNKNOWN/recovery 或不可重复副作用边界；不通过机械切片规避准入。当前 owner 和有效行为均已有批准依据，无新 EXTRACT 或替代服务。仅在这些条件成立且具体设计获确认后可 Reuse；发现必须改变当前边界则停止对应实施、升级 Independent Architecture 并先达到 IMPLEMENTATION_READY。

Must：完整退出旧源码闭包；当前能力、共享合同及必要/安全检查有效；除明确确认的四条 CURRENT inventory 外历史/计数/ceilings/检查和 workflow/依赖/保护不变；无真实状态操作或新增内部试用前置。Should：复用原 guard、AST/decoded-import、当前 SDK 和现有测试，不新造验收平台。Out of Scope：当前 SDS 产品改造、凭据刷新/接管、平台 qualification、运行验收、浏览器/Redis 清理、迁移、全仓 dead-code 工程、其它孤立包及依赖升级。Accepted Risks：无新增；真实状态、外部人工调用 UNKNOWN，未执行的运行/用户验收保持 NOT_RUN。UI/IA/Figma 增量 N/A：保留现有 Workbench/POD 页、交互和产品 authority；只退出不被它们使用的旧 BFF/DTO/缓存源。

**精确取代条件**：本文获确认后，仅在 §5/§6.1 矩阵上取代 #880 对旧 SDS AuthRedis 配置的暂留、#664 对旧 SDS 只读 BFF 的暂留，以及 code-health 三工具 retained-deferred 的当前适用含义；另依用户新决定，仅六个旧SDS样例键取代#847/#850的共享模板键值KEEP，不改其它键值或其历史原文。不重写旧设计、dated 表格/计数或历史评审。当前 integration/sds、POD、平台 API/规则及 product_source 合同不被取代。

## 3. 实际消费者：contract → implementation → injection → consumer

调查结合根模块 Imports/TestImports/XTestImports、追踪文件引用、实际入口依赖、独立 debug 模块和前端请求/类型调用。只看根 `go list ./...` 会漏掉 debug 子模块与 Next 自动注册路由，必须沿尾部一起核对。无 Go importer 不等于没有脚本/BFF。

| 对象 | 实际消费者/装配 | 结论 |
| --- | --- | --- |
| sdslogin/bootstrap、sds/httpapi、sds/httpbootstrap | 无当前 app/cmd 外部 importer；bootstrap 只构建旧 login Service、HTTP module 并设置旧 client 的全局 LocalLoginProvider | 同批 RETIRE，不转接当前应用 |
| sdslogin | 唯一旧生产 importer 是本闭包 bootstrap；旧 service/Redis/browser/HTTP 方法及专属自测无当前业务注入 | 与旧 login DTO、状态文件/Redis 接线源码同退，不调用 ClearState |
| sds/client | 旧 design/template/httpapi/httpbootstrap/usecase/login/bootstrap 及闭包自测；另有 hack/debug 子模块 test-sds 手工入口 | 全部旧 importer 同批退出，不留下调试 importer |
| sds/template、design、workflow、usecase、adapter 父包 | import/call 链只在旧闭包内：bootstrap → usecase → adapter/workflow → design/template → client；旧模板/渲染/上传/同步返回值无当前外部消费者 | 全链 RETIRE；不为旧行为复制 facade/helper |
| sds/adapter/product_source | 独立子包、10 文件；marketplace/shein/publishing 的 pod_submit_readiness_policy 消费 NormalizeExecution/SubmissionBlocked/ReadinessMessage；纯值逻辑仅依赖 canonical，不依赖旧父包 | 完整 KEEP，包路径与值/hash/规范化/readiness 合同不搬家；禁止按 adapter 或 sds 父目录递归删除 |
| 当前 SDS/POD | runtime/currentapplication → app/pod → integration/sds + product/pod + Submission/Asset/Collection；当前 CredentialSource/私密文件格式、binding revision 和 owner 已独立 | 完整 KEEP；不会读旧 Platforms.SDS、cookie 自动恢复或全局 Redis key |
| 旧前端 /api/sds/* | 4 个 Next GET route 自动注册并经 shared.ts 指向旧 /api/v1/sds；无当前 Workbench 页消费者。旧 schema 仅自测，types 仅这些 route/schema 与 recent-variants helper，helper 仅自测 | 12 个旧 BFF/DTO/schema/cache/解析工件及闲置请求类型同退；不能把自动路由当成普通零引用文件忽略 |
| 当前 SDS Workbench | workbench/supply/catalogs/sds 页面使用 components/workbench/pod、lib/contracts/pod、lib/api/pod 及当前 service BFF | KEEP 当前目录/合同/权限；共享 lib/server/upstream-identity.ts 的运行行为及全部有效身份转发测试 KEEP；仅旧四路由消费的 AuthenticatedUpstreamRequest 三行闲置类型依用户例外 RETIRE |
| 专属手工工具 | wrapper → cloakbrowser-poc package scripts → 两 SDS 登录脚本；整个 package manifest/lock 只服务这两入口 | 5 源工件同退；共享浏览器/当前采集/SHEIN 不依赖该 Node 工具目录 |
| 混合配置/启动尾部 | Platforms.SDS / EffectiveSDSAuthRedis / LoginServiceConfig 和两个 builder 的生产消费者只在旧闭包；UI launcher 只额外写两个旧 SDS API base | 只移除精确旧尾部，保留当前 Redis/Browser/SHEIN/POD 配置与启动路径 |
| 共享路由策略 SDS 专属尾部 | authz/httpapi/route_authorization.go 仍有旧 SDS health public 例外、sds/sds-login 模块识别及 sds-login 平台管理员映射；全追踪 Go 检索除旧模块/原guard外无对应descriptor，当前 POD module=supply-market且显式AuthPolicyCurrentIdentity/Permission | 仅删除这些旧 SDS 声明；不改共享认证/授权算法或当前路由策略 |

当前 `current-application` 依赖旧目录命中的只有保留的 product_source，并同时依赖当前 integration/sds；`shein-login-worker`、`image-agent-temporal-worker` 无本批旧依赖。设计不宣称现有外部脚本、用户浏览器或真实部署已经无调用；若发现具体当前外部可观察契约/保留消费者，按 §7 停止，不加 compatibility。

## 4. 配置、持久化与平台副作用 owner

| 边界 | 旧实现与风险 | 退休/保留处置 |
| --- | --- | --- |
| 旧 auth/cookie/payload/browser state | client AuthStateStore/SessionStore、login Service 读写/删除本地 JSON；按旧 tenant/identifier 选账户/Profile/路径，暴露 status/auth-state/login/manual-login/state HTTP | 只删源码，不读文件、不运行构造或清理，不改权限、不迁移、不接管旧身份；物理文件/Profile/会话 UNKNOWN |
| 旧共享 Redis | sdslogin.RedisStateStore 构造即 Ping，Load/Save/Clear 操作全局 `sds:auth:global`，Save 无 TTL；wrapper/PoC 也可写同 key | 同批删调用/配置源，不连接/扫描/读取/删除 key，不建立新 key/fence/自动恢复 owner；当前 Config.Redis、原生 go-redis、SHEIN key/TTL/安装/执行归属控制表保留 |
| 旧 LocalLoginProvider | bootstrap 通过全局 setter 把旧 Service 注入 client；auth bootstrap 可从文件/静态值/远端或本地浏览器恢复认证并重试 | 连同 setter/provider/DTO 退出；不改成当前 Credentials 的 fallback，不据旧 token 推导当前绑定 |
| 旧平台读写 | 模板查询、OSS 签名/上传、materials 登记、render/sync/save、读取替代及补偿删除；默认写重试/登录后重试不符合当前不可重复写/UNKNOWN 规则 | 只删旧实现/专属测试，不发送请求、删除成品或素材、不做资格验证；已批准当前 integration transport/Submission permit/readback/UNKNOWN owner 保留 |
| 手工工具/调试命令 | 启浏览器/Profile、port-forward/子进程、Redis/网络/文件输出、cookie/token 打印；源码还含敏感默认值 | 只移除追踪源/manifest/锁，禁止执行/安装依赖/验证凭据；不递归清目录，不处理 ignored node_modules/.venv/browser/artifact/cache 或历史 Git 对象 |
| 旧 recent-variants | localStorage 键 `listingkit:sds:recent-variants` 和 change event，只有旧 helper/自测 | 删除源码，不访问/清浏览器存储，不迁移到当前 Collection/Product |
| 当前 POD | 当前 server-only CredentialSource 与 verified source、Product POD binding/operation/material/fence、Submission 唯一 intent/attempt/SendPermit、Asset explicit approval | schema、事务、权限/tenant/actor、binding revision、幂等、UNKNOWN/recovery/预算/外部副作用保持；不读当前 private credential 文件或复核旧真实 qualification |
| 当前独立 SHEIN | 当前授权 CLI、组织/Store、独立登录与 Store 同步、可选执行 fence | 实现/配置/凭据/Profile/Redis/UNKNOWN 均保持；不与旧 SDS login 的通用类型名称混淆 |

架构问题从根因处理：旧登录全局状态、旧数字身份和会自动重试/删除的同步编排不能继续成为当前 provider owner。当前批准架构已有独立 owner，本批删除无接线的旧替代路径，而非修补旧登录/重试协议。通用 Redis、浏览器或当前 SDS 本身不属于退休对象。

## 5. RETIRE / KEEP / EXTRACT 准确矩阵

### 5.1 RETIRE：61 个旧包 Go（40 生产 + 21 专属测试）

九组：client 16、template 4、design 10、workflow 4、adapter 父包 3、usecase 3、httpapi 6、httpbootstrap 2、sdslogin 含 bootstrap 13。准确文件如下：

```text
internal/sds/adapter/service.go
internal/sds/adapter/service_test.go
internal/sds/adapter/types.go
internal/sds/client/auth.go
internal/sds/client/auth_bootstrap.go
internal/sds/client/auth_bootstrap_local.go
internal/sds/client/auth_bootstrap_local_test.go
internal/sds/client/auth_bootstrap_test.go
internal/sds/client/auth_state.go
internal/sds/client/auth_test.go
internal/sds/client/client.go
internal/sds/client/config.go
internal/sds/client/config_test.go
internal/sds/client/errors.go
internal/sds/client/request.go
internal/sds/client/session_store.go
internal/sds/client/session_store_test.go
internal/sds/client/transient_upload_error.go
internal/sds/client/transient_upload_error_test.go
internal/sds/design/material_helpers.go
internal/sds/design/models.go
internal/sds/design/payload.go
internal/sds/design/render_validation.go
internal/sds/design/render_validation_test.go
internal/sds/design/rendered_images.go
internal/sds/design/sensitive_words.go
internal/sds/design/service.go
internal/sds/design/service_test.go
internal/sds/design/types.go
internal/sds/httpapi/config.go
internal/sds/httpapi/config_test.go
internal/sds/httpapi/handler.go
internal/sds/httpapi/handler_test.go
internal/sds/httpapi/http_module.go
internal/sds/httpapi/http_module_test.go
internal/sds/httpbootstrap/support.go
internal/sds/httpbootstrap/support_test.go
internal/sds/template/models.go
internal/sds/template/service.go
internal/sds/template/service_test.go
internal/sds/template/types.go
internal/sds/usecase/service.go
internal/sds/usecase/service_test.go
internal/sds/usecase/types.go
internal/sds/workflow/approved_asset.go
internal/sds/workflow/service.go
internal/sds/workflow/service_test.go
internal/sds/workflow/types.go
internal/sdslogin/api.go
internal/sdslogin/automation.go
internal/sdslogin/automation_test.go
internal/sdslogin/bootstrap/build.go
internal/sdslogin/bootstrap/build_test.go
internal/sdslogin/build.go
internal/sdslogin/build_test.go
internal/sdslogin/http_module.go
internal/sdslogin/http_module_test.go
internal/sdslogin/redis_store.go
internal/sdslogin/service.go
internal/sdslogin/service_test.go
internal/sdslogin/types.go
```

### 5.2 RETIRE：20 个专属尾部工件

```text
hack/debug/test-sds/main.go
hack/debug/test-sds/README.md
scripts/refresh-sds-auth-via-portforward.ps1
tools/cloakbrowser-poc/package.json
tools/cloakbrowser-poc/package-lock.json
tools/cloakbrowser-poc/sds-login.mjs
tools/cloakbrowser-poc/sds-login-redis.mjs
internal/core/config/sds_auth_redis_test.go
web/listingkit-ui/src/app/api/sds/categories/route.ts
web/listingkit-ui/src/app/api/sds/products/[productId]/route.ts
web/listingkit-ui/src/app/api/sds/products/route.ts
web/listingkit-ui/src/app/api/sds/shared.ts
web/listingkit-ui/src/app/api/sds/shipment-areas/route.ts
web/listingkit-ui/src/lib/api/sds-products-schema.ts
web/listingkit-ui/src/lib/api/sds-products-schema.test.ts
web/listingkit-ui/src/lib/api/api-error.ts
web/listingkit-ui/src/lib/api/response-schema.ts
web/listingkit-ui/src/lib/types/sds.ts
web/listingkit-ui/src/lib/utils/sds-recent-variants.ts
web/listingkit-ui/src/lib/utils/sds-recent-variants.test.ts
```

只删精确追踪源，不删除 hack/debug 父模块/go.mod/go.sum、tools 父根、sds/adapter 或 sds 父根，不清 ignored 目录。专属 PoC 的两个 package 文件属于本次退休源，不授权其它依赖/锁的 tidy/升级/删除。

### 5.3 15 个混合源：只去旧尾部

| 文件 | 准确退出对象 | 保留 |
| --- | --- | --- |
| internal/core/config/type_platform.go | PlatformsConfig.SDS；EffectiveSDSAuthRedis；SDSPlatformConfig/SDSAuthBootstrapConfig/LoginServiceConfig/SDSLoginServiceConfig 及唯一 strings import | Temu/Shein PlatformConfig、Alibaba1688、其它类型；当前 SHEIN 不消费此 LoginServiceConfig |
| internal/core/config/defaults.go | setSDSDefaults 与唯一调用 | 其它 defaults/Redis/Browser 与 setDefaults 算法 |
| internal/core/config/loader_platform.go | BuildLoginServiceConfig、BuildSDSAuthBootstrapConfig | BuildPlatformConfig/HTTP client |
| internal/core/config/loader_builder.go | BuildConfig 的 Platforms.SDS initializer 全部 | 当前 Platforms/Redis/Browser/POD/其它加载 |
| internal/core/config/config.go | 所有 platforms.sds.* 显式 env bindings（loginService/authRedis/authBootstrap） | 原 Viper/env alias/读取/校验及其它绑定，不新增全局未知键拒绝策略 |
| internal/core/config/config_test.go | TestConfigBuild 的 SDS fixture 设置/断言 | 原测试和其它所有配置断言 |
| internal/core/config/config_env_test.go | 三个 SDS 专属 env 测试函数 | 其它身份/Redis/Browser/SHEIN/当前配置测试 |
| config/config-dev.yaml、config/config-prod.yaml | 仅 platforms.sds 子树与直接专属注释 | 其它平台、共享 Redis/browser/current schema；config-test.yaml 无此子树，零改 |
| scripts/start-listingkit-local-ui.ps1 | Set-UiApiBases 中 SDS_API_BASE/SDS_LOGIN_API_BASE 两项赋值 | 当前 UI/loopback/身份/token handoff、其它 API base 和启动流程 |
| scripts/start-listingkit-local-ui.Tests.ps1 | 同一场景 names 中两个 SDS 项及两项断言 | 测试本体、环境恢复、loopback 拒绝、其它 API base/身份/启动检查 |
| internal/authz/httpapi/route_authorization.go | RouteRequiresZitadelAuth 的 GET /api/v1/sds-login/health 例外、return表达式的 sds/sds-login 两模块项；routeRequiredPermission 的 sds-login case | 显式 AuthPolicy 优先级、SHEIN health、其它模块/permission、SMS callback/HMAC边界、角色/组织authorizer及全部错误响应；不改当前POD显式身份/权限 |
| web/listingkit-ui/src/lib/server/upstream-identity.ts | 仅三行闲置 AuthenticatedUpstreamRequest 类型，旧四 SDS route 是其全部消费者；用户已明确允许该准确 KEEP 例外 | 全部运行行为、其它类型/导入及全部原身份/权限/租户测试 |
| deployments/kubernetes/listingkit-workbench/base/configmap.yaml | 仅TASK_PROCESSOR_SDS_LOGIN_SERVICE_DEFAULT_HEADLESS及两行SDS专属注释 | 其它全部键值，特别是SHEIN headless/worker、ImageAgent、UI/身份/存储配置 |
| deployments/kubernetes/listingkit-workbench/base/secret.example.yaml | 仅TASK_PROCESSOR_SDS_ACCESS_TOKEN及SDS_LOGIN_SERVICE_BASE_URL/SHARED_KEY/TENANT_ID/IDENTIFIER五个样例键 | 其它全部键值与共享凭据/allowlist不变量；不读任何实际Secret/private manifest |

不扩展 retired runtime reject registry 或引入兼容键/替代配置。删除后旧键不会被当前 owner 使用；原未知键/环境解析语义保持。确认实施时以 AST/配置文本检查保护准确字段/default/env 不复活。

### 5.4 KEEP：当前 owner、合同与全部既定能力

product_source 10 文件逐个保持：

```text
internal/sds/adapter/product_source/apply.go
internal/sds/adapter/product_source/apply_test.go
internal/sds/adapter/product_source/baseline.go
internal/sds/adapter/product_source/baseline_test.go
internal/sds/adapter/product_source/boundary_guard_test.go
internal/sds/adapter/product_source/execution.go
internal/sds/adapter/product_source/execution_test.go
internal/sds/adapter/product_source/images.go
internal/sds/adapter/product_source/images_test.go
internal/sds/adapter/product_source/model.go
```

- `internal/integration/sds/**`、`internal/app/pod/**`、`internal/product/pod/**`、`internal/app/runtime/currentapplication/pod_credentials.go`/`supply_market.go` 及当前 POD worker/HTTP/Console、credential 安全/准入/合成协议测试保持；当前 UI `workbench/supply/catalogs/sds/**`、`components/workbench/pod/**`、`lib/contracts/pod.ts`、`lib/api/pod.ts` 和当前 service BFF 保持。
- `web/listingkit-ui/src/lib/server/upstream-identity.ts` 的全部运行行为及全部有效测试保持；仅按用户确认例外删除三行闲置 AuthenticatedUpstreamRequest 类型，其余内容零改。测试请求中用 /api/sds/products 作为普通合成 URL 不代表依赖旧 route handler，保留原身份/租户测试，不为文本同名而删除。
- 当前采集/1688/Amazon 来源、图像/百炼图片参数/风险判断、有效平台 API/DTO/规则、AI/prompt/Asset/预算、MappingInfo、两种 SHEIN 属性、PriceManager/PriceCalculator、原规范化/hash/JSON 零值槽位、Store 观察同步、当前身份/计费、独立 SHEIN 登录与 Store 同步及 UNKNOWN/fence 全保留。
- 当前 DB/Goose、原生 Redis/core/config.RedisConfig、共享 browser/CloakBrowser、workerpool/通用队列/Retry/Temporal、共享 http/image/json/logging 等保留。其它零默认 importer 的显式保留 API/规则、localagent/Prompt/其它集成不纳入本次。
- 无新 EXTRACT：当前行为已归现有 owner；保留 product_source 原包而非复制/搬迁。若刷新后发现必须抽取的新有效规则，先明确具体行为/owner并报告，不能实现时临时拓展设计。

```text
Legacy decision: RETIRE（用户确认，已整批源码实施）
Reusable behavior: 无新 EXTRACT；当前 SDS/POD、product_source 值规则、共享身份/浏览器/Redis及现役能力原样保留
Current owner: Product POD/Submission/Asset/Collection + integration/sds + 当前 app/runtime/Console；SHEIN readiness/product_source及独立 SHEIN 原 owner
Cutover/deletion condition: §1–§7（含 §6.1 四条 CURRENT 行）经复核和用户明确确认，Issue/PR Ready 先读回，实际消费者刷新无冲突；完整源码/专属尾部同退，无真实状态处置
```

## 6. 必要验证与交付证据

设计阶段：追踪源/消费图、当前入口依赖元数据和原文档检查；不执行旧 SDS 测试/工具或当前 provider。本文三个文档候选跑适用原文档检查与原 CI，复核真实差异；CI 只证明准确设计候选，不是源级实现、实际 main CI 或运行验收。

确认实施后，在原退休设施里先为本闭包缺席/禁入及精确配置退出写 RED，再整批删除得到 GREEN，不新建通用检测/验收平台。开发自检使用原 guard、配置/文档检查、product_source/readiness 与当前 integration/sds/app/pod 合成/loopback测试、前端身份/退休/当前 POD 相关测试和类型检查、原 UI launcher 有界 fixture 检查、三个当前入口 `go list -deps` 和适用编译（不执行产物）。不新建 PG/Redis/browser/container；原需要外部条件的测试按原条件如实 NOT_RUN/skip，不伪称实际执行。debug 子模块源码/decoded import 亦检查，保留其它 debug 源/模块文件。稳定候选跑完整原 CI；不重复人工全量 CI/矩阵。

实现者自检不等于最终/用户验收。独立最终交付检查一次完整 diff/保留路径/现有证据；修复仅复核增量。准确 HEAD、本地执行/未变化证据、原 CI、独立结论与用户授权留 PR，不把滚动状态写成反复源码提交。

### 6.1 原检查、文档、inventory 与 workflow 的精确处置

以下为用户确认的同批实施范围，当前已精确应用；原设计阶段全部零改的证据留PR：

| 对象 | 同批最小校准 | 不得改变 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 复用原工具增加 §5 的81工件/九旧包精确缺席、生产/测试 decoded imports 禁入、混合配置 owner 顶层声明/字段及 YAML/default/env 缺席；adapter 仅 parent package 禁入，product_source 明确保留；三个当前入口旧依赖退出 | 全部先前退休/安全合同、读取/解析/依赖失败报错；不 missing-skip、不用 dummy/alias 满足检查 |
| tests/import_boundaries_test.go / TestSDSClientRetiredManagementImportsStayBlocked | 旧 root 必须读取的管理禁入检查转为更强 client 根缺席/全追踪 decoded import 禁入保护，可复用同批 guard，保持原测试名以保留文档引用 | 不能简单忽略不存在目录；原其它 management/tenant/SDK/decoded import 检查不动 |
| 同文件 businessHTTPPackages | 仅移除 sds/httpapi 和 sdslogin 两项已退出 HTTP 允许前缀 | 当前 HTTP owner/所有其它允许拒绝项、解析逻辑不改 |
| 同文件旧 route helper bans | 保留 /api/v1/sds/、/api/v1/sds-login/、appendSDS* 等旧 helper/alias/module 禁止项；将“委派旧 module”的陈旧失败说明改成旧路由已退休、保留当前 POD route owner | 不能删禁止条件或恢复旧路由；SHEIN 有效模块与其它检查保持 |
| 同文件 listingkit_support 精确 import allowlist | 仅删旧 sds/usecase 允许项 | 其它条目/静态解析/错误处理保持；已退休 helper不能复活 |
| phase3 product/depguard/repository_structure | **零改**：sds 根仍含 product_source，原 platform-neutral/product边界扫描有效；sdslogin/data 不得出现及 .local artifact fixture 仍是有效负例 | .golangci.yml、SDK边界、字面量decoder、build-target扫描、本地数据检测继续有效 |
| internal/core/config 原测试/退休 guard | 按 §5.2/§5.3 删除专属自测/fixture；在原 guard 精确保护旧 Platforms.SDS/类型/builder/default/env 退出 | Viper、RedisConfig/Config.Redis、Browser/SHEIN/当前POD、安全/身份测试及 retired registry语义保持 |
| web src/test/legacy-ui-retirement.test.ts | 原 absence/import/request-target 设施覆盖12个旧工件、旧 route/helper/type imports及旧请求目标，负例不得匹配当前 /api/service/pod 或 Workbench 页面 | 共享 upstream identity全部运行行为/原测试与当前 POD/Collection/permission 合同；不删基于通用URL的身份安全用例 |
| upstream-identity 原共享模块/原退休guard | 仅删除用户确认的三行闲置类型；原Go退休guard保护该类型名缺席，原前端guard补两个解析文件缺席/禁入 | 共享身份运行实现、Session/NextRequest仍用导入、全部有效测试；不把闲置类型例外扩展为身份重构 |
| UI launcher 原Pester | 仅 §5.3 两env项/断言退出，保留原合成fixture和环境恢复；通过原适用检查验证，不调用启动应用/真实凭据 | token handoff、loopback/权限拒绝、当前 UI/其它入口 |
| authz/httpapi 原策略/测试与原退休guard | §5.3精确移除SDS-only路径/module/permission声明；在原退休guard保护route_authorization.go这四个准确旧字符串缺席，保留原策略单测和当前POD descriptor检查 | TestRoutePolicyPreservesVerifiedIdentityAndExplicitPublicAuthority、TestRoleMiddlewareUsesVerifiedContextAndOriginalPermissionPolicy完整保留；现有身份/权限/组织/拒绝、SHEIN和SMS规则不变，不把修改泛化成auth重构 |
| 两部署样例与原Go退休guard | 仅§5.3六键/两注释删除；原guard保护这两个样例的旧SDS前缀缺席 | 其它键值、工作负载/权限/安装注入与原安全检查不变；不render/apply |
| hack/debug/README.md | 仅去掉已删除test-sds的示例命令；原guard禁止该旧命令 | test-amazon及其它当前debug条目、父模块/其它源不变 |
| deployments/kubernetes/listingkit-workbench/README.md | 顶部只加#885 SDS source supersession窄注记，旧BFF/login说明失效、当前POD/SHEIN保持、生产NOT_READY与无执行权限明确 | 原历史body逐字保持，不重写旧IAM/部署/凭据/rollback/drain证据 |
| internal/sds/README.md | 顶部加窄历史适用注记与当前 product_source/integration/sds/POD owner 链接，旧说明不再是操作入口 | 2026-04-23及后续历史正文/真实样本不重写，不复核真实状态 |
| docs/refactoring/code-health-decisions.md | 仅新增窄 superseded 注记，指向三条 SDS 手工工具 retained-deferred 行的本轮替代依据 | 旧表格/状态/计数/失败原文不改，不扩展其它工具 |
| docs/architecture/project-boundaries.md、Architecture README、Legacy Register、本文 | 当前 SDS guard 解释/新设计状态按准确确认校准；设计阶段只本文与 PROPOSED索引/登记 | 既有当前owner/guard清单、dated设计/计划/产品证据不改；#664和#880只在§5精确对象上被取代 |
| **CURRENT inventory四行** | 用户已具体确认，仅删除下方四行；设计阶段零改证据保留 | 历史/计数/ceilings/映射、accuracy/parser/depguard/guard逻辑全部不改 |
| workflow/依赖/保护 | 两套 workflow 及其它全部 workflow 零改；根/前端/debug go.mod/go.sum/package锁零改，仅 §5.2 专属 PoC 自有 package.json/package-lock.json 随源退休 | 必需与安全检查、基线、ceilings、分支保护不降级；不新增/retry/cancel CI 或绕过门禁 |

**用户明确允许并已应用的唯一 inventory 补丁**（从 CURRENT consumer table 删除准确四行，无新增行，无历史/计数更改）：

```diff
-| `task-processor/internal/core/logger` | `task-processor/internal/sds/client` | marketplace business consumer | marketplace owner / Phase 4 |
-| `task-processor/internal/core/logger` | `task-processor/internal/sds/design` | marketplace business consumer | marketplace owner / Phase 4 |
-| `task-processor/internal/core/logger` | `task-processor/internal/sdslogin` | marketplace business consumer | marketplace owner / Phase 4 |
-| `task-processor/internal/integration/httpimage` | `task-processor/internal/sds/design` | marketplace business consumer | marketplace owner / Phase 4 |
```

完整删除使这四条当前消费者失效；若坚持完整 inventory 零改，原 accuracy检查与完整退休将直接冲突。不能保留假消费者、dummy import、跳过检查或改统计逻辑。用户本轮已明确确认 §1–§7（含§6.1）并明确允许这四行最小校准，准入/读回先于应用；不扩展到其它inventory记录或检查逻辑。

## 7. 实施准入、停止点与关闭边界

1. 当前完成只读映射和正式设计，Design Basis N/A；PR 为 DESIGN_REVIEW，不把源码置 Ready。完成原文档CI及一次有界 GitHub 设计复核，按当前 Must 分类 findings；未返回精确SHA/结论时不宣称复核通过、不重复触发同SHA Running评审。
2. 具体 §1–§7（含 §5 的 UI/debug/工具尾部与 §6.1 四行 inventory 处置）获用户确认后，先更新/读回 #301/PR 的 Reuse Existing Architecture / Ready，再修改生产源；唯一 Writer 同一PR完整实施。不因61包文件/前端/config边界机械拆批。此前范围/旧SHA授权不授权此次源码/合并。
3. 开工同步一次 main，最终候选前再同步一次；刷新根模块、debug子模块、前端路由/DTO/helper、工具/配置消费者和保留能力。遇当前实际消费者、外部可观察兼容义务、新有效规则抽取、新 owner/状态/权限/持久化/副作用变化，停止对应源码实施，报告具体当前证据及最小设计差异；不得临时新增 fallback/恢复/迁移或扩大删除。
4. 源码完成后按 §6/§6.1 跑必要开发检查、原CI和一次独立最终交付检查。具体发现先分类；只复核修复增量。禁止为设计或源码自建验收平台/新运行环境，未运行保留 NOT_RUN。
5. 稳定准确 HEAD 才交用户另行合并授权；正常 expected-head 合并后，只用实际 merge SHA、event=push/head_branch=main 对应完整 run 的 completed/SUCCESS 确认，不以 PR CI/Required Gate 单项或其它 main提交替代。
6. #301 必须 OPEN，#29不重开；当前段之外从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”开始历史后缀逐字保留，不改历史勾选，不超正文长度。源码退休完成不代表全仓清零或运行/产品验收，也不授权关单。
7. 始终保留独立 SHEIN 登录与 Store 同步及此前有效能力；不操作既有实例、数据、凭据、浏览器存储或进程，不读实际私密文件，不增加内部试用前置。只删除追踪源码，不把删除路径变成真实资源清理。
