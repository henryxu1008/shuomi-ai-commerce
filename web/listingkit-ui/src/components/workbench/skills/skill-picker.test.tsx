import {expect,it,vi,afterEach} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {SkillPicker} from "./skill-picker";
import {skillRequest} from "@/lib/api/skill";
vi.mock("@/lib/api/skill",()=>({skillRequest:vi.fn(),skillError:()=>"版本不可用"}));
afterEach(()=>{cleanup();vi.clearAllMocks()});
const version={skill:{id:"11111111-1111-4111-8111-111111111111",owner:{kind:"USER",id:"alice"}},version:1,package:{name:"title-method",digest:"a".repeat(64)}};
it("retains the selected exact version and blocks confirmation after it disappears",async()=>{
 vi.mocked(skillRequest).mockResolvedValueOnce({items:[version]}).mockResolvedValueOnce({items:[]});
 const changed=vi.fn();render(<SkillPicker scope={{userId:"alice",organizationId:"org"}} onChange={changed}/>);
 fireEvent.click(screen.getByLabelText("使用技能 / 方法"));await screen.findByRole("option",{name:/title-method/});fireEvent.change(screen.getByLabelText("准确技能版本"),{target:{value:"11111111-1111-4111-8111-111111111111:1"}});
 await waitFor(()=>expect(changed).toHaveBeenLastCalledWith({skillId:version.skill.id,version:1,digest:"a".repeat(64)},true));
 fireEvent.click(screen.getByRole("button",{name:"刷新技能版本"}));await screen.findByText(/所选版本已不可用/);
 expect(changed).toHaveBeenLastCalledWith({skillId:version.skill.id,version:1,digest:"a".repeat(64)},false);
 fireEvent.click(screen.getByLabelText("使用技能 / 方法"));expect(changed).toHaveBeenLastCalledWith(undefined,true);
});

it("uses the image permission endpoint and leaves no-method preparation available",async()=>{
 vi.mocked(skillRequest).mockResolvedValue({items:[]});const changed=vi.fn();
 render(<SkillPicker scope={{userId:"alice",organizationId:"org"}} target="product-image" onChange={changed}/>);
 await waitFor(()=>expect(vi.mocked(skillRequest).mock.calls.some(([,path])=>path==="organization/image-candidates")).toBe(true));
 expect(screen.getByRole("option",{name:"不使用技能"})).toBeInTheDocument();expect(changed).toHaveBeenLastCalledWith(undefined,true);
});
