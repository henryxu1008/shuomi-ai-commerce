import { z } from "zod";
export const enterpriseRoleKey = z.string().regex(/^sumi_role_[a-f0-9]{32}_(0[1-9]|[1-5][0-9]|6[0-4])$/);
export const assignableRole = z.union([z.literal("listingkit_admin"), enterpriseRoleKey]);
// The read-only catalog/system projection can outgrow the 32-module write limit.
export const enterpriseModuleCatalogLimit = 64;
const uniqueModuleIds = z.array(z.string().regex(/^[a-z][a-z-]{0,39}$/)).refine(v => new Set(v).size === v.length);
export const moduleIds = uniqueModuleIds.max(32);
const projectedModuleIds = uniqueModuleIds.max(enterpriseModuleCatalogLimit);
export const roleDefinition = z.object({ id: assignableRole, name: z.string().min(1).max(160), modules: projectedModuleIds, version: z.number().int().min(0).max(1e9), system: z.boolean() }).strict().refine(v => v.system ? v.id === "listingkit_admin" && v.version === 0 : v.id !== "listingkit_admin" && v.version > 0 && moduleIds.safeParse(v.modules).success);
export type EnterpriseRole = z.infer<typeof roleDefinition>;
