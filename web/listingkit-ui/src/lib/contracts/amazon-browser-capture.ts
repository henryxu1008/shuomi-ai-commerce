import { z } from "zod";
import { collectionDetailSchema } from "./product-collection";
import { isAcquisitionUUID } from "./product-acquisition";
import { parseTree, getNodeValue, type Node, type ParseError } from "jsonc-parser";
import { amazonCaptureBody } from "./amazon-capture-body";

export const AMAZON_CAPTURE_BASE="/api/workbench/sourcing/amazon/browser-captures";
export const AMAZON_CAPTURE_MAX_BYTES=2*1024*1024;
export const AMAZON_CAPTURE_RESPONSE_MAX_BYTES=AMAZON_CAPTURE_MAX_BYTES+8192;
const text=z.string().refine(v=>new TextEncoder().encode(v).length<=8192 && v.isWellFormed());
const attr=z.strictObject({name:text,value:text});
const offer=z.strictObject({amount:text,currency:text.nullable(),minQuantity:text.nullable().optional(),condition:z.enum(['new','used']).optional(),conditionNote:text.refine(v=>!!v.trim()&&new TextEncoder().encode(v).length<=256).optional()});
const price=offer.extend({alternative:offer.optional()}).refine(p=>(!p.conditionNote||p.condition==='used')&&(!p.alternative||p.condition==='new'&&p.alternative.condition==='used')&&(!p.alternative?.conditionNote||p.alternative.condition==='used'));
const images=z.array(z.strictObject({url:text,role:z.enum(["main","gallery","detail","variant"])})).max(256);
const variant=z.strictObject({sourceID:text.nullable(),sku:text.nullable(),title:text.nullable(),attributes:z.array(attr).max(256),price:price.nullable(),images});
export const amazonCaptureSchema=z.strictObject({captureVersion:z.literal(2),source:z.literal("amazon"),evidence:z.strictObject({schemaVersion:z.literal(2),site:z.enum(["us","ca","uk","de","jp"]),sourceID:text.regex(/^[A-Z0-9]{10}$/),sourceURL:text,title:text.nullable(),description:text.nullable(),attributes:z.array(attr).max(256),variants:z.array(variant).max(256),priceFacts:z.array(price).max(256),images,observation:z.strictObject({scope:z.enum(["selected_product","partial_family"]),parentASIN:text.nullable()}),capturedAt:text,contentSHA256:text.regex(/^[a-f0-9]{64}$/),parserVersion:z.literal("amazon-browser-dom/v1"),warnings:z.array(z.strictObject({code:text,field:text})).max(256),missingFacts:z.array(z.strictObject({field:text,reason:text})).max(256)})}).refine(({evidence:e})=>e.attributes.length+e.variants.length+e.variants.reduce((n,v)=>n+v.attributes.length+v.images.length+(v.price?.alternative?1:0),0)+e.priceFacts.length+e.priceFacts.filter(p=>p.alternative).length+e.images.length+e.warnings.length+e.missingFacts.length<=1024);
export type AmazonCapturePayload=z.infer<typeof amazonCaptureSchema>;
export async function freezeAmazonCapture(input:unknown){
 const value=amazonCaptureSchema.parse(input);
 const body=amazonCaptureBody(value);
 const hash=await crypto.subtle.digest("SHA-256",new TextEncoder().encode(body));
 return {payload:value,body,digest:Array.from(new Uint8Array(hash),b=>b.toString(16).padStart(2,"0")).join("")};
}
const id=z.string().refine(isAcquisitionUUID);
const sourceRecord=z.strictObject({type:text.optional(),detail:text.optional(),platform:text.optional(),source_id:text.optional(),source_version:text.optional(),reference_type:text.optional(),url:text.optional(),snapshot_id:text.optional(),checksum:text.optional(),captured_at:text.optional(),metadata:z.record(text,text).optional(),source_run_id:text.optional(),request_id:text.optional(),notes:z.array(text).max(256).optional()});
const trace=z.strictObject({sources:z.array(sourceRecord).max(256).optional(),confidence:z.number().optional(),is_inferred:z.boolean().optional(),needs_review:z.boolean().optional()});
const catalogAttr=z.strictObject({name:text.optional(),value:text.optional(),trace:trace.optional()});
const productSchema=collectionDetailSchema.shape.product;
const catalogImages=z.array(productSchema.shape.images.unwrap().element.extend({trace:trace.optional()}).strict()).max(4096);
const snapshot=productSchema.extend({attributes:z.array(catalogAttr).max(4096).optional(),images:catalogImages.optional(),variants:z.array(productSchema.shape.variants.unwrap().element.extend({attributes:z.array(catalogAttr).max(4096).optional(),images:catalogImages.optional(),trace:trace.optional()}).strict()).max(4096).optional(),sources:z.array(sourceRecord).max(4096).optional()}).strict();
export const amazonCaptureResultSchema=z.strictObject({state:z.literal("saved"),captureID:id,publicationID:id,key:id,payloadSHA256:z.string().regex(/^[a-f0-9]{64}$/),productKey:z.string().regex(/^amazon-[0-9a-f-]{36}$/),catalogVersion:z.string().regex(/^[1-9][0-9]{0,18}$/),batchID:id,currentBatchID:id,itemID:id,archived:z.boolean(),replayed:z.boolean(),snapshot}).refine(v=>v.captureID===v.publicationID);
export type AmazonCaptureResult=z.infer<typeof amazonCaptureResultSchema>;
export const amazonCaptureErrors:Record<string,number>={INVALID_CAPTURE:400,CAPTURE_FORBIDDEN:403,CAPTURE_NOT_FOUND:404,CAPTURE_CONFLICT:409,SOURCE_TOO_LARGE:413,CAPTURE_STATE_INVALID:503,CAPTURE_UNAVAILABLE:503,OUTCOME_UNKNOWN:503,AUTHENTICATION_REQUIRED:401,ORGANIZATION_SELECTION_REQUIRED:409,ORGANIZATION_CONTEXT_CHANGED:409,IDENTITY_CONTEXT_CHANGED:409,ORGANIZATION_ACCESS_DENIED:403,ORGANIZATION_ACCESS_REVOKED:403,ORGANIZATION_SUSPENDED:403,PERMISSION_DENIED:403,DEPENDENCY_UNAVAILABLE:503};

// Reject duplicate object keys before Zod/JSON coercion; the Go owner remains
// authoritative for canonical digest, identity and factual mapping.
export function parseAmazonJSON(raw:string,maxBytes=AMAZON_CAPTURE_MAX_BYTES):unknown{
 if(new TextEncoder().encode(raw).length>maxBytes)throw new Error("SOURCE_TOO_LARGE");
 const errors:ParseError[]=[];const root=parseTree(raw,errors,{allowTrailingComma:false,disallowComments:true});if(!root||errors.length)throw new Error("INVALID_CAPTURE");
 let count=0;function check(node:Node,depth:number){if(++count>100000||depth>32)throw new Error("INVALID_CAPTURE");if(node.type==="object"){const keys=new Set<string>();for(const prop of node.children??[]){const key=prop.children?.[0]?.value;if(typeof key!=="string"||keys.has(key))throw new Error("INVALID_CAPTURE");keys.add(key);}}for(const child of node.children??[])check(child,depth+1);}
 check(root,0);return getNodeValue(root);
}
