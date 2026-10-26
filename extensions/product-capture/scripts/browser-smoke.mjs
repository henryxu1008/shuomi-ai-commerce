// Task-only browser harness. It substitutes the public page response and a
// transport-only receiver; it cannot prove Web/BFF/SRC1/Catalog acceptance.
import { chromium } from '@playwright/test';
import { createServer } from 'node:http';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';

const root = fileURLToPath(new URL('..', import.meta.url));
const head = execFileSync('git',['rev-parse','HEAD'],{cwd:root,encoding:'utf8'}).trim();
const browserName = process.argv.includes('--edge') ? 'edge' : 'chrome';
const automated = process.argv.includes('--cdp');
const executablePath = browserName === 'edge' ? 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe' : 'C:/Program Files/Google/Chrome/Application/chrome.exe';
await mkdir(resolve(root, 'artifacts'), { recursive: true });
const artifacts = await mkdtemp(resolve(root, 'artifacts', `${browserName}-`));
const profile = resolve(artifacts, 'profile');
const receiver = `<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="referrer" content="no-referrer"><title>Issue399 Fixture Receiver</title></head><body>
<h1>Transport Fixture — 不是真实后端</h1><p id="status">等待受限capture消息</p><button id="unknown" disabled>模拟响应丢失</button><pre id="summary"></pre>
<script>
const params = new URLSearchParams(location.hash.slice(1));
const extensionId=params.get('extensionId'),handoffId=params.get('handoffId'),idempotencyKey=params.get('idempotencyKey');
window.fixtureReceived=null;
async function readCapture(){
  if(!extensionId){document.querySelector('#status').textContent='恢复模式：只有原key，无自动POST';return;}
  let response;
  for(let attempt=0;attempt<3;attempt++){
    response=await chrome.runtime.sendMessage(extensionId,{version:1,type:'capture.read',handoffId,idempotencyKey});
    if(response?.type==='capture.payload')break;
    await new Promise(resolve=>setTimeout(resolve,250));
  }
  if(response?.type!=='capture.payload'){document.querySelector('#status').textContent='CAPTURE_UNAVAILABLE';return;}
  window.fixtureReceived=response;
  history.replaceState(null,'','#operationKey='+idempotencyKey);
  document.querySelector('#summary').textContent=JSON.stringify(response.payload,null,2);
  document.querySelector('#status').textContent='FIXTURE_CAPTURE_RECEIVED';document.querySelector('#unknown').disabled=false;
}
document.querySelector('#unknown').onclick=async()=>{
  await chrome.runtime.sendMessage(extensionId,{version:1,type:'capture.status',handoffId,idempotencyKey,outcome:'outcome_unknown'});
  document.querySelector('#status').textContent='FIXTURE_OUTCOME_UNKNOWN';
};
readCapture().catch(()=>document.querySelector('#status').textContent='CAPTURE_UNAVAILABLE');
</script></body></html>`;
const server = createServer((req, res) => {
  if (req.method !== 'GET' || !['/capture/1688','/capture/amazon'].includes(req.url)) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'Content-Type':'text/html; charset=utf-8', 'Referrer-Policy':'no-referrer' }); res.end(receiver);
});
await new Promise((done, reject) => { server.once('error',reject); server.listen(4399,'127.0.0.1',done); });
let context;
let familyMode=false,releaseChild;
const childGate=new Promise(resolve=>{releaseChild=resolve;});
let familyChildRequests=0;
const familyReaders=new Set();
try {
  context = await chromium.launchPersistentContext(profile, {
    executablePath, headless:automated, ignoreDefaultArgs:['--disable-extensions'], viewport:null,
    args:['--no-first-run','--no-default-browser-check','--disable-background-networking', ...(automated ? ['--enable-unsafe-extension-debugging','--remote-debugging-port=0'] : [])],
  });
  await context.route('**/*', async route => {
    const url = new URL(route.request().url());
    if (url.origin === 'https://detail.1688.com' && url.pathname === '/offer/981645030344.html') {
      await route.fulfill({ contentType:'text/html; charset=utf-8', body:await readFile(resolve(root,'tests/fixtures/product.html'),'utf8') });
    } else if (url.origin === 'https://www.amazon.com' && url.pathname === '/dp/B0BDLGZCTY') {
      await route.fulfill({ contentType:'text/html; charset=utf-8', body:await readFile(resolve(root,`tests/fixtures/${familyMode?'amazon-family-product':'amazon-inline-product'}.html`),'utf8') });
    } else if (familyMode && url.origin==='https://www.amazon.com' && ['/dp/B0CC1YM6SJ','/dp/B000000002'].includes(url.pathname)) {
      familyChildRequests++;
      familyReaders.add(route.request().frame().page());
      await childGate;
      await route.fulfill({contentType:'text/html; charset=utf-8',body:await readFile(resolve(root,`tests/fixtures/${url.pathname.endsWith('B000000002')?'amazon-family-next-child':'amazon-family-child'}.html`),'utf8')});
    } else if (url.origin === 'http://127.0.0.1:4399' || url.protocol === 'chrome-extension:') await route.continue();
    else await route.abort();
  });
  const page = context.pages()[0];
  if (!automated) await page.goto(browserName === 'edge' ? 'edge://extensions' : 'chrome://extensions');
  console.log(JSON.stringify({ stage:'LOAD_UNPACKED_IN_TASK_PROFILE', browserName, extensionPath:resolve(root,'dist-fixture'), artifacts }));
  let browserCDP;
  let appPromise;
  if (automated) {
    browserCDP = await context.browser().newBrowserCDPSession();
    // Chrome writes this file inside the atomically allocated task profile.
    // Never discover targets on a conventional/shared debugging port.
    const [port, browserPath] = (await readFile(resolve(profile, 'DevToolsActivePort'), 'utf8')).trim().split(/\r?\n/);
    if (!/^[0-9]+$/.test(port) || Number(port) < 1 || Number(port) > 65535
      || !/^\/devtools\/browser\/[a-f0-9-]+$/i.test(browserPath)) throw Error('Invalid task debugger endpoint');
    console.log(await browserCDP.send('Extensions.loadUnpacked',{path:resolve(root,'dist-fixture')}));
  }
  const worker = context.serviceWorkers()[0] ?? await context.waitForEvent('serviceworker',{timeout:600000});
  const id = worker.url().split('/')[2];
  const manifest = await worker.evaluate(() => chrome.runtime.getManifest());
  // The shipped build's grant must stay byte-identical to what the manifest test pins; a
  // smoke run that only checks "no host_permissions" would pass a widened grant silently.
  if (JSON.stringify(manifest.permissions) !== JSON.stringify(['activeTab','scripting','storage']) || JSON.stringify(manifest.host_permissions) !== JSON.stringify(['https://detail.1688.com/*','https://www.amazon.com/*','https://www.amazon.ca/*','https://www.amazon.co.uk/*','https://www.amazon.de/*','https://www.amazon.co.jp/*'])) throw Error('Unexpected permissions');
  const recreatedReturns=[];
  async function recreateAndReturn(sourcePage,receiver,capture,platform){
    const originalURL=receiver.url(), pageCount=context.pages().length;
    const control=await context.newCDPSession(sourcePage);
    await control.send('ServiceWorker.enable');
    await control.send('ServiceWorker.stopAllWorkers');
    const afterStop=await receiver.evaluate(async ({id,capture})=>{
      try { return await chrome.runtime.sendMessage(id,{version:1,type:'capture.read',handoffId:capture.handoffId,idempotencyKey:capture.idempotencyKey}); }
      catch(error){
        if(error.message==='Could not establish connection. Receiving end does not exist.')return {code:'LISTENER_UNAVAILABLE'};
        throw error;
      }
    },{id,capture});
    if(!['CAPTURE_UNAVAILABLE','LISTENER_UNAVAILABLE'].includes(afterStop?.code))throw Error('Worker recreation retained stale payload');
    await sourcePage.bringToFront();
    await sourcePage.getByRole('button',{name:'刷新状态',exact:true}).click();
    const button=sourcePage.getByRole('button',{name:'返回应用核实',exact:true});
    await button.waitFor(); await button.click();
    await receiver.waitForFunction(()=>document.visibilityState==='visible');
    if(context.pages().length!==pageCount || receiver.url()!==originalURL)throw Error('Worker recreation replaced or navigated the original receiver');
    const stateWorker=context.serviceWorkers().find(value=>value.url()===`chrome-extension://${id}/background.js`);
    if(!stateWorker)throw Error('Source return did not revive the worker');
    const stored=await stateWorker.evaluate(()=>chrome.storage.session.get(null));
    const entries=Object.values(stored);
    if(entries.length!==1 || entries[0].phase!=='bound' || !Number.isInteger(entries[0].tabId)
      || JSON.stringify(Object.keys(entries[0]).sort())!==JSON.stringify(['phase','revision','sourceURL','tabId','version'])
      || [capture.idempotencyKey,capture.handoffId].includes(entries[0].revision))throw Error('Unexpected session navigation contents');
    const panel=await sourcePage.locator('#shuomi-product-capture').innerText();
    if(panel.includes(capture.idempotencyKey)||panel.includes(capture.handoffId))throw Error('Private transport identity leaked into source entry');
    recreatedReturns.push({platform,sameOriginalTab:true,sameOriginalURL:true,noExtraTabs:true,oldPayloadUnavailable:afterStop.code,sessionFields:Object.keys(entries[0]).sort()});
  }
  await page.goto('https://detail.1688.com/offer/981645030344.html');
  console.log(JSON.stringify({ stage:'CLICK_SOURCE_PAGE_CAPTURE', id }));
  if (automated) {
    const button=page.getByRole('button',{name:'采集并在应用确认',exact:true});
    await button.waitFor();
    await page.waitForFunction(()=>!document.querySelector('#shuomi-product-capture')?.shadowRoot?.querySelector('#primary')?.disabled);
    // A site-script click must not create a capture. Real browser input follows.
    await button.evaluate(element=>element.click());
    if(context.pages().length!==1)throw Error('Synthetic source-page click opened an application');
    // Retain the initial source DOM/revision while its still-empty worker dies.
    // Do not focus/refresh the panel before the first trusted capture click.
    const initialControl=await context.newCDPSession(page);
    await initialControl.send('ServiceWorker.enable');
    await initialControl.send('ServiceWorker.stopAllWorkers');
    await initialControl.detach();
    await page.screenshot({path:resolve(artifacts,'source-page.png')});
    appPromise=context.waitForEvent('page',{timeout:20000});
    await button.click();
  }
  let app = await (appPromise ?? context.waitForEvent('page', { timeout:600000 })).catch(async () => {
    const found = context.pages().find(p=>p.url().startsWith('http://127.0.0.1:4399'));
    if(!found)throw Error('No own receiver tab');return found;
  });
  await app.waitForURL('http://127.0.0.1:4399/capture/1688**');
  await app.locator('#status').filter({hasText:'FIXTURE_CAPTURE_RECEIVED'}).waitFor({timeout:20000});
  let capture = await app.evaluate(()=>window.fixtureReceived);
  if (!capture || /PASSWORD_CANARY|AUTH_CANARY|TOKEN_CANARY|SESSION_CANARY/.test(JSON.stringify(capture))) throw Error('Missing or sensitive capture');
  if (capture.payload.evidence.variants[0].sourceID !== '99999999999999999999') throw Error('SKU precision lost');
  if (capture.payload.evidence.variants[0].sku !== null) throw Error('Provider variant ID was fabricated into a SKU');
  const firstCapture=capture;
  const firstApp=app, firstURL=app.url(), pagesBeforeReturn=context.pages().length;
  await page.bringToFront();
  await page.getByRole('button',{name:'刷新状态',exact:true}).click();
  await page.getByRole('button',{name:'返回应用核实',exact:true}).click();
  await firstApp.waitForFunction(()=>document.visibilityState==='visible');
  if(context.pages().length!==pagesBeforeReturn || firstApp.url()!==firstURL)throw Error('Source return replaced or navigated the original app tab');
  const wrong = await context.newPage(); await wrong.goto(app.url());
  const rejection = await wrong.evaluate(async ({id,capture})=>chrome.runtime.sendMessage(id,{version:1,type:'capture.read',handoffId:capture.handoffId,idempotencyKey:capture.idempotencyKey}),{id,capture});
  if(rejection.code!=='CAPTURE_UNAVAILABLE')throw Error('Other application tab read accepted');
  await wrong.close();
  if(automated)await recreateAndReturn(page,firstApp,firstCapture,'1688');
  let amazonSource;
  if(automated){
    const amazon=await context.newPage(); await amazon.goto('https://www.amazon.com/dp/B0BDLGZCTY');
    amazonSource=amazon;
    await amazon.getByRole('button',{name:'返回已有采集',exact:true}).waitFor();
    await amazon.getByRole('button',{name:'采集当前商品（新操作）',exact:true}).click();
    await amazon.getByRole('button',{name:'取消',exact:true}).click();
    if(context.pages().length!==pagesBeforeReturn+1)throw Error('Cancel started a new capture');
    await amazon.getByRole('button',{name:'采集当前商品（新操作）',exact:true}).click();
    const secondApp=context.waitForEvent('page',{timeout:20000});
    await amazon.getByRole('button',{name:'确认采集当前商品',exact:true}).click();
    app=await secondApp;
    await app.waitForURL('http://127.0.0.1:4399/capture/amazon**');
    await app.locator('#status').filter({hasText:'FIXTURE_CAPTURE_RECEIVED'}).waitFor({timeout:20000});
    capture=await app.evaluate(()=>window.fixtureReceived);
    if(capture?.payload?.source!=='amazon' || capture.idempotencyKey===firstCapture.idempotencyKey)throw Error('Amazon explicit new capture failed');
    if(firstApp.url()!==firstURL)throw Error('New capture modified previous app operation');
    await writeFile(resolve(artifacts,'amazon-capture.json'),JSON.stringify(capture.payload,null,2));
    await amazon.bringToFront();
    await amazon.getByRole('button',{name:'刷新状态',exact:true}).click();
    await amazon.getByRole('button',{name:'返回应用核实',exact:true}).waitFor();
    const panelBounds=await amazon.locator('#shuomi-product-capture').boundingBox();
    const viewport=await amazon.evaluate(()=>({width:innerWidth,height:innerHeight}));
    if(!panelBounds || panelBounds.y<0 || panelBounds.y+panelBounds.height>viewport.height || panelBounds.x+panelBounds.width>viewport.width)throw Error('Source panel outside the viewport');
    await amazon.screenshot({path:resolve(artifacts,'amazon-source-page.png')});
  }
  await app.locator('#unknown').click();
  await app.locator('#status').filter({hasText:'FIXTURE_OUTCOME_UNKNOWN'}).waitFor();
  await app.screenshot({path:resolve(artifacts,'receiver.png')});
  const recoveryURL=app.url();
  if(!recoveryURL.endsWith(`#operationKey=${capture.idempotencyKey}`))throw Error('Original recovery key was lost');
  let workerRestartRejected=false;
  let workerStopOutcome=null;
  if(automated){
    await recreateAndReturn(amazonSource,app,capture,'amazon');
    workerStopOutcome=recreatedReturns.at(-1).oldPayloadUnavailable;
    workerRestartRejected=true;
  }
  await app.reload();
  await app.locator('#status').filter({hasText:'恢复模式：只有原key，无自动POST'}).waitFor();
  if(app.url()!==recoveryURL)throw Error('Reload changed recovery identity');
  let familyResult=null;
  if(automated){
    familyMode=true;
    await amazonSource.reload();
    await amazonSource.getByRole('button',{name:'采集当前商品（新操作）',exact:true}).click();
    await amazonSource.getByRole('checkbox',{name:'同时采集其他颜色／尺寸（本次按钮）',exact:true}).check();
    const progressPromise=context.waitForEvent('page',{timeout:20000}).catch(async error=>{
      const live=context.serviceWorkers().find(w=>w.url()===`chrome-extension://${id}/background.js`);
      const native=await live.evaluate(async()=>{
        const [tab]=await chrome.tabs.query({url:'https://www.amazon.com/dp/B0BDLGZCTY'});
        return {tabId:tab.id,documents:await chrome.scripting.executeScript({target:{tabId:tab.id},func:()=>null}),stored:await chrome.storage.session.get(null)};
      });
      console.log(JSON.stringify({stage:'FAMILY_ENTRY_FAILURE',status:await amazonSource.locator('#shuomi-product-capture').evaluate(e=>e.shadowRoot.querySelector('[role="status"]').textContent),native}));
      throw error;
    });
    await amazonSource.getByRole('button',{name:'确认采集当前商品',exact:true}).click();
    const progress=await progressPromise;
    await progress.waitForURL(`chrome-extension://${id}/popup.html#amazon-family=*`);
    await progress.getByRole('status').filter({hasText:'1/3 项已观察'}).waitFor();
    const activeWorker=context.serviceWorkers().find(w=>w.url()===`chrome-extension://${id}/background.js`);
    const nativeBinding=await activeWorker.evaluate(async()=>({contexts:await chrome.runtime.getContexts({contextTypes:['TAB']}),stored:await chrome.storage.session.get(null)}));
    const nav=Object.values(nativeBinding.stored).find(v=>v.sourceDocumentId);
    if(!nav || nav.phase!=='bound' || !nativeBinding.contexts.some(c=>c.documentId===nav.documentId&&c.documentUrl===progress.url()))throw Error('Native family document/fragment was not bound');
    if(/family-grey|39\.98|customerVisiblePrice|idempotencyKey/.test(JSON.stringify(nativeBinding.stored)))throw Error('Family facts leaked into session navigation');
    const beforeRestart=context.pages().length;
    const control=await context.newCDPSession(amazonSource);
    await control.send('ServiceWorker.enable');await control.send('ServiceWorker.stopAllWorkers');await control.detach();
    await amazonSource.getByRole('button',{name:'刷新状态',exact:true}).click();
    await amazonSource.getByRole('button',{name:'查看采集进度',exact:true}).click();
    await progress.waitForFunction(()=>document.visibilityState==='visible');
    if(context.pages().length!==beforeRestart || familyChildRequests!==1)throw Error('Family worker recovery duplicated a page or source read');
    const familyAppPromise=context.waitForEvent('page',{timeout:20000});
    releaseChild();
    const familyApp=await familyAppPromise;
    await familyApp.waitForURL('http://127.0.0.1:4399/capture/amazon**');
    await familyApp.locator('#status').filter({hasText:'FIXTURE_CAPTURE_RECEIVED'}).waitFor();
    const familyCapture=await familyApp.evaluate(()=>window.fixtureReceived);
    const evidence=familyCapture?.payload?.evidence;
    if(evidence?.observation?.scope!=='partial_family'||evidence.observation.parentASIN!=='B0D76YXPWZ'||evidence.variants.length!==3)throw Error('Family aggregation missing');
    if(evidence.variants[1].sourceID!=='B0CC1YM6SJ'||evidence.variants[1].price?.amount!=='44.98'||evidence.variants[1].price.currency!=='USD'||!evidence.variants[1].images[0].url.includes('family-beige')||evidence.variants[0].price?.amount!=='39.98')throw Error('Family child facts mixed');
    if(evidence.variants[2].sourceID!=='B000000002'||evidence.variants[2].price?.amount!=='49.98'||!evidence.variants[2].images[0].url.includes('family-black'))throw Error('Next family child facts mixed');
    if(amazonSource.url()!=='https://www.amazon.com/dp/B0BDLGZCTY'||familyChildRequests!==2||familyReaders.size!==1)throw Error('Family changed anchor or duplicated a child read/reader');
    const afterHandoff=context.pages().length, familyURL=familyApp.url();
    await progress.getByRole('button',{name:'返回应用核实',exact:true}).click();
    await familyApp.waitForFunction(()=>document.visibilityState==='visible');
    if(context.pages().length!==afterHandoff||familyApp.url()!==familyURL)throw Error('Family return did not preserve receiver');
    await writeFile(resolve(artifacts,'amazon-family-capture.json'),JSON.stringify(familyCapture.payload,null,2));
    familyResult={nativeDocumentBound:true,workerRebuildSameProgress:true,eachChildReadOnce:true,oneReusedReader:true,oneReceiver:true,independentChildFacts:true,originalAnchorUnchanged:true,sameReceiverReturn:true};
  }
  await writeFile(resolve(artifacts,'capture.json'),JSON.stringify(firstCapture.payload,null,2));
  const buildHashes={};
  for(const name of ['manifest.json','background.js','extractor.js','page.js','popup.js','popup.html','popup.css']) buildHashes[name]=createHash('sha256').update(await readFile(resolve(root,'dist-fixture',name))).digest('hex');
  await writeFile(resolve(artifacts,'receipt.json'),JSON.stringify({head,buildHashes,browserName,browserVersion:context.browser()?.version(),id,manifest,
    result:'PASS_TRANSPORT_FIXTURE_ONLY',ownProfileDynamicEndpoint:automated,sourcePageEntry:automated,initialCaptureAfterWorkerStop:automated,syntheticClickRejected:automated,originalTabReturn:true,amazonExplicitNewCapture:automated,ownTabCapture:true,wrongAppTabRejected:true,unknownStatusSent:true,workerRestartRejected,workerStopOutcome,recreatedReturns,reloadPreservedKey:true,
    familyResult,backendIntegration:'NOT_RUN',real1688:'NOT_RUN',realAmazon:'NOT_RUN'},null,2));
  console.log(JSON.stringify({stage:'TRANSPORT_FIXTURE_PASS',artifacts}));
} finally {
  releaseChild();
  if(context)await context.close();
  await new Promise(done=>server.close(done));
}
