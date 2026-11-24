import {z} from "zod";

const id=z.uuid();
const number=z.string().regex(/^[1-9][0-9]{0,25}$/);
const date=z.iso.datetime({offset:true});
const attempt={attemptId:id};
const ok=z.object({ok:z.literal(true)});
const version=z.strictObject({version:z.number().int().positive()});
const choice=z.object({id:number,businessId:number,name:z.string()});
const pair=z.strictObject({sourceId:id,planId:id});
const connection=z.object({connected:z.boolean(),version:z.number().int().positive(),
  businessId:number.or(z.literal("")),catalogId:number.or(z.literal("")),whatsAppId:number.optional()});
const product=z.object({retailerId:z.string(),title:z.string(),description:z.string(),brand:z.string(),
  productUrl:z.url(),imageUrl:z.url(),priceCents:z.number().int().positive()});
const result=z.object({id,status:z.enum(["not_sent","claimed","succeeded","failed_definitive","outcome_unknown","cancelled"]),
  remoteId:number.optional(),observation:z.object({catalog:z.enum(["unverified","found"]),
    whatsApp:z.enum(["unverified","not_selected","matched","different"]),facebook:z.literal("unverified")}).optional()});
const emptyArray=<T extends z.ZodType>(item:T)=>z.array(item).nullish().transform(value=>value??[]);

export const metaContracts={
  status:{method:"GET",response:connection},
  begin:{method:"POST",body:version,response:z.object({attemptId:id,expiresAt:date,authorizationUrl:z.url().refine(value=>{
    const url=new URL(value);
    return url.protocol==="http:"&&url.hostname==="127.0.0.1"&&url.port!==""&&url.pathname==="/dialog/oauth"&&!url.username&&!url.password;
  })})},
  complete:{method:"POST",body:z.strictObject({...attempt,state:z.string().length(43),code:z.string().min(1).max(1024)}),response:ok},
  choices:{method:"POST",body:z.strictObject(attempt),response:z.object({catalogs:emptyArray(choice),accounts:emptyArray(choice)})},
  bind:{method:"POST",body:z.strictObject({...attempt,catalogId:number,whatsAppId:number.or(z.literal(""))}),response:connection},
  disconnect:{method:"POST",body:version,response:ok},
  options:{method:"GET",response:z.object({title:z.string(),brand:z.string(),
    variants:emptyArray(z.object({id:z.string(),label:z.string()})),images:emptyArray(z.object({id:z.string(),url:z.url()})),
    records:emptyArray(z.object({id,createdAt:date}))})},
  prepare:{method:"POST",body:z.strictObject({sourceId:id,storeId:id,variantId:z.string().min(1).max(512),
    imageId:z.string().min(1).max(200),brand:z.string().max(100),productUrl:z.url().max(2048),price:z.string().max(16)}),
    response:z.object({id,sourceId:id,storeId:id,catalogId:number,businessId:number,whatsAppId:number.optional(),variantLabel:z.string(),product,createdAt:date})},
  confirm:{method:"POST",body:pair,response:result},
  result:{method:"POST",body:pair,response:result},
} as const;

export type MetaAction=keyof typeof metaContracts;
export type MetaRouteContract=`meta-${MetaAction}`;
export type MetaResponse<A extends MetaAction>=z.output<(typeof metaContracts)[A]["response"]>;

export function metaUpstreamPath(action:MetaAction,parts:string[]):string|null {
  if(parts[0]==="supply-preparations"&&parts[1]==="meta-catalog"&&parts[2]===action){
    if(action==="options")return parts.length===4&&id.safeParse(parts[3]).success?parts.join("/"):null;
    if(action==="prepare"||action==="confirm"||action==="result")return parts.length===3?parts.join("/"):null;
    return null;
  }
  if(["options","prepare","confirm","result"].includes(action))return null;
  return parts[0]==="stores"&&id.safeParse(parts[1]).success&&parts[2]==="connection"&&parts[3]==="meta-catalog"&&
    (action==="status"?parts.length===4:parts.length===5&&parts[4]===action)?parts.join("/"):null;
}
