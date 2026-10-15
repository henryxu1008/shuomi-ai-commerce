# 优选申请资格读取：正常不可申请结果

Execution: #710；补充冻结的 [供应市场 v1](supply-market-v1.md)。
Design Basis: Independent Architecture（有界资格读取响应合同修正）。
Status: IMPLEMENTATION_READY；2026-10-15 独立只读评审通过，原自营消费者阻塞已修订关闭。

## 用户结果与依据

2026-10-15 用户要求修复资格查询导致整页报错：不合格商品正常显示为不可申请，
仅已优化的自有商品可进入申请，保留已有数据，检查实际页面但不提交申请。
沿用此前明确的单件申请；不新增批量申请、来源、过滤器、AI/优化调用或资质上传。
Figma Authority: tg48P46SSXl6TBy9lZwg63，429:2712（概览）、997:764（选择）、
1004:892（供货）；保留当前结构，以不可选行呈现真实未达条件商品。

## 根因与合同

当前 Choice 在判断是否优化前调用 PublicProjection。合法保存的无图片自有商品
不满足发布投影，返回 INVALID_REQUEST，导致选择列表和概览整体失败。
资格读取不能要求每个已保存商品都已可发布；发布投影约束本身保持不变。

ProductChoice 仍由当前 SupplyMarket / Collection / Catalog / Review owner 提供。
继续返回经过原 sealed selection 与 ReadEffective 校验的 selection，以及真实 optimized。
Product 改为可省略的指针；新增可省略的 ineligibleReason，只有两种不可申请原因：

| 正常读取结果 | optimized | ineligibleReason | product |
| --- | --- | --- | --- |
| 实际 Apply 已核验且公开投影完整 | true | 省略 | 原完整 allowlist |
| 尚无实际 Apply | false | not_optimized | 投影完整时保留，否则省略 |
| Apply 已核验但公开投影不满足要求 | true | incomplete_product | 省略 |

不是新的持久化状态机；这些结果每次读取计算，不缓存为领域事实。
只将 PublicProjection 的 ErrInvalid 作为 incomplete_product；不捕获授权失败、
快照不可用、版本冲突、伪造/失配 lineage、超时或取消。这些真实错误继续失败并允许重试。
不向资格响应补充原始快照、私密图片、内部 trace 或伪造缩略图。
同一 Choice 还被自营草稿消费：create_official_draft 原本不要求优化。
因此 not_optimized 且投影完整时仍返回 product，自营草稿以 product 存在判断可继续，
优选页仍严格要求 optimized=true 且无不可申请原因。投影不完整时，自营页正常提示并
禁用保存；两页在读取中、读取失败或 selection.itemId 失配时都不可消费旧 Choice。

contract → implementation → injection → consumer：ProductChoice（三种读取结果）
→ 现有 EffectiveProductReader（校验后映射）→ 原模块注入不变
→ 同源 BFF 的严格 union schema、选择页和 ApplicationOverview。
选择页显示所有本页未归档 own 项；不合格行使用现有列表标题及分组、无图占位、禁用单选，
明确显示“未完成智能优化”或“商品资料不符合申请要求”。可申请数量只计真实可申请行。
未知资格/真实读取错误不伪装为不合格或零数，整页继续 fail closed。
概览仅统计 optimized=true、product 非空且无 ineligibleReason 的项，忽略正常不合格项。

## 不变量与边界

- 原 Organization + Actor + Member、权限复核、sealed selection、原始发布及版本完全保留。
- optimized 仍仅表示原真实 Apply lineage；不以图片、手工保存或 UI 状态代替优化证据。
- submit_selected 的 selectOwn(requireApply)、ReadEffective、PublicProjection、资质和披露
  校验完全保留；资格读取不能授予写权限或构造新的 Apply 回执。
- 无表、DDL、持久化事实、owner、事务、幂等键、UNKNOWN/retry/recovery 协议改变。
- 无新外部副作用；无 Legacy 触碰，Legacy decision: N/A。
- 当前仅内部保留实例；部署采用同候选 backend/UI，不运行初始化或更改卷/数据。
  本次用户要求修复后实际检查保留实例；合并新 PR 与关闭 Issue 没有授权。

## 验证与交接

TDD：无图片未优化项正常读取；真实 Apply 但投影不合格正常不可申请；原合法 Apply 可申请；
伪造/失配版本仍错误。概览混合项只计合法项，真实依赖错误仍整体失败。
前端：混合列表保留不可申请行并禁用；只有合法 own 行能进供货步骤；未优化但投影完整
商品仍可按原规则进入自营草稿，无图项不能保存自营草稿。资格刷新失败/失效
清除可操作旧选择，企业切换和原单件命令不变；BFF union 拒绝伪造 eligible 形状。
运行：用现有无图项复现修复后的 HTTP200/不可申请和正常概览；保留数据及原作用域，
不上传、不保存商品、不提交申请。如没有真实合格存量商品，正向流程仍为 NOT_RUN；
可用现有隔离 fixture 检查供货页面但必须单列，不能声称真实资格验证通过。
