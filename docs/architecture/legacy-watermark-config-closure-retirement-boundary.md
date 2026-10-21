# #301 旧水印检测/去除与专属配置完整退休设计

Status: **PROPOSED / DESIGN_ONLY**。用户已同意先做本完整闭包的设计，尚未确认本文或授权源码实施。当前 Design Basis 为 **N/A**（只读调查、本文、Architecture README/Legacy Register 提案入口及 Issue/PR 维护）；未来实施拟 **Reuse Existing Architecture / NOT_READY**，待 §7 的明确确认。调查基线 main `398ee52a7caa9769f0197c67f46f17947b7cbcfd`，其 push/main CI39625847629 completed/SUCCESS 仅为基线证据，不冒充本设计或未来实现 CI。#301 保持 OPEN，#29 不重开；一个 Writer、一个分支、一个主要 PR，不按检测器、去除器或配置机械拆批。

## 1. 用户结果、范围与权限

使用者是维护当前应用、图像、采集、Store 和独立 SHEIN worker 的维护者。预期源码结果是移除闲置的旧水印 Processor/检测/去除/直接 AI 请求整链，以及只为其保留的类型、示例与配置入口；当前用户继续使用已有图像能力和原 owner。没有证据表明本残留正在阻塞当前用户操作，不新增业务入口、替代水印服务或内部试用前置。

当前只授权设计。正式 Go 源码、测试、YAML、workflow、inventory、依赖/锁、ceilings 和保护规则均不改。未来拟在确认后同批处理 §5 的 12 个完整文件、两个混合配置源及 §6.1 的准确检查/文档尾部；不把当前提案入口写成已退休。

不执行旧 constructor、Processor、示例、AI/provider、业务 CLI、应用或 dry-run；不读有效环境配置、现有 API key、私密 binding/Profile/代理文件、Secret、token、日志或报告，不操作既有实例、数据、凭据、进程或服务。不启动 PostgreSQL/Redis/browser/AI 服务，不调用 kubectl/SQL。配置调查仅针对受 Git 追踪的源声明/样例键结构，不读取运行配置或凭据值。`go list` 只加载依赖元数据，不运行包业务代码。物理状态和外部人工使用 UNKNOWN，运行/用户验收 NOT_RUN。

## 2. Design Basis、Authority 与不变量

当前 **Design Basis: N/A**；未来拟 **Reuse Existing Architecture**，直接复用：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)和[Issue 驱动规则](../engineering/issue-driven-development.md)。仅 EXTRACT/RETIRE，无兼容类、旧配置适配、fallback、迁移或第二事实源。
- [当前百炼图片接线](bailian-product-image-wiring-v1.md)、[当前图像视觉 brief](image-skill-visual-brief-v2.md)及其 product/image、ImageAgent/provider、资产/计量原 owner。本文只退休无业务消费者的旧包，不改变这些当前合同。
- [#836 配置闭包](legacy-runtime-config-closure-retirement-boundary.md)、[#801 旧图片包](legacy-temu-image-processing-retirement-boundary.md)、[#807 图片 API/prompt 尾部](legacy-temu-image-api-prompt-retirement-boundary.md)和[#872 配置/helper 闭包](legacy-queue-proxy-config-tail-closure-retirement-boundary.md)的当前保留边界。本文未来仅取代其中对本次准确旧包/字段的暂留含义，不扩大到图像、AI、共享库或配置重构。

本轮不新增产品能力、公共领域合同、事实/schema owner、状态机、事务、恢复协议、授权/组织隔离、计费或现役 provider 调用边界。当前图像规则、百炼生成参数、资产保存/批准、计量/预算、采集、Store 观察同步、独立 SHEIN 登录/Store 同步和 UNKNOWN 不自动接管合同均保持。UI/IA/Figma N/A。

Must：完整旧闭包一起退出；保留同名但有效的当前图像行为；原必要/安全检查保留；无真实状态操作和新试用前置。Should：复用原 guard、标准库和当前成熟图像组件，不抽取没有当前消费者的算法库。Out of Scope：新增水印能力、模型/SDK替换、通用配置严格化、依赖清理、历史文档整理、实际服务/数据处置。Accepted Risks：无新增，不把未访问的外部状态判为空。

## 3. contract → implementation → injection → consumer

基线以全仓受追踪 Go 源和实际 `go list` 核对，不能只从文件名或“没有 importer”判断退休：

| 对象 | 实际接线/消费者证据 | 当前结论 |
| --- | --- | --- |
| `internal/pkg/watermark` | 全仓 Go import 只有 `internal/core/config/config.go` 和包内 `example_test.go`；外部没有 Processor、Detector/Remover manager、DefaultConfig 的业务调用 | 无当前业务执行入口；不是当前 product/image/provider 合同 |
| `core/config.Config.Watermark *watermark.Config` | 是唯一生产引用。core/config 内 watermark 命中只有该 import 和字段，BuildConfig、defaults、env bindings、loader、validator 均没有该字段接线 | 架构残留根因是 schema 类型引用把闲置实现带入编译依赖，不是缺少一层新 wrapper |
| `config/config-dev.yaml` 的 `watermark` | 唯一受追踪配置 YAML 的旧 watermark 根；子树含 detection/removal/AI/performance 声明，下一个根为 imageagent。只核对源键结构；当前 prompt 中同词语义保留 | 是专属旧样例入口，不是已确认有效运行配置或当前产品能力 |
| 当前三个入口 | `go list -deps` 对 current-application、shein-login-worker、image-agent-temporal-worker 均包含 watermark；源码没有调用其运行方法 | 说明仍有编译依赖，不说明正在检测/去除或调用 AI；删字段后应退出依赖图 |
| `internal/product/image/heuristics.go` | 当前 AssessIPRisk 根据 logo/文字等 audit 给出图像风险理由；未 import 旧 watermark 包 | 保留原图像风险规则，不以“watermark”文字命中删除 |
| `internal/integration/bailian/image.go` | 当前图片请求带 `parameters.watermark=false`；未 import 旧包 | 保留当前 provider 参数和预算/资产接线，不改成旧移除服务 |

旧包的 Config → Processor → DetectorManager/RemoverManager → traditional/AI/hybrid 与 inpaint/blur/crop/AI 全部是包内闭包，示例是唯一构造调用者。不存在已核实的当前业务切换需求，不将示例或历史 “retain pending owning-domain review” 作为 Architecture Authority。当前名称和包路径不同的有效图像 API/规则继续原 owner。

## 4. 配置、持久化与副作用 owner

| 边界 | 源码中的旧行为 | 未来处置及保留 owner |
| --- | --- | --- |
| 配置/进程内状态 | Config/DefaultConfig、Processor 持有配置、UpdateConfig 重新创建 manager；全局仅保留类型字段 | 旧声明同退；core/config 与当前 runtime 配置继续原 owner。不新增旧配置 reader、alias、兼容承诺或新一轮 unknown-key 检查 |
| 纯图像处理 | 区域/置信度、传统检测、合并与 inpaint/blur/crop 在内存处理 image.Image | 无当前业务消费者，随旧包 RETIRE；不为潜在复用新增算法库或把旧质量值当当前验收标准。现役 imagex/imaging 等原样 |
| AI 外部请求 | detector_ai 的 OpenAI/Anthropic POST、图像编码和响应解析；remover_ai 向配置的 LaMa ServerURL POST | 整条旧调用源码删除，不运行、不转接当前 AI owner、不新建计量/预算/重试机制。现有 provider、凭据、服务或费用状态 UNKNOWN，不撤权/销毁 |
| 名义配置槽 | VisionAPI.MaxCost、CommercialAPI、Lama ModelPath/GPU、Performance CacheEnabled/TTL 等声明 | 类型/默认值随包同退；这些槽位不是已核实的额度账本、缓存或文件持久化 owner，不因此迁移/清理任何状态 |
| 数据库/缓存/文件 | 包内没有 GORM/SQL/Redis、文件写入或独立 durable business asset owner；HTTP body 读取不构成新持久化 owner | 不新增 schema、迁移、outbox、恢复/清理器；图像资产/审批、当前身份计费与 Store/worker 的真实持久化不变 |

没有新的事务、幂等、retry/UNKNOWN/recovery 或资源所有权协议需要设计。旧来源代码退出不等于旧服务、凭据、图像或数据已物理退出，也不授权探测。若刷新发现实际保留消费者、当前外部可观察契约或必须改变这些边界，停止相应实施并按 §7 报告。

## 5. RETIRE / KEEP / EXTRACT 完整矩阵

### 5.1 一次 RETIRE 的准确对象

| 对象 | 未来同批删除边界 |
| --- | --- |
| 11 个生产 Go 文件 | `internal/pkg/watermark/{types,processor,detector,detector_traditional,detector_ai,detector_hybrid,remover,remover_inpaint,remover_blur,remover_crop,remover_ai}.go` 整体删除；包含专属 Config/DTO/defaults/managers/AI 请求/解析/区域与质量 helper |
| 1 个专属示例文件 | `internal/pkg/watermark/example_test.go` 同退；不运行示例，不为旧实现补 provider fixture 或新验收框架 |
| 混合配置源 1 | `internal/core/config/config.go` 只删 watermark import 和 `Config.Watermark` 字段；其余 schema/loading/validation/env/logger/平台/身份/图像配置原样 |
| 混合配置源 2 | `config/config-dev.yaml` 只删 `watermark` 根子树及其专属说明；前邻 Redis、后邻 imageagent 及所有无关键保持，不读取/修改生效配置或凭据 |

全局 builder/default/env/validator 没有另一个 watermark 入口，不虚构迁移或扩大删配置。只按准确文件/节点操作，不递归删除混有共享消费者的祖先目录，不留空包、假 importer、别名、fallback 或替代 owner。

### 5.2 KEEP：当前消费者和合同

- product/image 的图像审核与风险规则、当前图像处理/填充/上传/资产批准、ImageAgent/Temporal 与当前 AI/prompt/预算/计量/产品合同；百炼 `watermark=false` 生成参数和当前图像 brief/prompt 全部原样。
- `internal/pkg/imagex`、downloader/httpclient/jsonx、成熟 imaging/SDK 依赖和其它有效共享基础层，不因旧调用者减少而同批清理。
- 全部非 watermark 的 core/config/schema/默认/加载/环境/校验；`retired_product_runtime.go` 等当前拒绝旧配置的有效安全 guard 不因名称含 retired 而删除。
- 当前采集、平台 API/规则、共享值合同、身份与计费、Store 观察/同步、独立 SHEIN 登录/Store 同步和 UNKNOWN/fence 边界；当前 UI、token handoff 和原生身份工具。

**无新 EXTRACT**。旧纯算法理论上可复用不构成本次当前需求；当前有效行为已在现 owner，不从旧服务再包装一套。

```text
Legacy decision: RETIRE
Reusable behavior: 无新抽取；有效图像/API/规则/共享能力继续现 owner
Current owner: product/image、ImageAgent/当前 provider 与资产/计量原 owner；core/config 仅保留当前 schema
Cutover/deletion condition: 用户确认本设计与 §6.1，刷新无当前业务消费者，准确旧包/字段/样例与检查/文档尾部同批退出；不承诺或操作物理状态
```

## 6. 必要验证与证据边界

设计阶段仅做源声明/实际依赖元数据、键结构、文档索引/引用一致性和原适用文档检查；不为纯文档制造 RED，不执行旧实现或新增验收环境。当前设计 CI/复核不得冒充未来实现、合并后 main CI、运行或用户验收。

确认实施后，复用既有 retirement/AST/import decoding/YAML/依赖 guard：先让准确旧包/字段/样例缺席断言 RED，再删除至 GREEN；读取/解析失败必须 fail closed。当前入口的实际依赖应不再包含旧包；相关 current config、imagex、当前图像风险/provider 参数等原有效检查保留。稳定候选运行原必需 CI，未变化代码证据明确复用，不额外建设 fault matrix、runner 或验证工具的验证工具。

### 6.1 原检查、文档、inventory 与 workflow 的精确处置

| 对象 | 未来同批最小校准 | 必须保留 |
| --- | --- | --- |
| 包内 example_test.go | 与旧包同退；不把旧示例当当前产品需求或运行证据 | 当前图像/安全/资产/预算/权限/幂等/组织隔离的有效测试 |
| tests/listingkit_backend_retirement_test.go | 在现有设施加一条完整 watermark 包缺席、生产 import 禁入、样例根缺席及当前入口真实依赖退出断言；重用现有 parser/decoder，不建扫描器 | 读取/解析失败拒绝、其它退休根/共享 Namespace/当前 SHEIN worker 与全部有效合同 guard；不删整测试或 missing-skip |
| internal/core/config/config_test.go | 原 TestCurrentConfigDoesNotRequireRetiredTaskInputs 加 Watermark 字段/专属 defaults/env 键缺席；原 YAML 设施覆盖开发样例仅目标根退出 | 其它全部配置、browser/platform/Retry/身份/图像加载/校验及现有拒绝旧输入断言，不扩大为全局配置 parser 重写 |
| Architecture README / Legacy Register | 当前只加 PROPOSED 入口；实施后仅将本矩阵更新为确认/源退并给准确依据 | 其它 owner、文档引用、历史行/计数/失败不改；新 guard 只有实现后才登记为当前覆盖 |
| docs/refactoring/phase2-runtime-inventory.md | **零改**。pkg/watermark 暂留行属于 dated 技术分类，未来由本文和 Register 的确认注记精确取代；CURRENT consumer register 没有 watermark 行，无失效行需要删 | 完整文件、历史/计数、CURRENT 表、provider inventories、ceilings 和准确性检查逻辑均原样；core/config 其它 provider 依赖仍有效 |
| 原测试/说明尾部 | 原直接文档 guard/引用检查验证新的提案/确认入口；本批没有其它已核实的当前旧操作手册或发布入口 | test-baseline.txt 和 dated 旧证据不改写；当前 bailian/image brief、prompt 的 watermark 语义保留 |
| 全部 workflow / 依赖与锁 / 保护 | **零改**；原 CI、必需与安全检查正常运行，无新的安装/编译目标/豁免 | 触发/分类/权限/并发/Required CI Gate、商业/数据库安全检查、ceilings 均原样 |

当前只提交本文和两个提案索引；上述源码/测试/配置更新尚未应用。若实际 guard 与冻结边界出现冲突，先报告准确失败和最小处置，不先改 inventory/workflow/ceilings 来求绿。

## 7. 确认、交接与停止点

1. 以准确设计 HEAD 交付真实消费者/owner 和 §1–§6.1 矩阵；一次原 GitHub 设计检查只审本闭包，同 SHA Running 不重复触发，不默认增 AI 互验线程。N/A 只覆盖设计交付，不授予源码实施权限。
2. 用户明确确认 §1–§7（含 §6.1）后，在 #301/同一主要 PR 记录 **Reuse Existing Architecture / Ready**、冻结设计 HEAD 和唯一 Writer，再正式修改生产源码。普通当前边界不重复全局架构审核。
3. 按 §5/§6.1 一次完成源码/示例/配置/guard/直接文档尾部；必要开发自检、提交/推送和范围内 CI/review 修复留同一主要 PR，不机械拆小批。设计复核不替代完整实现后的独立最终检查，修复只复核真实增量。
4. 发现保留消费者、当前可观察兼容义务、必须新增 owner/迁移/兼容、权限/租户/事实/状态/恢复/副作用变化，或需要额外 inventory/workflow/源删除时，停止对应部分并报告 Must、阻塞层级和最小差异。只有新的真实 BLOCKER 才重开冻结架构，不能靠假设扩建设计。
5. 完整交接准确实现 HEAD、原 CI、独立最终检查和保留能力/NOT_RUN；等待用户准确 HEAD 合并授权。正常 expected-head 合并后核对真实 merge SHA 的 push/main CI，#301始终 OPEN；部署、真实环境操作、运行/用户验收、关单和下一业务扩展均不自动开始。
