"use client";
import {DescriptionProjectionNote} from "../description-view";
import {descriptionPlainText} from "@/lib/contracts/product-description";
import {TemuLocalAttributes} from "./temu-local-attributes";
import {TemuSupplierFields} from "./temu-supplier-fields";
import {useEffect,useRef,useState,type ReactNode} from "react";
import Image from "next/image";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {temuSupplyRules,supplyInventory,readSupplyCompletionDraft,SupplyAPIError,type SupplyScope} from "@/lib/api/supply-chain";
import {temuTargetInputSchema,inventorySchema,temuRulesSchema,type SupplySourceDetail,type TemuSupplyTarget,type SupplyCompletionDraft} from "@/lib/contracts/supply-chain";
import type {TemuDraftInput} from "@/lib/contracts/temu-model";
import type {SupplyCommandState} from "./use-supply-command";
import {initialTemuDraft,emptyTemuLocalSKU} from "./temu-draft-fields";

type Rules=z.infer<typeof temuRulesSchema>;
type Inventory=z.infer<typeof inventorySchema>;
type Local=NonNullable<TemuDraftInput["local"]>;
type Supplier=NonNullable<TemuDraftInput["supplier"]>;
type LocalSKU=NonNullable<Local["skuList"]>[number];
const modes={local:"本土",fully_managed:"全托管",semi_managed:"半托管"};
const catKeys=["cat1Id","cat2Id","cat3Id","cat4Id","cat5Id","cat6Id","cat7Id","cat8Id","cat9Id","cat10Id"] as const;
const rootSelection={parentCatId:"0",catId:"0"};

export function TemuTargetEditor({scope,source,storeId,existing,command,disabled,saved,completionAvailable=false}:{scope:SupplyScope;source:SupplySourceDetail;storeId:string;existing?:TemuSupplyTarget;command:SupplyCommandState;disabled:boolean;saved:number;completionAvailable?:boolean}){
 const [draft,setDraft]=useState<TemuDraftInput|undefined>(()=>existing?structuredClone(existing.input.temu):undefined);
 const [modeChecked,setModeChecked]=useState(!existing);
 const [selection,setSelection]=useState(existing?.input.temu.selection??{parentCatId:"0",catId:"0"});
 const [rules,setRules]=useState<Rules>(),[inventory,setInventory]=useState<Inventory>(),[error,setError]=useState<string>();
 const [loadedRequest,setLoadedRequest]=useState<string>(),[refresh,setRefresh]=useState(0),[completion,setCompletion]=useState<SupplyCompletionDraft>();
 const [selectedImages,setSelectedImages]=useState<Set<string>>(new Set());
 const [parents,setParents]=useState<Array<{id:string;name:string}>>(()=>existing?.input.temu.mode!=="local"?existing?.temuRuleSnapshot?.supplierCategoryPath?.slice(0,-1).map(c=>({id:c.catId,name:c.catName}))??[]:[]);
 const current=useRef(draft);useEffect(()=>{current.current=draft},[draft]);
 const version=existing?.effectiveVersion??source.source.source.version,applied=existing?.applyReceiptId;
 const requestedSelection=modeChecked?selection:rootSelection;
 const requestKey=JSON.stringify([scope,source.source.id,storeId,existing?.revision??0,version,applied,requestedSelection,refresh]);
 const loading=loadedRequest!==requestKey;
 const provenance={itemId:source.source.id,originalPublicationId:source.source.source.publicationId,originalSnapshotVersion:source.source.source.version,effectiveCatalogVersion:version,...applied?{applyReceiptId:applied}:{},targetPlatform:"temu" as const};
 useEffect(()=>{const c=new AbortController();let live=true;
  void temuSupplyRules(scope,{sourceId:source.source.id,storeId,expectedRevision:existing?.revision??0,effectiveVersion:version,...applied?{applyReceiptId:applied}:{},temuRules:requestedSelection},c.signal).then(value=>{if(!live)return;setRules(value);const mode=value.temu.mode;
   if(current.current&&current.current.mode!==mode){setError("店铺模式已变化。请按当前模式重新准备，共同资料将保留为待校验草稿，价格、币种、站点、库存和半托承诺需重新填写。");return}
   setModeChecked(true);
   setDraft(old=>old??initialTemuDraft({title:source.product.title,description:source.product.description,description_format:source.product.description_format},mode));setError(undefined);
  }).catch(e=>{if(live)setError(code(e))}).finally(()=>{if(live)setLoadedRequest(requestKey)});
  return()=>{live=false;c.abort()};
 },[scope,source,storeId,existing,version,applied,requestedSelection,refresh,requestKey]);
 useEffect(()=>{const c=new AbortController();let live=true;void supplyInventory(scope,provenance,c.signal).then(v=>{if(live)setInventory(v)}).catch(e=>{if(live)setError(code(e))});return()=>{live=false;c.abort()};
 // Immutable source provenance remains exact throughout this editor session.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[scope,source,version,applied,saved]);
 useEffect(()=>{if(!completionAvailable)return;const c=new AbortController();let live=true;void readSupplyCompletionDraft(scope,source.source.id,c.signal).then(v=>{if(live)setCompletion(v)}).catch(e=>{if(live&&(!(e instanceof SupplyAPIError)||e.code!=="NOT_FOUND"))setError(code(e))});return()=>{live=false;c.abort()};},[completionAvailable,scope,source.source.id]);
 function change(edit:(value:TemuDraftInput)=>void){setDraft(old=>{if(!old)return old;const copy=structuredClone(old);edit(copy);return copy})}
 function resetCurrentMode(){
  if(!rules||!draft||disabled||loading)return;
  const mode=rules.temu.mode;
  const next=initialTemuDraft({title:draft.supplier?.productName??draft.local?.goodsBasic.goodsName,description:draft.local?.goodsBasic.goodsDesc},mode);
  if(draft.supplier&&next.supplier){
   next.supplier=structuredClone(draft.supplier);next.images=structuredClone(draft.images);next.packageUnits=structuredClone(draft.packageUnits);
   next.selection={...draft.selection,supplierSiteId:"0"};
   delete next.supplier.productSemiManagedReq;delete next.supplier.productShipmentReq;
   for(const skc of next.supplier.productSkcReqs??[])for(const sku of skc.productSkuReqs??[]){sku.currencyType="";delete sku.supplierPrice;delete sku.siteSupplierPrices;delete sku.productSkuStockQuantityReq}
  }
  current.current=next;setDraft(next);setSelection(next.selection);setParents(next.supplier?parents:[]);setRules(undefined);setError(undefined);setModeChecked(true);setRefresh(n=>n+1);
 }
 function local(edit:(value:Local)=>void){change(v=>{if(v.local)edit(v.local)})}
 function supplier(edit:(value:Supplier)=>void){change(v=>{if(v.supplier)edit(v.supplier)})}
 function sizePage(offset:string){const next={...selection,supplierSizeOffset:offset};setSelection(next);change(v=>{v.selection=next})}
 function sizeSelection(id:string,checked:boolean){
  if(!draft?.supplier||!rules)return;
  const ids=checked?[...draft.supplier.sizeTemplateIds??[],id]:draft.supplier.sizeTemplateIds?.filter(value=>value!==id)??[];
  const pages=[rules.temu.supplierSizeCharts,...rules.temu.supplierSizeChartPages??[]].filter(p=>!!p);
  const offsets=new Set<string>();
  for(const page of pages)if(page.offset&&page.offset!=="0"&&page.sizeSpecDataList?.some(c=>!!c.businessId&&ids.includes(c.businessId)))offsets.add(page.offset);
  if(ids.some(id=>!pages.some(p=>p.sizeSpecDataList?.some(c=>c.businessId===id))))for(const offset of selection.supplierSizeOffsets??[])if(offset!=="0")offsets.add(offset);
  if(offsets.size>16){setError("所选尺码表超出当前有界读取页数，请移除不需要的表。");return}
  const ordered=Array.from(offsets).sort((a,b)=>BigInt(a)<BigInt(b)?-1:1);
  const next={...selection,supplierSizeOffsets:Array.from({length:16},(_,i)=>ordered[i]??"0")};setSelection(next);change(v=>{v.selection=next;if(v.supplier)v.supplier.sizeTemplateIds=ids});
 }
 function pickCategory(id:string){if(!rules||!draft)return;const categories=draft.mode==="local"?rules.temu.localCategories:rules.temu.supplierCategories;
  const item=categories?.find(c=>c.catId===id);if(!item)return;
  const leaf="leaf" in item?item.leaf:item.isLeaf;
  const path=[...parents.map(p=>p.id),id];const next={...selection,supplierSizeOffset:"0",parentCatId:leaf?selection.parentCatId:id,catId:leaf?id:"0",...(draft.mode!=="local"?{supplierCategoryPath:Array.from({length:10},(_,i)=>path[i]??"0")}: {})};setSelection(next);
  change(v=>{v.selection=next;if(v.local)v.local.goodsBasic.catId=next.catId;if(v.supplier){const path=[...parents.map(p=>p.id),id];catKeys.forEach((k,i)=>{v.supplier![k]=path[i]??"0"})}});
  if(!leaf)setParents(old=>[...old,{id,name:item.catName}]);
 }
 function backCategory(){const nextParents=parents.slice(0,-1),next={...selection,parentCatId:nextParents.at(-1)?.id??"0",catId:"0",...(draft?.mode!=="local"?{supplierCategoryPath:Array.from({length:10},(_,i)=>nextParents[i]?.id??"0")}: {})};setParents(nextParents);setSelection(next);change(v=>{v.selection=next;if(v.local)v.local.goodsBasic.catId="0";if(v.supplier)catKeys.forEach((k,i)=>{v.supplier![k]=nextParents[i]?.id??"0"})});}
 function freight(id:string){const next={...selection,costTemplateId:id};setSelection(next);change(v=>{v.selection=next;if(v.local)v.local.goodsServicePromise.costTemplateId=id});}
 function save(){const parsed=temuTargetInputSchema.safeParse({sourceId:source.source.id,storeId,expectedRevision:existing?.revision??0,effectiveVersion:version,...applied?{applyReceiptId:applied}:{},temu:draft});if(!parsed.success){setError("请检查编号、数字格式与资料数量；未填数值请留空。");return}command.execute("save-target",parsed.data)}
 const target=rules?.temu.localTarget,spec=target?.specification,categories=draft?.mode==="local"?rules?.temu.localCategories:rules?.temu.supplierCategories;
 const unavailable=!!target?.issues?.length||rules?.temu.issues.includes("site_specification_unverified");
 return <div className="space-y-6"><DescriptionProjectionNote format={completion?.content.descriptionFormat??source.product.description_format}/>
  {error?<p role="alert" className="text-sm text-red-700">{error}</p>:null}
  {rules&&draft&&rules.temu.mode!==draft.mode?<Button variant="outline" disabled={disabled||loading} onClick={resetCurrentMode}>按当前模式重新准备</Button>:null}
  <div className="flex flex-wrap items-center gap-3"><p className="font-semibold">TEMU · {draft?modes[draft.mode]:"正在核实店铺"}{target?.region?.label?` · ${target.region.label}`:""}</p><Button size="sm" variant="outline" disabled={loading} onClick={()=>{setModeChecked(false);setRefresh(n=>n+1)}}>刷新官方规则</Button></div>
  {unavailable?<p className="rounded-lg bg-amber-50 p-3 text-sm">站点规格尚未核实。可以保存商品资料，发布入口保持关闭。</p>:null}
  {draft?.mode!=="local"&&draft?<p className="text-sm text-amber-700">全托管与半托管使用各自的供货价和站点规则。资料或规则缺失时会显示原因，并保持待补全。</p>:null}
  {completion&&draft?<p className="text-sm text-muted-foreground">仅带入标题和描述，候选属性、价格和库存尚未带入。</p>:null}
  {completion&&draft?<Button variant="outline" disabled={disabled} onClick={()=>change(v=>{if(v.local){v.local.goodsBasic.goodsName=completion.content.title;v.local.goodsBasic.goodsDesc=descriptionPlainText(completion.content.description??"",completion.content.descriptionFormat)}if(v.supplier)v.supplier.productName=completion.content.title})}>带入待补全草稿 R{completion.revision}</Button>:null}
  {draft?<><Field label="商品标题"><Input disabled={disabled} value={draft.local?.goodsBasic.goodsName??draft.supplier?.productName??""} onChange={e=>change(v=>{if(v.local)v.local.goodsBasic.goodsName=e.target.value;if(v.supplier)v.supplier.productName=e.target.value})}/></Field>
   {draft.local?<Field label="商品描述"><Textarea disabled={disabled} value={draft.local.goodsBasic.goodsDesc??""} onChange={e=>local(v=>{v.goodsBasic.goodsDesc=e.target.value})}/></Field>:null}
   <section className="space-y-3"><h3 className="font-semibold">官方类目</h3><p className="text-xs text-muted-foreground">{parents.map(p=>p.name).join(" / ")||"顶级类目"}{selection.catId!=="0"?` · 已选 ${categories?.find(c=>c.catId===selection.catId)?.catName??selection.catId}`:""}</p><Select aria-label="官方类目" value={selection.catId==="0"?"":selection.catId} disabled={disabled||loading} onChange={e=>pickCategory(e.target.value)}><option value="">选择类目</option>{categories?.map(c=><option key={c.catId} value={c.catId}>{c.catName}{("leaf" in c?c.leaf:c.isLeaf)?"":" /"}</option>)}</Select>{parents.length?<Button size="sm" variant="outline" disabled={disabled||loading} onClick={backCategory}>返回上一级</Button>:selection.parentCatId!=="0"?<Button size="sm" variant="outline" disabled={disabled||loading} onClick={backCategory}>返回顶级类目</Button>:null}</section>
   {draft.local?<>
    <div className="grid gap-3 sm:grid-cols-2"><Field label="发货时限"><Select disabled={disabled} value={draft.local.goodsServicePromise.shipmentLimitDay??""} onChange={e=>local(v=>{v.goodsServicePromise.shipmentLimitDay=e.target.value||null})}><option value="">选择实际承诺</option><option value="1">1 个工作日</option><option value="2">2 个工作日</option></Select></Field><Field label="运费模板"><Select disabled={disabled||loading} value={selection.costTemplateId??""} onChange={e=>freight(e.target.value)}><option value="">选择当前店铺模板</option>{rules?.temu.freightTemplates?.map(t=><option key={t.templateId} value={t.templateId}>{t.templateName}</option>)}</Select></Field>
    <Field label="包装重量单位"><Select disabled={disabled} value={draft.packageUnits.weight} onChange={e=>change(v=>{v.packageUnits.weight=e.target.value})}><option value="">选择单位</option>{spec?.weight?.unit?<option value={spec.weight.unit}>{spec.weight.unit}</option>:null}</Select></Field><Field label="包装尺寸单位"><Select disabled={disabled} value={draft.packageUnits.dimension} onChange={e=>change(v=>{v.packageUnits.dimension=e.target.value})}><option value="">选择单位</option>{spec?.dimension?.unit?<option value={spec.dimension.unit}>{spec.dimension.unit}</option>:null}</Select></Field>
    <Field label="商品外部编号"><Input disabled={disabled} value={draft.local.goodsBasic.externalGoodsId??""} onChange={e=>local(v=>{v.goodsBasic.externalGoodsId=e.target.value})}/></Field></div>
    <TemuLocalAttributes rules={rules?.temu} values={draft.local.goodsProperty??[]} disabled={disabled} onChange={values=>local(v=>{v.goodsProperty=values})}/>
    <section className="space-y-4"><h3 className="font-semibold">SKU、价格与包装</h3>{draft.local.skuList?.map((sku,i)=><LocalSKUFields key={i} index={i} sku={sku} rules={rules} disabled={disabled} onChange={edit=>local(v=>{if(v.skuList?.[i])edit(v.skuList[i])})} onRemove={()=>local(v=>{v.skuList=v.skuList?.filter((_,j)=>j!==i)??[]})}/>)}<Button variant="outline" disabled={disabled} onClick={()=>local(v=>{v.skuList=[...v.skuList??[],emptyTemuLocalSKU()]})}>添加 SKU</Button></section>
   </>:<TemuSupplierFields draft={draft} rules={rules?.temu} disabled={disabled||loading} onChange={supplier} onSizePage={sizePage} onSizeSelection={sizeSelection}/>}
   <section className="space-y-3"><h3 className="font-semibold">来源图片与批准素材</h3><p className="text-sm text-muted-foreground">先确认本次完整来源图选择，再为商品或 SKU 选择已批准的素材。</p><div className="grid gap-3 sm:grid-cols-3">{source.images.map((image,i)=><label key={image.id} className="space-y-2 rounded-lg border p-3"><Image src={image.url} alt="" width={120} height={120} unoptimized/><span className="flex gap-2 text-sm"><input type="checkbox" aria-label={`使用来源图 ${i+1}`} checked={selectedImages.has(image.id)} disabled={disabled} onChange={e=>setSelectedImages(old=>{const n=new Set(old);if(e.target.checked)n.add(image.id);else n.delete(image.id);return n})}/>来源图 {i+1}</span></label>)}</div><Button variant="outline" disabled={disabled||!selectedImages.size} onClick={()=>command.execute("approve",{selection:provenance,images:Array.from(selectedImages).map(id=>({id,role:"gallery"})),approved:[]})}>确认完整图片选择</Button>
    <TemuImages draft={draft} inventory={inventory} disabled={disabled} onChange={images=>change(v=>{v.images=images})}/>
   </section>
   {existing?.temu.issues.length?<ul className="space-y-1 rounded-lg bg-amber-50 p-3 text-sm">{existing.temu.issues.map((issue,i)=><li key={i}>{issue.message}</li>)}</ul>:null}
   <p className="text-xs text-muted-foreground">保存后重新读取当前官方要求并校验。创建回执、审核结果和上架状态分别显示。</p><Button disabled={disabled||loading||draft.mode!==rules?.temu.mode} onClick={save}>保存并校验平台资料</Button>
  </>:<p role="status" className="text-sm">正在读取当前店铺的官方资料要求…</p>}
 </div>;
}

function LocalSKUFields({index,sku,rules,disabled,onChange,onRemove}:{index:number;sku:LocalSKU;rules?:Rules;disabled:boolean;onChange:(edit:(v:LocalSKU)=>void)=>void;onRemove:()=>void}){
 const prefix=`SKU ${index+1}`,spec=rules?.temu.localTarget?.specification;
 return <div className="space-y-3 rounded-xl border p-4"><h4 className="font-medium">{prefix}</h4><div className="grid gap-3 sm:grid-cols-2">
  <Field label={`${prefix} 外部编号`}><Input disabled={disabled} value={sku.externalSkuId??""} onChange={e=>onChange(v=>{v.externalSkuId=e.target.value})}/></Field>
  <Field label={`${prefix} 基础价格`}><Input disabled={disabled} inputMode="decimal" value={sku.price.basePrice.amount} onChange={e=>onChange(v=>{v.price.basePrice.amount=e.target.value})}/></Field>
  <Field label={`${prefix} 币种`}><Select disabled={disabled} value={sku.price.basePrice.currency} onChange={e=>onChange(v=>{v.price.basePrice.currency=e.target.value})}><option value="">选择币种</option>{spec?.baseCurrency?<option value={spec.baseCurrency}>{spec.baseCurrency}</option>:null}</Select></Field>
  <Field label={`${prefix} 库存数量`}><Input disabled={disabled} inputMode="numeric" value={sku.quantity??""} onChange={e=>onChange(v=>{v.quantity=e.target.value||null})}/></Field>
  {([['weight','重量'],['length','长度'],['width','宽度'],['height','高度']] as const).map(([key,label])=><Field key={key} label={`${prefix} 包装${label}`}><Input disabled={disabled} inputMode="decimal" value={sku.packageInfo[key]} onChange={e=>onChange(v=>{v.packageInfo[key]=e.target.value})}/></Field>)}
 </div><LocalSpecs values={sku.specDetails??[]} rules={rules} disabled={disabled} onChange={values=>onChange(v=>{v.specDetails=values})}/><Button variant="ghost" size="sm" disabled={disabled} onClick={onRemove}>移除 {prefix}</Button></div>;
}

function LocalSpecs({values,rules,disabled,onChange}:{values:NonNullable<LocalSKU["specDetails"]>;rules?:Rules;disabled:boolean;onChange:(v:NonNullable<LocalSKU["specDetails"]>)=>void}){
 const parents=rules?.temu.localVariations?.variations??[];
 return <div className="space-y-2">{values.map((value,i)=>{const parent=parents.find(p=>p.parentSpecId===value.parentSpecId);return <div key={i} className="grid gap-2 sm:grid-cols-3"><Field label={`规格 ${i+1}`}><Select disabled={disabled} value={value.parentSpecId??""} onChange={e=>onChange(values.map((v,j)=>i===j?{parentSpecId:e.target.value||null}:v))}><option value="">选择官方规格</option>{parents.map(p=><option key={p.parentSpecId??""} value={p.parentSpecId??""}>{p.parentSpecName}</option>)}</Select></Field>
  {parent?.variationType!=="1"?<Field label={`规格 ${i+1} 值`}><Select disabled={disabled||!parent} value={value.specId??""} onChange={e=>{const chosen=parent?.specList?.find(s=>s.specId===e.target.value);onChange(values.map((v,j)=>i===j?{parentSpecId:v.parentSpecId,...chosen?{specId:chosen.specId,specName:chosen.specName}:{}}:v))}}><option value="">{parent?.variationType==="2"?"选择官方值或填写自定义值":"选择官方值"}</option>{parent?.specList?.map(s=><option key={s.specId??s.specName} value={s.specId??""}>{s.specName}</option>)}</Select></Field>:null}
  {parent&&parent.variationType!=="0"&&!value.specId?<Field label={`规格 ${i+1} 自定义值`}><Input disabled={disabled} value={value.specName??""} onChange={e=>onChange(values.map((v,j)=>i===j?{parentSpecId:v.parentSpecId,specName:e.target.value}:v))}/></Field>:null}<Button variant="ghost" size="sm" disabled={disabled} onClick={()=>onChange(values.filter((_,j)=>i!==j))}>移除规格</Button></div>})}<Button variant="outline" size="sm" disabled={disabled||values.length>=2} onClick={()=>onChange([...values,{}])}>添加规格</Button></div>;
}

function TemuImages({draft,inventory,disabled,onChange}:{draft:TemuDraftInput;inventory?:Inventory;disabled:boolean;onChange:(v:TemuDraftInput["images"])=>void}){
 const slots=draft.images??[],assets=inventory?.assets??[],[field,setField]=useState("gallery"),[skc,setSKC]=useState("0"),[sku,setSKU]=useState("0"),[asset,setAsset]=useState("");
 const fields=draft.mode==="local"?["gallery","detail","size","sku"]:["gallery","material","outer","skc","thumb"];
 const usage:Record<string,string>={detail:"1",gallery:"2",sku:"3",size:"4",material:"5",outer:"6",skc:"2",thumb:"3"};
 const labels:Record<string,string>={gallery:"商品展示图",detail:"详情图",size:"尺码图",sku:"SKU 图",material:"素材图",outer:"外包装图",skc:"SKC 展示图",thumb:"SKU 缩略图"};
 function add(){if(!asset)return;onChange([...slots,{field:field as NonNullable<TemuDraftInput["images"]>[number]["field"],skc,sku,sort:String(slots.filter(s=>s.field===field&&s.skc===skc&&s.sku===sku).length),assetId:asset,usage:usage[field]}])}
 return <div className="space-y-3"><div className="grid gap-3 sm:grid-cols-3"><Field label="图片用途"><Select disabled={disabled} value={field} onChange={e=>setField(e.target.value)}>{fields.map(f=><option key={f} value={f}>{labels[f]}</option>)}</Select></Field>{draft.supplier?<Field label="图片 SKC"><Select disabled={disabled} value={skc} onChange={e=>{setSKC(e.target.value);setSKU("0")}}>{draft.supplier.productSkcReqs?.map((_,i)=><option key={i} value={String(i)}>SKC {i+1}</option>)}</Select></Field>:null}{["sku","thumb"].includes(field)?<Field label="图片 SKU"><Select disabled={disabled} value={sku} onChange={e=>setSKU(e.target.value)}>{(draft.local?.skuList??draft.supplier?.productSkcReqs?.[Number(skc)]?.productSkuReqs)?.map((_,i)=><option key={i} value={String(i)}>SKU {i+1}</option>)}</Select></Field>:null}<Field label="已批准图片"><Select disabled={disabled} value={asset} onChange={e=>setAsset(e.target.value)}><option value="">选择素材</option>{assets.map((a,i)=><option key={a.id} value={a.id}>素材 {i+1} · {a.width??"?"}×{a.height??"?"}</option>)}</Select></Field></div><Button size="sm" variant="outline" disabled={disabled||!asset} onClick={add}>添加图片</Button>{slots.map((s,i)=><div key={i} className="flex items-center gap-3 rounded-lg border p-3">{assets.find(a=>a.id===s.assetId)?.url?<Image src={assets.find(a=>a.id===s.assetId)!.url} alt="" width={60} height={60} unoptimized/>:null}<span className="text-sm">{labels[s.field]} · {s.field==="sku"||s.field==="thumb"?`SKU ${Number(s.sku)+1} · `:""}第 {Number(s.sort)+1} 张</span><Button size="sm" variant="ghost" disabled={disabled} onClick={()=>{const remaining=slots.filter((_,j)=>j!==i);const seen=new Map<string,number>();onChange(remaining.map(v=>{const key=`${v.field}:${v.skc}:${v.sku}`,sort=seen.get(key)??0;seen.set(key,sort+1);return {...v,sort:String(sort)}}))}}>移除图片</Button></div>)}</div>;
}
function Field({label,children}:{label:string;children:ReactNode}){return <label className="grid gap-1.5 text-sm"><span>{label}</span>{children}</label>}
function code(e:unknown){return e instanceof SupplyAPIError?e.code:"暂时无法读取官方资料，请刷新重试。"}
