# 旧 ownership-controller 与 auto-shard 配置退休设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **DESIGN_REVIEW / 待用户确认；仅设计，尚未授权实施**。
- 本轮 Design Basis: **N/A**。只读调查与文档不改变产品、领域合同、业务运行、权限、持久化或外部副作用；不修改 Go、测试、CI、配置或部署源。
- 调查基线：`main @ 05ae7947f8558ebb31f69e1b251482458ac80c29`（含 #729 的源码退休）；准确设计 HEAD、CI 和独立复核记录在关联 PR。
- 用户范围：先明确旧 ownership-controller 部署声明与 auto-shard 专属配置的消费者、持久化 owner 和删除边界，设计确认后再实施。保留当前采集、Store 观察同步、有效共享合同和独立 SHEIN 登录 worker；不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、Authority 与非目标

维护者获得一份可直接审查的源码删除矩阵：消除旧执行器已不存在、部署声明仍指向历史镜像、共享配置仍描述旧分片能力的脱节。当前内部试用继续使用原入口，不以本设计、后续源码退休或旧环境处理为前置，也不新增刊登/协调/恢复能力。

依据为 [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)，以及已确认的 [三 CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md) 退休边界。前批保留 ownership-controller 声明和共享配置的事实仍然有效；本提案不把未实施的删除写成完成。

当前能力继续服从 [Store 当前应用](store-center-current-application-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[观察准备接线](store-observations-preparation-runtime.md)、[独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md) 和 [发布预检退休](release-identity-preflight-retirement.md)。无 UI/IA 变化，Figma N/A。

Must：当前能力、原授权/租户/身份、事实 owner、安全检查和共享合同保持；不复活旧 CLI/worker/消息链，不引入 fallback、双读双写、ID 映射、迁移或第二事实源。Should：现行文档准确区分已退源码、拟退声明及未知运行态。Out of Scope：真实集群/镜像/配置/凭据/Redis/DB/队列/Profile 的探测或处置、排空/恢复/补偿、当前产品扩张与通用治理/验收工具。本提案不扩大 Threat Model，不新增 Accepted Risk，不把未知运行态写成安全或已验收。

## 2. Design Basis 与正式开工条件

本轮是 N/A 文档工作；下列矩阵是**待确认提案**，不是正式源码实施准入。用户确认后，拟按 **Reuse Existing Architecture** 在 #29/#301 正文记录用户结果、批准版本、当前 owner、调用路径、不变量、验收和 Ready，再在同一主要 PR 继续。复用原 Hard-Cut 和保留合同，不新建事实/schema owner、授权、状态机、事务/retry/UNKNOWN/recovery 或外部副作用。

实现范围跨 Go 配置、静态声明和测试，是对现有批准退休的有界收尾，不是新的跨系统运行协议。文件数或子系统提示线须按实施前实际 diff 核对并说明既有批准矩阵覆盖；不机械拆 PR，也不为删除发明新平台。

若发现实际当前执行消费者、具体外部可观察合同，或必须改变保留消费者的身份/权限、领域合同、事实 owner、持久化、状态/事务/恢复或不可安全重复副作用，停止对应正式实现并报告。需要新边界时转 Independent Architecture、适用独立评审与 IMPLEMENTATION_READY；不以旧文档/镜像、测试、无调用推断兼容例外。

## 3. 实际 contract → implementation → injection → consumer

只读执行 `go list -f '{{.ImportPath}}|{{join .Deps ","}}' ./cmd/...` 并核对全仓已跟踪 Go 符号引用；没有运行任何 main。基线有 **31 个命令根，14 个编译依赖 internal/core/config**；包级编译可达不代表某字段被执行器消费。

| 对象 | 基线实际消费者/接线 | 结论 |
| --- | --- | --- |
| `ownership-controller.yaml` | 唯一 Kustomize 引用为同目录 `kustomization.yaml` 的唯一 resource；Deployment 名为 shein-listing-ownership-controller，固定历史镜像，command 为 `/app/listing-service` | 没有当前 ownership-controller 命令/重建入口；声明仍在，不代表实例存在或健康 |
| 声明的外部引用 | `shein-listing-config-heavy` 的唯一仓库消费引用是拟退 workload；`shein-listing-config-lite` 无仓库消费者。prod-store-auto-shard 的 kustomization 只列这两个 ConfigMap，无其它入口引用该 overlay | 两个特定 ConfigMap 是旧专属配置资源，不因内容含通用字段就成为有效共享合同；与其唯一 Kustomize 装配一并列入有条件 RETIRE。Secret/其它共享对象不随之删除 |
| `RabbitMQConfig.AutoShard` | `BuildConfig → BuildRabbitMQConfig` 装配 Enabled/Role/Platform/Interval/PageSize/LockKey/LockTTL/CandidateNodes/TargetStoresPerNode/NodeWeights | 共享配置包仍加载旧字段，包本身必须保留 |
| auto-shard 类型/行为 | 三个 AutoShardRole 常量、AutoShardConfig、EffectiveRole/HasValidRole/IsCoordinator/IsWorker；生产引用只在 config 类型、loader、validator 内 | 全仓没有 config 外的 AutoShard 字段/方法执行消费者；AutoShardCoordinator、RedisStoreAssignmentProvider 及历史 Redis ownership key 的当前 Go 消费者均为零 |
| 环境绑定/校验 | config.go 的 9 个 autoShard 已知键各绑定 primary `TASK_PROCESSOR_RABBITMQ_AUTO_SHARD_*` 与 deprecated `RABBITMQ_AUTO_SHARD_*`；validator 在 RabbitMQ/autoShard enabled 时校验旧 role、candidateNodes、interval、lockTTL | 只删字段还会遗留无能力的配置表面；必须连同精确绑定/校验闭包退休。NodeWeights 当前没有同组显式绑定条目，不虚构第 10 项 |
| 两份 YAML 配置输入 | prod-store-auto-shard 的 heavy/lite ConfigMap 各有嵌入 YAML 的 `rabbitmq.autoShard` 子树；README 仍给出 `kubectl apply -k` 的旧操作 | 实施基线刷新后若消费者仍仅为拟退 workload/无其它消费者，整体退这两个 YAML 和 overlay kustomization；README 改为退休说明，不能保留可重新创建旧配置的入口 |
| 共用解析 helper | getStringSlice 还用于 proxy/授权等列表；getStringIntMap 还用于 Amazon concurrencyControl.perRegion | helper 保留，不能因旧 NodeWeights/candidateNodes 使用它们就删除 |
| 当前命令 | current-application、shein-login-worker、1688-local-agent、image-agent-temporal-worker 及另 10 个 schema/准备命令编译依赖 core/config；其它 17 个命令也在完整根列表内核对 | 不删共享 config/RabbitMQ，不执行命令或 schema 工具；当前采集、Store 观察和 worker 接线不改 |

架构问题在于：没有当前执行器的 auto-shard 字段仍参与共享装配和条件校验，而历史 workload 又固定到不可由当前源码重建的旧制品。只删 workload/子树会遗留孤立的旧配置 overlay；特定资源有无当前消费者与其内容是否通用必须分别判断。采取 RETIRE 整个专属闭包，保留有效共享行为，不重建协调器或增加 disabled/stub 包装。

## 4. 持久化 owner 与失败/恢复边界

| 边界 | 可确认的所有权/证据 | 本提案决定 |
| --- | --- | --- |
| Kubernetes Deployment | 静态名称/引用由声明定义；实际对象若存在，其 Kubernetes reconciliation 和历史镜像进程归原运行 owner | 删除源码文件不等于 delete/apply/prune 或停止 Pod；不查询集群，不改变副本/镜像/节点/权限 |
| 历史动态 ownership | [2026-06-23 设计](../superpowers/specs/2026-06-23-shein-listing-ownership-controller-design.md)约定 coordinator 写 `listing:queue:owner:{tenant_id}:{store_id}`、`listing:queue:mode:{tenant_id}:{store_id}`、`listing:queue:owner:node:{node_id}`，worker 读节点集合 | 这是历史协议说明，不是当前镜像/实例/键的实测；原运行 owner 不变，不扫描/删除/改写键，不转交当前 worker |
| 分片锁/租约 | 声明/样例配置指定 `listing:queue:auto-shard:lock`、lockTTL、candidateNodes/weights；声明还设置 Redis DB 9 | 配置数值不证明实际 TTL、释放、fencing、唯一 writer 或无在途操作；不删除锁、调整 TTL 或建设新的 recovery owner |
| Store/任务持久化与 broker | 历史设计从管理 Store API 读取资格；listingadmin/根 listingruntime 仍有其它有效消费者和原 owner。DB、Redis、broker、平台不是共同事务 | 原表/行/索引、owner/tenant、队列/消息与 claim/recovery 安全合同保持；不按数字字段或无此控制器推定可删仓库，不重放/接管任务 |
| ConfigMap / Secret / 镜像 | 静态配置是输入，两个专属 ConfigMap 源文件拟退休；实际 ConfigMap/Secret/外部凭据及镜像原制品各有原 owner，不读取真实内容 | 仅删仓库专属 YAML/装配，不 delete/apply/prune 真实 ConfigMap；其它共享配置、Secret/ExternalSecret/Namespace、镜像及历史资源均保留 |
| 当前业务与独立登录 | 当前 Organization/Store/Product/observations/Submission 各原 owner；独立登录的原数字账号、DB、Redis Cookie/attempt/验证码、Profile、锁、Store 同步与 health/shutdown 归原 worker | 全部保留，不把队列 ownership 当当前 Organization ownership，不搬运身份、键、Profile 或 retry/UNKNOWN 责任 |

响应丢失、重试、重启、并发或部分 Redis 写入的历史运行结果不能从这份 YAML 和历史设计推导；这里没有核验历史镜像实现，也没有真实运行态证据。源码退休不宣称排空成功、租约安全、跨 DB/Redis 原子或 exactly-once，不新增 outbox/Saga/reconciler/故障注入平台。真实运行/用户/平台验收 NOT_RUN，生产 NOT_READY；这些未知不是内部试用的新前置。

## 5. 拟议删除矩阵与保留边界

| 源码/声明 | 用户确认后的拟议处置 | 精确限制 |
| --- | --- | --- |
| `deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/ownership-controller.yaml` | RETIRE 整个旧 workload 文件 | 仅仓库声明；不操作集群/镜像/Secret 或实际实例 |
| 同目录 `kustomization.yaml` | RETIRE 只有该 workload 的装配文件 | 不留下空协调器/替代入口；README 可留退休说明。不递归删除目录；先刷新无其它 resource/消费者 |
| `internal/core/config/type_rabbitmq.go` | 精确 RETIRE AutoShard 字段、AutoShardConfig、3 个 AutoShardRole 常量及该接收者的 4 个方法；只去掉因此闲置的 strings import | RabbitMQConfig 其余字段、NodeRole/NodeConfig 和同名 NodeConfig.HasValidRole 等全部保留；time import 仍有效 |
| `internal/core/config/loader_rabbitmq.go` | 仅退 AutoShard 构造子块 | BuildRabbitMQConfig 签名、Node/Consumer/ResultReporter/LoadMonitor/DeadLetter/watchdogs/Deduplicator/StoreAPI 及其它参数保持 |
| `internal/core/config/config.go` | 仅退上述 9 个 autoShard 已知键的 primary/deprecated 绑定项 | 其它绑定、dotenv、Viper、默认值/通用解析、加载与验证入口保持；不引入新全局 strict-key checker、旧键 shim 或自动重选 |
| `internal/core/config/validator_rabbitmq.go` | 仅退 `if rabbitmq.AutoShard.Enabled` 子树 | nil/disabled、URL、重连、prefetch/retry、Node role/concurrency/ports/ownedBuckets 等原校验不变，不整文件删除 |
| `deployments/kubernetes/shein-listing/overlays/prod-store-auto-shard/configmap-heavy.yaml`、`configmap-lite.yaml` 及该目录 `kustomization.yaml` | 实施刷新后仍无保留消费者时，RETIRE 这三个专属源文件 | heavy 唯一仓库消费为拟退 workload、lite 无消费者、装配只列二者；不能只删 autoShard 子树后留下旧配置部署入口。不触及实际 ConfigMap 或其它仍有消费者的配置/schema/helper；不递归删除目录 |
| 通用解析/邻接能力 | 保留 getStringSlice、getStringIntMap、loader_helpers/loader_builder 及其当前调用；保留 Node、DeadLetter、watchdog 等其它配置 | 不把无本控制器消费者外推为其它配置全部退休；不搬迁/包装旧协议，不处理未来独立债务 |
| 当前命令/worker/共享仓库与配置资源 | 保留 current-application、采集、Store observations、独立登录、有效平台 API/规则/合同及 listingadmin/listingruntime | 不修改真实 manifest/环境文件，不删持久化实体/索引/数据，不重建旧分片消费者 |

移除旧键的正式类型、绑定和 getter 不等于新增旧配置兼容。通用 Viper 对非消费/未知键的原处理保持；不为本次退休增加拒绝任意原始 Viper.Get 的新公共合同，也不保留旧执行开关。无保留消费者的专属样例整体退出，不把它们包装成共享部署模板；实际历史环境输入的处置仍归原运行 owner，本批不读取或自动改写它们。

## 6. 原测试、文档与安全断言的处置

本轮只写文档，下表为确认后实施的测试方案；没有把尚未执行的 RED/GREEN 记成 PASS。

| 位置 | 拟议最小处置 | 必须保持 |
| --- | --- | --- |
| `internal/core/config/auto_shard_role_test.go` | RETIRE 整个专属角色/默认/校验测试文件和其专用 fixture | 只证明旧 owner 行为，不为保活旧测试增加新协调器 |
| `internal/core/config/config_env_test.go` | primary binding 混合测试仅移除 4 个 autoShard Setenv 和对应 4 个断言；CSV 测试保留原测试与语义，fixture 改用现有 `LISTINGKIT_PLATFORM_ADMIN_USERS` → `listingkit.platformAdminUsers` | 其它 Amazon/RabbitMQ Node/端口/OpenAI/商业 DB/授权/配置源/取消测试与 CSV 解析能力完整保留，不整删混合测试或通用 helper |
| `tests/listingkit_backend_retirement_test.go` | 在原退休 guard 增补五个 YAML 的 absence（两个旧 workload overlay 文件、两个专属 ConfigMap 和其 kustomization）与专属 config 声明/绑定无复活检查；原 TestRetiredListingCLIsHaveNoBuildOrDeploymentEntrypoints 的必读文件列表仅移除已退 ownership overlay kustomization 一项，改由 absence 覆盖 | 不遇缺文件跳过、不放松其它必读项；所有原旧 CLI/runtime 禁止复活、实际命令依赖和当前两命令 build 断言保持 |
| 共享配置回归 | 复用原 core/config 测试；同源未知配置/CSV、getStringIntMap 的 Amazon perRegion、Node 角色和 RabbitMQ 有效校验按真实影响验证 | 不新增配置框架、runner 或验证工具的工具；不删合法安全检查，不把原始 Viper 返回值当旧执行能力 |
| 现行文档/清单 | 两个 overlay README 均改为退休说明并移除旧配置部署操作；校准 SHEIN 根 README、Legacy register、repository-structure/dependency-baseline 的实际现行引用；前批设计用明确后续状态说明，不回写历史当时证据 | 不保留 `kubectl apply -k .../prod-store-auto-shard` 旧入口或假称仍有共享 ConfigMap 消费者；不将未知实例标记已退，不把未触及模块列成已清零 |
| 历史设计/计划 | 保留 2026-06-23 原文，加明确当前状态/替代依据链接 | 历史角色兼容、Redis 读写/锁清理或 rollout 步骤不是当前授权，不继续执行旧计划 |

### 6.1 CI 与构建目标

**两套 workflow、Makefile、保护规则及现行命令目标零修改。** 当前没有 ownership-controller 独立 build/CI 目标；不能为退休配置减少 Backend/Architecture/CodeHealth/PG/资金/并发/发布权限等必要和安全检查。保留 current-application 与 shein-login-worker 两 build、listingadmin 三项 race 及其它原检查。原路径选择的 SKIPPED 与 PASS 分列；不为了本提案追加试用门槛或验收平台。

## 7. 确认后实施顺序与删除条件

1. 用户确认本设计 §5、§6、§6.1 和本节后，在 #29/#301 记录 Reuse Existing Architecture / Ready；核对最新 main、当前消费者和单 Writer，保持同一主要 PR。若矩阵外出现消费者，先报告，不为让测试通过扩大删除。
2. 在原 guard 增补专属闭包 absence，先确认 RED 捕获仍在的声明/配置，再按矩阵精确删除并收敛 GREEN；混合 helper/测试按 §6 保留或改 fixture。对实际影响运行原 core/config 与必要共享/worker 回归，核对所有命令编译图，不执行 main、schema、provider 或运维入口。
3. 重新核对两个命名 ConfigMap 和 overlay 的仓库实际消费者；若仍仅为拟退 workload/无其它消费者，连同三个专属源 YAML 及 README 旧 apply 入口退出。保留其它有效配置 schema/helper/资源与生产路径，真实 ConfigMap/数据不动；核对 CI/构建/保护无改动。只读源码解析/编译不是 kubectl apply、server-side dry-run 或真实环境验收。
4. 稳定候选提交/推送，完成适用必需 CI 和一次集中独立最终检查；绑定实际 HEAD 与组合，不把设计 CI/旧 SHA/作者自检写成产品验收。提交范围内真实问题沿原 PR 分类修复；达到实现 In Review 后等待用户决定，合并另需最终准确 HEAD 授权。

源码删除条件是：批准矩阵、适用 Ready、消费者闭包核对、有效合同/安全检查保持及准确候选证据满足。**不存在以真实旧环境排空/删锁/删数据为这批源码合并或内部试用前置的条件。** 不复制历史配置到新 owner，不将镜像或运行态处置塞入 PR。若以后要停止/归档旧实例，由用户另定准确环境/动作与原运行 owner；本设计没有发出该操作指令。

```text
Legacy decision: RETIRE（提案待确认）
Reusable behavior: 通用配置/解析/有效校验、当前采集/观察/平台合同及独立登录能力保留。
Current owner: 当前各 Domain/Capability/config 与独立 worker；历史集群/DB/Redis/broker/镜像各原运行/持久化 owner。
Cutover/deletion condition: 用户确认 + 适用Ready + 本矩阵专属消费者/检查满足后删源码；不操作或转交真实资源/状态。
```

## 8. 本轮交付与停止点

本轮交付本提案、索引和历史上下文标注；仅做现有文档/link/diff 检查与适用 PR CI/独立最终复核。设计结果不等于删除已执行、IMPLEMENTATION_READY、合并许可、真实能力验收或生产发布。用户确认前不改正式 Go、测试、配置、CI 或部署源；#29/#301 保持 OPEN。
