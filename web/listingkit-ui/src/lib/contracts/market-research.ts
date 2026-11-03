import {z} from "zod";
import {collectionID,collectionVersion,collectionItemSchema,type CollectionItem} from "./product-collection";
const revision=z.number().int().positive().max(Number.MAX_SAFE_INTEGER),digest=z.string().regex(/^[a-f0-9]{64}$/);
const text=(max:number)=>z.string().refine(v=>!v.includes("\0")&&new TextEncoder().encode(v).length<=max);
const researchSelection=z.strictObject({itemId:collectionID,revision,publicationId:collectionID,version:collectionVersion});
export type ResearchSelection=z.infer<typeof researchSelection>;
export const selectionOf=(item:CollectionItem):ResearchSelection=>({itemId:item.id,revision:item.revision,publicationId:item.source.publicationId,version:item.source.version});
const source=collectionItemSchema.shape.source;
const observation=z.strictObject({query:z.strictObject({mode:z.enum(["keyword","category","asin"]),keyword:text(800).optional(),categoryNode:text(20).optional(),asinCount:z.number().int().min(0).max(200),limit:z.number().int().min(1).max(200),fields:z.array(text(128)).max(32)}),item:collectionItemSchema,ref:z.strictObject({selection:researchSelection,source,jobId:collectionID,checksum:text(256).min(1)}),batchId:collectionID,site:text(4).min(1),asin:z.string().regex(/^[A-Z0-9]{10}$/),title:text(2000),sourceUrl:z.url().refine(v=>new URL(v).protocol==="https:"),capturedAt:z.iso.datetime({offset:true}),parserVersion:text(128),currency:text(16).optional(),price:z.number().finite().positive().optional(),priceBasis:text(64).optional(),missing:z.array(text(128)).max(256)}).refine(v=>v.item.id===v.ref.selection.itemId&&v.item.revision===v.ref.selection.revision&&v.item.source.publicationId===v.ref.selection.publicationId&&v.item.source.version===v.ref.selection.version&&v.item.source.kind==="amazon_data"&&v.batchId===v.item.batchId);
const researchState=z.enum(["TO_RESEARCH","PENDING_CONFIRMATION","APPROVED_FOR_TEST","REJECTED"]);
const candidateSchema=z.strictObject({id:collectionID,state:researchState,reason:text(4096).optional(),revision,selectedLinkId:collectionID.optional(),evidenceDigest:digest.optional(),observation:observation.optional(),evidence:z.array(z.strictObject({id:collectionID,observation:observation.optional()})).max(21),history:z.array(z.strictObject({state:researchState,reason:text(4096).optional(),evidenceDigest:digest.optional(),revision,at:z.iso.datetime({offset:true})})).max(20)});
export type ResearchCandidate=z.infer<typeof candidateSchema>;
export const researchPage=z.strictObject({items:z.array(candidateSchema).max(50),nextCursor:collectionID.optional(),incomplete:z.boolean()});
const change={candidateId:collectionID,expectedRevision:revision};
const researchCommand=z.discriminatedUnion("action",[
 z.strictObject({action:z.literal("link"),selections:z.array(researchSelection).min(1).max(200).refine(v=>new Set(v.map(s=>s.itemId)).size===v.length)}),
 z.strictObject({action:z.literal("select"),...change,linkId:collectionID}),z.strictObject({action:z.literal("unlink"),...change,linkId:collectionID}),
 z.strictObject({action:z.literal("review"),...change,evidenceDigest:digest}),z.strictObject({action:z.literal("reset"),...change,evidenceDigest:digest}),
 z.strictObject({action:z.literal("decide"),...change,evidenceDigest:digest,state:z.enum(["APPROVED_FOR_TEST","REJECTED"]),reason:text(4096).refine(v=>!!v.trim())})]);
export type ResearchCommand=z.infer<typeof researchCommand>;
export const researchIntent=z.strictObject({key:collectionID,body:researchCommand});
export const researchReceipt=z.strictObject({key:collectionID,revisions:z.record(collectionID,revision).refine(v=>Object.keys(v).length<=200),candidateIds:z.array(collectionID).max(200),replayed:z.boolean()});
export const researchTransfer=z.array(researchSelection).min(1).max(5).refine(v=>new Set(v.map(s=>s.itemId)).size===v.length);
export function researchEndpoint(url:URL,method:string){
 const match=url.pathname.match(/^\/api\/workbench\/research\/projects\/([0-9a-f-]{36})(?:\/commands(?:\/([0-9a-f-]{36}))?)?$/i);
 if(!match||!collectionID.safeParse(match[1]).success||url.hash)return null;
 const path="/"+match[1]+(url.pathname.includes("/commands")?"/commands"+(match[2]?"/"+match[2]:""):"");
 const keys=[...url.searchParams.keys()];
 if(method==="GET"&&!path.includes("/commands")){if(keys.some(k=>k!=="after")||keys.length>1||keys.length===1&&!collectionID.safeParse(url.searchParams.get("after")).success)return null;return {path,input:undefined,output:researchPage,expected:false};}
 if(url.search)return null;
 if(method==="POST"&&path.endsWith("/commands"))return {path,input:researchCommand,output:researchReceipt,expected:false};
 if(method==="GET"&&match[2]&&collectionID.safeParse(match[2]).success)return {path,input:undefined,output:researchReceipt,expected:false};
 return null;
}
