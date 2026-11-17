"use client";

import {useState} from "react";
import Link from "next/link";
import {AccountDialog} from "@/components/workbench/account/account-dialog";
import {ConsolePage,ConsoleState} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import type {MarketScope} from "@/lib/api/supply-market";
import type {MarketCategory,MarketCommand} from "@/lib/contracts/supply-market";
import {useManagedCategoryLevel} from "./category-fields";
import {MarketBoundary,MarketReadError,useMarketCommands} from "./shared";

type Edit = {action:"category_create";parentId?:string}|{action:"category_rename"|"category_disable"|"category_enable";category:MarketCategory};
const titles={category_create:"新增类目",category_rename:"修改类目名称",category_disable:"停用类目",category_enable:"恢复类目"};

export function MarketCategoryManagement({expectedUserId}:{expectedUserId:string}) {
 return <MarketBoundary admin expectedUserId={expectedUserId}>{scope=><CategoryManagement scope={scope}/>}</MarketBoundary>;
}

function CategoryManagement({scope}:{scope:MarketScope}) {
 const [open,setOpen]=useState<Record<string,boolean>>({});
 const [edit,setEdit]=useState<Edit>();
 const [name,setName]=useState("");
 const commands=useMarketCommands(scope,true);
 const first=useManagedCategoryLevel(scope);
 const roots=first.data?.pages.flatMap(page=>page.items)??[];
 function show(next:Edit) {
  setName(next.action==="category_rename"?next.category.name:"");
  setEdit(next);
 }
 async function submit() {
  if(!edit||commands.locked)return;
  const action=edit.action;
  let request:MarketCommand;
  if(action==="category_create")request={action,parentCategoryId:edit.parentId,categoryName:name.trim()};
  else if(edit.action==="category_rename")request={action:edit.action,id:edit.category.id,expectedRevision:edit.category.revision,categoryName:name.trim()};
  else request={action:edit.action,id:edit.category.id,expectedRevision:edit.category.revision};
  const receipt=await commands.execute(request);
  if(receipt?.category){
   if(edit.action==="category_create"&&edit.parentId)setOpen(value=>({...value,[edit.parentId!]:true}));
   setEdit(undefined);
  }
 }
 const editingName=edit?.action==="category_create"||edit?.action==="category_rename";
 const nameValid=!!name.trim()&&new TextEncoder().encode(name.trim()).length<=120;
 return <ConsolePage title="商品类目" description="管理硕米自营和硕米优选共用的商品类目。商品发布时归入一个二级类目。"
  breadcrumbs={[{label:"供应发布与人工评估",href:"/workbench/supply/operator"},{label:"商品类目"}]}
  actions={<><Button variant="outline" asChild><Link href="/workbench/supply/operator" prefetch={false}>返回供应发布</Link></Button>{first.isSuccess?<Button disabled={!!commands.locked} onClick={()=>show({action:"category_create"})}>新增一级类目</Button>:null}</>}>
  {!edit?commands.notice:null}
  {first.isPending?<ConsoleState kind="loading" title="正在读取类目"/>:first.error?<MarketReadError error={first.error} retry={()=>void first.refetch()}/>:!roots.length?<ConsoleState kind="empty" title="还没有商品类目">先新增一级类目，再添加二级类目。</ConsoleState>:<div className="space-y-3">
   {roots.map(parent=><Card className="space-y-3 p-4" key={parent.id}>
    <div className="flex flex-wrap items-center justify-between gap-3">
     <div><h2 className="font-semibold">{parent.name}</h2><p className="text-sm text-muted-foreground">{parent.enabled?"已启用":"已停用"}</p></div>
     <div className="flex flex-wrap gap-2">
      <Button variant="ghost" aria-expanded={!!open[parent.id]} onClick={()=>setOpen(value=>({...value,[parent.id]:!value[parent.id]}))}>{open[parent.id]?"收起二级类目":"查看二级类目"}</Button>
      <Button variant="outline" disabled={!!commands.locked} onClick={()=>show({action:"category_create",parentId:parent.id})}>新增二级类目</Button>
      <CategoryActions category={parent} show={show} disabled={!!commands.locked}/>
     </div>
    </div>
    {open[parent.id]?<ChildCategories scope={scope} parent={parent} show={show} disabled={!!commands.locked}/>:null}
   </Card>)}
   {first.hasNextPage?<Button variant="outline" disabled={first.isFetching} onClick={()=>void first.fetchNextPage()}>加载更多一级类目</Button>:null}
  </div>}
  {edit?<AccountDialog title={titles[edit.action]} onClose={()=>{if(!commands.busy)setEdit(undefined)}}>
   <form className="space-y-4" onSubmit={event=>{event.preventDefault();void submit()}}>
    {commands.notice}
    {editingName?<label className="block space-y-1"><span>类目名称</span><Input value={name} maxLength={120} required onChange={event=>setName(event.target.value)}/></label>:null}
    {edit.action==="category_disable"?<p>停用「{edit.category.name}」后，不能用于新的审核和发布。已上架商品仍可查看和按原类目筛选。{!edit.category.parentId?"下属二级类目也会暂时停止使用。":""}</p>:null}
    {edit.action==="category_enable"?<p>恢复「{edit.category.name}」。{!edit.category.parentId?"单独停用的二级类目仍保持停用。":""}</p>:null}
    <div className="flex flex-wrap justify-end gap-2">
     <Button type="button" variant="outline" disabled={commands.busy} onClick={()=>setEdit(undefined)}>取消</Button>
     <Button type="submit" disabled={!!commands.locked||editingName&&!nameValid}>{editingName?"保存":titles[edit.action]}</Button>
    </div>
   </form>
  </AccountDialog>:null}
 </ConsolePage>;
}

function CategoryActions({category,show,disabled}:{category:MarketCategory;show:(edit:Edit)=>void;disabled:boolean}) {
 return <>
  <Button variant="ghost" disabled={disabled} aria-label={"修改 "+category.name} onClick={()=>show({action:"category_rename",category})}>改名</Button>
  <Button variant="ghost" disabled={disabled} aria-label={(category.enabled?"停用 ":"恢复 ")+category.name} onClick={()=>show({action:category.enabled?"category_disable":"category_enable",category})}>{category.enabled?"停用":"恢复"}</Button>
 </>;
}

function ChildCategories({scope,parent,show,disabled}:{scope:MarketScope;parent:MarketCategory;show:(edit:Edit)=>void;disabled:boolean}) {
 const children=useManagedCategoryLevel(scope,parent.id);
 if(children.isPending)return <p role="status">正在读取二级类目</p>;
 if(children.error)return <MarketReadError error={children.error} retry={()=>void children.refetch()}/>;
 const items=children.data.pages.flatMap(page=>page.items);
 if(!items.length)return <p className="text-sm text-muted-foreground">暂无二级类目</p>;
 return <div className="space-y-2">
  <ul className="divide-y rounded-md border">
   {items.map(category=><li className="flex flex-wrap items-center justify-between gap-2 px-3 py-1" key={category.id}>
    <div>{category.name} <span className="text-sm text-muted-foreground">{!category.enabled?"已停用":parent.enabled?"已启用":"随一级类目停用"}</span></div>
    <div className="flex flex-wrap gap-1"><CategoryActions category={category} show={show} disabled={disabled}/></div>
   </li>)}
  </ul>
  {children.hasNextPage?<Button variant="outline" disabled={children.isFetching} onClick={()=>void children.fetchNextPage()}>加载更多二级类目</Button>:null}
 </div>;
}
