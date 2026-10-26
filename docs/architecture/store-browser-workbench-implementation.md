# 店铺浏览器接口实施交接

执行 [Issue #954](https://github.com/qq550723504/shuomi-ai-commerce/issues/954)，与浏览器 [Issue #14](https://github.com/qq550723504/openbrowser-platform/issues/14) 同一交付批次。Design Basis 为已合并浏览器设计 [a42729e / store-workbench](https://github.com/qq550723504/openbrowser-platform/tree/a42729e3a802cd7e2e972f67d937c654fe89c422/design/store-workbench)，Independent Architecture 已 IMPLEMENTATION_READY。固定 HEAD、CI 和最终独立审阅证据在 PR 维护。

## 当前消费者与入口

浏览器复用既有 Native 身份、企业 context、Store catalog/summary，并消费 Store owner 的四条新路由：

| 方法与路径（前缀 `/api/v1/workbench/stores/:store_id`） | 权威与结果 |
| --- | --- |
| `POST /browser-operations` | CurrentIdentity + LiveWrite + browser.use；domain 再查 store.read、当前 Store grant/active；原 Store→grant 锁顺序、同库事务写操作与审计 |
| `GET /browser-operations/:operation_id` | CurrentIdentity + None；固定 issuer/subject + 全局 store/op 定位原操作；不依赖已清除的企业 header，不产生执行能力 |
| `POST /browser-operations/:operation_id/reports` | 原主体报告原操作；immutable kind/outcome/reason；撤权后仍可提交原结果，不授予新动作 |
| `GET /audit` | 当前 Store 范围；普通成员固定本人，管理员可筛 actor；scope/filter/time/cursor 服务端绑定，limit 1..100 |

唯一首次准入为 `executionAuthorized=true`；重复 POST 和所有 GET/报告均 false。请求键不同载荷 409；操作/报告和 Store 原 AuditRepository 同事务。15 秒请求与 body read 边界，命令体 16 KiB；无自由文本、PID、目录、cookie、token 或商家成功声明。审计来自 Store grant receipts、服务端准入和客户端声明三类事实；响应 `canFilterActor` 只投影原 Store 管理员判定用于界面展示，不构成权限来源。

## 权限与安装

`store-browser` 是现有企业角色目录中的显式模块，包含已有 `workbench.store.read` 和新增 `workbench.store.browser.use`，不包含 store.update，不自动加入旧 `stores` 模块或既有角色。原 32 模块持久化边界保持，不改 IAM。

新增 Store-owned 表为 `workbench_store_browser_operations` 和 `workbench_store_browser_reports`。现有 Store 缺这两表时原 Store 可继续工作；新接口返回不可用。部分表、同名 view/RLS 或不符合合同的结构关闭，运行时不执行 DDL。

既有 schema-owner CLI 提供显式模式：`--browser-workbench --confirm-database <精确库名> --config <私有 owner 配置>`。仅接受当前 Store 原版本历史，复用原 Goose version table 新增 2026102501；同一新版本事务添加两表及新表 SELECT/INSERT，并读回验证结构与 serving role 实际不可变权限。不得与其他安装/观测/对齐模式组合；不执行旧 schema 步骤，不改旧事实/授权，不放宽原 runtime privilege guard。此命令用于后续获授权维护，本批没有运行它。

保留实例的前序 core 对齐仍由原 `--align-core-contract --confirm-database <精确库名> --config <私有 owner 配置>` 执行，只接受原 Goose 历史并事务修改既定四项 CHECK、追加 2026101601。若实例已安装 observations / operations cockpit，分别显式附加 `--retained-observations` / `--retained-operations-cockpit`；这两个声明仅可用于 core 对齐，不能与安装开关或 browser 模式组合，也不从现场自动推断能力。省略声明时保持原裸 core 边界。

该维护入口复用 `RuntimeCapabilities` 和原 runtime 有效权限白名单，严格核对固定 public core、声明的可选 schema/owner、原生列/默认值/约束/索引及完整必需权限；未知 schema/对象、结构漂移、PUBLIC/列级额外权限、grant option 或非法历史均拒绝，不自动修复。两个可选结构 JSON 合同来自已有 persistence owner 的 `schema.sql`，隔离 PostgreSQL fixture 用原生 Install 创建后验证；不含 OID、业务数据或 ACL，不是第二套安装器。它们随原生结构有意变更时须一起审阅；本轮固定 PostgreSQL 17 原生表达式，不据离线 PASS 宣称现存库已兼容。

原生顺序保持 **1601 → 2501**。browser 安装只要求已对齐的必需 core，不隐式要求未启用的 TikTok/Walmart 表；平台 serving guard 和新表实际 SELECT/INSERT 授权保持原语义。core 对齐不安装可选模块、不改现有 grants/事实/历史，也不扩为平台维护或 browser 权限回退工具。本轮仅以合成数据测试上述入口，未对保留环境执行维护命令。

## 验证与剩余条件

离线回归覆盖首次/重复/冲突准入、read+use+Store grant 无 update、撤销后原主体报告、原子审计、本人查询和游标隔离、严格 HTTP 输入、可选表损坏关闭、目标 runtime ACL 读回及默认关闭模块。沿用当前组合与 route descriptor 白名单，不移除 guard。

真实 token/API audience 兼容性、公开发行实例、保留库安装、实际浏览器/内核/商家操作及发布部署都不是本批验证结果。浏览器配套 PR 为集成依赖；应用内的客户端报告不证明商家操作成功。实现 PR 仅进入独立审阅/后续试用阶段，不自动合并或关闭执行 Issue。
