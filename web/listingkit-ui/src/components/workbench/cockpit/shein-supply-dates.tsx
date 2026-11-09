import {z} from "zod";
import {supplyObservationSchema} from "@/lib/contracts/shein-supply-orders";
import styles from "./cockpit-overview.module.css";

type Dates=NonNullable<z.infer<typeof supplyObservationSchema>["originalDates"]>;
const reasons={missing:"字段缺失",null:"空值",empty:"空字符串",unrecognized:"格式未识别",unexpected_type:"原值类型待核实",unusable_date:"原文日期不可用"};
export function SheinSupplyDates({data}:{data:Dates}){
 return <section aria-label="已保存供货单原文日期观察" className="grid gap-3 py-4">
  <h3>已保存供货单 · 原文日期观察</h3>
  <p className={styles.panelNote}>{data.range.startDate} 至 {data.range.endDate}（含起止日）。仅筛选当前已保存批次；按原文年月日比较，时区未确认，不作转换。</p>
  <div className="grid gap-3 sm:grid-cols-2">{([{name:"急采 · 下单日期",field:"addTime",group:data.urgent},{name:"备货 · 下发日期",field:"allocateTime",group:data.stock}]).map(({name,field,group})=><section key={field} aria-label={name} className="rounded-md border p-3">
   <h4>{name}</h4><p className={styles.panelNote}>日期来源：<code>{field}</code></p>
   <dl className={styles.publicationCounts}><div><dt>原单数</dt><dd>{group.orderIds.length}</dd></div><div><dt>订购数量（观察值）</dt><dd>{group.ordered.value??"待核实"}</dd>{group.ordered.missingLines>0&&<small>{group.ordered.missingLines} 行数量缺失或有歧义</small>}</div></dl>
   {group.orderIds.length>0&&<details><summary>查看匹配原单</summary><p className="break-all text-sm">{group.orderIds.join("、")}</p></details>}
  </section>)}</div>
  <dl className={styles.publicationCounts}><div><dt>日期待核实</dt><dd>{data.undated.length}</dd></div><div><dt>JIT 母单（不计入两组）</dt><dd>{data.motherIds.length}</dd></div><div><dt>母子分类待核实</dt><dd>{data.unknownMotherIds.length}</dd></div><div><dt>采购类型待核实</dt><dd>{data.unknownPurchaseTypeIds.length}</dd></div></dl>
  <p className={styles.panelNote}>区间外 {data.outsideRangeIds.length} 单。缺失或格式无法确认的日期不归入任何一天；母单和分类待核实单均不叠加。这里只表示原文日期，不确认事件发生或平台全量；1970 等值不自动改作缺失。订购数量不是当日发货或收货量。</p>
  {data.undated.length>0&&<details><summary>查看日期待核实原单</summary><ul className="grid gap-1 text-sm">{data.undated.map(row=><li key={row.id} className="break-all">{row.id} · {row.field==="addTime"?"急采下单":"备货下发"}（{row.field}）· {reasons[row.state]}</li>)}</ul></details>}
 </section>;
}
