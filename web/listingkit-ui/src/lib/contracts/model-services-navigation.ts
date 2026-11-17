import { routeFact } from "./model-route-pricing";
import { isAcquisitionUUID } from "./product-acquisition";

export const modelServicesPath = "/workbench/account/organization/model-route-pricing";
const modelServiceCategories = ["text", "multimodal", "image-generation"] as const;
export type ModelServiceCategory = (typeof modelServiceCategories)[number];

const chatPath = /^\/workbench\/ai\/chat(?:\/[A-Za-z0-9_-]{1,128})?$/;
const imagePaths = [
  /^\/workbench\/supply\/acquisition\/operation\/[A-Za-z0-9_-]{1,128}$/,
  /^\/workbench\/listing\/missing\/[A-Za-z0-9_-]{1,128}(?:\/shein)?$/,
  /^\/workbench\/data\/mine\/own\/[A-Za-z0-9_-]{1,128}\/images$/,
  /^\/workbench\/agents\/mine(?:\/product\.image\.agent)?$/,
];

function businessReturn(category: ModelServiceCategory, value: unknown) {
  if (typeof value !== "string" || value.length > 1536 || /[\\#\u0000-\u001f]/.test(value)) return undefined;
  const path = value.split("?", 1)[0];
  if (category !== "text" && path === "/workbench/listing/missing") {
    const query = new URLSearchParams(value.slice(path.length));
    const source = query.getAll("source");
    const preparation = query.getAll("preparation");
    const fixedDraft = source.length === 1 && isAcquisitionUUID(source[0]) &&
      preparation.length === 1 && isAcquisitionUUID(preparation[0]) && query.size === 2;
    return fixedDraft ? value : undefined;
  }
  const valid = category === "text" ? chatPath.test(path) : imagePaths.some(pattern => pattern.test(path));
  return valid ? value : undefined;
}

export function modelServicesHref(category: ModelServiceCategory, organizationId: string, returnTo?: string) {
  const search = new URLSearchParams({ purpose: category, org: organizationId });
  const back = businessReturn(category, returnTo);
  if (back) search.set("back", back);
  return modelServicesPath + "?" + search;
}

export function readModelServicesNavigation(query: Record<string, unknown>) {
  const category = modelServiceCategories.find(value => value === query.purpose);
  const organization = routeFact.safeParse(query.org);
  if (!category || !organization.success) return undefined;
  return { category, organizationId: organization.data, returnTo: businessReturn(category, query.back) };
}

export type ModelServicesNavigation = ReturnType<typeof readModelServicesNavigation>;
