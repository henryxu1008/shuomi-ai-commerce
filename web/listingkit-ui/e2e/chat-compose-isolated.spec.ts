import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

const secret = "codex-chat-compose-synthetic-ui-only";
test.skip(process.env.CHAT_COMPOSE_UI_FIXTURE !== "1" || process.env.AUTH_SECRET !== secret, "requires the explicit isolated synthetic UI fixture");

for(const manual of [false,true])test(`creates a skill in Chat with ${manual?"manual selection":"automatic invocation"} and recovers original requests`,async({page,context,baseURL},info)=>{
 test.setTimeout(60_000);
 const origin=baseURL!,namespace=createHash("sha256").update(JSON.stringify([origin,"https://chat-compose.invalid","chat-compose-fixture"])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
 const id="11111111-1111-4111-8111-111111111111",skillId="22222222-2222-4222-8222-222222222222",root="019aaf91-67a0-7000-8000-aaaaaaaaaaaa",revisedKey="019aaf91-67a1-7000-8000-bbbbbbbbbbbb",digest="a".repeat(64),base="/api/workbench/chat/conversations",authoring=`${base}/${id}/skill-authoring`;
 const selected={version:2,digest},requirement="创建保留品牌和型号的标题技能，用 Python 合并连续空格",modification="也保留规格";
 const draft={name:"short-title",displayName:"标题整理",description:"整理标题",instructions:"保留型号。",script:{description:"合并重复空格",source:"def normalize(title):\n    return ' '.join(title.split())"}};
 const revised={...draft,instructions:"保留型号和规格。"};
 const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:"2026-11-18T00:00:00Z",UpdatedAt:"2026-11-18T00:00:00Z",businessScope:{version:"workbench-scope-v1",kind:"general"}};
 let creates=0,createKey="",initialTurn="",initialBody="",generationComplete=false,revision=false,saves=0,savePending=false,saved=false,modelGenerations=0;
 const turnKeys:string[]=[];
 const job=()=>({key:root,generationKey:revision?revisedKey:root,revision:revision?revisedKey:root,state:generationComplete?"COMPLETE":"UNKNOWN",requirement,draft:generationComplete?(revision?revised:draft):null,billing:generationComplete?"committed":"pending",savePending,savedSkillId:saved?skillId:null});
 const messages=()=>turnKeys.flatMap((key,index)=>[{ID:`00000000-0000-4000-8000-${String(index*2+1).padStart(12,"0")}`,ConversationID:id,Sequence:index*2+1,Author:"USER",Content:index?modification:requirement,CreatedAt:"2026-11-18T00:00:00Z"},{ID:`00000000-0000-4000-8000-${String(index*2+2).padStart(12,"0")}`,ConversationID:id,Sequence:index*2+2,Author:"ASSISTANT",Content:"技能草稿",kind:"AGENT_SKILL_REF",turnKey:key,CreatedAt:"2026-11-18T00:00:00Z"}]);
 await page.route("**/*",async route=>{
  const req=route.request(),url=new URL(req.url()),path=url.pathname;
  if(url.origin!==origin)return route.abort();if(!path.startsWith("/api/"))return route.continue();
  const json=(body:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(body)});
  if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,projectCenterAvailable:true,aiWorkbenchChatTurnsReady:true,aiWorkbenchChatIntentReadiness:"AVAILABLE",aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
  if(path===base+"/scopes")return json({options:[{scope:{version:"workbench-scope-v1",kind:"general"},confirmation:digest}],next:"",incomplete:false});
  if(path===base+"/skill-authoring")return json({skill:{name:"创建技能",version:3,digest:"c".repeat(64),description:"创作文字技能"},titleSkill:{name:"创建技能",...selected,description:"起草个人标题技能"},ready:true,scripts:true});
  if(path===base&&req.method()==="POST"){
   creates++;if(creates===1){createKey=req.headers()["idempotency-key"];return json({code:"OUTCOME_UNKNOWN"},503);}
   expect(req.headers()["idempotency-key"]).toBe(createKey);return json({conversation,replay:true});
  }
  if(path===`${base}/${id}`)return json({conversation,messages:messages(),proposals:[],before:""});
  if(path===`${base}/${id}/turns`){
   const key=req.headers()["idempotency-key"];
   if(!initialTurn){initialTurn=key;initialBody=req.postData()!;expect(key).not.toBe(createKey);expect(req.postDataJSON()).toEqual({content:requirement,...(manual?{creatorOnly:true}:{})});turnKeys.push(key);modelGenerations++;return json({state:"UNKNOWN",kind:"CREATE_TITLE_SKILL",skill:selected,authoring:job()});}
   if(key===initialTurn){expect(req.postData()).toBe(initialBody);generationComplete=true;return json({state:"COMPLETE",kind:"CREATE_TITLE_SKILL",skill:selected,authoring:job()});}
   expect(req.postDataJSON()).toEqual({content:modification,skill:selected,revision:{root,baseRevision:root,snapshot:draft}});
   if(!turnKeys.includes(key)){turnKeys.push(key);modelGenerations++;}revision=true;return json({state:"COMPLETE",kind:"CREATE_TITLE_SKILL",skill:selected,authoring:job()});
  }
  if(path.startsWith(`${base}/${id}/turns/`))return json({state:generationComplete?"COMPLETE":"UNKNOWN",kind:"CREATE_TITLE_SKILL",skill:selected,authoring:job()});
  if(path===authoring+"/validate")return json({valid:true});
  if(path===authoring+"/"+root+"/save-package"){
   saves++;expect(req.postDataJSON()).toEqual({revision:revisedKey,draft:{...revised,instructions:"保留已核实的型号和规格。"}});savePending=true;return json({code:"OUTCOME_UNKNOWN"},503);
  }
  if(path===authoring+"/"+root+"/recover-save"){expect(req.postData()).toBeNull();saved=true;savePending=false;return json(job());}
  return json({code:"NOT_FOUND"},404);
 });
 await page.setViewportSize(manual?{width:390,height:844}:{width:1440,height:1000});
 await page.goto("/workbench/ai/chat/new");
 const store=(await page.getByRole("button",{name:"选择店铺",exact:true}).boundingBox())!,skillButton=page.getByRole("button",{name:"选择技能",exact:true}),skill=(await skillButton.boundingBox())!;
 expect(skill.x).toBeGreaterThanOrEqual(store.x+store.width);expect(Math.abs(skill.y-store.y)).toBeLessThan(1);
 if(manual){
  await skillButton.click();
  const drawer=page.getByRole("dialog",{name:"选择技能"});
  await expect(drawer.getByRole("button",{name:"创建商品标题技能",exact:true})).toHaveCount(0);
  await drawer.getByRole("button",{name:/创建技能/}).click();
 }
 await expect(page.getByText("进入技能创作",{exact:true})).toHaveCount(0);await expect(page.getByRole("button",{name:/项目操作/})).toHaveCount(0);
 await page.getByLabel("需求",{exact:true}).fill(requirement);await page.getByRole("button",{name:"开始对话",exact:true}).click();
 await expect(page.getByRole("button",{name:"核对原创建请求",exact:true})).toBeEnabled();await page.reload();await page.getByRole("button",{name:"核对原创建请求",exact:true}).click();
 await expect(page).toHaveURL(new RegExp("/chat/"+id));await page.reload();
 expect(modelGenerations).toBe(1);await expect(page.getByLabel("技能需求",{exact:true})).toHaveCount(0);
 await page.getByRole("button",{name:"核实原生成请求",exact:true}).click();
 await expect(page.getByRole("textbox",{name:"正文",exact:true})).toHaveValue(draft.instructions);
 await page.getByRole("button",{name:"让 Chat 修改",exact:true}).click();
 await page.getByRole("textbox",{name:"需求",exact:true}).fill(modification);await page.getByRole("button",{name:"发送消息",exact:true}).click();
 const editors=page.getByRole("textbox",{name:"正文",exact:true});await expect(editors).toHaveCount(2);await expect(editors.last()).toHaveValue(revised.instructions);
 await editors.last().fill("保留已核实的型号和规格。");
 const save=page.getByRole("button",{name:"保存到我的技能",exact:true}).last();await expect(save).toBeDisabled();await expect(save).toBeEnabled();await expect(editors.last()).toHaveValue("保留已核实的型号和规格。");expect(saves).toBe(0);await save.click();
 await expect(page.getByRole("button",{name:"核实原保存",exact:true})).toBeVisible();await page.reload();
 await page.getByRole("button",{name:"核实原保存",exact:true}).last().click();await expect(page.getByRole("link",{name:"查看我的技能"}).last()).toHaveAttribute("href","/workbench/agents/skills/mine?skillId="+skillId);
 expect(saves).toBe(1);expect(modelGenerations).toBe(2);
 await expect(page.getByText("平台渠道按用量扣 AI 点数，企业自有渠道由厂商计费。模型规划不会直接修改商品。",{exact:true})).toBeVisible();
 await expect(page.getByText(/\d[\d.,]*\s*(?:AI\s*)?点数|点数金额|费用详情/)).toHaveCount(0);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.screenshot({path:info.outputPath(`chat-skill-${manual?"manual":"automatic"}.png`)});
});
test("creates a private text skill and uses it with original-message recovery", async ({page, context, baseURL}) => {
  test.setTimeout(60_000);
  const origin = baseURL!;
  const namespace = createHash("sha256").update(JSON.stringify([origin,"https://chat-compose.invalid","chat-compose-fixture"])).digest("hex").slice(0,32);
  const name = `shuomi.${namespace}.session-token`;
  const token = await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
  await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
  const conversationId = "11111111-1111-4111-8111-111111111111";
  const scope = {OrganizationID:"org",ActorID:"user"};
  const now = new Date().toISOString();
  const conversation = {ID:conversationId,Scope:scope,Title:"交接笔记",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:now,UpdatedAt:now,businessScope:{version:"workbench-scope-v1",kind:"general"}};
  const conversationPath = `/api/workbench/chat/conversations/${conversationId}`;
  const authoringPath = conversationPath + "/skill-authoring";
  const key = "019aaf91-67a0-7000-8000-aaaaaaaaaaaa";
  const v3Digest = "a".repeat(64);
  const savedSkill = {id:"22222222-2222-4222-8222-222222222222",version:1,digest:"b".repeat(64)};
  const requirement = "把交接笔记整理成待办事项";
  const input = "校对说明书；小胡负责更新进度";
  const laterInput = "周五核对物料清单";
  const oversized = "需要核对物料。".repeat(250);
  const answer = `| 事项 | 负责人 |
| --- | --- |
| 校对说明书 | 待确认 |
| 更新进度 | 小胡 |

<img src="https://chat-compose.invalid/pixel" />`;
  const draft = {name:"handoff-notes",displayName:"交接待办",description:"把交接笔记整理为待办事项。",instructions:"提取事项和负责人。"};
  let saved = false, generated = false, complete = false, messageKey = "", unknownKey = "", sends = 0, unknownReads = 0, storeRoutes = 0;
  let rejectedKey = "";
  let generationRequests = 0, generationChecks = 0, saveRequests = 0, saveChecks = 0;
  const job = () => ({key,revision:key,state:"COMPLETE",requirement,draft,billing:"usage_only",savePending:saveRequests>0&&!saved,savedSkillId:saved?savedSkill.id:null,purpose:"chat-text",...(saved?{savedSkill}:{})});
  await page.route("**/*", async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname;
    if (url.origin !== origin) return route.abort();
    if (!path.startsWith("/api/")) return route.continue();
    const json = (body:unknown, status=200) => route.fulfill({status,contentType:"application/json",body:JSON.stringify(body)});
    if (path === "/api/workbench/context") return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,projectCenterAvailable:true,aiWorkbenchChatTurnsReady:true,aiWorkbenchChatIntentReadiness:"AVAILABLE",aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"UNAVAILABLE",aiWorkbenchPlanningReadiness:"NEEDS_CONFIGURATION",organizations:[{id:"org",name:"合成企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read","workbench.project.manage"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
    if (path === "/api/workbench/chat/conversations/skill-authoring") return json({skill:{name:"创建技能",description:"创作文字技能",version:3,digest:v3Digest},titleSkill:{name:"创建技能",description:"创作商品标题技能",version:2,digest:"c".repeat(64)},ready:true,scripts:true,textReady:true});
    if (path === conversationPath) return json({conversation,proposals:[],before:"",messages:complete ? [{ID:"33333333-3333-4333-8333-333333333333",ConversationID:conversationId,Sequence:2,Author:"ASSISTANT",Content:answer,CreatedAt:now,textSkillVersion:1}] : []});
    if (path === conversationPath + "/turns") {
      expect(req.postDataJSON()).toEqual({content:requirement,creatorOnly:true});
      return json({state:"COMPLETE",kind:"CLARIFY"});
    }
    if (path === authoringPath) return json({drafts:generated?[job()]:[]});
    if (path === authoringPath + "/preview") {
      expect(req.postDataJSON()).toEqual({requirement,skillVersion:3,skillDigest:v3Digest});
      return json({key,quote:"Q".repeat(43),expiresAt:new Date(Date.now()+60_000).toISOString()});
    }
    if (path === authoringPath + "/" + key + "/confirm") {
      generationRequests++;
      expect(req.postDataJSON()).toEqual({requirement,skillVersion:3,skillDigest:v3Digest,quote:"Q".repeat(43)});
      return json({code:"OUTCOME_UNKNOWN"},503);
    }
    if (path === authoringPath + "/" + key + "/recover-generation") {
      generationChecks++;
      expect(req.postData()).toBeNull();
      generated = true;
      return json(job());
    }
    if (path === authoringPath + "/validate") return json({valid:true});
    if (path === authoringPath + "/" + key + "/save-package") {
      expect(req.postDataJSON()).toEqual({revision:key,draft:{...draft,instructions:"提取事项和负责人，缺失时标记待确认。"}});
      saveRequests++;
      return json({code:"OUTCOME_UNKNOWN"},503);
    }
    if (path === authoringPath + "/" + key + "/recover-save") {
      saveChecks++;
      expect(req.postData()).toBeNull();
      saved = true;
      return json(job());
    }
    if (path === conversationPath + "/intent") {storeRoutes++; return json({code:"INVALID_REQUEST"},400);}
    if (path === conversationPath + "/messages") {
      if (req.postDataJSON().content === oversized) {
        rejectedKey = req.headers()["idempotency-key"];
        return json({code:"TEXT_SKILL_INPUT_LIMIT",messageAccepted:false},413);
      }
      if (complete) {
        expect(req.postDataJSON()).toEqual({content:laterInput,textSkill:savedSkill});
        const sentKey = req.headers()["idempotency-key"];
        if (!unknownKey) unknownKey = sentKey;
        expect(sentKey).toBe(unknownKey);
        expect(sentKey).not.toBe(messageKey);
        unknownReads++;
        return json({state:"PLANNER_UNKNOWN"},202);
      }
      expect(req.postDataJSON()).toEqual({content: input,textSkill:savedSkill});
      sends++;
      expect(req.headers()["idempotency-key"]).not.toBe(rejectedKey);
      if (sends === 1) {messageKey = req.headers()["idempotency-key"]; return json({code:"OUTCOME_UNKNOWN"},503);}
      expect(req.headers()["idempotency-key"]).toBe(messageKey);
      complete = true; return json({state:"COMPLETE",assistantMessageId:"33333333-3333-4333-8333-333333333333"});
    }
    if (path === "/api/workbench/skills/personal") return json({items:saved?[{canEdit:true,canPublish:true,canUse:true,canCopy:false,canExport:false,version:{skill:{id:savedSkill.id,owner:{kind:"USER",id:"user"},creator:"user",headVersion:1,revision:1,createdAt:now,updatedAt:now},version:1,package:{name:draft.name,description:draft.description,displayName:draft.displayName,target:"chat-text",digest:savedSkill.digest,files:null},state:"ENABLED",visibility:"PRIVATE",allowCopy:false,allowExport:false,createdAt:now}}]:[],nextCursor:""});
    return json({code:"NOT_FOUND"},404);
  });
  await page.setViewportSize({width:390,height:844});
  await page.goto(`/workbench/ai/chat/${conversationId}`);
  await page.getByRole("button",{name:"选择技能",exact:true}).click();
  await page.getByRole("dialog",{name:"选择技能"}).getByRole("button",{name:/创建技能/}).click();
  await page.getByRole("textbox",{name:"需求",exact:true}).fill(requirement);
  await page.getByRole("button",{name:"发送消息",exact:true}).click();
  await page.reload();
  await expect(page.getByRole("textbox",{name:"需求",exact:true})).toHaveValue(requirement);
  await page.getByRole("button",{name:"用这段需求创建通用文字技能",exact:true}).click();
  await expect(page.getByLabel("技能需求")).toHaveValue(requirement);
  expect(generationRequests).toBe(0);
  await page.getByRole("button",{name:"生成草稿",exact:true}).click();
  await expect(page.getByText("本次操作暂未确认，请核实原请求。",{exact:true})).toBeVisible();
  await page.reload();
  await page.getByRole("button",{name:"选择技能",exact:true}).click();
  await page.getByRole("button",{name:"查看本会话的文字技能草稿",exact:true}).click();
  await page.getByRole("button",{name:"核实原生成请求",exact:true}).click();
  await page.getByRole("textbox",{name:"正文",exact:true}).fill("提取事项和负责人，缺失时标记待确认。");
  await page.getByRole("button",{name:"保存并在 Chat 启用",exact:true}).click();
  await expect(page.getByRole("button",{name:"核实原保存",exact:true})).toBeEnabled();
  await page.reload();
  await page.getByRole("button",{name:"选择技能",exact:true}).click();
  await page.getByRole("button",{name:"查看本会话的文字技能草稿",exact:true}).click();
  await page.getByRole("button",{name:draft.displayName,exact:true}).click();
  await page.getByRole("button",{name:"核实原保存",exact:true}).click();
  await page.getByRole("button",{name:"在 Chat 中使用",exact:true}).click();
  expect([generationRequests,generationChecks,saveRequests,saveChecks]).toEqual([1,1,1,1]);
  const text = page.getByRole("textbox",{name:"要处理的文字",exact:true});
  await text.fill(oversized);
  await page.getByRole("button",{name:"发送消息",exact:true}).click();
  await expect(page.getByText(/超过了当前模型的输入上限/)).toBeVisible();
  await expect(text).toHaveValue(oversized);
  await expect(text).toBeEnabled();
  await text.fill(input);
  await page.getByRole("button",{name:"发送消息",exact:true}).click();
  await expect(page.getByRole("button",{name:"核对原消息"})).toBeEnabled();
  await page.reload();
  await expect(page.getByRole("textbox",{name:"要处理的文字",exact:true})).toHaveValue(input);
  await page.getByRole("button",{name:"核对原消息"}).click();
  await expect(page.getByText("使用文字技能 v1")).toBeVisible();
  const table = page.getByRole("table");
  await expect(table.getByRole("columnheader",{name:"事项"})).toBeVisible();
  await expect(table.getByRole("row",{name:"校对说明书 待确认"})).toBeVisible();
  await expect(page.locator('img[src="https://chat-compose.invalid/pixel"]')).toHaveCount(0);
  expect(sends).toBe(2);
  await page.getByRole("button",{name:"选择技能",exact:true}).click();
  await page.getByRole("dialog",{name:"选择技能"}).getByRole("button",{name:draft.displayName}).click();
  await page.getByRole("textbox",{name:"要处理的文字",exact:true}).fill(laterInput);
  await page.getByRole("button",{name:"发送消息",exact:true}).click();
  await expect(page.getByText(/核对不会重新发送/)).toBeVisible();
  await page.reload();
  await expect(page.getByRole("textbox",{name:"要处理的文字",exact:true})).toBeDisabled();
  await page.getByRole("button",{name:"核对原消息"}).click();
  await expect(page.getByText(/核对不会重新发送/)).toBeVisible();
  expect(unknownReads).toBe(2);
  expect(storeRoutes).toBe(0);
});
test("retains the input through the original create retry without dispatching a message", async ({page,context,baseURL},info) => {
  const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
  await page.addInitScript(() => {
    const target = window as unknown as {SpeechRecognition: unknown; webkitSpeechRecognition: unknown; voiceFixture: {starts: number; current?: {onresult?: (event: unknown) => void}}};
    target.voiceFixture = {starts:0};
    class FakeSpeechRecognition {
      onresult?: (event: unknown) => void;
      onend?: () => void;
      constructor() { target.voiceFixture.current = this; }
      start() { target.voiceFixture.starts++; }
      stop() { setTimeout(() => this.onend?.(), 0); }
      abort() {}
    }
    target.SpeechRecognition = FakeSpeechRecognition;
    target.webkitSpeechRecognition = FakeSpeechRecognition;
  });
  const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
  const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
  await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
  const id="11111111-1111-4111-8111-111111111111",confirmation="a".repeat(64);
  const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:"2026-11-12T00:00:00Z",UpdatedAt:"2026-11-12T00:00:00Z",businessScope:{version:"workbench-scope-v1",kind:"general"}};
  let creates=0,firstKey="",firstBody="",messages=0;
  await page.route("**/*",async route=>{
    const req=route.request(),url=new URL(req.url()); if(url.origin!==origin)return route.abort();
    if(!url.pathname.startsWith("/api/"))return route.continue();
    const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
    if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
    if(path==="/api/workbench/chat/conversations/scopes") {const kind=url.searchParams.get("kind")!,projectId=url.searchParams.get("projectId"),storeId=url.searchParams.get("storeId");return json({options:[{scope:{version:"workbench-scope-v1",kind,...(projectId?{projectId}:{}),...(storeId?{storeId}:{})},confirmation,...(projectId?{projectName:"合成项目",storeName:"合成关联店铺"}:storeId?{storeName:"合成店铺"}:{})}],next:"",incomplete:false});}
    if(path==="/api/workbench/chat/conversations"&&req.method()==="POST"){creates++;if(creates===1){firstKey=req.headers()["idempotency-key"];firstBody=req.postData()!;return json({code:"OUTCOME_UNKNOWN"},503);}expect(req.headers()["idempotency-key"]).toBe(firstKey);expect(req.postData()).toBe(firstBody);return json({conversation,replay:true});}
    if(path===`/api/workbench/chat/conversations/${id}`&&req.method()==="GET")return json({conversation,messages:[],proposals:[],before:""});
    if(path.endsWith("/messages")){messages++;return json({code:"NOT_FOUND"},404);}
    return json({code:"NOT_FOUND"},404);
  });
  await page.goto("/workbench/ai/chat/new");
  for (const viewport of [{width:1440,height:900},{width:390,height:844}]) {
    await page.setViewportSize(viewport);
    for (const theme of ["light", "dark"]) {
      await page.evaluate(value => localStorage.setItem("listingkit-theme", value), theme);
      await page.reload();
      await expect(page.locator("html")).toHaveClass(new RegExp(theme));
      await expect(page.getByLabel("需求")).toBeVisible();
      await expect(page.getByRole("button",{name:"开始对话",exact:true})).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      await page.screenshot({path:info.outputPath(`chat-compose-${viewport.width}-${theme}.png`)});
    }
  }
  await page.getByLabel("需求").fill("【合成测试】请整理商品描述");
  expect(await page.evaluate(() => (window as unknown as {voiceFixture:{starts:number}}).voiceFixture.starts)).toBe(0);
  await page.getByRole("button",{name:"开始语音输入"}).click();
  await page.evaluate(() => {
    const fixture = (window as unknown as {voiceFixture:{current:{onresult:(event:unknown)=>void}}}).voiceFixture;
    const item = Object.assign([{transcript:"，补充卖点"}], {isFinal:true});
    fixture.current.onresult({resultIndex:0,results:[item]});
  });
  await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述，补充卖点");
  await page.getByRole("button",{name:"停止语音输入"}).click();
  await page.getByLabel("需求").fill("【合成测试】请整理商品描述");
  await page.screenshot({path:info.outputPath("chat-compose-desktop.png")});
  await page.getByRole("button",{name:"开始对话",exact:true}).click();
  await expect(page.getByRole("button",{name:"核对原创建请求"})).toBeVisible();
  await page.reload(); await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述");
  await page.getByRole("button",{name:"核对原创建请求"}).click();
  await page.waitForURL(`**/workbench/ai/chat/${id}`);
  await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述");
  expect(creates).toBe(2); expect(messages).toBe(0);
  await page.setViewportSize({width:390,height:844});
  await page.goto("/workbench/ai/chat/new?projectId=fixture-project");
  await expect(page.getByText("工作背景：合成项目 · 合成关联店铺")).toBeVisible();
  await page.screenshot({path:info.outputPath("chat-compose-project-mobile.png")});
  await page.goto("/workbench/ai/chat/new?storeId=fixture-store");
  await expect(page.getByText("工作背景：合成店铺")).toBeVisible();
  expect(messages).toBe(0); expect(creates).toBe(2);
});

test("routes a store question once and hides cached totals after revoked access",async({page,context,baseURL})=>{
 const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
 const id="11111111-1111-4111-8111-111111111111",messageId="22222222-2222-4222-8222-222222222222",stamp="2026-11-12T00:00:00Z";
 const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:stamp,UpdatedAt:stamp,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 let intentCalls=0,modelKey="",revoked=false,answered=false;
 await page.route("**/*",async route=>{
  const req=route.request(),url=new URL(req.url());if(url.origin!==origin)return route.abort();if(!url.pathname.startsWith("/api/"))return route.continue();
  const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
  if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",aiWorkbenchStoreRouteReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read","workbench.store.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
  if(path===`/api/workbench/chat/conversations/${id}`&&req.method()==="GET"){
   const ref={ID:messageId,ConversationID:id,Sequence:2,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",...(revoked?{storeOverviewUnavailable:true}:{currentStoreOverview:{total:3,platforms:{shein:2,temu:1},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}})};
   return json({conversation,messages:answered?[ref]:[],proposals:[],before:""});
  }
  if(path===`/api/workbench/chat/conversations/${id}/intent`&&req.method()==="POST"){
   intentCalls++;if(intentCalls===1){modelKey=req.headers()["idempotency-key"];return json({code:"OUTCOME_UNKNOWN"},503)}
   expect(req.headers()["idempotency-key"]).toBe(modelKey);expect(req.postDataJSON()).toEqual({content:"我有几个店铺？"});answered=true;return json({state:"COMPLETE"});
  }
  return json({code:"NOT_FOUND"},404);
 });
 await page.goto(`/workbench/ai/chat/${id}`);await page.getByLabel("需求").fill("我有几个店铺？");expect(intentCalls).toBe(0);
 await page.getByRole("button",{name:"发送消息"}).click();await page.getByRole("button",{name:"重试同一消息"}).click();
 await expect(page.getByText("当前可查看 3 家店铺。")).toBeVisible();expect(intentCalls).toBe(2);
 revoked=true;await page.evaluate(()=>{window.dispatchEvent(new Event("blur"));window.dispatchEvent(new Event("focus"));});
 await expect(page.getByText("当前可查看 3 家店铺。")).toHaveCount(0);await expect(page.getByText("当前无权查看或店铺数据暂不可用。")).toBeVisible();
});

test("keeps project operations out of Chat and resumes its rebound conversation",async({page,context,baseURL})=>{
  const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
  const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32);
  const name="shuomi."+namespace+".session-token";
  const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
  await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
  const projectId="11111111-1111-4111-8111-111111111111";
  const oldId="22222222-2222-4222-8222-222222222222",newId="33333333-3333-4333-8333-333333333333";
  const storeA="44444444-4444-4444-8444-444444444444",storeB="55555555-5555-4555-8555-555555555555";
  const confirmation="a".repeat(64),stamp="2026-11-17T00:00:00Z";
  const conversation=(id:string,storeId:string)=>({
    ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,
    MetadataRevision:1,NextSequence:1,CreatedAt:stamp,UpdatedAt:stamp,businessScope:{version:"workbench-scope-v1",kind:"project",projectId,storeId},
  });
  let planningReady=false,projectsSaved=0,conversationsCreated=0,modelCalls=0;
  await page.route("**/*",async route=>{
    const req=route.request(),url=new URL(req.url());
    if(url.origin!==origin)return route.abort();
    if(!url.pathname.startsWith("/api/"))return route.continue();
    const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)});
    const path=url.pathname;
    if(path==="/api/workbench/context")return json({
      user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,
      aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",projectCenterAvailable:true,
      aiWorkbenchExecutionReadiness:planningReady?"AVAILABLE":"NEEDS_CONFIGURATION",aiWorkbenchPlanningReadiness:planningReady?"AVAILABLE":"NEEDS_CONFIGURATION",
      organizations:[{id:"org",name:"合成验收企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read","workbench.project.read","workbench.project.manage","workbench.store.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}],
    });
    if(path==="/api/workbench/chat/conversations/scopes"){
      const kind=url.searchParams.get("kind");
      const scope={version:"workbench-scope-v1",kind};
      if(kind==="project")return json({options:[{scope:{...scope,projectId,storeId:storeB},confirmation,projectName:"合成项目",storeName:"乙店"}],next:"",incomplete:false});
      if(kind==="store"){
        const ids=url.searchParams.has("storeId")?[url.searchParams.get("storeId")!]:[storeA,storeB];
        return json({options:ids.map(storeId=>({scope:{...scope,storeId},confirmation,storeName:storeId===storeA?"甲店":"乙店"})),next:"",incomplete:false});
      }
      return json({options:[{scope,confirmation}],next:"",incomplete:false});
    }
    if(path==="/api/workbench/projects"&&req.method()==="POST"){
      projectsSaved++;
      expect(req.postDataJSON()).toMatchObject({title:"对话新建项目",goal:"准备商品英文描述",storeId:storeA});
      return json({id:projectId,revision:1,replayed:false});
    }
    if(path==="/api/workbench/chat/conversations"&&req.method()==="POST"){
      conversationsCreated++;
      expect(req.postDataJSON()).toEqual({scopeSelection:{kind:"project",projectId},scopeConfirmation:confirmation});
      return json({conversation:conversation(newId,storeB),replay:false});
    }
    if(path==="/api/workbench/chat/conversations/"+oldId)return json({conversation:conversation(oldId,storeA),messages:[],proposals:[],before:""});
    if(path==="/api/workbench/chat/conversations/"+newId)return json({conversation:conversation(newId,storeB),messages:[],proposals:[],before:""});
    if(path.endsWith("/messages")){modelCalls++;return json({code:"NOT_FOUND"},404);}
    if(path.endsWith("/connection")){
      return json({platform:"shein",attemptId:"",state:"verified",connectionStatus:"connected",version:1,observedAt:stamp});
    }
    return json({code:"NOT_FOUND"},404);
  });
  await page.setViewportSize({width:390,height:844});
  await page.goto("/workbench/ai/chat/new");
  const storeTrigger = page.getByRole("button",{name:"选择店铺",exact:true});
  await expect(storeTrigger).toBeVisible();
  await expect(page.getByRole("button",{name:/项目操作/})).toHaveCount(0);
  await expect(page.getByRole("button",{name:"创建项目",exact:true})).toHaveCount(0);
  expect(projectsSaved).toBe(0);
  expect(conversationsCreated).toBe(0);
  planningReady=true;
  await page.goto("/workbench/ai/chat/"+oldId+"?projectTask=1");
  await expect(page.getByRole("button",{name:/项目操作/})).toHaveCount(0);
  await page.getByLabel("需求").fill("这是尚未提交的草稿");
  await page.reload();
  await expect(page.getByLabel("需求")).toHaveValue("这是尚未提交的草稿");
  await page.getByRole("button",{name:"在新对话中继续"}).click();
  await expect(page.getByLabel("需求")).toHaveValue("这是尚未提交的草稿");
  await expect(page.getByText("工作背景：合成项目 · 乙店")).toBeVisible();
  await page.getByRole("button",{name:"开启项目会话"}).click();
  await page.waitForURL(url=>url.pathname==="/workbench/ai/chat/"+newId && url.searchParams.get("projectTask")==="1");
  await expect(page.getByLabel("需求")).toHaveValue("这是尚未提交的草稿");
  await expect(page.getByText("项目关联店铺：乙店")).toBeVisible();
  await expect(page.getByText("本次任务店铺：未选择")).toBeVisible();
  await page.getByRole("button",{name:"选择店铺",exact:true}).click();
  await page.getByRole("button",{name:"乙店",exact:true}).click();
  await expect(page.getByText("本次任务店铺：乙店")).toBeVisible();
  expect(conversationsCreated).toBe(1);
  expect(modelCalls).toBe(0);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});

test("sends two ordered images and recovers the original request after reload", async ({page,context,baseURL})=>{
  test.setTimeout(60_000);
  const origin=baseURL!;
  const namespace=createHash("sha256").update(JSON.stringify([origin,"https://chat-compose.invalid","chat-compose-fixture"])).digest("hex").slice(0,32);
  const name=`shuomi.${namespace}.session-token`;
  const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-images",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
  await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
  const id="11111111-1111-4111-8111-111111111111", questionId="22222222-2222-4222-8222-222222222222", answerId="33333333-3333-4333-8333-333333333333";
  const base="/api/workbench/chat/conversations", detail=`${base}/${id}`;
  const question="按顺序说出两张商品图的颜色";
  const pictures=[
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC",
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGNgYPgPAAEDAQAIicLsAAAAAElFTkSuQmCC",
  ];
  const date="2026-11-18T00:00:00Z",businessScope={version:"workbench-scope-v1",kind:"general"};
  const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:date,UpdatedAt:date,businessScope};
  let firstKey="", firstBody="", requests=0, creates=0, complete=false;
  await page.route("**/*",async route=>{
    const req=route.request(),url=new URL(req.url()),path=url.pathname;
    if(url.origin!==origin)return route.abort();
    if(!path.startsWith("/api/"))return route.continue();
    const json=(body:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(body)});
    if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,projectCenterAvailable:true,aiWorkbenchChatTurnsReady:true,aiWorkbenchChatIntentReadiness:"AVAILABLE",aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
    if(path===base+"/scopes")return json({options:[{scope:businessScope,confirmation:"a".repeat(64)}],next:"",incomplete:false});
    if(path===base+"/image-questions")return json({available:true});
    if(path===base&&req.method()==="POST"){creates++;return json({conversation,replay:false});}
    if(path===detail)return json({conversation,messages:complete?[
      {ID:questionId,ConversationID:id,Sequence:1,Author:"USER",Content:question,CreatedAt:date,imageCount:2},
      {ID:answerId,ConversationID:id,Sequence:2,Author:"ASSISTANT",Content:"图 1 是红色，图 2 是蓝色。",CreatedAt:date},
    ]:[],proposals:[],before:""});
    if(path===detail+"/image-questions"){
      requests++;
      const key=req.headers()["idempotency-key"],body=req.postData()!;
      expect(req.postDataJSON()).toEqual({content:question,images:pictures});
      if(requests===1){firstKey=key;firstBody=body;return json({code:"OUTCOME_UNKNOWN"},503);}
      expect(key).toBe(firstKey);
      expect(body).toBe(firstBody);
      complete=true;
      return json({state:"COMPLETE",userMessageId:questionId,sourceSequence:1,assistantMessageId:answerId});
    }
    const attachmentBase=detail+`/messages/${questionId}/images/`;
    if(path.startsWith(attachmentBase)){
      const picture=pictures[Number(path.slice(attachmentBase.length))];
      return picture?route.fulfill({status:200,contentType:"image/png",body:Buffer.from(picture,"base64")}):json({code:"NOT_FOUND"},404);
    }
    if(req.method()==="POST")throw new Error(`unexpected Chat mutation ${path}`);
    return json({code:"NOT_FOUND"},404);
  });
  await page.setViewportSize({width:390,height:844});
  await page.goto("/workbench/ai/chat/new");
  await expect(page.getByRole("button",{name:"选择图片"})).toBeEnabled();
  const images=page.locator('input[type="file"][multiple]');
  await images.setInputFiles([
    {name:"red.png",mimeType:"image/png",buffer:Buffer.from(pictures[0],"base64")},
    {name:"blue.png",mimeType:"image/png",buffer:Buffer.from(pictures[1],"base64")},
  ]);
  await expect(page.getByRole("img",{name:"图 2"})).toBeVisible();
  await page.getByRole("textbox",{name:"需求",exact:true}).fill(question);
  await page.getByRole("button",{name:"开始对话",exact:true}).click();
  await expect(page).toHaveURL(new RegExp("/chat/"+id));
  await expect(page.getByRole("img",{name:"图 2"})).toBeVisible();
  expect(requests).toBe(0);
  await page.getByRole("button",{name:"发送消息",exact:true}).click();
  await expect(page.getByRole("button",{name:"核实原图片提问",exact:true})).toBeEnabled();
  await page.reload();
  await expect(page.getByRole("textbox",{name:"需求",exact:true})).toHaveValue(question);
  await expect(page.getByRole("textbox",{name:"需求",exact:true})).toBeDisabled();
  await page.getByRole("button",{name:"核实原图片提问",exact:true}).click();
  await expect(page.getByText("图 1 是红色，图 2 是蓝色。")).toBeVisible();
  await expect(page.getByRole("img",{name:"图 1"})).toBeVisible();
  await expect(page.getByRole("img",{name:"图 2"})).toBeVisible();
  expect(creates).toBe(1);
  expect(requests).toBe(2);
});
