import {afterEach,expect,it,vi} from "vitest";
import {listCompletionHistory,readCompletionRevision} from "./supply-chain";
import {supplyPath} from "../contracts/supply-chain";
import {completionHistoryNavigation,completionHistoryHref} from "../contracts/completion-draft-navigation";

const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
const scope={userId:"actor",organizationId:"org"};
const source={id,preparationId:other,collectionItemId:other,collectionRevision:1,source:{productKey:"own",publicationId:other,version:"1",kind:"own"}};
const record={id,draftId:other,source,revision:1,status:"pending_completion",createdAt:"2026-10-15T00:00:00Z",content:{title:"",description:"\n"}};
const page={source,draftId:other,currentRevision:2,items:[{...record,id:other,revision:2},record]};
afterEach(()=>vi.unstubAllGlobals());
it("reads saved history and exact revisions, binding source and revision",async()=>{
 const fetcher=vi.fn().mockResolvedValue(Response.json(page));vi.stubGlobal("fetch",fetcher);
 await expect(listCompletionHistory(scope,id)).resolves.toEqual(page);
 expect(fetcher.mock.calls[0][0]).toBe(`/api/workbench/supply-preparations/sources/${id}/completion-draft/history?limit=20`);
 fetcher.mockResolvedValue(Response.json(record));await expect(readCompletionRevision(scope,id,1)).resolves.toEqual(record);
 expect(fetcher.mock.calls[1][0]).toBe(`/api/workbench/supply-preparations/sources/${id}/completion-draft/history/1`);
 for(const wrong of [{...record,revision:2},{...record,source:{...source,id:other}}]){fetcher.mockResolvedValue(Response.json(wrong));await expect(readCompletionRevision(scope,id,1)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"})}
});
it("rejects mismatched history identity, order, head bounds and cursor",async()=>{
 const fetcher=vi.fn();vi.stubGlobal("fetch",fetcher);
 for(const wrong of [
  {...page,source:{...source,id:other}},
  {...page,items:[{...record,draftId:id}]},
  {...page,items:[record,{...record,revision:2}]},
  {...page,items:[{...record,revision:3}]},
  {...page,items:[{...record,source:{...source,source:{...source.source,version:"2"}}}]},
  {...page,nextBefore:2}, {...page,items:null}, {...page,items:[],nextBefore:1}, {...page,currentRevision:0},
  {...page,items:[{...record,revision:2},record]},
 ]){fetcher.mockResolvedValue(Response.json(wrong));await expect(listCompletionHistory(scope,id)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"})}
 fetcher.mockResolvedValue(Response.json(page));await expect(listCompletionHistory(scope,id,{before:2})).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
 await expect(listCompletionHistory(scope,id,{limit:1})).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
 fetcher.mockResolvedValue(Response.json({...page,currentRevision:0,items:[]}));await expect(listCompletionHistory(scope,id)).resolves.toMatchObject({items:[]});
});
it("bounds revisions and navigation while retaining the three return contexts",()=>{
 const path=["supply-preparations","sources",id,"completion-draft","history"];
 expect(supplyPath("GET",path)).toBe("completion-history");expect(supplyPath("GET",[...path,"1"])).toBe("completion-history-detail");
 expect(supplyPath("POST",path)).toBeNull();expect(supplyPath("GET",[...path,"01"])).toBeNull();
 expect(completionHistoryNavigation({preparation:other,item:other,from:"1",to:"2"},"compare")).toEqual({preparationId:other,itemId:other,from:1,to:2});
 for(const query of [{from:"2",to:"1"},{from:"1",to:"1"},{from:["1","2"],to:"3"},{from:"01",to:"3"},{from:"1",to:"9007199254740992"},{from:"1",to:"2",returnUrl:"https://other"}])expect(completionHistoryNavigation(query,"compare")).toBeNull();
 expect(completionHistoryNavigation({from:"1"},"history")).toBeNull();
 expect(completionHistoryHref(id,other,other,{from:1,to:2})).toBe(`/workbench/supply/mine/missing/${id}/history/compare?preparation=${other}&item=${other}&from=1&to=2`);
});
