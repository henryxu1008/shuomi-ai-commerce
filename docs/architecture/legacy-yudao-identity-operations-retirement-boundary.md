# 旧 Yudao 身份导入与 owner 恢复运维链：完整退休闭包

Refs [#529](https://github.com/qq550723504/task-processor/issues/529) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **DESIGNING / 待独立设计检查及用户确认，尚未实施、未进入 Ready。** Design Basis: **Reuse Existing Architecture**。调查基线 main `4fdc30b0c0d6c4535279f3fe103d9453e28cfdfc`（2026-10-20，#839 合并后）。用户已同意先设计这一完整闭包；本次授权仅覆盖静态调查、设计和 PR 维护。准确设计 HEAD、检查结论与确认记录写主要 PR。一个 Writer、一个主要设计/实施 PR，不按本地/远端、脚本个数或证据拆批。

## 1. Product Outcome / Scope

使用者：维护当前内部试用、身份、采集、Store 观察及独立 SHEIN 登录/Store 同步的操作者。可见结果：仓库停止提供已取消的历史 Yudao 租户/用户导入、旧 ID 元数据映射和 owner 恢复入口；清理清单不再把这些入口当作当前恢复能力。当前用户沿原登录、组织开通、授权和正常交付路径操作，不增加内部试用前置。

完整闭包为三个用户导入脚本、一个 owner 恢复 dry-run、各自内联 helper/报告生成合同、当前清单的保留结论与直接历史指引尾部。只删两个 local 入口、留下远端导入/恢复脚本，或只标注过时而继续提供可执行工具，均不算完成。不新增迁移、替代恢复工具、兼容入口或另一份用户事实。

这项源码维护不解决一个已证实的当前登录/启动故障，也不宣称新增业务能力或生产发布能力。生产仍 NOT_READY；源删除与真实 IAM、数据库、已有组织/用户、元数据、报告、实例的状态分开。

## 2. Design Basis / Authority / Threat Model

**Reuse Existing Architecture**：复用已批准的 [Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Auth / Tenancy 当前合同](auth-and-tenancy.md)、[#698 发布身份预检退休](release-identity-preflight-retirement.md)及 [SHEIN worker 身份/持久化边界](shein-worker-identity-persistence-boundary.md)。取消旧迁移/旧 ID 恢复已由当前产品决定覆盖；本批仅关闭准确源码入口，不新增或改变登录/权限/租户/组织合同、事实/schema owner、状态机、事务、retry/UNKNOWN/recovery 或外部副作用。UI/Figma N/A。脚本含跨 DB/IAM 操作，并不授权执行或重设计这些旧操作；本次不新增跨系统机制。

若实际发现仍有效的当前外部可观察契约或持久化运行态要求保留兼容，不自行构造替代 owner：停止该部分，提供具体证据交用户决定。若必须改变当前身份/权限/持久化/恢复等边界，升级 Independent Architecture，完成适用准入后才实施。不能用本设计为旧身份迁移重新授权。

- Must：整批关闭四个旧入口及专属尾部；明确取代旧“暂缓保留”的依据；当前认证、组织、共享基础设施和全部保留能力原样；原必需/安全检查保留；不操作真实环境或数据。
- Should：复用原退休 guard、文档索引和已有纯检查；不为内联通用语法另建库、替代 CLI 或扫描/验收平台。
- Out of Scope：当前 IAM/手机开通/成员邀请重构，独立 SHEIN worker 原数字账户 owner 替换，全局数字身份清零，当前 ZITADEL 安装/升级/备份恢复，真实身份/密码/元数据清理，旧数据迁移、旧 ID 映射、fallback/双读双写/第二事实源，workflow/inventory/计数/依赖/ceilings/保护调整。
- Accepted Risks：无新增。静态调查无法证明外部没有人工调用者，也无法确认任何真实历史资产是否存在、归属或可删除；这些状态保持 UNKNOWN，不作为本批运行验收 PASS 或清理授权。
- Threat Model：把含旧名字的当前 ZITADEL、手机号探针、共享 Secret/DB/网络误删；将 DryRun 当作离线操作；把旧 retained-deferred 当成新的迁移授权；将源码删除扩成真实身份/元数据删除，或把旧恢复要求重新加为试用前置。只处理本批实际边界。

## 3. 实际消费者与闭包证据

仅调查 Git tracked 源文件、精确文件名、报告名、旧元数据键及其直接引用，覆盖 cmd/internal/hack/tools/scripts/tests/.github/deployments/config/docs。没有运行脚本（含 DryRun）、应用 constructor、kubectl、Docker、SQL、IAM/provider 请求，也未读取私有配置或生成报告。

| 入口 / 基线行数 | 实际调用与行为 | 当前结论 |
| --- | --- | --- |
| `scripts/migrate-yudao-users-to-zitadel.ps1` / 436 | 独立人工入口，无当前程序/workflow/test 调用；历史计划引用。读取旧 `system_tenant/system_users`（含密码 hash）及 ZITADEL 三个 projection；创建组织、导入用户、写旧元数据，按旧 ID 查重/复用；输出 JSON 文件和 stdout | 整个远端导入/旧 ID 复用闭包 RETIRE |
| `scripts/migrate-yudao-users-to-local-zitadel.ps1` / 188 | 独立人工单租户入口；只有 Code Health 的历史保留行。通过 Kubernetes 读取旧租户/用户，向配置的 ZITADEL 创建组织、导入 hash、写元数据；JSON 仅 stdout，无 OutputPath | 与全量/远端入口一起 RETIRE；名字 local 不保证不接真实环境 |
| `scripts/migrate-yudao-all-users-to-local-zitadel.ps1` / 300 | 独立人工全量入口；只有同表历史保留行。读旧租户/用户，为每租户创建组织并导入用户/元数据；JSON 文件和 stdout | 整体 RETIRE，不保留分批/限量形式 |
| `scripts/yudao-zitadel-owner-recovery-dry-run.ps1` / 135 | 历史 owner reconciliation 计划引用；读旧 `system_users`（含停用/删除用户）和 `projections.user_metadata5`；按旧 tenant/user 键匹配已有 subject，生成 counts/fingerprint JSON 文件和 stdout | 报告合同随取消的迁移恢复场景一起 RETIRE；它不写 IAM，但不是离线无副作用工具 |

三个 importer 调用 `/v2/organizations`、组织 metadata、`/management/v1/users/human/_import`、用户 metadata。内联 Base64、组织/login 命名、JSON 行解析、SQL/HTTP 包装、旧 ID 查重表及 recovery fingerprint 都只有本脚本使用；四文件无 dot-source、Import-Module 或另一个脚本调用。因此没有还需搬运的专属 helper 文件，也不为保留通用算法创造消费者。

报告默认名为 `yudao-zitadel-import-report.json`、`yudao-zitadel-local-full-import.json`、`yudao-zitadel-owner-recovery-dry-run.json`；tracked 源中未找到消费这些生成报告的当前程序。报告不是新系统的持久化事实或当前准入回执。只退休生成合同，不查找、读取或删除现存 `.local/tmp` 等目录中的实际文件。

直接文档尾部不能只按四个完整脚本名搜索：`2026-08-10-listingkit-owner-reconciliation.md` 仍要求 Restore 旧命令，`2026-08-11-listingkit-owner-reconciliation-policy-design.md` 仍把 migration script / recovery step 写成当前策略；两者目前没有 SUPERSEDED 顶注。旧 manual-deploy 计划同样仍指向已退预检。另两份已有历史顶注的 owner 设计还保留“预检 reads 不变”的过时边界。§6.1逐项列出这组同根因引用：只新增/校准窄历史顶注，不重写正文，也不执行其 agent/sub-skill 或迁移步骤。

**架构问题**：当前产品已禁止旧账号迁移/ID 映射，但四个可执行源仍提供这条操作路径，清单仍称两个入口是恢复能力。根因是产品决定没有收口到完整运维入口及其当前说明。解决方法是按已批准决定关闭完整源闭包，并保留原本正确的当前 IAM/组织 owner；不是增强旧导入的幂等/补偿或新建迁移框架。

## 4. 配置、持久化与平台副作用 owner

| 边界 | 旧行为 / 配置 owner | 本批处置 |
| --- | --- | --- |
| 源 PostgreSQL | 三个 importer 读 `system_tenant/system_users`；recovery 读全部候选用户。namespace/target/database/user 等由显式参数及各脚本默认值选择 | 参数/内联连接包装随入口删除；不读写旧数据库，不把用户 hash/ID 搬到当前模型 |
| ZITADEL projection 读取 | 远端 importer 读 `projections.login_names3_users`、`org_metadata2`、`user_metadata5`；recovery 读用户 metadata | 关闭专属读取代码；不操作 ZITADEL schema/projection、部署或实际 metadata |
| IAM mutation | 三个 importer 依调用者 AccessToken/OrgId 创建组织、以旧 bcrypt hash 导入用户，并写 `yudao_*` / `migration_*` metadata；旧命名/查重/异常统计只属于这些 importer | 无继任执行 owner；不提取旧身份映射、用户名策略、hash 导入或恢复协议到当前账号/组织开通 |
| 报告与失败/重试 | 两个 importer 与 recovery 写本地 JSON；单租户 importer 只 stdout。旧工具的 partial success、重试与无响应结果不构成当前 recovery 合同 | 生成代码退出；历史报告/旧运行是否存在及实际结果 UNKNOWN，不重跑、不补偿、不确认成功 |
| 当前身份/组织 | `internal/authruntime/zitadel`、`internal/app/httpapi/server_auth.go` 与 `account_identity.go`、当前组织/角色/成员及原注入 owner | 全部原样；verified token、Organization scope、撤权/UNKNOWN、正常创建/自助修改等安全断言不动 |
| 独立 SHEIN 与 Store | 原账号仓库、Redis/Profile/Cookie/attempt、当前 Store 观察和独立登录/同步合同 | 原 owner 全部保留；不凭新 Organization 分配/接管旧数字账户、映射元数据或 durable business assets |

已有 IAM 用户/组织、DB/metadata、凭据、报告以及保留 worker 的物理资产保持原所有者；其实际存在与业务归属未检查，不能由“源码无人调用”推断无主。源码删除不 revoke/delete/drop/reset 任何实体，不修复/迁移旧数据，也不改变当前恢复/幂等责任。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 整批删除的四个准确源码入口

```text
scripts/migrate-yudao-users-to-zitadel.ps1
scripts/migrate-yudao-users-to-local-zitadel.ps1
scripts/migrate-yudao-all-users-to-local-zitadel.ps1
scripts/yudao-zitadel-owner-recovery-dry-run.ps1
```

1059 行仅用于说明调查完整度，不是拆批或交付价值指标。不按 `yudao`、`zitadel`、`migration` 或 `scripts` 做目录/关键词批量删除。

### 5.2 必须保留

| 保留对象 | 当前依据 / 边界 |
| --- | --- |
| 当前 ZITADEL adapter、认证装配、个人资料/密码、Organization/成员/角色、当前手机号 onboarding | 当前 owner 与当前原生身份流程，不消费四个迁移脚本；不更改领域合同、权限或 token/credential 处理 |
| `internal/listingkit/phoneonboardingpreflight/**`、`hack/debug/listingkit-phone-onboarding-preflight/**` | 原生 Login V2 非生产可行性探针及其纯/fake 测试；虽有 listingkit/preflight 旧名字，却不是旧 Yudao 导入/recovery。保留其原 non-production 确认、有限超时和凭据边界，不执行探针 |
| `deployments/docker/zitadel/**`、`deployments/kubernetes/zitadel/**` | 当前共享身份基础设施；包括 `docker-compose.yudao-db.yml` 显式选择的 DB 网络 overlay。网络装配不等于账号导入，不能以 yudao 命名认定专属 |
| `scripts/tests/listingkit-local-zitadel-compose-test.ps1` | 只检查 base 本地 Postgres/无 Yudao 网络、overlay 显式选择旧网络；没有 importer 调用。保留原行为，不为删迁移入口调整该断言 |
| 当前 schema/正常 schema evolution、Secret/示例/装配、备份恢复、通用 DB 查询/port-forward、凭据及共享基础层 | 没有最后消费者随四入口退出；不接管 #692、#841 或其它会话与保留环境 |
| 此前全部当前能力 | 采集/workerpool、通用 queue/Retry、平台 API/DTO/有效规则、两种 SHEIN 属性、AI/SKU/spec、PriceHandler/PriceManager/PriceCalculator、AI/prompt/图像、当前身份/tenant/资金/计费、Store 观察及独立 SHEIN 登录和 Store 同步原样 |

### 5.3 EXTRACT 判断

**本批无新增抽取。** 四脚本的 Base64/JSON/HTTP/hash/命名操作不是待迁移的业务能力；标准库与当前 owner 已有相应原语。旧 hash 导入、旧 metadata 映射与 owner 匹配本身属于已取消的场景，不能换名抽取或包一层当前 Service。当前有效身份/API/规则只复用原 owner，不改代码。

```text
Legacy decision: RETIRE
Reusable behavior: 当前认证/组织/手机号、共享基础设施及通用语法沿原 owner/标准库保留，无需新增抽取。
Current owner: 当前 authruntime/Organization/Store/SHEIN 原 owner 不变；旧导入/恢复无替代 owner。
Cutover/deletion condition: 用户确认本设计，两个 Issue 明确 Ready；原 guard 加入四入口并确认 RED 后整批删除/自检，最终检查与准确 HEAD CI 完成后另交合并授权。无运行停用、数据迁移或旧用户清理前置。
```

## 6. 实施与验证顺序

设计确认后，同一 Writer 在同一主要 PR 连续实施：

1. 记录用户确认的设计 HEAD/§5/§6/§6.1/§7，在 #529/#301 写 Ready 后再修改正式脚本或测试。复核 actual main 与活跃 owner，不为了持续最新反复 merge main。
2. 在原 `TestLegacyReleaseIdentityPreflightIsRetired` 的 absence 列表中仅加入§5.1四个路径，运行准确测试确认因四入口仍存在而 RED；保留全部原路径与有效权限/发布断言。
3. 整批删除四脚本及内联生成合同；完成§6.1当前/历史说明收口。准确测试 GREEN，精确文件名/报告名/旧 key 搜索确认剩余命中只有本设计/历史引用，不新增调用者或替代入口。
4. 使用 `git diff` 与基线对照§5.2路径、所有 workflow/inventory/计数/ceilings/依赖/保护均零改；运行原 Greenfield/架构索引与适用退休 guard。纯代码/文档检查不需要启动应用、Docker、真实 DB、IAM 或脚本。当前身份/成员/手机号及 Compose 测试源未变时明确复用原证据，不把未执行写为新 SHA PASS，不复制原 CI 全量检查。
5. 稳定候选形成前同步一次 actual main，只验证真实组合风险；集中最终独立检查完整 diff、保留边界和原 CI。一批交付，不派生 runner/专项验收工具或多个小 PR。

### 6.1 精确修改与零改边界

| 文件 / 集合 | 允许的最小动作 | 不能改变 |
| --- | --- | --- |
| §5.1四脚本 | 用户确认后整文件删除 | 不复制为新 CLI/module、兼容失败入口、模板或示例恢复脚本 |
| `tests/listingkit_backend_retirement_test.go` | 原身份预检退休 guard 的准确 absence 列表增加四路径 | 原所有路径、权限/发布/worker/current schema/安全断言；无新扫描器、subprocess 或真实运行 |
| `docs/refactoring/code-health-decisions.md` | 顶部窄注释：用户确认后，两个 local importer 的 retained-deferred 与通用 support-candidate 行中仅其迁移保留结论被本设计取代；实施前只能标“提案/未实施” | 原历史两行、当时 detector 证据/风险/计数及其他所有候选结论原样，不伪称旧清理当时错误，也不把零调用当唯一退休依据 |
| `docs/superpowers/plans/2026-08-11-listingkit-owner-reconciliation-policy.md` | 实施时仅校准现有 HISTORICAL/SUPERSEDED 顶注，明确新增四个退休入口及已退预检不能重建；删去顶注中“existing read-only preflight policies unchanged”的当前含义 | 原任务正文/测试/提交证据与日期原样；不按当时 sub-skill/迁移步骤重新派工或执行 |
| `docs/superpowers/plans/2026-08-10-listingkit-owner-reconciliation.md`、`docs/superpowers/specs/2026-08-11-listingkit-owner-reconciliation-policy-design.md` | 实施时各加窄 SUPERSEDED 顶注，链接 Greenfield、#698 及本设计；说明下方 restore/backfill、旧 metadata/migration/recovery 均非当前策略或操作、不得重建/执行四入口 | 原 Goal/Approved policy/历史任务、验收和当时 evidence 原样；不能保留一个无顶注的“当前恢复”入口 |
| `docs/superpowers/plans/2026-08-10-listingkit-manual-deploy-gate.md` | 实施时加窄历史/SUPERSEDED 顶注，链接 #698；旧预检与发布入口已退/失败关闭，不能按计划恢复。独立 SHEIN worker、原 OIDC/RBAC/安全检查仍保留 | 历史 release plan 正文/回执不改；不改任何发布 workflow、驱动或权限，不新建当前发布链 |
| `docs/superpowers/specs/2026-08-10-listingkit-owner-reconciliation-design.md`、`docs/superpowers/specs/2026-08-11-listingkit-system-owned-exceptions-design.md` | 实施时仅校准已有 SUPERSEDED 顶注：原“预检读取仍保留/等待另退”已由 #698 取代，本批关闭四个导入/恢复源码；其余历史决定仍为当时证据 | 正文、旧 snapshot/计数/事实与当前 schema 均不改，不恢复 seed/exception reader/preflight |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md`、`docs/architecture/README.md`、本文 | 当前提案→用户确认/源码退休的准确状态与边界索引 | 历史矩阵/计数、既有当前 owner 与其它保留决定 |
| 其它历史引用 | 2026-08-02 三份计划中当时“保留未提交修改”的证据原样，不能拿它当当前会话所有权；已有明确全篇 SUPERSEDED 的 source-account cutover/preflight 文档及 K3S 历史说明不扩写 | 不重写历史为新 SHA 验证，不把旧身份恢复计划重新变成当前 Authority；不能据此漏掉上列没有顶注或顶注边界过时的准确文档 |
| 全部 workflow、inventory（含当前消费者表与历史计数）、ceilings、依赖/锁文件、lint/depguard/保护 | **零改**；四脚本不产生 Go logger/OpenAI/current consumer 行变化 | 不跳过/改名/删必需或安全检查，不刷新历史统计 |
| 运行/私密资源 | **零操作** | 不执行四脚本（含 DryRun），不读/删报告、凭据、旧 DB、IAM、进程、队列、Profile、volume 或保留实例 |

本设计替代旧 retained-deferred 的前提是当前 Greenfield 产品决定明确取消历史账号迁移，且用户确认本文准确闭包；静态无调用者只是消费者核查证据。设计 PR 本身不能提前标注四入口已退休。

## 7. 删除条件、检查点与停止点

- **设计阶段停止点**：主要 PR 给出完整矩阵、基线、原独立设计检查的准确 SHA/结论与本地文档检查；交用户确认设计。此时不删除脚本、不改测试或当前运行配置，Issue 未 Ready。复用现有架构，不重开全局 IAM 设计；最多两轮正常设计检查，finding 先分类再修改。
- **实施准入**：用户明确确认本文及§5/§6/§6.1/§7，两个 Issue Ready 记录可追溯后再实施。若出现真实当前兼容义务或必须改变保留 owner/身份/持久化边界，停止相关改动并报告，不增加临时桥接。
- **最终交付检查**：一轮集中检查真实四入口 diff、原 guard、历史/当前说明、零改与保留能力、适用原 CI 的准确候选；实现者自检不等于用户验收。发现问题按 AGENTS 分类，同根因批量修复后只复核增量。
- **合并停止点**：交准确最终 HEAD、CI、独立检查与已知限制，等待用户单独合并授权；无管理员绕过、强推或保护变更。授权后正常 expected-head 合并，再核验实际 merge SHA 的 `event=push / branch=main` CI；PR CI 不替代 main CI。
- 不执行部署、真实用户/组织/metadata 删除、迁移、回放、恢复或报告清理，不关闭 #529/#301。生产 NOT_READY、runtime/provider/用户实际验收 NOT_RUN；不以此新增内部试用准入。独立 SHEIN worker 数字账户债务与其它父目标继续保留，不宣称全仓数字身份清零。
