# 旧 TEMU 类目与模板查询编排退休边界设计

## 1. 状态、用户结果与范围

- 日期：2026-10-18；执行依据：#29 / #301 与用户本轮明确授权。
- 状态：**USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #773 冻结设计 `a451f0cbd7b5bd3b6a4dabb027fa8843a544b4af` 的 §5/§6/§6.1/§7；五个 handler 及两个专属 DTO/helper 文件已按矩阵退出。合并和真实操作仍须单独授权，原设计 CI/review 不替代实施证据。
- 当前 **Design Basis: Reuse Existing Architecture；Admission: Ready**。用户确认后，生产修改前已在 #29/#301 登记冻结依据、owner/不变量/验证并读回 Ready。原设计 N/A 仅覆盖当时文档/只读映射；准确实现 HEAD、CI 与最终独立检查维护于 PR，不自行签发用户验收。
- 取证基线：actual main `5e738e3ea69ad7a940a67dc221f67e1f88f7ba23`，含 #771 已合并 SKU/SKC 退休；之后 #772 只改 UI/产品说明，与本范围无交集。唯一 Writer 复用原附属独占 worktree，新分支 `codex/issue29-temu-category-template-retirement-design`；已核对接单及现有 PR，无同范围 owner，不接管 #692。

使用者是维护当前内部试用、采集及 Store 能力的维护者。交付为五个旧 handler 和专属输入输出 DTO 的实际消费者、有效规则、持久化与平台副作用 owner、删除/保留/检查矩阵，避免旧 Task/raw payload 编排再次成为当前入口。**当前没有用户操作被本维护阻塞；设计与后续源码退休均不增加内部试用前置。**

已退休整文件范围为 `internal/temu/category` 三个 handler 与 `internal/temu/template` 两个 handler、两个 DTO/helper 文件，共七文件 872 行。没有新增 EXTRACT 代码；有效规则复用当前 owner。没有新页面/交互，Figma N/A。

非目标：新增刊登或类目推荐产品、默认发货/运费/儿童类目策略、API 接线、AI 调用、公共领域合同、schema/身份/恢复平台或验收工具；修复其它保留实现；运行 constructor/main、真实平台/API/provider/采集、上传/Query/Preview/DryRun；读取配置/凭据/DB/Redis/Profile/历史运行日志；部署、重放、迁移、清理及 Issue 关闭。

## 2. Design Basis、当前 owner 与不变量

Architecture / Contract：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：只允许 EXTRACT / RETIRE；不新增 wrapper、fallback、旧 ID 映射、双读双写或第二事实源。
- [TEMU 当前平台设计](temu-platform-v1.md) 的当前店铺/模式/官方规则快照与明确商品事实合同；实现 owner 为 `internal/marketplace/temu/goods`、`rules`、`model` 及既有官方执行/Submission。当前选择与校验不依赖这些 seller 网页 Task handler。
- 此前用户确认的 [保存/提交](legacy-temu-listing-submit-retirement-boundary.md)、[前置](legacy-temu-preflight-retirement-boundary.md)、[变体](legacy-temu-variant-retirement-boundary.md)、[筛选/映射](legacy-temu-filter-mapping-retirement-boundary.md)、[Store 初始化](legacy-temu-store-init-retirement-boundary.md)、[SPU/价格查询](legacy-temu-spu-price-query-retirement-boundary.md)、[SKU/SKC](legacy-temu-sku-skc-retirement-boundary.md)：旧入口已退出，平台与共享能力保留。旧文档中 category/template 的保留陈述是当批边界；用户确认本文后，仅 §5 七文件由本矩阵替代，平台 DTO、其它保留项及历史证据不被覆盖。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份与持久化](shein-worker-identity-persistence-boundary.md)：当前与保留数字运行链各守原 owner，不从旧账号/Cookie/Profile 补当前身份，不操作运行资产。

本次实施采用 **Reuse Existing Architecture**：七文件只退出无装配的旧编排及专属进程值；有效校验已有当前 owner，平台 API/DTO 与共享上下文不变。不新增或改变公共领域合同、事实/schema、身份/权限/tenant、状态机/事务、幂等/retry/UNKNOWN/recovery、AI 准入/预算/计量或平台副作用 owner。七生产文件 872 行删除，加原 guard/清单/直接冲突说明，未命中 30 文件或 2,500 生产行提示线；若实际需要改变上述边界，停止对应部分并补适用 Independent Architecture 准入。

Must：保留平台 API/同名 DTO/re-export、有效规则/共享合同、AI/SKU 映射与 AIBatchInput、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、两种 SHEIN 属性、Reader/Cache/Mapping、当前采集/Store 观察及独立 SHEIN 登录 worker/Store sync；保留 AI、规格、图像、属性、文本和其它有效实现。两套及其它 workflow、必需/安全检查、保护/ceiling、依赖不改。

Should：原 guard、当前清单及直接冲突说明与源码一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限于误删平台/共享合同、旧默认值被提升为当前规则、静态无消费者被误报为运行资产清零；不扩展为新的 IAM、重试、验收或规则平台。

## 3. 实际消费者与调用闭包

在上述基线，全仓 Go 精确导入搜索未发现 `task-processor/internal/temu/category` 或 `task-processor/internal/temu/template` 的导入。五个 handler/constructor、两专属 DTO 的引用只在各自候选包内；没有外部直接实例化或注册。`.github`、scripts、config、cmd 的源码/声明检索没有发现动态名称装配。`go list ./cmd/...` 得到 **31 个当前命令**，依赖并集内两个候选包均为 **0**；只求静态图，未运行命令 main。这些证据不证明历史实例、数据、日志或远端资产已清零。

| 原 contract → implementation → injection → consumer | 实际行为 | 设计决定 |
| --- | --- | --- |
| CategoryHandler.Handle(pipeline.TaskContext) → HandleTemu | 要求 GetTask/TemuProduct/CatID；写四个 false flag，按 IsClothes/IsBooks 写发货秒数和 FulfillmentType=1 | 无外部注入/消费者；RETIRE 旧 Task 校验和预填，不修改平台 Product/ServicePromise |
| CategoryRecommendHandler → api.NewCategoryAPI(借用 APIClient) → Recommend | 商品名取 Amazon.Title 或 raw GoodsName；前三推荐按关键词排除儿童类目，按非零层级拼 CatIDs 并写 CategoryTree/CatID | RETIRE 编排、专属 keyword/helper；原 Category API/DTO 保留，当前官方类目路径规则见 §4.1 |
| CategoryDisclaimHandler.Handle → HandleTemu → CategoryAPI.GetDisclaimer | 读 CatID，复制 PromptList 到 raw CategoryDisclaimer；缺 client/类目可跳过，查询错误被警告后返回 nil | RETIRE 此宽松编排；不把跳过/吞错搬成当前准入，平台声明合同及提示值类型保留 |
| TemplateQueryHandler → buildTemplateQueryInput/ToRequest → 借用 QueryAPI.QueryTemplateAdvanced → build/applyTemplateQueryOutput | ClickType=8，复制 commit/goods/category/version；将响应模板和三项规格输入写回 TemplateState；缺 client 返回 nil | RETIRE handler 与两专属 DTO/helper；平台两套 TemplateQuery 请求/响应及 mixed TemplateState 保留 |
| CostTemplateHandler.Handle → HandleTemu → api.NewQueryAPI(借用 APIClient) → QueryCostTemplate | ClickType=8/version=1/QueryAll=true；先选可用默认模板，再首个可用，最后所有禁用也强选首项；写 raw CostTemplateID | RETIRE 编排、选择 helper/默认值；当前明确运费选择/官方验证及 API 合同保留 |

架构问题：这些 receiver 把旧 Task 前置、借用 raw payload、平台查询、自动选择/默认赋值和宽松失败处理混在一起。尤其“全部禁用仍强选”与当前明确选择有效模板的合同不一致，固定发货承诺也不能代替当次模式/店铺/类目要求。当前无装配路径，此源码风险对当前试用阻塞归 **NOT_APPLICABLE**；通过退出专属编排收敛，不修复无消费者旧链、不宣称已有事故，也不以本设计开放当前发布。

## 4. 持久化、身份与平台副作用 owner

| 边界 | 实际 owner / 效果 | 本批处理 |
| --- | --- | --- |
| 当前 Product/Asset、Target/CompletionDraft/Submission、Store/Observation | 原当前领域 owner；保存、权限、发送/UNKNOWN 和恢复各有原合同 | 无调用切换或新持久化，不把旧 raw Product 或 TemplateState 作为第二事实源 |
| TemuTaskContext.TemuProduct / GoodsBasic / GoodsServicePromise | 七文件只借用进程内 DTO，写 CatIDs/CategoryTree/CatID/CategoryDisclaimer/flags/服务承诺/CostTemplateID；没有专属 DB、Redis、文件保存或恢复入口 | 删除 writer caller，不删共享 DTO/字段/SetTemuProduct；不读取、迁移、删除真实记录或历史 payload |
| TemplateState 四字段 | TemplateInfo/UserInputParentSpecList 仍被 SKU AI input、属性映射/编排消费；InputMaxSpecNum/SingleSpecValueNum 同属此前保留合同 | 四字段与 RuntimeState.QueryAPI、SetAPIClients、相关平台类型全部保留；writer 消失不授权删共享状态或补新 producer |
| 类目推荐 / 免责声明 | `internal/temu/api/category.API` 经借用 ClientAPI.SendTEMURequest，原 POST 路径 `/mms/marigold/category/recommend` 与 `/mms/marigold/category/query_disclaim` | 只退出 handler caller；API、NewAPI/原 re-export、鉴权/响应错误 owner 原样保留 |
| 模板 / 成本模板查询 | `internal/temu/api/query.API` 的 QueryTemplateAdvanced / QueryCostTemplate 经原 postQuery；POST `/mms/marigold/query/commit/query_template`、`.../query_cost_template` | 名称含 Query 不视为无副作用或可安全重试证明；不调用端点，不改变 API/超时/鉴权/retry/error owner，不新增查询/重放或 durable receipt |
| 借用 APIClient / Cookie / MallID / Store runtime | 原客户端、鉴权/Cookie、数字 Store 仓库及其既有 tenant/失败处理 owner；handler 创建的只是 API wrapper | 全部保留，不改凭据/身份/租户、client factory、Store ID 同步或恢复，不运行 constructor |
| 独立 SHEIN 登录与 Store sync、当前采集/观察 | 原 worker / Store / acquisition / observations owner | 源码与运行能力保持；同名 SHEIN childrenRelatedKeywords、类目/模板实现不在闭包中 |

### 4.1 有效行为与旧策略逐项处置

本批**没有需要新增的 EXTRACT 代码**。下列当前约束已有正确 owner，保持实现与测试；七文件的 Task DTO 搬运和旧选择策略不因“纯值”就复制成闲置 helper。

| 行为 | 当前依据 / owner | 决定 |
| --- | --- | --- |
| 类目非空、选择与商品一致、层级/末级正确 | marketplace TEMU `goods.Build/checkLocal`、`checkSupplierCategory` 与 RuleSnapshot/官方类目模型校验当前同店铺、模式、完整路径及可选末级 | 原位保留全部当前检查；不以旧 CatID!=0、跳过层级拼接或推荐首项替代/放宽，也不复制第二份规则 |
| 备货时间、履约与成本/运费模板 | 当前 goods 的明确备货数值、当次官方模板归属检查和既有 supplier 规则；当前设计不授权旧网页默认值 | 原位保留；RETIRE 固定 86400/172800/259200、FulfillmentType=1、自动默认/首项/禁用项选择；数字与注释均不作为当前平台规格权威 |
| 当前属性/规格模板的完整性、类目关联、已选值与当前规则 | 当前 goods/rules、平台 model；旧 AI/SKU/Spec/属性仍消费共享 TemplateInfo/UserInputParentSpecList | 原位保留这些规则和全部共享类型；只退七文件的 Task request/response 搬运，不抽新 mapper 或删除 TemplateState |
| 商品已有类目即跳过、Amazon 名称回退、前三项限制、儿童名称子串 heuristics、稀疏层级拼接 | 无当前产品合同要求用这些旧推荐策略替代当次官方可选类目、资质和用户明确资料 | RETIRE 该 handler 专属策略/childrenRelatedKeywords；不撤销官方限制、不修改当前规则或 SHEIN 同名规则，不宣称推荐结果代表资格 |
| 声明 PromptList 转写、ClickType/QueryAll/commit version 默认、缺 client/类目跳过/吞错 | 旧 Task transport/预填，平台 DTO/API 与当前明确事实/失败关闭合同各有 owner | RETIRE 专属 wrapper/值搬运及宽松分支；保留原 API/平台模型，不复制默认值为当前领域要求 |

## 5. 精确删除与保留矩阵（用户已确认并实施）

| 路径 / 对象 | 决定 | 删除边界 / 保留条件 |
| --- | --- | --- |
| category/handler.go（124 行） | RETIRE 全文件 | CategoryHandler/constructor、Handle/HandleTemu、processCategoryInfo、两个预填 helper 全退 |
| category/recommend_handler.go（293） | RETIRE 全文件 | CategoryRecommendHandler/constructor、Name/HandleTemu、recommendCategory、isChildrenRelatedCategory/selectNonChildrenCategory、此包 childrenRelatedKeywords 全退；不匹配 SHEIN 同名声明 |
| category/disclaim_handler.go（109） | RETIRE 全文件 | CategoryDisclaimHandler/constructor、Task 适配/HandleTemu/getCategoryDisclaimer 全退 |
| template/query_handler.go（74） | RETIRE 全文件 | TemplateQueryHandler/constructor、Name/HandleTemu/queryTemplate 全退 |
| template/cost_handler.go（185） | RETIRE 全文件 | CostTemplateHandler/constructor、Name/Handle/HandleTemu/buildCostTemplateRequest/queryCostTemplate/extractCostTemplateID 全退 |
| template/query_input.go（49）、query_output.go（38） | RETIRE 两全文件 | 仅专属 TemplateQueryInput/ToRequest、buildTemplateQueryInput、TemplateQueryOutput、buildTemplateQueryOutput/applyTemplateQueryOutput；不保留 alias、空 receiver 或新 adapter |
| `internal/temu/api/category`、`api/query`、`api/template`、`api/api.go` | 原样保留 | RecommendRequest/Response/Result/Category、DisclaimRequest/Response/Result/DisclaimerDTO、query 与 template 各自 TemplateQueryRequest/Response/Result/TemplateInfo/UserInputParentSpec、CostTemplateRequest/Response/Result/Item、API/QueryAPI 及原 re-export/constructor 保留；不能按近似类型名删除 |
| `api/product`、mixed context/handlerbase/pipeline/shared | 原样保留 | Product/GoodsBasicInfo/CategoryTree/Disclaimer/ServicePromise、所有共享字段、TemplateState 四字段、QueryAPI/SetAPIClients、GetTask/GetContext 及其它消费者不动 |
| marketplace TEMU goods/rules/model/publishing/productpolicy 与已有有效测试 | 原样保留 | 当前类目/模板/备货/规格等正确合同及此前抽取规则保持，无新门槛/默认值/额外 API |
| AI/SKU 映射、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、Spec/图像/属性/文本/平台 | 原样保留 | 八个 ai_mapping*、AIBatchInput/receiver、共享映射与其它当前/保留能力不在专属闭包中 |
| 当前采集/Store 观察、Reader/Cache/Mapping、两 SHEIN 属性、独立 SHEIN 登录 worker/Store sync | 原样保留 | 身份/tenant/持久化/恢复 owner 不动；不操作环境资产，不扩其它平台同名删除 |

Legacy decision: **RETIRE** 七文件与专属闭包；有效行为复用现有当前 owner，无新增抽取。
Reusable behavior: 当前 marketplace TEMU 的类目/模板/备货/规格规则、原平台 API/DTO 和共享能力原位保留。
Current owner: marketplace TEMU goods/rules/model/publishing、原 API/client/共享上下文、当前 Product/Store/Observation/Submission 与独立 worker。
Cutover/deletion condition: 用户明确确认 §5/§6/§6.1/§7；实施前登记 Reuse Existing Architecture / Ready、刷新无保留消费者，按原 guard RED→GREEN 删除并核对保留能力、清单/说明、准确实现 HEAD CI/最终检查；不创建兼容、fallback 或第二事实源。

## 6. 验证与验收边界

设计阶段已完成七文件及 API/共享消费者只读映射、精确导入/声明装配检索、31 命令静态依赖图和前批保留边界对照，只交付 Markdown 并做 diff/相对链接检查。本次实施前已再次刷新 main/消费者；原退休 guard 在七文件尚在时准确 RED，全部保留断言通过，精确删除后 GREEN。适用保留验证、准确实施 HEAD 的原 CI 与最终独立检查维护于 PR；旧设计证据不替代实施检查、平台效果验证或用户验收。

本次实施的必要验证边界：

1. 开工同步一次实际 main，重新检查五个 handler/两个 DTO 的外部导入、直接实例化、注册及当前命令依赖；新增真实消费者、兼容义务或有效规则不能在旧闭包内擅改，停止对应项报告。
2. 先在原退休 guard 加准确类别/模板目录禁令，七文件尚在时确认 RED，再精确删除七文件并 GREEN。此次不新增业务规则，无需为删除旧算法制造业务 RED 或新测试框架。
3. 核对 §5 平台/API/context、当前规则与此前保留实现和测试未改；复用当前 goods/rules/publishing/productpolicy 及 SKU mapping 回归、原 import/provider/logger/命令/worker/consumer/ceiling checks，只验证实际增量。没有新 provider/API 请求。
4. 必要时编译 retained context/handlerbase/SKU/Spec/AI/property/image/API 包，`current-application`、`shein-login-worker` 只 build 不执行。完整必需检查交给稳定实施候选原 CI，不人工重复全量，不增加场景矩阵/runner。
5. CI/独立检查和源码图不能替代真实平台/运行/用户验收；这些维持 NOT_RUN。#29/#301 仍有其它保留债务，保持 OPEN，不将本批维护变成试用或发布前置。

### 6.1 原检查、清单与文档的精确处置

| 现有位置 | 本次实施的最小变更 | 不得扩大 / 削弱 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 在原 TestListingKitTaskBackendSourcesAreRetired 的目录缺席列表增加 `internal/temu/category`、`internal/temu/template`，覆盖改名重建的 Go 文件；原保留 map 补 API/category、QueryTemplate/Advanced/CostTemplate、两套平台模板 DTO、四个 TemplateState 字段及当前规则 owner 的准确断言 | 原 root/product 子树、五个原 mixed 包 AST 扫描、SKU declaration fixture、全部原断言保留；两个退休目录不加入要求必须存在的 AST 遍历，不为目录消失加广泛 skip；不禁平台 category/template 包或 SHEIN 同名类型 |
| 原 import/provider/logger/Phase2/命令/worker guard | 当前候选不直接 import OpenAI/provider，不在相关豁免中；目录删除后复用原规则 | 不加 skip/allowlist，不改 root/扫描/deny/helper、PriceHandler 唯一 management 豁免、ceiling 或必需/安全覆盖 |
| docs/refactoring/phase2-runtime-inventory.md 当前 consumer register | 仅移除 `core/logger → temu/category` 与 `core/logger → temu/template` 两行失去的直接消费者 | dated inventory、historical debt list、其它 consumer 行及 closure ceiling 不改，不把整套 logger/平台能力写为已退休 |
| Legacy register / 本文 / 架构索引与直接冲突的前批当前说明 | 已标准确 source-retired 七文件，说明共享上下文/API/规则仍保留；对前批 category/template 全保留陈述加本设计的窄替代说明 | 历史设计取证/审批保留；code-health 当前无这两包专属行，不发明行或改其它基线；设计阶段只提交本文与索引 |
| .github/workflows/ci.yml、commercial-readiness.yml 及其它 workflow、.golangci.yml/依赖/保护 | **零变更** | 原检查目标/必需、安全检查与条件分类均保持；本批不为无 runtime 消费者的维护改 CI/发布/环境前置 |

## 7. 确认、实施与停止条件

1. 用户已明确确认冻结设计 §5/§6/§6.1/§7，授权在同一 #773 实施七文件退休。原设计交付与确认保留为历史依据；实现、自检及 PR 维护不是用户验收或生产发布批准。
2. 生产修改前已在同一 PR 对应 Issue 登记 Reuse Existing Architecture / Ready，引用冻结设计、当前 owner/路径/不变量及验证并读回；随后才实施七文件退休和原 guard/清单精确校准。准确实现 HEAD 与交付状态以 PR 当前证据为准，不在本文滚动。
3. 发现新当前消费者、具体外部可观察兼容义务、仍有效规则无正确 owner，或必须改变身份/tenant/事实/状态/事务/恢复/副作用协议，停止受影响部分报告；不临时增加 adapter/fallback，不自动扩成 IAM/规则/恢复平台。
4. 实施完成以准确 HEAD 的保留验证、原 CI 及一次最终交付检查进入 In Review；已有未变证据可明确复用，finding 先分类，修复只复核增量。设计 CI/review 不冒充实施 CI/review。
5. 合并仍需准确最终 HEAD 单独授权并正常 expected-head/保护；本设计不授权 Issue 关闭、部署、真实实例/provider/凭据/业务数据操作。当前采集、Store 观察与独立 SHEIN 登录/Store sync 保留，本项不增加内部试用前置。
