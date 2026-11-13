import {expect,it} from "vitest";
import {aiMessageBody,aiWorkbenchPath,parseAIWorkbenchResponse} from "./ai-workbench";
const id="11111111-1111-4111-8111-111111111111";
const selection={subject:{kind:"own_product",id,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true,sellingPoints:true}};
it("keeps only validated live store projections and the explicit intent route",()=>{
 const stamp="2026-10-18T00:00:00Z", conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"actor"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:2,CreatedAt:stamp,UpdatedAt:stamp};
 const currentStoreOverview={total:3,platforms:{shein:2,temu:1},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp};
 const message={ID:id,ConversationID:id,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ACTIVE",currentStoreOverview};
 expect(aiWorkbenchPath("POST",["chat","conversations",id,"intent"])).toBe("store-route");
 expect(parseAIWorkbenchResponse("conversation-read",200,{conversation,messages:[message],proposals:[],before:""})).toMatchObject({messages:[{currentStoreOverview}]});
 expect(parseAIWorkbenchResponse("conversation-read",200,{conversation,messages:[{...message,currentStoreOverview:{...currentStoreOverview,total:-1}}],proposals:[],before:""})).toBeNull();
});
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

it("keeps original content policy adoption on task responses and rejects title/content proof confusion",()=>{
 const policyAdoption={execution:{scope:{OrganizationID:"org",ActorID:"actor"},runId:id,requestKey:id,binding:{ContextKind:"own_product",ContextID:id,ProductKey:"product",CatalogVersion:"3",PublicationID:"effective-publication",TargetPlatform:"product"},ContentSelection:selection.parameters,configuration:{Kind:"agent-configuration-v1",ID:id,Digest:"a".repeat(64)},mode:{Kind:"agent-execution-mode-v1",ID:id,Digest:"b".repeat(64)}},identity:{OrganizationID:"org",ActorID:"actor",MemberID:"member"},mode:"full_access",modeRevision:"1",policyVersion:"agent-execution-policy-v1",actionVersion:"product.content.policy-adopt-v1",proposalDigest:"c".repeat(64),requiresHuman:false};
 const task={id,conversationId:id,proposalId:id,kind:"product.content.localize",taskSelection:selection,title:"Localize",goalSummary:"Localize",createdAt:"2026-10-18T00:00:00Z",projectionAvailable:true,state:"COMPLETED",canStart:false,canReconcile:false,canResume:false,canReview:false,productDetailsAvailable:true,policyAdoption};
 for(const [route,payload] of [["task-read",{task}],["task-list",{tasks:[task],next:""}],["confirm",{task,replay:false}]] as const)expect(parseAIWorkbenchResponse(route,200,payload)).toEqual(payload);
 expect(parseAIWorkbenchResponse("task-read",200,{task:{...task,kind:"product.title.optimize",taskSelection:undefined}})).toBeNull();
 expect(parseAIWorkbenchResponse("task-read",200,{task:{...task,policyAdoption:{...policyAdoption,actionVersion:"product.title.policy-adopt-v1"}}})).toBeNull();
});
