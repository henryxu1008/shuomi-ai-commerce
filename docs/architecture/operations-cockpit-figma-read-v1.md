# 运营驾驶舱四页 Figma 与只读投影补充

Execution: [#619](https://github.com/qq550723504/task-processor/issues/619)，原 owner [#627](https://github.com/qq550723504/task-processor/issues/627) / merged #629。
Design Basis: Independent Architecture（仅新只读查询和响应合同）。
Admission: IMPLEMENTATION_READY / FROZEN；2026-10-11 独立只读复核完成，正式生产代码在此准入后开工。

## 产品结果与依据

2026-10-11 用户确认：沿用已确认首版范围，按 Figma 修整四个页面；缺失的真实统计和筛选先补设计。
当前企业成员在目标管理、店铺矩阵、经营预警、经营建议中查看实际经营数据与依据，筛选授权店铺，进入现有目标设置、人工录入和处理路径。
复用冻结的 [operations-cockpit-v1](./operations-cockpit-v1.md)，其中已确认人工收入成本、规则建议、创建人或当前 manage 维护目标。

Figma Product Authority: file `tg48P46SSXl6TBy9lZwg63`，page `31:463`：

| 页面 | 暗色根 / 内容 | 亮色根 |
| --- | --- | --- |
| 目标管理 | 1413:359 / 1413:451 | 1413:628 |
| 店铺矩阵 | 427:851 / 427:931 | 427:1051 |
| 经营预警 | 427:1379 / 427:1459 | 427:1579 |
| 经营建议 | 427:1907 / 1298:359 | 427:2107 |

已读取上述暗色实际 context 与截图；亮色继续使用当前主题 token，并在读取对应 context 后做必要的局部调整。共享 Console 导航、品牌、根入口四个子菜单沿用当前实现。
Figma 示例数字、任务工作区、自动执行、自动分析、AI 预测收益、处理/恢复状态及店铺主力/成长标签不是本轮已确认产品事实，不能伪造。
本轮不增加模型调用、平台操作、新财务源、写命令、角色、schema、事实 owner、状态机、迁移或恢复协议。

## 当前 owner 与调用路径

StoreCenter 是当前授权店铺身份/平台/站点 owner；operationscockpit 是人工区间事实、目标/版本与计算 owner。已保存订单观察继续来自原 observations owner，与人工财务统计分开展示。
现有 Handler → member-scoped Directory → 原 Repository.Snapshot（相同事务快照）→ 原 Rules / Matrix 计算 → 有界筛选和汇总 → 原 live 权限/Store scope 重验 → 原 no-store HTTP → BFF strict Zod → enterprise/actor query key → CockpitPage。
不新增 SQL、跨库读取、缓存、外部网络请求或写入。Snapshot、goal scope/metadata、创建人与 manage、幂等回执及 UNKNOWN 原操作恢复保持冻结合同。

## 新只读合同

1. `GET stores` 增加可选单值 `platform`（最多64 Unicode字符）与 `region`（最多100字符），精确匹配当前 Directory 字段。选项来自当前 capabilities 授权目录，不能从当前分页猜测。
2. 原 `search` 明确为最多100 Unicode字符、不区分大小写的包含匹配。stores 搜索当前店铺名称；alerts/advice 搜索规则标题或当前授权店铺名称。原 level/kind、financial state 与 sort 的枚举非法值返回400，重复单值参数由原 query helper 拒绝。
3. 附带的显式 storeId 仍先验证全部当前授权，之后才应用平台/站点过滤；未知/越权 ID 返回403，不能借空筛选隐藏越权。平台/站点只用于 stores；目标/规则仍消费既有相同目标 scope。过滤至空目录传入空 StoreIDs，Repository 不得扩展成全量（当前 Snapshot 空集合表示空）。
4. stores `states` 和 `summary` 仍涵盖选定店铺/平台/站点全部授权店铺，在姓名/state 筛选及分页之前计算。四卡是当前范围店铺数、录入完整、亏损、数据不完整。说明明确统计范围；未录入不能当健康/零利润；完整利润汇总沿用原 checked 整数分计算。
5. `GET alerts` 与 `GET advice` 响应统一新增必填 `summary: { total, critical, attention, data }`；值为同一 Snapshot 经原 Rules、level/kind/search 筛选后的全量结果，在50条分页前计数。计数是0..501整数，三类之和等于 total，total 与外层 total 相同。两页共享一次投影，不另请求统计、不拼分页。
6. summary 不含处理/恢复/预测状态。goalUnavailable、observations 继续单独标识，当前结果不能代表不可读的目标；零规则不代表所有数据完整。错误/不可用/刷新失败不展示旧指标为当前成功结果。
7. 输入不扩大原16KiB query、500店铺、366天、10s deadline、1MiB response边界。平台/站点参数只接受 stores，其他路径400；所有原 read/module/tenant/Store授权检查保持。

## 页面消费

目标：启用配置摘要四格、目标摘要/评估规则/人工处理/版本历史双列卡。目标净利润与进度来自实际当前 goal/evaluation；空目标和范围不可读状态明确。目标设置与历史恢复沿用原命令，不新增推荐值或自动策略。
矩阵：紧凑周期与筛选面板、四卡、真实矩阵表、2–5店铺比较选择条。周期使用 capabilities.today（UTC+8）与原完成日语义；近7/30完整日、本月至昨日、自定义；本月尚无完成日不能提交空区间。平台/站点、状态/名称变化重置页码与比较选择，保持 sort=profit 与缺数据可见。
预警：同一周期/授权店铺/等级/类型/标题搜索面板，四项全结果统计，原规则排序的预警列表，依据与人工处理链接。可以进入建议页携带原 period/store/kind/level 和 ruleId；ruleId 仅客户端选择当前已获授权响应中存在的规则，不成为后端内容读取或授权依据。
建议：紫色强调、建议队列和选中依据/处理详情双栏；真实 severity、记录依据与版本、原 actionPath 保留。无预测利润/付费执行/新任务入口。规则不可用/空队列明确，而非虚构建议。
现有 Card/Input/Button/主题 tokens 与完全相同的 Console 静态资产复用；动态财务和店铺数据不用 Figma 样例填充。窄屏单列、表格局部滚动，页面不横向溢出。

## 验证与交付

先 RED → GREEN：alerts 与 advice 的超过50条全结果统计、severity/kind/search筛选、当前授权 store 名、越权显式ID不能由筛选吞掉、平台/站点与 summary范围、非法枚举与 Unicode bounds。原目标/事实幂等、scope与恢复测试继续执行；不改其生产路径。
前端复用 Vitest验证必填/一致 summary、不是分页统计、筛选参数/清页与选择、原UNKNOWN恢复及组织切换；typecheck/目标lint/相关Go测试与必需CI。
独立检查仅新只读合同准入与最终真实diff/运行交付两个适用点，不重做冻结原架构。
现有保留本地程序只更新backend/UI镜像，以当前main程序为基线；其余服务/六层Compose、原命名卷、授权、schema与TLS保持。不改变用户在另一交付中确定的main试用范围，不附加未合入main的其他PR。提供四页真实读取截图和入口；不录入经营数据或调用 provider。开发自检与独立交付检查不等于用户产品验收。
Legacy decision: N/A；无 legacy调用/兼容/迁移。真实非空经营数据、平台/provider/paid操作、生产部署和用户验收 NOT_RUN。

## 独立复核

2026-10-11 既有独立 Reviewer `/root/native_auth_review` 对基线 c023edad14ad9255dee3e04a67b539bdaf2c83b8 与本设计 SHA256 66749D7AA9E2F5BADA2222FD91E2805D11906312144C1476CA5B0BEF8AF623E4 只读复核，结论 IMPLEMENTATION_READY，无 BLOCKER。
R1 IMPLEMENTATION_TEST：录入完整合计盈利/盈亏平衡/亏损，矩阵摘要范围与分页前规则统计准确；R2 IMPLEMENTATION_TEST：显式越权ID先拒绝、空筛选不扩范围、参数/Unicode bounds、响应前撤权与跨企业/刷新失败清旧统计。本增量合并前收敛，不重开原架构。
运行范围注记根据当前 MAIN-TRIAL 已确认的 main 试用决定更新，仅限定本批软件组合，不改变只读合同与准入。
