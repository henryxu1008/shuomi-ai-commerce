import {taskCenterSummarySchema} from "@/lib/contracts/task-center-summary";
import {readBoundedStrictJSON} from "./strict-json-response";
import type {AIScope} from "./ai-workbench";

export async function fetchTaskCenterSummary(scope:AIScope,query:string,signal:AbortSignal){
 const response=await fetch(`/api/workbench/task-center-summary?${query}`,{method:"GET",headers:{Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId},credentials:"same-origin",redirect:"error",cache:"no-store",signal});
 if(!response.ok)throw Error("summary unavailable");
 return taskCenterSummarySchema.parse(await readBoundedStrictJSON(response,16384,signal));
}
