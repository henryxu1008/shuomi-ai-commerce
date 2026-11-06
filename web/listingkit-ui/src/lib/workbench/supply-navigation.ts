import {collectionID} from "@/lib/contracts/product-collection";
import type {Stage} from "@/lib/contracts/supply-chain";

export const listingHome = "/workbench/supply/mine";
export const listingStages = [
 {id:"waiting", label:"待上传"},
 {id:"missing", label:"待补全"},
 {id:"review", label:"待审核"},
 {id:"uploaded", label:"已上传"},
] as const;

export type SupplyStageContext = {
 preparation?: string;
 store?: string;
 operation?: string;
 plan?: string;
 mode?: "local" | "platform";
};

export function supplyStageHref(stage:Stage,context:SupplyStageContext){
 const query=new URLSearchParams();
 for(const key of ["preparation","store","operation","plan"] as const){
  const value=context[key];if(value&&collectionID.safeParse(value).success)query.set(key,value);
 }
 if(stage==="missing"&&context.mode==="platform")query.set("mode","platform");
 if(stage==="waiting"&&context.mode==="local")query.set("mode","local");
 return `${listingHome}/${stage}${query.size?`?${query}`:""}`;
}
