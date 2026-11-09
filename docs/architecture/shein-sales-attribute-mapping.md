# SHEIN 单商品多颜色、多 SKU：销售属性映射

Execution: [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158)；关联 #1092。

Design Basis: **Independent Architecture**。
Admission Status: **IMPLEMENTATION_READY / FROZEN**。2026-11-09 独立 R1 核对设计提交 `912557a43316ecfe3bb562cfbc73ea03a4b6d4c9`，0 BLOCKER；四项 IMPLEMENTATION_TEST 见 §11，在实施/合并前收敛。本轮只有设计和独立架构评审；生产实现保持 Backlog，等待用户下一阶段决定。

调查基线：`main 0685b62e1`。唯一 Writer：当前主会话；工作分支 `codex/issue1158-sales-attribute-mapping`。本文不是已实现能力，也不是实例更新或发布许可。

## 1. Product Outcome / Must / Out of Scope

使用者是把已保存自有商品和固定待补全候选 R1 带入店铺平台草稿的运营人员。当前被阻塞的操作是：源商品已经有颜色、尺码和三条 SKU，但基础平台草稿只复制编码，没有可用的官方销售属性映射。

本轮用户决定：**同一商品、同一普通平台 Target** 内保留两个颜色组（SKC）、三条真实 SKU，不通过拆成两个独立草稿、补不存在的白 L 或删除黑 L 绕过校验。

| 准确源行 | 同一 Target 内的位置 | 库存 | 图片归属 |
| --- | --- | --- | --- |
| sourceIndex 0，黑 M | 黑色 SKC / M SKU | 明确 0 | 公共图保持公共；该行专属图仅属于该行 |
| sourceIndex 1，黑 L | 黑色 SKC / L SKU | 未填写 | 同上 |
| sourceIndex 2，白 M | 白色 SKC / M SKU | 未填写 | 白 M 专属图如存在，仅随白 M |

Must：仅这三行、源行完整一对一；0 和未填区别；价格/币种不自动转换；图片身份、用途、归属保持；预览后显式确认；同一普通 Target CAS 保存新版本并能刷新重开；原自有商品、Source、R1 不变；权限、规则漂移、原键恢复和发布边界不变。

范围：SHEIN US、自营、现有普通 Target 的已保存基线、同一准确 Source/effective Catalog 版本及固定结构化 R1；最多 256 SKU、1 个主销售属性、至多 2 个次销售属性；只选店铺已有可见枚举值。本例颜色/尺码仅是源标签，不硬编码成平台 ID。其余模式和平台不改语义。

Out of Scope：生产代码/schema、原 35444 配置/数据/部署、真实模型调用、资产批准或上传、商品/草稿业务保存、发布、账本操作和原 15.445282 AI 点释放；自动翻译/自由文本语义推断、自定义属性值创建、换汇、补库存、SKU 增删、任意跨版本来源重绑、通用映射规则平台。实现和隔离保存测试均须后续阶段授权。

不新增 Accepted Risk。外部平台是否接受稀疏规格集合尚未证实，按 §3 保持发布阻断，不将这一事实降级成已接受风险。

## 2. 产品依据与现状

权威顺序遵循 [AGENTS](../../AGENTS.md)：本次用户明确决定 > 当前有效领域合同 > 当前 Issue。Figma 只辅助理解流程，不作为页面、布局、导航或组件标准。复用 [当前 owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[结构化候选](completion-draft-structured.md)、[候选带入](completion-shein-import.md)、[原生诊断合同](shein-native-validator-contract.md) 和 [类目规则读取](listing-candidate-category-rules.md)；这些文档中的旧 Figma 权威措辞不恢复。

当前代码调查（不是新增产品需求）：

- `internal/app/listing/operations.go` 的初始 adapt 及前端 `draft-fields.ts` 只把源 SKU 编码放进一个空 SKC；尚未把 Catalog `Variant.Attributes` 映射成官方主/次销售属性。
- `internal/app/listing/rules.go` 先从店铺末级类目取得 canonical productType，再读 `query-attribute-template` 和实际组合的 linked rules。官方 ID/选项来自准确 Store/category，不能来自提示词或源标签。
- `target/shein_import.go` 目前只带入文字、商品属性及选定现有位置的价格/库存；不建颜色组、不改变 SKU 拓扑。材料 Agent proposal 也只改现有位置，不代表这条映射路径已经存在。
- `goods/attributes.go:validateVariantMatrix` 同时拒绝同 SKC 重复组合、比较所有 SKC 的完整非主属性值签名集合。后一个检查要求行数和完整组合集合相同，强于仅属性维度一致。
- Target 已有 records/head/commands、单 Product PostgreSQL 事务、CAS、命令重放和 UNKNOWN 原键核对；本轮不新建 family/child、表、队列或第二正文 owner。

## 3. 官方规格集合核实：已知、未证实与处理

2026-11-09 只读核实 [商品发布/编辑官方文档](https://open.sheincorp.com/zh/documents/apidoc/detail/3001352-1000001)，对应实际代码使用的 `/open-api/goods/product/publishOrEdit`。不仅使用搜索摘要，还通过该公开页面自身的只读文档接口取得最新正文：`GET /api/api/apiDoc/queryApiPublishDocDetailInfoById?id=3001352&isLatest=true`，解析到 doc ID `3002086`、version `159`、更新时间 `2026-11-03 10:18:28`，标注自运营/全托管/半托管。同样访问旧 doc ID 3001812/3002018 的 latest 读取也解析到该版本，不能把它们当成相互独立的模式放行证据。

已确认：主销售属性只有一个；SKU 次销售属性至多两个，并有跨 SKC 属性数量要求。`sku_list` 字段还要求销售属性及属性值数量一致；该字段的颜色/尺码示例在**官方返回正文中也未写完整**。因此不能只凭较窄的 `sale_attribute_list` 字段就认定 2+1 行允许发布，也不能据这段不完整说明推导必须把商品拆成两份。另一份[已发布商品销售属性修改方案](https://open.sheincorp.com/zh/documents/system/41d3edcc-f519-4823-99a0-340dea8bb3d1)限定的是修改场景，其属性名一致/组内不可重复要求不能替代新发布的完整资格证明。

结论与边界：

1. **本地保存**允许真实的 2+1 规格结构；保存草稿不代表可发布。现有 Target 已允许带 native issues 的草稿持久化，无须拆稿或造行。
2. 继续严格检查同一主属性 ID、准确次属性维度/数量、官方可见值、组内重复组合、类目 main status/linked rules、大小上限等可确定约束。
3. 本轮**不删除完整集合不一致时的发布阻断**。对本例在共享 native builder 保留非 Ready 与空 SubmissionPayload；诊断改为可定位的 `variant_set_unconfirmed`（建议字段 `skc_list.1.sku_list`），向用户说明“各颜色规格集合不同，平台发布资格待确认；可保留当前草稿”。不再断言用户的真实商品结构本身非法，不提示必须拆稿或补齐白 L。
4. 属性维度不一致与同组重复仍是确定错误，分别定位到属性/源 SKU；不能用新的“待确认”信息掩盖已确定的错误。相同完整规格集合继续按原规则评估，不为所有商品新增全局开关。
5. 若要允许本例真实发布，需取得适用自营接口/类目的明确官方规则答复，或在另行授权的非生产平台环境获得准确三行 payload 的验证证据，再有界修订共享规则。不得通过 UI 开关、fixture 声称、用户点击或其他模式文档取消该阻断。本轮不增加供应商确认流程的通用系统。

诊断 code/message 变化必须沿既有 native validator 的 rule/projection version 与输入绑定合同接线。当前 `OfficialDraftRuleVersion` 为 `shein.official_draft.v1`；后续实施须明确本次新执行使用的版本及所有消费者，不把新结果冒充原版本。历史 Target/result、报告、原 key 回执保留原版本和结果，不用当前 builder 重算覆盖；不新增迁移、fallback 或第二规则源。这是 IT-4 的实施/合并检查，不是另开架构的前置。

文档证据保存于本机任务目录 `C:/Users/Henry/.codex/private/issue1158-sales-attribute-mapping-20261109/official-evidence.json` 及公开原响应 `3001352.json`；保存响应 SHA256 `d11cda18cf6555e33455a857adc2b24deb7b74460040cffdf3d90ec9d40b9dc1`。其中无项目凭据或业务数据；仓库只保存上述必要结论、定位与复核方法。

## 4. Owner / contract → implementation → injection → consumer

| 事实/行为 | 唯一 owner / 复用路径 |
| --- | --- |
| 原商品、源行及图片原归属 | Catalog publication + Preparation SourceSelector，只读准确版本 |
| R1 的文字和候选数值 | CompletionDraft 的固定 record/revision，不读最新 head 替换 |
| 店铺身份、类目与可用官方属性 | Store Center live scope + app/listing RuleReader |
| 已批准素材和真实图像证据 | Asset ApprovedInventory / 原 image observation，不能从源图直接造批准 |
| 映射预览、完整派生和本次操作绑定 | 现有 Listing TargetService，有界 SHEIN 方法 |
| 草稿正文、head、命令回执 | 原 TargetRepository / 同 Product DB 单事务 |
| native issues / 发布 wire | Marketplace SHEIN goods；所有消费者共用 |

拟议调用链：现有 Target 带入页/编辑字段 → Console BFF 严格 DTO → `app/listing/httpapi` 新增只读 `POST /shein-sales-mapping-preview` → TargetService `PreviewSheinSalesMapping` → 当前 Source/Products/Completion/Rules/Assets ports → 返回完整 `TargetInput` 与逐行前后变化。它不保存，不生成平台 SKU，不调用模型或外部 mutation。

确认仍调用原 `POST targets` / `TargetService.Create` / `TargetPrepared` / `SaveTarget`。扩展现有 TargetInput 可选 typed `sheinSalesMapping` 操作绑定（与 SheinImport、CompletionDraft、ColorPartition 及其他平台分支互斥），含准确 baseRecordId/revision、固定 R1、selection、规则/merchant/inventory hashes、派生版本标识。正文仍是 `input.draft`，映射信息只是本次 provenance/重算依据，不成为可独立修改的第二份草稿或业务规则库。沿既有 JSON record 存储，无 DDL 或新 grants。

app/runtime 复用 TargetDependencies 现有注入；所需 source/image 来源从当前只读 Product/Asset ports 取得，handler/UI 不导入持久化实现或持有店铺密钥。HTTP、BFF、浏览器 durable intent、读取/历史/回执解码同步消费完整字段。不是只改前端提交 JSON。

普通新编辑、adapt、材料建议及其他新请求**清除上一次一次性操作绑定**，保留实际业务 draft；历史 record 和原命令回执保留原字段。材料 Agent/自动 Operation 本轮不接入这项新映射命令，不能绕过本轮显式映射确认。

## 5. 输入绑定与映射算法

### 5.1 固定输入

仅从**已保存普通 Target head**开始；首次无 Target 时先走已有创建基础稿流程，不把本次确认写成新建第二个 Target。拒绝 colorPartition；TargetIdentity 继续沿原 org/actor/source/store 身份。R1 必须与基线同 SourceItem、固定源快照/effectiveVersion/applyReceipt lineage；R1 head 后续增加不影响准确旧记录的选择。

Preview request 包含 sourceId、baseRecordId、固定 completionDraft{recordId,revision}、selection。服务端读取基线推导 target/store/site/mode/category/productType/expectedRevision，不把客户端 ID、rules、ready 或 scope 当授权。本轮不在映射同时换类目、店铺或源版本；回原编辑流程保存新基线后再预览。

selection 的有界语义：

- `dimensions`：准确源规格名称 → 官方 attributeId + main/secondary 层级，最多 3 项且层级合法，来源标签只是精确定位字符串。主属性必须是当前模板允许的 Type1/MainLabel1；次属性 Type1，排除本次主 ID，至多两个。
- `values`：每个源维度下出现的**不同原始值** → 当前已存在的可见 optionId；由用户明确确认一次，供所有对应源行复用。相同标签若有歧义则保持未确认，不按“黑/Black”等关键词字典猜值。名称相同最多作为显示候选，不等于语义已确认。本轮不新增模型。
- `rows`：每个固定 sourceIndex → 当前基线 skcIndex/skuIndex 的完整显式一对一对应；服务端核对准确源行与基线编码并展示。编码可能为空或重复，不能靠编码推断源身份；不满足当前平台唯一编码要求时提示原字段补全，不自动造码。位置 0 必须显式提交，遗漏/null 不能默认为 0。
- `images`：§7 的明确旧槽 → 新用途/所有者选择；不接受 URL、新批准身份或把专属图改公共的指令。

所有行从固定 Catalog `Variant.Attributes` 读取规格；R1 只持有 sourceIndex 和候选数值，不能从 R1 不存在的规格字段、SKU 编码或标题推断颜色。规格重复键、缺维度、未映射值、多个源值映射到同一官方值造成组合碰撞时拒绝本次预览，并显示具体源行/维度/值。

### 5.2 派生

1. 深复制完整已保存基线；建立精确源行 → 旧位置 → 新位置双射。两边总数相等、无重复/遗漏/额外行，所有源 SKU 恰好出现一次。
2. 按已确认的官方主属性值建 SKC；按源行相对顺序保留 SKU。稳定顺序为各主值首次出现顺序、组内 sourceIndex 升序；同 selection 输入顺序变化先规范为该顺序。输出仅来自真实源行，不做颜色×尺码笛卡尔积。
3. 只替换主/次销售属性及必要容器结构；每个 SKU 深复制其准确旧行其他业务字段。一个主值一个 SKC，SKU 次属性集合用已确认维度；同组组合重复拒绝。
4. 源→平台主值合并若会把两份非空 SKC 元信息（supplierCode、名称、色块图等）合并冲突，或原一个 SKC 的非公共元信息要分配到多个新组，须定位并停止本次预览；不静默取第一份、复制编码或丢弃字段。初始空 SKC 容器可按明确新组分配，保留 SPU 公共文字；有内容的普通稿仍受此保护。
5. SPU 商品属性、文字、尺码表、样品、包装、条码、单 SKU 价格库存等不属于销售属性映射的字段保持；所有关联原 SKC/SKU 位置或属性值的结构必须按同一行双射核对，不能仅改 SKU 数组后让样品等引用指向另一行。无法无歧义保持时明确报冲突，不静默删除、改值或复制。实际组合的 linked rules 必须按新 draft 重新读取。
6. 按当前规则再次重算，返回前后逐行展示与完整 input；任何 selection/基线/店铺上下文变化清掉预览，重新确认。映射失败保留表单选择，不能显示保存成功或部分应用。

## 6. 库存、价格与 R1

本命令**只映射销售属性，不在同一确认里覆盖价格/库存/文字**，避免与既有 SheinImport 双重写入。用户先沿已批准的 R1 带入流程把数值带到基线；本轮预览核对固定候选与基线的对应库存。fixture 为 0/未填/未填，如有差异定位 SKU.stock 冲突，回既有带入/编辑操作处理，不能静默选择其中一方。

黑 M 明确 0 对应原 warehouse 下 Quantity=0 的条目；黑 L/白 M 未填对应无 stock 条目，不能制造 `Quantity=0`。多仓情形不猜目标仓；R1 非空数值须能对上唯一明确的已选仓项，否则拒绝本次核对。R1 null 不是实际仓库零库存，也不自动清掉已有非零仓项来满足样例。

价格沿每个基线 SKU 深复制原字符串/币种/站点及 cost 字段。源 CNY29 与平台 fixture USD12.50 不做换汇、推导或覆盖。价格是否符合当前平台应用模式、库存未填、真实图片缺项仍由 native issues 显示；映射成功不是 ReadyForUpload。

## 7. 图片归属与资产边界

原商品图库、正文插图、R1 和任何 Asset fact 均只读。源公共图与源 SKU 图是原版本中的不同归属，不能因 URL 相同或 SKU 新数组位置变化而混同；原 35444 记录只有公共测试图时不声称存在白 M 专属图。独立的白 M 图仅作为后续隔离 fixture。

针对基线已有 `OfficialImageSlot`：

- SPU 公共槽保持原 AssetID/group/type/sort，不自动复制到所有 SKC/SKU；图从公共改专属、从专属改公共均不属于本映射。
- SKU 槽必须经已确认源行双射移动到同一源 SKU 的新位置；白 M 图只能到白 M，绝不能按新 index 复制给黑 M。AssetID、内容、类型、顺序及批准身份保持。
- SKC/色块槽仅在其原成员经映射全部属于同一新主值时移动。旧 SKC 被拆到多个颜色、成员不一致或旧槽用途不明即拒绝预览并定位，不复制给多个颜色、不丢槽；使用者先在已有图片编辑流程确定归属。
- 已批准 Asset 的 `Presentation.VariantID` / `OfficialPlacement` 若约束具体归属，服务端必须核对一致；Target 位置重排不能修改批准 provenance。无法在原批准条件下引用时，本命令停止并提示回现有 Asset owner 流程，不能在保存时顺带批准/改写 placement。

映射命令不自动把源图库图片转成批准平台槽：未批准或尚未绑定的源图仍在原 Catalog 归属下展示待处理；不伪造批准/remoteURL。保存预览绑定准确 ApprovedInventoryHash，保存重新读取并比较；图片观察沿原 owner 来源，不能接受浏览器给出的内容 hash/尺寸/远端上传成功。

## 8. 权限、保存、幂等与恢复

沿 supply.manage、verified org/user/member/token、SourceSelector、Completion/Target scoped reads 和 live Store Rules 权限。只读用户不能预览/确认新映射；跨企业/成员/源/store/recordId 错配拒绝。导航查询参数不改变 scope；切企业、登出、撤权、过期取消表单请求和迟到结果，不放行新写。

Preview 不持久化、不发授权令牌。确认沿原 durable command key + **完整规范 TargetInput** + expectedRevision。新写服务端从准确 base/R1/Source 和当前 rules/merchant/inventory 重建同一输出，比较整个 draft 和 selection/绑定；夹带任何未选字段修改、hash 漂移或基线 head 变化返回 409，禁止自动合并或偷偷换最新版本。输入大小上限沿原 2 MiB、映射最多 256 行、10s Target deadline；linked requests 沿既有分块/并发上限。

复用原 `TargetPrepared.Read` 提交检查：不可变引用及完整 inputHash/授权 proof 再核对，官方网络请求在事务前完成；慢规则读取后重新获取同一准确 Source 的短期 proof，事务等待超过期限按原合同拒绝/回滚。不声称本地事务能锁住外部规则或另一 owner；不引入跨库写事务。

同 Product DB 单事务追加一个 Target record、CAS 更新同一 head、保存原命令回执：任何一步失败都回滚。并发不同键同 expectedRevision 只有一个成功。**同 key/同规范完整输入优先返回原回执**，不因当前 head/规则/库存变化而重做；同 key/异 payload 冲突。新 key 携旧 expectedRevision 不能再保存成功；重新读取后主动再次确认是新版本操作，不能称其“重试”。

COMMIT/HTTP 超时、取消或响应丢失时，仅保存原 key/payload 查原命令或同键重试；沿现有 useSupplyCommand、FindTargetCommand，不另造恢复 worker/新 key，不补偿删记录。回执显示本次版本，显式重读最新 head；原回执不能冒充当前内容。取消预览不写任何记录。

普通稿无此绑定仍可经原人工编辑保存，但整个服务端共享 native builder 的 §3 发布阻断仍然生效；不能靠移除绑定、Agent/Operation、上传 plan 或直接 record 路由绕过。新增映射元数据不是新发布权限，后续发送继续重读准确 head、Source、Asset、Store/rules 并经原 Submission owner。

## 9. 诊断与用户操作

复用已有带入预览/字段组件、主题、差异表和返回上下文，不新建任务中心。用户一次选择维度和不同值的对应，查看“同一草稿、2 颜色组、3 SKU”的逐行结果后确认保存新版本；未确认项禁用该确认。无原生 confirm。

错误绑定源 sourceIndex + 显示编码 + 维度/字段 + 新位置：例如“黑 L / 尺码：所选值已从店铺模板停用”“白 M / 图片：该槽不能从原黑色组迁入”。库存 0 显示 `0`，空值显示“未填写”；不写成缺货。规格集合未确认、库存或图片不足仍可看草稿详情和继续编辑，但不得展示可发布或构造发送 payload。

范围内 UI 只服务上述明确任务；不强制 Figma 菜单层级，不增加未经授权的人工审批步骤或泛化规则配置页。未保存返回复用已有应用确认弹窗、取消/Esc/焦点恢复与内容保留。

## 10. 后续实施验证与交接（本轮 NOT_RUN）

正式实现需后续授权；先写能捕获当前缺口/错误发布诊断的 RED 测试，再最小实现。复用现有 domain、HTTP/BFF、fresh PostgreSQL 及浏览器合成环境，不改原库、不开发新验收平台。

| 必需验证 | 判断标准 |
| --- | --- |
| 三行映射/本地保存刷新 | 一个普通 Target、2 SKC、仅黑 M/黑 L/白 M，源/R1 摘要不变 |
| 数值 | stock 0/无条目/无条目；价格币种与 warehouse 精确保持，无转换 |
| 图 | 公共图仍公共，合成白 M 专属图随该行重排；无源图自动批准；冲突槽拒绝 |
| 属性与集合 | 正确官方维度/值；组内重复、主 ID/次维度不一致拒绝；2+1 可保存但 variant_set_unconfirmed、非 Ready、无 wire |
| 同根因消费者 | 人工保存、材料校验、Operation/Plan 和发送前构建共用诊断，移除映射绑定也不能放行稀疏集合 |
| 绑定/篡改/漂移 | source/R1/base/store/scope 错配、未知字段/漏0、无关 draft 篡改、规则/商户/库存漂移拒绝 |
| 原协议 | 同键重放、异载荷冲突、CAS 并发、单事务失败回滚、proof 等待过期、响应丢失后原键核对 |
| 浏览器消费者 | 预览→取消/重新预览→保存→刷新重开；完整 DTO/intent/历史/回执保留；基础桌面和窄屏键盘路径可用 |

实现阶段只运行相关必要检查和既有必需 CI；完整用户路径形成后做一次集中独立最终检查，修复仅复核增量。模型、真实平台接受三行/发布、原 35444 部署及产品验收均为 NOT_RUN，不用 CI 或合成 fixture 替代。

Legacy decision：**EXTRACT 当前合格能力，不引入 legacy owner/兼容/迁移/双读双写**。本设计只追加现有 JSON 操作绑定，后续发布前后端同步；不允许旧解析器丢字段读写，不提供保留实例降级或迁移方案。#1225 schema 修复独立保留。

## 11. 历史拆色方向与评审记录

本设计取代 `shein-color-split-drafts.md` 中“本例必须拆色以满足平台”的推进依据；#1220 历史代码/CI/隔离记录不删除，不擅自整体回滚、删表或删除已有隔离草稿，也不把其 localOnly 子稿转换为普通稿。当前路径不调用 split、不依赖 family/partition，不继续原 35444 的拆色开通。

独立 Architecture Review R1 已完成，准确候选 `912557a43316ecfe3bb562cfbc73ea03a4b6d4c9`，结论 **IMPLEMENTATION_READY，0 BLOCKER**。Reviewer 实际核对当前源码/合同、三文件文档 diff，并独立只读读取官方正文，确认 doc3002086/version159 及未完整示例；未执行实现测试或真实业务操作。报告位于 `C:/Users/Henry/.codex/private/issue1158-sales-attribute-mapping-20261109/architecture-review-r1.md`。

| Finding | Classification / 层级 | 实施处理 |
| --- | --- | --- |
| IT-1 准确来源、完整行双射和零/空数值 | IMPLEMENTATION_TEST，实施/合并前 | 服务端完整重派生、严格0位置/来源、分支互斥及旧一次性绑定清除；完整 DTO/hash/回执/原键/CAS/事务/proof，不能仅比较元数据 |
| IT-2 非空 SKC、关联结构和图片批准归属 | IMPLEMENTATION_TEST，实施/合并前 | 全部非公共元信息及索引引用保持，冲突停止；公共图不复制、专属图只随原行，尊重原批准条件，不写 Asset |
| IT-3 稀疏集合跨消费者发布阻断 | IMPLEMENTATION_TEST，实施/合并前 | 共享 builder 非 Ready/无 wire，普通编辑移除绑定、材料/Plan/Operation/发送重建仍不能绕过；确定属性错误独立保留 |
| IT-4 native 诊断版本与历史 | IMPLEMENTATION_TEST，实施/合并前 | 新执行版本/绑定及实际消费者同步，历史结果和原回执不倒改；沿原协议，无新迁移/fallback |

上述要求细化已明确的 owner/合同，不改变用户 Must，不增加新的权限/状态/事务/持久化/副作用边界；落仓后只做对应文字增量复核，不重复全局设计审查。实现开工必须同时满足设计准入及用户阶段授权；IMPLEMENTATION_READY 不自动授权本轮未包含的生产编码或真实操作。
