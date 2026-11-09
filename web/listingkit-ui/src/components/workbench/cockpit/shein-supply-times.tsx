"use client";

import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {observationRequest,type ObservationScope} from "@/lib/api/store-observations";
import {supplyTimeDetailSchema,supplyTimeLabels} from "@/lib/contracts/shein-supply-time-detail";
import {Button} from "@/components/ui/button";
import {Table,TableHeader,TableBody,TableRow,TableHead,TableCell} from "@/components/ui/table";
import styles from "./shein-supply-orders.module.css";

const states={missing:"字段缺失",null:"明确空值（null）",empty:"空字符串",local_datetime:"无偏移日期格式；时区未确认",offset_datetime:"原文含偏移；未转换",unrecognized:"格式未识别",unexpected_type:"非预期类型；仅保留原值"};

export function SheinSupplyTimes({scope,storeId,syncId,ids}:{scope:ObservationScope;storeId:string;syncId:string;ids:string[]}){
 const [selected,setSelected]=useState("");
 const orderId=ids.includes(selected)?selected:"";
 const query=useQuery({queryKey:["operations-cockpit",scope.userId,scope.organizationId,"shein-supply-time",storeId,syncId,orderId],enabled:!!orderId,retry:false,gcTime:0,queryFn:({signal})=>observationRequest(`supply-orders/stores/${storeId}/syncs/${syncId}/records/${encodeURIComponent(orderId)}`,scope,supplyTimeDetailSchema,signal)});
 const record=query.isFetching||query.error?undefined:query.data;
 return <details className={styles.matched}><summary>查看业务时间原始值</summary><div className={styles.timeBody}>
  <p className={styles.note}>保留平台原文和缺失状态，时区未确认，不作转换；不以同步时间补原单日期。格式可识别不代表事件已确认，1970 等值不自动视为缺失。</p>
  <label className={styles.timeSelect}>已保存采购单<select aria-label="查看时间的采购单" className="min-w-0 max-w-full rounded-md border bg-background p-2" value={orderId} onChange={e=>setSelected(e.target.value)}><option value="">请选择原单</option>{ids.map(id=><option key={id} value={id}>{id}</option>)}</select></label>
  {orderId&&query.isFetching&&<p role="status">正在读取已保存时间…</p>}
  {orderId&&query.error&&<p role="alert">原单时间暂不可用。<Button variant="outline" onClick={()=>void query.refetch()}>重试原单详情</Button></p>}
  {record&&<><p className={styles.note}>原单：{record.id} · 来源：{record.supplyOrder.businessTimes.source} · 时区：未确认</p><Table className={styles.timeTable} aria-label="采购单原始时间"><TableHeader><TableRow><TableHead>业务时间</TableHead><TableHead>平台原值</TableHead><TableHead>状态</TableHead></TableRow></TableHeader><TableBody>{Object.entries(supplyTimeLabels).map(([field,label])=>{const v=record.supplyOrder.businessTimes.values[field as keyof typeof supplyTimeLabels];return <TableRow key={field}><TableCell>{label}<code>{field}</code></TableCell><TableCell className={styles.raw}>{v.raw===null?"无原始值":JSON.stringify(v.raw)}</TableCell><TableCell>{states[v.state]}<small>原类型：{v.rawType}</small></TableCell></TableRow>})}</TableBody></Table><p className={styles.note}>引号内为原文，空白和控制字符以转义显示；要求／截止时间与实际发生时间分别保留。</p></>}
 </div></details>;
}
