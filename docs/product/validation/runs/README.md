> **状态：SUPERSEDED — 旧数字 Task 验收指引（2026-10-20，#844）**
> 按[用户确认的 #844 冻结设计](https://github.com/qq550723504/task-processor/blob/ddc6414b0b93964d4de47d34d002ba990257efa1/docs/architecture/legacy-task-diagnostics-retirement-boundary.md)，旧 Task/Workspace、preview/readiness/submit 链及其专属诊断/回归工具已源码退休。后文使用方式、run 类型和最低验收集保留为旧路径历史说明，不是当前仓库或内部试用准入，也不要求新建等价模板或替代流程。
> 已有 dated runs、失败/阻塞/关闭证据保留；远端旧实例、任务和私有清单状态 UNKNOWN，本文不确认其停止，也不授权重放、删除或清理。当前商业收费/外部准入/生产发布门禁及实际旧环境恢复 SOP 保持；不能以源码退休宣称这些验收已通过。

# ListingKit 真实接口验收记录

本目录用于保存每轮真实接口联调和端到端验收报告。

## 命名规范

```text
YYYY-MM-DD-<platform>-<source>-<scenario>.md
```

示例：

```text
2026-06-16-shein-sds-save-draft-success.md
2026-06-16-shein-sds-publish-failed-recovery.md
2026-06-16-shein-1688-readiness-blocked.md
```

## 使用方式

1. 复制 [`../listingkit-real-api-validation-report-template.md`](../listingkit-real-api-validation-report-template.md)。
2. 填写真实 task_id、输入、状态流转、workspace payload、readiness、提交阶段、失败恢复和最终结论。
3. 每轮联调后必须更新 [`../unknown-state-and-blocker-tracker.md`](../unknown-state-and-blocker-tracker.md)。
4. 如果报告结论是 `blocked` 或 `fail`，必须在报告里写清关闭标准。

## Run 类型

| 类型 | 何时使用 | 结论要求 |
| --- | --- | --- |
| preflight | 真实店铺、token、SDS 登录态或远端 API 条件尚未齐备时，用来记录当前阻塞条件和下一步动作。 | 只能是 `blocked` 或 `partial`，不能替代真实成功 / 失败路径。 |
| success path | 真实任务从创建走到保存草稿或发布成功。 | 必须包含 task_id、提交 attempt、远端 id、关键接口响应摘要。 |
| failure path | 真实任务失败，并且 UI 能解释、恢复或明确需要工程介入。 | 必须包含失败 phase、错误响应、恢复动作和关闭标准。 |
| readiness path | 真实任务出现 readiness blocker，并完成修复或记录无法修复原因。 | 必须更新 unknown blocker 清单。 |

## 最低验收集

每个主要版本至少保留：

- 1 条真实环境 preflight 记录，用来确认店铺、token、SDS 登录态、AI、图片、对象存储和队列条件。
- 1 条 SHEIN SDS 保存草稿成功路径。
- 1 条 SHEIN SDS 发布成功路径。
- 1 条可恢复失败路径。
- 1 条 readiness blocked 修复路径。

## 已退休的回归数据集工具（历史）

旧 preview/readiness 清单校验器、专属 Pester 测试与占位示例已按 #844 退出源码。
其[历史脚本](https://github.com/qq550723504/task-processor/blob/4c5f5b16fa1a3afc05566b0f80538a05f99ab4e2/scripts/listingkit-regression-dataset-check.ps1)和
[历史示例](https://github.com/qq550723504/task-processor/blob/4c5f5b16fa1a3afc05566b0f80538a05f99ab4e2/docs/product/validation/listingkit-production-regression-dataset.example.json)
只供追溯，不是活动命令或当前验收入口。仓库外清单、token、既有校验结果和报告保持原 owner；本批未定位、读取或清理，不要求操作员补跑或迁移。

## 不能忽略的问题

以下问题不能只记录为备注，必须进入待关闭清单：

- 未知任务状态。
- 未知 readiness blocker key。
- 空错误响应。
- UI 无下一步动作。
- 重复点击导致重复远端提交。
- 保存草稿 / 发布状态和 SHEIN 远端结果不一致。
