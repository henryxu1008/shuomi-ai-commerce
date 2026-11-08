import { z } from "zod";
import type { CockpitScope } from "./operations-cockpit";
import { walmartOrderAmountMetrics } from "../contracts/walmart-order-amount";
import { readBoundedStrictJSON } from "./strict-json-response";
export async function readWalmartOrderAmount(scope:CockpitScope,query:string,signal:AbortSignal){
 const response=await fetch("/api/workbench/store-observations/orders/walmart-saved-amount-metrics?"+query,{
  signal,cache:"no-store",credentials:"same-origin",redirect:"error",
  headers:{Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId},
 });
 if(response.status===401||response.status===403)return {message:"当前身份无查看订单权限"};
 if(!response.ok)throw new Error("Saved order amounts unavailable");
 const envelope=z.strictObject({organizationId:z.literal(scope.organizationId),userId:z.literal(scope.userId),data:walmartOrderAmountMetrics});
 return {result:envelope.parse(await readBoundedStrictJSON(response,65536,signal)).data};
}
