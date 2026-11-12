import {act,cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {AcquisitionAPIError} from "@/lib/api/product-acquisition";
import {OnlineAcquisitionHub} from "./online-acquisition-hub";

const state=vi.hoisted(()=>({acquire:vi.fn(),verify:vi.fn(),read:vi.fn(),push:vi.fn()}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:state.push}),useSearchParams:()=>new URLSearchParams("agent_kind=content")}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org"},acquisitionIntentReady:true,pendingAcquisitionIntent:null,setPendingAcquisitionIntent:()=>true,registerOrganizationSwitchGuard:()=>()=>{}})}));
vi.mock("@/lib/api/product-acquisition",async original=>({...await original<object>(),acquire1688:state.acquire,verify1688:state.verify,readAcquisition:state.read}));
vi.mock("@/components/workbench/collections/collection-page",()=>({CollectionDialog:({title,children,onClose}:{title:string;children:React.ReactNode;onClose:()=>void})=><section role="dialog" aria-label={title}><button onClick={onClose}>关闭</button>{children}</section>}));

afterEach(()=>{cleanup();sessionStorage.clear();vi.unstubAllGlobals();vi.clearAllMocks()});

it("keeps the real 1688 request alive and its original key recoverable when switching to Amazon",async()=>{
 let rejectRequest!:(reason:unknown)=>void;
 state.acquire.mockImplementation(()=>new Promise((_,reject)=>{rejectRequest=reject}));
 const options={sites:[],customSites:[],fields:[],maximumRows:200,formats:[],acquisitionReady:false,unavailableReason:"测试环境未启用"};
 const fetcher=vi.fn<(url:string,init?:RequestInit)=>Promise<Response>>(async url=>Response.json(url.endsWith("/options")?options:[]));
 vi.stubGlobal("fetch",fetcher);
 render(<OnlineAcquisitionHub has1688 hasAmazon initial="1688"/>);
 fireEvent.change(screen.getByLabelText("1688 商品页或 offer ID"),{target:{value:"123"}});
 fireEvent.click(screen.getByRole("button",{name:"提交采集"}));
 await waitFor(()=>expect(state.acquire).toHaveBeenCalledTimes(1));
 const [original,signal]=state.acquire.mock.calls[0];
 fireEvent.click(screen.getByRole("button",{name:"Amazon"}));
 expect(signal.aborted).toBe(false);
 await waitFor(()=>expect(fetcher).toHaveBeenCalledTimes(2));
 expect(fetcher.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
 await act(async()=>{rejectRequest(new AcquisitionAPIError("OUTCOME_UNKNOWN",503))});
 fireEvent.click(screen.getByRole("button",{name:"1688"}));
 expect(screen.getByLabelText("1688 商品页或 offer ID")).toHaveValue("123");
 expect(screen.getByRole("button",{name:"使用原 key 核实"})).toBeEnabled();
 expect(state.acquire).toHaveBeenCalledTimes(1);
 expect(original.key).toBeTruthy();
 fireEvent.click(screen.getByRole("button",{name:"Amazon"}));
 expect(fetcher).toHaveBeenCalledTimes(2);
});

it.each(["submit","verify","read"] as const)("does not leave an Amazon draft when hidden 1688 %s completes",async action=>{
 let finish!:(value:unknown)=>void;
 const pending=()=>new Promise(resolve=>{finish=resolve});
 state.acquire.mockReset();state.verify.mockReset();state.read.mockReset();state.push.mockReset();
 if(action==="submit")state.acquire.mockImplementation(pending);
 if(action==="verify"){state.acquire.mockRejectedValueOnce(new AcquisitionAPIError("OUTCOME_UNKNOWN",503));state.verify.mockImplementation(pending)}
 if(action==="read")state.read.mockImplementation(pending);
 const options={sites:[{code:"us",domain:"www.amazon.com",name:"美国"}],customSites:[],fields:["asin"],price:{offerId:"test-offer",pricingVersion:"test-price",currency:"CNY",unitPriceFen:"5"},maximumRows:200,formats:["csv"],acquisitionReady:true};
 const fetcher=vi.fn<(url:string,init?:RequestInit)=>Promise<Response>>(async url=>Response.json(url.endsWith("/options")?options:[]));
 vi.stubGlobal("fetch",fetcher);
 render(<OnlineAcquisitionHub has1688 hasAmazon initial="1688"/>);
 const operationId="11111111-1111-4111-8111-111111111111";
 if(action==="read"){
  fireEvent.change(screen.getByLabelText("操作 ID"),{target:{value:operationId}});fireEvent.click(screen.getByRole("button",{name:"读取"}));
  await waitFor(()=>expect(state.read).toHaveBeenCalledTimes(1));
 }else{
  fireEvent.change(screen.getByLabelText("1688 商品页或 offer ID"),{target:{value:"123"}});fireEvent.click(screen.getByRole("button",{name:"提交采集"}));
  await waitFor(()=>expect(state.acquire).toHaveBeenCalledTimes(1));
  if(action==="verify"){fireEvent.click(await screen.findByRole("button",{name:"使用原 key 核实"}));await waitFor(()=>expect(state.verify).toHaveBeenCalledTimes(1));}
 }
 fireEvent.click(screen.getByRole("button",{name:"Amazon"}));
 const start=screen.getByRole("button",{name:"新建采集任务"});await waitFor(()=>expect(start).toBeEnabled());fireEvent.click(start);
 const dialog=screen.getByRole("dialog",{name:"新建 Amazon 采集任务"});
 const input=within(dialog).getByLabelText("ASIN 或当前站点 HTTPS 商品链接（每行一个）");fireEvent.change(input,{target:{value:"B0CGRQYCPT"}});
 await act(async()=>{finish({outcome:"published",operationId,productKey:"crawler:1688:123"})});
 expect(state.push).not.toHaveBeenCalled();expect(input).toHaveValue("B0CGRQYCPT");expect(dialog).toBeVisible();
 fireEvent.click(within(dialog).getByRole("button",{name:"关闭"}));fireEvent.click(screen.getByRole("button",{name:"1688"}));
 expect(state.push).not.toHaveBeenCalled();
 expect(screen.getByRole("link",{name:"查看采集商品"})).toHaveAttribute("href",`/workbench/supply/acquisition/operation/${operationId}?agent_kind=content`);
 expect(fetcher.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});
