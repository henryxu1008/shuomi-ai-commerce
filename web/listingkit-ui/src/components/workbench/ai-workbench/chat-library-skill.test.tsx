import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ChatSkillDrawer} from "./chat-skill-authoring";
import {ChatLibrarySkillSelection,useChatLibrarySkill} from "./chat-library-skill";
import {parseSkillLaunch,rememberChatSkillLocation,skillLaunchHref} from "@/lib/contracts/skill-launch";
import type {SkillVersion} from "@/lib/contracts/skill";
const f=vi.hoisted(()=>({skills:vi.fn(),workbench:vi.fn()}));
vi.mock("@/lib/api/skill",()=>({skillRequest:f.skills}));
vi.mock("@/lib/api/ai-workbench",()=>({requestAIWorkbench:f.workbench,AIWorkbenchError:class extends Error{}}));
const id="11111111-1111-4111-8111-111111111111",version=(kind:string,name:string,target="product-title")=>({skill:{id:kind==="PLATFORM"?"66666666-6666-4666-8666-666666666666":id,owner:{kind,id:kind==="USER"?"actor":"org"}},version:2,package:{name,target,digest:"a".repeat(64),description:"按已核实商品处理"}});
function draw(){const chosen=vi.fn();render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><ChatSkillDrawer scope={{userId:"actor",organizationId:"org"}} authorizationKey="permissions" onSelect={vi.fn()} onSelectLibrary={chosen}/></QueryClientProvider>);fireEvent.click(screen.getByRole("button",{name:"选择技能"}));return chosen}
beforeEach(()=>{vi.clearAllMocks();HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=vi.fn();f.workbench.mockResolvedValue({skill:{name:"创建技能",version:2,digest:"b".repeat(64),description:"生成技能草稿"},ready:true});f.skills.mockImplementation(async(_scope,path)=>({items:path.includes("image")?[version("ORGANIZATION","团队图片规则","product-image")]:[version("USER","我的标题规则"),version("PLATFORM","官方标题规则")]}))});afterEach(cleanup);
it("lists admitted personal, official and team image skills next to creator without submitting",async()=>{const chosen=draw();fireEvent.click(await screen.findByRole("button",{name:/我的标题规则/}));expect(chosen).toHaveBeenCalledWith(expect.objectContaining({version:2,package:expect.objectContaining({name:"我的标题规则"})}));expect(f.workbench.mock.calls.every(([v])=>v.method==="GET")).toBe(true);expect(f.skills.mock.calls.every(([,p])=>["organization/candidates","organization/image-candidates"].includes(p))).toBe(true)});
it("keeps image and creator available when title candidates fail, and retries the failed group",async()=>{let failed=true;f.skills.mockImplementation(async(_s,p)=>{if(!p.includes("image")&&failed)throw Error("unavailable");return {items:[version("ORGANIZATION",p.includes("image")?"团队图片规则":"团队标题规则",p.includes("image")?"product-image":"product-title")]}});draw();await screen.findByText("标题技能读取失败，请重试。");expect(screen.getByRole("button",{name:/团队图片规则/})).toBeEnabled();expect(screen.getByRole("button",{name:/创建技能/})).toBeEnabled();failed=false;fireEvent.click(screen.getByRole("button",{name:"重新读取标题技能"}));await waitFor(()=>expect(screen.getByRole("button",{name:/团队标题规则/})).toBeEnabled())});
it("filters the same admitted candidates using the library's official, mine and team categories",async()=>{draw();await screen.findByRole("button",{name:/我的标题规则/});fireEvent.click(screen.getByRole("button",{name:"官方模板"}));expect(screen.queryByRole("button",{name:/我的标题规则/})).toBeNull();expect(screen.getByRole("button",{name:/官方标题规则/})).toBeEnabled();expect(screen.getByRole("button",{name:/创建技能/})).toBeEnabled();fireEvent.click(screen.getByRole("button",{name:"我的技能"}));expect(screen.getByRole("button",{name:/我的标题规则/})).toBeEnabled();expect(screen.queryByRole("button",{name:/官方标题规则/})).toBeNull();expect(screen.queryByRole("button",{name:/创建技能/})).toBeNull();fireEvent.click(screen.getByRole("button",{name:"团队技能"}));expect(screen.getByRole("button",{name:/团队图片规则/})).toBeEnabled();expect(screen.queryByRole("button",{name:/我的标题规则/})).toBeNull();});

it("allows another product after reselecting the same skill and keeps both original request links",async()=>{
 const scope={userId:"actor",organizationId:"org"};
 const chat="/workbench/ai/chat/55555555-5555-4555-8555-555555555555";
 const chosen=version("USER","我的标题规则") as SkillVersion;
 function Selection(){
  const choice=useChatLibrarySkill(scope,chat,"permissions");
  return <>
   <button onClick={()=>choice.choose(chosen)}>选我的标题规则</button>
   {choice.launch&&<ChatLibrarySkillSelection launch={choice.launch} authorizationKey="permissions" onClear={()=>choice.choose()}/>}
  </>;
 }
 sessionStorage.clear();
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 const tree=<QueryClientProvider client={client}><Selection/></QueryClientProvider>;
 const initial=render(tree);
 fireEvent.click(screen.getByRole("button",{name:"选我的标题规则"}));
 const firstLink=await screen.findByRole("link",{name:"前往标题处理页"});
 const launch=parseSkillLaunch(new URL(firstLink.getAttribute("href")!,"https://test").searchParams);
 if(!launch||"invalid" in launch)throw Error("missing skill selection");
 const firstPath=skillLaunchHref("/workbench/data/mine/own/33333333-3333-4333-8333-333333333333/optimize?agent_key=44444444-4444-4444-8444-444444444444&agent_actor=actor&agent_org=org",launch);
 rememberChatSkillLocation(launch,firstPath);
 initial.unmount();
 const returned=render(tree);
 expect(await screen.findByRole("link",{name:/核实或查看原标题操作/})).toHaveAttribute("href",firstPath);
 fireEvent.click(screen.getByRole("button",{name:"取消选择"}));
 fireEvent.click(screen.getByRole("button",{name:"选我的标题规则"}));
 const nextLink=await screen.findByRole("link",{name:"选择另一件商品"});
 const nextUrl=new URL(nextLink.getAttribute("href")!,"https://test");
 expect(nextUrl.pathname).toBe("/workbench/agents/skills/official/use");
 expect(nextUrl.searchParams.has("agent_key")).toBe(false);
 const secondPath=skillLaunchHref("/workbench/data/mine/own/66666666-6666-4666-8666-666666666666/optimize?agent_key=77777777-7777-4777-8777-777777777777&agent_actor=actor&agent_org=org",launch);
 rememberChatSkillLocation(launch,secondPath);
 returned.unmount();
 render(tree);
 expect(await screen.findByRole("link",{name:/第 2 次/})).toHaveAttribute("href",secondPath);
 expect(screen.getByRole("link",{name:/第 1 次/})).toHaveAttribute("href",firstPath);
 sessionStorage.clear();
});
