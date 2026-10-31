import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreOfficialConnection } from "./store-official-connection";
import type { WorkbenchStore } from "@/lib/api/workbench-stores";
const state = vi.hoisted(() => ({
  read: vi.fn(),
  begin: vi.fn(),
	remember: vi.fn(),
	applications:vi.fn(),
	install:vi.fn(),
	refresh:vi.fn(),
}));
vi.mock("@/lib/api/store-connection", async (original) => ({
  ...(await original<typeof import("@/lib/api/store-connection")>()),
  getStoreConnection: state.read,
  beginStoreConnection: state.begin,
	rememberStoreAuthorization: state.remember,
	getStoreApplications:state.applications,
	installStoreToken:state.install,
	refreshStoreConnection:state.refresh,
}));
vi.mock("./store-service-actions", () => ({ StoreServiceActions: () => null }));
const store = {
  id: "11111111-1111-4111-8111-11111111111a",
  version: 2,
  recordStatus: "active",
} as WorkbenchStore;
const client = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
function tree(org = "org-1",platform:WorkbenchStore["platform"]="shein",canWrite=true) {
  return (
    <QueryClientProvider client={client}>
      <StoreOfficialConnection
        store={{...store,platform}}
        scope={{ expectedUserId: "actor", expectedOrganizationId: org }}
        canWrite={canWrite}
        administrator
        onChanged={vi.fn()}
      />
    </QueryClientProvider>
  );
}
beforeEach(() => {
  client.clear();
  state.remember.mockReset();
	state.begin.mockReset();
	state.install.mockReset();
	state.refresh.mockReset();
	state.applications.mockResolvedValue([{appId:"self-app",revision:"v1~self_operated",type:"self_operated"}]);
  state.read.mockResolvedValue({
    connectionStatus: "disconnected",
    attemptId: null,
  });
});
it.each(["tiktokshop","walmart"] as const)("explicitly refreshes the current %s generation",async(platform)=>{
 const details=platform==="tiktokshop"?{tiktok:{grantGeneration:7}}:{walmart:{generation:7}};
 state.read.mockResolvedValue({platform,connectionStatus:"expired",state:"bound",...details});
 state.refresh.mockResolvedValue({});
 const view=render(tree("org-1",platform));
 await userEvent.click(await screen.findByRole("button",{name:"刷新官方授权"}));
 expect(state.refresh).toHaveBeenCalledWith({expectedUserId:"actor",expectedOrganizationId:"org-1"},store.id,7,expect.any(AbortSignal));
 view.rerender(tree("org-1",platform,false));
 expect(screen.queryByRole("button",{name:"刷新官方授权"})).toBeNull();
});
it.each(["refresh_unknown","refresh_dispatched","unknown"])("does not offer TikTok refresh in %s",async(stateName)=>{
 state.read.mockResolvedValue({platform:"tiktokshop",connectionStatus:"unavailable",state:stateName,tiktok:{grantGeneration:7}});
 render(tree("org-1","tiktokshop"));
 await screen.findByText("连接服务未配置或暂不可用");
 expect(screen.queryByRole("button",{name:"刷新官方授权"})).toBeNull();
 expect(state.refresh).not.toHaveBeenCalled();
});
it("installs a TEMU token only for the original attempt and clears the private input",async()=>{
 const attemptId="22222222-2222-4222-8222-22222222222b";
 state.applications.mockResolvedValue([{platform:"temu",identityPartition:"CN",appId:"temu-app",revision:"v1",authorizationMethod:"manual_token",supportedModes:6}]);
 state.begin.mockResolvedValue({attemptId,expiresAt:"2026-10-12T03:00:00Z",authorizationMethod:"manual_token",authorizationUrl:""});
 state.read.mockResolvedValue({platform:"temu",connectionStatus:"disconnected",attemptId:"",state:"disconnected"});
 state.install.mockResolvedValue({platform:"temu",connectionStatus:"connected",attemptId,state:"verified"});
 render(tree("org-1","temu"));
 await screen.findByText("未连接");
 state.read.mockResolvedValue({platform:"temu",identityPartition:"CN",authorizationMethod:"manual_token",connectionStatus:"disconnected",attemptId,state:"awaiting_consent"});
 await userEvent.click(await screen.findByRole("button",{name:"开始连接"}));
 const token=await screen.findByLabelText("TEMU 授权 token");
 await userEvent.type(token,"synthetic-token");
 await userEvent.click(screen.getByRole("button",{name:"验证并连接"}));
 expect(state.install).toHaveBeenCalledWith({expectedUserId:"actor",expectedOrganizationId:"org-1"},store.id,{attemptId,accessToken:"synthetic-token"},expect.any(AbortSignal));
 expect(token).toHaveValue("");
 expect(state.remember).not.toHaveBeenCalled();
 expect(sessionStorage.getItem("listingkit.shein.pending")).toBeNull();
});
afterEach(cleanup);
it.each(["switch","unmount"])("clears an unsubmitted TEMU private input after %s",async(mode)=>{
 state.applications.mockResolvedValue([{platform:"temu",identityPartition:"CN",appId:"temu-app",revision:"v1",authorizationMethod:"manual_token",supportedModes:6}]);
 state.read.mockResolvedValue({platform:"temu",identityPartition:"CN",authorizationMethod:"manual_token",connectionStatus:"disconnected",attemptId:"22222222-2222-4222-8222-22222222222b",state:"awaiting_consent"});
 const view=render(tree("org-1","temu"));
 const input=await screen.findByLabelText("TEMU 授权 token");
 await userEvent.type(input,"unsubmitted-private-token");
 if(mode==="switch")view.rerender(tree("org-2","temu")); else view.unmount();
 expect(input).toHaveValue("");
 expect(state.install).not.toHaveBeenCalled();
});
it.each(["switch", "unmount"])(
  "does not persist or navigate a late begin result after %s",
  async (mode) => {
    let resolve!: (value: unknown) => void;
    state.begin.mockImplementationOnce(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const view = render(tree());
    await screen.findByText("未连接");
    await userEvent.click(screen.getByRole("button", { name: "前往官方授权" }));
    if (mode === "switch") view.rerender(tree("org-2"));
    else view.unmount();
    await act(async () => {
      resolve({
        attemptId: "22222222-2222-4222-8222-22222222222b",
        expiresAt: "2026-09-29T00:00:00Z",
        authorizationUrl: "https://open.sheincorp.com/authorization",
      });
    });
    expect(state.remember).not.toHaveBeenCalled();
  },
);
it("requires a configured application choice and sends the chosen managed application",async()=>{
	state.applications.mockResolvedValue([
		{appId:"self-app",revision:"v1~self_operated",type:"self_operated"},
		{appId:"semi-app",revision:"v1~semi_managed",type:"semi_managed"},
		{appId:"full-app",revision:"v1~fully_managed",type:"fully_managed"},
	]);
	state.begin.mockRejectedValue(new Error("controlled fixture"));
	render(tree());
	const choices=await screen.findByRole("combobox",{name:"官方应用类型"});
	await screen.findByRole("option",{name:"全托管 · full-app"});
	expect(screen.getByRole("button",{name:"前往官方授权"})).toBeDisabled();
	await userEvent.selectOptions(choices,"full-app");
	await userEvent.click(screen.getByRole("button",{name:"前往官方授权"}));
	expect(state.begin).toHaveBeenCalledWith({expectedUserId:"actor",expectedOrganizationId:"org-1"},store.id,store.version,expect.any(String),"full-app",undefined);
});
