import {z} from "zod";
import {aiPointUnit} from "./ai-points";
export const routeFact=z.string().min(1).max(128).regex(/^[^\s\x00-\x1f\x7f]+$/);

const modelCategory=z.enum(["text","image-generation","multimodal"]);
const nonnegative=z.number().int().min(0).max(1e12);
const positive=nonnegative.refine(v=>v>0);
export const officialRates=z.strictObject({inputText:nonnegative,cachedInputText:nonnegative,inputImage:nonnegative,cachedInputImage:nonnegative,outputText:nonnegative,outputImage:nonnegative,outputImagePerItem:nonnegative.optional()});
const officialPrice=z.strictObject({unitVersion:aiPointUnit,priceVersion:routeFact,fxVersion:routeFact,currency:z.string().regex(/^[A-Z]{3}$/),cnyMicrosPerCurrencyUnit:positive,model:routeFact,channel:routeFact,usageVersion:routeFact,rates:officialRates});
const reference=z.strictObject({id:routeFact.max(64),version:routeFact});
const sharedSelection=z.strictObject({schemaVersion:z.literal(1),unitVersion:aiPointUnit,defaultChannel:routeFact,text:reference,image:reference,multimodal:reference});
export const routeInput=z.strictObject({expectedRevision:routeFact,selection:sharedSelection});
const modelOption=z.strictObject({category:modelCategory,reference,channel:routeFact,model:routeFact,ready:z.boolean(),reason:routeFact.optional(),price:officialPrice.optional()});
const preview=z.strictObject({ready:z.boolean(),models:z.array(modelOption).length(3)}).refine(v=>v.ready===v.models.every(m=>m.ready)&&new Set(v.models.map(m=>m.category)).size===3);
const date=z.iso.datetime({offset:true}).refine(value=>!/\.\d{10}/.test(value));
const interval={sourceUrl:z.url().max(2048),verifiedAt:date,effectiveFrom:date,effectiveUntil:date};
export const referenceImageEstimate=z.strictObject({inputTokens:z.number().int().min(1).max(2**30),outputTokens:z.number().int().min(1).max(2**30),operationScope:z.literal("product-image-effects-v1"),size:z.literal("1024x1024"),quality:z.literal("auto"),allowAutoSize:z.literal(true),maximumPromptBytes:z.number().int().min(1).max(65536),maximumReferenceImages:z.number().int().min(1).max(16),maximumReferenceBytes:z.number().int().min(1).max(64*1024*1024),maximumReferencePixels:z.number().int().min(1).max(2**30)});
const publishedPrice=z.strictObject({id:routeFact.max(64),version:routeFact.optional(),region:routeFact.max(64),serviceType:routeFact.max(64),image:z.strictObject({width:z.number().int().positive().max(8192),height:z.number().int().positive().max(8192),quality:routeFact.max(64),referenceEstimate:referenceImageEstimate.optional()}).optional(),provider:z.enum(["openai","google","qwen","deepseek"]),channel:routeFact,model:routeFact,currency:z.string().regex(/^[A-Z]{3}$/),usageVersion:routeFact,rates:officialRates,...interval});
const publishedFX=z.strictObject({version:routeFact.optional(),currency:z.string().regex(/^[A-Z]{3}$/),cnyMicrosPerCurrencyUnit:positive,rateDate:z.string().regex(/^\d{4}-\d{2}-\d{2}$/),...interval});
export const priceInput=z.strictObject({expectedRevision:routeFact,prices:z.array(publishedPrice).max(32),fx:z.array(publishedFX).max(8)}).refine(v=>v.prices.every(p=>!p.version)&&v.fx.every(f=>!f.version));
const priceSnapshot=z.strictObject({schemaVersion:z.literal(1),unitVersion:aiPointUnit,prices:z.array(publishedPrice).max(32),fx:z.array(publishedFX).max(8)});
const routePayload=z.strictObject({bindingId:z.literal(""),credentialVersion:z.literal(""),rates:z.strictObject({}),sharedModels:sharedSelection.optional(),platformPrices:priceSnapshot.optional()}).refine(p=>!!p.sharedModels!==!!p.platformPrices);
const routeScope=z.strictObject({organizationId:routeFact,capability:z.enum(["ai.models.enterprise","ai.models.official-prices"])});
const routeRevision=z.strictObject({id:routeFact,scope:routeScope,parent:routeFact,schemaVersion:z.literal(1),payload:routePayload,digest:routeFact,actorId:routeFact,publishedAt:date,key:routeFact});
export const routeCurrent=z.strictObject({status:z.enum(["READY","NEEDS_CONFIGURATION","UNCONFIGURED"]),revision:routeRevision.nullable(),preview:preview.nullable()});
export const routeOptions=z.strictObject({organizationId:routeFact,channels:z.array(routeFact).min(1).max(128),models:z.array(modelOption).max(256)});
export const routeReceipt=z.strictObject({receipt:routeRevision});
export const routePreview=z.strictObject({selection:sharedSelection,expectedRevision:routeFact,preview});
export const pricePreview=z.strictObject({expectedRevision:routeFact,snapshot:priceSnapshot});
export const routeHistory=z.strictObject({revisions:z.array(routeRevision).max(50),nextCursor:z.union([routeFact,z.literal("")])});
export const routeCapabilities=z.strictObject({organizationId:routeFact,unitVersion:aiPointUnit,available:z.boolean(),canMaintainPrices:z.boolean()});
export type RouteRevision=z.infer<typeof routeRevision>;
export type RouteInput=z.infer<typeof routeInput>;
export type ModelOption=z.infer<typeof modelOption>;
export type PriceInput=z.infer<typeof priceInput>;
export const modelLabels={text:"文本模型","image-generation":"图片生成模型",multimodal:"多模态理解模型"} as const;
export function formatCurrencyMicros(value:number):string{const n=BigInt(value),tail=(n%BigInt(1000000)).toString().padStart(6,"0").replace(/0+$/g,"");return (n/BigInt(1000000)).toString()+(tail?"."+tail:"")}
export function parseCurrencyMicros(value:string):number{if(!/^(0|[1-9]\d*)(\.\d{1,6})?$/.test(value))throw new Error("INVALID");const [a,b=""]=value.split("."),n=BigInt(a)*BigInt(1000000)+BigInt(b.padEnd(6,"0"));if(n>BigInt("1000000000000"))throw new Error("INVALID");return Number(n)}

// Go persists RFC3339 nanoseconds canonically. Compare price facts, not JSON
// timestamp spelling or an omitted zero optional dimension, after publication.
export function canonicalPriceFacts(value:Pick<PriceInput,"prices"|"fx">):string{
 const instant=(value:string)=>{const match=/^(.*T[0-9]{2}:[0-9]{2}:[0-9]{2})(?:\.([0-9]{1,9}))?(Z|[+-][0-9]{2}:[0-9]{2})$/.exec(value);if(!match)throw new Error("INVALID_TIME");const seconds=new Date(match[1]+match[3]).toISOString().replace(/\.000Z$/,""),fraction=(match[2]??"").replace(/0+$/g,"");return seconds+(fraction?"."+fraction:"")+"Z"};
 const interval=<T extends {verifiedAt:string;effectiveFrom:string;effectiveUntil:string}>(entry:T)=>({...entry,verifiedAt:instant(entry.verifiedAt),effectiveFrom:instant(entry.effectiveFrom),effectiveUntil:instant(entry.effectiveUntil)});
 return JSON.stringify({prices:value.prices.map(value=>{const entry=withoutPublicationVersion(value);return interval({...entry,rates:{...entry.rates,outputImagePerItem:entry.rates.outputImagePerItem||undefined}})}),fx:value.fx.map(value=>interval(withoutPublicationVersion(value)))});
}

export function withoutPublicationVersion<T extends {version?:string}>(value:T):Omit<T,"version">{const copy={...value};delete copy.version;return copy}
