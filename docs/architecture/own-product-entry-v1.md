# 自有商品独立录入页与多变体 v1

Design Basis: **Independent Architecture**. Admission: **IMPLEMENTATION_READY / FROZEN**.

## 1. Product outcome and authority

2026-10-13 当前用户决定：自有商品需要多变体；修复深色模式；“用新的页面来做，不要使用模态框”。
获授权的当前企业操作人在“我的数据 → 自有商品库”进入独立新增页，录入一款商品和颜色/尺码等变体，保存后返回库并读取准确原始资料。

Figma Authority: [final UI / IA](../product/final-ui-ia-authority.md), `tg48P46SSXl6TBy9lZwg63 / 31:463`, 自有商品库 `3500:1100`、添加 `3502:405`。本次已读取添加节点的高保真 design context 和截图。其商品资料入口、工作台归属和视觉语言继续复用；用户本轮决定明确取代添加弹窗交互。完整新增页按既有 ConsolePage、Card、Input、Table 和工作台 theme tokens 组合，不增加顶层商品中心。没有新增页 Figma frame，不声称它已存在。

Scope / Must:
- `/workbench/data/mine/own/new` 独立页面，库入口使用正常导航。
- 商品名称、品牌、描述、公共图片；单规格及多规格；规格组合、按变体 SKU/币种/来源价格/来源库存/图片；公共值批量填写；显式删除不销售组合。
- 相同组合重新生成时保留已填字段及 SourceID；规格变更不自动生成/删除行，用户显式生成时提示移除旧组合。
- 保存沿原 create_product；读回展示全部变体及对应图片。新增及自己的商品详情使用正常页面，非模态框。
- 浅色/深色统一使用现有主题 tokens；修复同一 CollectionDialog 及 source uploader 的硬编码亮色表面，保留其他弹窗业务行为。

Out of Scope: 更新已保存原始商品（原始版本不可变）、Excel 多规格分组格式、库存实时管理、订单/采购/资金、AI、平台刊登、自动适配、新媒体/保存/恢复平台、schema/历史迁移、部署/合并/关单/共享实例数据操作。

## 2. Owners and call path

复用 [my-supply-chain-shein-v1](my-supply-chain-shein-v1.md) 的 Source/Catalog/Collection 原子保存、actor-private 权限和 UNKNOWN 合同。

`ownProductSchema / OwnProduct → create_product → Collection.Service / OwnEnvelope → 原 Product UoW / NewOwnProductWriter → SourcePublication / Catalog version / Collection reference → 原 ReadItem + exact VersionedSnapshotReader → 独立新增页/详情页`。

Collection 持操作、分组和来源引用；Sourcing 持用户提供的证据及 publication；Catalog 持不可变原始版本；Asset 仍持显式批准。UI 本地表单只是未保存输入，不能签发已保存、适配或发布事实。不增加表、schema 或第二份商品正文。

## 3. Contract delta and invariants

唯一正式合同扩展：`OwnVariant.images?: string[]`，最多40个HTTPS图片链接，限制与公共图一致；转换成现有 `ProductVariantCandidate.Images`。复用现有 Catalog 变体图片能力，无新媒体 owner。
每张变体图片 SourceID 用 `StableID(productID, variant.SourceID, URL)`，Role 按变体内顺序 main/detail。公共图片仍用原 `StableID(productID, URL)`；变体图留在变体内，不扩散到每个其他变体。图片永远只是候选，保存不批准、不下载、不调用平台。

新增页只挂一个原 SourceImageUploader：先上传到公共图片池，再逐变体选择已填/上传的图片，避免多个 uploader 竞争同一既有全局 pending-media key。不修改媒体恢复合同。

每个变体 sourceId 在新表单生成一次（UUID），与可编辑 SKU 分离；相同组合再生成保留该ID。保存合同拒绝同一商品内重复 sourceId、重复非空 SKU；检查字符串、属性数量、有限非负价格、整数非负库存、币种格式及HTTPS图片。未填数值不自动成为0；用户显式填0可保存。单规格未填SKU可沿现合同保存基础资料，不伪造变体。

只读评审发现现 `sourcing.snapshotVariants` 仅正价格/成本时生成Price，导致自有录入显式0价丢失币种；这是当前Must的实现缺陷。最小局部修复限定已校验自有来源（SourceType=user_input、SourcePlatform=own_product）保留其声明的0价，不让未知来源缺失价格成为0。Catalog非nil Price编码明确amount（包括0）；自有详情按原OwnVariant必填stock合同显示0。通过OwnEnvelope及准确PG/JSON读回测试收敛，不新建事实owner/schema或通用presence协议。

HTTP严格结构解析之后，在同一有界raw request内检查product/products的每个variant必填price/stock：缺少或null拒绝，显式数字0保留。不能让Go值类型的默认0替代用户声明。未知/重复字段仍由原sigjson严格解析拒绝，不用自定义解码绕过现严格解析。

实际 source publication 单集合上限256、aggregate items上限1024、单字符串8192 bytes、编码2MiB。现 collection/UI 标称1000变体、1MiB描述高于真实下游合同，不能将其当作真实保存能力。本轮对自有输入统一到真实上限：最大256变体，description最大8192 bytes，并计算属性+变体+公共图+变体图合计预算；过大输入在正式写入前拒绝。生成先计算笛卡尔积及预算，再分配行，不放宽底层限制。

规格名称和值trim，重复名称/同维重复值拒绝。行属性由显式规格值产生，组合键以排序后的属性键值JSON确定；SKU不参与组合键。批量修改只在用户点击时应用选定公共值，不覆盖未指定字段。删除图片时移除对应选图引用；相同图片可由用户明确分配给多个变体。

## 4. Authorization, transaction, idempotency and recovery

权限/租户完全沿现 verified effective organization + actor/member + workbench.collection.manage/read。页面等待身份/企业稳定，以 org+actor 重新挂载表单，切换不带走上一个身份资料；请求携带原 expected org/user。服务端仍实时核对权限，不能依靠隐藏按钮授权。

持久化仍是原 Product UoW：用户证据、Catalog publication、Collection 引用和命令回执共同提交；失败全部回滚。相同scope+key+payload重放，异载荷冲突。扩展图片进入原 payload hash 和 raw evidence checksum。

新增页复用原 pendingCollectionIntent 存储与恢复：发送前保存精确原键/载荷；pending时锁定表单；结果不确定先查原键，未确认时保留意图，显式重试复用原载荷；禁止用新键重试。共享命令hook从现 CollectionPage提取既有行为给两消费者使用，不新建恢复机制。原媒体pending仍原owner处理。保存成功且清除pending成功后才导航到自有库；read-back失败显示错误而不是成功正文。

页面读取自己的商品详情沿原ReadItem；不存在/跨actor/跨org拒绝，不接受客户端productKey替代item授权。源商品不会因本轮而被改写，后续适配继续固定原source版本和明确Asset批准。

## 5. Legacy and verification

Legacy decision: **N/A** — 使用当前Product/Collection/Console；不触碰RETIRE owner。删除旧新增模态表单属于当前交互切换，不新增兼容入口。

TDD:
- RED→GREEN捕获6个颜色尺码组合、组合更新保留已填行/稳定ID、删除、空值与0、重复规格/SKU、图片绑定和预算拒绝。
- 服务端OwnEnvelope/HTTP严格解析测试图片保持变体、非法URL/重复身份/超预算在写前拒绝；既有publication/PG读回测试证明准确Catalog变体及图片，原键重放/异载荷不重复资产或商品。
- 页面保存、UNKNOWN原键查询/重试、权限/身份变化、正常新增链接、全部变体详情读回。复用既有测试，不建新验收runner。
- 必要typecheck、scoped lint/Go tests、既有适用CI。稳定完整候选进行一次独立最终检查；修复只复核相应增量。
- 浏览器只在独立预览核对布局/规格输入、深浅配色，不向共享实例写测试商品。真实媒体/SHEIN/付费AI、共享部署、用户验收NOT_RUN。

Implementation complete enters In Review; 不签发产品验收/上线，不合并/部署/关单。

## 6. Review

2026-10-13 独立只读 `/root/own_entry_review` 核对实际 Source/Catalog/Collection、媒体和pending合同后给出 **IMPLEMENTATION_READY**，无设计BLOCKER。准入前只新增设计与捕获当前缺口的测试；正式生产代码尚未修改。

三个 IMPLEMENTATION_TEST（合并前）：显式0价/库存的准确PG+JSON读回；HTTP缺少/null数字与合法0、重复身份/SKU、图片/预算拒绝；提取hook后UNKNOWN原键/原载荷、NOT_FOUND重试、存储清理失败不导航、foreign pending/身份切换/供给命令阻塞。均在当前owner局部实现和既有测试收敛，不重开设计。

## 7. 用户反馈后的有界增量

用户后续明确指出详情预览过于简略，并询问商品属性填写入口。当前Issue #675的增量 `Design Basis: Reuse Existing Architecture` 在正式修改前补齐：新增页消费既有 `OwnProduct.attributes`，属性名称/值成对填写，整行空白忽略，trim后重名或未完成行拒绝；共用原256属性、1024 aggregate和2MiB预算。通用属性与每个SKU的销售规格分别保存。

独立详情页消费原ReadItem完整资料，增加真实图片的大图/缩略图浏览、选中SKU的对应图片与来源价格/库存、全部变体、通用属性、描述及保存信息；只改变录入和读取展示。没有商品图片或属性时如实显示未填写，不添加虚构素材、发布状态或可修改已保存原始资料的入口。原授权/租户、Source/Catalog/Collection UoW、状态机、媒体批准、UNKNOWN owner和schema均不变，不重开已冻结的独立架构准入。
