# 自营与优选商品筛选

Execution: #1292 / PR #1295。Design Basis: Independent Architecture（只读公共查询合同增量）。
Admission: IMPLEMENTATION_READY / FROZEN；独立设计复核通过（本聊天final_ui_review，2026-11-15），无BLOCKER。实现及用户验收另行记录。

## 用户结果与范围

2026-11-15 用户要求自营与优选加入筛选，并随后明确增加城市。卖家在原商品列表组合名称、发货省市、
价格区间及币种、最高起订量、最长供货周期，得到正确的匹配总数和分页商品。
界面复用现有筛选栏、Input/Select/Button，标签使用日常业务用语；Figma不作为UI标准。
不增加分类/品牌事实、库存同步、报价/交易、排序、批量选入或通用筛选平台。

## Owner、依据与调用路径

复用 [supply-market.md](supply-market.md) 已批准发布/只读/精确选入合同、
[project-boundaries.md](../../project-boundaries.md)、当前用户页面自然文案决定和全新系统基线。
SupplyMarket仍是明确发布及查询owner；筛选只消费原release_json的product和supply。
路径：MarketList → listMarket → marketEndpoint/BFF现有proxy → HTTP ListMarket →
Service原授权与读取后复核 → Repository.ListReleases同一谓词生成count和分页 → 原DTO。
不改变装配注入、返回DTO、鉴权、租户或select_release合同。

## 查询合同

原after/limit/keyword/kind保留。Query新增MarketFilters值对象，参数仅市场列表接受：

| 参数 | 结构校验与含义 |
| --- | --- |
| province | 最多100 UTF-8字节，无NUL；精确匹配原供货省份，不解析自由文本 |
| city | 最多100 UTF-8字节，无NUL；精确匹配原供货城市；非空时province必须非空，省市AND组合 |
| minPrice / maxPrice | 可选有限非负数；同时存在时min≤max；边界包含 |
| priceCurrency | 最多16 UTF-8字节、非空；有价格边界时必须提供，无价格边界时不得单独提供 |
| maxMOQ | 可选整数1..1e9，供货minimumQuantity≤该值 |
| maxLeadDays | 可选整数0..3650，供货leadDays≤该值，0仍是有效条件 |

新数值参数使用最多32字符的非负十进制格式（不使用指数或正负号）。重复/未知参数、无效数字或过长输入返回INVALID_REQUEST；HTTP和BFF校验一致。
其他records/events/record-releases端点拒绝这些市场专用条件；不得静默忽略。
UI默认人民币并可选其他常用币种；只有设置价格边界才发送币种。空条件不发，0不当空。
此默认仅为表单选择，不推断商品币种，不转换汇率，不制造缺失价格。
城市增量复用本合同的原只读筛选路径和不变量（Reuse Existing Architecture）。
界面复用供货资料原 SupplyLocationFields，通过[共享中国省市目录](../../platform/geography/supply-region-catalog.md)取得原省市选项。未选省份时城市禁用，
选省份后可选“全部城市”或对应城市；切换省份清空城市，清除筛选同时清空两项。
服务端只按原发布省市值精确组合，不用地名关键词推断或补造业务事实。

价格匹配要求至少一个已发布款式同时满足原currency和全部价格边界。
不采用商品min/max区间重叠替代真实款式匹配：款式10元和100元不能匹配40..60元。
不同币种不混算；无数值价/无币种不能满足价格条件；0价可匹配；不从priceNote提取数字。
实施发现PublicVariant原float64/omitempty会丢弃0价；按现有可选JSON价格合同改用可选数值指针，
在原PublicProjection保留明确0价、缺价继续省略。新发布按原owner序列化，不重写现有发布或迁移数据。
SQL使用参数化谓词，JSON数值转换用类型检查的CASE，不执行任何用户SQL片段。

## 分页、状态与持久化

所有筛选AND组合，在count和id游标/limit之前执行；原active/channel条件仍生效。
总数是全部匹配商品数，不是当前页过滤数；nextCursor来自匹配结果的limit+1行。
原列表轻量投影继续使用，不为筛选放大返回payload或批量拉完数据。
表单草稿和已应用条件分开；提交/清除时重置after；请求key包含全部已应用条件及scope。
保留原分页和重新开始入口，不把上一组筛选的游标用于新条件。
没有新表/列/索引、持久化业务事实、状态机、事务、恢复协议或第二来源。
只读失败保留表单，通过原读取错误及重试处理；无写入幂等/retry/UNKNOWN变化。

## 权限、资源与副作用

沿用原读取权限、组织上下文、读取后授权复核和无缓存规则。
只返回原active明确发布；不暴露申请、来源/私密字段，不改变租户边界。
沿用limit≤50、现有超时及2MiB响应限制；新增参数有字节/数值范围限制。
没有AI/provider调用、计费、对象写入、平台发布或外部副作用。
原35444若更新，仅替换必要UI与后端镜像，先核对原后端与新候选源码差异及配置，
保留配置/身份/权限/数据库/卷；标准builder产出可追溯镜像，不执行初始化。
Legacy decision: N/A，不引入兼容、迁移、fallback或双读双写。

## 验证与交付

先RED再GREEN：Query/HTTP/BFF参数拒绝；Postgres真实筛选、组合、总数、分页、
多币种、缺价、0价和实际款式匹配；组件应用/清除重置游标及0天参数。
复用现有Postgres testcontainers约定及组件测试，不增加验收平台；只跑相关检查。
稳定候选标准构建，实际浏览器各渠道使用不同条件确认结果，清除恢复；不重复选入。
独立最终复核实际diff和路径；CI与开发自检不代替用户实际验收。最终结果交回原35444。
