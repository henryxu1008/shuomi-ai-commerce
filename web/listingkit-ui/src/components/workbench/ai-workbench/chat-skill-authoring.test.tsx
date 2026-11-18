import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ChatSkillAuthoring,ChatSkillTurn} from "./chat-skill-authoring";
import {requestAIWorkbench} from "@/lib/api/ai-workbench";
import type {AISkillAuthoringJob} from "@/lib/contracts/ai-workbench";
vi.mock("@/lib/api/ai-workbench",async original=>({...await original<object>(),requestAIWorkbench:vi.fn().mockResolvedValue({valid:true})}));
afterEach(cleanup);
it("preserves an edited draft when an older history read changes revision during editing",async()=>{
 const scope={userId:"user",organizationId:"org"},key="11111111-1111-4111-8111-111111111111",revision="22222222-2222-4222-8222-222222222222";
 const job:AISkillAuthoringJob={key,generationKey:key,revision:key,state:"COMPLETE",requirement:"生成",billing:"committed",savePending:false,savedSkillId:null,draft:{name:"title",displayName:"标题",description:"整理",instructions:"原规则"}};
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 const tree=(current:AISkillAuthoringJob)=><QueryClientProvider client={client}><ChatSkillAuthoring scope={scope} authorizationKey="member" conversation={key} initialJob={current}/></QueryClientProvider>;
 const view=render(tree(job));
 fireEvent.change(screen.getByLabelText("正文"),{target:{value:"保留我的人工编辑"}});
 view.rerender(tree({...job,revision,generationKey:revision,draft:{...job.draft!,instructions:"新回复"}}));
 await waitFor(()=>expect(screen.getByLabelText("正文")).toHaveValue("保留我的人工编辑"));
 expect(screen.getByRole("button",{name:"保存到我的技能"})).toBeDisabled();
 expect(await screen.findByText(/当前草稿已更新/)).toBeVisible();
});
it("keeps the message editor mounted across a fresh canonical revision read",async()=>{
 const scope={userId:"user",organizationId:"org"},key="11111111-1111-4111-8111-111111111111",revision="22222222-2222-4222-8222-222222222222";
 const job:AISkillAuthoringJob={key,generationKey:key,revision:key,state:"COMPLETE",requirement:"生成",billing:"committed",savePending:false,savedSkillId:null,draft:{name:"title",displayName:"标题",description:"整理",instructions:"原规则"}};
 const queryKey=["chat-turn","user","org","member",key,key];
 const client=new QueryClient({defaultOptions:{queries:{retry:false,refetchOnMount:false}}});
 const result={state:"COMPLETE",kind:"CREATE_TITLE_SKILL",skill:{version:2,digest:"a".repeat(64)},authoring:job};
 client.setQueryData(queryKey,result);
 render(<QueryClientProvider client={client}><ChatSkillTurn scope={scope} authorizationKey="member" conversation={key} turnKey={key} onRevise={vi.fn()}/></QueryClientProvider>);
 fireEvent.change(screen.getByLabelText("正文"),{target:{value:"保留我的人工编辑"}});
 await act(async()=>{client.setQueryData(queryKey,{...result,authoring:{...job,revision,generationKey:revision,draft:{...job.draft!,instructions:"新回复"}}});});
 await waitFor(()=>expect(screen.getByLabelText("正文")).toHaveValue("保留我的人工编辑"));
 expect(screen.getByRole("button",{name:"保存到我的技能"})).toBeDisabled();
 fireEvent.click(screen.getByRole("button",{name:"使用最新草稿并替换当前编辑"}));
 expect(screen.getByLabelText("正文")).toHaveValue("新回复");
});

it("keeps manually edited content when the original save was never accepted", async () => {
 const scope={userId:"user",organizationId:"org"},key="11111111-1111-4111-8111-111111111111";
 const job:AISkillAuthoringJob={key,generationKey:key,revision:key,state:"COMPLETE",requirement:"生成",billing:"committed",savePending:false,savedSkillId:null,draft:{name:"title",displayName:"标题",description:"整理",instructions:"原规则"}};
 const request=vi.mocked(requestAIWorkbench);
 request.mockImplementation(async (args) => {
  if(args.route==="skill-authoring-save-package") throw new Error("connection lost before dispatch");
  if(args.route==="skill-authoring-save-recover") return job;
  return {valid:true};
 });
 try {
  const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
  render(<QueryClientProvider client={client}><ChatSkillAuthoring scope={scope} authorizationKey="member" conversation={key} initialJob={job}/></QueryClientProvider>);
  fireEvent.change(screen.getByLabelText("正文"),{target:{value:"保留我的人工编辑"}});
  await waitFor(()=>expect(screen.getByRole("button",{name:"保存到我的技能"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button",{name:"保存到我的技能"}));
  fireEvent.click(await screen.findByRole("button",{name:"核实原保存"}));
  await waitFor(()=>expect(screen.getByLabelText("正文")).toHaveValue("保留我的人工编辑"));
 } finally { request.mockReset().mockResolvedValue({valid:true}); }
});

it.each(["newer canonical revision","lagging canonical projection"])("preserves the correct draft after recovering with %s",async incoming=>{
 const scope={userId:"user",organizationId:"org"},key="11111111-1111-4111-8111-111111111111",second="22222222-2222-4222-8222-222222222222",third="33333333-3333-4333-8333-333333333333";
 const job:AISkillAuthoringJob={key,generationKey:key,revision:key,state:"COMPLETE",requirement:"生成",billing:"committed",savePending:false,savedSkillId:null,draft:{name:"title",displayName:"标题",description:"整理",instructions:"原规则"}};
 const fresh=(revision:string,instructions:string)=>({...job,revision,generationKey:revision,draft:{...job.draft!,instructions}});
 const request=vi.mocked(requestAIWorkbench);
 request.mockImplementation(async args=>{
  if(args.route==="skill-authoring-save-package")throw new Error("connection lost before dispatch");
  if(args.route==="skill-authoring-save-recover")return fresh(second,"核实时的规则");
  return {valid:true};
 });
 try{
  const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
  const tree=(current:AISkillAuthoringJob)=><QueryClientProvider client={client}><ChatSkillAuthoring scope={scope} authorizationKey="member" conversation={key} initialJob={current}/></QueryClientProvider>;
  const view=render(tree(job));
  fireEvent.change(screen.getByLabelText("正文"),{target:{value:"人工规则"}});
  await waitFor(()=>expect(screen.getByRole("button",{name:"保存到我的技能"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button",{name:"保存到我的技能"}));
  fireEvent.click(await screen.findByRole("button",{name:"核实原保存"}));
  await screen.findByRole("button",{name:"使用最新草稿并替换当前编辑"});
  const advanced=incoming==="newer canonical revision";
  if(advanced) await act(async()=>{view.rerender(tree(fresh(third,"最新规则")));});
  expect(screen.getByLabelText("正文")).toHaveValue("人工规则");
  fireEvent.click(screen.getByRole("button",{name:"使用最新草稿并替换当前编辑"}));
  if(!advanced) await act(async()=>{view.rerender(tree({...job,requirement:"原请求的更新投影"}));});
  expect(screen.getByLabelText("正文")).toHaveValue(advanced?"最新规则":"核实时的规则");
  await waitFor(()=>expect(screen.getByRole("button",{name:"保存到我的技能"})).toBeEnabled());
 }finally{request.mockReset().mockResolvedValue({valid:true});}
});
