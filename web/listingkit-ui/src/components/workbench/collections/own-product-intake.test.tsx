import {webcrypto} from "node:crypto";
import {cleanup,render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
import {MarketAPIError} from "@/lib/api/supply-market";
import {OwnProductIntake} from "./own-product-intake";

const {request}=vi.hoisted(()=>({request:vi.fn()}));
vi.mock("@/lib/api/pod",()=>({podRequest:request}));

const scope={userId:"actor",organizationId:"org"};
const notes="黑 M，SKU BLK-M，CNY 29，库存 0";
const prepared={configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:"11111111-1111-4111-8111-111111111111",Digest:"a".repeat(64)},
 quote:{InputHash:"b".repeat(64),ProfileDigest:"c".repeat(64),MaximumTokens:2000,MaximumCostMicros:1000,Currency:"CNY"}};
const panel=()=>render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={vi.fn()}/>);
afterEach(()=>{cleanup();sessionStorage.clear();request.mockReset();vi.unstubAllGlobals()});

it("shows a row and field reason without adopting or calling again",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup();
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"stopped",stopReason:"invalid_model_output",validationError:{field:"rows[0].stock",code:"stock_evidence"}}));
 const adopt=vi.fn();
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),notes);
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 const error=await screen.findByRole("alert");
 expect(error).toHaveTextContent("第 1 个 SKU");
 expect(error).toHaveTextContent("库存");
 expect(error).toHaveTextContent("明确");
 expect(screen.queryByRole("button",{name:"采纳到商品表单"})).not.toBeInTheDocument();
 expect(request).toHaveBeenCalledTimes(2);
 expect(adopt).not.toHaveBeenCalled();
});

it("shows only the three supplied variants and preserves zero versus missing before save",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup(),adopt=vi.fn();
 const rows=[
  {attributes:{颜色:"黑",尺码:"M"},evidence:"黑 M，库存明确为 0",sku:"BLK-M",stockSuggestion:{quantity:0,evidence:"库存明确为 0"}},
  {attributes:{颜色:"黑",尺码:"L"},evidence:"黑 L，库存未填写",sku:"BLK-L"},
  {attributes:{颜色:"白",尺码:"M"},evidence:"白 M，库存为5",sku:"WHT-M",stockSuggestion:{quantity:5,evidence:"库存为5"}},
 ];
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"human_review_required",candidate:{title:"黑白T恤",rows}}));
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),rows.map(row=>row.evidence).join("\n"));
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 const region=await screen.findByRole("region",{name:"AI 生成候选"});
 expect(region.querySelectorAll("li")).toHaveLength(3);
 expect(region.querySelectorAll("li")[0]).toHaveTextContent("待确认库存建议：0");
 expect(region.querySelectorAll("li")[1]).toHaveTextContent("未提供库存建议");
 expect(region.querySelectorAll("li")[2]).toHaveTextContent("待确认库存建议：5");
 expect(adopt).not.toHaveBeenCalled();
 expect(request).toHaveBeenCalledTimes(2);
});

it("reconstructs a missing run after reload and confirms only under the original key",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup();
 request.mockResolvedValueOnce(prepared).mockRejectedValueOnce(new MarketAPIError("OUTCOME_UNKNOWN",502));
 const original=panel();
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),notes);
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 await screen.findByRole("alert");
 const originalKey=request.mock.calls[0][4].key;
 const originalBody=request.mock.calls[1][4].body;

 original.unmount();
 request.mockRejectedValueOnce(new MarketAPIError("NOT_FOUND",404));
 panel();
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),"已更换的资料");
 await user.click(screen.getByRole("button",{name:"核实原生成结果"}));
 const recover=await screen.findByRole("button",{name:"按原请求重新检查用量"});
 await user.click(recover);
 expect(await screen.findByRole("alert")).toHaveTextContent("恢复原说明和图库顺序");
 expect(request).toHaveBeenCalledTimes(3);
 expect(screen.getByRole("button",{name:"检查生成用量"})).toBeDisabled();

 await user.clear(screen.getByLabelText("给 AI 的商品资料"));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),notes);
 request.mockResolvedValueOnce(prepared).mockResolvedValueOnce({requestKey:originalKey,phase:"human_review_required",candidate:{title:"三规格上衣"}});
 await user.click(recover);
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 await screen.findByRole("region",{name:"AI 生成候选"});
 expect(request.mock.calls.map((call)=>[call[1],call[4]?.key])).toEqual([
  ["prepare",originalKey],[originalKey+"/confirm",originalKey],[originalKey,undefined],["prepare",originalKey],[originalKey+"/confirm",originalKey],
 ]);
 expect(request.mock.calls[4][4].body).toEqual(originalBody);
});

it("keeps an existing unknown run locked to readback",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup();
 request.mockResolvedValueOnce(prepared).mockRejectedValueOnce(new MarketAPIError("OUTCOME_UNKNOWN",502));
 panel();
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),notes);
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 await screen.findByRole("alert");
 const key=request.mock.calls[0][4].key;
 request.mockResolvedValueOnce({requestKey:key,phase:"stopped",stopReason:"model_outcome_unknown"});
 await user.click(screen.getByRole("button",{name:"核实原生成结果"}));
 await screen.findByText("生成结果待核实。可用原请求键再次查看。");
 expect(screen.queryByRole("button",{name:"按原请求重新检查用量"})).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:"检查生成用量"})).toBeDisabled();
 expect(request).toHaveBeenCalledTimes(3);
});

const priceCandidate={title:"保留有效标题",rows:[
 {attributes:{颜色:"黑",尺码:"M"},evidence:"黑 M",sku:"BLK-M",stockSuggestion:{quantity:0,evidence:"库存 0 或 5"},priceSuggestion:{amount:0,currency:"CNY",evidence:"CNY 0 或 CNY 10"}},
 {attributes:{颜色:"白",尺码:"M"},evidence:"白 M",stockSuggestion:{quantity:5,evidence:"库存5"},priceSuggestion:{amount:10,currency:"USD",evidence:"USD 10"}},
 {attributes:{颜色:"黑",尺码:"L"},evidence:"黑 L",priceSuggestion:{amount:20,evidence:"价格20"}},
]};
const priceNotes="黑 M，CNY 0 或 CNY 10，尚未确认，库存 0 或 5；白 M，USD 10，库存5；黑 L，价格20";
async function generatePriceCandidate(user:ReturnType<typeof userEvent.setup>){
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"human_review_required",candidate:priceCandidate}));
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),priceNotes);
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 await screen.findByRole("region",{name:"AI 生成候选"});
}

it("adopts valid fields without confirming any price and never dispatches for adoption",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup(),adopt=vi.fn();
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await generatePriceCandidate(user);
 expect(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 3 个 SKU 的商品单价和币种")).toBeDisabled();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 await user.click(screen.getByText("查看本次完整原说明"));
 expect(screen.getByText(priceNotes,{selector:"p"})).toBeVisible();
 await user.click(screen.getByRole("button",{name:"采纳到商品表单"}));
 expect(adopt).toHaveBeenCalledWith(priceCandidate,false,[],new Set(),new Set());
 expect(request).toHaveBeenCalledTimes(2);
 expect(screen.queryByRole("button",{name:"采纳到商品表单"})).not.toBeInTheDocument();
});

it("passes only the currently checked price rows, including explicit zero",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup(),adopt=vi.fn();
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await generatePriceCandidate(user);
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种"));
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的库存数量"));
 await user.click(screen.getByLabelText("确认第 2 个 SKU 的商品单价和币种"));
 await user.click(screen.getByLabelText("确认第 2 个 SKU 的商品单价和币种"));
 await user.click(screen.getByRole("button",{name:"收起整理资料"}));
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 expect(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种")).toBeChecked();
 await user.click(screen.getByRole("button",{name:"采纳到商品表单"}));
 expect(adopt).toHaveBeenCalledWith(priceCandidate,false,[],new Set([0]),new Set([0]));
 expect(request).toHaveBeenCalledTimes(2);
});

it("does not revive price confirmation after input or gallery changes and restoration",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup(),adopt=vi.fn();
 const props={scope,member:"member",uploaded:new Set<string>(),disabled:false,onAdopt:adopt};
 const view=render(<OwnProductIntake {...props} images={[]}/>);
 await generatePriceCandidate(user);
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种"));
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的库存数量"));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),"修改");
 expect(screen.queryByRole("region",{name:"AI 生成候选"})).not.toBeInTheDocument();
 await user.clear(screen.getByLabelText("给 AI 的商品资料"));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),priceNotes);
 expect(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种"));
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的库存数量"));
 view.rerender(<OwnProductIntake {...props} images={["https://example.test/new.png"]}/>);
 view.rerender(<OwnProductIntake {...props} images={[]}/>);
 expect(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 expect(request).toHaveBeenCalledTimes(2);
});

it("resets confirmations when a new candidate replaces an unchanged input",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup();
 panel();
 await generatePriceCandidate(user);
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的商品单价和币种"));
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的库存数量"));
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"human_review_required",candidate:priceCandidate}));
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 expect(await screen.findByLabelText("确认第 1 个 SKU 的商品单价和币种")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 expect(request).toHaveBeenCalledTimes(4);
});

it("reads a pending candidate by the original key without confirming prices or dispatching again",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup();
 const signature=JSON.stringify({notes:priceNotes,images:[],image:""});
 const hash=await webcrypto.subtle.digest("SHA-256",new TextEncoder().encode(signature));
 const digest=Array.from(new Uint8Array(hash),v=>v.toString(16).padStart(2,"0")).join("");
 const key="11111111-1111-4111-8111-111111111111";
 sessionStorage.setItem('own-intake:["actor","org","member"]',JSON.stringify({key,digest}));
 request.mockResolvedValueOnce({requestKey:key,phase:"human_review_required",candidate:priceCandidate});
 panel();
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),priceNotes);
 await user.click(screen.getByRole("button",{name:"核实原生成结果"}));
 expect(await screen.findByLabelText("确认第 1 个 SKU 的商品单价和币种")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 expect(request).toHaveBeenCalledTimes(1);
 expect(request.mock.calls[0][1]).toBe(key);
 expect(request.mock.calls[0][4]).toBeUndefined();
 await user.click(screen.getByText("查看本次完整原说明"));
 expect(screen.getByText(priceNotes,{selector:"p"})).toBeVisible();
});

it("confirms stock independently, supports cancellation and shows full notes without any prices",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const user=userEvent.setup(),adopt=vi.fn();
 const candidate={rows:[
  {attributes:{颜色:"黑"},evidence:"黑",stockSuggestion:{quantity:0,evidence:"库存 0 或 5"}},
  {attributes:{颜色:"白"},evidence:"白",stockSuggestion:{quantity:5,evidence:"库存5"}},
 ]};
 const notes="黑，库存 0 或 5 尚待核对；白，库存5";
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"human_review_required",candidate}));
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),notes);
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 expect(await screen.findByLabelText("确认第 1 个 SKU 的库存数量")).not.toBeChecked();
 expect(screen.getByLabelText("确认第 2 个 SKU 的库存数量")).not.toBeChecked();
 await user.click(screen.getByText("查看本次完整原说明"));
 expect(screen.getByText(notes,{selector:"p"})).toBeVisible();
 await user.click(screen.getByLabelText("确认第 1 个 SKU 的库存数量"));
 await user.click(screen.getByLabelText("确认第 2 个 SKU 的库存数量"));
 await user.click(screen.getByLabelText("确认第 2 个 SKU 的库存数量"));
 await user.click(screen.getByRole("button",{name:"采纳到商品表单"}));
 expect(adopt).toHaveBeenCalledWith(candidate,false,[],new Set(),new Set([0]));
 expect(request).toHaveBeenCalledTimes(2);
});
