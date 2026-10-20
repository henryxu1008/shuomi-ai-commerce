import {collectionID} from "./product-collection";

export function completionDraftHref(sourceId:string,preparationId:string,itemId?:string){
 const query=new URLSearchParams({preparation:collectionID.parse(preparationId)});
 if(itemId)query.set("item",collectionID.parse(itemId));
 return `/workbench/supply/mine/missing/${collectionID.parse(sourceId)}?${query}`;
}

export function completionStructuredHref(sourceId:string,preparationId:string,itemId?:string){
 const [path,query]=completionDraftHref(sourceId,preparationId,itemId).split("?");return `${path}/structure?${query}`;
}

// These IDs carry navigation context only. The editor checks the actual source
// before prefilling; scope and authorization come from the existing APIs.
export function completionDraftNavigation(query:Record<string,string|string[]|undefined>){
 if(Object.keys(query).some(key=>key!=="preparation"&&key!=="item"))return null;
 const preparation=query.preparation,item=query.item;
 if(preparation!==undefined&&(typeof preparation!=="string"||!collectionID.safeParse(preparation).success)||item!==undefined&&(typeof item!=="string"||!collectionID.safeParse(item).success))return null;
 return {preparationId:preparation as string|undefined,itemId:item as string|undefined};
}

export function completionRevision(value:unknown):number|null {
 if(typeof value!=="string"||! /^[1-9][0-9]*$/.test(value))return null;
 const parsed=Number(value);return Number.isSafeInteger(parsed)?parsed:null;
}

export function completionHistoryNavigation(query:Record<string,string|string[]|undefined>,mode:"history"|"compare"){
 const {from,to,...navigation}=query;
 if(mode==="history"&&(from!==undefined||to!==undefined))return null;
 const checked=completionDraftNavigation(navigation);if(!checked)return null;
 if(mode==="history")return checked;
 const left=completionRevision(from),right=completionRevision(to);
 if(left===null||right===null||left>=right)return null;
 return {...checked,from:left,to:right};
}

export function completionHistoryHref(sourceId:string,preparationId?:string,itemId?:string,view:{revision?:number;from?:number;to?:number}={}){
 const query=new URLSearchParams();
 if(preparationId)query.set("preparation",collectionID.parse(preparationId));
 if(itemId)query.set("item",collectionID.parse(itemId));
 let suffix="";
 if(view.revision!==undefined){if(completionRevision(String(view.revision))===null)throw new Error("Invalid saved revision");suffix=`/${view.revision}`}
 if(view.from!==undefined||view.to!==undefined){if(completionRevision(String(view.from))===null||completionRevision(String(view.to))===null||view.from!>=view.to!)throw new Error("Invalid comparison");suffix="/compare";query.set("from",String(view.from));query.set("to",String(view.to))}
 return `/workbench/supply/mine/missing/${collectionID.parse(sourceId)}/history${suffix}${query.size?`?${query}`:""}`;
}

export function completionRestoreHref(sourceId:string,revision:number,preparationId?:string,itemId?:string){
 const [path,query]=completionHistoryHref(sourceId,preparationId,itemId,{revision}).split("?");
 return `${path}/restore${query?`?${query}`:""}`;
}
