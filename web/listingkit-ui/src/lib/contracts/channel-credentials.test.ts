import {expect,it} from "vitest";
import {credentialInput,credentialPending} from "./channel-credentials";

it("persists only original scoped references and rejects every secret-bearing pending shape",()=>{
 const pending={key:"c3b3f136-ed39-4db9-965f-6eab3f8f522b",userId:"manager",organizationId:"org-a",connectionId:"shared",expectedRevision:"9007199254740993",action:"set-key"};
 expect(credentialPending.parse(pending).expectedRevision).toBe("9007199254740993");
 for(const extra of [{apiKey:"synthetic-secret"},{input:{apiKey:"synthetic-secret"}},{body:"synthetic-secret"},{writerDatabase:{}}])expect(credentialPending.safeParse({...pending,...extra}).success).toBe(false);
});
it("bounds actual UTF8 Key bytes and keeps disable commands secret-free",()=>{
 expect(credentialInput.safeParse({action:"set-key",expectedRevision:"0",account:"account",apiKey:"界".repeat(1366)}).success).toBe(false);
 expect(credentialInput.safeParse({action:"disable",expectedRevision:"1",apiKey:"synthetic-secret"}).success).toBe(false);
 expect(credentialInput.safeParse({action:"set-key",expectedRevision:"0",apiKey:" synthetic-secret"}).success).toBe(false);
});

it("keeps the original human account label contract rather than treating it as a model identifier",()=>{
 expect(credentialInput.safeParse({action:"set-key",expectedRevision:"0",account:"企业 A 共用账户",apiKey:"synthetic"}).success).toBe(true);
 expect(credentialInput.safeParse({action:"set-key",expectedRevision:"0",account:"界".repeat(43),apiKey:"synthetic"}).success).toBe(false);
});
