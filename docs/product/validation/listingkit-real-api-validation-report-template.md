> **状态：SUPERSEDED FOR CURRENT SOURCE / HISTORICAL TASK TEMPLATE（2026-10-20，#844）**
> 按[用户确认的 #844 冻结设计](https://github.com/qq550723504/task-processor/blob/ddc6414b0b93964d4de47d34d002ba990257efa1/docs/architecture/legacy-task-diagnostics-retirement-boundary.md)，旧数字 Task/Workspace 源码链已取消，本文不再是当前仓库或内部试用的 CURRENT 验收门禁。用户明确确认取消该旧流程，准确取代原 2026-09-26“必须切换等价模板才能历史化”的前置；不为已取消流程新建替代模板，也不将旧字段带入当前 BusinessTask。
> 以下模板字段/正文原样保留，用于追溯既有旧路径事实；远端旧实例是否仍存活为 UNKNOWN，不宣布服务停用、不撤销其证据或实际恢复责任。当前商业收费/外部准入/生产发布门禁及恢复 SOP 保持；历史记录/NOT_RUN 不等于当前验收通过。
> 本文不拥有产品 UI/IA 或长期架构权威；当前产品/领域 owner 与原安全合同保持。

# ListingKit 真实接口验收报告模板

> 用途：每轮真实接口联调都复制本模板生成一份 run 记录。建议文件名：`YYYY-MM-DD-<platform>-<task_id>.md`。

## 1. 基本信息

| 字段 | 内容 |
| --- | --- |
| Run ID |  |
| 日期 |  |
| 记录人 |  |
| 环境 | dev / staging / prod-like |
| 租户 |  |
| 店铺 |  |
| 目标平台 | SHEIN / Amazon / TEMU / Walmart |
| 来源类型 | SDS / 1688 / image_url / text / manual |
| task_id |  |
| batch_id |  |
| 结论 | pass / fail / blocked / partial |

## 2. 输入信息

### 来源素材

```text
粘贴来源 URL、SDS 商品 ID、图片 URL、描述文本或其他输入摘要。
```

### 创建任务请求摘要

```json
{
  "source": "",
  "target_platform": "",
  "store_id": "",
  "payload_summary": ""
}
```

### 创建任务响应摘要

```json
{
  "task_id": "",
  "status": "",
  "message": ""
}
```

## 3. 状态流转

| 时间 | 状态 | 页面 / 接口 | 说明 | 是否符合预期 |
| --- | --- | --- | --- | --- |
|  | pending / queued |  |  |  |
|  | processing / running |  |  |  |
|  | completed / needs_review |  |  |  |
|  | failed / error |  |  |  |

### 未知状态

| 状态值 | 来源接口 | 页面表现 | 处理结论 |
| --- | --- | --- | --- |
|  |  |  |  |

## 4. Workspace payload 验收

| 区域 | 是否有数据 | 问题 | 备注 |
| --- | --- | --- | --- |
| 商品事实 |  |  |  |
| 类目 |  |  |  |
| 普通属性 |  |  |  |
| 销售属性 |  |  |  |
| 图片 |  |  |  |
| 价格 |  |  |  |
| SKU / 变体 |  |  |  |
| 提交报告 |  |  |  |
| 历史版本 |  |  |  |

## 5. Readiness 验收

| blocker key | severity | domain | 页面展示 | 是否可跳转修复 | 备注 |
| --- | --- | --- | --- | --- | --- |
|  | blocker / warning / info |  |  |  |  |

### 未知 blocker

| blocker key / label | 来源接口 | 页面表现 | 是否阻断提交 | 后续处理 |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

## 6. 人工修复记录

| 时间 | 修复区域 | 修改内容 | 保存结果 | 备注 |
| --- | --- | --- | --- | --- |
|  | 类目 / 属性 / 图片 / 价格 / SKU |  | success / failed |  |

## 7. 提交验收

### 保存草稿

| 字段 | 内容 |
| --- | --- |
| action | save_draft |
| idempotency_key |  |
| attempt_id |  |
| 最终状态 | succeeded / failed / recovering |
| 失败 phase |  |
| 远端 draft id |  |
| 是否重复提交 | yes / no / unknown |

### 发布

| 字段 | 内容 |
| --- | --- |
| action | publish |
| idempotency_key |  |
| attempt_id |  |
| 最终状态 | succeeded / failed / recovering |
| 失败 phase |  |
| 远端 product / publish id |  |
| 是否重复提交 | yes / no / unknown |

### 提交阶段记录

| 时间 | phase | 状态 | 错误码 | 错误信息 | 是否可恢复 |
| --- | --- | --- | --- | --- | --- |
|  | validate |  |  |  |  |
|  | prepare_product |  |  |  |  |
|  | upload_images |  |  |  |  |
|  | pre_validate |  |  |  |  |
|  | submit_remote |  |  |  |  |
|  | persist_result |  |  |  |  |

## 8. 失败恢复

| 失败类型 | 用户看到什么 | 运营动作 | 是否恢复 | 是否需要工程介入 | 记录字段 |
| --- | --- | --- | --- | --- | --- |
|  |  |  | yes / no | yes / no | task_id / attempt_id / response / screenshot |

## 9. 证据附件

- 页面截图：
- 接口响应摘要：
- 日志关键字段：
- 远端平台记录：

## 10. 结论

```text
本轮是否通过：
主要问题：
必须关闭的问题：
可后续优化的问题：
是否允许进入下一轮：
```

## 11. Follow-up

| 优先级 | 问题 | owner | 截止时间 | 状态 |
| --- | --- | --- | --- | --- |
| P0 / P1 / P2 |  |  |  | open / closed |
