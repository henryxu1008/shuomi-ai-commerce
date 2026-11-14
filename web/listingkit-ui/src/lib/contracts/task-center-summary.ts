import {z} from "zod";

export const taskSummaryStates=["WAITING_CONFIRMATION","RUNNING","COMPLETED","ERROR","PAUSED"] as const;
export type TaskSummaryState=typeof taskSummaryStates[number];
const count=z.number().int().nonnegative().max(5000);
const counts=z.strictObject({WAITING_CONFIRMATION:count,RUNNING:count,COMPLETED:count,ERROR:count,PAUSED:count});
export const taskCenterSummarySchema=z.strictObject({status:z.enum(["complete","partial"]),counts:counts.nullable(),excluded:count,capturedAt:z.string().datetime(),reasons:z.array(z.enum(["unavailable","limit","changed"])).max(3)}).refine(v=>(v.status==="complete")===(v.counts!==null));
export type TaskCenterSummary=z.infer<typeof taskCenterSummarySchema>;
export type SummaryTask={id:string;kind?:string;agentRunId?:string;reviewId?:string;state?:string;projectionAvailable:boolean};
type SummaryReview={proposal_id:string;workbench_scoped?:boolean};
type SummaryRun={agentId:string;runId:string;state?:string};
export function taskRunKey(task:SummaryTask):string|undefined{
 const agent=task.kind==="product.content.localize"?"product.content.agent":task.kind==="product.title.optimize"?"product.title.agent":undefined;
 return agent&&task.agentRunId?`${agent}:${task.agentRunId}`:undefined;
}
export function summarizeTaskRecords(tasks:SummaryTask[],reviews:SummaryReview[],runs:SummaryRun[]){
 const result:Record<TaskSummaryState,number>={WAITING_CONFIRMATION:0,RUNNING:0,COMPLETED:0,ERROR:0,PAUSED:0};
 const linkedReviews=new Set<string>(),linkedRuns=new Set<string>();let excluded=0;
 const seen=new Map<string,string>(),taskFacts=new Map<string,string>();
 function add(key:string,state?:string){const value=state??"unclassified";if(seen.has(key)){if(seen.get(key)!==value)throw Error("inconsistent record");return}seen.set(key,value);if(taskSummaryStates.includes(state as TaskSummaryState))result[state as TaskSummaryState]++;else excluded++}
 for(const task of tasks){if(!task.projectionAvailable||!taskSummaryStates.includes(task.state as TaskSummaryState))throw Error("unreadable task");const facts=JSON.stringify([task.state,task.kind,task.agentRunId,task.reviewId]);if(taskFacts.has(task.id)&&taskFacts.get(task.id)!==facts)throw Error("inconsistent task");taskFacts.set(task.id,facts);add(`task:${task.id}`,task.state);if(task.reviewId)linkedReviews.add(task.reviewId);const run=taskRunKey(task);if(run)linkedRuns.add(run)}
 for(const review of reviews)if(!review.workbench_scoped&&!linkedReviews.has(review.proposal_id))add(`review:${review.proposal_id}`,"WAITING_CONFIRMATION");
 for(const run of runs)if(!linkedRuns.has(`${run.agentId}:${run.runId}`))add(`run:${run.agentId}:${run.runId}`,run.state);
 return {counts:result,excluded};
}
