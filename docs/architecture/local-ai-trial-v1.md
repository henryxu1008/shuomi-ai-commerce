# Local 一次性 AI 试用来源 v1

Design Basis: Independent Architecture。
Admission Status: NOT_IMPLEMENTATION_READY / DESIGN_IN_PROGRESS。

执行 Issue：[Local 一次性 AI 试用来源 #724](https://github.com/qq550723504/task-processor/issues/724)。

本文先于生产实现。当前只记录人类已批准产品来源和待独立检查的最小合同；A 更新/读回优先，尚未派独立检查、发放或启模型。

## 1. Outcome / authority / scope

依据 [PD-LOCAL-AI-TRIAL-2026-10-15](../product/local-ai-trial-v1.md)；完整人类消息ID和参数在该产品文档及执行 Issue。只为原 Local 企业提供一次10000非现金ai_point，目标成员月限10000，tariff输入/输出每百万200/800点，完成原专用品标题优化→Review→Apply→真实Choice→供货页/Figma检查到提交前。累计上游预算不超过CNY100。

Figma沿产品依据，无新增页面；原资源明细若缺少新来源文案，只补“本轮AI试用赠额”标签，不公开内部source编码。权限、正常资源和用量读取组件复用。

不做通用赠点/营销、任意资源mint、自动续发、现金钱包/购买订单、新schema、迁移/回填、跨组织分配、新IAM/预算/恢复平台或真实申请提交。Legacy decision: N/A；不消费RETIRE owner。

## 2. Owner / contract / injection / consumer

唯一资源fact owner仍 `internal/ledger/orgresource`，持久化 `internal/integration/orgresource`。原bucket/operation/source_claim/event/audit及债务事实复用；Commercial/money、成员月限、模型invocation计量与 UNKNOWN、Review/Catalog/Collection/Choice继续原owner。

新增专用 `TrialAIGrantService` / typed executor；输入只有组织、operation ID、可信operator principal，不接收浏览器任意resource/quantity/source。runtime装配的专用authorizer提供准确批准的组织、固定policy及approvalEvidenceID，并验证可信native operator；人类／企业角色字符串不能替代该授权。执行只允许ai_point、固定10000及固定来源，adapter也校验这些常量，不能将execution DTO变通用mint。

调用链：原owner受保护批准清单/既有资源库窄角色 → 一次性native `local-ai-trial-grant` → 专用authorizer/service → 当前GORM repository/事务runner/positive credit helper → 原资源账本 → 正常资源和模型消费。无常驻HTTP路由、不在serving启动时seed。native默认只读检查/读回，明确grant动作才写；严格小JSON、未知字段拒绝，DSN/凭据私密输入，错误脱敏，不输出或接受命令行明文密码。

资源库连接复用 `commercial_owner_runtime` 和现 `VerifyRuntimePermissions`；不调用AutoMigrate/initializer、不临时增加权限。角色/schema不满足即停止交实际差异。私有operator及资源库管理员属于既有可信运维边界，不新增IAM或对恶意DB管理员的防护产品。

## 3. Identity / persistence / concurrency

固定 `operation_type=grant_local_ai_trial`、`source_type=local_ai_trial_v1`。source_identity由固定首次试用policy+组织+ai_point的规范内容digest得到；不含operationID、部署ID、当次时间或caller可换的campaign，重建部署不能再领。scope组织由批准清单准确绑定，本轮仅Local。

fingerprint绑定组织、固定来源/10000点、不可变批准证据/policy，排除operation ID和当次时间。同键同输入、同source异键同输入均返回原不可变回执且不新增事实；相同键或source不同载荷冲突。原operation和source claim两道身份都检查，不用“换请求编号”再发。更改模型、部署或新提供批准证据不自动新增来源。

沿现SERIALIZABLE事务runner及有界冲突重试：事务内replay→创建缺失bucket并锁定→再次replay→现 `applyPositiveCredit` debt fence→同事务写operation、source claim、一次credit event/audit并更新bucket。仅增加可用净额度，allocated/reserved/consumed不重置；gross10000优先偿还原资源债务，回执记录gross/debtRepaid/net及余额，不通过追加赠额补齐净值。

唯一性、row lock和串行化由原数据库实现，不用进程锁。任何失败回滚全部该次写入；重复／并发即使不同operation键也只一次credit。actor记录可信operator身份，批准证据另存不可变回执/现approval列，不伪称浏览器已认证操作者或商业付款。

## 4. UNKNOWN / recovery / authorization

COMMIT或响应丢失不是确定失败。复用原runner分类，仅原org/source/operation/fingerprint读回；已提交则返回原回执，明确缺失后才允许原键重试。不得换source/operation制造第二次赠额，不自动删除、负向撤销或覆盖已消费/预占事实。

原operator身份/批准组织仍验证后才读回，首次grant还须当前明确启用批准。停用不能将已提交source解释为未领取；只读回执入口与是否允许新grant分开。授权/私密输入不完整、跨组织、非法参数、读取失败或漂移都失败关闭，无业务写入。

Grant、成员月限、凭据/应用配置是各自既有边界，不宣称跨库原子。按固定顺序记录各自不可变/版本回执；后步失败保留前步新事实，继续原身份核实，不补第二笔grant或假装整体回滚。A和B实例变更均只由原运行owner执行。

## 5. Model and runtime admission

已批准tariff版本建议 `local-dashscope-deepseek-v4-1-flash-trial-20261015-v1`，输入/输出每百万200/800点；provider成本按官方北京忙时原价每百万CNY2/8保守估算，启用前重新核对。10k额度只能在无其他启用route/续发、完整usage和UNKNOWN保留预占等原前提下作为累计预算约束，不能把单run上限冒充总预算；无法证明预算上界则不派发。

复用原成本/点数计量、价格版本冻结和余数守恒。单call最多65536 prompt/4096 completion tokens、45秒；单Run最多2call/12step/139264tokens/CNY1/120秒，预算仅降低原上限。保留scope、当前成员/权限、原lease/幂等和UNKNOWN；不得补member position余额前置。

Member限额沿正常LiveWrite账户入口设置10000；先读实际版本，保留消耗/预占，不用SQL替代。原credential-provision导入既有default私密引用，输出真实credentialVersion/endpoint digest，启用准确content route和个人/企业当前准入，不伪造版本。Chat/Store原开放范围、容量、OIDC/CA及持久任务不变。

B真实执行在A更新和登录/草稿历史/比较/刷新读回交接后，绑定准确最终源码/CI与原owner具体配置/私密备份/停止清单。原Key不打印不更换；不复制到git/build context。只新增此次必要事实，不重置已有数据。

## 6. Validation / review / delivery

必要TDD：未授权/错组织/缺证据拒绝，固定resource/quantity；同键重放、异键同源一次credit、不同载荷冲突；原真实PostgreSQL并发/事务及debt优先、提交后响应丢失读回，复用现testcontainers/隔离测试，不连接保留业务库或造新验收框架。native非法输入/只读零写入与敏感错误脱敏；如文案直接消费新增reason则只补对应现有测试。

独立检查只覆盖新增发放边界，最多两轮正常设计评审；达到IMPLEMENTATION_READY后才正式生产编码。完成完整有界路径后一次最终实际diff检查，finding按AGENTS分类；未变模型／财务／Review证据复用。

实际使用必须分别记录：一次grant与回执、余额/月限真实读回、真实模型调用/费用/候选、正常Review/Apply新Catalog版本与真实资格、供货页/Figma截图和表单检查。止于提交前，CI/受控测试不冒充这些结果，不自行签发产品验收或关闭父Issue。
