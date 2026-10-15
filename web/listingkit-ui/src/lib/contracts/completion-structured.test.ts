import {expect,it} from "vitest";
import {completionInputSchema,completionRecordSchema} from "./supply-chain";

const id="11111111-1111-4111-8111-111111111111";
const structured={attributes:[{name:"材质",value:""}],variants:[{sourceIndex:0,price:{currency:"CNY",amount:0},stock:0},{sourceIndex:1,price:null,stock:null}]};
const input={sourceId:id,expectedRevision:1,content:{title:"候选",description:"",structured}};
it("accepts structured zero/null and explicit empty attributes without inventing old fields",()=>{
 expect(completionInputSchema.parse(input)).toEqual(input);
 expect(completionInputSchema.parse({...input,content:{title:"old",description:""}}).content).not.toHaveProperty("structured",expect.anything());
 expect(completionInputSchema.safeParse({...input,content:{...input.content,structured:{attributes:[],variants:[]}}}).success).toBe(true);
});
it("rejects ambiguous rows, invalid amounts and the whole candidate budget",()=>{
 for(const candidate of [
  {...structured,attributes:[{name:"材质",value:"x"},{name:" 材质 ",value:"y"}]},
  {...structured,attributes:null},
  {...structured,variants:[structured.variants[1],structured.variants[0]]},
  {...structured,variants:[{sourceIndex:0,price:{currency:"cny",amount:0},stock:0}]},
  {...structured,variants:[{sourceIndex:0,price:{currency:"CNY",amount:-1},stock:0}]},
  {...structured,variants:[{sourceIndex:0,price:null,stock:0.5}]},
  {...structured,variants:[{sourceIndex:0,price:null,stock:Number.MAX_SAFE_INTEGER+1}]},
  {...structured,attributes:Array.from({length:10},(_,i)=>({name:`属性${i}`,value:"x".repeat(8192)}))},
  {...structured,attributes:[{name:"a",value:"<".repeat(8192)},{name:"b",value:"<".repeat(8192)}]},
 ])expect(completionInputSchema.safeParse({...input,content:{...input.content,structured:candidate}}).success).toBe(false);
});
it("keeps old record and command wire shapes readable",()=>{
 const record={id,draftId:id,revision:1,source:{id,preparationId:id,collectionItemId:id,collectionRevision:1,source:{kind:"own",productKey:"product",publicationId:id,version:"1"}},content:{title:"old",description:""},status:"pending_completion",createdAt:"2026-10-15T00:00:00Z"};
 expect(completionRecordSchema.parse(record)).toEqual(record);
 expect(JSON.stringify(completionInputSchema.parse({sourceId:id,expectedRevision:0,content:record.content}))).toBe(JSON.stringify({sourceId:id,expectedRevision:0,content:record.content}));
});
