import { amazonPageSource } from './validation';

// Fixed paths and controls only. Never read form values, query parameters or
// page scripts, and never treat a verification/sign-in page as product facts.
export function amazonInterruptionURL(input: string): 'AMAZON_VERIFICATION_REQUIRED' | 'AMAZON_SIGN_IN_REQUIRED' | undefined {
  let url: URL;
  try {
    url = new URL(input);
    if (url.username || url.password || url.port) return;
    amazonPageSource(`${url.origin}/dp/B000000000`);
  } catch { return; }
  if (/^\/(?:checkpoint\/continue|errors\/validateCaptcha)\/?$/.test(url.pathname)) return 'AMAZON_VERIFICATION_REQUIRED';
  if (/^\/ap\/signin\/?$/.test(url.pathname)) return 'AMAZON_SIGN_IN_REQUIRED';
}

export function amazonDocumentError(doc: Document) {
  const interruption = amazonInterruptionURL(doc.URL);
  if (interruption) return interruption;
  // Strict product admission precedes DOM checks; other sites/pages cannot
  // manufacture a supported verification or product-ready result.
  try { amazonPageSource(doc.URL); } catch { return; }
  if (doc.querySelector('input#continueCheck[type="checkbox"], form#validateCaptcha')) return 'AMAZON_VERIFICATION_REQUIRED';
  if (doc.querySelector('form[name="signIn"] input[type="password"], form#signIn input[type="password"]')) return 'AMAZON_SIGN_IN_REQUIRED';
  // The original parser owns title text, visibility and its cumulative byte /
  // node / time budget. Readiness must not concatenate the subtree first.
  if (doc.readyState === 'loading' || !doc.querySelector('#productTitle')) return 'PAGE_NOT_READY';
}
