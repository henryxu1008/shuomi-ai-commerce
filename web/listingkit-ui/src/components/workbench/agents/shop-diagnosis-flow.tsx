"use client";
import Link from "next/link";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableHeader,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import {
  configurationRequest,
  type ConfigurationScope,
} from "@/lib/api/agent-configuration";
import { MarketAPIError } from "@/lib/api/supply-market";
import {
  inspectDiagnosis,
  prepareDiagnosis,
  executeDiagnosis,
  readDiagnosisRun,
} from "@/lib/api/shop-diagnosis";
import {
  diagnosisIntentSchema,
  diagnosisInputSchema,
  diagnosisSavedPreparationSchema,
  diagnosisRecentSchema,
  type DiagnosisIntent,
  type DiagnosisInspection,
  type DiagnosisPreparation,
} from "@/lib/contracts/shop-diagnosis";
import {
  cockpitCapabilities,
  moneyText,
} from "@/lib/contracts/operations-cockpit";
import { cockpitRequest, cockpitErrorText } from "@/lib/api/operations-cockpit";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import { executionModeViewSchema } from "@/lib/contracts/execution-mode";

import { useResourcePending } from "../resources/resource-pending";
import { configurationError } from "./agent-configuration-hooks";
import { executionModeLabels } from "./execution-mode-settings";
import styles from "./agents.module.css";

export function ShopDiagnosisFlow({
  scope,
  entry,
  nonce,
}: {
  scope: ConfigurationScope;
  entry: CatalogEntry;
  nonce: number;
}) {
  const context = useWorkbenchContext();
  const queries = useQueryClient();
  const pending = useResourcePending(
    {
      expectedUserId: scope.userId,
      expectedOrganizationId: scope.organizationId,
    },
    ["shop-diagnosis"],
    diagnosisIntentSchema,
    { storage: "local", maxLength: 16384 },
  );
  const [selected, setSelected] = useState(""),
    [startDate, setStartDate] = useState(""),
    [endDate, setEndDate] = useState(""),
    [goal, setGoal] = useState(
      "请根据本店已录入经营事实，用中文提出可核对的建议并说明缺失信息。",
    ),
    [facts, setFacts] = useState<DiagnosisInspection>(),
    [busy, setBusy] = useState(false),
    [failure, setFailure] = useState<unknown>(),
    [after, setAfter] = useState(""),
    [chosenRun, setChosenRun] = useState<{
      contextId: string;
      key: string;
    } | null>(null);
  const active = useRef(true),
    flight = useRef(false),
    controller = useRef<AbortController | null>(null);
  const cacheKey =
    "shop.operations-diagnosis.prepared:" +
    JSON.stringify([scope.userId, scope.organizationId]);
  const cacheRaw = useSyncExternalStore(
    (listener) => {
      window.addEventListener("storage", listener);
      window.addEventListener("resource-pending", listener);
      return () => {
        window.removeEventListener("storage", listener);
        window.removeEventListener("resource-pending", listener);
      };
    },
    () => {
      try {
        return localStorage.getItem(cacheKey) ?? "";
      } catch {
        return "unreadable";
      }
    },
    () => "",
  );
  let saved: DiagnosisPreparation | undefined,
    cacheFailed = false;
  if (cacheRaw) {
    try {
      if (cacheRaw.length > 16384) throw new Error();
      saved = diagnosisSavedPreparationSchema.parse(JSON.parse(cacheRaw));
    } catch {
      cacheFailed = true;
    }
  }
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
      controller.current?.abort();
    };
  }, []);
  const register = context.registerOrganizationSwitchGuard;
  useEffect(
    () => register(() => !flight.current && !pending.command && !pending.error),
    [register, pending.command, pending.error],
  );
  const keys = ["shop-diagnosis", scope.userId, scope.organizationId];
  const mode = useQuery({
    queryKey: [...keys, "mode", nonce],
    queryFn: ({ signal }) =>
      configurationRequest(
        scope,
        "execution-modes/registered_agent/shop.operations-diagnosis.agent",
        executionModeViewSchema,
        { signal },
      ),
    enabled: entry.agent.activation !== "NOT_ENABLED",
  });
  const recent = useQuery({
    queryKey: [...keys, "recent", after, nonce],
    queryFn: ({ signal }) =>
      configurationRequest(
        scope,
        "shop.operations-diagnosis.agent/recent-runs?pageSize=20" +
          (after ? "&cursor=" + after : ""),
        diagnosisRecentSchema,
        { signal },
      ),
    enabled: entry.canReadRuns,
  });
  const shops = useQuery({
    queryKey: [...keys, "shops", nonce],
    queryFn: ({ signal }) =>
      cockpitRequest(scope, "capabilities", cockpitCapabilities, { signal }),
  });
  const validInput =
    diagnosisInputSchema.safeParse({
      contextId: "11111111-1111-4111-8111-111111111111",
      storeId: selected,
      startDate,
      endDate,
    }).success &&
    !!shops.data?.access.storesRead &&
    !!shops.data?.access.adviceRead;
  const currentPointer =
    chosenRun ??
    (pending.command && pending.command.kind !== "prepare"
      ? { contextId: pending.command.contextId, key: pending.command.key }
      : saved
        ? {
            contextId: saved.prepared.contextId,
            key: saved.prepared.requestKey,
          }
        : null);
  const run = useQuery({
    queryKey: [...keys, "run", currentPointer?.contextId, currentPointer?.key],
    queryFn: ({ signal }) =>
      readDiagnosisRun(
        scope,
        currentPointer!.contextId,
        currentPointer!.key,
        signal,
      ),
    enabled: !!currentPointer,
    refetchInterval: (q) => (q.state.data?.phase === "running" ? 3000 : false),
    refetchIntervalInBackground: false,
  });
  const locked =
    busy || !pending.ready || pending.error || cacheFailed || !!pending.command;
  const textReady = entry.capabilities.some(
    (c) => c.id === "text.generate" && c.readiness === "AVAILABLE",
  );
  const input = () =>
    diagnosisInputSchema.parse({
      contextId: crypto.randomUUID(),
      storeId: selected,
      startDate,
      endDate,
    });
  const preparedFactsMatch = !!saved && !!facts &&
    facts.ref.StoreID === saved.prepared.inspectionRef.StoreID &&
    facts.ref.StartDate === saved.prepared.inspectionRef.StartDate &&
    facts.ref.EndDate === saved.prepared.inspectionRef.EndDate &&
    facts.ref.EvidenceDigest === saved.prepared.inspectionRef.EvidenceDigest;
  function savePreparation(value: DiagnosisPreparation) {
    const raw = JSON.stringify(diagnosisSavedPreparationSchema.parse(value));
    localStorage.setItem(cacheKey, raw);
    if (localStorage.getItem(cacheKey) !== raw)
      throw new Error("无法保留原准备结果");
    window.dispatchEvent(new Event("resource-pending"));
  }
  async function inspect() {
    if (flight.current) return;
    flight.current = true;
    setBusy(true);
    setFailure(undefined);
    const abort = new AbortController();
    controller.current = abort;
    try {
      const original = saved?.input;
      const result = await inspectDiagnosis(scope, original ? {
        contextId: original.contextId, storeId: original.storeId,
        startDate: original.startDate, endDate: original.endDate,
      } : input(), abort.signal);
      if (active.current) {
        setFacts(result);
        setChosenRun(null);
      }
    } catch (e) {
      if (active.current) setFailure(e);
    } finally {
      flight.current = false;
      if (active.current) setBusy(false);
    }
  }
  async function send(intent: DiagnosisIntent) {
    if (flight.current || !active.current || !pending.ready) return;
    if (intent.kind === "start" && !pending.command && !preparedFactsMatch) {
      setFailure(new Error("请先核对本次准备结果对应的经营事实。"));
      return;
    }
    flight.current = true;
    setBusy(true);
    setFailure(undefined);
    const abort = new AbortController();
    controller.current = abort;
    try {
      pending.persist(intent);
      if (intent.kind === "prepare") {
        const prepared = await prepareDiagnosis(scope, intent, abort.signal);
        if (
          prepared.contextId !== intent.body.contextId ||
          prepared.requestKey !== intent.key
        )
          throw new MarketAPIError("OUTCOME_UNKNOWN", 502);
        if (active.current) {
          savePreparation({ input: intent.body, prepared });
          setChosenRun({
            contextId: prepared.contextId,
            key: prepared.requestKey,
          });
          pending.clear(intent);
        }
      } else {
        setChosenRun({ contextId: intent.contextId, key: intent.key });
        const result = await executeDiagnosis(scope, intent, abort.signal);
        if (
          result.contextId !== intent.contextId ||
          result.requestKey !== intent.key
        )
          throw new MarketAPIError("OUTCOME_UNKNOWN", 502);
        if (active.current) {
          queries.setQueryData(
            [...keys, "run", intent.contextId, intent.key],
            result,
          );
          pending.clear(intent);
          await recent.refetch();
        }
      }
    } catch (e) {
      if (active.current) {
        setFailure(e);
        if (
          e instanceof MarketAPIError &&
          e.code !== "OUTCOME_UNKNOWN" &&
          [400, 403, 404, 409, 412, 422].includes(e.status)
        )
          pending.clear(intent);
      }
    } finally {
      flight.current = false;
      if (active.current) setBusy(false);
    }
  }
  async function verify() {
    const intent = pending.command;
    if (!intent || flight.current) return;
    flight.current = true;
    setBusy(true);
    setFailure(undefined);
    try {
      const contextId =
        intent.kind === "prepare" ? intent.body.contextId : intent.contextId;
      const result = await readDiagnosisRun(scope, contextId, intent.key);
      if (!active.current) return;
      setChosenRun({ contextId, key: intent.key });
      if (
        intent.kind === "prepare" ||
        intent.kind === "start" ||
        (intent.kind === "resume" &&
          BigInt(result.revision) > BigInt(intent.body.revision))
      )
        pending.clear(intent);
      else
        setFailure(
          new Error("原继续请求尚未确认，可再次核实或明确重试原请求。"),
        );
      queries.setQueryData([...keys, "run", contextId, intent.key], result);
    } catch (e) {
      if (active.current) setFailure(e);
    } finally {
      flight.current = false;
      if (active.current) setBusy(false);
    }
  }
  function clearPreparation() {
    if (
      locked ||
      run.data?.phase === "running" ||
      run.data?.usageStatus === "unknown_reserved"
    )
      return;
    localStorage.removeItem(cacheKey);
    window.dispatchEvent(new Event("resource-pending"));
    setChosenRun(null);
    setFacts(undefined);
  }
  const display = run.data?.inspection ?? facts;
  const prepareAllowed =
    entry.canUse &&
    entry.agent.activation === "ENABLED" &&
    textReady &&
    !!mode.data &&
    !locked &&
    !saved &&
    run.data?.phase !== "running" &&
    run.data?.usageStatus !== "unknown_reserved" &&
    validInput &&
    facts?.ref.StoreID === selected &&
    facts?.ref.StartDate === startDate &&
    facts?.ref.EndDate === endDate;
  return (
    <Card className={styles.panel}>
      <h2>店铺经营诊断</h2>
      <p>
        选择一个店铺和 1–366
        天的经营周期，先读取已录入事实，再确认生成建议。人工财务金额和平台保存观察分别呈现，未知数据保持未知。
      </p>
      {failure ? (
        <p role="alert">
          {failure instanceof Error && !(failure instanceof MarketAPIError)
            ? failure.message
            : configurationError(failure)}
        </p>
      ) : null}
      {cacheFailed || pending.error ? (
        <p role="alert">原请求记录无法读取，请恢复浏览器存储后再继续。</p>
      ) : null}
      {!saved && !pending.command ? (
        <>
          {shops.error ? (
            <p role="alert">{cockpitErrorText(shops.error)}</p>
          ) : null}
          {shops.data &&
          (!shops.data.access.storesRead || !shops.data.access.adviceRead) ? (
            <p role="alert">需要经营明细与经营建议两项读取权限。</p>
          ) : null}
          <div className="grid gap-3 md:grid-cols-3">
            <label>
              店铺
              <select
                aria-label="店铺"
                className="mt-2 w-full rounded-md border bg-background p-2"
                value={selected}
                disabled={locked}
                onChange={(e) => {
                  setSelected(e.target.value);
                  setFacts(undefined);
                }}
              >
                <option value="">请选择可访问店铺</option>
                {shops.data?.stores
                  .filter((s) => s.status === "active")
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name} · {s.platform}
                    </option>
                  ))}
              </select>
            </label>
            <label>
              开始日期
              <Input
                aria-label="开始日期"
                type="date"
                value={startDate}
                disabled={locked}
                onChange={(e) => {
                  setStartDate(e.target.value);
                  setFacts(undefined);
                }}
              />
            </label>
            <label>
              结束日期
              <Input
                aria-label="结束日期"
                type="date"
                value={endDate}
                disabled={locked}
                onChange={(e) => {
                  setEndDate(e.target.value);
                  setFacts(undefined);
                }}
              />
            </label>
          </div>
          <Button
            variant="outline"
            disabled={
              locked ||
              !validInput ||
              run.data?.phase === "running" ||
              run.data?.usageStatus === "unknown_reserved"
            }
            onClick={() => void inspect()}
          >
            读取经营事实
          </Button>
          <label className="mt-4 block">
            本次诊断目标
            <Input
              aria-label="本次诊断目标"
              value={goal}
              disabled={locked}
              onChange={(e) => setGoal(e.target.value)}
            />
          </label>
          <Button
            disabled={!prepareAllowed}
            onClick={() => {
              const key = crypto.randomUUID();
              void send({
                kind: "prepare",
                key,
                body: {
                  ...input(),
                  requestKey: key,
                  expectedModeRevision: mode.data!.revision,
                  goalSummary: goal,
                },
              });
            }}
          >
            准备分析
          </Button>
        </>
      ) : null}
      {!textReady ? (
        <p>AI 分析需要当前企业的独立模型配置；可以先读取经营事实。</p>
      ) : null}
      {saved ? (
        <section className="space-y-3 rounded-xl border p-4">
          <h3>确认本次分析</h3>
          <p>{saved.input.goalSummary}</p>
          <p>
            周期 {saved.input.startDate} 至 {saved.input.endDate} · 当前模式{" "}
            {mode.data ? executionModeLabels[mode.data.mode] : "待核对"} ·
            单次调用最多{" "}
            {(saved.prepared.maximumCallCostMicros / 1000000).toFixed(6)}{" "}
            {saved.prepared.currency} · 本次总预算最多{" "}
            {(saved.prepared.maximumCostMicros / 1000000).toFixed(6)}{" "}
            {saved.prepared.currency}
          </p>
          <p>
            建议保存在原运行历史，所有模式均只提供建议，不修改店铺、财务或商品。
          </p>
          {!run.data && !preparedFactsMatch ? <>
            <p>请核对本次准备结果对应的经营事实；事实已变化时，重新准备分析。</p>
            <Button variant="outline" disabled={locked} onClick={() => void inspect()}>
              核对确认依据
            </Button>
          </> : null}
          <Button
            disabled={
              locked ||
              !!run.data ||
              entry.agent.activation !== "ENABLED" ||
              !entry.canUse ||
              !textReady ||
              !preparedFactsMatch
            }
            onClick={() =>
              void send({
                kind: "start",
                key: saved!.prepared.requestKey,
                contextId: saved!.prepared.contextId,
                body: {
                  configurationSnapshotRef:
                    saved!.prepared.configurationSnapshotRef,
                },
              })
            }
          >
            确认生成建议
          </Button>
          <Button
            variant="outline"
            disabled={
              locked ||
              run.data?.phase === "running" ||
              run.data?.usageStatus === "unknown_reserved"
            }
            onClick={clearPreparation}
          >
            准备另一项诊断
          </Button>
        </section>
      ) : null}
      {pending.command ? (
        <section className="space-y-3 rounded-xl border p-4">
          <p>
            保留原请求 {pending.command.key}
            。先核实原结果，不会自动发起另一笔分析。
          </p>
          <Button disabled={busy} onClick={() => void verify()}>
            核实原请求
          </Button>
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => void send(pending.command!)}
          >
            {pending.command.kind === "prepare" ? "重试原准备" : "重试原请求"}
          </Button>
        </section>
      ) : null}
      {busy ? <p role="status">正在处理原请求…</p> : null}
      {display ? <FactTable inspection={display} /> : null}
      {run.data ? (
        <section className="space-y-3">
          <h3>
            {run.data.validated
              ? "建议已生成"
              : run.data.phase === "running"
                ? "分析执行中"
                : "分析尚未完成"}
          </h3>
          <p>
            {run.data.goalSummary} ·{" "}
            {executionModeLabels[run.data.executionMode]}
          </p>
          {run.data.stopReason ? <p>停止原因：{run.data.stopReason}</p> : null}
          {run.data.usageStatus === "unknown_reserved" ? (
            <p>原操作结果或费用待核实，请保留原请求。</p>
          ) : (
            <p>
              已观察 {run.data.tokens} tokens，费用{" "}
              {(run.data.costMicros / 1000000).toFixed(6)} {run.data.currency}
            </p>
          )}
          {run.data.validated && run.data.proposal ? (
            <>
              <p>AI 建议：{run.data.proposal.overview}</p>
              {run.data.proposal.recommendations.map((a, index) => (
                <section key={index} className="rounded-lg border p-3">
                  <h4>建议 {index + 1}</h4>
                  <p>{a.advice}</p>
                  <ul>
                    {a.evidenceIds.map((id) => {
                      const f = run.data!.inspection?.facts.find(
                        (f) => f.id === id,
                      );
                      return (
                        <li key={id}>
                          依据：{f ? factLabel(f.path) : "未知"} ·{" "}
                          {f ? factValue(f.path, f.value) : "未知"}
                        </li>
                      );
                    })}
                  </ul>
                </section>
              ))}
            </>
          ) : null}
          {run.data.unresolved.length ? (
            <p>未解决信息：{run.data.unresolved.join("；")}</p>
          ) : null}
          {run.data.phase === "interrupted" ? (
            <Button
              disabled={locked || !entry.canUse || !textReady}
              onClick={() =>
                void send({
                  kind: "resume",
                  key: run.data!.requestKey,
                  contextId: run.data!.contextId,
                  body: {
                    configurationSnapshotRef:
                      run.data!.configurationSnapshotRef,
                    revision: run.data!.revision,
                    feedback: run.data!.goalSummary,
                  },
                })
              }
            >
              按原请求继续分析
            </Button>
          ) : null}
        </section>
      ) : run.error &&
        !(run.error instanceof MarketAPIError && run.error.status === 404) ? (
        <p role="alert">{configurationError(run.error)}</p>
      ) : null}
      {entry.canReadRuns ? (
        <section className="space-y-2">
          <h3>本人诊断历史</h3>
          {recent.error ? (
            <p role="alert">{configurationError(recent.error)}</p>
          ) : recent.data?.items.length ? (
            <ul>
              {recent.data.items.map((r) => (
                <li key={r.runId}>
                  <Button
                    variant="ghost"
                    disabled={locked}
                    onClick={() =>
                      setChosenRun({
                        contextId: r.operationId,
                        key: r.requestKey,
                      })
                    }
                  >
                    {new Date(r.startedAt).toLocaleString("zh-CN")} ·{" "}
                    {r.phase === "human_review_required"
                      ? "查看分析结果"
                      : r.phase}
                  </Button>
                </li>
              ))}
            </ul>
          ) : (
            <p>暂无对比历史。</p>
          )}
          {recent.data?.nextCursor ? (
            <Button
              variant="outline"
              onClick={() => setAfter(recent.data!.nextCursor)}
            >
              下一页历史
            </Button>
          ) : null}
          {after ? (
            <Button variant="outline" onClick={() => setAfter("")}>
              返回第一页
            </Button>
          ) : null}
        </section>
      ) : null}
    </Card>
  );
}
function factLabel(path: string): string {
  const labels: Record<string, string> = {
    "store.name": "店铺名称",
    "store.platform": "平台",
    "financial.period": "人工财务周期（UTC+8）",
    "financial.unit": "金额单位",
    "financial.source": "财务来源",
    "financial.complete": "财务完整覆盖",
    "financial.record_count": "有效财务记录数",
    "financial.source_digest": "原财务版本依据",
    "financial.records": "财务记录与版本",
    "financial.excluded": "跨区间排除记录",
    "financial.gaps": "未覆盖日期",
    "financial.complete_coverage": "财务完整覆盖",
    "financial.amounts": "尚无已录入金额",
    "financial.revenue": "已录入收入",
    "financial.refunds": "已录入退款",
    "financial.procurement": "采购成本",
    "financial.logistics": "物流成本",
    "financial.platform": "平台费用",
    "financial.advertising": "广告成本",
    "financial.other": "其他成本",
    "financial.net_revenue": "已录入净收入",
    "financial.net_profit": "已录入净利润",
    "financial.margin": "原生利润率分数",
    "financial.growth": "上一个等长周期利润变化（精确分数）",
    "observations.source_digest": "原平台观察版本依据",
  };
  if (labels[path]) return labels[path];
  if (path.startsWith("financial.rule.")) return "现有经营规则";
  const m =
    /^observations\.(products|orders)\.(state|complete|source|latest|counts)$/.exec(
      path,
    );
  if (m)
    return (
      (m[1] === "products" ? "已保存商品" : "已保存订单") +
      (
        {
          state: "数据状态",
          complete: "观察完整性",
          source: "来源范围及时间",
          latest: "最新同步状态与范围",
          counts: "原范围计数",
        } as Record<string, string>
      )[m[2]]
    );
  return path;
}
function factValue(path: string, value: string): string {
  if (path === "financial.unit") return "人民币 · 分（显示为元）";
  if (path === "financial.source") return "人工录入";
  if (path.endsWith(".complete")) return value === "true" ? "完整" : "不完整";
  if (path.endsWith(".state"))
    return (
      (
        {
          available: "有保存观察",
          incomplete: "保存观察不完整",
          latest_incomplete: "最新观察未完整完成",
          unavailable: "尚无可用保存观察",
          permission_required: "需要平台观察读取权限",
          unsupported: "暂不支持离线观察",
          offline_unavailable: "暂不支持离线读取；未查询平台",
        } as Record<string, string>
      )[value] ?? value
    );
  if (
    /^financial\.(revenue|refunds|procurement|logistics|platform|advertising|other|net_revenue|net_profit)$/.test(
      path,
    ) &&
    /^-?\d+$/.test(value) &&
    Number.isSafeInteger(Number(value))
  )
    return moneyText(Number(value));
  try {
    const decoded: unknown = JSON.parse(value);
    if (path.endsWith(".source") || path.endsWith(".latest")) {
      if (Array.isArray(decoded))
        return decoded.length
          ? decoded
              .map(
                (v: Record<string, unknown>) =>
                  "同步 " +
                  String(v.SyncID) +
                  " · 状态 " +
                  String(v.Status) +
                  (v.ObservedAt
                    ? " · 观察时间 " + String(v.ObservedAt)
                    : " · 尚无观察时间") +
                  (v.Start && v.End
                    ? " · 原范围 " + String(v.Start) + " 至 " + String(v.End)
                    : " · 未提供原范围"),
              )
              .join("；")
          : "暂无保存来源";
    }
    if (path.endsWith(".counts") && decoded && typeof decoded === "object") {
      const v = decoded as Record<string, unknown>;
      return [
        "Total",
        "Active",
        "OffShelf",
        "Pending",
        "Transit",
        "Exceptional",
        "Unknown",
      ]
        .map(
          (key, i) =>
            ["总数", "在售", "已下架", "待处理", "运输中", "异常", "未知状态"][
              i
            ] +
            "：" +
            String(v[key]),
        )
        .join(" · ");
    }
    if (
      path.startsWith("financial.rule.") &&
      decoded &&
      typeof decoded === "object"
    )
      return String((decoded as Record<string, unknown>).Reason);
    if (
      (path === "financial.margin" || path === "financial.growth") &&
      decoded &&
      typeof decoded === "object"
    ) {
      const v = decoded as Record<string, unknown>;
      return (
        String(v.numerator) + " / " + String(v.denominator) + "（原生精确分数）"
      );
    }
  } catch {}
  return value;
}
function FactTable({ inspection }: { inspection: DiagnosisInspection }) {
  const ref = inspection.ref,
    complete =
      inspection.facts.find((f) => f.path === "financial.complete")?.value ===
      "true";
  return (
    <section className="space-y-3">
      <h3>已保存经营事实</h3>
      <p>
        {complete
          ? "所选人工财务周期已完整覆盖。"
          : "人工财务数据不足：已录入金额仅代表部分事实，不能评价完整利润或店铺健康。"}
      </p>
      <p>
        平台观察的原同步范围独立于所选财务周期；计数不代表该周期销量或收入。
      </p>
      <div className="flex flex-wrap gap-3">
        <Link
          href={
            "/workbench/overview/stores?storeId=" +
            ref.StoreID +
            "&startDate=" +
            ref.StartDate +
            "&endDate=" +
            ref.EndDate
          }
        >
          查看原经营明细
        </Link>
        <Link href="/workbench/store-products">查看保存商品</Link>
        <Link href="/workbench/store-orders">查看保存订单</Link>
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>事实</TableHead>
              <TableHead>保存值及依据</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {inspection.facts.map((f) => (
              <TableRow key={f.id}>
                <TableCell>{factLabel(f.path)}</TableCell>
                <TableCell className="break-all whitespace-normal">
                  {factValue(f.path, f.value)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <p>
        缺失信息：
        {inspection.missing.length
          ? inspection.missing.map(factLabel).join("；")
          : "上述来源未报告其他缺口，仍需人工判断建议。"}
      </p>
    </section>
  );
}
