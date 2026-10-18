import {afterEach,beforeAll,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {SkillManagerTrigger,SkillManager} from "./skill-manager";
import {skillRequest} from "@/lib/api/skill";
vi.mock("@/lib/api/skill",()=>({skillRequest:vi.fn(),skillError:()=>"技能服务不可用"}));
beforeAll(()=>{HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")}});
afterEach(()=>{cleanup();vi.clearAllMocks()});
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
