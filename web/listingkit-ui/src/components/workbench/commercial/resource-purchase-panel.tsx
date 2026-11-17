"use client";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { AccountDialog } from "../account/account-dialog";
import {
  getCommercialOrder,
  getCommercialResources,
  getCommercialWallet,
  type CommercialResources,
} from "@/lib/api/commercial-billing";
import {
  createResourceOrder,
  createResourceQuote,
  positiveInt64,
  type ResourceOffer,
  type ResourceOffers,
  type ResourceOrder,
  type ResourceQuote,
} from "@/lib/api/resource-purchase";
import styles from "./commercial.module.css";
import { resourceDefinitions, resourcePurchaseHref, walletBalanceId } from "./resource-catalog-display";
import { ResourcePurchaseAnchor } from "./resource-purchase-anchor";
function resourceMoney(v: string) {
  const n = BigInt(v);
  return `¥${(n / BigInt(100)).toLocaleString("zh-CN")}.${(n % BigInt(100)).toString().padStart(2, "0")}`;
}
export const pendingSchema = z
  .object({
    offerId: z.string().min(1).max(128),
    quoteId: z.string().min(1).max(128),
    key: z.string().uuid(),
    orderId: z.string().min(1).max(128).optional(),
  })
  .strict();
type Pending = z.infer<typeof pendingSchema>;
const storageKey = (user: string, org: string) =>
  `resource-purchase.pending:${JSON.stringify([user, org])}`;
function subscribe(callback: () => void) {
  window.addEventListener("resource-pending", callback);
  window.addEventListener("storage", callback);
  return () => {
    window.removeEventListener("resource-pending", callback);
    window.removeEventListener("storage", callback);
  };
}
export function readPending(raw: string) {
  try {
    if (!raw) return { value: null, invalid: false };
    const v = pendingSchema.safeParse(JSON.parse(raw));
    return { value: v.success ? v.data : null, invalid: !v.success };
  } catch {
    return { value: null, invalid: true };
  }
}
const errorCode = (error: unknown) =>
  error &&
  typeof error === "object" &&
  "code" in error &&
  typeof error.code === "string"
    ? error.code
    : "DEPENDENCY_UNAVAILABLE";
const failures: Record<string, string> = {
  INVALID_REQUEST: "购买数量或金额无效，请重新确认。",
  AUTHENTICATION_REQUIRED: "登录已失效。",
  FORBIDDEN: "当前企业购买权限已撤销。",
  PERMISSION_DENIED: "无购买权限。",
  ORGANIZATION_ACCESS_REVOKED: "企业访问已撤销。",
  ORGANIZATION_CONTEXT_CHANGED: "企业上下文已变化。",
  IDENTITY_CONTEXT_CHANGED: "登录身份已变化。",
  QUOTE_EXPIRED: "报价已过期，请重新获取报价。",
  OFFER_UNAVAILABLE: "价格暂不可用，请刷新报价目录。",
  INSUFFICIENT_FUNDS: "企业钱包余额不足。",
  NOT_FOUND: "原报价或订单不存在，请联系支持核对。",
  IDEMPOTENCY_CONFLICT: "原订单身份冲突，请联系支持核对。",
  RECONCILIATION_REQUIRED: "正在核对原订单，请勿重新购买。",
  DEADLINE_EXCEEDED: "请求超时，结果尚未确认。请恢复原订单，勿重新购买。",
  DEPENDENCY_UNAVAILABLE: "服务暂不可用，结果尚未确认。请恢复原订单。",
};
const noEffectErrors = new Set([
  "QUOTE_EXPIRED",
  "OFFER_UNAVAILABLE",
  "PAYMENT_METHOD_UNAVAILABLE",
  "INSUFFICIENT_FUNDS",
  "NOT_FOUND",
]);
function amountMinor(value: string): string | null {
  if (!/^(0|[1-9][0-9]*)(\.[0-9]{1,2})?$/.test(value) || value.length > 22)
    return null;
  const [major, minor = ""] = value.split(".");
  const n = (
    BigInt(major) * BigInt(100) +
    BigInt(minor.padEnd(2, "0"))
  ).toString();
  return positiveInt64(n) ? n : null;
}
type Props = {
  userId: string;
  organizationId: string;
  organizationName: string;
  permissions: string[];
  offers: ResourceOffers;
};
export function ResourcePurchasePanel(props: Props) {
  return (
    <ScopedPurchase
      key={JSON.stringify([props.userId, props.organizationId, props.permissions])}
      {...props}
    />
  );
}
function ScopedPurchase({
  userId,
  organizationId,
  organizationName,
  permissions,
  offers,
}: Props) {
  const client = useQueryClient();
  const key = storageKey(userId, organizationId);
  const raw = useSyncExternalStore(
    subscribe,
    () => {
      try {
        return sessionStorage.getItem(key) ?? "";
      } catch {
        return "__unavailable__";
      }
    },
    () => "__loading__",
  );
  const pending = readPending(raw);
  const ready = raw !== "__loading__" && !pending.invalid;
  const [quote, setQuote] = useState<ResourceQuote | null>(null);
  const [selected, setSelected] = useState<ResourceOffer | null>(null);
  const [order, setOrder] = useState<ResourceOrder | null>(null);
  const [balances, setBalances] = useState<CommercialResources | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [confirmation, setConfirmation] = useState<"确认购买" | "确认领取" | null>(null);
  const active = useRef<AbortController | null>(null);
  const locked = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      active.current?.abort();
    };
  }, []);
  const canPurchase = permissions.includes("workbench.commercial.purchase");
  const wallet = useQuery({
    queryKey: [
      "workbench",
      organizationId,
      "commercial-wallet",
      userId,
      "resource-purchase",
    ],
    queryFn: ({ signal }) =>
      getCommercialWallet(userId, organizationId, signal),
    enabled: offers.items.length > 0,
    gcTime: 0,
    staleTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
  });
  function persist(value: Pending | null) {
    if (!mounted.current) return false;
    try {
      if (value) sessionStorage.setItem(key, JSON.stringify(value));
      else sessionStorage.removeItem(key);
      window.dispatchEvent(new Event("resource-pending"));
      return true;
    } catch {
      setMessage("无法保存原订单恢复信息，已停止提交购买请求。");
      return false;
    }
  }
  async function run(
    action: (signal: AbortSignal) => Promise<void>,
    onError?: (error: unknown) => string | void,
  ) {
    if (locked.current) return;
    locked.current = true;
    setBusy(true);
    setMessage("");
    const controller = new AbortController();
    active.current = controller;
    try {
      await action(controller.signal);
    } catch (error) {
      if (!controller.signal.aborted && mounted.current) {
        setMessage(
          onError?.(error) ?? failures[errorCode(error)] ?? "本次请求未确认，请核对原订单。",
        );
      }
    } finally {
      locked.current = false;
      if (mounted.current) setBusy(false);
      if (active.current === controller) active.current = null;
    }
  }
  async function choose(
    offer: ResourceOffer,
    selection:
      | { quantity: string; amountMinor?: never }
      | { amountMinor: string; quantity?: never },
  ) {
    if (!ready || pending.value || busy || locked.current || !canPurchase) return;
    setConfirmation(offer.settlement_mode === "ZERO_PRICE" ? "确认领取" : "确认购买");
    await run(async (signal) => {
      const result = await createResourceQuote(
        userId,
        organizationId,
        offer.offer_id,
        selection,
        signal,
      );
      signal.throwIfAborted();
      if (
        result.organization_id !== organizationId ||
        result.offer_id !== offer.offer_id ||
        result.product_kind !== offer.product_kind ||
        result.resource_type !== offer.resource_type ||
        result.pricing_version !== offer.pricing_version ||
        BigInt(result.resource_quantity) * BigInt(offer.unit_price_minor) !==
          BigInt(result.total_minor)
      ) {
        setMessage("价格已变化。请取消，刷新页面价格后再试。");
        return;
      }
      setSelected(offer);
      setQuote(result);
    }, (error) => {
      const code = errorCode(error);
      if (code === "OFFER_UNAVAILABLE" || code === "QUOTE_EXPIRED") {
        return "当前价格已失效。请取消，刷新页面价格后再试。";
      }
      if (code === "DEPENDENCY_UNAVAILABLE" || code === "DEADLINE_EXCEEDED" || code === "INVALID_UPSTREAM_RESPONSE") {
        return "暂时无法确认价格，请取消后重试。";
      }
    });
  }
  function closeConfirmation() {
    if (confirmation) active.current?.abort();
    setConfirmation(null);
    setQuote(null);
    setSelected(null);
    setMessage("");
  }
  async function handle(
    result: ResourceOrder,
    intent: Pending,
    signal: AbortSignal,
  ) {
    signal.throwIfAborted();
    if (
      result.organization_id !== organizationId ||
      result.quote_id !== intent.quoteId ||
      (intent.orderId && result.order_id !== intent.orderId)
    )
      throw { code: "INVALID_UPSTREAM_RESPONSE" };
    if (!persist({ ...intent, orderId: result.order_id })) return;
    setOrder(result);
    setQuote(null);
    setSelected(null);
    if (result.status === "FULFILLED") {
      try {
        const zero=result.settlement_mode==="ZERO_PRICE";
        const [readback,walletReadback] = await Promise.all([
          getCommercialResources(userId,organizationId,signal),
          zero?Promise.resolve(null):getCommercialWallet(userId,organizationId,signal),
        ]);
        signal.throwIfAborted();
        const type = resourceDefinitions.find(
          (d) =>
            d.type ===
            (
              {
                AI_POINT: "ai_point",
                DATA_ROW: "data_row",
                STORE_RENEWAL_PERIOD: "store_renewal_period",
              } as const
            )[result.product_kind],
        );
        if (
          readback.organization_id !== organizationId ||
          (!zero && walletReadback?.organization_id !== organizationId) ||
          !type ||
          readback.resources.find((r) => r.resource_type === type.type)
            ?.state !== "recorded"
        )
          throw new Error("unconfirmed");
        setBalances(readback);
        if(walletReadback)client.setQueryData(["workbench",organizationId,"commercial-wallet",userId,"resource-purchase"],walletReadback);
        if (persist(null)) setMessage("资源已到账，余额已重新读取。");
        void client.invalidateQueries({
          queryKey: ["workbench", organizationId],
          refetchType: "active",
        });
      } catch {
        signal.throwIfAborted();
        setMessage(
          "订单已完成，但余额回读尚未确认。请查询原订单并重试余额回读。",
        );
      }
    } else if (result.status === "CANCELLED") {
      persist(null);
      setMessage(
        failures[result.failure_code ?? ""] ?? "原订单已取消，未购买资源。",
      );
    } else
      setMessage(
        result.status === "RECONCILIATION_REQUIRED"
          ? "正在核对原订单，请勿重新购买。"
          : "原订单正在处理，请稍后查询订单状态。",
      );
  }
  async function submit() {
    if (!ready || !quote || !selected || pending.value || !canPurchase || busy || locked.current)
      return;
    if (
      quote.settlement_mode!=="ZERO_PRICE" && (wallet.data?.organization_id !== organizationId ||
      BigInt(wallet.data.available_minor) < BigInt(quote.total_minor))
    ) {
      setMessage("企业钱包余额不足或尚未确认，请刷新钱包。");
      return;
    }
    const intent = {
      offerId: selected.offer_id,
      quoteId: quote.quote_id,
      key: crypto.randomUUID(),
    };
    if (!persist(intent)) return;
    setQuote(null);
    setSelected(null);
    setConfirmation(null);
    await run(
      async (signal) => {
        const result = await createResourceOrder(
          userId,
          organizationId,
          intent.quoteId,
          intent.key,
          signal,
        );
        await handle(result, intent, signal);
      },
      (error) => {
        if (noEffectErrors.has(errorCode(error))) persist(null);
      },
    );
  }
  async function resume() {
    const intent = pending.value;
    if (!intent || busy) return;
    await run(
      async (signal) => {
        const result = intent.orderId
          ? await getCommercialOrder(
              userId,
              organizationId,
              intent.orderId,
              signal,
            )
          : await createResourceOrder(
              userId,
              organizationId,
              intent.quoteId,
              intent.key,
              signal,
            );
        signal.throwIfAborted();
        if (result.kind !== "RESOURCE_PURCHASE")
          throw { code: "INVALID_UPSTREAM_RESPONSE" };
        await handle(result, intent, signal);
      },
      (error) => {
        if (!intent.orderId && noEffectErrors.has(errorCode(error)))
          persist(null);
      },
    );
  }
  const purchaseCards = (
    <div className={styles.resourcePurchaseCards}>
      {resourceDefinitions.map((def) => (
        <ResourcePurchaseAnchor key={def.type} id={def.type}>
          <ResourceCard
            definition={def}
            offers={offers.items.filter((o) => o.resource_type === def.type)}
            disabled={!ready || !!pending.value || (busy && !confirmation) || !canPurchase}
            onQuote={choose}
          />
        </ResourcePurchaseAnchor>
      ))}
    </div>
  );
  const pendingError = pending.invalid ? (
    <p role="alert">原订单恢复信息无效，请联系支持核对；已暂停新购买。</p>
  ) : null;
  const purchaseRestriction = !canPurchase ? <p>购买需由企业管理员操作。</p> : null;
  const walletRequired = quote && quote.settlement_mode !== "ZERO_PRICE";
  const walletBalance = wallet.data?.organization_id === organizationId
    ? wallet.data?.available_minor
    : undefined;
  const insufficientFunds = walletRequired && walletBalance !== undefined &&
    BigInt(walletBalance) < BigInt(quote.total_minor);
  const quotedResource = resourceDefinitions.find(({ type }) => type === quote?.resource_type);
  const quoteDetails = quote && selected && !pending.value ? (
    <>
      <dl className={styles.orderFacts}>
        <div>
          <dt>企业</dt>
          <dd>{organizationName}（{organizationId}）</dd>
        </div>
        <div>
          <dt>资源</dt>
          <dd>{quotedResource?.name}</dd>
        </div>
        <div>
          <dt>到账数量</dt>
          <dd>{quote.resource_quantity} {quotedResource?.unit}</dd>
        </div>
        <div>
          <dt>{quote.settlement_mode === "ZERO_PRICE" ? "合计（无需充值）" : "应付总价"}</dt>
          <dd>{resourceMoney(quote.total_minor)}</dd>
        </div>
        {quote.remainder_minor !== undefined ? (
          <div>
            <dt>未使用金额</dt>
            <dd>{resourceMoney(quote.remainder_minor)} 留在企业钱包</dd>
          </div>
        ) : null}
        <div>
          <dt>报价有效期</dt>
          <dd>{quote.expires_at.replace("T", " ").replace("Z", " UTC")}</dd>
        </div>
      </dl>
      {wallet.isError && walletRequired ? (
        <p role="alert">钱包余额暂不可确认，请刷新后重试。</p>
      ) : null}
      {insufficientFunds ? (
        <p role="alert">企业钱包余额不足。<Link className="underline" href={resourcePurchaseHref(walletBalanceId)} onClick={closeConfirmation}>前往充值</Link></p>
      ) : null}
      <div className={styles.actions}>
        <Button
          disabled={busy || Boolean(walletRequired && (walletBalance === undefined || insufficientFunds))}
          onClick={() => void submit()}
        >
          {quote.settlement_mode === "ZERO_PRICE" ? `确认领取${quote.resource_quantity}条` : "确认购买"}
        </Button>
        <Button variant="outline" onClick={closeConfirmation}>
          取消
        </Button>
      </div>
    </>
  ) : null;
  return (
    <div className={styles.stack}>
      <p className={styles.observation}>付费资源从企业钱包扣款，免费资源按数量领取。</p>
      {purchaseRestriction}
      {pendingError}
      {purchaseCards}
      {confirmation ? (
        <AccountDialog title={confirmation} onClose={closeConfirmation}>
          <div className={styles.confirmation}>
            {busy && !quote ? <p role="status">正在确认本次价格…</p> : null}
            {message ? <p role="alert">{message}</p> : null}
            {quoteDetails ?? (
              <div className={styles.actions}><Button variant="outline" onClick={closeConfirmation}>取消</Button></div>
            )}
          </div>
        </AccountDialog>
      ) : null}
      {pending.value ? (
        <Card role="region" aria-label="原订单恢复" className={styles.panel}>
          <h2>原订单待确认</h2>
          <p>
            报价编号：{pending.value.quoteId}
            {pending.value.orderId
              ? ` · 订单编号：${pending.value.orderId}`
              : ""}
          </p>
          <Button
            disabled={busy || (!pending.value.orderId && !canPurchase)}
            onClick={() => void resume()}
          >
            {pending.value.orderId ? "查询原订单" : "恢复原订单"}
          </Button>
        </Card>
      ) : null}
      {busy && !confirmation ? <p role="status">正在确认购买结果…</p> : null}
      {message && !confirmation ? <p role={balances ? "status" : "alert"}>{message}</p> : null}
      {order ? (
        <Card role="region" aria-label="资源订单" className={styles.panel}>
          <h2>资源订单</h2>
          <p>
            {order.order_id} · {order.status}
          </p>
          <div className={styles.actions}>
            <Button asChild variant="outline">
              <Link
                href={`/workbench/plans/orders/${encodeURIComponent(order.order_id)}`}
                prefetch={false}
              >
                查看订单
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/workbench/plans/entitlements" prefetch={false}>
                查看我的权益
              </Link>
            </Button>
          </div>
        </Card>
      ) : null}
    </div>
  );
}
function ResourceCard({
  definition,
  offers,
  disabled,
  onQuote,
}: {
  definition: (typeof resourceDefinitions)[number];
  offers: ResourceOffer[];
  disabled: boolean;
  onQuote: (
    offer: ResourceOffer,
    selection:
      | { quantity: string; amountMinor?: never }
      | { amountMinor: string; quantity?: never },
  ) => Promise<void>;
}) {
  const [offerId, setOfferId] = useState("");
  const offer = offers.find((o) => o.offer_id === offerId) ?? offers[0];
  const zero = offer?.settlement_mode === "ZERO_PRICE";
  const [mode, setMode] = useState("quantity");
  const [value, setValue] = useState("1");
  const byAmount = mode === "amount" && !zero;
  const selection = byAmount ? amountMinor(value) : positiveInt64(value) ? value : null;
  const unitPriceMinor = offer ? BigInt(offer.unit_price_minor) : BigInt(0);
  const quantity = selection === null ? null
    : !byAmount ? BigInt(selection)
    : unitPriceMinor > BigInt(0) ? BigInt(selection) / unitPriceMinor : null;
  const totalMinor = quantity === null ? null : (quantity * unitPriceMinor).toString();
  const withinRange = !!offer && quantity !== null &&
    quantity >= BigInt(offer.min_quantity) && quantity <= BigInt(offer.max_quantity);
  const estimateAvailable = withinRange && totalMinor !== null &&
    (totalMinor === "0" || positiveInt64(totalMinor));
  const price = offer ? (
    <p className={styles.offerPrice}>
      {resourceMoney(offer.unit_price_minor)} / {definition.unit}
    </p>
  ) : <p>价格未配置</p>;
  return (
    <Card
      role="region"
      aria-label={`${definition.name}购买`}
      data-resource={definition.type}
      className={`${styles.resourcePurchaseCard} ${styles[definition.tone]}`}
    >
      <h3>{definition.name}</h3>
      {price}
      <p>{definition.detail}</p>
      {offer ? (
        <>
          {offers.length > 1 ? (
            <label>
              报价选项
              <select
                value={offer.offer_id}
                onChange={(e) => setOfferId(e.target.value)}
              >
                {offers.map((o) => (
                  <option key={o.offer_id} value={o.offer_id}>
                    {o.offer_id} · {resourceMoney(o.unit_price_minor)} /{" "}
                    {definition.unit}
                  </option>
                ))}
              </select>
            </label>
          ) : null}
          {definition.type !== "store_renewal_period" && !zero ? (
            <label>
              购买方式
              <select
                aria-label="购买方式"
                value={mode}
                onChange={(e) => {
                  setMode(e.target.value);
                  setValue("1");
                }}
              >
                <option value="quantity">按数量</option>
                <option value="amount">按金额</option>
              </select>
            </label>
          ) : null}
          <label>
            {byAmount ? "金额（元）" : "购买数量"}
            <input
              aria-label={byAmount ? "金额（元）" : "购买数量"}
              inputMode={byAmount ? "decimal" : "numeric"}
              maxLength={22}
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
          </label>
          <p className={styles.subtle}>
            数量范围：{offer.min_quantity}–{offer.max_quantity}{" "}
            {definition.unit}
          </p>
          <p className={styles.estimate} aria-live="polite">
            {estimateAvailable
              ? `预计总价 ${resourceMoney(totalMinor)}${byAmount ? ` · 预计到账 ${quantity} ${definition.unit}` : ""}`
              : byAmount ? "请输入能购买上述数量范围的金额。" : "请输入有效的购买数量。"}
          </p>
        </>
      ) : null}
      <Button
        disabled={disabled || !offer || !estimateAvailable}
        onClick={() => {
          if (offer && selection)
            void onQuote(
              offer,
              byAmount
                ? { amountMinor: selection }
                : { quantity: selection },
            );
        }}
      >
        {zero ? "领取" : "购买"}
      </Button>
    </Card>
  );
}
