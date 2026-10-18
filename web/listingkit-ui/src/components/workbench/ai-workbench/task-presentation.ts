import { AIWorkbenchError } from "@/lib/api/ai-workbench";

export type TaskMode = "all" | "running" | "pending" | "completed" | "errors";
export const filters: { mode: TaskMode; label: string; href: string }[] = [
  { mode: "all", label: "全部", href: "/workbench/ai/tasks" },
  { mode: "pending", label: "待确认", href: "/workbench/ai/tasks/pending" },
  { mode: "running", label: "执行中", href: "/workbench/ai/tasks/running" },
  { mode: "completed", label: "已完成", href: "/workbench/ai/tasks/completed" },
  { mode: "errors", label: "异常任务", href: "/workbench/ai/tasks/errors" },
];
export const taskState: Record<string, string> = { RUNNING: "执行中", WAITING_CONFIRMATION: "待确认", COMPLETED: "已完成", ERROR: "异常", PAUSED: "已暂停" };
export const reasonText: Record<string, string> = { EXECUTION_MODE_CHANGED:"权限模式或准入已变化，保留原任务并重新确认新运行", START_NOT_CLAIMED: "执行尚未启动，可使用原任务启动", EXECUTION_OUTCOME_UNKNOWN: "执行结果无法确认，不会自动重发模型请求",
  TITLE_PROFILE_UNAVAILABLE: "原任务使用的标题模型配置已不可用，请检查企业配置",
  HUMAN_REVIEW_REQUIRED: "需要人工审核标题建议", AGENT_RESULT_NOT_REVIEWABLE: "标题建议未通过验证，无法提交审核",
  AGENT_INTERRUPTED_DEADLINE_EXPIRED: "原执行期限已过，不能继续此中断任务", invalid_model_output: "模型返回格式不符合要求，已记录实际用量",
  pending: "提案待审核", accepted: "提案已接受，待应用", applied: "提案已应用", rejected: "提案已拒绝" };
export const errorText = (error: unknown) => error instanceof AIWorkbenchError ? ({
  TASK_OUTCOME_UNKNOWN: "操作结果暂无法确认。刷新原任务状态后，可用同一操作键重试查询原收据",
  OUTCOME_UNKNOWN: "网络结果暂无法确认。刷新原任务状态后，可用同一操作键重试查询原收据",
  REVISION_MISMATCH: "原任务状态已变化，请刷新后再操作",
} as Record<string, string>)[error.code] ?? `请求未完成：${error.code}` : "任务服务暂不可用";
