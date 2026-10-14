"use client";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import type {TemuDraftInput,TemuRuleSnapshot} from "@/lib/contracts/temu-model";

type Values=NonNullable<NonNullable<TemuDraftInput["local"]>["goodsProperty"]>;

export function TemuLocalAttributes({rules,values,disabled,onChange}:{rules?:TemuRuleSnapshot;values:Values;disabled:boolean;onChange:(values:Values)=>void}) {
 const attrs=rules?.localAttributes?.attributeList??[];
 const childRefs=new Set<string>(),active=new Map<string,Set<string>>();
 for(const attr of attrs)for(const group of attr.attributeValueDetail??[])for(const option of group.attributeValueList??[])for(const child of option.childAttribute??[]){
  if(!child.childAttributeId)continue;
  childRefs.add(child.childAttributeId);
  if(values.some(v=>v.refPid===attr.refPid&&v.vid===option.vid)){
   const groups=active.get(child.childAttributeId)??new Set<string>();groups.add(child.childAttributeValueGroupId);active.set(child.childAttributeId,groups);
  }
 }
 function update(index:number,edit:(v:Values[number])=>void){const copy=structuredClone(values);edit(copy[index]!);onChange(copy)}
 if(!attrs.length)return null;
 return <section className="space-y-4"><h3 className="font-semibold">类目属性</h3>{attrs.filter(a=>a.refPid).map(attr=>{
  const ref=attr.refPid!,name=attr.attributeName,control=attr.attributeRules?.controlType;
  const enabled=!childRefs.has(ref)||!!active.get(ref)?.size;
  const known=["0","1","5","6","7","8","16","19"].includes(control??"");
  const options=attr.attributeValueDetail?.filter(group=>!childRefs.has(ref)||active.get(ref)?.has(group.groupId)).flatMap(group=>group.attributeValueList??[])??[];
  const rows=values.map((value,index)=>({value,index})).filter(row=>row.value.refPid===ref);
  const max=attr.attributeRules?.chooseMaxNum;
  const canAdd=enabled&&known&&!!max&&BigInt(max)>BigInt(rows.length);
  return <div key={ref} className="space-y-2 rounded-lg border p-3"><p className="text-sm font-medium">{name}{attr.required||attr.attributeRules?.transnationalAttribute?" *":""}</p>
   {!enabled?<p className="text-xs text-muted-foreground">先选择启用此属性的父属性值。</p>:!known?<p className="text-xs text-amber-700">当前属性输入规则尚未核实。</p>:null}
   {rows.map(({value,index},i)=><div key={index} className="grid gap-2 sm:grid-cols-3">
    <label className="grid gap-1.5 text-sm">{name} 值 {i+1}{["1","16","19"].includes(control??"")?<select className="h-10 rounded-md border px-3" disabled={disabled||!enabled||!known} value={value.vid??(control==="19"&&value.value!==undefined?"custom":"")} onChange={e=>update(index,v=>{delete v.vid;delete v.value;if(e.target.value==="custom")v.value="";else if(e.target.value)v.vid=e.target.value})}><option value="">选择官方值</option>{options.filter(v=>v.vid).map(option=><option key={option.vid} value={option.vid!}>{option.value}</option>)}{control==="19"?<option value="custom">自定义值</option>:null}</select>:<Input disabled={disabled||!enabled||!known} value={value.value??""} onChange={e=>update(index,v=>{v.value=e.target.value;delete v.vid})}/>}</label>
    {control==="19"&&value.value!==undefined&&!value.vid?<label className="grid gap-1.5 text-sm">{name} 自定义值 {i+1}<Input disabled={disabled||!enabled} value={value.value} onChange={e=>update(index,v=>{v.value=e.target.value})}/></label>:null}
    {control==="16"?<label className="grid gap-1.5 text-sm">{name} 数值 {i+1}<Input disabled={disabled||!enabled} inputMode="decimal" value={value.numberInputValue??""} onChange={e=>update(index,v=>{v.numberInputValue=e.target.value})}/></label>:null}
    {attr.attributeValueUnitList?.length?<label className="grid gap-1.5 text-sm">{name} 单位 {i+1}<select className="h-10 rounded-md border px-3" disabled={disabled||!enabled} value={value.valueUnitId??""} onChange={e=>update(index,v=>{if(e.target.value)v.valueUnitId=e.target.value;else delete v.valueUnitId})}><option value="">选择单位</option>{attr.attributeValueUnitList.filter(u=>u.valueUnitId).map(unit=><option key={unit.valueUnitId} value={unit.valueUnitId!}>{unit.valueUnitName}</option>)}</select></label>:null}
    <Button variant="ghost" size="sm" disabled={disabled} onClick={()=>onChange(values.filter((_,j)=>j!==index))}>移除 {name} 值 {i+1}</Button>
   </div>)}
   <Button variant="outline" size="sm" disabled={disabled||!canAdd} onClick={()=>onChange([...values,{refPid:ref}])}>添加 {name} 属性值</Button>
  </div>;
 })}</section>;
}
