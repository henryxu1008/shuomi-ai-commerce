import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ProjectPage} from "./project-page";
const f=vi.hoisted(()=>({panel:vi.fn(),request:vi.fn(),gateAllowed:true,context:{user:{id:"user-a"},effectiveOrganization:{id:"org-a"},permissions:["workbench.project.read","workbench.collection.read"],marketResearchAvailable:true,projectCenterAvailable:true,isLoading:false,isSwitching:false,selectionRequired:false,error:null,blockingError:null}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/project-center",async original=>({...await original<typeof import("@/lib/api/project-center")>(),projectRequest:f.request}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:vi.fn()})}));
vi.mock("./research-selection-gate",()=>({ResearchSelectionGate:({children}:{children:(value:object,after:string)=>React.ReactNode})=>f.gateAllowed?<>{children({},"")}</>:<p role="alert">交接已失效</p>}));
vi.mock("./research-panel",()=>({ResearchPanel:(props:unknown)=>{f.panel(props);return <p>交接内容</p>;}}));
afterEach(()=>{cleanup();sessionStorage.clear();f.panel.mockReset();f.request.mockReset();f.gateAllowed=true;});

it("passes the selected step through the verified handoff panel",async()=>{
 const id="550e8400-e29b-41d4-a716-446655440000";
 f.request.mockImplementation(async(_scope,path)=>path.endsWith("/visit")?{id,revision:1}:{id,title:"调研项目",goal:"目标",kind:"PRODUCT_RESEARCH",dueDate:"",revision:1,archived:false,references:[]});
 render(<ProjectPage center="selection" projectId={id} workflowStep="transfer" researchToken="original-token"/>);
 await screen.findByText("交接内容");
 await waitFor(()=>expect(f.panel).toHaveBeenLastCalledWith(expect.objectContaining({workflowStep:"transfer",researchToken:"original-token"})));
});

it("does not expose steps when the original handoff gate rejects access",async()=>{
 const id="550e8400-e29b-41d4-a716-446655440000";
 f.gateAllowed=false;
 f.request.mockImplementation(async(_scope,path)=>path.endsWith("/visit")?{id,revision:1}:{id,title:"调研项目",goal:"目标",kind:"PRODUCT_RESEARCH",dueDate:"",revision:1,archived:false,references:[]});
 render(<ProjectPage center="selection" projectId={id} workflowStep="transfer" researchToken="bad-token"/>);
 expect(await screen.findByRole("alert")).toHaveTextContent("交接已失效");
 expect(screen.queryByRole("navigation",{name:"智能选品步骤"})).toBeNull();
 expect(f.panel).not.toHaveBeenCalled();
});
