import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import {test,expect} from "@playwright/test";

const secret="codex-own-intake-isolated-browser-only";
const backend=process.env.OWN_INTAKE_BROWSER_BACKEND;
test.skip(process.env.OWN_INTAKE_BROWSER!=="1"||process.env.AUTH_SECRET!==secret||!backend,"requires the isolated Go/PostgreSQL fixture");

test("uploads, generates and adopts exactly three SKUs, saves, then refreshes the details",async({page,context,baseURL},info)=>{
 test.setTimeout(120000);
 const origin=baseURL!,issuer="https://own-intake-ui.invalid",client="own-intake-fixture";
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32);
 const name=`shuomi.${namespace}.session-token`;
 const value=await encode({secret,salt:name,maxAge:600,token:{accessToken:"admin",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"B",userId:"admin",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value,url:origin,httpOnly:true,sameSite:"Lax"},{name:"shuomi_effective_organization",value:"B",url:origin,httpOnly:true,sameSite:"Lax"}]);
 const permissions=["workbench.collection.read","workbench.collection.manage","workbench.agent.use","workbench.agent.local.write"];
 await page.route("**/*",async route=>{
  const url=new URL(route.request().url());
  if(url.origin!==origin)return route.abort();
  if(url.pathname==="/api/workbench/context"){
   return route.fulfill({json:{user:{id:"admin"},homeOrganizationId:"B",effectiveOrganizationId:"B",effectiveMemberId:"member-B-admin",selectionRequired:false,organizations:[{id:"B",name:"隔离验收企业",roles:[],permissions}]}});
  }
  if(url.pathname==="/api/image-proxy"){
   const asset=new URL(url.searchParams.get("url")??origin);
   if(asset.hostname!=="intake-media.example.com")return route.abort();
   const response=await page.request.get(backend+"/fixture-media"+asset.pathname);
   return route.fulfill({response});
  }
  return route.continue();
 });
 await page.goto("/workbench/data/mine/own/new");
 await expect(page.getByRole("heading",{name:"添加自有商品",level:1})).toBeVisible();
 const agent=await page.request.get("/api/workbench/agents/product.own.intake.agent",{headers:{"X-Expected-User-ID":"admin","X-Expected-Organization-ID":"B"}});
 expect(agent.status(),await agent.text()).toBe(200);
 expect((await agent.json()).canUse).toBe(true);
 const images=page.getByRole("region",{name:"商品图片"});
 const uploaded:string[]=[];
 for(const [name,fill] of [["common.png","#111111"],["white-m.png","#eeeeee"]]){
  const base64=await page.evaluate(color=>{
   const canvas=document.createElement("canvas");canvas.width=2;canvas.height=2;
   const ctx=canvas.getContext("2d")!;ctx.fillStyle=color;ctx.fillRect(0,0,2,2);
   return canvas.toDataURL("image/png").split(",")[1];
  },fill);
  await images.getByLabel("图片文件").setInputFiles({name,mimeType:"image/png",buffer:Buffer.from(base64,"base64")});
  const posted=page.waitForResponse(r=>r.url().endsWith("/api/workbench/collections/media")&&r.request().method()==="POST");
  await images.getByRole("button",{name:"上传图片",exact:true}).click();
  const response=await posted;
  expect(response.status(),await response.text()).toBe(200);
  uploaded.push((await response.json()).url);
  await expect(images.getByAltText(`商品图片 ${uploaded.length}`)).toBeVisible();
 }
 await page.getByRole("button",{name:"开始整理资料"}).click();
 await page.getByLabel("给 AI 的商品资料").fill("黑 M，SKU BLK-M，CNY 29，库存 0\n黑 L，SKU BLK-L，CNY 29\n白 M，SKU WHT-M，CNY 29，图片2");
 await page.getByLabel("让 AI 查看一张图片").selectOption({label:"图片 2"});
 const prepare=page.waitForResponse(r=>r.url().endsWith("/own-product-intake/prepare"));
 await page.getByRole("button",{name:"检查生成用量"}).click();
 const prepared=await prepare;
 expect(prepared.status(),await prepared.text()).toBe(200);
 const originalKey=prepared.request().headers()["idempotency-key"];
 await page.route("**/own-product-intake/*/confirm",route=>route.abort("failed"),{times:1});
 await page.getByRole("button",{name:"确认生成候选"}).click();
 await expect(page.getByRole("alert").filter({hasText:"请求结果待核实"})).toBeVisible();
 await page.getByRole("button",{name:"核实原生成结果"}).click();
 const recovering=page.waitForResponse(r=>r.url().endsWith("/own-product-intake/prepare"));
 await page.getByRole("button",{name:"按原请求重新检查用量"}).click();
 const recovered=await recovering;
 expect(recovered.status(),await recovered.text()).toBe(200);
 expect(recovered.request().headers()["idempotency-key"]).toBe(originalKey);
 expect(await recovered.json()).toEqual(await prepared.json());
 const generating=page.waitForResponse(r=>r.url().endsWith("/confirm")&&r.url().includes("own-product-intake"));
 await page.getByRole("button",{name:"确认生成候选"}).click();
 const generated=await generating;
 expect(generated.status(),await generated.text()).toBe(200);
 expect(generated.request().headers()["idempotency-key"]).toBe(originalKey);
 const data=await generated.json();
 expect(data.candidate.rows.map((r:{sku:string})=>r.sku)).toEqual(["BLK-M","BLK-L","WHT-M"]);
 const preview=page.getByRole("region",{name:"AI 生成候选"});
 await expect(preview.getByRole("listitem")).toHaveCount(3);
 await expect(preview.getByText("白 / L",{exact:true})).toHaveCount(0);
 await expect(page.getByLabel("商品名称")).toHaveValue("");
 await preview.getByLabel("确认候选中的 SKU 图片归属").check();
 await preview.getByRole("button",{name:"采纳到商品表单"}).click();
 await expect(page.getByLabel("商品名称")).toHaveValue("隔离验收 三规格上衣");
 await expect(page.getByLabel(/^SKU \d$/)).toHaveCount(3);
 await expect(page.getByLabel("库存 1")).toHaveValue("0");
 await expect(page.getByLabel("库存 2")).toHaveValue("");
 await expect(images.getByLabel("公共图片 1")).toBeChecked();
 await expect(images.getByLabel("公共图片 2")).not.toBeChecked();
 await expect(page.getByLabel("变体图片 3")).toHaveValue(uploaded[1]);
 await page.screenshot({path:info.outputPath("adopted-three-skus.png"),fullPage:true});
 const saving=page.waitForResponse(r=>r.url().endsWith("/collections/commands")&&r.request().method()==="POST");
 await page.getByRole("button",{name:"保存商品",exact:true}).click();
 const saved=await saving;
 expect(saved.status(),await saved.text()).toBe(200);
 const {itemId}=await saved.json();
 await page.waitForURL(/\/workbench\/data\/mine\?view=own/);
 await page.getByRole("link",{name:/隔离验收 三规格上衣/}).click();
 await expect(page.getByText("全部变体 · 3")).toBeVisible();
 expect(new URL(page.url()).pathname).toBe("/workbench/data/mine/own/"+itemId);
 await page.reload();
 await expect(page.getByText("全部变体 · 3")).toBeVisible();
 const row=(sku:string)=>page.getByRole("row").filter({has:page.getByRole("cell",{name:sku,exact:true})});
 await expect(row("BLK-M").getByRole("cell",{name:"0",exact:true})).toBeVisible();
 await expect(row("BLK-L").getByRole("cell",{name:"未填写",exact:true})).toBeVisible();
 await expect(row("BLK-M").getByRole("img")).toHaveCount(0);
 await expect(row("BLK-L").getByRole("img")).toHaveCount(0);
 await expect(row("WHT-M").getByRole("img")).toHaveAttribute("src",new RegExp(encodeURIComponent(uploaded[1]).replace(/[.*+?^${}()|[\]\\]/g,"\\$&")));
 const response=await page.request.get("/api/workbench/collections/items/"+itemId,{headers:{"X-Expected-User-ID":"admin","X-Expected-Organization-ID":"B"}});
 expect(response.status(),await response.text()).toBe(200);
 const detail=await response.json();
 expect(detail.product.images.map((image:{url:string})=>image.url)).toEqual([uploaded[0]]);
 expect(detail.product.variants.map((row:{sku:string;stock?:number;images?:{url:string}[]})=>[row.sku,row.stock,row.images?.map(v=>v.url)??[]])).toEqual([
  ["BLK-M",0,[]],["BLK-L",undefined,[]],["WHT-M",undefined,[uploaded[1]]],
 ]);
 await page.screenshot({path:info.outputPath("saved-refreshed-three-skus.png"),fullPage:true});
});
