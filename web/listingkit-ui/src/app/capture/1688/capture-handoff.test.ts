import { expect,it } from "vitest";
import { parseCaptureEntry } from "./capture-handoff";
const key="22222222-2222-4222-8222-22222222222b",handoffId="11111111-1111-4111-8111-11111111111a",extensionId="a".repeat(32);
const url=`http://localhost/capture/1688#extensionId=${extensionId}&handoffId=${handoffId}&idempotencyKey=${key}`;
it("admits only the exact platform's handoff or original-key recovery envelope",()=>{
 expect(parseCaptureEntry(url)).toEqual({kind:"handoff",extensionId,handoffId,key});
 expect(parseCaptureEntry(url,"amazon")).toBeNull();
 expect(parseCaptureEntry(url.replace("/1688#","/amazon#"),"amazon")).toEqual({kind:"handoff",extensionId,handoffId,key});
 expect(parseCaptureEntry(`http://localhost/capture/1688#operationKey=${key}`)).toEqual({kind:"recovery",key});
});
it.each([`${url}&orgId=org`,`${url}&handoffId=${handoffId}`,url.replace("/1688#","/1688/#"),url.replace("/1688#","/1688?x=1#"),url.replace(extensionId,"bad"),`${url}&operationKey=${key}`,`http://localhost/capture/1688#operationKey=${key}&payload=x`])("rejects a non-contract envelope %#",input=>expect(parseCaptureEntry(input)).toBeNull());
