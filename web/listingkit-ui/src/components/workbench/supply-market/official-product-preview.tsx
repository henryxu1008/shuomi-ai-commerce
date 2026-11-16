/* eslint-disable @next/next/no-img-element */
import {Card} from "@/components/ui/card";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import type {readMarketRecord} from "@/lib/api/supply-market";

export function OfficialProductPreview({product}:{product:NonNullable<Awaited<ReturnType<typeof readMarketRecord>>["product"]>}){
 return <Card className="space-y-5 p-6">
  <h2 className="text-lg font-semibold">商品公开内容</h2>
  <div className="grid gap-6 md:grid-cols-2">
   <div className="space-y-3"><img src={product.images[0]} alt={product.title} className="aspect-square max-h-96 w-full rounded-md border border-border object-contain"/>{product.images.length>1?<div className="grid grid-cols-4 gap-2">{product.images.slice(1).map((src,i)=><img key={src} src={src} alt={`商品图片 ${i+2}`} className="aspect-square w-full rounded-md border border-border object-contain"/>)}</div>:null}</div>
   <div className="min-w-0 space-y-4 break-words"><h3 className="text-lg font-medium">{product.title}</h3>{product.brand?<p>品牌：{product.brand}</p>:null}{product.description?<p className="whitespace-pre-wrap">{product.description}</p>:null}<dl className="space-y-2">{Object.entries(product.attributes??{}).map(([name,value])=><div key={name} className="flex flex-wrap gap-2"><dt className="text-muted-foreground">{name}</dt><dd>{value}</dd></div>)}</dl></div>
  </div>
  {product.variants?.length?<Table><TableHeader><TableRow><TableHead>SKU / 款式</TableHead><TableHead>规格</TableHead><TableHead>价格</TableHead><TableHead>库存</TableHead><TableHead>款式图片</TableHead></TableRow></TableHeader><TableBody>{product.variants.map(v=><TableRow key={v.sourceId}><TableCell>{v.sku||v.title||v.sourceId}</TableCell><TableCell>{Object.entries(v.attributes??{}).map(([name,value])=>`${name}：${value}`).join(" / ")||"—"}</TableCell><TableCell>{v.currency&&v.price!==undefined?`${v.currency} ${v.price}`:"未提供"}</TableCell><TableCell>{v.stock}</TableCell><TableCell><div className="flex gap-2">{v.images?.map(src=><img key={src} src={src} alt={`${v.sku||v.title||v.sourceId} 款式图`} className="h-16 w-16 object-contain"/>)}</div></TableCell></TableRow>)}</TableBody></Table>:null}
 </Card>;
}
