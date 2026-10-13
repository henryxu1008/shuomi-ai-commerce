"use client";

import {useEffect,useMemo,useState} from "react";
import Image from "next/image";
import Link from "next/link";
import {ArrowLeft,ChevronLeft,ChevronRight,ImageOff,Package} from "lucide-react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {readCollectionItem,CollectionAPIError,type CollectionScope} from "@/lib/api/product-collection";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
import {Button} from "@/components/ui/button";
import {Badge} from "@/components/ui/badge";
import {Card} from "@/components/ui/card";
import {Select} from "@/components/ui/select";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import {cn} from "@/lib/utils/cn";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {collectionFailureText} from "./use-collection-command";

const library="/workbench/data/mine?view=own";
type Product=CollectionDetail["product"];
type Variant=NonNullable<Product["variants"]>[number];
const imageProxy=(url:string)=>`/api/image-proxy?url=${encodeURIComponent(url)}`;
const variantLabel=(variant:Variant)=>variant.attributes?.map(a=>`${a.name}：${a.value}`).join(" / ")||variant.title||"默认规格";

export function OwnProductDetailPage({itemId}:{itemId:string}){
 const context=useWorkbenchContext();
 const userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.selectionRequired||context.error||context.blockingError)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 return <ScopedOwnProductDetail key={`${scope.userId}:${scope.organizationId}:${itemId}`} scope={scope} itemId={itemId}/>;
}
function ScopedOwnProductDetail({scope,itemId}:{scope:CollectionScope;itemId:string}){
 const [detail,setDetail]=useState<CollectionDetail|null>(null),[error,setError]=useState(""),[reload,setReload]=useState(0);
 useEffect(()=>{const controller=new AbortController();let active=true;void readCollectionItem(scope,itemId,controller.signal).then(value=>{if(active){if(value.item.source.kind!=="own")setError("NOT_FOUND");else {setDetail(value);setError("")}}}).catch(failure=>{if(active)setError(failure instanceof CollectionAPIError?failure.code:"DEPENDENCY_UNAVAILABLE")});return()=>{active=false;controller.abort()}},[scope,itemId,reload]);
 return <ConsolePage title={detail?.product.title||"自有商品资料"} description="查看商品图片、通用属性和每个SKU的原始资料。" breadcrumbs={[{label:"数据服务",href:"/workbench/data"},{label:"我的数据",href:library},{label:"自有商品资料"}]} actions={<Button asChild variant="outline"><Link href={library}><ArrowLeft size={16}/>返回自有商品库</Link></Button>}>
  {error?<ConsoleState kind="error" title={collectionFailureText[error]??"暂时无法读取商品资料"}><Button variant="outline" onClick={()=>setReload(v=>v+1)}>重试读取</Button></ConsoleState>:!detail?<ConsoleState kind="loading" title="正在读取商品资料"/>:<OwnProductContents key={`${detail.item.id}:${detail.item.source.version}`} detail={detail}/>}
 </ConsolePage>;
}

function OwnProductContents({detail}:{detail:CollectionDetail}){
 const {product,item}=detail,variants=product.variants??[],attributes=product.attributes??[];
 const [selectedIndex,setSelectedIndex]=useState(0),[selectedImage,setSelectedImage]=useState("");
 const selected=variants[selectedIndex];
 const images=Array.from(new Set([...(selected?.images??[]),...(product.images??[])].map(image=>image.url)));
 const activeImage=images.includes(selectedImage)?selectedImage:images[0];
 function selectVariant(index:number,image=""){setSelectedIndex(index);setSelectedImage(image)}
 const imageCount=new Set([...(product.images??[]),...variants.flatMap(v=>v.images??[])].map(image=>image.url)).size;
 return <div className="space-y-6">
  <div className="flex flex-wrap items-center justify-between gap-3"><div className="flex flex-wrap items-center gap-2"><Badge variant="secondary">自有商品</Badge><Badge variant="outline">资料已保存</Badge><span className="text-sm text-muted-foreground">{variants.length} 个SKU · {attributes.length} 项商品属性 · {imageCount} 张图片</span></div><nav aria-label="商品详情章节" className="flex gap-5 text-sm text-muted-foreground"><a className="hover:text-foreground" href="#own-product-variants">SKU资料</a><a className="hover:text-foreground" href="#own-product-attributes">商品属性</a><a className="hover:text-foreground" href="#own-product-description">商品描述</a></nav></div>
  <div className="grid items-start gap-6 xl:grid-cols-[minmax(280px,2fr)_minmax(0,3fr)]">
   <ProductGallery images={images} url={activeImage} onSelect={setSelectedImage}/>
   <Card className="space-y-6 p-6" role="region" aria-label="选中SKU资料">
    <div className="flex items-start justify-between gap-4"><div><h2 className="text-lg font-semibold">商品概览</h2><p className="mt-1 text-sm text-muted-foreground">{product.category_path?.length?product.category_path.join(" / "):"已保存的自有商品原始资料"}</p></div><Package size={24} className="shrink-0 text-muted-foreground" aria-hidden="true"/></div>
    <dl className="grid gap-4 sm:grid-cols-2"><div><dt className="text-xs text-muted-foreground">品牌</dt><dd className="mt-1 text-sm font-medium">{product.brand||"未填写"}</dd></div><div><dt className="text-xs text-muted-foreground">保存时间</dt><dd className="mt-1 text-sm">{item.createdAt?new Date(item.createdAt).toLocaleString("zh-CN"):"未提供"}</dd></div></dl>
    {selected?<><label className="grid gap-2 text-sm">当前SKU<Select value={String(selectedIndex)} onChange={event=>selectVariant(Number(event.target.value))}>{variants.map((variant,index)=><option key={index} value={String(index)}>{variant.sku||"未填写SKU"} · {variantLabel(variant)}</option>)}</Select></label>
     {selected.attributes?.length?<div className="flex flex-wrap gap-2">{selected.attributes.map((attribute,index)=><Badge key={index} variant="outline" className="whitespace-normal break-words">{attribute.name}：{attribute.value}</Badge>)}</div>:null}
     <div className="grid gap-5 rounded-xl border border-border bg-muted/40 p-5 sm:grid-cols-2"><div><p className="text-xs text-muted-foreground">来源价格</p><p className="mt-2 text-3xl font-semibold tracking-tight">{selected.price?.amount!==undefined?`${selected.price.currency||""} ${selected.price.amount}`.trim():"未填写"}</p></div><div><p className="text-xs text-muted-foreground">来源库存</p><p className="mt-2 text-3xl font-semibold tracking-tight">{selected.stock??0} 件</p></div></div>
     <dl className="grid gap-4 sm:grid-cols-2"><div><dt className="text-xs text-muted-foreground">SKU编码</dt><dd className="mt-1 break-all text-sm font-medium">{selected.sku||"未填写"}</dd></div>{selected.barcode?<div><dt className="text-xs text-muted-foreground">商品条码</dt><dd className="mt-1 break-all text-sm">{selected.barcode}</dd></div>:null}</dl>
    </>:<p className="rounded-lg border border-dashed border-border p-5 text-sm text-muted-foreground">只保存了基础资料，尚未录入SKU。</p>}
    <p className="border-t border-border pt-4 text-xs leading-5 text-muted-foreground">价格和库存是录入时提供的来源资料，后续仍需按目标平台要求适配。查看或保存资料不会自动发布商品。</p>
   </Card>
  </div>
  <Card id="own-product-variants" className="scroll-mt-6 overflow-hidden"><div className="flex flex-wrap items-center justify-between gap-3 border-b border-border p-6"><div><h2 className="text-lg font-semibold">全部变体 · {variants.length}</h2><p className="mt-1 text-sm text-muted-foreground">选择一个SKU，在上方查看它的图片与资料。</p></div></div>{!variants.length?<p className="p-6 text-sm text-muted-foreground">未录入SKU资料。</p>:<Table className="min-w-[800px]"><TableHeader className="bg-muted"><TableRow><TableHead>规格</TableHead><TableHead>SKU</TableHead><TableHead>币种</TableHead><TableHead>价格</TableHead><TableHead>库存</TableHead><TableHead>变体图片</TableHead><TableHead>查看</TableHead></TableRow></TableHeader><TableBody>{variants.map((variant,index)=><TableRow key={index} data-state={index===selectedIndex?"selected":undefined}><TableCell className="max-w-64 whitespace-normal break-words">{variantLabel(variant)}</TableCell><TableCell className="max-w-52 break-all">{variant.sku||"未填写"}</TableCell><TableCell>{variant.price?.currency??"未填写"}</TableCell><TableCell>{variant.price?.amount??"未填写"}</TableCell><TableCell>{variant.stock??0}</TableCell><TableCell><div className="flex gap-2">{variant.images?.length?variant.images.map((image,i)=><button type="button" key={`${image.url}:${i}`} className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring" aria-label={`查看${variant.sku||index+1}图片${i+1}`} onClick={()=>selectVariant(index,image.url)}><Image unoptimized src={imageProxy(image.url)} width={48} height={48} className="h-12 w-12 rounded-md border border-border object-contain" alt={`${variant.sku||index+1} 图片 ${i+1}`}/></button>):<span className="text-xs text-muted-foreground">未指定</span>}</div></TableCell><TableCell><Button type="button" variant={index===selectedIndex?"secondary":"outline"} size="sm" aria-pressed={index===selectedIndex} onClick={()=>selectVariant(index)}>{index===selectedIndex?"正在查看":"查看SKU"}</Button></TableCell></TableRow>)}</TableBody></Table>}</Card>
  <div className="grid items-start gap-6 lg:grid-cols-2">
   <Card id="own-product-attributes" className="scroll-mt-6 overflow-hidden" role="region" aria-label="商品属性"><div className="border-b border-border p-6"><h2 className="text-lg font-semibold">商品属性</h2><p className="mt-1 text-sm text-muted-foreground">同款商品的通用信息，与SKU销售规格分别保存。</p></div>{attributes.length?<dl className="divide-y divide-border">{attributes.map((attribute,index)=><div key={index} className="grid grid-cols-[minmax(80px,1fr)_minmax(0,2fr)] gap-4 px-6 py-4"><dt className="break-words text-sm text-muted-foreground">{attribute.name||"未命名属性"}</dt><dd className="whitespace-pre-wrap break-words text-sm">{attribute.value||"未填写"}</dd></div>)}</dl>:<p className="p-6 text-sm text-muted-foreground">尚未填写商品属性。</p>}</Card>
   <Card id="own-product-description" className="scroll-mt-6 space-y-4 p-6"><h2 className="text-lg font-semibold">商品描述</h2><p className="whitespace-pre-wrap break-words text-sm leading-7 text-muted-foreground">{product.description||"尚未填写商品描述。"}</p>{product.selling_points?.length?<ul className="list-disc space-y-2 pl-5 text-sm">{product.selling_points.map((point,index)=><li key={index}>{point}</li>)}</ul>:null}</Card>
  </div>
  <Card className="space-y-4 p-6"><h2 className="text-base font-semibold">保存信息</h2><dl className="grid gap-4 text-sm sm:grid-cols-2"><div><dt className="text-xs text-muted-foreground">资料版本</dt><dd className="mt-1">{item.source.version}</dd></div><div><dt className="text-xs text-muted-foreground">来源标识</dt><dd className="mt-1 break-all">{item.source.productKey}</dd></div></dl></Card>
 </div>;
}

function ProductGallery({images,url,onSelect}:{images:string[];url:string|undefined;onSelect:(url:string)=>void}){
 const [failedURL,setFailedURL]=useState("");
 const index=url?images.indexOf(url):0,failed=!!url&&failedURL===url;
 const setFailed=(failed:boolean)=>setFailedURL(failed?url??"":"");
 function selectImage(next:number){onSelect(images[next]!)}
 return <Card className="overflow-hidden" role="region" aria-label="商品图片"><div className="flex items-center justify-between border-b border-border px-5 py-4"><h2 className="text-base font-semibold">图片浏览</h2><span className="text-xs text-muted-foreground">{url?`${index+1} / ${images.length}`:"没有图片"}</span></div><div className="p-5"><div className="relative flex aspect-[4/3] w-full max-h-[440px] items-center justify-center overflow-hidden rounded-lg border border-border bg-muted/30">{url&&!failed?<Image key={url} unoptimized src={imageProxy(url)} alt="商品大图" width={640} height={640} className="h-full w-full object-contain" onError={()=>setFailed(true)}/>:<div className="space-y-3 p-6 text-center text-muted-foreground"><ImageOff size={36} className="mx-auto" aria-hidden="true"/><p className="text-sm">{url?"图片暂时无法加载":"尚未添加商品图片"}</p>{url?<Button type="button" variant="outline" size="sm" onClick={()=>setFailed(false)}>重试加载图片</Button>:null}</div>}</div>
   {url?<><div className="mt-3 flex items-center justify-between gap-3"><Button type="button" size="icon" variant="outline" aria-label="上一张图片" disabled={index===0} onClick={()=>selectImage(index-1)}><ChevronLeft size={18}/></Button><a className="text-xs text-primary underline-offset-4 hover:underline" href={imageProxy(url)} target="_blank" rel="noopener noreferrer">打开图片</a><Button type="button" size="icon" variant="outline" aria-label="下一张图片" disabled={index===images.length-1} onClick={()=>selectImage(index+1)}><ChevronRight size={18}/></Button></div><div className="mt-4 flex gap-2 overflow-x-auto pb-1">{images.map((image,i)=><button type="button" key={image} aria-label={`查看图片 ${i+1}`} aria-pressed={i===index} className={cn("h-16 w-16 shrink-0 overflow-hidden rounded-md border-2 p-1 outline-none focus-visible:ring-2 focus-visible:ring-ring",i===index?"border-primary bg-primary/5":"border-border hover:border-primary/50")} onClick={()=>selectImage(i)}><Image unoptimized src={imageProxy(image)} width={56} height={56} className="h-full w-full object-contain" alt=""/></button>)}</div></>:null}
  </div></Card>;
}
