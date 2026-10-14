"use client";

import {useCallback,useMemo,useRef,useState} from "react";
import Image from "next/image";
import {useRouter} from "next/navigation";
import {Plus,Trash2,ArrowLeft} from "lucide-react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {SourceImageUploader} from "./source-image-uploader";
import {useCollectionCommand} from "./use-collection-command";
import {CollectionCommandFeedback} from "./collection-command-feedback";
import {buildVariantRows,newVariantRow,prepareOwnProduct,prepareProductAttributes,removeVariantImage,specificationSignature,type ProductAttributeRow,type Specification,type VariantRow} from "./own-product-model";
import {formFromOwnProduct,type OwnProductFormState} from "./own-product-editor-model";
import {collectionCommandSchema,type CollectionDetail} from "@/lib/contracts/product-collection";
import type {CollectionScope} from "@/lib/api/product-collection";

const library="/workbench/data/mine?view=own";
export function OwnProductEntryPage(){
 const context=useWorkbenchContext();
 const userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.selectionRequired||context.error||context.blockingError)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 return <OwnProductForm key={`${scope.userId}:${scope.organizationId}`} scope={scope}/>;
}

export function OwnProductForm({scope,edit}:{scope:CollectionScope;edit?:{detail:CollectionDetail;initial:OwnProductFormState}}){
 const context=useWorkbenchContext(),router=useRouter();
 const [frozen]=useState(()=>{const pending=context.pendingCollectionIntent;if(edit&&pending?.userId===scope.userId&&pending.organizationId===scope.organizationId&&pending.command.action==="revise_product"&&pending.command.itemId===edit.detail.item.id)return pending.command;return null});
 const [initial]=useState(()=>frozen?formFromOwnProduct(frozen.product):edit?.initial);
 const [title,setTitle]=useState(initial?.title??""),[brand,setBrand]=useState(initial?.brand??""),[description,setDescription]=useState(initial?.description??"");
 const [attributes,setAttributes]=useState<ProductAttributeRow[]>(initial?.attributes??[{id:"initial",name:"",value:""}]);
 const [images,setImages]=useState<string[]>(initial?.images??[]),[imageURL,setImageURL]=useState(""),[mediaBlocked,setMediaBlocked]=useState(false);
 const imagesRef=useRef<string[]>(initial?.images??[]);
 const [mode,setMode]=useState<"single"|"multiple">(initial?.mode??"single");
 const [single,setSingle]=useState(()=>initial?.rows[0]??newVariantRow()),[rows,setRows]=useState<VariantRow[]>(initial?.mode==="multiple"?initial.rows:[]);
 const [specs,setSpecs]=useState<Specification[]>(initial?.specs??[{id:"initial",name:"",values:""}]),[generated,setGenerated]=useState(()=>initial?specificationSignature(initial.specs):"");
 const [bulk,setBulk]=useState({prefix:"",currency:"",price:"",stock:""}),[error,setError]=useState("");
 const commands=useCollectionCommand(scope,(receipt,intent)=>{
  if(intent.command.action==="create_product"&&receipt.itemId)router.push(`${library}&saved=${receipt.itemId}`);
  if(intent.command.action==="revise_product"&&receipt.itemId===intent.command.itemId)router.push(`/workbench/data/mine/own/${intent.command.itemId}`);
 },!!context.pendingSupplyIntent||!context.supplyIntentReady);
 const disabled=commands.locked,stale=mode==="multiple"&&!!rows.length&&generated!==specificationSignature(specs);
 function patchRow(id:string,patch:Partial<VariantRow>){if(mode==="single")setSingle(row=>({...row,...patch}));else setRows(prior=>prior.map(row=>row.id===id?{...row,...patch}:row))}
 const addImages=useCallback((values:string[])=>{
  const next=[...new Set([...imagesRef.current,...values])];
  if(next.length>40){setError("每个商品最多保存40张公共图片。");return}
  const checked=prepareImageURLs(next);if(!checked){setError("请填写有效的HTTPS图片链接，每行一个。");return}
  imagesRef.current=next;setImages(next);setImageURL("");setError("");
 },[]);
 function removeImage(url:string){const next=imagesRef.current.filter(v=>v!==url);imagesRef.current=next;setImages(next);setRows(prior=>prior.map(v=>removeVariantImage(v,url)));setSingle(v=>removeVariantImage(v,url))}
 function generate(){try{const attributeCount=Object.keys(prepareProductAttributes(attributes)).length;setRows(buildVariantRows(specs,rows,images.length,attributeCount));setGenerated(specificationSignature(specs));setError("")}catch(failure){setError((failure as Error).message)}}
 function applyBulk(){
  if(!bulk.prefix&&!bulk.currency&&!bulk.price.trim()&&!bulk.stock.trim()){setError("请先填写需要批量设置的值。");return}
  setRows(prior=>prior.map((row,index)=>({...row,...(bulk.prefix?{sku:`${bulk.prefix.trim()}-${index+1}`}:{}) ,...(bulk.currency?{currency:bulk.currency}:{}) ,...(bulk.price.trim()?{price:bulk.price}:{}) ,...(bulk.stock.trim()?{stock:bulk.stock}:{})})));setError("");
 }
 const activeRows=mode==="single"?[single]:rows;
 const formIdentity=JSON.stringify({title,brand,description,images,attributes,mode,single,rows,specs}),originalForm=useRef(formIdentity);
 function leave(){if(edit&&!commands.pending&&originalForm.current!==formIdentity&&!window.confirm("还有未保存的修改，确定返回商品详情吗？"))return;router.push(edit?`/workbench/data/mine/own/${edit.detail.item.id}`:library)}
 const pageTitle=edit?"编辑自有商品":"添加自有商品";
 return <ConsolePage title={pageTitle} description={edit?"修改已保存的商品资料，保存后生成新的资料版本。":"录入商品资料和销售规格，保存到当前企业下的自有商品库。"} breadcrumbs={[{label:"数据服务",href:"/workbench/data"},{label:"我的数据",href:library},{label:pageTitle}]} actions={<Button variant="outline" disabled={commands.busy||mediaBlocked} onClick={leave}><ArrowLeft size={16}/>{edit?"取消并返回详情":"返回自有商品库"}</Button>}>
  <form className="space-y-6" onSubmit={event=>{event.preventDefault();if(disabled||mediaBlocked)return;if(stale){setError("规格已修改，请先更新变体组合。");return}try{const product=prepareOwnProduct({title,brand,description,images,mode,rows:activeRows,attributes});const command=edit?{action:"revise_product" as const,itemId:edit.detail.item.id,expectedRevision:edit.detail.item.revision,expectedSourceVersion:edit.detail.item.source.version,expectedPublicationId:edit.detail.item.source.publicationId,product}:{action:"create_product" as const,product};const parsed=collectionCommandSchema.safeParse(command);if(!parsed.success)throw new Error(parsed.error.issues[0]?.message??"请检查填写内容。");setError("");void commands.run(parsed.data)}catch(failure){setError((failure as Error).message)}}}>
   <CollectionCommandFeedback state={commands}/>
   {edit&&commands.error==="REVISION_CONFLICT"?<Button type="button" variant="outline" onClick={()=>{if(window.confirm("重新读取会放弃当前未保存的修改，确定读取最新资料吗？"))window.location.reload()}}>重新读取最新资料</Button>:null}
   {edit?<Card className="space-y-2 border-border bg-muted/40 p-4"><p className="text-sm font-medium">基于资料版本 {frozen?.expectedSourceVersion??edit.detail.item.source.version}</p><p className="text-sm text-muted-foreground">保存会追加新版本，已有草稿继续使用它们选定的资料。{frozen?"正在显示待核实请求的原始内容。":""}</p></Card>:null}
   {context.pendingSupplyIntent?<p className="text-sm text-muted-foreground">有待核实的供应链操作，请先返回“我的供应链”处理后再保存商品。</p>:null}
   {!context.permissions.includes("workbench.collection.manage")?<p role="alert" className="text-sm text-destructive">当前没有添加自有商品的权限。</p>:null}
   <fieldset disabled={disabled} className="min-w-0 space-y-6 disabled:opacity-70">
    <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,3fr)_minmax(300px,2fr)]">
     <Card className="space-y-5 p-6"><div><h2 className="text-lg font-semibold">基本资料</h2><p className="mt-1 text-sm text-muted-foreground">所有变体共用商品名称、品牌和描述。</p></div>
      <label className="grid gap-2 text-sm">商品名称<Input required maxLength={2000} value={title} placeholder="输入商品名称" onChange={e=>setTitle(e.target.value)}/></label>
      <label className="grid gap-2 text-sm">品牌<Input value={brand} maxLength={2000} placeholder="选填" onChange={e=>setBrand(e.target.value)}/></label>
      <label className="grid gap-2 text-sm">商品描述<Textarea rows={6} maxLength={8192} value={description} placeholder="材质、用途及其他商品资料" onChange={e=>setDescription(e.target.value)}/></label>
     </Card>
     <Card className="space-y-5 p-6"><div><h2 className="text-lg font-semibold">商品图片</h2><p className="mt-1 text-sm text-muted-foreground">第一张作为商品主图；在变体表格中为不同规格选择对应图片。</p></div>
      {images.length?<div className="grid grid-cols-3 gap-3">{images.map((url,index)=><div key={url} className="min-w-0 space-y-2 rounded-lg border border-border p-2"><Image unoptimized src={`/api/image-proxy?url=${encodeURIComponent(url)}`} alt={`商品图片 ${index+1}`} width={120} height={120} className="aspect-square w-full rounded-md object-contain"/><div className="flex items-center justify-between text-xs"><span>{index===0?"主图":"图片 "+(index+1)}</span><Button type="button" size="icon" variant="ghost" aria-label={`移除图片 ${index+1}`} disabled={mediaBlocked} onClick={()=>removeImage(url)}><Trash2 size={14}/></Button></div></div>)}</div>:<p className="rounded-lg border border-dashed border-border p-5 text-center text-sm text-muted-foreground">尚未添加图片</p>}
      <label className="grid gap-2 text-sm">图片链接<Textarea rows={2} disabled={mediaBlocked} value={imageURL} placeholder="每行一个HTTPS图片链接" onChange={e=>setImageURL(e.target.value)}/></label>
      <Button type="button" variant="outline" disabled={!imageURL.trim()||mediaBlocked||images.length>=40} onClick={()=>addImages(imageURL.split(/\r?\n/).map(v=>v.trim()).filter(Boolean))}><Plus size={16}/>添加图片链接</Button>
      <SourceImageUploader scope={scope} allowNewUpload={images.length<40} disabled={commands.busy||!!commands.pending||!context.permissions.includes("workbench.collection.manage")} onBlocked={setMediaBlocked} onImage={url=>addImages([url])}/>
     </Card>
    </div>
    <Card className="space-y-5 p-6"><div className="flex flex-wrap items-start justify-between gap-3"><div><h2 className="text-lg font-semibold">商品属性</h2><p className="mt-1 text-sm text-muted-foreground">填写材质、产地、适用季节等同款通用信息。颜色、尺码等SKU差异请放在销售规格中。</p></div><Button type="button" variant="outline" disabled={attributes.length>=256} onClick={()=>setAttributes(prior=>[...prior,{id:crypto.randomUUID(),name:"",value:""}])}><Plus size={16}/>添加商品属性</Button></div>
     {attributes.length?<div className="space-y-3">{attributes.map((attribute,index)=><div key={attribute.id} className="grid items-end gap-3 sm:grid-cols-[minmax(120px,1fr)_minmax(0,3fr)_auto]"><label className="grid gap-2 text-sm">属性名称 {index+1}<Input maxLength={200} value={attribute.name} placeholder="例如：材质" onChange={e=>setAttributes(prior=>prior.map(a=>a.id===attribute.id?{...a,name:e.target.value}:a))}/></label><label className="grid gap-2 text-sm">属性值 {index+1}<Input maxLength={8192} value={attribute.value} placeholder="例如：100%棉" onChange={e=>setAttributes(prior=>prior.map(a=>a.id===attribute.id?{...a,value:e.target.value}:a))}/></label><Button type="button" size="icon" variant="ghost" aria-label={`删除商品属性 ${index+1}`} onClick={()=>setAttributes(prior=>prior.filter(a=>a.id!==attribute.id))}><Trash2 size={16}/></Button></div>)}</div>:<p className="text-sm text-muted-foreground">尚未填写商品属性，可按商品资料添加。</p>}
    </Card>
    <Card className="space-y-5 p-6"><div><h2 className="text-lg font-semibold">销售规格</h2><p className="mt-1 text-sm text-muted-foreground">选择单规格，或按颜色、尺码等规格生成多个SKU。</p></div>
     <div className="flex flex-wrap gap-4">{(["single","multiple"] as const).map(value=><label key={value} className="flex cursor-pointer items-center gap-2 rounded-lg border border-border bg-background px-4 py-3 text-sm"><input type="radio" name="variant-mode" checked={mode===value} onChange={()=>{setMode(value);setError("")}}/>{value==="single"?"单规格商品":"多规格商品"}</label>)}</div>
     {mode==="multiple"?<><div className="space-y-3">{specs.map((spec,index)=><div key={spec.id} className="grid items-end gap-3 sm:grid-cols-[minmax(120px,1fr)_minmax(0,3fr)_auto]"><label className="grid gap-2 text-sm">规格名称 {index+1}<Input value={spec.name} placeholder="例如：颜色" onChange={e=>setSpecs(prior=>prior.map(s=>s.id===spec.id?{...s,name:e.target.value}:s))}/></label><label className="grid gap-2 text-sm">规格值 {index+1}<Input value={spec.values} placeholder="例如：红色,蓝色（用逗号分隔）" onChange={e=>setSpecs(prior=>prior.map(s=>s.id===spec.id?{...s,values:e.target.value}:s))}/></label><Button type="button" variant="ghost" size="icon" aria-label={`删除规格 ${index+1}`} disabled={specs.length===1} onClick={()=>setSpecs(prior=>prior.filter(s=>s.id!==spec.id))}><Trash2 size={16}/></Button></div>)}</div>
      <div className="flex flex-wrap gap-3"><Button type="button" variant="outline" onClick={()=>setSpecs(prior=>[...prior,{id:crypto.randomUUID(),name:"",values:""}])}><Plus size={16}/>添加规格</Button><Button type="button" onClick={generate}>{rows.length?"更新变体组合":"生成变体组合"}</Button></div>
      <p className="text-xs text-muted-foreground">更新时保留相同组合已填写的资料，并移除不在当前规格中的组合。每个商品最多256个变体，规格与图片总量也受保存限制。</p>{stale?<p role="status" className="text-sm text-warning-foreground">规格已修改，请更新变体组合后保存。</p>:null}
     </>:<p className="text-sm text-muted-foreground">可先保存基础资料；填写SKU后，需要同时声明币种、价格和库存。</p>}
    </Card>
    {(mode==="single"||rows.length>0)?<Card className="overflow-hidden"><div className="space-y-4 border-b border-border p-6"><div className="flex flex-wrap items-center justify-between gap-2"><h2 className="text-lg font-semibold">{mode==="single"?"SKU资料":"变体资料"}</h2><span className="text-sm text-muted-foreground">{activeRows.length} 个{mode==="single"?"规格":"变体"}</span></div>
     {mode==="multiple"?<div className="rounded-lg border border-border bg-muted p-4"><p className="mb-3 text-sm font-medium">批量填写</p><div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5"><label className="grid gap-2 text-xs">SKU前缀<Input value={bulk.prefix} placeholder="选填，生成前缀-序号" onChange={e=>setBulk(v=>({...v,prefix:e.target.value}))}/></label><label className="grid gap-2 text-xs">公共币种<Input maxLength={3} value={bulk.currency} placeholder="例如 CNY" onChange={e=>setBulk(v=>({...v,currency:e.target.value.toUpperCase()}))}/></label><label className="grid gap-2 text-xs">公共价格<Input type="number" min="0" step="any" value={bulk.price} onChange={e=>setBulk(v=>({...v,price:e.target.value}))}/></label><label className="grid gap-2 text-xs">公共库存<Input type="number" min="0" step="1" value={bulk.stock} onChange={e=>setBulk(v=>({...v,stock:e.target.value}))}/></label><Button type="button" className="self-end" variant="outline" onClick={applyBulk}>应用到全部变体</Button></div><p className="mt-3 text-xs text-muted-foreground">仅应用已填写的公共值，会覆盖各行对应字段。价格和库存填0表示明确声明0。</p></div>:null}
     <p className="text-xs text-muted-foreground">价格和库存是你提供的来源资料，保存后仍需按目标平台要求适配。</p></div>
     <Table className="min-w-[960px]"><TableHeader className="bg-muted"><TableRow><TableHead>规格组合</TableHead>{edit?<TableHead>SKU名称</TableHead>:null}<TableHead>SKU</TableHead><TableHead>币种</TableHead><TableHead>价格</TableHead><TableHead>库存</TableHead><TableHead>变体图片</TableHead>{mode==="multiple"?<TableHead>操作</TableHead>:null}</TableRow></TableHeader><TableBody>{activeRows.map((row,index)=><TableRow key={row.id}><TableCell><span className="block max-w-52 break-words text-sm">{Object.values(row.attributes).join(" / ")||"默认规格"}</span></TableCell>{edit?<TableCell><Input aria-label={`SKU名称 ${index+1}`} className="min-w-36" maxLength={2000} value={row.title??""} onChange={e=>patchRow(row.id,{title:e.target.value})}/></TableCell>:null}<TableCell><Input aria-label={`SKU ${index+1}`} className="min-w-36" maxLength={200} value={row.sku} onChange={e=>patchRow(row.id,{sku:e.target.value})}/></TableCell><TableCell><Input aria-label={`币种 ${index+1}`} className="w-24" maxLength={3} placeholder="CNY" value={row.currency} onChange={e=>patchRow(row.id,{currency:e.target.value.toUpperCase()})}/></TableCell><TableCell><Input aria-label={`价格 ${index+1}`} className="w-28" type="number" min="0" step="any" value={row.price} onChange={e=>patchRow(row.id,{price:e.target.value})}/></TableCell><TableCell><Input aria-label={`库存 ${index+1}`} className="w-28" type="number" min="0" step="1" value={row.stock} onChange={e=>patchRow(row.id,{stock:e.target.value})}/></TableCell><TableCell><Select aria-label={`变体图片 ${index+1}`} className="min-w-32" value={row.image} onChange={e=>patchRow(row.id,{image:e.target.value})}><option value="">未指定</option>{row.originalImages?.filter(url=>!images.includes(url)).map((url,i)=><option key={url} value={url}>原SKU图片 {i+1}</option>)}{images.map((url,i)=><option key={url} value={url}>{i===0?"主图":"图片 "+(i+1)}</option>)}</Select>{edit&&row.originalImages?.length?<p className="mt-2 max-w-48 text-xs text-muted-foreground">原有{row.originalImages.length}张SKU图片。保持原选择会保留图组；改选会替换图组。</p>:null}</TableCell>{mode==="multiple"?<TableCell><Button type="button" variant="ghost" size="icon" aria-label={`删除变体 ${index+1}`} onClick={()=>setRows(prior=>prior.filter(v=>v.id!==row.id))}><Trash2 size={16}/></Button></TableCell>:null}</TableRow>)}</TableBody></Table>
    </Card>:null}
   </fieldset>
   <Card className="sticky bottom-3 z-10 flex flex-wrap items-center justify-between gap-4 border-border bg-card p-4 text-card-foreground shadow-lg"><div className="min-w-0"><p className="text-sm">{mode==="multiple"?`${rows.length} 个变体`:(single.sku?"1 个SKU":"基础商品资料")} · {images.length} 张公共图片</p><p className="mt-1 text-xs text-muted-foreground">保存到自有商品库，图片在适配时由你确认。</p>{error?<p role="alert" className="mt-2 text-sm text-destructive">{error}</p>:null}</div><Button type="submit" disabled={disabled||stale||mediaBlocked}>{commands.busy?"正在保存…":edit?"保存新版本":"保存商品"}</Button></Card>
  </form>
 </ConsolePage>;
}

function prepareImageURLs(urls:string[]){return urls.every(value=>{try{const url=new URL(value);return value.length<=2048&&url.protocol==="https:"&&!!url.hostname&&!url.username&&!url.password&&!url.hash}catch{return false}})}
