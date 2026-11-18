import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ManagedModelChannels} from "./managed-model-channels";

afterEach(()=>{
 cleanup();
 sessionStorage.clear();
 vi.unstubAllGlobals();
});

it("recovers an added channel by the original receipt without sending its Key again",async()=>{
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true;}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false;}});
 const organizationId="org-a",userId="operator",memberId="member-a";
 const connectionId="uc-92ff6319-fdb7-5c55-9fa6-c7c16c0048d1";
 const template="openai-mini-v1";
 const registration={organizationId,connectionId,template,name:"官方渠道",revision:"1"};
 const connection={organizationId,connectionId,provider:"openai",channel:"openai-direct",region:"global",account:"default",
  revision:"1",keyPresent:true,enabled:true,models:[],baseUrl:"https://api.openai.com/v1",registration};
 let requestKey="",committed=false;
 const fetch=vi.fn(async(url:string,init?:RequestInit)=>{
  if(init?.method==="POST"){
   requestKey=new Headers(init.headers).get("Idempotency-Key")??"";
   expect(JSON.parse(String(init.body))).toEqual({expectedMemberId:memberId,input:{template,name:"官方渠道",apiKey:"synthetic-private-key"}});
   committed=true;
   throw new Error("fixture lost acknowledgement");
  }
  if(url.endsWith("/receipts/"+requestKey)){
   return Response.json({actorId:userId,memberId,organizationId,requestKey,result:{connection,registration}});
  }
  return Response.json({organizationId,channels:committed?[connection]:[],bindings:[],templates:[template],
   canSave:true,canRename:true,canAttach:true,modelQueryTargets:null});
 });
 vi.stubGlobal("fetch",fetch);
 const client=new QueryClient();
 const user=userEvent.setup();
 render(<QueryClientProvider client={client}><ManagedModelChannels scope={{organizationId,userId}} memberId={memberId}
  blocked={false} onSaved={()=>client.invalidateQueries({queryKey:["model-services"]})} onCredentials={vi.fn()}/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"添加渠道"}));
 await user.type(screen.getByLabelText("渠道名称"),"官方渠道");
 await user.type(screen.getByLabelText("API Key"),"synthetic-private-key");
 await user.click(screen.getByRole("button",{name:"保存"}));
 await waitFor(()=>expect(screen.getByLabelText("API Key")).toHaveValue(""));
 await screen.findByRole("alert");
 const saved=JSON.stringify({...sessionStorage});
 expect(saved).toContain(requestKey);
 expect(saved).not.toContain("synthetic-private-key");
 expect(saved).not.toContain("apiKey");
 await user.click(screen.getByRole("button",{name:"取消"}));
 await user.click(screen.getByRole("button",{name:"查询本次结果"}));
 expect(await screen.findByRole("heading",{name:"官方渠道"})).toBeVisible();
 expect(screen.getByRole("heading",{name:"尚未关联模型"})).toBeVisible();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
 await user.click(screen.getByRole("button",{name:"返回渠道列表"}));
 expect(screen.getByRole("button",{name:"官方渠道"})).toBeVisible();
 client.clear();
});
