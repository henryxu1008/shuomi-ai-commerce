import { expect,it } from "vitest";
import { freezeToolIntent,restoreToolIntent,forgetToolIntent,toolPendingSchema } from "./pending";

it("retains large file bytes through navigation and keeps a blocking marker after byte loss",async()=>{
 const scope="tool-pending-test:"+crypto.randomUUID(),key=crypto.randomUUID();
 const body=JSON.stringify({kind:"DATA",title:"需求",description:"规则",scenario:"商品管理",contactName:"测试联系人",contactMethod:"test",consent:true,files:[{name:"flow.txt",data:btoa("x".repeat(150000))}]});
 const intent={path:"requests",key,absent:true,body};
 const record=await freezeToolIntent(scope,intent);
 expect(record.body).toBeUndefined();expect(toolPendingSchema.safeParse(record).success).toBe(true);
 expect(restoreToolIntent(scope,record)).toEqual(intent);
 expect(restoreToolIntent(scope+"other",record)).toBeNull();
 await expect(freezeToolIntent(scope,{...intent,body:body.replace('"test"','"changed"')})).rejects.toThrow("Original command differs");
 forgetToolIntent(scope,key);
 expect(restoreToolIntent(scope,record)).toBeNull();
 expect(record.key).toBe(key);expect(toolPendingSchema.safeParse(record).success).toBe(true);
});
it("refuses a marker that changes a versioned progress command into a create",()=>{
 expect(toolPendingSchema.safeParse({path:"requests",key:crypto.randomUUID(),revision:"1",digest:"a".repeat(64)}).success).toBe(false);
});
