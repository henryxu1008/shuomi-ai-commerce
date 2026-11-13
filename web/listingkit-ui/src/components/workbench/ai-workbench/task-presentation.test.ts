import { describe, expect, it } from "vitest";
import { taskReason } from "./task-presentation";

describe("task reason", () => {
  it("uses content terminology for content tasks and preserves title terminology for title tasks", () => {
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.content.localize")).toBe("需要人工审核内容建议");
    expect(taskReason("AGENT_RESULT_NOT_REVIEWABLE", "product.content.localize")).toBe("内容建议未通过验证，无法提交审核");
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.title.optimize")).toBe("需要人工审核标题建议");
  });
});
