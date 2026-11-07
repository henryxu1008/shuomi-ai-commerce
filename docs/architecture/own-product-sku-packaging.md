# 自有商品逐 SKU 含包装事实录入

Execution: [#1113](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1113)。Design Basis: **Independent Architecture**。Admission: **NOT_READY / DESIGN_REVIEW_PENDING**。本文先于生产代码；评审证据绑定提交，不能由作者自行签发准入或产品验收。

## 用户结果、依据与范围

当前企业的自有商品管理者在原新增／编辑页为准确 SKU 填写真实含包装重量及长宽高，保存后在详情、手工历史和差异中回看。沿原详情的 SHEIN 离线诊断，显式使用新资料版本生成并检查新包；缺项仍显示未知，已有草稿和诊断继续指向原版。

用户夜间委派明确授权这条业务路径的设计、独立评审、开发、合成验证与 Draft PR，要求不与既有 Writer 重叠。#1108／Draft #1111 明确排除包装录入；本批堆叠基线为 #1111 `83055bf2945988619bde1c245749c3503b550175`，完整 tree `3e76bf7ed9acefbbb9a3049e3c4e45f82db11e9e`。已核对最新 open Issues／PR 及包装／own 分支，没有发现本范围后续负责人。#1111 原 owner／分支不写入。

产品依据为 [全新系统基线](../product/greenfield-no-legacy-migration.md)、[最终 UI/IA Authority](../product/final-ui-ia-authority.md)、[原录入](own-product-entry.md)、[版本更新](own-product-edit.md)、[手工历史](own-product-history.md)及 [SHEIN 库存与包装缺项](shein-offline-stock-packaging.md)。Figma `tg48P46SSXl6TBy9lZwg63 / 31:463`、自有库 `3500:1100`／添加 `3502:405` 是已有产品依据；复用原独立页面、Console 与主题组件，不声称已有本增量 frame。

Must：同 SKU 准确绑定、固定单位、部分未知可保存、输入校验、完整版本保留、正常保存／读取／离线复检、既有库存 0 与权限隔离。Out of Scope：真实采集、AI／付费／采购／平台写入；SHEIN 目标编辑器改造、库存同步、仓库／图片分配；自动升级或重做旧草稿；其它来源编辑、新表／数据迁移／历史回填、通用框架或测试平台；合并、部署、关单、保留实例操作。没有新增 Accepted Risk。

## 事实 owner 与精确合同

Collection 拥有 actor-private 商品条目及用户命令；Sourcing 保留每次不可变用户声明；Catalog 保留同一次 Product UoW 中的版本快照。包装是来源 SKU 的事实，不能放入销售规格属性、商品级规格、Store 库存证据或独立补丁表。

在 `OwnVariant`、`ProductVariantCandidate` 与 `catalog.Variant` 增加可选 `packaging` 值对象；共同的中性值类型由 Catalog 定义，Sourcing／Collection 沿已有依赖使用。字段如下，单位由字段名表达，首版不提供隐式换算或单位默认：

| 字段 | 含义／单位 | 未知 | 已知值 |
| --- | --- | --- | --- |
| `weightGrams` | 一件该 SKU 含包装重量，g | 遗漏或 null | 有限正数，<= JS 安全整数上界 |
| `lengthCm` | 含包装长，cm | 遗漏或 null | 同上 |
| `widthCm` | 含包装宽，cm | 遗漏或 null | 同上 |
| `heightCm` | 含包装高，cm | 遗漏或 null | 同上 |

包装整体遗漏／null、空对象及部分缺项均允许，表示对应字段未知；UI 空字符串不调用 Number 转为 0。0、负值、NaN／Infinity、数值字符串、数组、未知键拒绝，不能自动纠正为默认值。平台范围／精度由原离线诊断继续判断，来源保存不假装已经满足 SHEIN。四个数均可空，固定大小；既有 2 MiB 写入／Source／Catalog 限制、3 MiB 详情限制、SKU／总项数限制继续有效，无新集合或无限文本。

值对象字段用可空指针保留存在性；包装整体 `omitempty`，原无包装的 Source／Catalog 编码与摘要不增加字段。输入／Source 规范化和 snapshot 投影深复制四个数，不能让修改副本写回旧快照。原严格 HTTP 和 Zod 合同允许新增嵌套对象，并拒绝未知字段；null 规范为未知，不作为已知 0。

准确身份为原 `variant.sourceId`／`source_id`，不以数组位置、可编辑 SKU 编码或标题关联。表单行 ID 保持 sourceId；规格重生成只为完全相同组合保留原行，包含包装；新增组合包装全未知；删除行不把包装转给其它行。包装不参与规格维度推导。历史差异按原 source_id 对齐四个字段；改 SKU 编码仍属于同一个变体。

## 生产调用与消费者

`原 OwnProductForm → product-collection Zod → 原 BFF／严格 Collection HTTP → Collection.Service.Mutate → OwnEnvelope／OwnRevisionEnvelope → Sourcing.NormalizePublicationEnvelope → ToSnapshot → 原 Source/Catalog TransactionWriter → 原 Collection 引用／回执 → ReadItem／ReadOwnVersion → 同一表单／详情／历史差异`。

仍是完整商品声明，未提供的包装在新版本中是未知；清空已填数值即显式把新版本该项置未知，不改原版。使用原 `create_product`／`revise_product`，不新增 endpoint、publisher、命令、schema 或注入 owner。旧 Source／Catalog 字节在读取时按原 receipt/input/envelope/snapshot hash 校验，不重算为新值；保留无包装的固定向量验证。

`Catalog.ProjectCanonical` 为同一 variant 把包装投影到既有 canonical `Weight {value, unit:g}`／`Dimensions {length,width,height,unit:cm}`；只有该 SKU 提供数据才建立对应对象。部分缺尺寸在既有 SHEIN 内部结构中以缺项哨兵 0 传递，原 `formatSourceFloat(0)` 输出空白；它不作为已知零值保存。现 `buildSourceVariants → buildSKUDraft → preview → v2.3 sku_facts` 已逐 SKU 消费这两个字段，不回退商品规格、相邻 SKU、属性文本或目标编辑器数据。该映射是单向只读投影，不产生第二事实源。

公共 canonical-inspection Tool 使用严格输出 schema：本增量明确把 `packaging` 加到原 variant schema，消费原 Catalog 包装事实，不用丢弃字段规避验证；同一共享 projection 的商品／素材 inspect 注册表回归必须覆盖含包装快照，防止重复 #1111 的严格 schema 回归。除此不扩增 Agent 或 provider 行为。

## 版本、事务、权限与恢复

复用原 Product 单库 UoW：Source 发布、Catalog 不可变快照／head、Collection 当前引用／item revision／batch revision 和命令回执同事务提交。任何 commit 前错误全回滚。CAS 仍核对 item revision、准确 source publication/version；两个相同基准不同键至多一方成功。无跨库写、后台同步、事件或补偿状态机。

同 scope＋原 key＋相同完整载荷重放原回执；同键更改包装冲突。响应丢失／commit 不确定沿原 pending 冻结完整 product，包括包装，先查原 by-key；不得改数字或造新键自动重试。新页挂载 pending 时仍按原载荷恢复，同 SKU 字段不串值。

实时 verified actor／企业／membership 和 `workbench.collection.manage/read` 不变；不能从客户端指定 org／actor／productKey／版本或绕过活跃 own 条目。历史沿原 actor-private 授权和结束前重验。前端以 actor＋org＋item 挂载，换 scope 丢弃局部表单；明确读／写权限撤销隐藏可读包装及可写表单，不能在旧渲染中继续暴露它们；临时服务错误不伪装撤权，保留原恢复合同。新字段不授权其它来源／他人条目。

新保存只更新当前 own 引用，原已建 Preparation／离线包／报告固定来源不升级。旧版无包装仍是未知，新版提供真实数字；每版独立摘要。诊断入口用服务端新 item source version 构建新包／新键，原结果仍按原版本读取，不对旧回执填入新事实。

## 页面语义

复用原新增／编辑表单，在原 SKU 资料下按稳定行 ID 提供「含包装重量 g／长 cm／宽 cm／高 cm」四个有可见标签的输入。使用原 Input／Card／主题、响应式分组和可键盘操作控件；不再拉宽原规格表、不另造目标编辑器、不添加包装批量复制。说明“一件该 SKU 的含包装实测值；留空为未知，可保存后补”。新增尚未声明 SKU 时填包装不能悄悄丢弃，原 SKU／币种／价格／库存必填提示仍生效。

保存反馈、原键 pending、冲突重新读取和保存新版本沿原命令组件。详情与旧版复用原 OwnProductContents，显示当前选中 SKU 四项及未知，切换只换准确 SKU。差异仍显示按稳定身份匹配的真实变化。显示来源事实不宣称可发布，库存明确 0 保留。

## 验证与交付门禁

按现有 TDD：先增加能失败的定向用例并记录 RED，生产实现后运行同组至 GREEN；无环境执行不能记成 PASS。重点如下，不新建 runner：

1. Collection 严格 JSON／domain 校验，有限正数／null／缺项／非法形状；两个不同 SKU 及库存 0，输入副本不变。
2. Source 规范化／Catalog 同 SKU 投影／深复制、带包装摘要变更；无包装原固定向量不变；公共 inspect 注册表接受新合法输出。
3. 原实际 Collection HTTP create → read → revise → history／old version → SHEIN builder／diagnostic 的合成 PostgreSQL 用例：一 SKU 完整、一 SKU 部分未知；重排、改码、保存新版本、同键重放／异载荷冲突、撤权和旧草稿保持。扩展原 #1111 integration 测试，不替代未执行矩阵。
4. 前端模型／表单：准确预填／组合同值保留／新组合未知、包装不能变销售规格、清空不变0、pending 原载荷、库存0、详情／差异同源，换scope／撤权隔离。
5. 风险匹配类型／lint／必要 CI 与一次最终独立复核。云环境无 Go 工具链／前端依赖且外网禁用时，直接检查标为 NOT_RUN；通过已连接 GitHub 工具与既有托管 CI 验证，不绕过代理或改变 CI／权限规则。

可用入口沿 `/workbench/data/mine?view=own`，新增 `/workbench/data/mine/own/new`（以实际路由为准），编辑／详情／历史沿既有 item 路由。源码／CI 交付不等于实例升级、真实业务或用户验收；均按阶段保留 NOT_RUN。一个 Writer、一个主要 Draft PR 堆叠于 #1111；只有授权执行者未来按依赖顺序处理合并。本批 Legacy decision **N/A**。
