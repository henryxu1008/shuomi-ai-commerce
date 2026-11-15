"use client";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {requestAIWorkbench,type AIScope} from "@/lib/api/ai-workbench";
import type {AIScheduledPlan} from "@/lib/contracts/ai-workbench";
import {scheduleTime} from "./schedule-form";
import {occurrenceStates,reasons} from "./schedule-presentation";
import styles from "./schedules.module.css";

export function ScheduleRecentResult({plan,scope,authorizationKey,refreshing}:{plan:AIScheduledPlan;scope:AIScope;authorizationKey:string;refreshing:boolean}){
 const recent=useQuery({queryKey:["ai-schedules",scope.userId,scope.organizationId,authorizationKey,plan.id,"recent"],enabled:plan.detailsAvailable,
  queryFn:({signal})=>requestAIWorkbench({route:"schedule-history",method:"GET",path:`schedules/${plan.id}/occurrences?limit=1`,scope,signal}),
  retry:false,gcTime:0,refetchOnWindowFocus:"always",refetchOnReconnect:"always"});
 const occurrence=recent.data?.occurrences[0];
 const label=occurrence?(occurrence.state==="RECONCILED"&&occurrence.reason==="REPORT_SAVED"?"日报已生成":({CLAIMED:"正在处理",DISPATCHED:"正在处理",BLOCKED:"执行受阻",UNKNOWN:"结果待确认"} as Record<string,string>)[occurrence.state]??occurrenceStates[occurrence.state]):"";
 return <div className={styles.recentResult} role="group" aria-label="最近执行结果">
  {!plan.detailsAvailable?<span className={styles.hint}>当前不可查看</span>:refreshing||recent.isFetching||recent.isPending?<span className={styles.hint}>正在读取结果…</span>:recent.isError?<span className={styles.hint}>结果暂不可用</span>:!occurrence?<span className={styles.hint}>尚未执行</span>:<>
   <span className={styles.state} data-state={occurrence.state}><span aria-hidden="true"/>{label}</span>
   <time className={styles.hint} dateTime={occurrence.scheduledAt}>{scheduleTime(occurrence.scheduledAt,plan.rule.timeZone)}</time>
   {occurrence.reason&&occurrence.reason!=="REPORT_SAVED"?<span className={styles.hint}>{reasons[occurrence.reason]??"可在执行历史中查看详情"}</span>:null}
   {occurrence.reportId?<Link href={`/workbench/ai/reports?reportId=${occurrence.reportId}`} prefetch={false}>查看日报</Link>:null}
  </>}
 </div>;
}
