"use client";

import {useEffect,useMemo,useState} from "react";
import Image from "next/image";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {readCollectionItem,CollectionAPIError,type CollectionScope} from "@/lib/api/product-collection";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {collectionFailureText} from "./use-collection-command";

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
 const product=detail?.product;
 return <ConsolePage title={product?.title||"自有商品资料"} description="已保存的原始商品资料。后续适配保留此来源版本。" breadcrumbs={[{label:"数据服务",href:"/workbench/data"},{label:"我的数据",href:"/workbench/data/mine?view=own"},{label:"自有商品资料"}]} actions={<Button asChild variant="outline"><Link href="/workbench/data/mine?view=own">返回自有商品库</Link></Button>}>
  {error?<ConsoleState kind="error" title={collectionFailureText[error]??"暂时无法读取商品资料"}><Button variant="outline" onClick={()=>setReload(v=>v+1)}>重试读取</Button></ConsoleState>:!product?<ConsoleState kind="loading" title="正在读取商品资料"/>:<div className="space-y-6">
   <Card className="space-y-4 p-6"><h2 className="text-lg font-semibold">基本资料</h2><p className="text-sm">品牌：{product.brand||"未填写"}</p><p className="whitespace-pre-wrap text-sm text-muted-foreground">{product.description||"没有商品描述"}</p><div className="flex flex-wrap gap-3">{product.images?.map((image,index)=><Image key={`${image.url}:${index}`} unoptimized src={`/api/image-proxy?url=${encodeURIComponent(image.url)}`} alt={`商品图片 ${index+1}`} width={128} height={128} className="rounded-lg border border-border object-contain"/>)}</div><p className="break-all text-xs text-muted-foreground">原始版本 {detail.item.source.version} · {detail.item.source.productKey}</p></Card>
   <Card className="overflow-hidden"><div className="border-b border-border p-6"><h2 className="text-lg font-semibold">全部变体 · {product.variants?.length??0}</h2></div>{!product.variants?.length?<p className="p-6 text-sm text-muted-foreground">未录入SKU资料。</p>:<Table className="min-w-[760px]"><TableHeader className="bg-muted"><TableRow><TableHead>规格</TableHead><TableHead>SKU</TableHead><TableHead>币种</TableHead><TableHead>价格</TableHead><TableHead>库存</TableHead><TableHead>变体图片</TableHead></TableRow></TableHeader><TableBody>{product.variants.map((variant,index)=><TableRow key={variant.source_id??index}><TableCell>{variant.attributes?.map(a=>`${a.name}：${a.value}`).join(" / ")||variant.title||"默认规格"}</TableCell><TableCell>{variant.sku||"未填写"}</TableCell><TableCell>{variant.price?.currency??"未填写"}</TableCell><TableCell>{variant.price?variant.price.amount??0:"未填写"}</TableCell><TableCell>{variant.stock??0}</TableCell><TableCell><div className="flex gap-2">{variant.images?.length?variant.images.map((image,i)=><Image unoptimized key={`${image.url}:${i}`} src={`/api/image-proxy?url=${encodeURIComponent(image.url)}`} width={48} height={48} className="rounded-md border border-border object-contain" alt={`${variant.sku||index+1} 图片 ${i+1}`}/>):<span className="text-xs text-muted-foreground">未指定</span>}</div></TableCell></TableRow>)}</TableBody></Table>}</Card>
  </div>}
 </ConsolePage>;
}
