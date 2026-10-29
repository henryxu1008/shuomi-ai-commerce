> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。仅替代本文件 listingruntime.ProductImportMapping/原 types.go 必须存在、旧 MappingInfo 编码及 listingruntime 仍属当前依赖的准确保留含义；本轮最后值类型完整退休。SourceIdentity 四 JSON 零值槽位、六 Source*、原 trim/规范化/hash/读取拒绝/当前版本及 opaque metadata 继续保留，不删当前序列化字段或改变业务事实 owner。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

# 旧刊登 CLI 与调度链退休边界设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29)、[#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **用户已确认 §5、§6、§6.1；Reuse Existing Architecture / Ready 后正式实施**。
- 实施 Design Basis: **Reuse Existing Architecture**；只退休已登记旧入口及零消费者专属链，不改变保留 owner 的权限、状态机、持久化、恢复与副作用合同。#29/#301 在生产修改前已记录 Ready/In Progress。
- 调查基线：`main @ 8a3f97850d04d2109bda4b432cbc0361289c04b9`。
- 用户决定：先设计 `listing-scheduler`、`shein-listing`、`temu-listing` 与调度链的退休，确认后再实施；保留当前采集、Store 观察同步、有效共享基础层和独立 SHEIN 登录 worker，不操作运行实例或数据，不增加内部试用前置。

## 1. 用户结果、Authority 与非目标

后续 [control-plane / 数字任务派发恢复设计](legacy-listing-control-plane-retirement-boundary.md) 已获用户确认并在#726实施两剩余入口及专属编排退休。本文件§5的control-plane保留要求是#723历史批准边界，已由该后续有界决定替代；任务仓库、安全检查、有效共享合同及登录worker仍保留。不宣称全部数字任务清零，不授权运行实例/数据操作或新增试用前置。

维护者能够清楚区分当前应用的有效入口与旧数字账号刊登/定时任务运行链，确认后才删除能重新启动旧链的源码及专属构建接线。当前用户继续沿已有内部试用应用完成采集、保存和草稿操作，Store 观察同步及独立登录 worker 按各自原合同保留。本设计不交付新刊登能力，也不证明任何旧实例正在运行或已安全停用。

适用依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md) 与 [module mapping](../refactoring/module-target-mapping.md)：EXTRACT / RETIRE；不恢复 Task-first、旧新 ID 映射、迁移、fallback、双读双写或第二事实源。
- [项目领域边界](project-boundaries.md)、#29 已合并的 root Task/API/Workspace 退休：旧 standalone numeric leaves 仍是债务，不能因仍可编译而成为当前 Product/Listing owner。
- [当前 Store](store-center-current-application-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[观察内部准备接线](store-observations-preparation-runtime.md) 与 [当前 SHEIN 供货链](my-supply-chain-shein-v1.md)：当前 Organization/Store、官方连接与 Submission 保持各自原 owner；缺配置仍按原范围不可用。
- [独立 SHEIN 登录 worker 边界](shein-worker-identity-persistence-boundary.md)：原账号仓库、Cookie/attempt/验证码/Profile/健康与关闭保留，查重按已有 tenant 隔离修复；保留该 worker 不授权其它旧刊登/调度入口。
- [发布预检退休](release-identity-preflight-retirement.md)：原安全验证与独立 worker 合同保留，旧发布入口不能被改造成 current-application 的交付通道，生产仍 NOT_READY。

本轮无 UI/IA 改动，Figma N/A。不新建 IAM、Provider、队列、状态平台、迁移/备份工具、验收 runner 或新恢复协议。不执行业务命令入口、Docker、Kubernetes、DB/Redis/RabbitMQ 查询、schema 初始化、浏览器或平台调用；不读取真实配置、账号、凭据、Profile 或运行数据。

## 2. Design Basis 与实施准入

设计阶段 N/A 仅适用于原文档候选。用户于 2026-10-16 明确确认 §5、§6、§6.1，实施严格限定为本设计登记旧路径 RETIRE、零消费者专属实现/测试和必要引用校准。正式修改前，#29/#301 已按 **Reuse Existing Architecture** 记录原合同、owner、不变量、验收及 Ready；起始 main `15200f4905ecc10f81c1c330c03d750061d9fbb7` 已同步。

该路径不新增产品流程、事实/数据库 owner、权限模型、状态/事务/恢复边界或外部副作用。预期会跨多个旧子系统、文件数可能超过提示线；适用依据是既有整体 Hard-Cut 和明确保留边界，而非按文件数机械拆 PR。保持一个主要 Delivery Batch；实现/提交可分为可验证的小步骤。

若实际需要改变当前消费者的业务语义、事实 owner、授权/组织边界、状态机、跨数据库一致性、补偿/retry/UNKNOWN 或平台发送边界，必须停止对应正式实现，改按 Independent Architecture、适用独立检查及 IMPLEMENTATION_READY。不得借退休实现新的当前执行器，也不得把旧 DTO/Service 改名或移目录继续供当前业务调用。

本批不新增高风险边界，不重复全局 Architecture Review；最终源码差异仍在原 #723 集中独立检查。用户确认记录及实现前准入见 #29/#301，旧文档 CI 不作为实现候选证据。

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

本批保留 `cmd/listing-control-plane` / `internal/listingcontrol` / 其 runtime，以及 `shein-import-platform-recovery` 等非目标入口的源码，继续登记其独立债务和准确消费者。保留不把它们提升为当前 Product/Organization 架构权威，也不授权把它们接到 current-application。若后续需要一起退休，先交用户明确决定和相应准入，不在这三个 CLI 的实施中扩成全链重写。

源码树退休和已运行二进制/服务的停用是两项工作。本轮没有实例/消息拓扑证据，不推断旧实例存在或不存在。未来若获准实际停用，必须由原运行 owner 先明确同一范围内的生产者、消费者及在途副作用处置，避免仍在生产消息时先停消费、丢弃任务或重复发送；真实未完成资产/外部可观察契约一旦有具体证据，按 Hard-Cut 规则报告用户，不能自行加兼容/迁移。该限制仅适用于未来旧实例处置，**不是内部试用的新前置**。

## 6. 已确认删除矩阵与实施顺序

以下矩阵已由用户明确确认；源码退休与运行实例/数据操作分开，后者未授权。

| 工件 | 决策与删除条件 |
| --- | --- |
| 三个 `cmd`、`app/runtime/listing` / `listingscheduler` 的启动、flags/options、专属 health/debug/paused-recovery glue | RETIRE；用户确认本设计及正式 Ready 后物理删除。无命令别名、失败后 fallback 或借新命令继续运行旧实现；当前命令/登录 worker 的健康和关闭保持 |
| `app/bootstrap/listingruntime`、旧 shared resources/schedulers、`app/consumer` / `runner` / `scheduler` 及平台 module 专属接线 | RETIRE；全部命令的实际消费者当前只有三 CLI，但须刷新全仓生产引用，按完整旧链排空。原 ports、通用 fetcher、task/workerpool/HTTP/browser/metrics/logging/队列基础层或其它仍有效消费者不 blanket-delete |
| `listingruntime/local` 旧 DB/Redis/client/factory/config wrapper、仅为旧链存在的 adapters | RETIRE；零当前消费者后删除，禁止把旧 LocalDataProvider/RuntimeResources 换名包装为新能力。根 `listingruntime` 的仍有效合同保持；不删除真实 DB/Redis |
| `listingkit/sheinsync` 与 `listingkit/store` 精确 sync adapter / mem-store / 专属测试 | RETIRE；local/shared synced-feed 等原消费者退出且全仓生产引用归零后删除。独立现有平台规则/当前 Store observation 与共享 API 不在整包连带删除范围 |
| `platforms/*`、`shein/*`、`temu/*` 的旧 pipeline/factory/调度/活动/价格/库存调用 | 只退休已确认旧 CLI 专属的 glue/实现。仍被当前 marketplace/publishing、采集、登录或其它有效 owner 消费的合同/规则/客户端保留；需要 EXTRACT 时逐项指明现有正确 owner、原有效行为和消费者，不能以搬目录代替退休 |
| `listingadmin`、根 `listingruntime`、`app/taskstatus`、control-plane 与其它工具共享工件 | 本候选保持；有实际剩余消费者。只可在未来精确引用归零且适用合同/安全断言已归正确 owner 后另行提出，不能整包或按数字类型删除 |
| `deployments/docker/Dockerfile.listing`、专属 build/push/deploy 脚本、配置与源码交付引用 | 先列清准确调用者。只为三 CLI 服务的构建目标/操作入口可 RETIRE；混合脚本中 retained control-plane、登录 worker、有效 shared runtime/base、安全 verifier 等不能连带删除。脚本中的 kubectl/docker/Redis 命令只读调查，未来源码删改也不得执行它们 |
| `deployments/kubernetes/{shein-listing,temu-listing}` 及 scheduler 声明、监控/代理/ownership/controller 等引用 | 不按目录整删：SHEIN 目录混有 control-plane/ownership-controller 等非目标对象。逐对象识别专属声明/引用、保留不在本次范围的对象和安全合同；源码清单变更不是 delete/apply/rollout/Secret/PVC/镜像/卷操作授权 |
| `Dockerfile.shein-login-worker`、浏览器 installer、`Dockerfile.shein-runtime` 等可复用 runtime/base、采集/当前应用 Compose 与私密 manifest | 保留现有有效 owner/消费者。不因 SHEIN 字样或同一历史镜像家族而清理；不重构基础镜像或接管原运行更新 |
| 原实现细节测试 / 有效安全测试 | 三 CLI 专属 flags、module 构造和旧协议细节分类 NOT_APPLICABLE 后随 owner 删除。当前隔离、权限/撤权、幂等/状态/claim、原 control-plane 并发/恢复、Cookie/attempt 与 worker 健康检查保持；只改依赖已退休文件的精确断言，复用原 guard，不新建验收平台 |

建议在同一主要实施 PR 依次完成：确认准确 Ready/测试分类与所有 caller → 退休三个入口/专属装配 → 排空零消费者的 local/sync/专属平台 glue → 校准专属构建/交付引用及原 guard → 集中候选 CI/独立检查。它们是实现/提交边界，不为此机械拆 PR；任何仍有有效消费者的工件留在原 owner 并如实记录，不为“清零”扩大删除。

### 6.1 已确认的 CI 引用处置

基线 `.github/workflows/ci.yml` 的 Backend 仍执行 `go build ./cmd/shein-listing`，并保留 control-plane/runtime build、listingadmin claim/rollback/recovery race 等检查。直接删命令而保留编译引用会破坏 CI；整体跳过 Backend 或删共享安全回归不符合当前 Must。

用户已同时明确授权在**原 Backend job / 原构建步骤**中，将仅指向退休 `shein-listing` 的编译引用更新为保留的 `current-application` 与 `shein-login-worker` 目标，并按原 retirement/dependency guard 验证三命令及旧链没有恢复。原 control-plane 构建、相关并发/PG/权限回归、Required CI Gate、job 名称、trigger/concurrency/权限和分支保护均保持。本批仅执行此处两目标替换；不跳过、删除或放宽任何安全/必需检查。

## 7. 实施范围、验证与交付停止点

实际全仓生产引用与所有保留 cmd 图核对后，已排空并物理退休：三个 cmd、两个专属 runtime、旧 app/bootstrap（含孤立根装配、fetchers/processors）、consumer/runner/scheduler、listingruntime/local、listingkit/sheinsync/store、platforms module、SHEIN/TEMU scheduler 和只消费旧任务的 SHEIN pipeline/TEMU 根 processor。platformtask、scheduler alias 及 platformbase 的 BaseFactory/调度展示/转换三个文件随唯一调用者退出；platformbase 其余中立 fetcher/error 合同、平台子包/客户端/规则保留。未按平台或共享包名整删。

旧 CLI flags/debug/recover-paused-tasks、module/processor 构造、数字任务 pipeline/调度协议与旧 local/sync 实现细节测试随 owner 退休，分类 **NOT_APPLICABLE**：当前用户操作不再经过它们。仍有效的当前授权/租户隔离、官方 Store/Submission 幂等/UNKNOWN、原 control-plane claim/rollback/recovery、独立 worker Cookie/attempt/账户/健康/关闭测试保持。针对已消失路径的架构细节守卫同步退出，由既有源缺席与真实命令依赖 guard 覆盖完整退休集合；不采用遇缺文件就 skip。

精确交付引用：旧 Dockerfile.listing、三个 CLI 专属 build/push/deploy、单店同步与 shard rollout 入口退休；Makefile 改为构建 current-application、control-plane 和原独立 login worker。原 Backend job 内只替换 shein-listing 编译目标。Kubernetes 只移除旧消费/调度 workload、专属服务/HPA/patch 和模板，并校准原 kustomization 引用；Namespace、配置、Secret/ExternalSecret、监控、安全声明保持。混合目录中 control-plane 与 ownership-controller 保留；后者仍引用原历史镜像，**不是当前源码可重建入口**，本批不更新或执行。control-plane 继续使用其原专属构建脚本/Dockerfile，独立 login worker 使用原 Dockerfile/runtime/base。

验证复用原检查：retirement/build-target guard 已先 RED 捕获旧入口，再验证 GREEN；当前采集、Store observations、图片/Temporal、登录 worker 与保留 control-plane 用实际 Go 图与相关编译/原回归核对。原 PG、money、claim/rollback/recovery race、Required CI Gate 等保持，稳定候选 CI 和集中独立检查结果按准确 SHA 写 #723；旧文档候选 CI 不作为本批验证。未变代码的历史证据如复用须明确，不能称为新 SHA 直接执行。

交付状态以 #723 当前证据为准；完成实现后 In Review，待独立验证/用户试用，合并另按最终准确 HEAD 授权。当前应用沿原私密 manifest 入口，独立登录 worker 沿原命令与私有配置；本批不增加启动或内部试用前置。**不执行入口、Docker/Kubernetes、真实 DB/Redis/RabbitMQ、浏览器/provider 或数据操作**；不迁移/删除表、记录、消息、镜像、卷、Profile、日志或 secrets。真实运行和用户验收 NOT_RUN，生产 NOT_READY，#29/#301 保持 OPEN。
