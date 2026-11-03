import {z} from "zod";
import {collectionID} from "./product-collection";

const text=(max:number)=>z.string().refine(v=>new TextEncoder().encode(v).length<=max);
const hash=z.string().regex(/^[0-9a-f]{64}$/);
export const candidateCategoryIdSchema=z.string().regex(/^[1-9][0-9]{0,18}$/).pipe(z.string().refine(v=>BigInt(v)<=BigInt("9223372036854775807")));
const nonnegativeId=z.string().regex(/^(0|[1-9][0-9]{0,18})$/).pipe(z.string().refine(v=>BigInt(v)<=BigInt("9223372036854775807")));
const bounded=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
export const candidateRuleExplanationSchema=z.strictObject({schemaVersion:z.literal("listing-rule-explanation-v1"),summary:text(8192).min(1),references:z.array(z.strictObject({evidenceId:hash,callId:text(128).min(1),ruleDigest:hash})).min(1).max(16),unknowns:z.array(text(1024).min(1)).max(64)});

// Compose the current owner's schemas. Browser/BFF do not compute rules or
// maintain an independent rule catalogue.
export function candidateRulesDisclosureSchema<F extends z.ZodType,A extends z.ZodType>(fill:F,attributes:A){
 return z.strictObject({schemaVersion:z.literal("shein-candidate-rules-v1"),binding:z.strictObject({ContextKind:z.literal("listing-candidate-rules-v1"),ContextID:collectionID,ProductKey:text(128).min(1),CatalogVersion:candidateCategoryIdSchema,PublicationID:text(128).min(1),TargetPlatform:z.literal("shein"),CandidateRules:z.strictObject({BaseRevision:candidateCategoryIdSchema,BaseDigest:hash,StoreID:collectionID,Site:z.literal("shein-us"),MerchantDigest:hash,CandidateCategoryID:candidateCategoryIdSchema,SelectorDigest:hash})}),sourceId:collectionID,originalPublicationId:text(128).min(1),originalVersion:candidateCategoryIdSchema,applyReceiptId:text(128),evidenceId:hash,evidence:z.strictObject({tool:z.strictObject({ID:z.literal("listing.materials.rules.query"),Version:z.literal("v1.0.0")}),callId:text(128).min(1),ruleDigest:hash,readAt:z.iso.datetime({offset:true}),algorithmVersion:z.literal("shein-candidate-rules-v1"),officialVersionStatus:z.literal("unknown")}),categoryPath:z.array(z.strictObject({id:candidateCategoryIdSchema,name:text(8192),productTypeId:nonnegativeId})).min(1).max(100),fill,attributes,images:z.strictObject({version:z.literal("shein-images-v1"),new_scheme:z.boolean(),groups:z.array(z.strictObject({group:z.enum(["spu","skc","sku","detail"]),SKC:bounded,SKU:bounded,main_when_any:z.boolean(),types:z.array(z.strictObject({type:bounded,minimum:bounded,maximum:bounded})).max(20)})).max(10000)}).optional(),coverage:z.array(z.strictObject({section:z.enum(["category","fill","attributes","images","conditional"]),status:z.enum(["available","unknown","not_evaluated","not_applicable"]),reason:text(1024)})).length(5).refine(rows=>new Set(rows.map(r=>r.section)).size===5&&rows.find(r=>r.section==="conditional")?.status==="not_evaluated"),unknowns:z.array(text(1024)).max(64)});
}
