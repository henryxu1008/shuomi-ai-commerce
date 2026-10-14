import { agentResultSchema, contentAgentResultSchema, agentReviewLinkSchema, agentStartRequestSchema, contentAgentStartRequestSchema } from "../contracts/product-agent";
import { sameContentSelection, type ContentSelection } from "../contracts/content-selection";
import { isAcquisitionUUID } from "../contracts/product-acquisition";
import type {TemplateRef} from "../contracts/agent-configuration";
import {sameSkill,type SkillSelection} from "../contracts/skill";
import type { AcquisitionContext } from "./product-acquisition";
import { readBoundedStrictJSON } from "./strict-json-response";
export class ProductAgentError extends Error {
    constructor(public code: string) { super(code); }
}
export async function requestProductAgent(action: "start" | "read" | "resume" | "review", operationId: string, key: string, scope: AcquisitionContext, signal: AbortSignal, revision?: string, feedback?: string, targetPlatform?: string, knowledgeBaseId?: string,templateSelection?:TemplateRef,expectedModeRevision?:string,contentSelection?:ContentSelection,kind:"title"|"content"="title",skillSelection?:SkillSelection) {
    const content=kind==="content";
    const start={expectedModeRevision,targetPlatform,...(templateSelection?{templateSelection}:{}),...(knowledgeBaseId?{knowledgeSelection:{knowledgeBaseId}}:{}),...(contentSelection?{contentSelection}:{}),...(skillSelection?{skillSelection}:{})};
    if (!isAcquisitionUUID(operationId) || !isAcquisitionUUID(key) || action === "start" && !(content?contentAgentStartRequestSchema:agentStartRequestSchema).safeParse(start).success)
        throw new ProductAgentError("INVALID_AGENT_REQUEST");
    const base = `/api/workbench/sourcing/1688/acquisitions/${operationId}/${content?"content-agent":"product-agent"}/runs`;
    const path = action === "start" ? base : `${base}/${key}${action === "read" ? "" : `/${action}`}`;
    const controller = new AbortController();
    const abort = () => controller.abort();
    signal.addEventListener("abort", abort, { once: true });
    if (signal.aborted)
        abort();
    const timeout = setTimeout(abort, 130000);
    try {
        const headers = new Headers({ "X-Expected-User-ID": scope.userId, "X-Expected-Organization-ID": scope.organizationId, Accept: "application/json" });
        if (action !== "read") {
            headers.set("Content-Type", "application/json");
            if (action === "start")
                headers.set("Idempotency-Key", key);
        }
        const response = await fetch(path, { method: action === "read" ? "GET" : "POST", headers, body: action === "read" ? undefined : JSON.stringify(action === "start" ? start : action === "resume" ? { revision, feedback } : {}), credentials: "same-origin", redirect: "error", cache: "no-store", signal: controller.signal });
        const payload = await readBoundedStrictJSON(response, 128 * 1024, controller.signal);
        if (!response.ok) {
            const code = payload && typeof payload === "object" && "code" in payload && typeof payload.code === "string" ? payload.code : "OUTCOME_UNKNOWN";
            throw new ProductAgentError(code);
        }
        const result = (action === "review" ? agentReviewLinkSchema : content?contentAgentResultSchema:agentResultSchema).safeParse(payload);
        if (response.status !== 200 || !result.success || result.data.operationId !== operationId || result.data.requestKey !== key)
            throw new ProductAgentError("OUTCOME_UNKNOWN");
        if (action === "start" && "targetPlatform" in result.data && result.data.targetPlatform !== targetPlatform)
            throw new ProductAgentError("OUTCOME_UNKNOWN");
        if (action === "start" && contentSelection && "contentSelection" in result.data && !sameContentSelection(contentSelection,result.data.contentSelection)) throw new ProductAgentError("OUTCOME_UNKNOWN");
        if(action==="start"&&content&&"contentSelection" in result.data){const ref=result.data.skillVersionRef;if(!sameSkill(skillSelection,ref?{skillId:ref.id,version:ref.version,digest:ref.digest}:undefined))throw new ProductAgentError("OUTCOME_UNKNOWN")}
        if ("policyAdoption" in result.data && result.data.policyAdoption && (result.data.policyAdoption.identity.ActorID!==scope.userId || result.data.policyAdoption.identity.OrganizationID!==scope.organizationId)) throw new ProductAgentError("OUTCOME_UNKNOWN");
        return result.data;
    }
    catch (error) {
        if (error instanceof ProductAgentError)
            throw error;
        throw new ProductAgentError(action === "read" ? "DEPENDENCY_UNAVAILABLE" : "OUTCOME_UNKNOWN");
    }
    finally {
        clearTimeout(timeout);
        signal.removeEventListener("abort", abort);
    }
}
