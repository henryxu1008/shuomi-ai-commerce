import { z } from "zod";
import * as c from "../contracts/acquisition-plan";
import type {AcquisitionContext} from "./product-acquisition";
import {AcquisitionAPIError} from "./product-acquisition";
import {readBoundedStrictJSON} from "./strict-json-response";
import {acquisitionErrorStatuses,isAcquisitionUUID} from "../contracts/product-acquisition";

const base="/api/workbench/sourcing/1688";
export const listAcquisitionPlans=(scope:AcquisitionContext,signal?:AbortSignal,after?:string)=>send(`${base}/plans${after&&isAcquisitionUUID(after)?`?after=${after}`:""}`,scope,c.planListSchema,signal);
export const listAcquisitionRules=(scope:AcquisitionContext,signal?:AbortSignal)=>send(`${base}/rules`,scope,c.ruleListSchema,signal);
export const readAcquisitionPlan=async(scope:AcquisitionContext,id:string,signal?:AbortSignal)=>{const p=await send(`${base}/plans/${id}`,scope,c.acquisitionPlanSchema,signal);if(p.id!==id)throw new AcquisitionAPIError("INVALID_UPSTREAM_RESPONSE",502);return p;};
export const readAcquisitionPlanByKey=(scope:AcquisitionContext,key:string,signal?:AbortSignal)=>send(`${base}/plans/by-key/${key}`,scope,c.acquisitionPlanSchema,signal);
export const createAcquisitionPlan=(scope:AcquisitionContext,key:string,input:c.CreatePlan,signal?:AbortSignal)=>send(`${base}/plans`,scope,c.acquisitionPlanSchema,signal,{body:c.createPlanSchema.parse(input),key});
export const advanceAcquisitionPlan=async(scope:AcquisitionContext,plan:c.AcquisitionPlan,signal?:AbortSignal)=>{const p=await send(`${base}/plans/${plan.id}/advance`,scope,c.acquisitionPlanSchema,signal,{body:{expectedRevision:plan.revision}});if(p.id!==plan.id)throw new AcquisitionAPIError("OUTCOME_UNKNOWN",503);return p;};
export const mutateAcquisitionRule=async(scope:AcquisitionContext,key:string,input:c.RuleMutation,signal?:AbortSignal)=>{const value=await send(`${base}/rules`,scope,c.ruleTemplateSchema,signal,{body:c.ruleMutationSchema.parse(input),key});if(input.id&&value.id!==input.id)throw new AcquisitionAPIError("OUTCOME_UNKNOWN",503);return value;};

async function send<T>(url:string,scope:AcquisitionContext,schema:z.ZodType<T>,signal?:AbortSignal,mutation?:{body:unknown;key?:string}):Promise<T>{
 const safe=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
 if(!safe.test(scope.userId)||!safe.test(scope.organizationId))throw new AcquisitionAPIError("INVALID_REQUEST",400);
 const controller=new AbortController(),abort=()=>controller.abort();signal?.addEventListener("abort",abort,{once:true});if(signal?.aborted)abort();
 const timer=setTimeout(abort,55000),unknown=()=>new AcquisitionAPIError(mutation?"OUTCOME_UNKNOWN":"ACQUISITION_UNAVAILABLE",503);
 try{
  const headers=new Headers({Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId});
  if(mutation){headers.set("Content-Type","application/json");if(mutation.key)headers.set("Idempotency-Key",mutation.key);}
  const response=await fetch(url,{method:mutation?"POST":"GET",headers,body:mutation?JSON.stringify(mutation.body):undefined,credentials:"same-origin",redirect:"error",cache:"no-store",signal:controller.signal});
  if(!/^application\/json(?:\s*;|$)/i.test(response.headers.get("content-type")??""))throw unknown();
  const raw=await readBoundedStrictJSON(response,2<<20,controller.signal);
  if(!response.ok){const envelope=z.object({code:z.string()}).passthrough().safeParse(raw);if(envelope.success&&acquisitionErrorStatuses[envelope.data.code]===response.status)throw new AcquisitionAPIError(envelope.data.code,response.status);throw unknown();}
  const parsed=schema.safeParse(raw);if(response.status!==200||!parsed.success)throw unknown();return parsed.data;
 }catch(e){if(e instanceof AcquisitionAPIError)throw e;throw unknown();}
 finally{clearTimeout(timer);signal?.removeEventListener("abort",abort);}
}
