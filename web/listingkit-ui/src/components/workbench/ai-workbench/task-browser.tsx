"use client";

import Link from "next/link";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import { fetchProductTitleProposals } from "@/lib/api/product-title-review-client";
import { contentSelectionLabel } from "@/lib/contracts/content-selection";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
import { ConsolePage, ConsoleState } from "../console/console-page";
import { ProposalSession, ReviewError, reviewQueryPolicy, reviewAccessDenied } from "../task-center/review-session";
import { TaskDetail } from "./task-detail";
import { errorText, filters, taskState, reasonText, type TaskMode } from "./task-presentation";
import layout from "../task-center/task-center.module.css";
import styles from "./task-browser.module.css";

type Selection = { kind: "task" | "review"; id: string };
type Row = Selection & { title: string; caption: string; summary: string; state?: string; available: boolean };

export function TaskBrowser({ scope, authorizationKey, mode, tasksAvailable, productReviewAvailable, sheinRecordsAvailable, initialProposalId }: {
  scope: AIScope; authorizationKey: string; mode: TaskMode; tasksAvailable: boolean; productReviewAvailable: boolean; sheinRecordsAvailable: boolean; initialProposalId?: string;
}) {
  const context = useWorkbenchContext(), client = useQueryClient();
  const [selection, setSelection] = useState<Selection | undefined>(initialProposalId ? {kind:"review", id:initialProposalId} : undefined);
  const [search, setSearch] = useState("");
  const [reviewRef, setReviewRef] = useState<string>();
  const [pausedOnly, setPausedOnly] = useState(false);
  const [reviewLocked, setReviewLocked] = useState(false);
  const [taskLocked, setTaskLocked] = useState(false);
  const locked = reviewLocked || taskLocked;
  const [generation, setGeneration] = useState(0);
  const [reviewPosition, setReviewPosition] = useState({ cursors:[undefined] as (string | undefined)[], page:1 });
  const [accessError, setAccessError] = useState<unknown>();
  const detailHeading = useRef<HTMLHeadingElement>(null);
  const listHeading = useRef<HTMLHeadingElement>(null);
  const rowButtons = useRef(new Map<string, HTMLButtonElement>());
  const showReviews = productReviewAvailable && (mode === "all" || mode === "pending");
  const tasks = useInfiniteQuery({
    queryKey: ["ai-tasks", scope.userId, scope.organizationId, authorizationKey, mode, generation], enabled: tasksAvailable,
    initialPageParam: "", queryFn: ({signal, pageParam}) => requestAIWorkbench({route:"task-list", method:"GET", path:`tasks?limit=50${pageParam ? `&after=${pageParam}` : ""}`, scope, signal}),
    getNextPageParam: (last, _pages, lastParam) => last.next && last.next !== lastParam ? last.next : undefined,
    retry:false, staleTime:0, gcTime:0, refetchOnWindowFocus:false, refetchOnReconnect:false,
  });
  const reviews = useQuery({ ...reviewQueryPolicy,
    queryKey:["workbench", scope.userId, scope.organizationId, authorizationKey, "task-center-reviews", generation, reviewPosition],
    enabled:showReviews && !accessError,
    queryFn: ({signal}) => fetchProductTitleProposals({organizationId:scope.organizationId,limit:20,cursor:reviewPosition.cursors.at(-1),signal}),
  });
  if (reviewAccessDenied(reviews.error) && accessError !== reviews.error) {
    setAccessError(reviews.error); setReviewLocked(false);
    if (!taskLocked) setSelection(undefined);
  }
  const page = Array.from(new Map(tasks.data?.pages.flatMap(part => part.tasks ?? []).map(item => [item.id, item]) ?? []).values());
  const matchesMode = (state?: string) => pausedOnly ? state === "PAUSED" : mode === "all" ||
    mode === "pending" && state === "WAITING_CONFIRMATION" || mode === "running" && state === "RUNNING" ||
    mode === "completed" && state === "COMPLETED" || mode === "errors" && (state === "ERROR" || state === "PAUSED");
  const taskRows: Row[] = page.filter(item => matchesMode(item.state)).map(item => ({
    kind:"task",id:item.id,title:item.title,caption:item.productDetailsAvailable ? `${item.productKey ?? ""} · ${item.targetPlatform ?? ""}` : "商品详情暂不可用",
    summary:item.projectionAvailable ? reasonText[item.reason ?? ""] ?? item.goalSummary ?? "" : "状态暂不可用",
    state:item.state,available:item.projectionAvailable,
  }));
  // Correlation only removes a proposal when its task is actually in this visible page.
  const matchesSearch = (row: Row) => `${row.title} ${row.caption} ${row.summary}`.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase());
  const visibleTaskRows = taskRows.filter(matchesSearch);
  const linked = new Set(page.filter(item => visibleTaskRows.some(row => row.id === item.id)).map(item => item.reviewId).filter(Boolean));
  const reviewRows: Row[] = !showReviews || accessError || reviews.isError || reviews.isFetching || pausedOnly ? [] : (reviews.data?.items ?? []).filter(item => !linked.has(item.proposal_id)).map(item => ({
    kind:"review",id:item.proposal_id,title:"policy" in item && item.policy === "product-content-review-v1" ? "完善与本地化标准商品内容" : "调整标准商品标题",
    caption:item.product_key,summary:`${item.state === "accepted" ? "已接受，待应用" : "提案待审核"} · 基线 ${item.base_version} · 提案修订 ${item.proposal_revision}${"content_selection" in item ? ` · ${contentSelectionLabel(item.content_selection)}` : ""}`,
    state:"WAITING_CONFIRMATION",available:true,
  }));
  const rows = [...visibleTaskRows, ...reviewRows.filter(matchesSearch)];
  rows.sort((a,b) => Number(b.state === "WAITING_CONFIRMATION") - Number(a.state === "WAITING_CONFIRMATION"));
  const searching = tasksAvailable && mode !== "all" && !tasks.isPending && !tasks.isFetching && !tasks.isError && taskRows.length === 0 && tasks.hasNextPage;
  const fetchNextPage = tasks.fetchNextPage, loadedPages = tasks.data?.pages.length ?? 0;
  useEffect(() => { if (searching && !locked) void fetchNextPage(); }, [searching, locked, fetchNextPage, loadedPages]);
  const refetchReviews = reviews.refetch;
  const reviewChanged = useCallback(() => {
    void refetchReviews();
    void client.invalidateQueries({queryKey:["ai-task",scope.userId,scope.organizationId,authorizationKey]});
    void client.invalidateQueries({queryKey:["ai-tasks",scope.userId,scope.organizationId,authorizationKey]});
  }, [refetchReviews, client, scope.userId, scope.organizationId, authorizationKey]);
  // A partial Task projection cannot discard the Review owner's unknown intent.
  const observeReview = useCallback((id?: string) => { if (id && !locked) setReviewRef(id); }, [locked]);
  function select(next?: Selection) {
    if (locked) return;
    if (next) detailHeading.current?.focus();
    else (selection && rowButtons.current.get(selection.kind+selection.id) || listHeading.current)?.focus();
    setSelection(next); setReviewRef(next?.kind === "task" ? page.find(item=>item.id===next.id)?.reviewId : undefined);
    const url = new URL(window.location.href);
    if (next?.kind === "review") url.searchParams.set("proposal_id", next.id); else url.searchParams.delete("proposal_id");
    window.history.replaceState(null,"",url);
  }
  function refresh() {
    if (locked) return;
    select(); setGeneration(value=>value+1); setReviewPosition({cursors:[undefined],page:1}); setAccessError(undefined);
  }
  const renderReview = (id: string) => <ProposalSession key={id} proposalId={id} organizationId={scope.organizationId} scope={JSON.stringify([scope,authorizationKey])}
    userId={scope.userId} roles={context.roles} permissions={context.permissions} onLocked={setReviewLocked} interactionLocked={taskLocked}
    onDenied={error=>{setReviewLocked(false);setAccessError(error);if(!taskLocked)setSelection(undefined);}} onChanged={reviewChanged} />;
  const canChat = tasksAvailable && context.effectiveOrganization?.capabilities?.["workbench.chat.use"] === true && context.aiWorkbenchPlanningReadiness === "AVAILABLE";
  const tasksReady = !tasksAvailable || tasks.isSuccess && !tasks.isFetching && !tasks.isError && page.every(item=>item.projectionAvailable && item.state);
  const reviewsReady = !showReviews || reviews.isSuccess && !reviews.isFetching && !reviews.isError && !accessError;
  const pendingTasks = page.filter(item=>item.state === "WAITING_CONFIRMATION");
  const pendingLinked = new Set(pendingTasks.map(item=>item.reviewId).filter(Boolean));
  const pending = pendingTasks.length + (showReviews ? (reviews.data?.items ?? []).filter(item=>!pendingLinked.has(item.proposal_id)).length : 0);
  const metrics = [
    {label:"待你处理",state:"WAITING_CONFIRMATION",value:(tasksAvailable || showReviews) && tasksReady && reviewsReady ? pending : undefined},
    {label:"执行中",state:"RUNNING",value:tasksAvailable && tasksReady ? page.filter(item=>item.state==="RUNNING").length : undefined},
    {label:"已完成",state:"COMPLETED",value:tasksAvailable && tasksReady ? page.filter(item=>item.state==="COMPLETED").length : undefined},
  ];
  const available = tasksAvailable || showReviews || sheinRecordsAvailable && (mode === "all" || mode === "completed");
  let detail: ReactNode = <ConsoleState kind="empty" title="选择一条任务或待确认事项">查看当前状态、建议与依据，再决定下一步操作。</ConsoleState>;
  if (selection?.kind === "task" && tasksAvailable) detail = <><TaskDetail key={selection.id} scope={scope} authorizationKey={authorizationKey} taskId={selection.id} embedded interactionLocked={reviewLocked} onLocked={setTaskLocked} onReviewReference={productReviewAvailable ? observeReview : undefined} />{productReviewAvailable && reviewRef ? renderReview(reviewRef) : null}</>;
  if (selection?.kind === "review" && productReviewAvailable) detail = renderReview(selection.id);
  return <ConsolePage title="任务中心" description="查看正在执行、等待确认和已经完成的业务任务。" breadcrumbs={findConsoleRoute(filters.find(item=>item.mode===mode)!.href)?.trail}
    actions={<>{<input className={styles.search} type="search" aria-label="搜索已加载记录" placeholder="搜索任务或商品" value={search} disabled={locked} onChange={event=>setSearch(event.target.value)} />}
      {canChat ? <Button asChild><Link href="/workbench/ai/chat/new" prefetch={false} aria-disabled={locked || undefined} tabIndex={locked ? -1 : undefined} onClick={event=>{if(locked)event.preventDefault();}}>向硕米发起任务</Link></Button> : null}</>}>
    <div className={layout.metrics}>
      {metrics.map(metric=><Card className={layout.metric} key={metric.state}><span className={layout.swatch} aria-hidden="true" /><div><h2>{metric.label}</h2><p>已加载</p></div><strong className={styles.metricNumber} data-state={metric.state}>{metric.value ?? "—"}</strong></Card>)}
      <Card className={layout.scope}><p>工作范围</p><h2>当前企业</h2><p>本人业务任务 · 审核事项沿原业务权限</p></Card>
    </div>
    <div className={styles.toolbar}><nav className={layout.filters} aria-label="任务状态">
      {filters.filter(filter=>tasksAvailable || filter.mode==="all" || filter.mode==="pending" && productReviewAvailable || filter.mode==="completed" && sheinRecordsAvailable).map(filter=><Link key={filter.mode} href={filter.href} aria-current={!pausedOnly && mode===filter.mode ? "page":undefined} aria-disabled={locked || undefined} tabIndex={locked ? -1 : undefined} prefetch={false} onClick={event=>{if(locked)event.preventDefault();}}>{filter.label}</Link>)}
      {tasksAvailable ? <Button variant="outline" disabled={locked} aria-pressed={pausedOnly} onClick={()=>{select();setPausedOnly(value=>!value);}}>已暂停</Button> : null}
    </nav><Button variant="outline" disabled={locked} onClick={refresh}>刷新状态</Button></div>
    <p className={layout.coverage}>搜索和数量仅覆盖已加载记录；待确认事项优先显示。接受建议后仍需单独确认应用。</p>
    {!available ? <ConsoleState kind="unavailable" title="暂未启用">当前应用未启用硕米 Chat 与业务任务。</ConsoleState> :
    <div className={layout.columns}>
      <Card className={layout.list}><header className={layout.panelHeader}><h2 tabIndex={-1} ref={listHeading}>任务列表</h2><span>已显示 {rows.length} 项</span></header>
        {tasksAvailable && tasks.isPending ? <ConsoleState kind="loading" title={mode==="all" ? "正在读取任务" : "正在查找符合筛选的任务"} /> : null}
        {tasksAvailable && tasks.isError ? <ConsoleState kind="error" title={tasks.data ? "部分任务读取失败" : "任务读取失败"}>{errorText(tasks.error)} <Button variant="outline" disabled={locked} onClick={()=>void(tasks.isFetchNextPageError ? tasks.fetchNextPage() : tasks.refetch())}>重试</Button></ConsoleState> : null}
        {showReviews && (reviews.isPending || reviews.isFetching) && !accessError ? <ConsoleState kind="loading" title="正在读取标题提案">正在校验当前授权范围。</ConsoleState> : null}
        {accessError || showReviews && reviews.isError ? <ReviewError error={accessError ?? reviews.error} recover={refresh} /> : null}
        <ul className={layout.rows} aria-label="业务任务与待确认事项">{rows.map(row=><li key={row.kind+row.id}>
          <button ref={element=>{if(element)rowButtons.current.set(row.kind+row.id,element);else rowButtons.current.delete(row.kind+row.id);}} className={layout.row} disabled={locked} aria-pressed={selection?.kind===row.kind && selection.id===row.id} onClick={()=>select(row)}>
            <span className={layout.rowMarker+" "+styles.marker} data-state={row.state} aria-hidden="true" />
            <span className={layout.rowMain}><strong>{row.title}</strong><span>{row.caption} · {row.kind==="review" ? "待确认事项" : "业务任务"}</span><span>{row.summary}</span></span>
            <span className={styles.rowStatus}><span className={styles.badge} data-state={row.state}>{row.available ? taskState[row.state ?? ""] ?? "状态未知" : "状态暂不可用"}</span><span className={layout.rowAction}>查看详情 ›</span></span>
          </button>
        </li>)}</ul>
        {searching || tasks.isFetchingNextPage ? <ConsoleState kind="loading" title="正在查找符合筛选的任务" /> : null}
        {!rows.length && !searching && (!tasksAvailable || !tasks.isPending && !tasks.isError) && (!showReviews || reviews.isSuccess && !reviews.isFetching) ? <ConsoleState kind="empty" title={search ? "已加载记录中没有匹配项" : "当前筛选无任务"}>新任务会在确认方案后显示；独立建议按原审核权限读取。</ConsoleState> : null}
        {tasksAvailable && tasks.hasNextPage && !searching ? <Button variant="outline" disabled={locked || tasks.isFetching} onClick={()=>void tasks.fetchNextPage()}>加载更早任务</Button> : null}
        {showReviews && !accessError ? <div className={styles.pagination}><span>第 {reviewPosition.page} 页</span>
          <Button variant="outline" size="sm" disabled={locked || reviews.isFetching || reviewPosition.cursors.length<=1} onClick={()=>{select();setReviewPosition(old=>({cursors:old.cursors.slice(0,-1),page:old.page-1}));}}>上一页</Button>
          <Button variant="outline" size="sm" disabled={locked || reviews.isFetching || !reviews.data?.next_cursor} onClick={()=>{const cursor=reviews.data?.next_cursor;if(cursor){select();setReviewPosition(old=>({cursors:[...old.cursors,cursor].slice(-50),page:old.page+1}));}}}>下一页</Button>
          <p>提案每页最多 20 条，按提案标识排序；未读取的分页不计入数量。</p>
        </div> : null}
      </Card>
      <Card className={layout.detail} role="region" aria-label="任务详情"><header className={layout.panelHeader}><h2 tabIndex={-1} ref={detailHeading}>任务详情</h2>
        {selection ? <Button size="sm" variant="outline" disabled={locked} onClick={()=>select()}>返回列表</Button> : null}</header>{detail}</Card>
    </div>}
    {sheinRecordsAvailable && (mode==="all" || mode==="completed") ? <Card className={styles.history}><h2>历史已完成工作记录</h2><p>本地资料准备记录仍由原业务来源读取。</p><Button asChild variant="outline"><Link href="/workbench/ai/tasks/completed/history" prefetch={false} aria-disabled={locked || undefined} tabIndex={locked ? -1 : undefined} onClick={event=>{if(locked)event.preventDefault();}}>查看历史记录</Link></Button></Card> : null}
  </ConsolePage>;
}
