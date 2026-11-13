import { describe, expect, it } from "vitest";
import { taskPoints, taskReason } from "./task-presentation";

describe("task reason", () => {
  it("uses content terminology for content tasks and preserves title terminology for title tasks", () => {
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.content.localize")).toBe("需要人工审核内容建议");
    expect(taskReason("AGENT_RESULT_NOT_REVIEWABLE", "product.content.localize")).toBe("内容建议未通过验证，无法提交审核");
    expect(taskReason("HUMAN_REVIEW_REQUIRED", "product.title.optimize")).toBe("需要人工审核标题建议");
  });
});

it("distinguishes settled points from unknown or partial settlement", () => {
  expect(taskPoints("settled", "2665766")).toBe("已结算 2.665766 AI 点");
  expect(taskPoints("partial", "900000")).toBe("截至当前已结算 0.9 AI 点");
  expect(taskPoints("unavailable", undefined)).toBe("结算中或暂不可用");
});
