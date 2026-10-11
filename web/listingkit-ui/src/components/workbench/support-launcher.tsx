"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { parseSupportSession, type SupportSession } from "@/lib/contracts/platform-support";

const endpoint = "/api/support/session";
const prefix = "chatwoot-widget:";
type Phase = "closed" | "connecting" | "open" | "checking" | "failed";

async function sessionRequest(signal: AbortSignal, body?: { subject: string; widgetAuthToken: string }) {
  const response = await fetch(endpoint, { method: body ? "POST" : "GET", cache: "no-store", credentials: "same-origin", signal,
    headers: { Accept: "application/json", ...(body ? { "Content-Type": "application/json" } : {}) }, ...(body ? { body: JSON.stringify(body) } : {}) });
  if (!response.ok) {
    void response.body?.cancel().catch(() => undefined);
    throw new Error(response.status === 401 || response.status === 409 ? "登录状态已变化，请重新登录后重试。" : response.status === 503 ? "客服尚未配置，请稍后重试。" : "暂时无法连接客服，请稍后重试。");
  }
  return readBoundedStrictJSON(response, 4096, signal);
}

export function SupportLauncher({ available }: { available: boolean }) {
  const [phase, setPhase] = useState<Phase>("closed");
  const [session, setSession] = useState<SupportSession | null>(null);
  const [error, setError] = useState("");
  const current = useRef<SupportSession | null>(null);
  const currentPhase = useRef<Phase>("closed");
  const frame = useRef<HTMLIFrameElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const closeButton = useRef<HTMLButtonElement>(null);
  const request = useRef<AbortController | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const generation = useRef(0);
  const identifying = useRef(false);
  const loads = useRef(0);

  const changePhase = useCallback((value: Phase) => { currentPhase.current = value; setPhase(value); }, []);
  const discard = useCallback(() => {
    generation.current++; request.current?.abort(); clearTimeout(timer.current);
    current.current = null; identifying.current = false; loads.current = 0;
  }, []);
  const fail = useCallback((attempt: number, message: string) => {
    if (generation.current !== attempt) return;
    discard(); setSession(null); setError(message); changePhase("failed");
  }, [changePhase, discard]);
  const close = useCallback(() => { discard(); setSession(null); setError(""); changePhase("closed"); trigger.current?.focus(); }, [changePhase, discard]);

  const begin = useCallback(async () => {
    discard(); setSession(null); setError(""); changePhase("connecting");
    const attempt = generation.current;
    const controller = new AbortController(); request.current = controller;
    timer.current = setTimeout(() => fail(attempt, "连接客服超时，请重试。"), 20_000);
    closeButton.current?.focus();
    try {
      const next = parseSupportSession(await sessionRequest(controller.signal));
      if (generation.current !== attempt) return;
      if (!next || next.origin === window.location.origin) throw new Error("客服配置不可用，请稍后重试。");
      current.current = next; setSession(next);
    } catch (cause) { fail(attempt, cause instanceof Error ? cause.message : "暂时无法连接客服，请稍后重试。"); }
  }, [changePhase, discard, fail]);

  const checkSession = useCallback(async () => {
    if (currentPhase.current !== "open" || !current.current) return;
    const expected = current.current;
    const attempt = generation.current;
    changePhase("checking");
    const controller = new AbortController(); request.current?.abort(); request.current = controller;
    timer.current = setTimeout(() => fail(attempt, "连接客服超时，请重试。"), 20_000);
    try {
      const checked = parseSupportSession(await sessionRequest(controller.signal));
      if (generation.current !== attempt) return;
      if (!checked || checked.subject !== expected.subject || checked.origin !== expected.origin || checked.websiteToken !== expected.websiteToken) throw new Error("登录状态已变化，请重新打开客服。");
      clearTimeout(timer.current); changePhase("open");
    } catch (cause) { fail(attempt, cause instanceof Error ? cause.message : "暂时无法连接客服，请稍后重试。"); }
  }, [changePhase, fail]);

  useEffect(() => { if (phase === "connecting") closeButton.current?.focus(); }, [phase]);

  useEffect(() => {
    async function receive(event: MessageEvent) {
      const expected = current.current;
      const source = frame.current?.contentWindow;
      if (!expected || !source || event.origin !== expected.origin || event.source !== source || typeof event.data !== "string" || event.data.length > 65536 || !event.data.startsWith(prefix)) return;
      let message;
      try { message = JSON.parse(event.data.slice(prefix.length)); } catch { return; }
      if (!message || typeof message !== "object" || message.event !== "loaded") return;
      if (currentPhase.current === "open" || currentPhase.current === "checking") { fail(generation.current, "客服连接已变化，请重新打开客服。"); return; }
      if (identifying.current || currentPhase.current !== "connecting") return;
      const token = message.config?.authToken;
      if (typeof token !== "string" || !/^[A-Za-z0-9._-]{1,2048}$/.test(token)) { fail(generation.current, "暂时无法连接客服，请稍后重试。"); return; }
      identifying.current = true;
      const attempt = generation.current;
      try {
        const result = await sessionRequest(request.current!.signal, { subject: expected.subject, widgetAuthToken: token });
        if (generation.current !== attempt) return;
        if (!result || typeof result !== "object" || (result as { ok?: unknown }).ok !== true) throw new Error("暂时无法连接客服，请稍后重试。");
        const send = (value: unknown) => source.postMessage(prefix + JSON.stringify(value), expected.origin);
        send({ event: "config-set", locale: "zh_CN", hideMessageBubble: true, position: "right", enableFileUpload: true, enableEmojiPicker: true, darkMode: "auto" });
        send({ event: "toggle-open", isOpen: true });
        clearTimeout(timer.current); changePhase("open");
      } catch (cause) { fail(attempt, cause instanceof Error ? cause.message : "暂时无法连接客服，请稍后重试。"); }
    }
    function escape(event: KeyboardEvent) { if (event.key === "Escape" && currentPhase.current !== "closed") close(); }
    const focus = () => { void checkSession(); };
    window.addEventListener("message", receive);
    window.addEventListener("focus", focus);
    window.addEventListener("keydown", escape);
    const interval = setInterval(focus, 30_000);
    return () => { window.removeEventListener("message", receive); window.removeEventListener("focus", focus); window.removeEventListener("keydown", escape); clearInterval(interval); discard(); };
  }, [changePhase, checkSession, close, discard, fail]);

  return <>
    <Button ref={trigger} disabled={!available} title={available ? "使用咨询、问题反馈与人工客服" : "客服服务暂未接入"} variant="outline" aria-expanded={phase !== "closed"} aria-controls="platform-support-panel" onClick={() => phase === "closed" ? void begin() : close()}>联系客服</Button>
    {phase !== "closed" ? <section id="platform-support-panel" role="dialog" aria-modal="false" aria-labelledby="platform-support-title" className="fixed bottom-4 right-4 z-50 flex h-[min(640px,calc(100dvh-32px))] w-[min(400px,calc(100vw-32px))] flex-col overflow-hidden rounded-xl border border-border bg-card text-foreground shadow-xl">
      <header className="flex items-center justify-between border-b border-border px-4 py-2"><h2 id="platform-support-title" className="text-sm font-semibold">平台客服</h2><Button ref={closeButton} variant="ghost" size="icon" aria-label="关闭客服" onClick={close}><X aria-hidden="true" /></Button></header>
      {phase === "failed" ? <div className="p-5"><p role="alert" className="text-sm">{error}</p><Button className="mt-4" variant="outline" onClick={() => void begin()}>重试连接</Button></div> : <>
        {phase !== "open" ? <p role="status" className="absolute left-4 top-20 text-sm text-muted-foreground">正在连接客服...</p> : null}
        {session ? <iframe ref={frame} title="平台客服聊天" className="min-h-0 w-full flex-1 border-0" style={{ visibility: phase === "open" ? "visible" : "hidden" }} tabIndex={phase === "open" ? 0 : -1} aria-hidden={phase !== "open"} referrerPolicy="no-referrer" sandbox="allow-scripts allow-forms allow-same-origin allow-downloads" src={`${session.origin}/widget?website_token=${encodeURIComponent(session.websiteToken)}`} onLoad={() => { if (++loads.current > 1) fail(generation.current, "客服连接已变化，请重新打开客服。"); }} /> : null}
      </>}
    </section> : null}
  </>;
}
