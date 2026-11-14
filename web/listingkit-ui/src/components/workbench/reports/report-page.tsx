"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Clock, FileText, FolderOpen, Star } from "lucide-react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConsolePage, ConsoleState } from "../console/console-page";
import { ReportError, readReportIntent, reportRequest, writeReportIntent } from "@/lib/api/report-center";
import { reportResultSchema, reportSchema, reportsPageSchema, reportsSummarySchema, reportText, sourceHref, type Report, type ReportIntent, type ReportScope, type ReportSource } from "@/lib/contracts/report-center";
import { ReportSourcePicker } from "./source-picker";
import { ResourceDialog } from "../resources/resource-dialog";
import {WorkbenchViewNav,workbenchViewPage} from "../console/workbench-view-nav";
import {initialReportMerge,mergeReportPage,type ReportMergeState} from "./report-list-merge";
import styles from "./report-page.module.css";
type View = "overview" | "recent" | "favorites" | "all";
const labels = { overview: "我的报告", recent: "最近报告", favorites: "收藏报告", all: "全部报告" };
const types = { TITLE_REVIEW: "标题审核", SHEIN_RECORD: "商品资料", SHEIN_TARGET_RECORD: "商品资料", STORE_DAILY_REPORT:"经营报告" };
const errorText = (e: unknown) => e instanceof ReportError ? ({ OUTCOME_UNKNOWN: "操作结果尚未确认，请重试上次操作，避免重复保存或收藏。", CONFLICT: "来源内容已变化，请重新选择后保存。", FORBIDDEN: "当前权限不足，权限恢复后可重试上次操作。", NOT_FOUND: "本人当前企业范围内未找到该来源或报告。", INTENT_STORAGE_UNAVAILABLE: "暂时无法保存操作状态，请检查浏览器设置。", ORGANIZATION_CONTEXT_CHANGED: "企业已切换，请返回原企业后继续。", IDENTITY_CONTEXT_CHANGED: "登录用户已变化，请重新登录后读取。" }[e.code] ?? "报告服务暂不可用，请稍后重试。") : "报告服务暂不可用，请稍后重试。";
const policy = { retry: false, gcTime: 0, staleTime: 0, refetchOnWindowFocus: false, refetchOnReconnect: false } as const;
// Store grants can change without changing the role-based context key. Never
// render a previous response while current report authorization is rechecked.
const readPolicy = { ...policy, refetchOnWindowFocus: "always", refetchOnReconnect: "always" } as const;
export function ReportPage({ view = "overview", titleReviewAvailable = false, sheinTargetSourcesAvailable = false, sheinOfflineSourcesAvailable=false, initialOfflineRecord, initialTargetRecord, initialReport }: { view?: View; titleReviewAvailable?: boolean; sheinTargetSourcesAvailable?: boolean; sheinOfflineSourcesAvailable?:boolean; initialOfflineRecord?:string; initialTargetRecord?: string;initialReport?:string }) {
  const context = useWorkbenchContext();
  const scope = context.user && context.effectiveOrganization && !context.selectionRequired && !context.isSwitching && !context.isLoading && !context.error && !context.blockingError ? { userId: context.user.id, organizationId: context.effectiveOrganization.id } : null;
  const authorizationKey = JSON.stringify([context.roles, context.permissions]);
  return <ConsolePage className={workbenchViewPage} title={labels[view]} description={view === "overview" ? "查找和整理保存在当前企业的报告。" : "查阅已保存的报告，收藏重要结果并下载使用。"} breadcrumbs={[{ label: "AI工作台" }, { label: "我的报告", href: "/workbench/ai/reports" }, ...(view === "overview" ? [] : [{ label: labels[view] }])]}>
    <WorkbenchViewNav className={styles.viewNavigation} label="报告视图">{(["overview", "recent", "favorites", "all"] as const).map(item => <Link key={item} href={item === "overview" ? "/workbench/ai/reports" : `/workbench/ai/reports/${item}`} prefetch={false} aria-current={view === item ? "page" : undefined}>{item === "overview" ? "报告概览" : labels[item]}</Link>)}</WorkbenchViewNav>
    {!scope ? <ConsoleState kind={context.isSwitching || context.isLoading ? "loading" : "unavailable"} title="企业上下文不可用">请选择可访问的企业并登录。</ConsoleState> : !context.permissions.includes("workbench.report.read") ? <ConsoleState kind="unavailable" title="报告尚不可用">当前企业尚未启用报告，或当前用户没有查看权限。</ConsoleState> : <ScopedReports key={JSON.stringify([scope.userId, scope.organizationId, authorizationKey, initialTargetRecord, initialOfflineRecord,initialReport,titleReviewAvailable, sheinTargetSourcesAvailable,sheinOfflineSourcesAvailable])} scope={scope} authorizationKey={authorizationKey} view={view} manage={context.permissions.includes("workbench.report.manage")} tasksAvailable={context.aiWorkbenchAvailable} titleReviewAvailable={titleReviewAvailable} targetAvailable={sheinTargetSourcesAvailable && context.permissions.includes("workbench.supply.read")} initialTargetRecord={initialTargetRecord} initialOfflineRecord={initialOfflineRecord} initialReport={initialReport} offlineAvailable={sheinOfflineSourcesAvailable&&context.permissions.includes("listingkit.admin.read")} />}
  </ConsolePage>;
}
function ScopedReports({ scope, authorizationKey, view, manage, tasksAvailable, titleReviewAvailable, targetAvailable, offlineAvailable, initialOfflineRecord, initialTargetRecord,initialReport }: { scope: ReportScope; authorizationKey: string; view: View; manage: boolean; tasksAvailable: boolean; titleReviewAvailable: boolean; targetAvailable: boolean; offlineAvailable:boolean; initialOfflineRecord?:string; initialTargetRecord?: string;initialReport?:string }) {
  const client = useQueryClient(), prefix = ["reports", scope.userId, scope.organizationId, authorizationKey];
  const [selected, setSelected] = useState<string|undefined>(initialReport), [showSave, setShowSave] = useState(!!initialTargetRecord && targetAvailable || !!initialOfflineRecord&&offlineAvailable), [intentOverride, setIntent] = useState<ReportIntent | null | undefined>(), [feedback, setFeedback] = useState("");
  const recovery = useQuery({ ...policy, queryKey: [...prefix, "intent"], queryFn: () => readReportIntent(scope), staleTime: Infinity });
  const intent = intentOverride === undefined ? recovery.data ?? null : intentOverride, intentReady = recovery.isSuccess;
  const notice = feedback || (recovery.isError ? errorText(recovery.error) : intent ? "上次操作结果尚未确认，请先重试上次操作。" : "");
  const summary = useQuery({ ...readPolicy, queryKey: [...prefix, "summary"], queryFn: ({ signal }) => reportRequest(scope, "/summary", reportsSummarySchema, signal) });
  const detail = useQuery({ ...readPolicy, queryKey: [...prefix, "detail", selected], enabled: !!selected, queryFn: ({ signal }) => reportRequest(scope, `/${selected}`, reportSchema, signal) });
  const mutation = useMutation({ mutationKey: [...prefix, "write"], mutationFn: (command: ReportIntent) => reportRequest(scope, command.operation === "save" ? "" : `/${command.id}/favorite`, reportResultSchema, undefined, command), onSuccess: (result) => {
    try { writeReportIntent(scope, null); setIntent(null); setSelected(result.report.id); setShowSave(false); setFeedback(result.replayed ? "已找回上次保存的报告，收藏显示当前状态。" : "操作已保存。"); void client.invalidateQueries({ queryKey: prefix }); } catch (e) { setFeedback(errorText(e)); }
  }, onError: (e) => { setFeedback(errorText(e)); if (e instanceof ReportError && ["CONFLICT", "INVALID_REQUEST", "NOT_FOUND"].includes(e.code)) { try { writeReportIntent(scope, null); setIntent(null); } catch (storageError) { setFeedback(errorText(storageError)); } } } });
  function submit(command: ReportIntent) { if (!manage || mutation.isPending || intent || !intentReady) return; try { writeReportIntent(scope, command); setIntent(command); setFeedback(""); mutation.mutate(command); } catch (e) { setFeedback(errorText(e)); } }
  const busy = mutation.isPending || !!intent || !intentReady;
  const showPicker = showSave && manage && (titleReviewAvailable || targetAvailable || offlineAvailable);
  const previousPicker = useRef(false), feedbackRef = useRef<HTMLDivElement>(null), scopeRef = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    if (previousPicker.current && !showPicker && busy) (feedbackRef.current ?? scopeRef.current)?.focus();
    previousPicker.current = !!showPicker;
  }, [showPicker, busy]);
  const feedbackNotice = notice ? <div ref={feedbackRef} tabIndex={-1} className={styles.feedback} role="status"><p>{notice}</p>{intent && !mutation.isPending && manage ? <Button variant="outline" onClick={() => mutation.mutate(intent)}>重试上次操作</Button> : null}</div> : null;
  return <>
    <div className={styles.reportActions}>{manage ? <Button aria-haspopup="dialog" onClick={() => setShowSave(true)} disabled={busy || !(titleReviewAvailable || targetAvailable || offlineAvailable)}>添加已有结果</Button> : null}<span ref={scopeRef} tabIndex={-1}>当前企业 · 本人的报告</span></div>
    {manage && !titleReviewAvailable ? <p>标题审核报告保存暂未开放；已保存的历史报告仍可查阅、收藏和下载。</p> : null}
    {!showPicker ? feedbackNotice : null}
    {showPicker ? <ResourceDialog title="添加已有结果" locked={mutation.isPending} onClose={() => setShowSave(false)}>{feedbackNotice}<ReportSourcePicker scope={scope} authorizationKey={authorizationKey} tasksAvailable={tasksAvailable} titleReviewAvailable={titleReviewAvailable} targetAvailable={targetAvailable} initialTargetRecord={initialTargetRecord} initialOfflineRecord={initialOfflineRecord} offlineAvailable={offlineAvailable} disabled={busy} save={(source: ReportSource) => submit({ operation: "save", key: crypto.randomUUID(), source: source.ref })} /></ResourceDialog> : null}
    {view === "overview" ? <>
      {summary.isPending || summary.isFetching ? <ConsoleState kind="loading" title="正在读取报告汇总" /> : summary.isError ? <ConsoleState kind="error" title="汇总读取失败"><p>{errorText(summary.error)}</p><Button variant="outline" onClick={() => void summary.refetch()}>重试</Button></ConsoleState> : <div className={styles.summary}><strong>已保存 {summary.data.saved} 份报告</strong><span>近30天 {summary.data.recent} 份</span><span>收藏 {summary.data.favorites} 份</span><span>涉及 {summary.data.stores} 个店铺</span><span>项目关联暂未开放</span></div>}
      <div className={styles.cards}>{([
        { view: "recent", Icon: Clock, description: "快速查阅最近30天保存的成果", uses: ["查看最近保存的报告", "检查保存时版本与来源", "下载后继续使用"], value: summary.data?.recent, note: "按保存时间展示，保留当时状态" },
        { view: "favorites", Icon: Star, description: "沉淀常用结果，方便再次查阅", uses: ["收藏重要报告", "集中查阅个人常用结果", "取消不再需要的收藏"], value: summary.data?.favorites, note: "本人收藏，只在当前企业可见" },
        { view: "all", Icon: FolderOpen, description: "查找本人已保存的历史报告", uses: ["按类型和商品查找", "查看只读内容与原始来源", "下载报告"], value: summary.data?.saved, note: "保存报告不会执行商品操作" },
      ] as const).map(card => <Card className={styles.feature} data-tone={card.view} key={card.view}><card.Icon size={30} aria-hidden="true" /><h2>{labels[card.view]}</h2><p>{card.description}</p><h3>主要用途</h3><ul>{card.uses.map(use => <li key={use}>{use}</li>)}</ul><div className={styles.usage}><span>已保存</span><strong>{summary.isSuccess && !summary.isFetching ? `${card.value} 份` : "—"}</strong><p>{card.note}</p></div><Button asChild><Link href={`/workbench/ai/reports/${card.view}`} prefetch={false}>进入{labels[card.view]}</Link></Button></Card>)}</div>
    </> : <ReportList scope={scope} prefix={prefix} view={view} open={setSelected} productSaveAvailable={manage&&(targetAvailable||offlineAvailable)} canManage={manage} saveAvailable={manage&&(titleReviewAvailable||targetAvailable||offlineAvailable)} />}
    {selected ? <Card className={styles.detail} role="region" aria-label="报告详情"><header><h2>只读报告详情</h2><Button variant="ghost" onClick={() => setSelected(undefined)}>关闭详情</Button></header>{detail.isPending || detail.isFetching ? <ConsoleState kind="loading" title="正在读取报告" /> : detail.isError ? <ConsoleState kind="error" title="报告读取失败"><p>{errorText(detail.error)}</p><Button variant="outline" onClick={() => void detail.refetch()}>重试</Button></ConsoleState> : <ReportDetail scope={scope} report={detail.data} manage={manage} busy={busy} favorite={() => submit({ operation: "favorite", key: crypto.randomUUID(), id: detail.data.id, favorite: !detail.data.favorite })} />}</Card> : null}
  </>;
}
type ListCursor = {kind:"single";cursor:string} | {kind:"merged";state:ReportMergeState};
function ReportList({ scope, prefix, view, open, productSaveAvailable, canManage, saveAvailable }: { scope: ReportScope; prefix: string[]; view: Exclude<View, "overview">; open: (id: string) => void; productSaveAvailable:boolean; canManage:boolean; saveAvailable:boolean }) {
  const [purpose, setPurpose] = useState<"all"|"title"|"product"|"operating">("all"), [search, setSearch] = useState(""), [query, setQuery] = useState("");
  const initial:ListCursor = purpose === "product" ? {kind:"merged",state:initialReportMerge()} : {kind:"single",cursor:""};
  const reports = useInfiniteQuery({ ...readPolicy, queryKey: [...prefix, "list", view, purpose, query], initialPageParam:initial, queryFn: async ({ signal, pageParam }) => {
    const fetchPage=(kind:string,cursor:string)=>{const params=new URLSearchParams({view,limit:"20"});if(kind)params.set("kind",kind);if(query)params.set("search",query);if(cursor)params.set("cursor",cursor);return reportRequest(scope,`?${params}`,reportsPageSchema,signal);};
    if(pageParam.kind==="merged") {const result=await mergeReportPage(pageParam.state,fetchPage);return {items:result.items,next:result.next?{kind:"merged",state:result.next} as ListCursor:undefined};}
    const result=await fetchPage(purpose==="title"?"TITLE_REVIEW":purpose==="operating"?"STORE_DAILY_REPORT":"",pageParam.cursor);
    return {items:result.items,next:result.nextCursor&&result.nextCursor!==pageParam.cursor?{kind:"single",cursor:result.nextCursor} as ListCursor:undefined};
  }, getNextPageParam: last => last.next });
  const [loadingMore, setLoadingMore] = useState(false);
  async function loadMore() {
    if (loadingMore || reports.isFetching) return;
    setLoadingMore(true);
    try {
      // A later page can be authorized after a store grant changed. Rebuild
      // prior pages first, using fresh cursors, rather than append to stale
      // report titles that the caller may no longer be allowed to see.
      const current = await reports.refetch();
      if (current.isSuccess && current.hasNextPage) await reports.fetchNextPage();
    } finally { setLoadingMore(false); }
  }
  const items = reports.data?.pages.flatMap(p => p.items) ?? [];
  return <>
    <div className={styles.toolbar}><div className={styles.purposes} role="group" aria-label="报告用途">{([["all","全部"],["title","标题审核"],["product","商品资料"],["operating","经营报告"]] as const).map(([value,label])=><Button key={value} variant={value===purpose?"secondary":"outline"} aria-pressed={purpose===value} onClick={()=>setPurpose(value)}>{label}</Button>)}</div><form onSubmit={e=>{e.preventDefault();if(new TextEncoder().encode(search).length<=128)setQuery(search);}}><label className="sr-only" htmlFor="report-search">报告标题或商品关键词</label><input id="report-search" value={search} onChange={e=>setSearch(e.target.value)} maxLength={128} placeholder="搜索报告或商品"/><Button type="submit" variant="outline">搜索</Button></form><Button variant="ghost" onClick={()=>void reports.refetch()}>刷新</Button></div>
    {purpose==="product"?<div className={styles.filters} role="group" aria-label="平台筛选"><span>平台</span><Button variant="secondary" aria-pressed="true">SHEIN</Button></div>:null}
    {reports.isPending || reports.isFetching || loadingMore ? <ConsoleState kind="loading" title="正在读取报告" /> : reports.isError ? <ConsoleState kind="error" title="报告列表读取失败"><p>{errorText(reports.error)}</p><Button variant="outline" onClick={() => void reports.refetch()}>重试</Button></ConsoleState> : items.length === 0 ? <div className={styles.empty}><ConsoleState kind="empty" title={purpose !== "all" || query ? "没有找到符合条件的报告" : view === "favorites" ? "尚无收藏报告" : "当前范围暂无报告"}>{purpose !== "all" || query ? <><p>{purpose === "product" && !productSaveAvailable ? "当前无法保存商品资料报告，已有历史报告仍可查看。" : "试试其他用途或搜索词。"}</p><Button variant="outline" onClick={()=>{setPurpose("all");setSearch("");setQuery("");}}>清除筛选</Button></> : view === "favorites" ? (canManage ? "打开已保存的报告后，可以收藏常用结果。" : "暂无可查看的收藏报告。") : !canManage ? "暂无可查看的报告。" : !saveAvailable ? "当前暂无可添加的结果。" : productSaveAvailable ? "可以添加已保存的商品资料，留作报告查看。" : "可以添加已有结果，保存后在这里查看。"}</ConsoleState></div> : <Card className={styles.list}><div className={styles.table}><table><thead><tr><th>名称</th><th>用途</th><th>商品</th><th>平台</th><th>保存时间</th></tr></thead><tbody>{items.map(item => <tr key={item.id}><td data-label="名称"><button className={styles.rowLink} onClick={() => open(item.id)}>{item.favorite ? <Star size={18} className={styles.star} aria-label="已收藏" /> : <FileText size={18} aria-hidden="true" />}{item.title}</button></td><td data-label="用途">{types[item.ref.kind]}</td><td data-label="商品">{item.productKey||"—"}</td><td data-label="平台">{item.ref.kind === "TITLE_REVIEW" || item.ref.kind==="STORE_DAILY_REPORT" ? "—" : "SHEIN"}</td><td data-label="保存时间"><time dateTime={item.capturedAt}>{new Date(item.capturedAt).toLocaleString("zh-CN")}</time></td></tr>)}</tbody></table></div>{reports.hasNextPage ? <Button variant="outline" disabled={reports.isFetching || loadingMore} onClick={() => void loadMore()}>加载更多</Button> : <p>已显示当前筛选范围全部报告</p>}</Card>}
  </>;
}

function download(report: Report, format: "json" | "txt") {
  const content = format === "json" ? JSON.stringify(report, null, 2) + "\n" : reportText(report);
  const blob = new Blob([content], { type: format === "json" ? "application/json;charset=utf-8" : "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob), link = document.createElement("a"); link.href = url; link.download = `report-${report.id}.${format}`; document.body.append(link); link.click(); link.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000);
}
const technicalLabels = new Set(["原同步记录","资料记录 ID","目标 ID","原来源 ID","店铺 ID","原资料 SHA-256","商品 SHA-256","资产清单 SHA-256","规则 SHA-256","资料摘要 SHA-256","诊断摘要 SHA-256","保存时 ReadyForUpload","原操作","规则版本","策略版本","诊断范围","外部新鲜度","发布事实 ID","资料修订","原来源类型","原商品版本","有效商品版本","审核策略","质量评分"]);
function ReportDetail({ report, manage, busy, favorite,scope }: { scope:ReportScope;report: Report; manage: boolean; busy: boolean; favorite: () => void }) {
  const [downloadError,setDownloadError]=useState("");
  async function downloadCurrent(format:"txt"|"json") {
    setDownloadError("");
    try{const current=report.ref.kind==="STORE_DAILY_REPORT"?await reportRequest(scope,`/${report.id}`,reportSchema):report;download(current,format)}
    catch{setDownloadError("当前无法下载这份报告，请刷新后重试。")}
  }
  const checkSection="保存时离线诊断（不代表平台发布许可）", buildSection="保存时构建结果（不代表当前上传许可或离线诊断）";
  const hiddenSection=(title:string)=>title==="资料依据"||title==="报告范围"||title==="未评估的外部条件";
  const businessLabel=(title:string,label:string)=> title===buildSection?"待处理项":title===checkSection&&!['离线状态','评估时间'].includes(label)?"检查项":label;
  const sectionTitle=(title:string)=>({"保存时资料身份":"资料信息","保存时资料摘要（非完整 SKU/媒体导出）":"商品资料摘要","保存时构建结果（不代表当前上传许可或离线诊断）":"保存时待处理项","保存时离线诊断（不代表平台发布许可）":"保存时检查结果","保存时 Apply 回执":"应用记录","标题审核历史快照":"标题审核结果"}[title]??title);
  const statusLabels:Record<string,string>={ready:"已通过",ready_with_warnings:"通过，有待关注项",blocked:"待处理",warning:"需留意",blocking:"待处理",unknown:"待确认",pending:"待审核",accepted:"已接受",rejected:"已拒绝",applied:"已应用"};
  const displayValue=(title:string,label:string,value:string)=>{
    if((title===checkSection&&label!=="评估时间")||label==="保存时状态"){
      const [first,...lines]=value.split("\n"),[status,...rest]=first.split(" · ");return `${statusLabels[status]??"待核对"}${rest.length?` · ${rest.join(" · ")}`:""}${lines.length?`\n${lines.join("\n")}`:""}`;
    }
    return value;
  };
  const sections=report.content.sections.flatMap(section=>{
    if(hiddenSection(section.title))return [];
    const fields=section.fields.filter(field=>!technicalLabels.has(field.label));
    return fields.length?[{title:sectionTitle(section.title),fields:fields.map(field=>({label:businessLabel(section.title,field.label),value:displayValue(section.title,field.label,field.value)})),summary:section.title.includes("非完整 SKU/媒体导出")}]:[];
  });
  const technicalSections=report.content.sections.flatMap(section=>{
    const fields=hiddenSection(section.title)?section.fields:section.fields.filter(field=>technicalLabels.has(field.label)||section.title===buildSection||section.title===checkSection&&!['离线状态','评估时间'].includes(field.label));
    return fields.length?[{...section,fields}]:[];
  });
  return <>{downloadError?<p role="alert">{downloadError}</p>:null}<h3>{report.title}</h3><p>{report.ref.kind==="STORE_DAILY_REPORT"?"这是该次执行保存的经营资料，后续数据变化不会覆盖此报告。":"这是保存时的记录，不代表商品当前已上架或可以发布。"}</p><dl className={styles.provenance}><dt>用途</dt><dd>{types[report.ref.kind]}</dd>{report.ref.kind!=="TITLE_REVIEW"&&report.ref.kind!=="STORE_DAILY_REPORT"?<><dt>平台</dt><dd>SHEIN</dd></>:null}{report.productKey?<><dt>商品标识</dt><dd>{report.productKey}</dd></>:null}<dt>保存时间</dt><dd>{new Date(report.capturedAt).toLocaleString("zh-CN")}</dd>{report.sourceAt ? <><dt>原结果时间</dt><dd>{new Date(report.sourceAt).toLocaleString("zh-CN")}</dd></> : null}</dl><div className={styles.actions}>{manage ? <Button variant="outline" aria-pressed={report.favorite} disabled={busy} onClick={favorite}>{report.favorite ? "取消收藏" : "收藏报告"}</Button> : null}<Button variant="outline" onClick={() => void downloadCurrent("txt")}>下载文本</Button><Button asChild variant="outline"><Link href={report.scheduledOrigin?`/workbench/ai/tasks?view=schedules&schedule_id=${report.scheduledOrigin.planId}`:sourceHref(report.ref)} prefetch={false}>打开原来源</Link></Button></div>{sections.map((section,index)=><section key={index} className={styles.section}><h4>{section.title}</h4>{section.summary?<p>此处为数量摘要，不包含完整规格和图片。</p>:null}<dl>{section.fields.map((field,n)=><div key={n}><dt>{field.label}</dt><dd>{field.value}</dd></div>)}</dl></section>)}{report.content.sections.some(section=>section.title==="未评估的外部条件")?<p>部分平台条件尚未检查。</p>:null}<details className={styles.technical}><summary>来源与技术信息</summary><Button variant="outline" onClick={() => void downloadCurrent("json")}>下载原始数据</Button><dl className={styles.provenance}><dt>来源编号</dt><dd>{report.ref.id}</dd><dt>来源版本</dt><dd>{report.ref.version}</dd><dt>内容 SHA-256</dt><dd>{report.digest}</dd></dl>{technicalSections.map((section,index)=><section key={index} className={styles.section}><h4>{section.title}</h4><dl>{section.fields.map((field,n)=><div key={n}><dt>{field.label}</dt><dd>{field.value}</dd></div>)}</dl></section>)}</details></>;
}
