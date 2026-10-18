import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ChatTaskSelection} from "./chat-task-selection";
const read=vi.hoisted(()=>vi.fn());
vi.mock("@/lib/api/product-collection",()=>({listCollectionBatches:read,listCollectionItems:read}));
afterEach(()=>{cleanup();vi.clearAllMocks()});
it("does not expose cached product names after permissions change for the same actor and organization",async()=>{
 const client=new QueryClient({defaultOptions:{queries:{retry:false,staleTime:30000}}});
 const old={items:[{id:"11111111-1111-4111-8111-111111111111",name:"Private imported products",count:1}],total:1};
 client.setQueryData(["pod","actor","org","collections","batches",undefined],old);
 client.setQueryData(["pod","actor","org","collections","old-role","batches",undefined],old);
 read.mockRejectedValue(new Error("FORBIDDEN"));
 render(<QueryClientProvider client={client}><ChatTaskSelection scope={{userId:"actor",organizationId:"org"}} authorizationKey="revoked-role" subject={null} parameters={{targetLanguage:"en-US",description:true}} onSubject={vi.fn()} onParameters={vi.fn()} disabled={false}/></QueryClientProvider>);
 expect(screen.queryByRole("option",{name:/Private imported products/})).toBeNull();
 await screen.findByText("无法读取我的数据");
 expect(screen.queryByRole("option",{name:/Private imported products/})).toBeNull();
 await waitFor(()=>expect(read).toHaveBeenCalled());client.clear();
});
