import {act,cleanup,renderHook} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {MarketAPIError} from "@/lib/api/supply-market";
import {freezeBatch,finalCommand,emptySupply,selectedBatchSchema,type SelectedCommand} from "./selected-batch-intent";
import {useSelectedBatch} from "./use-selected-batch";
import {useMarketCommands} from "./shared";

const state=vi.hoisted(()=>({write:vi.fn(),resolve:vi.fn(),invalidate:vi.fn(),context:{user:{id:"u"},effectiveOrganization:{id:"o"},isLoading:false,isSwitching:false,error:null,blockingError:null,registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<object>(),writeMarket:state.write,resolveMarket:state.resolve}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@tanstack/react-query",async original=>({...await original<object>(),useQueryClient:()=>({invalidateQueries:state.invalidate})}));
const scope={userId:"u",organizationId:"o"},storageKey='resource.pending:["u","o","supply-market","member"]';
const command=():SelectedCommand=>({action:"submit_selected",selection:{itemId:crypto.randomUUID(),expectedRevision:1,originalPublicationId:"publication",originalVersion:"1",effectiveVersion:"2",applyId:crypto.randomUUID()},supply:{...emptySupply(),stock:10,province:"浙江省",city:"杭州市"},fileIds:[crypto.randomUUID()],disclosure:true});
const inputs=(count=4)=>Array.from({length:count},(_,index)=>({title:"商品"+(index+1),command:command()}));
const receipt=(key=crypto.randomUUID())=>({operationId:key,recordId:crypto.randomUUID(),revision:1,replayed:false});
let locks:ReturnType<typeof vi.fn>;
beforeEach(()=>{
  localStorage.clear();vi.clearAllMocks();state.context={...state.context,user:{id:"u"},effectiveOrganization:{id:"o"},isSwitching:false};
  const held=new Set<string>();locks=vi.fn(async(name:string,_options:unknown,fn:(lock:unknown)=>Promise<unknown>)=>{if(held.has(name))return fn(null);held.add(name);try{return await fn({name})}finally{held.delete(name)}});
  Object.defineProperty(navigator,"locks",{value:{request:locks},configurable:true});
});
afterEach(()=>{cleanup();vi.restoreAllMocks()});

it("keeps mixed outcomes, refreshes with zero POST, then continues exact keys and payloads while skipping success",async()=>{
  const items=inputs(),first=receipt();
  state.write.mockResolvedValueOnce(first).mockRejectedValueOnce(new MarketAPIError("INVALID_REQUEST",400)).mockRejectedValueOnce(new MarketAPIError("OUTCOME_UNKNOWN",502));
  let h=renderHook(()=>useSelectedBatch(scope));
  await act(async()=>{await h.result.current.start(items)});
  expect(state.write).toHaveBeenCalledTimes(3);
  expect(h.result.current.results.map(r=>r.state)).toEqual(["submitted","failed","unknown","waiting"]);
  const original=JSON.parse(localStorage.getItem(storageKey)!);
  expect(original.items).toHaveLength(4);h.unmount();state.write.mockClear();
  state.resolve.mockImplementation(async(_scope,intent)=>{if(intent.key===original.items[0].intent.key)return first;throw new MarketAPIError("NOT_FOUND",404)});
  h=renderHook(()=>useSelectedBatch(scope));expect(state.write).not.toHaveBeenCalled();
  await act(async()=>{await h.result.current.verify()});
  expect(state.resolve).toHaveBeenCalledTimes(4);expect(state.write).not.toHaveBeenCalled();expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(original);
  state.write.mockReset().mockRejectedValueOnce(new MarketAPIError("INVALID_REQUEST",400)).mockResolvedValueOnce(receipt()).mockResolvedValueOnce(receipt());
  await act(async()=>{await h.result.current.resume()});
  expect(state.write.mock.calls.map(call=>call[1])).toEqual(original.items.slice(1).map((item:{intent:unknown})=>item.intent));
  expect(h.result.current.results.map(r=>r.state)).toEqual(["submitted","failed","submitted","submitted"]);
  expect(localStorage.getItem(storageKey)).toBeNull();
  expect(locks).toHaveBeenCalledWith(storageKey,{ifAvailable:true},expect.any(Function));
});

it.each(["POST","GET"])("does not clear or advance a %s with missing recordId or revision",async method=>{
  const items=inputs(2),batch=freezeBatch(items);
  if(method==="GET")localStorage.setItem(storageKey,JSON.stringify(batch));
  state.write.mockResolvedValue({operationId:crypto.randomUUID(),revision:0,replayed:false});state.resolve.mockResolvedValue({operationId:crypto.randomUUID(),revision:0,replayed:true});
  const h=renderHook(()=>useSelectedBatch(scope));
  await act(async()=>{if(method==="GET")await h.result.current.verify();else await h.result.current.start(items)});
  expect((method==="GET"?state.resolve:state.write)).toHaveBeenCalledTimes(1);
  expect(h.result.current.results.map(r=>r.state)).toEqual(["unknown","waiting"]);expect(localStorage.getItem(storageKey)).not.toBeNull();
});
it.each(["recordId","operationId"] as const)("rejects duplicate %s receipts instead of assigning them to a second product",async field=>{
  const a=receipt(),b=receipt();b[field]=a[field];state.write.mockResolvedValueOnce(a).mockResolvedValueOnce(b);
  const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.start(inputs(3))});
  expect(state.write).toHaveBeenCalledTimes(2);expect(h.result.current.results.map(r=>r.state)).toEqual(["submitted","unknown","waiting"]);expect(localStorage.getItem(storageKey)).not.toBeNull();
});
it.each(["PERMISSION_DENIED","IDENTITY_CONTEXT_CHANGED","ORGANIZATION_CONTEXT_CHANGED"])("stops after %s and keeps the immutable journal",async code=>{
  state.write.mockRejectedValue(new MarketAPIError(code,403));const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.start(inputs())});
  expect(state.write).toHaveBeenCalledTimes(1);expect(localStorage.getItem(storageKey)).not.toBeNull();
});
it("stops after the live enterprise changes during a response",async()=>{
  let finish!:(value:ReturnType<typeof receipt>)=>void;state.write.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));
  const h=renderHook(()=>useSelectedBatch(scope));let operation!:Promise<void>;
  await act(async()=>{operation=h.result.current.start(inputs())});
  state.context={...state.context,effectiveOrganization:{id:"other"}};h.rerender();
  await act(async()=>{finish(receipt());await operation});
  expect(state.write).toHaveBeenCalledTimes(1);expect(localStorage.getItem(storageKey)).not.toBeNull();expect(h.result.current.results[0]?.state).toBe("unknown");
});
it("never clears a replacement journal from an old response closure",async()=>{
  const replacement=freezeBatch(inputs(1));state.write.mockImplementation(async()=>{localStorage.setItem(storageKey,JSON.stringify(replacement));return receipt()});
  const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.start(inputs())});
  expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(replacement);expect(state.write).toHaveBeenCalledTimes(1);
});
it("keeps the original journal after a receipt query error and makes no POST",async()=>{
  const batch=freezeBatch(inputs(2)),encoded=JSON.stringify(batch);localStorage.setItem(storageKey,encoded);
  state.resolve.mockRejectedValue(new MarketAPIError("DEPENDENCY_UNAVAILABLE",503));
  const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.verify()});
  expect(state.write).not.toHaveBeenCalled();expect(localStorage.getItem(storageKey)).toBe(encoded);expect(state.resolve).toHaveBeenCalledTimes(1);
});
it("never re-posts an observed success when a later receipt query unexpectedly returns 404",async()=>{
  state.write.mockResolvedValueOnce(receipt()).mockRejectedValueOnce(new MarketAPIError("OUTCOME_UNKNOWN",502));
  const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.start(inputs(2))});
  expect(h.result.current.results[0]?.state).toBe("submitted");state.write.mockClear();state.resolve.mockRejectedValue(new MarketAPIError("NOT_FOUND",404));
  await act(async()=>{await h.result.current.resume()});
  expect(state.write).not.toHaveBeenCalled();expect(localStorage.getItem(storageKey)).not.toBeNull();
});
it("an old single verification closure cannot consume a later batch intent",async()=>{
  const intent={key:crypto.randomUUID(),admin:false,command:{action:"select_release",id:crypto.randomUUID(),expectedRevision:1}};
  localStorage.setItem(storageKey,JSON.stringify(intent));const h=renderHook(()=>useMarketCommands(scope)),oldVerify=h.result.current.verify;
  const replacement=freezeBatch(inputs(1));localStorage.setItem(storageKey,JSON.stringify(replacement));
  await act(async()=>{await oldVerify()});expect(state.resolve).not.toHaveBeenCalled();expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(replacement);
});
it.each(["corrupt","unwritable","no-lock","held-lock"])("fails closed for %s storage or lock",async fault=>{
  if(fault==="corrupt")localStorage.setItem(storageKey,"corrupt");
  if(fault==="unwritable")vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("denied")});
  if(fault==="no-lock")Object.defineProperty(navigator,"locks",{value:undefined,configurable:true});
  if(fault==="held-lock")locks.mockImplementation(async(_name,_options,fn)=>fn(null));
  const h=renderHook(()=>useSelectedBatch(scope));await act(async()=>{await h.result.current.start(inputs(1))});expect(state.write).not.toHaveBeenCalled();
});
it("all member command consumers share the batch slot and cannot overwrite or resolve it as a single command",async()=>{
  const batch=freezeBatch(inputs(1));localStorage.setItem(storageKey,JSON.stringify(batch));
  const h=renderHook(()=>useMarketCommands(scope));expect(h.result.current.locked).toBe(true);
  await act(async()=>{await h.result.current.verify();await h.result.current.execute({action:"select_release",id:crypto.randomUUID(),expectedRevision:1})});
  expect(state.resolve).not.toHaveBeenCalled();expect(state.write).not.toHaveBeenCalled();expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(batch);
});
it("restored single operations use the same lock and remain usable on the application page",async()=>{
  const intent={key:crypto.randomUUID(),admin:false,command:{action:"select_release",id:crypto.randomUUID(),expectedRevision:1}};
  localStorage.setItem(storageKey,JSON.stringify(intent));state.resolve.mockResolvedValue(receipt());
  const h=renderHook(()=>({single:useMarketCommands(scope),batch:useSelectedBatch(scope)}));
  await act(async()=>{await h.result.current.batch.start(inputs(1));await h.result.current.single.verify()});
  expect(state.write).not.toHaveBeenCalled();expect(state.resolve).toHaveBeenCalledTimes(1);expect(localStorage.getItem(storageKey)).toBeNull();expect(locks).toHaveBeenCalledWith(storageKey,{ifAvailable:true},expect.any(Function));
});
it("preserves per-item values while defaults and shared attachment references change",()=>{
  const a=command(),b=command(),shared=crypto.randomUUID();
  const draft={title:"A",selection:a.selection,supply:a.supply,inheritLocation:true,inheritAfterSale:true,useCommonFiles:true,fileIds:[shared,...a.fileIds]};
  const common={province:"广东省",city:"广州市",afterSaleNote:"公共售后"};
  const first=finalCommand(draft,common,[shared]);
  const second=finalCommand({...draft,selection:b.selection,supply:{...b.supply,stock:37,priceNote:"每箱报价"},inheritLocation:false,inheritAfterSale:false},common,[shared]);
  expect(first.supply).toMatchObject({...common,stock:10});expect(first.fileIds).toEqual([shared,...a.fileIds]);
  expect(second.supply).toMatchObject({stock:37,priceNote:"每箱报价",province:"浙江省",city:"杭州市",afterSaleNote:""});
});
it("bounds and strictly validates journal identities and each final command",()=>{
  expect(()=>freezeBatch(inputs(21))).toThrow();const batch=freezeBatch(inputs(2));
  batch.items[1]!.intent.key=batch.items[0]!.intent.key;expect(selectedBatchSchema.safeParse(batch).success).toBe(false);
  const duplicate=freezeBatch(inputs(2));duplicate.items[1]!.intent.command=duplicate.items[0]!.intent.command;expect(selectedBatchSchema.safeParse(duplicate).success).toBe(false);
  const item=inputs(1);item[0]!.command.fileIds=Array.from({length:7},()=>crypto.randomUUID());expect(()=>freezeBatch(item)).toThrow();
});
