import { afterEach,expect,it,vi } from "vitest";
import { supplyCommand,readSupplyCommand,readSupplyCompletionDraft,listItemCompletionDrafts,type SupplyIntent } from "./supply-chain";
import { parseSupplyIntent } from "./supply-intent";
import { completionInputSchema,completionRecordSchema,supplyPath } from "../contracts/supply-chain";
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
const intent:SupplyIntent={userId:"actor",organizationId:"org",key:id,route:"save-completion-draft",command:{sourceId:id,expectedRevision:0,content:{title:"",description:"候选正文"}}};
const record={id,draftId:other,revision:1,status:"pending_completion",createdAt:"2026-10-12T00:00:00Z",content:{title:"",description:"候选正文"},source:{id,preparationId:other,collectionItemId:id,collectionRevision:1,source:{productKey:"original",publicationId:id,version:"1",kind:"own"}}};
const reply=(value:unknown)=>Response.json(value);
afterEach(()=>vi.unstubAllGlobals());
it("accepts incomplete native text and retains the original intent without Store facts",()=>{
 expect(completionInputSchema.safeParse(intent.command).success).toBe(true);
 expect(parseSupplyIntent(JSON.stringify(intent))).toEqual(intent);
 expect(completionInputSchema.safeParse({...intent.command as object,storeId:id}).success).toBe(false);
 expect(completionInputSchema.safeParse({...intent.command as object,content:{title:"中".repeat(342),description:""}}).success).toBe(false);
 expect(completionRecordSchema.safeParse({...record,ready:true}).success).toBe(false);
});
it("saves and reads the original immutable command, then refreshes the actual head",async()=>{
 const fetcher=vi.fn().mockResolvedValue(reply({record,replayed:false}));vi.stubGlobal("fetch",fetcher);
 await expect(supplyCommand(intent)).resolves.toMatchObject({record});
 expect(fetcher.mock.calls[0][0]).toBe("/api/workbench/supply-preparations/completion-drafts");
 expect(new Headers(fetcher.mock.calls[0][1].headers).get("Idempotency-Key")).toBe(id);
 fetcher.mockResolvedValue(reply({record,replayed:true}));await expect(readSupplyCommand(intent)).resolves.toMatchObject({record});
 expect(fetcher.mock.calls[1][0]).toBe(`/api/workbench/supply-preparations/completion-drafts/commands/${id}`);
 fetcher.mockResolvedValue(reply(record));await expect(readSupplyCompletionDraft(intent,id)).resolves.toEqual(record);
});
it("keeps UNKNOWN when a valid receipt has different text, source or revision",async()=>{
 for(const changed of [{...record,revision:2},{...record,source:{...record.source,id:other}},{...record,content:{title:"unexpected",description:""}}]){
  vi.stubGlobal("fetch",vi.fn().mockResolvedValue(reply({record:changed,replayed:false})));
  await expect(supplyCommand(intent)).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
  await expect(readSupplyCommand(intent)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
 }
});
it("reads item heads across preparations and rejects mismatched discovery scope",async()=>{
 const fetcher=vi.fn().mockResolvedValue(reply({collectionItemId:id,items:[record],total:1}));vi.stubGlobal("fetch",fetcher);
 await expect(listItemCompletionDrafts(intent,id)).resolves.toMatchObject({items:[record]});expect(fetcher.mock.calls[0][0]).toBe(`/api/workbench/supply-preparations/collection-items/${id}/completion-drafts?limit=20`);
 for(const page of [{collectionItemId:other,items:[record],total:1},{collectionItemId:id,items:[{...record,source:{...record.source,collectionItemId:other}}],total:1},{collectionItemId:id,items:[{...record,source:{...record.source,source:{...record.source.source,kind:"acquisition"}}}],total:1}]){fetcher.mockResolvedValue(reply(page));await expect(listItemCompletionDrafts(intent,id)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"})}
});
it("recognizes the bounded native routes",()=>{
 expect(supplyPath("POST",["supply-preparations","completion-drafts"])).toBe("save-completion-draft");
 expect(supplyPath("GET",["supply-preparations","sources",id,"completion-draft"])).toBe("completion-draft");
 expect(supplyPath("GET",["supply-preparations",id,"completion-drafts"])).toBe("completion-drafts");
 expect(supplyPath("GET",["supply-preparations","collection-items",id,"completion-drafts"])).toBe("item-completion-drafts");
 expect(supplyPath("GET",["supply-preparations","completion-drafts","commands",id])).toBe("completion-command");
 expect(supplyPath("POST",["supply-preparations","sources",id,"completion-draft"])).toBeNull();
});
