import { DraftStore } from '../../src/draft-store';
import type { FamilySeed } from '../../src/amazon/family-checkpoint';
import type { AmazonCapturePayload } from '../../src/amazon/wire';
// Existing page tests exercise a real durable journal, with only native Chrome
// surfaces mocked. They do not let a page supply a final, uncommitted payload.
export async function familyJournal(id:string,sourceURL:string) {
  const name='family-test-'+crypto.randomUUID();
  let store=new DraftStore(name),d=await store.create(sourceURL,'family','https://app.example.com/capture/amazon',id);
  return {
    store:()=>store,
    restart:async()=>{store=new DraftStore(name);await store.ready;d=(await store.get(id))!;},
    current:async()=>{d=(await store.get(id))!;return {draft:{revision:d.revision,generation:d.generation,phase:d.phase},checkpoint:await store.checkpoint(id)};},
    handle:async(m:Record<string,unknown>)=>{
      if(m.expectedRevision!==d.revision||m.generation!==d.generation)throw Error('DRAFT_CHANGED');
      const args=[id,d.revision,d.generation] as const;
      if(m.type==='family.resume'){if(d.phase==='paused')d=await store.resume(...args);}
      if(m.type==='family.seed')d=await store.seed(...args,m.seed as FamilySeed);
      if(m.type==='family.observe')d=await store.observe(...args,m.observation as {capture:AmazonCapturePayload;parentASIN:string|null});
      if(m.type==='family.pause'&&d.phase==='capturing')d=await store.pause(...args,String(m.reason??''));
      if(m.type==='family.finish')d=await store.finishFamily(...args);
      return {revision:d.revision,generation:d.generation,phase:d.phase,payload:d.payload};
    },
  };
}
