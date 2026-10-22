# 硕米智能引擎 / Shuomi AI Commerce 品牌与仓库更名清单

依据 [#884](https://github.com/qq550723504/task-processor/issues/884) 的 2026-10-22 产品决定。
本文件交付 Phase 0 影响盘点与 Phase 1 文档品牌落地；不是实际更名或部署指令。
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
| 源码引用 | `extensions/1688-capture/package.json` 与 lock 的 `@task-processor/1688-capture`；前端 `listingkit-ui` 包名、目录、lock | 保留工程标识，无发布命名空间迁移。外部 npm 发布/依赖方 UNKNOWN。owner：扩展/前端维护者 |
| 构建引用 | Makefile、Dockerfile、release/build 脚本与 config | 实际入口与路径保持；根 config YAML/Makefile 没有命中 `task-processor`，不能据此断言所有运行配置无关。owner：构建/运行 owner |
| CI 标识 | `.github/workflows/ci.yml` concurrency 的 `task-processor-ci-*` 与 classifier 测试 | 保留，避免切换时产生另一组并发任务；通知 URL/仓库名消费 `github.repository`，改名后仍需实测。owner：CI 维护者 |
| 安全身份 | `release-policy.yaml` 的 `releaseAuthority.repository`、`policy/listingkit-release-authority/{release_authority,negative_fixtures}.rego` 与 `tests/listingkit_release_authority_policy_test.go` | **切换前阻碍**：硬编码旧仓库身份。不能当文案替换或改为接受任意仓库；需授权后保持精确身份、负例和拒绝语义，评估该安全边界准入。owner：release authority |
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
2. **精确安全身份**：先明确 release authority 策略的影响、owner、适用设计准入与发布方式；
   准备同一切换变更中的 policy/guard/test 精确一致更新及原负例验证。保留 namespace；
   不删除检查、不接受任意仓库、不借更名修改 CI gate/保护。未收敛前不执行切换。
3. **外部集成**：由 owner 解决上表会影响当前消费者的 UNKNOWN，尤其 webhook、App/OIDC、
   外部 `uses:`/clone 与发布引用。GitHub 官方说明 action 调用不会因仓库更名自动重定向；
   不用普通链接重定向替代集成验证。未确认影响时保持待切换。
4. **实际切换**：上述条件满足后，由获授权 owner 更名并记录新 URL、时间及仓库 ID；
   更新当前 README badge/模板/活动开发文档 URL、准确策略引用及各 clone 的 remote。
   保留历史证据和 Go 模块、目录、镜像、资源名；不在旧名称下新建仓库覆盖重定向。
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
- 更名回退不保证已失败 CI/webhook 自动恢复。重新核对 clone/fetch/push、准确 HEAD 的 CI、
  hook 投递及已确认的外部消费者；未验证保持 UNKNOWN，不能通过关闭安全检查完成回退。

## 本批验证与未执行

仅做文档差异、相对链接和保留命令/路径检查；无新业务行为，不添加 TDD 测试或验证工具。
CI 按原工作流执行，准确 HEAD/结果和独立最终检查记录于对应 PR，不在本文件滚动追加。
Go build/test、前端运行、真实 provider、镜像发布、部署、实际更名与切换后集成验证均 NOT_RUN。
历史性能/验收记录保持原文，不因本批品牌统一成为当前 PASS。
