import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { ChatContextPicker } from "./chat-context-picker";

const fixture=vi.hoisted(()=>({templates:vi.fn(),knowledge:vi.fn()}));
vi.mock("@/lib/api/agent-configuration",()=>({configurationRequest:fixture.templates}));
vi.mock("@/lib/api/knowledge",async original=>({...await original<object>(),knowledgeRequest:fixture.knowledge}));

it("lets a user choose names without automatically selecting or exposing internal identifiers",async()=>{
 const template={templateId:"11111111-1111-4111-8111-111111111111",agentId:"product.content.agent",name:"英文模板",version:"3",lifecycle:"ACTIVE",targetPlatform:"product"};
 const base={id:"22222222-2222-4222-8222-222222222222",name:"品牌知识",state:"ACTIVE"};
 fixture.templates.mockResolvedValue({items:[template],nextCursor:""});
 fixture.knowledge.mockResolvedValue({items:[base],pagination:{page:1,pageSize:100,total:1}});
 const chooseTemplate=vi.fn(),chooseKnowledge=vi.fn(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 render(<QueryClientProvider client={client}><ChatContextPicker scope={{userId:"u",organizationId:"o"}} authorizationKey="a" agentId="product.content.agent" target="product" enabled disabled={false} templateId="" templateRevision="" knowledgeBaseId="" onTemplate={chooseTemplate} onKnowledge={chooseKnowledge}/></QueryClientProvider>);
 expect(await screen.findByRole("option",{name:"英文模板"})).toBeInTheDocument();
 expect(await screen.findByRole("option",{name:"品牌知识"})).toBeInTheDocument();
 expect(chooseTemplate).not.toHaveBeenCalled();expect(chooseKnowledge).not.toHaveBeenCalled();
 fireEvent.change(screen.getByLabelText("模板"),{target:{value:`${template.templateId}@3`}});
 expect(chooseTemplate).toHaveBeenCalledWith(template.templateId,"3");
 fireEvent.change(screen.getByLabelText("知识库"),{target:{value:base.id}});
 expect(chooseKnowledge).toHaveBeenCalledWith(base.id);
 expect(screen.queryByText(/11111111|22222222/)).not.toBeInTheDocument();
 client.clear();
});

it("keeps an unavailable prior selection for recovery and allows clearing it without submitting",async()=>{
 fixture.templates.mockRejectedValue(new Error("unavailable"));
 fixture.knowledge.mockResolvedValue({items:[],pagination:{page:1,pageSize:100,total:0}});
 const chooseTemplate=vi.fn(),ready=vi.fn(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 render(<QueryClientProvider client={client}><ChatContextPicker scope={{userId:"u",organizationId:"o"}} authorizationKey="a" agentId="product.content.agent" target="product" enabled disabled={false} templateId="11111111-1111-4111-8111-111111111111" templateRevision="3" knowledgeBaseId="" onTemplate={chooseTemplate} onKnowledge={()=>{}} onReady={ready}/></QueryClientProvider>);
 expect(await screen.findByText("模板暂时无法读取。",{exact:false})).toBeVisible();
 expect(ready).toHaveBeenLastCalledWith(false);
 expect(screen.getByLabelText("模板")).toBeEnabled();
 fireEvent.change(screen.getByLabelText("模板"),{target:{value:""}});
 expect(chooseTemplate).toHaveBeenCalledWith("","");
 client.clear();
});

it("does not silently substitute another version or an incompatible template",async()=>{
 const id="11111111-1111-4111-8111-111111111111";
 fixture.templates.mockResolvedValue({items:[{templateId:id,agentId:"product.content.agent",name:"新版本",version:"4",lifecycle:"ACTIVE",targetPlatform:"product"},{templateId:"33333333-3333-4333-8333-333333333333",agentId:"product.content.agent",name:"其他平台",version:"1",lifecycle:"ACTIVE",targetPlatform:"shein"}],nextCursor:""});
 fixture.knowledge.mockResolvedValue({items:[],pagination:{page:1,pageSize:100,total:0}});
 const choose=vi.fn(),ready=vi.fn(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 render(<QueryClientProvider client={client}><ChatContextPicker scope={{userId:"u",organizationId:"o"}} authorizationKey="a" agentId="product.content.agent" target="product" enabled disabled={false} templateId={id} templateRevision="3" knowledgeBaseId="" onTemplate={choose} onKnowledge={()=>{}} onReady={ready}/></QueryClientProvider>);
 expect(await screen.findByText("此前的选择暂不可用，请重新选择后发送。")).toBeVisible();
 expect(await screen.findByRole("option",{name:"新版本"})).toBeInTheDocument();
 expect(ready).toHaveBeenLastCalledWith(false);
 expect(screen.queryByRole("option",{name:"其他平台"})).not.toBeInTheDocument();
 expect(choose).not.toHaveBeenCalled();
 fireEvent.change(screen.getByLabelText("模板"),{target:{value:`${id}@4`}});
 expect(choose).toHaveBeenCalledWith(id,"4");
 client.clear();
});

it("hides previously loaded choices after access is lost",async()=>{
 const id="11111111-1111-4111-8111-111111111111";
 fixture.templates.mockResolvedValueOnce({items:[{templateId:id,agentId:"product.content.agent",name:"私有模板",version:"1",lifecycle:"ACTIVE",targetPlatform:"product"}],nextCursor:""}).mockRejectedValue(new Error("forbidden"));
 fixture.knowledge.mockResolvedValue({items:[],pagination:{page:1,pageSize:100,total:0}});
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}}),ready=vi.fn();
 render(<QueryClientProvider client={client}><ChatContextPicker scope={{userId:"u",organizationId:"o"}} authorizationKey="a" agentId="product.content.agent" target="product" enabled disabled={false} templateId={id} templateRevision="1" knowledgeBaseId="" onTemplate={()=>{}} onKnowledge={()=>{}} onReady={ready}/></QueryClientProvider>);
 expect(await screen.findByRole("option",{name:"私有模板"})).toBeInTheDocument();
 await act(async()=>{await client.invalidateQueries({queryKey:["chat-context-templates"]})});
 await waitFor(()=>expect(screen.queryByRole("option",{name:"私有模板"})).not.toBeInTheDocument());
 expect(ready).toHaveBeenLastCalledWith(false);
 client.clear();
});
