import type {ReportSummary} from "@/lib/contracts/report-center";

type Kind = "SHEIN_RECORD" | "SHEIN_TARGET_RECORD";
type Stream = {kind:Kind; cursor:string | null; buffer:ReportSummary[]};
export type ReportMergeState = {streams:Stream[]};
type FetchPage = (kind:Kind,cursor:string)=>Promise<{items:ReportSummary[];nextCursor:string}>;
export function initialReportMerge():ReportMergeState {return {streams:[{kind:"SHEIN_RECORD",cursor:"",buffer:[]},{kind:"SHEIN_TARGET_RECORD",cursor:"",buffer:[]}]};}
function compare(a:ReportSummary,b:ReportSummary) {
 const [asec,af=""] = a.capturedAt.slice(0,-1).split("."), [bsec,bf=""] = b.capturedAt.slice(0,-1).split(".");
 if(asec!==bsec)return asec>bsec?-1:1;
 const av=af.padEnd(9,"0"),bv=bf.padEnd(9,"0");if(av!==bv)return av>bv?-1:1;
 return a.id===b.id?0:a.id>b.id?-1:1;
}
export async function mergeReportPage(previous:ReportMergeState,fetchPage:FetchPage,limit=20):Promise<{items:ReportSummary[];next:ReportMergeState | undefined}> {
 const state={streams:previous.streams.map(s=>({...s,buffer:[...s.buffer]}))},items:ReportSummary[]=[];
 while(items.length<limit){
  await Promise.all(state.streams.map(async stream=>{
   if(stream.buffer.length || stream.cursor===null)return;
   const cursor=stream.cursor,page=await fetchPage(stream.kind,cursor);
   if((page.nextCursor===cursor && cursor!=="") || (!page.items.length && page.nextCursor) || page.items.some(item=>item.ref.kind!==stream.kind))throw new Error("invalid report page");
   stream.buffer=page.items;stream.cursor=page.nextCursor||null;
  }));
  const available=state.streams.filter(stream=>stream.buffer.length);
  if(!available.length){if(state.streams.some(s=>s.cursor!==null))continue;break;}
  available.sort((a,b)=>compare(a.buffer[0],b.buffer[0]));
  items.push(available[0].buffer.shift()!);
 }
 return {items,next:state.streams.some(s=>s.buffer.length || s.cursor!==null)?state:undefined};
}
