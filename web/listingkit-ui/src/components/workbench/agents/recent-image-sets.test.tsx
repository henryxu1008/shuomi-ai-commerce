import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {RecentImageSets} from "./recent-image-sets";
import {imageSetRecentSchema} from "@/lib/contracts/product-image-set";

vi.mock("../acquisition/product-image-set-panel",()=>({ProductImageSetPanel:({onSaved}:{onSaved?:()=>void})=><button onClick={onSaved}>已核实保存结果</button>}));
vi.mock("@/lib/api/product-image-set",async original=>({...await original<typeof import("@/lib/api/product-image-set")>(),imageSetRequest:async(_scope:unknown,_action:unknown,_schema:unknown,options:{runId:string})=>({runId:options.runId,plan:{Source:{ContextKind:"supply",OperationID:"11111111-1111-4111-8111-111111111111",EffectiveVersion:"1"},Target:{Platform:"product"}}})}));
const scope={userId:"actor",organizationId:"org"};
const ids=Array.from({length:7},(_,i)=>"22222222-2222-4222-8222-"+String(i+1).padStart(12,"0"));
function item(index:number,status="blocked",released=false){
 return {runId:ids[index],contextKind:"supply",contextId:"11111111-1111-4111-8111-111111111111",status,targetPlatform:"product",createdAt:"2026-10-18T08:00:00Z",...released?{releaseSummary:{releasedPoints:1}}:{}};
}
let fetch:ReturnType<typeof vi.fn<(input:unknown,init?:RequestInit)=>Promise<Response>>>;
beforeEach(()=>{
 fetch=vi.fn();vi.stubGlobal("fetch",fetch);
});
afterEach(()=>{cleanup();vi.unstubAllGlobals()});

it("shows all four proved releases while preserving original run IDs and read-only entry points",async()=>{
 const payload={items:ids.slice(0,4).map((_,index)=>item(index,index%2?"failed":"blocked",true)),nextCursor:""};
 const original=structuredClone(payload);
 fetch.mockResolvedValue(Response.json(payload));
 render(<RecentImageSets scope={scope} nonce={0}/>);
 await waitFor(()=>expect(screen.getAllByRole("listitem")).toHaveLength(4));
 for(const [index,row] of screen.getAllByRole("listitem").entries()){
  expect(within(row).getByText("未生成，点数已释放 · 供应链商品 · 通用素材")).toBeVisible();
  expect(within(row).getByText(new RegExp(ids[index]))).toBeVisible();
  expect(within(row).getByRole("button",{name:"查看原任务"})).toBeEnabled();
 }
 expect(payload).toEqual(original);
 expect(fetch).toHaveBeenCalledTimes(1);
 expect(fetch.mock.calls[0][0]).toBe("/api/workbench/agents/product.image.agent/recent-runs?pageSize=20");
 const init=fetch.mock.calls[0][1]!;
 expect(init.method??"GET").toBe("GET");
 expect(new Headers(init.headers).get("X-Expected-User-ID")).toBe("actor");
 expect(new Headers(init.headers).get("X-Expected-Organization-ID")).toBe("org");
});

it("keeps UNKNOWN and other unproved results in their original display states",async()=>{
 fetch.mockResolvedValue(Response.json({items:[item(0),item(1,"failed"),item(2,"executing"),item(3,"awaiting_final_approval")],nextCursor:""}));
 render(<RecentImageSets scope={scope} nonce={0}/>);
 expect(await screen.findByText("已停止 · 供应链商品 · 通用素材")).toBeVisible();
 for(const label of ["失败","生成中","待人工审核"])expect(screen.getByText(label+" · 供应链商品 · 通用素材")).toBeVisible();
 expect(screen.queryByText(/未生成，点数已释放/)).toBeNull();
});

it("uses each page's released summary without carrying it into an UNKNOWN page",async()=>{
 fetch.mockImplementation(async input=>Response.json(String(input).includes("&cursor=next")?{items:[item(4)],nextCursor:""}:{items:[item(0,"blocked",true)],nextCursor:"next"}));
 render(<RecentImageSets scope={scope} nonce={0}/>);
 expect(await screen.findByText("未生成，点数已释放 · 供应链商品 · 通用素材")).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"下一页"}));
 expect(await screen.findByText("已停止 · 供应链商品 · 通用素材")).toBeVisible();
 expect(screen.queryByText(/未生成，点数已释放/)).toBeNull();
 expect(screen.getAllByRole("listitem")).toHaveLength(1);
 fireEvent.click(screen.getByRole("button",{name:"返回第一页"}));
 expect(await screen.findByText("未生成，点数已释放 · 供应链商品 · 通用素材")).toBeVisible();
 expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});

it("rejects a malformed release summary instead of presenting it as released",async()=>{
 const invalid={items:[{...item(0),releaseSummary:{releasedPoints:0}}],nextCursor:""};
 expect(imageSetRecentSchema.safeParse(invalid).success).toBe(false);
 fetch.mockResolvedValue(Response.json(invalid));
 render(<RecentImageSets scope={scope} nonce={0}/>);
 expect(await screen.findByRole("alert")).toHaveTextContent("当前能力或依赖不可用");
 expect(screen.queryByRole("listitem")).toBeNull();
 expect(screen.queryByText(/未生成，点数已释放/)).toBeNull();
});

it("refreshes the current page from the server after a verified save without changing an unproved result",async()=>{
 let saved=false;
 fetch.mockImplementation(async input=>Response.json(String(input).includes("&cursor=next")?{items:[item(4,saved?"completed":"awaiting_final_approval"),item(5)],nextCursor:""}:{items:[item(0,"awaiting_final_approval")],nextCursor:"next"}));
 render(<RecentImageSets scope={scope} nonce={0}/>);
 fireEvent.click(await screen.findByRole("button",{name:"下一页"}));
 await screen.findByText(new RegExp(ids[4]));
 fireEvent.click(within(screen.getAllByRole("listitem")[0]).getByRole("button",{name:"查看原任务"}));
 const callback=await screen.findByRole("button",{name:"已核实保存结果"});
 expect(fetch).toHaveBeenCalledTimes(2);
 expect(screen.queryByText("已保存素材 · 供应链商品 · 通用素材")).toBeNull();
 saved=true;fireEvent.click(callback);
 expect(await screen.findByText("已保存素材 · 供应链商品 · 通用素材")).toBeVisible();
 expect(screen.getByText("已停止 · 供应链商品 · 通用素材")).toBeVisible();
 expect(screen.queryByText(/未生成，点数已释放/)).toBeNull();
 expect(fetch).toHaveBeenCalledTimes(3);
 expect(fetch.mock.calls.at(-1)?.[0]).toBe("/api/workbench/agents/product.image.agent/recent-runs?pageSize=20&cursor=next");
 expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});
