import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {EnterpriseTextChoice,PlatformChannelConfiguration} from "./model-channel-configuration";

function prepare() {
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true;}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false;}});
 return new QueryClient();
}
const scope={userId:"actor",organizationId:"org-a"};
const ref={id:"pm-official-a",version:"v1"};
const a={category:"text",reference:ref,channel:"openai-direct",channelName:"渠道甲",source:"platform",model:"gpt-5.4-mini-2026-03-17",ready:true,quote:"quote-a"};
const b={...a,reference:{...ref,id:"pm-official-b"},channelName:"渠道乙",quote:"quote-b"};

afterEach(()=>{cleanup();sessionStorage.clear();vi.unstubAllGlobals();});

it("switches only the chosen text option and reads the saved channel again",async()=>{
 const client=prepare(),user=userEvent.setup(),saved=vi.fn(async()=>{});
 let switched=false;
 vi.stubGlobal("fetch",vi.fn(async(_url:string,init:RequestInit)=>{
  if(init.method==="POST") {
   expect(JSON.parse(String(init.body))).toEqual({reference:b.reference,quote:"quote-b",expectedRevision:"prior",expectedMemberId:"member"});
   switched=true;
   return Response.json({...scope, userId:undefined,actorId:scope.userId,requestKey:new Headers(init.headers).get("Idempotency-Key"),revision:"next",reference:b.reference});
  }
  return Response.json({organizationId:scope.organizationId,view:{revision:switched?"next":"prior",current:switched?b:{...a,ready:false,reason:"CHANNEL_DISABLED"},options:[b],purposes:["aiworkbench_chat_plan","product.content.decision"]}});
 }));
 render(<QueryClientProvider client={client}><EnterpriseTextChoice scope={scope} member="member" onSaved={saved}/></QueryClientProvider>);
 expect(await screen.findByText("当前渠道已停用，请切换。")).toBeVisible();
 await user.click(screen.getByRole("button",{name:"切换"}));
 await user.click(screen.getByRole("radio",{name:/渠道乙/}));
 expect(screen.getByText("本次切换影响硕米 Chat、商品内容。")).toBeVisible();
 await user.click(screen.getByRole("button",{name:"确认切换"}));
 await waitFor(()=>expect(saved).toHaveBeenCalledOnce());
 expect(await screen.findByText("渠道乙")).toBeVisible();
 expect(screen.queryByRole("dialog")).toBeNull();
 client.clear();
});

it("clears the platform key and recovers a successful create after a lost reply",async()=>{
 const client=prepare(),user=userEvent.setup(),saved=vi.fn(async()=>{});
 const platformOwner="platform-model-channel-owner",connectionId="uc-92ff6319-fdb7-5c55-9fa6-c7c16c0048d1",template="openai-mini-v1";
 const registration={organizationId:platformOwner,connectionId,template,name:"主渠道",revision:"1"};
 const connection={organizationId:platformOwner,connectionId,provider:"openai",channel:"openai-direct",region:"global",account:"default",revision:"1",enabled:true,keyPresent:true,models:[],registration,baseUrl:"https://api.openai.com/v1"};
 let key="";
 const fetch=vi.fn(async(url:string,init:RequestInit)=>{
  if(url.endsWith("/platform-prices"))return Response.json({status:"UNCONFIGURED",revision:null,preview:null});
  if(init.method==="POST"){
   key=new Headers(init.headers).get("Idempotency-Key")!;
   expect(JSON.parse(String(init.body))).toEqual({name:"主渠道",template,apiKey:"synthetic-platform-key",expectedMemberId:"member"});
   throw new Error("lost response");
  }
  if(url.endsWith("/receipts/"+key)){
   return Response.json({organizationId:scope.organizationId,actorId:scope.userId,requestKey:key,result:{registration,connection}});
  }
  return Response.json({organizationId:scope.organizationId,canWrite:true,channels:key?[{connection,revision:"NONE"}]:[],organizations:["org-a"]});
 });
 vi.stubGlobal("fetch",fetch);
 render(<QueryClientProvider client={client}><PlatformChannelConfiguration scope={scope} member="member" onSaved={saved}/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"添加平台渠道"}));
 await user.type(screen.getByLabelText("渠道名称"),"主渠道");
 await user.type(screen.getByLabelText("API Key"),"synthetic-platform-key");
 await user.click(screen.getByRole("button",{name:"保存"}));
 expect(screen.getByLabelText("API Key")).toHaveValue("");
 expect(JSON.stringify({...sessionStorage})).not.toContain("synthetic-platform-key");
 await user.click(screen.getByRole("button",{name:"取消"}));
 await user.click(await screen.findByRole("button",{name:"查询本次结果"}));
 expect(await screen.findByRole("button",{name:"主渠道"})).toBeVisible();
 expect(fetch.mock.calls.filter(([,call])=>call.method==="POST")).toHaveLength(1);
 client.clear();
});
