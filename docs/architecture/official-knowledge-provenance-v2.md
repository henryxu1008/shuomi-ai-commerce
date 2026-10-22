# 官方知识库 V2：产品内依据，不使用 GitHub 地址

Status: IMPLEMENTATION_READY / FROZEN。独立设计复核完成后进入实现。
Execution: [#884](https://github.com/qq550723504/task-processor/issues/884)，同一 Writer / PR #887。
Design Basis: Independent Architecture（官方 Article 公共读取合同变化）。

## 产品决定与用户结果

2026-10-22 用户明确要求「官方知识库不要使用github地址」。本决定替代 #884 中此前
“本次保留官方v1固定GitHub出处/白名单”的准备方案，也替代 [V1设计](official-knowledge-v1.md)
中仅与外部GitHub来源链接有关的合同；历史版本和评审证据保留，不宣称旧方案已实现新要求。

用户仍通过 AI工作台 → 知识库 → 官方知识库 → 查看内容，在产品内阅读真实指南、准确版本、
维护方及依据名称。页面/API/目录构建都不使用GitHub地址，不仅是隐藏链接。
Figma/Product Authority：[当前 UI authority](../product/final-ui-ia-authority.md)、原官方页面
`682:359`与本次用户批准的局部变更。保留原页面/来源卡片，只将外部链接变为纯文本依据；不新增导航。

Scope：移除官方来源URL字段与GitHub白名单、发布指南v2、准确读取与拒绝旧shape、现有页面/文档/测试同步。
Out of Scope：CMS/新资料站/内链路由、爬虫/远程加载、数据库/schema/数据迁移、AI引用/provider、
身份/租户/权限、发布状态机、企业知识、共享实例/部署/资源/凭据、实际仓库更名或合并。
不增加新Accepted Risk，原只读Threat Model保持：内容是被动文本，不变成HTML/授权/动态URL。

## Owner 与最小调用路径

canonical owner 仍为 `internal/knowledge/official` 的编译期嵌入目录，不建企业资料副本或第二事实源。
contract → implementation → injection → consumer：Article来源名称合同 → 原Go目录/嵌入正文 →
原HTTP module/正常Workbench装配 → 原鉴权BFF/Zod → 原官方目录/正文页面。
路由、owner、租户/身份/knowledge.read、no-store、超时/大小上限及取消/撤权行为均不变。

ArticleSummary字段不变。Article仍为 Summary + body + sources，但来源变为：

```json
{"sources":[{"title":"企业知识与引用规则"},{"title":"知识选择与版本依据"}]}
```

- Go `Source` 只保留 `Title`，删除URL字段；校验名称非空、UTF-8、512 bytes上限，原1..10项与文章大小约束不变。
- 前端source schema严格只接受title；旧 `{title,url}` payload必须拒绝，不能兼容、忽略url或转发旧shape。
- 页面依据卡片使用文本列表，不创建a/href/target，不增加可点击但无内容的内链。
- 内置来源名称仍据现有真实资料维护，不虚构新内容。仓库版本与评审继续作为工程维护依据，
  不进入官方资料运行合同；代码依赖管理/工程Issue中的GitHub链接不是本次清理对象。
- 无外部URL解析、服务端fetch、重定向、token或供应商调用；不以字符串黑名单建立通用内容治理。

## 版本、切换与失败

当前已发布指南v1完整事实留在Git历史，不能原地改写。本次嵌入指南v2：正文中的
“v1/固定版本仓库出处”结尾改成准确的v2/产品内依据说明，更新正文digest/日期与revision。
文档正文其余操作事实保持，不声称新业务能力已开放。

新程序只发现/提供v2；v1准确GET返回既有404，不映射latest、不返回旧来源、不并存旧URL接口。
这复用V1 §7已批准的“后续release可不提供某revision”的合同，不新增withdrawal API或状态机。
无数据库持久化、数据删除/回填、旧ID映射、双读双写或新恢复协议。
代码发布使用原应用/前端方式，但本任务不更新当前保留实例；旧实例仍需原owner另获授权更新。

后端与前端严格合同在同一主要PR交付；旧server/newclient或反向组合按原schema失败，不做fallback。
实际安装如以后获准，使用同一候选配套backend/UI，不能把拒绝旧shape解释为已安装成功。
构建损坏/digest错误拒绝构造，文章不存在保持404，鉴权失败清除正文；正常只读GET可安全重试，
沿用原取消/deadline/大小限制，无write/UNKNOWN/补偿步骤。

## 验证与交接

业务合同变化用TDD：先增加“嵌入v2没有GitHub/URL字段、v1准确404、前端拒绝URL字段、页面无出处链接”
的目标测试并确认RED，再修改Go目录/正文、Zod和页面；复用现有测试体系，不建runner。
覆盖源名称无效/过长、digest、准确版本和副本不可变；保留原HTTP/Workbench授权/组织/取消测试。
复用原Go official owner/adapter/装配、前端BFF/页面测试和既有必需CI；完整正常实例试读未执行保持NOT_RUN。
交接入口仍为 `/workbench/ai/knowledge/official`，当前新版本详情为
`/workbench/ai/knowledge/official/ai-commerce-guide/revisions/2`，未部署前不宣称已可用。

Legacy decision: RETIRE（V1外部URL来源读取shape从当前运行合同退出）。
Reusable behavior: 原嵌入目录、准确版本、纯文本显示及鉴权/取消/有界读取。
Current owner: 官方知识owner及原HTTP/BFF/Console，不依赖legacy owner。
Cutover/deletion condition: 新读取合同同PR配套、相关测试与必要CI完成，用户/独立验证另行确认运行结果。

## Review / Admission

Round 1：独立 Architecture Review 对设计HEAD `24fa7579bfee600e4c228ae5309560e001781f67`
于2026-10-22T08:57:59.409499Z完成，无finding/未解决finding，
[复核记录](https://github.com/qq550723504/task-processor/pull/887#issuecomment-6335777014)。
准入结论：IMPLEMENTATION_READY。该依据先于正式Go/TS/内容修改；按本冻结合同实施，
只有真实BLOCKER才重开架构，非阻塞细节由实现测试收敛。
