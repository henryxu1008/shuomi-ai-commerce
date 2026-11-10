import {expect,it} from "vitest";
import {supplyPath} from "./supply-chain";
import {parseSupplyIntent} from "../api/supply-intent";
import {sheinImportNavigation,sheinImportHref} from "./shein-import-navigation";
import {sheinImportFixture,importIds} from "@/test/shein-import-fixture";
it("admits only the narrow local namespace and preserves explicit navigation purpose",()=>{
 expect(supplyPath("GET",["local-platform-drafts","context"])).toBe("local-context");
 expect(supplyPath("GET",["local-platform-drafts","records",importIds.saved])).toBe("record");
 for(const path of ["operations","images","uploads","shein-publication","shein-color-splits"])expect(supplyPath("POST",["local-platform-drafts",path])).toBeNull();
 expect(sheinImportNavigation({preparation:importIds.batch,draftKind:"local"})).toMatchObject({draftKind:"local"});
 expect(sheinImportNavigation({preparation:importIds.batch,draftKind:"other"})).toBeNull();
 expect(sheinImportHref(importIds.source,importIds.batch,undefined,1,importIds.saved,"local")).toContain("draftKind=local");
});
it("does not recover a test save through the official namespace or change purpose on retry",()=>{
 const f=sheinImportFixture();
 const localTrial={contractVersion:1,trialId:"issue1158-local-draft-v1",sourceId:importIds.source,storeId:importIds.store,completionDraft:{recordId:importIds.candidate,revision:1},fixtureVersion:"issue1158-local-draft-v1",fixtureHash:"a".repeat(64)};
 const intent={userId:"user",organizationId:"org",key:importIds.saved,route:"save-target",command:{...f.base.input,localTrial}};
 expect(parseSupplyIntent(JSON.stringify(intent))).toBeNull();
 expect(parseSupplyIntent(JSON.stringify({...intent,draftKind:"official"}))).toBeNull();
 expect(parseSupplyIntent(JSON.stringify({...intent,draftKind:"local"}))).toMatchObject({draftKind:"local",command:{localTrial}});
 expect(parseSupplyIntent(JSON.stringify({...intent,draftKind:"local",command:f.base.input}))).toBeNull();
});

