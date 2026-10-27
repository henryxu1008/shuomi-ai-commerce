import { afterEach, expect, it, vi } from "vitest";
import {
  createRoleInput,
  getEnterpriseRoles,
  parseEnterpriseRoles,
  saveEnterpriseRole,
  saveRoleInput,
} from "./enterprise-roles";
import { roleDefinition } from "./enterprise-role-schema";
const key = "e5836e4e-9c13-4df4-a67d-4c4e896b941a";
const id = "sumi_role_e87cb45c05ad389dff6dea6e7bf581ee_01";
afterEach(() => vi.unstubAllGlobals());
it("rejects a save receipt for another role without changing the original intent", async () => {
  const fetch = vi
    .fn()
    .mockResolvedValue(
      Response.json({
        schemaVersion: "enterprise-role-mutation-v1",
        userId: "actor",
        organizationId: "org",
        operationId: key,
        role: {
          id: id.slice(0, -2) + "02",
          name: "other",
          modules: [],
          version: 2,
          system: false,
        },
      }),
    );
  vi.stubGlobal("fetch", fetch);
  await expect(
    saveEnterpriseRole(
      { expectedUserId: "actor", expectedOrganizationId: "org" },
      key,
      id,
      { modules: [], expectedVersion: 1 },
    ),
  ).rejects.toMatchObject({ code: "INVALID_UPSTREAM_RESPONSE" });
  expect(fetch.mock.calls[0][1].headers.get("Idempotency-Key")).toBe(key);
});

function roleCatalogPayload(size: number) {
  const modules = Array.from({ length: size }, (_, index) =>
    index === 0
      ? "store-browser"
      : `module-${String.fromCharCode(97 + Math.floor(index / 26))}${String.fromCharCode(97 + (index % 26))}`,
  );
  return {
    schemaVersion: "enterprise-roles-v1",
    userId: "actor",
    organizationId: "org",
    items: [
      {
        id: "listingkit_admin",
        name: "管理员",
        modules,
        version: 0,
        system: true,
      },
    ],
    catalog: modules.map((module) => ({
      id: module,
      group: "目录",
      label: module,
      available: true,
      permissions: [],
    })),
    canManage: true,
    canCreate: true,
    remainingSlots: 64,
  };
}

it("retains the original 32-module read boundary case", () => {
  expect(parseEnterpriseRoles(roleCatalogPayload(32)).catalog).toHaveLength(32);
});

it.each([33, 48, 64])(
  "reads a %i-module catalog and full system projection through the real parser and client",
  async (size) => {
    const payload = roleCatalogPayload(size);
    expect(parseEnterpriseRoles(payload).items[0].modules).toHaveLength(size);
    const fetch = vi.fn().mockResolvedValue(Response.json(payload));
    vi.stubGlobal("fetch", fetch);
    const result = await getEnterpriseRoles({
      expectedUserId: "actor",
      expectedOrganizationId: "org",
    });
    expect(result.catalog).toHaveLength(size);
    expect(fetch).toHaveBeenCalledOnce();
    expect(fetch.mock.calls[0][0]).toBe("/api/account/roles");
  },
);

it("keeps a finite read catalog and system projection bound", () => {
  const oversizedCatalog = roleCatalogPayload(65);
  oversizedCatalog.items[0].modules = [];
  expect(() => parseEnterpriseRoles(oversizedCatalog)).toThrow();
  expect(() => roleDefinition.parse(roleCatalogPayload(65).items[0])).toThrow();
});

it("keeps custom role definitions and create/save inputs limited to 32 modules", () => {
  const modules = roleCatalogPayload(32).items[0].modules;
  const custom = { id, name: "角色", modules, version: 1, system: false };
  expect(roleDefinition.safeParse(custom).success).toBe(true);
  expect(createRoleInput.safeParse({ name: "角色", modules }).success).toBe(true);
  expect(saveRoleInput.safeParse({ modules, expectedVersion: 1 }).success).toBe(true);
  const tooMany = roleCatalogPayload(33).items[0].modules;
  expect(roleDefinition.safeParse({ ...custom, modules: tooMany }).success).toBe(false);
  expect(createRoleInput.safeParse({ name: "角色", modules: tooMany }).success).toBe(false);
  expect(saveRoleInput.safeParse({ modules: tooMany, expectedVersion: 1 }).success).toBe(false);
  expect(createRoleInput.safeParse({ name: "浏览器", modules: ["store-browser"] }).success).toBe(true);
});

it("still rejects duplicate modules and roles, unavailable modules and missing catalog entries", () => {
  const duplicateModules = roleCatalogPayload(33);
  duplicateModules.items[0].modules[32] = duplicateModules.items[0].modules[0];
  expect(() => parseEnterpriseRoles(duplicateModules)).toThrow();
  const duplicateRoles = roleCatalogPayload(33);
  duplicateRoles.items.push(duplicateRoles.items[0]);
  expect(() => parseEnterpriseRoles(duplicateRoles)).toThrow();
  const unavailable = roleCatalogPayload(33);
  unavailable.catalog[0].available = false;
  expect(() => parseEnterpriseRoles(unavailable)).toThrow();
  const missing = roleCatalogPayload(33);
  missing.items[0].modules[0] = "missing-module";
  expect(() => parseEnterpriseRoles(missing)).toThrow();
});
