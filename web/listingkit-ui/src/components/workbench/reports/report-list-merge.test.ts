import {expect,it,vi} from "vitest";
import {mergeReportPage,initialReportMerge} from "./report-list-merge";
import type {ReportSummary} from "@/lib/contracts/report-center";
const item=(id:number,kind:"SHEIN_RECORD"|"SHEIN_TARGET_RECORD",capturedAt:string):ReportSummary=>({id:`00000000-0000-4000-8000-${String(id).padStart(12,"0")}`,ref:{kind,id:`11111111-1111-4111-8111-${String(id).padStart(12,"0")}`,version:`sha256:${"a".repeat(64)}`},title:`报告 ${id}`,productKey:`product-${id}`,storeId:"",capturedAt,favorite:false});
it("merges both product report sources across pages without dropping or duplicating records",async()=>{
 const a=[item(1,"SHEIN_RECORD","2026-11-14T10:00:00Z"),item(3,"SHEIN_RECORD","2026-11-14T08:00:00Z"),item(5,"SHEIN_RECORD","2026-11-14T06:00:00Z")];
 const b=[item(2,"SHEIN_TARGET_RECORD","2026-11-14T09:00:00Z"),item(4,"SHEIN_TARGET_RECORD","2026-11-14T07:00:00Z")];
 const fetch=vi.fn(async(kind:string,cursor:string)=>{const list=kind==="SHEIN_RECORD"?a:b;const offset=cursor?Number(cursor):0;return {items:list.slice(offset,offset+2),nextCursor:offset+2<list.length?String(offset+2):""};});
 let state:ReturnType<typeof initialReportMerge> | undefined=initialReportMerge();const all:ReportSummary[]=[];
 do {const page=await mergeReportPage(state,fetch,2);all.push(...page.items);state=page.next;} while(state);
 expect(all.map(x=>x.title)).toEqual(["报告 1","报告 2","报告 3","报告 4","报告 5"]);
 expect(new Set(all.map(x=>x.id)).size).toBe(5);
});
it("does not silently drop a failed source",async()=>{
 await expect(mergeReportPage(initialReportMerge(),async(kind)=>{if(kind==="SHEIN_RECORD")throw new Error("failed");return {items:[],nextCursor:""};},20)).rejects.toThrow("failed");
});

it("orders equal-second timestamps at nanosecond precision and breaks exact ties by ID",async()=>{
 const a=[item(1,"SHEIN_RECORD","2026-11-14T10:00:00.123456789Z"),item(3,"SHEIN_RECORD","2026-11-14T10:00:00.123456700Z")];
 const b=[item(2,"SHEIN_TARGET_RECORD","2026-11-14T10:00:00.123456789Z"),item(4,"SHEIN_TARGET_RECORD","2026-11-14T10:00:00.123456701Z")];
 const page=await mergeReportPage(initialReportMerge(),async kind=>({items:kind==="SHEIN_RECORD"?a:b,nextCursor:""}),20);
 expect(page.items.map(x=>x.title)).toEqual(["报告 2","报告 1","报告 4","报告 3"]);expect(page.next).toBeUndefined();
});
it("continues a single populated source and replays the same cursor without mutation",async()=>{
 const list=Array.from({length:7},(_,index)=>item(index+1,"SHEIN_RECORD",`2026-11-14T10:00:0${7-index}Z`));
 const fetch=async(kind:string,cursor:string)=>{const offset=cursor?Number(cursor):0;return kind==="SHEIN_RECORD"?{items:list.slice(offset,offset+2),nextCursor:offset+2<list.length?String(offset+2):""}:{items:[],nextCursor:""};};
 const first=await mergeReportPage(initialReportMerge(),fetch,3);expect(first.items.map(x=>x.title)).toEqual(["报告 1","报告 2","报告 3"]);
 const state=first.next!;const second=await mergeReportPage(state,fetch,3);const retry=await mergeReportPage(state,fetch,3);
 expect(second).toEqual(retry);expect(second.items.map(x=>x.title)).toEqual(["报告 4","报告 5","报告 6"]);
 const final=await mergeReportPage(second.next!,fetch,3);expect(final.items.map(x=>x.title)).toEqual(["报告 7"]);expect(final.next).toBeUndefined();
});
