import {afterEach,beforeAll,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {SkillManagerTrigger,SkillManager} from "./skill-manager";
import {skillRequest} from "@/lib/api/skill";
import {parseSkillLaunch} from "@/lib/contracts/skill-launch";
vi.mock("@/lib/api/skill",()=>({skillRequest:vi.fn(),skillError:()=>"技能服务不可用"}));
beforeAll(()=>{HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")}});
afterEach(()=>{cleanup();vi.clearAllMocks()});
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

it("installs the official release only through the maintainer action with no client packages",async()=>{
 vi.mocked(skillRequest).mockImplementation(async(_scope,path)=>path.endsWith("/info")?{available:true,canCreate:true,canInstallOfficial:true,reason:""}:path.endsWith("/install")?{installed:[],existing:[]}:{items:[],nextCursor:""});
 render(<SkillManager scope={{userId:"maintainer"}} home="personal" initialSpace="official" presentation="page"/>);
 const install=await screen.findByRole("button",{name:"安装官方模板"});fireEvent.click(install);
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path,,init])=>path==="official/install"&&init?.body==="{}")).toBe(true));
 expect(screen.queryByRole("button",{name:"新建技能"})).not.toBeInTheDocument();
});
