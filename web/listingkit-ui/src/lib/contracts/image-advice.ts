import {z} from "zod";
const id=z.string().min(1).max(128);
const version=z.string().max(19).regex(/^[1-9][0-9]*$/).refine(v=>BigInt(v)<=BigInt("9223372036854775807"));
const integer=z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const target=z.strictObject({recordId:id,revision:version,storeId:id,site:id,categoryId:integer,ruleDigest:id,sourceId:id,originalPublicationId:id,originalVersion:version,applyReceiptId:z.string().max(128)});
const placement=z.strictObject({group:z.enum(["spu","skc","sku","detail"]),skc:integer,sku:integer,type:integer.min(1),sort:integer.min(1).max(100),site:id});
const fact=z.strictObject({assetId:id,role:z.enum(["design","main","white_background","gallery"]),approved:z.literal(true),width:integer,height:integer,dimensionsStatus:z.enum(["known","unknown"]),placement:placement.optional()}).refine(v=>v.dimensionsStatus==="known"?v.width>0&&v.height>0:v.width===0||v.height===0);
const check=z.strictObject({id,code:z.enum(["dimensions_known","dimensions_unknown","dimensions_conflict","placement_conflict","required_image_missing","approved_assets_unavailable","rule_unavailable","platform_not_evaluated","owner_image_issue"]),status:z.enum(["known_satisfied","known_conflict","unknown","not_applicable"]),assetIds:z.array(id).max(10000),ruleRef:z.string().max(128),reason:z.string().min(1).max(1024)});
const suggestion=z.strictObject({kind:z.enum(["verify_fact","select_approved_asset","check_placement","open_existing_images"]),checkIds:z.array(id).min(1).max(10000),assetIds:z.array(id).max(10000),reason:z.string().min(1).max(1024)});
export const imageAdviceSchema=z.strictObject({
 schemaVersion:z.literal("image-advice-v1"),
 binding:z.strictObject({productKey:id,catalogVersion:version,publicationId:id,targetPlatform:id,inventoryDigest:id,target:target.optional()}),
 evidence:z.strictObject({toolId:z.enum(["product.image.analyze","listing.materials.inspect"]),toolVersion:id,callId:id,digest:id}),
 facts:z.array(fact).max(10000),checks:z.array(check).max(10000),suggestions:z.array(suggestion).max(10000),unresolved:z.array(z.string().min(1).max(1024)).max(10000)
}).superRefine((a,ctx)=>{
 const native=a.evidence.toolId==="listing.materials.inspect";
 if(native? a.evidence.toolVersion!=="v3.0.0"||!a.binding.target:a.evidence.toolVersion!=="v1.0.0"||!!a.binding.target)ctx.addIssue({code:"custom",message:"Advice must match its exact Tool/context"});
 const facts=new Set(a.facts.map(v=>v.assetId)),checks=new Set(a.checks.map(v=>v.id));
 if(facts.size!==a.facts.length||checks.size!==a.checks.length||a.checks.some(c=>c.assetIds.some(id=>!facts.has(id)))||a.suggestions.some(s=>s.checkIds.some(id=>!checks.has(id))||s.assetIds.some(id=>!facts.has(id))))ctx.addIssue({code:"custom",message:"Advice references facts that were not disclosed"});
});
export type ImageAdvice=z.infer<typeof imageAdviceSchema>;
export function adviceHasRecordedStep(advice:ImageAdvice|undefined,steps:{tool?:string;callId?:string;auditStatus?:string;invocationId?:string}[]){
 return !advice||steps.some(s=>s.tool===advice.evidence.toolId&&s.callId===advice.evidence.callId&&s.auditStatus==="recorded"&&!s.invocationId);
}
