# 旧刊登 CLI 与调度链退休边界设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29)、[#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **DESIGN_REVIEW / 待用户确认；正式实施未授权**。
- 本轮 Design Basis: **N/A（纯文档、只读源码/依赖及持久化映射）**；不改变任何生产业务路径、权限、状态机、事实 owner、schema、配置、CI 或外部副作用。
- 调查基线：`main @ 8a3f97850d04d2109bda4b432cbc0361289c04b9`。
- 用户决定：先设计 `listing-scheduler`、`shein-listing`、`temu-listing` 与调度链的退休，确认后再实施；保留当前采集、Store 观察同步、有效共享基础层和独立 SHEIN 登录 worker，不操作运行实例或数据，不增加内部试用前置。

## 1. 用户结果、Authority 与非目标

维护者能够清楚区分当前应用的有效入口与旧数字账号刊登/定时任务运行链，确认后才删除能重新启动旧链的源码及专属构建接线。当前用户继续沿已有内部试用应用完成采集、保存和草稿操作，Store 观察同步及独立登录 worker 按各自原合同保留。本设计不交付新刊登能力，也不证明任何旧实例正在运行或已安全停用。

适用依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md) 与 [module mapping](../refactoring/module-target-mapping.md)：EXTRACT / RETIRE；不恢复 Task-first、旧新 ID 映射、迁移、fallback、双读双写或第二事实源。
- [项目领域边界](project-boundaries.md)、#29 已合并的 root Task/API/Workspace 退休：旧 standalone numeric leaves 仍是债务，不能因仍可编译而成为当前 Product/Listing owner。
- [当前 Store](store-center-current-application-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[观察内部准备接线](store-observations-preparation-runtime.md) 与 [当前 SHEIN 供货链](my-supply-chain-shein-v1.md)：当前 Organization/Store、官方连接与 Submission 保持各自原 owner；缺配置仍按原范围不可用。
- [独立 SHEIN 登录 worker 边界](shein-worker-identity-persistence-boundary.md)：原账号仓库、Cookie/attempt/验证码/Profile/健康与关闭保留，查重按已有 tenant 隔离修复；保留该 worker 不授权其它旧刊登/调度入口。
- [发布预检退休](release-identity-preflight-retirement.md)：原安全验证与独立 worker 合同保留，旧发布入口不能被改造成 current-application 的交付通道，生产仍 NOT_READY。

本轮无 UI/IA 改动，Figma N/A。不新建 IAM、Provider、队列、状态平台、迁移/备份工具、验收 runner 或新恢复协议。不执行业务命令入口、Docker、Kubernetes、DB/Redis/RabbitMQ 查询、schema 初始化、浏览器或平台调用；不读取真实配置、账号、凭据、Profile 或运行数据。

## 2. Design Basis 与实施准入

当前只完成文档设计与直接引用校准，N/A 不表示后续生产修改可跳过准入。未来候选若严格限定为本设计 §6 的已登记旧路径 RETIRE、零消费者专属代码/测试删除及必要引用校准，可在用户确认后按 **Reuse Existing Architecture** 写入执行 Issue：复用上述批准的 Hard-Cut、当前身份/Store/Submission 与 worker 合同，明确准确 Scope、owner、不变量、CI 引用处置和 Ready，再开始实现。

该路径不新增产品流程、事实/数据库 owner、权限模型、状态/事务/恢复边界或外部副作用。预期会跨多个旧子系统、文件数可能超过提示线；适用依据是既有整体 Hard-Cut 和明确保留边界，而非按文件数机械拆 PR。保持一个主要 Delivery Batch；实现/提交可分为可验证的小步骤。

若实际需要改变当前消费者的业务语义、事实 owner、授权/组织边界、状态机、跨数据库一致性、补偿/retry/UNKNOWN 或平台发送边界，必须停止对应正式实现，改按 Independent Architecture、适用独立检查及 IMPLEMENTATION_READY。不得借退休实现新的当前执行器，也不得把旧 DTO/Service 改名或移目录继续供当前业务调用。

本次纯文档没有新增高风险实施边界，独立 Architecture Review N/A；最终设计差异仍走原 PR 独立检查。**用户尚未确认本设计，正式实施不 Ready；文档/CI/review 不是用户确认。**

## 3. 实际 contract → implementation → injection → consumer

已实际执行全部 `cmd/...` 的 Go 依赖查询，并对以下根做明确比对；这是编译图证据，不是运行或平台验收：

| 命令 | 相关实际依赖与边界 |
| --- | --- |
| `listing-scheduler` | `app/runtime/listingscheduler` → `app/bootstrap/resources`、`app/runner`、`app/bootstrap/schedulers` → `app/scheduler`、`listingruntime/local`、`listingkit/sheinsync` / `listingkit/store`；包含 consumer 合同 |
| `shein-listing` / `temu-listing` | `app/runtime/listing` → `app/bootstrap/listingruntime` → platform module registry / `app/consumer` → 同一 shared resources/local runtime；各平台 processor/pipeline 与 scheduler factory 注入 RabbitMQ/Store/原 API |
| `current-application` | 有 `listingadmin` 与根 `listingruntime` 的有效共享合同；无上述旧 runtime/bootstrap/consumer/runner/scheduler/local/shein-sync 包，不因目录名而整包退休共享层 |
| `1688-local-agent` / `1688-public-browser-collector` | 无上述旧链；当前采集不经旧刊登调度器 |
| `image-agent-temporal-worker` | 有当前 Store owner，无上述旧链；原 Temporal/图片能力保留 |
| `shein-login-worker` | 原 `listingadmin` / `sheinlogin` / managed 两项工厂/bootstrap；无旧刊登 runtime、local、shein-sync 或当前 HTTP application |
| `listing-control-plane` | 仍依赖 `listingadmin`、根 `listingruntime`、`app/taskstatus`，但无旧 consumer/runner/scheduler/local/shein-sync；它是尚未纳入这三个 CLI 删除授权的独立生产者/恢复入口，见 §5 |

基线全部命令中，`listingruntime/local`、`app/consumer`、`app/runner`、`app/scheduler`、`app/bootstrap/resources` 的命令消费者恰为本次三个 CLI。`app/taskstatus` 还被 control-plane 消费；`listingadmin` 还被 current application、登录 worker、control-plane、`shein-import-platform-recovery` 和 `issue36-local-trial-init` 等引用。Go 根隔离不等于所有包外源码引用已排空，删除前仍须刷新全仓直接消费者。

旧链的具体装配行为：

1. `listing.Run` 读旧配置后，正常分支要求 RabbitMQ，构建 ServiceManager/platform registry/resources，注入 processor、运行健康验证、平台 module 配置，最后启动消费。`TASK_PROCESSOR_DEBUG_TASK_ID` 路径另开原 DB、初始化原 ImportTask repository 并直接执行一条旧任务；这不是安全的只读调试模式。
2. `shein-listing recover-paused-tasks` 另行连接 DB/Redis，Plan 只读不等于无连接；`--execute` 会恢复原暂停任务。该子命令与 flags/专属测试须随宿主退休，不能遗留为旧链续跑后门。
3. `listingscheduler.Run` 通过 `BuildSharedResources` 注入 Store/runtime、RabbitMQ 与 SHEIN/TEMU task factory，启动定时任务和动态配置刷新。平台工厂创建改价、商品/库存同步、活动报名等旧任务，仍可能发出平台 mutation，不能当作只读 Store observation。
4. `BuildSharedResources` 建旧 DB/Redis handle、local repositories、Cookie provider、可选 Amazon crawler、RabbitMQ client；SHEIN module 还配置 prompts、Store guard、watchdog、店铺队列分配和 auto-shard。保留的通用 crawler/HTTP/browser/queue/锁不因此成为整包删除目标。
5. 旧 `sheinsync` repository 的外部源码引用主要在 `listingruntime/local` 和 shared resources 的 synced-inventory feed。当前 Store 观察链用当前 Store/官方授权/Temporal 合同；两者不是同一个同步 owner，不用旧表作为当前观察 fallback。

## 4. 持久化、状态与外部副作用 owner

| 边界 | 基线实际 owner / 身份 / 行为 | 退休设计约束 |
| --- | --- | --- |
| 旧账号与刊登配置 DB | `listingadmin.StoreRepository` / `listing_store`；数字 TenantID + 原行 ID；filter/profit/pricing/operation/scheduled-task 等仓库提供旧执行配置 | current Organization、当前 StoreID、外部 merchant/store 字符串各不等价；不解析或映射。原 Store 仓库仍被登录 worker 和其它工具消费，不删表/DTO/仓库整包 |
| 旧任务/派发 DB | `GormImportTaskRepository` / `listing_product_import_task`、`listing_dispatch_event`；claim token、queued/processing/paused/terminal、rollback 与 recovery 仍由原 repository/control-plane/taskstatus 消费 | 三 CLI 的消费/触发能力退出，不意味着这些持久化事实或恢复 owner 已退出。原任务不转换成当前 Listing/Submission/BusinessTask，不恢复、不补发、不迁移 |
| 旧商品、导入映射、库存和 raw evidence | `listingadmin` 的 ProductData、ProductImportMapping、InventoryRecord、RawJSONData 等原仓库/adapter | 数据库相同不构成当前 Product/Catalog/Asset 或新 Source owner。按具体消费者判断源码去留，不删除真实记录或把这些事实导入当前系统 |
| 旧 SHEIN 同步/活动 DB | `listingkit/sheinsync` 合同与 `listingkit/store` GORM adapter；`listingkit_shein_synced_products`、`listingkit_shein_sds_cost_groups`、`listingkit_shein_sync_jobs`、`listingkit_shein_activity_candidates`、`listingkit_shein_activity_enrollment_runs` / `listingkit_shein_activity_enrollment_items` | 六表的 AutoMigrate 只记录源码存在，不证明本次启动调用或实际安装；未来可删零消费者实现/测试，**不发 DROP、不清表、不接管历史任务**。当前 Store observations 保持独立 |
| Redis 旧运行态 | local runtime / 原 control-plane；如 `listing:daily:count:<tenant>:<store>:<date>`、`listing:remaining:quota:*`、`listing:task:pause:<platform>:<tenant>:<store>`、owner/queue mode/调度锁 | 配额 Lua、暂停/分配/锁原边界不变，不把应用层局部锁称为跨系统事务；不扫描、删除、改键或复制到当前 owner。部分键仍可能有 control-plane 消费者 |
| Redis 登录事实 / Profile | 原 `sheinlogin` 登录 worker，原 Cookie/attempt/验证码、stream、tenant+store Profile/锁/session；旧 local Cookie reader 只是另一个读取路径 | 删除旧刊登 Cookie reader 不删除登录 writer/数据或改变其 tenant-qualified 读写、成功提交和 UNKNOWN/取消边界；不执行 Profile/缓存维护 |
| RabbitMQ 消息 / 确认 | 原 control-plane/调度/consumer 与平台 task handler；含平台/店铺队列、task ID/claim、结果/重试与 DLQ | broker delivery/ack、DB claim/status、Redis 配额/锁和平台副作用是不同边界；未证明原全链 exactly-once。源码退休不 purge/delete/requeue/ack 消息，不将延迟/未知结果当成安全重发 |
| 外部调用与进程/文件 | SHEIN/TEMU 旧 processor/scheduler API、结果上报 HTTP；旧 logger/prompts/browser 临时目录 | 活动报名、价格/库存/刊登和结果上报不是新系统当前副作用合同。全部真实调用 NOT_RUN；源码删除不删除日志、文件、镜像、配置或 credentials |
| 当前业务事实 | 当前 Organization/Store、Sourcing/Catalog/Asset、CompletionDraft/Listing/Submission、Store observations 各原 owner | 原授权、幂等、fence、retry/UNKNOWN/恢复、官方应用凭据和 Temporal 队列保持；不以旧 Cookie/数字 Store/旧 sync feed 补齐缺失能力 |

本设计不改写原状态机来“安全兼容”退休路径。故障、响应丢失、重试、重启、取消和并发的原实现不自动成为当前业务保证；未完成/未知副作用留在原 owner，不用新登录、新任务或当前 Submission 重放旧任务。新系统按空业务数据正常创建，历史状态与真实资源处置不在本批授权内。

## 5. 相邻生产者与保留边界

架构上的实际耦合是：三个 CLI 只是旧链的消费/部分调度入口，`listing-control-plane` 仍能读原 Store/ImportTask、claim/publish、watchdog/recovery，并使用相同 DB/Redis/RabbitMQ 资源。因此**删除三个命令不能宣称整个数字任务链已退休**；也不能顺手删除 `app/taskstatus`、任务仓库/dispatch 协议、共享 DTO、原并发安全测试或 control-plane 的构建。

本设计候选保留 `cmd/listing-control-plane` / `internal/listingcontrol` / 其 runtime，以及 `shein-import-platform-recovery` 等非目标入口的源码，继续登记其独立债务和准确消费者。保留不把它们提升为当前 Product/Organization 架构权威，也不授权把它们接到 current-application。若后续需要一起退休，先交用户明确决定和相应准入，不在这三个 CLI 的实施中扩成全链重写。

源码树退休和已运行二进制/服务的停用是两项工作。本轮没有实例/消息拓扑证据，不推断旧实例存在或不存在。未来若获准实际停用，必须由原运行 owner 先明确同一范围内的生产者、消费者及在途副作用处置，避免仍在生产消息时先停消费、丢弃任务或重复发送；真实未完成资产/外部可观察契约一旦有具体证据，按 Hard-Cut 规则报告用户，不能自行加兼容/迁移。该限制仅适用于未来旧实例处置，**不是内部试用的新前置**。

## 6. 待确认删除矩阵与实施顺序

以下全部为设计候选，本轮生产源码/测试/构建/部署/数据均未修改。

| 工件 | 决策与删除条件 |
| --- | --- |
| 三个 `cmd`、`app/runtime/listing` / `listingscheduler` 的启动、flags/options、专属 health/debug/paused-recovery glue | RETIRE；用户确认本设计及正式 Ready 后物理删除。无命令别名、失败后 fallback 或借新命令继续运行旧实现；当前命令/登录 worker 的健康和关闭保持 |
| `app/bootstrap/listingruntime`、旧 shared resources/schedulers、`app/consumer` / `runner` / `scheduler` 及平台 module 专属接线 | RETIRE 候选；全部命令的实际消费者当前只有三 CLI，但须刷新全仓生产引用，按完整旧链排空。原 ports、通用 fetcher、task/workerpool/HTTP/browser/metrics/logging/队列基础层或其它仍有效消费者不 blanket-delete |
| `listingruntime/local` 旧 DB/Redis/client/factory/config wrapper、仅为旧链存在的 adapters | RETIRE 候选；零当前消费者后删除，禁止把旧 LocalDataProvider/RuntimeResources 换名包装为新能力。根 `listingruntime` 的仍有效合同保持；不删除真实 DB/Redis |
| `listingkit/sheinsync` 与 `listingkit/store` 精确 sync adapter / mem-store / 专属测试 | RETIRE 候选；local/shared synced-feed 等原消费者退出且全仓生产引用归零后删除。独立现有平台规则/当前 Store observation 与共享 API 不在整包连带删除范围 |
| `platforms/*`、`shein/*`、`temu/*` 的旧 pipeline/factory/调度/活动/价格/库存调用 | 只退休已确认旧 CLI 专属的 glue/实现。仍被当前 marketplace/publishing、采集、登录或其它有效 owner 消费的合同/规则/客户端保留；需要 EXTRACT 时逐项指明现有正确 owner、原有效行为和消费者，不能以搬目录代替退休 |
| `listingadmin`、根 `listingruntime`、`app/taskstatus`、control-plane 与其它工具共享工件 | 本候选保持；有实际剩余消费者。只可在未来精确引用归零且适用合同/安全断言已归正确 owner 后另行提出，不能整包或按数字类型删除 |
| `deployments/docker/Dockerfile.listing`、专属 build/push/deploy 脚本、配置与源码交付引用 | 先列清准确调用者。只为三 CLI 服务的构建目标/操作入口可 RETIRE；混合脚本中 retained control-plane、登录 worker、有效 shared runtime/base、安全 verifier 等不能连带删除。脚本中的 kubectl/docker/Redis 命令只读调查，未来源码删改也不得执行它们 |
| `deployments/kubernetes/{shein-listing,temu-listing}` 及 scheduler 声明、监控/代理/ownership/controller 等引用 | 不按目录整删：SHEIN 目录混有 control-plane/ownership-controller 等非目标对象。逐对象识别专属声明/引用、保留不在本次范围的对象和安全合同；源码清单变更不是 delete/apply/rollout/Secret/PVC/镜像/卷操作授权 |
| `Dockerfile.shein-login-worker`、浏览器 installer、`Dockerfile.shein-runtime` 等可复用 runtime/base、采集/当前应用 Compose 与私密 manifest | 保留现有有效 owner/消费者。不因 SHEIN 字样或同一历史镜像家族而清理；不重构基础镜像或接管原运行更新 |
| 原实现细节测试 / 有效安全测试 | 三 CLI 专属 flags、module 构造和旧协议细节分类 NOT_APPLICABLE 后随 owner 删除。当前隔离、权限/撤权、幂等/状态/claim、原 control-plane 并发/恢复、Cookie/attempt 与 worker 健康检查保持；只改依赖已退休文件的精确断言，复用原 guard，不新建验收平台 |

建议在同一主要实施 PR 依次完成：确认准确 Ready/测试分类与所有 caller → 退休三个入口/专属装配 → 排空零消费者的 local/sync/专属平台 glue → 校准专属构建/交付引用及原 guard → 集中候选 CI/独立检查。它们是实现/提交边界，不为此机械拆 PR；任何仍有有效消费者的工件留在原 owner 并如实记录，不为“清零”扩大删除。

### 6.1 必须明确确认的 CI 引用处置

基线 `.github/workflows/ci.yml` 的 Backend 仍执行 `go build ./cmd/shein-listing`，并保留 control-plane/runtime build、listingadmin claim/rollback/recovery race 等检查。直接删命令而保留编译引用会破坏 CI；整体跳过 Backend 或删共享安全回归不符合当前 Must。

未来用户确认退休时，需同时明确授权在**原 Backend job / 原构建步骤**中，将仅指向退休 `shein-listing` 的编译引用更新为保留的 `current-application` 与 `shein-login-worker` 目标，并按原 retirement/dependency guard 验证三命令及旧链没有恢复。原 control-plane 构建、相关并发/PG/权限回归、Required CI Gate、job 名称、trigger/concurrency/权限和分支保护均保持。此处是具体待确认改动，**本轮不修改 CI**；若未获该项授权，不能先删其目标再临时放宽检查。

## 7. 验证与交付停止点

本轮执行：准确源码/调用者调查、全部命令实际 `go list` 图与上述比对、持久化/副作用 owner 的源码映射、直接文档链接、范围/diff 检查及原适用文档 guard；原 PR CI/独立设计检查按准确候选记录在 PR。测试可编译源码，但不执行任何命令入口。所有真实 DB/Redis/RabbitMQ/平台/浏览器/实例与数据项目 NOT_RUN。

后续正式实施复用原检查，当前全部仍为待执行：

- 在原 hard-cut / actual dependency guard 里先 RED 捕获目标，再 GREEN 证明命令/专属旧链缺席；当前采集、Store observation、图片 worker、独立登录 worker与保留 control-plane 的依赖/健康/关闭/有效合同不变。
- 保留当前官方 Store/Organization/Submission 的授权、隔离、幂等和 UNKNOWN/恢复回归；保留原 control-plane/任务仓库尚有消费者的 claim/rollback/recovery 与 worker Cookie/attempt 回归。
- 必要的当前命令编译、原准确候选 CI 和最终集中独立检查；只验证真实组合风险，不人工重复原 CI 全部矩阵，不追加 runner 或人工流程。
- 仅源码/打包引用退休，不读取/清理历史业务记录、队列、对象、镜像、卷、Profile、日志或 secrets；未执行的真实验收继续 NOT_RUN。

设计交付停在 **In Review / 待用户确认**。用户需确认 §5 的保留生产者/共享边界、§6 的精确退休范围及 §6.1 的 CI 编译引用处置，随后另行记录正式实施 Ready；不自动实施、合并、部署或关闭 #29/#301。生产仍 NOT_READY，原内部试用和 #717/运行更新继续由各原 owner 按已有授权推进。
