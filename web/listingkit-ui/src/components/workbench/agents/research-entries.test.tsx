import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { ResearchEntries } from "./research-entries";
afterEach(cleanup);

it("offers the existing workflows independently of agent activation", () => {
  render(<ResearchEntries enabled permissions={["workbench.project.read", "workbench.collection.read"]} />);
  expect(screen.getByRole("link", {name:"市场调研"})).toHaveAttribute("href", "/workbench/data/research");
  expect(screen.getByRole("link", {name:"智能选品"})).toHaveAttribute("href", "/workbench/supply/selection");
});

it("does not advertise inaccessible business workflows as available", () => {
  const view = render(<ResearchEntries enabled={false} permissions={[]} />);
  expect(screen.queryByRole("link", {name:"市场调研"})).toBeNull();
  expect(screen.getAllByText("暂未开放")).toHaveLength(2);
  view.rerender(<ResearchEntries enabled permissions={["workbench.project.read"]} />);
  expect(screen.getByRole("link", {name:"市场调研"})).toBeVisible();
  expect(screen.queryByRole("link", {name:"智能选品"})).toBeNull();
  expect(screen.getByText("需要商品资料查看权限")).toBeVisible();
});
