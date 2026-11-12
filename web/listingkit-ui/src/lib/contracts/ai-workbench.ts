import { z } from "zod";
import { taskMetrics } from "./cockpit-metrics";
import {sameContentSelection,contentSelectionSchema} from "./content-selection";
import {collectionVersion} from "./product-collection";
import { isAcquisitionUUID } from "./product-acquisition";
import { productKnowledgeSchema } from "./product-knowledge";
import {executionModeSchema,modeRevisionSchema,modeSnapshotRefSchema,titlePolicyAdoptionSchema,contentPolicyAdoptionSchema} from "./execution-mode";

const id = z.string().refine(isAcquisitionUUID);
const aiTaskSubject = z.strictObject({kind:z.enum(["own_product","acquisition"]),id,catalogVersion:collectionVersion,publicationId:z.string().min(1).max(128).regex(/^[A-Za-z0-9][A-Za-z0-9._:-]*$/)});
export const aiTaskSelection = z.strictObject({subject:aiTaskSubject,capability:z.literal("product.content.localize"),parameters:contentSelectionSchema});
export type AITaskSubject=z.infer<typeof aiTaskSubject>;
const timestamp = z.string().datetime({ offset: true });
const scope = z.object({ OrganizationID: z.string(), ActorID: z.string() });
const scopeId = z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const workScopeSelection = z.discriminatedUnion("kind", [
  z.strictObject({kind:z.literal("general")}),
  z.strictObject({kind:z.literal("project"),projectId:scopeId}),
  z.strictObject({kind:z.literal("store"),storeId:scopeId}),
]);
export type WorkScopeSelection = z.infer<typeof workScopeSelection>;
const fixedScope = z.strictObject({version:z.literal("workbench-scope-v1"),kind:z.enum(["general","project","store"]),projectId:scopeId.optional(),storeId:scopeId.optional()});
const scopeView = z.object({scope:fixedScope,projectName:z.string().optional(),storeName:z.string().optional()});
const scopeOption = scopeView.extend({confirmation:z.string().regex(/^[a-f0-9]{64}$/)});
export type WorkScopeOption = z.infer<typeof scopeOption>;
const conversation = z.object({
  businessScope:fixedScope.optional(),
  ID: id, Scope: scope, Title: z.string(), Favorite: z.boolean(), Archived: z.boolean(),
  MetadataRevision: z.number().int().positive(), NextSequence: z.number().int().positive(),
  CreatedAt: timestamp, UpdatedAt: timestamp,
});
const message = z.object({
  ID: id, ConversationID: id, Sequence: z.number().int().positive(),
  Author: z.enum(["USER", "ASSISTANT"]), Content: z.string(), CreatedAt: timestamp,
});
const proposal = z.object({
 businessScope:scopeView.optional(),
 kind:z.enum(["product.title.optimize","product.content.localize"]).optional(),taskSelection:aiTaskSelection.optional(),executionProfileReady:z.boolean().optional(),maximumSteps:z.number().int().positive().optional(),maximumModelCalls:z.number().int().positive().optional(),maximumRuntimeSeconds:z.number().int().positive().optional(),
  executionMode:executionModeSchema.optional(),modeRevision:modeRevisionSchema.optional(),modeUnavailableReason:z.string().optional(),
  id, confirmedTaskId: id.optional(), digest: z.string().length(64), sourceSequence: z.number().int().positive(),
  goalSummary: z.string(), operationId: z.string().optional(), productKey: z.string().optional(),
  targetPlatform: z.enum(["shein", "temu", "amazon","product"]).optional(),
  templateId: z.string().optional(), templateRevision: z.string().optional(), knowledgeBaseId: z.string().optional(),
  providerId: z.string().optional(), modelId: z.string().optional(), maximumTokens: z.number().int().optional(),
  maximumCostMicros: z.number().int().optional(), currency: z.string().optional(),
  humanReviewRequired: z.literal(true), detailsAvailable: z.boolean(), titleProfileReady: z.boolean(), executionAuthorized: z.boolean(),
});
const task = z.object({
 businessScope:scopeView.optional(),scopeUnavailable:z.boolean().optional(),
 kind:z.enum(["product.title.optimize","product.content.localize"]).optional(),taskSelection:aiTaskSelection.optional(),
  executionMode:executionModeSchema.optional(),modeRevision:modeRevisionSchema.optional(),modeSnapshotRef:modeSnapshotRefSchema.optional(),policyAdoption:z.union([titlePolicyAdoptionSchema,contentPolicyAdoptionSchema]).optional(),
  id, conversationId: z.union([id,z.literal("")]), proposalId: z.union([id,z.literal("")]), scheduleId:id.optional(), scheduledOccurrenceId:id.optional(), title: z.string(), goalSummary: z.string(), createdAt: timestamp,
  projectionAvailable: z.boolean(), state: z.enum(["RUNNING", "WAITING_CONFIRMATION", "COMPLETED", "ERROR", "PAUSED"]).optional(),
  reason: z.string().optional(), canStart: z.boolean(), canReconcile: z.boolean(), canResume: z.boolean(), canReview: z.boolean(),
  productDetailsAvailable: z.boolean(), operationId: z.string().optional(), productKey: z.string().optional(),
  targetPlatform: z.string().optional(), agentRunId: z.string().optional(), agentPhase: z.string().optional(),
  agentRevision: z.string().optional(), providerId: z.string().optional(), modelId: z.string().optional(),
  tokens: z.number().int().optional(), estimatedCostMicros: z.number().int().optional(), currency: z.string().optional(),
  usageStatus: z.string().optional(), reviewId: z.string().optional(), reviewState: z.string().optional(),
  knowledge: productKnowledgeSchema.optional(),
}).superRefine((value,context)=>{
 if(value.scheduleId||value.scheduledOccurrenceId){if(!value.scheduleId||!value.scheduledOccurrenceId||value.conversationId!==""||value.proposalId!=="")context.addIssue({code:"custom",message:"Scheduled source must be exclusive"});}
 else if(!isAcquisitionUUID(value.conversationId)||!isAcquisitionUUID(value.proposalId))context.addIssue({code:"custom",message:"Chat source required"});
 const proof=value.policyAdoption;
 if(!proof)return;
 const content=proof.actionVersion==="product.content.policy-adopt-v1";
 if(content!==(value.kind==="product.content.localize"))context.addIssue({code:"custom",message:"Policy proof must match exact task kind"});
 if(content&&value.taskSelection){const {subject,parameters}=value.taskSelection,binding=proof.execution.binding;
 if(subject.kind!==binding.ContextKind||subject.id!==binding.ContextID||subject.catalogVersion!==binding.CatalogVersion||subject.publicationId!==binding.PublicationID||!sameContentSelection(parameters,proof.execution.ContentSelection))context.addIssue({code:"custom",message:"Policy proof must match frozen content intent"});}
});
const scopedCreateBody = z.strictObject({ favorite: z.boolean().optional(), scopeSelection:workScopeSelection,scopeConfirmation:z.string().regex(/^[a-f0-9]{64}$/) });
export const aiCreateBody = z.union([scopedCreateBody,z.strictObject({favorite:z.boolean().optional()})]);
const aiTitleMessageBody = z.strictObject({
  content: z.string().min(1).refine(v => new TextEncoder().encode(v).length <= 8192), operationId: id,
  targetPlatform: z.enum(["shein", "temu", "amazon"]), templateId: id.optional(),
  templateRevision: z.string().regex(/^[1-9][0-9]*$/).optional(), knowledgeBaseId: id.optional(),
});
export const aiMessageBody=z.union([aiTitleMessageBody,z.strictObject({content:aiTitleMessageBody.shape.content,taskSelection:aiTaskSelection,templateId:id.optional(),templateRevision:aiTitleMessageBody.shape.templateRevision,knowledgeBaseId:id.optional()})]).refine(value=>Boolean(value.templateId)===Boolean(value.templateRevision));
export const aiMetadataBody = z.strictObject({ title: z.string().max(256).optional(), favorite: z.boolean().optional(), archived: z.boolean().optional() });
export const aiResumeBody = z.strictObject({ revision: z.string().regex(/^[1-9][0-9]*$/),
  feedback: z.string().refine(v => new TextEncoder().encode(v).byteLength <= 8192) });

export type AIConversation = z.infer<typeof conversation>;
export type AIProposal = z.infer<typeof proposal>;

const positiveDecimal=z.string().regex(/^[1-9][0-9]{0,18}$/);
const decimal=z.string().regex(/^(0|[1-9][0-9]{0,18})$/);
const aiScheduleRule=z.strictObject({frequency:z.enum(["once","daily","weekly"]),timeZone:z.string().min(1).max(128),atUtc:timestamp.optional(),hour:z.number().int().min(0).max(23),minute:z.number().int().min(0).max(59),weekday:z.number().int().min(0).max(6),untilUtc:timestamp.optional()});
export const aiScheduleInput=z.strictObject({proposalId:id.optional(),scheduleId:id.optional(),selection:aiTaskSelection.optional(),goalSummary:z.string().min(1).max(8192).optional(),rule:aiScheduleRule,maxOccurrences:positiveDecimal,confirmation:z.string().regex(/^[a-f0-9]{64}$/).optional()});
export const aiScheduleControl=z.strictObject({expectedRevision:positiveDecimal});
export const aiScheduleEdit=aiScheduleControl.extend({rule:aiScheduleRule,confirmation:z.string().regex(/^[a-f0-9]{64}$/)});
const scheduleBudget={maxOccurrences:positiveDecimal,unit:z.literal("ai-micropoint-v1"),perRunUpperBoundMicros:positiveDecimal,totalUpperBoundMicros:positiveDecimal};
const schedulePlan=z.object({id,revision:positiveDecimal,state:z.enum(["PENDING_REGISTRATION","ACTIVE","PAUSED","CLOSED","REGISTRATION_UNKNOWN"]),registrationState:z.enum(["PENDING","MATCHED","UNKNOWN"]),goalSummary:z.string().optional(),selection:aiTaskSelection.optional(),detailsAvailable:z.boolean(),rule:aiScheduleRule,...scheduleBudget,consumedOccurrences:decimal,nextAt:timestamp.optional(),activeOccurrenceId:id.optional(),createdAt:timestamp});
export type AIScheduledPlan=z.infer<typeof schedulePlan>;
export type AIScheduleInput=z.infer<typeof aiScheduleInput>;
const scheduleQuote=z.object({confirmation:z.string().regex(/^[a-f0-9]{64}$/),selection:aiTaskSelection,goalSummary:z.string(),rule:aiScheduleRule,...scheduleBudget,nextAt:timestamp,executionMode:z.enum(["default","read_only"])});
export type AIScheduleQuote=z.infer<typeof scheduleQuote>;
const scheduledSave=z.object({plan:schedulePlan,planSaved:z.literal(true)});

export type AIWorkbenchRoute = "conversation-list" | "conversation-create" | "conversation-read" | "conversation-metadata" |
  "scope-options" | "task-scope-options" | "subject-read" | "message" | "confirm" | "task-list" | "task-cockpit-metrics" | "task-read" | "task-start" | "task-resume" | "task-review" |
  "schedule-availability"|"schedule-list"|"schedule-read"|"schedule-history"|"schedule-quote"|"schedule-create"|"schedule-edit"|"schedule-pause"|"schedule-resume"|"schedule-close";

export function aiWorkbenchPath(method: string, path: string[]): AIWorkbenchRoute | null {
  if(path[0]==="schedules"){
    if(path.length===1)return method==="GET"?"schedule-list":method==="POST"?"schedule-create":null;
    if(path.length===2&&path[1]==="availability"&&method==="GET")return "schedule-availability";
    if(path.length===2&&path[1]==="quote"&&method==="POST")return "schedule-quote";
    if(!isAcquisitionUUID(path[1]??""))return null;
    if(path.length===2)return method==="GET"?"schedule-read":method==="PATCH"?"schedule-edit":null;
    if(path.length===3&&path[2]==="occurrences"&&method==="GET")return "schedule-history";
    if(path.length===3&&method==="POST"&&["pause","resume","close"].includes(path[2]??""))return `schedule-${path[2]}` as AIWorkbenchRoute;
    return null;
  }
  if(method==="GET"&&path.length===4&&path[0]==="chat"&&path[1]==="subjects"&&["own_product","acquisition"].includes(path[2]??"")&&isAcquisitionUUID(path[3]??""))return "subject-read";
  if (path[0] === "chat" && path[1] === "conversations") {
    if (path.length === 2) return method === "GET" ? "conversation-list" : method === "POST" ? "conversation-create" : null;
    if (path.length === 3 && path[2] === "scopes" && method === "GET") return "scope-options";
    if (!isAcquisitionUUID(path[2] ?? "")) return null;
    if (path.length === 3) return method === "GET" ? "conversation-read" : method === "PATCH" ? "conversation-metadata" : null;
    if (path.length === 4 && path[3] === "messages" && method === "POST") return "message";
    if (path.length === 6 && path[3] === "proposals" && isAcquisitionUUID(path[4] ?? "") && path[5] === "confirm" && method === "POST") return "confirm";
  }
  if (path[0] === "tasks") {
    if (path.length === 2 && path[1] === "cockpit-metrics" && method === "GET") return "task-cockpit-metrics";
    if (path.length === 2 && path[1] === "scopes" && method === "GET") return "task-scope-options";
    if (path.length === 1 && method === "GET") return "task-list";
    if (!isAcquisitionUUID(path[1] ?? "")) return null;
    if (path.length === 2 && method === "GET") return "task-read";
    if (path.length === 3 && method === "POST") {
      if (path[2] === "start") return "task-start";
      if (path[2] === "resume") return "task-resume";
      if (path[2] === "review") return "task-review";
    }
  }
  return null;
}

export const aiResponseSchemas = {
 "schedule-availability":z.object({enabled:z.boolean(),ready:z.boolean()}),
 "schedule-list":z.object({plans:z.array(schedulePlan),next:z.string()}),
 "schedule-read":z.object({plan:schedulePlan}),
 "schedule-history":z.object({occurrences:z.array(z.object({id,scheduledAt:timestamp,planRevision:positiveDecimal,state:z.enum(["CLAIMED","DISPATCHED","SKIPPED","BLOCKED","UNKNOWN","RECONCILED"]),reason:z.string().optional(),businessTaskId:id.optional()})),next:z.string()}),
 "schedule-quote":scheduleQuote,"schedule-create":scheduledSave,"schedule-edit":scheduledSave,"schedule-pause":scheduledSave,"schedule-resume":scheduledSave,"schedule-close":scheduledSave,
 "task-cockpit-metrics": taskMetrics,
 "scope-options":z.strictObject({options:z.array(scopeOption).max(20),next:z.string(),incomplete:z.boolean()}),
 "task-scope-options":z.strictObject({options:z.array(scopeOption).max(20),next:z.string(),incomplete:z.boolean()}),
 "subject-read":z.strictObject({subject:aiTaskSubject,productKey:z.string().min(1).max(128)}),
    "conversation-list": z.object({ conversations: z.array(conversation), next: z.string() }),
    "conversation-create": z.object({ conversation, replay: z.boolean() }),
    "conversation-read": z.object({ conversation, messages: z.array(message), proposals: z.array(proposal), before: z.string(),scopeUnavailable:z.boolean().optional() }),
    "conversation-metadata": z.object({ conversation }),
    message: z.object({ state: z.enum(["READY_TO_DISPATCH", "COMPLETE", "FAILED_BEFORE_DISPATCH", "PLANNER_INVALID_OUTPUT", "PLANNER_UNKNOWN"]),
      userMessageId: id.optional(), sourceSequence: z.number().int().positive().optional(), assistantMessageId: z.string().optional(), proposalId: z.string().optional(), proposal: proposal.optional(),scopeUnavailable:z.boolean().optional() }),
    confirm: z.object({ task, replay: z.boolean() }),
    "task-list": z.object({ tasks: z.array(task), next: z.string(), incomplete:z.boolean().optional() }),
    "task-read": z.object({ task }),
    "task-start": z.object({ task, replay: z.boolean() }),
    "task-resume": z.object({ task, replay: z.boolean() }),
    "task-review": z.object({ task, replay: z.boolean() }),
} as const;
export function parseAIWorkbenchResponse(route: AIWorkbenchRoute, status: number, value: unknown) {
  if (!(status === 200 || route === "message" && status === 202)) return null;
  const checked = aiResponseSchemas[route].safeParse(value);
  return checked.success ? checked.data : null;
}
