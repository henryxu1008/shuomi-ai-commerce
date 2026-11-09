"use client";

import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {observationRequest,type ObservationScope} from "@/lib/api/store-observations";
import {supplyTimeDetailSchema,supplyTimeLabels} from "@/lib/contracts/shein-supply-time-detail";
import {Button} from "@/components/ui/button";

const states={missing:"字段缺失",null:"明确空值（null）",empty:"空字符串",local_datetime:"无偏移日期格式；时区未确认",offset_datetime:"原文含偏移；未转换",unrecognized:"格式未识别",unexpected_type:"非预期类型；仅保留原值"};

export function SheinSupplyTimes({scope,storeId,syncId,ids}:{scope:ObservationScope;storeId:string;syncId:string;ids:string[]}){
 const [selected,setSelected]=useState("");
 const orderId=ids.includes(selected)?selected:"";
 const query=useQuery({queryKey:["operations-cockpit",scope.userId,scope.organizationId,"shein-supply-time",storeId,syncId,orderId],enabled:!!orderId,retry:false,gcTime:0,queryFn:({signal})=>observationRequest(`supply-orders/stores/${storeId}/syncs/${syncId}/records/${encodeURIComponent(orderId)}`,scope,supplyTimeDetailSchema,signal)});
 const record=query.isFetching||query.error?undefined:query.data;
 return <details className="grid gap-3 p-3"><summary>查看业务时间原始值</summary>
  <p className="text-sm text-muted-foreground">平台时区未确认，仅展示原文，不做转换或日期汇总。格式可识别不代表事件已确认；1970 等值不自动视为缺失。同步时间不会补入原单日期。</p>
  <label className="grid gap-2 py-3">已保存采购单<select aria-label="查看时间的采购单" className="min-w-0 max-w-full rounded-md border p-2" value={orderId} onChange={e=>setSelected(e.target.value)}><option value="">请选择原单</option>{ids.map(id=><option key={id} value={id}>{id}</option>)}</select></label>
  {orderId&&query.isFetching&&<p role="status">正在读取已保存时间…</p>}
  {orderId&&query.error&&<p role="alert">原单时间暂不可用。<Button variant="outline" onClick={()=>void query.refetch()}>重试原单详情</Button></p>}
  {record&&<><p className="break-all text-sm">原单：{record.id} · 来源：{record.supplyOrder.businessTimes.source} · 时区：未确认</p><dl className="grid gap-3 py-3">{Object.entries(supplyTimeLabels).map(([field,label])=>{const v=record.supplyOrder.businessTimes.values[field as keyof typeof supplyTimeLabels];return <div key={field} className="min-w-0 rounded-md border p-3"><dt>{label} <code className="break-all text-xs">{field}</code></dt><dd className="whitespace-pre-wrap break-all font-mono text-sm">{v.raw===null?"无原始值":JSON.stringify(v.raw)}</dd><dd className="text-sm text-muted-foreground">{states[v.state]} · 原类型：{v.rawType}</dd></div>})}</dl><p className="text-sm text-muted-foreground">引号内为原文，空白和控制字符以转义显示；要求／截止时间与实际发生时间分别保留。</p></>}
 </details>;
}
