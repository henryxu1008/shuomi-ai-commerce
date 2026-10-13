import {act,cleanup,renderHook,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {z} from "zod";
import {useConfigurationRead} from "./agent-configuration-hooks";
const state=vi.hoisted(()=>({read:vi.fn()}));
vi.mock("@/lib/api/agent-configuration",async()=>({...await vi.importActual<object>("@/lib/api/agent-configuration"),configurationRequest:state.read}));
afterEach(()=>{cleanup();state.read.mockReset()});
it("hides the previous enterprise mode immediately while the same path is read in the new scope",async()=>{
 const schema=z.object({mode:z.string()});let complete:(value:{mode:string})=>void=()=>{};
 state.read.mockResolvedValueOnce({mode:"full_access"}).mockImplementationOnce(()=>new Promise(resolve=>{complete=resolve}));
 const hook=renderHook(({organizationId})=>useConfigurationRead({userId:"actor",organizationId},"execution-modes/registered_agent/product.title.agent",schema),{initialProps:{organizationId:"A"}});
 await waitFor(()=>expect(hook.result.current.data?.mode).toBe("full_access"));hook.rerender({organizationId:"B"});
 expect(hook.result.current.data).toBeUndefined();expect(state.read.mock.calls[1][0]).toEqual({userId:"actor",organizationId:"B"});
 await act(async()=>complete({mode:"read_only"}));expect(hook.result.current.data?.mode).toBe("read_only");
});
