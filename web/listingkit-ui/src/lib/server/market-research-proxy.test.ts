import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {proxyMarketResearch} from "./market-research-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const id="550e8400-e29b-41d4-a716-446655440000",fetcher=vi.fn();
beforeEach(()=>{vi.stubGlobal("fetch",fetcher);fetcher.mockReset();process.env.LISTINGKIT_PUBLIC_BASE_URL="https://app.example";process.env.LISTINGKIT_SERVICE_API_BASE="http://localhost:8080/api/v1";});afterEach(()=>vi.unstubAllGlobals());
function request(body='{"action":"link","selections":[{"itemId":"'+id+'","revision":1,"publicationId":"'+id+'","version":"1"}]}'){return new Request("https://app.example/api/workbench/research/projects/"+id+"/commands",{method:"POST",headers:{Origin:"https://app.example","Sec-Fetch-Site":"same-origin",Cookie:WORKBENCH_COOKIE_NAME+"=org-a","X-Expected-User-ID":"user-a","X-Expected-Organization-ID":"org-a","Content-Type":"application/json","Idempotency-Key":id},body});}
it("preserves the original key and uses only the research route",async()=>{fetcher.mockResolvedValue(Response.json({key:id,candidateIds:[id],revisions:{[id]:2},replayed:false}));expect((await proxyMarketResearch(request(),"token","user-a")).status).toBe(200);expect(fetcher.mock.calls[0][0]).toBe("http://localhost:8080/api/v1/workbench/research/projects/"+id+"/commands");expect(fetcher.mock.calls[0][1].headers.get("Idempotency-Key")).toBe(id);});
it("rejects forged scope, duplicate keys and empty selection before dispatch",async()=>{for(const body of ['{"action":"link","selections":[]}','{"action":"link","action":"reset","selections":[]}','{"action":"link","actorId":"other","selections":[]}'])expect((await proxyMarketResearch(request(body),"token","user-a")).status).toBe(400);expect((await proxyMarketResearch(request(),"token","other")).status).toBe(409);expect(fetcher).not.toHaveBeenCalled();});
it("retains ambiguity after dispatch and rejects another key receipt",async()=>{fetcher.mockRejectedValueOnce(Error());expect(await (await proxyMarketResearch(request(),"token","user-a")).json()).toEqual({code:"OUTCOME_UNKNOWN"});fetcher.mockResolvedValue(Response.json({key:"550e8400-e29b-41d4-a716-446655440001",candidateIds:[],revisions:{},replayed:false}));expect((await proxyMarketResearch(request(),"token","user-a")).status).toBe(502);});

it("preserves a definite unsupported-source rejection instead of manufacturing unknown",async()=>{fetcher.mockResolvedValue(Response.json({code:"SOURCE_NOT_SUPPORTED"},{status:400}));const r=await proxyMarketResearch(request(),"token","user-a");expect(r.status).toBe(400);expect(await r.json()).toEqual({code:"SOURCE_NOT_SUPPORTED"});});

function supplierPreviewRequest() {
  const url = new URL(`https://app.example/api/workbench/research/projects/${id}/candidates/${id}/supplier-review`);
  url.search = new URLSearchParams({
    itemId: id,
    revision: "1",
    publicationId: `source-run:acquisition:${id}`,
    version: "1",
  }).toString();
  return new Request(url, {
    headers: {
      Cookie: `${WORKBENCH_COOKIE_NAME}=org-a`,
      "X-Expected-User-ID": "user-a",
      "X-Expected-Organization-ID": "org-a",
    },
  });
}

function supplierPreview(count: number) {
  return {
    revision: 7,
    linkId: id,
    evidenceDigest: "a".repeat(64),
    market: { asin: "B000123456", title: "Fixture Amazon", attributes: { pack: "1 piece" } },
    offer: {
      title: "Fixture 1688",
      attributes: {},
      variants: Array.from({ length: count }, (_, index) => ({
        id: `sku-${index + 1}`,
        label: `Variant ${index + 1}`,
        attributes: { pack: "2 pieces" },
      })),
    },
  };
}

it.each([1, 201, 256])("returns all %i saved SKU choices", async (count) => {
  const preview = supplierPreview(count);
  fetcher.mockResolvedValue(Response.json(preview));
  const response = await proxyMarketResearch(supplierPreviewRequest(), "token", "user-a");
  expect(response.status).toBe(200);
  expect(await response.json()).toEqual(preview);
  expect(fetcher.mock.calls[0][1].headers.get("X-Requested-Organization-ID")).toBe("org-a");
});

it("rejects an over-limit or malformed supplier preview", async () => {
  const malformed = supplierPreview(1);
  const variant = { ...malformed.offer.variants[0], extra: true };
  for (const preview of [
    supplierPreview(257),
    { ...malformed, offer: { ...malformed.offer, variants: [variant] } },
  ]) {
    fetcher.mockResolvedValueOnce(Response.json(preview));
    const response = await proxyMarketResearch(supplierPreviewRequest(), "token", "user-a");
    expect(response.status).toBe(502);
    expect(await response.json()).toEqual({ code: "DEPENDENCY_UNAVAILABLE" });
  }
});

it("rejects a supplier preview from a mismatched session before dispatch", async () => {
  const response = await proxyMarketResearch(supplierPreviewRequest(), "token", "other");
  expect(response.status).toBe(409);
  expect(fetcher).not.toHaveBeenCalled();
});
