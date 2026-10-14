import {expect,it} from "vitest";
import {skillPackageSchema,skillFilesSchema} from "./skill";
it("round-trips the Go-supported 256-byte path and 1024 Unicode-rune description",()=>{
 const path="references/"+"x".repeat(242)+".md",description="😀".repeat(1024);
 const value={name:"title-method",description,digest:"a".repeat(64),files:[{path:"SKILL.md",text:"instructions"},{path,text:"reference"}]};
 expect(new TextEncoder().encode(path).length).toBe(256);
 expect(skillPackageSchema.parse(JSON.parse(JSON.stringify(value)))).toEqual(value);
 expect(skillPackageSchema.safeParse({...value,description:description+"😀"}).success).toBe(false);
 expect(skillFilesSchema.safeParse([{path:path+"x",text:"x"}]).success).toBe(false);
 expect(skillFilesSchema.safeParse([{path:"references/"+"中".repeat(100)+".md",text:"x"}]).success).toBe(false);
});
