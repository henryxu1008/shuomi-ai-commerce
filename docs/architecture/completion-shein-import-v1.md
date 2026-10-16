# CompletionDraft → SHEIN Target 带入预览与确认

Design Basis: Independent Architecture
Admission Status: IMPLEMENTATION_READY — 2026-10-16 独立 R1 核对设计提交 83f34ad3698beb51be17f808b25b9c98a05a54f3，无架构 BLOCKER；按本基线实施。

## Authority / Outcome

依据 [本轮产品决定](../product/completion-shein-import-v1.md)、[结构化候选](completion-draft-structured-v1.md)、[SHEIN 供应链](my-supply-chain-shein-v1.md) 和 [当前 UI/IA](../product/final-ui-ia-authority.md)。用户在独立页面将固定候选的选定字段预览后追加为本地 SHEIN Target 新版本，保留草稿、商品、批次返回路径，不发布。

## Owner 与调用路径

CompletionDraft 原 record/head/command 是不可变候选的唯一 owner；SourceItem/Catalog、Store/官方规则/ApprovedAsset 和 Target 原 record/head/command 各自保持 owner。没有新表、状态机、队列、迁移、跨库写事务或 adoption 事实。

新增有界 `POST supply-preparations/shein-completion-preview` 只读预览，接收 sourceId、baseRecordId、completionDraft{recordId,revision} 和 selection。Console BFF 严格 DTO/响应绑定 → supply HTTP → 现有 TargetService（已注入 SourceSelector、CompletionVersions、Records、Rules）→ 返回完整 TargetInput。它不是保存、授权凭证或发布入口。

TargetInput 新增可选 `sheinImport`，仅 SHEIN 可用：baseRecordId、固定 completionDraft 引用、selection、rulesHash、merchantHash。所有字段进入原 inputHash，并随原 TargetRecord 保存。普通 Target 编辑不得延续已保存的旧 sheinImport；只有本轮明确预览并确认的命令携带它。TikTok 的 completionDraft 文字绑定不变，不允许平台字段交叉混用。

## 版本、映射与覆盖合同

预览按当前 scope 读取不可变 baseRecordId 和 CompletionDraft recordId；核对同一准确 SourceItem、sourceId、SHEIN 平台、相同目标身份。目标当前 head 必须等于 baseRecordId/Rm，否则 409。候选 Rn 可以是历史版本；候选 head 后续增长不影响该选择，禁止静默换成最新。Target effectiveVersion/applyReceiptId 保持基线，原 Product/Review owner 继续核对它们。

selection 中标题/描述是布尔选择。属性项用候选数组 index、目标 attributeId、可选 optionId；不得重复来源或目标属性。目标要求 type 3/4、dimension 1、show=1、status 非停用、mode 已知。mode 0 用候选值作整数；其他 mode 显式选择已有可见 option；mode 4 的 extra_value 用候选原值，其他 mode 不携带 extra_value。替换所选目标 attributeId 的全部旧值，其他属性保留，不映射销售/尺码/SKC/SKU 属性。完整业务合法性仍归现有 goods.BuildOfficial，问题不伪装为 Ready。

SKU 项固定 sourceIndex → skcIndex/skuIndex，双方唯一、有界且至少选择 price/stock 一项；从固定候选结构读取数值，不能由请求另填。价格按当前 application mode 及 currency 严格匹配，采用原单位/表示，无转换/舍入。自营只覆盖目标价格列表为 USD/shein-us 单项；托管只覆盖 cost_info，其余原价字段保留。库存覆盖明确选中 warehouse ID 的一项，其他仓保留；无仓/全托管只能覆盖唯一无仓项，有歧义的已有仓项拒绝。未填写禁止选择，0 与省略区分。数量和数组限制沿原合同，映射最多 256 属性/256 SKU。

服务端从基线完整 draft 深拷贝、按 selection 派生唯一输出，不能信任客户端全量 draft。预览读取基线规则后映射，再按结果读取当前规则（包含 linked 规则）；按新规则重算同一输出，不一致/失效即拒绝。返回完整 input 及规则/商户绑定 hash，页面以该 input 展示变化，编辑映射即清除预览，只有显式确认可提交。

## Save、并发、幂等及恢复

确认复用原 `POST targets`、expectedRevision=Rm、durable key/payload、FindTargetCommand、原 TargetPrepared/SQL CAS 和事务。新写时重新读取同 scope 候选/基线，检查来源、revision、当前 head 及基线版本，使用当前官方规则与商户绑定重算 draft，与请求完整 draft 精确比较；rulesHash/merchantHash 漂移返回 409。不得夹带其他字段变更。

候选与基线在 Prepared.Read 的持久化边界再次按 immutable ID 核对；不在 SQL 事务内重复官方网络请求。商户/规则核对继承既有 Target 当前读取和 expiry 合同，不新增跨系统锁或声称官方规则可被本地事务锁定。当前 supply.manage/SourceSelector/live Store 访问和 expiry 按原保存路径复核。目标 head 由原事务锁和 CAS 最终判定，409 不自动合并或换基线。

同 key/同完整 payload 已提交时优先返回原回执，即使当前 head 或规则已变化；同 key/异 payload 409。响应丢失/超时/重启保留原 key 和完整 input，沿原命令核对/重试，不能新建键或将历史回执显示成最新 head。页面核对成功显示“本次保存版本”，重新读取是显式操作。取消预览无持久化效果；没有 UNKNOWN worker 或自动恢复副作用。

## 权限、资源和 Threat Model

普通读取沿 supply.read；预览/确认要求 supply.manage、当前登录 org/user/member/token 及 scoped Source/Completion/Target/Store。导航 query 只承载严格 ID，不能决定 scope；核对 preparation/item 与实际 SourceItem，所有返回 href 来自已核对数据。跨企业/他人记录不可读，无权限不发预览/保存。没有配置和正式规则时 fail closed，fixture 不作真实平台证据。

沿原 2 MiB body/record 限制、Target 10s deadline、固定 max 256 映射；JSON 严格字段与 BFF schema，数值安全范围。不会保存凭据、URL returnTo 或新授权缓存。保持资产批准、图片校验、发布权限、Submission 状态机及外部副作用边界；不调用任何发布/上传入口。

Legacy decision: N/A。复用当前 Target/Completion/Store/规则 owner，不依赖登记 RETIRE 的 Task-first 或旧 Listing Workspace。

## 验证与交接

TDD 覆盖固定旧候选/源错配、显式一对一映射、空/零、币种和仓库、只覆盖选中字段、篡改完整 payload、规则/商户漂移、目标并发 409、scope、原 key replay。使用现有 domain/HTTP/SQL fixture 和 Console 测试，不新建验证平台。前端验证独立页、前后预览、修改后重新预览、409 显式重读、UNKNOWN 原命令恢复、主题及返回上下文。沿既有必要 CI；真实 Store/provider、保留实例与产品验收 NOT_RUN，分别授权/确认。

适用两个检查点：本合同独立高风险边界检查，以及完整实现后的最终交付检查；不增加常驻多 writer 或重复全局架构审查。

R1 finding：IMPLEMENTATION_TEST（合并层）：带入保存及原键回执核对必须比较完整规范 TargetInput 和本次 Rm+1，不能仅比较 source/store/version 元数据；沿原恢复协议补测试和最小修复。
