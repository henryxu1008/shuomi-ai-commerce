"use client";

import {useEffect,useRef,useState,type ClipboardEvent,type DragEvent} from "react";
import {useQuery} from "@tanstack/react-query";
import {ImagePlus} from "lucide-react";
import {Button} from "@/components/ui/button";
import {AIWorkbenchError,requestAIWorkbench,requestChatImage,type AIScope} from "@/lib/api/ai-workbench";
import {aiChatImageBody,maximumChatImages,maximumChatImageBytes,maximumChatImagesTotal} from "@/lib/contracts/ai-workbench";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";

type PendingImageQuestion = {key:string;content:string;conversation:string};
type ImageDraft = { pictures: Blob[]; pending?: PendingImageQuestion };
const unchanged = Symbol();
const toPointer=(scope:AIScope,member:string,entry:string)=>"chat-image-draft:"+JSON.stringify([scope.userId,scope.organizationId,member,entry]);
const dataId=(scope:AIScope,member:string,pointer:string)=>JSON.stringify([scope.userId,scope.organizationId,member,pointer]);
const validType=(type:string)=>type==="image/jpeg"||type==="image/png";
export const imageQuestionLengthNotice="图片提问不能超过 8192 字节。";
export const imageQuestionFits=(question:string)=>aiChatImageBody.shape.content.safeParse(question).success;

async function runImageDatabase<T>(run:(store:IDBObjectStore,transaction:IDBTransaction,finish:(result:T)=>void)=>void,write=false):Promise<T>{
  return new Promise((resolve,reject)=>{
    const request=indexedDB.open("shuomi-chat-images",1);
    request.onupgradeneeded=()=>request.result.createObjectStore("drafts");
    request.onerror=()=>reject(request.error);
    request.onblocked=()=>reject(new Error("image drafts are unavailable"));
    request.onsuccess=()=>{
      const db=request.result,tx=db.transaction("drafts",write?"readwrite":"readonly");
      let result:T;
      tx.oncomplete=()=>{db.close();resolve(result)};
      tx.onabort=()=>{db.close();reject(tx.error??new Error("image drafts are unavailable"))};
      run(tx.objectStore("drafts"),tx,value=>{result=value});
    };
  });
}

function changeDraft(id:string,change:(existing:ImageDraft|undefined)=>ImageDraft|undefined|typeof unchanged) {
  return runImageDatabase<ImageDraft|undefined>((store,tx,finish)=>{
    const read=store.get(id);
    read.onsuccess=()=>{
      const existing=read.result as ImageDraft|undefined;
      try{
        const draft=change(existing);
        if(draft===unchanged){finish(existing);return}
        if(draft)store.put(draft,id);else store.delete(id);
        finish(draft);
      }catch{tx.abort()}
    };
  },true);
}

function readDraft(id:string) {
  return runImageDatabase<ImageDraft|undefined>((store,_tx,finish)=>{
    const request=store.get(id);
    request.onsuccess=()=>finish(request.result as ImageDraft|undefined);
  });
}

function base64(blob:Blob):Promise<string>{
  return new Promise((resolve,reject)=>{
    const reader=new FileReader();
    reader.onerror=()=>reject(reader.error);
    reader.onload=()=>resolve(String(reader.result).split(",",2)[1]??"");
    reader.readAsDataURL(blob);
  });
}

export function useChatImageDraft(scope:AIScope,member:string,entry:string,enabled:boolean,authorizationKey:string) {
  const pointerKey=toPointer(scope,member,entry);
  const [stored,setStored]=useState<{owner:string;pointer?:string;draft?:ImageDraft}>();
  const [unwritten,setUnwritten]=useState<{owner:string;pictures:Blob[]}>();
  const [notice,setNotice]=useState("");
  const [failedRead,setFailedRead]=useState<string>();
  const [readAttempt,setReadAttempt]=useState(0);
  const [busy,setBusy]=useState(false);
  const mutation=useRef(false);
  const status=useQuery({
    queryKey:["chat-image-readiness",scope.userId,scope.organizationId,member,authorizationKey],
    queryFn:({signal})=>requestAIWorkbench({route:"image-question-readiness",method:"GET",path:"chat/conversations/image-questions",scope,signal}),
    enabled:enabled&&!!member,retry:false,
  });
  useEffect(()=>{
    if (!enabled||!member)return;
    let active=true;
    void (async()=>{
      try{
        const pointer=sessionStorage.getItem(pointerKey)??undefined;
        if(pointer&&!isAcquisitionUUID(pointer))throw new Error("image record invalid");
        const draft=pointer?await readDraft(dataId(scope,member,pointer)):undefined;
        if(active){
          setStored({owner:pointerKey,pointer,draft});
          setFailedRead(undefined);
        }
      }catch{
        if(active)setFailedRead(pointerKey);
      }
    })();
    return ()=>{active=false};
  },[enabled,member,pointerKey,readAttempt,scope]);
  const current=enabled&&stored?.owner===pointerKey?stored:undefined;
  const recoveryBlocked=enabled&&!!member&&!current;
  const recoveryFailed=recoveryBlocked&&failedRead===pointerKey;
  const buffered=unwritten?.owner===pointerKey?unwritten:undefined;
  const pictures=buffered?.pictures??current?.draft?.pictures??[];

  async function edit(pictures:Blob[]) {
    if(!current||mutation.current||current.draft?.pending)return;
    mutation.current=true;setBusy(true);setUnwritten({owner:pointerKey,pictures});
    try{
      const pointer=current.pointer??crypto.randomUUID();
      // Persist the tab pointer first. A failed IndexedDB write cannot send.
      if(!current.pointer)sessionStorage.setItem(pointerKey,pointer);
      if(sessionStorage.getItem(pointerKey)!==pointer)throw new Error("image draft changed");
      const draft=await changeDraft(dataId(scope,member,pointer),existing=>existing?.pending?unchanged:pictures.length?{pictures}:undefined);
      setStored({owner:pointerKey,pointer,draft});setUnwritten(undefined);setNotice("");
    }catch{setNotice("图片暂时无法保存，请检查浏览器存储后重试。")}
    finally{mutation.current=false;setBusy(false)}
  }

  function add(files: File[]) {
    if(!files.length)return;
    if(files.some(file=>!validType(file.type)||!file.size||file.size>maximumChatImageBytes)){
      setNotice("请选择 JPG 或 PNG 图片，每张最多 768 KiB。");return;
    }
    const group=[...pictures,...files.map(file=>file.slice(0,file.size,file.type))];
    if(group.length>maximumChatImages||group.reduce((bytes,file)=>bytes+file.size,0)>maximumChatImagesTotal){
      setNotice("每条最多 4 张图片，合计最多 2 MiB。请移除一张或更换图片。");return;
    }
    void edit(group);
  }

  function paste(event:ClipboardEvent) {
    const files=Array.from(event.clipboardData.files).filter(file=>file.type.startsWith("image/"));
    if(files.length){event.preventDefault();add(files)}
  }

  function drop(event:DragEvent) {
    const files=Array.from(event.dataTransfer.files);
    if(files.length){event.preventDefault();add(files)}
  }

  function handoff(conversation:string,pointer:string){
    const destination=toPointer(scope,member,"conversation:"+conversation);
    const existing=sessionStorage.getItem(destination);
    if(existing&&existing!==pointer)throw new Error("another image draft exists");
    sessionStorage.setItem(destination,pointer);
    if(sessionStorage.getItem(pointerKey)===pointer)sessionStorage.removeItem(pointerKey);
  }

  async function send(conversation:string,content:string,signal:AbortSignal,saveEditableQuestion:(question:string)=>boolean){
    if(!current?.pointer||mutation.current||buffered||!pictures.length)return;
    if(!current.draft?.pending&&(!content.trim()||!imageQuestionFits(content))){
      if(content.trim())setNotice(imageQuestionLengthNotice);
      return;
    }
    mutation.current=true;setBusy(true);setNotice("");
    const pointer=current.pointer,id=dataId(scope,member,pointer);
    let original:PendingImageQuestion|undefined;
    async function finishRequest(request:PendingImageQuestion,keepEditable:boolean) {
      if(keepEditable&&!saveEditableQuestion(request.content)){
        setNotice("原问题和图片已保留，请检查浏览器存储后核实原请求。");
        return false;
      }
      const next=await changeDraft(id,existing=>{
        if(!existing||existing.pending?.key!==request.key)return unchanged;
        return keepEditable?{pictures:existing.pictures}:undefined;
      });
      setStored({owner:pointerKey,pointer,draft:next});
      return true;
    }
    try{
      if(sessionStorage.getItem(pointerKey)!==pointer)throw new Error("image draft changed");
      const draft=await changeDraft(id,existing=>{
        if(!existing?.pictures.length)throw new Error("image draft is missing");
        if(existing.pending){if(existing.pending.conversation!==conversation)throw new Error("image request belongs to another conversation");return unchanged}
        if(!content.trim()||!imageQuestionFits(content))throw new Error("image question cannot be accepted");
        return {...existing,pending:{key:crypto.randomUUID(),content,conversation}};
      });
      if(!draft?.pending||signal.aborted)return;
      setStored({owner:pointerKey,pointer,draft});
      original=draft.pending;
      // An oversized draft from an earlier browser could not have passed the
      // unchanged proxy contract. Keep it editable without attempting delivery.
      if(!imageQuestionFits(original.content)){
        if(await finishRequest(original,true))setNotice("原问题超出发送限制，已恢复为可编辑草稿。");
        return;
      }
      const body={content:draft.pending.content,images:await Promise.all(draft.pictures.map(base64))};
      const result=await requestAIWorkbench({route:"image-question-submit",method:"POST",path:`chat/conversations/${conversation}/image-questions`,scope,body,key:draft.pending.key,signal});
      if(signal.aborted)return;
      if(result.state==="COMPLETE"||result.state==="FAILED_BEFORE_DISPATCH"||result.state==="PLANNER_INVALID_OUTPUT"){
        if(!await finishRequest(original,result.state!=="COMPLETE"))return;
        if(result.state==="FAILED_BEFORE_DISPATCH")setNotice("图片提问未发送到模型，原问题和图片已保留。");
        if(result.state==="PLANNER_INVALID_OUTPUT")setNotice("本次回复未能显示，原问题和图片已保留。请在用量明细查看用量。");
      }else{
        setNotice("图片提问的结果待确认。请核实原请求。");
      }
      return result;
    }catch(error){
      if(signal.aborted)return;
      if(original&&error instanceof AIWorkbenchError&&error.messageAccepted===false){
        try{
          if(await finishRequest(original,true))setNotice("图片未发送。请检查图片格式和大小后重试。");
          return;
        }catch{
          setNotice("原图片提问已保留，请检查浏览器存储后重试。");
          return;
        }
      }
      setNotice(error instanceof AIWorkbenchError&&error.code==="STORAGE_LIMIT_EXCEEDED"?
        "企业存储空间不足，原问题和图片已保留。":"无法确认这次图片提问，请检查当前模型和浏览器存储后核实原请求。");
    }finally{mutation.current=false;if(!signal.aborted)setBusy(false)}
  }
  return {pictures,ready:!!current,recoveryBlocked,recoveryFailed,retryRead:()=>{setFailedRead(undefined);setReadAttempt(attempt=>attempt+1)},available:enabled&&status.data?.available===true,notice,busy,unsaved:!!buffered,pending:current?.draft?.pending,pointer:current?.pointer,
    add,paste,drop,handoff,send,retrySave:()=>{if(buffered)void edit(buffered.pictures)},remove:(index:number)=>void edit(pictures.filter((_,n)=>index!==n))};
}

function ImagePreview({blob,index,onRemove}:{blob:Blob;index:number;onRemove?:()=>void}){
  const [source,setSource]=useState("");
  useEffect(()=>{let active=true;
    const url=URL.createObjectURL(blob);
    queueMicrotask(()=>{if(active)setSource(url)});
    return()=>{active=false;URL.revokeObjectURL(url)}},[blob]);
  return <figure className="grid gap-1">
    {/* The URL refers to a private Blob retrieved from an authenticated endpoint. */}
    {/* eslint-disable-next-line @next/next/no-img-element */}
    {source&&<img src={source} alt={`图 ${index+1}`} className="h-20 w-20 rounded-md object-contain"/>}
    {onRemove&&<Button type="button" size="sm" variant="ghost" onClick={onRemove}>移除图 {index+1}</Button>}
  </figure>;
}

export function ChatImagePicker({draft,disabled}:{draft:ReturnType<typeof useChatImageDraft>;disabled?:boolean}){
  const input=useRef<HTMLInputElement>(null);
  return <>
    {draft.pictures.length>0&&<div className="flex flex-wrap gap-2" aria-label="已选图片">{draft.pictures.map((blob,n)=><ImagePreview key={n} blob={blob} index={n} onRemove={disabled||draft.busy||draft.pending?undefined:()=>draft.remove(n)}/>)}</div>}
    {(draft.available||draft.pictures.length>0)&&<>
      <Button type="button" size="sm" variant="outline" disabled={disabled||draft.busy||!!draft.pending||!draft.ready} onClick={()=>input.current?.click()}><ImagePlus aria-hidden size={16}/> 选择图片</Button>
      <input ref={input} type="file" hidden multiple accept="image/jpeg,image/png" onChange={event=>{draft.add(Array.from(event.target.files??[]));event.target.value=""}}/>
    </>}
    {draft.recoveryFailed&&<div>
      <p role="status">无法读取保留的图片，请检查浏览器存储。</p>
      <Button type="button" size="sm" variant="outline" onClick={draft.retryRead}>重新读取图片</Button>
    </div>}
    {draft.recoveryBlocked&&!draft.recoveryFailed&&<p role="status">正在读取保留的图片…</p>}
    {draft.notice&&<p role="status">{draft.notice}</p>}
    {draft.unsaved&&!draft.busy&&<Button type="button" size="sm" variant="outline" disabled={disabled} onClick={draft.retrySave}>重试保存图片</Button>}
  </>;
}

export function ChatHistoryImage({scope,conversation,message,index}:{scope:AIScope;conversation:string;message:string;index:number}){
  const [picture,setPicture]=useState<Blob>();
  const [failed,setFailed]=useState(false);
  const {userId,organizationId}=scope;
  useEffect(()=>{
    const controller=new AbortController();
    void requestChatImage({userId,organizationId},conversation,message,index,controller.signal)
      .then(result=>{if(!controller.signal.aborted)setPicture(result)})
      .catch(()=>{if(!controller.signal.aborted)setFailed(true)});
    return ()=>controller.abort();
  },[userId,organizationId,conversation,message,index]);
  return picture?<ImagePreview blob={picture} index={index}/>:<p role="status">{failed?`图 ${index+1} 暂时无法读取`:`正在读取图 ${index+1}…`}</p>;
}
