import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {SkillUsePage} from "./skill-use-page";
import {parseSkillLaunch} from "@/lib/contracts/skill-launch";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.collection.read","workbench.supply.read","workbench.supply.manage","listingkit.image_agent.read","listingkit.image_agent.write","local_agent.write","workbench.agent.use"]},products:vi.fn(),skills:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/product-collection",()=>({listOwnProducts:f.products}));
vi.mock("@/lib/api/skill",()=>({skillRequest:f.skills}));
const id="11111111-1111-4111-8111-111111111111",item="22222222-2222-4222-8222-222222222222";
const launch={userId:"actor",organizationId:"org",target:"product-image" as const,selection:{skillId:id,version:2,digest:"a".repeat(64)}};
beforeEach(()=>{vi.clearAllMocks();f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.collection.read","workbench.supply.read","workbench.supply.manage","listingkit.image_agent.read","listingkit.image_agent.write","local_agent.write","workbench.agent.use"];f.skills.mockResolvedValue({items:[{skill:{id,owner:{kind:"PLATFORM"}},version:2,package:{name:"image",displayName:"标准商品套图",target:"product-image",digest:launch.selection.digest}}]});f.products.mockResolvedValue({items:[{id:item,title:"已有商品",source:{kind:"own",version:"1"}}],nextCursor:"",total:1})});
afterEach(cleanup);
it("reads existing products and carries exact image choice to explicit transfer without writing",async()=>{
 render(<SkillUsePage launch={launch} titleAvailable imageAvailable/>);
 const crumb=within(screen.getByRole("navigation",{name:"面包屑"}));expect(crumb.getByRole("link",{name:"AI工作台"})).toHaveAttribute("href","/workbench/ai");expect(crumb.getByRole("link",{name:"技能库"})).toHaveAttribute("href","/workbench/agents/skills");
 const link=await screen.findByRole("link",{name:"选择此商品"});const url=new URL(link.getAttribute("href")!,"https://test");expect(url.pathname).toBe(`/workbench/data/mine/own/${item}/prepare`);expect(parseSkillLaunch(url.searchParams)).toEqual(launch);
 expect(f.products).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},{keyword:"",limit:20,after:undefined},expect.any(AbortSignal));
 expect(f.skills.mock.calls.every(([,path])=>path==="organization/image-candidates")).toBe(true);
});
it("supports existing product search and cursor paging",async()=>{
 f.products.mockResolvedValueOnce({items:[],total:2,nextCursor:"next"}).mockResolvedValue({items:[{id:item,title:"搜索结果",source:{kind:"own",version:"1"}}],total:1,nextCursor:""});render(<SkillUsePage launch={launch} titleAvailable imageAvailable/>);
 fireEvent.click(await screen.findByRole("button",{name:"下一页"}));await waitFor(()=>expect(f.products).toHaveBeenLastCalledWith(expect.anything(),expect.objectContaining({after:"next"}),expect.anything()));
 fireEvent.change(screen.getByLabelText("商品名称"),{target:{value:"灯"}});fireEvent.click(screen.getByRole("button",{name:"搜索"}));await waitFor(()=>expect(f.products).toHaveBeenLastCalledWith(expect.anything(),{keyword:"灯",after:undefined,limit:20},expect.anything()));
});
it("does not read products or offer launch after organization drift or missing permission",()=>{
 f.context.effectiveOrganization.id="other";const view=render(<SkillUsePage launch={launch} titleAvailable imageAvailable/>);expect(screen.getByText("技能入口已失效或身份、企业已变化")).toBeVisible();expect(f.products).not.toHaveBeenCalled();expect(f.skills).not.toHaveBeenCalled();
 f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.collection.read"];view.rerender(<SkillUsePage launch={launch} titleAvailable imageAvailable/>);expect(screen.getByText("当前身份没有此技能所需的商品处理权限")).toBeVisible();expect(f.products).not.toHaveBeenCalled();
});
it("does not replace an unavailable exact version with a newer version",async()=>{
 f.skills.mockResolvedValue({items:[]});render(<SkillUsePage launch={launch} titleAvailable imageAvailable/>);await screen.findByText(/所选准确版本当前不可执行/);expect(screen.queryByRole("link",{name:"选择此商品"})).not.toBeInTheDocument();expect(f.products).not.toHaveBeenCalled();
});
