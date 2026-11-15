import {expect,it} from "vitest";
import {skillPackageSchema,skillFilesSchema,skillRef,skillVersionSchema} from "./skill";
it("round-trips the Go-supported 256-byte path and 1024 Unicode-rune description",()=>{
 const path="references/"+"x".repeat(242)+".md",description="😀".repeat(1024);
 const value={name:"title-method",description,digest:"a".repeat(64),files:[{path:"SKILL.md",text:"instructions"},{path,text:"reference"}]};
 expect(new TextEncoder().encode(path).length).toBe(256);
 expect(skillPackageSchema.parse(JSON.parse(JSON.stringify(value)))).toEqual(value);
 expect(skillPackageSchema.safeParse({...value,description:description+"😀"}).success).toBe(false);
 expect(skillFilesSchema.safeParse([{path:path+"x",text:"x"}]).success).toBe(false);
 expect(skillFilesSchema.safeParse([{path:"references/"+"中".repeat(100)+".md",text:"x"}]).success).toBe(false);
});

it("preserves explicit saved script metadata and the separate reference policy",()=>{
 const execution={schemaVersion:"skill-script-v1",profile:"python-title-json-v1",entrypoint:"scripts/main.py",phase:"title-candidate-postprocess",inputContract:"title-candidate-v1",outputContract:"title-result-v1"};
 const pkg={name:"title-script",description:"Format candidate",target:"product-title",digest:"a".repeat(64),files:[{path:"SKILL.md",text:"method"},{path:"execution.json",text:JSON.stringify(execution)},{path:"scripts/main.py",text:"pass"}],execution};
 expect(skillPackageSchema.parse(pkg)).toEqual(pkg);
 const at="2026-11-15T00:00:00Z";
 const version=skillVersionSchema.parse({skill:{id:"d68f289c-2e45-4d0a-9a13-d53db4425739",owner:{kind:"USER",id:"actor"},creator:"actor",headVersion:1,revision:1,createdAt:at,updatedAt:at},version:1,package:pkg,state:"ENABLED",visibility:"PRIVATE",allowCopy:true,allowExport:true,createdAt:at});
 expect(skillRef(version).policyVersion).toBe("skill-script-method-v1");
 expect(skillPackageSchema.safeParse({...pkg,execution:{...execution,profile:"shell"}}).success).toBe(false);
});
