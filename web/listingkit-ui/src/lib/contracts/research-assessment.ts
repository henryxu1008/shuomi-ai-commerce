import {z} from "zod";
import {collectionID,collectionVersion} from "./product-collection";

const text=(max:number)=>z.string().refine(v=>!v.includes("\0")&&new TextEncoder().encode(v).length<=max);
const revision=z.number().int().positive().max(Number.MAX_SAFE_INTEGER);
const hash=z.string().regex(/^[a-f0-9]{64}$/);
const basis=z.strictObject({source:text(240),asOf:text(10)});
const costValue=z.discriminatedUnion("kind",[
 z.strictObject({kind:z.literal("value"),value:text(16),basis:basis.optional()}),
 z.strictObject({kind:z.literal("not_applicable"),reason:text(240)}),
]);
const outcome=z.strictObject({basis,costs:z.strictObject({procurement:costValue,logistics:costValue,platformFixed:costValue,platformRate:costValue,taxFixed:costValue,taxRate:costValue,advertising:costValue,exchangeCost:costValue})});
const costInput=z.strictObject({channel:text(120),fulfillment:text(120),specification:text(120),quantity:text(16),price:text(16),refundRate:text(16),salesBasis:basis,rule:z.enum(["fixed_percent","unsupported"]),purchaseCurrency:z.enum(["USD","CNY"]),exchangeRate:text(16),exchangeBasis:basis,completed:outcome,refunded:outcome});
const money=z.string().regex(/^-?(0|[1-9][0-9]*)\.[0-9]{2}$/).max(40);
const costResult=z.discriminatedUnion("status",[
 z.strictObject({status:z.literal("INSUFFICIENT_INPUT"),fields:z.array(text(120)).max(80)}),
 z.strictObject({status:z.literal("UNSUPPORTED_COST_RULE")}),
 z.strictObject({status:z.literal("COMPUTABLE"),gross:money,contribution:money,contributionSign:z.enum(["negative","zero","positive"]),breakEven:money.nullable(),completedContribution:money.nullable(),refundedContribution:money.nullable()}),
]);
export const assessmentRef=z.strictObject({assessmentId:collectionID,revision:z.literal(1),contextHash:hash,inputHash:hash});
export const assessmentSupplierRef=z.strictObject({requestKey:collectionID,candidateRevision:revision});
const assessmentInput=z.strictObject({cost:costInput,supplierReview:assessmentSupplierRef.optional()});
export const saveAssessment=z.strictObject({action:z.literal("save_assessment"),candidateId:collectionID,expectedRevision:revision,linkId:collectionID,evidenceDigest:hash,assessment:assessmentInput});
export const assessmentPreview=z.strictObject({input:costInput,result:costResult,ruleVersion:z.literal("manual-unit-economics-v1")});
export const assessmentView=z.strictObject({id:collectionID,candidateRevision:revision,at:z.iso.datetime({offset:true}),applicability:z.enum(["CURRENT","HISTORICAL","DETAIL_UNAVAILABLE"]),detail:z.strictObject({ref:assessmentRef,input:costInput,result:costResult,ruleVersion:z.literal("manual-unit-economics-v1"),market:z.strictObject({itemId:collectionID,revision,publicationId:collectionID,version:collectionVersion}),supplierReview:assessmentSupplierRef.optional(),supplierBinding:z.enum(["UNKNOWN","CONFIRMED","PENDING_REVIEW"]),supplierSkuId:text(128).optional(),supplierQuantity:z.number().int().min(1).max(1_000_000).optional(),supplierUnit:z.literal("piece").optional(),supplierVersion:collectionVersion.optional()}).optional()}).refine(v=>v.applicability==="DETAIL_UNAVAILABLE"?!v.detail:!!v.detail&&v.detail.ref.assessmentId===v.id);
export const assessmentPage=z.strictObject({items:z.array(assessmentView).max(5),nextBefore:revision.optional()});
export type AssessmentView=z.infer<typeof assessmentView>;
export type AssessmentRef=z.infer<typeof assessmentRef>;
export type AssessmentSupplierRef=z.infer<typeof assessmentSupplierRef>;
