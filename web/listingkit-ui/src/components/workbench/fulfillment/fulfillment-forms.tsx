"use client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Card } from "@/components/ui/card";
import type {
  FulfillmentCatalog,
  FulfillmentDetail,
  FulfillmentInput,
  FulfillmentOptions,
  FulfillmentReceipt,
  FulfillmentSource,
} from "@/lib/api/fulfillment";
import styles from "./fulfillment.module.css";

export type Write = (
  action: FulfillmentInput["action"],
  payload: Record<string, unknown>,
  operation?: string,
) => Promise<FulfillmentReceipt | null>;
const number = (form: FormData, name: string) => Number(form.get(name) ?? 0);
const today = () => new Date().toLocaleDateString("en-CA");
export function MappingForm({
  source,
  catalog,
  disabled,
  write,
}: {
  source: FulfillmentSource;
  catalog: FulfillmentCatalog;
  disabled: boolean;
  write: Write;
}) {
  const [mapping, setMapping] = useState<
    Record<string, FulfillmentCatalog["items"][number]>
  >({});
  const [units, setUnits] = useState<Record<string, string>>({});
  const lines = source.source.lines.filter(
    (line) => line.quantity > line.delivered,
  );
  return (
    <Card className={styles.panel}>
      <h2>关联 SKU 与单位</h2>
      <p>
        将平台原商品逐行关联到仓库商品。需求数量使用原单剩余数量；本次确认不向供应商下单。
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          void write("bind_demand", {
            delivery_date: String(data.get("date")),
            lines: lines.map((line) => {
              const item = mapping[line.id];
              return {
                source_line: line.id,
                item_code: item?.name ?? "",
                uom: units[line.id] ?? "",
                warehouse: catalog.source_warehouse,
              };
            }),
          });
        }}
      >
        <div className={styles.tableWrap}>
          <table>
            <thead>
              <tr>
                <th>平台商品 / 原行</th>
                <th>待履约</th>
                <th>仓库商品</th>
                <th>单位</th>
              </tr>
            </thead>
            <tbody>
              {lines.map((line) => (
                <tr key={line.id}>
                  <td>
                    {line.sku}
                    <small>{line.id}</small>
                  </td>
                  <td>{line.quantity - line.delivered}</td>
                  <td>
                    <Select
                      aria-label={`关联 ${line.sku} ${line.id}`}
                      required
                      value={mapping[line.id]?.name ?? ""}
                      onChange={(e) => {
                        const item = catalog.items.find(
                          (item) => item.name === e.target.value,
                        );
                        setMapping((previous) => {
                          const next = { ...previous };
                          if (item) next[line.id] = item;
                          else delete next[line.id];
                          return next;
                        });
                        setUnits((previous) => ({
                          ...previous,
                          [line.id]: item?.stock_uom ?? "",
                        }));
                      }}
                    >
                      <option value="">选择准确商品</option>
                      {mapping[line.id] &&
                      !catalog.items.some(
                        (item) => item.name === mapping[line.id].name,
                      ) ? (
                        <option value={mapping[line.id].name}>
                          {mapping[line.id].item_name} · {mapping[line.id].name}
                        </option>
                      ) : null}
                      {catalog.items.map((item) => (
                        <option value={item.name} key={item.name}>
                          {item.item_name} · {item.name}
                        </option>
                      ))}
                    </Select>
                  </td>
                  <td>
                    <Select
                      required
                      aria-label={`原单单位 ${line.sku} ${line.id}`}
                      value={units[line.id] ?? ""}
                      onChange={(event) =>
                        setUnits({ ...units, [line.id]: event.target.value })
                      }
                    >
                      <option value="">核对原商品单位</option>
                      {mapping[line.id]?.uoms.map((unit) => (
                        <option value={unit.uom} key={unit.uom}>
                          {unit.uom}（1 = {unit.conversion_factor}{" "}
                          {mapping[line.id].stock_uom}）
                        </option>
                      ))}
                    </Select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <label>
          计划交付日期
          <Input type="date" name="date" required min={today()} />
        </label>
        <Button
          disabled={disabled || lines.some((line) => !mapping[line.id])}
          type="submit"
        >
          确认关联并准备履约
        </Button>
      </form>
    </Card>
  );
}
export function DemandPanel({
  detail,
  disabled,
  canStock,
  write,
}: {
  detail: FulfillmentDetail;
  disabled: boolean;
  canStock: boolean;
  write: Write;
}) {
  const [packageName, setPackageName] = useState("");
  return (
    <Card className={styles.panel}>
      <h2>库存分配与装箱</h2>
      <p>
        {detail.movement_kind === "owned_transfer"
          ? "实际交仓前货物仍属本企业；交接后移入在途仓。"
          : "实际交接后记录本次出库。"}{" "}
        装箱准备保留草稿，实际交接再扣减源仓。
      </p>
      <div className={styles.tableWrap}>
        <table>
          <thead>
            <tr>
              <th>商品</th>
              <th>需求</th>
              <th>已交接</th>
              <th>预留</th>
              <th>现货可分配</th>
              <th>可采购</th>
            </tr>
          </thead>
          <tbody>
            {detail.lines.map((line) => (
              <tr key={line.line}>
                <td>
                  {line.item_code}
                  <small>{line.uom}</small>
                </td>
                <td>{line.qty}</td>
                <td>{line.delivered_qty}</td>
                <td>{line.reserved_qty}</td>
                <td>{line.reservable_qty}</td>
                <td>{line.purchase_remaining}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Button
        variant="outline"
        disabled={
          disabled ||
          !canStock ||
          !detail.lines.some((line) => line.reservable_qty > 0)
        }
        onClick={() =>
          void write("reserve", {
            demand: detail.name,
            lines: detail.lines
              .filter((line) => line.reservable_qty > 0)
              .map((line) => ({
                demand_line: line.line,
                qty: line.reservable_qty,
              })),
          })
        }
      >
        预留可用库存
      </Button>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          const lines = detail.lines
            .map((line) => ({
              demand_line: line.line,
              qty: number(data, line.line),
            }))
            .filter((line) => line.qty > 0);
          if (lines.length)
            void write("pack", {
              demand: detail.name,
              package_id: packageName,
              lines,
            });
        }}
      >
        <h3>装箱准备</h3>
        <label>
          本次包裹编号
          <Input
            required
            maxLength={140}
            value={packageName}
            onChange={(e) => setPackageName(e.target.value)}
            placeholder="使用仓库实际包裹编号"
          />
        </label>
        <div className={styles.fields}>
          {detail.lines.map((line) => (
            <label key={line.line}>
              {line.item_code} · 本次装箱
              <Input
                name={line.line}
                type="number"
                min="0"
                max={Math.floor(
                  Math.min(line.reserved_qty, line.qty - line.delivered_qty),
                )}
                step="1"
                defaultValue="0"
              />
            </label>
          ))}
        </div>
        <Button disabled={disabled || !canStock || !packageName} type="submit">
          保存装箱准备
        </Button>
      </form>
    </Card>
  );
}
export function ProcurementPanel({
  detail,
  catalog,
  disabled,
  canPurchase,
  canStock,
  write,
}: {
  detail: FulfillmentDetail;
  catalog: FulfillmentCatalog;
  disabled: boolean;
  canPurchase: boolean;
  canStock: boolean;
  write: Write;
}) {
  const [priceError, setPriceError] = useState("");
  return (
    <Card className={styles.panel}>
      <h2>采购与到货</h2>
      <p>
        生成内部采购单后，在供应商平台完成下单和付款，再登记外部单号。只有实际合格收货进入可用库存。
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          const lines = detail.lines
            .map((line) => ({
              demand_line: line.line,
              qty: number(data, `qty:${line.line}`),
              rate: number(data, `rate:${line.line}`),
            }))
            .filter((line) => line.qty > 0);
          if (
            lines.some(
              (line) =>
                String(data.get(`rate:${line.demand_line}`) ?? "").trim() ===
                "",
            )
          ) {
            setPriceError("请为本次采购商品明确填写单价，可填写 0。");
            return;
          }
          setPriceError("");
          if (lines.length)
            void write("create_purchase", {
              demand: detail.name,
              supplier: String(data.get("supplier")),
              delivery_date: String(data.get("date")),
              lines,
            });
        }}
      >
        <div className={styles.fields}>
          <label>
            供应商
            <Select name="supplier" required defaultValue="">
              <option value="">选择已配置供应商</option>
              {catalog.suppliers.map((supplier) => (
                <option value={supplier.name} key={supplier.name}>
                  {supplier.supplier_name}
                </option>
              ))}
            </Select>
          </label>
          <label>
            预计交付日期
            <Input name="date" type="date" min={today()} required />
          </label>
        </div>
        <div className={styles.tableWrap}>
          <table>
            <thead>
              <tr>
                <th>商品</th>
                <th>本次采购</th>
                <th>单价（{catalog.currency}）</th>
              </tr>
            </thead>
            <tbody>
              {detail.lines.map((line) => (
                <tr key={line.line}>
                  <td>
                    {line.item_code} · {line.uom}
                  </td>
                  <td>
                    <Input
                      aria-label={`采购数量 ${line.item_code}`}
                      type="number"
                      min="0"
                      max={line.purchase_remaining}
                      step="1"
                      defaultValue="0"
                      name={`qty:${line.line}`}
                    />
                  </td>
                  <td>
                    <Input
                      aria-label={`采购单价 ${line.item_code}`}
                      name={`rate:${line.line}`}
                      type="number"
                      min="0"
                      step="0.0001"
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {priceError ? <p role="alert">{priceError}</p> : null}
        <Button
          type="submit"
          disabled={
            disabled ||
            !canPurchase ||
            !detail.lines.some((line) => line.purchase_remaining > 0)
          }
        >
          确认生成采购单
        </Button>
      </form>
      {detail.purchases.length === 0 ? (
        <p>尚无关联采购单；已有库存可以直接预留。</p>
      ) : (
        detail.purchases.map((purchase) => (
          <section key={purchase.name} className={styles.subpanel}>
            <h3>
              {purchase.name} · {purchase.supplier}
            </h3>
            <p>
              预计交付：{purchase.delivery_date} · {purchase.currency} ·{" "}
              {purchase.external_order
                ? `外部单号：${purchase.external_order}`
                : "尚未登记外部下单"}
            </p>
            {!purchase.external_order ? (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  const data = new FormData(event.currentTarget);
                  void write("record_external_order", {
                    purchase: purchase.name,
                    external_order: String(data.get("external")),
                  });
                }}
              >
                <label>
                  供应商平台订单号
                  <Input name="external" required maxLength={140} />
                </label>
                <Button variant="outline" disabled={disabled || !canPurchase}>
                  登记已在外部下单
                </Button>
              </form>
            ) : null}
            <form
              onSubmit={(event) => {
                event.preventDefault();
                const data = new FormData(event.currentTarget);
                const lines = purchase.lines
                  .map((line) => ({
                    purchase_line: line.line,
                    qty: number(data, line.line),
                    rejected_qty: number(data, `rejected:${line.line}`),
                    short_qty: number(data, `short:${line.line}`),
                  }))
                  .filter((line) => line.qty + line.rejected_qty > 0);
                if (lines.length)
                  void write("receive", { purchase: purchase.name, lines });
              }}
            >
              <div className={styles.tableWrap}>
                <table>
                  <thead>
                    <tr>
                      <th>商品</th>
                      <th>采购数量</th>
                      <th>累计到货（含拒收）</th>
                      <th>未到货</th>
                      <th>本次合格实收</th>
                      <th>本次拒收隔离</th>
                      <th>本次报告短缺</th>
                    </tr>
                  </thead>
                  <tbody>
                    {purchase.lines.map((line) => (
                      <tr key={line.line}>
                        <td>
                          {line.item_code} · {line.uom}
                        </td>
                        <td>{line.qty}</td>
                        <td>{line.received_qty}</td>
                        <td>{line.qty - line.received_qty}</td>
                        <td>
                          <Input
                            aria-label={`合格收货 ${purchase.name} ${line.item_code}`}
                            name={line.line}
                            type="number"
                            step="1"
                            min="0"
                            max={line.qty - line.received_qty}
                            defaultValue="0"
                          />
                        </td>
                        <td>
                          <Input
                            aria-label={`拒收隔离 ${purchase.name} ${line.item_code}`}
                            name={`rejected:${line.line}`}
                            type="number"
                            min="0"
                            step="1"
                            max={line.qty - line.received_qty}
                            defaultValue="0"
                          />
                        </td>
                        <td>
                          <Input
                            aria-label={`本次短缺 ${purchase.name} ${line.item_code}`}
                            name={`short:${line.line}`}
                            type="number"
                            min="0"
                            step="1"
                            max={line.qty - line.received_qty}
                            defaultValue="0"
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Button
                variant="outline"
                disabled={
                  disabled ||
                  !canStock ||
                  purchase.status === "Closed" ||
                  purchase.lines.every((line) => line.received_qty >= line.qty)
                }
              >
                确认登记合格收货
              </Button>
            </form>
            {purchase.receipts.map((receipt) => (
              <section key={receipt.name}>
                <h4>
                  收货记录 {receipt.name} · {receipt.date}
                </h4>
                {receipt.lines.map((row) => (
                  <p key={row.purchase_line}>
                    {purchase.lines.find(
                      (line) => line.line === row.purchase_line,
                    )?.item_code ?? row.purchase_line}
                    ：合格 {row.accepted_qty}，拒收隔离 {row.rejected_qty}
                    ，本次报告短缺 {row.short_qty}
                  </p>
                ))}
              </section>
            ))}
            {purchase.has_more_receipts ? (
              <p>还有收货记录，请翻阅收货记录下一页。</p>
            ) : null}
            {purchase.status !== "Closed" &&
            purchase.lines.some((line) => line.received_qty < line.qty) ? (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  const data = new FormData(event.currentTarget);
                  void write("close_purchase", {
                    purchase: purchase.name,
                    reason: String(data.get("reason")),
                  });
                }}
              >
                <label>
                  剩余关闭原因
                  <Input name="reason" required maxLength={500} />
                </label>
                <label>
                  <input type="checkbox" required />{" "}
                  已与供应商确认剩余不再交付，本次仅关闭内部剩余采购
                </label>
                <Button variant="ghost" disabled={disabled || !canPurchase}>
                  确认关闭内部剩余采购
                </Button>
              </form>
            ) : null}
          </section>
        ))
      )}
    </Card>
  );
}
export function PlatformForm({
  detail,
  kind,
  options,
  disabled,
  canExecute,
  canAddress,
  addressConfigured,
  write,
  onOptions,
}: {
  detail: FulfillmentDetail;
  kind: FulfillmentInput["kind"];
  options: FulfillmentOptions | undefined;
  disabled: boolean;
  canExecute: boolean;
  canAddress: boolean;
  addressConfigured: boolean;
  write: Write;
  onOptions: (payload: Record<string, unknown>) => void;
}) {
  const [delivery, setDelivery] = useState(""),
    [address, setAddress] = useState(""),
    [deliveryType, setDeliveryType] = useState("");
  return (
    <Card className={styles.panel}>
      <h2>{kind === "consumer_order" ? "商家自发货" : "平台交仓"}</h2>
      {kind === "consumer_order" ? (
        <section className={styles.subpanel}>
          <h3>履约地址</h3>
          <p>
            “接受订单并获取履约地址”会让平台订单转为待发货，并限制买家修改地址。准备后还需单独确认。
          </p>
          <div className={styles.actions}>
            <Button
              variant="outline"
              disabled={
                disabled || !canExecute || !canAddress || !addressConfigured
              }
              onClick={() =>
                void write("prepare", {
                  purpose: "accept_address",
                  demand: detail.name,
                  shipping_route: 0,
                })
              }
            >
              准备接受订单并获取履约地址
            </Button>
            <Button
              variant="ghost"
              disabled={
                disabled || !canExecute || !canAddress || !addressConfigured
              }
              onClick={() =>
                void write("prepare", {
                  purpose: "read_address",
                  demand: detail.name,
                  shipping_route: 0,
                })
              }
            >
              准备仅读取履约地址
            </Button>
          </div>
        </section>
      ) : (
        <p>
          使用原授权供应商备货单，销售市场显示未知。当前运输入口支持官方允许的直送自送、自提、上门查货或车队方式；其它方式保持未开放。
        </p>
      )}
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          void write(
            "prepare",
            kind === "consumer_order"
              ? {
                  purpose: "consumer_tracking",
                  demand: detail.name,
                  delivery,
                  shipping_route: 0,
                  channel: String(data.get("channel")),
                  tracking: String(data.get("tracking")),
                }
              : {
                  purpose: "supply_shipping",
                  demand: detail.name,
                  delivery,
                  shipping_route: 0,
                  address_id: Number(address),
                  warehouse_id: number(data, "warehouse"),
                  delivery_type: Number(deliveryType),
                  arrival_time:
                    String(data.get("arrival") ?? "").replace("T", " ") + ":00",
                  reserve_time:
                    deliveryType === "2"
                      ? String(data.get("reserve") ?? "").replace("T", " ") +
                        ":00"
                      : "",
                  motorcade: String(data.get("motorcade") ?? ""),
                },
          );
        }}
      >
        <label>
          原装箱单
          <Select
            required
            value={delivery}
            onChange={(e) => setDelivery(e.target.value)}
          >
            <option value="">选择本次原包裹</option>
            {detail.deliveries.map((note) => (
              <option key={note.name} value={note.name}>
                {note.package_id} · {note.name} ·{" "}
                {note.docstatus === 0 ? "准备中" : "已交接"}
              </option>
            ))}
          </Select>
        </label>
        {kind === "consumer_order" ? (
          <>
            <Button
              variant="outline"
              type="button"
              disabled={disabled}
              onClick={() =>
                onOptions({
                  purpose: "consumer_tracking",
                  demand: detail.name,
                  delivery,
                  shipping_route: 0,
                })
              }
            >
              核对当前可用物流
            </Button>
            <div className={styles.fields}>
              <label>
                平台当前承运渠道
                <Select name="channel" required defaultValue="">
                  <option value="">先核对当前可用物流</option>
                  {options?.channels?.map((channel) => (
                    <option
                      value={channel.expressChannelCode}
                      key={channel.expressChannelCode}
                    >
                      {channel.expressIdCode} · {channel.expressChannelCode}
                    </option>
                  ))}
                </Select>
              </label>
              <label>
                实际运单号
                <Input name="tracking" required maxLength={100} />
              </label>
            </div>
            <p>
              上传受理只表示平台接受本次运单信息；真实揽收、发货和签收分别核实。
            </p>
          </>
        ) : (
          <>
            <Button
              variant="outline"
              type="button"
              disabled={disabled}
              onClick={() =>
                onOptions({
                  purpose: "supply_shipping",
                  demand: detail.name,
                  delivery,
                  shipping_route: 0,
                  ...(address ? { address_id: Number(address) } : {}),
                })
              }
            >
              核对当前发货资料
            </Button>
            <div className={styles.fields}>
              <label>
                原供应商发货地址
                <Select
                  required
                  value={address}
                  onChange={(e) => {
                    setAddress(e.target.value);
                    onOptions({
                      purpose: "supply_shipping",
                      demand: detail.name,
                      delivery,
                      shipping_route: 0,
                      address_id: Number(e.target.value),
                    });
                  }}
                >
                  <option value="">选择官方返回地址</option>
                  {options?.addressList?.map((row) => (
                    <option key={row.addressId} value={row.addressId}>
                      {row.addressName}
                    </option>
                  ))}
                </Select>
              </label>
              <label>
                商家仓库
                <Select name="warehouse" required defaultValue="">
                  <option value="">选择官方返回仓库</option>
                  {options?.supplierWarehouseList?.map((row) => (
                    <option
                      key={row.supplierWarehouseId}
                      value={row.supplierWarehouseId}
                    >
                      {row.warehouseName}
                    </option>
                  ))}
                </Select>
              </label>
              <label>
                交仓方式
                <Select
                  required
                  value={deliveryType}
                  onChange={(e) => setDeliveryType(e.target.value)}
                >
                  <option value="">选择官方允许方式</option>
                  {options?.deliveryTypeList
                    ?.filter((row) =>
                      [2, 4, 21].includes(row.deliveryTypeValue),
                    )
                    .map((row) => (
                      <option
                        value={row.deliveryTypeValue}
                        key={row.deliveryTypeValue}
                      >
                        {row.deliveryTypeName}
                      </option>
                    ))}
                </Select>
              </label>
              <label>
                预约到仓时间（北京时间）
                <Input type="datetime-local" name="arrival" required />
              </label>
              {deliveryType === "2" ? (
                <>
                  <label>
                    原车队
                    <Select name="motorcade" required defaultValue="">
                      <option value="">选择官方返回车队</option>
                      {options?.motorcadeList?.map((row) => (
                        <option key={row.motorcadeId} value={row.motorcadeId}>
                          {row.motorcadeName}
                        </option>
                      ))}
                    </Select>
                  </label>
                  <label>
                    预约取件时间（北京时间）
                    <Input type="datetime-local" name="reserve" required />
                  </label>
                </>
              ) : null}
            </div>
            <p>
              本入口按原备货单全部剩余数量、一个实际包裹准备；系统会重核原单全部
              SKU、发货资格与收货仓。不能用分批收货推断平台允许分批发货。
            </p>
          </>
        )}
        <Button type="submit" disabled={disabled || !canExecute || !delivery}>
          准备并核对本次平台操作
        </Button>
      </form>
    </Card>
  );
}
