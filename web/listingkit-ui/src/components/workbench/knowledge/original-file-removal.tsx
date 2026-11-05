"use client";

import Link from "next/link";
import { useEffect, useId, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { KnowledgeError, knowledgeRequest, originalFileSchema, type KnowledgeScope, type OriginalFile } from "@/lib/api/knowledge";

type Intent = { key: string; file: OriginalFile };

function Facts({ file, confirm = false, status }: { file: OriginalFile; confirm?: boolean; status?: string }) {
  return <dl className="knowledge-original-facts">
    <div><dt>资料编号</dt><dd>{file.sourceId}</dd></div>
    <div><dt>资料版本</dt><dd>v{file.revisionNumber}</dd></div>
    <div><dt>原文件大小</dt><dd>{file.bytes.toLocaleString("zh-CN")} 字节</dd></div>
    {confirm && <div><dt>资料状态</dt><dd>已停用</dd></div>}
    {status && <div><dt>容量状态</dt><dd>{status}</dd></div>}
  </dl>;
}

function Confirmation({ file, close, submit }: { file: OriginalFile; close: () => void; submit: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const cancel = useRef<HTMLButtonElement>(null);
  const title = useId();
  const warning = useId();
  useEffect(() => {
    const previous = document.activeElement;
    const element = dialog.current;
    element?.showModal();
    cancel.current?.focus();
    return () => {
      if (element?.open) element.close();
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
    };
  }, []);

  return <dialog ref={dialog} className="knowledge-original-dialog" aria-labelledby={title} aria-describedby={warning}
    onCancel={event => { event.preventDefault(); close(); }}>
    <h3 id={title}>永久删除原文件？</h3>
    <p id={warning} className="knowledge-original-warning">原文件将永久删除，无法恢复。</p>
    <Facts file={file} confirm />
    <p className="console-description">只有确认原文件已删除并取得容量回执后，才显示已释放。</p>
    <div className="knowledge-actions">
      <Button ref={cancel} variant="outline" onClick={close}>取消</Button>
      <Button variant="destructive" onClick={submit}>确认永久删除</Button>
    </div>
  </dialog>;
}

const copy = {
  ready: ["原文件管理", "符合条件的原文件可由当前企业的知识库管理员永久删除。", "按资料编号、版本和大小核对原文件。已停用资料的正文仍不可查看。"],
  pending: ["删除待核实", "原文件删除结果待核实。确认前，这份文件仍计入企业已用容量。", "继续核实会接着处理这一次操作。"],
  capacity: ["容量待确认", "原文件已确认删除，容量释放结果尚未确认。容量请以「资源与额度」的实际读数为准。", "继续核实只查询或接续原容量释放。"],
  released: ["已释放", "", "该资料保持停用，操作记录可核对。"],
};

export function OriginalFileRemoval({ scope, sourceId, onAuthorityError }: {
  scope: KnowledgeScope;
  sourceId: string;
  onAuthorityError: (error: unknown) => void;
}) {
  const client = useQueryClient();
  const path = "knowledge-sources/" + sourceId + "/original-file";
  const queryKey = ["knowledge-original-file", scope.userId, scope.organizationId, sourceId];
  const request = useRef<AbortController | null>(null);
  const [confirmation, setConfirmation] = useState<OriginalFile | null>(null);
  const [intent, setIntent] = useState<Intent | null>(null);
  const [busy, setBusy] = useState(false);
  const [readingAgain, setReadingAgain] = useState(false);
  const [notice, setNotice] = useState("");
  const title = useId();
  useEffect(() => () => request.current?.abort(), []);

  const read = (signal: AbortSignal) => knowledgeRequest(scope, path, originalFileSchema, { signal });
  const current = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      try { return await read(signal); }
      catch (error) { if (!signal.aborted) onAuthorityError(error); throw error; }
    },
    retry: false, gcTime: 0, refetchOnWindowFocus: false, refetchOnReconnect: false,
  });
  const file = current.data;

  async function advance(confirmed: Intent | null, reconcile: boolean) {
    if (request.current) return;
    const controller = new AbortController();
    const { signal } = controller;
    request.current = controller;
    setBusy(true);
    setNotice("");
    if (!reconcile) setIntent(confirmed);
    try {
      await client.cancelQueries({ queryKey, exact: true });
      signal.throwIfAborted();
      const target = reconcile ? await read(signal) : confirmed?.file;
      let result = target;
      if (target?.operationId && target.state !== "RELEASED") {
        result = await knowledgeRequest(scope, path + "/" + target.operationId + "/continue", originalFileSchema,
          { method: "POST", headers: { "Idempotency-Key": target.operationId }, signal });
      } else if (confirmed && target?.canDelete && target.revisionId === confirmed.file.revisionId &&
          target.sourceVersion === confirmed.file.sourceVersion) {
        result = await knowledgeRequest(scope, path + "/" + target.revisionId + "/delete", originalFileSchema,
          { method: "POST", headers: { "Idempotency-Key": confirmed.key, "If-Match": '"' + target.sourceVersion + '"' }, signal });
      } else if (!target?.operationId) {
        setNotice("原文件状态已变化，请重新核对。");
      }
      if (signal.aborted) return;
      client.setQueryData(queryKey, result);
      setIntent(null);
    } catch (error) {
      if (signal.aborted) return;
      onAuthorityError(error);
      if (error instanceof KnowledgeError && error.status < 500 && error.code !== "OUTCOME_UNKNOWN" && !reconcile) {
        setIntent(null);
        setNotice("原文件状态或权限已变化，请重新核对。");
        void client.invalidateQueries({ queryKey, exact: true });
      } else {
        setNotice("暂时无法核实，请稍后继续。");
      }
    } finally {
      request.current = null;
      if (!signal.aborted) setBusy(false);
    }
  }

  const readUnavailable = current.error instanceof KnowledgeError &&
    current.error.status >= 500 && current.error.code !== "REQUEST_CANCELLED";
  if (!intent && !file?.operationId && (readUnavailable || readingAgain)) {
    return <section className="knowledge-original-panel" aria-labelledby={title}>
      <h4 id={title}>原文件管理</h4>
      <p role="status">{readingAgain ? "正在读取原文件状态…" : "暂时无法读取原文件状态。"}</p>
      <div className="knowledge-original-actions">
        <Button variant="outline" disabled={readingAgain} onClick={() => {
          setReadingAgain(true);
          void current.refetch().finally(() => setReadingAgain(false));
        }}>重新读取原文件状态</Button>
      </div>
    </section>;
  }

  if (!file || (!file.canDelete && !file.operationId && !intent)) return null;
  const state = file.state === "RELEASED" ? "released" : file.state === "OBJECT_ABSENCE_PROVEN" ? "capacity"
    : intent || file.operationId ? "pending" : "ready";
  const [label, description, note] = copy[state];
  return <section className="knowledge-original-panel" data-state={state} aria-labelledby={title}>
    <div aria-live="polite">
      {file.operationId && <p className="knowledge-original-operation">操作单号 {file.operationId}</p>}
      <h4 id={title}>{label}</h4>
      <p>{state === "released" ? "已收到容量释放回执。该文件的 " + file.bytes.toLocaleString("zh-CN") + " 字节已从企业已用容量和知识库分类用量中扣除。" : description}</p>
    </div>
    <Facts file={file} status={state === "released" ? "已释放容量" : state === "capacity" ? "释放待确认" : "计入已用容量"} />
    <div className="knowledge-original-actions">
      <p className="console-description">{note}</p>
      {state === "released"
        ? <Button asChild variant="outline"><Link href="/workbench/account/organization/resources" prefetch={false}>查看资源与额度</Link></Button>
        : state === "ready"
          ? <Button variant="outline" disabled={busy || current.isError || current.isFetching} className="knowledge-original-delete" onClick={() => setConfirmation(file)}>永久删除原文件</Button>
          : <Button variant="outline" disabled={busy} onClick={() => void advance(intent, true)}>{busy ? "正在核实…" : "继续核实"}</Button>}
    </div>
    {notice && <p role="status" className="knowledge-reason">{notice}</p>}
    {confirmation && <Confirmation file={confirmation} close={() => setConfirmation(null)} submit={() => {
      setConfirmation(null);
      void advance({ key: crypto.randomUUID(), file: confirmation }, false);
    }} />}
  </section>;
}
