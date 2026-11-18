import {proxyPOD} from "./pod-proxy";
import {proxyProjectCenter} from "./project-center-proxy";
import {researchEndpoint} from "@/lib/contracts/market-research";
import {proxySheinUSTrial,sheinTrialPath} from "./shein-us-trial-proxy";
import {amazonFirstPageEndpoint,amazonFirstPagePath} from "./amazon-first-page-proxy";

export function proxyMarketResearch(request:Request, token:string, userId:string) {
  const path = new URL(request.url).pathname;
  if (sheinTrialPath(path)) return proxySheinUSTrial(request, token, userId);
  if (amazonFirstPagePath(path)) {
    return proxyProjectCenter(request, token, userId, {
      endpoint: amazonFirstPageEndpoint, base: "research/projects", inputBytes: 0, outputBytes: 4096,
      timeoutMs: request.method === "POST" ? 200_000 : 12_000,
    });
  }
  if (path.includes("/analysis/")) return proxyPOD(request, token, userId);
  return proxyProjectCenter(request, token, userId, {endpoint:researchEndpoint,base:"research/projects",inputBytes:65536,outputBytes:2*1024*1024});
}
