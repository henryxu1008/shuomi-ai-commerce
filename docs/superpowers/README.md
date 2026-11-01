# 设计合同与保留证据

返回[文档首页](../README.md)。本目录只保留仍有消费者的设计合同、尚未被撤销的准入基线，以及必要的真实/冻结证据。普通实施步骤、阶段scope/checkpoint、已完成修复和已替代版本链不再常驻当前目录。

当前开工依据以执行Issue、[产品决定](../product/README.md)、[架构索引](../architecture/README.md)及[AGENTS.md](../../AGENTS.md)为准。下列旧设计中的owner、版本和UI须按各自已有的准确替代说明读取；保留设计不代表实现、实例启用或产品验收。当前源码入口见[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。

## 仍有效的合同与设计依据

| 范围 | 保留入口 | 阅读边界 |
| --- | --- | --- |
| AI与Tool | [AI Capability & Agent Platform 设计](specs/2026-08-06-ai-capability-agent-platform-design.md)、[Provider-Neutral AI Contracts Design](specs/2026-08-21-provider-neutral-ai-contracts-design.md)、[Phase 2A 商业工具（Commerce Tool）基础设计](specs/2026-08-31-commerce-tool-foundation-design.md)、[B0 Canonical Product Inspection Tool 设计](specs/2026-09-05-commerce-tool-canonical-inspection-design.md) | 当前窄接口、注册表、权限和领域owner；已退宽接口/旧task读取不恢复。 |
| Product与离线资料 | [阶段三产品域目标架构设计](specs/2026-09-01-internal-target-architecture-phase3-product-design.md)、[SHEIN 离线诊断输入、freshness 与组织范围读取设计](specs/2026-09-05-shein-offline-validation-input-binding-design.md) | 当前Source/Catalog/Asset与保存时诊断合同；历史任务/读取owner按现行边界替代。 |
| 身份与企业上下文 | [Zitadel Authentication Runtime Extraction Design](specs/2026-08-25-zitadel-authruntime-design.md)、[硕米智能引擎工作台首期设计：店铺中心与 ZITADEL 多企业空间](specs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)、[硕米账号入口架构边界：ZITADEL Login V2 + Auth.js](specs/2026-09-03-shuomi-account-entry-zitadel-native-flow-design.md)、[硕米登录 Phase1：ZITADEL 原生能力收敛决策](specs/2026-09-05-shuomi-login-phase1-zitadel-native-simplification.md) | ZITADEL/Auth.js/verified identity和多企业职责；原生登录的后续产品决定优先。 |
| Source Account | [Source Account SA1 contract](specs/2026-09-09-source-account-sa1-contract.md)、[Source Account SA-2 BFF contract](specs/2026-09-09-source-account-sa2-bff-contract.md) | 有界SA1/SA2合同，不能由文档存在推断真实平台连接或迁移权限。 |
| 1688批量与定位 | [Issue #399 1688 `window.context` 定位规则设计（v1 提案）](specs/2026-09-20-issue399-1688-context-locator-design.md)、[1688 批量导入（本地执行器驱动既有浏览器采集链路）设计](specs/2026-09-21-1688-batch-local-agent-design.md) | 当前源码仍明确引用；原状态/未裁定项、匿名边界、收费及浏览器路径按准确后续决定读取。 |
| 图像下载安全 | [Image Download Safety Boundary Design](specs/2026-08-21-image-download-safety-design.md) | 保留HTTPS/SSRF/重定向/TLS/大小/取消约束；旧local/safeimagehttp路径不是当前调用入口，当前实现归integration/httpimage。 |
| ImageAgent事实与恢复 | [Image Agent Workflow Design](specs/2026-08-26-image-agent-workflow-design.md)、[Image Agent Recovery and Compatibility Design](specs/2026-08-27-image-agent-recovery-compatibility-design.md)、[Image Agent Budget Authorization Design](specs/2026-08-28-image-agent-budget-authorization-design.md) | 保留当前事实、预算、授权、原请求和UNKNOWN合同；已退Signal/旧版本兼容按页首精确退休边界处理。 |
| ImageAgent副作用 | [Image Agent V3 Effect Policy Design](specs/2026-08-29-image-agent-effect-policy-design.md)、[Image Agent Effect Terminalization Design](specs/2026-08-29-image-agent-effect-terminalization-design.md)、[Image Agent external-effect finalization design](specs/2026-08-29-image-agent-external-effect-finalization-design.md)、[Image Agent External Effect Recovery Workflow Design](specs/2026-08-29-image-agent-external-recovery-workflow-design.md)、[Image Agent Candidate Release Gate Design](specs/2026-08-29-image-agent-candidate-release-gate-design.md) | effect最终事实、未决effect不得终结、唯一恢复owner和候选隔离；不授权旧版本重放、发布或消费生产历史。 |

## 冻结合同与证据

- [保留计划与V7基线](plans/README.md)：两份仍承载合同的V7基线、现有测试保护的七份历史计划，以及三个被字节冻结架构正文引用的准确目标。不是新的实施队列。
- 冻结/取消依据：[硕米智能引擎第一阶段硬切产品与架构契约](specs/2026-09-02-shuomi-console-phase1-hard-cut-design.md)、[硕米企业资源账本与店铺服务生命周期设计](specs/2026-09-03-shuomi-resource-ledger-store-design.md)、[Issue 30 sourcing cutover](specs/2026-09-05-issue-30-sourcing-cutover.md)、[Source Account / 1688 Organization Ownership Cutover](specs/2026-09-05-source-account-organization-cutover.md). 保留现有guard和原文；旧迁移前置不恢复。
- 实际验证：[canonical subject](verification/2026-08-09-listingkit-canonical-zitadel-subject.md)、[owner reconciliation](verification/2026-08-10-listingkit-owner-reconciliation.md)。仅证明原基线的实际执行与限制，不作为当前命令或生产许可。

## 已退出文档的完整历史

完整原文、勾选、批准、失败和验证结果可在[cfd08cf54固定版本](https://github.com/henryxu1008/shuomi-ai-commerce/tree/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/superpowers)查阅，包括[原plans](https://github.com/henryxu1008/shuomi-ai-commerce/tree/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/superpowers/plans)、[原specs](https://github.com/henryxu1008/shuomi-ai-commerce/tree/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/superpowers/specs)和[根目录旧文档](https://github.com/henryxu1008/shuomi-ai-commerce/tree/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs)。每份去留及原Git blob记录在[文档清理Issue #1050](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1050)关联PR中，不在本页重复维护滚动审计台账。

清理只改变当前阅读入口及文档位置；当前产品、安全、权限、计费、持久化和恢复合同不因旧文件退出而撤销。不要为追溯而执行历史命令或恢复旧API、兼容层、迁移、双读双写或第二事实源。
