# 旧 TEMU SPU 构建与最高零售价查询编排退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户在当前会话“确认设计”，批准冻结 `2b151ffec1af2866c7d8031889a10a3a3f8ddbe3` 的 §5/§6/§6.1/§7；七项结构检查已抽到既有规则 owner，四目标生产文件及专属 helper 已退休。合并和真实操作仍须单独授权。
- 调查基线：main **28d2305233f272609da80c17a5e0462d681dd9fc**，已包含 #760；实施开工同步 actual main **a59131ba782504e1b994aa6b9f306864d27c8b3d** 无增量。准确实现 HEAD、CI、独立最终检查维护于主要 PR；旧设计证据仅历史，不替代实施检查。
- 当前 **Design Basis: Reuse Existing Architecture**：用户确认后，生产修改前已在 #29/#301 登记冻结依据、原 owner/不变量与 **Ready**，再进入实施。原设计期 N/A 只覆盖当时文档/只读映射。
- 保留当前采集、Store 观察、有效平台 API/同名 DTO、有效规则/共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、Reader/Cache/Mapping、AIBatchInput/SKU 映射、独立 SHEIN 登录 worker 及其 Store sync。两套及其它 workflow、必需/安全检查、保护/ceiling 不改，不操作实例或数据，不增加试用前置。

## 1. 用户结果与范围

使用者为维护当前内部试用、采集及 Store 能力的维护者。交付为值规则归已有 TEMU publishing、无装配旧 SPU/零售价查询编排的源码退休，按本矩阵保留原规则和事实/调用 owner，防止旧 Task 编排被误接为当前执行能力。当前用户入口不变，不将本维护作为试用前置。

删除前范围是 internal/temu/product 下的 build_spu_handler.go（128行）、spu_builder.go（431行）、spu_validator.go（156行）、price_query_handler.go（212行），共 **4 个生产文件 / 927 行**及文件内部独占 helper；没有调用这些类型的专属测试。已先抽取一项已有 SPU 结构值校验，再完整退休四个旧 owner。product 包及其余保留实现仍在。

**PriceHandler 不是专属闭包**：sku/builder.go 构造它，sku/item_builder.go 借它计算售价和默认库存；本次保留原文件、constructor、全部方法、ProfitRuleAPI、SKU 消费者和原行为。API 查询能力、AI/属性/文本/规格/SKU 实现、mixed context 同样不随旧 caller 退出。

非目标：新增当前刊登/AI 执行入口、Product factory、平台默认值、价格/库存策略、IAM/cache、状态机、表/schema、公共合同、provider/runner/验收工具；运行、恢复、重放或清理历史状态；新增试用或生产前置。

## 2. Design Basis、Authority 与不变量

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：仅 EXTRACT/RETIRE，无 compatibility、旧 ID 映射、wrapper/fallback、双读双写或第二事实源。
- 已确认的 [TEMU 保存/提交](legacy-temu-listing-submit-retirement-boundary.md)、[前置](legacy-temu-preflight-retirement-boundary.md)、[变体](legacy-temu-variant-retirement-boundary.md)、[筛选](legacy-temu-filter-mapping-retirement-boundary.md)、[Store/初始化](legacy-temu-store-init-retirement-boundary.md)：旧 Task 入口退出，当前规则与共享合同按 owner 保留；既有 marketplace 私有值函数的抽取方式继续适用。
- [TEMU publishing](../../internal/marketplace/temu/publishing/README.md)：TEMU 校验与 payload 规则的既有 owner；私有值快照/纯函数不依赖旧 Task/API/状态，不新增执行入口。
- [当前 Store](store-center-current-application-v1.md)、[观察](store-center-platform-observations-v1.md)、[独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：当前 Organization Store、保留数字 Store、凭据/Profile、同步及恢复各归原 owner，不互相替代。Figma N/A，无 UI/IA 修改。

实施采用 **Reuse Existing Architecture**：用户确认矩阵后，生产修改前已在 #29/#301 登记冻结设计、确认、用户结果、上述引用、原 owner/调用、验证、不变量、唯一 Writer 与 Ready，再实施 §5。抽取仅将原已执行的七项值检查交给已有 TEMU 规则 owner，复用现有私有 submissionSpec；不新增公共领域合同、权限/身份/tenant、canonical fact、表/schema、状态机/事务、幂等、retry/UNKNOWN/recovery 或副作用 owner。若实际需要改变这些边界，停止对应部分并补适用 Independent Architecture 准入。

Must：完整保留 §5 明确能力及原规则行为，原防线不减。Should：源码、原 guard、当前清单和直接冲突说明一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删有效 SKU/Price/API/共享 owner，将旧 AI 降级、默认值或 raw payload 解释为当前合同，复活旧入口，或误报真实运行态清零；不扩展为新 IAM/预算/通用恢复系统。

## 3. 实际消费者、注入与调用

删除前全仓 Go 搜索并在实施开工刷新：两个入口 NewBuildSpuHandler/NewPriceQueryHandler 无外部构造、直接实例化或注册；NewSpuBuilder/NewSpuValidator 只在 BuildSpuHandler 构造链内使用。四种 TEMU 类型只出现在上述四文件。下表、§4/§4.1 的退休对象行为是删除前取证；保留实现不变。SHEIN 同名 BuildSpuHandler 是另一实现，不在本范围。

go list ./cmd/... 共 **31 个当前命令**；依赖并集中的旧 temu/product、temu/sku、temu/context、temu/handlerbase 均为 **0**。product 包仍由 SKU 两文件导入并消费 PriceHandler，不能由命令静态图推定整个包或其共享行为可删。未运行 main 或 constructor；静态图不是实例/用户验收。

| 合同 → 实现 | 注入与消费 | 本批处置 |
| --- | --- | --- |
| ChatCompleter / ProfitRuleAPI / SkuBuilder / SpecHandler → BuildSpuHandler | 旧入口创建 SpuBuilder/SpuValidator；basic/extension 后并行 AI 重写和 SKU 构建，再 service/sale、校验与摘要 | 退出旧编排，不将接口名当当前 provider/计量/恢复准入 |
| SpuBuilder | 调 TextProcessor、skugen、PropertyMapper、注入的 SKU/Spec；写 TemuProduct 标题/描述/属性/规格/SKC/固定字段。priceHandler/openaiClient 字段仅构造赋值，没有 receiver 使用 | 退休聚合器/helper；原文本、编码、属性、SKU、规格和 PriceHandler 保留，不误称该字段调用了利润 API |
| SpuValidator | 唯一构造方是旧 BuildSpuHandler；按层次检查值，再统计价格/库存并写日志 | 七项已有结构检查按 §4.1 抽成私有值规则；Task/context/原 DTO/receiver、统计摘要退休 |
| PriceQueryHandler | 旧 Task/Product/GoodsID 门控 → 收集非删除 SKU 价格/币种并去重 → NewProductAPI.QueryMaxRetailPrice → 内存回填 SKU MaxRetailPrice 及 PriceQueryResponse | 退出旧查询装配/回填，不删 API/DTO/re-export 或 mixed context |
| PriceHandler / SKU | NewSkuBuilder → NewPriceHandler；SkuItemBuilder 调 CalculateVariantPriceWithRuntime、GetDefaultStock/WithRuntime | 全部保留；它不是上述入口独占的实现 |

## 4. 持久化、调用效果与失败 owner

| 边界 | 实际 owner / 效果 | 删除/保留约束 |
| --- | --- | --- |
| 当前 Product/Store/Observation/Submission | 当前领域、Organization 授权及原数据库/工作流 owner | 四文件没有这些事实的 durable owner；不把 TemuProduct 或查询结果迁入当前 Product/Store，不新建保存/同步流程 |
| 旧 SPU 值与 PriceQueryResponse | TemuTaskContext/TemuProduct 为借用的进程值；本四文件没有 DB 事务、outbox、幂等键或恢复 receipt | 不将内存回填描述成结果落库。mixed PublishState/PriceQueryResponse、TemuProduct/Setter/StoreInfo、SKU runtime/映射及 API 同名 DTO 原样保留 |
| AI 重写 | BuildSpuHandler → 既有 AIContentRewriter → ChatCompleter.CreateChatCompletion，可能发生外部请求/费用；响应写标题/描述/要点 | 只退出旧 caller；AI/helper、provider port、prompt/logger/计量/预算及原调用 owner 不改，不调用 provider。保留代码不是当前预算/计量或真实执行验收 PASS |
| 属性/SKU/图像/规格 | PropertyMapper、注入的 SKU/Spec owner；SKU 路径可查询平台规格、上传图像及维护借用的 ASIN/SKU map | 保留所有实现/接口/原副作用边界，不调用或顺带退休传递依赖，不把内存 map 误称 durable receipt |
| 最高零售价 API | api/product.API.QueryMaxRetailPrice 经 ClientAPI.SendTEMURequest 向 /mms/marigold/price/retail/max/info 发 POST；API 检查 request/Success 并返回原 DTO | 原 endpoint、请求响应、API/api.go re-export、client/鉴权/错误能力全保留。这里只识别源码中的查询调用，不据名称推定可重试或实际平台效果，不访问平台 |
| Store/利润/价格/库存 | listingadmin Store/ProfitRuleAPI、APIClient/Cookie、PriceHandler/SKU 与原 PriceManager/PriceCalculator | 原身份/租户/凭据、Store 持久化及计算行为不动。不把随机库存、nil runtime 默认 tenant/store=1 或旧倍数升格为当前规则；已有身份限制见 #760，保留不等于授权 PASS |

架构问题需明确：旧 BuildSpuHandler 让 AI 和 SKU 共用可变 TaskContext，AI 重写失败返回 nil/警告，属性失败也可只记录日志。它没有当前 durable 效果判定、失败/响应丢失、取消、重试或 UNKNOWN 协议；不能以名称“构建/查询”把它复用为当前可信执行链。这里不宣称已发生竞态或账务事故。当前无装配消费者，分类 **NOT_APPLICABLE（当前用户路径/试用阻塞）**，通过 RETIRE 退出该编排，不修旧链或新建恢复平台；共享 AI/文本/属性/SKU 的既有实现仍保留。

源码退休不判断历史请求是否成功、不释放任何预占/锁、不恢复任务、不读写真实实例或数据。发现具体当前消费者、可观察兼容义务或当前 Must 后果时，停止对应删除并报告，不能自动加兼容路径。

### 4.1 有效规则与旧 payload 行为逐项处置

| 原行为 | 决策 / 当前 owner | 精确限制 |
| --- | --- | --- |
| GoodsName 非空、SKC 列表非空、每 SKC 的 SKU 列表非空、每 SKU 的规格列表非空、每规格 SpecID/ParentSpecID/SpecName 非空（七项） | **EXTRACT** 已实现的确定性值检查到既有 marketplace/temu/publishing；不保留 SpuValidator owner | 已复用私有 submissionSpec，私有函数 validateSPUStructure(name string, skcs [][][]submissionSpec) error。仅按名称→SKC→SKU→规格和原索引顺序返回首个失败；不修改输入，不新增公开 DTO/API、Task、raw Product、模型或 logger 依赖 |
| 上述检查的边界 | 保留原精确空字符串/len 语义；原本未过滤任何 SKU，不能新增删除项过滤或 trim | 空白名称/字段、非空字符串0或TEMP_不是该规则原本拒绝的值；不凭提取增加要求。原价格/库存正数检查是注释，不能启用为新门槛。旧日志/错误字面不是公共合同，保留可诊断的失败类别/索引 |
| 标题/描述/要点清洗、外部编码生成 | 原 TextProcessor/rules、marketplace productpolicy/publishing、skugen 及其测试/消费者原位保留 | RETIRE SpuBuilder 的旧 Amazon→raw payload 装配；不复制规则或保留 builder wrapper |
| TEMP_ 检查、规格解析/有效性 | 现有 SKU item/spec handler 和 SpecResolverService 原样保留 | RETIRE builder 中的重复检查，不新增 mapper/规格身份 owner；纯结构规则不冒充规格解析或覆盖校验 |
| 商品规格属性聚合、vid/template 查找、固定 feature/Checked/IsCustomized 等 raw 字段 | RETIRE 专属聚合/helper，保留平台 DTO、模板/Spec/SKU owner 及既有覆盖/VID 纯校验 | 不把旧字段默认、map 顺序、Number Of Products/pcs 特判或其大小写缺陷提升为当前规则；不借此修旧 payload |
| en/站点100、美国原产地、默认描述/要点、ShipmentLimitSecond=2 等 | RETIRE 旧组合/赋值；保留字段/API和其它有效规则 | 同数值发货注释与单位在 #760 已披露不一致，不根据注释发明当前平台规则 |
| 价格/库存摘要、查询价格去重和回填 | RETIRE 四文件内专属 helper | 不提取旧非零字符串筛选、浮点乘100截断、币种拼 key/忽略解析失败为新货币规则；原平台价格 API/DTO、PriceHandler 和定价 owner 保留 |

抽取是保留现有可验证规则的代码归属，不新增当前提交入口或用户操作前置；沿用 #749/#750 已批准的私有值函数方式，不保留旧 owner/适配器。若实施发现需要新规则、公开合同或新消费者，先报告并补相应准入。

## 5. 精确抽取、删除与保留矩阵

以下是**用户确认后已实施的精确矩阵**。

| 对象 | 决策 | 删除/保留条件 |
| --- | --- | --- |
| product/spu_validator.go | 先按 §4.1 EXTRACT 值校验，再 RETIRE 全文件 | 无旧 DTO adapter、receiver、context 参数/摘要/价格库存统计或 alias 留壳；一项纯规则保留七项检查 |
| product/build_spu_handler.go | RETIRE 全文件 | 类型/constructor/并行编排/AI caller/专属方法全部退出，不退休 AIContentRewriter/helper/provider |
| product/spu_builder.go | RETIRE 全文件 | constructor/字段及 basic/extension/SKU/service/sale、编号/文本/规格聚合/template/helper全部退出；原 PriceHandler、TextProcessor/skugen/property/SKU/Spec 保留 |
| product/price_query_handler.go | RETIRE 全文件 | Task 门控、API caller、收集/去重/回填/日志/helper退出；QueryMaxRetailPrice及同名DTO/API工厂/re-export保留 |
| marketplace/temu/publishing/spu_validation_policy.go 及对应测试 | EXTRACT 一项私有值校验 | 复用现有 submissionSpec，不导入旧 temu、model、pipeline、provider 或 Task/API DTO，不新增正式入口/状态/公共合同 |
| product/price_handler.go、SKU 两消费者及 ProfitRuleAPI/DTO/仓库 | 保留全部原实现 | 不因 builder 有一处构造就误删；原售价/库存/倍数/Store 配置/tenant/owner行为不变，不新增使用旧默认身份的消费者 |
| product 其余 name/description/brand 规则、marketplace publishing/productpolicy、skugen、AI/property/rules/image/spec/template/category/SKU | 保留全部原实现与有效测试 | 不扩大专属闭包或修“顺带发现”的旧逻辑，不改 provider/平台/图像能力 |
| mixed context/PublishState/PriceQueryResponse/TemuProduct/StoreInfo/SKU runtime/映射、handlerbase SkuBuilder/SpecHandler | 保留原合同 | 本轮只退四个 owner，不删除/改写混合字段或公共接口，不为保留合同新增 fallback 或消费者 |
| 平台 API/所有同名 DTO、SHEIN 同名 BuildSpuHandler、PriceManager/PriceCalculator及此前约定能力 | 保留原 owner | 不全仓封禁同名类型，当前采集/Store观察和独立SHEIN登录/Store sync不变 |

Legacy decision: **EXTRACT 一项有效值校验；RETIRE 四个旧 owner 及专属 helper**。
Reusable behavior: 七项既有结构约束进入现有 TEMU publishing；其它有效规则/API/工具原 owner 保留。
Current owner: marketplace TEMU 规则、原 PriceHandler/SKU、平台 API/client、Product/Store/观察/独立 worker 各保持。
Cutover/deletion condition: 用户确认 §5/§6/§6.1/§7、Issue Reuse Existing Architecture / Ready；刷新消费者，先纯规则与原退休 guard 的 RED→GREEN，再精确删除、登记校准、保留回归及准确实现 HEAD CI/最终检查。源码退出不授权运行态退出。

## 6. 验证与交付

设计阶段只提交本文和架构索引，旧设计证据不是实施或产品验收 PASS。用户确认后已完成纯值规则 TDD RED→GREEN；原退休 guard 在四文件仍存在时准确 RED，再精确删除后 GREEN，保留断言无失败。下列是实施验证范围，准确实现 HEAD 的结果和最终独立检查维护于 PR，不人工重复完整 CI。

用户确认后的最小实施验证：

1. 刷新四种类型/constructor/直接实例化、全部 helper及当前命令依赖；发现新消费者或当前兼容义务停止对应删除。
2. 值规则 TDD：七类拒绝及稳定首错/索引、nil/空嵌套列表、通过的完整输入、精确空白/非空0/TEMP_、输入不变；先 RED，再最小纯实现 GREEN。原未启用的价格/库存条件不增补，既有覆盖/VID/SKU值规则测试保留。
3. 原 TestListingKitTaskBackendSourcesAreRetired 增加四文件缺席和**仅限 TEMU product**的八个准确类型/constructor AST 禁令，先在原源码存在时 RED，再精确删除 GREEN；不全仓封禁 SHEIN 同名或 raw API。
4. 同一原 guard 补保留 PriceHandler/相关方法、SKU 两消费者、ProfitRuleAPI、QueryMaxRetailPrice/API DTO/re-export、mixed context、现有规则与 SHEIN 同名合同断言；原全部断言保留。
5. 原 product import/OpenAI 防线及 consumer register/ceiling、provider constructor/logger/LLM guards；既有 marketplace 规则、product 规则和 SKU相关包编译/适用回归；current-application、独立 shein-login-worker 只编译。保持任务风险匹配，不增 runner/真实环境验收。

### 6.1 原 guard、登记、文档与 workflow 精确处置

| 工件 | 本次改动 | 保留边界 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 按 §6 精确增加四文件/八符号、纯规则/保留合同断言 | product 包与原命令图 guard 保留，不禁止整包，不新建通用扫描器 |
| TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked | 只移除 build_spu_handler.go、spu_builder.go 两条对应豁免；保留 price_handler.go | 稳定父目录索引、原 product/store 两子树、management deny和其它扫描不变，无 skip/helper 放宽 |
| TestTEMUOpenAIImportsStayAllowlisted | 只移除同两文件的对应 OpenAI 豁免 | 原 root/扫描/deny和AI/image/SKU等其它条目不变；不顺带清理无关历史豁免 |
| phase2-runtime-inventory.md 当前 consumer register | 四文件已删除，只移除 integration/openai → temu/product 一行 | core/logger → temu/product 仍有 PriceHandler/name/description/brand 消费，保留；其它行、dated inventory/closure ceiling不动 |
| code-health-decisions.md 的 product 行 | 仅校准本次四文件/专属编排已有退休授权、有效值规则抽取及保留 PriceHandler/其它规则；原20 declarations/dated报告保留 | 不把 product 整包标 removed；旧 configured processor 依据不再作为这四文件的保留权威，当前只使用仍存在的包检查目标 |
| external-client-boundary-inventory.md 的当前 TEMU product/OpenAI/guard 描述 | 注明旧SPU/query caller退出，product的直接OpenAI imports退出；management product仍有保留PriceHandler，原三条product豁免变一条 | 只校准该当前热点及准确guard结构；不重写dated记录、其它树或把原API/AI模块宣称退休 |
| Legacy register、本文/架构索引、TEMU publishing README | 现注明 source-retired和私有纯值规则owner | 历史保留矩阵是当批边界，不能恢复这四旧owner；其余有效共享能力不变 |
| 两套及其它 workflow、.golangci.yml、依赖、必需/安全检查/保护/ceiling | **零修改** | 不减少 job/目标/检测范围；provider constructor/logger/LLM inventory 不因删除借用port的caller而手工降低 |

product 目录仍存在，原逐目录 alias/guard无需做缺失目录适配；#760 的 store 两条原扫描及稳定父目录处置保持。若 CI 发现其它同范围直接依赖精确目标的原断言，先定位/分类并核对 sibling，保留有效防线，按最小缺口收敛；不删 gate 或加通用 skip。

## 7. 确认与实施停止点

1. #768 先以 Draft PR 交付设计、准确设计 HEAD CI和独立检查；#29/#301保持OPEN，旧正文历史保留。当前已继续实施，旧设计证据不替代实施结果。
2. 用户已明确确认冻结 §5、§6、§6.1、§7，在同一分支/PR继续；生产修改前已登记 Reuse Existing Architecture / Ready及唯一Writer/不变量。设计确认授权上述有界实施，不代替最终准确HEAD合并授权。
3. 开工同步一次实际 main，最终候选前再次核对，只处理真实组合风险；有新消费者/具体兼容义务、新规则/公共合同、权限/持久化/状态/副作用边界变化时停止对应项报告，不扩删或补wrapper。
4. 纯规则/原guard RED→GREEN、精确抽取删除/登记校准、保留回归、准确实现HEAD CI/最终检查后交付 In Review；不用设计CI/review冒充实现证据。
5. 合并仍需用户准确最终HEAD单独授权和正常expected-head/保护。设计及后续实施都不授权Issue关闭、部署、真实provider/实例/凭据/数据操作，不增加试用前置；产品/平台/实例与用户验收分别保留 NOT_RUN。
