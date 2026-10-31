"use client";
import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import {
  ConsolePage,
  ConsoleState,
  ConsoleToolbar,
} from "../console/console-page";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Card } from "@/components/ui/card";
import {
  listWorkbenchStores,
  type WorkbenchStore,
} from "@/lib/api/workbench-stores";
import {
  fulfillmentCapabilities,
  fulfillmentCatalog,
  fulfillmentDetail,
  fulfillmentID,
  fulfillmentOptions,
  fulfillmentPrivate,
  fulfillmentReceipt,
  fulfillmentRequest,
  fulfillmentSource,
  fulfillmentShipment,
  type FulfillmentInput,
  type FulfillmentOptions,
  type FulfillmentReceipt,
  type FulfillmentScope,
} from "@/lib/api/fulfillment";
import {
  loadFulfillmentIntent,
  saveFulfillmentIntent,
} from "@/lib/api/fulfillment-intent";
import {
  DemandPanel,
  MappingForm,
  PlatformForm,
  ProcurementPanel,
  type Write,
} from "./fulfillment-forms";
import styles from "./fulfillment.module.css";

const flowSteps = [
  { id: "source", label: "原单关联" },
  { id: "purchase", label: "采购与收货" },
  { id: "stock", label: "库存与装箱" },
  { id: "platform", label: "平台发货" },
  { id: "logistics", label: "交接与物流" },
] as const;
type FlowStep = (typeof flowSteps)[number]["id"];
const stages: Record<FulfillmentReceipt["stage"], string> = {
  PREPARED: "待核对确认",
  CONFIRMED: "已确认，等待执行",
  CLAIMED: "执行已领取，待核实",
  DISPATCHING: "已进入发送阶段，待核实",
  SUCCEEDED: "已受理 / 已记录",
  UNKNOWN: "结果未知，待核实",
  DEFINITIVE_NO_EFFECT: "明确未生效",
  CANCELLED: "已取消准备",
  PROCESSING: "处理中",
};
const purposes: Record<string, string> = {
  accept_address: "接受订单并获取履约地址",
  read_address: "读取履约地址",
  consumer_tracking: "上传实际运单",
  supply_shipping: "创建备货发货单 / 预约",
};
const initial = (params: ReturnType<typeof useSearchParams>, name: string) =>
  params?.getAll(name).length === 1 &&
  fulfillmentID.safeParse(params.get(name)).success
    ? params.get(name)!
    : "";
export function FulfillmentPage({
  procurement = false,
}: {
  procurement?: boolean;
}) {
  const context = useWorkbenchContext(),
    params = useSearchParams();
  const org = context.effectiveOrganization?.id ?? "",
    user = context.user?.id ?? "";
  const scope = useMemo(
    () => ({ organizationId: org, userId: user }),
    [org, user],
  );
  if (context.error || context.blockingError)
    return (
      <ConsoleState kind="error" title="企业上下文不可用">
        <Button onClick={() => void context.retry()}>重新确认</Button>
      </ConsoleState>
    );
  if (context.isLoading || context.isSwitching)
    return <ConsoleState kind="loading" title="正在确认企业…" />;
  if (!org || !user || context.selectionRequired)
    return <ConsoleState kind="unavailable" title="请先选择可访问的企业" />;
  return (
    <ScopedPage
      key={`${org}:${user}:${procurement}`}
      procurement={procurement}
      scope={scope}
      permissions={context.permissions}
      initialStore={initial(params, "storeId")}
      initialSource={initial(params, "sourceId")}
      initialKind={
        params?.get("kind") === "platform_supply"
          ? "platform_supply"
          : "consumer_order"
      }
    />
  );
}
function ScopedPage({
  scope,
  permissions,
  procurement,
  initialStore,
  initialSource,
  initialKind,
}: {
  scope: FulfillmentScope;
  permissions: string[];
  procurement: boolean;
  initialStore: string;
  initialSource: string;
  initialKind: FulfillmentInput["kind"];
}) {
  const [step, setStep] = useState<FlowStep>(
    procurement ? "purchase" : "source",
  );
  const queries = useQueryClient(),
    prefix = useMemo(
      () => ["fulfillment", scope.organizationId, scope.userId],
      [scope.organizationId, scope.userId],
    );
  const lifetime = useRef(new AbortController()),
    busy = useRef(false);
  const [store, setStore] = useState(initialStore),
    [kind, setKind] = useState(initialKind),
    [sourceID, setSourceID] = useState(initialSource),
    [lookup, setLookup] = useState(initialSource);
  const [intent, setIntent] = useState<FulfillmentInput | null>(null),
    [hydrated, setHydrated] = useState(false),
    [working, setWorking] = useState(false),
    [message, setMessage] = useState("");
  const [pages, setPages] = useState({
      catalog: 1,
      purchase: 1,
      delivery: 1,
      operation: 1,
      receipt: 1,
    }),
    [options, setOptions] = useState<FulfillmentOptions>(),
    [receipt, setReceipt] = useState<FulfillmentReceipt | null>(null);
  const [shipment, setShipment] = useState<z.infer<
    typeof fulfillmentShipment
  > | null>(null);
  const [privateData, setPrivateData] = useState<z.infer<
    typeof fulfillmentPrivate
  > | null>(null);
  useEffect(() => {
    const controller = new AbortController();
    lifetime.current = controller;
    let active = true;
    void Promise.resolve().then(() => {
      if (active) {
        setIntent(loadFulfillmentIntent(scope));
        setHydrated(true);
      }
    });
    return () => {
      active = false;
      controller.abort();
      void queries.cancelQueries({ queryKey: prefix });
      queries.removeQueries({ queryKey: prefix });
    };
  }, [scope, prefix, queries]);
  useEffect(() => {
    if (!privateData) return;
    const remaining = Date.parse(privateData.expires) - Date.now();
    const timer = setTimeout(
      () => setPrivateData(null),
      Math.max(0, Math.min(remaining, 86400000)),
    );
    return () => clearTimeout(timer);
  }, [privateData]);
  const command = (
    action: FulfillmentInput["action"],
    payload: Record<string, unknown>,
    operation = crypto.randomUUID(),
  ): FulfillmentInput => ({
    store,
    kind,
    source_id: sourceID,
    action,
    payload,
    operation_id: operation,
  });
  const read = <T,>(
    action: FulfillmentInput["action"],
    payload: Record<string, unknown>,
    schema: z.ZodType<T>,
    signal?: AbortSignal,
  ) =>
    fulfillmentRequest(
      scope,
      command(action, payload),
      schema,
      signal ?? lifetime.current.signal,
    );
  const caps = useQuery({
    queryKey: [...prefix, "capabilities", permissions],
    queryFn: ({ signal }) =>
      fulfillmentRequest(
        scope,
        "capabilities",
        fulfillmentCapabilities,
        signal,
      ),
    retry: false,
  });
  const available = caps.data?.available === true;
  const stores = useQuery({
    queryKey: [...prefix, "stores", permissions],
    enabled: available,
    retry: false,
    queryFn: async ({ signal }) => {
      const out: WorkbenchStore[] = [];
      for (let page = 1; page <= 5; page++) {
        const result = await listWorkbenchStores(
          { page, pageSize: 100, platform: "shein", status: "active" },
          scope.organizationId,
          signal,
        );
        if (result.pagination.total > 500) throw new Error("请缩小店铺范围");
        out.push(...result.items);
        if (page * 100 >= result.pagination.total) break;
      }
      return out;
    },
  });
  const selected =
    available && !!store && fulfillmentID.safeParse(sourceID).success;
  const identity = [store, kind, sourceID];
  const currentSelection = useRef("");
  useEffect(() => {
    currentSelection.current = [store, kind, sourceID].join(":");
  }, [store, kind, sourceID]);
  const source = useQuery({
    queryKey: [...prefix, "source", ...identity, permissions],
    enabled: selected,
    queryFn: ({ signal }) => read("source", {}, fulfillmentSource, signal),
    retry: false,
  });
  const found = useQuery({
    queryKey: [...prefix, "demand", ...identity, permissions],
    enabled: selected,
    queryFn: ({ signal }) =>
      read(
        "find_demand",
        {},
        z.object({ demand: z.string().nullable() }).strict(),
        signal,
      ),
    retry: false,
  });
  const catalog = useQuery({
    queryKey: [...prefix, "catalog", ...identity, pages.catalog, permissions],
    enabled: selected,
    queryFn: ({ signal }) =>
      read("catalog", { page: pages.catalog }, fulfillmentCatalog, signal),
    retry: false,
  });
  const detail = useQuery({
    queryKey: [
      ...prefix,
      "details",
      ...identity,
      found.data?.demand,
      pages,
      permissions,
    ],
    enabled: selected && !!found.data?.demand,
    queryFn: ({ signal }) =>
      read(
        "details",
        {
          demand: found.data!.demand,
          purchase_page: pages.purchase,
          delivery_page: pages.delivery,
          operation_page: pages.operation,
          receipt_page: pages.receipt,
        },
        fulfillmentDetail,
        signal,
      ),
    retry: false,
  });
  const reset = () => {
    setOptions(undefined);
    setReceipt(null);
    setPrivateData(null);
    setShipment(null);
    setPages({
      catalog: 1,
      purchase: 1,
      delivery: 1,
      operation: 1,
      receipt: 1,
    });
    setMessage("");
  };
  const refresh = async () => {
    setPrivateData(null);
    await queries.invalidateQueries({ queryKey: prefix });
  };
  const send = async (
    input: FulfillmentInput,
  ): Promise<FulfillmentReceipt | null> => {
    if (busy.current || !hydrated || lifetime.current.signal.aborted)
      return null;
    busy.current = true;
    setWorking(true);
    setMessage("");
    setPrivateData(null);
    try {
      if (!(await saveFulfillmentIntent(scope, input, null))) {
        setIntent(loadFulfillmentIntent(scope));
        setMessage(
          "未能保存原操作，或已有待核实操作。本次没有发送；请先恢复原操作。",
        );
        return null;
      }
      if (lifetime.current.signal.aborted) return null;
      setIntent(input);
      const result = await fulfillmentRequest(
        scope,
        input,
        fulfillmentReceipt,
        lifetime.current.signal,
      );
      if (lifetime.current.signal.aborted) return null;
      setReceipt(result);
      if (await saveFulfillmentIntent(scope, null, input)) setIntent(null);
      setMessage(stages[result.stage]);
      await refresh();
      return result;
    } catch {
      if (!lifetime.current.signal.aborted)
        setMessage(
          "未取得可确认结果。保留原操作；只可核对或以相同原操作重试，不能另建操作绕过。",
        );
      return null;
    } finally {
      busy.current = false;
      if (!lifetime.current.signal.aborted) setWorking(false);
    }
  };
  const write: Write = (action, payload, operation) =>
    send(command(action, payload, operation));
  const can = (purpose: string) =>
    permissions.includes(`workbench.fulfillment.${purpose}`);
  const disabled = working || !hydrated || intent !== null;
  const resolvePending = async (cancelOriginal = false) => {
    if (!intent || busy.current) return;
    const original = intent;
    busy.current = true;
    setWorking(true);
    const id = ["confirm", "ensure", "cancel"].includes(original.action)
      ? String(original.payload.operation)
      : original.operation_id;
    try {
      const result = await fulfillmentRequest(
        scope,
        {
          ...original,
          action: cancelOriginal ? "cancel_original" : "read",
          operation_id: id,
          payload: cancelOriginal
            ? {
                command: ["confirm", "ensure", "cancel"].includes(
                  original.action,
                )
                  ? "prepare"
                  : original.action,
              }
            : { operation: id },
        },
        fulfillmentReceipt,
        lifetime.current.signal,
      );
      if (lifetime.current.signal.aborted) return;
      setReceipt(result);
      if (await saveFulfillmentIntent(scope, null, original)) setIntent(null);
      setMessage(
        "已核对到原生持久回执。已领取或结果未知的原效果仍受原操作约束，请按回执继续核实。",
      );
      await refresh();
    } catch {
      if (!lifetime.current.signal.aborted)
        setMessage(
          "未能确认原回执或取消结果。继续保留原操作，不能按没有生效处理。",
        );
    } finally {
      busy.current = false;
      if (!lifetime.current.signal.aborted) setWorking(false);
    }
  };
  const openPending = () => {
    if (!intent) return;
    reset();
    setStore(intent.store);
    setKind(intent.kind);
    setLookup(intent.source_id);
    setSourceID(intent.source_id);
  };
  return (
    <ConsolePage
      title={procurement ? "采购管理" : "订单履约"}
      description="SHEIN 原单、原生 ERPNext 采购和库存、平台回执在同一流程中核对。真实平台联调待完成。"
      breadcrumbs={
        procurement
          ? [
              { label: "供应市场" },
              { label: "我的供应链", href: "/workbench/supply/mine" },
              { label: "采购管理" },
            ]
          : [
              { label: "订单管理", href: "/workbench/store-orders" },
              { label: "原单履约" },
            ]
      }
      actions={
        <Button
          variant="outline"
          disabled={working}
          onClick={() => void refresh()}
        >
          刷新原单与回执
        </Button>
      }
    >
      {caps.isPending ? (
        <ConsoleState kind="loading" title="正在确认履约能力…" />
      ) : !available ? (
        <ConsoleState kind="unavailable" title="履约暂不可用">
          请核对当前成员权限、官方连接、独立 ERP 站点与履约工作进程配置。
        </ConsoleState>
      ) : (
        <>
          <ConsoleToolbar>
            <form
              className={styles.fields}
              onSubmit={(e) => {
                e.preventDefault();
                if (fulfillmentID.safeParse(lookup).success) {
                  reset();
                  setSourceID(lookup);
                }
              }}
            >
              <label>
                当前授权店铺
                <Select
                  value={store}
                  onChange={(e) => {
                    reset();
                    setStore(e.target.value);
                    setSourceID("");
                  }}
                  required
                  disabled={working}
                >
                  <option value="">选择 SHEIN 店铺</option>
                  {stores.data?.map((item) => (
                    <option value={item.id} key={item.id}>
                      {item.name}
                    </option>
                  ))}
                </Select>
              </label>
              <label>
                原单类型
                <Select
                  value={kind}
                  onChange={(e) => {
                    reset();
                    setKind(e.target.value as FulfillmentInput["kind"]);
                    setSourceID("");
                  }}
                  disabled={working}
                >
                  <option value="consumer_order">
                    美国站消费者订单 · 自发货
                  </option>
                  <option value="platform_supply">原供应商备货单 · 交仓</option>
                </Select>
              </label>
              <label>
                平台原单号
                <Input
                  required
                  maxLength={128}
                  value={lookup}
                  onChange={(e) => setLookup(e.target.value)}
                  disabled={working}
                />
              </label>
              <Button disabled={working || !store}>查看原单</Button>
            </form>
            {stores.isError ? (
              <p role="alert">无法取得当前授权店铺，请刷新核对。</p>
            ) : null}
          </ConsoleToolbar>
          {intent ? (
            <Card className={styles.notice}>
              <h2>有一笔原操作待核实</h2>
              <p>
                {intent.source_id} · {intent.action} · {intent.operation_id}
              </p>
              <p>
                刷新或切换页面不会自动重发。核对后仍使用同一原操作与原载荷。
              </p>
              <div className={styles.actions}>
                <Button
                  variant="outline"
                  disabled={working}
                  onClick={openPending}
                >
                  返回这笔原单
                </Button>
                <Button
                  variant="outline"
                  disabled={working}
                  onClick={() => void resolvePending()}
                >
                  只核对原回执
                </Button>
                <Button disabled={working} onClick={() => void send(intent)}>
                  以原操作继续核对 / 重试
                </Button>
                <Button
                  variant="ghost"
                  disabled={working || !can(["receive","reserve","pack","dispatch"].includes(intent.action)?"stock":["bind_demand","create_purchase","record_external_order","close_purchase"].includes(intent.action)?"purchase":"execute")}
                  onClick={() => void resolvePending(true)}
                >
                  核对并取消尚未生效的原操作
                </Button>
              </div>
              <p>
                取消会以原键阻止迟到请求；已完成的单据保留原结果，已领取的外部效果保留核实状态。
              </p>
            </Card>
          ) : null}
          {message ? (
            <p role="status" className={styles.notice}>
              {message}
            </p>
          ) : null}
          {receipt ? (
            <Card className={styles.panel}>
              <h2>原操作回执</h2>
              <p>
                {receipt.operation} · {stages[receipt.stage]}
              </p>
              {receipt.target ? <p>原生单据：{receipt.target}</p> : null}
              {receipt.result.shipping_id ? (
                <p>平台备货发货单：{receipt.result.shipping_id}</p>
              ) : null}
              <p>受理回执不表示承运商已揽收、送达或平台已入库。</p>
            </Card>
          ) : null}
          {selected ? (
            <>
              <nav aria-label="履约子流程" className={styles.actions}>
                {flowSteps.map((flow) => (
                  <Button
                    key={flow.id}
                    variant={step === flow.id ? "default" : "outline"}
                    aria-current={step === flow.id ? "step" : undefined}
                    disabled={working}
                    onClick={() => {
                      setPrivateData(null);
                      setStep(flow.id);
                    }}
                  >
                    {flow.label}
                  </Button>
                ))}
              </nav>
              {found.data?.demand === null && step !== "source" ? (
                <ConsoleState kind="empty" title="先关联这笔原单">
                  <Button variant="outline" onClick={() => setStep("source")}>
                    进入原单关联
                  </Button>
                </ConsoleState>
              ) : null}
              {source.isError ? (
                <ConsoleState kind="error" title="暂时无法核对平台原单">
                  已保存的采购、收货与回执仍可在当前权限下查看。新增分配和平台操作需重新取得原单。
                </ConsoleState>
              ) : source.data ? (
                <Card className={styles.panel}>
                  <h2>平台原单 · {source.data.source.id}</h2>
                  <dl className={styles.facts}>
                    <div>
                      <dt>销售市场</dt>
                      <dd>
                        {kind === "platform_supply"
                          ? "未知（原授权供应商备货）"
                          : source.data.source.market}
                      </dd>
                    </div>
                    <div>
                      <dt>原平台状态</dt>
                      <dd>{source.data.source.status}</dd>
                    </div>
                    <div>
                      <dt>履约时限</dt>
                      <dd>
                        {source.data.source.deadline.startsWith("0001-")
                          ? "未返回当前时限"
                          : source.data.source.deadline}
                      </dd>
                    </div>
                    <div>
                      <dt>当前可新增履约</dt>
                      <dd>
                        {source.data.eligible
                          ? "符合本入口当前条件"
                          : "不符合 / 待核对"}
                      </dd>
                    </div>
                  </dl>
                  <div className={styles.tableWrap}>
                    <table>
                      <thead>
                        <tr>
                          <th>原 SKU</th>
                          <th>原需求</th>
                          <th>平台已发 / 原商品状态</th>
                          <th>平台实收</th>
                          <th>平台入库</th>
                          <th>平台次品</th>
                        </tr>
                      </thead>
                      <tbody>
                        {source.data.source.lines.map((line) => (
                          <tr key={line.id}>
                            <td>
                              {line.sku}
                              <small>{line.id}</small>
                            </td>
                            <td>{line.quantity}</td>
                            <td>
                              {kind === "consumer_order"
                                ? ({
                                    1: "待处理",
                                    2: "待发货",
                                    3: "待平台发货",
                                    4: "已发货",
                                    5: "已签收",
                                    6: "已退款",
                                    7: "待揽收",
                                  }[line.goodsStatus ?? 0] ?? "未识别")
                                : line.delivered}
                            </td>
                            <td>{line.received ?? "未知"}</td>
                            <td>{line.stored ?? "未知"}</td>
                            <td>{line.defective ?? "未知"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>
              ) : (
                <ConsoleState kind="loading" title="正在核对原单…" />
              )}
              {found.isError || catalog.isError || detail.isError ? (
                <ConsoleState kind="error" title="未取得完整原生履约记录">
                  请刷新核对；不要重新关联或另建单据。
                </ConsoleState>
              ) : null}
              {!found.isError &&
              step === "source" &&
              found.data?.demand === null &&
              source.data?.eligible &&
              catalog.data ? (
                <MappingForm
                  key={identity.join(":")}
                  source={source.data}
                  catalog={catalog.data}
                  disabled={disabled || !can("purchase")}
                  write={write}
                />
              ) : null}
              {catalog.data && (step === "source" || step === "purchase") ? (
                <div className={styles.pagination}>
                  <Button
                    variant="ghost"
                    disabled={pages.catalog === 1 || working}
                    onClick={() =>
                      setPages({ ...pages, catalog: pages.catalog - 1 })
                    }
                  >
                    上一页商品 / 供应商
                  </Button>
                  <span>第 {pages.catalog} 页，每页最多 20 项</span>
                  <Button
                    variant="ghost"
                    disabled={
                      working ||
                      (catalog.data.items.length < 20 &&
                        catalog.data.suppliers.length < 20)
                    }
                    onClick={() =>
                      setPages({ ...pages, catalog: pages.catalog + 1 })
                    }
                  >
                    下一页
                  </Button>
                </div>
              ) : null}
              {detail.data && catalog.data ? (
                <>
                  <p>
                    原生履约需求：{detail.data.name} · 数据保存在当前企业独立
                    ERPNext 站点。
                  </p>
                  {step === "purchase" ? (
                    <ProcurementPanel
                      key={detail.data.name}
                      detail={detail.data}
                      catalog={catalog.data}
                      disabled={disabled}
                      canPurchase={can("purchase")}
                      canStock={can("stock")}
                      write={write}
                    />
                  ) : null}
                  {step === "stock" ? (
                    <DemandPanel
                      key={detail.data.name}
                      detail={detail.data}
                      disabled={disabled || !source.data?.eligible}
                      canStock={can("stock")}
                      write={write}
                    />
                  ) : null}
                  {step === "platform" && source.data?.eligible ? (
                    <PlatformForm
                      key={identity.join(":")}
                      detail={detail.data}
                      kind={kind}
                      options={options}
                      disabled={disabled}
                      canExecute={can("execute")}
                      canAddress={can("address")}
                      addressConfigured={caps.data?.addressConfigured === true}
                      write={write}
                      onOptions={(payload) => {
                        const selectedKey = identity.join(":");
                        void read("options", payload, fulfillmentOptions)
                          .then((value) => {
                            if (
                              !lifetime.current.signal.aborted &&
                              currentSelection.current === selectedKey
                            )
                              setOptions(value);
                          })
                          .catch(() =>
                            setMessage(
                              "当前物流资料核对失败，请检查原单资格与配置。",
                            ),
                          );
                      }}
                    />
                  ) : null}
                  {step === "platform" || step === "logistics" ? (
                    <Card className={styles.panel}>
                      <h2>平台操作与物流核对</h2>
                      <p>
                        分别查看平台申报、实际交接和平台收货。消费者轨迹可在
                        <Link
                          href={`/workbench/store-orders?storeId=${encodeURIComponent(store)}`}
                          prefetch={false}
                        >
                          订单管理
                        </Link>
                        核对；托管收货 / 入库数量以上方原平台观察为准。
                      </p>
                      {detail.data.operations.length === 0 ? (
                        <p>尚无平台操作。</p>
                      ) : (
                        detail.data.operations.map((operation) => (
                          <section
                            className={styles.subpanel}
                            key={operation.operation}
                          >
                            <h3>
                              {purposes[operation.purpose]} ·{" "}
                              {stages[operation.stage]}
                            </h3>
                            <p>
                              {operation.operation}
                              {operation.delivery
                                ? ` · 原装箱单 ${operation.delivery}`
                                : ""}
                            </p>
                            {operation.result.shipping_id ? (
                              <p>
                                平台发货单：{operation.result.shipping_id}{" "}
                                <Button
                                  variant="outline"
                                  disabled={working}
                                  onClick={() => {
                                    const selectedKey = identity.join(":");
                                    void read(
                                      "shipment",
                                      { operation: operation.operation },
                                      fulfillmentShipment,
                                    )
                                      .then((value) => {
                                        if (
                                          !lifetime.current.signal.aborted &&
                                          currentSelection.current ===
                                            selectedKey
                                        ) {
                                          setShipment(value);
                                          setStep("logistics");
                                        }
                                      })
                                      .catch(() =>
                                        setMessage(
                                          "暂时无法读取这张原发货单的物流观察。",
                                        ),
                                      );
                                  }}
                                >
                                  查询原发货单物流
                                </Button>
                              </p>
                            ) : null}
                            <dl className={styles.facts}>
                              {Object.entries(operation.selection).map(
                                ([key, value]) => (
                                  <div key={key}>
                                    <dt>
                                      {{
                                        channel: "物流渠道",
                                        tracking: "实际运单",
                                        address_id: "原供应商地址 ID",
                                        warehouse_id: "供应商仓 ID",
                                        delivery_type: "交仓方式",
                                        shipping_route: "发货路线",
                                        arrival_time: "预约到仓（北京）",
                                        reserve_time: "预约取件（北京）",
                                        motorcade: "原车队",
                                        delivery: "原装箱单",
                                        demand: "原需求",
                                        purpose: "操作用途",
                                      }[key] ?? key}
                                    </dt>
                                    <dd>{String(value)}</dd>
                                  </div>
                                ),
                              )}
                            </dl>
                            {operation.stage === "PREPARED" ? (
                              <p>
                                确认将使用准备时的原单、包裹、物流和权限。任何变化都会停止发送；接受订单会锁定买家地址。
                              </p>
                            ) : null}
                            <div className={styles.actions}>
                              {step === "platform" &&
                              operation.stage === "PREPARED" ? (
                                <Button
                                  disabled={disabled || !can("execute")}
                                  onClick={() =>
                                    void write(
                                      "confirm",
                                      {
                                        operation: operation.operation,
                                        version: operation.revision,
                                      },
                                      operation.operation,
                                    )
                                  }
                                >
                                  确认执行这笔原操作
                                </Button>
                              ) : null}
                              {step === "platform" &&
                              operation.stage === "CONFIRMED" ? (
                                <Button
                                  disabled={disabled || !can("execute")}
                                  onClick={() =>
                                    void write(
                                      "ensure",
                                      {
                                        operation: operation.operation,
                                        version: operation.revision,
                                      },
                                      operation.operation,
                                    )
                                  }
                                >
                                  启动原操作
                                </Button>
                              ) : null}
                              {step === "platform" &&
                              ["PREPARED", "CONFIRMED"].includes(
                                operation.stage,
                              ) ? (
                                <Button
                                  variant="ghost"
                                  disabled={disabled || !can("execute")}
                                  onClick={() =>
                                    void write(
                                      "cancel",
                                      {
                                        operation: operation.operation,
                                        version: operation.revision,
                                      },
                                      operation.operation,
                                    )
                                  }
                                >
                                  取消尚未领取的准备
                                </Button>
                              ) : null}
                              {step === "platform" &&
                              operation.stage === "SUCCEEDED" &&
                              ["accept_address", "read_address"].includes(
                                operation.purpose,
                              ) ? (
                                <Button
                                  variant="outline"
                                  disabled={working || !can("address")}
                                  onClick={() => {
                                    const selectedKey = identity.join(":");
                                    void read(
                                      "read_private",
                                      { operation: operation.operation },
                                      fulfillmentPrivate,
                                    )
                                      .then((value) => {
                                        if (
                                          !lifetime.current.signal.aborted &&
                                          currentSelection.current ===
                                            selectedKey &&
                                          Date.parse(value.expires) > Date.now()
                                        )
                                          setPrivateData(value);
                                      })
                                      .catch(() =>
                                        setMessage(
                                          "地址已过期或当前权限不允许读取。",
                                        ),
                                      );
                                  }}
                                >
                                  临时查看本次履约地址
                                </Button>
                              ) : null}
                            </div>
                          </section>
                        ))
                      )}
                    </Card>
                  ) : null}
                  {step === "logistics" && shipment ? (
                    <Card className={styles.panel}>
                      <h2>原备货发货单物流观察 · {shipment.deliveryCode}</h2>
                      <p>
                        当前读取，平台时间为北京时间。平台取件与到仓记录独立于商家实际交接和平台入库。
                      </p>
                      <dl className={styles.facts}>
                        {[
                          ["平台承运商", shipment.expressCompanyName],
                          ["平台运单", shipment.expressCode],
                          ["平台创建时间", shipment.addTime],
                          ["实际取件时间", shipment.takeParcelTime],
                          ["预计到仓", shipment.preReceiptTime],
                          ["平台实际到仓", shipment.receiptTime],
                        ].map(([label, value]) => (
                          <div key={label}>
                            <dt>{label}</dt>
                            <dd>
                              {value && !value.startsWith("1970-")
                                ? value
                                : "未知 / 尚未返回"}
                            </dd>
                          </div>
                        ))}
                      </dl>
                    </Card>
                  ) : null}
                  {step === "platform" && privateData && can("address") ? (
                    <Card className={styles.private}>
                      <h2>本次履约地址</h2>
                      <p>
                        到期时间：{privateData.expires}
                        。关闭、切换原单或离开页面后清除显示。
                      </p>
                      <dl>
                        {Object.entries(privateData.private).map(
                          ([name, value]) => (
                            <div key={name}>
                              <dt>{name}</dt>
                              <dd>{value}</dd>
                            </div>
                          ),
                        )}
                      </dl>
                      <Button
                        variant="outline"
                        onClick={() => setPrivateData(null)}
                      >
                        关闭地址
                      </Button>
                    </Card>
                  ) : null}
                  {step === "logistics" ? (
                    <Card className={styles.panel}>
                      <h2>实际交接记录</h2>
                      <p>
                        货物实际交给承运商或平台仓后再确认。平台发货单受理本身不扣减源仓。
                      </p>
                      {detail.data.deliveries.map((delivery) => {
                        const platform = detail.data!.operations.find(
                          (op) =>
                            op.purpose === "supply_shipping" &&
                            op.delivery === delivery.name &&
                            op.stage === "SUCCEEDED",
                        );
                        return (
                          <section
                            className={styles.subpanel}
                            key={delivery.name}
                          >
                            <h3>
                              {delivery.package_id} ·{" "}
                              {delivery.docstatus === 1
                                ? "已实际交接"
                                : "装箱准备"}
                            </h3>
                            <p>
                              {delivery.name} ·{" "}
                              {delivery.lines.reduce(
                                (sum, line) => sum + line.qty,
                                0,
                              )}{" "}
                              件
                            </p>
                            {delivery.docstatus === 0 ? (
                              <form
                                onSubmit={(event) => {
                                  event.preventDefault();
                                  void write("dispatch", {
                                    demand: detail.data!.name,
                                    delivery: delivery.name,
                                    ...(kind === "platform_supply"
                                      ? {
                                          platform_operation:
                                            platform?.operation,
                                        }
                                      : {}),
                                  });
                                }}
                              >
                                <label>
                                  <input
                                    type="checkbox"
                                    required
                                    disabled={disabled}
                                  />{" "}
                                  我已核对该包裹及实际交接数量
                                </label>
                                <Button
                                  disabled={
                                    disabled ||
                                    !can("stock") ||
                                    (kind === "platform_supply" && !platform)
                                  }
                                >
                                  确认记录实际交接
                                </Button>
                                {kind === "platform_supply" && !platform ? (
                                  <p>
                                    需先取得这张原装箱单的平台创建成功回执；可翻阅平台操作页核对。
                                  </p>
                                ) : null}
                              </form>
                            ) : null}
                          </section>
                        );
                      })}
                    </Card>
                  ) : null}
                  {(["purchase", "delivery", "operation", "receipt"] as const)
                    .filter((key) =>
                      step === "purchase"
                        ? key === "purchase" || key === "receipt"
                        : step === "stock"
                          ? key === "delivery"
                          : step === "platform" || step === "logistics"
                            ? key === "delivery" || key === "operation"
                            : false,
                    )
                    .map((key) => (
                      <div className={styles.pagination} key={key}>
                        <Button
                          variant="ghost"
                          disabled={working || pages[key] === 1}
                          onClick={() =>
                            setPages({ ...pages, [key]: pages[key] - 1 })
                          }
                        >
                          上一页
                          {key === "purchase"
                            ? "采购单"
                            : key === "delivery"
                              ? "装箱单"
                              : key === "receipt"
                                ? "收货记录"
                                : "平台操作"}
                        </Button>
                        <span>第 {pages[key]} 页</span>
                        <Button
                          variant="ghost"
                          disabled={
                            working ||
                            !(key === "purchase"
                              ? detail.data!.has_more_purchases
                              : key === "delivery"
                                ? detail.data!.has_more_deliveries
                                : key === "receipt"
                                  ? detail.data!.purchases.some(
                                      (po) => po.has_more_receipts,
                                    )
                                  : detail.data!.has_more_operations)
                          }
                          onClick={() =>
                            setPages({ ...pages, [key]: pages[key] + 1 })
                          }
                        >
                          下一页
                        </Button>
                      </div>
                    ))}
                </>
              ) : null}
            </>
          ) : (
            <ConsoleState kind="empty" title="选择店铺并输入一笔平台原单">
              只读取当前授权店铺的原单，不会自动接受订单、采购或发货。
            </ConsoleState>
          )}
        </>
      )}
    </ConsolePage>
  );
}

export function FulfillmentEntry({
  procurement = false,
}: {
  procurement?: boolean;
}) {
  const context = useWorkbenchContext(),
    org = context.effectiveOrganization?.id ?? "",
    user = context.user?.id ?? "";
  const ready =
    !!org &&
    !!user &&
    !context.isSwitching &&
    !context.isLoading &&
    !context.error &&
    !context.blockingError &&
    context.permissions.includes("workbench.fulfillment.read");
  if (!ready) return null;
  return (
    <ScopedEntry
      key={`${org}:${user}`}
      scope={{ organizationId: org, userId: user }}
      permissions={context.permissions}
      procurement={procurement}
    />
  );
}
function ScopedEntry({
  scope,
  permissions,
  procurement,
}: {
  scope: FulfillmentScope;
  permissions: string[];
  procurement: boolean;
}) {
  const caps = useQuery({
    queryKey: [
      "fulfillment",
      scope.organizationId,
      scope.userId,
      "entry",
      permissions,
    ],
    queryFn: ({ signal }) =>
      fulfillmentRequest(
        scope,
        "capabilities",
        fulfillmentCapabilities,
        signal,
      ),
    retry: false,
  });
  if (!caps.data?.available) return null;
  return (
    <Card className={styles.notice}>
      <h2>{procurement ? "采购管理" : "订单履约"}</h2>
      <p>
        关联原单后生成采购、登记实际收货、预留库存、装箱并核对平台回执。真实平台联调待完成。
      </p>
      <Link
        href={
          procurement
            ? "/workbench/supply/mine/procurement"
            : "/workbench/store-orders/fulfill"
        }
        prefetch={false}
      >
        {procurement ? "进入采购管理" : "进入订单履约"}
      </Link>
    </Card>
  );
}
