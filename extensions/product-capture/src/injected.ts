import { captureDocument } from './extractor';
import { captureAmazonDocument } from './amazon/extractor';
import { capturePageSource } from './platforms';
export async function runCapture() {
  try {
    const source = capturePageSource(document.URL);
    const capture = source.platform === 'amazon' ? captureAmazonDocument : captureDocument;
    return { ok: true, payload: await capture(document, document.URL, new Date()) };
  }
  catch { return { ok: false, code: 'CAPTURE_REJECTED' }; }
}
