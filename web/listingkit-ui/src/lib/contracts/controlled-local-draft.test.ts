import {expect,it} from "vitest";
import {sheinImportFixture,importIds} from "@/test/shein-import-fixture";
import {targetInputSchema,targetRecordSchema,rulesSchema} from "./supply-chain";

const marker=()=>({contractVersion:1,trialId:"issue1158-local-draft-v1",sourceId:importIds.source,storeId:importIds.store,completionDraft:{recordId:importIds.candidate,revision:1},fixtureVersion:"issue1158-local-draft-v1",fixtureHash:"a".repeat(64)});
it("preserves complete local provenance without inventing official credentials",()=>{
 const f=sheinImportFixture(),localTrial=marker();
 const merchant={...f.base.merchant,platform:"shein",connection_revision:0,application_id:"",application_revision:"",service_expires_at:"0001-01-01T00:00:00Z"};
 const input={...f.base.input,localTrial};
 expect(targetInputSchema.parse(input)).toMatchObject({localTrial});
 const record={...f.base,merchant,input,localTrial};
 expect(targetRecordSchema.parse(record)).toMatchObject({localTrial,input:{localTrial},merchant:{connection_revision:0,application_id:""}});
 expect(rulesSchema.parse({...f.rules,merchant,rules:{...f.rules.rules,authority:"local_trial"},initialInput:input})).toMatchObject({initialInput:input,rules:{authority:"local_trial"}});
 for(const changed of [{...record,localTrial:undefined},{...record,input:{...input,localTrial:undefined}},{...record,merchant:f.base.merchant},{...record,result:{...record.result,ready_for_upload:true}},{...record,localTrial:{...localTrial,storeId:importIds.saved}}])expect(targetRecordSchema.safeParse(changed).success).toBe(false);
 expect(targetInputSchema.safeParse({...input,localTrial:{...localTrial,unknown:true}}).success).toBe(false);
 expect(targetRecordSchema.safeParse({...record,result:{...record.result,submission_payload:{invalid:true}}}).success).toBe(false);
 expect(rulesSchema.safeParse({...f.rules,merchant,rules:{...f.rules.rules,authority:"official"}}).success).toBe(false);
 expect(rulesSchema.safeParse({...f.rules,merchant:f.base.merchant,rules:{...f.rules.rules,authority:"local_trial"}}).success).toBe(false);
});
