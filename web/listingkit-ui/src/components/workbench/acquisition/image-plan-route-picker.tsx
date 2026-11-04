"use client";
import {useEffect,useState} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {formatMicropoints} from "@/lib/contracts/ai-points";
import {imageSetRequest,type ImageSetScope} from "@/lib/api/product-image-set";
import {imageGenerationOptionsSchema,type ImageRouteSelection} from "@/lib/contracts/product-image-set";
export function ImagePlanRoutePicker({scope,value,disabled,onChange,onValidity}:{scope:ImageSetScope;value:ImageRouteSelection|null;disabled:boolean;onChange:(value:ImageRouteSelection|null)=>void;onValidity:(valid:boolean)=>void}){
 const[data,setData]=useState<z.infer<typeof imageGenerationOptionsSchema>>(),[failure,setFailure]=useState(false),[nonce,setNonce]=useState(0);
 useEffect(()=>{const abort=new AbortController();void imageSetRequest(scope,"model-options",imageGenerationOptionsSchema,{signal:abort.signal}).then(result=>{if(!abort.signal.aborted){setData(result);setFailure(false)}}).catch(()=>{if(!abort.signal.aborted){setData(undefined);setFailure(true)}});return()=>abort.abort()},[scope,nonce]);
 const selected=data?.options.length===1&&!data.options[0].route?data.options[0]:undefined,valid=!!selected&&!value;
 useEffect(()=>onValidity(valid),[valid,onValidity]);
 return <section aria-label="企业配置的图片模型" className="space-y-2 rounded-lg border border-border p-4"><label className="flex flex-col gap-2 text-sm font-medium">企业图片模型<input readOnly value={selected?.model??"尚未取得可用的企业模型"}/></label>{selected?<p className="text-xs text-muted-foreground">{selected.model} · {selected.quality} · 原始输出 1024×1024 · {selected.referencePriceVersion?`参考估算预留 ${formatMicropoints(selected.points)} 点；按本次用量及冻结参考价结算。`:`每张预留上界 ${formatMicropoints(selected.points)} 点；最终按完整实际用量结算。`}{selected.referencePriceVersion?<><br/>参考版本 {selected.referencePriceVersion}；缺输入细分时按较高输入参考费率估算。缺基础用量或超出预留保持待核实，不自动补扣或重试。</>:null}</p>:null}<p className="text-xs text-muted-foreground">模型由企业“模型与费率”统一配置，本页使用已发布模型。确认后准确模型、报价及技能版本固定。</p>{value?<><p role="status">原计划绑定 {value.BindingID} / {value.RevisionID} 只读保留，不跨配置继续使用。原请求须先核实；新计划可明确使用当前企业模型。</p><Button variant="outline" size="sm" disabled={disabled||!selected} onClick={()=>onChange(null)}>新计划使用企业模型</Button></>:null}{failure?<p role="status">企业图片模型读取失败，暂不可准备新计划；原请求和结果保留。</p>:null}<Button variant="outline" size="sm" disabled={disabled} onClick={()=>{setData(undefined);setNonce(v=>v+1)}}>重新读取图片模型</Button></section>;
}
