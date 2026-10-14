import { z } from "zod";
import { knowledgeId } from "../api/knowledge";
import { templateRefSchema, configVersion } from "./agent-configuration";
import { executionModeSchema, modeRevisionSchema } from "./execution-mode";
const hash=z.string().regex(/^[a-f0-9]{64}$/), integer=z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const text=(n:number)=>z.string().refine(s=>new TextEncoder().encode(s).length<=n&&!/[\u0000-\u001f\u007f]/.test(s));

export const artworkPrepareSchema=z.strictObject({theme:text(512).refine(s=>!!s.trim()),requirements:text(4096),contextId:knowledgeId,expectedModeRevision:modeRevisionSchema,template:templateRefSchema.optional()});
const artworkCommandSchema=z.strictObject({planRevision:integer.positive(),planDigest:hash.optional(),quoteDigest:hash.optional(),resultDigest:hash.optional(),slotId:z.literal("artwork").optional(),attempt:integer.positive().optional()});
const candidate=z.strictObject({assetId:z.string().min(1).max(256),url:z.url().refine(s=>{const u=new URL(s);return u.protocol==="https:"&&!u.username&&!u.password&&!u.hash}),hash,width:integer.positive(),height:integer.positive()});
export const artworkRunSchema=z.strictObject({runId:knowledgeId,requestKey:knowledgeId,contextId:knowledgeId,theme:text(512),requirements:text(4096),status:z.enum(["planning","awaiting_plan_approval","executing","evaluating","repairing","awaiting_final_approval","blocked","completed","failed","cancelled"]),planRevision:integer.positive(),planDigest:hash,quoteDigest:hash,points:integer.positive(),settledPoints:integer,candidates:z.array(candidate).max(1),resultDigest:z.union([hash,z.literal("")]),confirmationActionId:z.union([knowledgeId,z.literal("")]),generationAdmitted:z.boolean(),generationAllowed:z.boolean(),saveAvailable:z.boolean(),saved:z.strictObject({operationId:knowledgeId,batchId:knowledgeId,itemId:knowledgeId,productKey:z.string().min(1).max(128),publicationId:z.string().min(1).max(128),sourceVersion:configVersion,artifactHash:hash}).nullable(),mode:executionModeSchema,modeRevision:modeRevisionSchema,template:templateRefSchema,block:z.strictObject({Code:text(192),SlotID:text(128)}).nullable(),pendingCommand:z.strictObject({ActionID:knowledgeId,Kind:text(128),Phase:text(128),PlanRevision:integer.positive()}).nullable(),recoverableEffects:z.array(z.strictObject({SlotID:z.literal("artwork"),Attempt:integer.positive(),Code:text(192)})).max(1).nullable()}).refine(v=>(v.status==="completed")===!!v.saved).refine(v=>!v.saveAvailable||v.status==="awaiting_final_approval"&&v.generationAllowed&&!!v.resultDigest&&v.candidates.length===1&&!v.pendingCommand).refine(v=>!v.generationAllowed||v.mode!=="read_only");
export const artworkRecentSchema=z.strictObject({items:z.array(z.strictObject({runId:knowledgeId,contextKind:z.literal("pod-artwork-v1"),contextId:knowledgeId,status:artworkRunSchema.shape.status,targetPlatform:z.literal("sds"),createdAt:z.iso.datetime({offset:true})})).max(40),nextCursor:z.string().max(180)});
export const artworkIntentSchema=z.discriminatedUnion("kind",[
 z.strictObject({kind:z.literal("prepare"),key:knowledgeId,body:artworkPrepareSchema}),
 z.strictObject({kind:z.enum(["confirm","save","recover","resume","cancel"]),key:knowledgeId,runId:knowledgeId,body:artworkCommandSchema})
]);
export type ArtworkRun=z.infer<typeof artworkRunSchema>;
export type ArtworkIntent=z.infer<typeof artworkIntentSchema>;
export function artworkEndpoint(url:URL,method:string):{path:string;output:z.ZodType;input?:z.ZodType}|null {
 const prefix="/api/workbench/pod-artwork/";if(!url.pathname.startsWith(prefix)||url.search)return null;
 const path=url.pathname.slice(prefix.length),p=path.split("/");
 if(method==="GET"&&p.length===2&&p[0]==="runs"&&knowledgeId.safeParse(p[1]).success)return {path,output:artworkRunSchema};
 if(method==="GET"&&p.length===3&&p[0]==="by-key"&&p.slice(1).every(s=>knowledgeId.safeParse(s).success))return {path,output:artworkRunSchema};
 if(method==="POST"&&path==="prepare")return {path,output:artworkRunSchema,input:artworkPrepareSchema};
 if(method==="POST"&&p.length===3&&p[0]==="runs"&&knowledgeId.safeParse(p[1]).success&&["confirm","save","recover","resume","cancel"].includes(p[2])){
 const rev=artworkCommandSchema.pick({planRevision:true});
 const input=p[2]==="confirm"?rev.extend({planDigest:hash,quoteDigest:hash}):p[2]==="save"?rev.extend({resultDigest:hash}):p[2]==="recover"?rev.extend({slotId:z.literal("artwork"),attempt:integer.positive()}):rev;
 return {path,output:artworkRunSchema,input};
 }
 return null;
}
