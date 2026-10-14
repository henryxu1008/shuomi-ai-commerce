import {expect,it} from "vitest";
import {selectionEndpoint,selectionRunSchema,selectionPrepareSchema} from "./product-selection";
const contextId="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222";
it("admits only bounded explicit candidates and fixed analysis routes",()=>{
 const candidate={itemId:key,expectedRevision:"1",originalPublicationId:"source",originalVersion:"1"};
 expect(selectionPrepareSchema.safeParse({contextId,requestKey:key,expectedModeRevision:"0",goalSummary:"核对价格和规格",candidates:[candidate,{...candidate,itemId:contextId}]}).success).toBe(true);
 expect(selectionPrepareSchema.safeParse({contextId,requestKey:key,expectedModeRevision:"0",goalSummary:"",candidates:[candidate,candidate]}).success).toBe(false);
 expect(selectionEndpoint(new URL("https://console.example/api/workbench/product-selection/contexts/"+contextId+"/runs/"+key),"GET")).not.toBeNull();
 expect(selectionEndpoint(new URL("https://console.example/api/workbench/product-selection/publish"),"POST")).toBeNull();
 expect(selectionEndpoint(new URL("https://console.example/api/workbench/product-selection/inspect?tenant=other"),"POST")).toBeNull();
});
it("does not accept a repaired-out diagnostic as a successful suggestion",()=>{
 const response={runId:contextId,contextId,requestKey:key,phase:"human_review_required",revision:"1",stopReason:"repair_limit",configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:contextId,Digest:"a".repeat(64)},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:key,Digest:"b".repeat(64)},executionMode:"read_only",modeRevision:"0",goalSummary:"",validated:false,unresolved:["missing facts"],tokens:0,costMicros:0,currency:"CNY",usageStatus:"observed"};
 expect(selectionRunSchema.safeParse(response).success).toBe(true);
 expect(selectionRunSchema.safeParse({...response,validated:true}).success).toBe(false);
 expect(selectionRunSchema.safeParse({...response,credential:"private"}).success).toBe(false);
});
