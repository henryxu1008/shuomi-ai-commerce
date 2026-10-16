import {collectionID} from "./product-collection";
import {completionDraftNavigation,completionRevision} from "./completion-draft-navigation";
export function sheinImportHref(sourceId:string,preparationId:string,itemId?:string,revision?:number,recordId?:string){
 const query=new URLSearchParams({preparation:collectionID.parse(preparationId)});if(itemId)query.set("item",collectionID.parse(itemId));if(recordId)query.set("target",collectionID.parse(recordId));if(revision!==undefined){if(completionRevision(String(revision))===null)throw new Error("Invalid candidate revision");query.set("revision",String(revision))}
 return `/workbench/supply/mine/missing/${collectionID.parse(sourceId)}/shein?${query}`;
}
export function sheinImportNavigation(query:Record<string,string|string[]|undefined>){
 const {target,revision,...rest}=query,navigation=completionDraftNavigation(rest);if(!navigation||target!==undefined&&(typeof target!=="string"||!collectionID.safeParse(target).success)||revision!==undefined&&completionRevision(revision)===null)return null;
 return {...navigation,recordId:target as string|undefined,revision:revision===undefined?undefined:completionRevision(revision)!};
}
