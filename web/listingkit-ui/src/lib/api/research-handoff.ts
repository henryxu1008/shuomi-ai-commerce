import {dataRequest} from "./data-services";
import {researchRequest} from "./market-research";
import {projectRequest,type ProjectScope} from "./project-center";
import {inspectResearch} from "./research-analysis";
import {fetchWorkbenchContext} from "./workbench-context";
import {jobSchema} from "../contracts/data-services";
import {researchFixtureStatus,researchPage,researchPaginationFixtureStatus} from "../contracts/market-research";
import {project} from "../contracts/project-center";
import {researchAnalysisInput,type ResearchInspection} from "../contracts/research-analysis";
import {candidateMatchesJob,researchFixtureJob,researchHandoffAllowed,researchPaginationFixtureJob,researchScopeKey,sameResearchJob,specificationFacts,type ResearchHandoff} from "../workbench/research-handoff";
import type {z} from "zod";

type Input = z.infer<typeof researchAnalysisInput>;
export function matchesResearchInspection(out:ResearchInspection, input:Input) {
  return out.candidates.length===input.candidates.length && input.candidates.every(choice=>
    out.candidates.some(row=>row.research.CandidateID===choice.candidateId &&
      row.research.CandidateRevision===choice.expectedRevision &&
      row.research.SelectedLinkID===choice.linkId && row.research.EvidenceDigest===choice.evidenceDigest &&
      row.ref.ItemID===choice.selection.itemId && row.ref.ItemRevision===choice.selection.expectedRevision &&
      row.ref.PublicationID===choice.selection.originalPublicationId && row.ref.CatalogVersion===choice.selection.originalVersion));
}

export async function verifyResearchHandoff(scope:ProjectScope,draft:ResearchHandoff,signal:AbortSignal,approved=false) {
  const confirmed=draft.confirmed;
  if (!confirmed) throw new Error("No confirmed research");
  const input=researchAnalysisInput.parse({candidates:confirmed.choices});
  const deadline=AbortSignal.any([signal,AbortSignal.timeout(10_000)]);
  const jobRequest=draft.job.state==="IMPORTED" && draft.job.fixture==="pagination"
    ? researchRequest(scope,draft.projectId,researchPaginationFixtureStatus,deadline,{fixture:"readPagination"}).then(result=>{
      if(!result.ready)throw new Error("Synthetic pagination samples unavailable");
      return researchPaginationFixtureJob(result.batch);
    })
    : draft.job.state==="IMPORTED"
    ? researchRequest(scope,draft.projectId,researchFixtureStatus,deadline,{fixture:"read"}).then(result=>{
      if(!result.ready)throw new Error("Synthetic research samples unavailable");
      return researchFixtureJob(result.batch);
    })
    : dataRequest(scope,"amazon/jobs/"+draft.job.id,jobSchema,undefined,deadline);
  const [currentProject,job,page,inspection]=await Promise.all([
    projectRequest(scope,"/"+draft.projectId,project,deadline),
    jobRequest,
    researchRequest(scope,draft.projectId,researchPage,deadline,{after:confirmed.after}),
    inspectResearch(scope,draft.projectId,input,deadline),
  ]);
  deadline.throwIfAborted();
  const representative=page.items.find(candidate=>candidate.id===confirmed.representative);
  if (currentProject.id!==draft.projectId || currentProject.archived || currentProject.kind!=="PRODUCT_RESEARCH" ||
      !sameResearchJob(draft.job,job) || page.incomplete || !representative ||
      approved && (representative.state!=="APPROVED_FOR_TEST" || representative.revision!==confirmed.approvalRevision) || inspection.projectId!==draft.projectId ||
      !matchesResearchInspection(inspection,input) || !input.candidates.every(choice=>page.items.some(candidate=>
        candidate.id===choice.candidateId && candidateMatchesJob(candidate,job) &&
        String(candidate.revision)===choice.expectedRevision && candidate.selectedLinkId===choice.linkId &&
        candidate.evidenceDigest===choice.evidenceDigest))) throw new Error("Research changed");
  const sources=new Set(inspection.candidates.flatMap(specificationFacts).map(fact=>fact.sourceKey));
  if (confirmed.requirements.specs.some(spec=>!sources.has(spec.sourceKey))) throw new Error("Specification source changed");
  const context=await fetchWorkbenchContext(deadline);
  const organization=context.organizations.find(org=>org.id===context.effectiveOrganizationId);
  const permissions=organization?.permissions??[];
  if (context.selectionRequired || !organization || !context.effectiveMemberId ||
      researchScopeKey(context.user.id,organization.id,context.effectiveMemberId,permissions)!==draft.scope ||
      !researchHandoffAllowed(permissions,approved)) throw new Error("Research authorization changed");
  deadline.throwIfAborted();
  return {inspection,representative};
}
