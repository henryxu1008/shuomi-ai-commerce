"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useInfiniteQuery,useQuery,useQueryClient } from "@tanstack/react-query";
import { ArrowUp } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ConsolePage, ConsoleState, ConsoleToolbar } from "../console/console-page";
import { ChatIntroduction, NewConversationIntroduction } from "./chat-introduction";
import { ChatVoiceInput } from "./chat-voice-input";
import { requestAIWorkbench, type AIScope, AIWorkbenchError } from "@/lib/api/ai-workbench";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { WorkScopePicker, scopeLabel, scopeSelection } from "./work-scope-picker";
import { aiCreateBody, aiMessageBody, aiStoreRouteBody, type WorkScopeOption, type WorkScopeSelection, type AIConversation, type AIProposal } from "@/lib/contracts/ai-workbench";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
import {executionModeLabels,executionModeAdoptionText} from "../agents/execution-mode-settings";
import {ChatTaskSelection} from "./chat-task-selection";
import {ChatContextPicker} from "./chat-context-picker";
import {aiTaskSelection,type AITaskSubject} from "@/lib/contracts/ai-workbench";
import {contentSelectionLabel,type ContentSelection} from "@/lib/contracts/content-selection";
import styles from "./chat-page.module.css";

type ChatMode = "home" | "new" | "recent" | "saved" | "archived";
type MessageBody = ReturnType<typeof aiMessageBody.parse> | ReturnType<typeof aiStoreRouteBody.parse>;
type PendingMessage = { key: string; body: MessageBody; route?: "message"|"store-route"; draftId?: string; draftSlot?: "handoff" | "resolved"; owner?: AIScope; conversationId?: string };
type LocalDraft = { id:string; text:string };
function readLocalDraft(raw:string | null): LocalDraft | null { if(raw===null)return null; const value:unknown=JSON.parse(raw); if(!value || typeof value!=="object" || !("id" in value) || typeof value.id!=="string" || !isAcquisitionUUID(value.id) || !("text" in value) || typeof value.text!=="string") throw new Error("invalid draft"); return {id:value.id,text:value.text}; }
type ConfirmationReceipt = { key: string; taskId?: string };
const pendingMessageLegacyKey = (scope: AIScope, id: string) => `ai-workbench:pending-message:${scope.userId}:${scope.organizationId}:${id}`;
const pendingMessageStorageKey = (scope: AIScope, id: string) => `ai-workbench:pending-message-v2:${JSON.stringify([scope.userId,scope.organizationId,id])}`;
type PendingCreate = {key:string; body: {scopeSelection:WorkScopeSelection; scopeConfirmation:string} | {favorite?:boolean}; draft?:string; context?:string; conversationId?:string; owner?:AIScope};
const pendingCreateStorageKey = (scope: AIScope) => `ai-workbench:pending-create:${scope.userId}:${scope.organizationId}`;
const pendingComposeStorageKey = (scope: AIScope) => `ai-workbench:pending-create-v2:${JSON.stringify([scope.userId,scope.organizationId])}`;
const composeDraftKey = (scope: AIScope, context: string) => `ai-workbench:compose:${JSON.stringify([scope.userId,scope.organizationId,context])}`;
const conversationDraftKey = (scope: AIScope, id: string) => `ai-workbench:conversation-draft:${JSON.stringify([scope.userId,scope.organizationId,id])}`;
const resolvedDraftKey = (scope: AIScope, id: string) => `ai-workbench:resolved-draft:${JSON.stringify([scope.userId,scope.organizationId,id])}`;
const confirmStorageKey = (scope: AIScope, id: string) => `ai-workbench:confirm:${scope.userId}:${scope.organizationId}:${id}`;
const stateText: Record<string, string> = {
  OUTCOME_UNKNOWN: "结果暂无法确认，请重试以核实之前的操作。",
  PROPOSAL_STALE: "商品或会话已变化，请刷新并重新提出方案。",
  REVISION_MISMATCH: "会话内容已变化，请刷新后重试当前编辑。",
  DEPENDENCY_UNAVAILABLE: "服务暂不可用，请稍后重试。",
  FORBIDDEN: "当前企业或权限已变化，请重新加载。",
  CONVERSATION_ARCHIVED: "该会话已归档，不能继续发送消息。",
};
function errorText(error: unknown) { return error instanceof AIWorkbenchError ? stateText[error.code] ?? "请求未完成，请稍后重试。" : "请求暂不可用，请稍后重试。"; }

export function ChatPage({ mode, conversationId }: { mode: ChatMode | "detail"; conversationId?: string }) {
  const context = useWorkbenchContext();
  const route = findConsoleRoute(mode === "detail" ? "/workbench/ai/chat" : mode === "home" ? "/workbench/ai/chat" : `/workbench/ai/chat/${mode}`);
  const scope = context.user && context.effectiveOrganization && !context.selectionRequired && !context.isSwitching && !context.error && !context.blockingError
    ? { userId: context.user.id, organizationId: context.effectiveOrganization.id } : null;
  const canUse = context.effectiveOrganization?.capabilities?.["workbench.chat.use"] === true;
  const planningReadiness = context.aiWorkbenchPlanningReadiness;
  const titleReadiness = context.aiWorkbenchTitleReadiness;
  const canPlan = canUse && planningReadiness === "AVAILABLE";
  const authorizationKey = `${context.effectiveMemberId ?? ""}:${context.aiWorkbenchExecutionKind}:${context.aiWorkbenchExecutionReadiness}:${JSON.stringify([context.roles, context.permissions])}:${canUse}:${planningReadiness}:${context.aiWorkbenchStoreRouteReadiness ?? "UNAVAILABLE"}:${titleReadiness}`;
  const title = mode === "home" ? "硕米Chat" : mode === "new" ? "新建会话" : mode === "recent" ? "最近会话" : mode === "saved" ? "收藏会话" : mode === "archived" ? "归档会话" : "业务会话";
  const description = mode === "home" ? "发起新会话、继续历史工作、收藏重要讨论。" : mode === "new" ? "从已有商品与明确目标开始一项工作。" : mode === "recent" ? "选择一条会话，继续上次的交流。" : mode === "saved" ? "重要会话集中保存，打开后可继续查看与交流。" : mode === "archived" ? "查看已归档的讨论，按需恢复会话。" : "查看对话、补充需求，核对方案后再确认执行。";
  return <ConsolePage className="console-chat" title={title} breadcrumbs={route?.trail}
    description={mode === "new" && context.aiWorkbenchExecutionKind === "product.content.localize" ? undefined : description}>
    {!scope ? <ConsoleState kind={context.isSwitching || context.isLoading ? "loading" : "unavailable"} title="企业上下文不可用">请先选择可访问的企业并登录。</ConsoleState> :
      !context.aiWorkbenchAvailable ? <ConsoleState kind="unavailable" title="暂未启用">当前应用未启用硕米 Chat 与业务任务。</ConsoleState> :
      <ScopedChat key={`${scope.userId}:${scope.organizationId}:${authorizationKey}`} scope={scope} authorizationKey={authorizationKey} canUse={canUse} canPlan={canPlan} planningReadiness={planningReadiness} titleReadiness={titleReadiness} mode={mode} conversationId={conversationId} />}
  </ConsolePage>;
}

function ScopedChat({ scope, authorizationKey, canUse, canPlan, planningReadiness, titleReadiness, mode, conversationId }: { scope: AIScope; authorizationKey: string; canUse: boolean; canPlan: boolean; planningReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; titleReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; mode: ChatMode | "detail"; conversationId?: string }) {
  if (mode === "detail" && conversationId) return <ConversationDetail key={conversationId} scope={scope} authorizationKey={authorizationKey} canUse={canUse} canPlan={canPlan} planningReadiness={planningReadiness} titleReadiness={titleReadiness} id={conversationId} />;
  return <ChatCollection scope={scope} authorizationKey={authorizationKey} canUse={canUse} canPlan={canPlan} planningReadiness={planningReadiness} mode={mode === "detail" ? "home" : mode} />;
}

function planningStatus(readiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE") {
  return readiness === "NEEDS_CONFIGURATION" ? "当前企业的规划模型需要配置；可管理会话，模型发送待配置。" : "当前企业的任务规划尚未开放；可管理会话，模型发送待配置。";
}

function ChatCollection({ scope, authorizationKey, canUse, canPlan, planningReadiness, mode }: { scope: AIScope; authorizationKey: string; canUse: boolean; canPlan: boolean; planningReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; mode: ChatMode }) {
  const router = useRouter();
  const context = useWorkbenchContext();
  const search = useSearchParams();
  const selectedOperationId = search.get("operationId");
  const fixedScopeEnabled = context.aiWorkbenchExecutionKind !== "product.title.optimize";
  const directCompose = context.aiWorkbenchExecutionKind === "product.content.localize";
  const projectId = search.get("projectId"), storeId = search.get("storeId");
  const invalidContext = Boolean(projectId && storeId) || Boolean((projectId || storeId) && !directCompose);
  const scopeContext = projectId ? `project:${projectId}` : storeId ? `store:${storeId}` : "general";
  const draftKey = composeDraftKey(scope, scopeContext);
  const [compose, setCompose] = useState("");
  const composeInput = useRef<HTMLTextAreaElement>(null);
  const [voiceActive, setVoiceActive] = useState(false);
  const [draftReady, setDraftReady] = useState("");
  const [draftError, setDraftError] = useState("");
  const scopePreview = useQuery({queryKey:["ai-compose-scope",scope.userId,scope.organizationId,authorizationKey,scopeContext],enabled:mode === "new" && directCompose && !invalidContext,
    queryFn:({signal}) => {const query = new URLSearchParams(projectId ? {kind:"project",projectId} : storeId ? {kind:"store",storeId} : {kind:"general"}); return requestAIWorkbench({route:"scope-options",method:"GET",path:`chat/conversations/scopes?${query}`,scope,signal});},retry:false,staleTime:0,refetchOnWindowFocus:false});
  const candidateScope = scopePreview.data?.options.length === 1 && !scopePreview.data.incomplete ? scopePreview.data.options[0] : undefined;
  const resolvedScope = candidateScope && (projectId ? candidateScope.scope.kind === "project" && candidateScope.scope.projectId === projectId : storeId ? candidateScope.scope.kind === "store" && candidateScope.scope.storeId === storeId : candidateScope.scope.kind === "general") ? candidateScope : undefined;
  const [createIntent, setCreateIntent] = useState<PendingCreate>();
  const legacyCreate = Boolean(createIntent && (createIntent.context === undefined || createIntent.draft === undefined));
  const [pendingChecked, setPendingChecked] = useState("");
  const [pendingError, setPendingError] = useState("");
  const [selectedScope, setSelectedScope] = useState<WorkScopeOption>();
  const [scopeGeneration, setScopeGeneration] = useState(0);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [titleFilter, setTitleFilter] = useState("");
  const isCollection = mode === "recent" || mode === "saved" || mode === "archived";
  const abort = useRef<AbortController | null>(null);
  const createStorageKey = directCompose ? pendingComposeStorageKey(scope) : pendingCreateStorageKey(scope);
  const [activeCreateKey,setActiveCreateKey] = useState(createStorageKey);
  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (active) { try { const rawNew = sessionStorage.getItem(createStorageKey); const oldKey=pendingCreateStorageKey(scope); const rawOld=directCompose ? sessionStorage.getItem(oldKey) : null; if(rawNew!==null && rawOld!==null)throw new Error("multiple pending records");if(rawOld!==null && (scope.userId.includes(":") || scope.organizationId.includes(":")))throw new Error("ambiguous pending identity");const raw=rawOld ?? rawNew;const source=rawOld!==null?oldKey:createStorageKey;setActiveCreateKey(source);if(raw !== null) { const value = JSON.parse(raw) as PendingCreate; const body = aiCreateBody.safeParse(value.body); if (!isAcquisitionUUID(value.key) || !body.success || (value.draft === undefined) !== (value.context === undefined) || value.draft !== undefined && typeof value.draft !== "string" || value.context !== undefined && typeof value.context !== "string" || value.conversationId !== undefined && !isAcquisitionUUID(value.conversationId) || directCompose && source===createStorageKey && (value.owner?.userId!==scope.userId || value.owner?.organizationId!==scope.organizationId)) throw new Error("invalid pending create"); setCreateIntent({key:value.key, body:body.data,draft:value.draft,context:value.context,conversationId:value.conversationId,owner:value.owner}); } setPendingError(""); } catch { setPendingError("原创建请求记录不可读取或身份无法确认，不能开始新请求；请保留记录并联系管理员。"); } setPendingChecked(createStorageKey); } });
    return () => { active = false; };
  }, [createStorageKey,directCompose,scope.userId,scope.organizationId]);
  useEffect(() => { let active=true; queueMicrotask(()=>{ if(!active || !directCompose)return; try { setCompose(sessionStorage.getItem(draftKey) ?? ""); setDraftError(""); } catch { setDraftError("当前浏览器无法保存草稿，请检查浏览器存储设置。"); } setDraftReady(draftKey); }); return ()=>{active=false}; },[draftKey,directCompose]);
  useEffect(() => () => abort.current?.abort(), []);
  const conversations = useInfiniteQuery({ queryKey: ["ai-chat", scope.userId, scope.organizationId, authorizationKey, "list", mode], initialPageParam: "", enabled: isCollection,
    queryFn: ({ signal, pageParam }) => requestAIWorkbench({ route: "conversation-list", method: "GET", path: `chat/conversations?limit=50${mode === "saved" ? "&saved=true" : mode === "archived" ? "&archived=true" : ""}${pageParam ? `&after=${pageParam}` : ""}`, scope, signal }),
    getNextPageParam: page => page.next || undefined,
    retry: false, staleTime: 0, refetchOnWindowFocus: false });
  const visible = useMemo(() => {
    const seen = new Set<string>();
    return (conversations.data?.pages.flatMap(page => page.conversations) ?? []).filter(item => {
      if (seen.has(item.ID) || item.Archived !== (mode === "archived") || (mode === "saved" && !item.Favorite)) return false;
      seen.add(item.ID);
      return true;
    });
  }, [conversations.data, mode]);
  const matching = visible.filter(item => (item.Title || "未命名会话").toLocaleLowerCase().includes(titleFilter.trim().toLocaleLowerCase()));
  function updateCompose(value: string) {
    setCompose(value);
    try { sessionStorage.setItem(draftKey, value); setDraftError(""); }
    catch { setDraftError("当前浏览器无法保存草稿，请检查浏览器存储设置。"); }
  }
  function insertVoiceText(text: string) {
    const input = composeInput.current;
    if (!input) return;
    const position = input.selectionEnd;
    input.setRangeText(text, position, position, "end");
    // Synchronize React's controlled-input value tracker with the native edit.
    input.value = input.value;
    updateCompose(input.value);
  }
  async function create() {
    if (mode !== "new") {
      router.push(`/workbench/ai/chat/new${selectedOperationId && isAcquisitionUUID(selectedOperationId) ? `?operationId=${selectedOperationId}` : ""}`);
      return;
    }
    if (directCompose && (pendingChecked !== createStorageKey || pendingError || voiceActive)) return;
    const text = createIntent?.draft ?? compose;
    if (directCompose && !legacyCreate && (invalidContext || draftReady !== draftKey || draftError || !text.trim() || new TextEncoder().encode(text).length > 8192 || createIntent && createIntent.context !== scopeContext)) return;
    const selected = directCompose ? resolvedScope : selectedScope;
    if (creating || fixedScopeEnabled && !createIntent && !selected) return;
    const intent: PendingCreate = createIntent ?? {key:crypto.randomUUID(),body:selected ? {scopeSelection:scopeSelection(selected),scopeConfirmation:selected.confirmation} : {},...(directCompose ? {draft:compose,context:scopeContext,owner:scope} : {})};
    setCreateIntent(intent); setCreating(true); setError("");
    try { sessionStorage.setItem(activeCreateKey, JSON.stringify(intent)); } catch { if(directCompose){if(!createIntent)setCreateIntent(undefined);setCreating(false);setDraftError("无法保存待恢复的请求和需求，请检查浏览器存储设置。");return;} }
    if(directCompose && intent.conversationId && typeof intent.draft === "string") { try {const target=conversationDraftKey(scope,intent.conversationId),existing=readLocalDraft(sessionStorage.getItem(target));if(existing && existing.id!==intent.key){setError("会话中已有不同的未发送内容，请先打开原会话核对。");setCreating(false);return;}if(!existing)sessionStorage.setItem(target,JSON.stringify({id:intent.key,text:intent.draft}));router.push(`/workbench/ai/chat/${intent.conversationId}`);}catch{setError("无法保存需求，请检查浏览器存储设置并重试。");}setCreating(false);return; }
    const controller = new AbortController(); abort.current = controller;
    try {
      const result = await requestAIWorkbench({ route: "conversation-create", method: "POST", path: "chat/conversations", scope, key: intent.key, body: intent.body, signal: controller.signal });
      if (controller.signal.aborted) return;
      if(directCompose && typeof intent.draft === "string") { try { const bound={...intent,conversationId:result.conversation.ID};sessionStorage.setItem(activeCreateKey,JSON.stringify(bound));setCreateIntent(bound);const target=conversationDraftKey(scope,result.conversation.ID);const existing=readLocalDraft(sessionStorage.getItem(target));if(existing && existing.id!==intent.key){setError("会话中已有不同的未发送内容，请先打开原会话核对。");return;}if(!existing)sessionStorage.setItem(target,JSON.stringify({id:intent.key,text:intent.draft})); } catch { setError("会话已创建，但无法保存需求；请保持页面并重试原请求。"); return; } }
      if(directCompose && typeof intent.draft === "string"){router.push(`/workbench/ai/chat/${result.conversation.ID}${selectedOperationId && isAcquisitionUUID(selectedOperationId) ? `?operationId=${encodeURIComponent(selectedOperationId)}` : ""}`);return;}
      setCreateIntent(undefined);
      try { sessionStorage.removeItem(pendingCreateStorageKey(scope)); } catch { /* navigation still succeeds */ }
      router.push(`/workbench/ai/chat/${result.conversation.ID}${selectedOperationId && isAcquisitionUUID(selectedOperationId) ? `?operationId=${selectedOperationId}` : ""}`);
    } catch (cause) { if (!controller.signal.aborted) {
      if (cause instanceof AIWorkbenchError && cause.conversationAccepted === false && (cause.code === "NOT_FOUND" || cause.code === "PROPOSAL_STALE")) {
        try { sessionStorage.removeItem(activeCreateKey); } catch { setError("无法更新本地恢复记录，请检查浏览器存储设置。"); return; }
        setCreateIntent(undefined); setActiveCreateKey(createStorageKey); setSelectedScope(undefined); setScopeGeneration(n=>n+1);
        if(directCompose) void scopePreview.refetch();
        setError("原范围已变化或不可读，请重新选择工作范围。");
      } else setError(errorText(cause));
    } }
    finally { if (!controller.signal.aborted) setCreating(false); abort.current = null; }
  }
  return <>
    <nav className={`${styles.toolbar} ${styles.viewNavigation}`} aria-label="会话视图">{([ ["new", "新建会话"], ["recent", "最近会话"], ["saved", "收藏会话"], ["archived", "归档会话"] ] as const).map(([view,label]) => <Button asChild key={view} variant={mode === view ? "secondary" : "outline"}><Link href={`/workbench/ai/chat/${view}${selectedOperationId && isAcquisitionUUID(selectedOperationId) ? `?operationId=${encodeURIComponent(selectedOperationId)}` : ""}`} prefetch={false} aria-current={mode === view ? "page" : undefined}>{label}</Link></Button>)}</nav>
    {canUse && !canPlan ? <ConsoleState kind="unavailable" title="规划暂不可用">{planningStatus(planningReadiness)}</ConsoleState> : null}
    {mode === "home" ? <ChatIntroduction contentTasks={context.aiWorkbenchExecutionKind === "product.content.localize"} createAction={canUse ? <Button onClick={create} disabled={creating}>{creating ? "正在创建…" : "进入新建会话 →"}</Button> : <p>当前企业仅可查看已有会话。</p>} /> :
      mode === "new" ? <NewConversationIntroduction directCompose={directCompose} canUse={canUse} createAction={canUse ? directCompose ? <>
        <label className={styles.composeLabel}><span className={styles.visuallyHidden}>需求</span><Textarea ref={composeInput} rows={5} value={createIntent?.draft ?? compose} disabled={creating || !!createIntent || draftReady !== draftKey} onChange={event=>updateCompose(event.target.value)} placeholder="例如：根据已有商品资料整理英文描述" /></label>
        {projectId || storeId ? <p>工作背景：{createIntent ? "原请求待核对" : resolvedScope ? scopeLabel(resolvedScope) : "正在核对或无法读取"}</p> : null}
        {invalidContext ? <p role="status">当前工作背景无效，无法创建会话。</p> : null}
        {scopePreview.isError || scopePreview.isSuccess && !resolvedScope ? <p role="status">工作背景暂不可用。<Button variant="outline" onClick={()=>void scopePreview.refetch()}>重新读取</Button></p> : null}
        {legacyCreate ? <p role="status">有此前尚未确认的创建请求，请先核对原请求。</p> : createIntent && createIntent.context !== scopeContext ? <p role="status">有尚未确认的创建请求，请回到原入口继续核对。</p> : null}
        {draftError ? <p role="status">{draftError}</p> : null}
        {pendingError ? <p role="status">{pendingError}</p> : null}
        {new TextEncoder().encode(createIntent?.draft ?? compose).length > 8192 ? <p role="status">需求不能超过 8192 字节。</p> : null}
        <div className={styles.composeFooter}><div className={styles.voiceControl}><ChatVoiceInput identity={JSON.stringify([scope.userId,scope.organizationId,scopeContext])} disabled={creating || !!createIntent || draftReady !== draftKey || invalidContext} onText={insertVoiceText} onActiveChange={setVoiceActive} /></div><Button size="icon" onClick={create} aria-label={creating ? "正在创建" : createIntent ? "核对原创建请求" : "开始对话"} title={creating ? "正在创建" : createIntent ? "核对原创建请求" : "开始对话"} disabled={voiceActive || creating || pendingChecked !== createStorageKey || !!pendingError || (!legacyCreate && (invalidContext || draftReady !== draftKey || !!draftError || !(createIntent?.draft ?? compose).trim() || new TextEncoder().encode(createIntent?.draft ?? compose).length > 8192 || !!createIntent && createIntent.context !== scopeContext || !resolvedScope && !createIntent))}><ArrowUp size={18} aria-hidden="true" /></Button></div>
      </> : <>{fixedScopeEnabled ? <WorkScopePicker key={scopeGeneration} scope={scope} authorizationKey={authorizationKey} disabled={creating || !!createIntent} selected={selectedScope} onSelect={setSelectedScope} /> : null}<Button onClick={create} disabled={creating || fixedScopeEnabled && !selectedScope && !createIntent}>{creating ? "正在创建…" : createIntent ? "核对原创建请求" : "新建会话"}</Button></> : <p>当前企业仅可查看已有会话。</p>} /> :
      <ConsoleToolbar><div className={styles.collectionToolbar}><label className={styles.searchField}>筛选已加载的会话标题<Input type="search" value={titleFilter} onChange={event => setTitleFilter(event.target.value)} placeholder="输入会话标题" aria-describedby="chat-filter-scope" /></label>
      {titleFilter ? <Button variant="ghost" onClick={() => setTitleFilter("")}>清空筛选</Button> : null}
      <div className={styles.toolbar}>{canUse ? <Button onClick={create} disabled={creating}>{creating ? "正在创建…" : "新建会话"}</Button> : null}
      {mode === "archived" ? <Button asChild variant="outline"><Link href="/workbench/ai/chat/recent" prefetch={false}>返回最近会话</Link></Button> :
        mode === "recent" ? <Button asChild variant="outline"><Link href="/workbench/ai/chat/archived" prefetch={false}>查看归档会话</Link></Button> : null}
      <Button variant="outline" onClick={() => void conversations.refetch()}>刷新</Button></div></div><p id="chat-filter-scope" className={styles.filterNote}>仅筛选已加载的标题；可加载更早会话继续查找。</p></ConsoleToolbar>}
    {error ? <ConsoleState kind="error" title="操作未完成">{error}</ConsoleState> : null}
    {isCollection ? conversations.isPending ? <ConsoleState kind="loading" title="正在读取会话" /> : conversations.isError && !conversations.data ? <ConsoleState kind="error" title="会话读取失败">{errorText(conversations.error)} <Button variant="outline" onClick={() => void conversations.refetch()}>重试</Button></ConsoleState> :
      <section className={styles.list} aria-label="会话列表">
        {conversations.isError ? <ConsoleState kind="error" title="部分会话读取失败">{errorText(conversations.error)} <Button variant="outline" onClick={() => void (conversations.isFetchNextPageError ? conversations.fetchNextPage() : conversations.refetch())}>重试</Button></ConsoleState> : null}
        {visible.length ? matching.length ? <Card className={styles.conversationRows}>{matching.map(item => <ConversationLink key={item.ID} item={item} />)}</Card> : <ConsoleState kind="empty" title="已加载的会话中没有匹配标题">清空筛选或加载更早会话继续查找。</ConsoleState> : <ConsoleState kind="empty" title={mode === "saved" ? "暂无收藏会话" : mode === "archived" ? "暂无归档会话" : "暂无会话"}>{mode === "archived" ? "已归档会话会出现在这里，可打开详情恢复。" : "新建会话后，会保存到当前账号。"}</ConsoleState>}
        {conversations.hasNextPage ? <Button variant="outline" disabled={conversations.isFetchingNextPage} onClick={() => void conversations.fetchNextPage()}>加载更早会话</Button> : null}
        {mode === "saved" ? <p className={styles.filterNote}>取消收藏仅移出本列表，原始会话仍保留；归档会话可在归档列表查看。</p> : null}</section> : null}
  </>;
}

function ConversationLink({ item }: { item: AIConversation }) {
  return <Link className={styles.conversationLink} href={`/workbench/ai/chat/${item.ID}`} prefetch={false}>
    <span className={styles.conversationText}><strong>{item.Title || "未命名会话"}</strong><small>{item.Archived ? "已归档" : "继续查看会话"}{item.Favorite ? " · 收藏" : ""}</small></span><time dateTime={item.UpdatedAt}>{new Date(item.UpdatedAt).toLocaleString("zh-CN")}</time><span className={styles.openConversation}>打开 →</span>
  </Link>;
}

function ConversationDetail({ scope, authorizationKey, canUse, canPlan, planningReadiness, titleReadiness, id }: { scope: AIScope; authorizationKey: string; canUse: boolean; canPlan: boolean; planningReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; titleReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; id: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const search = useSearchParams();
  const [operationId, setOperationId] = useState(search.get("operationId") ?? "");
  const execution=useWorkbenchContext();
  const contentTasks=execution.aiWorkbenchExecutionKind==="product.content.localize";
  const canRoute=canUse && execution.aiWorkbenchStoreRouteReadiness==="AVAILABLE";
  const [subject,setSubject]=useState<AITaskSubject|null>(null);
  const [parameters,setParameters]=useState<ContentSelection>({targetLanguage:"en-US",description:true});
  const [platform, setPlatform] = useState<"shein" | "temu" | "amazon">("shein");
  const [content, setContent] = useState("");
  const [draftSaveError, setDraftSaveError] = useState("");
  const [draftHydrated, setDraftHydrated] = useState(false);
  const [draftRetry, setDraftRetry] = useState(0);
  const [deferredDraft, setDeferredDraft] = useState<LocalDraft | null>(null);
  const activeDraft = useRef<{id:string;slot:"handoff"|"resolved"} | null>(null);
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [knowledgeBaseId, setKnowledgeBaseId] = useState("");
  const [contextExpanded, setContextExpanded] = useState(false);
  const [contextReady, setContextReady] = useState(true);
  const [templateId, setTemplateId] = useState("");
  const [templateRevision, setTemplateRevision] = useState("");
  const [pendingMessage, setPendingMessage] = useState<PendingMessage | null>(null);
  const [messageChecked,setMessageChecked] = useState("");
  const [messageRecoveryError,setMessageRecoveryError] = useState("");
  const [confirmReceipts, setConfirmReceipts] = useState<Record<string, ConfirmationReceipt>>({});
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const overviewKey = `ai-workbench:store-overview:${JSON.stringify([scope.userId,scope.organizationId,id])}`;
  const [pendingOverview,setPendingOverview]=useState<string|null>(null);
  const [overviewReady,setOverviewReady]=useState(false);
  const [overviewBusy,setOverviewBusy]=useState(false);
  useEffect(()=>{let active=true;queueMicrotask(()=>{if(!active)return;try{const key=sessionStorage.getItem(overviewKey);if(key!==null&&!isAcquisitionUUID(key))throw new Error("invalid original key");setPendingOverview(key);setOverviewReady(true)}catch{setError("原店铺查询记录不可读取，请检查浏览器存储。");setOverviewReady(false)}});return()=>{active=false}},[overviewKey]);
  const messageStorageKey = pendingMessageStorageKey(scope, id);
  const [activeMessageKey,setActiveMessageKey] = useState(messageStorageKey);
  const confirmationStorageKey = confirmStorageKey(scope, id);
  const [storeProjectionFresh,setStoreProjectionFresh]=useState(false);
  const focusGeneration=useRef(0);
  const verifiedGeneration=useRef(-1);
  const pageRevalidation=useRef(false);
  const detail = useInfiniteQuery({ queryKey: ["ai-chat", scope.userId, scope.organizationId, authorizationKey, id], initialPageParam: "",
    queryFn: async ({ signal, pageParam }) => {const generation=focusGeneration.current;const result=await requestAIWorkbench({ route: "conversation-read", method: "GET", path: `chat/conversations/${id}?limit=50${pageParam ? `&before=${pageParam}` : ""}`, scope, signal });if(!signal.aborted)verifiedGeneration.current=generation;return result;},
    getNextPageParam: page => page.before || undefined,
    retry: false, staleTime: 0, refetchOnWindowFocus: false });
  const refetchDetail=detail.refetch;
  useEffect(()=>{if(detail.isSuccess&&!detail.isFetching&&!detail.isError&&!pageRevalidation.current&&verifiedGeneration.current===focusGeneration.current)setStoreProjectionFresh(true)},[detail.isSuccess,detail.isFetching,detail.isError,detail.dataUpdatedAt]);
  useEffect(()=>{
    const hide=()=>{focusGeneration.current++;setStoreProjectionFresh(false)};
    const restore=()=>{const generation=++focusGeneration.current;setStoreProjectionFresh(false);void (async()=>{await queryClient.cancelQueries({queryKey:["ai-chat",scope.userId,scope.organizationId,authorizationKey,id],exact:true});if(generation!==focusGeneration.current)return;const result=await refetchDetail();if(generation===focusGeneration.current && verifiedGeneration.current===generation && result.isSuccess)setStoreProjectionFresh(true)})();};
    window.addEventListener("blur",hide);window.addEventListener("focus",restore);
    return()=>{focusGeneration.current++;window.removeEventListener("blur",hide);window.removeEventListener("focus",restore)};
  },[refetchDetail,queryClient,scope.userId,scope.organizationId,authorizationKey,id]);
  const newest = detail.data?.pages[0];
  const current = newest?.conversation;
  const draftHandoff = useRef(false);
  const userEdited = useRef(false);
  useEffect(()=>{let active=true;queueMicrotask(()=>{if(!active || !newest || newest.scopeUnavailable || draftHandoff.current || messageChecked!==messageStorageKey || messageRecoveryError)return;try{const saved=readLocalDraft(sessionStorage.getItem(conversationDraftKey(scope,id)));const resolved=readLocalDraft(sessionStorage.getItem(resolvedDraftKey(scope,id)));const pendingRaw=sessionStorage.getItem(pendingComposeStorageKey(scope));if(saved && pendingRaw){const pending=JSON.parse(pendingRaw) as PendingCreate;if(pending.owner?.userId===scope.userId && pending.owner?.organizationId===scope.organizationId && pending.conversationId===id && pending.key===saved.id){sessionStorage.removeItem(pendingComposeStorageKey(scope));if(pending.context)sessionStorage.removeItem(composeDraftKey(scope,pending.context));}}const messageRaw=sessionStorage.getItem(activeMessageKey);if(messageRaw){const message=JSON.parse(messageRaw) as PendingMessage;if(saved && message.draftId!==saved.id)setDeferredDraft(saved);if(message.draftId && message.draftSlot)activeDraft.current={id:message.draftId,slot:message.draftSlot};}else if(resolved){activeDraft.current={id:resolved.id,slot:"resolved"};if(saved)setDeferredDraft(saved);if(!userEdited.current)setContent(resolved.text);}else if(saved){activeDraft.current={id:saved.id,slot:"handoff"};if(!userEdited.current)setContent(saved.text);}draftHandoff.current=true;setDraftHydrated(true);setDraftSaveError("");}catch{setDraftSaveError("本地草稿不可读取，请保留记录并检查浏览器存储。");}});return()=>{active=false};},[newest,messageStorageKey,messageChecked,messageRecoveryError,activeMessageKey,scope,id,draftRetry]);
  const messages = detail.data?.pages.slice().reverse().flatMap(page => page.messages) ?? [];
  const latestUser = newest?.messages.filter(m => m.Author === "USER").at(-1)?.Sequence ?? 0;
  const activeProposal = newest?.proposals.find(p => p.sourceSequence === latestUser);
  const abort = useRef<AbortController | null>(null);
  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (!active) return;
      try {
      const currentSaved = sessionStorage.getItem(messageStorageKey),legacyKey=pendingMessageLegacyKey(scope,id),legacySaved=sessionStorage.getItem(legacyKey);
      if(currentSaved!==null && legacySaved!==null)throw new Error("multiple pending messages");
      if(legacySaved!==null && (scope.userId.includes(":") || scope.organizationId.includes(":")))throw new Error("ambiguous pending identity");
      const saved=legacySaved ?? currentSaved,source=legacySaved!==null?legacyKey:messageStorageKey;
      setActiveMessageKey(source);
      if (saved===null) {setMessageRecoveryError("");setMessageChecked(messageStorageKey);return;}
      const value: unknown = JSON.parse(saved);
      if (!value || typeof value !== "object" || !("key" in value) || typeof value.key !== "string" || !isAcquisitionUUID(value.key) || !("body" in value)) throw new Error("invalid pending message");
      if(source===messageStorageKey && (!("owner" in value) || !value.owner || typeof value.owner!=="object" || !("userId" in value.owner) || value.owner.userId!==scope.userId || !("organizationId" in value.owner) || value.owner.organizationId!==scope.organizationId || !("conversationId" in value) || value.conversationId!==id)) throw new Error("invalid pending owner");
      const route = "route" in value && value.route === "store-route" ? "store-route" : "message";
      if ("route" in value && value.route !== "store-route" && value.route !== "message") throw new Error("invalid route");
      const checked = route === "store-route" ? aiStoreRouteBody.safeParse(value.body) : aiMessageBody.safeParse(value.body);
      if (!checked.success) throw new Error("invalid pending body");
      const draftId="draftId" in value && typeof value.draftId === "string" && isAcquisitionUUID(value.draftId) ? value.draftId : undefined;
      const draftSlot="draftSlot" in value && (value.draftSlot === "handoff" || value.draftSlot === "resolved") ? value.draftSlot : undefined;
      if (("draftId" in value || "draftSlot" in value) && (!draftId || !draftSlot)) throw new Error("invalid draft identity");
      setPendingMessage({ key: value.key, body: checked.data, route, draftId, draftSlot });
      setContent(checked.data.content);
      if(route === "message") { const normal=aiMessageBody.parse(value.body);
        if("taskSelection" in normal){setSubject(normal.taskSelection.subject);setParameters(normal.taskSelection.parameters)}else{setOperationId(normal.operationId);setPlatform(normal.targetPlatform)}
        setTemplateId(normal.templateId ?? "");setTemplateRevision(normal.templateRevision ?? "");setKnowledgeBaseId(normal.knowledgeBaseId ?? "");if(normal.templateId || normal.templateRevision || normal.knowledgeBaseId)setContextExpanded(true);
      }
      setMessageRecoveryError("");setMessageChecked(messageStorageKey);
      } catch { setMessageRecoveryError("原消息恢复记录不可读取或身份无法确认，请保留记录并联系管理员。");setMessageChecked(messageStorageKey); }
    });
    return () => { active = false; };
  }, [messageStorageKey,scope.userId,scope.organizationId,id]);
  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (!active) return;
      try {
      const raw = sessionStorage.getItem(confirmationStorageKey);
      if (!raw) return;
      const values: unknown = JSON.parse(raw);
      if (!values || typeof values !== "object" || Array.isArray(values)) return;
      const safe: Record<string, ConfirmationReceipt> = {};
      for (const [proposalId, receipt] of Object.entries(values).slice(-50)) {
        if (!isAcquisitionUUID(proposalId) || !receipt || typeof receipt !== "object" || !("key" in receipt) ||
          typeof receipt.key !== "string" || !isAcquisitionUUID(receipt.key)) continue;
        // Browser storage only recovers an unfinished operation key. A saved
        // task ID is not the owner's durable confirmation receipt.
        safe[proposalId] = { key: receipt.key };
      }
      setConfirmReceipts(safe);
      } catch { /* a missing browser store cannot authorize an operation */ }
    });
    return () => { active = false; };
  }, [confirmationStorageKey]);
  useEffect(() => () => abort.current?.abort(), []);
  async function explicitStoreOverview(){
    if(!overviewReady||overviewBusy)return;
    const key=pendingOverview ?? crypto.randomUUID();
    try{sessionStorage.setItem(overviewKey,key)}catch{setError("无法保存店铺查询恢复记录，请检查浏览器存储。");return}
    setPendingOverview(key);setOverviewBusy(true);setError("");
    try{await requestAIWorkbench({route:"store-overview",method:"POST",path:`chat/conversations/${id}/store-overview`,scope,key,body:{status:"ALL_NON_DELETED"}});sessionStorage.removeItem(overviewKey);setPendingOverview(null);await detail.refetch()}
    catch(cause){setError(errorText(cause))}finally{setOverviewBusy(false)}
  }
  async function loadEarlierMessages(){
    const generation=++focusGeneration.current;pageRevalidation.current=true;setStoreProjectionFresh(false);
    const older=await detail.fetchNextPage();
    if(older.isError || generation!==focusGeneration.current){pageRevalidation.current=false;return}
    const checked=await detail.refetch();
    pageRevalidation.current=false;
    if(checked.isSuccess && generation===focusGeneration.current && verifiedGeneration.current===generation)setStoreProjectionFresh(true);
  }
  function preserveMessageInput(saved:PendingMessage): boolean {
    try {
      if(saved.draftId && saved.draftSlot){const key=saved.draftSlot==="handoff"?conversationDraftKey(scope,id):resolvedDraftKey(scope,id),existing=readLocalDraft(sessionStorage.getItem(key));if(existing && existing.id===saved.draftId && existing.text===saved.body.content){activeDraft.current={id:existing.id,slot:saved.draftSlot};return true;}}
      const key=resolvedDraftKey(scope,id),existing=readLocalDraft(sessionStorage.getItem(key));
      if(existing && (existing.id!==saved.key || existing.text!==saved.body.content))return false;
      sessionStorage.setItem(key,JSON.stringify({id:saved.key,text:saved.body.content}));activeDraft.current={id:saved.key,slot:"resolved"};return true;
    }catch{return false;}
  }
  async function send() {
    if(messageChecked!==messageStorageKey || messageRecoveryError || !draftHydrated || draftSaveError && !pendingMessage)return;
    const intentRoute = pendingMessage?.route === "store-route" || !pendingMessage && current?.businessScope?.kind === "general" && !subject && (contentTasks || !isAcquisitionUUID(operationId));
    const draft = pendingMessage ?? { key: crypto.randomUUID(), body: intentRoute ? {content} : {
      content, ...(contentTasks?{taskSelection:{subject,capability:"product.content.localize",parameters}}:{operationId,targetPlatform:platform}),
      ...(knowledgeBaseId ? { knowledgeBaseId } : {}), ...(templateId ? { templateId, templateRevision } : {}),
    } };
    const checked = intentRoute ? aiStoreRouteBody.safeParse(draft.body) : aiMessageBody.safeParse(draft.body);
    if (!checked.success) { setError("请选择有效商品、语言和字段，填写需求及模板修订号。"); return; }
    const saved:PendingMessage = { key: draft.key, body: checked.data, route:intentRoute?"store-route":"message", owner:scope, conversationId:id, ...(pendingMessage?.draftId ? {draftId:pendingMessage.draftId,draftSlot:pendingMessage.draftSlot} : !pendingMessage && activeDraft.current ? {draftId:activeDraft.current.id,draftSlot:activeDraft.current.slot} : {}) };
    setPendingMessage(saved); setPending(true); setError("");
    try { sessionStorage.setItem(activeMessageKey, JSON.stringify(saved)); } catch { setPending(false);setDraftSaveError("无法保存消息恢复记录，请检查浏览器存储后重试。");return; }
    const controller = new AbortController(); abort.current = controller;
    try {
      const result = intentRoute ? await requestAIWorkbench({route:"store-route",method:"POST",path:`chat/conversations/${id}/intent`,scope,key:saved.key,signal:controller.signal,body:saved.body}) : await requestAIWorkbench({ route: "message", method: "POST", path: `chat/conversations/${id}/messages`, scope, key: saved.key, signal: controller.signal, body: saved.body });
      if (controller.signal.aborted) return;
      if ("scopeUnavailable" in result && result.scopeUnavailable) { setError("工作范围暂不可读。请核对当前权限。"); await detail.refetch(); return; }
      if (result.state === "COMPLETE" || result.state === "ROUTING_RESULT_LOST" || result.state === "FAILED_BEFORE_DISPATCH" || result.state === "PLANNER_INVALID_OUTPUT" || result.state === "PLANNER_UNKNOWN") {
        if(result.state!=="COMPLETE" && !preserveMessageInput(saved)){setDraftSaveError("无法保存原需求，请勿刷新，检查浏览器存储后重试。");return;}
        if(result.state==="COMPLETE" && saved.draftId && saved.draftSlot){try{const key=saved.draftSlot==="handoff"?conversationDraftKey(scope,id):resolvedDraftKey(scope,id),retained=readLocalDraft(sessionStorage.getItem(key));if(retained && retained.id===saved.draftId && retained.text===saved.body.content){sessionStorage.removeItem(key);activeDraft.current=null;}}catch{setDraftSaveError("无法完成草稿清理，请保留原消息并检查浏览器存储。");return;}}
        try { sessionStorage.removeItem(activeMessageKey); } catch { setDraftSaveError("无法更新消息恢复记录，请检查浏览器存储后重试。");return; }
        setPendingMessage(null);setActiveMessageKey(messageStorageKey);
        if (result.state === "COMPLETE") setContent("");
      }
      if (result.state === "FAILED_BEFORE_DISPATCH") setError(intentRoute?"请求尚未发送到模型。请检查当前企业的模型配置；原额度状态由账本继续核对，需求已保留。":"规划尚未发送到模型。请检查当前企业的模型配置或联系管理员；修复后可用保留的需求发送新消息。");
      if (result.state === "PLANNER_INVALID_OUTPUT") setError("模型返回的规划内容格式无效；本轮已产生 AI 用量，不会自动重发。可修改保留的需求后发送新消息。");
      if (result.state === "PLANNER_UNKNOWN") setError("模型结果无法确认；本轮不会自动重发。请刷新查看状态。");
      if (result.state === "ROUTING_RESULT_LOST") setError("识别结果未在期限内保存；已保留原需求，不会自动再次识别。");
      if (result.state === "READY_TO_DISPATCH") setError("本次请求仍在处理，请重试查看结果。");
      await detail.refetch();
    } catch (cause) { if (!controller.signal.aborted) {
      if(cause instanceof AIWorkbenchError&&cause.messageAccepted===false&&["INVALID_REQUEST","PROPOSAL_STALE"].includes(cause.code)){
        if(!preserveMessageInput(saved)){setDraftSaveError("无法保存原需求，请勿刷新，检查浏览器存储后重试。");return;}
        try{sessionStorage.removeItem(activeMessageKey)}catch{setDraftSaveError("无法更新消息恢复记录，请检查浏览器存储后重试。");return;}
        setPendingMessage(null);setActiveMessageKey(messageStorageKey);
      }
      setError(errorText(cause));
    } }
    finally { setPending(false); abort.current = null; }
  }
  async function confirm(proposal: AIProposal) {
    const key = confirmReceipts[proposal.id]?.key || crypto.randomUUID();
    const receipts = { ...confirmReceipts, [proposal.id]: { key } };
    setConfirmReceipts(receipts);
    try { sessionStorage.setItem(confirmStorageKey(scope, id), JSON.stringify(receipts)); } catch { /* retain the in-memory key */ }
    setPending(true); setError(""); const controller = new AbortController(); abort.current = controller;
    try {
      const receipt = await requestAIWorkbench({ route: "confirm", method: "POST",
        path: `chat/conversations/${id}/proposals/${proposal.id}/confirm`, scope, key, signal: controller.signal });
      if (controller.signal.aborted) return;
      const completed = { ...receipts, [proposal.id]: { key, taskId: receipt.task.id } };
      setConfirmReceipts(completed);
      try { sessionStorage.setItem(confirmStorageKey(scope, id), JSON.stringify(completed)); } catch { /* receipt remains in the Workbench owner */ }
      router.push(`/workbench/ai/tasks/${receipt.task.id}`);
    } catch (cause) { if (!controller.signal.aborted) setError(errorText(cause)); }
    finally { setPending(false); abort.current = null; }
  }
  async function change(property: { title?: string; favorite?: boolean; archived?: boolean }) {
    if (!current) return; setPending(true); setError("");
    try { await requestAIWorkbench({ route: "conversation-metadata", method: "PATCH", path: `chat/conversations/${id}`,
      scope, revision: current.MetadataRevision, body: property }); await detail.refetch(); setEditingTitle(false); }
    catch (cause) { setError(errorText(cause)); }
    finally { setPending(false); }
  }
  if (detail.isPending) return <ConsoleState kind="loading" title="正在读取会话" />;
  if (detail.isError || !current) return <ConsoleState kind="error" title="会话不可用">{errorText(detail.error)} <Button variant="outline" onClick={() => void detail.refetch()}>重试</Button></ConsoleState>;
  if (newest?.scopeUnavailable) return <ConsoleState kind="unavailable" title="工作范围暂不可读"><Button variant="outline" onClick={()=>void detail.refetch()}>重新读取</Button></ConsoleState>;
  if (messageRecoveryError) return <ConsoleState kind="error" title="消息恢复不可用">{messageRecoveryError}</ConsoleState>;
  return <div className={styles.detail}>
    <Card className={styles.conversationHeader}><div className={styles.conversationIdentity}><h2>{current.Title || "未命名会话"}</h2><div className={styles.tags}>{current.Favorite ? <Badge variant="neutral">已收藏</Badge> : null}{current.Archived ? <Badge variant="neutral">已归档</Badge> : null}<span>当前企业：{execution.effectiveOrganization?.name || "当前企业"}</span><span>{current.businessScope?.kind === "general" ? "通用业务" : current.businessScope?.kind === "project" ? "项目工作范围" : current.businessScope?.kind === "store" ? "店铺工作范围" : "未记录工作范围"}</span></div></div><div className={styles.toolbar}>
      <Button asChild variant="ghost"><Link href="/workbench/ai/chat/recent" prefetch={false}>返回最近会话</Link></Button>
      {canUse && !editingTitle ? <Button variant="outline" disabled={pending} onClick={() => { setTitleDraft(current.Title); setEditingTitle(true); }}>修改标题</Button> : null}
      {canUse ? <Button variant="outline" disabled={pending} onClick={() => void change({ favorite: !current.Favorite })}>{current.Favorite ? "取消收藏" : "收藏"}</Button> : null}
      {canUse ? <Button variant="outline" disabled={pending} onClick={() => void change({ archived: !current.Archived })}>{current.Archived ? "恢复会话" : "归档"}</Button> : null}
      <Button variant="outline" onClick={() => void detail.refetch()}>刷新</Button></div></Card>
    {canUse && editingTitle ? <Card className={styles.titleEditor}><label>会话标题 <Input value={titleDraft} disabled={pending} onChange={event => setTitleDraft(event.target.value)} /></label><div className={styles.toolbar}>
      <Button disabled={pending || new TextEncoder().encode(titleDraft).length > 256} onClick={() => void change({ title: titleDraft })}>保存标题</Button>
      <Button variant="outline" disabled={pending} onClick={() => setEditingTitle(false)}>取消</Button>
      {new TextEncoder().encode(titleDraft).length > 256 ? <span>标题不能超过 256 字节。</span> : null}</div></Card> : null}
    <Card className={styles.messagePanel}><h2>会话消息</h2><div className={styles.messages} aria-label="会话消息">{messages.length ? messages.map(item => <Card key={item.ID} className={item.Author === "USER" ? styles.userMessage : styles.assistantMessage}>
      <small>{item.Author === "USER" ? "我" : "硕米"} · {new Date(item.CreatedAt).toLocaleString("zh-CN")}</small><p>{item.Content}</p>
      {item.kind === "STORE_OVERVIEW_REF" ? !storeProjectionFresh || detail.isFetching || detail.isError ? <p>正在重新核对店铺权限…</p> : item.currentStoreOverview ? <div><p>当前可查看 {item.currentStoreOverview.total} 家店铺。</p><p>平台：{Object.entries(item.currentStoreOverview.platforms).map(([name,count])=>`${name} ${count} 家`).join("、") || "暂无"}。</p><p>状态：启用 {item.currentStoreOverview.statuses.active ?? 0} 家、停用 {item.currentStoreOverview.statuses.disabled ?? 0} 家、删除中 {item.currentStoreOverview.statuses.deleting ?? 0} 家。</p><p>经营类型未确认：{item.currentStoreOverview.operatingTypeUnknown} 家。读取开始时间：{new Date(item.currentStoreOverview.readStartedAt).toLocaleString("zh-CN")}</p><Link href="/workbench/stores">前往店铺中心</Link></div> : <p>当前无权查看或店铺数据暂不可用。</p> : null}</Card>) : <ConsoleState kind="empty" title="开始讨论">先选定已有商品，再描述任务目标。</ConsoleState>}
      {detail.hasNextPage ? <Button variant="outline" disabled={detail.isFetching} onClick={() => void loadEarlierMessages()}>{detail.isFetchingNextPage ? "正在加载…" : "加载更早消息"}</Button> : null}</div></Card>
    {canUse && !current.Archived && current.businessScope?.kind === "general" ? <Button variant="outline" disabled={!overviewReady||overviewBusy} onClick={()=>void explicitStoreOverview()}>{overviewBusy?"正在读取店铺概况…":pendingOverview?"核对原店铺查询":"查看店铺概况"}</Button> : null}
    {current.businessScope && current.businessScope.kind !== "general" ? <p>查看当前企业所有可见店铺，请进入通用业务会话。<Link href="/workbench/ai/chat/new">新建通用会话</Link></p> : null}
    {activeProposal ? <ProposalCard proposal={activeProposal} scope={scope} canUse={canUse} titleReadiness={titleReadiness} disabled={pending || current.Archived}
      confirmedTaskId={activeProposal.confirmedTaskId ?? confirmReceipts[activeProposal.id]?.taskId} onConfirm={() => void confirm(activeProposal)} /> : null}
    {error ? <ConsoleState kind="error" title="操作状态">{error} <Button variant="outline" onClick={() => void detail.refetch()}>刷新状态</Button></ConsoleState> : null}
    {!current.Archived && canUse && (canPlan || pendingMessage || canRoute && current.businessScope?.kind === "general") ? <Card className={styles.composer}><h2>{canRoute && current.businessScope?.kind === "general" && !subject?"向硕米提问":contentTasks?"讨论内容任务":"讨论标题建议"}</h2><p>{contentTasks?"选择商品及待处理内容，确认方案后启动内容 Agent。":"选择原标题任务对象，确认方案后启动标题 Agent。"}</p>
      {contentTasks&&canPlan?<ChatTaskSelection authorizationKey={authorizationKey} scope={scope} subject={subject} parameters={parameters} onSubject={setSubject} onParameters={setParameters} disabled={pending||Boolean(pendingMessage)}/>:null}
      {!contentTasks&&canPlan?<div className={styles.formGrid}><label>采集操作 ID<Input disabled={Boolean(pendingMessage)} value={operationId} onChange={e => setOperationId(e.target.value.trim())} placeholder="从已保存商品采集结果复制操作 ID" /></label>
        <label>目标平台<Select disabled={Boolean(pendingMessage)} value={platform} onChange={e => setPlatform(e.target.value as typeof platform)}><option value="shein">SHEIN</option><option value="temu">Temu</option><option value="amazon">Amazon</option></Select></label></div>:null}
      <label className={styles.promptLabel}>需求<Textarea disabled={Boolean(pendingMessage) || !draftHydrated} rows={4} value={content} onChange={e => {userEdited.current=true;setContent(e.target.value);try{if(activeDraft.current){const key=activeDraft.current.slot==="handoff"?conversationDraftKey(scope,id):resolvedDraftKey(scope,id);const currentDraft=readLocalDraft(sessionStorage.getItem(key));if(!currentDraft || currentDraft.id!==activeDraft.current.id)throw new Error("draft changed");sessionStorage.setItem(key,JSON.stringify({...currentDraft,text:e.target.value}));}setDraftSaveError("");}catch{setDraftSaveError("需求未能保存到本地，刷新会丢失修改；请检查浏览器存储。");}}} placeholder={contentTasks?"例如：请根据已保存资料，将描述和卖点整理成英文。":"例如：请优化这个商品的标题，突出已有证据支持的卖点。"} /></label>{draftSaveError ? <p role="status">{draftSaveError}</p> : null}
      <details className={styles.optionalContext} open={contextExpanded} onToggle={event => setContextExpanded(event.currentTarget.open)}><summary>模板与知识库（可选）</summary>
        <ChatContextPicker scope={scope} authorizationKey={authorizationKey} agentId={contentTasks?"product.content.agent":"product.title.agent"} target={contentTasks?"product":platform} enabled={contextExpanded||Boolean(templateId)||Boolean(knowledgeBaseId)} disabled={pending||Boolean(pendingMessage)} templateId={templateId} templateRevision={templateRevision} knowledgeBaseId={knowledgeBaseId} onTemplate={(id,revision)=>{setTemplateId(id);setTemplateRevision(revision)}} onKnowledge={setKnowledgeBaseId} onReady={setContextReady}/></details>
      {pendingMessage ? <p>上次消息结果待确认，暂时不能修改；重试时系统会先核实，尚未开始的请求可能在准备就绪后发送。</p> : null}
      <div className={styles.toolbar}><Button disabled={pending || !draftHydrated || messageChecked!==messageStorageKey || !!messageRecoveryError || !!draftSaveError && !pendingMessage || !pendingMessage && !contextReady || (!pendingMessage && current.businessScope?.kind === "general" && !subject && (contentTasks || !isAcquisitionUUID(operationId)) ? !canRoute : !pendingMessage&&(contentTasks?!aiTaskSelection.safeParse({subject,capability:"product.content.localize",parameters}).success:!isAcquisitionUUID(operationId))) || !content.trim() || Boolean(templateId) !== Boolean(templateRevision)} onClick={() => void send()}>{pending ? "处理中…" : pendingMessage ? "重试同一消息" : "发送消息"}</Button>
        <span>模型规划可能消耗 AI 额度；不会直接修改商品。</span></div></Card> : current.Archived ? <ConsoleState kind="unavailable" title="会话已归档">已保存的任务仍可从任务中心查看。</ConsoleState> :
      <><ConsoleState kind="unavailable" title="只读会话">{canUse ? planningStatus(planningReadiness) : "当前企业没有发送或确认权限，可查看已有对话和提案。"}</ConsoleState>{content ? <label className={styles.promptLabel}>保留的需求<Textarea rows={4} value={content} readOnly /></label> : null}</>}
    {!draftHydrated && draftSaveError ? <Button type="button" variant="outline" onClick={()=>setDraftRetry(value=>value+1)}>重新读取草稿</Button> : null}
    {deferredDraft ? <Card className={styles.composer}><h2>另有保留的需求</h2><p>{deferredDraft.text}</p><Button type="button" variant="outline" disabled={pending || !!pendingMessage || !!content.trim() || !canUse || !canPlan && !(canRoute && current.businessScope?.kind === "general")} onClick={()=>{try{const existing=readLocalDraft(sessionStorage.getItem(conversationDraftKey(scope,id)));if(!existing || existing.id!==deferredDraft.id)throw new Error("draft changed");const prior=activeDraft.current;if(prior?.slot==="resolved"){const old=readLocalDraft(sessionStorage.getItem(resolvedDraftKey(scope,id)));if(old?.id===prior.id && old.text==="")sessionStorage.removeItem(resolvedDraftKey(scope,id));}activeDraft.current={id:existing.id,slot:"handoff"};setContent(existing.text);setDeferredDraft(null);}catch{setDraftSaveError("无法读取保留的需求，请检查浏览器存储。");}}}>使用保留的需求</Button></Card> : null}
  </div>;
}

function ProposalCard({ proposal, scope, canUse, titleReadiness, disabled, confirmedTaskId, onConfirm }: { proposal: AIProposal; scope: AIScope; canUse: boolean; titleReadiness: "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE"; disabled: boolean; confirmedTaskId?: string; onConfirm: () => void }) {
  const execution=useWorkbenchContext();
  const schedules=useQuery({queryKey:["ai-schedule-availability",scope.userId,scope.organizationId,JSON.stringify([execution.roles,execution.permissions])],enabled:canUse&&proposal.kind==="product.content.localize"&&!proposal.knowledgeBaseId,queryFn:({signal})=>requestAIWorkbench({route:"schedule-availability",method:"GET",path:"schedules/availability",scope,signal}),retry:false,refetchOnWindowFocus:false});
  const content=proposal.kind==="product.content.localize";
  const readiness=content?(execution.aiWorkbenchExecutionKind===proposal.kind?execution.aiWorkbenchExecutionReadiness:"UNAVAILABLE"):titleReadiness;
  const ready=content?proposal.executionProfileReady:proposal.titleProfileReady;
  return <Card className={styles.proposal}><div className={styles.proposalHead}><span>{confirmedTaskId?"已确认的执行方案":"待确认的执行方案"}</span><strong>{content?"商品内容建议":"标题建议"}</strong></div>
    <p>{proposal.goalSummary}</p><dl><dt>企业</dt><dd>当前企业</dd>{proposal.businessScope ? <><dt>工作范围</dt><dd>{scopeLabel(proposal.businessScope)}</dd></> : null}<dt>商品</dt><dd>{!proposal.detailsAvailable ? "当前无权查看" : proposal.taskSelection?.subject.kind==="own_product" ? <Link href={`/workbench/data/mine/own/${proposal.taskSelection.subject.id}`}>查看所选商品</Link> : proposal.operationId ? <Link href={`/workbench/supply/acquisition/operation/${proposal.operationId}`}>查看所选商品</Link> : "已选择商品"}</dd>
      {content?<><dt>任务对象</dt><dd>{proposal.taskSelection?`${proposal.taskSelection.subject.kind==="own_product"?"自有商品":"源商品"} · 版本 ${proposal.taskSelection.subject.catalogVersion}`:"不可用"}</dd><dt>语言与字段</dt><dd>{proposal.taskSelection?contentSelectionLabel(proposal.taskSelection.parameters):"不可用"}</dd></>:<><dt>目标平台</dt><dd>{proposal.targetPlatform ?? "不可用"}</dd></>}<dt>模板</dt><dd>{proposal.templateId ? "已选择" : "未选择"}</dd>
      <dt>知识库</dt><dd>{proposal.knowledgeBaseId ? "已选择" : "未选择"}</dd><dt>执行模型</dt><dd>{proposal.providerId && proposal.modelId ? `${proposal.providerId} / ${proposal.modelId}` : "不可用"}</dd>
      <dt>执行权限</dt><dd>{proposal.executionMode?`${executionModeLabels[proposal.executionMode]} · 版本 ${proposal.modeRevision}`:"权限模式不可用，请重新提出方案"}</dd><dt>结果处理</dt><dd>{proposal.executionMode?executionModeAdoptionText[proposal.executionMode]:"原提案仍可查看，暂不能执行"}</dd></dl>
    {!confirmedTaskId && canUse && proposal.detailsAvailable && readiness !== "AVAILABLE" ? <p>执行模型需要配置；当前方案暂不能确认。</p> : null}
    {!confirmedTaskId && canUse && proposal.detailsAvailable && readiness === "AVAILABLE" && !ready ? <p>提案的执行配置已变化，请重新提出方案。</p> : null}
    {!confirmedTaskId && canUse && proposal.detailsAvailable && !proposal.executionAuthorized ? <p>{proposal.modeUnavailableReason==="EXECUTION_LIMITS_CHANGED"?"任务执行上限已变化，请重新提出方案。":"需要当前企业的执行权限；已有方案仍可查看。"}</p> : null}
    {confirmedTaskId ? <Button asChild><Link href={`/workbench/ai/tasks/${confirmedTaskId}`} prefetch={false}>查看已确认任务</Link></Button> :
      canUse ? <Button disabled={disabled || !proposal.detailsAvailable || readiness !== "AVAILABLE" || !ready || !proposal.executionAuthorized} onClick={onConfirm}>确认并创建任务</Button> : null}
    {canUse&&content&&!proposal.knowledgeBaseId&&schedules.data?.enabled?<Button asChild variant="outline"><Link href={`/workbench/ai/tasks?view=schedules&schedule_proposal=${proposal.id}`} prefetch={false} aria-disabled={disabled||!schedules.data.ready||!proposal.executionAuthorized||!ready||undefined} onClick={event=>{if(disabled||!schedules.data?.ready||!proposal.executionAuthorized||!ready)event.preventDefault();}}>设为定时任务</Link></Button>:null}
  </Card>;
}
