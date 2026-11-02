# 工程交接与验证记录

返回[文档首页](../README.md)。本目录保存实现交接、传输/消费者合同和有界验证记录。
产品与领域设计分别从[产品](../product/README.md)和[架构](../architecture/README.md)进入，
当前用户操作从[运行说明](../operations/README.md)进入。

同一文件可能包含当前合同与准确基线的历史结果。先读页首替代范围及未开放条件；
`IMPLEMENTATION_READY`、源码完成、实际实例启用和用户验收分别确认，不能从旧 SHA/地址推断当前运行状态。

## 开发规则与工程交接

- [代码健康规则与分类决定](code-health-decisions.md)：当前防回归入口及有范围的历史分类，不用旧探测计数推导当前删除权限。

- [1688 单图白底生成：架构与执行交接](2026-09-25-issue487-main-image-execution-design.md)
- [企业空间操作记录 v1](account-audit-v1.md)
- [Account read-only contract (#346 → #348)](account-readonly-contract.md)
- [系统管理员模块展示修复](administrator-module-projection.md)
- [Amazon 单商品插件交接](amazon-browser-capture-runtime-handoff.md)
- [硕米统一 Console 基础](console-foundation.md)
- [Current application: personal referrals server slice](current-application-referrals.md)
- [Data Services v1 runtime handoff](data-services-runtime-handoff.md)
- [企业存储容量 V1 试用入口与启动条件](enterprise-storage-capacity-runbook.md)
- [Issue 驱动派工规则 V1](issue-driven-development.md)
- [Issue #357 isolated official identity runtime](issue357-real-provider-runtime.md)
- [ACC-1 account profile and shared presentation](issue409-account-profile.md)
- [ACC-2 membership implementation contract](issue410-membership-contract.md)
- [成员原生验证附件](issue410-membership-native-evidence.json)
- [ACC-3: resources and optional SourceAccount management](issue411-account-resources.md)
- [Knowledge execution context and Product Agent consumer (#558 / #559)](knowledge-context-consumption.md)
- [官方知识库首版交接](official-knowledge-handoff.md)
- [运营驾驶舱首版交接](operations-cockpit-runtime-handoff.md)
- [任务中心标题提案待确认（#345）](product-title-review-ui.md)
- [Personal referral registration: Slice A](referral-registration.md)
- [个人报告保存与原离线包接线交接](report-center-runtime-handoff.md)
- [Shein 只读诊断 BFF 与隔离联调](shein-diagnostic-bff-fixture.md)
- [SHEIN 资料诊断页面与受控浏览器验收](shein-diagnostic-ui.md)
- [任务中心：本地资料准备工作记录](shein-records-ui.md)
- [店铺经营诊断源码交接](shop-operations-diagnosis-handoff.md)
- [SRC-2B1 public 1688 acquisition](src2b-public-acquisition.md)
- [店铺浏览器后端与固定角色展示组合](store-browser-permission-combination.md)
- [Store Center 商品与订单只读首版接线](store-observations-integration-handoff.md)
- [供应市场 v1 正常程序接线](supply-market-runtime-handoff.md)
- [工具市场 v1 运行接线交接](tool-market-runtime-handoff.md)
- [统一任务中心：业务任务、原生运行与待确认事项（#798）](unified-task-center.md)
- [官方 ZITADEL 适配与受保护成员管理边界](zitadel-official-provider-boundary.md)

## 需要按历史范围阅读的专项记录

- [B1 定制源码交付（历史）](b1-native-conditional-delivery.md)
- [FV1：统一 Console 视觉与共享框架验收](console-fv1.md)
- [ImageAgent Organization scope admission](imageagent-organization-scope.md)
- [Organization Review invocation contract](imageagent-review-invocation.md)
- [347 Commercial read contract](issue347-commercial-read-contract.md)
- [Account UI: scope, design mapping and isolated verification](issue348-account-ui.md)
- [350 Console plans and entitlements](issue350-console-commercial-ui.md)
- [Issue #358 real-provider browser acceptance](issue358-real-provider-browser.md)

## 原始验证附件

截图、浏览器报告、数据库/HTTP 观察与受阻证据按原 Issue 保存，文件保持原件：

- [issue324](evidence/issue324)
- [issue328](evidence/issue328)
- [issue331](evidence/issue331)
- [issue334](evidence/issue334)
- [issue345](evidence/issue345)
- [issue348](evidence/issue348)
- [issue350](evidence/issue350)

原始附件不证明当前环境通过；真实模型、平台、部署、已有数据及进程操作仍由对应任务与授权决定。
