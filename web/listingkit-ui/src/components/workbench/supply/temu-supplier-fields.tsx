"use client";
import type {ReactNode} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import type {TemuDraftInput} from "@/lib/contracts/temu-model";
import {temuRuleSnapshotSchema} from "@/lib/contracts/temu-model";
import {emptyTemuSupplierSKU} from "./temu-draft-fields";

type Supplier=NonNullable<TemuDraftInput["supplier"]>;
type SKU=NonNullable<NonNullable<Supplier["productSkcReqs"]>[number]["productSkuReqs"]>[number];
type Snapshot=z.infer<typeof temuRuleSnapshotSchema>;
type Property=NonNullable<Supplier["productPropertyReqs"]>[number];
type Definition=NonNullable<NonNullable<Snapshot["supplierAttributes"]>["properties"]>[number];
type Spec=NonNullable<SKU["productSkuSpecReqs"]>[number];

export function TemuSupplierFields({draft,rules,disabled,onChange,onSizePage,onSizeSelection}:{draft:TemuDraftInput;rules?:Snapshot;disabled:boolean;onChange:(edit:(v:Supplier)=>void)=>void;onSizePage:(offset:string)=>void;onSizeSelection:(id:string,checked:boolean)=>void}) {
 const supplier=draft.supplier;if(!supplier)return null;
 const sites=supplier.productSemiManagedReq?.bindSiteIds??[];
 const sizePage=rules?.supplierSizeCharts,sizeOffset=BigInt(sizePage?.offset??"0"),sizeTotal=BigInt(sizePage?.totalCount??"0");
 const zero=BigInt(0),pageLength=BigInt(50),maximumOffset=BigInt(1000000);
 const knownSizeCharts=[...(sizePage?.sizeSpecDataList??[]),...(rules?.supplierSizeChartPages??[]).flatMap(p=>p.sizeSpecDataList??[])];
 const siteLabel=(id:string)=>rules?.semiSites?.find(s=>s.siteId===id)?.label??id;
 const sale=(rules?.supplierAttributes?.properties??[]).filter(p=>p.isSale===true);
 function specs(edit:(v:Supplier)=>void){onChange(v=>{edit(v);const used=new Set(v.productSkcReqs?.flatMap(c=>[...(c.mainProductSkuSpecReqs??[]),...(c.productSkuReqs??[]).flatMap(k=>k.productSkuSpecReqs??[])]).filter(s=>s.specId&&s.specId!=="0").map(s=>`${s.parentSpecId}:${s.specId}`));v.productSpecPropertyReqs=sale.flatMap(d=>(d.values??[]).filter(value=>used.has(`${d.parentSpecId}:${value.specId}`)).map(value=>({pid:d.pid,templatePid:d.templatePid,refPid:d.refPid,propName:d.name,vid:value.vid,propValue:value.value,valueUnit:"",specId:value.specId,parentSpecId:d.parentSpecId,valueGroupId:d.group?.id??"0",valueGroupName:d.group?.name??""})))})}

 return <section className="space-y-4">
  <h3 className="font-semibold">供货资料</h3>
  {draft.mode==="semi_managed"?<fieldset className="space-y-2"><legend>半托管销售站点</legend><p className="text-xs text-muted-foreground">一次绑定所选站点。泛欧、澳新等关联站点要求仍由官方规则检查。</p><div className="flex flex-wrap gap-3">{rules?.semiSites?.map(site=><label key={site.siteId} className="flex gap-2 text-sm"><input type="checkbox" aria-label={`销售站点 ${site.label}`} disabled={disabled||site.opening!=="open"} checked={sites.includes(site.siteId)} onChange={e=>onChange(v=>{const next=e.target.checked?[...sites,site.siteId]:sites.filter(id=>id!==site.siteId);v.productSemiManagedReq={...v.productSemiManagedReq,bindSiteIds:next};v.productSkcReqs?.forEach(skc=>skc.productSkuReqs?.forEach(sku=>{sku.siteSupplierPrices=sku.siteSupplierPrices?.filter(price=>price.siteId&&next.includes(price.siteId))}))})}/>{site.label}{site.opening!=="open"?"（未开放）":""}</label>)}</div></fieldset>:<p className="text-sm text-muted-foreground">全托管的消费者市场由 TEMU 分配，创建一个供应商货品。</p>}
  <SupplierProperties definitions={rules?.supplierAttributes?.properties??[]} values={supplier.productPropertyReqs??[]} disabled={disabled} onChange={values=>onChange(v=>{v.productPropertyReqs=values})}/>
  {draft.mode==="semi_managed"?<div className="grid gap-3 sm:grid-cols-2"><Field label="运费模板编号"><Input disabled={disabled} value={supplier.productShipmentReq?.freightTemplateId??""} placeholder="本店已有平台模板编号" onChange={e=>onChange(v=>{v.productShipmentReq={freightTemplateId:e.target.value,shipmentLimitSecond:v.productShipmentReq?.shipmentLimitSecond??null}})}/></Field><Field label="备货时长"><Select disabled={disabled} value={supplier.productShipmentReq?.shipmentLimitSecond??""} onChange={e=>onChange(v=>{v.productShipmentReq={freightTemplateId:v.productShipmentReq?.freightTemplateId??"",shipmentLimitSecond:e.target.value||null}})}><option value="">选择实际承诺</option><option value="86400">1天</option><option value="172800">2天</option><option value="259200">3天（仅限定制商品，资格由平台确认）</option></Select></Field></div>:null}
  {rules?.supplierSizeRequirement?.class?<section className="space-y-2">
   <h4 className="font-medium">本类目要求尺码表</h4><p className="text-xs text-muted-foreground">只引用本店已有、可直接发布的尺码表。可复用模板需要单独创建，本页不会自动创建。</p>
   {sizePage?<div className="flex items-center gap-3"><Button size="sm" variant="outline" disabled={disabled||sizeOffset===zero} onClick={()=>onSizePage((sizeOffset>pageLength?sizeOffset-pageLength:zero).toString())}>尺码表上一页</Button><span className="text-sm">第 {(sizeOffset/pageLength+BigInt(1)).toString()} 页</span><Button size="sm" variant="outline" disabled={disabled||sizeOffset+pageLength>=sizeTotal||sizeOffset+pageLength>maximumOffset} onClick={()=>onSizePage((sizeOffset+pageLength).toString())}>尺码表下一页</Button></div>:null}
   {sizePage?.sizeSpecDataList?.map(chart=><label key={chart.businessId??chart.name} className="flex gap-2 text-sm"><input type="checkbox" disabled={disabled||chart.reusable!==false||!chart.businessId} checked={!!chart.businessId&&!!supplier.sizeTemplateIds?.includes(chart.businessId)} onChange={e=>{if(chart.businessId)onSizeSelection(chart.businessId,e.target.checked)}}/>{chart.name||chart.businessId}{chart.reusable!==false?"（需要创建发布表）":""}</label>)}
  </section>:rules?.supplierSizeRequirement?.observed?<p className="text-xs text-muted-foreground">当前官方查询未返回尺码表要求。</p>:<p className="text-xs text-amber-700">尺码表要求尚未核实，商品可保存。</p>}
  {supplier.sizeTemplateIds?.length?<div className="space-y-1"><p className="text-sm">已选择 {supplier.sizeTemplateIds.length} 张尺码表，保存与提交时重新核验。</p>{supplier.sizeTemplateIds.map(id=><div key={id} className="flex items-center gap-2 text-sm"><span>{knownSizeCharts.find(c=>c.businessId===id)?.name||id}</span><Button size="sm" variant="ghost" disabled={disabled} onClick={()=>onSizeSelection(id,false)}>移除尺码表 {id}</Button></div>)}</div>:null}
  <Field label="产地国家或地区代码"><Input disabled={disabled} value={supplier.productWhExtAttrReq?.productOrigin.region1ShortName??""} placeholder="填写已有产地事实" onChange={e=>onChange(v=>{v.productWhExtAttrReq={...v.productWhExtAttrReq,outerGoodsUrl:v.productWhExtAttrReq?.outerGoodsUrl??"",productOrigin:{...v.productWhExtAttrReq?.productOrigin,region1ShortName:e.target.value}}})}/></Field>
  {supplier.productSkcReqs?.map((skc,ci)=><div key={ci} className="space-y-3 rounded-xl border p-4">
   <h4 className="font-medium">SKC {ci+1}</h4>
   <Field label={`SKC ${ci+1} 外部编号`}><Input disabled={disabled} value={skc.extCode} onChange={e=>onChange(v=>{v.productSkcReqs![ci].extCode=e.target.value})}/></Field>
   <SupplierSpecs label={`SKC ${ci+1} 主规格`} definitions={sale.filter(p=>p.mainSale===true)} values={skc.mainProductSkuSpecReqs??[]} disabled={disabled} onChange={values=>specs(v=>{v.productSkcReqs![ci].mainProductSkuSpecReqs=values})}/>
   {!sale.some(p=>p.mainSale===true)?<Button size="sm" variant="outline" disabled={disabled} onClick={()=>specs(v=>{v.productSkcReqs![ci].mainProductSkuSpecReqs=[{parentSpecId:"0",parentSpecName:"",specId:"0",specName:""}]})}>使用无主规格固定值</Button>:null}
   {skc.productSkuReqs?.map((sku,si)=>{
    const prefix=`SKC ${ci+1} SKU ${si+1}`;
    function edit(fn:(value:SKU)=>void){onChange(v=>{fn(v.productSkcReqs![ci].productSkuReqs![si])})}
    return <div key={si} className="space-y-3 rounded-lg bg-muted/30 p-3">
     <div className="grid gap-3 sm:grid-cols-2">
      <Field label={`${prefix} 外部编号`}><Input disabled={disabled} value={sku.extCode} onChange={e=>edit(v=>{v.extCode=e.target.value})}/></Field>
      <Field label={`${prefix} 供货币种`}><Select disabled={disabled} value={sku.currencyType} onChange={e=>edit(v=>{v.currencyType=e.target.value})}><option value="">选择实际供货币种</option><option value="CNY">CNY（分）</option><option value="USD">USD（美分）</option></Select></Field>
      {draft.mode==="fully_managed"?<Field label={`${prefix} 供货价（整数分）`}><Input disabled={disabled} inputMode="numeric" value={sku.supplierPrice??""} onChange={e=>edit(v=>{v.supplierPrice=e.target.value||undefined})}/></Field>:sites.map(site=><Field key={site} label={`${prefix} 站点 ${siteLabel(site)} 供货价（整数分）`}><Input disabled={disabled} inputMode="numeric" value={sku.siteSupplierPrices?.find(price=>price.siteId===site)?.supplierPrice??""} onChange={e=>edit(v=>{const prices=v.siteSupplierPrices??[];v.siteSupplierPrices=[...prices.filter(price=>price.siteId!==site),{siteId:site,supplierPrice:e.target.value||null}].sort((a,b)=>sites.indexOf(a.siteId??"")-sites.indexOf(b.siteId??""))})}/></Field>)}
     </div>
     <SupplierSpecs label={`${prefix} 规格`} definitions={sale} values={sku.productSkuSpecReqs??[]} disabled={disabled} onChange={values=>specs(v=>{v.productSkcReqs![ci].productSkuReqs![si].productSkuSpecReqs=values})}/>
     <SupplierPackaging prefix={prefix} value={sku.productSkuWhExtAttrReq} disabled={disabled} onChange={value=>edit(v=>{v.productSkuWhExtAttrReq=value})}/>
     <div className="space-y-2">{sku.productSkuStockQuantityReq?.warehouseStockQuantityReqs?.map((stock,wi)=><div key={wi} className="grid gap-3 sm:grid-cols-2"><Field label={`${prefix} 仓库 ${wi+1} 编号`}><Input disabled={disabled} value={stock.warehouseId} onChange={e=>edit(v=>{v.productSkuStockQuantityReq!.warehouseStockQuantityReqs![wi].warehouseId=e.target.value})}/></Field><Field label={`${prefix} 仓库 ${wi+1} 初始数量`}><Input disabled={disabled} inputMode="numeric" value={stock.targetStockAvailable??""} onChange={e=>edit(v=>{v.productSkuStockQuantityReq!.warehouseStockQuantityReqs![wi].targetStockAvailable=e.target.value||null})}/></Field><Button variant="ghost" size="sm" disabled={disabled} onClick={()=>edit(v=>{v.productSkuStockQuantityReq!.warehouseStockQuantityReqs=v.productSkuStockQuantityReq!.warehouseStockQuantityReqs!.filter((_,i)=>i!==wi)})}>移除仓库 {wi+1}</Button></div>)}<Button variant="outline" size="sm" disabled={disabled} onClick={()=>edit(v=>{v.productSkuStockQuantityReq={warehouseStockQuantityReqs:[...v.productSkuStockQuantityReq?.warehouseStockQuantityReqs??[],{warehouseId:"",targetStockAvailable:null}]}})}>{prefix} 添加仓库库存</Button><p className="text-xs text-muted-foreground">填写实际仓库与数量；零库存可明确保存，仓库资格仍须官方核实。</p></div>
     <Button size="sm" variant="ghost" disabled={disabled} onClick={()=>onChange(v=>{v.productSkcReqs![ci].productSkuReqs=v.productSkcReqs![ci].productSkuReqs?.filter((_,i)=>i!==si)??[]})}>移除 {prefix}</Button>
    </div>;
   })}
   <Button size="sm" variant="outline" disabled={disabled} onClick={()=>onChange(v=>{v.productSkcReqs![ci].productSkuReqs=[...v.productSkcReqs![ci].productSkuReqs??[],emptyTemuSupplierSKU()]})}>添加 SKC {ci+1} SKU</Button>
  </div>)}
  <Button variant="outline" disabled={disabled} onClick={()=>onChange(v=>{v.productSkcReqs=[...v.productSkcReqs??[],{extCode:"",productSkuReqs:[emptyTemuSupplierSKU()],mainProductSkuSpecReqs:[],previewImgUrls:[]}]})}>添加 SKC</Button>
  {supplier.productComplianceStatementReq?<p className="text-sm text-amber-700">资料包含合规协议版本 {supplier.productComplianceStatementReq.protocolVersion}；本页不会代替已有的合法批准依据或自动签署。</p>:null}
 </section>;
}

function SupplierPackaging({prefix,value,disabled,onChange}:{prefix:string;value:SKU["productSkuWhExtAttrReq"];disabled:boolean;onChange:(value:SKU["productSkuWhExtAttrReq"])=>void}) {
 function edit(fn:(value:NonNullable<SKU["productSkuWhExtAttrReq"]>)=>void){const next=structuredClone(value??{productSkuWeightReq:{value:null},productSkuVolumeReq:{len:null,width:null,height:null},productSkuSensitiveAttrReq:{},productSkuSensitiveLimitReq:{}});fn(next);onChange(next)}
 return <div className="grid gap-3 sm:grid-cols-2">
  <Field label={`${prefix} 包装重量（mg）`}><Input disabled={disabled} inputMode="numeric" value={value?.productSkuWeightReq?.value??""} onChange={e=>edit(v=>{v.productSkuWeightReq={...v.productSkuWeightReq,value:e.target.value||null}})}/></Field>
  {(["len","width","height"] as const).map(key=><Field key={key} label={`${prefix} 包装 ${key}（mm）`}><Input disabled={disabled} inputMode="numeric" value={value?.productSkuVolumeReq?.[key]??""} onChange={e=>edit(v=>{v.productSkuVolumeReq={len:null,width:null,height:null,...v.productSkuVolumeReq,[key]:e.target.value||null}})}/></Field>)}
  <Field label={`${prefix} 敏感属性`}><Select disabled={disabled} value={value?.productSkuSensitiveAttrReq?.isSensitive??""} onChange={e=>edit(v=>{v.productSkuSensitiveAttrReq={...v.productSkuSensitiveAttrReq,isSensitive:e.target.value||undefined}})}><option value="">确认实际属性</option><option value="0">非敏感</option><option value="1">敏感</option></Select></Field>
  {value?.productSkuSensitiveAttrReq?.isSensitive==="1"?<Field label={`${prefix} 敏感类型`}><Input disabled={disabled} value={value.productSkuSensitiveAttrReq.sensitiveList?.join(",")??""} placeholder="填写已确认的类型编号，以逗号分隔" onChange={e=>edit(v=>{v.productSkuSensitiveAttrReq={...v.productSkuSensitiveAttrReq,sensitiveList:e.target.value?e.target.value.split(",").map(s=>s.trim()):[]}})}/></Field>:null}
 </div>;
}

function SupplierProperties({definitions,values,disabled,onChange}:{definitions:Definition[];values:Property[];disabled:boolean;onChange:(value:Property[])=>void}) {
 return <section className="space-y-3"><h4 className="font-medium">官方商品属性</h4>{definitions.filter(d=>d.isSale===false).map(d=>{
  const current=values.filter(v=>v.refPid===d.refPid),custom=d.controlType==="0"||d.controlType==="3";
  function set(next:Property[]){onChange([...values.filter(v=>v.refPid!==d.refPid),...next])}
  function base():Property{return {pid:d.pid,templatePid:d.templatePid,refPid:d.refPid,propName:d.name,vid:"0",propValue:"",valueUnit:""}}
  return <div key={d.refPid??d.name} className="space-y-2 rounded-lg border p-3"><p className="text-sm">{d.name}{d.required?"（必填）":""}</p>{current.map((value,i)=><div key={i} className="grid gap-2 sm:grid-cols-3"><Field label={`${d.name} 值 ${i+1}`}><Select disabled={disabled} value={value.vid&&value.vid!=="0"?value.vid:""} onChange={e=>{const chosen=d.values?.find(v=>v.vid===e.target.value);set(current.map((v,j)=>i===j?{...v,vid:chosen?.vid??"0",propValue:chosen?.value??""}:v))}}><option value="">{custom?"选择官方值或填写":"选择官方值"}</option>{d.values?.map(v=><option key={v.vid??v.value} value={v.vid??""}>{v.value}</option>)}</Select></Field>{custom&&value.vid==="0"?<Field label={`${d.name} 自定义值 ${i+1}`}><Input disabled={disabled} value={value.propValue} onChange={e=>set(current.map((v,j)=>i===j?{...v,propValue:e.target.value}:v))}/></Field>:null}{d.controlType==="16"?<Field label={`${d.name} 数值 ${i+1}`}><Input disabled={disabled} inputMode="decimal" value={value.numberInputValue??""} onChange={e=>set(current.map((v,j)=>i===j?{...v,numberInputValue:e.target.value}:v))}/></Field>:null}{d.valueUnit?.length?<Field label={`${d.name} 单位 ${i+1}`}><Select disabled={disabled} value={value.valueUnit} onChange={e=>set(current.map((v,j)=>i===j?{...v,valueUnit:e.target.value}:v))}><option value="">选择明确单位</option>{d.valueUnit.map(unit=><option key={unit} value={unit}>{unit}</option>)}</Select></Field>:null}<Button size="sm" variant="ghost" disabled={disabled} onClick={()=>set(current.filter((_,j)=>i!==j))}>移除属性值</Button></div>)}<Button size="sm" variant="outline" disabled={disabled||!["0","1","3","16"].includes(d.controlType??"")||BigInt(current.length)>=BigInt(d.chooseMaxNum??d.inputMaxNum??"0")} onClick={()=>set([...current,base()])}>添加 {d.name} 属性值</Button>{!["0","1","3","16"].includes(d.controlType??"")?<p className="text-xs text-amber-700">该属性输入规则仍待核实。</p>:null}</div>;
 })}</section>;
}

function SupplierSpecs({label,definitions,values,disabled,onChange}:{label:string;definitions:Definition[];values:Spec[];disabled:boolean;onChange:(value:Spec[])=>void}) {
 return <div className="space-y-2">{definitions.map(d=><Field key={d.refPid??d.name} label={`${label} ${d.name}`}><Select disabled={disabled} value={values.find(s=>s.parentSpecId===d.parentSpecId)?.specId??""} onChange={e=>{const chosen=d.values?.find(v=>v.specId===e.target.value);onChange([...values.filter(v=>v.parentSpecId!==d.parentSpecId),...chosen?[{specId:chosen.specId,parentSpecId:d.parentSpecId,parentSpecName:d.name,specName:chosen.value}]:[]])}}><option value="">选择当前官方预设规格</option>{d.values?.filter(v=>v.specId).map(v=><option key={v.specId!} value={v.specId!}>{v.value}</option>)}</Select></Field>)}{definitions.some(d=>!d.values?.some(v=>v.specId))?<p className="text-xs text-amber-700">缺少可选规格时保持待补全；本页不会创建新的平台规格。</p>:null}</div>;
}
function Field({label,children}:{label:string;children:ReactNode}){return <label className="grid gap-1.5 text-sm"><span>{label}</span>{children}</label>}
