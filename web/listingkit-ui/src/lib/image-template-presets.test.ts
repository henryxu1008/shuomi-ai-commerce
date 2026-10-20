import {expect,it} from "vitest";
import {imageTemplateInputSchema,imageTemplateSchema} from "./contracts/image-set-configuration";
import {genericImagePresets,matchesGenericImagePreset} from "./image-template-presets";

it.each(genericImagePresets)("keeps $name as a valid single-image enterprise template input",preset=>{
 expect(imageTemplateInputSchema.safeParse({name:preset.name,targetPlatform:"product",image:preset.image}).success).toBe(true);
 expect(preset.image.carousel.length+preset.image.detail.length).toBe(1);
});
it("matches the entire configuration rather than a copied display name",()=>{
 const preset=genericImagePresets[0];
 const template=imageTemplateSchema.parse({templateId:"33333333-3333-4333-8333-333333333333",agentId:"product.image.agent",lifecycle:"ACTIVE",revision:"4",version:"3",schemaVersion:"image-config-v1",name:"My renamed template",targetPlatform:"product",createdAt:"2026-10-19T00:00:00Z",image:preset.image});
 expect(matchesGenericImagePreset(template,preset)).toBe(true);
 expect(matchesGenericImagePreset({...template,image:{...template.image,background:"red"},name:preset.name},preset)).toBe(false);
 expect(matchesGenericImagePreset({...template,lifecycle:"ARCHIVED"},preset)).toBe(false);
 expect(matchesGenericImagePreset({...template,targetPlatform:"shein"},preset)).toBe(false);
});
