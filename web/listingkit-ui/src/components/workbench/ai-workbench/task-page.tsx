"use client";

import { useState } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { ConsolePage, ConsoleState } from "../console/console-page";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
import { TaskDetail } from "./task-detail";
import { TaskBrowser } from "./task-browser";
import type { TaskMode } from "./task-presentation";
import {canDiscoverNativeRuns} from "./native-run-records";

export function BusinessTaskPage({ mode = "all", taskId, productReviewAvailable = false, sheinRecordsAvailable = false, nativeRunsAvailable = false, initialProposalId }: {
  mode?: TaskMode; taskId?: string; productReviewAvailable?: boolean; sheinRecordsAvailable?: boolean; nativeRunsAvailable?: boolean; initialProposalId?: string;
}) {
  const context = useWorkbenchContext();
  const scope = context.user && context.effectiveOrganization && !context.isLoading && !context.selectionRequired && !context.isSwitching && !context.error && !context.blockingError
    ? { userId: context.user.id, organizationId: context.effectiveOrganization.id } : null;
  const authorizationKey = JSON.stringify([context.roles, context.permissions]);
  const identity = scope ? JSON.stringify([scope, authorizationKey]) : "unavailable";
  const [entry, setEntry] = useState({ scope: scope ? identity : null as string | null, id: initialProposalId });
  if (scope && entry.scope !== identity) setEntry({ scope: identity, id: entry.scope === null ? entry.id : undefined });
  if (!scope && entry.scope !== null && entry.scope !== "unavailable") setEntry({ scope: "unavailable", id: undefined });
  if (!scope) return <ConsolePage title="任务中心"><ConsoleState kind={context.isLoading || context.isSwitching ? "loading" : "unavailable"} title="企业上下文不可用">请选择可访问的企业并登录。</ConsoleState></ConsolePage>;
  if (taskId) return <ConsolePage title="任务详情" breadcrumbs={findConsoleRoute("/workbench/ai/tasks")?.trail}>
    {context.aiWorkbenchAvailable ? <TaskDetail key={identity} scope={scope} authorizationKey={authorizationKey} taskId={taskId} /> : <ConsoleState kind="unavailable" title="暂未启用">当前应用未启用硕米 Chat 与业务任务。</ConsoleState>}
  </ConsolePage>;
  return <TaskBrowser key={identity + mode} scope={scope} authorizationKey={authorizationKey} mode={mode}
    nativeRunsAvailable={nativeRunsAvailable && canDiscoverNativeRuns(context.permissions)}
    tasksAvailable={context.aiWorkbenchAvailable} productReviewAvailable={productReviewAvailable} sheinRecordsAvailable={sheinRecordsAvailable}
    initialProposalId={entry.scope === identity ? entry.id : undefined} />;
}
