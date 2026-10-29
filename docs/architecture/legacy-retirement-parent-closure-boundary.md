> #301 / #985 **SUPERSEDED：独立 SHEIN 登录与身份回写完整退休**。按[用户已确认的完整边界](legacy-shein-login-identity-sync-closure-retirement-boundary.md) §1–§7/§4.2/§6.1，本文件涉及独立登录 CLI/runtime/service、Cookie/Profile/RedisConfig、Store 执行归属/身份回写及旧 session transport/商家仓库查询的 KEEP、安装或运行指引仅为历史；这些专属闭包已从源码退出，不留 alias/fallback/替代 owner。当前 Store 观察同步、官方授权及其它当前能力保留。既有表/记录/权限/Redis/Profile/凭据/进程无操作，运行停用 NOT_RUN/NOT_AUTHORIZED；不增加试用前置，#301 OPEN。以下起点 main 完整原正文、历史、批准、失败、数字与勾选逐字保留。

> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。仅替代本文件 listingruntime.ProductImportMapping/原 types.go 必须存在、旧 MappingInfo 编码及 listingruntime 仍属当前依赖的准确保留含义；本轮最后值类型完整退休。SourceIdentity 四 JSON 零值槽位、六 Source*、原 trim/规范化/hash/读取拒绝/当前版本及 opaque metadata 继续保留，不删当前序列化字段或改变业务事实 owner。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

# #29 / #301 源码退休收口核对与关闭边界设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **用户已确认冻结设计 `5fb20f877`；五文档及 #29/#301 当前段收口校准已实施，In Review**。Design Basis: **N/A**，仅现有证据与关闭边界记录，不改变产品范围或运行合同。源码核对基线仍为 main **`6cb487bfeebae1a0daeecbd3b80ce7a4df329a03`**（2026-10-20）。按 §5/§6/§6.1/§7 准备 #29 源级关单依据，#301 保持 OPEN；准确实施 HEAD、原 CI及最终独立检查留 [#854](https://github.com/qq550723504/task-processor/pull/854)。不修改生产源码、测试、workflow 或原验收勾选；合并及关单另行授权，不执行实例/数据/凭据操作。

## 1. 用户结果、Authority 与判断范围

使用者是维护当前应用、内部试用及决定父目标交付/关单的人员。交付结果是一份原剩余项、实际 owner/消费者、现有验证和真正关闭条件的完整映射，避免已退休入口反复派工，也避免为了清空 Issue 把仍在使用的 worker 或共享合同误删。本维护不阻塞当前核心试用操作，不增加内部试用前置。

依据为本轮用户明确范围、[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、两父 Issue 当前范围及原剩余项/验收。沿用 [最终 UI/IA Authority](../product/final-ui-ia-authority.md)，不修改页面/导航或把 Figma 终稿当当前功能验收。UI/Figma 新设计 N/A；没有执行视觉验收。

关键已确认边界：[#665 旧 handoff](https://github.com/qq550723504/task-processor/pull/665)、[#673 旧 UI](https://github.com/qq550723504/task-processor/pull/673)、[数字执行/持久化](legacy-numeric-task-execution-retirement-boundary.md)、[旧 Task 持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[完整 SHEIN](legacy-shein-listing-closure-retirement-boundary.md)、[TEMU](legacy-temu-listing-closure-retirement-boundary.md)、[Amazon](legacy-amazon-listing-closure-retirement-boundary.md)、[Publishing runtime](legacy-shein-publishing-runtime-closure-retirement-boundary.md)、[Store 管理](legacy-numeric-store-management-closure-retirement-boundary.md)、[独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)、[本地验收](legacy-local-acceptance-closure-retirement-boundary.md)、[发布执行](legacy-kubernetes-release-execution-closure-retirement-boundary.md)、[发布身份](legacy-kubernetes-release-principal-retirement-boundary.md) 与 [API token 工具](legacy-api-token-tools-retirement-boundary.md)。这些批准按准确对象生效，不因旧标题重新恢复已撤回要求。

Must：一次覆盖 #29 原 10 项剩余工作/7 条验收和 #301 原 6 条验收；保留有效当前能力及独立 SHEIN 登录/Store 同步；证据绑定代码与执行范围；源码、运行、生产、关单分别判断。Should：复用现有 guard/CI/文档，不新增验证工具。Out of Scope：全仓逐声明 dead-code 工程、当前 owner 重写、新身份/凭据/迁移系统、全量 Figma/平台交付、真实环境验收、workflow/inventory/依赖/保护修改。Accepted Risks 不新增；真实旧材料/实例状态 **UNKNOWN**，本轮真实运行和用户验收 **NOT_RUN**，生产 **NOT_READY**。

## 2. Design Basis、当前证据与适用限制

**Design Basis: N/A**：本轮及 §5 已确认校准仅文档/Issue 记录，不新增或改变领域合同、权限/租户、事实 owner、schema、状态机、事务、恢复、计费或外部副作用。没有未来生产代码删除的提前准入。若后续具体调查证明必须改生产路径，先给出准确闭包与适用 Design Basis、用户确认，再编码；不能把本文 N/A 套用于新的认证/持久化设计。

### 2.1 可复用的已执行证据

| 证据 | 实际绑定 | 支持的结论 / 不能推导的结论 |
| --- | --- | --- |
| [#852](https://github.com/qq550723504/task-processor/pull/852) 源 HEAD `64c225eb02ede3bb2fa4cce91071232d7661bb1d`，实际 merge 为历史源码退出基线 `dde52b79bcd80ec0a36290db62c450d1562a3f40` | merge parents=`b04459eab2366b5389d3219d2c242a04f63bccc4` + 该源 HEAD；tree=`cda2c74d51fd26af60a7d554c121b0df3c26e895`，与已检 PR 组合一致 | 旧 token 三源闭包和保留矩阵已合并；不证明任何现存 token/服务已撤销或停止 |
| [历史源码退出 push/main CI39415470994](https://github.com/qq550723504/task-processor/actions/runs/39415470994) completed/SUCCESS，headSha 精确等于历史源码退出基线 `dde52b79bcd80ec0a36290db62c450d1562a3f40` | Backend `go test ./... -count=1`、原 PostgreSQL 权限/商业读库绑定/启动 schema context、listingadmin race；Frontend（含 browser/accessibility）、Capture、Code Health、Release Authority、Admission、Classification、Required Gate 均成功 | 既有自动化合同在基线通过；Architecture/Windows job 依原 workflow 为 SKIPPED，不写 PASS。Backend 全套仍覆盖根 `tests` 包；不是生产/人类/真实平台验收 |
| [#852 准确源 HEAD 独立复核](https://github.com/qq550723504/task-processor/pull/852#issuecomment-6289028935) Completed，无 finding | 仅绑定 #852 的实际 diff；三项历史设计 P2 已在该 PR 分类修正 | 可复用未变化实现证据；不等于本文整个父目标结论已获复核或用户验收 |
| [当前基线 push/main CI39418897983](https://github.com/qq550723504/task-processor/actions/runs/39418897983) completed/SUCCESS | event=push/branch=main/headSha=`6cb487bfeebae1a0daeecbd3b80ce7a4df329a03`；Required Gate、Backend、Frontend、Capture、Code Health、Release Authority、Admission、Classification 成功，Architecture/Windows 依原 workflow SKIPPED | 已包括 #853 图像尺寸诊断组合；旧 CI 单独保留为历史证据，不能代替此基线；仍不是生产或用户验收 |
| 本轮在当前基线重新读取 `go list -deps`，八个既有命令均成功，无 forbidden dependency | 见下表；执行于 `6cb487bfe` 加本轮纯文档，生产树相同；不调用 main/constructor、数据库、浏览器或 provider | Windows/default tags 编译图无旧根依赖，不冒充跨平台运行；不是只从 #853 文件名推断组合没有风险 |
| 当前组合原导入守卫 `go test ./tests -run '^TestLegacyConsumersStayWithinDrainBaseline$' -count=1` **PASS 17.808s** | `6cb487bfe` 加本轮纯文档；tracked Go AST/legacy baseline 代码与当前候选一致 | 对同步 main 的组合做窄核对，覆盖 build tags/nested modules 的声明导入，不重复全套 CI或新建检查器 |
| `tests/legacy_hard_cut_test.go` 的 `legacyConsumerBaseline` 为空；`TestLegacyConsumersStayWithinDrainBaseline` | 解析 tracked Go（含 tests/build tags/nested modules，排除 testdata），拒绝新增 compatibility/tenantbridge import 及漂移 baseline；depguard 原规则保持 | 旧桥消费者归零与防复活证据，不是“全仓无任何数字字段/旧名字”的证明 |

### 2.2 当前依赖与 owner

| 现有命令 | 旧 bridge/compatibility/root ListingKit/app-task 依赖 | 与本设计有关的保留边界 |
| --- | --- | --- |
| `current-application` | 无 | `storecenter`、当前 Publishing 纯值/规则与 `listingruntime`；本基线不编入 `listingadmin`、`sheinlogin` 或 `sheinloginmanaged` |
| `image-agent-temporal-worker` | 无 | 当前 `storecenter`，不编入独立 SHEIN 登录和旧数字 Store 仓库 |
| `shein-login-worker` | 无 | 仍编入 `listingadmin`、`sheinlogin`、`sheinloginmanaged`，保留数字账号/Store DB、Redis/Profile/attempt 与同步 owner |
| `source-account-registry-schema-init`、`product-listing-api-schema-migrate` | 无 | 当前 schema/Organization/源账号初始化，不因旧命名删除正常 schema 工具 |
| `1688-local-agent`、`1688-batch-import`、`1688-public-browser-collector` | 无 | 当前采集入口；没有把纯去依赖证据当成真实登录/采集验收 |

`internal/compatibility` 只剩 README；root `internal/listingkit` 无根 Go package，仅保留原生 `phoneonboardingpreflight` 子包。`internal/tenantbridge` 已不存在。测试中的旧路径/负例是防复活证据；当前 Prompt HTTP 合同、固定拒绝的旧发布入口、原生 Login V2、当前工具及目录旧名称不等于旧业务执行 owner。不能只按 `listingkit`、`Task`、数字字段或零包外引用删除文件。

## 3. #29 原剩余项与验收映射

以下为用户已确认的证据映射，**原 Issue 验收勾选和历史正文保持**。准确设计 HEAD 的独立核对及用户确认已完成，本轮仅校准当前记录并准备源级关单依据；没有承诺所有产品/平台功能已经开放。

| ID / 原剩余工作 | 当前源码/消费者与原证据 | 收口判断 |
| --- | --- | --- |
| R1 root ListingKit 的 marketplace 规则退出 | root Go owner 缺席；`marketplace/shein/draft.Builder` → `publishing/shein.NewAssembler(AssemblerConfig{})` 为当前本地纯组装；validator/严格 codec 保留，项目边界已按 #827 校准 | 旧混合 owner 已退；不为目录整齐再搬当前规则 |
| R2 compatibility/listingkit 逐项 EXTRACT→RETIRE | 无业务 Go 源/消费者；空 legacy baseline、depguard、`TestLegacyConsumersStayWithinDrainBaseline` | 源码退出有证据；README 是禁止复活的规则说明 |
| R3 无生产调用的 legacy preview adapter | `TestLegacyPreviewAdapterStaysRetired` 和 renamed-revival guard 覆盖旧 `AdaptLegacyPreviewShell`；当前 `publishing/shein/preview_adapter.go` 被当前 assembler 调用 | 旧 adapter 与当前值投影不同；后者保留，不重新删同名能力 |
| R4 1688 Task handoff 退出主链 | #665 已合并；`TestLegacy1688TaskHandoffAndRouteStayRetired`、`tests/issue30_sourcing_boundaries_test.go`；当前来源 owner 为 Sourcing/Catalog | 旧 route/DTO/接线退出已有证据；#30/#307 新业务真实导入验收仍归原 owner |
| R5 单一 submission state owner | 当前 `listing/submission.ExecutionKernel` + official intent/receipt，当前 persistence adapter 实现原 port；#827 删除旧 attempt/event/lease/recovery 协议 | 当前 Listing 执行 owner 保留；平台 effects/receipt adapter 不是第二套状态机；真实发布 NOT_RUN |
| R6 runtime assembly 只做 wiring | `internal/app/*` 原 composition；`TestBusinessDomainsDoNotImportAppRuntimeAssembly`、assembly/import guards、项目边界 | 复用已有源码约束，不把收口变成整仓架构重写 |
| R7 TEMU/Amazon 不复制 Workbench/facade | 两个完整数字刊登闭包已退；`TestTemuListingClosureSourcesAreRetired`、`TestAmazonListingClosureSourcesAreRetired` 与八命令 dependency guard | 旧执行层已退出；平台 API/规则/来源采集和共享合同原样 |
| R8 ownership 文档及 depguard/import guard | 当前 owner 设计、上述 guard、零 legacy import baseline 与原 CI；原文档仍有 checkpoint 旧状态与未勾选项 | 源码有证据，当前适用说明需按 §6.1 校准；原历史段/计数保留 |
| R9 Commerce Tool 窄合同，不泄漏 SDK/client DTO | `internal/commercetool` 的固定 schema/authority 校验；当前 canonical inspection 只依赖 domain ports，`TestCommerceToolCanonicalInspectionUsesOnlyNarrowDomainPorts` 等；该工具/agent 范围无平台 SDK import | 当前窄 seam 有具体证据，不把未实现的未来 marketplace Tool 变为本父目标新需求；不宣称所有未来接口均已验证 |
| R10 领域事实与最终 UI 投影分离 | Product/Catalog、ApprovedAsset、Listing Record/Submission 各自 owner；#673 旧 Task-first UI 退出，`web/listingkit-ui/src/test/legacy-ui-retirement.test.ts` 与 login return-target 拒绝回归；当前 Console/Auth.js 保留 | 无需新顶层 Listing Center；旧目录名不等于旧产品。源码边界不等于完整 Figma 产品验收 |

| ID / 原验收标准 | 对应证据与关闭解释 |
| --- | --- |
| A1 新 marketplace 不复制 Product/BusinessTask/Workspace/submission/Agent 核心 | R1/R5/R7/R9 的当前 ports、Kernel 与 owner；验收是架构扩展边界，不要求本批实际新增 marketplace、扩功能或跑 provider |
| A2 SHEIN/TEMU/Amazon 规则由对应边界独立维护 | 当前 `marketplace/*`、平台规则和已确认纯值 owner；旧运行闭包退休。不能把所有有效规则强行搬目录当新 Must |
| A3 root 不再是新规则/功能 owner，余点有去向 | R1/R3 与依赖图；剩余 Login V2 子包及共享规则逐项 KEEP，不是永久旧业务 facade |
| A4 compatibility 不是长期 landing zone | R2，禁止新消费者的 AST/depguard 继续有效 |
| A5 没有平行 submission/retry/platform workbench 框架 | R5/R7；当前跨平台 Kernel 与 effects 适配、原 Temporal 基础层各守职责，旧数字恢复协议已退 |
| A6 Listing 可被页面/Agent/Tool 复用，不强建独立顶层导航 | R9/R10；当前事实与展示 authority 分离；未做真实产品视觉验收 |
| A7 自动化 guard 保护边界 | 基线 Backend 全套、原精确 absence/import/dependency/depguard 与 UI guards；不增加新 runner/验收系统 |

用户已确认：#29 的原 legacy 源码退出目标已有可复用证据，本轮未识别出在既定保留边界内必须新开生产删除批次的闭包。§6.1 当前记录已校准，R1–R10/A1–A7 作为源级关单依据准备；完成本 PR 适用 CI及最终独立检查后交用户决定合并，关单另行授权，当前仍 OPEN。不把 #30/#307 的业务/环境验收自动增设为 #29 关单前置。具体真实源码 finding 如出现，按 AGENTS.md 分类和完整闭包处理，不能以此表预先免除。

## 4. #301 原验收、保留债务与运行边界

| ID / 原验收 | 当前证据与未完成边界 |
| --- | --- |
| O1 空业务库/新配置正常初始化和创建读取，不调用旧迁移 | 当前 SourceAccountRegistry/schema/current-application 的原合同与 CI PostgreSQL fixture 提供部分执行证据；旧 import/migration/recovery 源已按 #842 等退出。整条命名新环境正常业务路径本轮未执行，不能由源码缺席推导成功 |
| O2 当前 Organization 唯一 owner，跨组织/撤权/依赖失败拒绝 | 当前 `authidentity`/Organization/授权装配、原 scope/权限/schema/隔离测试与基线 CI；独立 worker 仍用自己的数字 tenant/Store owner，不能将其改称新 Organization 事实来源。全父目标运行验收未完成 |
| O3 准确 legacy imports/callers，当前发布组合不引用旧 owner | 空 baseline、全 tracked AST guard、八命令 Go list 支持桥/旧根退出；worker 仍编译自己实际使用的旧数字账号仓库。当前应用成果和全仓数字 owner 退出分别记录 |
| O4 旧 resolver/bootstrap/兼容字段/路由无职责工件退出，保留回归 | #529/#695/#722 与后续完整闭包及原 guard 提供准确退休证据；Login V2、有效安全负例、当前共享合同和独立 worker 不能因旧名字删掉。本表不宣称全仓每个未使用声明均已逐一审计 |
| O5 不导入历史账号/商品/资产/任务/套餐/账务，不要求旧新对应 | Greenfield、旧 handoff/importer/Task/profile 链退休、当前源账号正常新建 owner与原边界 guard；本轮不读取任何真实数据来“证明没有导入” |
| O6 当前新系统与既有环境隔离，不授权停机/删除/部署 | 现有配置/启动 schema/权限与 #307 原隔离要求；真实环境归属、旧实例是否仍运行等为 UNKNOWN。无新环境执行即保留 NOT_RUN，不能把不操作写成已完成环境验收 |

### 4.1 不能消去的 worker 边界

[用户已确认的 worker 设计](shein-worker-identity-persistence-boundary.md)明确：保留独立 worker 与“全仓数字身份依赖清零”不能同时写成已完成，#301 保持开放。实际链仍为：

```text
cmd/shein-login-worker → app/runtime/sheinloginworker
  → listingadmin StoreRepository/StoreAPI + sheinlogin account provider
  → 原 tenant/Store 数字账号、DB/Redis/Profile/attempt/verification
  → sheinloginmanaged Store 同步与同 tenant 查重
```

#722 的跨 tenant 查重修复和 #831 的无消费者管理方法退出已合并；它们不代表剩余有效数字持久化 owner 已退出。当前 Organization、外部 SHEIN merchant 标识和旧数字 Store 行 ID 不互相等同；不得新增 ID bridge、迁移、凭据复制、fallback 或第二事实源来收口。

用户已确认 **#301 继续 OPEN**，明确“当前应用退出旧桥/旧执行链已有证据；独立 worker 数字身份/持久化债务按既有产品决定保留；父目标运行验收尚未完成”。本设计不新建替代 worker/IAM/账号页面，不重开其全局架构，不让其原生化、真实登录或数据处置成为内部试用前置。只有用户另行明确后续业务范围并满足其适用 Design Basis，才能解决该保留债务；若用户要改变父目标的阶段范围，也必须明确改目标，不能由 Agent 把延期/取消写为 PASS。

### 4.2 三类完成与权限

| 层级 | 充分记录 / 关闭条件 | 本轮结论 |
| --- | --- | --- |
| 有界源码完成 | 准确闭包已合并；保留合同/guard/准确 HEAD 复核与匹配实际 merge push/main CI 有证据 | #852 等已合入闭包有完成证据；本轮父级当前结论已按用户确认校准，准确实施 HEAD/原 CI/最终检查留 #854 |
| 运行/业务验收 | 仅由原 #30/#307 或对应身份/Store/worker owner 在命名、授权的范围执行；记录实际路径、环境/资源归属与 NOT_RUN | 本轮不执行、不增建平台、不追加试用门禁；既有原验收要求不被取消 |
| #29 关单 | §3 原项逐项核对、没有当前 Must 源码缺口；§6.1 经确认校准并完成适用 CI/独立检查；用户显式授权关单 | 源级关闭依据已按用户确认准备，待本 PR 适用 CI/最终检查及分别授权合并、关单；不要求为源级关闭新增真实 provider/全量产品验收 |
| #301 整体关单 | 原验收满足且保留债务得到合法处置，或用户显式改变阶段范围并准确保留未完成原目标；再另授权关单 | 当前条件不满足，继续 OPEN；不能因 bridge 为零或 #29 可关闭而联动关闭 |
| 生产发布/旧环境停机/数据处置 | 各自原 owner、明确环境与独立授权/验收 | NOT_READY / NOT_AUTHORIZED；任何源级关单都不授予这些动作 |

## 5. 真正退休闭包与已确认实施矩阵

本轮没有新增获准删除的生产闭包。旧 Task/queue/dispatch/recovery、平台数字刊登、废弃管理/内容配置、运维/发布、旧本地验收及 token 工具的准确删除已由各冻结设计覆盖。当前可执行采集/SourceAccountRegistry/Chain1 工具、Prompt 管理、平台 API/有效规则/纯值、原生身份/Login V2、Auth.js/issue357、Store 和独立 SHEIN 登录/同步具有当前 owner；保留它们不能被解释为允许新代码依赖已退休 owner。

| 处置 | 准确对象 | 边界 |
| --- | --- | --- |
| RETIRE（已实施的旧 owner） | 各引用冻结闭包及对应已合并 PR，含 #852 三 token 脚本 | 本轮只核对既有事实，不重复删除/复活，不宣称物理资源已退 |
| KEEP（仍有当前职责） | §2.2、§3 R3/R5/R9/R10、§4.1 的当前合同/工具/worker | 无源码/测试/配置/依赖改动；数字 worker 债务显式保留，不新增 Compatibility 类别 |
| EXTRACT | 无新增抽取 | 不以收口名义新建 facade、规则 owner、状态机、替代工具或验收器 |
| 已实施文档/记录校准 | 以下五份文档与 #29/#301 当前段 | 不改历史正文/计数；原 Must 不自动降低，不关闭 Issue |

```text
docs/architecture/legacy-retirement-parent-closure-boundary.md
docs/architecture/README.md
docs/refactoring/legacy-register.md
docs/refactoring/module-target-mapping.md
docs/refactoring/code-health-decisions.md
```

这是同一主要 PR 的冻结设计与已实施校准边界；**源码/测试 DELETE=0，源码/测试 EDIT=0，workflow/inventory/依赖/锁/保护 EDIT=0**。若发现真正未闭合生产消费者，先报告准确路径、owner、当前 Must 和完整删除/保留矩阵，再取得用户设计决定与对应准入；不把零删除结论当永久豁免，也不预先制造新退休批次。

```text
Legacy decision: RETIRE（既有闭包证据核对）；KEEP 表示明确当前职责，不是新增 compatibility 类别
Reusable behavior: 当前已确认的有效规则、平台/身份合同及 worker 能力继续由原 owner 承担
Current owner: Product/Marketplace/Listing/Organization/Store/SourceAccountRegistry 与独立 SHEIN worker 原 owner
Cutover/deletion condition: 本轮无新生产删除；父目标完成与关闭按 §3/§4 分别判断，真实运行和处置各需原范围授权
```

## 6. 必要核对与证据复用

当前基线 Backend 全套及原 AST/absence/导入/依赖/UI/授权/schema/race 合同已有准确 main CI；同步 #853 后补做八命令依赖和原导入守卫窄核对，明确绑定当前生产树。旧 CI 仅复用未变代码证据，不人工重跑完整 CI或执行工具来补“新鲜度”。本轮只做源码/Go list/消费者映射、文档/新增链接/原历史正文保护和适用的 Greenfield/Architecture README 文档检查，稳定实施候选走原未修改 PR CI；设计阶段证据与实施 HEAD 的 CI/最终独立检查分别记录在 #854。

没有新测试、checker、runner、故障矩阵、人工 session 或验收平台；不执行 PowerShell 验收/取 token 脚本，不读真实凭据/进程环境/私密配置/运行报告，不使用 Docker/Compose/kubectl、SQL、SMS/provider、浏览器或当前实例。源码静态证据、既有自动化 fixture、真实用户操作严格分列。

### 6.1 文档及 Issue 校准，两阶段分开

| 对象 | 冻结设计阶段记录 | 用户确认后的校准状态 |
| --- | --- | --- |
| 本文 | 原项映射、证据范围、真实保留债务及关闭建议，确认前 Ready=false | 用户已确认 `5fb20f877`；仅更新状态/结论，滚动实施 HEAD/CI/授权留主要 PR |
| Architecture README / Legacy Register / Module Target Mapping / Code Health Decisions | 只加“待确认收口设计”入口/顶注，原全部正文保持；不宣称全仓清零/父 Issue Done | 已更新同一本轮入口/顶注：旧 hotspot/checkpoint 按本次准确映射理解，#301 worker 债务保留；原历史段、失败、计数保持 |
| #29 当前段 | 只记录接单、N/A 与本轮设计范围；不勾旧原项 | 已校准为用户批准的 R1–R10/A1–A7 映射链接及源级关单依据准备；旧记录/验收原文不重写，关单另行授权 |
| #301 当前段 | 同上，保持 OPEN | 已明确当前应用成果、worker 保留债务与 O1–O6 执行边界；运行未执行继续 NOT_RUN，整体 OPEN |
| 原 worker/平台/API/Prompt/共享文档、所有 dated 记录、#300/#30/#307及其它活跃 owner | 不改 | 不接管、不关闭、不追加门禁；如需改变原产品目标，另由用户明确决定 |

#29/#301 上批当前段已逐字归档；本轮只切换当前校准段，两者的 #726 历史后缀和原验收勾选保持完整。Issue 长度限制不允许截断旧证据。设计与实施授权、适用 CI/复核证据保留在 #854；本次确认不授权合并或关单。

## 7. 独立检查、设计确认与停止点

原 GitHub 检查集中核对真实源码/当前消费者、原 10+7/6 项映射、证据绑定和限制、§5/§6/§6.1，以及“不借保留 worker 宣称 #301 全仓清零”与“不新加 #29 运行关单门禁”。这是 N/A 文档工作，不重复全局 Architecture Review，不新增 Writer/默认 AI 互验线程；同 SHA Running 不重复触发。finding 先按 AGENTS.md 分类，真实当前 Must 缺口在同一主要 PR 收敛，非 Blocker 建议不扩架构或试用前置。

用户已确认冻结设计，并授权本轮仅更新 §5 五文档及 #29/#301 当前段的适用结论；该校准已实施，Design Basis 仍为 N/A。授权覆盖必要现有检查、同 PR 提交/推送及维护，不覆盖生产路径的新变化。完成适用原 CI与准确实施 HEAD 最终独立检查后交用户决定合并；#29 仅准备源级关单依据，#301 继续 OPEN。关单另需明确授权，不由设计确认、合并或 AI 评审自动产生。

所有生产源码、workflow、凭据/实例/数据/IAM/provider操作和关单在本轮都未授权。原核心内部试用可沿已有交付继续，既有当前能力/独立 SHEIN 登录与 Store 同步不变；物理状态 UNKNOWN，真实运行/用户验收 NOT_RUN，生产 NOT_READY。
