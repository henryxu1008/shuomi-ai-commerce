import {z} from "zod";
import {collectionID,mediaHashSchema} from "./product-collection";

const note=z.string().trim().min(1).refine(value=>new TextEncoder().encode(value).byteLength<=4096);
const positive=z.number().int().positive().max(40);
const intakeBody=z.object({notes:note,gallery:z.array(mediaHashSchema).max(40),imageNumber:positive.optional(),imageUrl:z.url({protocol:/^https$/}).max(2048).optional()}).strict();
const ref=z.object({Kind:z.literal("agent-configuration-v1"),ID:collectionID,Digest:mediaHashSchema}).strict();
const quote=z.object({InputHash:mediaHashSchema,ProfileDigest:mediaHashSchema,MaximumTokens:z.number().int().positive(),MaximumCostMicros:z.number().int().nonnegative(),Currency:z.string().regex(/^[A-Z]{3}$/)}).strict();
export const intakePrepared=z.object({configurationSnapshotRef:ref,quote}).strict();
const intakeConfirm=intakeBody.extend({configurationSnapshotRef:ref,quoteDigest:mediaHashSchema});
const fact=z.object({value:z.string().min(1).max(512),evidence:z.string().min(1).max(512)}).strict();
const candidate=z.object({
 title:z.string().max(2000).optional(),
 description:z.string().max(8192).optional(),
 brand:fact.optional(),
 attributes:z.record(z.string(),fact).optional(),
 questions:z.array(z.string().max(512)).max(12).optional(),
 rows:z.array(z.object({
   attributes:z.record(z.string(),z.string()).nullable(),
   evidence:z.string().min(1).max(512),
   sku:z.string().max(200).optional(),currency:z.string().regex(/^[A-Z]{3}$/).optional(),
   price:z.number().finite().nonnegative().optional(),stock:z.number().int().nonnegative().optional(),
   image:positive.optional(),
 }).strict()).max(256).optional(),
}).strict();
export type IntakeCandidate=z.infer<typeof candidate>;
export type IntakePrepared=z.infer<typeof intakePrepared>;
export type IntakeBody=z.infer<typeof intakeBody>;
export const intakeResult=z.object({requestKey:collectionID,phase:z.enum(["running","human_review_required","interrupted","stopped"]),stopReason:z.string().max(128).optional(),candidate:candidate.optional()}).strict();

export function ownIntakeEndpoint(url:URL,method:string){
 if(url.search||url.hash)return null;
 const path=url.pathname.slice("/api/workbench/own-product-intake/".length);
 if(method==="POST"&&path==="prepare")return {path,input:intakeBody,output:intakePrepared};
 const [key,action,...rest]=path.split("/");
 if(rest.length||!collectionID.safeParse(key).success)return null;
 if(method==="GET"&&action===undefined)return {path,output:intakeResult};
 if(method==="POST"&&action==="confirm")return {path,input:intakeConfirm,output:intakeResult};
 return null;
}

