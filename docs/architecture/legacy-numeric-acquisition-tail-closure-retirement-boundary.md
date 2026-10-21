# #301 旧数字采集与平台身份尾链完整退休边界

实施准入：**用户已确认 / Design Basis: Reuse Existing Architecture / Ready → In Progress → In Review**。冻结设计为 `d3c33d86cb5edbad9a6958df9c9ba85a1259eb84` 的 §1–§7（含 §6.1）；用户在准确设计交付后回复“没问题，继续”，授权同一 #867 整批实施。下文设计阶段状态保留为准入历史，以本段及 #301 当前正文为当前实施状态。原单 Writer、分支和 PR 不变；合并、部署、关单及真实资源操作仍未授权。

Status: **PROPOSED / DESIGN ONLY / 待用户确认**。调查基线为 main `8cca4a45e5c9dfacef336e61728e239247fe3bca`，即 #863 实际合并提交；其准确 push/main CI 为 [39554937146](https://github.com/qq550723504/task-processor/actions/runs/39554937146)，已成功。本文不把该证据当成本设计未来实施的 CI，也不把源码退休当作运行验收。#301 保持 OPEN，#29 已源级关闭，不重开。

实施结果：§5 专属闭包已同批退出；混合源码只收缩旧数字上下文/配置/StoreInfo，当前规则与平台 transport 保留。§6.1 七条 CURRENT 行准确删除，历史/计数及 workflow/依赖/锁/ceilings零改。保留现有 Cookie 与 qualityMetrics 纯值声明（见下文实施发现），不恢复旧 owner；共享 Secret 源只去旧 remoteAPI 非凭据映射。原退休守卫已先 RED，相关纯规则/API/source/worker与当前构建已自检；准确实现 HEAD、CI及最终独立复核以 #867 为准，不把设计 CI当实现证据。源码交付不等于运行/用户验收，#301仍OPEN。

## 1. 用户结果、范围与权限

使用者是维护当前应用、采集、平台合同、Store 与独立 SHEIN 登录 worker 的维护者。当前问题是：最后的数字账号/Store owner 已在 #863 退出，旧采集工厂、数字任务结果协议、平台数字 Cookie provider 和指向不存在入口的发布源仍相互引用。它们没有当前命令装配，却让仓库继续保留旧账号/Task/cache 的构造合同和操作入口说明。

建议一个 Delivery Batch、一个 Writer、一个主要 PR 整批退出 §5 的生产源码、专属测试、配置、发布/部署/监控声明及直接文档尾部；不按目录、文件数或测试步骤拆 PR。保留的业务结果仍是当前采集、平台 API/规则、Store 观察，以及 #863 当前受权 CLI 支持的独立 SHEIN 登录、验证码/状态/取消、会话与 Store 身份同步。用户不需要为这项源码退休增加任何内部试用操作。

本轮只授权只读源码调查与本文/索引/登记文档。**生产源码、测试、配置、schema、workflow、inventory、依赖、锁、ceilings 和保护零改；不运行服务、浏览器、Docker、kubectl、SQL 或发布脚本，不读取既有凭据/私密配置/运行报告，不操作任何既有实例、数据、Profile 或平台。** 未来实施、合并、部署和关单分别待授权。无新增 Accepted Risk，无 UI/IA/Figma 增量。

## 2. Design Basis 与适用 Authority

当前设计文档工作：**Design Basis: N/A**，因为不改变生产运行语义。

未来实施提案：**Reuse Existing Architecture / NOT_READY**，待用户确认 §1–§7 并在 #301 记录 Ready 后方可编码。复用：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md) 与 [项目边界](project-boundaries.md)。只有 EXTRACT / RETIRE，无迁移、旧 ID 映射、兼容 wrapper、fallback、双读写或第二事实源。
- [#863 当前 SHEIN worker 设计](shein-worker-numeric-persistence-retirement-v1.md)：当前 Organization/Store UUID、native live 授权、独立 session transport、受权 CLI、Store 审计和唯一执行归属；UNKNOWN 不自动接管。此设计已实施，本文不重开其身份/持久化/副作用边界。
- [当前来源业务闭环](source-business-closure-v1.md)、当前 Product sourcing/Catalog/SRC owner、`integration/crawler/amazon.Processor` 的 source port，以及现有平台 HTTP、定价/品牌纯规则 owner。
- 先前数字执行/刊登、配置、运维与 API token 退休设计的准确历史证据。旧广泛保留表述仅在用户确认本矩阵后，由本矩阵明确替代；不重写旧证据。

本提案不增加/转交当前事实 owner、表、schema、授权、公共当前领域合同、事务、幂等键、状态机、retry/UNKNOWN/recovery 或外部副作用。超过 30 文件和大删除量的风险提示线由这里的完整矩阵及既有当前合同覆盖，不通过拆批规避。SHEIN 内部基础客户端的旧数字诊断参数/错误字段收缩已在 §5 单列，平台请求/响应 DTO、HTTP 方法、headers、认证失败识别及既有显式刷新合同不变。若真实消费者要求改变上述当前边界，停止对应项并报告，按适用 Independent Architecture 准入处理，不能在删除时临时发明新架构。

实施发现（IMPLEMENTATION_TEST）：编译证明当前 `SessionTransport.CookiesFromBrowserState` 仍消费旧 cookie_manager.go 中 `CookieData` 与 `buildCookiesFromJSON`。这两个既有纯值声明原样抽取至同包 `cookie_values.go`；SessionTransport 与测试字节保持，domain/expiry/SameSite不变。其余数字 manager/provider、字符串fallback和旧构造器仍退休，无新 owner/协议/副作用。同一根因的 Amazon `qualityMetrics` 仍引用 service_metrics.go 的 `normalizeMetricsRegion`/`copyInt64Map`，两函数同样原样归入现有 quality_metrics.go；旧 Task/去重/regionGuard service 指标主体退出。此项校正 §5.1 的文件独占判断，服从保留当前能力不变量。

## 3. 实际消费者与完整闭包

调查只执行了 `rg`、tracked 文件核对及 `go list` 元数据；没有调用应用 main 或构造会启动浏览器/网络的 service。基线上 `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...` 与全 tracked Go 符号检索相互核对，另对 **31 个** `./cmd/...` 读取完整依赖图。

### 3.1 不可达的旧采集/任务岛

```text
无当前 caller 的 platformbase ProductFetcherBuilder / fetchMode
  -> crawler/fetcher FetcherFactory / RemoteAPIProductFetcher
  -> marketplace/sourceproduct RawJsonDataClient / ProductFetcher / CacheManager
  -> 旧数字 tenant/store/import-task DTO 与旧 RawJSON 写入 port

无当前 caller 的 crawler/amazon.NewAPIService
  -> NewService -> BaseService + AsyncTaskProcessor
  -> shared.CrawlerTask / CrawlerResult / 数字 tenant 结果 Redis
  -> infra/httpx 的 crawler-only HTTP handler

无生产 import 的 domain/task
  -> TaskEventV2 / NormalizeTaskEventV2 / NormalizedTask / PlatformRoute
```

实际包级入边：platformbase 无；crawler/fetcher 仅 platformbase；sourceproduct 仅 crawler/fetcher；domain/task 无。31 个命令都不依赖这四根，也不依赖 crawler/shared **根包**或 infra/httpx。调用在测试或上述岛内闭合，不能把工厂声明当作当前装配。

`crawler/amazon` 包本身有有效消费者 `integration/crawler/amazon`，所以禁止整包删除。有效路径是其 `Source`/`Processor`/`LegacyCrawlSource` port -> `CreateProcessor` -> `AmazonProcessor.ProcessWithContext/ProcessBatchWithContext` -> browser pool / instance processor / extractor。NewAPIService/NewService 的数字 Task/HTTP 服务与这个 Processor 是不同声明。保留所有实际 Processor、浏览器池、批量/超时、代理、风险/质量、失败证据、域名/邮编/产品规则；通用 platform/workerpool 也保留。

### 3.2 SHEIN 数字 provider 尾链

`shein/client.APIClient` 的两个导出构造器只在同包测试使用；`CookieManager` 只由该客户端构造，`StoreConfigProvider/CookieProvider` 无真实生产实现注入。#863 worker 已直接使用 `SessionTransport` 和当前 scope 的 Cookie；不再调用这些数字 Store 构造器。

`BaseAPIClient` 则被九个有效 API 子包及 PriceManager/ErrorHandler 使用。它的 HTTP/错误/headers 行为必须保留；只移除旧数字 tenant/shop 诊断上下文，不删除 API 包或变更平台 payload。当前 worker 的 SessionTransport、其 headers helper 和 bounded read 完整保持。

### 3.3 孤立数字适配

`internal/pricing.CostCalculator` 没有生产包消费者；唯一数字 OperationStrategyProvider 查找接口没有生产实现，五个成本 helper 只返回默认值。实际成本纯公式已在 `core/config.CostCalculationConfig.CalculateTotalCost`，价格选择在 `marketplace/productpolicy`，它们和 SHEIN/TEMU PriceManager/PriceCalculator 保留，不另造计算器。

`authorizedbrand.ConfigFromStore` 仅有两个旧 StoreInfo 测试，没有生产调用；当前 publishing/content 直接使用现有 Config/Resolved/context。移除这个旧 Store adapter 和失去最后消费者的 StoreInfo，不改变品牌开关/trim/授权内容规则。

### 3.4 专属操作尾部的实际断链

`deployments/docker/Dockerfile.amazon-crawler-api` 仍编译 `./cmd/amazon-crawler-api/main.go`，该入口在当前 tracked 源中不存在。其 Compose、Kubernetes API/LB/监控、push 脚本仍宣传旧服务。应同批退出这些**源码声明**，不能再提供一个无法按当前源码构建的发布入口，更不能通过恢复旧 main 来维持旧设计。

既有运行镜像、外部 endpoint、DNS/云资源、监控对象、凭据和数据是否仍存在均 UNKNOWN；源码断链不证明其现实运行状态，也不授权删除/停机。

## 4. owner、持久化、副作用及父目标剩余项

| 对象 | 当前 owner / 本批处置 | 边界 |
| --- | --- | --- |
| 当前 Organization、成员、授权、计费 | 既有 native identity/authz/Organization/commercial/ledger | 全部保留；opaque ID 即使全为数字仍是当前合法 ID，不能 ParseInt 映射或依数字外形删除 |
| 当前采集/商品/lineage/资产批准 | SRC、Product sourcing、Catalog、Collection/Asset/Preparation | 当前创建/刷新/读取/批准合同与 #856/#859 证据保留；不再开展新来源功能或验收工具来代替退休 |
| 旧 RawJSON / 数字 Task 结果 | §3 无注入 port、旧 Redis 缓存/内存 Task | 只退 caller/DTO/源码，任何既有表、行、键、TTL、队列和未完成任务都不探测/迁移/清理；不把旧结果同步到 Catalog |
| 当前 SHEIN session/Store | #863 worker、当前 Store Center 与可选执行控制表 | 全部字节保持；原 request key/nonce/generation、live 授权、版本/审计事务、关闭证明与 UNKNOWN fence 不变 |
| 有效 SHEIN/TEMU/Amazon/SDS 平台合同 | 各平台 API/规则/transport 与 source adapter | 保留；SDS 等第三方自己的 numeric merchant/tenant 字段不是本地 Organization 解析 owner，不按字段名误退 |
| 两种 SHEIN 属性及剩余 listingruntime 值 | EnrichedSkuInfo/EnrichedSkcInfo；ProductImportMapping、PricingRule、OperationStrategy 的有效规则输入 | 完整保留当前字段/JSON tags及规则，不清空整个 listingruntime；只退无消费者的 StoreInfo |
| 当前 SourceIdentity/SourceRequest | sourcing、当前 1688/Amazon/localagent source adapters | SourceRequest 只减失去最后消费者的 TenantID；保留实际 Region/StoreID/source 值、SourceKey/lineage/正常化合同。SourceIdentity 的旧 alias 仍被适配器/localagent读取，持久化入口已显式拒绝这些字段；这项活跃来源值合同不作“无人消费”删除，也不视为数字 tenant resolver。其 compatibility 注释/别名债务如需改变，须由当前 sourcing owner 另按真实公共合同设计，不新增 alias 或放宽现有拒绝 |
| 通用基础设施 | queue/RabbitMQ、workerpool、Retry、Redis、HTTP helper、浏览器支持、Prometheus 存储告警、WeCom基础服务 | 保留，不因本批最后一个旧业务 caller 退出而盲目递归删除共享组件 |

失败、超时、响应丢失、重试/重启、取消和并发不引入任何新协议：退出的旧岛不获得替代恢复 owner，当前路径继续其原协议。没有把旧 Redis/HTTP 结果和当前业务库拼成新事务；没有新 provider 请求、刷新、写入、补偿或计费行为。

#301 原验收 O1/O2 已有 #856 当前源账号的受控 PG 组合证据，#859 当前来源业务证据归 #30/#307；O3/O4 在本批继续退剩余数字合同/旧 resolver 表面；O5 不承接历史数据；O6 不操作既有环境。它们仍须按实际范围分别记录，不能用“包不存在”“31命令可编译”或本批合并把全父目标运行/用户验收写为 PASS。完成本批也不自动关闭 #301。

## 5. 一个完整批次的删除、保留与抽取矩阵

### 5.1 专属源码和测试整批 RETIRE

以下数量只是准确基线文件集，不是拆 PR 单位；同目录出现新消费者时先报告，不使用 missing-skip。

| 闭包 | 精确路径 / 声明 | 同批条件 |
| --- | --- | --- |
| 原始采集/缓存/工厂 | platformbase 的 errors.go、fetcher_builder.go/test、fetcher_mode.go（4）；crawler/fetcher 的 contracts.go/test、remote_fetcher.go/test、amazon_source_adapter_test.go（5）；marketplace/sourceproduct 全部12个当前 Go 文件 | 全部退出，含旧 local/remote factory、缓存读写、RawJson DTO、validator、adapter、parser 和独占 fixtures；不搬目录或留类型 alias |
| Task 消息协议 | domain/task 的 message.go、normalize.go/test、normalized_task.go、platform.go/test（6） | 无当前消费者；不影响当前 BusinessTask、Temporal、submission.ExecutionKernel 或通用 RabbitMQ |
| Amazon 旧服务 | crawler/amazon 下 api_service.go、crawler_service.go/test、worker_processor.go、concurrency_control.go、region_guard.go/test、service_metrics.go/test、redis_config_test.go | 只此10文件；保留 processor/factory/instance/batch/browser/extractor、FetchError、域名/URL、产品检查、quality/失败证据及其全部有效测试 |
| 数字 Task 结果基础层 | crawler/shared 根下 base_service.go/test、errors.go、job_handler.go、redis_config_test.go、result.go、task.go/test、util.go（9） | 只删除根目录这些直接子文件，**不递归删除 shared/browser**；所有 browser manager/installer/proxy/fingerprint 和当前 SHEIN close proof 保留 |
| crawler-only HTTP | infra/httpx 的 crawler_service.go、crawler_base_handler.go/test、crawler_amazon_handler.go/test（5） | 数字 TenantResolver/tenant-scoped task HTTP 一并退；CORS/auth/logging/health/response/params/parser/metrics 纯基础层与测试保留 |
| SHEIN 数字 Cookie client | shein/client 的 api_client.go、cookie_manager.go、runtime_types.go、api_client_store_config_test.go | 整条 numeric StoreConfig/CookieProvider/refresh/环境 proxy override 及解析私有 helper 退出，无新凭据 owner；SessionTransport 不动 |
| 孤立 numeric Cookie error | shein/sherr/cookie_errors.go | 只退 CookieLoadError/New/Is；classification.go 去其专属分支，errors_test.go去其专属 case，保留其它认证/过滤/重试类型和分类断言 |
| 数字成本 wrapper | pricing/cost_calculator.go/test | 无新抽取；原成本纯公式/价格选择 owner 及测试保留，不影响 SHEIN/TEMU PriceManager/PriceCalculator |

### 5.2 混合源码精确收缩

| 文件/声明 | 收缩 | 明确保留 |
| --- | --- | --- |
| shein/client/base_client.go；error_handler.go；shein/api/types.go | 删除 BaseAPIClient 的 tenantID/shopID 字段、旧构造数字参数、GetTenantID/GetShopID，以及错误中仅作旧数字上下文的 TenantID/ShopID；CookieError 无消费者则整声明退，认证过期类型保留 Code/Message | BaseAPIClient、APIErrorHandler、所有平台 API/DTO/endpoint、headers/方法/响应解析、typed auth failure、显式 refresh/retry 及 cost-price 403 规则原样。现有 API 测试改用相同 localhost transport 的新签名，不跳过正负断言 |
| authorizedbrand/types.go 与 context_test.go；listingruntime/types.go | 退出仅旧 StoreInfo 转 Config 的 ConfigFromStore、两个专属 Store projection 测试和 StoreInfo 类型 | Config/Resolved/context、当前 enabled/trim/授权品牌内容规则及 context测试；ProductImportMapping 及两种 SHEIN 属性合同完整保留 |
| product/sourcing/source_request.go/test | 只删无当前读取的数字 TenantID 字段和退出 DTO 的专属测试初始化/断言 | 当前 SourceRequest 的其余字段、正常化、variant、identity、Key/SourceKey 和当前生产适配器行为不变 |
| core/config 的 type_amazon/type_platform/defaults/loader/loader_builder/loader_platform/config/validator_amazon 及对应测试 | 只移除 Temu/Shein FetchMode；Amazon DataFreshnessDays、ProductDedupe、RegionGuard、ConcurrencyControl、RemoteAPI 和仅服务它们的类型、默认/加载/env/校验/示例/断言 | 原 Viper/Manager/合并算法、HTTPClient、Amazon Enabled/Zipcodes/CrawlTimeout、Risk/Quality/Proxy/FailureArtifacts/SPAPI/ConfigPaths、SDS/身份/计费/图像与通用 Redis/queue/Retry/workerpool 配置完整保持；不删混合测试中的 SPAPI/当前权限等有效断言 |

不预建新 helper、规则库、client interface、runner 或恢复机制。有效行为复用已经存在的当前 owner；旧岛专属行为没有替代 owner。

### 5.3 专属发布/部署/监控源码及文档尾部

同批 RETIRE：`deployments/docker/Dockerfile.amazon-crawler-api`、`docker-compose.amazon-crawler-api.yml`；`scripts/push-amazon-crawler-api-dockerhub.ps1`；`config/config-amazon-crawler-api.yaml`；`deployments/kubernetes/amazon-crawler-api`、`amazon-crawler-external-lb`、`monitoring/amazon-crawler-api` 的当前 tracked 专属声明及说明；`monitoring/alertmanager-wecom/alertmanagerconfig.yaml` 中只匹配旧 Amazon service 的独立 AlertmanagerConfig 源。混合 kustomization/README 只去这些资源引用和旧发布指令；不把去掉 matcher 改成广泛接收所有告警。

已核对两个旧子树 namespace.yaml 都声明 `task-processor`，没有其它保留 kustomization 引用它们；本轮只退旧子树内的重复声明和引用。保留 `shein-listing/base/namespace.yaml`、`listingkit-workbench/base/namespace.yaml` 及其当前消费者，保留所有 SHEIN/其它 workload 凭据声明、WeCom 服务/部署/configmap/secret 引用和 `monitoring/prometheus-storage` 规则/窄路由。删除源码声明不授权删除实际 Namespace/Secret/endpoint/监控对象或修改现有凭据。

当前 `config-dev.yaml/config-prod.yaml` 等 tracked 示例只删上表已无消费者的键，不读取/重写任何运行配置或凭据值。另已核对四个共享源仍声明旧 Amazon remoteAPI 非凭据配置：`shein-listing/base/secret.example.yaml`、`overlays/prod-store-shards-example/secret.example.yaml`、`overlays/prod/secret.example.yaml`、`overlays/prod/external-secret.yaml`。同批只去其中对应已退出 RemoteAPIConfig 的 enabled/baseURL/timeout 配置项及映射；Secret/ExternalSecret 资源本身、其它有效配置与全部凭据键/引用原样保留，不复制/读取值或操作实际 Secret。源码树外/ignored 的同名配置、旧报告、缓存和 Profile 不碰。

直接文档同批加窄失效/替代顶注并校准当前操作段：Amazon commercialization checklist/execution-plan/runtime-flow、centralized-deployment-plan、Kubernetes readme/monitoring/WeCom README；旧 dated 记录只留历史，不改写为 PASS。Architecture README、Legacy Register、module-target-mapping/code-health/current guard 清单同步实际 owner。先前 #742/#743/#744/#819/#822/#839 对此次准确专属源的保留说明改为历史并引用本设计；平台 API、两属性类型、规则、当前采集与 worker 的保留决定继续有效。

```text
Legacy decision: RETIRE | EXTRACT
Reusable behavior: 当前 Processor/source port、平台 HTTP/错误/headers、价格/品牌纯值、current Store/worker 合同
Current owner: 现有 integration/crawler、平台 API、productpolicy/config 纯公式、publishing/authorizedbrand、Organization/Store/SHEIN worker
Cutover/deletion condition: 用户确认本完整矩阵并Ready；无生产装配的专属岛退出，当前接口/规则及真实依赖图保持；无旧资源迁移/接管/清理
```

## 6. 必要验证与交付

设计阶段只核对源码/依赖/链接和适用原文档 guard，不为纯文档制造 RED 或运行环境。实施阶段沿原 retirement guard 先 RED（准确源/声明仍存在），再执行删除/收缩后 GREEN。保留的纯规则/API/context/取消/权限/隔离/UNKNOWN 断言不得随旧实现测试一起删；不新建验证工具、完整故障矩阵或验收 session。

实施必要开发自检限相关原 absence/import/配置/当前 source adapter、平台/品牌/价格、Amazon Processor/browser 及 #863 worker 测试和构建/31命令元数据；完整原 CI 留稳定候选。无需为了源码删除另启动 PG/Redis/浏览器或真实 provider；如果实际发现必须做资源执行，先按具体必要性和范围取得授权。未变化代码的已绑定有效证据可复用，不能冒称新 SHA 直接执行。

### 6.1 原检查、inventory 与 workflow 的精确处置

| 对象 | 同批允许的最小校准 | 不得改变 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原保留列表只减 §5 真正退出的 fetcher/platformbase/StoreInfo/旧资源；转为准确 absence/声明禁入。现有 source normalizer、marketing/两属性、pricing、当前命令依赖和 SHEIN worker 隔离检查保持 | 所有非目标旧 absence、8命令真实 Go list、安全断言与读取失败语义；不以 missing skip 或空壳过测试 |
| tests/import_boundaries_test.go / target_architecture_phase2_test.go / 原 docs/config guards | 仅删已消失文件项/旧工厂专属期望，必要时用原扫描 owner 检查根缺席；指向旧 fetcher 的“应保留”文本改到本次 accurate absence/当前 source port | 原依赖方向、SDK/租户/商业/存储边界、ceilings/ratchet、解码/重建侦测。混合文件的其它当前检查完整保持 |
| #839 相关原资源/告警 guard | 仅将本轮旧 Amazon source/专属 route 的保留断言改为 absence，保留存储告警内容、同 namespace 窄路由、WeCom共享服务/凭据与其它 workload 的原断言 | 不取消共享 Namespace/Secret/存储告警，不放宽 alert routing |
| phase2-runtime-inventory.md | **未来实施时允许仅删除以下七条经实际依赖确认退出的 CURRENT 消费者行**：core/logger→crawler/fetcher；platform/redis→crawler/amazon、→crawler/shared；platform/queue/rabbitmq→crawler/fetcher、→platformbase；platform/workerpool→crawler/amazon、→crawler/shared | 历史/dated行、计数、其余消费者和 ceilings 不动；core/logger→crawler/amazon/browser/extractor 等仍有引用者保留。若刷新与七行矩阵不符，报告准确差异，不自行扩大校准 |
| 两套 workflow 及其它 CI | **零修改**，沿原触发/分类运行；原 listingadmin test-only 当前合同 race 和 current-application/shein-login-worker builds 保留 | 必需/安全/PG/资金/发布检查、权限、并发、pinned SHA、保护规则、依赖/锁全部保持 |

七行只是未来实施矩阵的一部分；本设计 PR 不应用 inventory 补丁。没有把检查历史重写成当前验证，也没有把 retired test target 移除来降低安全门禁。

## 7. 设计确认、实施准入与停止点

1. 当前交付为本文、Architecture README 和 Legacy Register 的提案指引，以及 #301 当前段；**源码/测试/配置/部署/workflow/inventory/依赖零改**。设计检查只核对实际消费者、完整闭包、保留能力、§5/§6/§6.1 与权限；不重开全仓架构，不默认增加 AI 互验线程或在同 SHA Running 时重复触发评审。
2. 用户确认准确设计 §1–§7 后，在同一主要 PR 记录 Reuse Existing Architecture / Ready，明确唯一 Writer；确认前实施 NOT_READY。如有真实当前公共合同/事实 owner/安全/持久化改变需求，先报告具体 Must 和适用 Independent Architecture，不能用“退休”绕过。
3. 用户确认后完整闭包同批连续实施、必要自检、提交/推送与 review/CI 修复；不机械拆小 PR，不恢复旧 entry、迁移、ID bridge、fallback 或新消费者。finding 先分类，检查同根因 siblings；非 Blocker hardening 留 Backlog，不新增试用前置。
4. 完整候选集中做一次适用独立最终交付检查，只核对实际 diff/受影响路径和有效证据；修复后只查相关增量。设计 CI 与实现 CI/最终审核 SHA 分开。实现者不签发用户验收。
5. 交出准确最终 HEAD、原 CI/复核、保留入口与限制供用户决定合并；合并需准确 HEAD 另行授权，之后核对实际 merge SHA 的 push/main CI。#301 保持 OPEN；任何关单、部署、真实资源切换/清理和 provider 操作均需单独授权。

本设计没有为 #301 的运行验收新增门槛，也不把取消历史迁移、源级零 import 或本批完成写成全部原验收 PASS。当前内部试用沿现有交付继续。
