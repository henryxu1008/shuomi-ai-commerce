# 工程规则与接线

返回[文档首页](../README.md)。需求见[产品依据](../product/README.md)，事实与边界见[架构合同](../architecture/README.md)，安装和用户操作见[操作说明](../operations/README.md)。

本目录只保留当前执行规则和仍在使用的消费者接线说明；准确版本、CI 和具体环境结果在执行 Issue / PR 中维护。文档中的实现说明不表示任意实例已启用。

<a id="rules"></a>

## 执行规则

- [Issue 驱动派工](issue-driven-development.md)
- [代码健康规则](code-health-rules.md)

<a id="identity"></a>

## 身份与成员

- [企业空间操作记录 v1](account-audit.md)
- [Account read-only contract (#346 → #348)](account-readonly-contract.md)
- [系统管理员模块展示修复](administrator-module-projection.md)
- [Current application: personal referrals server slice](referrals-runtime.md)
- [ACC-1 account profile and shared presentation](account-profile.md)
- [ACC-2 membership implementation contract](account-membership-contract.md)
- [ACC-3: resources and optional SourceAccount management](account-resources.md)
- [Personal referral registration: Slice A](referral-registration.md)
- [店铺浏览器后端与固定角色展示组合](store-browser-permission-combination.md)
- [官方 ZITADEL 适配与受保护成员管理边界](zitadel-official-provider-boundary.md)

<a id="commerce"></a>

## 采集、商品与店铺

- [Amazon 单商品插件交接](amazon-browser-capture-runtime-handoff.md)
- [Shein 只读诊断 BFF 与隔离联调](shein-diagnostic-bff-fixture.md)
- [SHEIN 资料诊断页面与受控浏览器验收](shein-diagnostic-ui.md)
- [任务中心：本地资料准备工作记录](shein-records-ui.md)
- [SRC-2B1 public 1688 acquisition](1688-public-acquisition.md)
- [Store Center 商品与订单只读首版接线](store-observations-integration-handoff.md)
- [供应市场 v1 正常程序接线](supply-market-runtime-handoff.md)

<a id="workspace"></a>

## 工作台与 AI 工作

- [硕米统一 Console 基础](console-foundation.md)
- [Knowledge execution context and Product Agent consumer (#558 / #559)](knowledge-context-consumption.md)
- [官方知识库首版交接](official-knowledge-handoff.md)
- [运营驾驶舱首版交接](operations-cockpit-runtime-handoff.md)
- [个人报告 v1 接线交接](report-center-runtime-handoff.md)
- [店铺经营诊断源码交接](shop-operations-diagnosis-handoff.md)
- [统一任务中心：业务任务、原生运行与待确认事项（#798）](unified-task-center.md)

<a id="services"></a>

## 资源与服务

- [Data Services v1 runtime handoff](data-services-runtime-handoff.md)
- [企业存储容量 V1 试用入口与启动条件](enterprise-storage-capacity-runbook.md)
- [工具市场 v1 运行接线交接](tool-market-runtime-handoff.md)
