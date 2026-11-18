import {z} from "zod";
import {routeFact} from "./model-route-pricing";
import {channelAPIKey,channelBaseUrl,channelName,channelRegistration,channelTemplate,managedConnection,modelQueryTargets} from "./channel-credentials";

const connectionID=z.string().regex(/^uc-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/);
const bindingID=z.string().regex(/^um-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/);
const revision=z.string().regex(/^[1-9][0-9]{0,18}$/).refine(v=>BigInt(v)<BigInt(2)**BigInt(62));
const member={expectedMemberId:routeFact};
const receipt={actorId:routeFact,organizationId:routeFact,memberId:routeFact,requestKey:routeFact};
const binding=z.strictObject({organizationId:routeFact,bindingId:bindingID,connectionId:connectionID,template:channelTemplate,enabled:z.boolean()});
const registeredConnection=managedConnection.refine(c=>connectionID.safeParse(c.connectionId).success&&c.baseUrl&&
 c.registration?.connectionId===c.connectionId&&c.registration.organizationId===c.organizationId);
export const managedChannels=z.strictObject({
 organizationId:routeFact,channels:z.array(registeredConnection).max(128),bindings:z.array(binding).max(128),
 templates:z.array(channelTemplate).max(2),canSave:z.boolean(),canRename:z.boolean(),canAttach:z.boolean(),
 modelQueryTargets:modelQueryTargets.nullable(),
}).refine(v=>v.channels.every(c=>c.organizationId===v.organizationId)&&v.bindings.every(b=>b.organizationId===v.organizationId));
export const createChannel=z.strictObject({...member,input:z.strictObject({
 template:channelTemplate,name:channelName,baseUrl:channelBaseUrl.optional(),account:channelName.optional(),apiKey:channelAPIKey,
}).refine(v=>v.template==="openai-mini-v1"?!v.baseUrl:!!v.baseUrl)});
export const renameChannel=z.strictObject({...member,input:z.strictObject({name:channelName,expectedRevision:revision})});
export const editBinding=z.strictObject({...member,input:z.discriminatedUnion("action",[
 z.strictObject({action:z.literal("attach"),expectedRevision:revision}),
 z.strictObject({action:z.literal("disable"),expectedRevision:revision,bindingId:bindingID}),
])});
export const createChannelReceipt=z.strictObject({...receipt,result:z.strictObject({registration:channelRegistration,connection:registeredConnection})})
 .refine(r=>r.result.registration.organizationId===r.organizationId&&r.result.connection.organizationId===r.organizationId&&
  r.result.connection.connectionId===r.result.registration.connectionId&&r.result.connection.revision==="1"&&r.result.registration.revision==="1");
export const renameChannelReceipt=z.strictObject({...receipt,result:channelRegistration}).refine(r=>r.result.organizationId===r.organizationId);
export const editBindingReceipt=z.strictObject({...receipt,result:binding}).refine(r=>r.result.organizationId===r.organizationId);

export const managedChannelPending=z.strictObject({
 key:z.uuid(),userId:routeFact,organizationId:routeFact,memberId:routeFact,
 operation:z.enum(["create","name","models"]),connectionId:connectionID.optional(),
}).refine(v=>v.operation==="create"?!v.connectionId:!!v.connectionId);
export type ManagedChannelPending=z.infer<typeof managedChannelPending>;
export type ManagedChannels=z.infer<typeof managedChannels>;
export type ManagedChannelReceipt=z.infer<typeof createChannelReceipt>|z.infer<typeof renameChannelReceipt>|z.infer<typeof editBindingReceipt>;
export function managedChannelPath(intent:Pick<ManagedChannelPending,"operation"|"connectionId">) {
 return "/managed-channels/"+(intent.operation==="create"?"create":intent.connectionId+"/"+intent.operation);
}
export function managedChannelRoute(path:string,method:string) {
 const parts=path.split("/").slice(2);
 if(method==="GET"&&parts.length===0)return {output:managedChannels};
 const create=parts[0]==="create";
 const operation=create?"create":parts[1];
 if(!create&&(!connectionID.safeParse(parts[0]).success||operation!=="models"&&operation!=="name"))return null;
 const length=create?1:2;
 const input=create?createChannel:operation==="name"?renameChannel:editBinding;
 const output=create?createChannelReceipt:operation==="name"?renameChannelReceipt:editBindingReceipt;
 if(method==="POST"&&parts.length===length)return {input,output};
 if(method==="GET"&&parts.length===length+2&&parts[length]==="receipts"&&routeFact.safeParse(parts[length+1]).success)return {output,receiptKey:parts[length+1]};
 return null;
}
