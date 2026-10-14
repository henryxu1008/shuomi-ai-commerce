import {podRequest} from "./pod";
import {artworkRunSchema,artworkIntentSchema,type ArtworkIntent} from "../contracts/pod-artwork";
import {knowledgeId} from "./knowledge";
import type {ConfigurationScope} from "./agent-configuration";
export const readArtworkRun=(scope:ConfigurationScope,id:string,signal?:AbortSignal)=>podRequest(scope,"runs/"+knowledgeId.parse(id),artworkRunSchema,signal,undefined,"pod-artwork");
export const resolveArtworkPrepare=(scope:ConfigurationScope,context:string,key:string,signal?:AbortSignal)=>podRequest(scope,"by-key/"+knowledgeId.parse(context)+"/"+knowledgeId.parse(key),artworkRunSchema,signal,undefined,"pod-artwork");
export function writeArtwork(scope:ConfigurationScope,input:ArtworkIntent,signal?:AbortSignal){const i=artworkIntentSchema.parse(input);return podRequest(scope,i.kind==="prepare"?"prepare":"runs/"+i.runId+"/"+i.kind,artworkRunSchema,signal,i,"pod-artwork")}
