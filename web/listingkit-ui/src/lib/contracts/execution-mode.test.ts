import { describe, expect, it } from "vitest";
import { executionModeInputSchema, executionModeViewSchema, modeRevisionSchema } from "./execution-mode";

const subject = { kind: "private_delivery", id: "1dc92db7-6e38-4d89-9b71-9b12432f3678" };
const view = { subject, definition: { id: "product.quality.check", version: "2.0.0" }, mode: "default", revision: "0", policyVersion: "agent-execution-policy-v1", canConfigure: true, updatedAt: "0001-01-01T00:00:00Z", updatedBy: "" };

describe("execution mode contracts", () => {
  it("preserves exact private-delivery identity and the default zero revision", () => {
    expect(executionModeViewSchema.parse(view).definition.version).toBe("2.0.0");
    for (const mode of ["default", "auto_approve", "read_only", "full_access"]) {
      expect(executionModeInputSchema.safeParse({ mode, expectedRevision: "0" }).success).toBe(true);
    }
  });
  it("rejects arbitrary authority, mode and noncanonical revision", () => {
    for (const revision of ["", "00", "01", "+1", "-1", "9223372036854775808"]) expect(modeRevisionSchema.safeParse(revision).success).toBe(false);
    expect(executionModeInputSchema.safeParse({ mode: "admin", expectedRevision: "0" }).success).toBe(false);
    expect(executionModeInputSchema.safeParse({ mode: "full_access", expectedRevision: "0", permissions: ["*"] }).success).toBe(false);
    expect(executionModeViewSchema.safeParse({ ...view, subject: { kind: "private_delivery", id: "foreign-name" } }).success).toBe(false);
    expect(executionModeViewSchema.safeParse({ ...view, mode: "unrestricted" }).success).toBe(false);
  });
  it("binds a mode command receipt to the returned subject, mode and revision", () => {
    const receipt = { commandId: "e1b16b08-46e7-4a6d-9059-2588e2822e69", operation: "set-execution-mode", subject, mode: "full_access", beforeRevision: "0", revision: "1", noop: false, committedAt: "2026-10-12T01:00:00Z" };
    const saved = { ...view, mode: "full_access", revision: "1", receipt };
    expect(executionModeViewSchema.safeParse(saved).success).toBe(true);
    expect(executionModeViewSchema.safeParse({ ...saved, mode: "read_only" }).success).toBe(false);
    expect(executionModeViewSchema.safeParse({ ...saved, receipt: { ...receipt, subject: { ...subject, id: "d545b0de-10f5-43d4-b2a2-885a77d93463" } } }).success).toBe(false);
  });
});
