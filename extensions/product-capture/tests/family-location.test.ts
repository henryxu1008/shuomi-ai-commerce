import { expect, it } from 'vitest';
import { FamilyLocation } from '../src/amazon/family-location';
import { returnStorage } from './fixtures/return-storage';
const source={id:7,url:'https://www.amazon.com/dp/B0BDLGZCTY',documentId:crypto.randomUUID()};
it('preserves native opaque document ids and canonicalizes the source URL without replacing the native id',async()=>{
 const storage=returnStorage(),a=new FamilyLocation(storage.session);await a.ready;
 const native='06D2D309827BBAF659E84EACD4A23A2D';
 await a.prepare({...source,url:source.url+'?th=1&psc=1',documentId:native},crypto.randomUUID());
 await a.bind(9,native);expect(a.current).toMatchObject({sourceURL:source.url,sourceDocumentId:native,documentId:native});
});
it('writes only pending navigation before tab creation and survives a worker reconstruction',async()=>{
 const storage=returnStorage(), a=new FamilyLocation(storage.session);await a.ready;
 const revision=crypto.randomUUID();await a.prepare(source,revision);
 expect(a.current?.phase).toBe('pending');
 const b=new FamilyLocation(storage.session);await b.ready;expect(b.current).toEqual(a.current);
 expect(JSON.stringify(storage.contents())).not.toMatch(/payload|price|handoffId|operationKey|parentASIN/);
 await b.bind(9,crypto.randomUUID());expect(b.current?.phase).toBe('bound');
 await expect(b.prepare(source,crypto.randomUUID())).rejects.toThrow();
});
it('never treats corrupt or failed storage as an idle family entry',async()=>{
 const storage=returnStorage();storage.session.get.mockRejectedValue(Error('offline'));
 const a=new FamilyLocation(storage.session);await a.ready;expect(()=>a.assertReadable()).toThrow('RETURN_STATE_UNAVAILABLE');
 await expect(a.prepare(source,crypto.randomUUID())).rejects.toThrow();
});
it('rejects a replacement progress document and refuses recording an arbitrary reader URL',async()=>{
 const storage=returnStorage(),a=new FamilyLocation(storage.session);await a.ready;await a.prepare(source,crypto.randomUUID());
 const doc=crypto.randomUUID();await a.bind(9,doc);await expect(a.bind(9,crypto.randomUUID())).rejects.toThrow('ENTRY_CHANGED');
 await expect(a.reader(11,'https://evil.example/dp/B0CC1YM6SJ')).rejects.toThrow();
 await a.reader(11,'https://www.amazon.com/dp/B0CC1YM6SJ');const r=a.current;if(r?.phase!=='bound')throw Error('not bound');expect(r.reader?.tabId).toBe(11);
 expect(r.documentId).toBe(doc);
});
