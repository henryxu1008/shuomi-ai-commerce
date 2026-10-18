import {afterEach,expect,it,vi} from "vitest";
import {requestAIWorkbench} from "./ai-workbench";
afterEach(()=>vi.unstubAllGlobals());
it("retains the owner's confirmed task reference through response validation",async()=>{
 const id="550e8400-e29b-41d4-a716-446655440000",taskId="5892474d-1c8a-47e4-9550-45ed946e8197";
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({conversation:{ID:id,Scope:{OrganizationID:"org",ActorID:"actor"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:2,CreatedAt:"2026-10-03T00:00:00Z",UpdatedAt:"2026-10-03T00:00:00Z"},messages:[],before:"",proposals:[{id,digest:"a".repeat(64),sourceSequence:1,goalSummary:"Generate content",confirmedTaskId:taskId,humanReviewRequired:true,detailsAvailable:false,titleProfileReady:false,executionAuthorized:false}]})));
 await expect(requestAIWorkbench({route:"conversation-read",method:"GET",path:`chat/conversations/${id}`,scope:{userId:"actor",organizationId:"org"}})).resolves.toMatchObject({proposals:[{confirmedTaskId:taskId}]});
});
it.each([false,undefined] as const)("preserves the explicit message acceptance evidence %s",async(messageAccepted)=>{
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({code:"INVALID_REQUEST",...(messageAccepted===false?{messageAccepted}:{})},{status:400})));
 const request=requestAIWorkbench({route:"message",method:"POST",path:"chat/conversations/id/messages",scope:{userId:"actor",organizationId:"org"},key:"key",body:{}});
 await expect(request).rejects.toMatchObject({code:"INVALID_REQUEST",messageAccepted});
});
