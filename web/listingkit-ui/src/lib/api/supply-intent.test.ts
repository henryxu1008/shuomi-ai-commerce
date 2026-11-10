import { afterEach, expect, it, vi } from "vitest";
import { parseSupplyIntent, saveSupplyIntent, loadSupplyIntent } from "./supply-intent";
import {unresolvedLocalTargetFixture} from "@/test/shein-import-fixture";
const id="550e8400-e29b-41d4-a716-446655440000";
const intent={userId:"actor",organizationId:"org",key:id,route:"transfer" as const,command:{batchId:id,expectedRevision:1}};
afterEach(()=>{sessionStorage.clear();vi.restoreAllMocks()});
it("retains the same complete command and identity across a page reload",()=>{
 expect(saveSupplyIntent(intent)).toBe(true);expect(loadSupplyIntent()).toEqual(intent);
 expect(saveSupplyIntent(null)).toBe(true);expect(loadSupplyIntent()).toBeNull();
});
it("preserves an unresolved original local save without assigning purpose or replacing its key",()=>{
 const prior=unresolvedLocalTargetFixture().intent,raw=JSON.stringify(prior);
 sessionStorage.setItem("listingkit.supply.intent",raw);
 expect(loadSupplyIntent()).toEqual(prior);
 expect(sessionStorage.getItem("listingkit.supply.intent")).toBe(raw);
 expect(saveSupplyIntent({...prior,key:id})).toBe(false);
 expect(sessionStorage.getItem("listingkit.supply.intent")).toBe(raw);
 expect(saveSupplyIntent(prior)).toBe(true);
 expect(loadSupplyIntent()).toEqual(prior);
});
it("retains the pinned product and target for a saved human review action",()=>{
 const review={...intent,route:"review-decision" as const,command:{proposalId:id,sourceId:id,recordId:id,productKey:"product",baseVersion:"2",input:{action:"accept",expected_revision:"1"}}};
 expect(saveSupplyIntent(review)).toBe(true);expect(loadSupplyIntent()).toEqual(review);
});
it("rejects injected credentials, unknown routes and invalid payloads",()=>{
 for(const value of [{...intent,command:{...intent.command,credential:"private"}},{...intent,route:"publish-again"},{...intent,key:"new"}])expect(parseSupplyIntent(JSON.stringify(value))).toBeNull();
});
it("refuses dispatch persistence when the browser cannot retain the original key",()=>{
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("quota")});expect(saveSupplyIntent(intent)).toBe(false);
});
it("does not treat an unreadable original intent as permission to replace it",()=>{
 vi.spyOn(Storage.prototype,"getItem").mockImplementation(()=>{throw new Error("storage unavailable")});
 const write=vi.spyOn(Storage.prototype,"setItem");
 expect(saveSupplyIntent(intent)).toBe(false);expect(write).not.toHaveBeenCalled();
});

it("retains only the original attempt identity and SPU for positive readback",()=>{
 const resolve={...intent,route:"resolve-upload" as const,command:{recordId:id,attemptId:id,spu:"spu-a"}};expect(saveSupplyIntent(resolve)).toBe(true);expect(loadSupplyIntent()).toEqual(resolve);
 expect(parseSupplyIntent(JSON.stringify({...resolve,command:{...resolve.command,evidence:{outcome:"succeeded"}}}))).toBeNull();
});
