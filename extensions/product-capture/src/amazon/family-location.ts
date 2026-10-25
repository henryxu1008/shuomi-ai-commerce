import { z } from 'zod';
import { UUID } from '../validation';
import { amazonPageSource } from './validation';
import type { CaptureTab } from '../controller';

const id = z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
// Native documentId is opaque (Edge emits 32 uppercase hex characters), not our transport UUID.
const documentId = z.string().regex(/^[A-Za-z0-9_-]{1,128}$/);
const source = z.string().max(2048).refine(v => { try { return amazonPageSource(v).sourceURL === v; } catch { return false; } });
const common = {version:z.literal(1),revision:z.string().regex(UUID),sourceURL:source,sourceTabId:id,sourceDocumentId:documentId};
const reader = z.strictObject({tabId:id,sourceURL:source});
const schema = z.discriminatedUnion('phase',[
  z.strictObject({...common,phase:z.literal('pending'),tabId:id.optional()}),
  z.strictObject({...common,phase:z.literal('bound'),tabId:id,documentId,reader:reader.optional()}),
]);
export type FamilyRecord = z.infer<typeof schema>;
type Area = { get(key:string):Promise<Record<string,unknown>>; set(items:Record<string,unknown>):Promise<void>; remove(key:string):Promise<void>; setAccessLevel(options:{accessLevel:'TRUSTED_CONTEXTS'}):Promise<void> };
const key = 'product-capture:family-location:v1';
export const familyURL = (revision:string) => `${chrome.runtime.getURL('popup.html')}#amazon-family=${revision}`;
export class FamilyLocation {
  private record?: FamilyRecord;
  private failed = false;
  readonly ready: Promise<void>;
  constructor(private readonly area:Area|undefined) { this.ready=this.load(); }
  private async load() {
    try {
      if(!this.area)throw Error('unavailable');
      await this.area.setAccessLevel({accessLevel:'TRUSTED_CONTEXTS'});
      const values=await this.area.get(key);
      if(!values || typeof values!=='object' || Array.isArray(values))throw Error('invalid');
      if(Object.hasOwn(values,key))this.record=schema.parse(values[key]);
    } catch { this.failed=true; }
  }
  assertReadable(){if(this.failed)throw Error('RETURN_STATE_UNAVAILABLE');}
  get unavailable(){return this.failed;}
  get current(){this.assertReadable();return this.record?structuredClone(this.record):undefined;}
  private async write(value:FamilyRecord) {
    this.assertReadable();try{await this.area!.set({[key]:value});this.record=value;}
    catch{this.failed=true;throw Error('RETURN_STATE_UNAVAILABLE');}
  }
  async prepare(tab:CaptureTab,revision:string) {
    this.assertReadable();if(this.record)throw Error('ACTION_IN_PROGRESS');
    if(!tab.url)throw Error('INVALID_PAGE');
    await this.write(schema.parse({version:1,revision,sourceURL:amazonPageSource(tab.url).sourceURL,sourceTabId:tab.id,sourceDocumentId:tab.documentId,phase:'pending'}));
  }
  async noteTab(tabId:number) {
    const r=this.current;if(!r || r.phase!=='pending')throw Error('ENTRY_CHANGED');
    await this.write(schema.parse({...r,tabId}));
  }
  async bind(tabId:number,documentId:string) {
    const r=this.current;if(!r || r.tabId!==undefined && r.tabId!==tabId || r.phase==='bound' && r.documentId!==documentId)throw Error('ENTRY_CHANGED');
    if(r.phase==='bound')return;
    await this.write(schema.parse({...r,phase:'bound',tabId,documentId}));
  }
  async reader(tabId:number,sourceURL:string) {
    const r=this.current;if(!r || r.phase!=='bound' || new URL(sourceURL).origin!==new URL(r.sourceURL).origin)throw Error('ENTRY_CHANGED');
    await this.write(schema.parse({...r,reader:{tabId,sourceURL}}));
  }
  async forgetReader() {
    const r=this.current;if(!r || r.phase!=='bound')throw Error('ENTRY_CHANGED');
    const {reader:unused,...rest}=r;void unused;await this.write(rest);
  }
  async clear() {
    this.assertReadable();try{await this.area!.remove(key);this.record=undefined;}
    catch{this.failed=true;throw Error('RETURN_STATE_UNAVAILABLE');}
  }
}
