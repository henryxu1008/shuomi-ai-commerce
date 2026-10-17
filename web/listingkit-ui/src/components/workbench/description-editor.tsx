"use client";
import {useEffect,useId,useRef,useState,type ReactNode} from "react";
import {EditorContent,EditorContext,useEditor,useEditorState} from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import ImageExtension from "@tiptap/extension-image";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {ImagePlus,Pilcrow} from "lucide-react";
import {Toolbar,ToolbarGroup,ToolbarSeparator} from "@/vendor/tiptap/components/tiptap-ui-primitive/toolbar";
import {Button as EditorButton} from "@/vendor/tiptap/components/tiptap-ui-primitive/button";
import {HeadingDropdownMenu} from "@/vendor/tiptap/components/tiptap-ui/heading-dropdown-menu";
import {ListDropdownMenu} from "@/vendor/tiptap/components/tiptap-ui/list-dropdown-menu";
import {MarkButton} from "@/vendor/tiptap/components/tiptap-ui/mark-button";
import {BlockquoteButton} from "@/vendor/tiptap/components/tiptap-ui/blockquote-button";
import {UndoRedoButton} from "@/vendor/tiptap/components/tiptap-ui/undo-redo-button";
import {LinkPopover} from "@/vendor/tiptap/components/tiptap-ui/link-popover";
import "@/vendor/tiptap/styles/keyframe-animations.css";
import "./description-editor.css";
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
export function DescriptionEditor({label,value,format,images,disabled=false,maxBytes,onChange,renderUpload,uploadBlocked=false}:{label:string;value:string;format?:DescriptionFormat;images:string[];disabled?:boolean;maxBytes:number;uploadBlocked?:boolean;onChange:(value:string,format:DescriptionFormat)=>void;renderUpload?:(insert:(url:string)=>void)=>ReactNode}){
 const id=useId(),latest=useRef({value,format,images,disabled,onChange});useEffect(()=>{latest.current={value,format,images,disabled,onChange}},[value,format,images,disabled,onChange]);
 const last=useRef({value,format}),[selected,setSelected]=useState(""),[notice,setNotice]=useState(""),[imagePanel,setImagePanel]=useState(false);
 const editor=useEditor({
  immediatelyRender:false,
  extensions:[StarterKit.configure({heading:{levels:[2,3]},code:false,codeBlock:false,horizontalRule:false,underline:false,trailingNode:false,link:{openOnClick:false,autolink:false,linkOnPaste:false,isAllowedUri:descriptionURL,HTMLAttributes:{target:"_blank",rel:"noopener noreferrer nofollow"}}}),SourceImage.configure({allowBase64:false})],
  content:descriptionEditorHTML(value,format,images),editable:!disabled,
  editorProps:{
   attributes:{role:"textbox","aria-label":label,"aria-multiline":"true",id,class:descriptionClasses+" min-h-48 p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring"},
   transformPastedHTML:html=>sanitizeDescriptionHTML(html,latest.current.images),
   handlePaste:()=>{setNotice("粘贴内容已按支持的格式过滤；插图请使用工具栏图片上传或选择。");return false},
   handleDrop:(_view,event)=>{if(event.dataTransfer?.files.length){setNotice("请点击工具栏图片上传，再插入描述。");return true}return false},
  },
  onUpdate:({editor})=>{
   if(latest.current.disabled)return;
   const html=sanitizeDescriptionHTML(editor.getHTML());
   last.current={value:html,format:"html-v1"};latest.current.onChange(html,"html-v1");
  },
 });
 const paragraph=useEditorState({editor,selector:({editor})=>!!editor?.isActive("paragraph")});
 useEffect(()=>{if(!editor)return;editor.setEditable(!disabled,false);if(last.current.value!==value||last.current.format!==format){editor.commands.setContent(descriptionEditorHTML(value,format,images),{emitUpdate:false});last.current={value,format}}},[editor,value,format,images,disabled]);
 const run=(action:()=>void)=>{if(editor&&!disabled){action();setNotice("")}};
 const insert=(url:string)=>{if(!descriptionURL(url))return;run(()=>editor!.chain().focus().setImage({src:url,alt:"商品描述插图"}).run())};
 const bytes=new TextEncoder().encode(value).length;
 return <div className="product-description-editor space-y-3" aria-label={label+"编辑区"}>
  <label htmlFor={id} className="text-sm font-medium">{label}</label>
  <div className="overflow-hidden rounded-lg border border-border bg-background">
   <EditorContext.Provider value={{editor}}><Toolbar aria-label={label+"排版"} tabIndex={disabled?-1:0}>
    <ToolbarGroup><UndoRedoButton action="undo" aria-label="撤销" tooltip="撤销" {...(disabled?{disabled:true}:{})}/><UndoRedoButton action="redo" aria-label="重做" tooltip="重做" {...(disabled?{disabled:true}:{})}/></ToolbarGroup><ToolbarSeparator/>
    <ToolbarGroup><EditorButton type="button" tabIndex={-1} aria-label="正文" tooltip="正文" data-style="ghost" aria-pressed={!!paragraph} data-active-state={paragraph?"on":"off"} disabled={disabled||!editor} onMouseDown={e=>e.preventDefault()} onClick={()=>run(()=>editor!.chain().focus().setParagraph().run())}><Pilcrow className="tiptap-button-icon"/></EditorButton><HeadingDropdownMenu levels={[2,3]} portal aria-label="标题格式" tooltip="标题格式" {...(disabled?{disabled:true}:{})}/><ListDropdownMenu types={["bulletList","orderedList"]} portal aria-label="列表格式" tooltip="列表格式" {...(disabled?{disabled:true}:{})}/><BlockquoteButton aria-label="引用" tooltip="引用" {...(disabled?{disabled:true}:{})}/></ToolbarGroup><ToolbarSeparator/>
    <ToolbarGroup><MarkButton type="bold" aria-label="加粗" tooltip="加粗" {...(disabled?{disabled:true}:{})}/><MarkButton type="italic" aria-label="斜体" tooltip="斜体" {...(disabled?{disabled:true}:{})}/><MarkButton type="strike" aria-label="删除线" tooltip="删除线" {...(disabled?{disabled:true}:{})}/><LinkPopover aria-label="链接" tooltip="链接" autoOpenOnLinkActive={false} {...(disabled?{disabled:true}:{})}/></ToolbarGroup><ToolbarSeparator/>
    <ToolbarGroup><EditorButton type="button" aria-label="图片" tooltip="图片" aria-expanded={imagePanel||uploadBlocked} aria-controls={id+"-images"} data-style="ghost" disabled={disabled||!editor} onClick={()=>setImagePanel(v=>!v)}><ImagePlus className="tiptap-button-icon"/></EditorButton></ToolbarGroup>
   </Toolbar><EditorContent editor={editor} className="simple-editor-content"/></EditorContext.Provider>
  </div>
  <section id={id+"-images"} hidden={!imagePanel&&!uploadBlocked} aria-label="描述图片" className="space-y-4 rounded-lg border border-border bg-card p-4">
   <div className="flex flex-wrap items-end gap-2"><label className="grid min-w-0 flex-1 gap-1 text-xs">已有图片<Select aria-label={label+"插图"} value={selected} disabled={disabled} onChange={e=>setSelected(e.target.value)}><option value="">选择已有商品图片</option>{images.map((url,index)=><option key={url} value={url}>图片 {index+1}</option>)}</Select></label><Button type="button" size="sm" variant="outline" disabled={disabled||!editor||!images.includes(selected)} onClick={()=>insert(selected)}>插入图片</Button></div>
   {renderUpload?.(insert)}
  </section>
  <p className={bytes>maxBytes?"text-xs text-destructive":"text-xs text-muted-foreground"}>{bytes} / {maxBytes} 字节 · 支持基础排版、HTTPS 链接与图片。{renderUpload?"点击工具栏图片，可直接上传并插入。":"从工具栏图片选择已有图片。"}</p>
  {notice?<p role="status" className="text-xs text-muted-foreground">{notice}</p>:null}
  <details className="rounded-lg border border-border p-3"><summary className="cursor-pointer text-sm">预览描述</summary><div className="mt-3"><DescriptionView value={value} format={format} images={images}/></div></details>
 </div>;
}
