import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
import type {TikTokDraftInput,TikTokSKU} from "@/lib/contracts/tiktokshop-model";

export function emptyTikTokSKU(global:boolean):TikTokSKU {
 return {sellerSku:"",salePrice:"",inventory:[],salesAttributes:[],...(global?{globalBasePrice:"",globalInventory:[]}:{})};
}
export function initialTikTokDraft(source:Pick<SupplySourceDetail["product"],"title"|"description">&{variants?:unknown[]},method:TikTokDraftInput["selection"]["method"]):TikTokDraftInput {
 return {selection:{method,categoryId:""},title:source.title??"",description:source.description??"",saveMode:"",mainImages:[],skus:[emptyTikTokSKU(method==="GLOBAL_PUBLISHING")],packageWeight:{value:"",unit:"KILOGRAM"},attributes:[],certifications:[]};
}
export function carryTikTokCompletion(draft:TikTokDraftInput,record:{id:string;revision:number;content:{title:string;description?:string}}){
 return {draft:{...structuredClone(draft),title:record.content.title,description:record.content.description??""},selection:{recordId:record.id,revision:record.revision}};
}
