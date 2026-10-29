import type { PlatformCapturePayload } from './wire';
import { Handoff } from './handoff';
import { validateCapture } from './validation';
import { capturePageSource } from './platforms';
import { DraftStore, type Draft } from './draft-store';

export type CaptureTab = Readonly<{ id?: number; url?: string; documentId?: string }>;
export type ReturnNavigation = Readonly<{
  prepare(sourceURL: string): Promise<void>;
  bind(tabId: number): Promise<void>;
  tabID(): number | undefined;
  open?(sourceURL: string, url: string): Promise<number>;
}>;
export class Controller {
  private draft?: Draft;
  private capturing?: Promise<void>;
  private delivering?: Promise<void>;
  private entering?: Promise<void>;
  private returning?: Promise<void>;
  private handed = false;
  private deliveryFailed = false;
  private flow: Handoff;
  readonly ready: Promise<void>;
  constructor(private readonly drafts: DraftStore, private readonly locate: () => Promise<CaptureTab>,
    app: string, private readonly extract: (tab?: CaptureTab) => Promise<PlatformCapturePayload>,
    private readonly createTab: (url: string) => Promise<number>, private readonly extensionID: () => string,
    private readonly validate: (value: unknown) => PlatformCapturePayload = validateCapture,
    private readonly receiver: (sourceURL: string) => string = () => app,
    private readonly activateTab: (id: number) => Promise<void> = async () => { throw Error('ORIGINAL_TAB_UNAVAILABLE'); },
    private readonly navigation?: ReturnNavigation, restoreLatest = true) {
    this.flow = new Handoff(app, drafts);
    this.ready = restoreLatest ? this.load() : Promise.resolve();
  }
  private async load() {
    const d = await this.drafts.latest();
    if (d?.payload && this.receiver(d.sourceURL) === d.app) { this.draft = d; this.flow = new Handoff(d.app, this.drafts); }
  }
  async restore(id: string) {
    await this.ready; const d = await this.drafts.get(id);
    if (!d?.payload || d.app !== this.receiver(d.sourceURL)) throw Error('DRAFT_CHANGED');
    this.draft = d; this.flow = new Handoff(d.app, this.drafts); this.handed = false;
  }
  capture(tab?: CaptureTab): Promise<void> {
    if (this.capturing) return this.capturing;
    this.capturing = (async () => {
      await this.ready; if (this.draft?.payload) return;
      const target = tab ?? await this.locate();
      if (!target.url) throw Error('INVALID_PAGE');
      const source = capturePageSource(target.url);
      let d = await this.drafts.create(source.sourceURL, 'single', this.receiver(source.sourceURL));
      this.draft = d;
      try {
        const payload = this.validate(await this.extract(target));
        d = await this.drafts.complete(d.id, d.revision, d.generation, payload);
        this.draft = d; this.flow = new Handoff(d.app, this.drafts);
      } catch (e) {
        this.draft = await this.drafts.pause(d.id, d.revision, d.generation, 'CAPTURE_REJECTED');
        throw e;
      }
    })().finally(() => { this.capturing = undefined; });
    return this.capturing;
  }
  async acceptPrepared(id: string) {
    if (this.isBusy() || this.draft?.payload || this.handed) throw Error('ACTION_IN_PROGRESS');
    await this.restore(id);
  }
  handoff(): Promise<void> {
    if (this.delivering) return this.delivering;
    if (this.handed&&this.flow.canReturn()) return Promise.resolve();
    this.delivering = (async () => {
      await this.ready;
      let d = this.draft && await this.drafts.get(this.draft.id);
      if (!d?.payload) throw Error('CAPTURE_REQUIRED');
      if(d.phase==='sending')d=await this.drafts.interrupt(d.id,d.revision,d.generation);
      this.flow = new Handoff(d.app, this.drafts);
      this.flow.prepare(d, crypto.randomUUID()); this.handed = true;
      try {
        const url = this.flow.handoffURL(this.extensionID());
        let id: number;
        if (this.navigation?.open) id = await this.navigation.open(d.sourceURL, url);
        else {
          await this.navigation?.prepare(d.sourceURL); id = await this.createTab(url); await this.navigation?.bind(id);
        }
        this.flow.bindTab(id); this.deliveryFailed = false;
      } catch (e) {
        this.deliveryFailed = true;
        throw e instanceof Error && e.message === 'RETURN_STATE_UNAVAILABLE' ? e : Error('HANDOFF_UNAVAILABLE');
      }
    })().finally(() => { this.delivering = undefined; });
    return this.delivering;
  }
  captureAndHandoff(tab?: CaptureTab): Promise<void> {
    if (this.entering) return this.entering;
    this.entering = Promise.resolve().then(async () => {
      await this.ready;if(this.handed){await this.returnToApplication();return;}await this.capture(tab); await this.handoff();
    }).finally(() => { this.entering = undefined; });
    return this.entering;
  }
  returnToApplication(): Promise<void> {
    if (this.returning) return this.returning;
    this.returning = (async () => {
      await this.ready;
      if(this.flow.canReturn()) {try{await this.activateTab(this.flow.applicationTabID()!);return;}catch{throw Error('ORIGINAL_TAB_UNAVAILABLE');}}
      // Explicit user restore issues a fresh short capability for the original
      // durable key/body. It never extracts again or submits to the backend.
      if (this.draft?.payload) { await this.handoff(); return; }
      const id = this.navigation?.tabID();
      if (id === undefined) throw Error('ORIGINAL_TAB_UNAVAILABLE');
      try { await this.activateTab(id); } catch { throw Error('ORIGINAL_TAB_UNAVAILABLE'); }
    })().finally(() => { this.returning = undefined; });
    return this.returning;
  }
  isBusy() { return Boolean(this.entering || this.capturing || this.delivering || this.returning); }
  async message(value: unknown, sender: chrome.runtime.MessageSender) { await this.ready; return this.flow.message(value, sender); }
  snapshot() {
    const payload = this.draft?.payload;
    return { ...this.flow.snapshot(), ...(this.deliveryFailed ? { outcome: 'outcome_unknown' } : {}),
      loading: this.isBusy(), captured: Boolean(payload), handed: this.handed,
      title: payload?.evidence.title ?? null, sourceURL: this.draft?.sourceURL ?? null,
      missingFacts: payload?.evidence.missingFacts ?? [], recoveryURL: this.flow.recoveryURL() };
  }
  recoveryURL() { return this.flow.recoveryURL(); }
  get draftID() {return this.draft?.id;}
}
