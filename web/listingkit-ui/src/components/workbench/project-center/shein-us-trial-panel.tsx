"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import type { ProjectScope } from "@/lib/api/project-center";
import { requestSheinTrial, SheinTrialError } from "@/lib/api/shein-us-trial";
import type { SheinTrialState } from "@/lib/contracts/shein-us-trial";
import styles from "./research-panel.module.css";

export function SheinUSTrialPanel({ scope, projectId, canWrite }: { scope: ProjectScope; projectId: string; canWrite: boolean }) {
  const [state, setState] = useState<SheinTrialState | null>(null);
  const [visible, setVisible] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [uncertainApproval, setUncertainApproval] = useState("");
  const lifetime = useRef<AbortController | null>(null);
  const flight = useRef(false);
  const confirmation = useRef<HTMLButtonElement>(null);
  const range = useRef<HTMLButtonElement>(null);
  const restoreRangeFocus = useRef(false);
  const { registerOrganizationSwitchGuard } = useWorkbenchContext();

  useEffect(() => {
    const controller = new AbortController();
    lifetime.current = controller;
    void requestSheinTrial(scope, projectId, controller.signal, "view").then((result) => {
      if (!controller.signal.aborted) { setState(result); setVisible(true); }
    }).catch(() => { /* Show the entry only after an authorized original read. */ });
    return () => controller.abort();
  }, [scope, projectId]);
  useEffect(() => registerOrganizationSwitchGuard(() => !flight.current), [registerOrganizationSwitchGuard]);
  useEffect(() => {
    if (confirming) confirmation.current?.focus();
    else if (restoreRangeFocus.current) {
      restoreRangeFocus.current = false;
      range.current?.focus();
    }
  }, [confirming]);

  const canSubmit = canWrite && state?.state === "NOT_STARTED" && state.approvalId !== uncertainApproval;
  const canResume = canWrite && state?.state === "CONFIRMED" && state.publishedCount === undefined;

  async function run(action: "view" | "submit" | "reconcile") {
    const controller = lifetime.current;
    if (!controller || controller.signal.aborted || flight.current) return;
    if (action === "submit" && !(confirming && canSubmit) && !canResume) return;
    if (action === "reconcile" && state?.state !== "UNKNOWN") return;
    const approval = state?.approvalId;
    flight.current = true;
    setBusy(true);
    setConfirming(false);
    setMessage("");
    try {
      const result = await requestSheinTrial(scope, projectId, controller.signal, action, approval);
      if (controller.signal.aborted) return;
      if (action === "view" && result.state === "CONFIRMED" && state?.state === "CONFIRMED" &&
          state.publishedCount !== undefined && result.approvalId === state.approvalId &&
          result.requestId === state.requestId && result.credits === state.credits) {
        setState({ ...result, publishedCount: state.publishedCount });
      } else setState(result);
      if (result.state === "NOT_STARTED" && result.approvalId === uncertainApproval)
        setMessage("还不能确认这次提交，请继续查看原单。");
      else setUncertainApproval("");
    } catch (error) {
      if (controller.signal.aborted) return;
      setState(null);
      const code = error instanceof SheinTrialError ? error.code : "";
      if (["AUTHENTICATION_REQUIRED", "FORBIDDEN", "IDENTITY_CONTEXT_CHANGED", "ORGANIZATION_CONTEXT_CHANGED"].includes(code)) {
        setVisible(false);
        setUncertainApproval("");
      } else if (action === "submit" && code === "OUTCOME_UNKNOWN") {
        setUncertainApproval(approval ?? "");
        setMessage("还不能确认这次提交，请先查看原单。");
      } else {
        setMessage("暂时无法确认原单，请查看当前状态。");
      }
    } finally {
      flight.current = false;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  if (!visible) return null;
  return <section className={styles.evidence} aria-label="SHEIN 美国站首试" aria-busy={busy}>
    <h4>SHEIN 美国站首试</h4>
    <div aria-live="polite">
      {message && <p role="status">{message}</p>}
      {state?.state === "UNKNOWN" && <p>用量待核对，暂留 {state.reservedCredits} credits。</p>}
      {state?.state === "OVER_LIMIT" && <p>实际用量 {state.credits} credits，超出首试额度。</p>}
      {state?.state === "CONFIRMED" && <p>实际用量 {state.credits} credits。{state.publishedCount !== undefined ? `已保存 ${state.publishedCount} 张商品卡。` : "原单已核账。"}</p>}
      {state && "requestId" in state && state.requestId && <p>原请求编号 {state.requestId}</p>}
    </div>
    {confirming && <p>取 SHEIN 美国站类目 1727 第一页，最多 20 张商品卡。只提交一次，最多 25 credits，新增现金费用 US$0。</p>}
    <div className={styles.actions}>
      {confirming ? <>
        <Button ref={confirmation} disabled={busy || !canSubmit} onClick={() => void run("submit")}>确认提交一次</Button>
        <Button variant="outline" onClick={() => { restoreRangeFocus.current = true; setConfirming(false); }}>取消</Button>
      </> : <>
        {canSubmit && <Button ref={range} disabled={busy} onClick={() => setConfirming(true)}>核对试取范围</Button>}
        {canResume && <Button disabled={busy} onClick={() => void run("submit")}>继续保存原单资料</Button>}
        {state?.state === "UNKNOWN" && <Button disabled={busy} onClick={() => void run("reconcile")}>核对原账</Button>}
        <Button variant="outline" disabled={busy} onClick={() => void run("view")}>查看原单</Button>
      </>}
    </div>
  </section>;
}
