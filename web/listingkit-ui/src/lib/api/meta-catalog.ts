import {z} from "zod";
import {metaContracts,type MetaAction,type MetaResponse} from "@/lib/contracts/meta-catalog";
import {readBoundedStrictJSON} from "./strict-json-response";

export type MetaScope={userId:string;organizationId:string};

export class MetaRequestError extends Error {
  constructor(readonly code:string){super("Meta catalog request failed");}
}

export async function metaRequest<A extends MetaAction>(
  scope:MetaScope,action:A,id:string,body?:unknown,key?:string,
):Promise<MetaResponse<A>>{
  const contract=metaContracts[action];
  const listing=action==="prepare"||action==="confirm"||action==="result"||action==="options";
  const base=listing?"/api/workbench/supply-preparations/meta-catalog":
    "/api/workbench/stores/"+z.uuid().parse(id)+"/connection/meta-catalog";
  const path=action==="status"?base:base+"/"+action+(action==="options"?"/"+z.uuid().parse(id):"");
  const headers:Record<string,string>={"X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId,Accept:"application/json"};
  if(body!==undefined)headers["Content-Type"]="application/json";
  if(key)headers["Idempotency-Key"]=key;
  const response=await fetch(path,{method:contract.method,headers,credentials:"same-origin",cache:"no-store",
    body:body===undefined?undefined:JSON.stringify(body)});
  const payload=await readBoundedStrictJSON(response,256*1024);
  if(!response.ok){
    const parsed=z.object({code:z.string()}).safeParse(payload);
    throw new MetaRequestError(parsed.success?parsed.data.code:"DEPENDENCY_UNAVAILABLE");
  }
  return contract.response.parse(payload) as MetaResponse<A>;
}
