import type {z} from "zod";
import type {supplyObservationSchema} from "@/lib/contracts/shein-supply-orders";
import {Table,TableHeader,TableBody,TableRow,TableHead,TableCell} from "@/components/ui/table";
import styles from "./shein-supply-orders.module.css";

type Dates=NonNullable<z.infer<typeof supplyObservationSchema>["originalDates"]>;
const reasons={missing:"字段缺失",null:"空值",empty:"空字符串",unrecognized:"格式未识别",unexpected_type:"原值类型待核实",unusable_date:"原文日期不可用"};
export function SheinSupplyDates({data}:{data:Dates}){
 const groups=[{name:"急采",date:"下单日期",group:data.urgent},{name:"备货",date:"下发日期",group:data.stock}];
 return <section aria-label="已保存供货单原文日期观察" className={styles.dates}>
  <div className={styles.range}><span>{data.range.startDate} 至 {data.range.endDate}</span><span>按原文日期 · 含起止日</span></div>
  <Table aria-label="所选日期的供货统计" className={styles.dateTable}>
   <TableHeader><TableRow><TableHead>采购类型</TableHead><TableHead>原单数</TableHead><TableHead>订购数量</TableHead></TableRow></TableHeader>
   <TableBody>{groups.map(({name,date,group})=><TableRow key={name} aria-label={`${name} · ${date}`}>
    <TableCell><div className={styles.kind}><strong>{name}</strong><span>{date}</span></div></TableCell>
    <TableCell><span className={styles.number}>{group.orderIds.length}</span></TableCell>
    <TableCell className={styles.quantity}><span className={styles.number}>{group.ordered.value??"待核实"}</span>{group.ordered.missingLines>0&&<small>{group.ordered.missingLines} 行数量缺失或有歧义</small>}</TableCell>
   </TableRow>)}</TableBody>
  </Table>
  <div className={styles.checks}>
   {data.undated.length>0&&<details><summary>日期待核实 {data.undated.length} 单 · 未计入日期统计</summary><ul>{data.undated.map(row=><li key={row.id}>{row.id} · {row.field==="addTime"?"急采下单":"备货下发"} · {reasons[row.state]}</li>)}</ul></details>}
   {data.unknownMotherIds.length>0&&<details><summary>母子分类待核实 {data.unknownMotherIds.length} 单 · 未计入两组</summary><p className={styles.ids}>{data.unknownMotherIds.join("、")}</p></details>}
   {data.unknownPurchaseTypeIds.length>0&&<details><summary>采购类型待核实 {data.unknownPurchaseTypeIds.length} 单 · 未计入两组</summary><p className={styles.ids}>{data.unknownPurchaseTypeIds.join("、")}</p></details>}
  </div>
  <p className={styles.footnote}><span>区间外 {data.outsideRangeIds.length} 单</span><span>JIT 母单已排除 {data.motherIds.length} 单</span></p>
  {groups.some(x=>x.group.orderIds.length>0)&&<details className={styles.matched}><summary>查看匹配原单</summary>{groups.filter(x=>x.group.orderIds.length>0).map(({name,group})=><p key={name}>{name}：{group.orderIds.join("、")}</p>)}</details>}
 </section>;
}
