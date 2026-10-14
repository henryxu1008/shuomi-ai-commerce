import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";

import { DEFAULT_ALLOWED_HOSTS, GET } from "./route";
const state=vi.hoisted(()=>({lookup:vi.fn(),fetch:vi.fn()}));
vi.mock("node:dns/promises",()=>({lookup:state.lookup,default:{lookup:state.lookup}}));
vi.mock("@/lib/server/request-log",()=>({logRequestWarn:vi.fn(),newRequestLogId:()=>"test"}));
beforeEach(()=>{
 vi.stubEnv("NODE_ENV","production");vi.stubEnv("IMAGE_PROXY_ALLOW_PRIVATE","");vi.stubEnv("IMAGE_PROXY_ALLOWED_HOSTS","");
 state.lookup.mockReset().mockResolvedValue([{address:"203.0.113.10",family:4}]);
 state.fetch.mockReset().mockImplementation(()=>Promise.resolve(new Response(new Uint8Array([1,2,3]),{headers:{"Content-Type":"image/png"}})));
 vi.stubGlobal("fetch",state.fetch);
});
afterEach(()=>{vi.unstubAllEnvs();vi.unstubAllGlobals();});
const preview=(url:string)=>GET(new NextRequest(`https://app.example/api/image-proxy?url=${encodeURIComponent(url)}`));

describe("image proxy default allowlist", () => {
  it("includes the COS hosts used by Studio image URLs", () => {
    expect(DEFAULT_ALLOWED_HOSTS).toEqual(
      expect.arrayContaining([
        "cos-1303159911.cos.na-ashburn.myqcloud.com",
        "shuomi-1303159911.cos.ap-hongkong.myqcloud.com",
        "cbu01.alicdn.com",
      ]),
    );
  });
});

it.each(["m.media-amazon.com","images-na.ssl-images-amazon.com","images-eu.ssl-images-amazon.com","images-fe.ssl-images-amazon.com"])("previews admitted Amazon evidence from %s",async host=>{
 const response=await preview(`https://${host}/images/I/source.jpg`);
 expect(response.status).toBe(200);
 expect(response.headers.get("Content-Type")).toBe("image/png");
 expect([...new Uint8Array(await response.arrayBuffer())]).toEqual([1,2,3]);
 expect(state.fetch).toHaveBeenCalledOnce();
});
it.each(["arbitrary.example","m.media-amazon.com.evil.example","sub.m.media-amazon.com"])("rejects a host outside the admitted Amazon image set: %s",async host=>{
 expect((await preview(`https://${host}/x.jpg`)).status).toBe(400);
 expect(state.fetch).not.toHaveBeenCalled();
});
it("rejects private DNS answers and redirects away from the allowed source",async()=>{
 state.lookup.mockResolvedValueOnce([{address:"127.0.0.1",family:4}]);
 expect((await preview("https://m.media-amazon.com/x.jpg")).status).toBe(400);
 expect(state.fetch).not.toHaveBeenCalled();
 state.fetch.mockResolvedValueOnce(new Response(null,{status:302,headers:{Location:"https://arbitrary.example/x.jpg"}}));
 expect((await preview("https://m.media-amazon.com/x.jpg")).status).toBe(400);
 expect(state.fetch).toHaveBeenCalledOnce();
});

const sourceTrialBase="https://localhost:39744/source-media-assets/source-media-trial";
const sourceTrialImage=`${sourceTrialBase}/product-source/${"a".repeat(64)}/${"b".repeat(64)}`;
it("allows only the configured source-media object in production without the general private switch",async()=>{
 vi.stubEnv("IMAGE_PROXY_SOURCE_MEDIA_TRIAL_BASE",sourceTrialBase);
 state.lookup.mockResolvedValue([{address:"127.0.0.1",family:4},{address:"::1",family:6}]);
 expect((await preview(sourceTrialImage)).status).toBe(200);
 expect(state.fetch).toHaveBeenCalledOnce();
 expect(state.fetch.mock.calls[0][1]).toMatchObject({redirect:"manual"});
 expect(state.fetch.mock.calls[0][1].headers).not.toHaveProperty("Authorization");
 expect(state.fetch.mock.calls[0][1].headers).not.toHaveProperty("Cookie");
});
it.each([
 sourceTrialImage.replace("39744","39745"),sourceTrialImage.replace("localhost","127.0.0.1"),
 sourceTrialImage.replace("source-media-trial","other-bucket"),`${sourceTrialImage}?download=1`,`${sourceTrialImage}#x`,
 `${sourceTrialBase}/../source-media-trial/product-source/${"a".repeat(64)}/${"b".repeat(64)}`,
 `${sourceTrialBase}/product-source/${"a".repeat(64)}/${"b".repeat(64)}%2fextra`,
 "https://localhost:39744/api/v1/context","https://localhost:39744/source-media-assets/source-media-trial/",
])("does not admit a different private target through the source-media trial: %s",async url=>{
 vi.stubEnv("IMAGE_PROXY_SOURCE_MEDIA_TRIAL_BASE",sourceTrialBase);
 state.lookup.mockResolvedValue([{address:"127.0.0.1",family:4}]);
 expect((await preview(url)).status).toBe(400);expect(state.fetch).not.toHaveBeenCalled();
});
it("requires loopback resolution and refuses redirects or non-source file types",async()=>{
 vi.stubEnv("IMAGE_PROXY_SOURCE_MEDIA_TRIAL_BASE",sourceTrialBase);
 expect((await preview(sourceTrialImage)).status).toBe(400);expect(state.fetch).not.toHaveBeenCalled();
 state.lookup.mockResolvedValue([{address:"127.0.0.1",family:4}]);
 state.fetch.mockResolvedValueOnce(new Response(null,{status:302,headers:{Location:sourceTrialImage}}));
 expect((await preview(sourceTrialImage)).status).toBe(400);expect(state.fetch).toHaveBeenCalledOnce();
 state.fetch.mockResolvedValueOnce(new Response("<svg/>",{headers:{"Content-Type":"image/svg+xml"}}));
 expect((await preview(sourceTrialImage)).status).toBe(400);
});
it("keeps the production source-media exception disabled by default",async()=>{
 state.lookup.mockResolvedValue([{address:"127.0.0.1",family:4}]);
 expect((await preview(sourceTrialImage)).status).toBe(400);expect(state.fetch).not.toHaveBeenCalled();
});
