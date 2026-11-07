"use client";

import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  announcementContent, announcementIntent, announcementPage, announcementPublish,
  notificationCommandSchema, NotificationError, notificationHref, notificationRequest,
  type Announcement, type AnnouncementIntent, type AnnouncementSummary, type PlatformAnnouncementScope,
} from "@/lib/api/notifications";
import { AccountDialog } from "../account/account-dialog";
import { ConsolePage, ConsoleState } from "../console/console-page";
import { useResourcePending } from "../resources/resource-pending";

const announcementsPath = "/workbench/notifications/announcements";
const breadcrumbs = [{ label: "通知中心", href: "/workbench/notifications" }, { label: "官方公告管理" }];
const categories = { SYSTEM: "系统公告", PRODUCT: "产品更新", POLICY: "规则更新", ACTIVITY: "活动" };
const audience = "当前登录域内的所有已登录用户";
const dateTime = new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" });
const historyKey = (scope: PlatformAnnouncementScope) => ["platform-announcements", scope.userId, scope.realm];

type AccessRejected = (error: NotificationError) => void;

function isAccessRejection(error: unknown): error is NotificationError {
  return error instanceof NotificationError &&
    (error.status === 401 || error.status === 403 || error.code === "IDENTITY_CONTEXT_CHANGED");
}

function onReadRejection<T>(request: Promise<T>, signal: AbortSignal, onAccessRejected?: AccessRejected) {
  return request.catch(error => {
    if (!signal.aborted && isAccessRejection(error)) onAccessRejected?.(error);
    throw error;
  });
}

function useAnnouncementHistory(scope: PlatformAnnouncementScope, after = "", access?: { rejected: boolean; onRejected: AccessRejected }) {
  return useQuery({
    queryKey: [...historyKey(scope), after],
    queryFn: ({ signal }) => onReadRejection(notificationRequest(scope, after ? "?after=" + after : "", announcementPage, { signal }), signal, access?.onRejected),
    enabled: !access?.rejected,
    retry: false,
    gcTime: 0,
  });
}

export function OfficialAnnouncementsEntry({ scope }: { scope: PlatformAnnouncementScope }) {
  const history = useAnnouncementHistory(scope);
  return history.isSuccess ? <Button asChild variant="outline"><Link href={announcementsPath} prefetch={false}>官方公告管理</Link></Button> : null;
}

export function OfficialAnnouncementsPage({ scope }: { scope: PlatformAnnouncementScope }) {
  const context = useWorkbenchContext();
  const unavailable = context.user && context.user.id !== scope.userId ||
    [context.error, context.blockingError].some(error => error?.code === "AUTHENTICATION_REQUIRED");
  return <ConsolePage title="官方公告管理" description="发布平台公告，查看发布记录或撤回已发布的公告。" breadcrumbs={breadcrumbs}>
    {unavailable ? <ConsoleState kind="error" title="登录身份已变化" /> : <AnnouncementManager scope={scope} />}
  </ConsolePage>;
}

type Review = { operation: "publish"; body: Announcement } | { operation: "withdraw"; item: AnnouncementSummary };

function AnnouncementManager({ scope }: { scope: PlatformAnnouncementScope }) {
  const [pages, setPages] = useState([""]);
  const [composing, setComposing] = useState(false);
  const [detail, setDetail] = useState<AnnouncementSummary>();
  const [review, setReview] = useState<Review>();
  const [busy, setBusy] = useState(false);
  const [accessFailure, setAccessFailure] = useState<NotificationError>();
  const [notice, setNotice] = useState<{ error: boolean; text: string }>();
  const flight = useRef<AbortController | null>(null);
  const newButton = useRef<HTMLButtonElement>(null);
  const returnFocus = useRef(false);
  const client = useQueryClient();
  const pending = useResourcePending(
    { expectedUserId: scope.userId, expectedOrganizationId: "" },
    ["official-announcement", scope.realm], announcementIntent, { storage: "session", maxLength: 34 * 1024 },
  );
  const history = useAnnouncementHistory(scope, pages.at(-1), { rejected: !!accessFailure, onRejected: rejectAccess });
  const locked = busy || !pending.ready || pending.error || !!pending.command || history.isError;
  useEffect(() => () => flight.current?.abort(), []);
  useEffect(() => {
    if (!composing && returnFocus.current) {
      returnFocus.current = false;
      newButton.current?.focus();
    }
  }, [composing]);

  function rejectAccess(error: NotificationError) {
    setAccessFailure(error);
    flight.current?.abort();
    returnFocus.current = false;
    setBusy(false);
    setNotice(undefined);
    setReview(undefined);
    setDetail(undefined);
    setComposing(false);
    setPages([""]);
    client.removeQueries({ queryKey: historyKey(scope) });
  }

  async function send(intent: AnnouncementIntent, replay: boolean) {
    if (flight.current) return;
    const controller = new AbortController();
    flight.current = controller;
    setBusy(true);
    setNotice(undefined);
    try {
      pending.persist(intent);
      setReview(undefined);
      try {
        const path = intent.operation === "publish" ? "" : "/" + intent.id + "/withdraw";
        const result = await notificationRequest(scope, path, notificationCommandSchema, {
          key: intent.key, body: intent.body, signal: controller.signal,
        });
        if (controller.signal.aborted) return;
        if (result.key !== intent.key || result.operation !== intent.operation || !result.resultId ||
            intent.operation === "withdraw" && result.resultId !== intent.id) {
          throw new NotificationError("OUTCOME_UNKNOWN");
        }
      } catch (error) {
        if (controller.signal.aborted) return;
        if (isAccessRejection(error)) {
          rejectAccess(error);
          if (!replay) pending.clear(intent);
          return;
        }
        const rejected = error instanceof NotificationError && error.status >= 400 && error.status < 500 && error.code !== "OUTCOME_UNKNOWN";
        const conflict = error instanceof NotificationError && error.code === "IDEMPOTENCY_CONFLICT";
        if (rejected && (!replay || conflict)) pending.clear(intent);
        if (conflict) void client.invalidateQueries({ queryKey: historyKey(scope) });
        setNotice({ error: true, text: rejected ? errorMessage(error) : "提交结果尚未确认，请核实原操作。" });
        return;
      }
      pending.clear(intent);
      if (intent.operation === "publish") {
        setPages([""]);
        setComposing(false);
      }
      void client.invalidateQueries({ queryKey: historyKey(scope) });
      void client.invalidateQueries({ queryKey: ["notifications"] });
      setNotice({ error: false, text: intent.operation === "publish" ? "官方公告已发布。" : "官方公告已撤回。" });
    } catch {
      setNotice({ error: true, text: "无法保存或清除本次操作记录。请核实当前公告状态。" });
    } finally {
      flight.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  function confirm(current: Review) {
    if (locked) return;
    const key = crypto.randomUUID();
    void send(current.operation === "publish"
      ? { operation: "publish", key, body: current.body }
      : { operation: "withdraw", key, id: current.item.id, body: { revision: 1 } }, false);
  }

  if (accessFailure) return <ConsoleState kind="error" title={errorMessage(accessFailure)}>
    <Button variant="outline" onClick={() => setAccessFailure(undefined)}>重新核对权限</Button>
  </ConsoleState>;
  if (history.isPending) return <ConsoleState kind="loading" title="正在读取公告记录" />;
  if (!history.data || isAccessRejection(history.error)) return <ConsoleState kind="error" title={errorMessage(history.error)}>
    <Button variant="outline" onClick={() => void history.refetch()}>重新读取</Button>
  </ConsoleState>;

  const page = history.data;
  return <div className="grid gap-4">
    {history.isError ? <ConsoleState kind="error" title="无法更新公告记录">
      <Button variant="outline" onClick={() => void history.refetch()}>重新读取</Button>
    </ConsoleState> : null}
    {notice ? <p role={notice.error ? "alert" : "status"}>{notice.text}</p> : null}
    {busy ? <p role="status">正在提交公告操作…</p> : null}
    {pending.error ? <p role="alert">无法读取此浏览器中的操作记录，公告操作暂不可用。</p> : null}
    {pending.command && !busy ? <Card className="grid gap-3 p-4">
      <p>上一笔{pending.command.operation === "publish" ? "发布" : "撤回"}的结果待确认。核实会沿用原提交内容和操作编号。</p>
      <div><Button variant="outline" disabled={history.isError} onClick={() => { if (pending.command) void send(pending.command, true); }}>核实原操作</Button></div>
    </Card> : null}
    {composing ? <AnnouncementForm disabled={locked} onBack={() => setComposing(false)} onReview={body => setReview({ operation: "publish", body })} /> : <>
      <div><Button ref={newButton} disabled={locked} onClick={() => { returnFocus.current = true; setNotice(undefined); setComposing(true); }}>新建公告</Button></div>
      {page.items.length ? <Card className="divide-y overflow-hidden">
        <ul className="divide-y" aria-label="已发公告">
          {page.items.map(item => <li key={item.id} className="flex flex-wrap items-start justify-between gap-3 p-4">
            <div className="min-w-0 break-words">
              <h2 className="font-semibold">{item.title}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{categories[item.category]} · {item.state === "PUBLISHED" ? "已发布" : "已撤回"}</p>
              <p className="text-sm text-muted-foreground">发布时间 <time dateTime={item.publishedAt}>{dateTime.format(new Date(item.publishedAt))}</time></p>
              {item.withdrawnAt ? <p className="text-sm text-muted-foreground">撤回时间 <time dateTime={item.withdrawnAt}>{dateTime.format(new Date(item.withdrawnAt))}</time></p> : null}
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" aria-label={`查看公告 ${item.title}`} onClick={() => setDetail(item)}>查看</Button>
              {item.state === "PUBLISHED" ? <Button variant="outline" disabled={locked} aria-label={`撤回公告 ${item.title}`} onClick={() => setReview({ operation: "withdraw", item })}>撤回</Button> : null}
            </div>
          </li>)}
        </ul>
      </Card> : <ConsoleState kind="empty" title="暂无已发公告" />}
      {pages.length > 1 || page.next ? <div className="flex flex-wrap gap-2">
        {pages.length > 1 ? <Button variant="outline" onClick={() => setPages(current => current.slice(0, -1))}>上一页</Button> : null}
        {page.next ? <Button variant="outline" onClick={() => setPages(current => [...current, page.next])}>下一页</Button> : null}
      </div> : null}
    </>}
    {detail ? <AccountDialog title="公告正文" onClose={() => setDetail(undefined)}><AnnouncementDetail scope={scope} id={detail.id} onAccessRejected={rejectAccess} /></AccountDialog> : null}
    {review ? <AccountDialog title={review.operation === "publish" ? "确认发布官方公告" : "确认撤回官方公告"} onClose={() => setReview(undefined)}>
      {review.operation === "publish" ? <>
        <p className="mb-4">发送范围为<strong>{audience}</strong>。发布后内容即对收件人可见。</p>
        <AnnouncementText content={review.body} />
      </> : <p>撤回「{review.item.title}」后，收件人将无法在通知中心查看这条公告。</p>}
      <footer><Button variant="outline" onClick={() => setReview(undefined)}>取消</Button>
        <Button disabled={locked} variant={review.operation === "withdraw" ? "destructive" : "default"} onClick={() => confirm(review)}>
          {review.operation === "publish" ? "确认发布" : "确认撤回"}
        </Button>
      </footer>
    </AccountDialog> : null}
  </div>;
}

function AnnouncementForm({ disabled, onBack, onReview }: { disabled: boolean; onBack: () => void; onReview: (body: Announcement) => void }) {
  const title = useRef<HTMLInputElement>(null);
  const [invalid, setInvalid] = useState(false);
  useEffect(() => { title.current?.focus(); }, []);
  function review(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const parsed = announcementPublish.safeParse({
      category: data.get("category"),
      title: String(data.get("title")).trim(),
      summary: String(data.get("summary")).trim(),
      paragraphs: String(data.get("body")).split(/\n\s*\n/).map(paragraph => paragraph.trim()).filter(Boolean),
      target: { kind: "none", id: "" },
    });
    setInvalid(!parsed.success);
    if (parsed.success) onReview(parsed.data);
  }
  return <Card className="p-4 sm:p-6">
    <h2 className="mb-2 text-lg font-semibold">新建官方公告</h2>
    <p className="mb-4 text-sm text-muted-foreground">发送范围为{audience}。</p>
    <form className="grid max-w-2xl gap-4" onSubmit={review}>
      <label className="grid gap-1 text-sm">类别
        <select name="category" className="h-9 rounded-md border border-input bg-background px-3">
          {Object.entries(categories).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
        </select>
      </label>
      <label className="grid gap-1 text-sm">标题<Input ref={title} name="title" required maxLength={256} /></label>
      <label className="grid gap-1 text-sm">摘要<Input name="summary" required maxLength={1024} /></label>
      <label className="grid gap-1 text-sm">正文<Textarea name="body" required rows={7} maxLength={32768} /></label>
      {invalid ? <p role="alert">请填写有效的标题、摘要和正文。正文最多 32 段，过长时请缩短后再试。</p> : null}
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="outline" onClick={onBack}>返回公告列表</Button>
        <Button type="submit" disabled={disabled}>核对发送范围</Button>
      </div>
    </form>
  </Card>;
}

function AnnouncementDetail({ scope, id, onAccessRejected }: { scope: PlatformAnnouncementScope; id: string; onAccessRejected: AccessRejected }) {
  const detail = useQuery({
    queryKey: [...historyKey(scope), "detail", id],
    queryFn: ({ signal }) => onReadRejection(notificationRequest(scope, "/" + id, announcementContent, { signal }), signal, onAccessRejected),
    gcTime: 0, retry: false,
  });
  if (detail.isPending) return <p role="status">正在读取正文…</p>;
  if (detail.isError) return <p role="alert">{errorMessage(detail.error)}</p>;
  return <AnnouncementText content={detail.data} />;
}

function AnnouncementText({ content }: { content: Announcement }) {
  const href = notificationHref(content.target);
  return <div className="grid gap-2 break-words">
    <p className="text-sm text-muted-foreground">{categories[content.category]}</p>
    <p className="font-semibold">{content.title}</p>
    <p className="text-sm text-muted-foreground">{content.summary}</p>
    {content.paragraphs.map((paragraph, index) => <p className="whitespace-pre-wrap" key={index}>{paragraph}</p>)}
    {href ? <Link className="underline" href={href} prefetch={false}>公告关联入口</Link> : null}
  </div>;
}

function errorMessage(error: unknown) {
  if (error instanceof NotificationError) {
    if (error.status === 403) return "只有平台管理员可以管理官方公告。";
    if (error.status === 401 || error.code === "IDENTITY_CONTEXT_CHANGED") return "登录身份已变化，请重新登录后再试。";
    if (error.code === "IDEMPOTENCY_CONFLICT") return "公告状态或操作记录已变化，请查看最新记录。";
    if (error.status === 400) return "公告内容或请求格式有误，请核对后再试。";
    if (error.status === 404) return "未找到公告。";
  }
  return "官方公告服务暂不可用。";
}
