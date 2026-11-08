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
  {attributes:{颜色:"黑",尺码:"M"},evidence:"黑 M，库存明确为 0",sku:"BLK-M",stock:0},
  {attributes:{颜色:"黑",尺码:"L"},evidence:"黑 L，库存未填写",sku:"BLK-L"},
  {attributes:{颜色:"白",尺码:"M"},evidence:"白 M，库存为5",sku:"WHT-M",stock:5},
 ];
 request.mockResolvedValueOnce(prepared).mockImplementationOnce((_scope,key)=>Promise.resolve({requestKey:key.replace("/confirm",""),phase:"human_review_required",candidate:{title:"黑白T恤",rows}}));
 render(<OwnProductIntake scope={scope} member="member" images={[]} uploaded={new Set()} disabled={false} onAdopt={adopt}/>);
 await user.click(screen.getByRole("button",{name:"开始整理资料"}));
 await user.type(screen.getByLabelText("给 AI 的商品资料"),rows.map(row=>row.evidence).join("\n"));
 await user.click(screen.getByRole("button",{name:"检查生成用量"}));
 await user.click(await screen.findByRole("button",{name:"确认生成候选"}));
 const region=await screen.findByRole("region",{name:"AI 生成候选"});
 expect(region.querySelectorAll("li")).toHaveLength(3);
 expect(region.querySelectorAll("li")[0]).toHaveTextContent("库存 0");
 expect(region.querySelectorAll("li")[1]).toHaveTextContent("库存 未填写");
 expect(region.querySelectorAll("li")[2]).toHaveTextContent("库存 5");
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
