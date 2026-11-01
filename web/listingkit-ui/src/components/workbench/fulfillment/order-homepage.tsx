"use client";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getWorkbenchStore } from "@/lib/api/workbench-stores";
import { fulfillmentCapabilities, fulfillmentDetail, fulfillmentRequest, type FulfillmentScope } from "@/lib/api/fulfillment";
import type { OrderBusinessCase } from "@/lib/contracts/order-business-case";
import { ConsoleState } from "../console/console-page";
import { PrimaryIntroduction } from "../console/primary-introduction";
import styles from "../console/primary-introduction.module.css";

export function OrderHomepage({ businessCase }: { businessCase?: OrderBusinessCase }) {
  const context = useWorkbenchContext();
  const organizationId = context.effectiveOrganization?.id ?? "";
  const userId = context.user?.id ?? "";
  const readable = (context.permissions ?? []).includes("workbench.fulfillment.read");
  const ready = !!organizationId && !!userId && !context.isLoading && !context.isSwitching && !context.selectionRequired && !context.error && !context.blockingError;
  const scope = { organizationId, userId };
  const capabilities = useQuery({
    queryKey: ["orders-home", organizationId, userId, context.permissions],
    enabled: ready && readable,
    queryFn: ({ signal }) => fulfillmentRequest(scope, "capabilities", fulfillmentCapabilities, signal),
    retry: false,
  });
  const available = ready && readable && capabilities.data?.available === true;
  return <PrimaryIntroduction kind="orders" available={[available, available, available, available]}
    actions={<Button asChild variant="outline"><Link href="/workbench/store-orders/records" prefetch={false}>平台订单记录</Link></Button>}>
    {!ready ? <ConsoleState kind="unavailable" title="请先确认当前企业" /> : !readable ? <ConsoleState kind="unavailable" title="当前成员没有订单履约读取权限" /> : capabilities.isPending ? <ConsoleState kind="loading" title="正在确认订单履约能力…" /> : capabilities.data?.available !== true ? <ConsoleState kind="unavailable" title="当前环境未开放订单履约"><p>平台订单同步与记录可从上方入口单独查看。</p></ConsoleState> : <>
      {businessCase?.organizationId === organizationId ? <SavedCase key={`${organizationId}:${userId}`} reference={businessCase} scope={scope} /> : null}
    </>}
  </PrimaryIntroduction>;
}

function caseLink(reference: OrderBusinessCase, step: string) {
  return `/workbench/store-orders/fulfill?${new URLSearchParams({ storeId: reference.storeId, sourceId: reference.sourceId, kind: reference.kind, step })}`;
}

function SavedCase({ reference, scope }: { reference: OrderBusinessCase; scope: FulfillmentScope }) {
  const query = useQuery({
    queryKey: ["order-business-case", scope.organizationId, scope.userId, reference.storeId, reference.kind, reference.sourceId],
    retry: false,
    queryFn: async ({ signal }) => {
      // Use the normal Store owner first, including live connection observation.
      const store = await getWorkbenchStore(reference.storeId, scope.organizationId);
      signal.throwIfAborted();
      const read = <T,>(action: "find_demand" | "details", payload: Record<string, unknown>, schema: z.ZodType<T>) => fulfillmentRequest(scope, {
        store: reference.storeId, kind: reference.kind, source_id: reference.sourceId,
        operation_id: crypto.randomUUID(), action, payload,
      }, schema, signal);
      const found = await read("find_demand", {}, z.object({ demand: z.string().nullable() }).strict());
      if (!found.demand) return { store, detail: null };
      const detail = await read("details", { demand: found.demand }, fulfillmentDetail);
      return { store, detail };
    },
  });
  if (query.isPending) return <ConsoleState kind="loading" title="正在读取业务案例…" />;
  if (query.isError) return <ConsoleState kind="error" title="业务案例暂时无法读取"><p>未能核对当前权限或原生单据。已保存的事实不会因此重新创建。</p><Button variant="outline" onClick={() => void query.refetch()}>重新读取案例</Button></ConsoleState>;
  const { store, detail } = query.data;
  if (!detail) return <ConsoleState kind="unavailable" title="业务案例尚未准备"><p>当前原单没有已保存的履约需求。</p></ConsoleState>;
  const quantity = detail.lines.reduce((sum, line) => sum + line.qty, 0);
  const reserved = detail.lines.reduce((sum, line) => sum + line.reserved_qty, 0);
  const handed = detail.lines.reduce((sum, line) => sum + line.delivered_qty, 0);
  return <Card className={styles.caseSummary} role="region" aria-label="当前业务案例">
    <div>
      <header><p>合成业务案例 · {reference.kind === "platform_supply" ? "原授权供应商备货，销售市场未知" : "消费者订单自发货"}</p><h2>{reference.title}</h2></header>
      <p>原需求 {detail.name} · 需求 {quantity} 件 · 当前预留 {reserved} 件 · 已移交 {handed} 件</p>
      {detail.purchases.map(purchase => {
        const ordered = purchase.lines.reduce((sum, line) => sum + line.qty, 0);
        const accepted = purchase.has_more_receipts ? null : purchase.receipts.reduce((sum, receipt) => sum + receipt.lines.filter(line => purchase.lines.some(original => original.line === line.purchase_line)).reduce((total, line) => total + line.accepted_qty, 0), 0);
        return <div key={purchase.name} className={styles.caseFacts}>
          <strong>{purchase.name}</strong>
          {accepted === null ? <p>合格收货数量需翻页核对</p> : <><p>采购 {ordered} 件 · 合格收货 {accepted} 件</p><p>尚未合格收货 {Math.max(0, ordered - accepted)} 件</p></>}
          <p>累计到货 {purchase.lines.reduce((sum, line) => sum + line.received_qty, 0)} 件（含拒收）</p>
        </div>;
      })}
      {detail.has_more_purchases ? <p>当前展示首批采购单，更多单据请进入采购页核对。</p> : null}
      {detail.operations.some(operation => operation.stage === "UNKNOWN") ? <p role="status">存在结果未知的原操作，请先核实，勿另建操作重发。</p> : null}
      <p className={styles.caseOrigin}>{store.name} · 原单 {reference.sourceId} · 全部交接和平台回执仅供合成试用。</p>
    </div>
    <Button asChild><Link href={caseLink(reference, "purchase")} prefetch={false}>打开采购与收货</Link></Button>
  </Card>;
}
