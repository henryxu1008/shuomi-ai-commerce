"use client";

import { Mic, MicOff } from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";

type Result = { isFinal: boolean; [index: number]: { transcript: string } };
type Recognition = {
  lang: string; interimResults: boolean; continuous: boolean;
  onresult: ((event: {resultIndex: number; results: ArrayLike<Result>}) => void) | null;
  onerror: ((event: {error: string}) => void) | null;
  onend: (() => void) | null;
  start: () => void; stop: () => void; abort: () => void;
};
type RecognitionConstructor = new () => Recognition;
type SpeechWindow = Window & { SpeechRecognition?: RecognitionConstructor; webkitSpeechRecognition?: RecognitionConstructor };

export function ChatVoiceInput({ identity, disabled, onText, onActiveChange }: {identity: string; disabled: boolean; onText: (text: string) => void; onActiveChange: (active: boolean) => void}) {
  const [active, setActive] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [supported, setSupported] = useState<boolean | null>(null);
  const [notice, setNotice] = useState("");
  const current = useRef<{identity: string; disabled: boolean; onText: (text: string) => void; onActiveChange: (active: boolean) => void}>({identity, disabled, onText, onActiveChange});
  useLayoutEffect(() => { current.current = {identity, disabled, onText, onActiveChange}; });
  const session = useRef<{engine: Recognition; identity: string} | null>(null);
  useEffect(() => { let alive = true; queueMicrotask(() => { const win = window as SpeechWindow; if (alive) setSupported(Boolean(win.SpeechRecognition || win.webkitSpeechRecognition)); }); return () => { alive = false; }; }, []);
  useEffect(() => {
    const entry = session.current;
    if (entry && (disabled || entry.identity !== identity)) {
      session.current = null; entry.engine.abort(); setActive(false); setStopping(false); onActiveChange(false);
    }
  }, [identity, disabled, onActiveChange]);
  useEffect(() => () => { const entry = session.current; session.current = null; if (entry) { entry.engine.abort(); current.current.onActiveChange(false); } }, []);
  function cancel() {
    const entry = session.current; session.current = null;
    entry?.engine.abort(); setActive(false); setStopping(false); current.current.onActiveChange(false);
  }
  function stop() {
    if (!session.current || stopping) return;
    try { setStopping(true); session.current.engine.stop(); }
    catch { cancel(); setNotice("语音识别未完成，请重试或使用键盘输入。"); }
  }
  function start() {
    if (session.current || current.current.disabled) return;
    const win = window as SpeechWindow, Constructor = win.SpeechRecognition || win.webkitSpeechRecognition;
    if (!Constructor) { setNotice("当前浏览器不支持语音输入，可继续键盘输入。"); return; }
    let engine: Recognition;
    try { engine = new Constructor(); } catch { setNotice("无法启动语音输入，请检查浏览器权限或使用键盘输入。"); return; }
    const entry = {engine, identity: current.current.identity};
    session.current = entry;
    const finalIndexes = new Set<number>();
    let hasTranscript = false;
    const valid = () => session.current === entry && current.current.identity === entry.identity && !current.current.disabled;
    engine.lang = "zh-CN"; engine.interimResults = false; engine.continuous = true;
    engine.onresult = event => {
      if (!valid()) return;
      let text = "";
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result?.isFinal && !finalIndexes.has(i)) { finalIndexes.add(i); text += result[0]?.transcript ?? ""; }
      }
      if (text) { hasTranscript = true; current.current.onText(text); }
    };
    engine.onerror = event => {
      if (!valid()) return;
      setNotice(event.error === "not-allowed" || event.error === "service-not-allowed" ? "麦克风或语音服务未获授权，请检查浏览器权限。" : event.error === "no-speech" ? "没有识别到语音，请重试。" : event.error === "audio-capture" ? "无法使用麦克风，请检查设备。" : "语音识别未完成，请重试或使用键盘输入。");
      cancel();
    };
    engine.onend = () => { if (valid()) { session.current = null; setActive(false); setStopping(false); current.current.onActiveChange(false); if (!hasTranscript) setNotice("没有识别到语音，请重试。"); } };
    try { engine.start(); setNotice(""); setActive(true); current.current.onActiveChange(true); }
    catch { session.current = null; setActive(false); current.current.onActiveChange(false); setNotice("无法启动语音输入，请检查浏览器权限或使用键盘输入。"); }
  }
  return <>
    <Button size="icon" variant="outline" onClick={active ? stop : start} disabled={disabled || supported !== true || stopping} aria-label={stopping ? "正在结束语音输入" : active ? "停止语音输入" : "开始语音输入"} aria-pressed={active} title={stopping ? "正在结束语音输入" : active ? "停止语音输入" : "开始语音输入"}>
      {active ? <MicOff size={18} aria-hidden="true" /> : <Mic size={18} aria-hidden="true" />}
    </Button>
    <span>浏览器可能将语音发送给其服务商识别</span>
    <span aria-live="polite">{notice || (stopping ? "正在完成语音识别" : active ? "正在听写，点击停止" : supported === false ? "当前浏览器不支持语音输入" : "")}</span>
  </>;
}
