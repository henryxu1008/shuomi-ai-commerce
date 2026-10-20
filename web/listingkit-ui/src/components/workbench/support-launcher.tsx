"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { isSupportSubject, type SupportCollection, type SupportConversation, type SupportHistory, type SupportMessage } from "@/lib/contracts/platform-support";

type Phase = "closed" | "loading" | "open" | "checking" | "failed";
async function api<T extends { subject: string }>(path: string, signal: AbortSignal, expected?: string, body?: FormData): Promise<T> {
  const response = await fetch(`/api/support/${path}`, { method: body ? "POST" : "GET", cache: "no-store", credentials: "same-origin", signal, ...(body ? { body } : {}) });
  if (!response.ok) { void response.body?.cancel().catch(() => undefined); throw new Error([401, 403, 404, 409].includes(response.status) ? "登录状态或会话已变化，请重新打开客服。" : response.status === 413 ? "图片不能超过 5 MB。" : "暂时无法连接客服，请稍后重试。"); }
  const result = await readBoundedStrictJSON(response, 2 * 1024 * 1024, signal) as T;
  if (!result || !isSupportSubject(result.subject) || expected && result.subject !== expected) throw new Error("登录状态已变化，请重新打开客服。");
  return result;
}
const mergeMessages = (older: SupportMessage[], newer: SupportMessage[]) => [...new Map([...older, ...newer].map(message => [message.id, message])).values()].sort((a, b) => a.id - b.id);

export function SupportLauncher({ available }: { available: boolean }) {
  const [phase, setPhase] = useState<Phase>("closed"); const [subject, setSubject] = useState("");
  const [conversations, setConversations] = useState<SupportConversation[]>([]); const [conversationId, setConversationId] = useState<number | null>(null);
  const [messages, setMessages] = useState<SupportMessage[]>([]); const [before, setBefore] = useState<number | null>(null);
  const [draft, setDraft] = useState(""); const [attachment, setAttachment] = useState<File | null>(null);
  const [sending, setSending] = useState(false); const [error, setError] = useState(""); const [unconfirmed, setUnconfirmed] = useState(false);
  const generation = useRef(0); const pending = useRef<AbortController | null>(null); const trigger = useRef<HTMLButtonElement>(null); const closeButton = useRef<HTMLButtonElement>(null); const file = useRef<HTMLInputElement>(null); const bottom = useRef<HTMLDivElement>(null);
  const discard = useCallback(() => { generation.current++; pending.current?.abort(); pending.current = null; }, []);
  const clear = useCallback(() => { setSubject(""); setMessages([]); setConversations([]); setConversationId(null); setBefore(null); setDraft(""); setAttachment(null); setSending(false); setUnconfirmed(false); }, []);
  const close = useCallback(() => { discard(); clear(); setPhase("closed"); setError(""); trigger.current?.focus(); }, [clear, discard]);
  const fail = useCallback((attempt: number, cause: unknown) => { if (generation.current !== attempt) return; discard(); clear(); setPhase("failed"); setError(cause instanceof Error ? cause.message : "暂时无法连接客服。"); }, [clear, discard]);
  const request = useCallback(() => { const controller = new AbortController(); pending.current = controller; const signal = AbortSignal.any([controller.signal, AbortSignal.timeout(25000)]); return { controller, signal, attempt: generation.current }; }, []);

  const begin = useCallback(async () => {
    discard(); clear(); setError(""); setPhase("loading"); const { controller, signal, attempt } = request();
    try {
      const identity = await api<{ subject: string }>("session", signal);
      const collection = await api<SupportCollection>("conversations", signal, identity.subject);
      const selected = collection.conversations[0]?.id ?? null;
      const history = selected ? await api<SupportHistory>(`conversations/${selected}/messages`, signal, identity.subject) : { messages: [], nextBefore: null };
      if (generation.current !== attempt) return;
      setSubject(identity.subject); setConversations(collection.conversations); setConversationId(selected); setMessages(history.messages); setBefore(history.nextBefore); setPhase("open");
    } catch (cause) { fail(attempt, cause); } finally { if (pending.current === controller) pending.current = null; }
  }, [clear, discard, fail, request]);

  const refresh = useCallback(async (focus = false) => {
    if (phase !== "open" || pending.current || sending) return;
    const { controller, signal, attempt } = request(); if (focus) setPhase("checking");
    try {
      await api<{ subject: string }>("session", signal, subject);
      if (conversationId) {
        const history = await api<SupportHistory>(`conversations/${conversationId}/messages`, signal, subject);
        if (generation.current !== attempt) return;
        setMessages(previous => mergeMessages(previous, history.messages));
      }
      if (generation.current === attempt) setPhase("open");
    } catch (cause) { fail(attempt, cause); } finally { if (pending.current === controller) pending.current = null; }
  }, [conversationId, fail, phase, request, sending, subject]);

  async function selectConversation(selected: number) {
    discard(); setPhase("loading"); setMessages([]); setBefore(null); setError(""); const { controller, signal, attempt } = request();
    try { const history = await api<SupportHistory>(`conversations/${selected}/messages`, signal, subject); if (generation.current !== attempt) return; setConversationId(selected); setMessages(history.messages); setBefore(history.nextBefore); setPhase("open"); }
    catch (cause) { fail(attempt, cause); } finally { if (pending.current === controller) pending.current = null; }
  }
  async function older() {
    if (!conversationId || !before || pending.current) return;
    const { controller, signal, attempt } = request();
    try { const history = await api<SupportHistory>(`conversations/${conversationId}/messages?before=${before}`, signal, subject); if (generation.current !== attempt) return; setMessages(previous => mergeMessages(history.messages, previous)); setBefore(history.nextBefore); }
    catch (cause) { fail(attempt, cause); } finally { if (pending.current === controller) pending.current = null; }
  }
  async function send() {
    if (sending || pending.current || phase !== "open" || !draft.trim() && !attachment) return;
    setSending(true); setError(""); const { controller, signal, attempt } = request(); let confirmed = false;
    try {
      const body = new FormData(); body.set("subject", subject); body.set("content", draft); if (conversationId) body.set("conversationId", String(conversationId)); if (attachment) body.set("attachment", attachment);
      const result = await api<{ subject: string; conversation: SupportConversation; messageId: number }>("conversations", signal, subject, body);
      if (generation.current !== attempt) return;
      confirmed = true; setDraft(""); setAttachment(null); setUnconfirmed(false); if (file.current) file.current.value = "";
      setConversationId(result.conversation.id); setConversations(previous => [result.conversation, ...previous.filter(value => value.id !== result.conversation.id)]);
      const history = await api<SupportHistory>(`conversations/${result.conversation.id}/messages`, signal, subject);
      if (generation.current !== attempt) return; setMessages(previous => mergeMessages(previous, history.messages)); setBefore(history.nextBefore);
    } catch (cause) {
      if (generation.current !== attempt) return;
      setUnconfirmed(!confirmed); setError(confirmed ? "消息已发送，记录刷新失败，请稍后刷新。" : "发送结果未确认。请先刷新记录，确认未发送后再决定是否重新发送。");
      if (cause instanceof Error && cause.message.includes("登录状态")) fail(attempt, cause);
    } finally { if (pending.current === controller) pending.current = null; if (generation.current === attempt) setSending(false); }
  }

  useEffect(() => { if (phase === "loading") closeButton.current?.focus(); }, [phase]);
  useEffect(() => { bottom.current?.scrollIntoView?.({ block: "nearest" }); }, [messages]);
  useEffect(() => { const focus = () => { void refresh(true); }; const timer = setInterval(() => { void refresh(); }, 5000); window.addEventListener("focus", focus); return () => { clearInterval(timer); window.removeEventListener("focus", focus); }; }, [refresh]);
  useEffect(() => { const escape = (event: KeyboardEvent) => { if (event.key === "Escape") close(); }; window.addEventListener("keydown", escape); return () => { window.removeEventListener("keydown", escape); discard(); }; }, [close, discard]);

  return <>
    <Button ref={trigger} disabled={!available} title={available ? "使用咨询、问题反馈与人工客服" : "客服服务暂未接入"} variant="outline" aria-expanded={phase !== "closed"} aria-controls="platform-support-panel" onClick={() => phase === "closed" ? void begin() : close()}>联系客服</Button>
    {phase !== "closed" ? <section id="platform-support-panel" role="dialog" aria-modal="false" aria-labelledby="platform-support-title" className="fixed bottom-4 right-4 z-50 flex h-[min(640px,calc(100dvh-32px))] w-[min(400px,calc(100vw-32px))] flex-col overflow-hidden rounded-xl border border-border bg-card text-foreground shadow-xl">
      <header className="flex items-center justify-between border-b border-border px-4 py-2"><div><h2 id="platform-support-title" className="text-xl font-semibold">平台客服</h2><p className="text-xs text-muted-foreground">使用咨询、问题反馈 · 人工稍后回复</p></div><Button ref={closeButton} variant="ghost" size="icon" aria-label="关闭客服" onClick={close}><X aria-hidden="true" /></Button></header>
      {phase === "loading" || phase === "checking" ? <p role="status" className="p-5 text-sm text-muted-foreground">正在读取客服记录...</p> : phase === "failed" ? <div className="p-5"><p role="alert" className="text-sm">{error}</p><Button className="mt-4" variant="outline" onClick={() => void begin()}>重新连接</Button></div> : <>
        <div className="flex items-center justify-between gap-2 border-b border-border px-4 py-2">{conversations.length > 1 ? <select aria-label="历史会话" value={conversationId ?? ""} onChange={event => void selectConversation(Number(event.target.value))} className="max-w-56 rounded border border-input bg-background p-1 text-xs">{conversations.map(value => <option key={value.id} value={value.id}>会话 #{value.id} · {new Date(value.createdAt * 1000).toLocaleDateString()}</option>)}</select> : <span className="text-xs text-muted-foreground">个人客服记录</span>}<Button size="sm" variant="ghost" disabled={sending} onClick={() => void refresh()}>刷新记录</Button></div>
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4" role="log" aria-label="客服消息">{before ? <Button size="sm" variant="ghost" onClick={() => void older()}>查看更早消息</Button> : null}{messages.length === 0 ? <p className="text-sm text-muted-foreground">描述你的问题，也可以附上报错截图。人工不在线时可留言。</p> : messages.map(message => <article key={message.id} className={`max-w-[90%] rounded-xl p-3 text-sm ${message.type === "incoming" ? "ml-auto bg-primary text-primary-foreground" : "bg-muted"}`}><p className="mb-1 text-xs opacity-70">{message.type === "incoming" ? "我" : "人工客服"}</p>{message.content ? <p className="whitespace-pre-wrap break-words">{message.content}</p> : null}{message.attachments.map(image => <a key={image.id} href={image.url} download className="mt-2 block underline">查看/下载{image.name}</a>)}<time className="mt-1 block text-xs opacity-60">{new Date(message.createdAt * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</time></article>)}<div ref={bottom} /></div>
        <form className="space-y-2 border-t border-border p-3" onSubmit={event => { event.preventDefault(); void send(); }}>
          {error ? <p role="alert" className="text-xs text-destructive">{error}</p> : null}
          <Textarea aria-label="消息" placeholder="输入使用问题或报错描述" maxLength={4000} value={draft} disabled={sending} onChange={event => setDraft(event.target.value)} className="min-h-20 resize-none" />
          <div className="flex items-center justify-between gap-2"><div className="min-w-0"><label className="cursor-pointer text-xs underline">添加截图<input ref={file} type="file" aria-label="添加截图" accept="image/png,image/jpeg,image/webp" className="sr-only" disabled={sending} onChange={event => { const next = event.target.files?.[0] ?? null; if (next && (next.size > 5 * 1024 * 1024 || !["image/png", "image/jpeg", "image/webp"].includes(next.type))) { setError("请选择不超过 5 MB 的 PNG、JPEG 或 WebP 图片。"); event.target.value = ""; return; } setAttachment(next); setError(""); }} /></label>{attachment ? <p className="max-w-44 truncate text-xs text-muted-foreground">{attachment.name}</p> : <p className="text-xs text-muted-foreground">PNG/JPEG/WebP，最多 5 MB</p>}</div><Button type="submit" size="sm" disabled={sending || !draft.trim() && !attachment}>{sending ? "正在发送..." : unconfirmed ? "重新发送" : "发送"}</Button></div>
        </form>
      </>}
    </section> : null}
  </>;
}
