import {z} from "zod";
import {routeFact} from "./model-route-pricing";

const credentialRevision=z.string().regex(/^(0|[1-9][0-9]{0,18})$/).refine(v=>BigInt(v)<BigInt(2)**BigInt(63));
const expectedRevision=credentialRevision.refine(v=>BigInt(v)<BigInt(2)**BigInt(62));

export const openAIQueryModel="gpt-5.4-mini-2026-03-17";
const modelQueryReference=z.strictObject({id:routeFact.max(64),version:routeFact});
export const modelQueryTargets=z.array(z.strictObject({connectionId:routeFact,reference:modelQueryReference})).max(128);
export const modelQueryInput=z.strictObject({expectedMemberId:routeFact,expectedRevision:expectedRevision.refine(v=>v!=="0"),reference:modelQueryReference});
export const modelQueryResult=z.strictObject({
 organizationId:routeFact,actorId:routeFact,memberId:routeFact,connectionId:routeFact,expectedRevision,
 reference:modelQueryReference,
 result:z.strictObject({
  model:z.literal(openAIQueryModel),
  status:z.enum(["found","unconfirmed","invalid-key","authentication-rejected","read-forbidden","region-restricted","rejected","not-found","rate-limited","timeout","unavailable"]),
  checkedAt:z.iso.datetime(),retryAfter:z.iso.datetime().optional(),
 }).refine(v=>!v.retryAfter||v.status==="rate-limited"&&Date.parse(v.retryAfter)>Date.parse(v.checkedAt)),
});
export type ModelQueryInput=z.infer<typeof modelQueryInput>;
export type ModelQueryResult=z.infer<typeof modelQueryResult>;
export function matchesModelQuery(result:ModelQueryResult,scope:{userId:string;organizationId:string},connectionId:string,input:ModelQueryInput){
 return result.organizationId===scope.organizationId&&result.actorId===scope.userId&&result.memberId===input.expectedMemberId&&result.connectionId===connectionId&&result.expectedRevision===input.expectedRevision&&result.reference.id===input.reference.id&&result.reference.version===input.reference.version;
}

const secret=z.string().min(1).max(4096).refine(v=>v.trim()===v&&new TextEncoder().encode(v).length<=4096);
const account=z.string().min(1).max(128).regex(/^[^\x00-\x1f\x7f-\x9f]+$/).refine(v=>v.trim()===v&&new TextEncoder().encode(v).length<=128);
export const credentialInput=z.discriminatedUnion("action",[
 z.strictObject({action:z.literal("set-key"),expectedRevision,expectedMemberId:routeFact.optional(),account:account.optional(),apiKey:secret}),
 z.strictObject({action:z.literal("disable"),expectedRevision,expectedMemberId:routeFact.optional()}),
]);
export const channelTemplate=z.enum(["openai-mini-v1","openai-compatible-mini-v1","grsai-product-image-v1"]);
export const channelName=account;
export const channelBaseUrl=z.url().max(2048).refine(v=>v.startsWith("https://"));
export const channelAPIKey=secret;
export const channelRegistration=z.strictObject({
 organizationId:routeFact,connectionId:routeFact,template:channelTemplate,name:account,revision:credentialRevision,
});
export const managedConnection=z.strictObject({
 organizationId:routeFact,connectionId:routeFact,provider:routeFact,channel:routeFact,region:routeFact,
 account:z.union([account,z.literal("")]),revision:credentialRevision,enabled:z.boolean(),keyPresent:z.boolean(),
 models:z.array(z.strictObject({id:routeFact,version:routeFact,model:routeFact,category:z.enum(["text","multimodal","image-generation"])})).max(128),
 registration:channelRegistration.optional(),baseUrl:channelBaseUrl.optional(),readOnly:z.literal(true).optional(),
}).refine(c=>c.revision==="0"?!c.enabled&&!c.keyPresent&&c.account==="":!!c.account&&c.keyPresent);
export const credentialConnections=z.strictObject({organizationId:routeFact,canManage:z.boolean(),connections:z.array(managedConnection).max(128),modelQueryTargets:modelQueryTargets.optional()}).refine(v=>{
 const connections=new Map(v.connections.map(c=>[c.connectionId,c]));
 const targets=v.modelQueryTargets??[];
 return connections.size===v.connections.length&&v.connections.every(c=>c.organizationId===v.organizationId)&&
  new Set(targets.map(t=>t.connectionId)).size===targets.length&&targets.every(t=>{
   const c=connections.get(t.connectionId);
   return c?.provider==="openai"&&c.channel==="openai-direct"&&c.models.some(m=>m.category==="text"&&m.model===openAIQueryModel&&m.id===t.reference.id&&m.version===t.reference.version);
  });
});
export const credentialReceipt=z.strictObject({receipt:z.strictObject({
 organizationId:routeFact,actorId:routeFact,requestKey:routeFact,connectionId:routeFact,expectedRevision:credentialRevision,
 action:z.enum(["set-key","disable"]),connection:managedConnection,
}).refine(r=>r.connection.organizationId===r.organizationId&&r.connection.connectionId===r.connectionId&&BigInt(r.connection.revision)===BigInt(r.expectedRevision)+BigInt(1)&&r.connection.enabled===(r.action==="set-key")&&r.connection.keyPresent)});

// This is the only persisted command shape. In particular it cannot accept an
// input/body/Key, even when the existing generic pending helper is reused.
export const credentialPending=z.strictObject({key:z.uuid(),userId:routeFact,organizationId:routeFact,connectionId:routeFact,expectedRevision:credentialRevision,action:z.enum(["set-key","disable"])});
export type ManagedConnection=z.infer<typeof managedConnection>;
export type CredentialInput=z.infer<typeof credentialInput>;
export type CredentialPending=z.infer<typeof credentialPending>;
export type CredentialReceipt=z.infer<typeof credentialReceipt>["receipt"];
export function matchesCredentialReceipt(receipt:CredentialReceipt,intent:Omit<CredentialPending,"key">&{key:string}){
 return receipt.organizationId===intent.organizationId&&receipt.actorId===intent.userId&&receipt.requestKey===intent.key&&receipt.connectionId===intent.connectionId&&receipt.expectedRevision===intent.expectedRevision&&receipt.action===intent.action;
}
