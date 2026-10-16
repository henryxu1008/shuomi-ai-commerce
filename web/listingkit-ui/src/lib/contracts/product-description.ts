import sanitize from "sanitize-html";
import {parseDocument} from "htmlparser2";
import {z} from "zod";

type ChildNode=ReturnType<typeof parseDocument>["children"][number];

export const descriptionFormatSchema=z.literal("html-v1").optional();
export type DescriptionFormat=z.infer<typeof descriptionFormatSchema>;
export const descriptionFormatLabel=(format?:string)=>format==="html-v1"?"富文本":"纯文本";
export const descriptionImageProxy=(url:string)=>`/api/image-proxy?url=${encodeURIComponent(url)}`;
export function descriptionURL(value:string){
 try{const u=new URL(value);return value.startsWith("https://")&&u.protocol==="https:"&&!!u.hostname&&!u.username&&!u.password&&!u.hash&&!/[\\\\\s]/.test(value)&&new TextEncoder().encode(value).length<=2048}catch{return false}
}
const tags=["p","br","h2","h3","strong","em","s","ul","ol","li","blockquote","a","img"];
export function sanitizeDescriptionHTML(value:string,images?:string[],proxy=false){
 return sanitize(value,{
  allowedTags:tags,allowedAttributes:{a:["href","target","rel"],img:["src","alt","title"]},
  allowedSchemes:["https"],allowProtocolRelative:false,disallowedTagsMode:"discard",nestingLimit:32,
  transformTags:{
   a:(_name,attrs):sanitize.Tag=>({tagName:"a",attribs:descriptionURL(attrs.href??"")?{href:attrs.href!,target:"_blank",rel:"noopener noreferrer nofollow"}:{}}),
   img:(_name,attrs)=>({tagName:"img",attribs:{...attrs,...(proxy?{src:descriptionImageProxy(attrs.src??"")}: {})}}),
  },
  exclusiveFilter:frame=>frame.tag==="img"&&(()=>{
   const src=proxy?new URL(frame.attribs.src,"https://description.invalid").searchParams.get("url")??"":frame.attribs.src??"";
   return !descriptionURL(src)||!!images&&!images.includes(src);
  })(),
 });
}
export function descriptionReferences(value:string,format?:string){
 if(format!=="html-v1")return [];
 const result:string[]=[];
 const visit=(nodes:ChildNode[])=>{for(const node of nodes)if(node.type==="tag"){if(node.name==="img"&&node.attribs.src)result.push(node.attribs.src);visit(node.children)}};
 visit(parseDocument(value).children);return [...new Set(result)];
}
export function descriptionPlainText(value:string,format?:string){
 if(!format)return value;
 if(format!=="html-v1")return "";
 let result="";
 const newline=()=>{if(result&&!result.endsWith("\n"))result+="\n"};
 const walk=(nodes:ChildNode[])=>{for(const node of nodes){
  if(node.type==="text"){result+=node.data;continue}
  if(node.type!=="tag"||node.name==="img")continue;
  const block=["p","h2","h3","li","blockquote","ul","ol"].includes(node.name);
  if(block)newline();
  if(node.name==="br")result+="\n";
  walk(node.children);if(block)newline();
 }};
 walk(parseDocument(sanitizeDescriptionHTML(value)).children);return result.replace(/^\p{White_Space}+|\p{White_Space}+$/gu,"");
}
export function descriptionEditorHTML(value:string,format?:string,images?:string[]){
 if(format==="html-v1")return sanitizeDescriptionHTML(value,images);
 return "<p>"+value.replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;").replaceAll("\n","<br>")+"</p>";
}
export function restoreDescription<T extends {title:string;description:string;descriptionFormat?:DescriptionFormat}>(current:T,old:{title:string;description:string;descriptionFormat?:DescriptionFormat}):T{
 const restored={...current,title:old.title,description:old.description};
 delete restored.descriptionFormat;
 if(old.descriptionFormat)restored.descriptionFormat=old.descriptionFormat;
 return restored;
}
