import {expect,it} from "vitest";
import {completionContentSchema} from "./completion-content";

const image="https://images.example.org/new.png";
const rich={title:"candidate",description:`<p><img src="${image}"></p>`,descriptionFormat:"html-v1" as const};
it("accepts referenced candidate images and leaves original content encoding unchanged",()=>{
 expect(completionContentSchema.parse({...rich,descriptionImages:[image]}).descriptionImages).toEqual([image]);
 const plain={title:"plain",description:"<b>literal</b>"};
 expect(JSON.stringify(completionContentSchema.parse(plain))).toBe(JSON.stringify(plain));
 expect(JSON.stringify(completionContentSchema.parse({...plain,descriptionImages:[]}))).toBe(JSON.stringify(plain));
});
it("rejects dangling, duplicate, dangerous, plain-text or excessive image declarations",()=>{
 for(const value of [{...rich,descriptionImages:[image,image]},{...rich,descriptionImages:["https://elsewhere.example.org/x.png"]},{...rich,descriptionImages:["javascript:alert(1)"]},{...rich,descriptionFormat:undefined,descriptionImages:[image]},{...rich,descriptionImages:Array.from({length:41},(_,i)=>`https://images.example.org/${i}.png`)}])expect(completionContentSchema.safeParse(value).success).toBe(false);
});
