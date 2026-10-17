import {collectionID} from "@/lib/contracts/product-collection";
import type {Stage} from "@/lib/contracts/supply-chain";

export function supplyStageHref(stage:Stage,context:{preparation?:string;store?:string;operation?:string;mode?:"local"|"platform"}){
 const query=new URLSearchParams();
 for(const key of ["preparation","store","operation"] as const){
  const value=context[key];if(value&&collectionID.safeParse(value).success)query.set(key,value);
 }
 if(stage==="missing"&&context.mode==="platform")query.set("mode","platform");
 if(stage==="waiting"&&context.mode==="local")query.set("mode","local");
 return `/workbench/supply/mine/${stage}${query.size?`?${query}`:""}`;
}
