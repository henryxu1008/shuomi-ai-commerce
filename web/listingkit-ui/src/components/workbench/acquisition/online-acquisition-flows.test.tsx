import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {AcquisitionAPIError} from "@/lib/api/product-acquisition";
import {OnlineAcquisitionHub} from "./online-acquisition-hub";

const state=vi.hoisted(()=>({acquire:vi.fn(),push:vi.fn()}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:state.push}),useSearchParams:()=>new URLSearchParams()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org"},acquisitionIntentReady:true,pendingAcquisitionIntent:null,setPendingAcquisitionIntent:()=>true,registerOrganizationSwitchGuard:()=>()=>{}})}));
vi.mock("@/lib/api/product-acquisition",async original=>({...await original<object>(),acquire1688:state.acquire}));

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
