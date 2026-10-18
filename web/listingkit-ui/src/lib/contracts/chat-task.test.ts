import {expect,it} from "vitest";
import {aiMessageBody,parseAIWorkbenchResponse} from "./ai-workbench";
const id="11111111-1111-4111-8111-111111111111";
const selection={subject:{kind:"own_product",id,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true,sellingPoints:true}};
it("accepts explicit source-neutral content and rejects mixed or model-selected authority",()=>{
 expect(aiMessageBody.safeParse({content:"Localize",taskSelection:selection}).success).toBe(true);
 for(const taskSelection of [{...selection,subject:{...selection.subject,kind:"unknown"}},{...selection,subject:{...selection.subject,catalogVersion:"03"}},{...selection,parameters:{targetLanguage:"und",title:true}},{...selection,parameters:{targetLanguage:"en-US"}},{...selection,providerId:"fake"}])expect(aiMessageBody.safeParse({content:"Localize",taskSelection}).success).toBe(false);
 expect(aiMessageBody.safeParse({content:"Localize",taskSelection:selection,operationId:id,targetPlatform:"shein"}).success).toBe(false);
 expect(aiMessageBody.safeParse({content:"Title",operationId:id,targetPlatform:"shein"}).success).toBe(true);
});
it("keeps the effective subject and complete task limits on a content proposal",()=>{
 const value={state:"COMPLETE",userMessageId:id,sourceSequence:1,assistantMessageId:id,proposalId:id,proposal:{id,digest:"a".repeat(64),sourceSequence:1,kind:"product.content.localize",goalSummary:"Localize",taskSelection:selection,targetPlatform:"product",humanReviewRequired:true,detailsAvailable:true,titleProfileReady:false,executionProfileReady:true,executionAuthorized:true,maximumTokens:6000,maximumCostMicros:1000,maximumSteps:8,maximumModelCalls:3,maximumRuntimeSeconds:60}};
 expect(parseAIWorkbenchResponse("message",200,value)).toMatchObject({proposal:value.proposal});
});
