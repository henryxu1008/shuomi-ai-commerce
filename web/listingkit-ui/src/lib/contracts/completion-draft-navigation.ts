import {collectionID} from "./product-collection";

export function completionDraftHref(sourceId:string,preparationId:string,itemId?:string){
 const query=new URLSearchParams({preparation:collectionID.parse(preparationId)});
 if(itemId)query.set("item",collectionID.parse(itemId));
 return `/workbench/supply/mine/missing/${collectionID.parse(sourceId)}?${query}`;
}

// These IDs carry navigation context only. The editor checks the actual source
// before prefilling; scope and authorization come from the existing APIs.
export function completionDraftNavigation(query:Record<string,string|string[]|undefined>){
 if(Object.keys(query).some(key=>key!=="preparation"&&key!=="item"))return null;
 const preparation=query.preparation,item=query.item;
 if(preparation!==undefined&&(typeof preparation!=="string"||!collectionID.safeParse(preparation).success)||item!==undefined&&(typeof item!=="string"||!collectionID.safeParse(item).success))return null;
 return {preparationId:preparation as string|undefined,itemId:item as string|undefined};
}
