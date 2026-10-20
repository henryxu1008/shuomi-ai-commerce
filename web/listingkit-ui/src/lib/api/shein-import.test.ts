import {afterEach,expect,it,vi} from "vitest";
import {supplyCommand,readSupplyCommand,previewSheinImport,type SupplyIntent} from "./supply-chain";
import {sheinImportRequestSchema,targetInputSchema,supplyPath} from "../contracts/supply-chain";
import {sheinImportFixture,importIds} from "@/test/shein-import-fixture";
const reply=(v:unknown)=>new Response(JSON.stringify(v),{headers:{"Content-Type":"application/json"}});
afterEach(()=>vi.unstubAllGlobals());
it("admits only SHEIN import and binds the read-only preview to the full selection",async()=>{
 const f=sheinImportFixture();expect(sheinImportRequestSchema.safeParse(f.request).success).toBe(true);expect(targetInputSchema.safeParse(f.preview.input).success).toBe(true);
 expect(targetInputSchema.safeParse({...f.preview.input,tiktok:{}}).success).toBe(false);
 expect(sheinImportRequestSchema.safeParse({...f.request,selection:{...f.request.selection,returnTo:"https://bad.test"}}).success).toBe(false);
 expect(supplyPath("POST",["supply-preparations","shein-completion-preview"])).toBe("shein-completion-preview");
 const fetcher=vi.fn().mockImplementation(()=>Promise.resolve(reply(f.preview)));vi.stubGlobal("fetch",fetcher);
 await expect(previewSheinImport({userId:"actor",organizationId:"org"},f.request)).resolves.toMatchObject(f.preview);
 fetcher.mockImplementation(()=>Promise.resolve(reply({input:{...f.preview.input,sheinImport:{...f.preview.input.sheinImport,selection:{...f.request.selection,title:false,description:true}}}})));
 await expect(previewSheinImport({userId:"actor",organizationId:"org"},f.request)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
});
it("rejects same-metadata wrong-content or wrong-revision receipts on save and original-key readback",async()=>{
 const f=sheinImportFixture();const intent:SupplyIntent={userId:"actor",organizationId:"org",key:importIds.saved,route:"save-target",command:f.preview.input};
 const fetcher=vi.fn().mockImplementation(()=>Promise.resolve(reply(f.saved)));vi.stubGlobal("fetch",fetcher);
 await expect(supplyCommand(intent)).resolves.toMatchObject(f.saved);
 for(const record of [{...f.saved.record,revision:3},{...f.saved.record,input:{...f.preview.input,draft:f.base.input.draft}},{...f.saved.record,input:{...f.preview.input,sheinImport:{...f.preview.input.sheinImport,baseRecordId:importIds.saved}}}]){
 fetcher.mockImplementation(()=>Promise.resolve(reply({...f.saved,record})));await expect(supplyCommand(intent)).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});await expect(readSupplyCommand(intent)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
 }
});
