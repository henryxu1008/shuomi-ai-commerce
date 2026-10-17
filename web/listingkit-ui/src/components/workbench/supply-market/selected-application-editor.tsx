"use client";
import {useCallback,useEffect,useState} from "react";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import type {MarketScope} from "@/lib/api/supply-market";
import {Qualifications} from "./qualifications";
import {SupplyLocationFields} from "./supply-location-fields";
import type {CommonSupply,SelectedDraft} from "./selected-batch-intent";
import styles from "./selected-application-form.module.css";

function ItemQualifications({scope,draft,hidden,onDraft,onLock}:{scope:MarketScope;draft:SelectedDraft;hidden:boolean;onDraft:(id:string,patch:Partial<SelectedDraft>)=>void;onLock:(id:string,locked:boolean)=>void}){
  const id=draft.selection.itemId,report=useCallback((locked:boolean)=>onLock(id,locked),[id,onLock]);
  return <div hidden={hidden} className={styles.upload}><h3>{draft.title}的商品专属资质附件</h3><Qualifications namespace={"qualification-item-"+id} scope={scope} ids={draft.fileIds} onIDs={ids=>onDraft(id,{fileIds:ids})} onLocked={report}/></div>;
}

export function SelectedApplicationEditor({scope,drafts,onDraft,common,onCommon,commonFiles,onCommonFiles,locked,uploadLocked,onUploadLocked,onReview,onBack}:{
  scope:MarketScope;drafts:SelectedDraft[];onDraft:(id:string,patch:Partial<SelectedDraft>)=>void;common:CommonSupply;onCommon:(value:CommonSupply)=>void;
  commonFiles:string[];onCommonFiles:(ids:string[])=>void;locked:boolean;uploadLocked:boolean;onUploadLocked:(locked:boolean)=>void;onReview:()=>void;onBack:()=>void;
}){
  const [activeId,setActiveId]=useState(drafts[0]?.selection.itemId);
  const [commonLocked,setCommonLocked]=useState(false),[itemLocks,setItemLocks]=useState<Record<string,boolean>>({});
  const reportItem=useCallback((id:string,locked:boolean)=>setItemLocks(previous=>previous[id]===locked?previous:{...previous,[id]:locked}),[]);
  useEffect(()=>onUploadLocked(commonLocked||drafts.some(d=>itemLocks[d.selection.itemId])),[commonLocked,itemLocks,drafts,onUploadLocked]);
  const active=drafts.find(d=>d.selection.itemId===activeId)??drafts[0];
  if(!active)return null;
  const update=(patch:Partial<SelectedDraft>)=>onDraft(active.selection.itemId,patch);
  return <form onSubmit={e=>{e.preventDefault();if(!locked&&!uploadLocked)onReview()}}>
    <fieldset disabled={locked} className={styles.fields}>
      <Card className={styles.formCard}><h2>公共信息</h2><p>作为默认值，商品可分别取消继承。库存、起订量、交期和价格须逐件填写。</p>
        <div className={styles.inputGrid}><SupplyLocationFields {...common} required={drafts.some(d=>d.inheritLocation)} onChange={location=>onCommon({...common,...location})}/></div>
        <label>公共售后政策<Textarea maxLength={4000} value={common.afterSaleNote} onChange={e=>onCommon({...common,afterSaleNote:e.target.value})}/></label>
        <div className={styles.upload}><h3>通用资质附件</h3><Qualifications scope={scope} ids={commonFiles} onIDs={onCommonFiles} onLocked={setCommonLocked}/></div>
      </Card>
      <Card className={styles.formCard}><h2>逐商品供货与资质</h2><p>每件独立申请、独立审核；公共信息不会复制为其它商品的库存或价格。</p>
        <label>正在编辑商品<Select value={active.selection.itemId} disabled={uploadLocked} onChange={e=>setActiveId(e.target.value)}>{drafts.map(d=><option key={d.selection.itemId} value={d.selection.itemId}>{d.title}</option>)}</Select></label>
        {drafts.filter(d=>d.selection.itemId!==active.selection.itemId&&itemLocks[d.selection.itemId]).map(d=><Button type="button" key={d.selection.itemId} variant="outline" disabled={commonLocked||itemLocks[active.selection.itemId]} onClick={()=>setActiveId(d.selection.itemId)}>继续处理 {d.title} 的原附件</Button>)}
        <fieldset aria-label={"商品供货："+active.title} className={styles.itemFields}>
          <legend>{active.title}</legend><div className={styles.inputGrid}>
            <label>可供库存（件）<Input type="number" required min={0} max={1e9} value={active.supply.stock} onChange={e=>update({supply:{...active.supply,stock:Number(e.target.value)}})}/></label>
            <label>最小起订量（件）<Input type="number" required min={1} max={1e9} value={active.supply.minimumQuantity} onChange={e=>update({supply:{...active.supply,minimumQuantity:Number(e.target.value)}})}/></label>
            <label>发货时效（天）<Input type="number" required min={0} max={3650} value={active.supply.leadDays} onChange={e=>update({supply:{...active.supply,leadDays:Number(e.target.value)}})}/></label>
            <label>生产能力<Input required={active.supply.stock===0} maxLength={4000} value={active.supply.capacity} onChange={e=>update({supply:{...active.supply,capacity:e.target.value}})}/></label>
          </div>
          <label>供货价说明<Textarea maxLength={4000} value={active.supply.priceNote} onChange={e=>update({supply:{...active.supply,priceNote:e.target.value}})}/></label>
          <label className={styles.check}><input type="checkbox" checked={active.inheritLocation} onChange={e=>update({inheritLocation:e.target.checked,supply:{...active.supply,...(active.inheritLocation?{province:common.province,city:common.city}:{})}})}/>继承公共发货地</label>
          {!active.inheritLocation?<div className={styles.inputGrid}><SupplyLocationFields province={active.supply.province} city={active.supply.city} onChange={location=>update({supply:{...active.supply,...location}})}/></div>:<p>发货地：{common.province||"未填写"} {common.city}</p>}
          <label className={styles.check}><input type="checkbox" checked={active.inheritAfterSale} onChange={e=>update({inheritAfterSale:e.target.checked,supply:{...active.supply,...(active.inheritAfterSale?{afterSaleNote:common.afterSaleNote}:{})}})}/>继承公共售后政策</label>
          {!active.inheritAfterSale?<label>商品售后政策<Textarea maxLength={4000} value={active.supply.afterSaleNote} onChange={e=>update({supply:{...active.supply,afterSaleNote:e.target.value}})}/></label>:<p>售后：{common.afterSaleNote||"未填写"}</p>}
          <label className={styles.check}><input type="checkbox" checked={active.useCommonFiles} disabled={uploadLocked} onChange={e=>update({useCommonFiles:e.target.checked})}/>引用通用资质附件</label>
        </fieldset>
        {drafts.map(draft=><ItemQualifications key={draft.selection.itemId} scope={scope} draft={draft} hidden={draft.selection.itemId!==active.selection.itemId} onDraft={onDraft} onLock={reportItem}/>)}
      </Card>
    </fieldset>
    <Card className={styles.footer}><p>离开前未提交的信息暂不保存。每件合并附件后需 1–6 件；提交后逐件独立审核。</p><div><Button type="button" variant="outline" disabled={locked||uploadLocked} onClick={onBack}>返回选择商品</Button><Button type="submit" disabled={locked||uploadLocked}>核对本批申请</Button></div></Card>
  </form>;
}
