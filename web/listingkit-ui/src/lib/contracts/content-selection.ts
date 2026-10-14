import { z } from "zod";

export const contentFieldLabels = { title: "标题", description: "描述", sellingPoints: "卖点", seoKeywords: "SEO关键词" } as const;
export type ContentField = keyof typeof contentFieldLabels;
export const contentSelectionSchema = z.strictObject({
  targetLanguage: z.string().min(1).max(128).refine((value) => {
    try { return value.split("-")[0] !== "und" && Intl.getCanonicalLocales(value)[0] === value; }
    catch { return false; }
  }, { message: "请输入规范语言标签，例如 en-US、zh-Hans 或 ja" }),
  title: z.boolean().optional(),
  description: z.boolean().optional(),
  sellingPoints: z.boolean().optional(),
  seoKeywords: z.boolean().optional(),
}).refine((value) => Object.keys(contentFieldLabels).some((field) => value[field as ContentField]), { message: "至少选择一个文本字段" });
export type ContentSelection = z.infer<typeof contentSelectionSchema>;
export function sameContentSelection(a: ContentSelection, b: ContentSelection) {
  return a.targetLanguage === b.targetLanguage && Object.keys(contentFieldLabels).every((field) => !!a[field as ContentField] === !!b[field as ContentField]);
}
export const contentParametersSchema = z.strictObject({ schema: z.literal("content-config-v1"), selection: contentSelectionSchema });
export function contentSelectionLabel(selection: ContentSelection) {
  return `${selection.targetLanguage} · ${Object.entries(contentFieldLabels).filter(([field]) => selection[field as ContentField]).map(([, label]) => label).join("、")}`;
}
