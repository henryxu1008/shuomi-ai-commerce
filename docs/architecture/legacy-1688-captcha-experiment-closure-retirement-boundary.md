> #301 / #942 **USER_CONFIRMED / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认冻结设计b78ae71ac的§1–§7（含§6.1），Reuse Existing Architecture / Ready及Issue/PR读回[6393895022](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6393895022)/[6393895515](https://github.com/qq550723504/shuomi-ai-commerce/pull/942#issuecomment-6393895515)先于生产修改。同一15工件整批收齐，无新生产EXTRACT/owner；六整退、18混合声明及两曲线专属旧测试退出，原聚合文件仅加一个闭包guard，真实RED→GREEN。25混合函数/point与7有效原测试、全部其它原guard精确保持；当前算法/人工取消/采集/有效debug/SHEIN登录/Store同步保持。workflow/完整inventory/依赖/保护与既有实例/数据/凭据/轨迹文件/进程不动，不增试用前置，#301 OPEN。准确实现HEAD、CI与最终交付复核留同一PR；设计复核不替代源码复核，合并另授权。以下b78ae71ac冻结设计完整Git blob逐字保留，其DESIGN_ONLY/待实施措辞仅为原阶段证据。

# 旧1688验证码试验与轨迹录制/回放完整退休闭包

## 1. 用户结果、范围与 Design Basis

状态：**DESIGN_ONLY / AWAITING_USER_CONFIRMATION**。Refs #301，父目标保持 OPEN。用户已同意先设计完整闭包，未授权源码实施、合并、关闭、部署或真实环境操作。本轮只修改本文、Architecture README、Legacy Register 三份文档；设计、源码、PR CI、实际 merge SHA 的 push/main CI、运行和产品验收分别记录。

使用者是维护当前采集与独立 SHEIN 登录/Store 同步的维护者。当前操作是识别仍编译在采集包里的闲置验证码试验、录制/文件回放与失真工具说明，避免维护者误把它们当作当前采集路径。交付一份可确认的完整删除/保留、实际消费者、文件/JSON owner、前批 KEEP 替代和原检查矩阵。不新增采集能力、验收工具或人工步骤，不把本批作为内部试用前置。

**Design Basis: N/A；文档工作 Ready。** 理由：本阶段只有调查和三文档设计，不改变生产路径、权限、状态、持久化或副作用。未来实施为 **Reuse Existing Architecture / Admission: NOT_READY**：复用现有采集、验证码处理、浏览器、日志和 local-agent owner，退出无当前调用的试验闭包；不新增生产 EXTRACT、事实 owner、schema、状态机、恢复协议或 provider 行为。不涉及 UI/IA，Figma N/A。15 工件、没有跨系统新机制，现有批准架构覆盖保留路径，不为形式重开全局架构。

用户确认 §1–§7（含 §6.1）后，先在 #301/主要 PR 记录并读回 Reuse Existing Architecture / Ready，再由原唯一 Writer 在同一分支/PR整批实施，不按文件数机械拆批。若刷新发现真实当前消费者或具体外部可观察契约，停止受影响删除并报告；不得加兼容、迁移、包装、fallback 或第二事实源。

Out of Scope：workflow、完整 inventory（CURRENT/历史/计数/ceilings/parser/检查）、依赖/锁、保护规则均零改；不 tidy、升级、迁包、复制算法、增加替代 CLI/runner、浏览器模拟框架或验证平台。不读取既有凭据、配置、profile、轨迹文件、数据或日志，不操作既有实例、浏览器、数据库、缓存、进程或真实站点。删除只针对受追踪源码，不递归清理目录或 ignored/untracked 状态。#29 不重开，#301 保持 OPEN。

## 2. 权威、基线与旧 KEEP 的准确替代

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[目标架构](project-target-architecture.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)和用户本轮范围。当前有价值的能力继续复用原 owner、Playwright、标准库和 Logrus，不重复实现。

调查 main **d5e1b5672cc8379557b520651d85cd9fde6ea1cd**，tree **275edb2f51b246ba22e0c32c663cfd690ed9887e**。已解析全部 **3266 个受追踪 Go 文件**，包含测试、build tags、嵌套 debug 模块，解码 import/alias 并检查同包声明、函数体引用与外部构造入口；目标包没有 dot/blank import。文本核对覆盖脚本、workflow、文档和专属路径，不从历史 inventory 数字推断可删。只读调查未执行浏览器/业务命令。

唯一 Writer 会话 `01a133ef-757c-73e3-b8f6-76958c642397`，复用自有独立 worktree / 分支；[接单与上一当前段逐字归档](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6393740905)。开放 #941 当前采集扩展、#931 依赖、#692 Store 设计均无本闭包交集，不接管、不重复派工。

| 旧依据 | 本批准确替代，须用户确认后生效 | 持续有效 |
| --- | --- | --- |
| [#893 日志门面](legacy-logger-facade-closure-retirement-boundary.md) §5 构造站点表广泛保留 `test-recorder` | 仅撤销 `hack/debug/test-recorder/main.go` 的继续保活；该工具只构造普通 Processor 并调用 Process，却宣称会录制/回放，没有连接 recorder | `test-1688`、`test-amazon`、`test-real-track-slider`、共享 `hack/debug/go.mod`/go.sum、当前日志传递与全部其它保留边界不变 |
| 同文的 crawler/alibaba1688 辅助函数广泛 KEEP | 仅替代 §5 的三整退文件和十八个精确混合声明；旧无调用的技术试验不因日志注入历史继续存在 | 当前 processor/page operator、实际验证码处理、取消/人工策略、采集质量、安装/下载、browser/logging owner不变 |
| inventory 的历史 crawler 数量/迁包阶段 | 无替代；保持原字节，只作为日期绑定历史证据 | CURRENT 真实消费者及所有 parser/计数/ceilings/检查保持；本批 **0 行** inventory 校准 |

未来仅对 #893 文档加本批准确顶注，原完整 Git blob 逐字保留；不改旧数字、历史勾选、批准/失败和其它 KEEP。本阶段不修改旧依据文件。

## 3. 实际消费者与完整闭包

### 3.1 当前路径必须继续成立

| 路径 | 当前 owner / 实际调用 | 本批决定 |
| --- | --- | --- |
| 原1688浏览器采集 | `integration/crawler/a1688.NewLegacyProcessor` → `crawler/alibaba1688.NewAlibaba1688Processor` → single processor → PageOperator | 全部 KEEP；不因 legacy 名称退出有效采集/Source snapshot/Prepare/Shutdown 合同 |
| 当前验证码处理 | `page_operator.go` 按原 manual 策略调用 `HandlePageCaptcha` 或 `HandlePageCaptchaWithoutManual` → handler → `performSliderAction` → `optimizedSlideWithHumanBehavior` → `optimizedSlideWithRealTrack` → `generateRealHumanTrackPoints` | 全部原样 KEEP；当前算法、`point`、`randomDelay`、等待/取消与产品页判断不变 |
| 其它现役验证码路径 | click/image/text/math/login、manual fallback、statistics、HandleCaptchaWithRetry 和既有测试 | KEEP；不顺带重新定义站点策略、删退有效算术规则或放宽原 deadline 断言 |
| 当前匿名 public collector | `integration/acquisition/a1688/browser/captcha.go` 的单次有界尝试、未解除 challenge 的错误分类 | 整文件/调用者零改；不合并两条采集路径，不改变不支持类型、预算和诚实失败语义 |
| local-agent | `localagent` 的 Crawler/Runner/Source snapshot、Prepare 与 product checker | KEEP，设备身份、租约、提交、验证/安全与错误 owner 不变 |
| 有效调试 | `hack/debug/test-1688`、`test-real-track-slider`、`test-amazon` 及共享 debug 模块 | 全部 KEEP；test-real-track-slider 的普通采集调用确实进入保留算法，只编译、不执行 |

`optimizedSlideWithRealTrack` 是根据固定参数生成轨迹点的当前算法，**不读取录制 JSON**。不能因名字含 RealTrack 将它与旧 recorder 一起删除。

### 3.2 已无当前调用的试验链

| 闭包 | 源码证据 | 决定 |
| --- | --- | --- |
| fast/human slider 变体 | `newFastSlider`/`newHumanSlider` 无引用，FastSlide/HumanSlide 没有调用；randomInt/randomFloat 只被两个试验文件使用 | 两文件及专属类型/构造/方法/随机辅助整退，无新算法 owner |
| 录制/文件回放 | `checkAndHandleCaptcha` 与 `handleSliderCaptchaWithRecording` 均无调用；globalRecorder 仅 init 与后一个闲置方法引用；LoadTrackFromFile 无调用 | recorder、init/global、三私有类型、保存/加载/回放及两个断开 handler 同退，不留空壳或替代入口 |
| 其它模拟/曲线变体 | §5.2 的模拟 typing/movement/scroll/wiggle/typo、Bezier 与 quick/human points/延迟/曲线没有当前调用根，部分 helper 仅互相引用；complexEasing/Variation 仅被本闭包和两项专属测试保活 | 同一闭包精确删除，不把无当前要求的技术算法抽取成新共享库 |
| 专属工具 | record-only/record-simple 自建 Playwright + 注入 JS + 写同名轨迹文件/可选回放；test-recorder 没有实际录制接线；无脚本/workflow引用 | 三个受追踪 main.go 整退；不执行、不替换、不递归删目录 |

跨包构造消费者为 `test-1688`、`test-real-track-slider`、`test-recorder`、`integration/crawler/a1688` 和 local-agent。前者中的 test-recorder 按本批退休，其余全部保留。旧轨迹类型是包内私有结构和工具局部结构，未进入当前商品、Store、SHEIN、Source snapshot 或 HTTP 合同；同名 `point` 的局部变量不能当作类型消费者，实际生成函数仍需要包内 `point`。

## 4. 配置、持久化与副作用 owner

| 边界 | 旧 owner 与本批效果 | 必须保持/停止条件 |
| --- | --- | --- |
| 轨迹 JSON | recorder 与两个专属工具分别定义 mousePoint/trackData，拥有 track/type/x/y/time 与距离/起终点等局部字段；保存到 cwd 的 `captcha_track.json`，加载入口未接当前路径 | 仅退代码，已有文件/外部调用状态 **UNKNOWN / NOT_INSPECTED**；不读、不删、不迁移、不提供兼容 decoder/回填 |
| globalRecorder | 包 init 只创建 recorder/组件日志的内存对象；实际录制/保存需调用断开入口 | 随整文件退出，无新全局 singleton；当前日志注入、其它初始化和并发/锁行为不改 |
| 浏览器/站点 | 旧录制工具会启动浏览器、注入监听、捕获/保存用户操作、移动鼠标并回放；本批不执行这些路径 | 当前浏览器采集、人工处理与 public collector 策略完全不变，不新增 provider 调用或试验 |
| 当前业务持久化 | 本闭包没有当前 DB/Redis/Store/SHEIN/SourceIdentity/商品 JSON owner | 全部零改；不新增 schema/迁移、配置/env key 删除、凭据/目录复用或第二事实源 |
| 目录/模块 | 受追踪对象只有 §5.1 六个源码文件；共享 `hack/debug/go.mod`/go.sum 保留 | 不改变依赖、不删除可能存在的 ignored/untracked 模块/产物，不执行工具或检查真实轨迹文件 |

若实施中出现具体当前外部可观察调用或持久化义务，立即报告用户，暂停相关删除；当前没有可自行执行的兼容例外。源码/fixture/编译/CI 均不证明已有环境状态、站点成功率或产品验收。

## 5. 完整十五工件删除/保留矩阵

### 5.1 六整文件 RETIRE

```text
internal/crawler/alibaba1688/captcha_fast_slider.go
internal/crawler/alibaba1688/captcha_human_slider.go
internal/crawler/alibaba1688/captcha_recorder.go
hack/debug/record-only/main.go
hack/debug/record-simple/main.go
hack/debug/test-recorder/main.go
```

共退出这六文件内26函数（含 recorder init、三个 main）、8类型及 globalRecorder。只删受追踪文件，不删整个包、共享 debug 模块或真实轨迹文件。

### 5.2 三精确生产混合文件

| 文件 | 仅删除的完整函数/专属注释 | 必须保持 |
| --- | --- | --- |
| `captcha_handler.go` | `CaptchaHandler.checkAndHandleCaptcha`、`CaptchaHandler.handleSliderCaptchaWithRecording` | 剩余12函数完整 body/签名/注释、全部 imports/常量、manual/cancel/retry/统计与策略不变；fmt/strings/time仍有消费者，不能删除 |
| `captcha_human_behavior.go` | `CaptchaHandler.easeInOutCubic`、`complexEasing`、`simulateHumanTyping`、`simulateHumanMouseMovement`、`generateBezierCurvePoints`、`simulateHumanScrolling`、`simulateRandomMouseWiggling`、`simulateTypingMistakes` | 仅移除孤立 fmt import；保留 `randomDelay`、`optimizedSlideWithRealTrack`、`generateRealHumanTrackPoints` 三函数和 `point` 两字段原字节；context/math/time/playwright与原当前算法/取消等待不变 |
| `captcha_slider.go` | `CaptchaHandler.generateQuickDragPoints`、`quickEaseFunction`、`generateHumanDragPoints`、`humanEaseFunction`、`calculateHumanStepDelay`、`calculateVerticalWobble`、`calculateStepDuration`、`complexEasingWithVariation` | 仅移除孤立 math import；其余10函数及imports、当前策略参数/距离/等待/人工/成功/跳转判断原样；与 humanSlider 同名方法按 receiver 区分 |

三文件合计精确退出18函数；25个剩余函数原样保留。不迁包/抽新 helper，不改当前行为来迎合删除。六整退+十八精确声明共 **44函数**，不含两项专属测试；数字用于差异核对，不作为验收目标。

### 5.3 两测试工件

| 工件 | 处置 |
| --- | --- |
| `internal/crawler/alibaba1688/captcha_slider_test.go` | 只删除 TestComplexEasing、TestComplexEasingWithVariation 和失去最后消费者的 math import；这是无当前消费者曲线的实现细节测试。其它7函数及原 fixture、算术/产品页真伪/类型/取消及时性/统计断言逐字保持 |
| `tests/listingkit_backend_retirement_test.go` | 只新增一个本闭包函数 TestLegacyAlibaba1688CaptchaExperimentClosureIsRetired，复用既有 trackedFiles/Go AST/失败关闭；所有原函数/helper/import不改，详见 §6.1 |

### 5.4 一旧 KEEP 顶注及三当前文档

| 工件 | 处置 |
| --- | --- |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 实施时仅前置准确 superseding 注记，替代 §2 两项 KEEP 的继续保活含义；原完整 Git blob逐字保留 |
| 本文 | 设计冻结后只前置当前实施状态，不用实现反写/重排已确认矩阵；发现真实边界冲突先报告 |
| `docs/architecture/README.md` | 前置本批阶段/链接，原完整正文不变 |
| `docs/refactoring/legacy-register.md` | 前置准确闭包/owner/保留边界，原完整历史、计数和勾选不变 |

本设计阶段严格只改最后三文档；另十二工件均是拟实施对象。其它生产、测试、样例、文档、依赖与配置零改。

## 6. 验证与交付边界

设计阶段只做读取/AST 调查、文档链接/矩阵与旧 blob 保留核对、git diff --check 和适用原文档检查；原 PR CI绑定准确设计 HEAD。不构建/执行业务入口、不生成运行态 fixture或安装浏览器，不把文档 CI 写成源码/运行验收。

未来实施在同一批内先添加 §6.1 原测试文件内的闭包 guard，确认原代码产生 RED，再删除完整闭包并确认 GREEN。不以放宽当前 deadline、取消、授权、challenge 或产品页成功断言收敛。保留的现役规则不为形式重写测试。

必要开发自检限原聚合 guard、本包保留的七项测试和原 handler 取消/manual 策略测试，以及受影响 import/依赖边界；真实浏览器/网站/DB/Redis/凭据不运行。稳定候选交原完整 PR CI，不重复手工全量 CI。编译只需当前 application、独立 SHEIN worker、1688 local-agent/public collector 和保留的 test-1688/test-real-track-slider，使用任务临时输出且不执行；共享 debug 模块参数/依赖不变。未变的其它入口证据可明确引用，不冒充新 SHA 的直接运行。

本批只交付源码可维护性与保持原消费者可编译，运行/站点/产品验收均 **NOT_RUN**。源码完成、用户确认、正常合并和准确 merge SHA 的 push/main CI独立记录；不增加用户人工验收步骤或试用前置。

### 6.1 原检查完整处置矩阵

| 检查 | 唯一允许修改/验证要求 |
| --- | --- |
| 新闭包 guard，放在既有 `tests/listingkit_backend_retirement_test.go` | 先 RED 后 GREEN；六精确路径必须缺席，不能把含 ignored 文件的整目录缺席作为要求。解析所有受追踪 Go，按原 `internal/crawler/alibaba1688` owner 禁入五退休类型/全局 recorder/旧自由函数及精确 receiver 方法；按 owner 区分同名变量/方法，不禁其它领域合法名称。保留 point 与三现役 human_behavior 函数、原 handler/slider 主调用及四个保留 debug/current consumer 路径的正向断言；解析/读取/缺少预期对象必须失败，不加 missing-skip。复用原 helper，不增 imports/helper/runner或新检查系统 |
| 原 TestComplexEasing / TestComplexEasingWithVariation | 两项删除，math孤立import退出；本文件其它七项函数/断言完全不变，无替代算法测试 |
| 原三个 Alibaba1688 import 边界检查 | 全部保持；当前包与 SourceAccount/ListingKit root 禁入仍成立，不改遍历起点、排除表或 fail-closed 逻辑 |
| 原 TestListingKitTaskBackendSourcesAreRetired、TestLegacyLoggerFacadeClosureIsRetired 及其它聚合/AST/depguard | 没有目标文件/声明的正向 KEEP 需转换；原逻辑和断言全部零改。实施若真实失败，先核对同根路径，报告精确冲突/补丁并由用户决定，不借“退休”默认改 guard |
| workflow/完整 inventory/依赖与检查逻辑 | 全部零改；CURRENT 四消费者仍有效，没有本批删除行；原 architecture/code health/安全/必需 gate保留，不改 CI目标/保护/skip |
| 原文档检查与历史 | 三当前文档仅前置；一旧 KEEP 文件仅实施后前置；其余原 blob、计数、历史结果/勾选保持，不重算或替换旧证据 |

“四个保留 debug/current consumer 路径”为 test-1688、test-real-track-slider、integration/crawler/a1688/processor.go 和 localagent/runner.go；它们只是精确正向检查点，不限制 §3 所列完整 KEEP。

## 7. 顺序、评审与停止点

1. 当前只设计三文档，提交/推送同一主要 PR，记录准确 HEAD、适用原 CI与设计复核。用户确认 §1–§7/§6.1 前不改另十二工件。
2. 获确认后重新核对实际 main/消费者/重叠 PR，只记录真实组合变化；先在 Issue/PR 写并读回 Reuse Existing Architecture / Ready，唯一 Writer在同一 PR连续完成 §5 全闭包、必要自检及原 CI，不按小 helper拆批。
3. 独立检查绑定真实 diff/实际消费者/轨迹文件边界、#893 KEEP准确替代与25函数保留。优先复用原 GitHub 复核；同 SHA已运行不重复触发，具体 finding先分类，修复只复核相关增量。**未经用户本轮授权不默认新增 AI 互验线程**；复核不可用如实记录并交用户决定，不借旧批授权派新代理，不以作者自检或绿色CI冒充独立验收。
4. 源码稳定后交准确 HEAD、原CI/独立复核及限制，请用户另行决定合并；不得自行合并/关闭/部署。获准确合并授权后正常合并并确认实际 merge SHA 的完整 push/main run，不用PR CI或单Gate替代。

出现真实当前消费者、KEEP/原检查冲突、必须修改当前取消/授权/持久化/副作用边界或现有数据可观察合同，先报告根因、影响、最小修订和成本，暂停受影响修改。未变化的能力/证据不重复审核，不追加产品需求或通用治理机制。#301 始终 OPEN；当前采集、有效规则/JSON、原浏览器/有效调试、独立 SHEIN 登录与 Store 同步保持，既有资源与 workflow/inventory不动，不增加内部试用前置。
