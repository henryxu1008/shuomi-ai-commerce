import {expect,it} from "vitest";
import {summarizeTaskRecords} from "./task-center-summary";

it("counts five disjoint states and correlates exact task, review and agent run identities",()=>{
 const tasks=[{id:"t1",kind:"product.content.localize",agentRunId:"run1",reviewId:"review1",state:"WAITING_CONFIRMATION",projectionAvailable:true},{id:"t2",state:"PAUSED",projectionAvailable:true}];
 const native=[{agentId:"product.content.agent",runId:"run1",state:"RUNNING"},{agentId:"product.image.agent",runId:"run1",state:"ERROR"},{agentId:"product.content.agent",runId:"run2"}];
 expect(summarizeTaskRecords(tasks,[{proposal_id:"review1"},{proposal_id:"review2"},{proposal_id:"review3",workbench_scoped:true}],native)).toEqual({counts:{WAITING_CONFIRMATION:2,RUNNING:0,COMPLETED:0,ERROR:1,PAUSED:1},excluded:1});
});
it("does not infer pending or completed from unclassified original runs",()=>{
 expect(summarizeTaskRecords([],[],[{agentId:"a",runId:"generated"},{agentId:"b",runId:"cancelled"}])).toEqual({counts:{WAITING_CONFIRMATION:0,RUNNING:0,COMPLETED:0,ERROR:0,PAUSED:0},excluded:2});
});
it("does not publish precise counts from unreadable task projections",()=>{
 expect(()=>summarizeTaskRecords([{id:"t",projectionAvailable:false}],[],[])).toThrow();
});
