import type { CaptureTab } from './controller';
import { capturePageSource, validatePlatformCapture } from './platforms';
import type { FamilyManifest } from './amazon/family-discovery';
import { captureErrorCode } from './entry-errors';
import { completeFamilyDimensions } from './amazon/dimensions';

export async function extractTab(tab:CaptureTab,signal?:AbortSignal) {
  signal?.throwIfAborted();
  if(tab.id===undefined || !tab.url)throw Error('INVALID_PAGE');
  const source=capturePageSource(tab.url);
  const target=tab.documentId?{tabId:tab.id,documentIds:[tab.documentId]}:{tabId:tab.id,frameIds:[0]};
  let results:chrome.scripting.InjectionResult[];
  try{results=await chrome.scripting.executeScript({target,world:'ISOLATED',files:['extractor.js']});}
  catch(e){if(tab.documentId)throw Error('PAGE_CHANGED');throw e;}
  signal?.throwIfAborted();
  const result=results[0]?.result as {ok?:boolean;payload?:unknown;code?:unknown}|undefined;
  if(results.length!==1 || results[0].frameId!==0 || !results[0].documentId)throw Error('CAPTURE_REJECTED');
  if(tab.documentId && results[0].documentId!==tab.documentId)throw Error('PAGE_CHANGED');
  if(result?.ok!==true){
    // Even rejected documents must still be the native main-frame document
    // that produced the result. This check emits no URL or verification token.
    try {
      const native=await chrome.scripting.executeScript({target:{tabId:tab.id,documentIds:[results[0].documentId]},world:'ISOLATED',func:()=>true});
      if(native.length!==1 || native[0].frameId!==0 || native[0].documentId!==results[0].documentId)throw Error('changed');
    }catch{throw Error('PAGE_CHANGED');}
    signal?.throwIfAborted();
    throw Error(captureErrorCode(result?.ok===false?result.code:undefined));
  }
  const payload=validatePlatformCapture(result.payload);
  await verifyDocument({id:tab.id,url:source.sourceURL,documentId:results[0].documentId});
  if(payload.evidence.sourceURL!==source.sourceURL)throw Error('PAGE_CHANGED');
  return {payload,documentId:results[0].documentId};
}
export async function verifyDocument(tab:Required<CaptureTab>) {
  try {
    const check=await chrome.scripting.executeScript({target:{tabId:tab.id,documentIds:[tab.documentId]},world:'ISOLATED',func:()=>document.URL});
    if(check.length!==1 || check[0].frameId!==0 || check[0].documentId!==tab.documentId || capturePageSource(check[0].result as string).sourceURL!==tab.url)throw Error('changed');
  }catch{throw Error('PAGE_CHANGED');}
}
export async function extractFamilyTab(tab:CaptureTab,required:boolean,signal?:AbortSignal) {
  const {payload,documentId}=await extractTab(tab,signal);
  if(!('source' in payload) || payload.source!=='amazon' || tab.id===undefined)throw Error('INVALID_PAGE');
  signal?.throwIfAborted();
  const result=await chrome.scripting.executeScript({target:{tabId:tab.id,documentIds:[documentId]},world:'ISOLATED',func:()=>{
    const entry=(globalThis as unknown as {CaptureEntry?:{familyInfo():{ok:boolean;manifest?:FamilyManifest;code?:string}}}).CaptureEntry;
    return entry?.familyInfo();
  }});
  signal?.throwIfAborted();
  await verifyDocument({id:tab.id,url:payload.evidence.sourceURL,documentId});
  signal?.throwIfAborted();
  if(result.length!==1 || result[0].frameId!==0 || result[0].documentId!==documentId)throw Error('PAGE_CHANGED');
  const value=result[0].result as {ok?:boolean;manifest?:FamilyManifest;code?:string}|undefined;
  if(!value?.ok && (required || value?.code!=='FAMILY_NOT_AVAILABLE'))throw Error(captureErrorCode(value?.code));
  const capture=await completeFamilyDimensions(payload,value?.manifest);
  signal?.throwIfAborted();
  // Composition may await WebCrypto. Its result still belongs only to the
  // original native document, including same-URL replacement documents.
  await verifyDocument({id:tab.id,url:payload.evidence.sourceURL,documentId});
  signal?.throwIfAborted();
  return {capture,manifest:value?.manifest};
}
