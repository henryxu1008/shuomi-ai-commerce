import {afterEach,expect,it,vi} from "vitest";
import {listMarket} from "./supply-market";
import {marketFiltersSchema} from "../contracts/supply-market";
afterEach(()=>vi.unstubAllGlobals());
it("sends small price bounds as decimal query values without losing precision",async()=>{
 const fetcher=vi.fn<(url:string,options:RequestInit)=>Promise<Response>>(async()=>Response.json({items:[],total:0}));vi.stubGlobal("fetch",fetcher);
 await listMarket({userId:"actor",organizationId:"org"},{kind:"official",minPrice:0.0000001,maxPrice:0.0000002,priceCurrency:"CNY"});
 const url=new URL(fetcher.mock.calls[0]![0],"https://console.example");
 expect(url.searchParams.get("minPrice")).toBe("0.0000001");expect(url.searchParams.get("maxPrice")).toBe("0.0000002");
});
it("rejects prices too small for the bounded decimal query instead of rounding to zero",()=>{
 expect(marketFiltersSchema.safeParse({minPrice:1e-100,priceCurrency:"CNY"}).success).toBe(false);
});
