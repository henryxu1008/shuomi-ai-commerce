import {expect,it,beforeEach,afterEach,vi} from "vitest";
import {parseSkillLaunch,skillLaunchHref,rememberChatSkillLocation,readChatSkillLocation} from "./skill-launch";
const launch={userId:"actor",organizationId:"org",target:"product-title" as const,selection:{skillId:"11111111-1111-4111-8111-111111111111",version:2,digest:"a".repeat(64)},chatReturn:"/workbench/ai/chat/new?projectId=22222222-2222-4222-8222-222222222222&projectTask=1"};
const values=new Map<string,string>();beforeEach(()=>{values.clear();vi.stubGlobal("sessionStorage",{getItem:(k:string)=>values.get(k)??null,setItem:(k:string,v:string)=>values.set(k,v)});});afterEach(()=>vi.unstubAllGlobals());
it("carries a scoped Chat return path without granting a different skill version",()=>{
 expect(parseSkillLaunch(new URL(skillLaunchHref("/workbench/agents/skills/official/use",launch),"https://test").searchParams)).toEqual(launch);
 for(const path of ["https://evil.test/","//evil.test/",String.raw`/\evil.test/workbench/ai/chat/new`,String.raw`/\[broken/workbench/ai/chat/new`,"/workbench/agents/skills/mine","/workbench/ai/chat/new?evil=1","/workbench/ai/chat/new?projectId=x"]){const p=new URL(skillLaunchHref("/",launch),"https://test").searchParams;p.set("skill_chat",path);expect(parseSkillLaunch(p)).toEqual({invalid:true})}
});
it("remembers only the original exact title request URL in the same Chat and browser scope",()=>{
 const path=skillLaunchHref("/workbench/data/mine/own/33333333-3333-4333-8333-333333333333/optimize?agent_key=44444444-4444-4444-8444-444444444444&agent_actor=actor&agent_org=org",launch);
 rememberChatSkillLocation(launch,path);expect(readChatSkillLocation(launch)).toBe(path);
 expect(readChatSkillLocation({...launch,userId:"other"})).toBeUndefined();expect(readChatSkillLocation({...launch,chatReturn:"/workbench/ai/chat/55555555-5555-4555-8555-555555555555"})).toBeUndefined();
 expect(()=>rememberChatSkillLocation(launch,path.replace("skill_version=2","skill_version=3"))).toThrow();
 expect(()=>rememberChatSkillLocation(launch,"https://evil.test/")).toThrow();expect(()=>rememberChatSkillLocation(launch,String.raw`/\evil.test`+path)).toThrow();expect(readChatSkillLocation(launch)).toBe(path);
});
