import { describe, expect, it } from "vitest";
import { compareKernelVersions, kernelIntent, kernelPublish, kernelQuery, matchesKernelArtifact } from "./kernel-releases";
const payload = { version: "150.0.7871.186", os: "windows", arch: "amd64", minClientVersion: "1.8.0", maxClientVersionExclusive: "1.8.1-0", archiveFormat: "zip", size: 193743241, sha256: "a".repeat(64), downloadUrl: "https://assets.example.test/kernel.zip" } as const;
describe("existing kernel publication contract", () => {
  it("accepts Chromium display versions and strictly ordered client SemVer", () => {
    expect(kernelPublish.safeParse(payload).success).toBe(true);
    for (const max of ["1.8.0", "1.7.9", "1.8.0-alpha", "01.9.0", "1.9", "1.8", "1.9.0-01"]) expect(kernelPublish.safeParse({ ...payload, maxClientVersionExclusive: max }).success).toBe(false);
    expect(kernelPublish.safeParse({ ...payload, minClientVersion: "1.8" }).success).toBe(false);
    expect(kernelPublish.safeParse({ ...payload, maxClientVersionExclusive: "" }).success).toBe(true);
    expect(kernelQuery.safeParse({ os: "windows", arch: "amd64", clientVersion: "1.8.0.0" }).success).toBe(false);
  });
  it("rejects unsafe metadata and extra fields", () => {
    for (const change of [{ size: 0 }, { size: 2 ** 31 + 1 }, { size: 1.5 }, { sha256: "A".repeat(64) }, { downloadUrl: "https://assets.example.test/kernel.zip?token=x" }, { downloadUrl: "https://user:pass@assets.example.test/k.zip" }, { downloadUrl: "http://assets.example.test/k.zip" }, { version: "bad\nversion" }, { recommended: true }]) expect(kernelPublish.safeParse({ ...payload, ...change }).success).toBe(false);
  });
  it("requires exact immutable metadata and a nonrecommended readback", () => {
    const id = "12345678-1234-4234-8234-123456789abc";
    const artifact = { ...payload, releaseId: id, recommended: false, downloadOrigins: ["https://assets.example.test"] };
    expect(matchesKernelArtifact(id, payload, artifact)).toBe(true);
    expect(matchesKernelArtifact(id, payload, { ...artifact, sha256: "b".repeat(64) })).toBe(false);
    expect(matchesKernelArtifact(id, payload, { ...artifact, recommended: true })).toBe(false);
  });
  it("compares numeric and hyphenated prereleases without numeric precision loss", () => {
    const ordered = ["1.8.0-alpha", "1.8.0-alpha-b", "1.8.0-beta.2", "1.8.0-beta.11", "1.8.0-rc.1", "1.8.0", "1.8.1-0"];
    for (let i = 1; i < ordered.length; i++) expect(compareKernelVersions(ordered[i - 1], ordered[i])).toBeLessThan(0);
    expect(compareKernelVersions("1.8.0+build", "1.8.0")).toBe(0);
    expect(compareKernelVersions("1.8.0-99999999999999999999", "1.8.0-100000000000000000000")).toBeLessThan(0);
    const intent = { key: "12345678-1234-4234-8234-123456789abc", clientVersion: "1.8.1-0", payload };
    expect(kernelIntent.safeParse(intent).success).toBe(false);
  });
});
