import {z} from "zod";

const text=(max:number)=>z.string().refine(value=>new TextEncoder().encode(value).length<=max&&!value.includes("\0"));
const amount=z.number().finite().min(0).max(Number.MAX_SAFE_INTEGER);
const completionStructuredSchema=z.object({
 attributes:z.array(z.object({name:text(200).refine(v=>v.trim().length>0),value:text(8192)}).strict()).max(256),
 variants:z.array(z.object({sourceIndex:z.number().int().min(0).max(255),price:z.object({currency:z.string().regex(/^[A-Z]{3}$/),amount}).strict().nullish().transform(v=>v??null),stock:amount.int().nullish().transform(v=>v??null)}).strict()).max(256),
}).strict().superRefine((value,ctx)=>{
 const names=new Set<string>();value.attributes.forEach((a,index)=>{const name=a.name.trim();if(names.has(name))ctx.addIssue({code:"custom",path:["attributes",index,"name"],message:"属性名称不能重复"});names.add(name)});
 value.variants.forEach((v,index)=>{if(v.sourceIndex!==index)ctx.addIssue({code:"custom",path:["variants",index,"sourceIndex"],message:"SKU必须对应固定来源顺序"})});
});
export const completionContentSchema=z.object({title:text(1024),description:text(32768),structured:completionStructuredSchema.nullish().transform(v=>v??undefined)}).strict();
export type CompletionStructured=z.infer<typeof completionStructuredSchema>;

// Go's existing input hash/size encoder escapes these JSON string characters.
export function completionPayloadBytes(input:unknown){return new TextEncoder().encode(JSON.stringify(input).replace(/[<>&\u2028\u2029]/g,c=>`\\u${c.charCodeAt(0).toString(16).padStart(4,"0")}`)).length}
export function completionContentEqual(left:unknown,right:unknown){
 const a=completionContentSchema.safeParse(left),b=completionContentSchema.safeParse(right);
 return a.success&&b.success&&JSON.stringify(a.data)===JSON.stringify(b.data);
}
