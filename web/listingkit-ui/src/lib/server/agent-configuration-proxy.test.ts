import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { proxyAgentConfiguration } from "./agent-configuration-proxy";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
const key = "8fc227bb-b572-4138-8e2a-5f1a0be98617";
function request(path: string, method = "GET", payload?: unknown) {
  return new Request("http://localhost:3000/api/workbench/agents/" + path, {
    method,
    headers: {
      "X-Expected-User-ID": "actor",
      "X-Expected-Organization-ID": "org-a",
      cookie: WORKBENCH_COOKIE_NAME + "=org-a",
      Origin: "http://localhost:3000",
      "Content-Type": "application/json",
      "Idempotency-Key": key,
      "If-Match": '"9223372036854775806"',
    },
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });
}
beforeEach(() => {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://localhost:9000/api/v1");
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it("rejects scope confusion before forwarding", async () => {
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  const r = request("mine");
  r.headers.set("X-Expected-Organization-ID", "org-b");
  expect((await proxyAgentConfiguration(r, "token", "actor")).status).toBe(409);
  expect(fetch).not.toHaveBeenCalled();
});
it("forwards exact bigint precondition and returns a committed receipt", async () => {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://localhost:9000/api/v1");
  const fetch = vi
    .fn()
    .mockResolvedValue(
      Response.json({
        commandId: key,
        operation: "disable",
        agentId: "product.title.agent",
        beforeRevision: "9223372036854775806",
        revision: "9223372036854775807",
        noop: false,
        committedAt: "2026-09-29T00:00:00Z",
      }),
    );
  vi.stubGlobal("fetch", fetch);
  const r = await proxyAgentConfiguration(
    request("product.title.agent/disable", "POST", {}),
    "token",
    "actor",
  );
  expect(r.status).toBe(200);
  expect(new Headers(fetch.mock.calls[0][1].headers).get("If-Match")).toBe(
    '"9223372036854775806"',
  );
});
it("does not forward arbitrary settings or a cross-origin configuration write", async () => {
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  const r = request("product.title.agent/templates", "POST", {
    name: "title",
    targetPlatform: "shein",
    allowedTools: ["unsafe"],
  });
  r.headers.delete("If-Match");
  expect((await proxyAgentConfiguration(r, "token", "actor")).status).toBe(400);
  expect(fetch).not.toHaveBeenCalled();
  const other = request("product.title.agent/disable", "POST", {});
  other.headers.set("Origin", "https://other.invalid");
  expect((await proxyAgentConfiguration(other, "token", "actor")).status).toBe(
    403,
  );
  expect(fetch).not.toHaveBeenCalled();
});

it("routes full image configuration, typed templates and recent runs through the real agent BFF",async()=>{
 const templateId="45a227bb-b572-4138-8e2a-5f1a0be98617";
 const fetch=vi.fn().mockResolvedValueOnce(Response.json({commandId:key,operation:"create-template",agentId:"product.image.agent",templateId,revision:"1",version:"1",noop:false,committedAt:"2026-10-09T00:00:00Z"}));
 vi.stubGlobal("fetch",fetch);
 const image={schema:"image-config-v1",mode:"standard",shareOriginals:true,background:"white",language:"en",carousel:[{id:"main",purpose:"product_identity"}],detail:[{id:"detail",purpose:"detail_closeup"}]};
 const r=request("product.image.agent/templates","POST",{name:"Full product images",targetPlatform:"product",image});r.headers.delete("If-Match");
 expect((await proxyAgentConfiguration(r,"token","actor")).status).toBe(200);
 expect(JSON.parse(fetch.mock.calls[0][1].body).image).toEqual(image);
 fetch.mockResolvedValueOnce(Response.json({items:[{runId:key,contextId:templateId,contextKind:"acquisition",status:"awaiting_final_approval",targetPlatform:"product",createdAt:"2026-10-09T00:00:00Z"}],nextCursor:""}));
 expect((await proxyAgentConfiguration(request("product.image.agent/recent-runs?pageSize=20"),"token","actor")).status).toBe(200);
});
it("rejects title parameters in image templates and another agent's mutation receipt",async()=>{
 const fetch=vi.fn().mockResolvedValue(Response.json({commandId:key,operation:"disable",agentId:"product.title.agent",revision:"2",noop:false,committedAt:"2026-10-09T00:00:00Z"}));vi.stubGlobal("fetch",fetch);
 const r=request("product.image.agent/templates","POST",{name:"bad",targetPlatform:"product",title:{}});r.headers.delete("If-Match");
 expect((await proxyAgentConfiguration(r,"token","actor")).status).toBe(400);expect(fetch).not.toHaveBeenCalled();
 const result=await proxyAgentConfiguration(request("product.image.agent/disable","POST",{}),"token","actor");
 expect(result.status).toBeGreaterThanOrEqual(500);expect((await result.json()).code).toBe("OUTCOME_UNKNOWN");
});

it("forwards an exact execution-mode intent without inventing activation or an If-Match header",async()=>{
 const subject={kind:"private_delivery",id:key};
 const receipt={commandId:"2b9ac69f-bf25-4d28-a85a-8036ea005013",requestKey:key,operation:"set-execution-mode",subject,mode:"full_access",beforeRevision:"0",revision:"1",noop:false,committedAt:"2026-10-12T01:00:00Z",updatedAt:"2026-10-12T01:00:00Z",updatedBy:"actor"};
 const view={subject,definition:{id:"product.quality.check",version:"2.0.0"},mode:"full_access",revision:"1",policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:receipt.committedAt,updatedBy:"actor",receipt};
 const fetch=vi.fn().mockResolvedValue(Response.json(view));vi.stubGlobal("fetch",fetch);
 const path="execution-modes/private_delivery/"+key;
 const req=request(path,"PUT",{mode:"full_access",expectedRevision:"0"});req.headers.delete("If-Match");
 const response=await proxyAgentConfiguration(req,"token","actor");
 expect(response.status).toBe(200);
 expect(fetch.mock.calls[0][0]).toBe("http://localhost:9000/api/v1/workbench/agents/"+path);
 expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({mode:"full_access",expectedRevision:"0"});
 expect(new Headers(fetch.mock.calls[0][1].headers).has("If-Match")).toBe(false);
 expect(new Headers(fetch.mock.calls[0][1].headers).get("Idempotency-Key")).toBe(key);
 fetch.mockResolvedValueOnce(Response.json({...view,receipt:{...receipt,requestKey:receipt.commandId}}));
 const mismatch=request(path,"PUT",{mode:"full_access",expectedRevision:"0"});mismatch.headers.delete("If-Match");
 const rejected=await proxyAgentConfiguration(mismatch,"token","actor");
 expect(rejected.status).toBe(502);expect((await rejected.json()).code).toBe("OUTCOME_UNKNOWN");
 for(const audit of [{updatedAt:undefined,updatedBy:undefined},{updatedAt:undefined,updatedBy:"actor"},{updatedAt:receipt.updatedAt,updatedBy:"other-actor"}]){
  fetch.mockResolvedValueOnce(Response.json({...view,receipt:{...receipt,...audit}}));
  const uncertain=request(path,"PUT",{mode:"full_access",expectedRevision:"0"});uncertain.headers.delete("If-Match");
  const invalid=await proxyAgentConfiguration(uncertain,"token","actor");
  expect(invalid.status).toBe(502);expect((await invalid.json()).code).toBe("OUTCOME_UNKNOWN");
 }
});

it("keeps uncertain mode writes unknown and rejects a mismatched subject response",async()=>{
 const path="execution-modes/registered_agent/product.title.agent";
 const fetch=vi.fn().mockResolvedValue(Response.json({subject:{kind:"registered_agent",id:"product.image.agent"},definition:{id:"product.image.agent",version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:"0001-01-01T00:00:00Z",updatedBy:""}));vi.stubGlobal("fetch",fetch);
 expect((await proxyAgentConfiguration(request(path),"token","actor")).status).toBeGreaterThanOrEqual(500);
 fetch.mockRejectedValue(new Error("response lost"));
 const req=request(path,"PUT",{mode:"read_only",expectedRevision:"1"});req.headers.delete("If-Match");
 const response=await proxyAgentConfiguration(req,"token","actor");
 expect((await response.json()).code).toBe("OUTCOME_UNKNOWN");
});
