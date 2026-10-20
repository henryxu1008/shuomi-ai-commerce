# 旧数字刊登运维源码：完整退休闭包设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **IMPLEMENTATION_READY；用户已确认冻结设计 `0871cb591` 并授权按§5/§6/§6.1/§7在 #839 整批实施；源码退休已完成，待原CI与最终独立检查/合并授权。** Design Basis: **Independent Architecture**。静态调查基线 main `dd169971651d02ea4b6d81e0b44d6c32e10c6724`（2026-10-20，#836 合并后）；准确设计 HEAD、检查结论与确认写主要 PR。一个 Writer、一个完整运维源码闭包，不按目录、平台或文件数拆 PR。设计先于实现；用户确认及两Issue Ready记录已读回（PR评论6281156340），再修改正式源。准确实施HEAD/测试/CI/最终检查留在PR；不是部署、运行清理或用户验收。

## 1. Product Outcome / Scope

使用者：维护当前应用、来源采集、图像、Store 观察及独立 SHEIN 登录/Store 同步的操作者。可见结果：仓库不再把已经退出的数字刊登配置、分片 rollout、旧任务修复 SQL、旧监控看板及 TaskEvent 兼容观察窗口当成当前运行入口；有效共享配置、凭据装配和监控保护有明确 owner。

同一批覆盖部署装配 → 专属配置/Secret 声明 → 监控采集/告警/看板 → rollout helper/test → 旧恢复 SQL → 当前文档及原 guard 的直接尾部。仅删两个 ConfigMap、只删看板不处理共用存储告警或留下旧 apply 建议，都不算闭包完成。源码动作不以旧实例停用或清理为前置，也不证明其停用。

保留此前全部能力：当前采集/workerpool/通用 queue/Retry、平台 API/DTO/纯规则、两种 SHEIN 属性、AI/SKU/spec 映射、PriceHandler/PriceManager/PriceCalculator、图像/AI/prompt、当前身份/tenant/owner/角色/资金/计费、当前 Store 及观察、独立 SHEIN 登录/查重/Redis/Profile/Store 同步。当前应用继续沿已有交付路径；不增加内部试用前置，生产 NOT_READY 不因此改变。

## 2. Design Basis / Authority / Threat Model

**Independent Architecture**：本批重新界定旧部署源与仍有消费者的共享资源，抽取一条跨应用的有效监控规则，范围预计超过30个文件。它不改变业务事实 owner、领域合同、身份/租户策略、持久化 schema、状态机或外部平台操作；不能用 N/A 或机械拆批替代明确边界。

Authority：[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[原 CLI 边界](legacy-listing-cli-retirement-boundary.md)、[ownership-controller 边界](legacy-ownership-controller-retirement-boundary.md)、[#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#836 配置闭包](legacy-runtime-config-closure-retirement-boundary.md)及本轮用户“完整闭包、先设计”的决定。此前保留部署/监控对象的表述，只在本文§5准确对象及用户确认后被替代；不是重开已冻结批次。UI/Figma N/A。

- Must：闭包完整；不删除仍服务当前能力的声明、凭据装配、通用工具或告警；不执行任何运行/数据操作，不削弱原必需/安全检查。
- Should：复用原 Prometheus Operator CRD、Kustomize 和退休 guard；有效规则只作必要归属抽取，不建通用框架或验证平台。
- Out of Scope：当前应用发布链、两套 CI 或任何 workflow、#692 Store schema、通用监控/告警平台重构、端口转发重写、DB 查询工具退休、依赖/ceiling/inventory/保护规则调整、凭据轮换、历史文档重写，以及兼容/fallback/双读双写/第二事实源。
- Accepted Risks：无新增。集群、真实 Secret/远端属性、监控安装/告警接收、数据库/队列/Profile/既有进程的状态 **UNKNOWN / NOT_RUN**。静态资源无源码 reader，不等于真实资源没有 owner。
- Threat Model：误把带旧名字的共享 Secret 当孤儿；删除通用存储保护；对通用观测/workerpool/queue 作整包删除；将源码删除理解为 apply/prune/卸载授权；继续要求已 hard-cut 的14天兼容观察作为试用前置。只处理这些实际边界，不扩张 IAM、恢复或验收工具。

## 3. 真实消费者与装配证据

调查仅检查 tracked 源码、脚本、清单及文档；没有运行应用、constructor、SQL、kubectl/Helm/Docker、provider 或读取私密配置。精确名称与 import/引用核对覆盖 cmd/internal/scripts/tests/.github/deployments/config，并检查直接文档入口。

| 对象 | 基线实际消费者 / 证据 | 结论 |
| --- | --- | --- |
| `shein-listing-config` / `temu-listing-config` | 只有各自声明及 base Kustomization；原 workload/CLI 已退出。独立登录 Deployment 没有对它们的 volume/env 引用 | 旧专属源声明退休；其中通用配置类型本身不退休 |
| `temu-listing-secret` | 仅 base 示例与 prod ExternalSecret 的 metadata/target；仓库没有其余代码、workflow、脚本、workload reader | 与 TEMU 旧部署源一并退休；不删除任何实际 Secret/远端属性 |
| `shein-listing-secret` | `scripts/sync-shein-store-proxies.ps1` 默认 DatabaseSecret；`shein-proxy-sync/shein-proxy-sync.yaml` 的窄 RBAC resourceNames、DATABASE_SECRET 与数据库读取装配 | **保留四份源声明、名字/字段/权限及 prod 装配**，不能因旧名称退休 |
| 独立 SHEIN 登录 | `cmd/shein-login-worker` → `app/runtime/sheinloginworker` → 原 StoreAPI/AccountRepository/`sheinlogin/bootstrap`；原 workload 使用 `listingkit-workbench-secret`、healthz 和自身 app 标签 | Go/测试/镜像/配置/清单/workflow/原 Store 同步均零改；不消费旧刊登 ConfigMap/PodMonitor |
| `task-processor` Namespace | Amazon crawler、listingkit-workbench 及共享对象仍使用；各 base 有同名声明 | 不是旧业务专属资源；仅删除 TEMU 独立重复源文件，保留 SHEIN/当前其它 Namespace 声明；绝不删除真实 namespace |
| 两个旧 PodMonitor、两份 dashboard | selector 为 `shein-listing-role=service` 或 `app=shein-listing-store-883`，查询旧 `listing_tasks_*` / `shein_*` / `task_event_decoded_total`；core/metrics、TaskEvent 编解码与旧 workload 已退 | 专属监控源闭包退休；不影响 Amazon 指标、当前 app/monitoring、queue/workerpool/AI 计量 |
| 旧 PrometheusRule 的八条 alert | 五条 SheinListing* + legacy decode/scrape 两条专属；最后 `TaskEventV2PrometheusStorageAtRisk` 的表达式只检查 monitoring 中 Prometheus PVC 空间 | 七条 RETIRE；最后一条有效纯规则 **EXTRACT** 到监控 owner，不能整删文件时丢失保护 |
| rollout helper/test | `Get-RolloutBatches` 仅由 `rollout-shein-shard-statefulset.Tests.ps1` dot-source，原 rollout 执行脚本已退；无其它脚本/workflow consumer | 两文件一起退出，不为保留通用算法新增调用者 |
| `fix-missing-shein-tasks.sql` | 查询 `listing_kit_tasks`，注释中 UPDATE completed→pending，承诺旧 Temporal 自动重跑；当前生产代码无该表读写 | 整退脚本源，不查询或执行任何 SQL；当前 BusinessTask/Temporal 恢复不动 |
| 通用诊断工具 | `start-shein-local-rabbitmq-portforward.ps1` 是人工 Postgres/Redis/RabbitMQ 转发工具；`db-query.ps1` 调用 `tools/db-query`，支持显式表名 | 原能力保留；仅移除 db-query help 的三条旧任务表实例，不退休工具/通用连接能力 |

**架构问题**：旧执行 owner 已不存在，源部署/监控仍暗示可用；通用 PVC 告警又挂在旧 TaskEvent 规则和旧 service 标签下。解决方式是关闭完整旧入口并抽取那一条已有有效保护，而非保留空 workload、构造新监控系统或扩大运行清理权限。

## 4. Owner、持久化与副作用

- 本批不增加业务事实或表。旧任务脚本无继任 owner；`listing_kit_tasks` 物理表/数据的存在与归属 UNKNOWN，不接管/迁移/重放/清理。
- 当前 Store 事实、账号读取/同步、Redis/Profile/Cookie/attempt、DB 事务/幂等/恢复/UNKNOWN 保留原 owner；特别是 Store 代理同步及其窄 RBAC 完全不改，不改动任何 tenant/身份/平台副作用。
- SHEIN 共享 Secret 源继续由原 Store 代理同步装配消费；四文件的字段/远端映射及 namespace 原样。真实凭据与平台密码不读取、不展示、不验证、不轮换。TEMU 源退出只停止仓库提供旧装配；没有实际资源删除/ExternalSecret 同步或 provider mutation。
- 通用 Prometheus 存储保护属于现有 `deployments/kubernetes/monitoring`；按§5.3作三份最小 CRD/Kustomize 源抽取，包含同 namespace 的窄 AlertmanagerConfig。Prometheus/Grafana/Alertmanager/PVC/保留期、版本、存储类和当前 Amazon 监控 owner 不变；不宣称告警已安装或实际送达。
- 全部为仓库提案；不调用 apply、delete、prune、rollout、helm、port-forward、停止进程或操作旧/保留实例。运行清理/发布须另外明确授权，不能由本设计的删除条件推导。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 整退23个准确源文件

以下是同一完整闭包的删除清单；不得递归按 `shein`、`temu`、`monitoring` 或 `scripts` 目录删除：

```text
deployments/kubernetes/shein-listing/base/configmap.yaml
deployments/kubernetes/shein-listing/overlays/staging/kustomization.yaml
deployments/kubernetes/shein-listing/overlays/prod-single-store-181/kustomization.yaml
deployments/kubernetes/shein-listing/overlays/prod-single-store-976/kustomization.yaml
deployments/kubernetes/shein-listing/overlays/prod-store-shards-gray/kustomization.yaml
deployments/kubernetes/shein-listing/overlays/prod-store-shards-example/kustomization.yaml
deployments/kubernetes/temu-listing/base/configmap.yaml
deployments/kubernetes/temu-listing/base/kustomization.yaml
deployments/kubernetes/temu-listing/base/namespace.yaml
deployments/kubernetes/temu-listing/base/secret.example.yaml
deployments/kubernetes/temu-listing/overlays/prod/external-secret.yaml
deployments/kubernetes/temu-listing/overlays/prod/kustomization.yaml
deployments/kubernetes/temu-listing/overlays/staging/kustomization.yaml
deployments/kubernetes/monitoring/shein-listing/README.md
deployments/kubernetes/monitoring/shein-listing/kustomization.yaml
deployments/kubernetes/monitoring/shein-listing/podmonitor.yaml
deployments/kubernetes/monitoring/shein-listing/podmonitor-store-883.yaml
deployments/kubernetes/monitoring/shein-listing/grafana-dashboard-configmap.yaml
deployments/kubernetes/monitoring/shein-listing/grafana-dashboard-business-configmap.yaml
deployments/kubernetes/monitoring/shein-listing/prometheusrule.yaml
scripts/lib/shein-rollout-batches.ps1
scripts/rollout-shein-shard-statefulset.Tests.ps1
scripts/fix-missing-shein-tasks.sql
```

先完成§5.3抽取再删除旧 PrometheusRule。TEMU 仅保留§5.2的退休说明；SHEIN prod 不再是刊登部署入口，只装配明确保留的共享 Namespace/Secret。

### 5.2 Mixed 文件与直接尾部

| 准确文件 | 允许改动 | 必须保留 |
| --- | --- | --- |
| `shein-listing/base/kustomization.yaml`（均相对 deployments/kubernetes） | 仅删 `configmap.yaml` resource | 原 Namespace 引用及元数据；prod/kustomization.yaml 原样 |
| `shein-listing/README.md`、`temu-listing/README.md` | 写明源退休、准确保留资源、当前入口与无运行操作 | 不保留旧 apply/分片/兼容承诺；不恢复旧链 |
| SHEIN overlays 的 `prod-auto-shard-statefulset/README.md`、`prod-store-auto-shard/README.md`、`prod-store-shards-example/README.md` | 校准剩余源与共享 Secret 边界，引用本文 | 历史只作历史；example Secret 模板不变，不新增部署步骤 |
| `deployments/kubernetes/readme.md` | 退出 SHEIN/TEMU 部署顺序/apply 示例及旧三平台任务队列说明，指向当前入口与本设计 | Amazon/身份/基础设施等其余独立内容不扩改，不执行其中命令 |
| `docs/database-connection-optimization.md` | 顶部标为历史方案且非当前runbook；原实施/数量/效果只留历史。退出“直接应用（推荐）”、旧apply/rollout/logs/exec验证等现行操作建议，指向当前入口及本文 | 不更改当前数据库/连接池实现或配置，不把历史优化/部署记作本轮实测；不执行任何DB/集群命令 |
| `deployments/kubernetes/monitoring/README.md` | 退出旧监控 apply/覆盖和当前态14天兼容窗口要求；说明§5.3当前共享保护源及已授权运维才可应用 | 原 Chart/版本/容量/存储、Amazon、适配器/Ingress 的独立授权边界；不把窗口历史记成PASS |
| `monitoring/alertmanager-wecom/alertmanagerconfig.yaml`（相对 deployments/kubernetes） | 仅将 service matcher 值 `amazon-crawler-api|shein-listing` 收窄为 `amazon-crawler-api`，退出旧分支 | matchType、task-processor namespace、groupBy/interval、receiver/url/timeout/Secret/权限及其它结构原样；通用存储告警用§5.3独立窄配置，不能伪称只改service就可跨namespace匹配 |
| `monitoring/alertmanager-wecom/README.md` | 只同步原task-processor配置的Amazon范围及monitoring中唯一共享存储规则的窄装配说明 | 原操作/凭据保护，不增加实际发送测试 |
| `scripts/db-query.ps1` | 只移除 help 中三条 `listing_kit_tasks` 示例，改为显式当前 owner 表的中性用法 | 参数、连接/执行与 tools/db-query 原样，不执行它 |
| `tests/listingkit_backend_retirement_test.go` | 按§6.1校准精确输入列表与原 guard | 所有仍有效的配置/安全/退休断言，不能skip整测试或降低 gate |
| `docs/architecture/README.md`；`docs/refactoring/legacy-register.md`、`module-target-mapping.md`、`code-health-decisions.md` | 只加当前窄 superseded 注记/索引 | 历史表/阶段计数/原证据；通用诊断工具的保留决定不变 |

### 5.3 先抽取一条已有共享保护

新增且仅新增运行源：`deployments/kubernetes/monitoring/prometheus-storage/prometheusrule.yaml`、同目录 `alertmanagerconfig.yaml` 和 `kustomization.yaml`。复用当前 Prometheus Operator PrometheusRule `monitoring.coreos.com/v1`、AlertmanagerConfig `monitoring.coreos.com/v1alpha1` / Kustomize；三者明确装配在 `monitoring` namespace，Rule资源及group名为 `prometheus-storage`。不新建程序、SDK、监控平台或安装流程。

只把旧 `TaskEventV2PrometheusStorageAtRisk` 的**原表达式**（monitoring 的 available/capacity `< 0.20`，与 `prometheus-.*` PVC info 以 namespace/persistentvolumeclaim join）、`for: 15m`、`namespace: monitoring`、`severity: warning` 抽出，保持这些值与条件完全相同；alert 名为 `PrometheusStorageAtRisk`、service 为 `monitoring`，summary/description/action 只描述通用存储风险，删除“TaskEvent迁移证据/批准删除decoder”的过期依赖。不引入新阈值、重试或状态协议。

原规则标签为 `namespace=monitoring`，原 AlertmanagerConfig 位于 `task-processor`；默认 `OnNamespace` 强制匹配两者一致，因此仅改service不能正确保留路由。最小修复：新增 `AlertmanagerConfig/prometheus-storage-wecom` 位于 `monitoring`，顶层route只匹配 **`service=monitoring` 与 `alertname=PrometheusStorageAtRisk`**（两者matchType `=`），接受隐式 `namespace=monitoring` 约束；groupBy、groupWait/groupInterval/repeatInterval、receiver及webhook/url/sendResolved/maxAlerts/timeout值完全复用原WeCom配置。Kustomization仅引用本规则和该窄配置。原task-processor配置只保留Amazon service，不混入monitoring；不全局禁用namespace enforcement，不扩其它服务或权限。

选择依据：[Operator API 的 OnNamespace 合同](https://prometheus-operator.dev/docs/api-reference/api/#alertmanagerconfigmatcherstrategytype)；仓库固定Chart [88.3.0 values](https://github.com/prometheus-community/helm-charts/blob/kube-prometheus-stack-88.3.0/charts/kube-prometheus-stack/values.yaml) 与 [Alertmanager模板](https://github.com/prometheus-community/helm-charts/blob/kube-prometheus-stack-88.3.0/charts/kube-prometheus-stack/templates/alertmanager/alertmanager.yaml)。静态核对：该版本默认 `alertmanagerConfigSelector: {}`、`alertmanagerConfigNamespaceSelector: {}`，模板输出两selector，默认matcher strategy为OnNamespace；仓库values未覆盖这些选择，新增同namespace配置无需改stack values、全局selector或RBAC。实施以相同固定版本的本地模板/静态对照及YAML断言验证选择关系，配置只匹配上述唯一规则，不构建路由模拟平台。真实安装/发送/收信仍NOT_RUN；不能用它掩盖静态错配，也不得为取得PASS访问集群或发企业微信。Amazon 原 alert/PodMonitor/dashboard、通用 stack values、WeCom 程序/Deployment/Service/Secret 与 Grafana Ingress 均零改。

### 5.4 严格保留清单

- `shein-listing/base/namespace.yaml`、`base/secret.example.yaml`、`overlays/prod/secret.example.yaml`、`overlays/prod/external-secret.yaml`、`overlays/prod/kustomization.yaml`、`overlays/prod-store-shards-example/secret.example.yaml`：原文零改；其中四份 Secret 源不得随旧目录递归退出。
- `shein-proxy-sync/**`、`scripts/sync-shein-store-proxies.ps1`、独立 SHEIN 登录 worker 全链、`listingkit-workbench/**`、Docker/Compose/其它 workload、全部 workflow 原样；不改变 Store/schema/#692 或当前发布身份准入。
- `scripts/start-shein-local-rabbitmq-portforward.ps1`、`tools/db-query/**`、通用 diagnostics、middleware/队列/Redis/Temporal/身份/存储及 Amazon 采集/监控零改；没有为旧规则继续运行其消费者的要求。
- 既有规则/平台/DTO/共享 Go 类型和§1保留能力零改，无 EXTRACT Go helper、兼容壳、no-op adapter、新 owner/队列/表/迁移。

```text
Legacy decision: RETIRE + selective EXTRACT
Reusable behavior: 仅原 Prometheus PVC 低空间规则；既有通用工具与共享声明直接保持
Current owner: deployments/kubernetes/monitoring；Store/登录及其它当前 owner 不变
Cutover/deletion condition: 用户确认本文，精确共享规则抽取/保留图验证完成，原 guard/必需CI通过与最终独立检查；仅源码，不要求或授权运行清理
```

## 6. 验证与验收证据

设计交付：独立检查须看真实源码消费者、八条 alert 的拆分、共享 Secret/namespace/登录清单、§5准确矩阵及§7停止点；不能只采信无引用搜索。准确设计 HEAD 与 findings 分类/结论写 PR，不把 CI 绿色当设计批准。设计文档/索引检查即可，不运行 app/构造器/操作脚本或补正式测试。

确认后的同批实施仅作适用必要验证：原 guard 的定点 RED→最小实施→GREEN；Kustomize 使用本地 render/build（若现有工具可用，无集群）验证保留的 SHEIN base/prod 与新共享规则/窄路由，不 render 已退休入口；YAML 对照原 alert 表达式/for/保留标签及四 Secret/其它保留文件原文，确认原matcher只退出旧分支、新配置同namespace且只匹配唯一存储规则、receiver设置保持；原依赖/架构/商业检查与原 CI。没有 Go 业务改动，不为了退休源重复运行采集/provider/运行验收或新建工具。原 CI 自动执行的验证如实记录。

### 6.1 原 guard / CI 的准确处置

- `TestRetiredTaskAndListingRuntimeConfigClosure` 的配置输入表只退出两个已删除 ConfigMap 行；保留三个 config YAML、两个 Amazon lite/heavy ConfigMap 及其全部旧键缺席/Alibaba1688保留断言。Secret 输入表只退出 TEMU base/prod 两行；保留 Workbench 与四 SHEIN Secret 输入的完整旧 RabbitMQ env 缺席断言。
- 在同一既有退休测试文件补§5.1准确缺席清单；校验保留 SHEIN base 不再引用 configmap、prod仍引用base/external-secret、四 Secret/namespace 与 Store代理/登录引用不丢失。校验新 PrometheusRule 只有一条有效规则、原条件/for/namespace/severity保持；旧七条及 PodMonitor/dashboard 缺席，当前 Amazon source 保留；原 Alertmanager matcher 只保留Amazon，其余原路由不改。新窄配置namespace等于规则namespace标签、两个精确matcher等于规则service/alertname、receiver值与原配置一致、Kustomization覆盖两资源；固定Chart默认选择合同支持发现该对象，不变更全局matcher strategy。复用 YAML 解析，不新建框架或泛化扫描工具。
- 原有 CLI/TaskEvent/control-plane/ownership/config 等缺席断言全部保持。Pester 仅退出那个无运行消费者 helper 的专属测试，不误删其它 PowerShell 测试；通用工具不因无自动调用而删除。
- `.github/workflows` **全部零改**（含两套 CI/Required Gate、安全、发布和独立workerworkflow）；依赖、inventory current/历史行、计数、ceilings、lint/depguard 配置和保护规则零改。这里没有 Go import edge 删除，不需要inventory校准。若原CI暴露矩阵外真实当前Must问题，先分类并报告，不自动放松检查/扩范围。
- 文档仅校准§5.2当前入口与索引，覆盖数据库优化runbook的推荐apply/rollout/验证尾部，并将其标为历史；同根因现行入口检索没有其它非历史运行文档漏项。历史设计、归档方案/观察窗口/证据不重写为当前PASS。不按裸字符串清空所有旧名称，仍有效的 Secret 和历史/guard引用合法。新增设计必须在原 `docs/architecture/README.md` 显式索引，不绕过原 `TestArchitectureReadmeCoversEveryArchitectureDocument`。

## 7. 执行与停止点

1. 设计阶段只调查与设计，已在两Issue/主要PR记录Design Basis/使用者/范围/非目标/单Writer；生产、测试、配置、脚本、workflow当时零改。准确设计0871cb591的独立检查完成，用户确认及Ready准入读回之后才开始本批正式实施。
2. 一次原 GitHub 独立设计检查；findings 先分类，最多两轮正常设计检查。未变代码/证据复用；不默认创建多 Reviewer，不因理论 hardening 重开全局架构。只有达到 IMPLEMENTATION_READY 且用户明确确认§5/§6/§6.1/§7、Issue Ready读回后，原 Writer 才实施。
3. 同一 PR 整批实施：先抽取唯一共享规则与定点 guard，再退出全部专属源及尾部；shared Secret/当前能力零改。稳定候选前核对实际 main，仅处理真实组合风险，不持续追 main、不按内部模块拆 PR。完整原 CI 留稳定候选，复核修复只看相关增量。
4. 最终交付检查读实际完整 diff、保留能力及原证据；不是用户验收。发现当前 consumer/持久化 owner 或外部可观察义务要求本文外保留/迁移/兼容时停止相关动作，报告具体证据；没有自行批准 Exception 或真实操作权限。
5. 合并另需用户授权准确最终 HEAD，使用 expected-head 正常合并，不绕过保护；合并后验证实际 merge SHA 的 push/main CI。设计/PR CI、main CI、运行/投递/provider/用户验收各自独立，后者目前 NOT_RUN。#29/#301 OPEN，部署/关单/实例/数据/凭据操作不在授权中，不增加内部试用前置。
