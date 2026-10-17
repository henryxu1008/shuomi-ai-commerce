import {useState} from "react";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import userEvent from "@testing-library/user-event";
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
 fireEvent.click(screen.getByRole("button",{name:"图片"}));
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

it("updates toolbar selection state without changing untouched description bytes",async()=>{
 const change=vi.fn();render(<DescriptionEditor label="描述" value="" images={[]} maxBytes={8192} onChange={change}/>);
 await screen.findByRole("textbox",{name:"描述"});
 fireEvent.click(screen.getByRole("button",{name:"加粗"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"加粗"})).toHaveAttribute("aria-pressed","true"));
 expect(change).not.toHaveBeenCalled();
});

it("uses the official toolbar menus and inserts only a confirmed direct upload",async()=>{
 const change=vi.fn();
 function Form(){
  const [content,setContent]=useState<{value:string;format?:DescriptionFormat}>({value:""});
  return <DescriptionEditor label="描述" {...content} images={[image]} maxBytes={8192} onChange={(value,format)=>{change(value,format);setContent({value,format})}} renderUpload={insert=><button type="button" onClick={()=>insert(image)}>确认上传夹具</button>}/>;
 }
 render(<Form/>);await screen.findByRole("textbox",{name:"描述"});
 expect(screen.getByRole("button",{name:"标题格式"})).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:/下划线|颜色|对齐/})).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"图片"}));
 expect(change).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"确认上传夹具"}));
 await waitFor(()=>expect(change).toHaveBeenCalledWith(expect.stringContaining(image),"html-v1"));
});

it("edits saved links through the official popover with the existing HTTPS policy",async()=>{
 Object.defineProperty(window,"matchMedia",{configurable:true,value:()=>({matches:false,addEventListener:()=>{},removeEventListener:()=>{}})});
 const change=vi.fn();render(<DescriptionEditor label="描述" value={'<p><a href="https://example.org/original" target="_blank" rel="noopener noreferrer nofollow">链接文字</a></p>'} format="html-v1" images={[]} maxBytes={8192} onChange={change}/>);
 const editor=await screen.findByRole("textbox",{name:"描述"});editor.focus();
 const selection=window.getSelection()!;const range=document.createRange();range.selectNodeContents(editor.querySelector("a")!);selection.removeAllRanges();selection.addRange(range);fireEvent(document,new Event("selectionchange"));
 await waitFor(()=>expect(screen.getByRole("button",{name:"链接"})).toHaveAttribute("aria-pressed","true"));
 fireEvent.click(screen.getByRole("button",{name:"链接"}));
 expect(await screen.findByRole("textbox",{name:"链接地址"})).toHaveValue("https://example.org/original");
 fireEvent.change(screen.getByRole("textbox",{name:"链接地址"}),{target:{value:"javascript:alert(1)"}});
 expect(screen.getByRole("button",{name:"应用链接"})).toBeDisabled();
 fireEvent.keyDown(screen.getByRole("textbox",{name:"链接地址"}),{key:"Enter"});expect(change).not.toHaveBeenCalled();
 fireEvent.change(screen.getByRole("textbox",{name:"链接地址"}),{target:{value:"https://example.org/updated"}});
 fireEvent.click(screen.getByRole("button",{name:"应用链接"}));await waitFor(()=>expect(change).toHaveBeenCalledWith(expect.stringContaining('href="https://example.org/updated"'),"html-v1"));
});

it("closes the official heading menu after selecting a supported level",async()=>{
 render(<DescriptionEditor label="描述" value="<h2>标题</h2>" format="html-v1" images={[]} maxBytes={8192} onChange={()=>{}}/>);
 await screen.findByRole("textbox",{name:"描述"});await userEvent.click(screen.getByRole("button",{name:"标题格式"}));
 await userEvent.click(await screen.findByRole("menuitem",{name:"三级标题"}));
 await waitFor(()=>expect(screen.queryByRole("menu")).not.toBeInTheDocument());
 expect(screen.getByRole("textbox",{name:"描述"}).querySelector("h3")).toHaveTextContent("标题");
});
