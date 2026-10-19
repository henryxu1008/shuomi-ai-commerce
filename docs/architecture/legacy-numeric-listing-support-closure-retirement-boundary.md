# 旧数字刊登共享管理与 Task/Workspace 尾部：完整退休边界

> Execution：[#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)
>
> Design Basis：**Independent Architecture**；状态：**DESIGN_REVIEW / AWAITING_USER_CONFIRMATION**，尚未达到 IMPLEMENTATION_READY。当前只交付设计，不修改生产源码或测试；用户确认与适用独立设计检查完成后，先在执行 Issue 登记 Ready，再实施。
>
> 调查基线：main `3ca37b2951c605ea22c96ea0e0274df6ee4f2527`，2026-10-19。上批 #822 已合入此提交，[对应 push/main CI 39202883365](https://github.com/qq550723504/task-processor/actions/runs/39202883365) SUCCESS；其授权与证据不覆盖本批。唯一 Writer 复用原独占 worktree 和分支 `codex/issue29-numeric-listing-admin-closure-retirement-design`；#821/#692 仍归原 owner。

## 1. 用户结果与一个完整 Delivery Batch

使用者是维护内部试用、当前采集、商品/图像/平台草稿、Store 观察及独立 SHEIN 登录 worker 的操作者。交付结果是：三条旧平台数字刊登执行链退出后，其无当前消费者的共享管理持久化、Task RPC、Workspace 转发层、数字 Task/状态/重试及专属指标/hook 一次收口。当前用户继续沿原入口操作，维护本身不作为内部试用的新前置，不宣称新增发布能力或完成父 Issue 全部目标。

这是同一个旧数字刊登架构的剩余支撑闭包，不按仓库、DTO、wrapper、目录、文件数或实现提交机械拆 PR。闭包在真实当前消费者处停止：Store、敏感词、话题、定价值规则、两种 SHEIN 属性、离线 validator 实际使用的 marketplace readiness 规则、平台 API、来源采集、Store observations 和独立登录 worker 不能被一起删除。marketplace workspace 的旧编辑/修复/历史/Task 投影同属闭包，不能因包内仍有 validator 消费者而全包保留。

本批拟整退 **200 个 tracked Go 文件（131 生产 / 69 测试，基线生产 13996 行）**，另精确清理七个 mixed 生产文件和三个 mixed 测试文件。没有整退的非 Go 运行工件。完整清单见附录 A；生产文件数/行数命中风险提示线，因此先形成独立设计，不通过拆批规避准入。

非目标：修复或重启旧数字刊登、续跑旧 Task、保留兼容 shell、建替代管理服务/Task RPC/状态机/指标平台/迁移或验收框架、移动旧表/数据、接管旧 owner、扩张官方平台或 AI 能力、改运行装配/部署/CI/workflow/依赖/保护。不访问实例、DB、Redis、队列、Profile、凭据、平台、模型或历史 Prompt 覆盖记录。

## 2. Product Authority、Design Basis 与不变量

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)：只有 EXTRACT / RETIRE；旧内部 Task、平台 Workbench 或兼容 shell 不是当前产品需求。
- [SHEIN worker 身份与持久化](shein-worker-identity-persistence-boundary.md)：原数字账号/Store、Redis/Profile、登录 attempt、查重与 Store ID 同步继续由原 owner 管理，不将其改成当前 Organization 身份或迁入当前业务。
- [SHEIN](legacy-shein-listing-closure-retirement-boundary.md)、[TEMU](legacy-temu-listing-closure-retirement-boundary.md)、[Amazon](legacy-amazon-listing-closure-retirement-boundary.md) 三个已确认整链边界：只窄替代本文 §5 明列对象的历史保留描述，其余保留能力与原证据不变。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[当前供应链](my-supply-chain-shein-v1.md)、当前 Product/Asset/Listing/Submission/Integration 合同继续作为事实、授权、版本、发送许可、receipt 与 UNKNOWN 的 authority。
- [旧 boundary checkpoint](../refactoring/listingkit-boundary-checkpoint.md) 仍允许 `internal/workspace/shein` 作为 compatibility shell；这是与当前 Hard-Cut 冲突的历史描述。本设计请求明确退休该 shell，原 checkpoint 加窄 superseded 注记而不重写历史。[external client inventory](external-client-boundary-inventory.md) 的 Task RPC 残留装配描述同样只作为退休前证据，不将它转为当前任务入口。

Independent Architecture 覆盖准确闭包删除和保留合同识别，**不新增或改变当前事实 owner、表/schema、身份/权限/tenant、事务、状态机、幂等、retry/UNKNOWN/recovery、AI 准入/预算/计量或平台副作用 authority**。没有新 UI/IA，Figma N/A。没有新增依赖、通用抽象或需要另造的开源替代组件。

Must：§5 的保留能力与原字段/JSON/类型身份、当前授权/owner/tenant/事务/数据约束保持；两套及全部 workflow、必需安全检查、.golangci.yml、ceilings、go.mod/go.sum 和保护规则零改。Should：用原测试/文档与一个主要 PR 清晰表达退休状态。Accepted Risks：无新增风险接受决定。

Threat Model 仅覆盖误删共享消费者或有效安全检查、把代码退休误作历史数据/运行资源清理、把旧 facade/DTO 重新包成当前 owner、混淆同名平台 DTO，以及遗漏旧入口防复活。理论扩展、全局治理和专项验收平台不在本轮范围。

## 3. 实际消费者、调用路径与根因

调查覆盖 tracked Go imports/限定名、构造器与 schema 方法调用、共享 helper 和原守卫；最终以标准Go AST解析全部 **4158 个 tracked Go 文件**，覆盖单行/分组/alias import，避免单纯文本限定名扫描漏掉编译边。`go list ./cmd/...` 为 **31 个命令**。`go list -deps ./cmd/...` 仍包含 listingadmin/listingruntime 及当前 `app/worker/imageagent`，但不含 taskrpcapi、workspace/shein、core/metrics 或 app/worker 根包。命令元数据不执行 main/constructor，也不证明运行实例或外部调用方不存在。

| contract → implementation → injection → consumer | 当前证据与根因 | 决定 |
| --- | --- | --- |
| 八类 listingadmin 管理模型 → GORM repo/query/facade → 原数字执行/HTTP | category/filter/profit/pricing/operation/product-data/import-mapping/raw-JSON 的包外生产 import 限定名、构造/注入及调用均已退出；仅本包相互引用及 fixture 存在 | 模型/表访问/管理 facade/DTO/专属 helper/旧测试整退；不是删除整个 listingadmin |
| listingadmin.StoreRepository/StoreAPI → worker runtime/bootstrap → sheinlogin/sheinloginmanaged | 实际消费 Store/查询/分页/owner 角色、StoreIdUpdate/StatusUpdate 与管理 DTO；TEMU API client 还消费 StoreAPI/StoreIdUpdate | 全部原样保留；表、数字 tenant、StoreID 同步和不支持的 cookie/pause 行为不变 |
| listingadmin.SensitiveWord/GenerationTopic* → shein/submitprep 与 publishing/shein | 当前敏感词清理和话题策略/覆盖仍消费 repository/DTO/owner 上下文 | 全部原样保留；schema/owner/tenant/查询顺序与当前语义不变 |
| listingruntime → pricing、SHEIN pricing/activity/authorizedbrand/productsync | OperationStrategy、PricingRule、StoreInfo、ProductImportMapping 四种值合同有真实消费者；另五种声明没有消费者 | 保留四种合同的全部字段/类型/方法；只退五种孤立声明，不搬定价规则或压缩保留字段 |
| Task RPC DTO/Client → localTaskRPCClient/handler → BuildModule/HTTP module | 包外生产 importer/注册均为零；status 已固定返回 retirement unavailable，queue-stats/health 只剩未装配表面 | 六文件整退，连同三条 `/api/v1/management/tasks/...` 只读路由，不创建新的状态/队列/BusinessTask facade |
| workspace/shein → marketplace/shein/workspace/publishing | 45 文件是 alias/forwarder 及自身测试，没有包外生产 importer；删除后 marketplace workspace 的唯一包外生产消费者是离线 validator | wrapper 与失去最后消费者的80个 marketplace编辑/修复/历史/Task文件同批整退；保留 validator 的11生产/10测试文件闭包，mixed按§5.2收口，不新增包装层 |
| model.Task/TaskStatus/RetryDecision → app/worker.TaskJobHandler + core/metrics + sherr.TaskHandledError | Task hook无构造/包外importer；core/metrics无生产importer；重试函数只剩自身fixture；handled_errors.go仍形成同包编译边，但该Task错误无创建/断言消费者 | 六个Task文件、一个hook、两个指标和handled_errors.go整退；sherr当前Retryable/NonRetryable及authorizedbrand消费者、model.amazon_product、通用workerpool/当前BusinessTask/Temporal不动 |
| import_task_owner / DTO 值 helper / numeric helper / ensureTextColumn | store owner 锁 helper 只供本批 product-data/mapping；三项 DTO 值 helper 只供本批 facade；numeric helper 只供本批模型；ensureTextColumn 只供旧 operation strategy schema | 同闭包退休，不制造新 helper owner 或原 Service wrapper；其余 owner/schema/paging helper 保留 |

根因是“同包仍被当前代码 import”或“历史上作为共享合同保留”被误当成所有内部管理链仍有当前消费者的证据。只删顶层执行器或 Workspace wrapper 会留下可再次误装配的旧仓库/Task/编辑/修复图；反向整删 listingadmin/listingruntime/marketplace workspace 又会破坏 worker 和当前内容/规则能力。本批按实际声明及注入边分界，一次完成余下无消费者支撑图。#821 当前58文件差异无本闭包重叠，原owner保持。

## 4. 持久化与平台副作用 owner

| 边界 | 退休前 owner / 副作用 | 本批处置与保留 |
| --- | --- | --- |
| 八张旧管理表 | `listing_category`、`listing_filter_rule`、`listing_profit_rule`、`listing_pricing_rule`、`listing_operation_strategy`、`listing_product_data`、`listing_product_import_mapping`、`listing_raw_json_data`；原 GORM 读写/soft-delete/upsert/owner 锁在本批 repo 内 | 删除源码与无消费者初始化方法；**不发 SQL、不 DROP、不迁移、不删除/重写已有行/索引/约束**。没有当前 owner 接管或映射历史事实；当前初始化并未调用这些方法，不删任何 Goose 版本/runner |
| 保留的四张表 | `listing_store`、`listing_sensitive_word`、`listing_generation_topic_policy`、`listing_generation_topic_override` | 原 repo/API/AutoMigrate/owner 约束/tenant 过滤/错误及事务保留，不借 mixed schema helper 删除调整列或默认值 |
| 当前业务事实 | Organization/SourceAccountRegistry/Product/Catalog/Asset/CompletionDraft/Target/Submission/Store observations | 各当前 owner 与授权/版本/CAS/幂等/事务/receipt/UNKNOWN/recovery 全部不变，不新建事实源、迁移或恢复协议 |
| 数字 Task/Workspace/RPC/指标 | Task JSON/状态/重试、未装配 RPC、alias wrapper、进程内 singleton 指标；不是当前 BusinessTask 或 Temporal owner | 仅删源码，不续跑/转换旧任务、清队列、清缓存/指标、停止进程或处理历史文件 |
| 独立 SHEIN worker | 原账户/Store DB、Redis/Profile、登录 attempt、查重、Store ID 同步及平台会话各原 owner | 全部保留，不启动/停止/重启/探查实例，不迁移账号/tenant/Profile，不改变凭据或实际 Store |
| 平台/模型/图片/存储 | 平台 API/DTO/client、当前许可/发送/回执 owner、AI/Prompt/GovernedModel/图像与对象存储 | 不构造或调用真实 client；原准入、预算、取消/超时、重试、UNKNOWN 与恢复责任保持；代码退休无发布/数据操作授权 |

没有新增跨数据库一致性问题或新的多步持久化流程，因此不引入 outbox/Saga/协调器。保留 owner 在失败、取消、并发、重试、重启和撤权时仍按原合同处理，不用删除旧测试暗中接受新风险。

## 5. 删除、保留与抽取矩阵

### 5.1 完整文件删除边界

| tracked 范围 | 总数 / 生产 / 测试 | RETIRE 内容 |
| --- | --- | --- |
| listingadmin 八类文件族 | 45 / 30 / 15 | category/filter_rule/profit_rule/pricing_rule/operation_strategy/product_data/product_import_mapping 各六文件，raw_json_data 三文件 |
| listingadmin 专属补充 | 14 / 9 / 5 | 七个 GORM facade/fixture，import_task_owner 两文件，numeric_helpers/dto_value_helpers/rule_status，owner_scoped_batch_test 和失去全部调用的 owner_test_helpers_test |
| internal/taskrpcapi | 6 / 4 / 2 | DTO/Client/local status adapter/handler/module/两 fixture |
| internal/workspace/shein | 45 / 35 / 10 | 所有旧 alias/forwarder/自身 fixture/doc，不涉及 marketplace 下同名 workspace |
| internal/marketplace/shein/workspace 精确子集 | 80 / 46 / 34 | editor/inspection/preview/repair/revision/history/Task队列等旧投影及专属fixture；保留11生产/10测试规则闭包与§5.2准确声明 |
| internal/model 精确六文件 | 6 / 3 / 3 | task、task_retry、task_status 及对应测试；amazon_product.go 保留 |
| internal/shein/sherr/handled_errors.go | 1 / 1 / 0 | TaskHandledError、TargetStatus/ErrorMessage与New/As/IsTaskHandledError整退；errors_test.go已无此类型fixture，整文件保留 |
| internal/core/metrics | 2 / 2 / 0 | task_metrics.go、shein_metrics.go；platform workerpool Metrics 保留 |
| internal/app/worker/job_handler.go | 1 / 1 / 0 | TaskJobHandler 与完成通知/Task JSON hook；**不删 app/worker/imageagent** |
| 合计 | **200 / 131 / 69** | 附录 A 精确文件清单；不按泛名或整个 shared/core/app/model 根删除 |

### 5.2 mixed 精确删除

| 文件 | 允许删除 | 必须保留 |
| --- | --- | --- |
| listingadmin/management_api_types.go | 下列29 type、4函数/方法与6个 ShelfStatus 常量，清失去使用的 context/io/productpolicy imports | StoreRespDTO/StorePageReqDTO/StoreStatusUpdateReqDTO/StoreIdUpdateReqDTO/StorePauseStatusRespDTO/StoreAPI/PageResult 七声明原字段/tag/签名；types.FlexibleTime 依赖 |
| listingadmin/schema_migrate.go | 仅 ensureTextColumn：唯一 caller 是本批旧 operation_strategy_repository | ensureOwnerAuditColumns、owner 约束、sensitiveWordLegacyColumnMigrations、PostgreSQL 类型处理、ensureUniqueIndex 及其余全部实现/测试 |
| listingruntime/types.go | StorePauseStatusDetail、ProductImportMappingUpsert、StoreService 三声明 | StoreInfo、ProductImportMapping **原完整字段/tag/类型**，含历史命名字段，不把字段清理混入此次退休 |
| listingruntime/rules.go | FilterRule、ProfitRule、FilterRule.ToFilterRule；清失去使用的 productpolicy import | PricingRule 原完整声明；operation_strategy.go 与 OperationStrategy.IsEnabled 全部不动 |
| listingadmin/owner_scoped_repository_test.go | 只退 TestCreateCategoryRejectsOwnerlessWrite、TestCreateFilterRuleRejectsOwnerlessWrite、TestCreateOperationStrategyRejectsOwnerlessWrite、TestCreatePricingRuleRejectsOwnerlessWrite、TestCreateProfitRuleRejectsOwnerlessWrite 五旧模型 fixture | Store 两项、GenerationTopicPolicy/Override、SensitiveWord 的拒绝空 owner 断言、两个共享 SQLite/assert helper及原错误/零写语义 |
| marketplace/shein/workspace/helpers.go | filterNonEmptyStrings、normalizeText 及失去使用的strings import | uniqueStrings、firstNonEmpty 原值规则，不加trim/normalize或改变去重/顺序 |
| marketplace/shein/workspace/state.go | IsSaleAttributeResolved、hasResolvedSaleAttributeDraft | AutoReviewNotes、FilterManualReviewNotes、IsCategoryResolved、IsAttributeResolved，供现用模板与人工备注检查 |
| marketplace/shein/workspace/readiness.go | SubmitChecklist/ChecklistGroupItem两类型；BuildSubmitChecklist、ChecklistItemCount、SubmitChecklistGroupForKey、FindLabels、FindKeys、CloneReadinessItems、ToActionItems、findReadinessItem、JoinReadinessLabels 九函数及strings import | Guidance/ReadinessCheckSpec/ReadinessTaxonomy/SubmitReadiness/ReadinessItem/ReadinessCheck六类型与BuildSubmitReadiness完整实现，原状态、taxonomy、复制/顺序/JSON合同不变 |
| marketplace/shein/workspace/state_test.go、readiness_test.go | state仅退TestBuildEditorDirtyHintsCollectsUniqueFields、TestIsSaleAttributeResolvedRequiresResolvedDraftValues；readiness首个TestBuildSubmitReadinessAndChecklist仅退checklist部分/对应fixture | 原人工备注过滤断言；readiness原blocking/warning/ready/复制及taxonomy断言，必要时仅重命名首测试以表达仍测的行为；不删整个文件 |

management_api_types 的29个退休 type：

```text
FilterRuleReqDTO FilterRuleRespDTO FilterRuleAPI
ProductImportMappingCreateReqDTO ProductImportMappingGetReqDTO ProductImportMappingCheckReqDTO
ProductImportMappingRespDTO ProductImportMappingGetBySkuReqDTO ProductImportMappingGetByTaskAndSkuReqDTO
ProductImportMappingGetByPlatformProductIdAndStoreReqDTO ProductImportMappingAPI
ProductDataAPI ProductDataDTO ProductDataBatchSaveReqDTO ProductDataItemDTO
ProductDataBatchUpdateAttributesReqDTO ProductAttributesItemDTO ProductDataListByStorePageReqDTO ProductDataRespDTO
CommonResult RawJsonDataReqDTO RawJsonDataRespDTO ProductVariantConfirmationReqDTO RawJsonDataCreateReqDTO RawJsonDataAPI
ImageDownloader ImageInfo OperationStrategyAPI OperationStrategyDTO
```

四函数/方法：FilterRuleRespDTO.ToFilterRule、NewProductDataItemDTO、NewProductDataBatchSaveReqDTO、OperationStrategyDTO.IsEnabled。六常量：ShelfStatusPending/Reviewing/OnShelf/OffShelf/Rejected/Deleted。限定在 listingadmin 原声明，**不删平台同名 DTO、downloader.ImageDownloader、imagex.ImageInfo、当前 StoreService 或其它包同名规则**。

### 5.3 保留与复用矩阵

| 保留范围 | 当前消费者 / 保留条件 |
| --- | --- |
| listingadmin Store/SensitiveWord/GenerationTopic 全文件族，gorm_store_api | worker runtime/bootstrap、sheinlogin/managed、TEMU API、当前 submitprep/publishing；原表、schema、数字 tenant/owner/角色/错误/Store ID 同步保留 |
| listingadmin access_scope、owner_constraint、repo_paging、schema 其余行为与有效测试 | 上述 retained repo 的身份上下文、owner/tenant、分页/约束/错误底线；不放宽测试开关或请求权限 |
| listingruntime 四种有效值类型 | OperationStrategy/PricingRule/StoreInfo/ProductImportMapping；保留全部字段/方法及其真实 consumers，非整包退休 |
| marketplace/shein/workspace 的11生产/10测试文件，validator、marketplace publishing与publishing/shein | 保留 §5.4 真实readiness规则依赖；旧编辑/修复/回滚/历史/Task投影同批退出，不保留孤立编排，也不改变当前Product/CompletionDraft自己的编辑与历史能力 |
| 当前采集/图像/Reader/Cache/Mapping | sourceproduct 的 RawJsonDataClient 是其自有输入，不依赖本批 listingadmin.RawJsonDataAPI；crawler/fetcher、model.Product、来源证据与通用数据/图像能力不动 |
| 平台 API/DTO/规则、SHEIN 两种属性/AmazonMonitorData、PriceManager/PriceCalculator 与共享 pricing | 全部保持原 owner、类型身份与语义，不因管理库退出删除纯规则或改价格默认行为 |
| Store current application/observations、独立 SHEIN 登录 worker/Store 同步 | 当前 Organization/Store 与独立数字 worker 边界均保持，不把二者混成单一身份或新增桥接 |
| core/config/logger、platform/logging/workerpool、app/worker/imageagent、当前 BusinessTask/Agent/Temporal/Submission | 只删旧 core/metrics 和根 Task hook；基础层、当前观测、资源/AI 计量/恢复及用户任务不动 |
| shein/sherr 的auth_errors/classification/cookie_errors/filtered_errors/retryable_errors/errors_test | 仅handled_errors.go退出；其余错误/分类与全部原fixture保留，authorizedbrand的当前错误/重试语义不变，不把旧Task完成状态改成新错误协议 |

Legacy decision: **RETIRE**。Reusable behavior：本轮无须新 EXTRACT；实际有效行为与合同已在 §5.3 原 owner 保留，wrapper 重复测试不生成新实现。Current owner：上述当前 Domain/Capability/persistence 与独立 worker 原 owner。Cutover/deletion condition：用户确认、适用独立设计检查与 Issue Ready 后，原防复活 RED→完整源码/mixed 删除→保留回归 GREEN→准确实现 CI/最终独立检查；无运行迁移或清理。

### 5.4 validator 的完整保留闭包

当前 `marketplace/shein/validator.evaluatePackage` 实际调用 BuildPackageTemplateValidation、BuildSubmitTemplateReadinessChecks、BuildSubmitPayloadReadinessChecks、BuildSubmitPayloadValidationReadinessChecks、BuildSubmitReadiness 及其输入/结果/Guidance类型。其传递依赖只保留下列文件，另按§5.2移除无消费者声明：

```text
helpers.go
package_template_validation.go
readiness.go
readiness_taxonomy.go
source_facts.go
state.go
submit_payload_readiness_checks.go
submit_payload_validation_readiness_checks.go
submit_readiness_check.go
submit_readiness_review_checks.go
submit_template_readiness_checks.go
```

以validator九个实际selector为根，对生产Go AST声明/identifier闭包逐项核对，结果为上述11文件的30个声明；其余声明不因包级可达而保留。同目录保留10测试：package_template_validation_test、readiness_test、readiness_taxonomy_test、source_facts_test、state_test、submit_payload_readiness_checks_test、submit_payload_validation_readiness_checks_test、submit_readiness_check_test、submit_readiness_review_checks_test、submit_template_readiness_checks_test（均.go）。原fixture helper位于这些保留文件内。validator两种输入合同、Snapshot/error、纯离线/无副作用保证和 **RuleVersion `shein.offline_package.v1.1`** 保持；本批不改实际被调用的任何值规则，因此不借退休调整版本、返回状态/顺序、code/path/message或阻塞分类。

原 Task/Workspace editor/revision/store resolution/freshness/repair 的 fixture 不成为当前规则 authority；它们只随无消费者编排退休。当前 validator、publishing 的类目/属性/图片/SKU/来源/价格等规则及上述有效断言仍运行，不从旧 shell 复制一套新规则。

## 6. 实施与验证

设计阶段只做文档适用检查、实际文件/import/调用/命令依赖核对与原设计 CI；不执行 constructor，不写正式生产/测试代码，不声称已源码退休。

用户确认后的同一批实施：

1. 开工同步一次最新 main，复核 §3/附录 A 和 #821 等当前增量；无同范围 Writer 后在 Issue 登记 Independent Architecture / IMPLEMENTATION_READY / Ready。保留当前已批准架构，不重开无关全局设计。
2. 先在原 retirement/import guard 加入准确完整闭包与 mixed 声明，得到仍有旧文件/声明/路由的目标 RED；保留断言保持。再同一 PR 整退 §5，不留 stub/alias/fallback 或第二事实源。
3. 运行原 listingadmin（含保留 owner/tenant/Store/SensitiveWord/Topic）、sheinlogin/bootstrap/managed/worker、当前 pricing/属性/marketing、marketplace workspace/validator/preview 与 sourceproduct 相关必要回归；验证四张 retained schema 的创建/约束/原有数据保留只使用既有隔离测试，不触真实数据库。复用当前各 owner 的有效规则断言，不补造专用验收工具。
4. 31命令真实依赖中四个整退包/旧 hook 不得重新出现；对 model/listingadmin/listingruntime 以准确声明/文件识别，不禁整包。current-application、shein-login-worker、product-listing-api-schema-migrate 原构建和原 schema/retirement 检查保持；新文件/跨 OS/build tag 仍受原 AST/导入守卫约束。
5. 稳定实现候选才执行完整原 PR CI 与一次集中独立最终检查；修复先分类，后续只复核实际增量。设计 CI、PR CI、准确合并 SHA 的 main CI、运行/provider/用户与生产验收分别记录。实现交 In Review，不自行签发产品验收或关闭父 Issue。

### 6.1 原测试、清单和文档的精确处置

| 原位置 | 实施处置 | 不能改变 |
| --- | --- | --- |
| 69个整退 fixture / §5.2 mixed tests | 只退退出模型、旧 wrapper及无消费者编辑/修复/历史、Task/status/retry/RPC/批量持久化形状；当前Store/SensitiveWord/Topic、身份/owner/tenant/事务/约束错误及§5.4 validator有效测试保留 | 不把旧安全测试“全保留”变成复活仓库，也不因旧repo/投影退休删仍有效owner的检查 |
| tests/listingkit_backend_retirement_test.go | absence/import原guard增加三个整退包（taskrpcapi、workspace/shein、core/metrics）、app/worker根文件及附录其余文件/声明/路由；mixed marketplace只禁止退出声明/文件、保持§5.4真实consumer闭包；原retained-token/file表仅移本批退出项 | 不禁 app/worker 子树、model 全包、listingadmin/listingruntime 全包、marketplace workspace，不skip或留空stub；StoreAPI/gorm_store/schema其余、四值合同/平台/worker及既有absence保持 |
| tests/import_boundaries_test.go 原 Task RPC 三检查 | TestStateAndTaskRPCUseManagementAPIPortAliases、TestTaskRPCAPIRetiredManagementImportsStayBlocked、TestTaskRPCAPIRetiredActionRoutesStayRetired 从“读取旧包证明有限用法”改为“准确旧包/模块/三条路由缺席及 import 禁入” | 读取/parse 错误仍 fail；扫描改用保留祖先的原 loader，并精确匹配子树，不全局忽略 ENOENT；原 app route-helper/old action/management/平台禁令不放宽 |
| 原 owner-scoped/Store/worker/marketing/schema/current command guard | 使用保留 owner 的原 fixture 与断言；原 retained 文件表只移 import_task_owner 两文件、product-data/mapping repo 等本批项 | 两套 workflow 原 `go test -race ./internal/listingadmin -count=1` 与日志 artifact 均保留；不能以本轮少了旧 fixture 删除 race step |
| phase2-runtime-inventory.md | **仅删除一条 CURRENT consumer 行：core/logger → core/metrics**；该包最后生产边本轮退出。其它 current 行与全部历史 debt/阶段计数/dated表保持；顶部可加窄 superseded 注记 | 保留 OpenAI→listingadmin（access_scope真实消费）、logger其它消费者；不改ceilings、.golangci或检查器，不继承 #817/#822 的其它清单删除权限 |
| 必要文档 | 本文/索引、Legacy Register、module-target-mapping/code-health 的直接相关 current 行；boundary checkpoint、external-client inventory、project-boundaries、历史平台/任务设计对 §5 的冲突保留语句加准确 superseded 注记 | 原历史正文/数字/证据保留，不批量清理文档，不将“设计”写成源码/用户/生产已通过 |
| 所有 workflow/依赖/保护/发布检查 | **零改动**，包括 ci.yml/commercial-readiness.yml、触发/并发/权限/mandatory、安全/PG/race、构建目标与pinned引用 | 不以退休闭包为由精简CI，不引入新试用前置或治理/验证平台 |

## 7. 确认、准入与停止条件

1. 当前只交付设计。用户确认 §5、§6、§6.1、§7，且适用独立设计检查没有当前 Must BLOCKER 后，才将当前 Issue 登记/读回 IMPLEMENTATION_READY / Ready 并修改正式源码；设计不默认授予实施、合并或真实环境操作权。
2. 确认后由原唯一 Writer 在同一分支/主要 PR 整批实现、必要自检、finding/CI 修复、提交推送，不机械拆下一轮 helper/DTO/fixture 小批。真正独立 main Blocker 或独立用户结果才另行处置。
3. 若出现新当前消费者、具体外部可观察契约、durable business asset 处置、当前 Must BLOCKER，或必须改变身份/tenant/权限/事实/事务/状态/恢复/平台或 AI 副作用边界，停止对应部分并报告；静态证据不授权运行调查或数据清理，不自行加兼容路径。
4. Finding 先按 AGENTS 分类并说明当前 Must、用户/安全后果与阻塞层级；正常设计检查最多两轮，非 Blocker 在实现测试/Backlog 收敛，不无限追加设计版本。未变化代码与证据复用，CI/AI review 不代替用户验收。
5. 合并另需最终准确 HEAD 的 expected-head 授权；部署/关单/实例或真实数据/Profile/凭据/Store/平台/模型/Prompt 操作均不在本轮授权内。#29/#301 保持 OPEN；runtime/provider/用户/生产验收 NOT_RUN，不增加内部试用前置。

## 附录 A：完整 tracked 文件清单

以下200文件是同一退休集合；不按 glob 外推其它文件，实际新消费者应触发 §7，而不是静默扩大删除。

```text
internal/app/worker/job_handler.go
internal/core/metrics/shein_metrics.go
internal/core/metrics/task_metrics.go
internal/listingadmin/category.go
internal/listingadmin/category_model.go
internal/listingadmin/category_model_test.go
internal/listingadmin/category_query.go
internal/listingadmin/category_query_test.go
internal/listingadmin/category_repository.go
internal/listingadmin/dto_value_helpers.go
internal/listingadmin/filter_rule.go
internal/listingadmin/filter_rule_model.go
internal/listingadmin/filter_rule_model_test.go
internal/listingadmin/filter_rule_query.go
internal/listingadmin/filter_rule_query_test.go
internal/listingadmin/filter_rule_repository.go
internal/listingadmin/gorm_filter_rule_api.go
internal/listingadmin/gorm_operation_strategy_api.go
internal/listingadmin/gorm_product_data_api.go
internal/listingadmin/gorm_product_data_api_test.go
internal/listingadmin/gorm_product_import_mapping_api.go
internal/listingadmin/gorm_product_import_mapping_api_test.go
internal/listingadmin/gorm_raw_json_data_api.go
internal/listingadmin/import_task_owner.go
internal/listingadmin/import_task_owner_test.go
internal/listingadmin/numeric_helpers.go
internal/listingadmin/operation_strategy.go
internal/listingadmin/operation_strategy_model.go
internal/listingadmin/operation_strategy_model_test.go
internal/listingadmin/operation_strategy_query.go
internal/listingadmin/operation_strategy_query_test.go
internal/listingadmin/operation_strategy_repository.go
internal/listingadmin/owner_scoped_batch_test.go
internal/listingadmin/owner_test_helpers_test.go
internal/listingadmin/pricing_rule.go
internal/listingadmin/pricing_rule_model.go
internal/listingadmin/pricing_rule_model_test.go
internal/listingadmin/pricing_rule_query.go
internal/listingadmin/pricing_rule_query_test.go
internal/listingadmin/pricing_rule_repository.go
internal/listingadmin/product_data.go
internal/listingadmin/product_data_model.go
internal/listingadmin/product_data_model_test.go
internal/listingadmin/product_data_query.go
internal/listingadmin/product_data_query_test.go
internal/listingadmin/product_data_repository.go
internal/listingadmin/product_import_mapping.go
internal/listingadmin/product_import_mapping_model.go
internal/listingadmin/product_import_mapping_model_test.go
internal/listingadmin/product_import_mapping_query.go
internal/listingadmin/product_import_mapping_query_test.go
internal/listingadmin/product_import_mapping_repository.go
internal/listingadmin/profit_rule.go
internal/listingadmin/profit_rule_model.go
internal/listingadmin/profit_rule_model_test.go
internal/listingadmin/profit_rule_query.go
internal/listingadmin/profit_rule_query_test.go
internal/listingadmin/profit_rule_repository.go
internal/listingadmin/raw_json_data.go
internal/listingadmin/raw_json_data_query_test.go
internal/listingadmin/raw_json_data_repository.go
internal/listingadmin/rule_status.go
internal/marketplace/shein/workspace/category_search.go
internal/marketplace/shein/workspace/category_search_test.go
internal/marketplace/shein/workspace/display_title_policy.go
internal/marketplace/shein/workspace/display_title_policy_test.go
internal/marketplace/shein/workspace/editor_context_builder.go
internal/marketplace/shein/workspace/editor_context_builder_test.go
internal/marketplace/shein/workspace/editor_context_model.go
internal/marketplace/shein/workspace/editor_dirty_hints.go
internal/marketplace/shein/workspace/editor_effects.go
internal/marketplace/shein/workspace/editor_effects_test.go
internal/marketplace/shein/workspace/editor_model.go
internal/marketplace/shein/workspace/editor_progress.go
internal/marketplace/shein/workspace/editor_progress_test.go
internal/marketplace/shein/workspace/editor_recommendations.go
internal/marketplace/shein/workspace/editor_recommendations_test.go
internal/marketplace/shein/workspace/editor_revision_from_context.go
internal/marketplace/shein/workspace/editor_revision_from_context_test.go
internal/marketplace/shein/workspace/editor_revision_minimal.go
internal/marketplace/shein/workspace/editor_revision_minimal_test.go
internal/marketplace/shein/workspace/editor_revision_model.go
internal/marketplace/shein/workspace/editor_revision_skeleton.go
internal/marketplace/shein/workspace/history_detail.go
internal/marketplace/shein/workspace/history_detail_test.go
internal/marketplace/shein/workspace/inspection.go
internal/marketplace/shein/workspace/inspection_test.go
internal/marketplace/shein/workspace/overview.go
internal/marketplace/shein/workspace/overview_test.go
internal/marketplace/shein/workspace/preview_card.go
internal/marketplace/shein/workspace/preview_card_test.go
internal/marketplace/shein/workspace/preview_projection.go
internal/marketplace/shein/workspace/preview_projection_test.go
internal/marketplace/shein/workspace/preview_support.go
internal/marketplace/shein/workspace/preview_support_test.go
internal/marketplace/shein/workspace/readiness_guidance.go
internal/marketplace/shein/workspace/readiness_guidance_test.go
internal/marketplace/shein/workspace/repair.go
internal/marketplace/shein/workspace/repair_center_from_readiness.go
internal/marketplace/shein/workspace/repair_center_from_readiness_test.go
internal/marketplace/shein/workspace/repair_revision.go
internal/marketplace/shein/workspace/repair_revision_test.go
internal/marketplace/shein/workspace/repair_test.go
internal/marketplace/shein/workspace/review_state.go
internal/marketplace/shein/workspace/review_state_test.go
internal/marketplace/shein/workspace/revision_apply_changes.go
internal/marketplace/shein/workspace/revision_apply_patch.go
internal/marketplace/shein/workspace/revision_apply_patch_test.go
internal/marketplace/shein/workspace/revision_diff.go
internal/marketplace/shein/workspace/revision_diff_model.go
internal/marketplace/shein/workspace/revision_diff_test.go
internal/marketplace/shein/workspace/revision_field_validation.go
internal/marketplace/shein/workspace/revision_field_validation_test.go
internal/marketplace/shein/workspace/revision_history_compare.go
internal/marketplace/shein/workspace/revision_history_compare_test.go
internal/marketplace/shein/workspace/revision_history_detail_builder.go
internal/marketplace/shein/workspace/revision_history_restore_detail.go
internal/marketplace/shein/workspace/revision_history_restore_detail_test.go
internal/marketplace/shein/workspace/revision_restore_draft.go
internal/marketplace/shein/workspace/revision_restore_draft_test.go
internal/marketplace/shein/workspace/revision_restore_presentation.go
internal/marketplace/shein/workspace/revision_restore_preview.go
internal/marketplace/shein/workspace/revision_restore_preview_builder.go
internal/marketplace/shein/workspace/revision_restore_preview_test.go
internal/marketplace/shein/workspace/revision_restore_request.go
internal/marketplace/shein/workspace/revision_success.go
internal/marketplace/shein/workspace/revision_success_model.go
internal/marketplace/shein/workspace/revision_success_test.go
internal/marketplace/shein/workspace/revision_validation.go
internal/marketplace/shein/workspace/revision_validation_test.go
internal/marketplace/shein/workspace/source_product.go
internal/marketplace/shein/workspace/source_product_test.go
internal/marketplace/shein/workspace/store_resolution.go
internal/marketplace/shein/workspace/store_resolution_test.go
internal/marketplace/shein/workspace/submit_freshness.go
internal/marketplace/shein/workspace/submit_freshness_readiness.go
internal/marketplace/shein/workspace/submit_freshness_readiness_test.go
internal/marketplace/shein/workspace/submit_freshness_test.go
internal/marketplace/shein/workspace/task_queue.go
internal/marketplace/shein/workspace/task_queue_test.go
internal/marketplace/shein/workspace/task_taxonomy.go
internal/marketplace/shein/workspace/task_taxonomy_test.go
internal/model/task.go
internal/model/task_retry.go
internal/model/task_retry_test.go
internal/model/task_status.go
internal/model/task_status_test.go
internal/model/task_test.go
internal/shein/sherr/handled_errors.go
internal/taskrpcapi/build.go
internal/taskrpcapi/build_test.go
internal/taskrpcapi/handler.go
internal/taskrpcapi/http_module.go
internal/taskrpcapi/http_module_test.go
internal/taskrpcapi/types.go
internal/workspace/shein/compatibility_guard_test.go
internal/workspace/shein/doc.go
internal/workspace/shein/editor_context_builder.go
internal/workspace/shein/editor_context_builder_test.go
internal/workspace/shein/editor_context_model.go
internal/workspace/shein/editor_dirty_hints.go
internal/workspace/shein/editor_effects.go
internal/workspace/shein/editor_model.go
internal/workspace/shein/editor_progress.go
internal/workspace/shein/editor_recommendations.go
internal/workspace/shein/editor_revision_from_context.go
internal/workspace/shein/editor_revision_minimal.go
internal/workspace/shein/editor_revision_minimal_test.go
internal/workspace/shein/editor_revision_model.go
internal/workspace/shein/editor_revision_skeleton.go
internal/workspace/shein/history_detail.go
internal/workspace/shein/inspection.go
internal/workspace/shein/inspection_payloads.go
internal/workspace/shein/inspection_test.go
internal/workspace/shein/overview.go
internal/workspace/shein/overview_test.go
internal/workspace/shein/readiness.go
internal/workspace/shein/readiness_guidance.go
internal/workspace/shein/readiness_guidance_test.go
internal/workspace/shein/readiness_test.go
internal/workspace/shein/repair.go
internal/workspace/shein/repair_test.go
internal/workspace/shein/revision_apply_changes.go
internal/workspace/shein/revision_diff.go
internal/workspace/shein/revision_diff_model.go
internal/workspace/shein/revision_field_validation.go
internal/workspace/shein/revision_field_validation_test.go
internal/workspace/shein/revision_history_compare.go
internal/workspace/shein/revision_history_detail_builder.go
internal/workspace/shein/revision_history_restore_detail.go
internal/workspace/shein/revision_restore_draft.go
internal/workspace/shein/revision_restore_presentation.go
internal/workspace/shein/revision_restore_preview.go
internal/workspace/shein/revision_restore_preview_builder.go
internal/workspace/shein/revision_restore_request.go
internal/workspace/shein/revision_success.go
internal/workspace/shein/revision_success_model.go
internal/workspace/shein/revision_success_test.go
internal/workspace/shein/revision_validation.go
internal/workspace/shein/state.go
```
