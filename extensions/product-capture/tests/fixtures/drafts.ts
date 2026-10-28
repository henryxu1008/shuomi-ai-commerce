import 'fake-indexeddb/auto';
import { DraftStore } from '../../src/draft-store';
import type { FamilyPorts } from '../../src/amazon/family-coordinator';
import { FamilyCoordinator } from '../../src/amazon/family-coordinator';
export const testDrafts = () => new DraftStore('test-' + crypto.randomUUID());
// Pure algorithm tests supply no-op ports, never a production storage fallback.
export const coordinator = (ports:FamilyPorts) => new FamilyCoordinator(ports,{resume:async()=>{},seed:async()=>{},observe:async()=>{},pause:async()=>{}});
