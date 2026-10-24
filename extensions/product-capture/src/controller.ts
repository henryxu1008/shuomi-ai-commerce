import type { PlatformCapturePayload } from './wire';
import { Handoff } from './handoff';
import { validateCapture } from './validation';

export type CaptureTab = Readonly<{ id?: number; url?: string; documentId?: string }>;
export type ReturnNavigation = Readonly<{
  prepare(sourceURL: string): Promise<void>;
  bind(tabId: number): Promise<void>;
  tabID(): number | undefined;
}>;

export class Controller {
  private payload?: PlatformCapturePayload;
  private capturing?: Promise<void>;
  private delivering?: Promise<void>;
  private entering?: Promise<void>;
  private returning?: Promise<void>;
  private handed = false;
  private deliveryFailed = false;
  private flow: Handoff;
  constructor(app: string, private readonly extract: (tab?: CaptureTab) => Promise<PlatformCapturePayload>,
    private readonly createTab: (url: string) => Promise<number>, private readonly extensionID: () => string,
    private readonly validate: (value: unknown) => PlatformCapturePayload = validateCapture,
    private readonly receiver: (payload: PlatformCapturePayload) => string = () => app,
    private readonly activateTab: (id: number) => Promise<void> = async () => { throw Error('ORIGINAL_TAB_UNAVAILABLE'); },
    private readonly navigation?: ReturnNavigation) {
    this.flow = new Handoff(app);
  }
  capture(tab?: CaptureTab): Promise<void> {
    if (this.capturing) return this.capturing;
    if (this.payload) return Promise.resolve();
    this.capturing = this.extract(tab).then(value => {
      const payload = this.validate(value);
      this.flow = new Handoff(this.receiver(payload));
      this.payload = payload;
    }).finally(() => { this.capturing = undefined; });
    return this.capturing;
  }
  handoff(): Promise<void> {
    if (this.delivering) return this.delivering;
    if (this.handed) return Promise.resolve();
    if (!this.payload) return Promise.reject(new Error('CAPTURE_REQUIRED'));
    this.flow.prepare(this.payload, crypto.randomUUID(), crypto.randomUUID());
    this.handed = true;
    const payload = this.payload;
    this.delivering = Promise.resolve().then(async () => {
      await this.navigation?.prepare(payload.evidence.sourceURL);
      const id = await this.createTab(this.flow.handoffURL(this.extensionID()));
      // Keep the receiver's bounded payload reads independent of storage latency.
      this.flow.bindTab(id);
      await this.navigation?.bind(id);
    }).catch(failure => { this.deliveryFailed = true; throw failure instanceof Error && failure.message === 'RETURN_STATE_UNAVAILABLE' ? failure : new Error('HANDOFF_UNAVAILABLE'); })
      .finally(() => { this.delivering = undefined; });
    return this.delivering;
  }
  captureAndHandoff(tab?: CaptureTab): Promise<void> {
    if (this.entering) return this.entering;
    // Reserve the whole entry before extraction can query or await a source tab.
    this.entering = Promise.resolve().then(async () => {
      if (this.handed) { await this.returnToApplication(); return; }
      await this.capture(tab);
      await this.handoff();
    }).finally(() => { this.entering = undefined; });
    return this.entering;
  }
  returnToApplication(): Promise<void> {
    if (this.returning) return this.returning;
    this.returning = Promise.resolve().then(async () => {
      const id = this.navigation ? this.navigation.tabID() : this.flow.applicationTabID();
      if (id === undefined) throw Error('ORIGINAL_TAB_UNAVAILABLE');
      try { await this.activateTab(id); }
      catch { throw Error('ORIGINAL_TAB_UNAVAILABLE'); }
    }).finally(() => { this.returning = undefined; });
    return this.returning;
  }
  isBusy() { return Boolean(this.entering || this.capturing || this.delivering || this.returning); }
  message(value: unknown, sender: chrome.runtime.MessageSender) { return this.flow.message(value, sender); }
  snapshot() {
    return { ...this.flow.snapshot(),
      ...(this.deliveryFailed ? { outcome: 'outcome_unknown' } : {}),
      loading: this.isBusy(), captured: Boolean(this.payload), handed: this.handed,
      title: this.payload?.evidence.title ?? null, sourceURL: this.payload?.evidence.sourceURL ?? null,
      missingFacts: this.payload?.evidence.missingFacts ?? [], recoveryURL: this.flow.recoveryURL(),
    };
  }
  recoveryURL() { return this.flow.recoveryURL(); }
}
