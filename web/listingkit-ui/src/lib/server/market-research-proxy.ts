import {proxyPOD} from "./pod-proxy";
import {proxyProjectCenter} from "./project-center-proxy";
import {researchEndpoint} from "@/lib/contracts/market-research";
import {proxySheinUSTrial,sheinTrialPath} from "./shein-us-trial-proxy";

export function proxyMarketResearch(request:Request, token:string, userId:string) {
  const path = new URL(request.url).pathname;
  if (sheinTrialPath(path)) return proxySheinUSTrial(request, token, userId);
  if (path.includes("/analysis/")) return proxyPOD(request, token, userId);
  return proxyProjectCenter(request, token, userId, {endpoint:researchEndpoint,base:"research/projects",inputBytes:65536,outputBytes:2*1024*1024});
}
