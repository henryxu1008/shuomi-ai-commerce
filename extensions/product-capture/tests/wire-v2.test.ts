import { testDrafts } from './fixtures/drafts';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { JSDOM } from 'jsdom';
import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { validateCapture } from '../src/validation';
import { Controller } from '../src/controller';

it('admits only the exact v2 combination, preserving variant images and the v1 fixture', () => {
 const v1 = JSON.parse(readFileSync('tests/fixtures/capture-v1.json','utf8'));
 expect(validateCapture(v1)).toEqual(v1);
 const v2 = structuredClone(v1);
 v2.captureVersion = 2; v2.evidence.schemaVersion = 2; v2.evidence.parserVersion = '1688-browser-dom/v3';
 v2.evidence.variants[0].images = [{url:'https://cbu01.alicdn.com/blue.jpg',role:'source'}];
 expect(validateCapture(v2)).toEqual(v2);
 expect(()=>validateCapture({...v2,captureVersion:1})).toThrow();
 expect(()=>validateCapture({...v2,evidence:{...v2.evidence,schemaVersion:1}})).toThrow();
 expect(()=>validateCapture({...v1,evidence:v2.evidence})).toThrow();
 delete v2.evidence.variants[0].images;
 expect(()=>validateCapture(v2)).toThrow();
});

it('uses the platform owner validator and a fixed receiver without stripping platform facts', async () => {
 const dom=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:'https://www.amazon.com/dp/B0BDLGZCTY'});const payload=await captureAmazonDocument(dom.window.document,dom.window.document.URL);dom.window.close();
 let opened = '';
 const control = new Controller(testDrafts(),async()=>({url:payload.evidence.sourceURL}),'https://app.example.com/capture/1688',async()=>payload,async url=>{opened=url;return 42;},()=>'extension-id',value=>{
   if(value!==payload) throw Error('INVALID_CAPTURE'); return payload;
 },()=> 'https://app.example.com/capture/amazon');
 await control.capture(); await control.handoff();
 expect(opened).toMatch(/^https:\/\/app.example.com\/capture\/amazon#/);
 expect(control.snapshot().sourceURL).toBe(payload.evidence.sourceURL);
});
