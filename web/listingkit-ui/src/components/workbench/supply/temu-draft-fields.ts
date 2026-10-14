import type {TemuDraftInput} from "@/lib/contracts/temu-model";

type Local=NonNullable<TemuDraftInput["local"]>;
type Supplier=NonNullable<TemuDraftInput["supplier"]>;
export function emptyTemuLocalSKU():NonNullable<Local["skuList"]>[number]{
 return {images:[],price:{basePrice:{amount:"",currency:""}},quantity:null,packageInfo:{weight:"",length:"",width:"",height:""},specDetails:[]};
}
export function emptyTemuSupplierSKU():NonNullable<NonNullable<Supplier["productSkcReqs"]>[number]["productSkuReqs"]>[number]{
 return {currencyType:"",extCode:"",thumbUrl:"",productSkuWhExtAttrReq:null,productSkuSpecReqs:[]};
}
export function initialTemuDraft(source:{title?:string|null;description?:string|null},mode:TemuDraftInput["mode"]):TemuDraftInput{
 const common={mode,selection:{parentCatId:"0",catId:"0"},packageUnits:{weight:"",dimension:""},images:[]};
 if(mode==="local")return {...common,local:{goodsBasic:{goodsName:source.title??"",goodsDesc:source.description??"",catId:"0"},goodsServicePromise:{shipmentLimitDay:null,costTemplateId:""},skuList:[emptyTemuLocalSKU()]}};
 return {...common,supplier:{productName:source.title??"",cat1Id:null,cat2Id:null,cat3Id:null,cat4Id:null,cat5Id:null,cat6Id:null,cat7Id:null,cat8Id:null,cat9Id:null,cat10Id:null,productSpecPropertyReqs:[],productPropertyReqs:[],materialImgUrl:"",productSkcReqs:[{extCode:"",productSkuReqs:[emptyTemuSupplierSKU()],mainProductSkuSpecReqs:[],previewImgUrls:[]}]}};
}
