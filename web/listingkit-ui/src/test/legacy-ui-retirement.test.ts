// @vitest-environment node
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const sourceRoot = path.resolve(process.cwd(), "src");

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
  ])("keeps source files under %s physically absent", (relative) => {
    const target = path.join(sourceRoot, relative);
    // Git does not track empty directories; local deletion may leave them behind.
    const remaining = !existsSync(target) ? [] : statSync(target).isDirectory()
      ? productionSources(target) : [target];
    expect(remaining, relative).toEqual([]);
  });

  it("keeps current routes free of retired imports and request/link targets", () => {
    const retired = /(?:@\/(?:components\/listingkit|app\/api\/listing-kits|lib\/types\/listingkit|lib\/listingkit)(?:\/|["'])|["'`]\/(?:api\/)?listing-kits(?:\/|["'`?]))/;
    const consumers = productionSources(sourceRoot).filter((file) =>
      retired.test(readFileSync(file, "utf8")),
    ).map((file) => path.relative(sourceRoot, file));
    expect(consumers).toEqual([]);
  });
});
