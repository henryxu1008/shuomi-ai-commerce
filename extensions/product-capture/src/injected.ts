import { captureDocument } from './extractor';
import { captureAmazonDocument } from './amazon/extractor';
import { capturePageSource } from './platforms';
import { discoverFamily } from './amazon/family-discovery';
import { amazonDocumentError } from './amazon/page-state';
import { captureException } from './entry-errors';
export function familyInfo() {
  try { return {ok:true,manifest:discoverFamily(document,document.URL)}; }
  catch(e) { return {ok:false,code:captureException(e)}; }
}
export async function runCapture() {
  try {
    const interruption = amazonDocumentError(document);
    if (interruption) throw Error(interruption);
    const source = capturePageSource(document.URL);
    const capture = source.platform === 'amazon' ? captureAmazonDocument : captureDocument;
    return { ok: true, payload: await capture(document, document.URL, new Date()) };
  }
  catch(e) { return { ok: false, code: captureException(e) }; }
}
