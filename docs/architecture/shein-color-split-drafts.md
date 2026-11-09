# SHEIN 按颜色拆分本地平台草稿：最小合同

Design Basis: Independent Architecture

Admission Status: NOT_READY — 待独立架构评审；本轮仅设计，不授权正式实现。

依据：用户 2026-11-09 明确要求原商品和候选 R1 不变，黑 M/黑 L 一份、白 M 一份；明确身份、SKU/图片归属、幂等与重试，先独立评审。执行进度归 [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158)。调查基线 `main 7c480d513660f5aa93bee92722c3bd4ad44ea0ce`。

## 1. Product Outcome / Must / Out of Scope

运营人员从已保存的 SHEIN 基础平台草稿与固定候选 R1 发起按颜色拆分预览，确认后得到两份可以分别重开、在固定 SKU 边界内编辑的本地平台草稿。原自有商品、Catalog publication、SourceItem、CompletionDraft R1 和原 Target 记录/head 不变。

| 分组 | 来源固定行 | 新草稿的完整 SKU 集合 | 库存 | 本轮基线价格 |
| --- | --- | --- | --- | --- |
| 黑色 | sourceIndex 0、1 | QA1195-BLK-M、QA1195-BLK-L | 0、未填写 | 各 USD12.50 |
| 白色 | sourceIndex 2 | QA1195-WHT-M | 未填写 | USD12.50 |

Must：两份都创建或都不创建；每个选定源 SKU 恰好归属一份；没有白 L；0 与空区分；不自动换汇；图片明确分配；响应丢失能用原键查证；原资料不变。初次拆分不是多次普通 `POST targets`。

范围仅 SHEIN US、自营、已保存基线、已有结构化颜色主销售属性、恰好两个 SKC/两个不同的官方枚举颜色值，最多 256 个来源 SKU。本轮实例为 2+1，不把值 12/34/35 或中文颜色名硬编码。其他平台、托管模式、自定义颜色值、任意多组、AI 自动分色、SKU 新增/删除/跨组移动、自动生成图片、换汇、库存补齐、重组/删除拆分家族不开放。

没有发布、上传、Submission/Plan/Operation/Agent adoption 接入；不改变已有发布许可，不宣称拆分稿可发布。未来上传消费拆分目标是新的有界任务，必须设计准确 target/record、远端 SPU/SKU 归属与原键恢复，不能仅删除禁止标志。

本轮不写生产代码、schema 或夹具，不操作原 35444 配置/数据，不调用模型、保存/发布业务资料或释放原 15.445282 AI 点预留。后续实现、隔离运行、合并、部署和用户验收分别授权。

## 2. 产品依据、平台证据与现有能力

最新用户决定及当前领域合同优先。Figma 只作业务流程辅助，不作布局、导航、组件或截图验收权威。按 [AGENTS](../../AGENTS.md)、[当前 owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[既有候选带入合同](completion-shein-import.md) 与[结构化候选合同](completion-draft-structured.md)设计；旧文档中冲突的 Figma 权威引用不恢复。

2026-11-09 只读查阅 [SHEIN 商品发布/编辑](https://open.sheincorp.com/zh/documents/apidoc/detail/3001352-1000001)：不同 SKC 下 SKU 销售属性及属性值数量须一致；新增时库存必填，0 有效；多 SKC 色块图要求与店铺图片规范分别存在。本地 `goods/attributes.go:194` 实现完整组合集合一致校验。本轮 2 对 1 确有官方依据，不删除校验。拆分后每稿一个 SKC，按各自当前规则重新 BuildOfficial；缺库存/图片仍可以保存本地草稿但保持非 Ready。

事实调查：

- `TargetIdentity` 当前仅包含 org/actor/source/store；`listing_preparation_targets` 对 org/actor/source/store/site 唯一。
- Target 已有 immutable record/head/command、PostgreSQL 事务、advisory locks、CAS 和 UNKNOWN 原键查询；复用，不新建 Saga/outbox/通用批处理平台。
- `CompletionDraft` 结构化 SKU 是固定 `sourceIndex`，不直接存颜色事实；颜色来自已保存平台 SKC 的官方销售属性，来源行与平台 SKU 的对应必须由用户明确选择并由程序检查结构。
- `listing-page`、upload plan、operations、材料 Agent 多处按原 `TargetIdentity` 查找。这是新多目标身份的真实接入缺口，不能只放宽数据库唯一键后让旧消费者自行选第一份。
- 上轮隔离记录证明创建/带入/保存/重开及 0/空、USD12.50、SPU 公共图片保持；它不是实际企业/Store/官方批准资格。此前所有运行结果均不冒充本设计的实现证据。

## 3. Owner 和 contract → implementation → injection → consumer

| 事实/合同 | 当前 owner | 处理 |
| --- | --- | --- |
| 原商品、源 publication、固定 sourceIndex | Catalog / ProductCollection / SourceSelector | 只读准确版本，不复制商品或制造第二 Source |
| 候选 R1 record/content | CompletionDraft | 只读指定 recordId/revision，head 后续增长不静默换版本 |
| 店铺连接、规则、授权 | Store + 官方规则 reader | 沿当前 scope/live access/expiry 检查 |
| 图片批准与真实维度 | Asset + 既有 ApprovedInventory/Image observation | 引用同一批准身份，不复制 Asset、批准或 URL 事实 |
| 两份颜色草稿及拆分命令回执 | 现有 `internal/listing/record/target` | 新 split 方法和 typed partition；原 records/heads 为正文唯一 owner |
| 原子写 | 现有 `integration/persistence/listing/record` / Product DB | 一个事务保存两份 record/head 和一个 split command |

Console/BFF 严格 DTO → `app/listing/httpapi` → TargetService 的 PreviewColorSplit / CreateColorSplit / ReadColorSplitCommand / ReadColorSplitFamily → 现有 Source/Completion/Store/Asset ports → Target 的受保护 prepared proof → 同一个 TargetRepository 的原子 split commit。由当前 native Product runtime 的构造与权限 grant/installer 注入，不在 UI/HTTP handler 中拼 GORM 或官方凭据。

UI 复用 SHEIN 带入页、差异预览、原操作键恢复组件、Target 编辑字段和只读历史页。一个“按颜色拆分”任务显示两份预览，一次确认；完成返回同一商品的两份拆分草稿。原列表仍是一件来源商品，明确显示“本地拆分草稿 2”，不把它计成新增自有商品或上传成功。返回保留 preparation/store/item/filter；取消预览无写入；不用原生 confirm。

## 4. 身份、持久化和路由

下面是拟实现的明确增量，不是现有 API。

- `familyId = StableID(org, actor, "shein-color-family-v1", sourceId, storeId, site, colorAttributeId)`。不包含本次 command key、候选 revision、颜色文案或数组顺序。同一来源/店铺/属性只首次创建一个家族；以后不通过换候选或换键重复拆分。
- `partitionId = StableID(familyId, colorAttributeId, officialColorValueId)`；本例两份有不同 partitionId。颜色名仅展示，重命名不改变身份。
- `targetId = StableID(org, actor, "shein-color-target-v1", familyId, partitionId)`；child record ID 用原 key + partitionId 的专用 namespace；各稿从 revision 1 开始，后续各自 CAS。
- 原普通 Target 的身份公式、记录/head 内容及读取不变。Target record 增加必需的 typed `colorPartition` 分支，包含 familyId、partitionId、颜色 ID、基线 record 引用、候选 R1 引用、固定 SKU membership 和 `localOnly`。不是缺字段猜测、旧 ID 映射或 fallback。
- 现有 `listing_target_records`/heads 增加明确 `purpose` 和 `partition_id`：普通稿为 `standard`/空；拆分稿为 `shein_color_local`/非空。CHECK 限定合法组合、site 与颜色分支；head 唯一键扩为 org/actor/source/store/site/purpose/partition_id。record/head、ID、body hash、scope、member、purpose/partition 列与正文必须一致，FK/仓库验证覆盖。
- 新增仅一个 Target-owner 的 `listing_target_split_commands` 表：org/actor/member/key/inputHash/familyId/baseRecordId/completionRecordId/两份初始 target+record+revision 引用/createdAt；PK(org,actor,key)，UNIQUE(org,actor,familyId)。外键含 scope，保证两个孩子均属于本命令的 source/store/site/family，回执不能引用别家或别家族。该表只持命令与初始版本引用，不再存两份可变草稿正文或第二 head。
- 拆分后编辑仍写原 Target records/heads/普通 record commands；split 回执永远指向首次创建记录，family 当前读取显式返回各孩子最新 head，不把旧回执冒充最新。
- 原 source/store 读取继续只返回普通稿。拆分稿只能通过明确 family/targetId 路由读取；服务端重新核对 source/store/preparation/item。不能用颜色字符串或 source/store 随机取一稿。

Schema 按新安装显式 installer/运行权限合同实现，serving 仅 verify，缺列/表 fail closed。没有迁移、回填、双读双写或兼容分支。原 35444 既有数据不在此变更授权内，也不承诺可原地部署；后续若要更新该保留库，须单独明确数据保留与安装路径，不能启动时自行 ALTER。

## 5. 固定 SKU 分组与数值

请求绑定准确 baseRecordId/revision、effectiveVersion/applyReceiptId、completion recordId/revision、store/site、官方 colorAttributeId 和两组 valueId。用户明确提供完整一对一 `sourceIndex → baseSKCIndex/baseSKUIndex`，本轮恰好 3 条；来源 SKU identity、supplier_sku、基线位置三者由 server 固定读取并展示确认，不从提示词/标题正则猜测颜色。

服务端验证：source/candidate/base 同 scope、同 SourceItem 和源版本；base 是当前普通 Target head；官方主销售属性可用且正好两个非空不同枚举颜色；sourceIndex 完整覆盖固定候选和源快照，无重复/遗漏/新增；base 全体 SKU 被映射且仅一次；supplier_sku 唯一。行号只是版本内定位，不是跨版本身份。

按基线已存 SKU 深拷贝分组，不生成新 SKU，不重建笛卡尔积，不切换颜色/尺码属性、不把未知库存当 0。R1 指定库存与基线有冲突时返回可定位 SKU/字段的冲突，由用户回普通补全流程处理；本轮要求 base 对应库存 0/空/空。保留每个 SKU 的原价格与币种，不从 CNY29 推导 USD12.50。基线文字/多语种/类目/品牌/包装等保持复制，描述提及其他颜色时预览明示“公共说明原文复制，可后续手动编辑”，不做语义重写或 AI 调用。

每组只保留自己 SKC，重排为 child SKC 0，SKU 顺序保持原相对顺序；尺码表仅保留明确关联到本组实际次销售属性值的条目，共享无关联尺寸项原样保留。无法确定关联的结构冲突不静默丢弃，定位字段并要求重新预览。linked rules 按每份实际保留的 SKU 组合重新查询；不使用父稿跨颜色的规则组合。

后续编辑可改文字、批准图片位置、价格或明确库存，但禁止增加/删除 supplier_sku、改 membership/颜色/销售规格、跨组移动或换 source/candidate 归属。membership 不可变且由原 split receipt 可追溯；每次新 revision 仍是当前 Target owner 校验。普通保存/候选带入/Agent/材料建议入口不得绕过此边界；不支持的入口明确拒绝拆分 purpose。

## 6. 图片归属与 Asset 复用

候选 R1 本身没有图片归属 owner；从原商品全部图库自动分配图片是不允许的。只消费准确 product/effectiveVersion/platform/purpose 的批准 inventory 与已保存基线 typed image slots。

- 预览逐张显示来自基线的公共 SPU 图片，用户明确勾选其复用到黑稿、白稿或两稿；默认不把图库所有图放入两份。输入是原 slot 定位 + 分配到的 partitionId 列表，不接受任意 URL、asset scope 或 approval 结果。
- 本例总览图经用户明确确认后可在两稿保持 SPU 公共图；两处引用同一 Asset ID，不复制存储、不变成 SKU 专属图。图片不是读取标题里的“公共图”自动判定。
- 原 SKC/detail 图只能随所属颜色走；SKU 专属图只随固定 supplier_sku 走。按固定映射重排 child SKC/SKU 索引；绝不因为父稿 SKC 1 变为 child 0 而落到黑稿。
- 不再需要的父多 SKC 色块槽在预览中明确列为“不带入”，不自动改成主图；用户要保留的图必须是当前单 SKC 规则允许的类型。未匹配/不再允许的 slot 给出字段提示，不丢到另一组。
- 只有当前准确 inventory 可用的图片能入选；批准/版本/hash 漂移时新保存拒绝。已有 source URL、批准 hash 与 observation 原样核对，不把新增分组当成新批准、免费生成或可发布证明。
- 使用每稿实际图片规范重新 Resolve/Build，缺黑/白主图继续待补全；不以公共图自动补 SKC 主图，不自动裁色块/生成图片，不虚构其资格。材料/AI图片流程未适配该 purpose 前不开放；现有合格 Asset 可复用。

## 7. API、预览绑定与普通编辑

拟增量：`POST .../shein-color-split-preview` 只读；`POST .../shein-color-splits` 显式确认；`GET .../shein-color-split-commands/:key` 原键核对；`GET .../sources/:sourceId/color-splits/:familyId` 返回组列表/当前 head；孩子编辑使用显式 targetId 和 expectedRevision 的现有 Target 保存路径增量。

预览包含规范化 plan（固定引用、完整 SKU 映射、图片明确选择、两稿派生 input、规则/merchant/inventory hash、planHash、localOnly），客户端不拥有派生业务值。排序仅规范 ID/映射列表，不舍入数值、不折叠 0/null；重复/缺字段严格拒绝。预览不是许可令牌。

确认发送完整规范 plan + 本次 command key。server 重读准确 refs、current base head、权限及各稿规则/Asset/Store，重新派生并比较完整计划/hash；只比较 metadata 不够。任一漂移 409，保持输入、让用户显式重读和预览，不静默覆盖/换成候选 head。每份 BuildOfficial 的 issues 完整保留；业务不完整可以保存，结构/归属/权限错误不能保存。

孩子新写必须由 TargetService 从已提交 family/partition 与 immutable membership 重建并校验；普通 HTTP 随意加 colorPartition 不能创建孩子或篡改归属。首次创建专用 split proof，后续编辑复用 CAS/command proof 并强制 localOnly；任何不支持分支的消费者 fail closed，不使用 standard identity 回退。

## 8. 原子事务、并发、幂等、UNKNOWN

只有 Target PostgreSQL 被写入；外部 owner 只读。当前 shared transaction/UoW 足够，不引入跨库 Saga、reconciler 或补偿删除。

1. 新请求先查同 scope/key 回执；同 hash 已提交时按当前读取权限返回完整原回执，不受候选/head 后续变化影响；同 key 异 hash 返回 409。
2. service 在事务外完成两稿的读取、派生、规则/图片核对和 prepared proof。没有正式 provider 写调用，规则查询/既有图片只读观察沿原受控 reader/deadline，不在数据库事务中发网络请求。
3. repository 一个事务，按统一排序取得 command、普通基线 Target、family、两个 child target 的 advisory locks；同 key/同 family/普通稿编辑用一致锁 ID，避免 ABBA。查原命令；查 family 已存在；核对普通基线 current_record_id/revision 与两个 child head 都不存在。
4. prepared.Read 在提交边界复核固定 scope/Source/Completion refs 与有界有效性；新的 live 授权检查按现有合同。写两份 revision1 records、两个 heads、一个 split command，最后复核 proof，再 COMMIT。任何一步失败全部回滚，绝不先调用两次现有 SaveTarget 形成部分成功。没有第三份“合并后正文”。
5. 两个 key 并发首次拆分同 family：一个全成功，另一个 409 `SPLIT_ALREADY_EXISTS`，返回有读取权限的已存在 family 导航；不能覆盖或偷偷新增第二家族。唯一约束是最终防线；同 key 并发只返回同一回执。
6. COMMIT 或 HTTP 响应不确定：`UNKNOWN`，保留原 key 和完整 payload。先原键只读核对；找到则显示“两份已保存”的初始回执，当前版本另读；未找到不是未提交证明，只能原 key/原 payload 重试，由同锁与唯一键判断。不可另建键、盲重拆、删除某稿或写补偿。
7. 取消仅取消浏览器等待，不能撤回已提交事务。页面刷新/重启恢复原 scope/source/store/family/key/payload；企业/用户/member 切换不把旧 pending 请求发往新 scope，沿现有 pending owner 契约恢复。找不到或无权限仍保持未核实，不显示未保存/成功。
8. 两个孩子后续各自 expectedRevision CAS；对黑稿的保存不会改白稿。原 split command 只读不更新。新候选 R2、新基线或重新分组不自动重建现有 family，超出首版固定 membership 的变更需另定范围。

外部授权/Asset/Store 撤销与数据库 COMMIT 之间不声称分布式原子；继承当前短时 live-check/expiry 合同，不扩展到锁住外部服务。这里仅保存 localOnly 资料，没有支付、库存扣减、上传或不可逆副作用；未来发布必须重新授权并检查当前资格。

## 9. 权限、Threat Model、资源与准入

读沿 supply.read，预览/创建/编辑沿 supply.manage；org/actor/member/token 从验证后的服务端上下文解析。按当前 owner 资格核对所有 source、candidate、base、family、partition、Asset、store/site，跨企业/他人/成员替换拒绝。浏览器路径参数、name、color、assetID、localOnly 布尔均不是授权依据。

当前 Must 威胁：伪造/跨租户引用、同键异载荷、重复创建、部分成功、SKU 丢失/重复、索引重排串图、0/空混淆、偷换币种、陈旧预览覆盖、新 identity 被旧上传/Agent消费者误用。客户端可以恶意改 DTO，server 必须完整派生核对。范围外：真实平台可用性/审批时效保证、图片内容自动语义识别、任意组数/重组、跨人协作扩权、生产发布和未授权数据迁移。

2 MiB 总请求/aggregate 记录边界（不是每孩子无限放大），恰好两组、最多256来源 SKU；沿现有 Target 10s request/deadline、解析/数组限额。两稿都在该总 deadline 内准备；超时无自动新请求。错误定位 group/sku/field，不打印凭据、完整图URL或原模型内容。

普通 `ReadTarget`/plan/operations/上传保持只消费 `standard`；拆分 purpose 必须在 handler、Target/Submission owner 和 repository 的资格校验中拒绝，包括直接 recordId、executeForAgent、materials、image execution 等可触达 sibling paths。`localOnly` 由 durable purpose 决定，不能靠 UI 隐藏按钮。并不封禁或改动普通父稿的现有路径，孩子不会抢它的 identity。

Legacy decision: N/A；新增当前事实，不迁移/兼容旧系统。复用当前 SQL/事务、类型/JSON、React组件及错误恢复；没有成熟开源库能代替该领域身份合同，也不自建基础框架。

## 10. 验证、验收与停止点

实施获准后复用现有 domain、HTTP、PostgreSQL 和浏览器夹具；不新建 runner 或专项验收平台。TDD 的必要场景：

- 固定例仅黑M/黑L与白M，supplier_sku/sourceIndex coverage完整无白L；父商品/R1/父Target全不变；库存0/空/空、USD12.50保持。
- 黑白公共图显式双引用；白M专属图只在白稿；父SKC1→childSKC0索引正确；非法Asset/错误组/漂移拒绝。
- 写第二稿、head或回执时故障全部回滚；同key并发/原key重试只两稿；异key同family冲突；Commit响应丢失原键返回两份原回执。
- 旧基线/换候选/同键不同图选择/越权引用/后续编辑变membership拒绝；黑稿CAS不改变白稿。
- durable localOnly不能被直接 recordId、旧plan/operation、Agent/材料/上传或伪造payload绕过；原 standard flow未变。
- 浏览器预览→一次确认→两份回执→刷新重开；返回取消不保存，未知结果保留原键；复用现有键盘/主题/响应式组件，仅验证本轮受影响流程。

开发自检、CI与独立代码复核不签发用户验收。原35444、真实Store/官方写入、模型、发布、账本均NOT_RUN。交付后先由用户决定是否实施及隔离验证；本轮在独立 Architecture Review 后停止。

## 11. 独立评审记录

待评审。Finding 须按 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE 分类，说明当前 Must、用户后果与阻塞层级；最多两轮正常架构评审。冻结后无新 BLOCKER 不重开架构。
