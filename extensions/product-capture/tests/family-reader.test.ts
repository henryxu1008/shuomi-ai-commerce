import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { waitForFamilyDocument } from '../src/amazon/family-reader';

const target='https://www.amazon.com/dp/B0CC1YM6SJ';
let live:Partial<chrome.tabs.Tab>;
let updated:Set<(id:number)=>void>,removed:Set<(id:number)=>void>;
let update:ReturnType<typeof vi.fn>;
beforeEach(()=>{
 vi.useFakeTimers();live={url:target,status:'complete',openerTabId:41};updated=new Set();removed=new Set();
 update=vi.fn(async()=>live);
 vi.stubGlobal('chrome',{tabs:{get:vi.fn(async()=>({...live})),update,
  onUpdated:{addListener:(f:(id:number)=>void)=>updated.add(f),removeListener:(f:(id:number)=>void)=>updated.delete(f)},
  onRemoved:{addListener:(f:(id:number)=>void)=>removed.add(f),removeListener:(f:(id:number)=>void)=>removed.delete(f)}}});
});
afterEach(()=>{vi.useRealTimers();vi.unstubAllGlobals();});
const tick=()=>vi.advanceTimersByTimeAsync(0);
const clean=()=>{expect(updated.size).toBe(0);expect(removed.size).toBe(0);expect(vi.getTimerCount()).toBe(0);};
it('keeps the 30-second deadline, without navigation or an automatic retry',async()=>{
 live.status='loading';const read=waitForFamilyDocument(50,target,new AbortController().signal,false);
 const rejected=expect(read).rejects.toThrow('CAPTURE_TIMEOUT');
 await vi.advanceTimersByTimeAsync(29999);expect(updated.size).toBe(1);
 await vi.advanceTimersByTimeAsync(1);await rejected;expect(update).not.toHaveBeenCalled();clean();
});
it('waits for native pending navigation to settle even when the old page is complete',async()=>{
 live.pendingUrl=target+'?ref=pending';let complete=false;
 const read=waitForFamilyDocument(50,target,new AbortController().signal,false).then(()=>{complete=true;});
 await tick();expect(complete).toBe(false);
 live.pendingUrl=undefined;updated.forEach(f=>f(50));await read;expect(update).not.toHaveBeenCalled();clean();
});
it.each(['https://example.com/', 'https://www.amazon.com/dp/B000000003', 'https://www.amazon.ca/checkpoint/continue'])('rejects manual navigation to %s',async url=>{
 live.url=url;await expect(waitForFamilyDocument(50,target,new AbortController().signal,false)).rejects.toThrow('PAGE_CHANGED');clean();
});
it('a closed reader stops the pending read and releases the deadline/listeners',async()=>{
 live.status='loading';const read=waitForFamilyDocument(50,target,new AbortController().signal,false);
 const rejected=expect(read).rejects.toThrow('FAMILY_INTERRUPTED');await tick();removed.forEach(f=>f(50));await rejected;clean();
});
it('cancellation stops a pending load without navigation or a late completion',async()=>{
 live.status='loading';const abort=new AbortController();const read=waitForFamilyDocument(50,target,abort.signal,false);
 const rejected=expect(read).rejects.toThrow('CAPTURE_CANCELLED');await tick();abort.abort();await rejected;
 live.status='complete';updated.forEach(f=>f(50));expect(update).not.toHaveBeenCalled();clean();
});
