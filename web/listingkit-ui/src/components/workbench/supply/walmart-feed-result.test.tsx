import {act,cleanup,render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,describe,expect,it,vi} from "vitest";
import {WalmartFeedResult} from "./walmart-feed-result";
import {publicationSchema} from "@/lib/contracts/supply-chain";

const state=vi.hoisted(()=>({read:vi.fn()}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplyPublicationStatus:state.read}));
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
function publication(feedId:string,receiptId=id,sourceId=id,storeId=id){return publicationSchema.parse({sourceId,storeId,receiptId,observedAt:"2026-10-30T00:00:00Z",walmart:{feedId,correlationId:id,wireHash:"a".repeat(64),selection:{feedType:"MP_ITEM_MATCH",version:"5-test",productType:""},skus:[{sourceSKUKey:"source",sellerSKU:"seller"}]}})}
beforeEach(()=>state.read.mockReset());afterEach(cleanup);
describe.each(["organization","actor","source","store","receipt"])("Walmart result binding: %s",change=>{
 it("discards retained results immediately when the input changes",async()=>{
  const scope={userId:"actor",organizationId:"org"};
  state.read.mockResolvedValue(publication("retained-old"));
  const view=render(<WalmartFeedResult value={publication("initial")} scope={scope} sourceId={id} storeId={id}/>);
  await userEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText(/retained-old/);
  view.rerender(<WalmartFeedResult value={publication("initial",change==="receipt"?other:id,change==="source"?other:id,change==="store"?other:id)} scope={{userId:change==="actor"?"other":scope.userId,organizationId:change==="organization"?"other":scope.organizationId}} sourceId={change==="source"?other:id} storeId={change==="store"?other:id}/>);
  expect(screen.queryByText(/retained-old/)).not.toBeInTheDocument();expect(screen.getByText(/initial/)).toBeInTheDocument();
 });
 it("ignores a late response for the previous input",async()=>{
  let resolve!:(value:ReturnType<typeof publication>)=>void;
  state.read.mockReturnValue(new Promise(done=>{resolve=done}));
  const scope={userId:"actor",organizationId:"org"};
  const view=render(<WalmartFeedResult value={publication("initial")} scope={scope} sourceId={id} storeId={id}/>);
  await userEvent.click(screen.getByRole("button",{name:"查询平台结果"}));
  view.rerender(<WalmartFeedResult value={publication("initial",change==="receipt"?other:id,change==="source"?other:id,change==="store"?other:id)} scope={{userId:change==="actor"?"other":scope.userId,organizationId:change==="organization"?"other":scope.organizationId}} sourceId={change==="source"?other:id} storeId={change==="store"?other:id}/>);
  await act(async()=>resolve(publication("late-old")));
  expect(screen.queryByText(/late-old/)).not.toBeInTheDocument();expect(screen.getByText(/initial/)).toBeInTheDocument();
 });
});

it("keeps the new request active when the old request finishes late",async()=>{
 let oldResolve!:(value:ReturnType<typeof publication>)=>void,newResolve!:(value:ReturnType<typeof publication>)=>void;
 state.read.mockReturnValueOnce(new Promise(done=>{oldResolve=done})).mockReturnValueOnce(new Promise(done=>{newResolve=done}));
 const view=render(<WalmartFeedResult value={publication("initial")} scope={{userId:"actor",organizationId:"org"}} sourceId={id} storeId={id}/>);
 await userEvent.click(screen.getByRole("button",{name:"查询平台结果"}));
 view.rerender(<WalmartFeedResult value={publication("next",other)} scope={{userId:"actor",organizationId:"org"}} sourceId={id} storeId={id}/>);
 await userEvent.click(screen.getByRole("button",{name:"查询平台结果"}));
 await act(async()=>oldResolve(publication("late-old")));
 expect(screen.getByRole("button",{name:"正在查询"})).toBeDisabled();expect(state.read).toHaveBeenCalledTimes(2);
 await act(async()=>newResolve(publication("new-result",other)));
 expect(screen.getByText(/new-result/)).toBeInTheDocument();expect(screen.queryByText(/late-old/)).not.toBeInTheDocument();
});

it("shows progress and historical page times separately from the current native window",async()=>{
 const hash="a".repeat(64),old="2026-10-30T00:00:00Z",now="2026-10-30T00:01:00Z";
 const observation={feedId:"feed",feedStatus:"PROCESSED",itemsReceived:1000,itemsSucceeded:0,itemsFailed:1000,itemsProcessing:0,items:Array.from({length:50},(_,i)=>({sku:`new-${i}`,ingestionStatus:"DATA_ERROR"})),partial:true,responseHashes:[hash],observedAt:now};
 const result=publicationSchema.parse({...publication("feed"),walmartResults:[{id,receiptId:id,generation:2,latest:true,observation,readProgress:{round:1,expectedHash:hash,startOffset:50,nextOffset:100,previousWindowId:other,pages:[{offset:50,count:50,responseHash:hash,observedAt:now}]},observedPages:[{offset:0,observation:{...observation,observedAt:old,items:observation.items.map((v,i)=>({...v,sku:`old-${i}`}))}}]}]});
 state.read.mockResolvedValue(result);
 render(<WalmartFeedResult value={publication("feed")} scope={{userId:"actor",organizationId:"org"}} sourceId={id} storeId={id}/>);
 await userEvent.click(screen.getByRole("button",{name:"查询平台结果"}));
 await screen.findByText(/本轮已读取 100 条明细/);
 expect(screen.getByText("SKU：new-0")).toBeInTheDocument();expect(screen.getByText("SKU：old-0")).toBeInTheDocument();
 expect(screen.getByText(`页面取得时间：${new Date(old).toLocaleString()}`)).toBeInTheDocument();
 expect(screen.getByText(/此前窗口页面仅供核对，不能拼成完整的新结果/)).toBeInTheDocument();expect(state.read).toHaveBeenCalledOnce();
});
