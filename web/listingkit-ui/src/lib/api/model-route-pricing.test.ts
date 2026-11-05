import {afterEach,expect,it,vi} from "vitest";
import {z} from "zod";
import {modelRouteRequest} from "./model-route-pricing";

afterEach(()=>vi.unstubAllGlobals());
it.each(["publish","commands"])("does not classify a receipt key named %s as a mutation",async key=>{
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({unexpected:true})));
 await expect(modelRouteRequest({userId:"manager",organizationId:"org-a"},"/connections/shared/receipts/"+key,z.strictObject({receipt:z.string()}))).rejects.toMatchObject({code:"UNAVAILABLE"});
});
it("keeps a malformed reply to an actual Key command unknown without retry",async()=>{
 const fetch=vi.fn().mockResolvedValue(Response.json({unexpected:true}));vi.stubGlobal("fetch",fetch);
 await expect(modelRouteRequest({userId:"manager",organizationId:"org-a"},"/connections/shared/commands",z.strictObject({receipt:z.string()}),{method:"POST"})).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
 expect(fetch).toHaveBeenCalledTimes(1);
});
