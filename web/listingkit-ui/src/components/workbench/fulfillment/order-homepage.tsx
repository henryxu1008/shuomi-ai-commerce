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
import { ConsolePage, ConsoleState } from "../console/console-page";
import styles from "./fulfillment.module.css";

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
  return <ConsolePage title="订单管理" description="从订单关联、采购收货到装箱、发货和物流，按同一原单继续办理。" breadcrumbs={[{ label: "订单管理" }]}
    actions={<Link href="/workbench/store-orders/records" prefetch={false}>平台订单记录</Link>}>
    {!ready ? <ConsoleState kind="unavailable" title="请先确认当前企业" /> : !readable ? <ConsoleState kind="unavailable" title="当前成员没有订单履约读取权限" /> : capabilities.isPending ? <ConsoleState kind="loading" title="正在确认订单履约能力…" /> : capabilities.data?.available !== true ? <ConsoleState kind="unavailable" title="当前环境未开放订单履约"><p>平台订单同步与记录可从上方入口单独查看。</p></ConsoleState> : <>
      {businessCase?.organizationId === organizationId ? <SavedCase key={`${organizationId}:${userId}`} reference={businessCase} scope={scope} /> : null}
      <Card className={styles.panel}><h2>办理其他订单</h2><p>选择当前授权店铺和原单，继续使用相同的采购、库存与发货流程。真实平台发货待联调。</p>
        <Link href="/workbench/store-orders/fulfill" prefetch={false}>进入订单履约</Link></Card>
    </>}
  </ConsolePage>;
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
  return <Card className={styles.panel}>
    <header><p>合成业务案例 · 可查看原单据并继续办理</p><h2>{reference.title}</h2><p>{store.name} · {reference.kind === "platform_supply" ? "原授权供应商备货，销售市场未知" : "消费者订单自发货"}</p></header>
    <div><Button variant="outline" disabled={query.isFetching} onClick={() => void query.refetch()}>{query.isFetching ? "正在更新案例…" : "刷新案例"}</Button></div>
    <p>原单 {reference.sourceId} · 原需求 {detail.name} · 需求 {quantity} 件</p>
    <p>本案例使用合成商品和供应商，全部交接与平台回执仅供试用。</p>
    <ol className="grid gap-4" aria-label="订单业务流程">
      <li><h3>1. 原单关联</h3><p>已关联原生需求 {detail.name}；原单与内部单据分别保留。</p><Link href={caseLink(reference, "source")} prefetch={false}>查看原单关联</Link></li>
      <li><h3>2. 采购与收货</h3>{detail.purchases.length ? detail.purchases.map(purchase => {
        const ordered = purchase.lines.reduce((sum, line) => sum + line.qty, 0);
        const accepted = purchase.has_more_receipts ? null : purchase.receipts.reduce((sum, receipt) => sum + receipt.lines.filter(line => purchase.lines.some(original => original.line === line.purchase_line)).reduce((total, line) => total + line.accepted_qty, 0), 0);
        return <div key={purchase.name}>
        <p><strong>{purchase.name}</strong> · {purchase.supplier}</p>
        {accepted === null ? <><p>采购 {ordered} 件</p><p>合格收货数量需翻页核对</p></> : <><p>采购 {ordered} 件 · 合格收货 {accepted} 件</p><p>尚未合格收货 {Math.max(0, ordered - accepted)} 件</p></>}
        <p>累计到货 {purchase.lines.reduce((sum, line) => sum + line.received_qty, 0)} 件（含拒收）</p>
        {purchase.receipts.map(receipt => <p key={receipt.name}>收货单 {receipt.name}</p>)}
        {purchase.has_more_receipts ? <p>还有收货记录，请进入采购页翻页核对。</p> : null}
      </div>; }) : <p>尚无采购单；已有库存的订单可直接办理库存与装箱。</p>}
        {detail.has_more_purchases ? <p>当前展示首批采购单，更多单据请进入采购页核对。</p> : null}
        <Button asChild><Link href={caseLink(reference, "purchase")} prefetch={false}>打开采购与收货</Link></Button></li>
      <li><h3>3. 库存与装箱</h3><p>当前预留 {reserved} 件 · 当前页包裹 {detail.deliveries.length} 个</p>
        {detail.deliveries.map(delivery => <p key={delivery.name}>{delivery.package_id} · {delivery.name} · {delivery.docstatus === 1 ? "已实际交接" : "装箱草稿"}</p>)}
        {detail.has_more_deliveries ? <p>还有包裹，请进入装箱页翻页核对。</p> : null}
        <Link href={caseLink(reference, "stock")} prefetch={false}>查看库存与装箱</Link></li>
      <li><h3>4. 平台发货</h3><p>{detail.operations.length ? `当前页已保存 ${detail.operations.length} 笔平台操作，请查看各笔原回执。` : "尚无平台操作回执；准备后仍需单独核对并确认。"}</p>
        {detail.operations.some(operation => operation.stage === "UNKNOWN") ? <p role="status">存在结果未知的原操作，请先核实，勿另建操作重发。</p> : null}
        {detail.has_more_operations ? <p>还有平台操作，请进入发货页翻页核对。</p> : null}
        <Link href={caseLink(reference, "platform")} prefetch={false}>查看平台发货</Link></li>
      <li><h3>5. 交接与物流</h3><p>已移交 {handed} / {quantity} 件。内部移交数量不代表平台实收或入库。</p><Link href={caseLink(reference, "logistics")} prefetch={false}>查看交接与物流</Link></li>
    </ol>
  </Card>;
}
