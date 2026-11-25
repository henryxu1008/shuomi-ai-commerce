"use client";
import {useEffect,useState,type ReactNode} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {formatMicropoints} from "@/lib/contracts/ai-points";
import {imageSetRequest,type ImageSetScope} from "@/lib/api/product-image-set";
import {imageGenerationOptionsSchema,providerImageUsageUnit,type ImageRouteSelection} from "@/lib/contracts/product-image-set";

function optionKey(route?:ImageRouteSelection|null){
 return route?`${route.BindingID}/${route.RevisionID}`:"default";
}

export function ImagePlanRoutePicker({scope,value,disabled,onChange,onValidity,onOwnedSelection,allowOwned=false,modelSettings}:{scope:ImageSetScope;value:ImageRouteSelection|null;disabled:boolean;onChange:(value:ImageRouteSelection|null)=>void;onValidity:(valid:boolean)=>void;onOwnedSelection?:(owned:boolean)=>void;allowOwned?:boolean;modelSettings?:ReactNode}){
 const[data,setData]=useState<z.infer<typeof imageGenerationOptionsSchema>>(),[failure,setFailure]=useState(false),[nonce,setNonce]=useState(0);
 useEffect(()=>{const abort=new AbortController();void imageSetRequest(scope,"model-options",imageGenerationOptionsSchema,{signal:abort.signal}).then(result=>{if(!abort.signal.aborted){setData(result);setFailure(false)}}).catch(()=>{if(!abort.signal.aborted){setData(undefined);setFailure(true)}});return()=>abort.abort()},[scope,nonce]);
 const options=data?.options.filter(option=>!option.route||allowOwned&&option.unitVersion===providerImageUsageUnit)??[];
 const selected=options.find(option=>optionKey(option.route)===optionKey(value));
 const valid=!!selected,owned=selected?.unitVersion===providerImageUsageUnit;
 const toapis=selected?.model==="gpt-image-2.6-flare"&&owned;
 const provider=toapis?"ToAPIs":"GRSai";
 const hasOwned=options.some(option=>option.unitVersion===providerImageUsageUnit);
 const defaultModel=options.find(option=>!option.route);
 useEffect(()=>onValidity(valid),[valid,onValidity]);
 useEffect(()=>onOwnedSelection?.(owned),[owned,onOwnedSelection]);
 return <section aria-label="企业配置的图片模型" className="space-y-2 rounded-lg border border-border p-4">
  <label className="flex flex-col gap-2 text-sm font-medium">企业图片模型{hasOwned?<select disabled={disabled} value={selected?optionKey(selected.route):""} onChange={event=>{
   const option=options.find(row=>optionKey(row.route)===event.target.value);
   if(option)onChange(option.route??null);
  }}>
   {!selected&&<option value="" disabled>选择图片模型</option>}
   {options.map(option=><option key={optionKey(option.route)} value={optionKey(option.route)}>{option.channelName?`${option.channelName} · ${option.model}`:`企业默认 · ${option.model}`}</option>)}
  </select>:<input readOnly value={defaultModel?.model??"尚未取得可用的企业模型"}/>}</label>
  {selected&&<p className="text-xs text-muted-foreground">{selected.model} · {selected.quality} · {toapis?"1K 档位，成品 1024×1024":owned?"成品 1024×1024":"原始输出 1024×1024"} · {owned?`企业向 ${provider} 付费，硕米不扣点。`:selected.referencePriceVersion?`参考估算预留 ${formatMicropoints(selected.points)} 点，按本次用量及冻结参考价结算。`:`每张预留上界 ${formatMicropoints(selected.points)} 点，最终按完整实际用量结算。`}{selected.referencePriceVersion?<><br/>参考版本 {selected.referencePriceVersion}。缺输入细分时按较高输入参考费率估算。缺基础用量或超出预留保持待核实。</>:null}</p>}
  <p className="text-xs text-muted-foreground">{owned?(toapis?"可生成主图、轮播图和详情图。确认后会将选用的 1688 公共原图链接发送给 ToAPIs。":"可生成主图、轮播图和详情图。确认后会将各项选用的原图发送给 GRSai。"):"确认后，本次使用的模型、报价和技能版本将固定。"}</p>
  {value&&!selected&&<><p role="status">原计划绑定 {value.BindingID} / {value.RevisionID} 只读保留。请核实原请求，再为新计划选择模型。</p>{!hasOwned&&<Button variant="outline" size="sm" disabled={disabled||!defaultModel} onClick={()=>onChange(null)}>新计划使用企业模型</Button>}</>}
  {failure&&<p role="status">企业图片模型读取失败。请重新读取，已有请求和结果仍可查看。</p>}
  {!selected&&(data||failure)&&modelSettings}<Button variant="outline" size="sm" disabled={disabled} onClick={()=>{setData(undefined);setNonce(v=>v+1)}}>重新读取图片模型</Button>
 </section>;
}
