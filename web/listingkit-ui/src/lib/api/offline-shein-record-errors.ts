import {z} from "zod";
import {parseSheinRecordListFailure} from "./shein-records";

// Reuse the current Workbench envelope and its code/status contract. Feature
// failures retain their original bounded wire format and explicit statuses.
const featureFailure=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/)});
const statuses:Record<string,readonly number[]>={
 AUTHENTICATION_REQUIRED:[401],IDENTITY_CONTEXT_CHANGED:[409],ORGANIZATION_CONTEXT_CHANGED:[409],
 FORBIDDEN:[403],NOT_FOUND:[404],INVALID_REQUEST:[400,405],COMMAND_CONFLICT:[409],NOT_READY:[422],TOO_LARGE:[413],
 DEPENDENCY_UNAVAILABLE:[502,503,504],INVALID_UPSTREAM_RESPONSE:[502],OUTCOME_UNKNOWN:[502,503,504],
};
export function parseOfflineRecordFailure(value:unknown,status:number){
 const workbench=parseSheinRecordListFailure(value,status);
 if(workbench&&"code" in workbench)return workbench;
 const feature=featureFailure.safeParse(value);
 return feature.success&&statuses[feature.data.code]?.includes(status)?feature.data:null;
}
export const isDefiniteOfflineRejection=(code:string)=>[
 "NOT_READY","FORBIDDEN","NOT_FOUND","INVALID_REQUEST","COMMAND_CONFLICT","TOO_LARGE",
 "AUTHENTICATION_REQUIRED","IDENTITY_CONTEXT_CHANGED","ORGANIZATION_SELECTION_REQUIRED","ORGANIZATION_CONTEXT_CHANGED",
 "ORGANIZATION_ACCESS_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_SUSPENDED","PERMISSION_DENIED",
].includes(code);
