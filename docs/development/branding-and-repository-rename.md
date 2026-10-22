# 硕米智能引擎 / Shuomi AI Commerce 品牌与仓库更名清单

依据 [#884](https://github.com/qq550723504/task-processor/issues/884) 的 2026-10-22 产品决定。
本文件交付 Phase 0/1 盘点与文档品牌落地，并记录后续更名前核实结果；不是实际更名或部署指令。
Design Basis: N/A，纯调查与文档，不改变产品、领域合同或运行语义。
UI / IA 继续使用[最终 Figma Authority](../product/final-ui-ia-authority.md)。

## 名称与本批结果

| 对象 | 当前决定 / 状态 |
| --- | --- |
| 中文产品名 | 硕米智能引擎 |
| 英文产品名 | Shuomi AI Commerce |
| 品类 / 定位 | AI Commerce Agent Platform，AI 驱动的跨境电商智能体平台 |
| 当前仓库 | `qq550723504/task-processor`，尚未更名 |
| 计划仓库 | `qq550723504/shuomi-ai-commerce`，等待切换条件与单独授权 |
| 展示层 | 本批统一 README、产品战略、前端说明、目录地图和本地账户构建/运行说明 |
| 仓库 About 建议文案 | `硕米智能引擎 / Shuomi AI Commerce — AI Commerce Agent Platform`；仅准备，尚未修改远端设置 |

`task-processor` 作为工程旧称、实际模块/路径/资源标识保留；ListingKit 仍按既有
EXTRACT / RETIRE 边界处理，不能因品牌变化恢复旧产品或误删有效前端代码。
当前启动仍见 [README](../../README.md#当前正式运行入口)、
[Console](../../web/listingkit-ui/README.md) 和[账户环境](../../deployments/docker/account-compose/README.md)。

## 盘点基线与复查方法

源码基线：`124a4de006326bf5930b1a3b4533254dd3591968`，2026-10-22。
以下数字只描述该提交中已跟踪文本文件；不读取私密配置、用户数据、忽略文件或运行实例。
文档改动后的命中数会变化，不将这些数字作为清零指标。

| 搜索范围 | 命中文件数 | 说明 |
| --- | ---: | --- |
| `task-processor` | 3013 | internal 2341、docs 511、cmd 51、deployments 33、tests 29、scripts 16、hack 11、其余目录/根文件 21 |
| `github.com/qq550723504/task-processor` | 151 | docs 142，其余 README、模板、部署/扩展说明、源码/测试共 9 |
| `"task-processor/` | 2453 | 含 Go import、边界规则、测试和文档；不是纯生产 import 数量 |
| `硕米智能引擎` 或 `Shuomi AI Commerce` | 30 | 文档、当前前端、邮件展示等；不意味着这些能力已验收 |

在仓库根目录可复查完整路径与行号；本批不增加扫描器或治理工具：

```powershell
git grep -n -I 'task-processor' 124a4de006326bf5930b1a3b4533254dd3591968 -- .
git grep -n -I 'qq550723504/task-processor' 124a4de006326bf5930b1a3b4533254dd3591968 -- .
git ls-files '*go.mod' '.github/workflows/*' '*Dockerfile*'
git worktree list
git remote -v
```

## 引用分类与 Phase 2 处理决定

按引用语义分类，同一文件可能同时含真实路径、产品名称与历史证据；不按整文件批量替换。
下表覆盖命中目录与需要额外核对的构建/配置入口。owner 表示现有职责，非新派工。

| 分类 | 具体引用 / 消费者 | 决定、影响与 owner |
| --- | --- | --- |
| 对外标识 | README、`docs/product/ai-commerce-agent-platform-strategy.md`、`web/listingkit-ui/README.md`、`docs/development/repository-structure.md`、账户 Compose README | 本批统一品牌；真实命令、路径和能力状态不改。owner：#884 Writer |
| 对外标识 | README CI badge、`.github/ISSUE_TEMPLATE/execution-task.md`、`.github/pull_request_template.md` 中仓库地址 | 保留当前有效 URL；更名后更新当前入口，历史 PR/Issue/SHA 链接原样保留。owner：仓库维护者 |
| 对外标识 | `web/listingkit-ui/src/app/layout.tsx`、marketing/legal 组件、现有邮件展示 | 当前中文品牌已为硕米智能引擎；本批不改 UI。英文展示或页面变更须先核对 Figma。owner：对应 UI/通知 owner |
| 源码引用 | 根 `go.mod` 的 `module task-processor`；cmd/internal/hack 下 import | **保留**。模块路径是独立源码标识，不由 GitHub 仓库名自动派生。更名不要求改 module，不新增 wrapper/双路径解析。owner：Go 构建/领域 owner |
| 源码引用 | `hack/debug/go.mod` 的 `task-processor/debugtools`、`require task-processor` 和本地 `replace`；`tools/go.mod`、`tmp/go.mod` | 保留所有模块及本地解析关系。将来若明确决定更改 module，必须原子覆盖嵌套模块与仓库外消费者，另行评估合同准入 |
| 源码引用 | `.golangci.yml` 的 depguard、tests 中 import/架构断言、scripts 中 AST/依赖解析与 fixture | 保留准确旧 module 字符串；禁止为名称统一削弱 guard。owner：现有边界/测试维护者 |
| 当前生产依据合同 | `internal/knowledge/official/catalog.go`、`web/listingkit-ui/src/lib/api/official-knowledge.ts`、原BFF/页面及相邻测试 | 用户新决定下，V2已退出sourceRoot/URL字段/双端GitHub白名单，以资料名称/维护方/准确版本表达依据；旧原子保留方案仅历史。owner：官方知识内容/读取owner |
| 源码引用 | `extensions/1688-capture/package.json` 与 lock 的 `@task-processor/1688-capture`；前端 `listingkit-ui` 包名、目录、lock | 保留工程标识，无发布命名空间迁移。外部 npm 发布/依赖方 UNKNOWN。owner：扩展/前端维护者 |
| 构建引用 | Makefile、Dockerfile、release/build 脚本与 config | 实际入口与路径保持；根 config YAML/Makefile 没有命中 `task-processor`，不能据此断言所有运行配置无关。owner：构建/运行 owner |
| CI 标识 | `.github/workflows/ci.yml` concurrency 的 `task-processor-ci-*` 与 classifier 测试 | 保留，避免切换时产生另一组并发任务；通知 URL/仓库名消费 `github.repository`，改名后仍需实测。owner：CI 维护者 |
| 静态安全元数据 | `release-policy.yaml` 的 `releaseAuthority.repository`、`policy/listingkit-release-authority/{release_authority,negative_fixtures}.rego` 与 `tests/listingkit_release_authority_policy_test.go` | **切换时同步项，不是当前发布故障证据**：四处旧仓库字符串只约束保留的静态策略一致性；旧 API/UI publisher 已固定失败退役，旧 OIDC/RBAC 字段已退出。准确调用路径与后续最小同步范围见下方核实增量。owner：原静态安全检查维护者 |
| 运行期资源 | SHEIN worker workflow、`scripts/build-push-shein-runtime-base.ps1`、Kustomize 的 `xuwei190/task-processor-*` 镜像与 tag | 保留 Docker Hub 命名与 release tag；GitHub 改名不等于镜像仓库迁移。制品消费者/部署现状 UNKNOWN。owner：镜像/运行 owner |
| 运行期资源 | Kubernetes `task-processor` namespace、`task-processor/crawler-tier`、SecretStore 名、MinIO DNS、monitoring receiver | 保留，禁止机械替换。命名影响选择器、DNS、RBAC 和现存资源；本批不访问集群。owner：各运行 owner |
| 运行期资源 | account/acquisition/referrals Compose 示例、chain1 Compose name、`.env.example`、Temporal dev 容器名与相邻脚本测试 | 保留项目前缀、容器/卷标识和原 stop/restart 路径。更换项目名会选择不同资源；本批不启动、停止或销毁环境。owner：原环境 owner |
| 历史证据 | docs 中日期/SHA/验收、旧设计、旧 ListingKit、PR/Issue/action 链接、D:/ 或 C:/ 工作区记录；hack/tmp 调试材料 | 原样保留，不将历史模块名或路径包装成新名称，也不推断历史运行结果已解决。当前有效指南只补品牌说明。owner：原记录/任务 owner |

本批结论：**无需为品牌变更修改 Go module/import；运行资源保留；仓库地址切换延后。**
若以后需要修改公共模块合同、解析、安全身份或运行资源 identity，先按 [AGENTS](../../AGENTS.md)
评估 Design Basis，未准入不修改正式路径；本清单不是该部分 IMPLEMENTATION_READY。

## 外部消费者与协作影响

以下为 2026-10-22 只读快照，切换前必须刷新；UNKNOWN 表示本次没有确认，不能按不存在处理。

| 对象 | 已观察事实 | 缺口 / 处理 |
| --- | --- | --- |
| 仓库/权限 | GitHub API 返回旧名、默认分支 main、description=null、has_pages=false；当前 CLI 身份具备 admin | 权限不是更名授权。About 文案仅准备 |
| 目标名称 | GET `repos/qq550723504/shuomi-ai-commerce` 返回 404 | 只证明当前未读到目标仓库；名称确实可用仍需 owner 在切换窗口确认，不能预先承诺 |
| 活跃 PR | 开工时开放 PR #692（独立 Store schema 文档任务），无 #884 PR/接单；另有其他活跃开发会话 | 不接管/合并其他任务；切换前核对各 owner 的推送、复核和 CI 是否会受影响 |
| 当前 CI | 基线 main 的 push run `39739004201` 当时 in_progress | 不在运行中更名，不把该次运行当本批 PR CI 或完成证据 |
| reusable workflow/action | 本仓库 5 个 workflow 未发现 `workflow_call` 或 `uses: qq550723504/task-processor...`；无实际 action.yml/action.yaml（负例 fixture 除外） | 仓库外私有消费者 UNKNOWN；需 owner 确认，不能只凭本仓库 grep 清零 |
| Webhook | API 可见 1 个 active hook，ID `631752654`，events 为 workflow_job/workflow_run | 没读取/展示 URL、secret；接收方是否绑定仓库字符串、改名后投递是否成功 UNKNOWN；由集成 owner 核对 |
| Apps / OIDC / runner / 外部部署 | 仓库源码只能说明已有 CI/release 接线 | 安装 App、第三方白名单、runner、OIDC subject、外部部署脚本的实际配置 UNKNOWN；不读 Secrets，不自行 mutation |
| Go 外部消费者 | 已确认本仓库本地 replace 与 import；没有仓库外依赖证明 | UNKNOWN；保留 module。若未来改 module 必须先明确消费者 |
| 容器镜像 | 源码固定 Docker Hub `xuwei190/task-processor-*`，不是从 GitHub 仓库名自动派生 | 仓库/tag 的真实存在与拉取结果 UNKNOWN；本次不登录 registry 或推送 |
| git/worktree | 共享主 checkout origin 仍是 `https://github.com/qq550723504/task-processor.git`；开工时本机已有 12 个 checkout，本批另建隔离工作区 | 同一 Git common dir 的 worktree 共享 remote 设置；切换后由协调 owner 更新一次并逐工作区核对，不重命名/移动活跃目录。其他机器/CI clone UNKNOWN |

## 实际更名前的阻碍与切换检查清单

这不是新增业务验收平台；仅列 #884 仓库切换必需的现有入口核对，不阻塞本批文档 PR。

1. **授权与窗口**：取得对实际 GitHub 更名的单独明确授权；确认目标可用、切换负责人及回退负责人。
   刷新开放 PR/Actions，和活跃 owner 确认窗口；不得从 admin 权限推定许可。
2. **静态策略同步**：四处仓库字符串按下方最小范围在实际更名的同一切换窗口内同步，原静态检查、
   负例及 namespace 保留；不新增 publisher 或重新启用旧发布。当前无需为了此项重新设计授权体系。
   若查到实际 OIDC 信任消费者，先单独明确该消费者的精确 subject 与适用准入，不能混为静态文案。
3. **外部集成**：由 owner 解决上表会影响当前消费者的 UNKNOWN，尤其 webhook、App/OIDC、
   外部 `uses:`/clone 与发布引用。GitHub 官方说明 action 调用不会因仓库更名自动重定向；
   不用普通链接重定向替代集成验证。未确认影响时保持待切换。
4. **实际切换**：上述条件满足后，由获授权 owner 更名并记录新 URL、时间及仓库 ID；
   更新当前 README badge/模板/活动开发文档 URL、准确策略引用及各 clone 的 remote。
   保留历史工程证据及 Go 模块、目录、镜像、资源名；官方知识V2已不依赖GitHub来源URL，
   不为更名重新加入URL/白名单。不在旧名称下新建仓库覆盖重定向。
5. **切换后核对**：新 URL clone/fetch 成功；由获授权 owner 用现有正常提交验证 push/Actions；
   PR/Issue 保留、必需检查/原策略负例保持；原 webhook 新事件投递和外部消费者由各 owner 确认。
   标签触发的部署仍需独立授权，不为测试更名发布部署 tag。
6. **记录边界**：PR CI、实际 merge-SHA push/main CI、集成状态分别记录于 #884/对应 PR；
   未执行项保持 NOT_RUN/UNKNOWN。不据此宣称部署或产品验收通过。

GitHub 行为依据：[Renaming a repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/renaming-a-repository)
（2026-10-22 查阅）。普通链接/Git 重定向不覆盖 action 调用和所有外部消费者；项目 Pages URL
也有独立影响，当前 API 快照未启用 Pages。切换时重新查阅，不依赖永久重定向。

## 回退

- **本批展示文档**：正常 revert 本批文档提交，不回滚其他业务 PR；未动运行资源，无数据回退步骤。
- **准备阶段**：任一切换条件不满足就保留旧仓库/remote，记录具体未满足项，不提前改 URL 或发布身份。
- **已执行更名后**：停止进一步切换/发布；由获授权 owner 核实旧名仍可用，再取得回退授权，
  将同一仓库改回 `task-processor`（不删除/重建仓库）。按实际已执行清单反向恢复 remote、
  当前链接和精确安全身份；不撤销无关业务提交、不改镜像/namespace/卷。
- 更名回退不保证已失败 CI/webhook 或 OIDC subject 自动恢复。GitHub 当前说明更名会迁移到含固定 ID
  的 subject 格式；不能承诺改回旧仓库名就回到旧 subject。重新读取 OIDC 配置，由实际信任 owner
  核对精确格式；不自动添加双 subject、通配符或回退身份。重新核对 clone/fetch/push、准确 HEAD 的 CI、
  hook 响应正文及已确认的外部消费者；未验证保持 UNKNOWN，不能通过关闭安全检查完成回退。

## 更名前核实增量（2026-10-22）

本节以 #886 合并提交 `3b50940f3b4ee9a5bd1f4491712ca43a355c0c33` 为源码基线，
补充同日仓库 API、设置页面及官方文档的只读证据，取代上方初次快照中已核实项的 UNKNOWN。
滚动 HEAD/CI/合并记录仍只维护在 #884/对应 PR；本节没有实际更名、配置修改、令牌签发或消息重投。

### 原发布策略疑点：已明确最小范围

原消费者链为：机器策略 YAML + 当前配置/已退役 workflow 文件 → 原 verifier 的固定输入
→ Rego 静态一致性检查、Go 合同测试 → 既有 CI。`policy.repository` 只与固定字符串比较，
不读取当前 GitHub 仓库上下文，不请求令牌，不安装任何发布身份。

依据 [#850 已确认退休边界](../architecture/legacy-kubernetes-release-principal-retirement-boundary.md)
和[部署参考当前顶部说明](../../deployments/kubernetes/listingkit-workbench/README.md)，
旧 OIDC/RBAC principal 已从源码退出；`listingkit-deploy.yml` / `listingkit-ui-deploy.yml`
只有 `permissions: {}` 与固定 `exit 1`。因此旧仓库字符串本身不能证明更名会打断当前发布。

| 项目 | Finding 分类 / 处置 |
| --- | --- |
| 将静态字符串认定为当前 publisher 的架构 BLOCKER | NOT_APPLICABLE；不存在该源码消费者，不为它新建身份、发布或兼容机制 |
| #884 当前准确仓库元数据要求 | 实际切换时同步四处值，保持原输入/规则/拒绝/负例，随后跑原检查；无需提前把旧仓库配置写成新仓库 |
| 当前运行环境是否仍有旧信任 | owner 在本会话明确确认没有当前使用中的外部 OIDC/云/Kubernetes 信任、Vercel 项目或离线部署名称依赖；不再作为本次更名前置。未做物理撤权检查，不将该声明写成历史资源已清零，不恢复旧 publisher |

四处同步值均为 `qq550723504/task-processor` → `qq550723504/shuomi-ai-commerce`：

1. `deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml` 的 `repository`。
2. `policy/listingkit-release-authority/release_authority.rego` 的精确仓库断言。
3. `policy/listingkit-release-authority/negative_fixtures.rego` 的精确仓库断言。
4. `tests/listingkit_release_authority_policy_test.go` 的精确仓库断言。

保持 namespace `task-processor`、原 verifier 输入、固定失败发布、canary/UI 隔离和负例拒绝。
适用验证为原 `TestListingKitReleaseAuthorityMachinePolicyOwnsAllSecurityInputs`、原策略 verifier
及既有必需 CI。仅上述元数据等值替换且不改变消费者/运行语义时，可按 N/A 机械调整准入；
这不授权真实更名，也不覆盖新增 OIDC/授权合同。当前没有改动这四个文件。

### 当前官方知识出处合同：由用户新决定替代

**最新决定（2026-10-22）**：用户要求「官方知识库不要使用github地址」。下文原子保留方案
仅为此前准备/评审证据，已被[产品内依据V2设计](../architecture/official-knowledge-provenance-v2.md)
替代，不再指导实际切换。V2将从API/前后端/正文中退出GitHub来源URL，采用来源名称、维护方与
准确版本；不只隐藏链接。V2已完成独立设计准入并按冻结合同实现，准确验证状态记录于PR；
未更新当前实例，不把源码交付写成运行已切换。

四处静态值不是全部生产引用。独立最终检查指出遗漏的当前官方知识来源组，已按
IMPLEMENTATION_TEST（当前盘点文档Must、切换正确性）收敛到如下明确处理，不做单边替换。

当前消费链：`NewEmbeddedCatalog` 生成固定版本文章 → `NewCatalog` 校验来源 →
`internal/app/httpapi/current_official_knowledge.go` 装配 → Workbench正常应用组合 →
official read API/BFF → 前端 `officialArticleSchema` 校验 → 官方知识正文/出处页面。
后端和前端都只接受 `https://github.com/qq550723504/task-processor/blob/`；
仅改 sourceRoot 会令 catalog 构建失败并影响 Workbench 装配，仅改服务端也会被前端/BFF拒绝。

依据 [Official Knowledge V1](../architecture/official-knowledge-v1.md) 与
[官方知识维护合同](../engineering/official-knowledge-handoff.md#内容维护与限制)，已发布准确版本
不原地修改。本次决定为**保留** `ai-commerce-guide` revision `1` 的正文、digest、日期与
固定 `60d432aa8b9391491278912e4268950ba65b58f0` SHA 出处，以及匹配该出处的原严格白名单。
出处是历史版本依据，不是新产品品牌文案；与本 Issue 保留历史引用的要求一致。
这不是旧业务owner兼容层，也不增加fallback、双读或任意URL许可。

| 同组路径 | 本次更名处理 / 验证 |
| --- | --- |
| `internal/knowledge/official/catalog.go`：sourceRoot + NewCatalog来源校验 | 两者原子保留；不单独替换任一字符串，不放宽HTTPS/host/user/query/路径约束 |
| `web/listingkit-ui/src/lib/api/official-knowledge.ts`：sourceURL/officialArticleSchema | 与后端相同旧prefix保留；BFF共用schema，无第二套许可 |
| `internal/knowledge/official/catalog_test.go` | 保留准确版本/不可变副本、已嵌入guide及恶意host/其他repo拒绝测试 |
| `web/listingkit-ui/src/lib/server/official-knowledge-proxy.test.ts` | 保留来源schema、准确版本和安全转发的相邻fixture/负例 |
| `web/listingkit-ui/src/components/workbench/knowledge/official-knowledge-page.test.tsx` | 保留固定出处呈现与权限/组织切换行为的fixture |
| `internal/app/httpapi/current_official_knowledge.go` / `current_application.go` / 原同组装配测试 | 消费者映射明确，不改装配；沿既有检查确认目录构建与文章读取，不启动或更新保留实例 |

切换检查清单明确包含：上述代码/fixture保持成组一致、原 catalog/官方读取/BFF/页面测试可复用，
更名后核对旧固定SHA出处仍解析到同一仓库/内容；未实际更名时该步保持NOT_RUN。
不得复用旧仓库名破坏重定向。若以后确需新canonical出处，先由原内容owner按正常版本发布
规则处理，并同步来源生成、后端校验、前端/BFF与上述测试；不能以更名为由改写已发布v1，
也不在本次准备PR里预加新prefix或兼容规则。

### 集成事实与剩余确认

| 对象 | 本次结果 | 对实际切换的影响 / owner 下一步 |
| --- | --- | --- |
| 稳定仓库 ID | `1108239623`；owner ID `6007227`，当前 public/main | 切换后核对仍为同一仓库，不删除重建。记录新 URL，保留 Issue/PR/历史 |
| 目标名称 | 公开仓库列表未见目标名，目标 GET 404；General 页面只输入过未提交名称草稿，未返回明确可用性提示，随后已丢弃草稿 | 未发现占用，不是预留承诺；获授权实际执行时由 GitHub 最终校验，不为取得可用性提示提前点击 Rename |
| 分支保护 | main strict=true、enforce_admins=true；唯一 required context 为 GitHub Actions 的 `Required CI Gate`（app_id=15368）；rulesets=[] | 原保护保持；当前没有由旧仓库字符串命名的必需 context，不改规则来方便更名 |
| runner | 仓库级 self-hosted runner 数量 0；当前 workflow 使用 github-hosted | 无本仓库 self-hosted 重注册事项；不推断其他机器无 clone/集成 |
| Environment / Secrets 元数据 | production、development-admission-publisher 两环境；后者保留 required reviewer。仓库四个 Secret 名为 DOCKERHUB_TOKEN、DOCKERHUB_USERNAME、KUBE_CONFIG、WECOM_WEBHOOK_URL；后者环境另有 DEVELOPMENT_ADMISSION_APP_PRIVATE_KEY；仓库及两环境变量均空，production 环境 Secret 为空 | 只读名字/保护，未读取值；这些环境/凭据没有因为更名而需要复制、轮换或删除的当前证据，保留原配置。App 私钥存在不证明 publisher 在当前执行 |
| 当前源码 OIDC | API：use_default=true、use_immutable_subject=false、sub_claim_prefix=`repo:qq550723504/task-processor`；5 个 workflow 无 `id-token` 或 `self-hosted` | 当前源码无令牌签发消费者。不能将历史 Kubernetes subject 当作当前接线；实际外部信任仍由 owner 确认 |
| 更名的 OIDC 影响 | GitHub 当前文档说明 2026-07-15 后的 rename/transfer 也迁移到 immutable subject 格式；owner 已确认无当前外部消费者 | 已排除当前信任切换前置，仍保留事实/回退说明。切换后重读配置，不签发 token、修改 trust 或自动开启 immutable 配置；以后新消费者按新准入使用实际 subject |
| 直接 Webhook | 唯一仓库 hook `631752654`，事件 workflow_job/workflow_run，receiver host 为 qyapi.weixin.qq.com。近期已有投递 HTTP 200，但响应正文 errcode=40008（消息格式错误） | 已确认的既有通知故障，不能把 200 写成业务成功。与新旧仓库名无关，分类 BACKLOG；不冒充已修复或自动视为可关闭。owner 可另行决定停用该直连或修复；本批不改 hook、不重投消息、不新建适配服务 |
| 已有 CI 通知 | ci.yml 构造企业微信 markdown 并检查 errcode=0；仓库/链接来自 github.repository。job 为 continue-on-error，日志可能因缺少 Secret 而跳过 | 复用已有能力；通知 job 绿灯不证明实际送达。切换后分别核对原 hook 响应正文、CI 通知日志，不把它们与 Required CI Gate 混为一谈 |
| GitHub Apps | owner 完成 GitHub 二次认证后只读核对8个详细配置：Governance 为 Only select repositories，准确选中 DOM 对象 selected-repository-1108239623；其余7个均为 All repositories | 安装授权范围已核实，无重新安装/扩大权限事项。Governance/Cursor/Vercel 的待批准权限升级保持未批准，不是更名条件；不将安装范围事实当成第三方服务端已通过更名实测 |
| 自有 Governance App | app_id=4799675，installation_id=158369358；Webhook Active=0、URL空，Setup/Callback URL空；Homepage仍为旧仓库URL | 无当前回调地址切换；Homepage仅展示链接，可在更名窗口同步。保留原仓库ID授权与权限，不新增publisher或处理消息 |
| 外部消费者 owner 声明 | 本会话用户明确回复「没有此类当前外部依赖」，对应问题明确覆盖当前 Vercel 项目、云/Kubernetes OIDC 信任或离线部署脚本对旧仓库地址/subject的依赖 | 该具体事实缺口已收敛，不继续要求为不存在的当前消费者建设或测试通用机制；历史资源/第三方底层缓存未执行验证仍不宣称清零 |
| 外部源码消费者 | owner 范围 GitHub code search：旧 full_name 在 .github/workflows 中0条，go.mod 定向搜索0条；本仓库无导出 action 或 workflow_call | 仅索引可见范围，无结果不是全网/私有/离线消费者清零；无需为品牌改 module。已知外部 owner 声明存在消费者时只收敛该具体路径 |
| 发布/外部状态 | 最近5条 GitHub deployment 记录均为历史 GitHub Actions production（2026-08-25/26）；本次 merge 的传统 commit statuses 为空 | 不证明 Kubernetes/Vercel 没有实际运行部署；不据此启动/停机/撤权。保留运行资源名，现有集成 owner 只确认名称依赖 |
| 活跃协作 | 本次核对开放 #692、#885；本批同一 Writer 续用原独立 worktree，未改共享 remote | 协调切换窗口，不接管其他任务。共享 Git common dir 更新一次 remote，其他独立 clone 由其 owner 更新；不移动工作区或结束用户会话 |

### 当前可以执行与尚未具备的条件

- **已核实准备**：源码消费者定位（包括当前官方知识出处生成/校验/读取及相邻测试）、静态策略最小同步范围、保留标识/保护/环境清单、8个App的安装授权、Webhook实际错误与回退限制已明确；owner确认没有当前外部名称/subject依赖。
- **最新范围**：用户已撤回官方知识GitHub来源的保留方案，V2独立准入及源码实现按新合同完成；最终CI/独立检查见PR，不把旧方案/旧评审或CI当新要求完成。
- **切换时同步/保留**：上述四处静态值、普通当前badge/模板/入口链接、Governance App Homepage展示链接与各clone remote需要同步；官方知识来源处理以V2当前设计为准。原CI/拒绝语义与运行资源保持。
- **不作为切换前置**：修复无关的既有直连通知、批准App权限升级、恢复旧publisher、重命名module/镜像/namespace、为无当前依赖的第三方项目新增部署测试。各自未执行项不记PASS。
- **仍需实际动作授权与窗口核对**：用户对实际更名的单独授权、当前PR/Actions及活跃owner状态、GitHub执行时的目标名最终校验、准确操作与回退负责人。开放PR本身不要求先合并或关闭；不擅自中断其他会话。当前只完成准备，没有执行更名。

官方依据（2026-10-22 查阅）：[GitHub OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#immutable-subject-claims)、
[仓库更名](https://docs.github.com/en/repositories/creating-and-managing-repositories/renaming-a-repository)、
[Webhook delivery API](https://docs.github.com/en/rest/repos/webhooks#deliveries)。仓库设置/API 快照与外部接收方实际状态分别记录；
未经 owner 事实确认的配置不做推断。

## 验证范围与未执行

最初Phase 0/1只做文档差异、相对链接和保留命令/路径检查；该历史阶段无新业务行为，未添加TDD测试。
用户后续批准的官方知识V2按冻结设计完成目标TDD、Go owner/HTTP装配与授权、前端BFF/页面和类型检查；
准确HEAD、CI、独立最终检查及限制记录于PR #887，不将旧文档阶段或旧SHA结果冒充本轮验证。
真实provider、镜像发布、实例更新/部署、实际更名、切换后集成与用户产品验收仍为NOT_RUN。
历史性能/验收记录保持原文，不因品牌统一或V2源码交付成为当前运行PASS。
