# 完整旧 TEMU 文本检查与要点闭包退休边界

## 1. 状态、用户结果与范围

- 日期：2026-10-18；Refs #29 / #301。用户要求减少机械切片、按完整退休闭包和一个主要 PR 推进，已确认 #812 冻结设计 b09ed052b8fb652b2c4cc9c9194bc111fd07040c 的 §5、§6、§6.1、§7，并授权同一 PR 实施。
- **USER_CONFIRMED / SOURCE_RETIRED；Design Basis: Reuse Existing Architecture**。生产修改前已在 #29/#301 登记并读回 Ready；§5 的完整闭包已源码退休，原 guard 按 §6.1 校准。设计阶段 N/A、CI/review 为历史，不替代实施验证；准确实施 HEAD、CI 和最终独立检查维护于 #812，合并另授权。
- 取证 actual main：**f8274448d0c8ff80cd6a55be60338e3d34029632**（含 #809/#810）；实施开工同步 actual main b161243b891540802a7f35ced8938ca69e66bb42，#811 的 Store/Chat 路由修复与本闭包无交集。唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397；复用附属独占 worktree issue29-temu-image-retirement/task-processor，新分支 codex/issue29-temu-text-bullet-retirement-design。已核对接单/关联 PR，无同范围 owner；#692 安装工作保持原 owner。

使用者为内部试用、采集/Store/内容维护者。本批结果是一次退出无装配的旧文本检查、文本预处理、要点优化/校验和所有失去最后消费者的专属文本策略；不把调用方与孤立依赖拆成后续小批。当前没有用户操作被此维护阻塞，**不增加内部试用前置**。无 UI 变化，Figma N/A。

完整候选：**6个整生产文件/976行、2个整旧测试/52行**；另精确删减 shared/strx 的2个混合生产文件和1个混合测试中的闭包。包括四旧 rules 文件、formatter/共享 regexp、CleanProductTitle/三私有 helper/CleanWhitespace及其专属测试；不整删 rules、publishing、strx、context 或 API 包。无新增 EXTRACT 业务代码、替代 formatter 或新 caller。

非目标：改变当前内容事实、语言/长度/品牌规则、平台 API/DTO/官方规则/有效共享合同，复制旧默认营销/猜测卖点/质量评分，补完旧 TODO、建立新验证/治理系统或兼容路径。不得运行 main/constructor、下载/上传/模型/真实平台 API/Preview/DryRun，读取实例配置/凭据/DB/Redis/既有 Prompt 覆盖记录/业务数据，或部署、迁移、重放、清理、关单。

## 2. Design Basis、authority 与不变量

实施采用 **Reuse Existing Architecture**；设计阶段 N/A 不代替生产准入。依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：先核对当前 requirement/owner，旧实现和测试只有 EXTRACT / RETIRE；无 wrapper、fallback、双读双写、旧 ID 映射或第二事实源。
- [#810 文案边界](legacy-temu-content-handler-retirement-boundary.md)、[#751 variant 边界](legacy-temu-variant-retirement-boundary.md)、[#749 保存/提交边界](legacy-temu-listing-submit-retirement-boundary.md)：当批有消费者的 formatter、TextProcessor/CleanProductTitle 保留属于当批边界。本设计以最后 caller 退出及当前 requirement 复核为依据，提议窄替代这些对象的保留，不恢复已退休对象或重写历史批准/计数。
- [TEMU 平台设计](temu-platform-v1.md) M3/M4、[当前 publishing owner](../../internal/marketplace/temu/publishing/README.md)、当前 goods/rules/model 与 productpolicy：官方 mode/category/snapshot 校验、Product/ApprovedAsset 与明确提交/effect/receipt/UNKNOWN owner 保持。没有找到要求继承本闭包旧 ASCII 剥离、统一100/200/2000字节、6条/700字节、标点重排或营销补词的当前已批准合同/保留消费者。这是仓库 requirement 核对结论，不宣称外部平台没有字符或文本规则。
- [Product Content / Chat 当前路径](chat-task-object-goal-design-brief.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：当前内容选择、生成/Review/采用、Product/版本事实、Store 观察及 worker 持久化均沿原 owner，不接旧文本策略。
- [#807 Prompt 边界](legacy-temu-image-api-prompt-retirement-boundary.md)：AIContentRewriter/content_rewriter prompt/key/catalog 和通用 Prompt 存储/管理/已有覆盖记录保持。

调用路径为旧 TaskContext → TemuTaskContext → TextCheckHandler/要点 handler，或孤立 TextProcessor → 独占 shared helper；要点格式化 → publishing formatter → 四 regexp。全部没有当前命令装配。退出这些旧 caller 和无保留消费者的策略；平台 QueryAPI/ClientAPI 与所有当前能力不变。

不变量：不改变保留合同的身份/tenant/权限、canonical fact/schema、状态机/事务、幂等/retry/UNKNOWN/recovery、AI 准入/预算/计量或平台/provider 副作用 owner；不新增装配、替代状态或公共领域合同。该有界闭包低于30范围文件/2500生产变更行，现有批准架构完整覆盖；若出现真实保留 consumer 或上述边界变化，停止受影响项并补适用独立准入。

Must：PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、AI/SKU/AIBatchInput、AIContentRewriter与文案prompt/key/catalog、当前图像、有效 API/DTO/平台规则/共享基础层、DimensionInfo/TextRenderer、Reader/Cache/Mapping、两种 SHEIN 属性、四个 TemplateState 字段、当前采集/Store观察、独立 SHEIN 登录 worker/Store sync及此前其它保留能力。两套及其它 workflow、必需/安全检查、依赖/保护/ceilings 不改；无新增 Accepted Risk。

Threat Model 限于误删有效消费者/规则/同名方法，把旧猜测/默认文案/TODO成功当真实产品事实或平台验证，以及把旧内存改写当数据清理授权；不扩成新内容治理平台、IAM 或试用验收矩阵。

## 3. 实际消费者、调用和架构问题

全仓生产/测试/构造器/类型/别名引用已核对。当前 **31个 cmd** 静态依赖并集中 temu/rules、temu/product、marketplace/temu/publishing 均为0；四旧 owner 无包外生产调用/构造/注册。未执行 main/constructor。源码证据不证明实例、外部调用方或历史数据不存在；发现具体当前兼容义务按 §7 报告。

| contract → implementation → injection → consumer | 实际行为 / 最后消费者 | 设计决定 |
| --- | --- | --- |
| TextCheckHandler → 根api.NewQueryAPI → query.API.CheckText → client.ClientAPI.SendTEMURequest | 构造器只见定义；从旧AmazonProduct.Title取值，POST /mms/marigold/query/commit/check_text，成功后写旧TemuProduct.GoodsBasic.GoodsName。不是纯值函数，也不是TODO假调用 | RETIRE未装配旧caller及内存赋值；保留 QueryAPI、CheckText、同名 request/response/re-export、client认证/请求/错误 owner；本轮不执行网络调用 |
| TextProcessor → strx.CleanProductTitle → 三私有过滤 helper/CleanWhitespace | 只有本文件定义/自调用和原guard保留断言；标题100、描述2000、单要点200字节/最多5条；空描述/默认要点自动添加营销文本 | RETIRE孤立receiver、默认文案、专属 shared 闭包；不把旧固定限制或去中文行为接入当前内容路径 |
| BulletPointsValidator → BulletPointsOptimizer / formatter | 构造器无外部装配；读取/覆写旧GoodsExtensionInfo.BulletPoints，校验结果/建议后继续优化和写回，未按IsValid拒绝。其DTO只供本文件 | RETIRE handler、优化器、BulletPointValidationResult/OptimizationStrategy及receiver helper，不转移质量事实/建议协议 |
| BulletPointsOptimizer | 只由旧validator调用；关键词推断产品类型、补人体工学/可调节等卖点、营销词替换、按关键词重排及截断 | RETIRE整闭包；不“修好”旧推断/排序后抽到当前 owner，不新增产品需求 |
| ValidateBulletPointsAPI | 只有TODO日志后返回nil，无真实HTTP验证 | 随旧receiver RETIRE；不能把它与TextCheckHandler的真实HTTP实现混为一谈，也不为它新接平台调用 |
| NormalizeProductSubmissionName → 四 regexp | 唯一生产caller是BulletPointsValidator.optimizePointFormat；其余只有自身测试/guard。该caller退出后，无当前 requirement 或保留consumer | 一并RETIRE formatter两生产文件/四 regexp/专属测试；不留空壳、改名副本或新 caller 维持“已抽取”代码 |
| strx.CleanProductTitle → removeEmojis/removeChineseCharacters/removeSpecialCharacters/CleanWhitespace | title函数只由TextProcessor消费，三个私有函数及包级CleanWhitespace只供此闭包；后者另有一专属测试。SHEIN TextCleaner.CleanWhitespace是其它包的方法 | 一并RETIRE这五个声明和专属测试；保留SHEIN同名方法及strx其它函数/测试，不按名字全局删除 |

发现的问题包括：旧要点校验没有阻止无效结果回写、根据缺少特性的关键词补卖点、默认营销断言、按字节处理文本和不同旧组件限制不一致。这些不是当前 requirement。无当前装配/事故证据，分类 **NOT_APPLICABLE（当前内部试用阻塞）**，以 RETIRE 处置；不扩大 Threat Model 或修复旧链。发现真实当前 consumer 时不得用此分类跳过 Must，须停受影响部分复核。

## 4. 行为归属、持久化和平台副作用

| 边界 | 当前 owner / 保留语义 | 本批处置 |
| --- | --- | --- |
| 标点/空白格式与旧ASCII清理 | 上述formatter/标题清理全部只被拟退旧caller使用；旧namespace、#810当批保留、7个formatter fixture、2个wrapper fixture或6个Whitespace fixture均不构成永久requirement | 无新EXTRACT；专属策略/测试随最后consumer退休，不迁移fixture或复制到当前owner维持旧需求 |
| 平台文本检查 | query.API.CheckText/QueryAPI、TextCheckRequest/Response及根alias、ClientAPI的认证/请求/错误处理继续归原平台API/client | caller退休不删除端点合同，不建立新验证服务/重试或UNKNOWN owner；不把平台回执当本地Product事实 |
| 旧GoodsName/BulletPoints/校验结果/策略值 | 旧TemuTaskContext及平台Product DTO的进程内数据；四旧文件没有DB/queue/file持久化写入 | 只删旧mutator/专属值；context、Product/Extension/BulletPoints字段及当前资料事实不删，不访问/迁移历史行 |
| 当前Product/内容/官方Submission/effect/Store | 原canonical Product/版本/Review/采用、官方发布事实及receipt/UNKNOWN、Store观察与独立SHEIN持久化owner | 原合同与事务/权限/恢复不变，不引入第二事实源或业务操作前置 |
| shared/strx其它声明 | CleanProductText及其filterAllowedRunes/isAllowedRune/collapseSpaces/collapseNewlines、ContainsIgnoreCase/FindSubstring/TruncateString/Contains/ToLower/ToUpper等不在旧root可达的最后消费者闭包 | 本轮不对它们作新的永久保留/当前装配判断，不增加caller或新guard保留权；保持原实现/其它测试，不整包做无关dead-code清扫 |
| TextRenderer/DimensionInfo、其它TEMU规则/定价/AI | 保留原owner；TextRenderer继续消费core/logger，PriceHandler也继续消费core/logger；AI/SKU/content prompt原调用保留 | 不整删rules/publishing/product/shared，logger/AI consumer行与guard/loader/ceilings不改 |

本次用户确认仅窄替代 #751/#749 对 TextProcessor/CleanProductTitle 和 #810 对 formatter/left regexp/fixture 的当批保留。用户确认的 §5、§6、§6.1、§7 是这些具体对象的删除依据；此前实现与 CI/批准保留为历史，其它有效能力不受影响。

## 5. 完整删除与保留矩阵（用户已确认，源码已退休）

以下六整生产文件共976行，两整测试共52行；mixed文件仅按准确声明删减，不把整文件行数算成删除量。

| 路径 / 基线 | 决定 | 精确边界 |
| --- | --- | --- |
| internal/temu/rules/text_check_handler.go（125） | RETIRE整文件 | TextCheckHandler/NewTextCheckHandler及全部receiver、旧HTTP编排/标题赋值；query API/DTO/alias/client原样保留 |
| internal/temu/rules/text_processor.go（100） | RETIRE整文件 | TextProcessor/NewTextProcessor及四方法；专属shared依赖见下，非全包清扫 |
| internal/temu/rules/bullet_points_validator.go（308） | RETIRE整文件 | BulletPointsValidator/NewBulletPointsValidator、BulletPointValidationResult及全部校验/清理/质量建议/API TODO/旧内存回写 |
| internal/temu/rules/bullet_points_optimizer.go（411） | RETIRE整文件 | BulletPointsOptimizer/NewBulletPointsOptimizer、OptimizationStrategy及全部receiver/private helper；无替代optimizer |
| internal/temu/rules/bullet_points_validator_test.go（25） | RETIRE整测试 | 两个旧wrapper fixture；不迁入其它owner |
| internal/marketplace/temu/publishing/submission_name_policy.go（23）及name_format_policy.go（9） | RETIRE两整文件 | NormalizeProductSubmissionName、rightAnyParenthesisSpacingPattern/commaSpacingPattern/punctuationSpacingPattern/leftParenthesisSpacingPattern、专属imports；无mixed残壳 |
| internal/marketplace/temu/publishing/submission_name_policy_test.go（27） | RETIRE整测试 | 全部7个fixture/表测试；包含#810承接的3例，当前已无保留consumer，不再次搬迁 |
| internal/shared/strx/cleaner.go（110，mixed） | RETIRE四声明/专属import | 精确删CleanProductTitle/removeEmojis/removeChineseCharacters/removeSpecialCharacters和不再用的regexp import，修正CleanProductText注释中失效比较；其它实现不改 |
| internal/shared/strx/string.go（50，mixed） | RETIRE一声明/专属import | 只删包级CleanWhitespace及不再用的regexp import；其它字符串函数保持 |
| internal/shared/strx/strings_test.go（163，mixed） | RETIRE一测试 | 只删TestCleanWhitespace及其6-case表；其它测试/import保持，不整删文件；SHEIN同名方法测试不动 |
| QueryAPI/CheckText/request/response/根alias/client，rules其余4文件及其它publishing/shared函数 | 保留范围边界 | rule_engine/rule_interface/percentage_sum_rule/text_renderer、有效平台规则/数值格式/私有验证/共享合同保持；不因“text/format”同名跟删 |
| 当前采集/Store/独立SHEIN、AI/SKU/prompt、PriceHandler/PriceManager/PriceCalculator及§2其它Must | 保留 | 不改装配/运行/数据或创建新consumer，不以实现者自检签用户验收 |

Legacy decision: **RETIRE** 完整旧文本/要点执行与专属最后消费者闭包。
Reusable behavior: 当前有效平台/内容/共享行为沿现有owner保留；拟退策略没有保留consumer或当前requirement，无新EXTRACT生产代码。
Current owner: 当前TEMU goods/rules/API/client、Product/Review/ApprovedAsset、原AI/SKU/prompt、PriceHandler、Store与独立SHEIN身份持久化。
Cutover/deletion condition: 用户确认本文§5/§6/§6.1/§7 → 执行Issue登记读回Reuse Existing Architecture / Ready → 刷新整个依赖闭包 → 原guard RED后精确退休/GREEN → 准确实施HEAD原CI/最终独立检查；无真实运行切换、迁移或兼容。

## 6. 验证与交付

设计阶段仅做源码/静态依赖/requirement/副作用映射和两份文档，原设计 CI/评审保留为历史。实施在生产修改前完成 Ready 读回，并执行原 guard RED→GREEN 与下列适用检查；准确候选和结果维护于 PR。运行/平台/provider/用户验收 **NOT_RUN**。

用户确认后在同一主要 PR 按以下方法实施全部闭包：

1. 开工同步actual main，刷新6整生产/2整测试、三个mixed文件和全部导出/私有声明引用、31命令依赖与保留合同。新真实consumer/具体当前兼容义务停相关项，不隐式扩大授权。
2. 在原退休guard先增加8整文件准确缺席与13个局部包级声明禁令，精确退出下表旧保留断言，并新增明确的平台文本检查合同断言。新增保留断言在删除前即应满足，完整guard因待退文件/声明尚在而RED；再一次精确退休确认GREEN，不伪造孤立formatter测试搬家或新规则修复。
3. 运行现有publishing/temu rules/shared strx与受影响product/sku/ai/context/API、当前TEMU goods/rules/productpolicy和SHEIN content/submitprep回归；原consumer/import/命令/独立worker/ceiling检查、current-application与shein-login-worker build。不运行main/constructor/provider/API；共享测试保留其它有效行为。
4. 稳定候选才运行完整原PR CI和最终独立检查；修复只复核真实增量，不重复全局架构审核/同SHA检查。实际scope保持单PR，不以formatter/shared helper再拆下一批；合并仍另取准确HEAD授权，main CI随后独立核验。
5. 若本地旧source残留目录导致原guard与干净checkout不同，使用准确候选的干净源码快照验证并分别报告，不改弱检查或操作运行资产，不把本地残留说成测试通过。

### 6.1 原检查与文档的精确处置

| 原位置 | 已确认实施变更 | 不得削弱 / 扩大 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go缺席列表 | 增8整文件路径（6生产+2测试） | 原退休文件、5 mixed包、SKU fixture、#810/#807及此前安全/身份/采集/Store断言保持 |
| 原rules混合AST walk | 只加10包级声明：TextCheckHandler/NewTextCheckHandler、TextProcessor/NewTextProcessor、BulletPointsValidator/NewBulletPointsValidator、BulletPointsOptimizer/NewBulletPointsOptimizer、BulletPointValidationResult、OptimizationStrategy | 覆盖type alias/var重建；不禁止其它包或field/selector/local/方法上的通用同名，不建新扫描器/验证平台 |
| 原publishing局部AST与shared/strx局部包级检查 | publishing加NormalizeProductSubmissionName；shared/strx只禁CleanProductTitle/CleanWhitespace包级声明，共3项，和rules合计13项 | 私有三helper/四regexp通过准确文件/混合diff核对退出，不全局禁generic名字；SHEIN TextCleaner.CleanWhitespace保持 |
| 原保留token map及CurrentCommands保留路径 | 退出四个旧token-map条目：submission_name_policy.go/name_format_policy.go、cleaner.go的CleanProductTitle、text_processor.go；去掉三formatter保留路径（两个生产+一个测试），并按下一行新增平台合同断言 | 旧保留断言按最后consumer变更校准；PriceHandler/TextRenderer/其它有效owner及原worker/命令检查保持，不删检查目标或放宽通用安全 |
| 同一token map中的平台文本检查保留断言（首轮IMPLEMENTATION_TEST补齐） | api/api.go精确断言QueryAPI=query.API、QueryAPIInterface=query.QueryAPI、NewQueryAPI=query.NewAPI及TextCheckRequest/Response指向query同名类型；query/query.go断言QueryAPI接口中CheckText的请求/响应签名、实际API.CheckText方法签名及POST端点/mms/marigold/query/commit/check_text；query/query_types.go断言两DTO声明及原Content/Type、Success/ErrorCode/Result JSON合同；client/interfaces.go断言ClientAPI.SendTEMURequest原签名 | 使用当前源码准确声明/目标/签名，不能只匹配名字或注释。这些保留合同删除前已存在，应GREEN；实施交付另核对整个api/client保留源码相对基线零diff。只补原guard，不新增测试框架、真实HTTP调用或平台准入要求 |
| 原两整测试与shared mixed测试 | 两旧wrapper/formatter测试整退，shared只退TestCleanWhitespace | 不移动孤立fixture到新owner维持无当前需求的策略；保留SHEIN同名方法和strx其余测试 |
| tests/import_boundaries_test.go / phase2-runtime-inventory.md | **零变更**：core/logger→temu/rules仍由TextRenderer消费，→temu/product仍由PriceHandler消费；目标无OpenAI豁免 | 不改loader/root/deny/ignore/ceilings、dated inventory或AI/logger准入；不把当批静态图冒充部署证据 |
| Legacy Register、code-health rules行、publishing README、本文/索引、#751/#749/#810直接冲突说明 | 批准实施后标记完整闭包source-retired，并窄替代对应保留；code-health rules行纠正旧“configured runtime reachable”表述，只列本次四文件退休，其余包保留。原声明数/报告为历史 | 原设计阶段只有本文/索引；实施按本矩阵更新，不整删rules/publishing/shared，不给无关预存函数新增永久保留权，不重写原批次证据 |
| 两套及其它workflow、依赖/保护 | **零变更** | 必需/安全检查、编译目标、触发/权限/并发与全部ceilings保持；不新增试用或发布前置 |

## 7. 确认与停止条件

1. 本设计一次交付完整调用/最后消费者/共享helper/测试/guard/文档闭包。用户明确确认§5、§6、§6.1和§7后，在同PR登记并读回Reuse Existing Architecture / Ready才修改生产；原设计阶段N/A仅覆盖设计；本次确认后已按Reuse Existing Architecture登记并读回Ready。
2. 冻结后范围内实现、finding分类与修复、开发自检、commit/push、PR维护连续完成，不逐项请示。只有出现新的真实保留consumer/具体当前兼容义务，或必须改变身份/tenant/权限/事实/schema/状态/事务/恢复/副作用边界，才停受影响部分并报告/补准入；不能以推测无限重开设计。
3. 不新增caller、wrapper/fallback、双写或新事实源来维持旧代码/测试；已抽取namespace和历史保留不是永久authority。未变有效代码/证据可引用，仅验证真实组合风险。
4. 实施完成In Review，交准确HEAD原CI/最终检查。finding先按AGENTS分类；非Blocker不自动扩大Threat Model或新增工具/前置，实现者/CI/AI不签发用户验收。
5. 合并须用户准确最终HEAD另授权并正常expected-head；#29/#301保持OPEN。部署/实例/凭据/已有覆盖记录/业务数据操作及关单没有授权，运行/平台/用户验收保持NOT_RUN，不增加内部试用前置。
