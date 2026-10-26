import { captureDocument } from './extractor';
import { captureAmazonDocument } from './amazon/extractor';
import { capturePageSource } from './platforms';
import { discoverFamily } from './amazon/family-discovery';
export function familyInfo() {
  try { return {ok:true,manifest:discoverFamily(document,document.URL)}; }
  catch(e) { return {ok:false,code:e instanceof Error?e.message:'FAMILY_INVALID'}; }
}
export async function runCapture() {
  try {
    const source = capturePageSource(document.URL);
    const capture = source.platform === 'amazon' ? captureAmazonDocument : captureDocument;
    return { ok: true, payload: await capture(document, document.URL, new Date()) };
  }
  catch { return { ok: false, code: 'CAPTURE_REJECTED' }; }
}
