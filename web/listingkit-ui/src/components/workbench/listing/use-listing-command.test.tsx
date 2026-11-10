import {act,cleanup,render,renderHook,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {useListingCommand} from "./use-listing-command";
import {ListingCommandFeedback} from "./command-feedback";
import {loadSupplyIntent,saveSupplyIntent} from "@/lib/api/supply-intent";
import {SupplyAPIError,type SupplyIntent,type SupplyScope} from "@/lib/api/supply-chain";
import {colorSplitPlanFixture,unresolvedLocalTargetFixture,importIds} from "@/test/shein-import-fixture";

const f=vi.hoisted(()=>({context:{pendingSupplyIntent:null as SupplyIntent|null,supplyIntentReady:true,setPendingSupplyIntent:vi.fn<(v:SupplyIntent|null)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},read:vi.fn(),write:vi.fn(),saved:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),supplyCommand:f.write,readSupplyCommand:f.read}));
const actor={userId:"actor",organizationId:"org"};
beforeEach(()=>{
 vi.clearAllMocks();sessionStorage.clear();
 sessionStorage.setItem("listingkit.supply.intent",JSON.stringify(unresolvedLocalTargetFixture().intent));
 f.context.pendingSupplyIntent=loadSupplyIntent();
 f.context.setPendingSupplyIntent.mockImplementation(intent=>{if(!saveSupplyIntent(intent))return false;f.context.pendingSupplyIntent=intent;return true});
 f.read.mockResolvedValue(unresolvedLocalTargetFixture().receipt);
});
afterEach(()=>{cleanup();sessionStorage.clear();vi.unstubAllGlobals()});

it("holds the original UNKNOWN through reload and blocks new writes before purpose selection",async()=>{
 const prior=unresolvedLocalTargetFixture().intent;
 const {result,rerender}=renderHook(({scope}:{scope:SupplyScope})=>useListingCommand(scope,f.saved),{initialProps:{scope:actor as SupplyScope}});
 expect(result.current.pending).toEqual(prior);expect(result.current.foreign).toBe(true);
 for(const scope of [actor,{...actor,draftKind:"official" as const,draftSourceId:importIds.source},{...actor,draftKind:"local" as const,draftSourceId:importIds.batch},{...actor,userId:"other",draftKind:"local" as const,draftSourceId:importIds.source},{...actor,organizationId:"other",draftKind:"local" as const,draftSourceId:importIds.source}]){
  rerender({scope});
  await act(async()=>{result.current.verify();result.current.retry();await result.current.execute("save-target",prior.command)});
 }
 expect(f.read).not.toHaveBeenCalled();expect(f.write).not.toHaveBeenCalled();expect(loadSupplyIntent()).toEqual(prior);
 rerender({scope:{...actor,draftKind:"local",draftSourceId:importIds.source}});
 expect(result.current.foreign).toBe(false);
 await act(async()=>{result.current.verify()});
 await waitFor(()=>expect(f.saved).toHaveBeenCalled());
 expect(f.read).toHaveBeenCalledWith({...prior,draftKind:"local"},expect.any(AbortSignal));
 expect(f.write).not.toHaveBeenCalled();expect(loadSupplyIntent()).toBeNull();
});
it("sends a source-bound color split once and recovers its original nested-source intent",async()=>{
 sessionStorage.clear();f.context.pendingSupplyIntent=null;
 const command=colorSplitPlanFixture();
 f.write.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN",503));f.read.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));
 const {result,rerender}=renderHook(({source}:{source:string})=>useListingCommand({...actor,draftKind:"official",draftSourceId:source},f.saved),{initialProps:{source:importIds.source}});
 await act(async()=>{await result.current.execute("create-color-split",command,importIds.saved)});
 expect(f.write).toHaveBeenCalledTimes(1);expect(loadSupplyIntent()).toMatchObject({key:importIds.saved,route:"create-color-split",command:{request:{sourceId:importIds.source}}});
 rerender({source:importIds.batch});await act(async()=>{result.current.verify()});expect(f.read).not.toHaveBeenCalled();
 rerender({source:importIds.source});await act(async()=>{result.current.verify()});await waitFor(()=>expect(result.current.error).toBe("NOT_FOUND"));
 expect(f.read).toHaveBeenCalledWith(loadSupplyIntent(),expect.any(AbortSignal));expect(f.write).toHaveBeenCalledTimes(1);
});

it.each(["NOT_FOUND","DEPENDENCY_UNAVAILABLE"])("keeps an unresolved original key read-only after %s",async code=>{
 const prior=unresolvedLocalTargetFixture().intent;
 f.read.mockRejectedValue(new SupplyAPIError(code,code==="NOT_FOUND"?404:503));
 const {result}=renderHook(()=>useListingCommand({...actor,draftKind:"local",draftSourceId:importIds.source},f.saved));
 await act(async()=>{result.current.verify()});
 await waitFor(()=>expect(result.current.error).toBe(code));
 expect(result.current.pending).toEqual(prior);expect(loadSupplyIntent()).toEqual(prior);
 await act(async()=>{result.current.retry();await result.current.execute("save-target",prior.command)});
 render(<ListingCommandFeedback state={result.current}/>);
 expect(screen.getByRole("button",{name:"核实原操作"})).toBeEnabled();
 expect(screen.queryByRole("button",{name:"重试原请求"})).not.toBeInTheDocument();
 expect(f.read).toHaveBeenCalledTimes(1);expect(f.write).not.toHaveBeenCalled();expect(f.saved).not.toHaveBeenCalled();
});
it("retains the original intent when the real GET returns a different saved payload",async()=>{
 const fixture=unresolvedLocalTargetFixture(),receipt=structuredClone(fixture.receipt);
 receipt.record.input.draft.product.multi_language_name_list[0].name="another payload";
 vi.stubGlobal("fetch",vi.fn().mockImplementation(()=>Promise.resolve(new Response(JSON.stringify(receipt),{headers:{"Content-Type":"application/json"}}))));
 const api=await vi.importActual<typeof import("@/lib/api/supply-chain")>("@/lib/api/supply-chain");
 f.read.mockImplementation(api.readSupplyCommand);
 const {result}=renderHook(()=>useListingCommand({...actor,draftKind:"local",draftSourceId:importIds.source},f.saved));
 await act(async()=>{result.current.verify()});
 await waitFor(()=>expect(result.current.error).toBe("DEPENDENCY_UNAVAILABLE"));
 expect(loadSupplyIntent()).toEqual(fixture.intent);expect(f.saved).not.toHaveBeenCalled();expect(f.write).not.toHaveBeenCalled();
});
it("keeps the existing explicitly scoped official retry on its original key",async()=>{
 const {localTrial:discarded,...command}=unresolvedLocalTargetFixture().intent.command;void discarded;
 const prior={...unresolvedLocalTargetFixture().intent,command,draftKind:"official" as const,draftSourceId:importIds.source};
 sessionStorage.setItem("listingkit.supply.intent",JSON.stringify(prior));f.context.pendingSupplyIntent=loadSupplyIntent();
 f.read.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));f.write.mockResolvedValue({replayed:true});
 const {result}=renderHook(()=>useListingCommand({...actor,draftKind:"official",draftSourceId:importIds.source},f.saved));
 await act(async()=>{result.current.verify()});await waitFor(()=>expect(result.current.error).toBe("NOT_FOUND"));
 await act(async()=>{result.current.retry()});await waitFor(()=>expect(f.saved).toHaveBeenCalled());
 expect(f.write).toHaveBeenCalledWith(prior,expect.any(AbortSignal));expect(loadSupplyIntent()).toBeNull();
});
