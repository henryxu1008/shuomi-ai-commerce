import {Card} from "@/components/ui/card";
import {Badge} from "@/components/ui/badge";
import {Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from "@/components/ui/table";
import type {CompletionStructured} from "@/lib/contracts/completion-content";
import type {SupplyCompletionDraft} from "@/lib/contracts/supply-chain";

type Value=string|number|null|undefined;
function ValueText({value}:{value:Value}){return <span className="whitespace-pre-wrap break-words">{value===undefined?"未保存或无此项":value===null||value===""?"未填写":String(value)}</span>}
export function CompletionStructuredDetails({value}:{value?:CompletionStructured}){
 return <Card className="space-y-4 overflow-hidden p-5"><h2 className="font-semibold">候选商品属性和 SKU</h2>{!value?<p className="text-sm text-muted-foreground">未保存结构化候选资料</p>:<><section className="space-y-2"><h3 className="text-sm font-medium">候选商品属性</h3>{value.attributes.length?<dl className="grid gap-3 sm:grid-cols-2">{value.attributes.map((a,index)=><div key={index} className="rounded-lg border border-border p-3"><dt className="text-xs text-muted-foreground">{a.name}</dt><dd className="mt-1 text-sm"><ValueText value={a.value}/></dd></div>)}</dl>:<p className="text-sm text-muted-foreground">未添加候选属性。</p>}</section><section className="space-y-2"><h3 className="text-sm font-medium">候选 SKU 价格和库存</h3>{value.variants.length?<Table><TableHeader><TableRow><TableHead>固定来源 SKU</TableHead><TableHead>币种</TableHead><TableHead>价格</TableHead><TableHead>库存</TableHead></TableRow></TableHeader><TableBody>{value.variants.map(v=><TableRow key={v.sourceIndex}><TableCell>SKU {v.sourceIndex+1}</TableCell><TableCell><ValueText value={v.price?.currency??null}/></TableCell><TableCell><ValueText value={v.price?.amount??null}/></TableCell><TableCell><ValueText value={v.stock}/></TableCell></TableRow>)}</TableBody></Table>:<p className="text-sm text-muted-foreground">此保存版本没有 SKU。</p>}</section></>}</Card>;
}
export function CompletionStructuredComparison({records}:{records:SupplyCompletionDraft[]}){
 const [left,right]=records,a=left!.content.structured,b=right!.content.structured;
 const rows:{label:string;left:Value;right:Value}[]=[{label:"结构化候选资料",left:a?"已保存":"未保存",right:b?"已保存":"未保存"},{label:"候选属性数量",left:a?.attributes.length,right:b?.attributes.length}];
 for(let i=0;i<Math.max(a?.attributes.length??0,b?.attributes.length??0);i++){
  rows.push({label:`商品属性 ${i+1} 名称`,left:a?.attributes[i]?.name,right:b?.attributes[i]?.name},{label:`商品属性 ${i+1} 值`,left:a?.attributes[i]?.value,right:b?.attributes[i]?.value});
 }
 rows.push({label:"SKU 数量",left:a?.variants.length,right:b?.variants.length});
 for(let i=0;i<Math.max(a?.variants.length??0,b?.variants.length??0);i++){
  const l=a?.variants[i],r=b?.variants[i];
  rows.push({label:`SKU ${i+1} 币种`,left:l?(l.price?.currency??null):undefined,right:r?(r.price?.currency??null):undefined},{label:`SKU ${i+1} 价格`,left:l?(l.price?.amount??null):undefined,right:r?(r.price?.amount??null):undefined},{label:`SKU ${i+1} 库存`,left:l?.stock,right:r?.stock});
 }
 return <Card className="overflow-hidden"><div className="space-y-2 border-b border-border p-5"><h2 className="font-semibold">商品属性和 SKU 差异</h2><p className="text-sm text-muted-foreground">按保存版本中的属性位置和固定来源 SKU 比较。未保存、未填写和 0 分别显示；历史恢复仍只采用标题和描述。</p></div><Table className="min-w-[640px]"><TableHeader><TableRow><TableHead>字段</TableHead><TableHead>变化</TableHead><TableHead>R{left!.revision}</TableHead><TableHead>R{right!.revision}</TableHead></TableRow></TableHeader><TableBody>{rows.map(row=><TableRow key={row.label}><TableCell className="font-medium">{row.label}</TableCell><TableCell><Badge variant="secondary">{row.left===row.right?"未变化":"已变化"}</Badge></TableCell><TableCell className="w-[30%]"><ValueText value={row.left}/></TableCell><TableCell className="w-[30%]"><ValueText value={row.right}/></TableCell></TableRow>)}</TableBody></Table></Card>;
}
