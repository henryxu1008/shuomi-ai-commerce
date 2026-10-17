import {z} from "zod";
import {MarketAPIError,marketIntentSchema,type MarketIntent} from "@/lib/api/supply-market";
import {marketID,marketReceiptSchema,type MarketCommand,type MarketReceipt} from "@/lib/contracts/supply-market";

export const selectedBatchLimit=20;
export const marketPendingLimit=1024*1024;
export type SelectedCommand=Extract<MarketCommand,{action:"submit_selected"}>;
export type Supply=SelectedCommand["supply"];
export type CommonSupply=Pick<Supply,"province"|"city"|"afterSaleNote">;
export type SelectedDraft={title:string;selection:SelectedCommand["selection"];supply:Supply;inheritLocation:boolean;inheritAfterSale:boolean;useCommonFiles:boolean;fileIds:string[]};
export const emptySupply=():Supply=>({stock:0,capacity:"",minimumQuantity:1,leadDays:0,province:"",city:"",priceNote:"",afterSaleNote:""});
export function finalCommand(draft:SelectedDraft,common:CommonSupply,commonFiles:string[]):SelectedCommand{
  return {action:"submit_selected",selection:draft.selection,supply:{...draft.supply,
    ...(draft.inheritLocation?{province:common.province,city:common.city}:{}),
    ...(draft.inheritAfterSale?{afterSaleNote:common.afterSaleNote}:{})},
    fileIds:[...new Set([...(draft.useCommonFiles?commonFiles:[]),...draft.fileIds])],disclosure:true};
}
export const selectionKey=(s:SelectedCommand["selection"])=>JSON.stringify([s.itemId,s.expectedRevision,s.originalPublicationId,s.originalVersion,s.effectiveVersion,s.applyId]);
export const selectedBatchSchema=z.object({
  kind:z.literal("selected_batch"),key:marketID,
  items:z.array(z.object({title:z.string().min(1).refine(s=>new TextEncoder().encode(s).length<=2000&&!s.includes("\0")),intent:marketIntentSchema.refine(i=>!i.admin&&i.command.action==="submit_selected")}).strict()).min(1).max(selectedBatchLimit),
}).strict().superRefine((batch,ctx)=>{
  const keys=new Set([batch.key]),ids=new Set<string>();
  for(const item of batch.items){
    const command=item.intent.command;
    if(command.action!=="submit_selected")continue;
    if(keys.has(item.intent.key)||ids.has(command.selection.itemId))ctx.addIssue({code:"custom",message:"Duplicate application identity"});
    keys.add(item.intent.key);ids.add(command.selection.itemId);
  }
  if(new TextEncoder().encode(JSON.stringify(batch)).length>marketPendingLimit)ctx.addIssue({code:"custom",message:"Pending applications exceed bound"});
});
export type SelectedBatch=z.infer<typeof selectedBatchSchema>;
export const marketPendingSchema=z.union([marketIntentSchema,selectedBatchSchema]);
export type MarketPending=MarketIntent|SelectedBatch;
export const isSelectedBatch=(i:MarketPending):i is SelectedBatch=>"kind" in i&&i.kind==="selected_batch";
export function freezeBatch(items:{title:string;command:SelectedCommand}[]):SelectedBatch{
  return selectedBatchSchema.parse({kind:"selected_batch",key:crypto.randomUUID(),items:items.map(item=>({title:item.title,intent:{key:crypto.randomUUID(),command:item.command,admin:false}}))});
}
export function selectedReceipt(value:MarketReceipt):MarketReceipt{
  const parsed=marketReceiptSchema.safeParse(value);
  if(!parsed.success||!parsed.data.recordId||parsed.data.revision<=0)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
  return parsed.data;
}
