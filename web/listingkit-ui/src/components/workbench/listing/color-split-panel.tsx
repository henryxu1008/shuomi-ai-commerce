"use client";
import {useEffect,useState} from "react";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Select} from "@/components/ui/select";
import {listColorSplitFamilies,previewColorSplit,SupplyAPIError,type SupplyScope} from "@/lib/api/supply-chain";
import {colorSplitReceiptSchema,targetReceiptSchema,type ColorSplitPlan,type ColorSplitReceipt,type SupplyCompletionDraft,type SupplySourceDetail,type SupplyTarget,type SupplyRules} from "@/lib/contracts/supply-chain";
import {sheinImportHref} from "@/lib/contracts/shein-import-navigation";
import {useListingCommand} from "./use-listing-command";
import {ListingCommandFeedback} from "./command-feedback";
import {ListingTargetEditor} from "./target-editor";

function EligibleColorSplitPanel({scope,detail,candidate,base,rules,canManage,familyId,onOpenChange}:{scope:SupplyScope;detail:SupplySourceDetail;candidate:SupplyCompletionDraft;base:SupplyTarget;rules?:SupplyRules;canManage:boolean;familyId?:string;onOpenChange?:(opened:boolean)=>void}){
 const router=useRouter();const [opened,setOpened]=useState(!!familyId),[families,setFamilies]=useState<ColorSplitReceipt[]>(),[loadError,setLoadError]=useState(false),[epoch,setEpoch]=useState(0);
 const [mapping,setMapping]=useState<Record<number,string>>({}),[shared,setShared]=useState<Record<string,boolean>>({}),[plan,setPlan]=useState<ColorSplitPlan>(),[preparing,setPreparing]=useState(false),[error,setError]=useState<string>(),[editing,setEditing]=useState<SupplyTarget>();
 const command=useListingCommand(scope,(value,intent)=>{
  if(intent.route==="create-color-split"){const saved=colorSplitReceiptSchema.parse(value);if(saved.records.some(r=>r.source.id!==detail.source.id||r.input.storeId!==base.input.storeId))return;setFamilies([saved]);setPlan(undefined);setOpened(true);const url=new URL(window.location.href);url.searchParams.set("family",saved.familyId);router.replace(url.pathname+url.search,{scroll:false});}
  if(intent.route==="save-target"){const saved=targetReceiptSchema.parse(value).record;if("colorPartition" in saved.input&&saved.input.colorPartition&&saved.source.id===detail.source.id&&saved.input.storeId===base.input.storeId){setEditing(undefined);setEpoch(n=>n+1)}}
 });
 useEffect(()=>{const c=new AbortController();let live=true;void listColorSplitFamilies(scope,detail.source.id,base.input.storeId,c.signal).then(v=>{if(live){setFamilies(v.items);setLoadError(false);}}).catch(()=>{if(live)setLoadError(true)});return()=>{live=false;c.abort()};},[scope,detail.source.id,base.input.storeId,epoch]);
 useEffect(()=>{onOpenChange?.(opened)},[opened,onOpenChange]);
 const skcs=base.input.draft.product.skc_list,attributeId=skcs[0]?.sale_attribute.attribute_id;
 const attribute=rules?.rules.attributes.attribute_infos.find(a=>a.attribute_id===attributeId);
 const colors=skcs.map((s,i)=>({id:s.sale_attribute.attribute_value_id,name:attribute?.value_info_list.find(v=>v.attribute_value_id===s.sale_attribute.attribute_value_id)?.attribute_value??`颜色分组 ${i+1}`}));
 const targets=skcs.flatMap((s,i)=>s.sku_list.map((sku,j)=>({key:`${i}:${j}`,label:`${colors[i]?.name} · ${sku.supplier_sku}`})));
 const disabled=!canManage||command.busy||!!command.pending||!command.ready||preparing;
 const sourceVariants=candidate.content.structured?.variants??[];
 const complete=sourceVariants.length===targets.length&&sourceVariants.length>0&&sourceVariants.every(v=>mapping[v.sourceIndex])&&new Set(Object.values(mapping)).size===targets.length;
 const change=()=>{setPlan(undefined);setError(undefined)};
 async function preview(){if(disabled||!complete||!attributeId)return;setPreparing(true);setError(undefined);setPlan(undefined);
  try{const value=await previewColorSplit(scope,{sourceId:detail.source.id,baseRecordId:base.id,completionDraft:{recordId:candidate.id,revision:candidate.revision},selection:{colorAttributeId:attributeId,skus:sourceVariants.map(v=>{const [skcIndex,skuIndex]=mapping[v.sourceIndex]!.split(":").map(Number);return {sourceIndex:v.sourceIndex,skcIndex:skcIndex!,skuIndex:skuIndex!}}),sharedImages:base.input.draft.images.flatMap((slot,n)=>slot.group==="spu"?[{slotIndex:n,colorValueIds:colors.filter(c=>c.id&&shared[`${n}:${c.id}`]).map(c=>c.id!)}]:[])}});setPlan(value)}catch(e){setError(e instanceof SupplyAPIError?e.code:"DEPENDENCY_UNAVAILABLE")}finally{setPreparing(false)}
 }
 const visible=families?.filter(f=>!familyId||f.familyId===familyId)??[];
 if(base.input.colorPartition||skcs.length!==2||base.merchant.application_type!=="self_operated")return null;
 return <Card className="space-y-4 p-5"><div><h2 className="font-semibold">按颜色拆分平台草稿</h2><p className="mt-1 text-sm text-muted-foreground">将现有两组颜色分别保存为本地草稿。原商品、候选和原平台草稿保留，暂不支持上传。</p></div>
 {!opened?<Button variant="outline" disabled={command.busy||!!command.pending||!families} onClick={()=>setOpened(true)}>{families?.length?"查看已保存颜色草稿":"按颜色拆分"}</Button>:null}
 {loadError?<p role="alert">拆分草稿读取失败。<Button variant="outline" onClick={()=>setEpoch(n=>n+1)}>重新读取拆分草稿</Button></p>:!families?<p>正在读取已有拆分草稿…</p>:null}
 {opened?<>{!editing?<Button variant="ghost" disabled={command.busy||!!command.pending} onClick={()=>{setOpened(false);setPlan(undefined);const url=new URL(window.location.href);url.searchParams.delete("family");router.replace(url.pathname+url.search,{scroll:false})}}>返回普通资料带入</Button>:null}{!command.pending?<ListingCommandFeedback state={command}/>:null}{command.error==="SPLIT_ALREADY_EXISTS"?<Button variant="outline" onClick={()=>setEpoch(n=>n+1)}>重新读取已保存颜色草稿</Button>:null}{visible.map(f=><section className="space-y-3" key={f.familyId}><h3 className="font-medium">本地拆分草稿 {f.records.length} 份</h3><p className="text-sm text-muted-foreground">每份独立保存；缺库存或图片仍为待补全。公共说明复制自原稿，可在各稿中手动修改。</p>{f.records.map(r=>{
 const color=colors.find(c=>c.id===r.input.colorPartition?.colorValueId)?.name??"颜色";
 return <div key={r.targetId} className="space-y-3 rounded-lg border border-border p-4"><h4 className="font-medium">{color}草稿 · 版本 {r.revision}</h4><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th>SKU</th><th>库存</th><th>价格</th></tr></thead><tbody>{r.input.draft.product.skc_list.flatMap(s=>s.sku_list).map(s=><tr key={s.supplier_sku}><td className="py-2">{s.supplier_sku}</td><td>{s.stock_info_list.length?s.stock_info_list.map(v=>v.inventory_num).join(" / "):"未填写"}</td><td>{s.price_info_list?.map(p=>`${p.currency} ${p.base_price}`).join(" / ")??"未填写"}</td></tr>)}</tbody></table></div><p className="text-sm">图片位置：{r.input.draft.images.map((v,n)=>v.group==="spu"?`公共图 ${n+1}`:v.group==="sku"?`${r.input.draft.product.skc_list[v.skc]?.sku_list[v.sku]?.supplier_sku} 专属图`:`${color}图片`).join("、")||"待补全"}</p><div className="flex flex-wrap gap-2"><Button variant="outline" disabled={disabled} onClick={()=>setEditing(r)}>查看并编辑{color}草稿</Button><Button asChild variant="ghost"><Link href={`/workbench/listing/ready?record=${r.id}`}>查看已保存版本</Link></Button></div></div>;
 })}</section>)}
 {editing?<section className="space-y-4 rounded-lg border border-border p-4"><div className="flex flex-wrap items-center justify-between gap-2"><h3 className="font-medium">编辑本地颜色草稿</h3><Button variant="outline" disabled={disabled} onClick={()=>setEditing(undefined)}>返回两份草稿</Button></div><ListingTargetEditor key={editing.id} scope={scope} source={detail} storeId={base.input.storeId} existing={editing} command={command} disabled={disabled} saved={epoch}/></section>:null}
 {families&&!families.length&&!loadError?<><fieldset disabled={disabled} className="space-y-4"><legend className="mb-2 text-sm font-medium">确认每条来源 SKU 对应的现有规格</legend>{sourceVariants.map(v=><label key={v.sourceIndex} className="grid gap-2 text-sm sm:grid-cols-2"><span>{detail.product.variants?.[v.sourceIndex]?.sku||`来源 SKU ${v.sourceIndex+1}`} · 库存 {v.stock??"未填写"}</span><Select aria-label={`拆分来源 SKU ${v.sourceIndex+1} 的目标`} value={mapping[v.sourceIndex]??""} onChange={e=>{setMapping(m=>({...m,[v.sourceIndex]:e.target.value}));change()}}><option value="">选择对应规格</option>{targets.map(t=><option key={t.key} value={t.key} disabled={Object.entries(mapping).some(([i,k])=>Number(i)!==v.sourceIndex&&k===t.key)}>{t.label}</option>)}</Select></label>)}
 {base.input.draft.images.map((v,n)=>v.group==="spu"?<div key={n} className="space-y-2"><p className="text-sm">公共图片 {n+1}：明确选择复用到哪份草稿</p>{colors.map(c=><label key={c.id} className="mr-4 inline-flex items-center gap-2 text-sm"><input type="checkbox" aria-label={`公共图片 ${n+1} 用于${c.name}`} checked={!!shared[`${n}:${c.id}`]} onChange={e=>{setShared(s=>({...s,[`${n}:${c.id}`]:e.target.checked}));change()}}/>{c.name}草稿</label>)}</div>:null)}<p className="text-sm text-muted-foreground">颜色图和 SKU 专属图只跟随原归属；不再需要的多颜色色块位置不带入。价格和空库存原样保留。</p></fieldset>
 {error?<p role="alert" className="text-sm text-destructive">{error==="REVISION_CONFLICT"?"原稿、库存或规则已变化，请重新读取后预览。":"拆分预览未完成，请检查 SKU 对应关系和图片选择。"}</p>:null}
 <Button variant="outline" disabled={disabled||!complete} onClick={()=>void preview()}>{preparing?"正在预览…":"预览两份颜色草稿"}</Button>
 {plan?<section className="space-y-3"><h3 className="font-medium">确认拆分内容</h3>{plan.groups.map(g=><div key={g.colorValueId} className="rounded-lg bg-muted p-3 text-sm"><p>{colors.find(c=>c.id===g.colorValueId)?.name}：{g.input.draft.product.skc_list[0]?.sku_list.map(s=>s.supplier_sku).join("、")}</p><p>库存：{g.input.draft.product.skc_list[0]?.sku_list.map(s=>s.stock_info_list.length?s.stock_info_list.map(v=>v.inventory_num).join("/"):"未填写").join("、")}</p><p>图片位置 {g.input.draft.images.length} 个；待补全 {g.issues.length} 项</p></div>)}<Button disabled={disabled} onClick={()=>void command.execute("create-color-split",plan)}>确认创建两份本地草稿</Button></section>:null}
 </>:null}</>:null}</Card>;
}

export function ColorSplitLinks({scope,detail,storeId}:{scope:SupplyScope;detail:SupplySourceDetail;storeId:string}){
 const [items,setItems]=useState<ColorSplitReceipt[]>(),[error,setError]=useState(false);
 useEffect(()=>{const c=new AbortController();let live=true;void listColorSplitFamilies(scope,detail.source.id,storeId,c.signal).then(v=>{if(live)setItems(v.items)}).catch(()=>{if(live)setError(true)});return()=>{live=false;c.abort()};},[scope,detail.source.id,storeId]);
 if(error)return <span className="text-xs text-muted-foreground">拆分草稿状态暂不可读</span>;
 return <>{items?.map(f=>{const href=sheinImportHref(detail.source.id,detail.source.preparationId,detail.source.collectionItemId,f.records[0].input.colorPartition!.completionDraft.revision,f.records[0].input.colorPartition!.baseRecordId);return <Link className="block text-sm text-primary underline" key={f.familyId} href={`${href}&family=${f.familyId}`}>本地拆分草稿 {f.records.length} 份</Link>})}</>;
}

export function ColorSplitPanel(props:Parameters<typeof EligibleColorSplitPanel>[0]){if(props.base.input.colorPartition||props.base.input.draft.product.skc_list.length!==2||props.base.merchant.application_type!=="self_operated")return null;return <EligibleColorSplitPanel {...props}/>;}
