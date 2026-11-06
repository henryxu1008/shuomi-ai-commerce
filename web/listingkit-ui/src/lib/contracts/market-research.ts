import {z} from "zod";
import {MAX_1688_ACQUISITION_VARIANTS} from "./product-acquisition";
import {collectionID,collectionVersion,isResearchFixtureProduct,collectionItemSchema,type CollectionItem} from "./product-collection";
const revision=z.number().int().positive().max(Number.MAX_SAFE_INTEGER),digest=z.string().regex(/^[a-f0-9]{64}$/);
const text=(max:number)=>z.string().refine(v=>!v.includes("\0")&&new TextEncoder().encode(v).length<=max);
const researchSelection=z.strictObject({itemId:collectionID,revision,publicationId:collectionID,version:collectionVersion});
export type ResearchSelection=z.infer<typeof researchSelection>;
export const selectionOf=(item:CollectionItem):ResearchSelection=>({itemId:item.id,revision:item.revision,publicationId:item.source.publicationId,version:item.source.version});
const source=collectionItemSchema.shape.source;
const reference={selection:researchSelection,source,jobId:collectionID};
const sharedObservation={item:collectionItemSchema,batchId:collectionID,title:text(2000),sourceUrl:z.url().refine(v=>URL.canParse(v)&&new URL(v).protocol==="https:"),parserVersion:text(128),missing:z.array(text(128)).max(256)};
const amazonObservation=z.strictObject({
 ...sharedObservation,ref:z.strictObject({...reference,checksum:text(256).min(1)}),
 query:z.strictObject({mode:z.enum(["keyword","category","asin"]),keyword:text(800).optional(),categoryNode:text(20).optional(),asinCount:z.number().int().min(0).max(200),limit:z.number().int().min(1).max(200),fields:z.array(text(128)).max(32)}),
 site:text(4).min(1),asin:z.string().regex(/^[A-Z0-9]{10}$/),capturedAt:z.iso.datetime({offset:true}),currency:text(16).optional(),price:z.number().finite().positive().optional(),priceBasis:text(64).optional(),
});
const goodsId=z.string().regex(/^[1-9][0-9]{0,63}$/);
const emptyQueryFields={asinCount:z.literal(0),limit:z.literal(0),fields:z.array(z.never()).length(0)};
// The source validates the calendar time with Go and preserves the original text.
const sheinCapturedAt=z.union([z.literal(""),z.iso.date(),z.string().regex(/^\d{4}-\d{2}-\d{2}T\d{1,2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:\d{2})$/)]);
const sheinObservation=z.strictObject({
 ...sharedObservation,
 ref:z.strictObject({...reference,checksum:z.string().regex(/^sha256:[0-9a-f]{64}$/),queryRef:collectionID,observationVersion:z.literal("1"),observationEncodingVersion:z.literal("shein-us-observation-json-v1")}),
 query:z.discriminatedUnion("mode",[z.strictObject({mode:z.literal("category"),categoryRef:goodsId,...emptyQueryFields}),z.strictObject({mode:z.literal("keyword"),keyword:text(800).min(1),...emptyQueryFields})]),
 site:z.literal(""),asin:z.literal(""),capturedAt:sheinCapturedAt,
 shein:z.strictObject({market:z.literal("US"),nativeGoodsId:goodsId,price:z.strictObject({
  amount:z.string().max(32).regex(/^(0|[1-9][0-9]*)(\.[0-9]+)?$/),currency:z.string().regex(/^[A-Z]{3}$/),basis:z.literal("listing_display"),promotionNote:text(512).nullable(),taxIncluded:z.boolean().nullable(),unit:text(128).nullable(),
 }).nullable()}),
}).refine(v=>{
 try {
  const url=new URL(v.sourceUrl);
  return url.host==="us.shein.com"&&!url.username&&!url.password&&url.pathname.endsWith(`-p-${v.shein.nativeGoodsId}.html`);
 } catch { return false; }
});
const fixtureObservation=amazonObservation.extend({
 sourceUrl:z.literal(""),capturedAt:z.literal(""),parserVersion:z.literal("research_acceptance_fixture/v1"),
 fixtureVersion:z.enum(["1","3"]),priceBasis:z.literal("synthetic_listing_offer"),
});
const observation=z.union([amazonObservation,sheinObservation,fixtureObservation]).refine(v=>{
 const {item,ref}=v;
 const kind="shein" in v?"shein_us_data":"amazon_data";
 return ("fixtureVersion" in v)===isResearchFixtureProduct(item.source.productKey)&&
  item.id===ref.selection.itemId&&item.revision===ref.selection.revision&&item.batchId===v.batchId&&
  item.source.kind===kind&&ref.source.kind===kind&&item.source.publicationId===ref.selection.publicationId&&
  item.source.version===ref.selection.version&&item.source.productKey===ref.source.productKey&&
  item.source.publicationId===ref.source.publicationId&&item.source.version===ref.source.version&&item.source.operationId===ref.source.operationId;
});
const fixtureOriginal=z.strictObject({id:text(32),referenceVersion:z.enum(["1","2","3"]),title:text(2000),site:z.enum(["us","1688"]),width:text(128),
 installation:text(128).optional(),pack:text(128).optional(),price:z.number().positive().optional(),currency:z.literal("USD").optional(),sku:text(128).optional()});
const fixtureItem=z.strictObject({item:collectionItemSchema,original:fixtureOriginal,checksum:z.string().regex(/^sha256:[0-9a-f]{64}$/)})
 .refine(v=>isResearchFixtureProduct(v.item.source.productKey));
const fixtureBatch=z.strictObject({id:collectionID,keyword:z.literal("under sink organizer"),amazon:z.tuple([fixtureItem,fixtureItem]),supplier:fixtureItem})
 .refine(({amazon:[a,b],supplier:s})=>a.item.batchId===b.item.batchId&&a.item.source.kind==="amazon_data"&&b.item.source.kind==="amazon_data"&&s.item.source.kind==="acquisition"&&
   a.original.id==="B000123456"&&a.original.referenceVersion==="3"&&b.original.id==="B000123457"&&b.original.referenceVersion==="1"&&s.original.id==="981645030344"&&s.original.referenceVersion==="2");
export const researchFixtureStatus=z.discriminatedUnion("ready",[
 z.strictObject({ready:z.literal(false)}),z.strictObject({ready:z.literal(true),batch:fixtureBatch}),
]);
export type ResearchFixtureBatch=z.infer<typeof fixtureBatch>;
const researchState=z.enum(["TO_RESEARCH","PENDING_CONFIRMATION","APPROVED_FOR_TEST","REJECTED"]);
const candidateSchema=z.strictObject({id:collectionID,supplierReview:z.literal(true).optional(),state:researchState,reason:text(4096).optional(),revision,selectedLinkId:collectionID.optional(),evidenceDigest:digest.optional(),observation:observation.optional(),evidence:z.array(z.strictObject({id:collectionID,observation:observation.optional()})).max(21),history:z.array(z.strictObject({state:researchState,reason:text(4096).optional(),evidenceDigest:digest.optional(),revision,at:z.iso.datetime({offset:true})})).max(20)});
export type ResearchCandidate=z.infer<typeof candidateSchema>;
export const researchPage=z.strictObject({items:z.array(candidateSchema).max(50),nextCursor:collectionID.optional(),incomplete:z.boolean()});
const change={candidateId:collectionID,expectedRevision:revision};
const comparison=z.strictObject({
 marketKey:text(128).refine(v=>!!v.trim()),
 supplierKey:text(128).refine(v=>!!v.trim()),
 supplierLevel:z.enum(["sku","product"]),
 result:z.enum(["SAME","DIFFERENT","PENDING"]),
});
const supplierStatus=z.enum(["CONFIRMED","PENDING_REVIEW"]);
const skuId=text(128).refine(v=>!!v.trim());
const quantity=z.number().int().min(1).max(1_000_000);
const supplierPublicationPrefixes=["source-run:acquisition:","source-run:research-acceptance:"];
const supplierSelection=researchSelection.extend({
 publicationId:z.string().refine(value=>supplierPublicationPrefixes.some(prefix=>value.startsWith(prefix)&&collectionID.safeParse(value.slice(prefix.length)).success)),
});
const supplierInput=z.strictObject({
 selection:supplierSelection,skuId,quantity,unit:z.literal("piece"),status:supplierStatus,
 reason:text(2048).refine(v=>!!v.trim()),comparisons:z.array(comparison).min(1).max(5),
}).refine(v=>v.status==="PENDING_REVIEW"||v.comparisons.every(c=>c.result!=="PENDING"));
const attributes=z.record(text(128),text(512));
export const supplierReviewPreview=z.strictObject({
 revision,linkId:collectionID,evidenceDigest:digest,
 market:z.strictObject({asin:text(10).refine(v=>v.length===10),title:text(2000),attributes,fixtureVersion:collectionVersion.optional()}),
 offer:z.strictObject({title:text(2000),attributes,fixtureVersion:collectionVersion.optional(),variants:z.array(z.strictObject({id:skuId,label:text(512),attributes})).max(MAX_1688_ACQUISITION_VARIANTS)}),
});
const supplierReviewDetail=z.strictObject({
 marketTitle:text(2000),asin:text(10).refine(v=>v.length===10),supplierTitle:text(2000),offerId:text(128),
 marketVersion:collectionVersion,supplierVersion:collectionVersion,skuId,skuLabel:text(512),quantity,
 marketFixtureVersion:collectionVersion.optional(),supplierFixtureVersion:collectionVersion.optional(),
 unit:z.literal("piece"),status:supplierStatus,reason:text(2048),at:z.iso.datetime({offset:true}),
 comparisons:z.array(comparison.extend({marketValue:text(512).optional(),supplierValue:text(512).optional()})).min(1).max(5),
});
export const supplierReviewPage=z.strictObject({items:z.array(z.strictObject({current:z.boolean(),detail:supplierReviewDetail.optional()})).max(5),nextBefore:revision.optional()});
export const supplierReviewPending=z.strictObject({key:collectionID});
const researchCommand=z.discriminatedUnion("action",[
 z.strictObject({action:z.literal("link"),selections:z.array(researchSelection).min(1).max(200).refine(v=>new Set(v.map(s=>s.itemId)).size===v.length)}),
 z.strictObject({action:z.literal("select"),...change,linkId:collectionID}),z.strictObject({action:z.literal("unlink"),...change,linkId:collectionID}),
 z.strictObject({action:z.literal("review"),...change,evidenceDigest:digest}),z.strictObject({action:z.literal("reset"),...change,evidenceDigest:digest}),
  z.strictObject({action:z.literal("decide"),...change,evidenceDigest:digest,state:z.enum(["APPROVED_FOR_TEST","REJECTED"]),reason:text(4096).refine(v=>!!v.trim())}),
 z.strictObject({action:z.literal("record_supplier_review"),...change,linkId:collectionID,evidenceDigest:digest,supplier:supplierInput}),
]);
export type ResearchCommand=z.infer<typeof researchCommand>;
export const researchIntent=z.strictObject({key:collectionID,body:researchCommand});
export const researchReceipt=z.strictObject({key:collectionID,revisions:z.record(collectionID,revision).refine(v=>Object.keys(v).length<=200),candidateIds:z.array(collectionID).max(200),replayed:z.boolean()});
export const researchTransfer=z.array(researchSelection).min(1).max(5).refine(v=>new Set(v.map(s=>s.itemId)).size===v.length);
export function researchEndpoint(url:URL,method:string){
 const fixturePath=url.pathname.match(/^\/api\/workbench\/research\/projects\/([0-9a-f-]{36})\/acceptance-fixture$/);
 if(fixturePath){
  if(url.search||url.hash||!collectionID.safeParse(fixturePath[1]).success||method!=="GET"&&method!=="POST")return null;
  return {path:"/"+fixturePath[1]+"/acceptance-fixture",input:method==="POST"?z.strictObject({}):undefined,output:researchFixtureStatus,expected:false};
 }
 const supplierPath=url.pathname.match(/^\/api\/workbench\/research\/projects\/([^/]+)\/candidates\/([^/]+)\/supplier-review$/);
 if(supplierPath){
  if(method!=="GET"||url.hash||!collectionID.safeParse(supplierPath[1]).success||!collectionID.safeParse(supplierPath[2]).success)return null;
  const keys=[...url.searchParams.keys()];
  const path="/"+supplierPath[1]+"/candidates/"+supplierPath[2]+"/supplier-review";
  const positive=(value:string|null)=>!!value&&/^[1-9][0-9]*$/.test(value)&&Number.isSafeInteger(Number(value));
  if(!keys.length||keys.length===1&&keys[0]==="before"&&positive(url.searchParams.get("before"))){
   return {path,input:undefined,output:supplierReviewPage,expected:false};
  }
  if(keys.length!==4||new Set(keys).size!==4||!positive(url.searchParams.get("revision"))||
   !supplierSelection.safeParse({itemId:url.searchParams.get("itemId"),revision:Number(url.searchParams.get("revision")),publicationId:url.searchParams.get("publicationId"),version:url.searchParams.get("version")}).success)return null;
  return {path,input:undefined,output:supplierReviewPreview,expected:false};
 }
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
