import {expect,it} from "vitest";
import {assessmentPage,assessmentView,saveAssessment} from "./research-assessment";
import {researchEndpoint,researchIntent} from "./market-research";
import {blankCostDraft} from "@/lib/workbench/research-cost-preview";
const id="11111111-1111-4111-8111-111111111111";
const body={action:"save_assessment",candidateId:id,expectedRevision:2,linkId:id,evidenceDigest:"a".repeat(64),assessment:{cost:blankCostDraft()}};
it("keeps the full original save intent and rejects computed results or client scope",()=>{
 expect(researchIntent.safeParse({key:id,body}).success).toBe(true);
 expect(saveAssessment.safeParse({...body,organizationId:"other"}).success).toBe(false);
 expect(saveAssessment.safeParse({...body,assessment:{...body.assessment,profit:"123"}}).success).toBe(false);
 expect(researchIntent.safeParse({key:id,body:{...body,action:"review"}}).success).toBe(false);
});
it("admits only bounded preview and exact assessment paths",()=>{
 const url=(tail:string)=>new URL("https://test.local/api/workbench/research/projects/"+id+"/candidates/"+id+tail);
 expect(researchEndpoint(url("/assessment-preview"),"POST")?.readOnly).toBe(true);
 expect(researchEndpoint(url("/assessments/"+id),"GET")).not.toBeNull();
 for(const tail of ["/assessments?before=01","/assessments?before=2&before=3","/assessments/"+id+"?latest=true","/assessment-preview?actorId=x"]){expect(researchEndpoint(url(tail),tail.includes("preview")?"POST":"GET")).toBeNull()}
});
it("a masked history never carries saved conditions or amounts",()=>{
 const masked={id,candidateRevision:3,at:"2026-11-09T00:00:00Z",applicability:"DETAIL_UNAVAILABLE"};
 expect(assessmentPage.parse({items:[masked]}).items).toHaveLength(1);
 expect(assessmentView.safeParse({...masked,detail:{input:blankCostDraft()}}).success).toBe(false);
});
