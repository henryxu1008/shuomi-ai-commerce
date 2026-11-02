# 工程规则、接线合同与实现交接

返回[文档首页](../README.md)。本目录解释代码如何消费既有合同，以及具体候选的交付与验证范围。
需求从[产品依据](../product/README.md)进入，领域规则从[合同导航](../architecture/topics.md)进入，
日常安装和用户操作从[操作说明](../operations/README.md)进入。

| 阅读目的 | 入口 |
| --- | --- |
| 接单、提交与评审 | [执行规则](#rules) |
| 找已有消费者合同或接线说明 | [身份与成员](#identity)、[商品与店铺](#commerce)、[Console / AI 工作](#workspace)、[资源与服务](#services) |
| 追溯准确基线的验收与原始附件 | [专项记录](#historical)、[原始附件](#evidence)、[历史归档](../archive/README.md) |

同一交接可能包含仍有效的合同和日期绑定的历史结果。先读页首的替代范围、启用条件及未开放项；
`IMPLEMENTATION_READY`、源码完成、实例启用和用户验收分别确认，不能从旧 SHA / 地址推断当前运行状态。

<a id="rules"></a>

## 执行规则

先按当前 Issue 范围接单；这里维护流程，不维护任务进度。

- [代码健康规则与分类决定](code-health-decisions.md)：当前防回归入口及有范围的历史分类，不用旧探测计数推导当前删除权限。
- [Issue 驱动派工规则 V1](issue-driven-development.md)

<a id="identity"></a>

## 身份、企业与成员交接

消费者合同、原生 provider 边界和实现交接按各自原范围阅读。

- [企业空间操作记录 v1](account-audit-v1.md)
- [Account read-only contract (#346 → #348)](account-readonly-contract.md)
- [系统管理员模块展示修复](administrator-module-projection.md)
- [Current application: personal referrals server slice](current-application-referrals.md)
- [Issue #357 isolated official identity runtime](issue357-real-provider-runtime.md)
- [ACC-1 account profile and shared presentation](issue409-account-profile.md)
- [ACC-2 membership implementation contract](issue410-membership-contract.md)
- [成员原生验证附件](issue410-membership-native-evidence.json)
- [ACC-3: resources and optional SourceAccount management](issue411-account-resources.md)
- [Personal referral registration: Slice A](referral-registration.md)
- [店铺浏览器后端与固定角色展示组合](store-browser-permission-combination.md)
- [官方 ZITADEL 适配与受保护成员管理边界](zitadel-official-provider-boundary.md)

<a id="commerce"></a>

## 采集、商品与店铺交接

从对应领域合同进入，再核对实际接线、启用条件和未执行的真实平台验收。

- [Amazon 单商品插件交接](amazon-browser-capture-runtime-handoff.md)
- [Shein 只读诊断 BFF 与隔离联调](shein-diagnostic-bff-fixture.md)
- [SHEIN 资料诊断页面与受控浏览器验收](shein-diagnostic-ui.md)
- [任务中心：本地资料准备工作记录](shein-records-ui.md)
- [SRC-2B1 public 1688 acquisition](src2b-public-acquisition.md)
- [Store Center 商品与订单只读首版接线](store-observations-integration-handoff.md)
- [供应市场 v1 正常程序接线](supply-market-runtime-handoff.md)

<a id="workspace"></a>

## Console、AI 工作与经营投影

页面/消费者引用原 owner；源码交付与实例开放、用户验收分别确认。

- [1688 单图白底生成：架构与执行交接](2026-09-25-issue487-main-image-execution-design.md)
- [硕米统一 Console 基础](console-foundation.md)
- [Knowledge execution context and Product Agent consumer (#558 / #559)](knowledge-context-consumption.md)
- [官方知识库首版交接](official-knowledge-handoff.md)
- [运营驾驶舱首版交接](operations-cockpit-runtime-handoff.md)
- [任务中心标题提案待确认（#345）](product-title-review-ui.md)
- [个人报告保存与原离线包接线交接](report-center-runtime-handoff.md)
- [店铺经营诊断源码交接](shop-operations-diagnosis-handoff.md)
- [统一任务中心：业务任务、原生运行与待确认事项（#798）](unified-task-center.md)

<a id="services"></a>

## 资源与服务交接

安装、授权和资源边界以原合同为准，当前运行步骤从操作说明进入。

- [Data Services v1 runtime handoff](data-services-runtime-handoff.md)
- [企业存储容量 V1 试用入口与启动条件](enterprise-storage-capacity-runbook.md)
- [工具市场 v1 运行接线交接](tool-market-runtime-handoff.md)

<a id="historical"></a>

## 需要按历史范围阅读的专项记录

以下保留原基线、验证结果、限制及仍有效的局部合同；不是当前实例通过证明。

- [B1 定制源码交付（历史）](b1-native-conditional-delivery.md)
- [FV1：统一 Console 视觉与共享框架验收](console-fv1.md)
- [ImageAgent Organization scope admission](imageagent-organization-scope.md)
- [Organization Review invocation contract](imageagent-review-invocation.md)
- [347 Commercial read contract](issue347-commercial-read-contract.md)
- [Account UI: scope, design mapping and isolated verification](issue348-account-ui.md)
- [350 Console plans and entitlements](issue350-console-commercial-ui.md)
- [Issue #358 real-provider browser acceptance](issue358-real-provider-browser.md)

<a id="evidence"></a>

## 原始验证附件

截图、浏览器报告、数据库/HTTP 观察与受阻证据按原 Issue 保存，文件保持原件。

- [issue324](evidence/issue324)
- [issue328](evidence/issue328)
- [issue331](evidence/issue331)
- [issue334](evidence/issue334)
- [issue345](evidence/issue345)
- [issue348](evidence/issue348)
- [issue350](evidence/issue350)

原始附件不证明当前环境通过；真实模型、平台、部署、已有数据及进程操作仍由对应任务与授权决定。
