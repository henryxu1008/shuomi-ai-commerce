import {sanitizeDescriptionHTML,descriptionFormatLabel} from "@/lib/contracts/product-description";

export const descriptionClasses="space-y-3 break-words text-sm text-foreground [&_h2]:text-xl [&_h2]:font-semibold [&_h3]:text-lg [&_h3]:font-semibold [&_p]:min-h-5 [&_ul]:list-disc [&_ul]:pl-6 [&_ol]:list-decimal [&_ol]:pl-6 [&_li]:my-1 [&_blockquote]:border-l-4 [&_blockquote]:border-border [&_blockquote]:pl-4 [&_blockquote]:text-muted-foreground [&_a]:text-primary [&_a]:underline [&_img]:max-h-96 [&_img]:max-w-full [&_img]:rounded-md [&_img]:object-contain";
export function DescriptionView({value="",format,images,empty="尚未填写商品描述。",source=false}:{value?:string;format?:string;images?:string[];empty?:string;source?:boolean}){
 if(format&&format!=="html-v1")return <p role="alert">无法识别描述格式。</p>;
 return <div className="space-y-2">
  {format==="html-v1"?<div className={descriptionClasses} dangerouslySetInnerHTML={{__html:sanitizeDescriptionHTML(value,images,true)}}/>:<p className="whitespace-pre-wrap break-words text-sm">{value||empty}</p>}
  {!format&&value!==""&&value.trim()===""?<p className="text-xs text-muted-foreground">仅含空白文本 · {value.length} 个字符</p>:null}
  {source?<details className="text-xs text-muted-foreground"><summary>查看描述原文 · {descriptionFormatLabel(format)}</summary><pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3">{value||"（空）"}</pre></details>:null}
 </div>;
}

export function DescriptionProjectionNote({format}:{format?:string}){return format==="html-v1"?<p className="text-sm text-muted-foreground">平台描述使用纯文字，排版、链接地址和描述插图不会复制。原富文本保留在商品或候选草稿。</p>:null}
