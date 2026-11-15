import {afterEach,beforeAll,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {SkillManagerTrigger,SkillManager} from "./skill-manager";
import {skillRequest,SkillError} from "@/lib/api/skill";
import {parseSkillLaunch} from "@/lib/contracts/skill-launch";
vi.mock("@/lib/api/skill",async original=>({...await original<typeof import("@/lib/api/skill")>(),skillRequest:vi.fn(),skillError:()=>"技能服务不可用"}));
beforeAll(()=>{HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")}});
afterEach(()=>{cleanup();vi.clearAllMocks()});
it("lets a maintainer create, check and save an official draft without exposing creation in the library",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path==="official/validate"?{valid:true,name:"new-skill",description:"description",digest:"a".repeat(64),fileCount:1}:path==="official"?{id:"11111111-1111-4111-8111-111111111111",version:1,revision:1,operation:"create",at:"2026-11-14T00:00:00Z"}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));
 fireEvent.change(screen.getByLabelText("技能名称（小写标识）"),{target:{value:"new-skill"}});
 fireEvent.change(screen.getByLabelText("技能简介"),{target:{value:"description"}});
 fireEvent.change(screen.getByLabelText("方法内容"),{target:{value:"Write a useful method."}});
 fireEvent.click(screen.getByRole("button",{name:"检查格式"}));
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path])=>path==="official/validate")).toBe(true));
 fireEvent.click(await screen.findByRole("button",{name:"保存草稿"}));
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path,,init])=>path==="official"&&init?.method==="POST")).toBe(true));
 const request=vi.mocked(skillRequest).mock.calls.find(([,path,,init])=>path==="official"&&init?.method==="POST");
 expect(JSON.parse(String(request?.[3]?.body)).files[0].text).toContain('listingkit-method-v1: "product-title"');
});
it("uses the image skill contract and requires an image brief for checking",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path==="official/validate"?{valid:true,name:"image-skill",description:"description",digest:"a".repeat(64),fileCount:2}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));
 fireEvent.change(screen.getByLabelText("技能名称"),{target:{value:"图片技能"}});fireEvent.change(screen.getByLabelText("技能名称（小写标识）"),{target:{value:"image-skill"}});
 fireEvent.change(screen.getByLabelText("技能用途"),{target:{value:"product-image"}});
 fireEvent.click(screen.getByRole("button",{name:"检查格式"}));expect(await screen.findByRole("alert")).toHaveTextContent("请填写图片生成说明");
 expect(vi.mocked(skillRequest).mock.calls.some(([,path])=>path==="official/validate")).toBe(false);
 fireEvent.change(screen.getByLabelText("图片生成说明"),{target:{value:"Keep the original product appearance."}});fireEvent.click(screen.getByRole("button",{name:"检查格式"}));
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path])=>path==="official/validate")).toBe(true));
 const request=vi.mocked(skillRequest).mock.calls.find(([,path])=>path==="official/validate");const files=JSON.parse(String(request?.[3]?.body)).files;
 expect(files.find((f:{path:string})=>f.path==="SKILL.md").text).toContain('listingkit-method-v1: "product-image"');
 expect(files.find((f:{path:string})=>f.path==="SKILL.md").text).toContain('listingkit-display-name: "图片技能"');
 expect(files.find((f:{path:string})=>f.path==="references/visual-brief.md").text).toBe("Keep the original product appearance.");
 expect(screen.queryByLabelText("文件内容")).not.toBeInTheDocument();
});
it.each(["product-image","product-title"])("saves a draft without adding an empty image brief after selecting %s",async finalTarget=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path,_schema,init)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path==="official"&&init?.method==="POST"?{skillId:"11111111-1111-4111-8111-111111111111",version:1,revision:1,operation:"create",at:"2026-11-14T00:00:00Z"}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));fireEvent.change(screen.getByLabelText("技能名称（小写标识）"),{target:{value:"test-skill"}});
 fireEvent.change(screen.getByLabelText("技能简介"),{target:{value:"Description"}});fireEvent.change(screen.getByLabelText("方法内容"),{target:{value:"A valid method."}});
 fireEvent.change(screen.getByLabelText("技能用途"),{target:{value:"product-image"}});if(finalTarget==="product-title")fireEvent.change(screen.getByLabelText("技能用途"),{target:{value:finalTarget}});
 fireEvent.click(screen.getByRole("button",{name:"保存草稿"}));
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path,,init])=>path==="official"&&init?.method==="POST")).toBe(true));
 const request=vi.mocked(skillRequest).mock.calls.find(([,path,,init])=>path==="official"&&init?.method==="POST");
 expect(JSON.parse(String(request?.[3]?.body)).files.every((f:{text:string})=>f.text.trim().length>0)).toBe(true);
});
it("does not publish an image draft without its image brief",async()=>{
 const id="11111111-1111-4111-8111-111111111111";const version={skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:1,revision:1},version:1,state:"DRAFT",package:{name:"image-skill",description:"图片方法",target:"product-image",digest:"a".repeat(64),files:[{path:"SKILL.md",text:"content"}]}};
 const projection={version,canUse:false,canCopy:false,canExport:false,canEdit:true,canPublish:true};
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path.endsWith("/versions/1")?projection:path.endsWith("/versions")?{items:[version]}:{items:[projection],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:/image-skill/}));expect(await screen.findByText("请先补充图片生成说明，再发布或启用。")).toBeVisible();
 expect(screen.getByRole("button",{name:"发布该版本"})).toBeDisabled();
});
it.each(["image","title"])("preserves the imported image brief in %s mode",async mode=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path,_schema,init)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path==="official"&&init?.method==="POST"?{skillId:"11111111-1111-4111-8111-111111111111",version:1,revision:1,operation:"create",at:"2026-11-14T00:00:00Z"}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));
 fireEvent.change(screen.getByLabelText("技能名称（小写标识）"),{target:{value:"test-skill"}});fireEvent.change(screen.getByLabelText("技能简介"),{target:{value:"Description"}});fireEvent.change(screen.getByLabelText("方法内容"),{target:{value:"A valid method."}});
 if(mode==="image"){fireEvent.change(screen.getByLabelText("技能用途"),{target:{value:"product-image"}});fireEvent.change(screen.getByLabelText("图片生成说明"),{target:{value:"old brief"}})}
 const file=new File(["ignored"],"visual-brief.md",{type:"text/markdown"});Object.defineProperty(file,"text",{value:async()=>"imported brief"});fireEvent.change(screen.getByLabelText("添加参考文本或模板"),{target:{files:[file]}});
 if(mode==="image")expect(await screen.findByDisplayValue("imported brief")).toBeVisible();else{const field=await screen.findByLabelText("文件内容");fireEvent.change(field,{target:{value:"edited imported brief"}});fireEvent.change(screen.getByLabelText("技能用途"),{target:{value:"product-image"}});expect(screen.getByLabelText("图片生成说明")).toHaveValue("edited imported brief")}
 fireEvent.click(screen.getByRole("button",{name:"保存草稿"}));await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path,,init])=>path==="official"&&init?.method==="POST")).toBe(true));
 const request=vi.mocked(skillRequest).mock.calls.find(([,path,,init])=>path==="official"&&init?.method==="POST");const files=JSON.parse(String(request?.[3]?.body)).files;
 expect(files.filter((f:{path:string})=>f.path==="references/visual-brief.md")).toEqual([{path:"references/visual-brief.md",text:mode==="image"?"imported brief":"edited imported brief"}]);
});
it("imports a skill into the draft editor without saving it",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));
 const file=new File(["ignored"],"SKILL.md",{type:"text/markdown"});Object.defineProperty(file,"text",{value:async()=>"---\nname: imported-skill\ndescription: Imported description\n---\nUseful method."});
 fireEvent.change(screen.getByLabelText("导入 SKILL.md / 文本包"),{target:{files:[file]}});
 expect(await screen.findByDisplayValue(/imported-skill/)).toBeVisible();
 expect(vi.mocked(skillRequest).mock.calls.some(([,path,,init])=>path==="official"&&init?.method==="POST")).toBe(false);
});
it("publishes and then enables an official draft through existing version actions",async()=>{
 const id="11111111-1111-4111-8111-111111111111";let state="DRAFT",revision=1;
 const version=()=>({skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:1,revision},version:1,state,package:{name:"new-skill",displayName:"新技能",description:"技能说明",target:"product-title",digest:"a".repeat(64)}});
 const projection=()=>({version:version(),canUse:state==="ENABLED",canCopy:false,canExport:false,canEdit:true,canPublish:true});
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>{if(path.endsWith("/info"))return {available:true,canCreate:true,canInstallOfficial:true,reason:""};if(path.endsWith("/publish")){state="READY";revision++;return {}}if(path.endsWith("/enable")){state="ENABLED";revision++;return {}}if(path.endsWith("/versions/1"))return projection();if(path.endsWith("/versions"))return {items:[version()]};return {items:[projection()],nextCursor:""}});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:/新技能/}));fireEvent.click(await screen.findByRole("button",{name:"发布该版本"}));
 await waitFor(()=>expect(state).toBe("READY"));const card=await screen.findByRole("button",{name:/新技能/});await waitFor(()=>expect(card).toBeEnabled());fireEvent.click(card);
 fireEvent.click(await screen.findByRole("button",{name:"启用该版本"}));await waitFor(()=>expect(state).toBe("ENABLED"));
 const actions=vi.mocked(skillRequest).mock.calls.filter(([,path])=>path.endsWith("/publish")||path.endsWith("/enable"));
 expect(actions.map(([,, ,init])=>new Headers(init?.headers).get("If-Match"))).toEqual(['"1"','"2"']);
});
it("retains official draft inputs when saving fails",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path,_schema,init)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path==="official"&&init?.method==="POST"?Promise.reject(new SkillError("SKILL_FORBIDDEN")):{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建或导入官方技能"}));
 fireEvent.change(screen.getByLabelText("技能名称（小写标识）"),{target:{value:"kept-skill"}});
 fireEvent.click(screen.getByRole("button",{name:"保存草稿"}));
 expect(await screen.findByRole("alert")).toBeVisible();expect(screen.getByLabelText("技能名称（小写标识）")).toHaveValue("kept-skill");
 fireEvent.click(screen.getByRole("button",{name:"取消编辑"}));expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});
it("requires the existing maintainer capability for the settings installation entry",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:false,canInstallOfficial:false,reason:""}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"reader"}} home="personal" initialSpace="official" presentation="page" maintenance/>);
 expect(await screen.findByText("当前身份无权管理官方模板。")).toBeVisible();
 expect(screen.queryByRole("button",{name:"安装官方模板"})).not.toBeInTheDocument();
 expect(vi.mocked(skillRequest).mock.calls.every(([,path])=>path!=="official/install")).toBe(true);
});
it.each(["official","personal","organization"] as const)("keeps %s cards concise while showing meaningful limits",async space=>{
 const version={skill:{id:"11111111-1111-4111-8111-111111111111",owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:7,state:"ENABLED",package:{name:"title-method",displayName:"标题整理",description:"整理标题",target:"product-title",digest:"a".repeat(64)}};
 const limited={...version,skill:{...version.skill,id:"22222222-2222-4222-8222-222222222222"},state:"DISABLED",package:{...version.package,name:"limited-method",displayName:"停用方法"}};
 const projection=(value:typeof version,canCopy:boolean)=>({version:value,canUse:true,canCopy,canExport:false,canEdit:false,canPublish:false});
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:false,canInstallOfficial:false,reason:""}:path.endsWith("/admission")?{policy:{allowPersonal:false,revision:"0"},canConfigure:false}:{items:[projection(version,true),projection(limited,false)],nextCursor:""});
 render(<SkillManager scope={{userId:"actor",organizationId:"org"}} home={space==="organization"?"organization":"personal"} initialSpace={space} presentation="page"/>);
 const card=await screen.findByRole("button",{name:/标题整理/});
 expect(within(card).getByText("商品标题")).toBeVisible();
 expect(card).not.toHaveTextContent(/v7|已启用|可复制/);
 const restricted=screen.getByRole("button",{name:/停用方法/});
 expect(restricted).toHaveTextContent("已停用");expect(restricted).toHaveTextContent("不可复制");
 fireEvent.click(screen.getByRole("button",{name:"商品标题"}));
 expect(within(screen.getByRole("button",{name:/标题整理/})).queryByText("商品标题")).toBeNull();
 expect(screen.getByRole("button",{name:/停用方法/})).toHaveTextContent("不可复制");
});
it("does not expose official installation in the skill library dialog",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:{items:[],nextCursor:""});
 render(<SkillManagerTrigger scope={{userId:"actor"}} space="personal"/>);
 fireEvent.click(screen.getByRole("button",{name:"我的技能"}));
 fireEvent.click(await screen.findByRole("button",{name:"官方模板"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"刷新列表"})).toBeEnabled());
 expect(screen.queryByRole("button",{name:"安装官方模板"})).not.toBeInTheDocument();
 expect(vi.mocked(skillRequest).mock.calls.every(([,path])=>path!=="official/install")).toBe(true);
});
it.each(["official","personal","organization"] as const)("filters and searches all loaded pages in %s",async space=>{
 const first={skill:{id:"11111111-1111-4111-8111-111111111111",owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:1,state:"ENABLED",package:{name:"title-method",displayName:"标题整理",description:"整理标题",target:"product-title",digest:"a".repeat(64)}};
 const second={...first,skill:{...first.skill,id:"22222222-2222-4222-8222-222222222222"},package:{...first.package,name:"image-method",displayName:"图片编排",target:"product-image"}};
 const projection=(version:typeof first)=>({version,canUse:true,canCopy:false,canExport:false,canEdit:false,canPublish:false});
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:false,canInstallOfficial:false,reason:""}:path.endsWith("/admission")?{policy:{allowPersonal:false,revision:"0"},canConfigure:false}:path.includes("cursor=")?{items:[projection(second)],nextCursor:""}:{items:[projection(first)],nextCursor:"next"});
 render(<SkillManager scope={{userId:"actor",organizationId:"org"}} home={space==="organization"?"organization":"personal"} initialSpace={space} presentation="page"/>);
 await screen.findByRole("button",{name:/图片编排/});fireEvent.click(screen.getByRole("button",{name:"商品图片"}));
 expect(screen.queryByRole("button",{name:/标题整理/})).not.toBeInTheDocument();expect(screen.getByRole("button",{name:/图片编排/})).toBeVisible();
 fireEvent.change(screen.getByRole("searchbox",{name:"搜索技能名称"}),{target:{value:"不存在"}});expect(screen.getByText("没有找到符合条件的技能")).toBeVisible();
 fireEvent.change(screen.getByRole("searchbox",{name:"搜索技能名称"}),{target:{value:"图片"}});expect(screen.getByRole("button",{name:/图片编排/})).toBeVisible();
});
it("makes the remaining pages explicit and adds them to search when requested",async()=>{
 const item=(n:number)=>({version:{skill:{id:`skill-${n}`,owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:1,state:"ENABLED",package:{name:`skill-${n}`,displayName:`技能 ${n}`,description:"方法",target:"product-title",digest:"a".repeat(64)}},canUse:true,canCopy:false,canExport:false,canEdit:false,canPublish:false});
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>{if(path.endsWith("/info"))return {available:true,canCreate:false,canInstallOfficial:false,reason:""};const q=new URLSearchParams(path.split("?")[1]);const n=Number(q.get("cursor")||"1");return {items:[item(n)],nextCursor:n<6?String(n+1):""}});
 render(<SkillManager scope={{userId:"actor"}} home="personal" initialSpace="official" presentation="page"/>);
 const more=await screen.findByRole("button",{name:"加载更多技能"});expect(screen.getByText("还有更多技能，加载后可一起筛选和搜索。")).toBeVisible();expect(screen.queryByRole("button",{name:"商品图片"})).not.toBeInTheDocument();
 fireEvent.change(screen.getByRole("searchbox",{name:"搜索技能名称"}),{target:{value:"技能 6"}});expect(screen.getByText("没有找到符合条件的技能")).toBeVisible();
 fireEvent.click(more);expect(await screen.findByRole("button",{name:/技能 6/})).toBeVisible();expect(screen.queryByRole("button",{name:"加载更多技能"})).not.toBeInTheDocument();
});
it("stops if a later batch returns an already visited cursor",async()=>{
 const item=(n:number)=>({version:{skill:{id:`skill-${n}`,owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:1,state:"ENABLED",package:{name:`skill-${n}`,description:"方法",target:"product-title",digest:"a".repeat(64)}},canUse:true,canCopy:false,canExport:false,canEdit:false,canPublish:false});
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>{if(path.endsWith("/info"))return {available:true,canCreate:false,canInstallOfficial:false,reason:""};const q=new URLSearchParams(path.split("?")[1]);const n=Number(q.get("cursor")||"1");return {items:[item(n)],nextCursor:n===6?"2":String(n+1)}});
 render(<SkillManager scope={{userId:"actor"}} home="personal" initialSpace="official" presentation="page"/>);
 fireEvent.click(await screen.findByRole("button",{name:"加载更多技能"}));expect(await screen.findByRole("alert")).toBeVisible();expect(screen.queryByRole("button",{name:"加载更多技能"})).not.toBeInTheDocument();
});
it("opens an individual skill in a dialog and closes without copying it",async()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const version={skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:1,state:"ENABLED",package:{name:"product-title-organize",displayName:"商品标题整理",description:"真实方法",target:"product-title",digest:"a".repeat(64)}};
 const projection={version,canUse:true,canCopy:true,canExport:false,canEdit:false,canPublish:false};
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:false,reason:""}:path.endsWith("/versions/1")?projection:path.endsWith("/versions")?{items:[version]}:{items:[projection],nextCursor:""});
 render(<SkillManager scope={{userId:"actor",organizationId:"org"}} home="personal" initialSpace="official" presentation="page"/>);
 const trigger=await screen.findByRole("button",{name:/商品标题整理/});await waitFor(()=>expect(trigger).toBeEnabled());trigger.focus();fireEvent.click(trigger);
 const dialog=await screen.findByRole("dialog",{name:"商品标题整理"});expect(within(dialog).getByRole("button",{name:"复制到本人"})).toBeEnabled();
 expect(vi.mocked(skillRequest).mock.calls.some(([,path])=>path.endsWith("/copy"))).toBe(false);
 fireEvent(dialog,new Event("cancel",{cancelable:true}));expect(screen.queryByRole("dialog",{name:"商品标题整理"})).not.toBeInTheDocument();expect(trigger).toHaveFocus();
});
it("keeps official maintenance actions out of the skill library",async()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const version={skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:1},version:1,state:"ENABLED",package:{name:"product-title-organize",displayName:"商品标题整理",description:"真实方法",target:"product-title",digest:"a".repeat(64)}};
 const item={version,canUse:true,canCopy:true,canExport:true,canEdit:true,canPublish:true};
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path.endsWith("/versions/1")?item:path.endsWith("/versions")?{items:[version]}:{items:[item],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer",organizationId:"org"}} home="personal" initialSpace="official" presentation="page"/>);
 const trigger=await screen.findByRole("button",{name:/商品标题整理/});await waitFor(()=>expect(trigger).toBeEnabled());fireEvent.click(trigger);
 const dialog=await screen.findByRole("dialog",{name:"商品标题整理"});
 expect(within(dialog).getByRole("link",{name:"处理已有商品"})).toBeVisible();
 expect(within(dialog).getByRole("button",{name:"复制到本人"})).toBeVisible();
 for(const name of ["保存为新版本","导出准确版本","停用该版本","撤销该版本"])expect(within(dialog).queryByRole("button",{name})).toBeNull();
 expect(screen.queryByRole("button",{name:"安装官方模板"})).toBeNull();
 expect(screen.queryByRole("button",{name:"新建或导入官方技能"})).toBeNull();
});
it("closes the detail and focuses the error if another version cannot be read",async()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const version={skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:2},version:1,state:"ENABLED",package:{name:"sample-skill",description:"真实方法",target:"product-title",digest:"a".repeat(64)}};
 const projection={version,canUse:true,canCopy:false,canExport:false,canEdit:false,canPublish:false};
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:false,canInstallOfficial:false,reason:""}:path.endsWith("/versions/2")?Promise.reject(new Error("offline")):path.endsWith("/versions/1")?projection:path.endsWith("/versions")?{items:[version,{...version,version:2}]}:{items:[projection],nextCursor:""});
 render(<SkillManager scope={{userId:"actor",organizationId:"org"}} home="personal" initialSpace="official" presentation="page"/>);
 fireEvent.click(await screen.findByRole("button",{name:/sample-skill/}));const dialog=await screen.findByRole("dialog",{name:"sample-skill"});
 fireEvent.change(within(dialog).getByLabelText("已保存版本"),{target:{value:"2"}});
 const alert = await screen.findByRole("alert");await waitFor(() => expect(alert).toHaveFocus());expect(screen.queryByRole("dialog",{name:"sample-skill"})).not.toBeInTheDocument();
});
it.each(["product-title-organize","product-title-localize","product-title-shorten","product-image-standard","product-image-scene","product-image-details"])("opens %s using the selected exact version rather than its head",async name=>{
 const image=name.startsWith("product-image"),id="11111111-1111-4111-8111-111111111111";
 const version={skill:{id,owner:{kind:"PLATFORM",id:"official"},headVersion:9},version:image?2:1,state:"ENABLED",package:{name,description:"真实方法",target:image?"product-image":"product-title",digest:"a".repeat(64)}};
 const projection={version,canUse:true,canCopy:false,canExport:false,canEdit:false,canPublish:false};
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:false,canInstallOfficial:false,reason:""}:path.endsWith(`/versions/${version.version}`)?projection:path.endsWith("/versions")?{items:[version]}:{items:[projection],nextCursor:""});
 render(<SkillManager scope={{userId:"actor",organizationId:"org"}} home="personal" initialSpace="official" presentation="page"/>);fireEvent.click(await screen.findByRole("button",{name:new RegExp(name)}));const link=await screen.findByRole("link",{name:"处理已有商品"});const parsed=parseSkillLaunch(new URL(link.getAttribute("href")!,"https://test").searchParams);expect(parsed).toEqual({userId:"actor",organizationId:"org",target:version.package.target,selection:{skillId:id,version:version.version,digest:version.package.digest}});
});
it("opens personal management without an effective organization and validates before saving",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,reason:""}:path.endsWith("/validate")?{valid:true,name:"product-title-normalization",description:"Normalize",digest:"a".repeat(64),fileCount:1}:{items:[],nextCursor:""});
 render(<SkillManagerTrigger scope={{userId:"alice"}} space="personal"/>);
 fireEvent.click(screen.getByRole("button",{name:"我的技能"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"新建技能"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"新建技能"}));
 fireEvent.click(screen.getByRole("button",{name:"检查格式"}));await screen.findByText(/格式检查通过/);
 expect(vi.mocked(skillRequest).mock.calls.some(([scope,path])=>scope.userId==="alice"&&!scope.organizationId&&path==="personal/validate")).toBe(true);
 expect(screen.queryByRole("button",{name:"生成建议"})).not.toBeInTheDocument();
});


