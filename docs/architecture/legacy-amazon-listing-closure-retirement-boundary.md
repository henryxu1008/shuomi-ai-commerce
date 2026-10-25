> **#948 窄替代注记（已获用户实施确认，源码已实施 / In Review）**
> #948 仅替代本批矩阵内闲置来源辅助：单数 FeatureExtractor、三个 text-validator 方法、变体便利合同/custom/debug 方法、旧错误构造/诊断及提示/邮编便利尾部；当前已注册提取器、有效规则、变体 JSON/转换及浏览器采集不变。
> 唯一准确范围见 [#948 冻结设计](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md) §1–§7（含 §6.1）；以下原全文为历史证据，原数字、勾选及其它 KEEP 不变。

> #301 / #947 **SUPERSEDED，仅无消费者来源便利合同**：用户已确认[完整设计§1–§7/§6.1](legacy-source-url-timeout-closure-retirement-boundary.md)。本文crawler/amazon来源广泛KEEP，不再保活准确矩阵内闲置URL/地区便利、TimeoutManager及专属接线；Amazon-as-source和当前model/有效规则/原浏览器debug保持，不按Task/metrics/worker同名整退。实际URL/ASIN/币种/marketplace/地区邮编、真实PoolManager deadline/取消、当前采集及独立SHEIN登录/Store同步保留。workflow/完整inventory及既有资源不动，无试用前置，#301 OPEN。以下原完整Git blob/历史/数字/勾选逐字保留。

> #301 / #946 **SUPERSEDED，仅无消费者的浏览器管理/统计准确矩阵**：用户已确认[完整退休设计§1–§7/§6.1](legacy-browser-management-tail-closure-retirement-boundary.md)。本文shared metrics不因同名删除的原则继续有效；本次仅按实际消费者退出质量快照整链、代理Snapshot/外层池投影及§5准确闲置声明/字段/测试尾部，不整体退休Task/metrics/worker。真实指纹/代理全部评分计数与时间戳/质量重试/周期健康日志、当前采集/原Amazon浏览器debug、独立SHEIN登录/Store同步和其它KEEP保持。无新生产EXTRACT/owner/兼容，workflow/完整inventory/既有资源不动；以下原完整Git blob/历史/数字/勾选逐字保留。

> #301 / #926 **SUPERSEDED，仅本批旧平台调用闭包**：用户已确认[完整退休设计 §1–§7/§6.1](legacy-platform-api-closure-retirement-boundary.md)。本文涉及旧 Amazon/TEMU API、SHEIN 闲置 Client/manager/DTO、SPAPI/旧平台包装配置的 KEEP/当前 owner 建议，只在该准确删除/保留矩阵内被替代；不移植旧 auth refresh/retry 或平台副作用。当前 SHEIN DTO/完整 JSON/有效规则、SessionTransport/三个读取端点、采集/原 Amazon 浏览器/debug、官方 Store、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留。只抽原仓库两纯函数，不加 compat；workflow/完整 inventory/计数/检查逻辑及既有实例/数据/凭据/进程不动，不增加试用前置。以下原完整正文、历史数字/勾选/批准/证据逐字保留。

> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。仅替代旧 shared/aiidentity ExecutionEnvelope/PersistedExecutionEnvelope、capture/restore/context/分类/来源白名单、Task tenant 辅助和两专属测试的保留/实施含义；五工件闭包已在同一 PR 源码退休。当前 Identity/WithIdentity/FromContext/request roles、authidentity 与现役 ImageAgent/Commercetool 授权/隔离/恢复完整保留；不复建旧无 scope 放行或旧信封，不执行历史 plan 的派工/迁移/运行步骤。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

> #301 / #915 **USER_CONFIRMED / SOURCE_IMPLEMENTED**：按[旧宿主、HTTP 与图像辅助完整退休闭包 §1–§7/§6.1](legacy-host-image-helper-closure-retirement-boundary.md)的用户确认矩阵，仅替代 imagex/httpclient 广泛 KEEP 中的准确孤立声明：旧编码/格式辅助和 DefaultConfig/NewWithTransport；保留当前解码/metadata/注册、HTTP/TLS/retry、Amazon API/原浏览器/debug 与共享合同。 仅该准确矩阵取代本文件对应 KEEP，其余边界及以下全部历史正文保留；workflow/inventory 与既有资源不动，独立 SHEIN 登录及 Store 同步保持，#301 OPEN。

> 当前适用边界（#301 / #912）：用户已确认并按[旧 SHEIN 闲置策略与辅助合同完整退休闭包](legacy-shein-policy-contract-closure-retirement-boundary.md) §1–§7（含 §6.1）整批实施，仅替代本文件中两种 SHEIN 属性/最后 AmazonMonitorData 及旧价格辅助的广泛 KEEP；当前 Amazon 来源采集与原浏览器/debug 不变及对应旧 KEEP 检查；准确删除/保留矩阵以新设计 §2/§5/§6.1 为准，其它当前 API/有效规则/采集/独立 SHEIN 登录与 Store 同步不变。下方原历史正文、数字和证据逐字保留，不提供旧 JSON 迁移或运行/数据操作授权。

# 旧 Amazon 刊登整链与专属闭包退休边界

> 用户已确认 [#909 productpolicy 完整退休边界](legacy-product-policy-closure-retirement-boundary.md) §1–§7（含 §6.1）。本文件对 `internal/marketplace/productpolicy` 六工件（doc/filter_rule/rule_checker/price_helper/fulfillment 及专属测试）和四项 KEEP 检查的保留含义已由该边界精确替代为 RETIRE；其余有效规则/API/采集、共享合同及独立 SHEIN 登录与 Store 同步不变。以下原正文、历史数字与验收证据保留，不恢复旧策略或授权既有实例/数据/凭据/进程操作。

> 当前校准：#867 用户确认的[数字采集与平台尾链完整闭包](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)仅替代本文对旧 fetch/cache/数字 Task HTTP、SHEIN 数字 Cookie/StoreInfo 及专属 Amazon 运维源的广泛保留表述；本文其余平台 API/规则、两种 SHEIN 属性、价格、当前采集/共享浏览器、Store 与独立 SHEIN 登录/同步的保留决定继续有效。历史执行证据不改，不授权运行资源操作。

> #824 后续按用户确认的[共享支撑边界](legacy-numeric-listing-support-closure-retirement-boundary.md)退休八类闲置 listingadmin 管理链、孤立 listingruntime 声明与旧 Task/Workspace 尾部。本文对这些准确对象的保留仅为历史；Store/敏感词/话题、四种有效值合同、Amazon API/来源采集、有效共享能力及独立 SHEIN worker/Store 同步仍保留，不操作运行态或数据。

> Execution：[#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)
>
> Design Basis：Independent Architecture / IMPLEMENTATION_READY；状态：SOURCE IMPLEMENTED / In Review。用户已确认冻结设计 `3a344153edefeaaab61433a686482d92d0d4310f` §5/§6/§6.1/§7；适用独立设计检查及唯一 IMPLEMENTATION_TEST 分类、Ready-before-production 记录见[#822 准入](https://github.com/qq550723504/task-processor/pull/822#issuecomment-6259806207)。118Go/20非Go与精确 mixed/schema/守卫已源码实施；准确实现HEAD、原CI与最终独立检查写同一PR，不用设计证据代替。
>
> 调查基线：main `e2f4a2f8b20ffce7530e6962038b9b04a0e76348`，2026-10-19。#819 按用户授权的 `ca9ca2cfc` 合入此 SHA，对应 push/main CI [39197893597](https://github.com/qq550723504/task-processor/actions/runs/39197893597) 成功；该授权和证据不覆盖本批。

## 1. 用户结果与完整 Delivery Batch

使用者是当前采集、商品/图像/草稿、Store 观察同步及独立 SHEIN 登录 worker 的内部试用操作者。维护结果是：没有当前入口的旧 Amazon 数字刊登管线与另一套 Task-first Workbench，连同最后消费者专属状态包、队列装配、DTO、调试入口、失效部署声明，在一个主要 PR 完整退出。当前操作继续使用原入口和事实 owner；不增加内部试用前置，不声称 Amazon 发布已可用。

闭包包括 §5 的118个 tracked Go 文件（91生产/27测试，生产11805行）、20个非 Go 文件、精确 mixed 声明、schema 初始化注册、守卫及直接冲突文档。按真实消费者与当前合同截止，不按目录、文件数、实现提交或检查次数机械拆 PR。切片只是同一批内的实现/回滚边界。

非目标：启用或修好旧 Amazon 刊登、创建新的 Amazon Agent/规则库/Workbench、包装旧 Service、保留旧队列或第二事实源、补建通用迁移/调度/恢复/验收系统、操作运行实例/数据库/Redis/Profile/凭据/平台/模型、部署或关闭 #29/#301。没有新依赖、新 provider 调用或新风险接受决定。

## 2. Product Authority、Design Basis 与不变量

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)：仅 EXTRACT / RETIRE；旧 Task-first 平台 Workbench 不能作为内部引擎继续保留。
- [最终 Product/UI Authority §1.1](../product/final-ui-ia-authority.md)：平台是当前共享能力的维度，不复制 Amazon 产品孤岛。本批不新增、修改或验收 UI/IA/交互，Figma 直接操作 N/A；不以旧页面/API 文档推定当前产品需求。
- [Product sourcing](../product/product-sourcing-handoff.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[观察接线](store-observations-preparation-runtime.md)、[独立 SHEIN worker](shein-worker-identity-persistence-boundary.md)：当前事实、平台副作用和登录/Store 同步 owner 保持。
- [旧数字执行链边界](legacy-numeric-task-execution-retirement-boundary.md)、[#819 TEMU 闭包](legacy-temu-listing-closure-retirement-boundary.md) 曾保留 Amazon 子包、`internal/state` 及通用时间 helper。当时有旧消费者；本设计**仅提议窄替代 §5 明列旧源码的保留决定**，不把历史事实改写成错误，也不推定已经获准删除。
- [Amazon Workbench API 文档](../amazon-listing-workbench-api.md) 仍标 `CURRENT CONTRACT`，与当前未注册的路由、全新系统及 Task-first hard-cut 冲突。撤销该旧内部合同是**本次需用户确认的明确设计决定**；不能只凭静态不可达就删除自称当前的合同。保留其历史正文，实施时改为 SUPERSEDED / RETIRED，不增加兼容路由。

未来实施属于 Independent Architecture：跨旧管线、Task/持久化注册和专属共享层，超过30文件/2500生产行，并退出一个旧公开形状。它不建立新公共合同、事实 owner、表、状态机、事务、授权/租户边界、幂等/retry/UNKNOWN/recovery 或 AI 预算/计量/平台发送 authority；只撤销列明旧内部合同与新安装的旧表注册，不迁移已有数据。

Must：当前采集（含 Amazon source）、图像、Product/ApprovedAsset/当前草稿与 Submission、Store 观察、有效平台 API/DTO/规则、共享 Reader/Cache/Mapping、两种 SHEIN 属性、SHEIN PriceManager/PriceCalculator、独立 SHEIN 登录 worker 及 Store 同步全部保留。两套及所有 workflow、依赖/保护规则、必需/安全检查、`.golangci.yml` 和 ceilings 零改。

Threat Model 限于误删当前消费者/合同、把旧默认值或伪成功迁入当前 owner、误禁同名 API/共享类型、误删登录状态、把源码退休变成真实数据/部署动作、弱化守卫。没有新增 Accepted Risk，不追加试用前置或全局安全工程。

## 3. 实际消费者、调用路径与根因

在调查 SHA 枚举4276个 tracked Go 文件的 import 声明（含测试及不同 build tag），检查构造器/调用、非 Go 配置、HTTP 路径、部署和默认资源。31个 `cmd/*` 的 `go list -deps` 并集成功；只读取元数据，未运行 main、业务构造器、测试或 provider。旧 Amazon 管线、state/httpbootstrap/timex 无命令依赖；`amazonlisting` 及 marketplace/amazon 的四个包仅经 schema 命令可达。`internal/amazon/api` 也经此链编译，仍按明确 API 保留边界保留。

| contract → implementation → injection → consumer | 当前证据与处置 |
| --- | --- |
| 已退 root Processor → amazon/model.Services → HandlerManager → pipeline/listing/schema/image/llm/attribute | 七个非 API Amazon 包无闭包外 Go importer，无当前命令或 runtime 构造；旧管线整退。Listing/Pricing/Inventory handler 写 mock ACTIVE，默认价19.99/库存100，不是当前平台事实 |
| GenerateRequest/Task → amazonlisting.Service/Processor/Repository → httpapi builder / workerpool / routes | httpapi 没有包外生产 importer，路由从未进入当前 assembly；store 只有自身和闭包测试消费者。旧6条 generate/tasks/workbench/review/submit 路由、自动修补与重试状态整退 |
| product-listing-api-schema-migrate → app/schema/productlisting → AutoMigrateRuntime(amazonlisting.Task) | 唯一闭包外生产 import，纯注册旧 `amazon_listing_tasks`。精确移除此 import/注册，保留 schema 命令、其他当前表、Goose runner/version/已应用记录；不执行命令或 DROP |
| amazonlisting → marketplace/amazon/model/publishing/workspace | 三包只供旧 Workbench 使用；api/doc.go 是无实现、无 importer 的旧落位说明。整个 marketplace/amazon 专属子树退出；真正平台 client `internal/amazon/api` 保留 |
| amazon/model.Services → state.MemoryManager → Cookie/ShopPause/DailyCount/ReListingQueue | state 唯一包外生产 importer 是本批 amazon/model，测试外引用也仅旧 pipeline。无 Redis 存储实现；cookie/队列/暂停是进程内 map，配额/暂停可能调用旧管理端口。整退代码，不碰任何实例状态 |
| amazonlisting/httpapi → httpbootstrap.PoolSubmitter/NewWorkerPool | httpbootstrap 唯一 importer 为本闭包；删专属15分钟池装配，平台 workerpool 与当前调用者保持 |
| amazon/pipeline/daily_limit → shared/timex | timex 唯一 importer 为本闭包，只有标准库 time 的格式包装，无当前消费者；整退，不新建替代格式层 |
| state → listingadmin.DailyListingCountAPI 与6 DTO | 闭包外只剩 management_api_types.go 自身声明；精确退休7声明，其余 Store/规则/模型/共享 DTO 与 owner 保留 |
| 旧 HTTP 调试命令与 K8s amazon-listing 声明 | debug 脚本仅调用旧 HTTP；K8s 引用已退 cmd/amazon-listing 和 /app/listing-service。无两套 workflow 或其他 kustomization 导入该部署目录；源码同批退，不操作集群/Secret/共享 namespace |
| 当前 SHEIN worker → app/runtime/sheinloginworker → sheinlogin/bootstrap、listingadmin、平台 DB | Go 图不依赖 internal/state；自身 Redis/Profile/登录 attempt/StoreID 与 cookie owner 独立。命令、原检查及 Store 同步能力保持，不能把同名 Cookie/State 一并删除 |

架构根因是历史“内部仍被引用”与文档 `CURRENT CONTRACT` 标签持续替代当前产品依据：失去入口的两套刊登 owner 仍互相托住类型与 schema 注册，形成看似共享的依赖。旧 mock 成功、推定品牌/价格/库存、进程内额度/暂停和无发送许可的直接 SP-API wrapper 都不应进入当前平台事实或授权执行链。本批解决方式是撤销旧 owner 与专属闭包，不补接旧路径。

静态源码不能证明外部部署/历史行/队列不存在；本批不获取其状态，也不据此签发资源删除或产品验收。当前并行 #605 / PR #821 的 SHEIN 上架设计与 PR #692 Store schema 退休由各自 owner 负责，本批不改其代码或设计。

## 4. 持久化、身份与平台副作用 owner

| 边界 | owner / 效果 | 允许与禁止 |
| --- | --- | --- |
| amazon_listing_tasks | 旧 amazonlisting.Task/store；保存执行身份、请求/草稿/结果、审核/提交记录、状态/重试与来源版本 | 退模型/仓库/新安装注册；不读/改/删旧行，不建映射/接管，不转存 BusinessTask。现有表若存在仍留原处，无降级/清理 SQL |
| Product/ApprovedAsset、AI credential/invocation ledger、image agent、Prompt 等当前表 | 当前 Product/Asset/AI/Image/Prompt owner；productlisting schema manifest 中其余分支 | 原注册、版本/权限/事务和数据保持，不新增 Goose 版本或修改 migrations.go/已应用历史。原 baseline 回调调用 current AutoMigrateRuntime 是已有耦合；本次只让新安装不再制造已退表，不建设迁移框架 |
| Cookie/暂停/重刊队列/每日额度 | 旧 state 的进程内 map，DailyCountAPI/StoreClient 可调用管理端；不是独立 SHEIN 登录持久化 owner | 仅删源码；不触发清理、补偿、恢复、配额回滚/重置或 Store 状态修改；StoreAPI/DTO/持久模型保持 |
| SP-API 创建/更新/校验、商品类型/schema 与图片上传 | 旧 publishing.SPAPISubmitter、schema fetcher/image wrapper 可发请求；平台底层 internal/amazon/api owns auth/sign/retry/rate limit 与请求/响应合同 | 退专属编排，不构造 client、不调用平台；API全包保留。当前 Submission 的许可/receipt/UNKNOWN/恢复与未来 Amazon admission 不变 |
| 旧 LLM mapper | amazon/llm 内建 prompt，直接 OpenAI CreateChatCompletion 路径 | 退调用/内建字符串；没有专属 tracked prompt YAML/Registry key 可删。通用 OpenAI、GovernedModel/AI预算/Prompt Store 与覆盖记录零改，无模型请求 |
| K8s 15文件、共享 namespace / Secret /外部 crawler | Git声明与集群资源不是同一事实；namespace名task-processor共享，remote crawler也可能有当前消费者 | 仅删除专属 tracked 声明；不 apply/delete/prune，不查/删 Secret/外部资源。amazon-crawler-api/external-lb 的资源、当前采集配置保留，只校正 README 中已退 listing 消费说明 |
| 独立 SHEIN 登录/Store 同步、当前 Store 观察/发布 | 各自 Profile/Redis/attempt、Store 与平台 owner | 所有身份、tenant、cookie、查重、StoreID同步、观察与发送边界保持；不访问实例或数据 |

保留能力在失败、重试、响应丢失、取消、重启和并发时仍由原 owner 决定结果；源码删除不触发任何操作。不存在新增跨库事务/恢复协议或需要 outbox/Saga 的业务写。若发现具体当前外部可观察兼容义务，先报告，不能自行加兼容或据此操作环境。

## 5. 删除、保留与抽取矩阵

### 5.1 完整 Go 删除边界

下表针对调查 SHA 的 tracked 全部 Go 文件（包含 doc.go、测试及 build-tag 文件），目录边界精确匹配；不能误删同名根。确认后同一主要 PR 整体退休，新增真实消费者须先反馈。

| 目录 | Go总数 / 生产 / 测试 | 决定 |
| --- | --- | --- |
| internal/amazon/attribute | 2 / 2 / 0 | RETIRE：旧配置映射与校验，无当前规则消费者 |
| internal/amazon/image | 2 / 2 / 0 | RETIRE：旧图片处理/上传编排，底层 imagex/downloader/API不动 |
| internal/amazon/listing | 10 / 10 / 0 | RETIRE：旧属性/默认值/变体/identifier/类目推荐 builder |
| internal/amazon/llm | 3 / 2 / 1 | RETIRE：旧属性 AI mapper/client/fixture |
| internal/amazon/model | 8 / 8 / 0 | RETIRE：旧 Services/TaskContext/ProductData/schema/错误类型；不是共享 internal/model |
| internal/amazon/pipeline | 21 / 16 / 5 | RETIRE：旧 handler/manager/status reason/fixture |
| internal/amazon/schema | 4 / 4 / 0 | RETIRE：旧 fetch/cache/parser/builder；API ProductTypeDefinition 合同保留 |
| internal/amazonlisting | 48 / 29 / 19 | RETIRE：含 api/httpapi/store，Task-first service/model/queue/workbench/review/submit/仓库与fixture整退 |
| internal/marketplace/amazon | 10 / 10 / 0 | RETIRE：api占位、model/publishing/workspace专属闭包；不是 internal/amazon/api |
| internal/state | 7 / 5 / 2 | RETIRE：Memory/Cookie/ShopPause/DailyCount/ReListingQueue及fixture；不是登录 worker state |
| internal/httpbootstrap | 1 / 1 / 0 | RETIRE：旧池 submitter/装配；平台 workerpool保持 |
| internal/shared/timex | 1 / 1 / 0 | RETIRE：孤立时间格式包装；其他 shared 包及原 pkg/timex 禁入规则保持 |
| hack/debug/test-amazon-listing | 1 / 1 / 0 | RETIRE：旧 HTTP 生成/轮询客户端，不运行 |
| **合计** | **118 / 91 / 27** | 不保留 stub、兼容 alias 或只供自身测试的第二套 owner |

### 5.2 非 Go、mixed 声明与注册

- 删除 marketplace/amazon 的5份 README（根、api、model、publishing、workspace），不留失效目录合同。
- 删除 deployments/kubernetes/amazon-listing 的15个 tracked 文件：README；base/{configmap,deployment,hpa,kustomization,namespace,secret.example,service}.yaml；overlays/prod/{external-secret,kustomization,patch-deployment,patch-hpa}.yaml；overlays/staging/{kustomization,patch-deployment,patch-hpa}.yaml。仅源码，不涉及集群资源。
- management_api_types.go 仅删除 DailyListingCountGetReqDTO、DailyListingCountSetReqDTO、DailyListingCountRespDTO、TryConsumeDailyQuotaReqDTO、TryConsumeDailyQuotaRespDTO、RollbackDailyQuotaReqDTO、DailyListingCountAPI 七项声明。其余字段/方法/DTO不顺手瘦身，StoreAPI暂停方法也保持。
- app/schema/productlisting/runtime.go 仅移除 amazonlisting import 与 AutoMigrate(&amazonlisting.Task{}) 分支；当前 schema/Goose流程保留。runtime_test.go 把旧表 execution-envelope 建表断言改为新安装不创建旧表，同时保持当前表/ledger/schema/版本/连接检查；只在临时 fixture 验证已有同名表/行不被触碰。
- docs/amazon-listing-workbench-api.md 改历史状态并链接本设计，保留原正文作为历史证据；deployments/kubernetes/readme.md 移除旧部署入口/顺序/命令，amazon-crawler-external-lb/README.md 仅校正旧消费者说明、保留当前 crawler 资源。现有架构/迁移文档中直接冲突的 Amazon/state/timex 保留叙述加窄 superseded 注记，历史数字/原验收证据不重写。

### 5.3 保留/复用/抽取矩阵

| 对象 | 决定与当前 owner |
| --- | --- |
| internal/amazon/api 全部17Go（16生产/1测试）与 config.SPAPI/marketplace 配置/解析 | 保留全部 auth/signing/request/retry/rate limit/catalog/product type/listing/details/pricing/inventory/upload/seller 合同。无当前命令可达不能覆盖用户的 API 保留决定；不执行 postman/真实平台测试 |
| internal/crawler/amazon、integration/crawler/amazon、crawler/fetcher、marketplace/sourceproduct/productpolicy、internal/model | 保留当前来源采集/中立产品与 Reader/Cache/Mapping/筛选规则；Amazon-as-source不等于旧 Amazon-as-target。共享 Task/metrics/worker不因名称相同而删除 |
| internal/listingadmin、internal/listingruntime 其余模型/仓库/端口/规则/StoreAPI | 保留原持久化、tenant/owner、并发、Store ID 同步与有效规则；只退 §5.2 七个纯旧配额声明，不删表/列/模型/数据 |
| Product/Catalog/Asset/Image、共享 pricing/有效规则、AI/Prompt/GovernedModel、Submission与Store观察 | 复用当前 owner，不搬迁旧属性/自动品牌/200字标题/默认库存/启发式 IP 规则。旧规则仅被旧owner自身使用且无当前需求，本批RETIRE，不为“纯函数”建立新规则库 |
| SHEIN 共享属性两种、AmazonMonitorData/productsync enriched 两种、PriceManager/PriceCalculator、独立登录 worker 及Store同步 | 全部保留；本批不动并行 SHEIN 设计或实现，不新增平台/worker门禁 |
| shared/hashx/mathx/ptr/resilience/strx、pkg/imagex/downloader/httpclient/jsonx、core/config/logger、kernel/module、platform/workerpool | 真实消费者仍在，保持原 owner/API/语义；仅 timex/httpbootstrap 达到完整专属闭包 |

Legacy decision: RETIRE。Reusable behavior：有效平台/共享规则及基础层已在上表当前 owner 保留，本批未发现需要从旧 owner 新抽取的当前行为。Current owner：上表现有 owner。Cutover/deletion condition：用户确认、独立设计准入、精确注册/守卫和原CI同步完成后，同一PR整退。若新证据要求 EXTRACT 或扩大 mixed 删除，先说明具体消费者与最小边界，不能静默扩大。

## 6. 实施与验证

确认与 Ready 后，同一 Writer 连续完成：①先为新安装不创建旧 Amazon Task 表写 RED，保留当前表/不触碰已有表的临时 fixture 断言；②精确移除注册与七声明，再整退 §5 全闭包；③同步守卫、10条当前消费者边和直接冲突文档；④必要自检、同一PR提交推送、一次稳定候选原CI与最终独立检查。不把这四步拆成机械小PR。

本地验证只覆盖变化风险：productlisting schema fixture（新安装、已有同名表留存、当前表/Goose版本/连接）、原退休/导入/依赖/消费清单守卫、保留 Amazon API 的离线编译、listingadmin 原 owner/并发及 SHEIN worker 现有编译/Store操作测试、当前来源采集的受影响编译路径。31cmd 的静态依赖从 schema 路径退出所有拟退包；API可继续保留为非注入平台合同。不执行真实平台 postman 测试、旧业务构造器或 schema 命令，不新增runner或运行验收平台，不人工重复完整CI。

准确记录设计SHA、实现SHA、本地结果、PR CI、独立检查；合并另获最终HEAD授权后才记录实际merge与对应push/main CI。设计CI不替代实现CI；源码退休不等于部署、provider、实例/数据或用户验收，后者保持 NOT_RUN。

### 6.1 原检查与消费者清单精确处置

| 原位置 | 最小调整与保持项 |
| --- | --- |
| tests/listingkit_backend_retirement_test.go | 原 absence/import 禁令加入 §5 精确子树/非Go/mixed声明/旧HTTP与部署重建；retained表只移 amazon/pipeline/store_info.go、manager.go。Amazon API、当前采集、TEMU/SHEIN平台、共享规则/DTO、独立worker/Store同步原断言保持；禁止整个amazon根禁入 |
| tests/import_boundaries_test.go | Amazon Services port 检查转为旧Services/context缺席与重引入禁令；State management、AmazonListing HTTP/业务/装配/图像约束从存续internal父root建index后按准确子树过滤。保留原禁入路径/解析错误；只撤本批准确allowlist条目，不用ENOENT忽略、skip或宽目录豁免。Amazon API外部client/管理别名原约束继续执行 |
| tests/target_architecture_phase2_test.go / phase3_product_test.go | RepositoryBuilders无AutoMigrate、AmazonListing不编排图像检查同样从存续父root扫描并精确过滤，新增旧路径重引入保护；SharedPackageTargetsExist仅撤timex目录必需项，并断言该专属根已退。其余共享包目标与ceiling/history零改 |
| app/schema/productlisting/runtime_test.go | 撤旧Task专属execution-envelope列断言，先RED验证旧表不再创建；保留AI/图像/Asset/Catalog/Prompt等原当前注册与Goose/连接检查，临时已有旧表/行fixture不触碰；共享aiidentity及其他有效身份/tenant测试保持 |
| 当前命令/来源采集/独立SHEIN worker依赖检查 | 增加 §5 准确包及带分隔符子包禁入，保留31cmd和当前worker原断言；amazon/model≠internal/model，state≠登录state，marketplace/amazon≠amazon/api；不禁同名外部平台类型 |
| phase2-runtime-inventory.md | 只删下表10条 Legacy consumer register **当前边**；其余当前行、全部dated历史/数字/归类、ceilings不动，不冻结文件后绕过源码一致性检查 |
| Legacy register/code-health-decisions/本文与索引、直接冲突旧设计 | 实施后写准确来源/边界；旧Amazon/current contract/state/timex描述窄superseded。历史Issue/PR/报告/原数字保持，不把设计阶段写成已退休 |
| .golangci.yml、所有workflow（含两套CI）、go.mod/go.sum、保护与必需安全门禁 | **零改**；保留listingadmin -race、安全/PG/release/command检查。若原CI出现超出本矩阵的真实合同冲突，先报告具体证据/最小范围，不自行松门禁 |

准确10条（均加 task-processor/internal/ 前缀）：

| Target | 退出的 consumer |
| --- | --- |
| core/logger | amazon/image |
| core/logger | amazon/listing |
| core/logger | amazon/llm |
| core/logger | amazon/pipeline |
| core/logger | amazon/schema |
| core/logger | state |
| platform/workerpool | amazonlisting |
| platform/workerpool | amazonlisting/httpapi |
| platform/workerpool | httpbootstrap |
| integration/openai | amazon/llm |

不能借此删除 core/logger → amazon/api 等保留边，不能批量重写历史 debt/inventory。旧纯实现fixture可退休，但当前安全、身份、事实/平台合同有效检查必须在原owner继续覆盖，不能用移除测试消除真实缺陷。

## 7. 确认、准入与停止条件

1. 用户已确认冻结设计3a344153e的 §5、§6、§6.1、§7；适用独立设计检查已完成，无BLOCKER，一条当前adapter文档问题按IMPLEMENTATION_TEST以窄来源注记收敛（原表保留，不扩大10条边删除）。#29/#301在生产修改前已登记并读回 Independent Architecture / IMPLEMENTATION_READY / Ready；当前源码已实施。保留真实平台API/来源采集/共享能力，后续限制不变。
2. 确认后同一Writer在同一PR整批实施、修复finding与CI、提交推送。不再拆机械小批；仅发生真正独立main Blocker或独立交付价值时按AGENTS处理，不把工程阈值当拆分目标。
3. 新当前消费者/外部可观察兼容义务、真实Must BLOCKER、必须改变当前身份/tenant/事实/状态/事务/恢复/平台副作用/AI预算边界时停止受影响项报告。静态证据不能推定运行态不存在，也不能以之取得实例/数据调查授权。
4. Finding先按AGENTS分类并指出当前Must、具体后果与阻塞层级；正常设计检查最多两轮，无新BLOCKER不无限追加设计版本，非阻塞项在实现测试/Backlog收敛。不得加wrapper/fallback/双事实源保留旧owner，也不得删有效守卫或保留空stub。
5. 实施完成进入In Review，交准确HEAD、原CI与一次最终独立检查；用户/指定验证者负责实际验收。合并另须最终准确HEAD授权；部署、真实平台/模型/Store/Profile/Secret/Prompt覆盖/数据动作、关单均未授权，不增加内部试用前置。
