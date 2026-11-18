import {expect,it} from "vitest";
import {aiWorkbenchPath,aiChatTurnBody,parseAIWorkbenchResponse} from "./ai-workbench";
const chat="550e8400-e29b-41d4-a716-446655440000",key="11111111-1111-4111-8111-111111111111";
it("keeps automatic and exact manual selection in the same Chat turn contract",()=>{
 expect(aiChatTurnBody.parse({content:"创建标题技能"})).toEqual({content:"创建标题技能"});
 expect(aiChatTurnBody.safeParse({content:"创建",skill:{version:2,digest:"a".repeat(64)}}).success).toBe(true);
 expect(aiChatTurnBody.safeParse({content:"创建",skill:{version:3,digest:"a".repeat(64)}}).success).toBe(false);
 expect(aiWorkbenchPath("POST",["chat","conversations",chat,"turns"])).toBe("chat-turn-submit");
 expect(aiWorkbenchPath("GET",["chat","conversations",chat,"turns",key])).toBe("chat-turn-read");
 expect(aiWorkbenchPath("POST",["chat","conversations",chat,"turns",key,"recover"])).toBe("chat-turn-recover");
 expect(parseAIWorkbenchResponse("chat-turn-read",200,{state:"PROCESSING",kind:"CREATE_TITLE_SKILL"})).toBeTruthy();
 expect(parseAIWorkbenchResponse("chat-turn-read",200,{state:"COMPLETE",kind:"CREATE_TITLE_SKILL"})).toBeNull();
});
