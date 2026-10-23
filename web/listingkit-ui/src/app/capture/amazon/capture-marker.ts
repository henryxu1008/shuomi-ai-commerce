import { z } from "zod";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
export type AmazonScope=Readonly<{userId:string;organizationId:string;memberId:string}>;
const identifier=z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const schema=z.strictObject({userId:identifier,organizationId:identifier,memberId:identifier,key:z.string().refine(isAcquisitionUUID),source:z.literal("amazon"),site:z.enum(["us","ca","uk","de","jp"]),sourceID:z.string().regex(/^[A-Z0-9]{10}$/),digest:z.string().regex(/^[a-f0-9]{64}$/)});
export type AmazonMarker=z.infer<typeof schema>;
const storageKey=(scope:AmazonScope,key:string)=>JSON.stringify(["amazon-browser-capture:v1",scope.userId,scope.organizationId,scope.memberId,key]);
export const sameAmazonScope=(a:AmazonScope,b:AmazonScope)=>a.userId===b.userId&&a.organizationId===b.organizationId&&a.memberId===b.memberId;
export function saveAmazonMarker(storage:Storage,value:AmazonMarker){try{const marker=schema.parse(value);const key=storageKey(marker,marker.key),encoded=JSON.stringify(marker);storage.setItem(key,encoded);return storage.getItem(key)===encoded;}catch{return false;}}
export function loadAmazonMarker(storage:Storage,scope:AmazonScope,key:string):AmazonMarker|null{try{const raw=storage.getItem(storageKey(scope,key));if(!raw||raw.length>2048)return null;const marker=schema.parse(JSON.parse(raw));return sameAmazonScope(marker,scope)&&marker.key===key?marker:null;}catch{return null;}}
