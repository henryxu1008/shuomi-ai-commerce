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
const validationIssue=z.object({
 field:z.string().regex(/^(candidate|title|description|brand|attributes|questions|rows|rows\[([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\]\.(attributes|evidence|sku|price|stock|image))$/),
 code:z.enum(["invalid_json","invalid_schema","invalid_value","empty_candidate","source_evidence","duplicate_variant","duplicate_sku","price_evidence","stock_evidence","image_reference"]),
}).strict();
export const intakeResult=z.object({requestKey:collectionID,phase:z.enum(["running","human_review_required","interrupted","stopped"]),stopReason:z.string().max(128).optional(),candidate:candidate.optional(),validationError:validationIssue.optional()}).strict();

export function intakeValidationMessage(issue:z.infer<typeof validationIssue>){
 const row=/^rows\[(\d+)\]\.(.+)$/.exec(issue.field);
 const fieldNames:Record<string,string>={candidate:"候选内容",title:"商品名称",description:"商品描述",brand:"品牌",attributes:"规格或属性",questions:"待确认项",rows:"SKU",evidence:"来源资料",sku:"SKU 编码",price:"价格或币种",stock:"库存",image:"图片"};
 const field=fieldNames[row?.[2]??issue.field]??"候选内容";
 const location=row?`第 ${Number(row[1])+1} 个 SKU 的${field}`:field;
 const reasons:Record<z.infer<typeof validationIssue>["code"],string>={
  invalid_json:"返回内容不是有效的商品候选 JSON。",invalid_schema:"返回字段与商品候选格式不符。",invalid_value:"内容格式或取值不符合要求。",empty_candidate:"返回内容为空。",
  source_evidence:"与原资料中的来源片段不一致。",duplicate_variant:"规格组合重复。",duplicate_sku:"SKU 编码重复。",price_evidence:"缺少明确且一致的价格或币种来源。",
  stock_evidence:"缺少明确且一致的库存数量来源；未填写不能作为 0。",image_reference:"引用了未提供或无法确认归属的图片。",
 };
 return `${location}：${reasons[issue.code]}请核对资料后重新整理。`;
}

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
