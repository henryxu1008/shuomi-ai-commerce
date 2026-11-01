import {z} from "zod";
import {reportID} from "./report-center";
import {savedSheinDiagnosticSchema,sheinDigestSchema} from "@/lib/api/shein-diagnostic";
import {sheinRecordListItemSchema} from "@/lib/api/shein-records";
const version=z.string().refine(v=>/^[1-9][0-9]{0,18}$/.test(v)&&BigInt(v)<=BigInt("9223372036854775807"));
const offlineInput=z.strictObject({product_key:z.string().min(1).max(128).refine(v=>v.trim()===v&&new TextEncoder().encode(v).length<=128),snapshot_version:version,store_id:reportID,country:z.literal("US"),language:z.literal("en"),action:z.enum(["save_draft","publish"])});
export const offlineCreate=z.strictObject({collectionItemId:reportID,input:offlineInput});
export const offlineIntent=offlineCreate.extend({key:reportID});
export const offlineReceipt=z.strictObject({record_id:reportID,input_hash:sheinDigestSchema,diagnostic_hash:sheinDigestSchema,diagnostic_status:z.enum(["ready","ready_with_warnings","blocked"]),diagnostic_only:z.literal(true)});
export const offlineSaved=z.strictObject({origin:z.literal("saved"),recordId:reportID,operationId:reportID,input:offlineInput,sourceVersion:sheinDigestSchema,createdAt:z.iso.datetime({offset:true}),diagnostic:savedSheinDiagnosticSchema}).refine(v=>v.input.action===v.diagnostic.action);
export const offlineList=z.strictObject({items:z.array(sheinRecordListItemSchema).max(100),next_cursor:z.string().regex(/^[A-Za-z0-9_-]{1,512}$/).nullable()}).refine(v=>new Set(v.items.map(i=>i.record_id)).size===v.items.length);
export type OfflineIntent=z.infer<typeof offlineIntent>;
export type OfflineSaved=z.infer<typeof offlineSaved>;
export function offlineEndpoint(url:URL,method:string):{suffix:string;output:z.ZodType;input?:typeof offlineCreate}|null{
 const prefix="/api/offline-shein-records";
 if(url.pathname!==prefix&&!url.pathname.startsWith(prefix+"/"))return null;
 const suffix=url.pathname.slice(prefix.length),list=suffix===""&&method==="GET";
 if(suffix.includes("%")||suffix.endsWith("/")&&suffix!==""||url.search.length>1024)return null;
 for(const [key,value]of url.searchParams){if(!list||url.searchParams.getAll(key).length!==1||!(key==="limit"&&/^[1-9][0-9]?$|^100$/.test(value)||key==="cursor"&&/^[A-Za-z0-9_-]{1,512}$/.test(value)))return null;}
 if(suffix==="")return method==="GET"?{suffix,output:offlineList}:method==="POST"?{suffix,output:offlineReceipt,input:offlineCreate}:null;
 if(method!=="GET")return null;
 const parts=suffix.slice(1).split("/");
 if(parts.length===1&&reportID.safeParse(parts[0]).success||parts.length===2&&parts[0]==="commands"&&reportID.safeParse(parts[1]).success)return{suffix,output:offlineSaved};
 return null;
}
