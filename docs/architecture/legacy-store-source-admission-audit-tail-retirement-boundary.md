# Store 旧来源准入与审计尾部完整退休边界

## §1 用户结果、Design Basis 与授权阶段

用户本轮明确授权：先完成四对象、专属测试、有效安全断言映射和相关文档的完整退休设计，同一主要 PR，不机械拆批，保留当前能力；安排一名独立 Reviewer，设计及原 CI 通过后交用户确认。**确认前不实施。**

- Execution Issue：[#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301)，保持 OPEN。
- 唯一 Writer：聊天 `01a194f8-c65c-7231-9e0d-acb39710885c`；独立工作区 `issue301-browser-tail-946/task-processor`；分支 `codex/issue301-store-source-audit-retirement`。
- 调查/开工基线：main `6cca95a556ea203e9fc18bb6e667e524e3459ef8`，已包含 #1044。开工已 fetch、检查当前 Issue/开放 PR/worktree，无同范围 Writer 或 PR；#1038/#1043/#1045 归原 owner。
- Design Basis：**Independent Architecture**。仅界定旧公开来源准入合同退出及安全断言的当前 owner，不重开 Store 全局架构，不新增授权机制。当前 **DESIGN_ONLY / DESIGNING / USER_CONFIRMATION_PENDING / SOURCE_NOT_STARTED / NOT_READY**。
- 本轮只提交 §5 的 D1–D4 四份设计/导航文档；P/T 和 D5 都是待用户确认的未来范围，当前零修改。设计复核和 CI 不自动授予实施权限。

使用者为维护当前 Store 管理、正式来源/商品用途授权及安全回归的开发者。可见结果是四对象的真实消费闭包、原三测试函数的逐项断言去向，以及准确删除/保留矩阵；未来回归直接验证现役 owner。**未证明当前用户业务被这些闲置对象阻塞**，不新增业务入口、试用前置、治理或验收工具。

Authority：本轮用户决定、[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)、[当前 Store 架构](store-center-current-application-v1.md)、[Store 观察](store-center-platform-observations-v1.md)、[当前供应链](my-supply-chain-shein-v1.md)、[#1041 资源退休边界](legacy-resource-transaction-closure-retirement-boundary.md)及[#1044 前批边界](legacy-imageagent-control-summary-tail-retirement-boundary.md)。没有 UI/IA/页面/交互变化，Figma 增量 **N/A**。

Must：四对象与专属旧调用完整退出；有效组织/Store 身份、撤权/生命周期、依赖失败/取消不成功放行的断言直接落当前 owner；当前管理读取、正式用途的 member/grant/service/connection 准入、审计/JSON/hash/资金/事务/UNKNOWN/恢复及观察保持。Should：复用当前函数、原测试/SQLite/GORM/Testify 与 AST/冻结后缀模式。Out of Scope：其它零引用候选、新来源导入功能、状态机/权限/持久化/恢复协议重写、兼容或数据迁移、新平台/验收系统、workflow/inventory/依赖/保护变更、部署/真实资源操作及关单。未新增 Accepted Risk。

## §2 精确旧要求替代与 Legacy 决策

| 依据 | 本批准确解释 | 保留 |
| --- | --- | --- |
| 旧 `source_access.go` 的“来源导入目标必须是活跃 SHEIN Store”组合 gate | 没有生产消费者；只退休此 Go 接口/函数及专属测试调用，不把它重建为当前来源、Store GET 或其它平台的前置 | 当前正常来源能力、Store 管理读取及正式操作用途各自原合同；不新增“无店铺导入”产品能力 |
| 当前 Store/旧计划对“审计、错误分类、失败关闭”的广泛 KEEP | 仅删除两个无调用的私有错误转换 helper，不退出整个审计文件、失败码、失败事件或错误值 | `AuditEvent`/Repository、全部字段/tag/枚举、规范化、审计记录/重放、当前业务错误处理及持久化 |
| #1044 的 Store 暂缓/Out of Scope | 对 #1044 原批范围始终有效；用户新决定只允许本新 PR 做设计，确认前仍不实施 Store | 不改写 #1044 的冻结正文，不把历史暂缓当成本轮设计禁令或实施授权 |

Legacy decision：**RETIRE**。Reusable behavior：原生 Store 管理、`ReadProductExecution/readOfficialMaterial`、正式商品 handle、现役审计/错误处理已经在正确 owner 内，本批只复用，不新增生产 EXTRACT。Current owner：Store Center domain/Repository/Service、`app/storecenter` 正式用途访问与现有 HTTP/组织成员授权、Resource/Store service 及观察各自原 owner。Cutover/deletion condition：用户确认 §1–§7（含 §4.2/§6.1）后，同一 PR 收齐 §5/§6.1，必要检查和适用源码最终复核完成；合并另授权。无 alias、转发壳、旧 gate 测试内重建或第二事实源。

## §3 四对象与实际调用闭包

检索覆盖全仓 tracked Go，包含测试/build-tag 文件；声明本身不算消费者，文档/guard 字符串不算运行调用。

| ID | 对象 / 文件 | 实际消费者 | 未来决定 |
| --- | --- | --- | --- |
| O1 | `SourcingStoreReader`，`internal/storecenter/source_access.go` | 只作为 O2 参数类型，无其它实现合同消费者 | 随整个文件退休 |
| O2 | `ValidateSourcingStoreAccess`，同文件 | 无生产调用；`source_access_test.go` 三函数共七个调用表达式 | 有效断言按 §6.1 映射后连旧文件退出，不保留同名替代 |
| O3 | `auditFailureFor`，`internal/storecenter/audit.go` | 只有声明，无生产/测试调用 | 只删函数体及声明 |
| O4 | `stableFailureFromAudit`，同文件 | 只有声明，无生产/测试调用 | 同上；不删 `AuditFailureAlreadyExists` 等仍属现役合同的值 |

旧测试闭包只有 `sourcingStoreReader` fake、其 Get 方法及三个 `TestSourcingStoreAccess*` 函数；没有共享测试消费者。O3/O4 没有专属测试，不虚构需要删除的审计测试。

当前 Store 接线为 `currentapplication.ApplicationFeatures.StoreCenterDB` → `cmd/current-application` 追加 `httpapi.WithStoreCenter` → current application 的 `factories.buildStoreCenter` → `buildCurrentStoreCenterModule`：原 runtime 提供 native record pool，builder 检查 schema/权限后构造 member-scoped repository / `NewGormAuditRepository` / `NewService`、现有 Store lifecycle executor及当前 Store HTTP，不消费四旧对象。正式商品访问沿 `OfficialProductAccess` → `MemberScopedStoreRepository.ReadProductExecution` → `readOfficialMaterial`，使用原成员/用途 callback、grant、记录/服务/连接/attempt 当前事实；实际发送前由原 handle 重新读取和授权。当前管理 GET 与正式执行准入不同，不让前者借旧来源 gate 改变可读范围。

架构问题是旧来源准入壳及测试仍与真实当前授权入口并存，两个审计转换 helper 也已没有 owner 调用。无生产入口不等于当前存在越权缺陷；处理方式是删除孤立壳、把有效断言放在实际当前入口，不另建授权或错误恢复层。

## §4 运行、授权、事实与副作用边界

### §4.1 未来整批实施顺序

1. 用户确认前不写 P/T/D5；本轮只读核对映射与文档检查。
2. 确认后先加 T3，在四对象和两旧文件尚存时记录真实 RED，再在 T2 追加 §6.1 指定的当前入口测试；原测试/fake/输入与断言不改。
3. 删除 P1/T1 的完整旧源及专属 fake/三测试，P2 严格只删 O3/O4；其余生产函数、字段、常量、imports 和全部现役审计测试零差异。
4. T2 直接调用当前 `ReadProductExecution`，不得复制 O2 算法、重造 `SourcingStoreReader` 或包成新的“兼容来源校验”。原内存 SQLite/GORM 表和 fixture 模式沿用，可在同一文件追加一个仅初始化当前事实/计数的局部 fixture，不新增工具/框架或生产 helper。
5. 同一 PR 收齐代码、测试及 D1–D5 窄说明；运行相关原生 Store/HTTP/app Store 测试、原 guard 和原 CI。发现 current owner 需要生产行为修复，先给具体当前 Must、分类及最小差异，不能借测试映射直接扩写准入。

### §4.2 必须保持的具体边界

| 边界 | 不变量 |
| --- | --- |
| 当前授权 | 原 Effective Organization、Actor/Member、用途、module/member Store grant、管理员规则、撤权与 rejoin 处理保持；不新增授权事实或旁路，不把旧“导入只认 SHEIN”推广到当前多平台 Store |
| 管理/用途分离 | 现有管理 GET 可管理 disabled Store 的行为不变；正式商品用途仍由原 reader 检查 active record、有效服务、connected connection、verified/bound 原 attempt 等。当前 TEMU/TikTok/Walmart 合同与原测试保持 |
| 身份与输入 | 当前 org/actor/member/StoreID 校验、限定 SQL、权限拒绝及不泄漏私有信息保持；旧测试 fake 的 Get 次数/参数形状不成为新领域协议 |
| 取消/依赖错误 | 当前 `readOfficialMaterial` 取消时 `ErrDependencyUnavailable`、非法 scope/无材料时 `ErrNotFound` 等原分类保持；不为保留旧 helper 的 `context.Canceled` 或 typed-nil 错误外观改动当前生产错误映射。保留取消/依赖失败不返回材料、不授权效果的实质断言 |
| 审计/持久化 | `AuditEvent`、全部 JSON 字段/tag、action/outcome/failure 值、normalization、request/action 幂等、actor/时间/字段白名单、payload fingerprint、审计重放/修复保持；无表/列/索引/Goose/SQL/grants/schema变化，不重解释已有失败记录 |
| 事务/资金/恢复 | 当前 Store CAS/锁/事务、记录与 paid service、Resource ConsumerCharge/#1041 图像结算、receipt、UNKNOWN/fence/响应丢失与恢复 owner 保持；不新加跨库原子性、重试/补偿/对账 |
| 外部效果 | 无新的 provider/DB/消息/浏览器动作；真实发布、观察、授权、服务及真实发送前复核沿原 owner，不因删除旧壳重发或延长凭据/handle 生命周期 |
| 真实资源 | 不访问/修改既有业务库、历史、凭据、配置、缓存、Profile、实例或进程；不部署、启停或清理。真实运行/产品验收 NOT_RUN，物理运行状态 UNKNOWN |

若发现四对象有清单外真实消费者/外部可观察义务，停止该部分并报告证据；不能用旧测试或文档推定兼容义务并新增 alias/fallback。

## §5 同一主要 PR 的完整范围矩阵

未来确认后收齐 **10 唯一路径：2生产、3测试、5文档**。本轮设计阶段仅 D1–D4；小步骤仅作提交/验证边界，不为源码、测试或文档机械拆 PR。

| ID | 路径 | 准确处置 |
| --- | --- | --- |
| P1 | `internal/storecenter/source_access.go` | 整文件 RETIRE，只包含 O1/O2 |
| P2 | `internal/storecenter/audit.go` | 只删 O3/O4；其余完整代码/值/字段/imports零差异 |
| T1 | `internal/storecenter/source_access_test.go` | 映射收齐后整退专属 fake/Get/三旧测试，无其它消费者 |
| T2 | `internal/storecenter/product_execution_test.go` | 只在原文件末尾追加 §6.1 的两个测试及其最小局部 fixture/计数；原完整文件作为前缀保持，原两个测试与 fake/方法/imports不改 |
| T3 | `tests/legacy_store_source_audit_retirement_test.go`（未来新增） | 两旧文件缺席、四对象禁入、当前 owner 必需、冻结 D1–D5 后缀与 T2 原完整前缀守卫；复用原 AST/读取/hash模式，不新增检查平台 |
| D1 | `docs/architecture/legacy-store-source-admission-audit-tail-retirement-boundary.md` | 本完整设计；用户确认后正文冻结，未来只前置准确当前注记 |
| D2 | `docs/refactoring/legacy-register.md` | 只前置四对象的设计/退休导航，保留全部原登记/历史/计数/勾选 |
| D3 | `docs/architecture/project-boundaries.md` | 只前置旧 gate 与现役 Store/审计 owner 的精确边界 |
| D4 | `docs/architecture/README.md` | 新设计显式反引号文件名索引，原导航完整后缀保留 |
| D5 | `docs/architecture/store-center-current-application-v1.md` | 未来确认实施时才加窄前注，仅说明四对象退出及当前授权/审计继续有效；原事实/设计完整保留 |

用户确认的准确设计 HEAD 作为 D1–D5 完整 Git blob 后缀及 T2 完整前缀的冻结基线（工作区检查可归一化 CRLF→LF）；滚动 HEAD/CI/评审进度写 PR。无需改 #1044 冻结文档，其 Store Out of Scope 描述原批历史，新的设计/实施权限由当前用户决定与本 Issue/PR记录。

KEEP/零差异：`audit_test.go`、全部其它现役测试、Service/HTTP/member/grant/official/product/observation/收费/浏览器/生命周期源码、wire/DTO/hash/当前 ledger、Store/Source/图像当前能力保持；所有 workflow、完整inventory/ceilings/算法、依赖/锁/保护、schema/配置/部署文件与其它候选零改。不接管其它 owner，不新增验收或试用前置。

## §6 验证方法与有效安全断言映射

本轮只核对四对象/全仓消费者、原测试实际断言、当前 owner 的真实接线及错误合同、10路径唯一性、链接/原文后缀和四文档 diff，运行原适用文档检查及原 PR CI。一名独立设计 Reviewer 只读实质核对；不写未来测试或先删生产代码，不执行应用/provider/实例/数据操作。

### §6.1 原三测试函数逐项映射

T1 原测试共有七个 O2 调用表达式；循环实例和 fake 计数不是额外生产消费者。每条旧断言明确属于当前安全语义或随旧 gate 退休的实现表面，不把未运行的未来测试记为 PASS。

| 原 T1 函数/断言 | 当前 owner / 准确测试去向 | 保留或退休的边界 |
| --- | --- | --- |
| `TestSourcingStoreAccessFailsClosed`：active 成功 | T2 新 `TestProductExecutionPreservesRetiredSourceAccessSafety` 的 active scoped success；复用现役 SHEIN record/service/grant/connection/attempt 内存 fixture，直接调用 `ReadProductExecution` | 返回当前有效材料且绑定原 org/Store/成员；不只伪造 active Store 指针 |
| 同函数：cross organization | 同新测试 foreign organization；原 HTTP `TestHandlerFailsClosedOnCrossOrganizationOrWrongStoreProducerOutput` 零改 | 不返回外组织材料；当前 SQL/成员和 producer-output 校验都保留，不复制旧 Get wrapper |
| 同函数：wrong store | 同新测试 nonmatching/missing StoreID；原 HTTP 同名函数的 wrong store item 断言零改 | 原“错误 Store 不得被放行”保持；native 按参数精确读取和 HTTP 拒绝错误 producer 输出分别验证 |
| 同函数：wrong platform=amazon | 同新测试 unsupported platform；现役 T2 `TestTemuNativeProductMaterialUsesCurrentStoreAuthorityAndExplicitPurpose` 全部原文零改 | 不返回不支持的平台材料；旧“来源导入仅 SHEIN”前置本身 RETIRE，不让当前合法 TEMU/TikTok/Walmart 被误禁 |
| 同函数：disabled / provisioning / deleting | 同新测试三个原 record status 值，当前有效服务/连接/grant fixture不变 | 正式用途无材料/`ErrNotFound`；不改成管理 GET 也必须拒绝这些可管理记录 |
| 同函数：missing Store | 同新测试 missing StoreID；原 HTTP `TestHandlerMakesUnknownAndForeignStoreNotFoundIndistinguishable` 零改 | 无材料，原当前 NOT_FOUND/隐私语义保持；旧 fake 返回 nil 指针这一形状随旧协议退出 |
| 同函数：repository failure | 原 T2 `TestProductExecutionRequiresCurrentOriginalMemberGrantServiceAndConnection` 的授权依赖失败及 DropTable 后数据库失败断言零改 | 原生错误为 `ErrDependencyUnavailable`，不伪装权威撤权；无需为同一既有证据复制故障测试或新建 fault 工具 |
| 同函数：每个 case 的 reader.org/id | 新测试当前 subject/original StoreID、返回 attempt身份及 foreign/nonmatching拒绝；原 HTTP `TestHandlerDerivesEveryServiceRequestOnlyFromEffectiveOrganizationIdentity` 零改 | 组织/Store限定读取的实质保持；旧 fake 的 Get 参数槽位/次数属旧接口实现细节，随 fake RETIRE |
| 同函数：空 org 在读取前拒绝 | 新测试 empty organization，追加只记录当前 authorizer调用的本地计数，期望无材料/`ErrNotFound`且authorizer零调用；原 `TestServiceReadValidationCallsNoDependencies` 零改 | 原 `readOfficialMaterial` 已在authorize/事务之前校验scope；不增新前置或改当前生产顺序 |
| 同函数：nil reader | 新测试 nil `*MemberScopedStoreRepository` receiver，直接当前方法、有效scope/authorizer/deadline，期望无材料/`ErrNotFound` | 当前缺失依赖失败关闭/不panic；旧 nil interface 外观与错误类型不新增兼容 |
| 同函数：已取消 ctx，`context.Canceled`且旧Get零调用 | 新测试 cancelled context，当前 authorizer计数不增加，无材料/`ErrDependencyUnavailable`；代码核对原ctx拒绝位于authorize/事务前 | 取消不成功放行保持；旧 `errors.Is(context.Canceled)` 外观和 Get计数对象 RETIRE，不改原生错误分类以适配旧测试 |
| `TestSourcingStoreAccessRechecksLifecycleOnReplay`：active→disabled，第二次拒绝/Get两次 | T2 新 `TestProductExecutionRechecksStoreStateOnNextRead`：同repo/subject/StoreID首次成功，仅fixture把record设disabled，第二次`ErrNotFound`，两次原authorizer调用 | 每次实际读取当前Store，不能缓存首次材料当新授权；不新建生产replay/cache/恢复owner。已有grant撤销/rejoin及发送前复核测试零改 |
| `TestSourcingStoreAccessRejectsTypedNilDependency`：typed nil reader→旧 `ErrDependencyUnavailable` | 与新测试 nil repository receiver 同一当前值；原 `NewService` 的isNilDependency与`TestNewHandlerRejectsNilAndTypedNilServices`等现役检查保留 | 失败关闭/不panic保留，旧接口装箱形状/错误表面 RETIRE；不改 `NewOfficialProductAccess` 或其它无关构造器，不以本批扩展全仓typed-nil hardening |

T2 只追加上述映射需要的两个测试与最小本地 fixture/计数，不复制全部当前回归或增加新威胁模型。原 T2 两函数/fixture代码完整保留为前缀；当前已存在的授权失败/撤权/rejoin/有效服务/连接/数据库失败检查不改 expected、不删除。这里的名字和 case 输入是未来允许的准确增量，当前尚未写入。

T3 在两原文件存在或四对象仍声明/引用时必须 RED；未来整退后 GREEN。O1/O2 的独立标识符应覆盖全仓 Go（含测试/build-tags）禁入，O3/O4 及旧专属 fake/三函数在整个 Store 子树禁入，不按通用 `Audit`/`Store` 子串封禁。必须读取/解析失败关闭；当前 AuditEvent/Repository/normalization、Service/原生材料及实用error/constants必需，D1–D5后缀与T2原前缀保持。不改其它原guard、inventory或CI算法。

确认后必要检查：T3真实RED→映射/删除→相关 `internal/storecenter`、`internal/storecenter/httpapi`、`internal/app/storecenter` 与新guard/原文档检查GREEN，稳定候选原CI与适用最终源码复核。实际未来测试、运行/provider/历史及产品验收当前均 **NOT_RUN**，设计CI不替代未来实现CI。不人工重复完整CI已覆盖检查，不新增runner/session/故障矩阵或验证工具。

## §7 准入、独立复核、交付与停止点

1. 本轮在同一主要 PR 提交 D1–D4，Issue记录接单及最新范围；源码阶段尚未Ready。新用户决定只授权Store设计，旧暂缓未被升级为实施/合并权限。
2. 稳定设计候选后只安排一名独立 Reviewer，核对全设计、四对象消费者、原三测试逐项映射、当前授权/错误/审计真实合同、§4.2/§5/§6.1及历史后缀；不默认增加Reviewer/Writer或新任务线程。
3. Finding先按AGENTS分类，写明当前Must、具体后果/阻塞层级；BLOCKER须命中明确类别。最多两轮正常Architecture Review；修复后同一Reviewer只复核相关增量，不扩Threat Model或验收矩阵。
4. 设计复核与准确候选原CI通过后交用户确认 **§1–§7（含§4.2/§6.1）**。没有用户明确确认，保持 USER_CONFIRMATION_PENDING / SOURCE_NOT_STARTED / NOT_READY，不登记 IMPLEMENTATION_READY，不改P/T/D5。
5. 未来确认后在本同一PR整批实施、修复和验证；设计review不替代源码最终复核。合并/实际mainCI、部署、真实运行/产品验收和父Issue关闭分别授权。#301 OPEN，不增加当前业务/试用前置。

停止条件：四对象新增清单外实际消费者/当前外部义务；有效断言无法按当前owner/错误合同映射；需要清单外生产修改或改变身份/权限、状态、schema/事实/事务/资金/审计值/JSON/hash/UNKNOWN/恢复/副作用；需要workflow/inventory/依赖/保护或既有资源操作。先报告准确证据、当前Must、分类和最小差异，不能临时发明兼容层或扩大生产修复。
