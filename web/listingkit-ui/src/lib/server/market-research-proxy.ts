import {proxyPOD} from "./pod-proxy";
import {proxyProjectCenter} from "./project-center-proxy";
import {researchEndpoint} from "@/lib/contracts/market-research";
export const proxyMarketResearch=(request:Request,token:string,userId:string)=>new URL(request.url).pathname.includes("/analysis/")?proxyPOD(request,token,userId):proxyProjectCenter(request,token,userId,{endpoint:researchEndpoint,base:"research/projects",inputBytes:65536,outputBytes:2*1024*1024});
