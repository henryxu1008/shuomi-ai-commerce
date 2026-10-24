# #301 旧 SHEIN 品牌查询与错误分类整链退休设计

Refs [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)。父 Issue 保持 OPEN，#29 不重开。

- Status: **DESIGN_REVIEW_PENDING / IMPLEMENTATION_NOT_AUTHORIZED**。用户本轮仅批准先做完整设计，确认后再实施；本 PR 当前只修改本文、Architecture README 与 Legacy Register 三文档。
- 本阶段 **Design Basis: N/A**：纯设计与当前索引，不改变生产路径、原检查、配置、持久化或平台调用。拟实施 **Independent Architecture / NOT_READY / Backlog**，以本文件精确替代前批 KEEP 和旧品牌响应 JSON 的保留边界；须完成适用设计复核、用户确认，并记录及读回 IMPLEMENTATION_READY / Ready 后才能改源码。
- 调查基线：实际 main **8495b5d664b5f3cfb5e1426b2433e21b33fc25ce**，tree **29d1e61a25b96af99278e8b1f5d2414087323a8e**。#926 已正常合并，匹配 push/main CI40081911984 SUCCESS 只是前批事实，不当本设计或未来实现证据。
- 唯一 Writer 原会话 `01a133ef-757c-73e3-b8f6-76958c642397`，复用独占 worktree，新分支 `codex/issue301-shein-brand-error-closure-design`；[接单及前批当前段归档6382724173](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6382724173)。开工已 fetch main；无同范围接单或 PR，独立 #692 Store schema 设计不接管。

## 1. 用户结果、范围与产品依据

使用者为当前 SHEIN 草稿、文案清理及独立登录/Store 同步能力的维护者。旧平台客户端退出后，未装配的品牌查询 Resolver 仍保活错误分类及品牌列表 DTO；可见结果是将这条完整死链、专属测试、保活检查和历史建议一次退出，当前品牌值、清理与草稿行为沿原 owner 保留。当前没有被此残留阻塞的核心用户操作，不以父目标清零为由新增产品、平台、运行验收或内部试用前置。一个 Delivery Batch / 一个主要 PR，不按文件或测试数量机械拆批。

Authority 为本轮用户“先做完整设计、确认后实施”的明确范围、[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及下文准确前批 KEEP 替代。UI/IA/Figma N/A：没有页面、菜单、交互或能力开放变化。原“Cookie 接口都可退休”提议已撤回，不作为依据。

Scope：旧 `authorizedbrand.ProductAPI/Resolver/Config`、全部查询/匹配 helper、专属 Resolver 测试；最后生产 consumer 退出后的 `internal/shein/sherr` 全包；仅该查询链使用的 `BrandItem/BrandListResponse`；§5/§6.1 的原 guard 和文档尾部。

Must：保留完整 `Resolved` 值、`WithResolved/FromContext`、现役清理和品牌保留/替换规则、当前草稿/预览/严格 JSON/规范化/hash/版本/读取拒绝、Product/SKC/SKU/属性 DTO、当前采集/原 Amazon 浏览器/debug、AI/图像、有效共享基础层、官方 Store 与独立 SHEIN 登录/Store 同步、Organization/授权/owner/claim/fence/UNKNOWN。workflow、完整 inventory（CURRENT/历史/计数/ceilings/accuracy/parser/检查）、依赖/锁/保护零改。

Should：复用原 owner、原 guard 和 Go 标准库 AST，不新建 Resolver、错误框架或查询 adapter。Out of Scope：平台品牌查询重新开放、品牌授权事实源、认证/IAM/租户改造、表/schema/migration/数据处理、其它平台 API 或 DTO 清扫、共享韧性/定价/图像重构。Accepted Risks 无新增；既有环境消费者、远端数据/凭据和在途请求 UNKNOWN，不读取、不停止、不清理。

Threat Model 限误删仍有调用者的品牌值/context/清理/草稿 JSON，旧字符串错误分类被转接到现役 worker，及原检查保活或 missing-skip；不扩展到无关全局安全目标。源码退休不证明真实环境停止或产品验收。

## 2. Design Basis、根因及前批 KEEP 替代

根因是把“存在包内引用”留作后续边界，却没有继续证明完整运行入口：#926 移除了真实品牌 API Client，Resolver 仍可通过窄接口和自身测试编译，但当前装配不构造它；它继续引用 `sherr` 和品牌响应 DTO。设计以 contract → implementation → injection → consumer 判定当前职责，不能只见 import 就永久保活，也不能只见零调用就越过用户已批准的 KEEP。

拟实施采用 Independent Architecture，集中确认此前明确 KEEP 的品牌查询合同和响应 JSON 退出；当前品牌值合同及持久化/授权 owner 不变。没有新增框架或外部副作用，不为复核设计重开无关架构。以下替代只在用户确认后生效：

| 旧依据 | 精确替代对象 | 继续有效 |
| --- | --- | --- |
| [#926 旧平台 API 闭包](legacy-platform-api-closure-retirement-boundary.md) §3/§5.2 的品牌 DTO 和整个 authorizedbrand KEEP | `resolver.go`、`resolver_test.go`、`Config`、`BrandItem/BrandListResponse` 的当前消费者/完整保留含义 | `Resolved`、context、现役清理/草稿、其它 Product/SKC/SKU JSON、worker/session/Store 全保留 |
| [#867 来源尾部](legacy-numeric-acquisition-tail-closure-retirement-boundary.md) 的 Config/Resolved/context 与其余 sherr KEEP | 只有 Config、旧 Resolver 和本轮五个 sherr 工件 | 原数字 Store adapter/旧 Cookie error 不复活；Resolved/context 与品牌清理保持 |
| [#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#817 SHEIN 刊登](legacy-shein-listing-closure-retirement-boundary.md) 的 sherr/authorizedbrand KEEP | 旧查询错误重试/认证/过滤分类及其专属测试、旧 Resolver | TaskHandledError 等已退内容继续缺席；当前业务值与平台规则不动 |
| [#831 Store 尾部](legacy-numeric-store-management-closure-retirement-boundary.md)、[#833 内容配置](legacy-numeric-content-management-closure-retirement-boundary.md)、[#836 运行配置](legacy-runtime-config-closure-retirement-boundary.md)、[#912 策略合同](legacy-shein-policy-contract-closure-retirement-boundary.md) 的广泛包 KEEP | 仅 §5 的旧查询与错误链 | 当前 Store、清理、共享值和其它明确 KEEP 不变 |
| 旧品牌设计规范/实施计划、Task 生命周期和 worker 身份文档 | 仅旧查询/错误 owner 的现役建议或装配图 | 原完整历史/批准/数字/勾选保留，当前 worker 身份及同步合同不改 |

没有生产 EXTRACT：`findByCode/findByName/trimMatchKey/newResolved` 只服务旧查询 Resolver；“正确的旧算法”本身不构成当前消费需求，不为它们新造公共 helper 或替代 owner。当前品牌值清理/装配直接保留已有实现，不经旧 Resolver。

## 3. 实际消费者与 owner

调查解析基线全部 **3,289 tracked Go** 的 import token、解码路径和 alias/selector，覆盖测试、build tags、嵌套 debug 模块；检查包内构造、helper、非 Go 引用、原守卫及历史依据。没有 dot/blank import 隐藏的相关调用。只读分析不运行旧构造器、业务 CLI 或请求。

| 合同/实现 | 注入与实际 consumer | 决策 |
| --- | --- | --- |
| authorizedbrand.ProductAPI → QueryBrandList → Resolver | NewResolver、Resolve、ResolveForProductBrand 仅 `resolver_test.go` 构造/调用；当前 app/runtime、publishing/marketplace、SHEIN worker 不构造；真实旧客户端已随 #926 退出 | 接口/Resolver/全部方法及专属测试 RETIRE |
| Config | 只在上述 Resolver 与测试消费；当前 CleanupContext/assembler 不消费 Config | 从 types.go 删除 Config；整个 Resolved 声明保留 |
| sherr | 唯一包外生产 importer 为 `authorizedbrand/resolver.go`；其余只在 Resolver 和自身测试；没有当前 worker 或官方 adapter importer | 与最后 consumer 同批整根 RETIRE，不搬认证/重试/字符串分类 |
| BrandItem/BrandListResponse | 包外 selector 只在 Resolver 及其 stub；没有当前 payload、schema、repository、HTTP route 或 worker 使用 | response.go 整退，不能扩大到 product 包或其它 JSON |
| Resolved/context → publishing/shein/assembler.go | 原 FromContext 读品牌 code/name，并写现有草稿/metadata；无远端查询 | 原 owner、值与行为 KEEP |
| Resolved/context → sensitive_content_sanitizer.go → content.CleanupContext/processor.go | 原上下文值进入现有清理；原合法品牌/替换和 payload 测试消费 | 生产路径、fixture 及全部有效断言 KEEP |
| sheinlogin/session_identity.go → SessionTransport | 商户、仓库、地址三个只读端点，原 Store 同步 owner/claim/fence/UNKNOWN；无 Resolver/sherr/品牌列表 DTO 边 | 全 KEEP，无配置/持久化/平台行为变化 |

`internal/shared/resilience.Retry` 仍服务 ImageAgent 和计费使用记录；不能因本轮错误包退出删除该共享 owner 或改重试策略。其它闲置工具和韧性实现不加入本闭包。

## 4. 状态、持久化、JSON 与副作用边界

### 4.1 保留能力与唯一 owner

品牌 `Resolved` 是当前装配/清理使用的内存值，其完整四字段 Enabled/Code/Name/NameEn 与原类型/零值保持；contextKey、WithResolved、FromContext 的 nil/disabled 拒绝和值复制行为、所有原 context 测试逐字保留。它不是 Organization 认证或品牌权利事实源，本轮不新增授权推断、自动查询或默认品牌选择。

草稿及当前 Product/SKC/SKU/属性、ValidationResult 的持久化与 JSON 仍由现有 publishing/marketplace/product/repository owner 管理；原字段、tags、嵌套结构、规范化/hash/版本和读取拒绝保持，不改数据库、schema、迁移或写入路径。

### 4.2 删除对象没有现役持久化 owner

旧 Config/Resolver/sherr 只拥有本地输入和 error 值；品牌列表 DTO 只作为旧接口的内存响应及 stub。完整消费者检查未见它们被当前 repository/schema/HTTP 注册或作为草稿嵌套类型。删除 response.go 不删除当前草稿中的品牌字段，不把“字段未被 selector 读”当裁剪其它 JSON 的理由。

全新安装与既有表/行均无行为变更；不建立或删除表，不改 Goose/迁移登记，不增加迁移、双读/双写、fallback 或第二事实源。已有配置、记录、凭据、文件/会话/进程不读、不写、不删。若实现时出现具体当前外部可观察或持久化依赖，停止该部分并报告用户，不私自加兼容。

### 4.3 平台与失败归属

旧 QueryBrandList 接口虽描述读取，但当前无真实注入；退出它不触发平台请求、注销、撤销或远端清理。旧认证字符串、404/业务过滤、RetryableError 判定随旧 owner 退出；不得迁入当前 SessionTransport、官方 adapter、共享 resilience 或独立登录 worker。

当前权限/租户、幂等/事务、retry/UNKNOWN/recovery、预算/计费、deadline/资源限制均无变更；没有新增平台副作用或恢复 owner。现役 Store UNKNOWN 仍不自动接管。生产运行/平台/用户验收 NOT_RUN，本设计不是内部试用前置。

## 5. 精确删除 / 保留矩阵

### 5.1 八个整退工件

| 位置 | 准确文件及全部专属内容 | 数量 |
| --- | --- | --- |
| internal/shein/authorizedbrand | resolver.go、resolver_test.go：ProductAPI、Resolver、NewResolver、Resolve/ResolveForProductBrand/queryBrandList、resolveFromItems/findByCode/findByName/newResolved/trimMatchKey 与全部 stub/专属测试 | 2 |
| internal/shein/sherr | auth_errors.go、classification.go、filtered_errors.go、retryable_errors.go、errors_test.go：全声明/方法/原测试；整个目录退出 | 5 |
| internal/shein/api/product | response.go：只含 BrandItem/BrandListResponse | 1 |

### 5.2 两个 mixed 工件与当前保留边界

| 工件 | 唯一允许修改 | KEEP |
| --- | --- | --- |
| internal/shein/authorizedbrand/types.go | 只删除 Config 声明及其专属注释/空行 | Resolved 完整声明逐字保持 |
| tests/listingkit_backend_retirement_test.go | 只 §6.1 的原 `assertLegacyPlatformAPICallClosureRetired` helper | 全部原测试入口、其它函数/global/helper 逐字零改 |

全文件 KEEP：authorizedbrand/context.go、context_test.go；全部 content 生产/测试；publishing/shein/assembler.go、sensitive_content_sanitizer.go、authorized_brand_test.go、submit_prep_test.go 及其它当前草稿/预览/规则；其它 Product/SKC/SKU/属性 DTO；当前 session_transport.go、cookie_values.go、三端点；sheinlogin、官方 integrations/Store、采集/原浏览器/debug、AI/图像、共享基础层、当前身份和计费。没有新生产文件、DTO 迁包、字段替换、alias 或适配器。

### 5.3 十二份专属历史建议与三份当前文档

未来实施只对以下原文增加准确 superseding 顶注；完整原文/顶注/数字/勾选/批准/链接/证据逐字保留为后缀。只替代 §2/§5 的旧 Resolver/Config/sherr/品牌列表 DTO 含义，不 blanket 撤销 authorizedbrand 或平台/JSON KEEP：

1. docs/architecture/legacy-platform-api-closure-retirement-boundary.md
2. docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md
3. docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md
4. docs/architecture/legacy-numeric-content-management-closure-retirement-boundary.md
5. docs/architecture/legacy-numeric-store-management-closure-retirement-boundary.md
6. docs/architecture/legacy-runtime-config-closure-retirement-boundary.md
7. docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md
8. docs/architecture/legacy-shein-listing-closure-retirement-boundary.md
9. docs/architecture/shein-worker-identity-persistence-boundary.md
10. docs/architecture/task-status-lifecycle.md
11. docs/superpowers/plans/2026-06-16-shein-listing-authorized-brand.md
12. docs/superpowers/specs/2026-06-16-shein-listing-authorized-brand-design.md

本文、Architecture README 和 Legacy Register 三份当前文档同步准确状态/边界，两个索引原完整正文保留为后缀。#923 Prompt 文档关于 WithResolved/合法品牌清理的 KEEP 继续有效，全文件零改。所有 inventory、历史 test-baseline.txt、依赖、锁、workflow、lint/保护规则均零改；历史文字或不存在路径的 allowlist 不要求无价值全仓清扫。

实施合计 **8 整退 + 2 mixed + 12 历史顶注 + 3 当前文档 = 25 工件**，一个主要 PR，不机械拆批。当前设计实际只修改三文档，以上是未来获准实施的完整清单。

```text
Legacy decision: RETIRE
Reusable behavior: 当前品牌值/context/清理/草稿规则原 owner 原样保留；无生产 EXTRACT
Current owner: authorizedbrand.Resolved/context、现役 content/publishing/marketplace、SHEIN login/session/Store
Cutover/deletion condition: 设计复核、用户确认§1–§7/§6.1、IMPLEMENTATION_READY/Ready记录与读回后，同PR一次收齐；合并另授权
```

## 6. 实施顺序与必要验证

获用户确认及 Ready 读回后，在原 helper 中转换准确 KEEP 为明确缺席，先记录基线对象仍存在的真实 RED。随后同批退出八文件、Config、原 helper 的准确条目，添加十二窄顶注、校准三当前文档至 GREEN；不跑旧构造器/请求，不为旧 resolver 测试造新查询 owner。

必要本地自检只覆盖原三退休 guard、authorizedbrand/context、content/品牌清理、publishing/marketplace SHEIN 草稿/JSON/预览、独立 worker/Store 的原适用合成 fixture、四项 Architecture README 检查。复用当前八命令依赖检查和必要当前命令编译，产物放任务 temp，不启动入口。不运行需要既有 .env/私密配置/实例的整包命令；CI 自有隔离 fixture 与运行验收分开，未变化代码证据明确引用，完整原 CI 留稳定候选，不重复全量或建设 runner/验收平台。

最终核对25准确路径、两个 mixed 允许区间、十二原历史后缀及两个索引后缀，当前保留声明/方法/测试和 workflow/inventory/依赖/锁/保护零差异。发现真实范围外消费者或原 CI 要改冻结零改项时，先一次查齐同根问题并报告准确最小补丁，不自行扩权。

### 6.1 原检查/测试精确转换矩阵

| 原位置 | 唯一允许转换 | 必须保留 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go / assertLegacyPlatformAPICallClosureRetired | 现有 retiredRoots 加 internal/shein/sherr，使整个目录 Lstat 必须缺席及全 tracked decoded import 禁入；现有退休文件列表加 Resolver 两文件和 response.go；requiredByFile 仅移除 response.go 的 BrandItem/BrandListResponse 必须存在行；retiredByOwner 给 authorizedbrand 加 Config、ProductAPI、Resolver、NewResolver、resolveFromItems、findByCode、findByName、newResolved、trimMatchKey 九项，三个 Resolver 方法由原 receiver 限定逻辑覆盖，给 api/product 加 BrandItem/BrandListResponse；requiredByFile 加 types.go 的 Resolved 及 context.go 的 WithResolved/FromContext 正向保护 | 原 helper 中其它文件/根/声明/DTO/端点/字段断言、parser/读取/解码错误处理逐字保持；不删除或修改其它原测试入口/global/helper；禁止泛名 Config/API/Resolver 全仓禁词 |
| 同 helper / tracked import 与 owner 限定 | 复用原 trackedFiles/parser/decodeGoImportPath/importMatchesPrefix，覆盖全部 tracked Go、测试、build tags/嵌套模块；只精确禁 sherr 根/子根，authorizedbrand 与 product 根仍保留；对 owner-qualified 退休声明新增准确禁入 | 当前 DTO、Resolved/context 正向保留；无 missing-skip、新共享 scanner 或其它 owner 误伤 |
| authorizedbrand/resolver_test.go、sherr/errors_test.go | 随专属旧执行 owner 整退；旧远端匹配/字符串重试语义不复制 | 原 context_test.go、content/processor_authorized_brand_test.go、cleanup_context_test.go、publishing/authorized_brand_test.go、submit_prep_test.go 等当前 fixture/业务断言全文件零改 |
| 其它原检查 | **零改**：TestCurrentCommandsDoNotCompileLegacyBusinessBackends、TestListingKitTaskBackendSourcesAreRetired、TestNumericAcquisitionAndPlatformTailClosureIsRetired、TestNumericListingSupportClosureSourcesAreRetired 及其它原函数，import_boundaries全文件、architecture_docs全文件、legacy_hard_cut、SKU/JSON/当前worker/Store/并发安全、lint/workflow/inventory | 原入口继续调用同一受管 helper，不新增退场单测/豁免或删除断言；现有 numeric support 声明扫描遍历全 tracked 源码，不依赖 sherr 目录必须存在 |

矩阵之外不改原函数、helper、global、当前 JSON/品牌值或有效测试。上表对 legacy 声明按 owner/receiver 限定；保留原独立 context 函数和 contextKey，不创建新的当前 API。

## 7. 审查、交付与停止点

### 7.1 设计准入与事实界限

本轮授权只是纯设计三文档；设计原 CI/实际审核 SHA/结论记录在主要 PR。适用独立设计复核聚焦前批 KEEP 替代、真实运行入口、旧 JSON/持久化边界、完整25工件、§6.1 与当前保留能力。设计候选形成后用户确认，先在 Issue/PR 记录并读回 IMPLEMENTATION_READY / Ready，再由唯一原 Writer 继续同一 PR 整批实施。设计证据不冒充实现、自检或运行证据。

finding 先按 AGENTS 分类，指出当前 Must/后果/阻塞层级；新事实确实命中 BLOCKER 才重开架构。普通实现/检查问题同一 PR 收敛，修复后只复核增量；最多两轮正常架构评审，不追求无限理论零 finding。

### 7.2 最终交付

形成完整实施后一次只读最终检查真实 diff、24工件、全部保留能力和既有证据。未变化代码证据可明确复用，main 同步只处理真实组合风险，不为 unrelated 模块重复全量审核。实现者自检/CI/AI review 不等于产品验收。最终交付准确 HEAD、原 CI 和真实独立复核后，由用户另行授权 expected-head 正常合并；合并后仅核对实际 merge SHA 匹配 push/main 完整 CI。

### 7.3 本轮权限

设计只提交/推送/维护同一主要 PR；生产实施、合并、部署、#301关闭和真实环境操作均 **NOT_AUTHORIZED**。未经用户本轮授权不默认新增 AI 互验线程；GitHub 复核同 SHA 已运行不重复触发。当前没有本轮子代理授权，不能复用 #926 的授权新增设计或源码复核者。

保留当前能力、独立 SHEIN 登录与 Store 同步；不操作既有实例/数据/凭据/进程，不读取实际 .env，不增加内部试用前置，#301保持OPEN。没有旧数据迁移或停止既有环境的授权，UNKNOWN 不自动接管。
