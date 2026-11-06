import {afterEach, beforeEach, expect, it, vi} from "vitest";
import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {ResearchError, researchRequest} from "@/lib/api/market-research";
import {researchReceipt, supplierReviewPage, supplierReviewPreview} from "@/lib/contracts/market-research";
import {ResearchSupplierReview} from "./research-supplier-review";

const fixtures = vi.hoisted(() => ({
  project:"1f04212a-5250-4767-9497-ecc1fc01fa22",
  candidate:"e86bb116-73a3-4155-b480-e8b125c3bf18",
  item:"f145bf32-6d89-4108-b8e5-7c783ca5b4a9",
  publication:"source-run:acquisition:a9250c43-d917-4720-8c30-88b6c445ee58",
  link:"2fda71ec-349c-4d62-9465-158f0e63944e",
}));

vi.mock("@/lib/api/market-research", async importOriginal => {
  const original = await importOriginal<typeof import("@/lib/api/market-research")>();
  return {...original, researchRequest:vi.fn()};
});

vi.mock("../pod/collection-picker", () => ({
  CollectionPicker: ({onSelect}:{onSelect:(item:unknown)=>void}) =>
    <button onClick={() => onSelect({id:fixtures.item,revision:2,source:{publicationId:fixtures.publication,version:"3"}})}>选合成货源</button>,
}));

const scope = {userId:"researcher",organizationId:"research-org"};
const request = vi.mocked(researchRequest);
const reason = "包装不同，作为研究货源";
const offer = {
  revision:7,linkId:fixtures.link,evidenceDigest:"a".repeat(64),
  market:{asin:"B000123456",title:"合成 Amazon 候选",attributes:{pack:"1 件装"}},
  offer:{title:"合成 1688 货源",attributes:{},variants:[{id:"blue",label:"蓝色",attributes:{pack:"2 件装"}}]},
};
const detail = {
  marketTitle:offer.market.title,asin:offer.market.asin,supplierTitle:offer.offer.title,offerId:"123456",
  marketVersion:"1",supplierVersion:"3",skuId:"blue",skuLabel:"蓝色",quantity:20,unit:"piece",status:"CONFIRMED",reason,
  at:"2026-11-06T00:00:00Z",comparisons:[{marketKey:"pack",supplierKey:"pack",supplierLevel:"sku",result:"DIFFERENT",marketValue:"1 件装",supplierValue:"2 件装"}],
};

beforeEach(() => {sessionStorage.clear();request.mockReset();});
afterEach(cleanup);

function panel(onSaved=vi.fn()) {
  return <ResearchSupplierReview scope={scope} projectId={fixtures.project} candidateId={fixtures.candidate} canWrite onSaved={onSaved}/>;
}

async function fill() {
  fireEvent.click(await screen.findByRole("button",{name:"选择已收藏的 1688 货源"}));
  fireEvent.click(screen.getByRole("button",{name:"选合成货源"}));
  fireEvent.change(await screen.findByLabelText("采购意向数量（件）"),{target:{value:"20"}});
  fireEvent.change(screen.getByLabelText("Amazon 规格字段"),{target:{value:"pack"}});
  fireEvent.change(screen.getByLabelText("1688 规格字段"),{target:{value:"pack"}});
  fireEvent.change(screen.getByLabelText("人工核对结果"),{target:{value:"DIFFERENT"}});
  fireEvent.change(screen.getByLabelText("人工理由"),{target:{value:reason}});
  fireEvent.click(screen.getByRole("button",{name:"保存人工货源确认"}));
}

it("submits an exact SKU and original evidence, then masks a previously visible record on a fresh read", async () => {
  let hidden = false, saved = false;
  request.mockImplementation(async (_scope,_project,schema,_signal,options) => {
    if (schema === supplierReviewPreview) return offer;
    if (schema === supplierReviewPage) return {items:saved ? [{current:true,...(hidden ? {} : {detail})}] : []};
    expect(options?.intent?.body).toMatchObject({
      action:"record_supplier_review",linkId:fixtures.link,expectedRevision:7,
      supplier:{skuId:"blue",quantity:20,unit:"piece",reason,selection:{itemId:fixtures.item,revision:2,publicationId:fixtures.publication,version:"3"}},
    });
    saved = true;
    return {key:options?.intent?.key,candidateIds:[fixtures.candidate],revisions:{[fixtures.candidate]:8},replayed:false};
  });
  const onSaved = vi.fn();
  render(panel(onSaved));
  await screen.findByText("还没有保存货源核对。");
  await fill();
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
  fireEvent.click(screen.getByRole("button",{name:"刷新核对记录"}));
  expect(await screen.findByText("人工理由："+reason)).toBeVisible();
  expect(screen.getByText(/蓝色 · blue · 20 件/)).toBeVisible();
  hidden = true;
  fireEvent.click(screen.getByRole("button",{name:"刷新核对记录"}));
  expect(await screen.findByText("原资料当前不可查看。")).toBeVisible();
  expect(screen.queryByText("人工理由："+reason)).not.toBeInTheDocument();
  expect(screen.queryByText(/蓝色 · blue · 20 件/)).not.toBeInTheDocument();
});

it("removes the open history when a new preview reports access loss", async () => {
  request.mockImplementation(async (_scope,_project,schema) => {
    if (schema === supplierReviewPreview) throw new ResearchError("FORBIDDEN",403);
    return {items:[{current:true,detail}]};
  });
  render(panel());
  expect(await screen.findByText("人工理由："+reason)).toBeVisible();
  fireEvent.click(screen.getByRole("button",{name:"选择已收藏的 1688 货源"}));
  fireEvent.click(screen.getByRole("button",{name:"选合成货源"}));
  expect(await screen.findByText("原资料当前不可查看。")).toBeVisible();
  expect(screen.queryByText("人工理由："+reason)).not.toBeInTheDocument();
  expect(screen.queryByText(/蓝色 · blue · 20 件/)).not.toBeInTheDocument();
});

it("keeps the original receipt key if a retry is denied before the original outcome is known", async () => {
  const posted:string[] = [];
  request.mockImplementation(async (_scope,_project,schema,_signal,options) => {
    if (schema === supplierReviewPreview) return offer;
    if (schema === supplierReviewPage) return {items:[{current:true,detail}]};
    if (options?.intent) {
      posted.push(options.intent.key);
      if (posted.length === 1) throw new ResearchError("OUTCOME_UNKNOWN",503);
      throw new ResearchError("FORBIDDEN",403);
    }
    expect(options?.lookup).toBe(posted[0]);
    if (posted.length === 1) throw new ResearchError("NOT_FOUND",404);
    return {key:posted[0],candidateIds:[fixtures.candidate],revisions:{[fixtures.candidate]:8},replayed:true};
  });
  const onSaved = vi.fn();
  render(panel(onSaved));
  expect(await screen.findByText("人工理由："+reason)).toBeVisible();
  await fill();
  await screen.findByText("保存结果暂未确认，请查询原请求。");
  expect(screen.queryByRole("button",{name:"按原请求重试"})).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:"查询货源核对原请求"}));
  fireEvent.click(await screen.findByRole("button",{name:"按原请求重试"}));
  await screen.findByText(/原资料当前不可查看/);
  expect(posted).toEqual([posted[0],posted[0]]);
  expect(JSON.parse(sessionStorage.getItem(sessionStorage.key(0) ?? "") ?? "null")).toEqual({key:posted[0]});
  expect(screen.queryByText("人工理由："+reason)).not.toBeInTheDocument();
  expect(screen.queryByRole("button",{name:"按原请求重试"})).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:"查询货源核对原请求"}));
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
});

it("recovers a lost response after reload using only the original request key", async () => {
  let key = "";
  request.mockImplementation(async (_scope,_project,schema,_signal,options) => {
    if (schema === supplierReviewPreview) return offer;
    if (schema === supplierReviewPage) return {items:[]};
    expect(schema).toBe(researchReceipt);
    if (options?.intent) {
      key = options.intent.key;
      throw new ResearchError("OUTCOME_UNKNOWN",503);
    }
    expect(options?.lookup).toBe(key);
    return {key,candidateIds:[fixtures.candidate],revisions:{[fixtures.candidate]:8},replayed:true};
  });
  const first = render(panel());
  await screen.findByText("还没有保存货源核对。");
  await fill();
  await screen.findByText("保存结果暂未确认，请查询原请求。");
  const stored = sessionStorage.getItem(sessionStorage.key(0) ?? "");
  expect(JSON.parse(stored ?? "null")).toEqual({key});
  first.unmount();
  const onSaved = vi.fn();
  render(panel(onSaved));
  expect(screen.queryByRole("button",{name:"按原请求重试"})).not.toBeInTheDocument();
  fireEvent.click(await screen.findByRole("button",{name:"查询货源核对原请求"}));
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
  expect(sessionStorage.length).toBe(0);
  expect(request.mock.calls.filter(call => call[4]?.intent)).toHaveLength(1);
});
