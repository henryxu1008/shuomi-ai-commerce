"use client";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConsoleState } from "../console/console-page";
import { requestAIWorkbench, AIWorkbenchError, type AIScope } from "@/lib/api/ai-workbench";
import { KnowledgeCitations } from "../task-center/knowledge-citations";
import { executionModeLabels } from "../agents/execution-mode-settings";
import { taskState, reasonText, errorText } from "./task-presentation";
import styles from "./task-page.module.css";

type TaskAction = "start" | "resume" | "review";
type PendingTaskAction = { key: string; revision: string; feedback: string };
const taskActionStorageKey = (scope: AIScope, taskId: string, action: TaskAction) =>
  `ai-workbench-task-action:${scope.userId}:${scope.organizationId}:${taskId}:${action}`;
function savedResumeFeedback(scope: AIScope, taskId: string): string {
  try {
    const saved = sessionStorage.getItem(taskActionStorageKey(scope, taskId, "resume"));
    if (saved) {
      const attempt = JSON.parse(saved) as PendingTaskAction;
      if (typeof attempt.feedback === "string") return attempt.feedback;
    }
  } catch { /* The request guard reports unavailable storage before dispatch. */ }
  return "";
}

export function TaskDetail({ scope, authorizationKey, taskId, onReviewReference, embedded = false, interactionLocked = false }: { scope: AIScope; authorizationKey: string; taskId: string; onReviewReference?: (id?: string) => void; embedded?: boolean; interactionLocked?: boolean }) {
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState(() => savedResumeFeedback(scope, taskId));
  const [error, setError] = useState("");
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const task = useQuery({ queryKey: ["ai-task", scope.userId, scope.organizationId, authorizationKey, taskId],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "task-read", method: "GET", path: `tasks/${taskId}`, scope, signal }),
    retry: false, staleTime: 0, refetchOnWindowFocus: false });
  const item = task.data?.task;
  useEffect(() => { if (task.isSuccess) onReviewReference?.(item?.reviewId); }, [task.isSuccess, item?.reviewId, onReviewReference]);
  const feedbackTooLarge = new TextEncoder().encode(feedback).byteLength > 8 * 1024;
  async function act(action: TaskAction) {
    if (!item) return; setPending(true); setError("");
    if (action === "resume" && feedbackTooLarge) {
      setError("反馈不能超过 8 KiB UTF-8 字节"); setPending(false); return;
    }
    const storageKey = taskActionStorageKey(scope, taskId, action);
    const revision = action === "resume" ? item.agentRevision ?? "" : "";
    const currentFeedback = action === "resume" ? feedback : "";
    let attempt: PendingTaskAction;
    try {
      const saved = sessionStorage.getItem(storageKey);
      attempt = saved ? JSON.parse(saved) as PendingTaskAction : { key: crypto.randomUUID(), revision, feedback: currentFeedback };
      if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(attempt.key) ||
          attempt.revision !== revision || attempt.feedback !== currentFeedback) {
        setError("上次操作的输入仍待核实；请先刷新原任务并用原输入重试"); setPending(false); return;
      }
      sessionStorage.setItem(storageKey, JSON.stringify(attempt));
    } catch { setError("无法保存操作键，暂不能安全提交"); setPending(false); return; }
    const controller = new AbortController(); abort.current = controller;
    try {
      await requestAIWorkbench({ route: `task-${action}`, method: "POST", path: `tasks/${taskId}/${action}`, scope,
        key: attempt.key, signal: controller.signal, ...(action === "resume" ? { body: { revision, feedback: currentFeedback } } : {}) });
      try { sessionStorage.removeItem(storageKey); } catch { /* The server receipt is terminal. */ }
      if (!controller.signal.aborted) await task.refetch();
    } catch (cause) {
      if (cause instanceof AIWorkbenchError && cause.code !== "TASK_OUTCOME_UNKNOWN" && cause.code !== "OUTCOME_UNKNOWN") {
        try { sessionStorage.removeItem(storageKey); } catch { /* The next request may replay the known result. */ }
      }
      if (!controller.signal.aborted) setError(errorText(cause));
    }
    finally { if (!controller.signal.aborted) setPending(false); abort.current = null; }
  }
  if (task.isPending) return <ConsoleState kind="loading" title="正在读取任务" />;
  if (task.isError || !item) return <ConsoleState kind="error" title="任务不可用">{errorText(task.error)} <Button variant="outline" onClick={() => void task.refetch()}>重试</Button></ConsoleState>;
  return <div className={styles.detail}>
    {!embedded ? <div className={styles.toolbar}><Button asChild variant="outline"><Link href="/workbench/ai/tasks" prefetch={false}>返回任务中心</Link></Button><Button variant="outline" onClick={() => void task.refetch()}>刷新状态</Button></div> : null}
    <Card className={styles.detailCard}><div className={styles.heading}><h2>{item.title}</h2><span className={styles.badge}>{item.projectionAvailable ? taskState[item.state ?? ""] ?? "状态未知" : "状态暂不可用"}</span></div>
      <p>{item.goalSummary}</p><dl><dt>当前状态</dt><dd>{item.projectionAvailable ? reasonText[item.reason ?? ""] ?? item.reason ?? "" : "Owner 暂不可用，无法确认当前状态"}</dd>
        <dt>企业</dt><dd>{scope.organizationId}</dd><dt>商品</dt><dd>{item.productDetailsAvailable ? item.productKey : "当前无权查看"}</dd>
        <dt>目标平台</dt><dd>{item.productDetailsAvailable ? item.targetPlatform : "不可用"}</dd>
        <dt>执行模型</dt><dd>{item.providerId && item.modelId ? `${item.providerId} / ${item.modelId}` : "尚无可用执行记录"}</dd>
        <dt>用量</dt><dd>{item.usageStatus === "unknown_reserved" ? "结果未知，额度仍保留" : item.usageStatus === "observed" ? `${item.tokens ?? 0} tokens · ${item.estimatedCostMicros ?? 0} μ${item.currency ?? ""}` : "尚无可用用量"}</dd>
        <dt>执行权限</dt><dd>{item.executionMode?`${executionModeLabels[item.executionMode]} · 版本 ${item.modeRevision}`:"权限模式不可用"}</dd><dt>审核</dt><dd>{item.policyAdoption?"按策略采用，未经逐项人工审阅":item.reviewState || "尚未生成"}</dd>
        <dt>创建时间</dt><dd>{new Date(item.createdAt).toLocaleString("zh-CN")}</dd></dl>
      <KnowledgeCitations knowledge={item.knowledge} />
      {item.productDetailsAvailable && item.operationId ? <Button asChild variant="outline"><Link href={`/workbench/supply/acquisition/operation/${item.operationId}`} prefetch={false}>查看来源商品</Link></Button> : null}
    </Card>
    {error ? <ConsoleState kind="error" title="操作未完成">{error}。请刷新原任务查看状态，不要新建任务。</ConsoleState> : null}
    <div className={styles.actions}>
      {item.canStart ? <Button disabled={pending || interactionLocked} onClick={() => void act("start")}>启动原任务</Button> : null}
      {item.canReconcile ? <Button variant="outline" disabled={pending || interactionLocked} onClick={() => void act("start")}>核实超期执行结果</Button> : null}
      {item.canResume ? <Card className={styles.actionCard}><label>继续执行的反馈<textarea rows={3} value={feedback} onChange={event => setFeedback(event.target.value)} /></label>
        {feedbackTooLarge ? <small>反馈不能超过 8 KiB UTF-8 字节。</small> : null}
        <Button disabled={pending || interactionLocked || !feedback.trim() || feedbackTooLarge} onClick={() => void act("resume")}>继续原任务</Button></Card> : null}
      {item.canReview && !item.reviewId ? <Button disabled={pending || interactionLocked} onClick={() => void act("review")}>提交标题供人工审核</Button> : null}
      {item.reviewId && !onReviewReference ? <Button asChild><Link href={`/workbench/ai/tasks/pending/other?proposal_id=${item.reviewId}`} prefetch={false}>查看并处理标题提案</Link></Button> : null}
    </div>
  </div>;
}
