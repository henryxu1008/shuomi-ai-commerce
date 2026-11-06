import {expect,it} from "vitest";
import {offlineCreate,offlineSaved,offlineEndpoint} from "./offline-shein-records";
import {diagnosticFixture} from "@/test/fixtures/shein-diagnostic";
import {parseSheinDiagnostic,savedSheinDiagnosticSchema} from "@/lib/api/shein-diagnostic";
const id="550e8400-e29b-41d4-a716-446655440001";
it("distinguishes saved exact approval from the original recomputed diagnostic",()=>{
 const original=diagnosticFixture("save_draft"),saved={...original,not_evaluated:original.not_evaluated.filter(v=>v!=="approved_asset_provenance_and_consent")};
 expect(parseSheinDiagnostic(original)).not.toBeNull();expect(parseSheinDiagnostic(saved)).toBeNull();
 expect(savedSheinDiagnosticSchema.safeParse(saved).success).toBe(true);expect(savedSheinDiagnosticSchema.safeParse(original).success).toBe(false);
 expect(savedSheinDiagnosticSchema.safeParse({...saved,not_evaluated:[]}).success).toBe(false);
});
it("keeps exact int64 input and rejects client facts or credentials",()=>{
 const input={product_key:"p",snapshot_version:"9223372036854775807",store_id:id,country:"US",language:"en",action:"save_draft"};
 expect(offlineCreate.safeParse({collectionItemId:id,input}).success).toBe(true);
 for(const mutation of [{...input,snapshot_version:1},{...input,snapshot_version:"01"},{...input,snapshot_version:"9223372036854775808"},{...input,payload:"client"}])expect(offlineCreate.safeParse({collectionItemId:id,input:mutation}).success).toBe(false);
 expect(offlineSaved.safeParse({payload:"not a saved result"}).success).toBe(false);
});
it("admits only bounded same-feature routes",()=>{
 const route=(path:string,method="GET")=>offlineEndpoint(new URL(`https://ui.test/api/offline-shein-records${path}`),method);
 expect(route("","POST")).not.toBeNull();expect(route(`/commands/${id}`)).not.toBeNull();expect(route("?limit=20&cursor=YWJj")).not.toBeNull();
 for(const bad of ["?limit=101","?limit=20&limit=20","?arbitrary=1",`/${id}?limit=20`,`/${id}/other`,`/commands/${id}/extra`,"//"])expect(route(bad)).toBeNull();
});
