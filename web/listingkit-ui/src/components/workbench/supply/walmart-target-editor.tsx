"use client";
import {DescriptionProjectionNote} from "../description-view";
import {descriptionPlainText} from "@/lib/contracts/product-description";
import {useEffect,useState} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Textarea} from "@/components/ui/textarea";
import {Select} from "@/components/ui/select";
import {walmartSupplyRules,supplyInventory,readSupplyCompletionDraft,SupplyAPIError,type SupplyScope} from "@/lib/api/supply-chain";
import {walmartTargetInputSchema,walmartCatalogSchema,inventorySchema,type WalmartSupplyTarget,type SupplySourceDetail,type SupplyCompletionDraft,type SupplyPublication} from "@/lib/contracts/supply-chain";
import {decodeWalmartDocument,type WalmartDraftInput} from "@/lib/contracts/walmart-model";
import type {SupplyCommandState} from "./use-supply-command";

type Props={scope:SupplyScope;source:SupplySourceDetail;storeId:string;existing?:WalmartSupplyTarget;command:SupplyCommandState;disabled:boolean;saved:number;completionAvailable?:boolean;publication?:SupplyPublication};
const sourceKey=(v:NonNullable<SupplySourceDetail["product"]["variants"]>[number])=>v.source_id||v.sku||"";
export function WalmartTargetEditor({scope,source,storeId,existing,command,disabled,saved,completionAvailable=false,publication}:Props){
 const version=existing?.effectiveVersion??source.source.source.version;
 const [draft,setDraft]=useState<WalmartDraftInput>(()=>existing?structuredClone(existing.input.walmart):{selection:{feedType:"MP_ITEM_MATCH",version:"",productType:""},skus:(source.product.variants??[]).map(v=>({sourceSKUKey:sourceKey(v),identifier:{productIdType:"GTIN",productId:v.barcode??""},condition:"",price:"",shippingWeight:""}))});
 const [attributeErrors,setAttributeErrors]=useState<Record<string,boolean>>({});
 const [catalog,setCatalog]=useState<z.infer<typeof walmartCatalogSchema>>();
 const [requirements,setRequirements]=useState<unknown>();const [busy,setBusy]=useState(false);const [error,setError]=useState("");
 const [inventory,setInventory]=useState<z.infer<typeof inventorySchema>>();const [selectedImages,setSelectedImages]=useState<string[]>([]);
 const [completion,setCompletion]=useState<SupplyCompletionDraft>();const [adopted,setAdopted]=useState(existing?.input.completionDraft);
 const selection={itemId:source.source.id,originalPublicationId:source.source.source.publicationId,originalSnapshotVersion:source.source.source.version,effectiveCatalogVersion:version,targetPlatform:"walmart" as const,...(existing?.applyReceiptId?{applyReceiptId:existing.applyReceiptId}:{})};
 const base={sourceId:source.source.id,storeId,expectedRevision:existing?.revision??0,effectiveVersion:version,...(existing?.applyReceiptId?{applyReceiptId:existing.applyReceiptId}:{})};
 const locked=disabled||busy||command.busy||!!command.pending;
 const repairResults=(publication?.walmartResults??[]).filter(v=>(v.repairableSKUs?.length??0)>0);


 useEffect(()=>{const c=new AbortController();let live=true;
  void supplyInventory(scope,selection,c.signal).then(v=>{if(live)setInventory(v)}).catch(e=>{if(live&&(!(e instanceof SupplyAPIError)||e.code!=="NOT_FOUND"))setError("尚未取得该版本的已批准图片。匹配新商品可先填写资料，新建商品须批准图片。")});
  if(completionAvailable)void readSupplyCompletionDraft(scope,source.source.id,c.signal).then(v=>{if(live)setCompletion(v)}).catch(()=>undefined);
  return()=>{live=false;c.abort()};
 // The immutable source/version is keyed by the parent editor session.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[scope,source.source.id,version,saved,completionAvailable]);
 function changeSKU(index:number,edit:(sku:WalmartDraftInput["skus"][number])=>void){setDraft(old=>{const next=structuredClone(old);edit(next.skus[index]!);return next})}
 async function loadRules(catalogOnly:boolean){setBusy(true);setError("");try{
  const result=await walmartSupplyRules(scope,{...base,walmartRules:{...draft.selection,...catalogOnly?{version:"",productType:""}:{}}});
  if("walmartCatalog" in result){setCatalog(result.walmartCatalog);setDraft(v=>({...v,selection:{...v.selection,version:result.walmartCatalog.version}}));setRequirements(undefined)}
  else{setRequirements(decodeWalmartDocument(result.walmart.schema))}
 }catch(e){setError(e instanceof SupplyAPIError?e.code:"未取得当前商家的完整官方规则，请刷新后重试。")}finally{setBusy(false)}}
 function save(){const checked=walmartTargetInputSchema.safeParse({...base,walmart:draft,...adopted?{completionDraft:adopted}:{}});if(!checked.success){setError("请补齐刊登方式、官方类目、逐 SKU 标识、价格和重量。");return}command.execute("save-target",checked.data)}
 function adopt(){if(!completion)return;setDraft(v=>({...v,skus:v.skus.map(s=>({...s,visible:{...s.visible,productName:completion.content.title,shortDescription:descriptionPlainText(completion.content.description,completion.content.descriptionFormat)}}))}));setAdopted({recordId:completion.id,revision:completion.revision})}
 return <div className="space-y-5"><DescriptionProjectionNote format={completion?.content.descriptionFormat??source.product.description_format}/>
  <p className="font-semibold">Walmart Marketplace · 美国站</p>
  <p className="text-sm text-muted-foreground">先选择匹配已有目录或提交新商品，保存后检查逐 SKU 资料，再在供应链明确批准刊登。匹配失败后需重新选择并批准新商品资料。</p>
  {error?<p role="alert" className="rounded-lg bg-red-50 p-3 text-sm">{error}</p>:null}
  <label className="grid gap-1 text-sm">刊登方式<Select value={draft.selection.feedType} disabled={locked} onChange={e=>{setDraft(v=>({...v,selection:{feedType:e.target.value as WalmartDraftInput["selection"]["feedType"],version:"",productType:""},skus:v.skus.map(s=>({...s,visible:e.target.value==="MP_ITEM"?{productName:source.product.title??"",shortDescription:descriptionPlainText(source.product.description??"",source.product.description_format)}:undefined}))}));setCatalog(undefined);setRequirements(undefined);setAdopted(undefined);setAttributeErrors({})}}><option value="MP_ITEM_MATCH">匹配 Walmart 已有目录商品</option><option value="MP_ITEM">提交新商品资料（卖家配送）</option></Select></label>
  <Button variant="outline" disabled={locked} onClick={()=>void loadRules(true)}>读取当前可用类目</Button>
  {draft.selection.feedType==="MP_ITEM"?<label className="grid gap-1 text-sm">商品类型<Select value={draft.selection.productType} disabled={locked||!catalog} onChange={e=>{setDraft(v=>({...v,selection:{...v.selection,productType:e.target.value}}));setRequirements(undefined)}}><option value="">选择官方商品类型</option>{catalog?.productTypes.map(pt=><option key={pt} value={pt}>{pt}</option>)}</Select></label>:null}
  <p className="text-xs text-muted-foreground">规则版本：{draft.selection.version||"尚未读取"}</p><Button variant="outline" disabled={locked||draft.selection.feedType==="MP_ITEM"&&!draft.selection.productType||!draft.selection.version} onClick={()=>void loadRules(false)}>查看本类目必填要求</Button>
  {requirements?<details className="rounded-lg border p-3"><summary>官方字段、允许值与条件要求</summary><pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap text-xs">{JSON.stringify(requirements,null,2)}</pre></details>:null}
  {!draft.skus.length||draft.skus.some(s=>!s.sourceSKUKey)?<p role="alert">原商品缺少可稳定识别的 SKU，请先在原商品中补齐。</p>:draft.skus.map((s,i)=><fieldset key={s.sourceSKUKey} className="space-y-3 rounded-xl border p-4" disabled={locked}>
   <legend className="px-2 text-sm font-medium">来源 SKU：{s.sourceSKUKey}</legend>
   <div className="grid gap-3 sm:grid-cols-2"><label className="grid gap-1 text-sm">商品标识类型<Select value={s.identifier.productIdType} onChange={e=>changeSKU(i,s=>{s.identifier.productIdType=e.target.value})}>{["GTIN","UPC","EAN","ISBN","WPID"].map(v=><option key={v}>{v}</option>)}</Select></label><label className="grid gap-1 text-sm">真实商品标识<Input value={s.identifier.productId} onChange={e=>changeSKU(i,s=>{s.identifier.productId=e.target.value})}/></label>
   <label className="grid gap-1 text-sm">商品成色<Input placeholder="按官方允许值填写，例如 New" value={s.condition} onChange={e=>changeSKU(i,s=>{s.condition=e.target.value})}/></label><label className="grid gap-1 text-sm">首次报价（USD）<Input inputMode="decimal" value={s.price} onChange={e=>changeSKU(i,s=>{s.price=e.target.value})}/></label><label className="grid gap-1 text-sm">运输重量（按规格单位）<Input inputMode="decimal" value={s.shippingWeight} onChange={e=>changeSKU(i,s=>{s.shippingWeight=e.target.value})}/></label></div>
   {draft.selection.feedType==="MP_ITEM"?<><label className="grid gap-1 text-sm">商品标题<Input value={String(s.visible?.productName??"")} onChange={e=>{setAdopted(undefined);changeSKU(i,s=>{s.visible={...s.visible,productName:e.target.value}})}}/></label><label className="grid gap-1 text-sm">描述<Textarea value={String(s.visible?.shortDescription??"")} onChange={e=>{setAdopted(undefined);changeSKU(i,s=>{s.visible={...s.visible,shortDescription:e.target.value}})}}/></label><label className="grid gap-1 text-sm">已批准主图<Select value={String(s.visible?.mainImageUrl??"")} onChange={e=>changeSKU(i,s=>{s.visible={...s.visible,mainImageUrl:e.target.value}})}><option value="">选择本商品已批准的图片</option>{inventory?.assets.map(a=><option key={a.id} value={a.url}>{a.role} · {a.id}</option>)}</Select></label><AttributeEditor label="本类目展示属性（品牌、规格、合规等）" value={Object.fromEntries(Object.entries(s.visible??{}).filter(([key])=>!["productName","shortDescription","mainImageUrl"].includes(key)))} onInvalid={invalid=>setAttributeErrors(v=>({...v,[`visible:${i}`]:invalid}))} onChange={value=>{changeSKU(i,s=>{s.visible={productName:s.visible?.productName??"",shortDescription:s.visible?.shortDescription??"",mainImageUrl:s.visible?.mainImageUrl??"",...value}})}}/></>:null}
   <AttributeEditor label={draft.selection.feedType==="MP_ITEM_MATCH"?"匹配与报价的其他必需属性":"报价与履约的其他必需属性"} value={s.attributes??{}} onInvalid={invalid=>setAttributeErrors(v=>({...v,[`attributes:${i}`]:invalid}))} onChange={value=>changeSKU(i,s=>{s.attributes=value})}/>
  </fieldset>)}
  {completion&&draft.selection.feedType==="MP_ITEM"?<div className="space-y-2 rounded-xl border p-3"><p className="text-sm">可采用本地候选的标题和描述；属性、价格、库存候选仍保存在原草稿，需在此逐项填写核实。</p><Button variant="outline" disabled={locked} onClick={adopt}>采用已保存标题和描述</Button></div>:null}
  {draft.selection.feedType==="MP_ITEM"?<details className="space-y-3 rounded-xl border p-3"><summary>批准原商品图片</summary>{source.images.map(image=><label key={image.id} className="flex items-center gap-2 text-sm"><input type="checkbox" checked={selectedImages.includes(image.id)} disabled={locked} onChange={e=>setSelectedImages(v=>e.target.checked?[...v,image.id]:v.filter(id=>id!==image.id))}/><span className="break-all">{image.url}</span></label>)}<Button disabled={locked||!selectedImages.length} onClick={()=>command.execute("approve",{selection,images:selectedImages.map((id,i)=>({id,role:i===0?"main":"gallery"})),approved:[]})}>批准所选图片</Button></details>:null}
  {existing?<section className="space-y-2 rounded-xl border p-3"><p className="text-sm">已保存目标版本 {existing.revision} · {existing.walmart.selection.feedType}</p>{existing.walmart.skus.map(s=><p key={s.sellerSKU} className="break-all text-xs">{s.sourceSKUKey} → {s.sellerSKU}</p>)}<p className="break-all text-xs">资料指纹：{existing.walmart.wireHash}</p><details><summary>查看已保存的准确刊登资料</summary><pre className="max-h-72 overflow-auto whitespace-pre-wrap text-xs">{JSON.stringify(decodeWalmartDocument(existing.walmart.payload),null,2)}</pre></details></section>:null}
  {repairResults.map(result=><section key={result.id} className="space-y-2 rounded-xl border p-3"><p className="text-sm">Feed {result.observation.feedId} 已确认以下 SKU 未接纳。修复后保存并明确批准所选失败项。</p>{result.repairableSKUs?.map(sku=><label key={sku} className="flex items-center gap-2 text-xs"><input type="checkbox" disabled={locked} checked={draft.repairFrom===result.receiptId&&!!draft.repairSKUs?.includes(sku)} onChange={e=>setDraft(v=>({...v,repairFrom:result.receiptId,repairSKUs:e.target.checked?[...(v.repairFrom===result.receiptId?v.repairSKUs??[]:[]).filter(s=>s!==sku),sku]:(v.repairSKUs??[]).filter(s=>s!==sku)}))}/>{sku}</label>)}</section>)}
  <Button disabled={locked||Object.values(attributeErrors).some(Boolean)||!draft.skus.length||!draft.selection.version||draft.selection.feedType==="MP_ITEM"&&!draft.selection.productType} onClick={save}>校验并保存 Walmart 草稿</Button><p className="text-xs text-muted-foreground">保存只生成本地草稿，不会提交 Walmart。GTIN、品牌和合规资格必须真实，不会自动申请豁免。</p>
 </div>
}
function AttributeEditor({label,value,onChange,onInvalid}:{label:string;value:Record<string,unknown>;onInvalid:(invalid:boolean)=>void;onChange:(value:Record<string,z.infer<ReturnType<typeof z.json>>>)=>void}){
 const [raw,setRaw]=useState(()=>JSON.stringify(value,null,2));const [invalid,setInvalid]=useState(false);
 return <label className="grid gap-1 text-sm">{label}<Textarea value={raw} rows={5} onChange={e=>{setRaw(e.target.value);const parsed=z.record(z.string(),z.json()).safeParse((()=>{try{return JSON.parse(e.target.value)}catch{return null}})());setInvalid(!parsed.success);onInvalid(!parsed.success);if(parsed.success)onChange(parsed.data)}}/>{invalid?<span role="alert">请输入有效的属性 JSON 对象；当前错误内容尚未采用。</span>:null}</label>
}
