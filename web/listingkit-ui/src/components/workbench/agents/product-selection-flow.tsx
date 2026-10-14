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
  inspectSelection,
  prepareSelection,
  executeSelection,
  readSelectionRun,
} from "@/lib/api/product-selection";
import {
  selectionIntentSchema,
  selectionInputSchema,
  selectionPrepareSchema,
  selectionSavedPreparationSchema,
  selectionRecentSchema,
  type SelectionIntent,
  type SelectionInspection,
  type SelectionPreparation,
} from "@/lib/contracts/product-selection";
import type { CollectionItem } from "@/lib/contracts/product-collection";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import { executionModeViewSchema } from "@/lib/contracts/execution-mode";
import { CollectionPicker } from "../pod/collection-picker";
import { useResourcePending } from "../resources/resource-pending";
import { configurationError } from "./agent-configuration-hooks";
import { executionModeLabels } from "./execution-mode-settings";
import styles from "./agents.module.css";

export function ProductSelectionFlow({
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
    ["product-selection"],
    selectionIntentSchema,
    { storage: "local", maxLength: 16384 },
  );
  const [selected, setSelected] = useState<CollectionItem[]>([]),
    [goal, setGoal] = useState(
      "请根据真实事实，用中文比较商品并指出缺失信息。",
    ),
    [facts, setFacts] = useState<SelectionInspection>(),
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
    "product.selection.prepared:" +
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
  let saved: SelectionPreparation | undefined,
    cacheFailed = false;
  if (cacheRaw) {
    try {
      if (cacheRaw.length > 16384) throw new Error();
      saved = selectionSavedPreparationSchema.parse(JSON.parse(cacheRaw));
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
  const keys = ["product-selection", scope.userId, scope.organizationId];
  const mode = useQuery({
    queryKey: [...keys, "mode", nonce],
    queryFn: ({ signal }) =>
      configurationRequest(
        scope,
        "execution-modes/registered_agent/product.selection.agent",
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
        "product.selection.agent/recent-runs?pageSize=20" +
          (after ? "&cursor=" + after : ""),
        selectionRecentSchema,
        { signal },
      ),
    enabled: entry.canReadRuns,
  });
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
      readSelectionRun(
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
  const factsReady = entry.capabilities.some(c => c.id === "product.selection.inspect" && c.readiness === "AVAILABLE");
  const goalCheck = selectionPrepareSchema.shape.goalSummary.safeParse(goal.trim());
  const input = () =>
    selectionInputSchema.parse({
      contextId: crypto.randomUUID(),
      candidates: selected.map((i) => ({
        itemId: i.id,
        expectedRevision: String(i.revision),
        originalPublicationId: i.source.publicationId,
        originalVersion: i.source.version,
      })),
    });
  const matchesFacts = (candidates: SelectionPreparation["input"]["candidates"]) =>
    !!facts && facts.products.length === candidates.length && candidates.every(c => facts.products.some(p => p.ref.ItemID === c.itemId && p.ref.ItemRevision === c.expectedRevision && p.ref.PublicationID === c.originalPublicationId && p.ref.CatalogVersion === c.originalVersion));
  const selectedFactsMatch = selected.length >= 2 && matchesFacts(selected.map(i => ({itemId:i.id,expectedRevision:String(i.revision),originalPublicationId:i.source.publicationId,originalVersion:i.source.version})));
  const preparedFactsMatch = !!saved && matchesFacts(saved.input.candidates);
  const preparedModeMatch = !!saved && !!mode.data && saved.prepared.modeRevision === mode.data.revision;
  function savePreparation(value: SelectionPreparation) {
    const raw = JSON.stringify(selectionSavedPreparationSchema.parse(value));
    localStorage.setItem(cacheKey, raw);
    if (localStorage.getItem(cacheKey) !== raw)
      throw new Error("无法保留原准备结果");
    window.dispatchEvent(new Event("resource-pending"));
  }
  async function inspect() {
    if (flight.current || !factsReady) return;
    flight.current = true;
    setBusy(true);
    setFailure(undefined);
    const abort = new AbortController();
    controller.current = abort;
    try {
      const result = await inspectSelection(scope, saved ? {contextId:saved.input.contextId,candidates:saved.input.candidates} : input(), abort.signal);
      if (active.current) {
        setChosenRun(null);
        setFacts(result);
      }
    } catch (e) {
      if (active.current) setFailure(e);
    } finally {
      flight.current = false;
      if (active.current) setBusy(false);
    }
  }
  async function send(intent: SelectionIntent) {
    if (flight.current || !active.current || !pending.ready) return;
    if (intent.kind === "start" && !pending.command && (!preparedFactsMatch || !preparedModeMatch)) {
      setFailure(new Error("请核对本次准备的事实和权限版本；权限变化后须重新准备。"));
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
        const prepared = await prepareSelection(scope, intent, abort.signal);
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
        const result = await executeSelection(scope, intent, abort.signal);
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
        if (e instanceof MarketAPIError && e.status === 409 && ["REVISION_MISMATCH", "CONFIGURATION_CHANGED"].includes(e.code)) void mode.refetch();
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
      const result = await readSelectionRun(scope, contextId, intent.key);
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
  const selectedSavedPointer = !!saved && chosenRun?.contextId === saved.prepared.contextId && chosenRun?.key === saved.prepared.requestKey;
  const display = run.data ? run.data.inspection : chosenRun && !selectedSavedPointer ? undefined : facts;
  const prepareAllowed =
    entry.canUse &&
    entry.agent.activation === "ENABLED" &&
    textReady &&
    !!mode.data &&
    !locked &&
    !saved &&
    selectedFactsMatch && goalCheck.success;
  return (
    <Card className={styles.panel}>
      <h2>选品对比</h2>
      <p>
        选择 2–5
        个已保存商品，先查看原始事实，再生成有引用的建议。缺少销量、完整成本或当前供货事实时会明确列出。
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
          <CollectionPicker
            scope={scope}
            kind="product"
            disabled={locked || selected.length >= 5}
            onSelect={(item) => {
              if (selected.some((i) => i.id === item.id)) return;
              setSelected([...selected, item]);
              setFacts(undefined);
            }}
          />
          <ul>
            {selected.map((i) => (
              <li key={i.id}>
                {i.title || i.source.productKey}
                <Button
                  variant="ghost"
                  disabled={locked}
                  onClick={() => {
                    setSelected(selected.filter((p) => p.id !== i.id));
                    setFacts(undefined);
                  }}
                >
                  移除候选
                </Button>
              </li>
            ))}
          </ul>
          <Button
            variant="outline"
            disabled={locked || !factsReady || selected.length < 2}
            onClick={() => void inspect()}
          >
            读取事实对比
          </Button>
          <label className="mt-4 block">
            本次比较目标
            <Input
              aria-label="本次比较目标"
              value={goal}
              disabled={locked}
              onChange={(e) => setGoal(e.target.value)}
            />
          </label>
          {!goalCheck.success ? <p>请填写简短、明确的比较目标；内容过长时请精简。</p> : null}
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
                  goalSummary: goalCheck.success ? goalCheck.data : "",
                },
              });
            }}
          >
            准备分析
          </Button>
        </>
      ) : null}
      {!factsReady ? <p>事实读取需要当前企业的智能体使用和我的数据读取权限。</p> : null}
      {!textReady && factsReady ? (
        <p>AI 分析需要当前企业的独立模型配置；可以先读取候选事实。</p>
      ) : null}
      {saved ? (
        <section className="space-y-3 rounded-xl border p-4">
          <h3>确认本次分析</h3>
          <p>{saved.input.goalSummary}</p>
          <p>
            候选 {saved.input.candidates.length} 个 · 准备权限版本 {saved.prepared.modeRevision} · 当前模式{" "}
            {mode.data ? executionModeLabels[mode.data.mode] : "待核对"} ·
            单次调用最多{" "}
            {(saved.prepared.maximumCallCostMicros / 1000000).toFixed(6)}{" "}
            {saved.prepared.currency} · 本次总预算最多{" "}
            {(saved.prepared.maximumCostMicros / 1000000).toFixed(6)}{" "}
            {saved.prepared.currency}
          </p>
          <p>建议保存在原运行历史，所有模式均只分析，不采购或修改商品。</p>
          {!preparedModeMatch ? <p role="alert">权限模式已变化或待核对，请重新准备；原请求编号和运行保留。</p> : null}
          {!run.data && !preparedFactsMatch ? <><p>请先核对本次准备对应的商品事实。</p><Button variant="outline" disabled={locked || !factsReady} onClick={()=>void inspect()}>核对确认依据</Button></> : null}
          {currentPointer?.contextId !== saved.prepared.contextId || currentPointer?.key !== saved.prepared.requestKey ? (
            <Button variant="outline" disabled={locked} onClick={() => setChosenRun({contextId:saved!.prepared.contextId,key:saved!.prepared.requestKey})}>
              返回待确认的对比
            </Button>
          ) : null}
          <Button
            disabled={
              locked ||
              !!run.data ||
              entry.agent.activation !== "ENABLED" ||
              !entry.canUse ||
              !textReady || !preparedFactsMatch || !preparedModeMatch
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
            准备另一组对比
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
              {run.data.proposal.recommendedItemId ? (
                <p>
                  建议优先核实：
                  {String(
                    run.data.inspection?.products
                      .find(
                        (p) =>
                          p.ref.ItemID ===
                          run.data!.proposal!.recommendedItemId,
                      )
                      ?.facts.find((f) => f.path === "title")?.value ??
                      run.data.proposal.recommendedItemId,
                  )}
                </p>
              ) : null}
              {run.data.proposal.assessments.map((a) => (
                <section key={a.itemId}>
                  <h4>
                    {(run
                      .data!.inspection?.products.find(
                        (p) => p.ref.ItemID === a.itemId,
                      )
                      ?.facts.find((f) => f.path === "title")
                      ?.value as string) || a.itemId}
                  </h4>
                  <p>{a.advice}</p>
                  <ul>
                    {a.evidenceIds.map((id) => {
                      const f = run
                        .data!.inspection?.products.find(
                          (p) => p.ref.ItemID === a.itemId,
                        )
                        ?.facts.find((f) => f.id === id);
                      return (
                        <li key={id}>
                          依据：{f ? factLabel(f.path) : "未知"} ·{" "}
                          {JSON.stringify(f?.value)}
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
          <h3>本人对比历史</h3>
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
const missingNames: Record<string, string> = {
  title: "名称",
  brand: "品牌",
  category_path: "类目",
  description: "描述",
  selling_points: "卖点",
  specifications: "规格",
  review: "源资料审核提示",
  warnings: "源资料提醒",
  variants: "SKU",
  additional_attributes: "其他属性未展示",
  additional_variants: "其他 SKU 未展示",
  additional_facts_omitted: "更多完整字段因展示预算省略",
  sales: "销量",
  market_demand: "市场需求",
  shipping: "运费",
  commission: "佣金",
  tax: "税费",
  landed_cost: "完整到岸成本",
};
function factLabel(path: string): string {
  if (missingNames[path]) return missingNames[path];
  const sku = /^variants\.(\d+)\.(.+)$/.exec(path);
  const skuNames: Record<string, string> = {
    sku: "编号",
    attributes: "属性",
    price: "原币价格",
    stock: "已保存库存",
    uncertainty: "待核实",
  };
  if (sku) return `SKU ${Number(sku[1]) + 1} · ${skuNames[sku[2]] ?? sku[2]}`;
  const attr = /^attributes\.(\d+)(\.uncertainty)?$/.exec(path);
  return attr
    ? `属性 ${Number(attr[1]) + 1}${attr[2] ? " · 待核实" : ""}`
    : path;
}
function FactTable({ inspection }: { inspection: SelectionInspection }) {
  const paths = [
    ...new Set(inspection.products.flatMap((p) => p.facts.map((f) => f.path))),
  ];
  return (
    <section className="space-y-3">
      <h3>已保存原始事实</h3>
      <p>
        价格保留原币种，不换算或推测利润；原始资料不保证今日库存和供货条件。
      </p>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>事实</TableHead>
              {inspection.products.map((p) => (
                <TableHead key={p.ref.ItemID}>
                  {String(
                    p.facts.find((f) => f.path === "title")?.value ??
                      p.ref.ProductKey,
                  )}
                  <br />
                  <Link href={"/workbench/data/mine?item=" + p.ref.ItemID}>
                    原商品 · 版本 {p.ref.CatalogVersion}
                  </Link>
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {paths.map((path) => (
              <TableRow key={path}>
                <TableCell>{factLabel(path)}</TableCell>
                {inspection.products.map((p) => {
                  const f = p.facts.find((f) => f.path === path);
                  return (
                    <TableCell key={p.ref.ItemID}>
                      {f
                        ? f.path.endsWith(".uncertainty")
                          ? "源资料由 AI 推断或待审核，需核实"
                          : typeof f.value === "string"
                            ? f.value
                            : JSON.stringify(f.value)
                        : "未知"}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {inspection.products.map((p) => (
        <p key={p.ref.ItemID}>
          {String(
            p.facts.find((f) => f.path === "title")?.value ?? p.ref.ProductKey,
          )}{" "}
          缺项：{p.missing.map(factLabel).join("；")}
        </p>
      ))}
    </section>
  );
}
