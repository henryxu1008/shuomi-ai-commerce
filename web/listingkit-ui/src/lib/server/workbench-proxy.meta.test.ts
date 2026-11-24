import {afterEach,expect,it,vi} from "vitest";
import {buildWorkbenchUpstreamRequest,WORKBENCH_COOKIE_NAME} from "@/lib/server/workbench-proxy";

afterEach(()=>vi.unstubAllEnvs());

it("exposes Meta routes only to the isolated build and the original organization",async()=>{
  const path=["stores","11111111-1111-4111-8111-11111111111a","connection","meta-catalog"];
  const makeRequest=(organizationId:string)=>new Request("http://localhost/api/workbench/"+path.join("/"),{
    headers:{cookie:WORKBENCH_COOKIE_NAME+"=org-a","X-Expected-Organization-ID":organizationId,"X-Expected-User-ID":"actor-a"},
  });
  vi.stubEnv("NEXT_PUBLIC_META_CATALOG_SIMULATION","false");
  const disabled=await buildWorkbenchUpstreamRequest(makeRequest("org-a"),path,"fixture","actor-a");
  expect((disabled as Response).status).toBe(404);
  vi.stubEnv("NEXT_PUBLIC_META_CATALOG_SIMULATION","true");
  const foreign=await buildWorkbenchUpstreamRequest(makeRequest("org-b"),path,"fixture","actor-a");
  expect((foreign as Response).status).toBe(409);
  const changedActor=await buildWorkbenchUpstreamRequest(makeRequest("org-a"),path,"fixture","actor-b");
  expect((changedActor as Response).status).toBe(409);
  const own=await buildWorkbenchUpstreamRequest(makeRequest("org-a"),path,"fixture","actor-a");
  expect(own).toMatchObject({responseContract:"meta-status"});
});
