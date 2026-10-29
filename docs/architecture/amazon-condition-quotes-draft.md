# Amazon 新品和二手来源报价：设计草案

Status: **IMPLEMENTATION_READY**。关联 #955。用户要求先评审，再开发、测试并提交一个主要 PR；不合并、重采、更新实例或插件、部署。保留数据、配置、加载目录和插件身份。

## 用户结果和范围

在原 Amazon 单商品和同族采集的确认页、已保存规格资料中，分别查看该规格的新品和二手来源报价；网页明确给出二手成色时显示成色。每个状态只收集页面明确归属、无歧义的当前报价，不遍历其他卖家。参考报价不是本店售价或库存。原 1688 路径和服务端采集不扩展。

依据：用户 2026-10-28 的决定；[已有接收与原键合同](plugin-capture-ingestion-v1.md)、[来源报价展示](plugin-capture-source-quotes-v1.md)、[同族采集](plugin-capture-amazon-variants-v1.md)、[产品与 Figma 权威](../product/final-ui-ia-authority.md)。当前 #955 有真实页面和 v10 收发证据。Product/Figma 沿现有确认和原始资料入口，不新增导航。

## 事实、归属及未决合同

页面报价按单个子 ASIN 和明确 new/used 区域独立读取；可见金额须绑定当前 ASIN 与报价状态。币种独立取自同一报价明确元数据、无歧义符号或金额/ASIN 一致的表单，不要求可见金额必须有表单才能记录。不同状态不互相抵消；同状态内冲突、不匹配、不可归属时该状态不记录或币种留空。排除参考价、单位价、隐藏内容、推荐及其他商品，保留原有字节、节点、集合和超时上限。二手成色只保留同一区域的来源原文，不猜测或规范化等级。逐页采集是不同时间的观察，不表示实时或完整卖家市场。

单一规格报价事实形状：以现有 `source.price.fact` 对象为唯一规格报价事实，保留 `amount`、`currency`、`minQuantity`，新增可省略的 `condition`（`new` 或 `used`）、仅二手可带的可省略 `conditionNote` 和可省略的 `alternative`。`alternative` 是不再嵌套的同结构报价。仅二手时主体标 `used`；两种状态均有明确金额时主体为 `new`、附加项为 `used`。缺 condition 表示未记录状态，不得附加其他报价或成色；同状态重复、非法条件、空或超长成色、未经验证的币种拒绝。旧事实没有 condition 时显示“状态未记录”，不猜新品、不回填；不能以通用属性绕过严格接收或把两份事实当作双写事实源。

正常来源 owner 为现有 Product sourcing 的 publication/envelope；Catalog/Collection 为原固定版本和引用。只将明确新品且满足现有金额、币种及精度条件的报价投影到 Catalog；二手仅作为来源事实。原商品级报价及规格报价已有不同作用域，不能把子规格报价混入根商品报价；两处展示应保留正确归属。读取旧固定版本不修改或迁移数据，旧状态未知不解释为新品。SKU、上架价格、库存、权限、租户与新表均不新增。

**2026-10-28 用户产品决定**：旧的未保存请求保留来源金额、币种和原键，状态未知时不写入商品价格；已保存版本不变。这明确解决了 v2 无状态价格原来可能投影 Catalog 与新规则的冲突，不推断是否有未完成请求。

交接保持一个当前 Amazon v2 严格合同：Amazon 专用 price 对象的既有字段顺序为 amount、currency、minQuantity；新增字段固定后置 condition、conditionNote、alternative，缺省字段不序列化，新增字段显式 null 拒绝。`minQuantity` 缺省规范为 null，保持现有字节；alternative 只包含 amount、currency、minQuantity、condition、conditionNote，固定顺序，不允许再嵌套。conditionNote 限 256 UTF-8 字节且仅 used，可见安全文字来源。新插件有确定状态才写 condition。旧载荷无新增字段时 TS 冻结和 Go 规范化仍须逐字节得到原始 canonical 和 SHA；现有全字段校验、重复键和未知字段拒绝不降级。当前 `checkBrowserWire` 须正确识别带 `omitempty` 的 JSON tag 中逗号前的字段名，并覆盖其余调用方不放宽；嵌套报价同样递归拒绝重复键。以共同的现行合同表示状态未知，不设 v2/v3 两套接收器、fallback 或适配器。原键标识、授权、digest 和 UNKNOWN 恢复仍不变。接收端新规则对任何无状态新提交保留事实但不投影 Catalog；已有 publication 回放返回既存固定结果，不重算或更新历史快照。验收依赖旧 golden canonical、摘要、原键重试和已保存固定版本测试；不预先宣称通过。

根商品 `priceFacts` 和规格 `price` 当前分别保留并展示锚定商品的报价，二者均须携带正确状态且不混子商品；同族各子 ASIN 的缺失/歧义诊断须完整传递。Amazon 使用独立严格 price wire 类型，既有 `browserPriceWire` 和 1688 接收合同不变。二手报价能否在缺匹配表单时仅凭已验证的可见区域准确绑定，尚需按真实页面证据验证；如果证据不足，保持未观察，不以模拟页面证明真实可用。

## 交接、权限和失败边界

保留现有插件 → 受限固定接收页 → 当前身份/企业授权 → Product UoW 内 SRC、Catalog、Collection 原子保存 → 精确版本读取；不新增身份 owner、外部请求、provider、费用或新的事务/恢复状态机。原键绑定原 payload 和身份；同键异载荷冲突，响应丢失以原键核实，不重采或自动重发。固定版本读取不回退 latest；历史版本不可变。

既有价格对象内的另一状态报价最多一个；每个 alternative 明确计作一项，并在扩展、TS 接收和 Go 接收三处统一计入现有 1024 聚合项预算。采集节点遍历沿原额度，完整规范载荷、SourceEnvelope 和响应均受既有大小及集合限制。171 个变体同时含两种报价须检验现有 1024 聚合项、2 MiB 写入和 3 MiB 详情预算；超限明确拒绝/沿原暂停，不截断冒充完整。先部署兼容增量的服务与 UI、再更新插件是未来切换建议，实际运行切换不在本次授权；旧实例无法接收新字段，不能把源码测试当成当前实例已支持。

## 评审与验证

Design Basis: **Independent Architecture**（来源报价合同和持久事实改变）。独立 Architecture Review 第二轮复核技术依据 SHA-256 `ab917ad55994540c801bfd574eb43d2e486f4be92ae6194ea9905ff54aa723b6`：IMPLEMENTATION_READY，无设计 BLOCKER。实现测试覆盖旧摘要/原键、根及规格状态、1688 严格合同隔离、总量和真实二手 DOM 证据限制；实机验证未授权。Legacy decision: 原有效提取规则 EXTRACT 到当前插件 owner；不包装旧服务，不引入双读/双写/适配器。

准入后 TDD：同页新品/二手不同价、单一二手、新品与二手各自归属、重复表单、错 ASIN/金额、同状态冲突、隐藏/推荐/参考价、未知币种、成色缺失；单商品与同族；插件/TS/Go 严格合同、规范哈希、固定旧事实读取、不可将二手投影 Catalog、原键/不同载荷/UNKNOWN；数据大小边界及既有 1688 行为。仅离线测试，不代表真实页面或用户验收。
