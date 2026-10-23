import type { PlatformCapturePayload } from './wire';
import { Handoff } from './handoff';
import { validateCapture } from './validation';

export class Controller {
  private payload?: PlatformCapturePayload;
  private capturing?: Promise<void>;
  private delivering?: Promise<void>;
  private handed = false;
  private deliveryFailed = false;
  private flow: Handoff;
  constructor(app: string, private readonly extract: () => Promise<PlatformCapturePayload>,
    private readonly createTab: (url: string) => Promise<number>, private readonly extensionID: () => string,
    private readonly validate: (value: unknown) => PlatformCapturePayload = validateCapture,
    private readonly receiver: (payload: PlatformCapturePayload) => string = () => app) {
    this.flow = new Handoff(app);
  }
  capture(): Promise<void> {
    if (this.capturing) return this.capturing;
    if (this.payload) return Promise.resolve();
    this.capturing = this.extract().then(value => {
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
    this.delivering = this.createTab(this.flow.handoffURL(this.extensionID())).then(id => this.flow.bindTab(id))
      .catch(() => { this.deliveryFailed = true; throw new Error('HANDOFF_UNAVAILABLE'); });
    return this.delivering;
  }
  message(value: unknown, sender: chrome.runtime.MessageSender) { return this.flow.message(value, sender); }
  snapshot() {
    return { ...this.flow.snapshot(),
      ...(this.deliveryFailed ? { outcome: 'outcome_unknown' } : {}),
      loading: Boolean(this.capturing), captured: Boolean(this.payload), handed: this.handed,
      title: this.payload?.evidence.title ?? null, sourceURL: this.payload?.evidence.sourceURL ?? null,
      missingFacts: this.payload?.evidence.missingFacts ?? [], recoveryURL: this.flow.recoveryURL(),
    };
  }
  recoveryURL() { return this.flow.recoveryURL(); }
}
