import { expect, it } from "vitest";
import { agentStartRequestSchema } from "./product-agent";
import { catalogEntrySchema, configReceiptSchema, templateSchema, materialsTemplateInputSchema } from "./agent-configuration";
const template = {
  templateId: "8fc227bb-b572-4138-8e2a-5f1a0be98617",
  revision: "9223372036854775807",
};
it("retains the exact template revision and explicit no-Knowledge choice", () => {
  const parsed = agentStartRequestSchema.parse({
    expectedModeRevision: "0",
    targetPlatform: "shein",
    templateSelection: template,
  });
  expect(parsed).toEqual({
    expectedModeRevision: "0",
    targetPlatform: "shein",
    templateSelection: template,
  });
  expect(parsed.knowledgeSelection).toBeUndefined();
});

it("accepts the image agent catalog and receipt without treating its parameters as title configuration",()=>{
  const entry={agent:{agentId:"product.image.agent",activation:"NOT_ENABLED",revision:"",activationEpoch:"",defaultTemplate:null,updatedAt:"2026-10-09T00:00:00Z"},name:"商品图片智能体",description:"整套生成与人工选择",definitionVersion:"v1.0.0",parameterSchema:"image-config-v1",canConfigure:true,canUse:true,canReadRuns:false,capabilities:[]};
  expect(catalogEntrySchema.safeParse(entry).success).toBe(true);
  expect(catalogEntrySchema.safeParse({...entry,parameterSchema:"title-config-v1"}).success).toBe(false);
  expect(configReceiptSchema.safeParse({commandId:template.templateId,operation:"enable",agentId:"product.image.agent",revision:"1",noop:false,committedAt:"2026-10-09T00:00:00Z"}).success).toBe(true);
});
it("rejects null, partial and rounded template references", () => {
  for (const value of [
    null,
    {},
    { ...template, revision: 3 },
    { ...template, revision: "03" },
    { ...template, revision: "invalid" },
    { ...template, revision: "9223372036854775808" },
  ])
    expect(
      agentStartRequestSchema.safeParse({
        expectedModeRevision: "0",
        targetPlatform: "shein",
        templateSelection: value,
      }).success,
    ).toBe(false);
});

it("keeps platform materials configuration exact and separate from title/content",()=>{
 const entry={agent:{agentId:"listing.materials.agent",activation:"NOT_ENABLED",revision:"",activationEpoch:"",defaultTemplate:null,updatedAt:"2026-10-13T00:00:00Z"},name:"平台上架资料智能体",description:"本地资料",definitionVersion:"v1.0.0",parameterSchema:"listing-materials-config-v1",canConfigure:true,canUse:false,canReadRuns:false,capabilities:[{id:"listing.materials.inspect",support:"REQUIRED",readiness:"UNAVAILABLE",reason:"需要正常供应链",observedAt:"2026-10-13T00:00:00Z"}]};
 expect(catalogEntrySchema.safeParse(entry).success).toBe(true);
 expect(catalogEntrySchema.safeParse({...entry,parameterSchema:"title-config-v1"}).success).toBe(false);
 const configured={templateId:template.templateId,agentId:"listing.materials.agent",lifecycle:"ACTIVE",revision:"1",version:"1",schemaVersion:"listing-materials-config-v1",name:"SHEIN 资料",targetPlatform:"shein",createdAt:"2026-10-13T00:00:00Z"};
 expect(templateSchema.parse(configured)).toEqual(configured);
 expect(materialsTemplateInputSchema.safeParse({name:configured.name,targetPlatform:"shein"}).success).toBe(true);
 for(const change of [{targetPlatform:"temu"},{defaultKnowledgeBaseId:template.templateId},{defaultKnowledgeBaseId:null},{content:{schema:"content-config-v1"}},{tools:["publish"]}]) {
  expect(materialsTemplateInputSchema.safeParse({name:configured.name,targetPlatform:"shein",...change}).success).toBe(false);
 }
});
