import { z } from "zod";
import { channelAPIKey, channelBaseUrl, channelName, channelRegistration, channelTemplate, credentialInput, credentialReceipt, managedConnection } from "./channel-credentials";
import { modelOption, modelReference, routeFact } from "./model-route-pricing";

const uc = z.string().regex(/^uc-[0-9a-f-]{36}$/);
export const platformOwner = "platform-model-channel-owner";
const platformConnection = managedConnection.refine(value => value.organizationId === platformOwner && uc.safeParse(value.connectionId).success);
const publication = z.strictObject({connectionId: uc, priceId: z.union([routeFact, z.literal("")]), organizations: z.array(routeFact).max(128)});
const context = {organizationId: routeFact};
const receipt = {...context, actorId: routeFact, requestKey: routeFact};

export const currentTextChoice = z.strictObject({...context, view: z.strictObject({
 revision: routeFact, current: modelOption.optional(), options: z.array(modelOption).max(256),
 purposes: z.array(routeFact).max(128).optional(),
})});
export const selectTextChoice = z.strictObject({expectedRevision: routeFact, reference: modelReference, quote: routeFact, expectedMemberId: routeFact});
export const textChoiceReceipt = z.strictObject({...receipt, revision: routeFact, reference: modelReference});

export const platformChannelList = z.strictObject({...context, canWrite: z.boolean(),
 channels: z.array(z.strictObject({connection: platformConnection, publication: publication.optional(), revision: routeFact})).max(128),
 organizations: z.array(routeFact).max(128),
});
export const platformCreate = z.strictObject({expectedMemberId: routeFact, name: channelName, template: channelTemplate, baseUrl: channelBaseUrl.optional(), apiKey: channelAPIKey});
export const platformCreateReceipt = z.strictObject({...receipt,result:z.strictObject({registration:channelRegistration,connection:platformConnection})});
export const platformPublish = z.strictObject({expectedMemberId: routeFact, expectedRevision: routeFact, channel: publication});
export const platformPublishReceipt = z.strictObject({...receipt,revision:routeFact,result:publication});
export const platformCredential = z.union(credentialInput.options.map(option=>option.extend({expectedMemberId:routeFact})));
export const platformCredentialReceipt = z.strictObject({...receipt,result:credentialReceipt.shape.receipt.refine(r=>r.organizationId===platformOwner)});

export function selectionEndpoint(parts: string[], method: string) {
 const [family, first, second, third, fourth] = parts;
 const get = method === "GET", post = method === "POST";
 const length = parts.length;
 if (family === "enterprise" && first === "text-choice") {
  if (get && length === 2) return {output:currentTextChoice};
  if (post && length === 3 && second === "select") return {input:selectTextChoice,output:textChoiceReceipt};
  if (get && length === 4 && second==="receipts" && routeFact.safeParse(third).success) return {output:textChoiceReceipt,receiptKey:third};
 }
 if (family !== "platform-channels") return;
 if (get && length === 1) return {output:platformChannelList};
 if (post && length === 2 && first === "create") return {input:platformCreate,output:platformCreateReceipt};
 if (get && length === 4 && first==="create" && second==="receipts" && routeFact.safeParse(third).success) return {output:platformCreateReceipt,receiptKey:third};
 if (!uc.safeParse(first).success) return;
 if (post && length === 3 && second==="publish") return {input:platformPublish,output:platformPublishReceipt,connectionId:first};
 if (get && length === 4 && second==="receipts" && routeFact.safeParse(third).success) return {output:platformPublishReceipt,receiptKey:third,connectionId:first};
 if (post && length === 3 && second==="credentials") return {input:platformCredential,output:platformCredentialReceipt,connectionId:first};
 if (get && length === 5 && second==="credentials" && third==="receipts" && routeFact.safeParse(fourth).success)
  return {output:platformCredentialReceipt,receiptKey:fourth,connectionId:first};
}
