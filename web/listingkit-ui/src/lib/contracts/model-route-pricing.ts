import {z} from "zod";
export const routeFact=z.string().min(1).max(128).regex(/^[^\s\x00-\x1f\x7f]+$/);
const positive=z.number().int().positive().max(Number.MAX_SAFE_INTEGER);
const points=z.string().regex(/^[1-9][0-9]{0,12}$/).refine(v=>{try{return BigInt(v)<=BigInt("1000000000000")}catch{return false}});
const cost=z.string().regex(/^(0|[1-9][0-9]*)(\.[0-9]{1,6})?$/).max(24).refine(v=>{try{const [a,b=""]=v.split(".");const n=BigInt(a)*BigInt(1000000)+BigInt(b.padEnd(6,"0"));return n>BigInt(0)&&n<=BigInt("1000000000000")}catch{return false}});
const textRates=z.strictObject({inputCost:cost,outputCost:cost,inputPoints:points,outputPoints:points});
const imageRates=z.strictObject({pointsPerImage:points});
const routeRates=z.union([textRates,imageRates]);
export const routeInput=z.strictObject({expectedRevision:routeFact,bindingId:routeFact,credentialVersion:routeFact,rates:routeRates,rollbackFrom:routeFact.optional()});
const text=z.strictObject({ClientName:routeFact,ProviderID:routeFact,AdapterKind:routeFact,ModelID:routeFact,EndpointIdentityDigest:routeFact,CredentialVersion:routeFact,RoutingPolicyVersion:routeFact,AdapterPolicyVersion:routeFact,PromptVersion:routeFact,OutputSchemaVersion:routeFact,UsageMappingVersion:routeFact,CostPricingVersion:routeFact,PointTariff:z.strictObject({priceVersion:routeFact,inputPointsPerMillionTokens:positive,outputPointsPerMillionTokens:positive}),Currency:z.string().length(3),InputMicrosPerMillion:positive,OutputMicrosPerMillion:positive,MaximumPromptTokens:positive,MaximumCompletionTokens:positive,MaximumInputBytes:positive,MaximumOutputBytes:positive,DeadlineBound:positive});
const image=z.strictObject({provider:routeFact,model:routeFact,protocol:routeFact,endpointDigest:routeFact,credentialReference:routeFact.max(192),configurationVersion:routeFact,priceVersion:routeFact,points:positive});
const base={bindingId:routeFact,credentialVersion:routeFact};
const routePayload=z.union([z.strictObject({...base,rates:textRates,text}),z.strictObject({...base,rates:imageRates,image})]);
const routeScope=z.strictObject({organizationId:routeFact,capability:routeFact});
const routeRevision=z.strictObject({id:routeFact,scope:routeScope,parent:routeFact,schemaVersion:z.literal(1),payload:routePayload,digest:routeFact,actorId:routeFact,publishedAt:z.iso.datetime({offset:true}),key:routeFact,rollbackFrom:routeFact.optional()});
export const routeCurrent=z.strictObject({status:z.enum(["READY","NEEDS_CONFIGURATION","UNCONFIGURED"]),revision:routeRevision.nullable()});
export const routeReceipt=z.strictObject({receipt:routeRevision});
export const routePreview=z.strictObject({payload:routePayload,expectedRevision:routeFact,digest:routeFact});
const routeBinding=z.strictObject({id:routeFact,credentialVersion:routeFact,provider:routeFact,model:routeFact,protocol:routeFact,usageMappingVersion:routeFact.optional(),currency:z.string().length(3).optional(),maximumCostMicros:positive.optional(),maximumPointsPerImage:positive.optional(),maximumInputBytes:positive.optional(),maximumOutputBytes:positive.optional(),maximumPromptTokens:positive.optional(),maximumCompletionTokens:positive.optional(),deadlineSeconds:positive.optional()});
export const routeBindings=z.strictObject({bindings:z.array(routeBinding).max(128)});
export const routeHistory=z.strictObject({revisions:z.array(routeRevision).max(50),nextCursor:z.union([routeFact,z.literal("")])});
export const routeCapabilities=z.strictObject({organizationId:routeFact,capabilities:z.array(z.strictObject({family:z.enum(["text","image"]),capability:routeFact})).max(7)});
export type RouteRevision=z.infer<typeof routeRevision>;
export type RoutePayload=z.infer<typeof routePayload>;
export type RouteInput=z.infer<typeof routeInput>;
export type RouteBinding=z.infer<typeof routeBinding>;
export const routeCapabilityLabels:Record<string,string>={product_agent_decision:"商品标题建议","product.content.decision":"商品内容生成","listing.materials.decision":"平台素材生成","product.selection.decision":"商品选品分析","shop.operations-diagnosis.decision":"店铺运营诊断",aiworkbench_chat_plan:"硕米 Chat 规划","imageagent.generation":"图片生成（共享用途）"};

export function routeRateError(field:string,value:string):string|undefined{return (field.endsWith("Cost")?cost:points).safeParse(value).success?undefined:field.endsWith("Cost")?"须大于 0，最多 6 位小数，且不超出费率上界":"须为大于 0 的整数，且不超出费率上界"}
