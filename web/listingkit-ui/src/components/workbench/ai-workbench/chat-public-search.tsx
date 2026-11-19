"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { modelRouteRequest, ModelRouteError, type ModelRouteScope } from "@/lib/api/model-route-pricing";
import { searchChatRead, searchChatReceipt, searchChatCommand } from "@/lib/contracts/web-search";
import { ModelServicesLink } from "../account/model-services-link";
import { useResourcePending } from "../resources/resource-pending";

const pendingSchema = z.strictObject({ key: z.uuid() });
const statuses = {
  held: "正在准备搜索", claimed: "正在核对搜索结果", not_sent: "本次搜索未发出", unknown: "搜索结果待核实",
  complete: "已完成，网页结果仅在原页面短时显示", no_results: "本次未找到可显示的网页",
};

export function ChatPublicSearch({ scope, id, member, canUse }: { scope: ModelRouteScope; id: string; member: string; canUse: boolean }) {
  const path = "/search/chat/" + id;
  const state = useQuery({
    queryKey: ["chat-public-search", scope, member, id],
    queryFn: ({ signal }) => modelRouteRequest(scope, path, searchChatRead, { signal }),
    retry: false, gcTime: 0, refetchOnWindowFocus: false,
  });
  const pending = useResourcePending({ expectedUserId: scope.userId, expectedOrganizationId: scope.organizationId },
    ["chat-public-search", member, id], pendingSchema, { storage: "session" });
  const [opened, setOpened] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [notice, setNotice] = useState("");
  const [missing, setMissing] = useState<string>();
  const [latest, setLatest] = useState<z.infer<typeof searchChatCommand>["result"]>();
  const running = useRef<AbortController | null>(null);
  const button = useRef<HTMLButtonElement>(null);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => () => running.current?.abort(), []);
  useEffect(() => { if (opened) input.current?.focus(); }, [opened]);
  useEffect(() => {
    if (!latest) return;
    const timer = setTimeout(() => setLatest(undefined), 10 * 60_000);
    return () => clearTimeout(timer);
  }, [latest]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const query = keyword.trim(), option = state.data?.option;
    if (!canUse || !member || !option?.preview || !option.available || !query || [...query].length > 200 || running.current || !pending.ready || pending.read()) return;
    const command = { key: crypto.randomUUID() }, controller = new AbortController();
    running.current = controller;
    setNotice("正在查找公开网页…");
    setLatest(undefined);
    try {
      pending.persist(command);
      const value = await modelRouteRequest(scope, path, searchChatCommand, {
        method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": command.key },
        body: JSON.stringify({ query, preview: option.preview, expectedMemberId: member }), signal: controller.signal,
      });
      if (controller.signal.aborted) return;
      setLatest(value.result);
      setNotice(statuses[value.result.receipt.status]);
      setKeyword("");
      if (value.result.receipt.status !== "held" && value.result.receipt.status !== "claimed") pending.clear(command);
      await state.refetch();
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ModelRouteError && [400, 409, 429].includes(error.status)) {
        if (pending.read()?.key === command.key) pending.clear(command);
        setNotice(error.code === "SEARCH_LIMIT" ? "今日搜索次数已用完。" : "搜索未提交，请重新核对服务或权限。");
        await state.refetch();
      } else {
        setNotice("本次搜索结果待核实，请查看原请求。");
      }
    } finally {
      if (running.current === controller) running.current = null;
    }
  }

  async function recover() {
    const original = pending.command;
    if (!original || running.current) return;
    const controller = new AbortController();
    running.current = controller;
    try {
      const value = await modelRouteRequest(scope, path + "/" + original.key, searchChatReceipt, { signal: controller.signal });
      if (controller.signal.aborted) return;
      setNotice(statuses[value.result.status]);
      if (value.result.status !== "held" && value.result.status !== "claimed") pending.clear(original);
      await state.refetch();
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ModelRouteError && error.code === "NOT_FOUND") {
        setMissing(original.key);
        setNotice("尚未找到原请求。放弃核对后，再次搜索会计为新请求。");
      } else setNotice("暂时无法确认原请求，请稍后查看。");
    } finally {
      if (running.current === controller) running.current = null;
    }
  }

  if (!state.data) return null;
  const option = state.data.option;
  const history = state.data.result;
  return <Card className="min-w-0 space-y-3 p-4">
    {(canUse || pending.command) && <Button ref={button} type="button" variant="outline" aria-expanded={opened} onClick={() => setOpened(!opened)}>查公开资料</Button>}
    {opened && <>
      {option.available ? <p>当前搜索服务 <strong>{option.name}</strong>。搜索词会发送给 Brave，{option.source === "platform" ? "厂商费用由平台承担" : "费用由企业向厂商支付"}。</p> : <><p>当前企业尚未选择可用的搜索服务。</p><ModelServicesLink category="search" /></>}
      {canUse && option.available && <form onSubmit={submit} className="space-y-2">
        <label className="block">公开搜索词<Input ref={input} value={keyword} onChange={event => setKeyword(event.target.value)} maxLength={200} disabled={Boolean(pending.command)} /></label>
        <Button type="submit" disabled={!keyword.trim() || !pending.ready || Boolean(pending.command)}>确认搜索</Button>
      </form>}
      {pending.command && <Button type="button" variant="outline" onClick={() => void recover()}>查看原请求</Button>}
      {pending.command && missing === pending.command.key && <Button type="button" variant="ghost" onClick={() => {
        pending.clear(pending.command!);
        setMissing(undefined);
        setNotice("已放弃核对。新搜索将单独计次。");
      }}>放弃核对</Button>}
      {pending.error && <p role="alert">暂时无法读取本次搜索记录。</p>}
      {notice && <p role="status">{notice}</p>}
      {latest?.results?.length ? <ul className="space-y-3 break-words">
        {latest.results.map((result, index) => <li key={index}><a href={result.url} target="_blank" rel="noopener noreferrer" className="font-medium underline">{result.title}</a><p>{result.snippet}</p><small>Brave · {new Date(latest.receipt.startedAt).toLocaleString("zh-CN")}</small></li>)}
      </ul> : null}
      <Button type="button" variant="ghost" onClick={() => { setOpened(false); button.current?.focus(); }}>收起</Button>
    </>}
    {history.length > 0 && <details><summary>搜索记录</summary><ul className="space-y-1 break-words">{history.map(row => <li key={row.key}>{row.query} · {statuses[row.status]}</li>)}</ul></details>}
  </Card>;
}
