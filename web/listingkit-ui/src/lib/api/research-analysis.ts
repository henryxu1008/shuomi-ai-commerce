import {podRequest} from "./pod";
import type {ConfigurationScope} from "./agent-configuration";
import {collectionID} from "../contracts/product-collection";
import {researchAnalysisInput,researchAnalysisIntent,researchInspectionSchema,researchPreparedSchema,researchRunSchema,researchRecentSchema,type ResearchAnalysisIntent} from "../contracts/research-analysis";
const path=(project:string)=>collectionID.parse(project)+"/analysis/";
export const inspectResearch=(scope:ConfigurationScope,project:string,input:unknown,signal?:AbortSignal)=>podRequest(scope,path(project)+"inspect",researchInspectionSchema,signal,{key:crypto.randomUUID(),body:researchAnalysisInput.parse(input)},"research/projects");
export const readResearchPreparation=(scope:ConfigurationScope,project:string,key:string,signal?:AbortSignal)=>podRequest(scope,path(project)+"preparations/"+collectionID.parse(key),researchPreparedSchema,signal,undefined,"research/projects");
export const readResearchRun=(scope:ConfigurationScope,project:string,key:string,signal?:AbortSignal)=>podRequest(scope,path(project)+"runs/"+collectionID.parse(key),researchRunSchema,signal,undefined,"research/projects");
export const recentResearchRuns=(scope:ConfigurationScope,project:string,after:string,signal?:AbortSignal)=>podRequest(scope,path(project)+"runs"+(after?"?after="+collectionID.parse(after):""),researchRecentSchema,signal,undefined,"research/projects");
export function prepareResearch(scope:ConfigurationScope,intent:Extract<ResearchAnalysisIntent,{kind:"prepare"}>,signal?:AbortSignal){const i=researchAnalysisIntent.parse(intent);return podRequest(scope,path(i.projectId)+"prepare",researchPreparedSchema,signal,i,"research/projects");}
export function executeResearch(scope:ConfigurationScope,intent:Exclude<ResearchAnalysisIntent,{kind:"prepare"}>,signal?:AbortSignal){const i=researchAnalysisIntent.parse(intent);if(i.kind==="prepare")throw new Error("Invalid execution");return podRequest(scope,path(i.projectId)+"runs"+(i.kind==="resume"?"/"+i.key+"/resume":""),researchRunSchema,signal,i,"research/projects");}
