import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { ResearchCostPreview } from "./research-cost-preview";

afterEach(cleanup);

function pendingRead() {
  let resolve!: (current: import("@/lib/workbench/research-cost-preview").CostResult) => void;
  const promise = new Promise<import("@/lib/workbench/research-cost-preview").CostResult>(done => { resolve = done; });
  return { promise, resolve };
}

it("ignores a late read after editing and retains input after a transient read failure", async () => {
  const old = pendingRead();
  const calculate = vi.fn().mockReturnValueOnce(old.promise).mockRejectedValueOnce(new Error("unavailable"))
    .mockResolvedValueOnce({status:"INSUFFICIENT_INPUT",fields:["price"]});
  render(<ResearchCostPreview calculate={calculate} />);
  const price = screen.getByLabelText("假设售价（USD/件）");
  fireEvent.change(price, { target: { value: "30" } });
  fireEvent.click(screen.getByRole("button", { name: "查看参考试算" }));
  fireEvent.change(price, { target: { value: "20" } });
  expect(calculate.mock.calls[0][1].aborted).toBe(true);
  await act(async () => old.resolve({status:"INSUFFICIENT_INPUT",fields:["price"]}));
  expect(screen.queryByText("资料仍有缺项")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "查看参考试算" }));
  expect(await screen.findByText(/已保留输入/)).toBeVisible();
  expect(price).toHaveValue("20");
  fireEvent.click(screen.getByRole("button", { name: "查看参考试算" }));
  expect(await screen.findByText("资料仍有缺项")).toBeVisible();
});

it("clears the draft when evidence is no longer visible", async () => {
  render(<ResearchCostPreview calculate={async () => {throw new Error("FORBIDDEN")}} />);
  const price = screen.getByLabelText("假设售价（USD/件）");
  fireEvent.change(price, { target: { value: "30" } });
  fireEvent.click(screen.getByRole("button", { name: "查看参考试算" }));
  expect(await screen.findByText(/原商品资料已变化或暂不可读/)).toBeVisible();
  expect(price).toHaveValue("");
  expect(screen.queryByText("资料仍有缺项")).not.toBeInTheDocument();
});

it("requires new procurement amounts after the currency changes", () => {
  render(<ResearchCostPreview calculate={async () => ({status:"INSUFFICIENT_INPUT",fields:["price"]})} />);
  fireEvent.change(screen.getByLabelText("正常成交 采购与包装消耗（USD/件）"), { target: { value: "8" } });
  fireEvent.change(screen.getByLabelText("采购币种"), { target: { value: "CNY" } });
  expect(screen.getByLabelText("正常成交 采购与包装消耗（CNY/件）")).toHaveValue("");
  expect(screen.getByLabelText("人工汇率（1 CNY 折合多少 USD）")).toHaveValue("");
});
