import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { capturePageSource, validatePlatformCapture } from '../src/platforms';

it('selects only the explicit product source before injection',()=>{
 expect(capturePageSource('https://detail.1688.com/offer/9.html').platform).toBe('1688');
 expect(capturePageSource('https://www.amazon.co.jp/dp/B012345678?tag=tracking')).toEqual({platform:'amazon',site:'jp',sourceID:'B012345678',sourceURL:'https://www.amazon.co.jp/dp/B012345678'});
 for(const url of ['https://www.amazon.fr/dp/B012345678','https://amazon.com.evil.test/dp/B012345678','https://www.amazon.com/s?k=product','https://detail.1688.com/login.html']) expect(()=>capturePageSource(url)).toThrow();
});

it('runs the exact source validator and never falls back on a rejected source',()=>{
 const payload=JSON.parse(readFileSync('tests/fixtures/capture-v2.json','utf8'));
 expect(validatePlatformCapture(payload)).toEqual(payload);
 expect(()=>validatePlatformCapture({...payload,source:'amazon'})).toThrow();
 expect(()=>validatePlatformCapture({...payload,source:'unsupported'})).toThrow();
});
