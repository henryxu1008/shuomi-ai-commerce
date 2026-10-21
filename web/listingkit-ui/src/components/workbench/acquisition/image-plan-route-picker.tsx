"use client";
import {useEffect,useState} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {imageSetRequest,type ImageSetScope} from "@/lib/api/product-image-set";
import {imageGenerationOptionsSchema,type ImageRouteSelection} from "@/lib/contracts/product-image-set";

export function ImagePlanRoutePicker({scope,value,disabled,onChange,onValidity}:{scope:ImageSetScope;value:ImageRouteSelection|null;disabled:boolean;onChange:(value:ImageRouteSelection|null)=>void;onValidity:(valid:boolean)=>void}){
 const [data,setData]=useState<z.infer<typeof imageGenerationOptionsSchema>>(),[failure,setFailure]=useState(false),[nonce,setNonce]=useState(0);
 useEffect(()=>{const c=new AbortController();void imageSetRequest(scope,"model-options",imageGenerationOptionsSchema,{signal:c.signal}).then(v=>{if(!c.signal.aborted){setData(v);setFailure(false)}}).catch(()=>{if(!c.signal.aborted){setData(undefined);setFailure(true)}});return()=>c.abort()},[scope,nonce]);
 const key=value?JSON.stringify(value):"default",selected=data?.options.find(o=>(o.route?JSON.stringify(o.route):"default")===key);
 const valid=!value||!!selected;
 useEffect(()=>onValidity(valid),[valid,onValidity]);
 return <section aria-label="本次计划图片模型" className="space-y-2 rounded-lg border border-border p-4">
  <label className="flex flex-col gap-2 text-sm font-medium">本次计划使用的图片模型<select className="rounded-md border border-input bg-background p-2 text-foreground" value={key} disabled={disabled} onChange={e=>onChange(e.target.value==="default"?null:data?.options.find(o=>o.route&&JSON.stringify(o.route)===e.target.value)?.route??null)}>
   <option value="default">跟随企业默认{data?.options[0]?`（${data.options[0].model}）`:""}</option>
   {data?.options.filter(o=>o.route).map(o=><option key={o.route!.RevisionID} value={JSON.stringify(o.route)}>GPT Image VIP · {o.points} 点/张</option>)}
   {value&&!selected?<option value={key}>原选版本 {value.RevisionID}（尚未核实可用）</option>:null}
  </select></label>
  {selected?<p className="text-xs text-muted-foreground">{selected.model} · {selected.quality} · 原始输出 1024×1024 · {selected.points} 点/张{selected.route?` · 版本 ${selected.route.RevisionID} · 严格校验，不缩放`:""}</p>:null}
  <p className="text-xs text-muted-foreground">仅用于本次新计划，企业默认保持不变；确认计划后模型、费率和准确技能版本固定。</p>
  {failure?<p role="status" className="text-sm text-warning-foreground">图片模型选项读取失败。原选择保留；企业默认仍在准备计划时重新报价，VIP 须核实后才能准备。</p>:null}
  {!valid?<p role="status" className="text-sm text-warning-foreground">原选版本不可用或尚未核实，请读取后显式选择已发布版本；不会自动换模型。</p>:null}
  <Button variant="outline" size="sm" disabled={disabled} onClick={()=>setNonce(v=>v+1)}>重新读取图片模型</Button>
 </section>;
}
