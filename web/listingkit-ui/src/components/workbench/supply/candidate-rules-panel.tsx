import type {MaterialsRun} from "@/lib/contracts/supply-chain";

const sections={category:"类目",fill:"填写规范",attributes:"属性模板",images:"图片要求",conditional:"关联条件"};
const statuses={available:"已取得",unknown:"未知",not_evaluated:"未评价",not_applicable:"不适用"};
const flag=(value:boolean|null)=>value===null?"未知":value?"是":"否";

export function CandidateRulesPanel({result}:{result:MaterialsRun}){
 const rules=result.queryRules,explanation=result.ruleExplanation;
 return <section className="space-y-5" aria-label="候选类目规则">
  <h3 className="font-semibold">候选类目规则</h3>
  <p>本次只查询并解释规则，不改变当前资料，不保存或发布。具体属性取值的关联条件仍未评价。</p>
  <p>候选类目：{rules?.categoryPath.map(v=>v.name||v.id).join(" / ")||result.candidateCategoryId}</p>
  {rules?<>
   <p className="text-sm text-muted-foreground">获取时间 {rules.evidence.readAt} · 官方规则版本未知 · 原资料 v{rules.binding.CandidateRules.BaseRevision}</p>
   <div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th>范围</th><th>状态</th><th>说明</th></tr></thead><tbody>{rules.coverage.map(v=><tr key={v.section} className="border-t border-border"><th className="p-3">{sections[v.section]}</th><td className="p-3">{statuses[v.status]}</td><td className="p-3">{v.reason}</td></tr>)}</tbody></table></div>
   <section className="space-y-2"><h4 className="font-semibold">官方基础填写要求</h4><p>默认语言 {rules.fill.default_language||"未知"} · 默认标题长度上限 {rules.fill.default_language_title_max_length??"未知"}</p>
    <div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th>字段</th><th>模块</th><th>显示</th><th>必填</th></tr></thead><tbody>{rules.fill.fill_in_standard_list.map((v,i)=><tr key={`${v.field_key}:${i}`} className="border-t border-border"><th className="p-3">{v.field_key}</th><td className="p-3">{v.module}</td><td className="p-3">{flag(v.show)}</td><td className="p-3">{flag(v.required)}</td></tr>)}</tbody></table></div>
   </section>
   <section className="space-y-2"><h4 className="font-semibold">官方属性模板</h4>{rules.attributes.attribute_infos.length?<ul className="list-inside list-disc">{rules.attributes.attribute_infos.map(v=><li key={v.attribute_id}>{v.attribute_name||`属性 ${v.attribute_id}`} · 显示标记 {v.attribute_is_show??"未知"} · 状态标记 {v.attribute_status??"未知"}{v.value_info_list.length?` · 候选值：${v.value_info_list.map(o=>o.attribute_value||String(o.attribute_value_id)).join("、")}`:""}</li>)}</ul>:<p>当前返回的模板没有属性条目；这不表示关联条件已完整评价。</p>}</section>
   <details><summary className="cursor-pointer">查看本次完整基础规则</summary><pre className="mt-3 max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-lg bg-secondary p-4 text-xs">{JSON.stringify({fill:rules.fill,attributes:rules.attributes,images:rules.images??null},null,2)}</pre></details>
   {!!rules.unknowns.length&&<section><h4 className="font-semibold">规则覆盖限制</h4><ul className="list-inside list-disc">{rules.unknowns.map((v,i)=><li key={i}>{v}</li>)}</ul></section>}
  </>:<p>本次尚未取得可披露的规则，请按原运行状态核对。</p>}
  {explanation&&<section className="space-y-2"><h4 className="font-semibold">智能体解释</h4><p className="whitespace-pre-wrap">{explanation.summary}</p><p className="text-sm text-muted-foreground">请结合上方实际规则核对。解释不代替规则事实或保存后的正式校验。</p>{!!explanation.unknowns.length&&<ul className="list-inside list-disc">{explanation.unknowns.map((v,i)=><li key={i}>{v}</li>)}</ul>}</section>}
 </section>;
}
