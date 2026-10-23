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
 const payload = {captureVersion:2 as const,source:'amazon',evidence:{sourceURL:'https://www.amazon.com/dp/B000000001',title:'Selected',missingFacts:[],site:'us',sourceID:'B000000001'}};
 let opened = '';
 const control = new Controller('https://app.example.com/capture/1688',async()=>payload,async url=>{opened=url;return 42;},()=>'extension-id',value=>{
   if(value!==payload) throw Error('INVALID_CAPTURE'); return payload;
 },()=> 'https://app.example.com/capture/amazon');
 await control.capture(); await control.handoff();
 expect(opened).toMatch(/^https:\/\/app.example.com\/capture\/amazon#/);
 expect(control.snapshot().sourceURL).toBe(payload.evidence.sourceURL);
});
