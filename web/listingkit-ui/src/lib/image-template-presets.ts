import type {SkillSelection} from "./contracts/skill";
import {imageSetTemplateSchema,type ImageAgentTemplate,type ImageSetTemplate} from "./contracts/image-set-configuration";

// Initial values for the existing enterprise template editor, never executable
// configuration or a substitute for a persisted exact template reference.
export const genericImagePresets = [
  {skillId:"2d5c0732-3243-5da5-851c-09ed3bcf017a",name:"通用商品主图",description:"主图 1 张 · 白色背景 · 中文。清晰展示原图中的同一件商品，不加入文字或额外配件。",image:{schema:"image-config-v1",mode:"standard",shareOriginals:true,background:"纯白背景，柔和自然阴影",language:"zh",carousel:[{id:"generic-product-main",purpose:"product_identity",brief:"以所选原图中的同一件商品为唯一主体，清晰展示完整外观。保留原图可见的形状、颜色、结构、材质和比例。主体居中，留出适当边距，不增加配件、包装、文字、标签或标识。"}],detail:[]}},
  {skillId:"18d622d6-a814-5f35-8f3d-a7709de509df",name:"通用使用场景",description:"主图 1 张 · 自然使用场景 · 中文。按真实用途安排环境，保持商品外观、颜色和比例。",image:{schema:"image-config-v1",mode:"standard",shareOriginals:true,background:"符合商品真实用途的简洁自然环境，背景轻度虚化",language:"zh",carousel:[{id:"generic-product-scene",purpose:"usage_scene",brief:"将所选原图中的同一件商品置于符合已确认用途的自然场景，商品仍是画面焦点。保持外观、颜色、结构和比例；不增加商品功能或随附配件，不加入文字、标签或标识。"}],detail:[]}},
  {skillId:"ef366850-7ffa-5b1d-b360-9a8db94f8c40",name:"通用卖点详情",description:"详情图 1 张 · 简洁留白 · 中文。仅呈现已确认事实；没有依据的卖点与规格不添加。",image:{schema:"image-config-v1",mode:"standard",shareOriginals:true,background:"简洁浅色背景，商品与文字之间保留清晰留白",language:"zh",carousel:[],detail:[{id:"generic-product-details",purpose:"key_benefits",brief:"围绕原图中商品的真实外观和已确认事实编排一张清晰详情图。可使用至多两条有依据的简短中文标签；没有对应事实则不加标签。保留原外观、颜色和比例，不虚构尺寸、材质、性能、认证、包装或配件，不呈现资料文档或内部说明。"}]}},
] as const satisfies readonly {skillId:string;name:string;description:string;image:ImageSetTemplate}[];

export type GenericImagePreset=typeof genericImagePresets[number];
export function genericImagePresetForSkill(skill:SkillSelection|undefined){return genericImagePresets.find(preset=>preset.skillId===skill?.skillId)}

export function matchesGenericImagePreset(template:ImageAgentTemplate,preset:GenericImagePreset){
  // The existing strict schema fixes field order and rejects extra parameters.
  return template.lifecycle==="ACTIVE"&&template.targetPlatform==="product"&&JSON.stringify(imageSetTemplateSchema.parse(template.image))===JSON.stringify(imageSetTemplateSchema.parse(preset.image));
}
