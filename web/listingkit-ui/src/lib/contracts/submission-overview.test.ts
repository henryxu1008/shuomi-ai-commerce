import { expect, it } from "vitest";
import { submissionOverviewMetrics } from "./cockpit-metrics";
import { supplyPath, supplyResponseSchema } from "./supply-chain";
const capturedAt="2026-11-08T00:00:00Z";
const rows=[
 {platform:"walmart",status:"complete",stores:2,counts:{confirmed:3,rejected:1,unknown:2}},
 {platform:"tiktokshop",status:"complete",stores:1,counts:{confirmed:1,rejected:2,unknown:1}},
 {platform:"shein",status:"complete",stores:1,counts:{confirmed:2,unknown:3}},
 {platform:"temu",status:"complete",stores:1,counts:{confirmed:1,unknown:1}},
];
const complete={status:"complete",capturedAt,rows,excludedStores:0,unknownTotal:7};
it("admits the read-only aggregate and checks unknown totals without inventing SHEIN/Temu failure counts",()=>{
 expect(submissionOverviewMetrics.safeParse(complete).success).toBe(true);
 expect(supplyPath("GET", ["supply-preparations","submission-overview-metrics"])).toBe("submission-overview-metrics");
 expect(supplyResponseSchema("submission-overview-metrics")).toBe(submissionOverviewMetrics);
 expect(submissionOverviewMetrics.safeParse({...complete,unknownTotal:8}).success).toBe(false);
 expect(submissionOverviewMetrics.safeParse({...complete,rows:rows.map(row=>row.platform==="shein"?{...row,counts:{...row.counts,rejected:0}}:row)}).success).toBe(false);
});
it("does not admit a numeric total, stale row numbers, or fake zeros when coverage is incomplete",()=>{
 const incomplete={status:"incomplete",capturedAt,excludedStores:0,rows:rows.map(row=>row.platform==="temu"?{platform:"temu",status:"unavailable",stores:1}:row)};
 expect(submissionOverviewMetrics.safeParse(incomplete).success).toBe(true);
 expect(submissionOverviewMetrics.safeParse({...incomplete,unknownTotal:7}).success).toBe(false);
 expect(submissionOverviewMetrics.safeParse({...incomplete,rows:rows.map(row=>row.platform==="temu"?{...row,status:"limited"}:row)}).success).toBe(false);
 expect(submissionOverviewMetrics.safeParse({...complete,excludedStores:1}).success).toBe(false);
});
