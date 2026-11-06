import type {ImageAdvice} from "@/lib/contracts/image-advice";
const roles:Record<string,string>={main:"主图",design:"设计图",white_background:"白底图",gallery:"附图"};
const states={known_satisfied:"已核对该项",known_conflict:"已知待处理",unknown:"待核对",not_applicable:"本次未评价"};
export function ImageAdvicePanel({advice}:{advice?:ImageAdvice}){
 if(!advice)return <section aria-label="图片事实与建议"><h3 className="font-semibold">图片事实与建议</h3><p>本次记录未提供图片检查。</p></section>;
 return <section aria-label="图片事实与建议" className="space-y-3">
  <h3 className="font-semibold">图片事实与建议</h3>
  {!!advice.facts.length&&<ul className="space-y-1">{advice.facts.map(f=><li key={f.assetId}>已批准{roles[f.role]??"素材"} · {f.assetId} · {f.dimensionsStatus==="known"?`${f.width} × ${f.height}`:"真实尺寸尚未核实"}</li>)}</ul>}
  {advice.checks.length>0&&<ul className="space-y-1">{advice.checks.map(c=><li key={c.id}><span>{states[c.status]}：</span>{c.reason}</li>)}</ul>}
  {advice.suggestions.length>0&&<div><h4 className="font-medium">建议下一步</h4><ul className="list-inside list-disc">{advice.suggestions.map((s,i)=><li key={i}>{s.reason}</li>)}</ul></div>}
  <p className="text-sm text-muted-foreground">这些建议不会准备任务、生成、保存或批准图片。后续操作请在原图片入口核对后确认。</p>
 </section>;
}
