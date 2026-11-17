"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { AIWorkbenchError, requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import { type AIAuthoringPreview, type AISkillAuthoringChoice, type AISkillAuthoringJob, type AITitleSkillDraft } from "@/lib/contracts/ai-workbench";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { AccountDialog } from "../account/account-dialog";

function useCatalog(scope: AIScope, authorizationKey: string, enabled: boolean) {
  return useQuery({
    queryKey: ["chat-skill-authoring-catalog", scope.userId, scope.organizationId, authorizationKey],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-catalog", method: "GET", path: "chat/conversations/skill-authoring", scope, signal }),
    enabled, retry: false, staleTime: 0,
  });
}

export function ChatSkillDrawer({ scope, authorizationKey, disabled, onSelect, onResume }: {
  scope: AIScope;
  authorizationKey: string;
  disabled?: boolean;
  onSelect: (skill: AISkillAuthoringChoice) => void;
  onResume?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const catalog = useCatalog(scope, authorizationKey, open);
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

type Quote = {key: string; quote: string; expiresAt: string};
type PendingGeneration = {key: string};
const editorFields = [
  ["displayName", "技能名称"], ["name", "技能标识（小写英文字母、数字和连字符）"],
  ["description", "用途"], ["instructions", "正文"],
] as const;
const generationStatus = {
  QUOTED: "待确认生成", READY: "生成结果待核实", COMPLETE: "已生成草稿",
  FAILED: "生成未开始，请检查模型是否可用或额度是否充足。",
  INVALID: "草稿格式未通过，可以修改需求后重试。",
  UNKNOWN: "本次生成结果未能确认。",
} as const;

function authoringError(error: unknown) {
  const code = error instanceof AIWorkbenchError ? error.code : "";
  if (code === "PROPOSAL_STALE") return "原确认已失效，请查看当前草稿后再试。";
  if (code === "INVALID_REQUEST") return "请检查填写内容和格式后重试。";
  if (code === "IDEMPOTENCY_CONFLICT") return "草稿已在其他页面更新或保存，请读取原结果。";
  return "本次操作暂未确认，请核实原请求。";
}

function readGenerationPointer(pointer: string): PendingGeneration | undefined {
  const raw = sessionStorage.getItem(pointer);
  if (raw && isAcquisitionUUID(raw)) return {key: raw};
}

function quoteExpired(quote: Quote): boolean {
  return Date.now() >= Date.parse(quote.expiresAt);
}

export function ChatSkillAuthoring({ scope, memberId, authorizationKey, conversation, selected, initialRequirement, expectedVersion, expectedDigest, onClose }: {
  scope: AIScope;
  memberId: string;
  authorizationKey: string;
  conversation: string;
  selected?: AISkillAuthoringChoice;
  expectedVersion: string | null;
  expectedDigest: string | null;
  initialRequirement: string;
  onClose: () => void;
}) {
  const catalog = useCatalog(scope, authorizationKey, !selected);
  const candidate = catalog.data?.ready ? catalog.data.skill : null;
  const matchesLaunch = !expectedVersion && !expectedDigest ||
    candidate && String(candidate.version) === expectedVersion && candidate.digest === expectedDigest;
  const skill = selected ?? (matchesLaunch ? candidate : null);
  const path = "chat/conversations/" + conversation + "/skill-authoring";
  const memberScope = useMemo(() => ({userId: scope.userId, organizationId: scope.organizationId}), [scope.userId, scope.organizationId]);
  const pointer = "ai-workbench:skill-generation:" + JSON.stringify([scope.userId, scope.organizationId, memberId, conversation]);
  const [requirement, setRequirement] = useState(initialRequirement);
  const [revisionRequest, setRevisionRequest] = useState("");
  const [prepared, setPrepared] = useState<{input: AIAuthoringPreview; quote?: Quote}>();
  const [prepRetry, setPrepRetry] = useState(0);
  const [original, setOriginal] = useState<PendingGeneration>();
  const [pointerReady, setPointerReady] = useState(false);
  const [job, setJob] = useState<AISkillAuthoringJob>();
  const [draft, setDraft] = useState<AITitleSkillDraft>();
  const [validation, setValidation] = useState<{draft: AITitleSkillDraft; valid: boolean; unavailable?: boolean}>();
  const [validationRetry, setValidationRetry] = useState(0);
  const [saveCheck, setSaveCheck] = useState(false);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const action = useRef<AbortController | null>(null);
  const lastConfirmation = useRef<{input: AIAuthoringPreview; quote: Quote} | undefined>(undefined);
  const list = useQuery({
    queryKey: ["chat-skill-authoring-list", scope.userId, scope.organizationId, memberId, authorizationKey, conversation],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-list", method: "GET", scope, path, signal }),
    retry: false, staleTime: 0,
  });

  useEffect(() => {
    let live = true;
    queueMicrotask(() => {
      if (!live) return;
      try { setOriginal(readGenerationPointer(pointer)); }
      catch { setNotice("当前浏览器无法读取原生成请求。"); }
      setPointerReady(true);
    });
    return () => { live = false; action.current?.abort(); };
  }, [pointer]);

  const pending = useMemo(() => original ?? (job?.state === "READY" ? {key: job.generationKey ?? job.key} : undefined), [original, job]);
  const editable = Boolean(draft && job && !job.savedSkillId && !job.savePending && !saveCheck && !pending);
  const input = useMemo<AIAuthoringPreview | undefined>(() => {
    if (!skill || !pointerReady || pending || saveCheck || job?.savePending || job?.savedSkillId) return;
    const text = (job && draft ? revisionRequest : requirement).trim();
    if (!text || new TextEncoder().encode(text).length > 8192) return;
    const next = {requirement: text, skillVersion: skill.version, skillDigest: skill.digest};
    if (!job) return next;
    if (skill.version !== 2 || !draft) return;
    return {...next, root: job.key, baseRevision: job.revision ?? job.key, snapshot: draft};
  }, [skill, pointerReady, pending, saveCheck, job, draft, revisionRequest, requirement]);

  useEffect(() => {
    if (!input || busy) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void requestAIWorkbench({route: "skill-authoring-preview", method: "POST", scope: memberScope,
        path: path + "/preview", body: input, signal: controller.signal})
        .then(quote => { if (!controller.signal.aborted) setPrepared({input, quote}); })
        .catch(() => { if (!controller.signal.aborted) setPrepared({input}); });
    }, 250);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [input, busy, memberScope, path, prepRetry]);

  useEffect(() => {
    if (!editable || !draft || busy) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void requestAIWorkbench({route: "skill-authoring-validate", method: "POST", scope: memberScope,
        path: path + "/validate", body: draft, signal: controller.signal})
        .then(() => { if (!controller.signal.aborted) setValidation({draft, valid: true}); })
        .catch(error => {
          if (!controller.signal.aborted) setValidation({draft, valid: false, unavailable: !(error instanceof AIWorkbenchError) || error.code !== "INVALID_REQUEST"});
        });
    }, 250);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [draft, editable, busy, memberScope, path, validationRetry]);

  function clearGeneration(key: string) {
    try { if (readGenerationPointer(pointer)?.key === key) sessionStorage.removeItem(pointer); } catch { /* original server result remains readable */ }
    setOriginal(current => current?.key === key ? undefined : current);
    if (lastConfirmation.current?.quote.key === key) lastConfirmation.current = undefined;
  }

  function displayJob(result: AISkillAuthoringJob) {
    setJob(result);
    setDraft(result.draft ?? undefined);
    setRevisionRequest("");
    setValidation(undefined);
    setSaveCheck(result.savePending);
    if (result.state !== "READY" && result.state !== "QUOTED") clearGeneration(result.generationKey ?? result.key);
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

  function startGeneration(confirmed: {input: AIAuthoringPreview; quote: Quote}) {
    const {quote} = confirmed;
    if (quoteExpired(quote)) {
      setPrepared(undefined);
      setPrepRetry(retry => retry + 1);
      setNotice("原确认已过期，正在重新准备。");
      return;
    }
    const next = {key: quote.key};
    try { sessionStorage.setItem(pointer, next.key); }
    catch { setNotice("浏览器无法保存恢复位置，请检查存储后重试。"); return; }
    setOriginal(next);
    lastConfirmation.current = confirmed;
    generate(next);
  }

  function generate(request: PendingGeneration) {
    const confirmed = lastConfirmation.current;
    const confirmNow = confirmed?.quote.key === request.key && !quoteExpired(confirmed.quote);
    void perform(async signal => {
      const result = confirmNow
        ? await requestAIWorkbench({route: "skill-authoring-confirm", method: "POST", scope, path: path + "/" + request.key + "/confirm", signal,
            body: {...confirmed.input, quote: confirmed.quote.quote}})
        : request.key[14] === "7"
          ? await requestAIWorkbench({route: "skill-authoring-recover-generation", method: "POST", scope,
              path: path + "/" + request.key + "/recover-generation", signal})
          : await requestAIWorkbench({route: "skill-authoring-generate", method: "POST", scope, path: path + "/" + request.key + "/generate", signal});
      if (!signal.aborted) {displayJob(result); void list.refetch();}
    }, error => {
      if (error instanceof AIWorkbenchError && error.code === "PROPOSAL_STALE") {
        if (confirmNow) {
          lastConfirmation.current = undefined;
          setNotice("请先核实原生成请求。");
          return;
        }
        clearGeneration(request.key);
        setPrepared(undefined);
        setPrepRetry(retry => retry + 1);
        void catalog.refetch();
        void list.refetch();
      }
    });
  }

  const checked = validation?.draft === draft && validation?.valid;
  function save(recovery: boolean) {
    if (!job || (!recovery && (!draft || !checked))) return;
    void perform(async signal => {
      const result = recovery
        ? await requestAIWorkbench({route: "skill-authoring-save-recover", method: "POST", scope, path: path + "/" + job.key + "/recover-save", signal})
        : await requestAIWorkbench({route: "skill-authoring-save-package", method: "POST", scope, path: path + "/" + job.key + "/save-package",
            body: {revision: job.revision ?? job.key, draft}, key: crypto.randomUUID(), signal});
      if (!signal.aborted) {
        if (recovery && !result.savePending && !result.savedSkillId && result.revision === job.revision) {
          setJob(result);
          setSaveCheck(false);
          setNotice("尚无保存记录，请核对内容后确认保存。");
        } else {
          displayJob(result);
        }
        void list.refetch();
      }
    }, error => {
      if (!(error instanceof AIWorkbenchError) || !["INVALID_REQUEST", "NOT_FOUND"].includes(error.code)) setSaveCheck(true);
      void list.refetch();
    });
  }

  const currentPrep = input && prepared?.input === input ? prepared : undefined;
  const others = list.data?.drafts.filter(item => (item.generationKey ?? item.key) !== (job?.generationKey ?? job?.key));
  const genButton = (label: string) => <Button disabled={busy || !currentPrep?.quote} onClick={() => {
    if (currentPrep?.quote) startGeneration({input: currentPrep.input, quote: currentPrep.quote});
  }}>{label}</Button>;
  const prepFailure = input && currentPrep && !currentPrep.quote
    ? <p role="status">暂时无法准备生成。 <Button variant="outline" onClick={() => {setPrepared(undefined); setPrepRetry(value => value + 1);}}>重试</Button></p>
    : null;
  return <Card className="grid gap-3 p-4">
    <div className="flex items-center justify-between gap-3"><h2>创建技能</h2><Button variant="ghost" onClick={onClose}>收起</Button></div>
    {notice && <p role="status">{notice}</p>}
    {list.isError && <p role="status">暂时无法读取本会话的技能草稿。 <Button variant="outline" onClick={() => void list.refetch()}>重新读取</Button></p>}
    {others?.length ? <div className="flex flex-wrap gap-2">{others.map(item =>
      <Button key={item.generationKey ?? item.key} variant="outline" disabled={busy || Boolean(pending)} onClick={() => displayJob(item)}>
        {item.state === "READY" ? "查看待核实的生成" : item.draft?.displayName || generationStatus[item.state]}
      </Button>)}</div> : null}
    {pending && <Button variant="outline" disabled={busy} onClick={() => generate(pending)}>核实原生成请求</Button>}
    {job && <div className="grid gap-2">
      <p role="status">{job.superseded ? "这次修改未覆盖当前草稿。" : generationStatus[job.state]}</p>
      {job.savedSkillId ? <p>已保存到我的技能。 <Link className="underline" href={"/workbench/agents/skills/mine?skillId=" + job.savedSkillId}>查看我的技能</Link></p>
        : saveCheck || job.savePending ? <Button variant="outline" disabled={busy} onClick={() => save(true)}>核实原保存</Button>
        : editable && draft && <div className="grid gap-3">
          {editorFields.map(([key, label]) => <label className="grid gap-1" key={key}>{label}
            {key === "instructions" ? <Textarea rows={7} value={draft[key]} disabled={busy} onChange={event => setDraft({...draft, [key]: event.target.value})} />
              : <Input value={draft[key]} disabled={busy} onChange={event => setDraft({...draft, [key]: event.target.value})} />}
          </label>)}
          {draft.script && <div className="grid gap-2">
            <p>包含 Python 脚本</p>
            {draft.script.description && <p>{draft.script.description}</p>}
            <details>
              <summary className="cursor-pointer">查看代码</summary>
              <label className="grid gap-1">脚本用途<Input value={draft.script.description} disabled={busy}
                onChange={event => {if (draft.script) setDraft({...draft, script: {...draft.script, description: event.target.value}});}} /></label>
              <label className="grid gap-1">Python 代码<Textarea rows={7} spellCheck={false} value={draft.script.source} disabled={busy}
                onChange={event => setDraft({...draft, script: {source: event.target.value, description: ""}})} /></label>
            </details>
            <Button variant="ghost" disabled={busy} onClick={() => setDraft({...draft, script: undefined})}>移除脚本</Button>
          </div>}
          {skill?.version === 2 && <div className="grid gap-2">
            <label className="grid gap-1">让 Chat 修改<Textarea rows={2} value={revisionRequest} disabled={busy} onChange={event => setRevisionRequest(event.target.value)} /></label>
            {revisionRequest.trim() && <p>本次修改会使用当前草稿{draft.script ? "和其中的 Python 代码" : ""}。</p>}
            {genButton("生成修改")}
            {prepFailure}
          </div>}
          {validation && validation.draft === draft && !validation.valid && <p role="status">
            {validation.unavailable ? "暂时无法检查格式。" : "格式未通过，请检查名称、正文或脚本长度。"}
            {validation.unavailable && <Button variant="outline" onClick={() => setValidationRetry(value => value + 1)}>重新检查</Button>}
          </p>}
          <p>确认后保存为个人私有草稿。{draft.script ? "包含 Python 脚本。" : ""}</p>
          <div><Button disabled={busy || !checked} onClick={() => save(false)}>保存到我的技能</Button></div>
        </div>}
    </div>}
    {!pending && !job && <div className="grid gap-3">
      {skill ? <p>{skill.name} · 官方 v{skill.version}。创建一份个人标题技能。</p> : <p>当前没有可用的创作技能，可以查看原草稿。</p>}
      <label className="grid gap-1">技能需求<Textarea rows={3} value={requirement} disabled={busy || !skill} onChange={event => setRequirement(event.target.value)} /></label>
      {genButton("生成草稿")}
      {prepFailure}
    </div>}
    {job && !pending && !saveCheck && !job.savePending && <Button variant="ghost" disabled={busy} onClick={() => {
      setJob(undefined); setPrepared(undefined); setDraft(undefined); setValidation(undefined); setRevisionRequest("");
    }}>准备另一个草稿</Button>}
  </Card>;
}
