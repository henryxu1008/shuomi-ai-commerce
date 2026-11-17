"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { AIWorkbenchError, requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import { formatMicropoints } from "@/lib/contracts/ai-points";
import { type AISkillAuthoringChoice, type AISkillAuthoringJob, type AITitleSkillDraft } from "@/lib/contracts/ai-workbench";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { AccountDialog } from "../account/account-dialog";

function useCatalog(scope: AIScope, enabled: boolean) {
  return useQuery({
    queryKey: ["chat-skill-authoring-catalog", scope.userId, scope.organizationId],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-catalog", method: "GET", path: "chat/conversations/skill-authoring", scope, signal }),
    enabled, retry: false, staleTime: 0,
  });
}

export function ChatSkillDrawer({ scope, disabled, onSelect, onResume }: {
  scope: AIScope;
  disabled?: boolean;
  onSelect: (skill: AISkillAuthoringChoice) => void;
  onResume?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const catalog = useCatalog(scope, open);
  const trigger = useRef<HTMLButtonElement>(null);
  const skill = catalog.isSuccess ? catalog.data.skill : null;
  return <>
    <Button ref={trigger} variant="outline" disabled={disabled} aria-haspopup="dialog" aria-expanded={open} onClick={() => setOpen(true)}>选择技能</Button>
    {open ? <AccountDialog title="选择技能" drawer onClose={() => setOpen(false)} returnFocusRef={trigger}>
      <div className="grid gap-3">
        {catalog.isFetching ? <p role="status">正在读取可用技能…</p>
          : catalog.isError ? <><p role="alert">技能暂不可用，请重试。</p><Button variant="outline" onClick={() => void catalog.refetch()}>重新读取</Button></>
          : skill ? <><Button variant="outline" className="h-auto justify-start whitespace-normal text-left"
            disabled={!catalog.data?.ready} onClick={() => { onSelect(skill); setOpen(false); }}>
            <span><strong>{skill.name}</strong> · 官方 v{skill.version}<br />{skill.description}</span>
          </Button>{!catalog.data?.ready && <p role="status">当前企业的技能创作模型尚未就绪。</p>}</>
          : <p>当前没有可用的创作技能。</p>}
        {onResume && <Button variant="ghost" onClick={() => { onResume(); setOpen(false); }}>查看本会话的技能草稿</Button>}
      </div>
    </AccountDialog> : null}
  </>;
}

type Quote = { key: string; maximumMicropoints: string; expiresAt: string };
const editorFields = [
  ["displayName", "技能名称"], ["name", "技能标识（小写英文字母、数字和连字符）"],
  ["description", "用途"], ["instructions", "正文"],
] as const;
const generationStatus = {
  QUOTED: "待确认生成", READY: "生成结果待核实", COMPLETE: "已生成草稿",
  FAILED: "未发送模型请求，请核对额度、版本或模型后再试。",
  INVALID: "模型已响应，结果未通过格式检查。",
  UNKNOWN: "原生成结果未能确认。",
} as const;

function authoringError(error: unknown) {
  const code = error instanceof AIWorkbenchError ? error.code : "";
  if (code === "PROPOSAL_STALE") return "报价或技能版本已变化，请重新核对点数。";
  if (code === "INVALID_REQUEST") return "请检查填写内容和格式后重试。";
  if (code === "IDEMPOTENCY_CONFLICT") return "这份草稿已有其他保存操作，请读取原结果。";
  return "本次操作暂未确认，请核实原请求。";
}

export function ChatSkillAuthoring({ scope, conversation, selected, initialRequirement, expectedVersion, expectedDigest, onClose }: {
  scope: AIScope;
  conversation: string;
  selected?: AISkillAuthoringChoice;
  expectedVersion: string | null;
  expectedDigest: string | null;
  initialRequirement: string;
  onClose: () => void;
}) {
  const catalog = useCatalog(scope, !selected);
  const candidate = catalog.data?.ready ? catalog.data.skill : null;
  const matchesLaunch = !expectedVersion && !expectedDigest ||
    candidate && String(candidate.version) === expectedVersion && candidate.digest === expectedDigest;
  const skill = selected ?? (matchesLaunch ? candidate : null);
  const path = "chat/conversations/" + conversation + "/skill-authoring";
  const pointer = "ai-workbench:skill-generation:" + JSON.stringify([scope.userId, scope.organizationId, conversation]);
  const [requirement, setRequirement] = useState(initialRequirement);
  const [quote, setQuote] = useState<Quote>();
  const [originalKey, setOriginalKey] = useState("");
  const [job, setJob] = useState<AISkillAuthoringJob>();
  const [draft, setDraft] = useState<AITitleSkillDraft>();
  const [checked, setChecked] = useState(false);
  const [saveCheck, setSaveCheck] = useState(false);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const action = useRef<AbortController | null>(null);
  const list = useQuery({
    queryKey: ["chat-skill-authoring-list", scope.userId, scope.organizationId, conversation],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-list", method: "GET", scope, path, signal }),
    retry: false, staleTime: 0,
  });

  useEffect(() => {
    let live = true;
    queueMicrotask(() => {
      if (!live) return;
      try {
        const key = sessionStorage.getItem(pointer);
        if (key && isAcquisitionUUID(key)) setOriginalKey(key);
      } catch { setNotice("当前浏览器无法读取原生成请求。"); }
    });
    return () => { live = false; action.current?.abort(); };
  }, [pointer]);

  function useJob(result: AISkillAuthoringJob) {
    setJob(result);
    setRequirement(result.requirement);
    setDraft(result.draft ?? undefined);
    setChecked(false);
    setSaveCheck(result.savePending);
    if (result.state !== "READY" && result.state !== "QUOTED") {
      try { if (sessionStorage.getItem(pointer) === result.key) sessionStorage.removeItem(pointer); } catch { /* server result remains readable */ }
      setOriginalKey(key => key === result.key ? "" : key);
      setQuote(undefined);
    }
  }

  async function perform(run: (signal: AbortSignal) => Promise<void>, failed?: (error: unknown) => void) {
    if (action.current) return;
    const controller = new AbortController();
    action.current = controller;
    setBusy(true);
    setNotice("");
    try { await run(controller.signal); }
    catch (error) {
      if (!controller.signal.aborted) {
        setNotice(authoringError(error));
        failed?.(error);
      }
    } finally {
      if (action.current === controller) action.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  function readQuote() {
    if (!skill || !requirement.trim() || new TextEncoder().encode(requirement).length > 8192) return;
    const key = crypto.randomUUID();
    setQuote(undefined);
    void perform(async signal => {
      const result = await requestAIWorkbench({route:"skill-authoring-quote", method:"POST", scope, path:path+"/quote", key, signal,
        body:{requirement, skillVersion:skill.version, skillDigest:skill.digest}});
      if (!signal.aborted) setQuote(result);
    });
  }

  function generate(key: string, confirmedQuote?: Quote) {
    if (confirmedQuote && Date.now() >= Date.parse(confirmedQuote.expiresAt)) {
      setQuote(undefined);
      setNotice("点数估算已过期，请重新核对点数。");
      return;
    }
    try { sessionStorage.setItem(pointer, key); }
    catch { setNotice("浏览器无法保存恢复位置，请检查存储后重试。"); return; }
    setOriginalKey(key);
    void perform(async signal => {
      const result = await requestAIWorkbench({route:"skill-authoring-generate", method:"POST", scope, path:path+"/"+key+"/generate", signal});
      if (!signal.aborted) { useJob(result); void list.refetch(); }
    }, error => {
      if (error instanceof AIWorkbenchError && error.code === "PROPOSAL_STALE") {
        try { sessionStorage.removeItem(pointer); } catch { /* no confirmed model dispatch */ }
        setOriginalKey("");
        setQuote(undefined);
        void catalog.refetch();
      }
    });
  }

  function checkDraft() {
    if (!draft) return;
    void perform(async signal => {
      await requestAIWorkbench({route:"skill-authoring-validate", method:"POST", scope, path:path+"/validate", body:draft, signal});
      if (!signal.aborted) { setChecked(true); setNotice("格式检查通过，请核对技能内容。"); }
    });
  }

  function save(recovery: boolean) {
    if (!job || (!recovery && (!draft || !checked))) return;
    void perform(async signal => {
      const result = recovery
        ? await requestAIWorkbench({route:"skill-authoring-save-recover", method:"POST", scope, path:path+"/"+job.key+"/recover-save", signal})
        : await requestAIWorkbench({route:"skill-authoring-save", method:"POST", scope, path:path+"/"+job.key+"/save", body:draft, key:crypto.randomUUID(), signal});
      if (!signal.aborted) {
        if (recovery && !result.savePending && !result.savedSkillId) {
          setJob(result);
          setSaveCheck(false);
          setNotice("尚无保存记录，请核对内容后确认保存。");
        } else {
          useJob(result);
        }
        void list.refetch();
      }
    }, error => {
      if (!(error instanceof AIWorkbenchError) || !["INVALID_REQUEST", "NOT_FOUND"].includes(error.code)) setSaveCheck(true);
      void list.refetch();
    });
  }

  const pendingGeneration = originalKey || (job?.state === "READY" ? job.key : "");
  const others = list.data?.drafts.filter(item => item.key !== job?.key);
  return <Card className="grid gap-3 p-4">
    <div className="flex items-center justify-between gap-3"><h2>创建技能</h2><Button variant="ghost" onClick={onClose}>收起</Button></div>
    {notice && <p role="status">{notice}</p>}
    {list.isError && <p role="status">暂时无法读取本会话的技能草稿。<Button variant="outline" onClick={() => void list.refetch()}>重新读取</Button></p>}
    {others?.length ? <div className="flex flex-wrap gap-2">{others.map(item =>
      <Button key={item.key} variant="outline" disabled={busy} onClick={() => useJob(item)}>{item.draft?.displayName || generationStatus[item.state]}</Button>)}</div> : null}
    {pendingGeneration && <Button variant="outline" disabled={busy} onClick={() => generate(pendingGeneration)}>核实原生成请求</Button>}
    {job && <div className="grid gap-2">
      <p role="status">{generationStatus[job.state]} {job.billing === "committed" ? "点数已结算。" : job.billing === "released" ? "本次无扣点。" : "点数状态待核实。"}</p>
      {job.savedSkillId ? <p>已保存到我的技能。<Link className="underline" href={"/workbench/agents/skills/mine?skillId="+job.savedSkillId}>查看我的技能</Link></p>
        : saveCheck || job.savePending ? <Button variant="outline" disabled={busy} onClick={() => save(true)}>核实原保存</Button>
        : draft && <div className="grid gap-3">
          {editorFields.map(([key, label]) => <label className="grid gap-1" key={key}>{label}
            {key === "instructions" ? <Textarea rows={7} value={draft[key]} disabled={busy} onChange={e => { setDraft({...draft, [key]:e.target.value}); setChecked(false); }} />
              : <Input value={draft[key]} disabled={busy} onChange={e => { setDraft({...draft, [key]:e.target.value}); setChecked(false); }} />}
          </label>)}
          <p>确认后的技能会保存为个人私有草稿。</p>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" disabled={busy} onClick={checkDraft}>检查格式</Button>
            <Button disabled={busy || !checked} onClick={() => save(false)}>保存到我的技能</Button>
          </div>
        </div>}
    </div>}
    {!pendingGeneration && !job && <div className="grid gap-3">
      {skill ? <p>{skill.name} · 官方 v{skill.version}。创建一份仅处理商品标题的文字技能。</p> : <p>当前没有可用的创作技能，可以查看原草稿。</p>}
      <label className="grid gap-1">技能需求<Textarea rows={3} value={requirement} disabled={busy || !skill} onChange={e => { setRequirement(e.target.value); setQuote(undefined); }} /></label>
      {quote && <p role="status">本次最多使用 {formatMicropoints(quote.maximumMicropoints)} AI 点数，按实际用量结算。</p>}
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" disabled={busy || !skill || !requirement.trim()} onClick={readQuote}>核对点数</Button>
        <Button disabled={busy || !quote} onClick={() => quote && generate(quote.key, quote)}>生成草稿</Button>
      </div>
    </div>}
    {job && !pendingGeneration && !saveCheck && !job.savePending && <Button variant="ghost" disabled={busy} onClick={() => {
      setJob(undefined); setQuote(undefined); setDraft(undefined); setChecked(false);
    }}>准备另一个草稿（生成时重新确认点数）</Button>}
  </Card>;
}
