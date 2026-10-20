import {z} from "zod";
import {readBoundedStrictJSON} from "./strict-json-response";
export type ModelRouteScope={userId:string;organizationId:string};
export class ModelRouteError extends Error{constructor(public code:string,public status=500){super(code)}}
export async function modelRouteRequest<T>(scope:ModelRouteScope,path:string,schema:z.ZodType<T>,init:RequestInit={}):Promise<T>{
 const publish=path.endsWith("/publish"),headers=new Headers(init.headers);headers.set("X-Expected-User-ID",scope.userId);headers.set("X-Expected-Organization-ID",scope.organizationId);headers.set("Accept","application/json");
 try{
  const r=await fetch("/api/model-route-pricing"+path,{...init,headers,credentials:"same-origin",cache:"no-store",redirect:"error"});
  const raw=await readBoundedStrictJSON(r,512*1024,init.signal??undefined);
  if(!r.ok){const e=z.strictObject({code:z.string()}).safeParse(raw);throw new ModelRouteError(e.success?e.data.code:publish?"OUTCOME_UNKNOWN":"UNAVAILABLE",r.status)}
  const parsed=schema.safeParse(raw);if(r.status!==200||!parsed.success)throw new Error();return parsed.data;
 }catch(e){if(e instanceof ModelRouteError)throw e;throw new ModelRouteError(publish?"OUTCOME_UNKNOWN":"UNAVAILABLE")}
}
