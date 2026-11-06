"use client";
import type {PriceInput} from "@/lib/contracts/model-route-pricing";
import styles from "./model-route-pricing.module.css";
type Estimate=NonNullable<NonNullable<PriceInput["prices"][number]["image"]>["referenceEstimate"]>;
const labels={inputTokens:"预计输入 Token",outputTokens:"预计图片输出 Token",maximumPromptBytes:"适用提示词最多字节",maximumReferenceImages:"适用来源图最多张数",maximumReferenceBytes:"适用来源图合计字节",maximumReferencePixels:"适用来源图合计像素"} as const;
export function ImageReferenceEstimate({value,onChange}:{value?:Estimate;onChange:(value:Estimate|undefined)=>void}){
 return <section className={styles.history}><label className={styles.field}><span><input type="checkbox" checked={!!value} onChange={event=>onChange(event.target.checked?{inputTokens:0,outputTokens:0,operationScope:"product-image-effects-v1",size:"1024x1024",quality:"auto",allowAutoSize:true,maximumPromptBytes:0,maximumReferenceImages:0,maximumReferenceBytes:0,maximumReferencePixels:0}:undefined)}/>使用 Flare 图片参考估算</span></label><p>这是平台维护的参考参数，不是厂商费用硬上界。适用于原白底、来源图编辑和原创图，保留 auto 参数；缺参数时不可发布。</p>{value?<div className={styles.grid}>{Object.entries(labels).map(([key,label])=><label className={styles.field} key={key}><span>{label}</span><input inputMode="numeric" value={value[key as keyof typeof labels]||""} onChange={event=>onChange({...value,[key]:/^[1-9][0-9]{0,10}$/.test(event.target.value)?Number(event.target.value):0})}/></label>)}</div>:null}</section>;
}
