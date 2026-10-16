"use client";
import {useEffect,useId,useRef,useState} from "react";
import {EditorContent,useEditor} from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import ImageExtension from "@tiptap/extension-image";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {DescriptionView,descriptionClasses} from "./description-view";
import {descriptionEditorHTML,descriptionURL,descriptionImageProxy,sanitizeDescriptionHTML,type DescriptionFormat} from "@/lib/contracts/product-description";

// A NodeView changes only presentation. The immutable document keeps original
// source URLs; image requests still pass through the existing controlled proxy.
const SourceImage=ImageExtension.extend({
 addNodeView(){return ({node})=>{
  const dom=document.createElement("img");
  const show=(attrs:Record<string,unknown>)=>{dom.src=descriptionImageProxy(String(attrs.src??""));dom.alt=String(attrs.alt??"");dom.title=String(attrs.title??"")};
  show(node.attrs);
  return {dom,update(next){if(next.type.name!=="image")return false;show(next.attrs);return true}};
 }},
});
export function DescriptionEditor({label,value,format,images,disabled=false,maxBytes,onChange}:{label:string;value:string;format?:DescriptionFormat;images:string[];disabled?:boolean;maxBytes:number;onChange:(value:string,format:DescriptionFormat)=>void}){
 const id=useId(),latest=useRef({value,format,images,disabled,onChange});useEffect(()=>{latest.current={value,format,images,disabled,onChange}},[value,format,images,disabled,onChange]);
 const last=useRef({value,format}),[url,setURL]=useState(""),[selected,setSelected]=useState(""),[notice,setNotice]=useState("");
 const editor=useEditor({
  immediatelyRender:false,
  extensions:[StarterKit.configure({heading:{levels:[2,3]},code:false,codeBlock:false,horizontalRule:false,underline:false,trailingNode:false,link:{openOnClick:false,autolink:false,linkOnPaste:false,isAllowedUri:descriptionURL,HTMLAttributes:{target:"_blank",rel:"noopener noreferrer nofollow"}}}),SourceImage.configure({allowBase64:false})],
  content:descriptionEditorHTML(value,format,images),editable:!disabled,
  editorProps:{
   attributes:{role:"textbox","aria-label":label,"aria-multiline":"true",id,class:descriptionClasses+" min-h-48 p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring"},
   transformPastedHTML:html=>sanitizeDescriptionHTML(html,latest.current.images),
   handlePaste:()=>{setNotice("粘贴内容已按支持的格式过滤；插图请从商品图片中选择。");return false},
   handleDrop:(_view,event)=>{if(event.dataTransfer?.files.length){setNotice("请先在商品图片区完成上传，再选择插入描述。");return true}return false},
  },
  onUpdate:({editor})=>{
   if(latest.current.disabled)return;
   const html=sanitizeDescriptionHTML(editor.getHTML());
   last.current={value:html,format:"html-v1"};latest.current.onChange(html,"html-v1");
  },
 });
 useEffect(()=>{if(!editor)return;editor.setEditable(!disabled,false);if(last.current.value!==value||last.current.format!==format){editor.commands.setContent(descriptionEditorHTML(value,format,images),{emitUpdate:false});last.current={value,format}}},[editor,value,format,images,disabled]);
 const run=(action:()=>void)=>{if(editor&&!disabled){action();setNotice("")}};
 const action=(name:string,call:()=>void,active=false)=><Button type="button" size="sm" variant={active?"secondary":"ghost"} aria-pressed={active} disabled={disabled||!editor} onMouseDown={e=>e.preventDefault()} onClick={()=>run(call)}>{name}</Button>;
 const bytes=new TextEncoder().encode(value).length;
 return <div className="space-y-3" aria-label={label+"编辑区"}>
  <label htmlFor={id} className="text-sm font-medium">{label}</label>
  <div className="overflow-hidden rounded-lg border border-border bg-background">
   <div role="toolbar" aria-label={label+"排版"} className="flex flex-wrap gap-1 border-b border-border bg-muted/40 p-2">
    {action("正文",()=>editor!.chain().focus().setParagraph().run(),!!editor?.isActive("paragraph"))}
    {action("小标题",()=>editor!.chain().focus().toggleHeading({level:2}).run(),!!editor?.isActive("heading",{level:2}))}
    {action("加粗",()=>editor!.chain().focus().toggleBold().run(),!!editor?.isActive("bold"))}
    {action("斜体",()=>editor!.chain().focus().toggleItalic().run(),!!editor?.isActive("italic"))}
    {action("删除线",()=>editor!.chain().focus().toggleStrike().run(),!!editor?.isActive("strike"))}
    {action("项目列表",()=>editor!.chain().focus().toggleBulletList().run(),!!editor?.isActive("bulletList"))}
    {action("编号列表",()=>editor!.chain().focus().toggleOrderedList().run(),!!editor?.isActive("orderedList"))}
    {action("引用",()=>editor!.chain().focus().toggleBlockquote().run(),!!editor?.isActive("blockquote"))}
    {action("撤销",()=>editor!.chain().focus().undo().run())}{action("重做",()=>editor!.chain().focus().redo().run())}
   </div>
   <EditorContent editor={editor}/>
  </div>
  <div className="flex flex-wrap items-end gap-2"><label className="grid min-w-48 flex-1 gap-1 text-xs">链接地址<Input aria-label={label+"链接地址"} type="url" placeholder="https://" value={url} disabled={disabled} onChange={e=>setURL(e.target.value)}/></label><Button type="button" size="sm" variant="outline" disabled={disabled||!editor||!descriptionURL(url)} onClick={()=>run(()=>{editor!.chain().focus().setLink({href:url}).run();setURL("")})}>为选中文字添加链接</Button>{action("移除链接",()=>editor!.chain().focus().unsetLink().run())}</div>
  <div className="flex flex-wrap items-end gap-2"><label className="grid min-w-48 flex-1 gap-1 text-xs">描述插图<Select aria-label={label+"插图"} value={selected} disabled={disabled} onChange={e=>setSelected(e.target.value)}><option value="">选择已有商品图片</option>{images.map((url,index)=><option key={url} value={url}>图片 {index+1}</option>)}</Select></label><Button type="button" size="sm" variant="outline" disabled={disabled||!editor||!images.includes(selected)} onClick={()=>run(()=>editor!.chain().focus().setImage({src:selected,alt:"商品描述插图"}).run())}>插入图片</Button></div>
  <p className={bytes>maxBytes?"text-xs text-destructive":"text-xs text-muted-foreground"}>{bytes} / {maxBytes} 字节 · 支持基础排版、HTTPS 链接与已有商品图片；新图片请先在商品图片区上传。</p>
  {notice?<p role="status" className="text-xs text-muted-foreground">{notice}</p>:null}
  <details className="rounded-lg border border-border p-3"><summary className="cursor-pointer text-sm">预览描述</summary><div className="mt-3"><DescriptionView value={value} format={format} images={images}/></div></details>
 </div>;
}
