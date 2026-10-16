import {useState} from "react";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {DescriptionEditor} from "./description-editor";
import {DescriptionView} from "./description-view";
import type {DescriptionFormat} from "@/lib/contracts/product-description";
Object.defineProperty(Range.prototype,"getClientRects",{configurable:true,value:()=>[]});
Object.defineProperty(Range.prototype,"getBoundingClientRect",{configurable:true,value:()=>new DOMRect()});
afterEach(cleanup);
const image="https://images.example.org/a.png";
it("keeps untouched plain bytes, edits through the real rich editor, and stores original image URLs",async()=>{
 const change=vi.fn();
 function Form(){
  const [content,setContent]=useState<{value:string;format?:DescriptionFormat}>({value:"<b>literal</b>\n "});
  return <DescriptionEditor label="商品描述" {...content} images={[image]} maxBytes={8192} onChange={(value,format)=>{change(value,format);setContent({value,format})}}/>;
 }
 render(<Form/>);
 const textbox=await screen.findByRole("textbox",{name:"商品描述"});
 expect(textbox).toHaveTextContent("<b>literal</b>");expect(textbox.querySelector("b")).toBeNull();expect(change).not.toHaveBeenCalled();
 fireEvent.change(screen.getByRole("combobox",{name:"商品描述插图"}),{target:{value:image}});
 fireEvent.click(screen.getByRole("button",{name:"插入图片"}));
 await waitFor(()=>expect(change).toHaveBeenCalledWith(expect.stringContaining('src="'+image+'"'),"html-v1"));
 expect(screen.getByRole("textbox",{name:"商品描述"}).querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image));
 expect(change.mock.lastCall?.[0]).not.toContain("/api/image-proxy");
 fireEvent.click(screen.getByText("预览描述"));expect(screen.getByText("预览描述").closest("details")?.querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image));
});
it("filters actual HTML paste and blocks editing when read-only",async()=>{
 const change=vi.fn();
 function Form({disabled=false}:{disabled?:boolean}){
  const [content,setContent]=useState<{value:string;format?:DescriptionFormat}>({value:""});
  return <DescriptionEditor label="候选描述" {...content} images={[image]} disabled={disabled} maxBytes={32768} onChange={(value,format)=>{change(value,format);setContent({value,format})}}/>;
 }
 const view=render(<Form/>);const textbox=await screen.findByRole("textbox",{name:"候选描述"});
 const html='<p onclick="evil()"><strong>safe</strong><script>evil()</script><a href="javascript:evil()">link</a></p><img src="https://foreign.example.org/a.png">';
 fireEvent.paste(textbox,{clipboardData:{getData:(type:string)=>type==="text/html"?html:"safe",types:["text/html","text/plain"],files:[]}});
 await waitFor(()=>expect(change).toHaveBeenCalled());
 const saved=change.mock.lastCall?.[0] as string;expect(saved).toContain("<strong>safe</strong>");expect(saved).not.toMatch(/evil|onclick|javascript|foreign|script/);
 change.mockClear();view.rerender(<Form disabled/>);expect(screen.getByRole("textbox",{name:"候选描述"})).toHaveAttribute("contenteditable","false");expect(screen.getByRole("button",{name:"加粗"})).toBeDisabled();expect(change).not.toHaveBeenCalled();
});
it("never interprets plain HTML-looking data and safely renders an untrusted rich read",()=>{
 const view=render(<DescriptionView value="<script>literal</script>"/>);expect(screen.getByText("<script>literal</script>")).toBeVisible();expect(view.container.querySelector("script")).toBeNull();
 view.rerender(<DescriptionView value={'<p onclick="evil()">safe<img src="'+image+'" onerror="evil()"></p><script>evil()</script>'} format="html-v1" images={[image]}/>);
 expect(view.container.querySelector("[onclick],[onerror],script")).toBeNull();expect(view.container.querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image));
});
