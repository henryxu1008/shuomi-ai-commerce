import {afterEach,expect,it,vi} from "vitest";
import {analysisRequest,AnalysisError} from "./product-image-analysis";
import {analysisResultSchema} from "../contracts/product-image-analysis";
const id="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",scope={userId:"actor",organizationId:"B",kind:"supply" as const,contextId:id};
afterEach(()=>vi.unstubAllGlobals());
it("uses one explicit POST and preserves the original key on transport uncertainty",async()=>{
 const fetch=vi.fn().mockRejectedValue(Error("response lost"));vi.stubGlobal("fetch",fetch);
 await expect(analysisRequest(scope,"prepare",analysisResultSchema,{requestKey:key,body:{kind:"source",imageId:"img",catalogVersion:"1",publicationId:"pub",expectedModeRevision:"0"}})).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
 expect(fetch).toHaveBeenCalledTimes(1);const [url,options]=fetch.mock.calls[0];expect(url).toContain(`/sources/${id}/images/analysis/prepare`);expect(new Headers(options.headers).get("Idempotency-Key")).toBe(key);
});
it("does not turn a failed readonly history request into an uncertain new action",async()=>{
 const fetch=vi.fn().mockRejectedValue(Error("offline"));vi.stubGlobal("fetch",fetch);
 await expect(analysisRequest(scope,"read",analysisResultSchema,{requestKey:key})).rejects.toEqual(expect.objectContaining<Partial<AnalysisError>>({code:"DEPENDENCY_UNAVAILABLE"}));
 expect(fetch).toHaveBeenCalledTimes(1);expect(fetch.mock.calls[0][1].method).toBe("GET");
});
