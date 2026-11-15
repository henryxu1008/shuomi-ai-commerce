import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {NextRequest} from "next/server";
const mocks=vi.hoisted(()=>({collect:vi.fn()}));
vi.mock("@/auth",()=>({serverAuth:(handler:(req:unknown)=>unknown)=>(req:NextRequest)=>handler(Object.assign(req,{auth:{}}))}));
vi.mock("@/lib/server/zitadel-auth",()=>({readZitadelIdentityFromSession:()=>({userId:"actor"})}));
vi.mock("@/lib/server/zitadel-server-token",()=>({readZitadelServerAccessToken:()=>"test-token"}));
vi.mock("@/lib/server/task-center-summary",()=>({collectTaskSummary:mocks.collect}));
import {GET} from "./route";
beforeEach(()=>{mocks.collect.mockReset()});
afterEach(()=>{vi.useRealTimers()});
it("bounds the complete request even if a dependency never responds",async()=>{
 vi.useFakeTimers();mocks.collect.mockImplementation(()=>new Promise(()=>{}));
 const pending=GET(new NextRequest("https://ui.test/api/workbench/task-center-summary?scopeKind=all"));
 await vi.advanceTimersByTimeAsync(12000);
 const response=await pending;expect((await response.json()).counts).toBeNull();expect(response.headers.get("cache-control")).toContain("no-store");
 const scoped=mocks.collect.mock.calls[0][0] as Request;expect(scoped.signal.aborted).toBe(true);
});
it("propagates disconnect cancellation to every read",async()=>{
 mocks.collect.mockImplementation(()=>new Promise(()=>{}));const controller=new AbortController();
 const pending=GET(new NextRequest("https://ui.test/api/workbench/task-center-summary",{signal:controller.signal}));controller.abort();
 expect((await (await pending).json()).status).toBe("partial");
});
