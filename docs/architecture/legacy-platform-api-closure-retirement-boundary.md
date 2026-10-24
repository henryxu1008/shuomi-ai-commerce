# #301 旧平台 API 调用实现完整退休闭包设计

Refs [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)。#301 保持 OPEN，#29 不重开。

- Status: **PROPOSED / DESIGNING / NOT_READY**。本轮用户仅授权完整设计；本文待适用独立设计复核与用户确认，不授权正式源码删除。
- Design Basis：设计交付 **N/A（仅三文档）**；拟实施 **Independent Architecture**。替代旧公共平台调用合同与显式 KEEP，跨平台且超过30工件，必须先明确 IMPLEMENTATION_READY / Issue/PR Ready，再改生产路径。
- 基线 main **067ffafa01c91376ff53a5e10c83d65e1b476bb4**，tree **e16fe8fb7b4c2e4a5474626bf73f62f95c481478**；#923 已合并，只复用前批完成事实，不冒充本轮设计/实现/运行证据。
- 形成设计候选前已同步实际 main **7d82be549f0aded91b57d9c3efee0b55d5da9c4f**（#925，tree **1514cde83b57243e222c4f77e5006481889164ef**）。该增量只涉及采集扩展/说明/脚本与共同索引，本批全部旧 API/配置/保留 owner 的调查代码未变化；共同 README 的 #925 项完整保留。本轮相对该 main 仍只有三文档，不重复实施 #925。
- 唯一 Writer 原会话 `01a133ef-757c-73e3-b8f6-76958c642397`，独占原 worktree，分支 `codex/issue301-legacy-platform-api-closure-design`；[接单及前批当前段逐字归档6381325202](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6381325202)。已 fetch 并同步上述 main。#925 采集入口有独立 owner，本轮不重复派工或改其源码/文档，只给共同 Architecture README 加前缀索引。

## 1. 用户结果、范围与产品依据

使用者是当前平台合同、来源采集与独立 SHEIN worker 的维护者；可见交付结果为旧平台调用实现及专属配置/测试/保活检查/文档尾部一次退出，当前用户仍能沿原入口采集、准备草稿、登录及同步 Store。当前没有因此阻塞的核心用户操作，不为父目标清零创造新产品、验收平台或内部试用前置。一个 Delivery Batch / 主要 PR 覆盖完整闭包，不按目录、行数、测试或平台机械拆批。

Authority 为本轮用户“可以，先按 Independent Architecture 设计、确认后实施”的直接决定、[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)与[Issue 驱动规则](../engineering/issue-driven-development.md)。原“通过 Cookie 访问的接口都可退休”提议已被用户撤回，不是本轮依据；不能以认证方式推导删除权限。UI/IA/Figma N/A：不改变当前页面、导航、用户交互、平台开放范围或发布准入。

Scope 为三组整链：`internal/amazon/api`，`internal/temu/api`，SHEIN 闲置 API/通用客户端/定价调用及完整 helper/DTO 尾部；覆盖其已失去消费者的配置类型、加载/default/env/校验分类、三个受管 YAML 样例及 §5/§6.1 的准确尾部。先 EXTRACT worker 使用的两项纯转换函数，当前 DTO 原位保留。不是退休 Amazon 浏览器爬虫，也不是删除所有 SHEIN API 路径。

Must：当前采集/原 Amazon 浏览器与 debug、AI/图像、当前平台规则/草稿/预览/严格解码/JSON/版本/hash/读取拒绝、有效共享基础层、官方 Store adapter、独立 SHEIN 登录及 Store 同步保持；原凭据、授权/Organization、owner/claim/fence/UNKNOWN 不变。workflow、完整 inventory（CURRENT/历史/计数/ceilings/检查逻辑）、依赖/锁/保护规则零改。不操作既有实例、数据、凭据、会话或进程，不读取实际 .env/有效私密配置，不调用平台/provider 或浏览器。

Should：复用现役 owner、原检查和 Go 标准库 AST/HTTP/测试，不新建抽象。Out of Scope：官方平台 adapter/现役 SessionTransport 退休，业务平台能力/接口重新设计，表/迁移/backfill/数据清理，旧 API 兼容层或替代客户端，通用工具/工程化平台、依赖清扫。Accepted Risks 无新增；旧环境消费者、凭据/远端记录与在途请求 UNKNOWN，运行和用户验收 NOT_RUN。

Threat Model 限误删现役 DTO/JSON 字段或 worker 只读流程，旧 refresh/retry/凭据逻辑被包装回当前 owner，原检查强制保活或改成 missing-skip，以及把源码退休误称为远端停止/数据清理。不得全仓禁用 Client、Product、API、PriceManager、Cookie、Warehouse 等同名概念。

## 2. Design Basis、根因与前批 KEEP 替代

根因是旧刊登/任务消费者先退出后，平台 API/manager/请求响应互相引用，显式历史 KEEP 与原“文件必须存在”检查继续保活；SHEIN 还把现役纯转换与闲置 HTTP Client 放在同一文件，当前测试尾部又构造旧 BaseAPIClient。零 importer 只是现状证据，不能撤销用户已经批准的 KEEP。本文提出准确替代，必须在用户确认后才生效。

| 旧依据 | 本轮确认后仅替代 | 仍然有效 |
| --- | --- | --- |
| [#822 Amazon 刊登闭包](legacy-amazon-listing-closure-retirement-boundary.md) 的整 API/SPAPI 配置 KEEP | 未装配的 amazon/api 全实现、专属 DTO/测试、SPAPI/市场解析与孤立 AttributeMapping 配置类型 | 当前 acquisition/amazon、原 crawler/amazon 浏览器 Processor/池、共享 Amazon product model、debug、当前配置中的浏览器风险/质量/超时/代理 |
| [#819 TEMU 刊登闭包](legacy-temu-listing-closure-retirement-boundary.md)、批量重上架/保存提交/前置/类目模板/图片等旧 KEEP | temu/api 整根（含 re-export/同名 DTO），Temu/Shein 旧 PlatformConfig 的 HTTP 包装配置 | 当前 marketplace/temu/goods/rules、官方 integration/temu、通用 HTTPClientConfig/BuildHTTPClientConfig/Retry 与 Alibaba1688 |
| [#912 SHEIN 策略合同闭包](legacy-shein-policy-contract-closure-retirement-boundary.md)、[#743](legacy-auto-pricing-retirement-boundary.md)、[#744](legacy-shein-activity-retirement-boundary.md)、[#817](legacy-shein-publishing-runtime-closure-retirement-boundary.md) | pricing_repo.go、api/product.PriceManager/manager/Client、营销等闲置 API 及通用 BaseAPIClient 的准确保留含义 | §5.2 当前 Product/SKC/SKU/属性/品牌/商户/仓库 DTO 与 JSON、清理/规则/草稿、SessionTransport/当前 Cookie 值解析、worker/Store |
| [#836](legacy-runtime-config-closure-retirement-boundary.md)、[#885](legacy-sds-runtime-closure-retirement-boundary.md)、[#917](legacy-config-facade-closure-retirement-boundary.md) | SPAPI 与 TEMU/SHEIN 专属配置入口/default/env/分类/原校验测试，不再要求无消费者字段存在 | 六组现役 Validate* 聚合、顺序/错误格式、1688/浏览器/AI/身份/计费/DB/Redis/当前加载与通用 Retry |
| #893 logger、#901 来源适配、#923 Prompt 等广泛 KEEP/目录与历史说明 | 仅本矩阵列出的旧调用 owner 与其保留建议 | 原已完成退休不复活；所有未列入的 owner、当前原检查与历史证据原样 |

不是把已退出平台副作用转给浏览器爬虫、官方 adapter、规则或登录 worker。当前只读 session 能力已由其原 owner 提供，无需兼容旧 BaseAPIClient，也不把旧 auth refresh/retry 复制到当前 transport。成熟现有标准库/组件足够，本轮只抽两项已有纯函数。

## 3. 实际消费者与 owner：contract → implementation → injection → consumer

调查覆盖所有 tracked Go 的 decoded import、alias/selector/声明、tests/build tags 与 hack/debug 独立模块，另核对原检查、配置键与非 Go 文档/受管样例。区分生产 caller、内部自调用、专属测试和保活检查；不执行旧 constructor、CLI 或请求。

| 对象 | 基线实际消费者 / 注入 | 结论 |
| --- | --- | --- |
| amazon/api 18 Go（17生产/1测试） | 外部 Go importer 为0；签名、refresh、rate limit/retry、catalog/listing/价格/库存/上传仅包内闭合，旧 Postman 测试不是现役调用依据 | 整根 RETIRE，SPAPI 配置/市场解析同闭包；不影响原浏览器 |
| temu/api 28 Go（26生产/2测试） | 整根外部 Go importer 为0；root alias、client/auth/retry、category/query/template/product/inventory/pricing/types 仅内部及专属测试 | 整根 RETIRE，不能只删 client 后留下 orphan DTO |
| SHEIN BaseAPIClient/APIErrorHandler、各 API Client/manager、pricing.PricingAPI | 无包外现役生产构造者；BaseAPIClient 被旧 API 及其测试引用，另被当前 session 测试最后两行构造 | 删除旧实现，转换该测试尾部，不能为测试复建兼容层 |
| 当前草稿/有效规则 | publishing/shein 与 marketplace/shein 使用 api/product/attribute DTO，authorizedbrand 使用 BrandItem/BrandListResponse；这些类型不依赖旧 Client | 保持 §5.2 类型完整 body/字段/tags/JSON、原 owner 与调用路径，不搬包或改 alias |
| 当前 SHEIN 登录读取 | sheinlogin/session_identity.go → NewSessionTransport.Read → 商户/仓库/地址三个 getter；地址响应调用 WarehousesFromStoreAddresses | transport、cookie解析、worker 源码全文件 KEEP；两纯函数抽到同 DTO 包 |
| Store 当前官方集成 | app/runtime/currentapplication/official_store_connection.go → integration/shein、integration/temu 与当前 Store owner | 全部 KEEP，与旧 Cookie/MALL_ID/BaseAPIClient 不是同一执行路径 |
| core/config 专属槽位 | .SPAPI/市场解析、Platforms.Temu/Shein 无 config 包外生产 reader；两平台只经 wrapper 构建旧 HTTP 值；AmazonConfigPaths.AttributeMapping 无 reader/default/loader | 仅准确专属字段/类型/构建/显式 env/默认/校验分类退出；当前通用 HTTP/1688 配置有实际消费者 |
| 当前 acquisition / 原 browser/debug | acquisition/amazon、crawler/amazon 原 Processor/浏览器池、internal/model/amazon_product.go、hack/debug/test-amazon 等有当前消费者 | 全 KEEP，禁止从旧 API 退休推导 crawler/model/factory/debug 退出 |

没有当前正式入口装配上述旧 mutation，也不新增接管者。未知仓库外/真实环境使用只记录 UNKNOWN；如发现具体当前外部可观察契约或持久化运行态要求兼容，停止对应删除并报告用户，不能自行增加 fallback、迁移或适配层。

## 4. 持久化、凭据、平台副作用与当前行为边界

### 4.1 owner 与副作用

旧 Amazon token/cache/签名、TEMU cookie/mall/auth retry、SHEIN refresh/error/manager 的 owner 随旧实现退出；不转存任何凭据、不撤销远端 token、不登出/停止真实会话、不删除 Cookie/Redis/文件。旧商品修改、上传、发布、价格/库存、上下架、营销、地址新增/检查等平台副作用不重路由、不重放，不发真实请求来证明它们已退出。

当前官方 Store adapter 的鉴权与同步事实、独立 SHEIN browser/session/商户仓库读取、Store claim/fence/UNKNOWN 和显式授权 CLI 继续在原 owner；本轮不新增状态机、事务、幂等身份、恢复/重试或持久化事实，不自动接管 UNKNOWN。原 worker 的地址列表后备读取流程属于现役合同，完整保留；不得因 Hard-Cut 规则错误删除这条现役平台读取路径。

### 4.2 JSON / schema /配置变化

当前 Product/SKC/SKU/Extra、属性、品牌、商户/仓库 DTO 及其所有传递类型保持整个声明、字段、JSON tags/零值、FlexibleString.String/UnmarshalJSON 不变。不能仅因某字段未被 selector 读取就剪字段；SKU.CostPrice 仍使用 CostPrice，SiteInfo 仍使用 SubSite。草稿保存/读取/预览、严格解码、规范化/hash/版本、SourceIdentity 四槽位/读取拒绝不变。所有数据库模型、repository、migration/schema 注册零改；已有记录不读不写。

配置 public Go/YAML 面：确认后移除 amazon.spapi 整槽及市场类型、AmazonConfigPaths/唯一 AttributeMapping 槽，Platforms.Temu/Shein 和专属 PlatformConfig/BuildPlatformConfig；这是真实删除合同，不能宣称配置面零变化。受管 config-dev/prod/test YAML 中仅删除实际存在的 amazon.spapi、platforms.temu、platforms.shein 子树及专属相邻注释，不增补不存在的块、不改其它键值或样例。样例没有 amazon.configPaths，本轮不触碰私密 config/.env。新鲜/现有配置文件的通用 Viper unknown-key/AutomaticEnv 行为不重写、不添加旧键 fallback/映射，也不声称未知键必须严格拒绝；只是旧类型不再解析消费，显式绑定与默认值退出。

HTTPClientConfig/BuildHTTPClientConfig/默认 TLS/timeout 与通用 Retry 的合同保持；1688/Browsers/Amazon 风险与质量配置全保留。旧 SPAPI 的七项凭据校验与专属错误分组退出；六组当前 Validate* 聚合、当前模块 rank 数值/顺序、格式化与其它校验不变。不调整当前使用者的安全开关。

### 4.3 唯一 EXTRACT

从 warehouse/client.go 原样抽取 WarehousesFromStoreAddresses 与 collectSaleCountries 到同包 warehouse/address_conversion.go，包名/签名/body 不变；旧文件随后整退。保留：空输入的非 nil 空 Data、空仓库 code 跳过、原条目顺序与重复项、名称/code/type、Meta.Count；国家顺序首见去重/过滤空值、无站点返回 nil。不同“有站点但全空国家”产生的空 slice 不改成 nil。不引入 repository/HTTP/日志依赖或新公共 abstraction。

当前 session_transport_test 的最后 BaseAPIClient 构造改为直接调用现役 sessionHeaders(origin,path,"self_operated")，原 self-operated 四项 header 断言及商户/仓库/地址两 origin 全部断言保留；session_transport.go/cookie_values.go 与 sheinlogin 源码零差异。仓库纯转换的有效值断言归入新 address_conversion_test.go，原死 Client 的 HTTP 测试退出。

## 5. 精确删除 / 保留 / 抽取矩阵

### 5.1 九十四个整退 Go 工件

下表为准确基线枚举；不把通用目录同名类型或现役子包扩大为整根删除。

| 根 | 整退文件（相对该根） | 数量 |
| --- | --- | --- |
| internal/amazon/api | auth.go、aws_signer.go、catalog.go、client.go、inventory.go、listing_details.go、listing_models.go、listing_operations.go、listing_postman_test.go、listings.go、logger_port.go、pricing.go、product_types.go、ratelimit.go、request.go、retry.go、seller.go、uploads.go | 18 |
| internal/temu/api | api.go；category/category.go；client/auth.go、auth_config.go、auth_error_detector.go、auth_factory.go、auth_request_sender.go、auth_retry_handler.go、client.go、config.go、cookie_logging_test.go、errors.go、http.go、http_test.go、interfaces.go、logger_port.go；inventory/inventory.go；pricing/pricing.go；product/api.go、extension_types.go、goods_types.go、request_types.go、sku_list.go、sku_types.go；query/query.go、query_types.go；template/types.go；types/types.go | 28 |
| internal/shein/api | 根 types.go；attribute/client.go；category/client.go、interface.go；image/client.go、client_test.go、interface.go；marketing/activity_types.go、client.go、client_test.go、interface.go、price_types.go、promotion_types.go、workflow_example.go；other/client.go；pricing/client.go、interface.go；translate/client.go、interface.go；warehouse/client.go、client_test.go（先抽§4.3）；product/client.go、client_test.go、cost_price_test.go、image_helpers.go、image_helpers_test.go、interface.go、inventory.go、inventory_manager.go、language_list.go、language_list_test.go、name_length_config.go、name_length_config_test.go、price.go、price_manager.go、product_manager.go、request.go、site_list.go、site_list_test.go、stock.go | 40 |
| internal/shein/client | base_client.go、base_client_test.go、error_handler.go、logger_port.go | 4 |
| internal/shein/pricing | pricing_repo.go | 1 |
| internal/core/config | amazon_marketplace.go、amazon_marketplace_test.go、loader_platform.go | 3 |

SHEIN root api.APIResponse/APIError/auth-expired helper 只服务死客户端，不是当前 SessionTransport/DTO 的依赖，整退根 types.go；但 api 下的四个现役 DTO 子包保留，不能整根 ban prefix。product/CollectSizeMapImageURLs 只有自身测试消费者，随旧图片调用尾部整退，不复制无现役消费的纯 helper。旧市场解析、SPAPI 凭据、Cookie/auth/retry/营销/wire 专属测试随 owner 退出；当前有效平台/安全规则不随之退出。

### 5.2 七个 mixed DTO 文件与一个全保留 DTO 文件

下列每个“保留”指整个原声明（含传递字段/方法/tag）逐字保留；仅删除同文件其它旧声明及失去使用的 import/comment。没有重命名/迁包/序列化替换。

| 文件 | 完整保留声明 | 删除边界 |
| --- | --- | --- |
| api/attribute/interface.go | AttributeInfo、AttributeValue、AttributeValueGroup、CustomAttributeRelation | AttributeAPI、AttributeTemplate/Info、ValidateAttributeResponse、AttributeValueNameMulti、PreAttributeValue、AddCustomAttributeValueRequest/Response |
| api/other/interface.go | FlexibleString 及两方法、SupplierOperateInfo/Response | OtherAPI、SpuLimitCountResponse/InfoWrapper/Info、BatchCheckOnWayResponse、GetUserResponse/UserInfo、ShelfQuotaResponse/Info |
| api/warehouse/interface.go | Warehouse/Response、StoreAddressListRequest、StoreAddress、StoreSiteInfo、StoreAddressListInfo | WarehouseAPI、StoreAddressAddRequest、StoreAddressCheckRequest/Info、CollectionOption、LatLng、CollectionJudgeRecord、ProviderInfo |
| api/product/model.go | Product、Extra、LanguageContent、ProductAttribute、ProductVideo、SiteInfo、SizeAttribute | 仅 ProductListing |
| api/product/cost_price.go | CostPrice | CostPriceQueryRequest/Response、CostPriceInfo、SkcCostData、CostSaleAttribute、SkuCostInfo、SaleAttributeItem、ChangeCostReason、IsCostPriceUnavailable |
| api/product/response.go | BrandItem、BrandListResponse | 其它仅服务旧 Product Client 的响应声明 |
| api/product/shelf.go | SubSite | ShelfOperateRequest/Response/Result、SkcSiteInfo |
| api/product/skc_sku.go | **全文件 KEEP**，含 SKC/SKU、SkcExtra/SkuExtra、SaleAttribute、FieldDisabledInfo、Quantity/Cost/Price/StockInfo、ImageInfo/Detail、SiteDetailImageInfo、DetailImage | 零改，不剪任何 JSON 字段 |

以上路径相对于 internal/shein。它们是当前领域消费者真实合同，不是保活旧 Service 的 compatibility 类别。其它全文件 KEEP 包括 client/session_transport.go、cookie_values.go、所有当前 publishing/marketplace/rules/content/authorizedbrand、sheinlogin、storecenter/official integrations、当前 acquisition/AI/images/身份/计费/schema 等。

### 5.3 十八个其它 mixed 工件与两个新纯值文件

与 §5.2 七 mixed 合计 **25 mixed 工件**：

| 精确文件 | 唯一允许修改 |
| --- | --- |
| internal/shein/client/endpoint.go | 只保留 apiPrefix、getSupplierOperateInfoEndpoint、getWarehousesEndpoint、storeAddressListEndpoint 四常量及原值；退其它旧调用常量/注释 |
| internal/shein/client/endpoints.go | 只保留 GetSupplierOperateInfoEndpoint/GetWarehousesEndpoint/GetStoreAddressListEndpoint 原签名/body；退 GetEndpoint map 与其它 getter |
| internal/shein/client/session_transport_test.go | 仅 §4.3/§6.1 最后旧构造尾部；其余整个测试/helper零改 |
| internal/core/config/type_amazon.go | 仅 AmazonConfig.SPAPI/ConfigPaths 与 SPAPIConfig/MarketplaceConfig/AmazonConfigPaths；其它原字段/type零改 |
| internal/core/config/type_platform.go | 仅 PlatformsConfig.Temu/Shein、PlatformConfig；Alibaba1688 与其所有字段零改 |
| internal/core/config/config.go | knownEnvBindings 仅七 amazon.spapi 键及各自 Primary/Deprecated 数组退出；其它显式/旧别名与加载逻辑零改 |
| internal/core/config/defaults.go | setAmazonDefaults 仅 SPAPI block，退 setTemuDefaults/setSheinDefaults 及两调用；Alibaba/defaults其它原样 |
| internal/core/config/loader.go | NewDefaultConfig 仅 Amazon.SPAPI 与 Platforms.Temu/Shein literal 退出 |
| internal/core/config/loader_builder.go | BuildConfig 仅两个 BuildPlatformConfig 行退出；同文件 BuildHTTPClientConfig 等所有现役 builder零改 |
| internal/core/config/validator_amazon.go | 条件 amazon.Enabled 或 SPAPI.Enabled 改为 amazon.Enabled；仅退末尾 SPAPI 七校验；所有浏览器规则原样 |
| internal/core/config/validator.go | moduleFromField 仅退 amazon.spapi/platforms.temu/platforms.shein 三分支；validationModuleRank 仅退 Amazon SP-API/Platforms.TEMU/Platforms.SHEIN 三 case，**不重排现役 rank 值**；其它聚合/格式化/helper零改 |
| internal/core/config/config_test.go | 仅 §6.1 四原函数 |
| internal/core/config/config_env_test.go | 仅 §6.1 两原函数 |
| internal/core/config/validator_dependencies_test.go | 仅 §6.1 两原函数 |
| config/config-dev.yaml、config/config-prod.yaml、config/config-test.yaml | 仅 §4.2 实际存在的三旧子树和专属注释；其它字节/键值不改，不读私密配置 |
| tests/listingkit_backend_retirement_test.go | 仅 §6.1 三原函数及它们直接调用的本轮局部 AST/缺席 helper；其它函数/global/helper逐字保留 |

新增仅 internal/shein/api/warehouse/address_conversion.go 与 address_conversion_test.go；先测试原纯值合同，再原样迁两函数，不新建平台 adapter/测试 runner。ConfigPaths 是未注入的属性映射路径槽，不是当前 browser profile/root/采集入口；不迁移其未知实际路径或内容。

### 5.4 二十四份直接历史依据的窄顶注

未来实施时只给以下每份原文增加准确 superseding 顶注；原完整正文/原顶注/链接/批准/历史数字/勾选/证据逐字后缀保留。只替代本文矩阵内的旧 API/manager/client/SPAPI/Temu/Shein 包装配置的 KEEP/当前 owner 含义，不 blanket 替代“平台 API”或其它有效能力。

1. docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md
2. docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md
3. docs/architecture/legacy-auto-pricing-retirement-boundary.md
4. docs/architecture/legacy-config-facade-closure-retirement-boundary.md
5. docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
6. docs/architecture/legacy-numeric-store-management-closure-retirement-boundary.md
7. docs/architecture/legacy-product-policy-closure-retirement-boundary.md
8. docs/architecture/legacy-runtime-config-closure-retirement-boundary.md
9. docs/architecture/legacy-sds-runtime-closure-retirement-boundary.md
10. docs/architecture/legacy-shein-activity-retirement-boundary.md
11. docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md
12. docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md
13. docs/architecture/legacy-temu-bulk-relist-retirement-boundary.md
14. docs/architecture/legacy-temu-category-template-retirement-boundary.md
15. docs/architecture/legacy-temu-image-api-prompt-retirement-boundary.md
16. docs/architecture/legacy-temu-image-orchestration-retirement-boundary.md
17. docs/architecture/legacy-temu-image-processing-retirement-boundary.md
18. docs/architecture/legacy-temu-listing-closure-retirement-boundary.md
19. docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md
20. docs/architecture/open-source-replacement-evaluation.md
21. docs/architecture/platform-boundary-strategy.md
22. docs/refactoring/code-health-decisions.md
23. docs/development/repository-structure.md
24. docs/refactoring/module-target-mapping.md

当前本文、Architecture README 与 Legacy Register 三文档同步当前状态/边界，两个索引原历史后缀保留。dependency-baseline、phase2-runtime-inventory、external-client-boundary-inventory 等所有 inventory/历史计数文件**零改**；其历史行不构成重建旧 owner 义务，由本文/当前 Register说明，不静默校准计数。其它仅历史 API 名称不全仓扫改。

实施拟定 **94整退 + 25 mixed + 2纯值文件 + 24窄顶注 + 3当前文档 = 148工件**，同一主要 PR 完成。设计阶段实际修改只有三当前文档。

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: worker仓库地址转换两项纯函数原样抽到同包；其余现役规则/JSON/SessionTransport/Store/采集直接保留原owner
Current owner: 当前shein DTO/warehouse纯值包、sheinlogin/session transport、官方integration/Store与各当前领域owner
Cutover/deletion condition: 完成适用设计复核和用户确认§1–§7/§6.1、记录并读回IMPLEMENTATION_READY/Ready；先抽纯函数再按148工件收齐；原有效检查及保留合同通过；合并另获授权
```

## 6. 实施顺序与必要验证

本轮只做三文档范围、索引/链接/历史后缀检查、原完整设计 PR CI 与适用只读设计复核；设计不执行生产/浏览器/平台路径，也不跑旧 API 专属测试。

获确认后先按 §6.1 扩展原 retirement guard 记录目标仍存在的准确 RED；补任务自有内存纯值 fixture，捕获抽取函数的空值/顺序合同（已有正确行为不伪造失败）。先 EXTRACT 两函数并转换现役 header 测试，再按整闭包删除/配置/原检查/历史顶注收齐至 GREEN。禁止 missing-skip、删除有效断言或为旧测试加替代 adapter。原当前 worker/Store/JSON 源码保持，只有原合成 fixture 作为保留证据，不做环境验收。

必要自检限 core/config 原适用合成测试、现役 session/warehouse 纯值测试、当前 SHEIN DTO/草稿/预览/严格JSON/有效规则、独立登录/Store 原适用合成测试、原 retirement/import guards 与 Architecture README 四项检查。保留原八命令依赖检查；编译 current-application、image-agent-temporal-worker、shein-login-worker、source-account-registry-schema-init、product-listing-api-schema-migrate、listingkit-schema-migrate、1688-local-agent、1688-batch-import、1688-public-browser-collector 及 hack/debug/test-amazon 独立模块，产物仅任务 temp、不执行。不得读取现有 .env/私密配置以运行测试，不建数据库/Redis/真实浏览器/provider环境；原CI隔离fixture不是运行/产品验收。

稳定候选用原完整 CI，不人工重复全量 CI 或加 runner/验收 session。最终严格核对148路径及 mixed允许范围、24原文历史后缀、两个索引后缀、全部 KEEP 和 workflow/inventory/依赖/锁/保护零差异；真实范围外新消费者/CI要改冻结项时，先报告准确原因与补丁，不能自行扩权。

### 6.1 原检查/测试精确转换矩阵

| 文件 / 原函数或项 | 唯一允许转换 | 必须保持 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go / TestCurrentCommandsDoNotCompileLegacyBusinessBackends | 原强制存在列表只去 §5.1 old API/pricing 文件，type_platform.go 仍存在；加准确旧根/子根缺席、SHEIN root api 仅直接Go文件缺席、mixed owner-qualified retired声明禁入及当前必需DTO声明保留；复用 trackedFiles/parser/decodeGoImportPath/importMatchesPrefix 对全 tracked Go（含tests/build tags/nested模块）禁旧 Amazon/TEMU根、SHEIN死子根与pricing根 imports，SHEIN api根只exact match，不禁保留子包；原八命令依赖禁入同样加准确根 | 原命令数/错误、现役DTO/SessionTransport/cookie文件存在、三个endpoint原值、当前worker/Store、type_platform/Alibaba/通用HTTP/Retry、workflow/race/安全原断言；解析/读取错误fail closed，无全仓通用名字禁词 |
| 同文件 / TestListingKitTaskBackendSourcesAreRetired | required map 只去 internal/temu/api 项，替换为同根fail-closed缺席保护（可调用上项同一局部helper） | 当前 marketplace/temu/goods/rules、shared ptr/appenv/store与SHEIN规则/清理各required项、其它原逻辑全保持 |
| 同文件 / TestNumericAcquisitionAndPlatformTailClosureIsRetired | 原两条强读 base_client.go/types.go 的字段禁入转为这两个文件明确缺席；其余不变 | 原数字映射/SourceIdentity/采集/配置禁入及全部tracked import检查保持 |
| internal/shein/client/session_transport_test.go / TestCurrentSessionRetainsProviderEndpointHeaders | 最后 b:=BaseAPIClient 与 h:=b.createHeaders 两行改为 h:=sessionHeaders("https://sellerhub.shein.com",GetSupplierOperateInfoEndpoint(),"self_operated")，仅调窄对应注释 | 原h四项断言、2 origins×3 endpoints的原header/request断言全部保留；另一个TLS/Cookie域/redirect测试、helper全不改 |
| warehouse/client_test.go → address_conversion_test.go | 旧Client HTTP测试随整文件退；把原有效仓库名称/code/count断言作为直接纯函数fixture，补空code/国家去重顺序/Meta.Count/空Data与nil/empty区别 | 不保留旧HTTP client构造；当前 worker 原merchant/warehouse/address只读/fallback与同步测试原样；不发请求或改worker |
| core/config/config_test.go / TestCurrentConfigDoesNotRequireRetiredTaskInputs | 去 PlatformConfig 构造反射，改检查 PlatformsConfig只有Alibaba1688、AmazonConfig无SPAPI/ConfigPaths；原defaults/env禁入列表加amazon.spapi/amazon.configPaths和platforms.temu/shein整前缀 | 原全部禁入/默认Validate断言保持，不加旧键兼容/静默迁移 |
| 同文件 / TestLoadFromBytesRetainsPlatformTransportLoginAndRetry | LoadFromBytes合成YAML改用现役Alibaba1688参数并检查其值；原42s/true HTTP行为改为同测试用显式合成Viper调用未变BuildHTTPClientConfig验证，保留DefaultRetry3 | 通用timeout/TLS输入/Retry合同，不把原TLS断言删掉或移给无效旧PlatformConfig |
| 同文件 / TestAmazonConfigDefaults、TestConfigBuild | 前者仅去五SPAPI默认值断言，absence在上项覆盖；后者只去旧Temu HTTP set/assert（通用行为由上一项保留） | 所有当前browser/Amazon风险/质量/1688/OpenAI断言不变 |
| core/config/config_env_test.go / TestNewViper_BindsPrimaryEnvironmentVariables、TestNewViper_BindsLegacyEnvironmentAliases | 各只删两条SPAPI t.Setenv与两条对应assert | 原AI/image/DB主env、其它alias行为与全部其它函数不变 |
| core/config/validator_dependencies_test.go / TestValidateAmazonConfig_SPAPIRequiresCredentials、TestValidateConfig_CatchesDependencyErrors | 只退第一整函数；第二只去SPAPI literal | 原有效OpenAI/依赖断言，TestValidateConfigPreservesValidationGroupOrder全函数/六聚合/全部format测试不变 |
| 其它原检查 | **零改**：assertLegacyLoggerFacadeRetired、TestSheinListingMixedDeclarationsAreRetired、TestSheinCanonicalDraftContractIsRetained、SKU guard、legacy_hard_cut/import_boundaries全部原函数、.golangci、Architecture检查、所有workflow/inventory | 已批准已退出旧owner不复活；不存在文件的历史allowlist/业务根字符串不要求无价值清扫 |

声明禁入必须限定本文旧owner与receiver，不能让 API/Client/PriceManager 等名字影响现役不同 owner。复用既有 Go AST/helper 检查与局部表，不创建新通用 checker/框架；新增小 helper 只能直接服务三原函数，不得修改其余 helper/global，尤其 logger facade helper零改。

## 7. 准入、评审、交付与停止点

### 7.1 设计交接 / 实施准入

本次提交只有本文、Architecture README 与 Legacy Register；设计原CI/准确SHA/实际review/finding分类留主要PR，不能用前批CI或自动review启动状态冒充通过。适用Independent Architecture复核检查真实消费者/当前owner、前批KEEP替代、两函数EXTRACT、配置公共面、平台与持久化边界及§6.1。完成复核且用户明确确认本文后，先记录并读回 IMPLEMENTATION_READY / Ready，再让唯一Writer在同一PR整批实施。

用户确认将批准上述准确旧API/配置KEEP替代与样例键退出，不授权其它cookie接口退出、未知记录处理或真实环境操作。本阶段合并/部署/#301关闭/实现仍 NOT_AUTHORIZED。#301 OPEN，#29不重开，不加试用前置。

### 7.2 最终交付检查

实现完成后一次集中只读最终检查完整真实diff、148工件/实际mixed范围、当前能力/JSON与SHEIN/Store保留及已有证据；不只采信作者报告。finding先分类，真正命中项目BLOCKER的新事实才重开冻结设计；普通实现或测试问题同PR修复，仅复核相关增量，理论hardening不阻止有界交付。实现/CI/AI复核均不等于用户或运行验收。

### 7.3 评审与授权

同HEAD GitHub review运行时不重复触发；仅明确启动失败后一次受控重试。正式reviews为空、标签或摘要布尔值不算实际PASS。**未经用户本轮授权不默认新增 AI 互验线程**，此前#923设计/源码代理授权不覆盖本轮。备用只读设计或最终源码复核先给准确HEAD和失败证据，获本轮授权后启动。

本轮不创建新执行者/任务、不接管其它owner；不执行平台请求、登录、同步或真实副作用来验收退休。将来合并仍需用户准确HEAD授权与GitHub expected-head正常合并，不能管理员绕过/强推/改保护。只用实际merge SHA对应的event=push/head_branch=main整run completed/SUCCESS确认main CI，不用PR CI或单Gate代替。环境UNKNOWN，运行/产品验收NOT_RUN。

