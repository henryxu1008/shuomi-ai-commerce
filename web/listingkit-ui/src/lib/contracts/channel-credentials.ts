import {z} from "zod";
import {routeFact} from "./model-route-pricing";

export const credentialRevision=z.string().regex(/^(0|[1-9][0-9]{0,18})$/).refine(v=>BigInt(v)<BigInt(2)**BigInt(63));
const expectedRevision=credentialRevision.refine(v=>BigInt(v)<BigInt(2)**BigInt(62));
const secret=z.string().min(1).max(4096).refine(v=>v.trim()===v&&new TextEncoder().encode(v).length<=4096);
const account=z.string().min(1).max(128).regex(/^[^\x00-\x1f\x7f-\x9f]+$/).refine(v=>v.trim()===v&&new TextEncoder().encode(v).length<=128);
export const credentialInput=z.discriminatedUnion("action",[
 z.strictObject({action:z.literal("set-key"),expectedRevision,account:account.optional(),apiKey:secret}),
 z.strictObject({action:z.literal("disable"),expectedRevision}),
]);
export const managedConnection=z.strictObject({
 organizationId:routeFact,connectionId:routeFact,provider:routeFact,channel:routeFact,region:routeFact,
 account:z.union([account,z.literal("")]),revision:credentialRevision,enabled:z.boolean(),keyPresent:z.boolean(),
 models:z.array(z.strictObject({id:routeFact,version:routeFact,model:routeFact,category:z.enum(["text","multimodal","image-generation"])})).max(128),
}).refine(c=>c.revision==="0"?!c.enabled&&!c.keyPresent&&c.account==="":!!c.account&&c.keyPresent);
export const credentialConnections=z.strictObject({organizationId:routeFact,canManage:z.boolean(),connections:z.array(managedConnection).max(128)}).refine(v=>v.connections.every(c=>c.organizationId===v.organizationId)&&new Set(v.connections.map(c=>c.connectionId)).size===v.connections.length);
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
