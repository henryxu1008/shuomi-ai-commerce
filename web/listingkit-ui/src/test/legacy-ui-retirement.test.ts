// @vitest-environment node
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const sourceRoot = path.resolve(process.cwd(), "src");
const repoRoot = path.resolve(process.cwd(), "../..");

function productionSources(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) return productionSources(file);
    return /\.[cm]?[jt]sx?$/.test(file) && !/\.(test|spec|type-test)\./.test(file)
      ? [file]
      : [];
  });
}

describe("retired Task-first frontend", () => {
  it.each([
    "app/listing-kits", "components/listingkit", "app/api/listing-kits",
    "app/api/shein-login", "app/api/sds-login", "lib/types/listingkit.ts",
    "lib/types/listingkit", "lib/api/create-task.ts", "lib/api/task-recovery.ts",
    "lib/query/use-child-task-retry.ts", "lib/api/submit.ts", "lib/listingkit",
    "lib/api/generated",
    "app/api/sds", "lib/api/sds-products-schema.ts", "lib/api/sds-products-schema.test.ts",
    "lib/types/sds.ts", "lib/utils/sds-recent-variants.ts", "lib/utils/sds-recent-variants.test.ts",
  ])("keeps source files under %s physically absent", (relative) => {
    const target = path.join(sourceRoot, relative);
    // Git does not track empty directories; local deletion may leave them behind.
    const remaining = !existsSync(target) ? [] : statSync(target).isDirectory()
      ? productionSources(target) : [target];
    expect(remaining, relative).toEqual([]);
  });

  it("keeps the old frontend SDK generator and make target retired", () => {
    expect(existsSync(path.join(repoRoot, "scripts/generate-api-contract.mjs"))).toBe(false);
    expect(readFileSync(path.join(repoRoot, "Makefile"), "utf8"))
      .not.toMatch(/(?:^api-contract-check:|generate-api-contract\.mjs)/m);
  });

  it("keeps current routes free of retired imports and request/link targets", () => {
    const retired = /(?:@\/(?:components\/listingkit|app\/api\/listing-kits|lib\/types\/listingkit|lib\/listingkit)(?:\/|["'])|["'`]\/(?:api\/)?listing-kits(?:\/|["'`?]))/;
    const consumers = productionSources(sourceRoot).filter((file) =>
      retired.test(readFileSync(file, "utf8")),
    ).map((file) => path.relative(sourceRoot, file));
    expect(consumers).toEqual([]);
  });

  it("keeps current sources free of old SDS imports and request targets", () => {
    const retired = /(?:@\/(?:app\/api\/sds|lib\/types\/sds|lib\/api\/sds-products-schema|lib\/utils\/sds-recent-variants)(?:\/|["'])|["'`]\/(?:api\/sds|api\/v1\/sds(?:-login)?)(?:\/|["'`?]))/;
    const consumers = productionSources(sourceRoot).filter((file) =>
      retired.test(readFileSync(file, "utf8")),
    ).map((file) => path.relative(sourceRoot, file));
    expect(consumers).toEqual([]);
  });
});
