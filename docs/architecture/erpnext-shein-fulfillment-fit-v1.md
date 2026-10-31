# ERPNext 与 SHEIN 双履约适配验证

执行 Issue：[#1037](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1037)。日期：2026-10-31。验证性质：明确标记的非正式、隔离 Spike，不是生产实现或用户验收。

## 结论

ERPNext 适合复用为本批采购、收货、库存预留和实物出库的单一 owner。原生业务单据和事务可以满足有界首版；普通 REST POST 没有原操作幂等语义，必须增加 ERP owner 内的最小原子命令，不能由 Go 侧先查后建或重复发 POST 补救。

本轮 14 项实际检查通过，另有 1 项明确适配缺口（普通 POST 重复创建）。该缺口的唯一键、同键载荷冲突、原单读回、并发原子创建及事务回滚路径已在非生产 Spike 验证，但尚未成为正式插件。原生通用采购/库存角色对本批过宽，最终 HTTP 验证改用 Frappe 原生 Custom Role/DocPerm，未新增 IAM。

## 固定基线

| 对象 | 准确版本 |
| --- | --- |
| ERPNext | v15.124.3，源码 9f33fd5d32ae8a75156b2e7c2f8be02503fcc248 |
| Frappe | v15.125.0，源码 e7458ff33c8c1b81f812d521a17df790804eed6d |
| 官方应用镜像 | frappe/erpnext:v15.124.3，digest sha256:02c7443608b6af7d088a512aba3a62ecafdba2346be853ddce577194f856af49 |
| 数据库 | 独立 MariaDB 10.6 系列；非现有业务 PostgreSQL |
| 官方 Compose 参考 | frappe/frappe_docker @ 61d35da827ebc04d01b64c9e78385597ccebc9d8 / pwd.yml |
| 运行范围 | orderfit-20261031，两个独立 ERP site/数据库，全部为合成数据 |

固定镜像提供 ERPNext/Frappe 当前已安装版本；未将参考 Compose 的 v16 镜像配置当作 v15 的执行证据。实验仅适配镜像、MariaDB 版本、独立项目、回环端口与本任务随机凭据。凭据及 site 文件保留在任务私有目录/卷，不进入仓库。

## 实际结果

| 检查 | 结果与可核对事实 |
| --- | --- |
| 采购单不生成库存 | PASS，采购 5 件后实际库存仍未增加 |
| 分批收货 | PASS，采购 5、实收 3，原采购行保留 2 件未交 |
| 不自动生成总账 | PASS，关闭公司持续库存会计后，收货的 GL Entry 为 0 |
| 原生库存预留 | PASS，5 件需求面对 3 件合格库存仅预留 3 |
| 分批交付 | PASS，交付 2 后原需求已交 2、仍需 3；对应实物库存余 1 |
| 并发预留 | PASS，两个连接同时为合计 4 件需求争用 3 件，最终分配为 2 和 1，总预留 3 |
| 保留所有权的仓间调拨 | PASS，5 件中 2 件移至平台/在途位置，源仓 3、目标 2，总数量仍 5，GL Entry 为 0 |
| 普通 REST POST 重复发送 | GAP，两次均成功并产生不同采购单 ID；标准 CRUD 不能用于未知结果自动重试 |
| 唯一外部操作键与原摘要读回 | PASS，重复创建被原生唯一约束拒绝，按原键读回原单 ID/摘要 |
| 同键不同摘要 | PASS，原生唯一约束拒绝替换；原摘要保持 |
| 跨 site 凭据 | PASS，site A 的有效 API key 访问 site B 返回 401，无订单正文 |
| 最小采购角色 | PASS，同一有效非管理员 API 身份可以创建采购单，但读取 Bank Account 返回 403 |
| ERP 事务内并发创建 | PASS，一个创建、一个读回同一单，数据库实际只有一张采购单 |
| 提交成功但响应未采用 | PASS，原键读取返回原单，异载荷返回冲突，无第二张采购单 |
| 提交前故障 | PASS，业务单据创建后、原事务提交前故障，采购单与操作回执均回滚为 0 |

最后成功记录的原生运行标识：1f9a08fd、94915aba、atomic-38bfb673。数据全部为临时合成业务，不是实际 SHEIN、供应商或买家数据。完整非敏感结果和脚本摘要见 [固定证据](evidence/erpnext-fulfillment-fit-v1.json)。

原生唯一字段冲突在此准确版本返回 HTTP 417 / UniqueValidationError，不能假定一定返回 409，更不能把任意 417 都解释为重复或无效果。正式插件应返回稳定的窄合同和安全错误，不向 Console 透传 Frappe 堆栈。

## 最小正式适配范围

1. 复用 ERPNext 原单据、行身份、库存账、SRE 与数量校验；不再建硕米可写库存账。
2. 一个有界 Frappe app 提供固定用途命令和原键读取；外部操作唯一键、绑定摘要、原单引用与原业务写入在同一 ERP 数据库事务内提交。
3. 复用 Frappe Custom Role/DocPerm 和每企业独立 site；API 凭据只给服务端。多 Company 不替代企业隔离。
4. 独立保存平台副作用意图和结果；调用原 Store 官方句柄，不把原生 ERP 出库提交当作 SHEIN 已收货或原平台操作成功。
5. 同币种与库存会计关闭路径已实测。非基础币种只消费已明确配置的 ERP 汇率与原币金额，缺失时阻止该次折算，不自动外呼汇率服务或补成 1。

## 证据限制

初始四次准备运行暴露空安装的预置资料、叶级客户分类、销售价目表等初始化要求，均保留原失败记录；它们不是目标业务缺陷的 RED。完成标准预置和合法主数据后才执行上述结果。并发检查明确调用原生预留命令，不把仅提交 Sales Order 当作已经预留。

操作回执 Spike 是能力验证，不作为正式消费者或可直接部署插件。正式实现仍需在冻结合同下按 TDD 完成准确绑定、撤权、原请求恢复和相关命令测试。

未执行真实 SHEIN 请求、供应商下单/付款、真实承运商、付费模型、现有实例接线、部署或用户验收；以上保持 NOT_RUN。ERPNext 初始化、权限和原子命令属于本实验新资源，不涉及保留实例 schema。

## 保留环境隔离

实验前记录原 39 个容器，检查时容器 ID 与镜像均未改变。实验卷和网络与原容器没有交集：0 个共享卷、0 个共享网络。Redis 官方镜像额外创建一个全新匿名缓存卷，它仅属于本次实验，不以“所有卷都有项目前缀”替代实际交集检查。

本轮没有向原实例执行写命令，没有重启、升级、初始化、复制业务数据或删除原卷。容器元数据对照不代表原应用在正常运行期间业务数据完全静止。

## 原始执行位置与来源

任务私有目录：独立工作区 .local/erpnext-fit。包含 probe_native.py、probe_extended.py、probe_atomic.py、每次运行记录、镜像/Compose 和资源交集证据；所有文件明确 NON_PRODUCTION_SPIKE，生产代码不得导入或调用。凭据文件和原始环境不进入提交。

上游来源：[ERPNext v15.124.3](https://github.com/frappe/erpnext/tree/9f33fd5d32ae8a75156b2e7c2f8be02503fcc248)、[Frappe v15.125.0](https://github.com/frappe/frappe/tree/e7458ff33c8c1b81f812d521a17df790804eed6d)、[官方 Docker](https://github.com/frappe/frappe_docker/tree/61d35da827ebc04d01b64c9e78385597ccebc9d8)、[采购与收货](https://docs.frappe.io/erpnext/purchase-receipt)、[库存预留](https://docs.frappe.io/erpnext/stock-reservation)、[Frappe API](https://docs.frappe.io/framework/user/en/api/rest)。
