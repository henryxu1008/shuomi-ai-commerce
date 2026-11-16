"use client";
import {useRef,useState} from "react";
import Link from "next/link";
import {ResourceDialog} from "@/components/workbench/resources/resource-dialog";
import {Input} from "@/components/ui/input";
import {Textarea} from "@/components/ui/textarea";
import {Button} from "@/components/ui/button";
import type {MarketScope} from "@/lib/api/supply-market";
import {useMarketCommands} from "./shared";

export function CatalogConnectionForm({scope,onClose}:{scope:MarketScope;onClose:()=>void}){
 const commands=useMarketCommands(scope),[input,setInput]=useState({name:"",website:"",contact:"",telephone:"",categories:""});
 const [submitting,setSubmitting]=useState(false);
 const busy=commands.busy||submitting;
 const nameRef=useRef<HTMLInputElement>(null);
 async function submit(){
  if(submitting||commands.locked||commands.saved)return;
  setSubmitting(true);
  try{await commands.execute({action:"submit_connection",connection:input})}finally{setSubmitting(false)}
 }
 const content=<div className="space-y-4">
  <p>填写货盘基本信息，提交后由专员人工评估并确认合作方案。方案确认后仍需完成实际接入。</p>
  {commands.notice}
  {commands.saved?.recordId?<p role="status">申请已保存。<Link href={"/workbench/supply/applications/"+commands.saved.recordId} prefetch={false}>查看对接进展</Link></p>:null}
  <form className="space-y-3" onSubmit={e=>{e.preventDefault();void submit()}}>
   <div className="grid gap-3 sm:grid-cols-2">{[{key:"name",label:"货盘名称",type:"text",max:200},{key:"website",label:"货盘网站（HTTPS）",type:"url",max:2048},{key:"contact",label:"联系人",type:"text",max:100},{key:"telephone",label:"联系电话",type:"tel",max:100}].map(f=><label key={f.key}>{f.label}<Input ref={f.key==="name"?nameRef:undefined} required type={f.type} maxLength={f.max} value={input[f.key as keyof typeof input]} onChange={e=>setInput({...input,[f.key]:e.target.value})}/></label>)}</div>
   <label className="block">供应品类<Textarea required maxLength={4000} value={input.categories} onChange={e=>setInput({...input,categories:e.target.value})}/></label>
   <div className="flex flex-wrap justify-end gap-3"><Button variant="outline" disabled={busy} type="button" onClick={onClose}>{commands.saved?"完成":"取消"}</Button><Button disabled={busy||commands.locked||!!commands.saved} type="submit">提交对接申请</Button></div>
  </form>
 </div>;
 return <ResourceDialog title="申请对接货盘" onClose={onClose} locked={busy} initialFocusRef={nameRef}>{content}</ResourceDialog>;
}
