import {z} from "zod";
import {researchEndpoint,researchReceipt,type researchIntent,type ResearchSelection} from "@/lib/contracts/market-research";
import {readBoundedStrictJSON} from "./strict-json-response";
import type {ProjectScope} from "./project-center";
export class ResearchError extends Error{constructor(public code:string, public readonly status?:number){super(code);}}
export async function researchRequest<T>(scope:ProjectScope,projectId:string,schema:z.ZodType<T>,signal:AbortSignal,options:{after?:string;intent?:z.infer<typeof researchIntent>;lookup?:string;supplier?:{candidate:string;selection?:ResearchSelection;before?:number}}={}):Promise<T>{
  const {intent,lookup,after,supplier}=options;
 let tail=intent?"/commands":lookup?"/commands/"+lookup:after?"?after="+after:"";
 if(supplier){
  const query=new URLSearchParams();
  if(supplier.selection)for(const [key,value] of Object.entries(supplier.selection))query.set(key,String(value));
  if(supplier.before)query.set("before",String(supplier.before));
  tail="/candidates/"+supplier.candidate+"/supplier-review"+(query.size?"?"+query:"");
 }
 const url="/api/workbench/research/projects/"+projectId+tail;
 if(!researchEndpoint(new URL(url,window.location.origin),intent?"POST":"GET"))throw new ResearchError("INVALID_REQUEST");
 const headers:Record<string,string>={Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId};if(intent){headers["Content-Type"]="application/json";headers["Idempotency-Key"]=intent.key;}
 let responseStatus:number|undefined;
 try{const response=await fetch(url,{method:intent?"POST":"GET",headers,body:intent?JSON.stringify(intent.body):undefined,signal,credentials:"same-origin",cache:"no-store",redirect:"error"});responseStatus=response.status;const payload=await readBoundedStrictJSON(response,2*1024*1024,signal);
 if(!response.ok){
   const parsedError=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/)}).safeParse(payload);
   const code=parsedError.success?parsedError.data.code:intent?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE";
   throw new ResearchError(code,response.status);
 }const parsed=schema.safeParse(payload);if(!parsed.success)throw Error();if(intent||lookup){const receipt=researchReceipt.parse(parsed.data);if(receipt.key!==(intent?.key??lookup))throw Error();}return parsed.data;
 }catch(e){if(e instanceof ResearchError)throw e;throw new ResearchError(intent?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE",responseStatus);}
}
