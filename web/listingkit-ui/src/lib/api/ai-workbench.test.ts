import {afterEach,expect,it,vi} from "vitest";
import {requestAIWorkbench} from "./ai-workbench";
afterEach(()=>vi.unstubAllGlobals());
it.each([false,undefined] as const)("preserves the explicit message acceptance evidence %s",async(messageAccepted)=>{
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({code:"INVALID_REQUEST",...(messageAccepted===false?{messageAccepted}:{})},{status:400})));
 const request=requestAIWorkbench({route:"message",method:"POST",path:"chat/conversations/id/messages",scope:{userId:"actor",organizationId:"org"},key:"key",body:{}});
 await expect(request).rejects.toMatchObject({code:"INVALID_REQUEST",messageAccepted});
});
