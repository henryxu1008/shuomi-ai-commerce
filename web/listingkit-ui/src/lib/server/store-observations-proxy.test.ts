import { afterEach, expect, it, vi } from "vitest";
import { proxyStoreObservations } from "./store-observations-proxy";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
const id = "d6f6ca0a-27e2-4c4a-b1aa-4505110ae635";
const headers = {
  "X-Expected-Organization-ID": "org-a",
  "X-Expected-User-ID": "actor-a",
  cookie: WORKBENCH_COOKIE_NAME + "=org-a",
  Origin: "http://localhost:3000",
};

it.each([undefined,null,1,2,99])("keeps order sales area %s through the authorized BFF",async salesArea=>{
 configure();
 const record={storeId:id,syncId:id,id:"order-a",observedAt:"2026-10-09T01:00:00Z",order:{id:"order-a",site:"shein-us",status:2,stockMode:null,salesArea,type:null,tag:null,reasons:[],items:[],packages:[],amount:null,supplyCost:null,createdAt:"",updatedAt:"",issuedAt:"",needDeliveryAt:"",handoverAt:"",expectedCollectAt:""}};
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:record})));
 const response=await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/orders/stores/${id}/syncs/${id}/records/order-a`,{headers}),"private","actor-a");
 expect(response.status).toBe(200);expect((await response.json()).data.order.salesArea).toBe(salesArea);
});
const walmartRecord = {storeId:id,syncId:id,id:"SKU-1",observedAt:"2026-10-18T00:00:00Z",walmartProduct:{sku:"SKU-1",productName:"Saved Walmart product"}};
it("reads native Walmart products through filtered, mixed list and detail endpoints",async()=>{
 configure();
 const temuRecord={storeId:id,syncId:id,id:"7",observedAt:walmartRecord.observedAt,temuProduct:{id:"7",productId:"7",entity:"local_goods",mode:"local",title:"Native TEMU",externalCode:"",imageUrl:"",saleState:"unknown",rawStatus:"",skus:[],sites:[]}};
 const list={items:[walmartRecord,temuRecord],next:"",summary:{total:2,active:0,offShelf:0,today:0,todayUnknown:0,pending:0,transit:0,exceptional:0,unknown:2},syncs:[],latest:[],complete:true};
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 for(const path of ["products?platform=walmart","products"]){
  fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:list}));
  const response=await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/${path}`,{headers}),"private","actor-a");
  expect(response.status).toBe(200);expect((await response.json()).data.items[0].walmartProduct.sku).toBe("SKU-1");
 }
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:walmartRecord}));
 expect((await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/products/stores/${id}/syncs/${id}/records/SKU-1`,{headers}),"private","actor-a")).status).toBe(200);
 fetch.mockClear();
 expect((await proxyStoreObservations(new Request("http://localhost:3000/api/workbench/store-observations/orders?platform=walmart",{headers}),"private","actor-a")).status).toBe(400);
 expect(fetch).not.toHaveBeenCalled();
 for(const bad of [{...walmartRecord,id:"OTHER"},{...walmartRecord,temuProduct:temuRecord.temuProduct}]){
  fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:bad}));
  expect((await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/products/stores/${id}/syncs/${id}/records/SKU-1`,{headers}),"private","actor-a")).status).toBe(502);
 }
});
function configure() {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1");
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
}

it("binds supply observations to the exact store and rejects date-based or generic supply reads",async()=>{
 configure();const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 const request=(path:string)=>proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/${path}`,{headers}),"private","actor-a");
 for(const path of ["supply-orders",`supply-orders/observation?storeId=${id}&startDate=2026-11-01`,`supply-orders/stores/${id}/syncs/${id}/records/A/packages/x/track`])expect((await request(path)).status).toBe(400);
 expect(fetch).not.toHaveBeenCalled();
 const payload={storeId:id,status:"not_synced",canSync:false,savedOrderIds:[]};
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:payload}));expect((await request(`supply-orders/observation?storeId=${id}`)).status).toBe(200);
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:{...payload,storeId:"c0d81244-c016-442a-8184-3cac18dce4ee"}}));expect((await request(`supply-orders/observation?storeId=${id}`)).status).toBe(502);
});
it("accepts supply capabilities with unknown site and rejects an invented US site",async()=>{
 configure();const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 const read=()=>proxyStoreObservations(new Request("http://localhost:3000/api/workbench/store-observations/supply-orders/capabilities",{headers}),"private","actor-a");
 const data={kind:"supply-orders",site:"unknown",platformUrl:"",runtimeReady:true,providerConfigured:true,available:true,canSync:true};
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data}));expect((await read()).status).toBe(200);
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:{...data,site:"shein-us"}}));expect((await read()).status).toBe(502);
});

it("forwards a native product platform filter and rejects unsupported order filters",async()=>{
 configure();
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:{items:[],next:"",summary:{total:0,active:0,offShelf:0,today:0,todayUnknown:0,pending:0,transit:0,exceptional:0,unknown:0},syncs:[],latest:[],complete:false}}));
 vi.stubGlobal("fetch",fetch);
 const native=await proxyStoreObservations(new Request("http://localhost:3000/api/workbench/store-observations/products?platform=temu",{headers}),"private","actor-a");
 expect(native.status).toBe(200);
 expect(fetch).toHaveBeenCalledTimes(1);
 expect(fetch.mock.calls[0][0].searchParams.get("platform")).toBe("temu");
 fetch.mockClear();
 const rejected=await proxyStoreObservations(new Request("http://localhost:3000/api/workbench/store-observations/orders?platform=temu",{headers}),"private","actor-a");
 expect(rejected.status).not.toBe(200);expect(fetch).not.toHaveBeenCalled();
});

it("forwards qualified native product snapshots while keeping native orders closed",async()=>{
 configure();const product={storeId:id,syncId:id,id:"9007199254740993",observedAt:"2026-10-12T01:00:00Z",temuProduct:{id:"9007199254740993",productId:"9007199254740993",entity:"local_goods",mode:"local",title:"Actual title",externalCode:"",imageUrl:"",saleState:"unknown",rawStatus:"",skus:[],sites:[]}};
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:product}));vi.stubGlobal("fetch",fetch);
 const native=await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/products/stores/${id}/syncs/${id}/records/9007199254740993`,{headers}),"private","actor-a");expect(native.status).toBe(200);expect((await native.json()).data.temuProduct.id).toBe("9007199254740993");
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:product}));
 const order=await proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/orders/stores/${id}/syncs/${id}/records/9007199254740993`,{headers}),"private","actor-a");expect(order.status).not.toBe(200);
});
it("keeps original scopes and only forwards readonly endpoints", async () => {
  configure();
  const fetch = vi
    .fn()
    .mockResolvedValue(
      Response.json({
        organizationId: "org-a",
        userId: "actor-a",
        data: {
          available: true, runtimeReady: true, providerConfigured: true,
          canSync: true,
          kind: "orders",
          site: "shein-us",
          platformUrl: "https://sellerhub.shein.com/",
        },
      }),
    );
  vi.stubGlobal("fetch", fetch);
  expect(
    (
      await proxyStoreObservations(
        new Request(
          "http://localhost:3000/api/workbench/store-observations/orders/capabilities",
          { headers },
        ),
        "private",
        "actor-a",
      )
    ).status,
  ).toBe(200);
  expect(fetch.mock.calls[0][0].pathname).toBe(
    "/api/v1/workbench/store-observations/orders/capabilities",
  );
  expect(
    new Headers(fetch.mock.calls[0][1].headers).get(
      "X-Requested-Organization-ID",
    ),
  ).toBe("org-a");
  fetch.mockResolvedValueOnce(
    Response.json({
      organizationId: "other",
      userId: "actor-a",
      data: {
        available: true, runtimeReady: true, providerConfigured: true,
        canSync: true,
        kind: "orders",
        site: "shein-us",
        platformUrl: "https://sellerhub.shein.com/",
      },
    }),
  );
  expect(
    (
      await proxyStoreObservations(
        new Request(
          "http://localhost:3000/api/workbench/store-observations/orders/capabilities",
          { headers },
        ),
        "private",
        "actor-a",
      )
    ).status,
  ).toBe(409);
});
it("refuses mutation, untrusted transport and stale selected scopes before dispatch", async () => {
  configure();
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  for (const [path, init, status] of [
    ["orders/ship", { method: "POST" }, 400],
    ["orders?waybillNo=foreign", {}, 400],
    ["orders?storeId=" + id + "&storeId=" + id, {}, 400],
    [
      "orders",
      { headers: { ...headers, cookie: WORKBENCH_COOKIE_NAME + "=other" } },
      409,
    ],
    [
      "orders/syncs",
      {
        method: "POST",
        headers: { ...headers, Origin: "https://foreign.test" },
      },
      403,
    ],
  ] as const) {
    expect(
      (
        await proxyStoreObservations(
          new Request(
            "http://localhost:3000/api/workbench/store-observations/" + path,
            { headers, ...init },
          ),
          "private",
          "actor-a",
        )
      ).status,
    ).toBe(status);
  }
  expect(fetch).not.toHaveBeenCalled();
});
it("rejects overposted owner and duplicate JSON rather than issuing a new operation", async () => {
  configure();
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  for (const body of [
    '{"kind":"orders","stores":[],"organizationId":"other"}',
    '{"kind":"orders","kind":"products","stores":[]}',
  ]) {
    expect(
      (
        await proxyStoreObservations(
          new Request(
            "http://localhost:3000/api/workbench/store-observations/orders/syncs",
            {
              method: "POST",
              headers: {
                ...headers,
                "Content-Type": "application/json",
                "Idempotency-Key": id,
              },
              body,
            },
          ),
          "private",
          "actor-a",
        )
      ).status,
    ).toBe(400);
  }
  expect(fetch).not.toHaveBeenCalled();
});

it("forwards only scoped single-store saved order amounts and rejects fabricated totals",async()=>{
 configure();
 const query=`?startDate=2026-10-01&endDate=2026-10-07&storeId=${id}`;
 const source={syncId:id,observedAt:"2026-10-08T00:00:00Z",range:{start:"2026-09-01T16:00:00Z",end:"2026-10-07T15:59:59Z"}};
 const data={status:"complete",capturedAt:"2026-10-08T00:00:00Z",source,matchedOrderCount:2,amounts:[{currency:"USD",amount:"0.30",orderCount:2}]};
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data}));vi.stubGlobal("fetch",fetch);
 const url="http://localhost:3000/api/workbench/store-observations/orders/saved-amount-metrics";
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(200);
 expect(fetch.mock.calls[0][0].pathname).toBe("/api/v1/workbench/store-observations/orders/saved-amount-metrics");
 fetch.mockClear();
 for(const suffix of ["",query+"&storeId="+id,query+"&actor=other",query+"&syncId="+id]){
  expect((await proxyStoreObservations(new Request(url+suffix,{headers}),"private","actor-a")).status).toBe(400);
 }
 expect(fetch).not.toHaveBeenCalled();
 fetch.mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:{status:"missing_fields",capturedAt:data.capturedAt,source,matchedOrderCount:2,missing:{amountCount:1,currencyCount:0,unknownTimeCount:0},amounts:data.amounts}}));
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(502);
 fetch.mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:{...data,matchedOrderCount:3}}));
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(502);
});

it("validates Walmart saved amounts independently including maximum line aggregates",async()=>{
 configure();const query=`?startDate=2026-10-01&endDate=2026-10-07&storeId=${id}`;
 const source={syncId:id,observedAt:"2026-10-08T00:00:00Z",range:{start:"2026-09-01T16:00:00Z",end:"2026-10-07T16:00:00Z"}};
 const data={status:"complete",capturedAt:source.observedAt,source,matchedOrderCount:250,amounts:[{currency:"USD",amount:"2499999999999999999999999.99975000",orderCount:250}]};
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data}));vi.stubGlobal("fetch",fetch);
 const url="http://localhost:3000/api/workbench/store-observations/orders/walmart-saved-amount-metrics";
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(200);
 expect(fetch.mock.calls[0][0].pathname).toBe("/api/v1/workbench/store-observations/orders/walmart-saved-amount-metrics");
 fetch.mockClear();expect((await proxyStoreObservations(new Request(url+query+"&storeId="+id,{headers}),"private","actor-a")).status).toBe(400);expect(fetch).not.toHaveBeenCalled();
});

it("validates TikTok saved amounts independently including maximum line aggregates",async()=>{
 configure();const query=`?startDate=2026-10-01&endDate=2026-10-07&storeId=${id}`;
 const source={syncId:id,observedAt:"2026-10-08T00:00:00Z",range:{start:"2026-09-01T16:00:00Z",end:"2026-10-07T16:00:00Z"}};
 const data={status:"complete",capturedAt:source.observedAt,source,matchedOrderCount:250,amounts:[{currency:"USD",amount:"24999999999999999999999.99999750",orderCount:250}]};
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data}));vi.stubGlobal("fetch",fetch);
 const url="http://localhost:3000/api/workbench/store-observations/orders/tiktok-saved-amount-metrics";
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(200);
 expect(fetch.mock.calls[0][0].pathname).toBe("/api/v1/workbench/store-observations/orders/tiktok-saved-amount-metrics");
 fetch.mockClear();expect((await proxyStoreObservations(new Request(url+query+"&storeId="+id,{headers}),"private","actor-a")).status).toBe(400);expect(fetch).not.toHaveBeenCalled();
});
it("validates Temu USD minor-unit aggregates and rejects mixed or fractional currency claims",async()=>{
 configure();const query=`?startDate=2026-10-01&endDate=2026-10-07&storeId=${id}`;
 const source={syncId:id,observedAt:"2026-10-08T00:00:00Z",range:{start:"2026-09-01T16:00:00Z",end:"2026-10-07T16:00:00Z"}};
 const data={status:"complete",capturedAt:source.observedAt,source,matchedOrderCount:250,amounts:[{currency:"USD",amount:"23058430092136939517.50",orderCount:250}]};
 const fetch=vi.fn().mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data}));vi.stubGlobal("fetch",fetch);
 const url="http://localhost:3000/api/workbench/store-observations/orders/temu-saved-amount-metrics";
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(200);
 expect(fetch.mock.calls[0][0].pathname).toBe("/api/v1/workbench/store-observations/orders/temu-saved-amount-metrics");
 for(const entry of [{currency:"CNY",amount:"1.00",orderCount:250},{currency:"USD",amount:"1.001",orderCount:250}]){
 fetch.mockResolvedValue(Response.json({organizationId:"org-a",userId:"actor-a",data:{...data,amounts:[entry]}}));
 expect((await proxyStoreObservations(new Request(url+query,{headers}),"private","actor-a")).status).toBe(502);
 }
});
