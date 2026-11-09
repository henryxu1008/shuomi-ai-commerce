import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ResearchAssessments} from "./research-assessments";
import {blankCostDraft} from "@/lib/workbench/research-cost-preview";
import type {ResearchCandidate} from "@/lib/contracts/market-research";
import type {AssessmentView} from "@/lib/contracts/research-assessment";
const mocks=vi.hoisted(()=>({request:vi.fn()}));
vi.mock("@/lib/api/market-research",async importOriginal=>({...await importOriginal<object>(),researchRequest:mocks.request}));
afterEach(()=>{cleanup();vi.clearAllMocks()});
const id="11111111-1111-4111-8111-111111111111",savedId="22222222-2222-4222-8222-222222222222";
const ref={assessmentId:savedId,revision:1 as const,inputHash:"a".repeat(64),contextHash:"b".repeat(64)};
const candidate={id,revision:2,selectedLinkId:id,evidenceDigest:"c".repeat(64),assessment:true,observation:{site:"us"},history:[]} as unknown as ResearchCandidate;
const saved:AssessmentView={id:savedId,candidateRevision:3,at:"2026-11-09T00:00:00Z",applicability:"CURRENT",detail:{ref,input:blankCostDraft(),result:{status:"INSUFFICIENT_INPUT",fields:["price"]},market:{itemId:id,revision:1,publicationId:id,version:"1"},ruleVersion:"manual-unit-economics-v1",supplierBinding:"UNKNOWN"}};
const scope={userId:"user",organizationId:"org"};
it("previews on the server, saves through the original command owner and finds the same record after remount",async()=>{
 let rows:AssessmentView[]=[];
 mocks.request.mockImplementation(async(_s,_p,_schema,_signal,options)=>options?.assessment?.preview?{input:options.assessment.preview.assessment.cost,result:saved.detail!.result,ruleVersion:"manual-unit-economics-v1"}:options?.assessment?.id?saved:{items:rows});
 const save=vi.fn(async command=>{expect(command.assessment.cost.price).toBe("30");rows=[saved];return {key:savedId,candidateIds:[id],revisions:{[id]:3},replayed:false,assessmentRef:ref}});
 const choose=vi.fn();const props={scope,projectId:id,candidate,disabled:false,canWrite:true,onSave:save,onChoose:choose};
 const view=render(<ResearchAssessments {...props}/>);
 fireEvent.click(screen.getByRole("button",{name:"填写经营假设"}));
 fireEvent.change(screen.getByLabelText("假设售价（USD/件）"),{target:{value:"30"}});
 fireEvent.click(screen.getByRole("button",{name:"查看参考试算"}));
 expect(await screen.findByText("资料仍有缺项")).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"保存本次评估"}));
 await waitFor(()=>expect(save).toHaveBeenCalledOnce());
 view.unmount();render(<ResearchAssessments {...props}/>);
 expect(await screen.findByText(savedId)).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"查看并使用此记录中的条件"}));
 await waitFor(()=>expect(choose).toHaveBeenLastCalledWith(ref));
 expect(screen.getByText(/使用此记录中的条件/,{selector:"p"})).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"填写经营假设"}));
 fireEvent.change(screen.getByLabelText("假设售价（USD/件）"),{target:{value:"31"}});
 expect(choose).toHaveBeenLastCalledWith(undefined);
 fireEvent.click(screen.getByRole("button",{name:"查看并使用此记录中的条件"}));
 expect(screen.getByText(/有未保存的编辑/)).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"放弃编辑并使用原记录"}));
 await waitFor(()=>expect(choose).toHaveBeenLastCalledWith(ref));
});
it("clears protected results when the exact saved record becomes unreadable",async()=>{
 mocks.request.mockImplementation(async(_s,_p,_schema,_signal,options)=>options?.assessment?.id?{...saved,applicability:"DETAIL_UNAVAILABLE",detail:undefined}:{items:[saved]});
 const choose=vi.fn();render(<ResearchAssessments scope={scope} projectId={id} candidate={candidate} disabled={false} canWrite onSave={vi.fn()} onChoose={choose}/>);
 await screen.findByText(savedId);fireEvent.click(screen.getByRole("button",{name:"查看并使用此记录中的条件"}));
 expect(await screen.findByText(/原依据暂不可读，已遮蔽/)).toBeVisible();expect(choose).toHaveBeenLastCalledWith(undefined);
 expect(screen.queryByText("资料仍有缺项")).not.toBeInTheDocument();
});

it("revalidates the explicit record after a technical revision without replacing it with latest",async()=>{
 mocks.request.mockImplementation(async(_s,_p,_schema,_signal,options)=>options?.assessment?.id?saved:{items:[]});
 const choose=vi.fn();const props={scope,projectId:id,candidate,disabled:false,canWrite:true,onSave:vi.fn(),onChoose:choose,selectedRef:ref};
 const view=render(<ResearchAssessments {...props}/>);
 await waitFor(()=>expect(choose).toHaveBeenLastCalledWith(ref));
 expect(screen.getByText(/使用此记录中的条件/,{selector:"p"})).toBeVisible();
 view.rerender(<ResearchAssessments {...props} candidate={{...candidate,revision:4}}/>);
 await waitFor(()=>expect(mocks.request.mock.calls.filter(call=>call[4]?.assessment?.id)).toHaveLength(2));
 expect(mocks.request.mock.calls.filter(call=>call[4]?.assessment?.id).every(call=>call[4].assessment.id===savedId)).toBe(true);
});

