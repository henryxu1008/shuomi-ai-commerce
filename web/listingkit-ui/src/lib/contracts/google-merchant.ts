import { z } from "zod";

const id = z.uuid();
const merchant = z.string().regex(/^[1-9][0-9]{0,18}$/);
const language = z.string().regex(/^[a-z]{2}$/);
const feed = z.string().regex(/^[A-Z0-9_-]{1,20}$/);
const page = z.string().max(4096).optional();
const list = <T extends z.ZodType>(item: T) => z.array(item).nullish().transform(value => value ?? []);
const status = z.object({status:z.enum(["connected","disconnected","expired","unavailable"]), accountId:merchant.optional(), dataSourceId:merchant.optional(), contentLanguage:language.optional(), feedLabel:feed.optional()});
const ok = z.object({ok:z.literal(true)});
const pair = z.strictObject({sourceId:id,planId:id});
const version = z.strictObject({version:z.number().int().positive()});
const countries = list(z.string().length(2));
const result = z.object({id,items:z.array(z.object({
  offerId:z.string().max(50), status:z.enum(["claimed","succeeded","failed_definitive","outcome_unknown","cancelled","not_sent"]),
  remoteProduct:z.string().optional(), observation:z.object({
    Product:z.string(),FreeListings:z.object({Approved:countries,Pending:countries,Disapproved:countries}),
    UnexpectedDestination:z.boolean(),Issues:list(z.object({Code:z.string(),Attribute:z.string(),Severity:z.string()})),
  }).optional(),
}))});
const offer = z.object({ID:z.string(),Title:z.string(),Description:z.string(),Link:z.url(),ImageLink:z.url(),Currency:z.string(),AmountMicros:z.number().int(),Availability:z.enum(["IN_STOCK","OUT_OF_STOCK"]),Condition:z.enum(["NEW","USED","REFURBISHED"]),Color:z.string(),Size:z.string()});
export const googleContracts = {
  status:{method:"GET",response:status},
  begin:{method:"POST",body:version,response:z.object({attemptId:id,authorizationUrl:z.url().refine(value=>{
    const url=new URL(value);return url.origin==="https://accounts.google.com"&&url.pathname==="/o/oauth2/auth";
  }),expiresAt:z.iso.datetime({offset:true})})},
  complete:{method:"POST",body:z.strictObject({attemptId:id,state:z.string().length(43),code:z.string().min(1).max(8192)}),response:ok},
  accounts:{method:"POST",body:z.strictObject({attemptId:id,page}),response:z.object({Accounts:list(z.object({ID:merchant,DisplayName:z.string()})),NextPageToken:z.string().max(4096)})},
  sources:{method:"POST",body:z.strictObject({attemptId:id,accountId:merchant,page}),response:z.object({Sources:list(z.object({ID:merchant,DisplayName:z.string(),ContentLanguage:z.string(),FeedLabel:z.string()})),NextPageToken:z.string().max(4096)})},
  bind:{method:"POST",body:z.strictObject({attemptId:id,accountId:merchant,dataSourceId:merchant,contentLanguage:language,feedLabel:feed,dedicatedSourceConfirmed:z.literal(true)}),response:status},
  disconnect:{method:"POST",body:version,response:ok},
  options:{method:"GET",response:z.object({title:z.string(),records:list(z.object({id,createdAt:z.iso.datetime({offset:true})})),variants:list(z.object({id:z.string(),label:z.string()})),images:list(z.object({id:z.string(),url:z.url()}))})},
  prepare:{method:"POST",body:z.strictObject({sourceId:id,storeId:id,imageId:z.string().min(1).max(200),variants:z.array(z.strictObject({
    variantId:z.string().min(1).max(512),link:z.url().max(2000),price:z.string().max(17),currency:z.string().regex(/^[A-Z]{3}$/),
    availability:z.enum(["IN_STOCK","OUT_OF_STOCK"]),condition:z.enum(["NEW","USED","REFURBISHED"]),color:z.string().max(100),size:z.string().max(100),
  })).min(1).max(10)}),response:z.object({id,sourceId:id,storeId:id,accountId:merchant,dataSourceId:merchant,contentLanguage:language,feedLabel:feed,version:z.string(),offers:z.array(offer).min(1).max(10),createdAt:z.iso.datetime({offset:true})})},
  confirm:{method:"POST",body:pair,response:result},
  result:{method:"POST",body:pair,response:result},
} as const;

export type GoogleAction = keyof typeof googleContracts;
export type GoogleRouteContract = `google-${GoogleAction}`;

export function googleUpstreamPath(action:GoogleAction,parts:string[]):string|null {
  if(action==="options")return parts.length===4&&parts[0]==="supply-preparations"&&parts[1]==="google"&&parts[2]===action&&id.safeParse(parts[3]).success?parts.join("/"):null;
  if(action==="prepare"||action==="confirm"||action==="result")return parts.length===3&&parts[0]==="supply-preparations"&&parts[1]==="google"&&parts[2]===action?parts.join("/"):null;
  return parts[0]==="stores"&&id.safeParse(parts[1]).success&&parts[2]==="connection"&&parts[3]==="google"&&
    (action==="status"?parts.length===4:parts.length===5&&parts[4]===action)?parts.join("/"):null;
}
