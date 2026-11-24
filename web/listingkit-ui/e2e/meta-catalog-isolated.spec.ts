import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import {expect,test} from "@playwright/test";

const secret="isolated-meta-workbench-browser-test";
const storeID=process.env.META_TEST_STORE!;
const sourceID=process.env.META_TEST_SOURCE!;
const graphOrigin=process.env.META_GRAPH_ORIGIN!;
test.skip(process.env.META_WORKBENCH_BROWSER!=="1"||process.env.AUTH_SECRET!==secret||
  !storeID||!sourceID||!graphOrigin,"requires the isolated Go Meta fixture");

test("customer authorizes a shared catalog, uploads once and queries the saved result",async({page,context,baseURL,request})=>{
  test.setTimeout(110_000);
  page.setDefaultTimeout(20_000);
  const origin=baseURL!;
  const namespace=createHash("sha256").update(JSON.stringify([origin,"https://meta-workbench-test.invalid","meta-workbench-test"])).digest("hex").slice(0,32);
  const name="shuomi."+namespace+".session-token";
  const value=await encode({
    secret,salt:name,maxAge:600,
    token:{accessToken:"fixture-meta-access",expiresAt:Math.floor(Date.now()/1000)+600,
      identityVersion:3,identity:{tenantId:"org-a",userId:"actor-a",userType:"zitadel",roles:[]}},
  });
  await context.addCookies([
    {name,value,url:origin,httpOnly:true,sameSite:"Lax"},
    {name:"shuomi_effective_organization",value:"org-a",url:origin,httpOnly:true,sameSite:"Lax"},
  ]);
  const permissions=["workbench.store.read","workbench.store.update","workbench.supply.read","workbench.supply.manage","workbench.supply.submit","workbench.collection.read"];
  await page.route("**/*",async route=>{
    const url=new URL(route.request().url());
    if(url.origin!==origin&&url.origin!==graphOrigin)return route.abort();
    if(url.origin===origin&&url.pathname==="/api/workbench/context"){
      return route.fulfill({json:{user:{id:"actor-a"},homeOrganizationId:"org-a",effectiveOrganizationId:"org-a",
        effectiveMemberId:"member-a",selectionRequired:false,organizations:[{id:"org-a",name:"测试企业",roles:[],permissions}]}});
    }
    return route.continue();
  });

  await page.goto("/workbench/stores/"+storeID+"#authorization");
  await page.getByRole("button",{name:"授权 Meta",exact:true}).click();
  await expect(page.getByRole("heading",{name:"模拟 Meta 授权"})).toBeVisible();
  await page.getByRole("button",{name:"允许硕米管理商品目录"}).click();
  await page.getByLabel("商品目录",{exact:true}).selectOption("123");
  await page.getByLabel("WhatsApp Business 账号（可选）").selectOption("456");
  await page.getByRole("button",{name:"确认连接"}).click();
  await expect(page.getByText("Meta 模拟目录已连接。")).toBeVisible();

  const productPath="/workbench/listing/meta-catalog/"+sourceID;
  await page.goto(productPath);
  await page.getByLabel("目标店铺").selectOption(storeID);
  await page.getByLabel("商品规格").selectOption("default");
  await page.getByLabel("已审核的商品主图").selectOption("approved-main");
  await page.getByLabel("买家可购买该商品的页面").fill("https://seller.example.org/desk");
  await page.getByLabel("销售价（USD）").fill("19.95");
  let finishPreview:()=>void=()=>{};
  const previewGate=new Promise<void>(resolve=>{finishPreview=resolve;});
  await page.route("**/supply-preparations/meta-catalog/prepare",async route=>{
    await previewGate;
    await route.continue();
  },{times:1});
  await page.getByRole("button",{name:"预览上传"}).click();
  await expect(page.getByLabel("商品规格")).toBeDisabled();
  await expect(page.getByLabel("目标店铺")).toBeDisabled();
  finishPreview();
  const preview=page.getByRole("region",{name:"待确认商品"});
  await expect(preview.getByText("目标目录 123")).toBeVisible();
  await expect(preview.getByText("Desk organizer · Standard",{exact:true})).toBeVisible();
  await expect(preview.getByText("USD 19.95")).toBeVisible();
  expect(await (await request.get(graphOrigin+"/fixture/writes")).json()).toBe(0);
  await page.getByRole("button",{name:"确认上传到 Meta"}).click();
  await expect(page.getByText("模拟商品目录已收到商品")).toBeVisible();
  expect(await (await request.get(graphOrigin+"/fixture/writes")).json()).toBe(1);

  await page.goto(productPath);
  await page.getByRole("region",{name:"最近的 Meta 上传记录"}).getByRole("link",{name:/的上传记录/}).click();
  await page.getByRole("button",{name:"查询提交结果"}).click();
  await expect(page.getByText("已在模拟目录中查到此商品")).toBeVisible();
  await expect(page.getByText("WhatsApp 已连接此目录")).toBeVisible();
  await expect(page.getByText("Facebook 商店展示待核验")).toBeVisible();
  expect(await (await request.get(graphOrigin+"/fixture/writes")).json()).toBe(1);
});
