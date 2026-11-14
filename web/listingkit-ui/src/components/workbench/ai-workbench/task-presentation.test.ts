import { describe, expect, it } from "vitest";
import { taskReason } from "./task-presentation";

describe("task reason", () => {
  it("uses content terminology for content tasks and preserves title terminology for title tasks", () => {
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.content.localize")).toBe("需要人工审核内容建议");
    expect(taskReason("AGENT_RESULT_NOT_REVIEWABLE", "product.content.localize")).toBe("内容建议未通过验证，无法提交审核");
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.title.optimize")).toBe("需要人工审核标题建议");
  });
  it("explains known stopped-task reasons without exposing internal codes", () => {
    expect(taskReason("budget_cost", "product.content.localize")).toBe("任务已达到费用上限，无法继续");
    expect(taskReason("budget_tokens", "product.content.localize")).toBe("任务已达到处理量上限，无法继续");
    expect(taskReason("tool_error", "product.content.localize")).toBe("任务处理遇到问题，请稍后重试");
    expect(taskReason("dependency_unavailable", "product.content.localize")).toBe("相关服务暂不可用，请稍后重试");
  });
});
