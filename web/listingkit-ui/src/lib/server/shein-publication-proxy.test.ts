import {afterEach,expect,it,vi} from "vitest";
import {buildWorkbenchUpstreamRequest,buildWorkbenchBrowserResponse,WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
afterEach(()=>vi.unstubAllEnvs());
it("binds each SHEIN route to the captured actor, enterprise, original receipt and key",async()=>{
 vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost");
 for(const action of ["status","prepare","confirm","result"]){
  const path=["supply-preparations","shein-publication",action],write=action==="prepare"||action==="confirm";
  const body={sourceId:id,storeId:other,receiptId:id,...action!=="status"?{skc:"skc"}:{},...["confirm","result"].includes(action)?{token:other}:{},...action==="result"?{key:id}:{}};
  const request=(actor="actor")=>new Request("http://localhost/api/workbench/"+path.join("/"),{method:"POST",headers:{cookie:WORKBENCH_COOKIE_NAME+"=org","X-Expected-Organization-ID":"org","X-Expected-User-ID":actor,Origin:"http://localhost","Content-Type":"application/json",...write?{"Idempotency-Key":id}:{}},body:JSON.stringify(body)});
  const accepted=await buildWorkbenchUpstreamRequest(request(),path,"server-token","actor");
  expect(accepted).not.toBeInstanceOf(Response);
  if(!(accepted instanceof Response)){expect(JSON.parse(String(accepted.init.body))).toEqual(body);expect(new Headers(accepted.init.headers).get("Authorization")).toBe("Bearer server-token");expect(accepted.expectedStoreId).toContain(id)}
  const foreign=await buildWorkbenchUpstreamRequest(request("other"),path,"server-token","actor");expect(foreign).toBeInstanceOf(Response);expect((foreign as Response).status).toBe(409);
 }
});
it("does not confirm a SHEIN result belonging to another original receipt",async()=>{
 const good={key:id,attemptId:id,sourceId:id,storeId:other,receiptId:id,skc:"skc",status:"outcome_unknown",message:"待核实"};
 const binding=JSON.stringify({sourceId:id,storeId:other,receiptId:id,skc:"skc",key:id});
 const accepted=await buildWorkbenchBrowserResponse(Response.json(good),"supply-shein-shelf-confirm",binding);expect(accepted.status).toBe(200);
 const bad=await buildWorkbenchBrowserResponse(Response.json({...good,receiptId:other}),"supply-shein-shelf-confirm",binding);expect(bad.status).toBeGreaterThanOrEqual(400);
 expect(await bad.text()).not.toContain(other);
});

