import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { ChatVoiceInput } from "./chat-voice-input";

type Event = {resultIndex: number; results: Array<{isFinal: boolean; 0: {transcript: string}}>};
let last: FakeRecognition;
function remember(value: FakeRecognition) { last = value; }
class FakeRecognition {
  onresult: ((event: Event) => void) | null = null;
  onerror: ((event: {error: string}) => void) | null = null;
  onend: (() => void) | null = null;
  start = vi.fn(); stop = vi.fn(); abort = vi.fn();
  constructor() { remember(this); }
}
const result = (transcript: string) => ({isFinal:true, 0:{transcript}});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
function setup(identity = '["u","o","general"]') {
  vi.stubGlobal("SpeechRecognition", FakeRecognition);
  const onText=vi.fn(), onActiveChange=vi.fn();
  const view=render(<ChatVoiceInput identity={identity} disabled={false} onText={onText} onActiveChange={onActiveChange} />);
  return {onText,onActiveChange,...view};
}
it("does not start without a click and joins each final result once", async () => {
  const {onText}=setup();
  expect(screen.queryByText("浏览器可能将语音发送给其服务商识别")).toBeNull();
  expect(screen.getByRole("button", {name:"开始语音输入"})).toHaveAttribute("aria-description", "语音可能由浏览器服务商识别");
  await waitFor(()=>expect(screen.getByRole("button", {name:"开始语音输入"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  expect(last.start).toHaveBeenCalledTimes(1);
  act(()=>last.onresult?.({resultIndex:0,results:[result("你好"),result("世界")]}));
  act(()=>last.onresult?.({resultIndex:0,results:[result("你好"),result("世界")]}));
  expect(onText).toHaveBeenCalledTimes(1);
  expect(onText).toHaveBeenCalledWith("你好世界");
});
it("aborts and ignores late results after identity changes", async () => {
  const {onText,onActiveChange,rerender}=setup();
  await waitFor(()=>expect(screen.getByRole("button", {name:"开始语音输入"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  const old=last;
  rerender(<ChatVoiceInput identity='["u","o","store:other"]' disabled={false} onText={onText} onActiveChange={onActiveChange} />);
  expect(old.abort).toHaveBeenCalledTimes(1);
  act(()=>old.onresult?.({resultIndex:0,results:[result("旧背景")]}));
  expect(onText).not.toHaveBeenCalled();
});
it("shows permission errors and allows a new attempt", async () => {
  setup(); await waitFor(()=>expect(screen.getByRole("button", {name:"开始语音输入"})).toBeEnabled()); fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  act(()=>last.onerror?.({error:"not-allowed"}));
  expect(screen.getByText(/未获授权/)).toBeVisible();
  expect(screen.getByRole("button", {name:"开始语音输入"})).toHaveAttribute("aria-description", "语音可能由浏览器服务商识别");
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  expect(last.start).toHaveBeenCalledTimes(1);
});
it("accepts the final result after stop and ignores results after end", async () => {
  const {onText}=setup(); await waitFor(()=>expect(screen.getByRole("button", {name:"开始语音输入"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  fireEvent.click(screen.getByRole("button", {name:"停止语音输入"}));
  expect(last.stop).toHaveBeenCalledTimes(1);
  expect(screen.getByRole("button", {name:"正在结束语音输入"})).toBeDisabled();
  act(()=>last.onresult?.({resultIndex:0,results:[result("最后一句")]}));
  act(()=>last.onend?.());
  act(()=>last.onresult?.({resultIndex:1,results:[result("最后一句"),result("迟到")]}));
  expect(onText).toHaveBeenCalledOnce(); expect(onText).toHaveBeenCalledWith("最后一句");
});
it("reports an empty end and a failed constructor", async () => {
  setup(); await waitFor(()=>expect(screen.getByRole("button", {name:"开始语音输入"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  act(()=>last.onend?.());
  expect(screen.getByText(/没有识别到语音/)).toBeVisible();
  vi.stubGlobal("SpeechRecognition", class { constructor() { throw new Error("unavailable"); } });
  fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
  expect(screen.getByText(/无法启动语音输入/)).toBeVisible();
});
it("keeps keyboard input available when speech recognition is unsupported", async () => {
  vi.stubGlobal("SpeechRecognition", undefined); vi.stubGlobal("webkitSpeechRecognition", undefined);
  render(<ChatVoiceInput identity="test" disabled={false} onText={vi.fn()} onActiveChange={vi.fn()} />);
  expect(screen.getByRole("button", {name:"开始语音输入"})).toBeDisabled();
  expect(await screen.findByText(/当前浏览器不支持/)).toBeVisible();
});
