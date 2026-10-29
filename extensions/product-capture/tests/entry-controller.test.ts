import { testDrafts } from './fixtures/drafts';
import { readFileSync } from 'node:fs';
import { afterEach, expect, it, vi } from 'vitest';
import { Controller } from '../src/controller';
import type { PlatformCapturePayload } from '../src/wire';

const payload = JSON.parse(readFileSync('tests/fixtures/capture-v2.json', 'utf8')) as PlatformCapturePayload;
const target = { id: 7, url: payload.evidence.sourceURL };
function fixture(extract = vi.fn(async () => payload), create = vi.fn(async () => 42), focus = vi.fn<(id: number) => Promise<void>>(async () => {})) {
  return { extract, create, focus, control: new Controller(testDrafts(),async()=>target,'https://app.example.com/capture/1688', extract, create, () => 'a'.repeat(32), undefined, undefined, focus) };
}
afterEach(() => vi.restoreAllMocks());

it('coalesces the whole capture and delivery, retaining the first triggering tab', async () => {
  let complete!: (value: PlatformCapturePayload) => void;
  let deliver!: (value: number) => void;
  const extract = vi.fn(() => new Promise<PlatformCapturePayload>(resolve => { complete = resolve; }));
  const create = vi.fn(() => new Promise<number>(resolve => { deliver = resolve; }));
  const f = fixture(extract, create);
  const first = f.control.captureAndHandoff(target);
  const second = f.control.captureAndHandoff({ id: 8, url: 'https://www.amazon.com/dp/B012345678' });
  expect(first).toBe(second);
  await vi.waitFor(() => expect(extract).toHaveBeenCalledOnce());
  expect(extract).toHaveBeenCalledWith(target);
  expect(f.control.isBusy()).toBe(true);
  complete(payload);
  await vi.waitFor(() => expect(create).toHaveBeenCalledOnce());
  expect(f.control.isBusy()).toBe(true);
  expect(f.control.captureAndHandoff(target)).toBe(first);
  deliver(42);
  await Promise.all([first, second]);
  expect(f.control.isBusy()).toBe(false);
  expect(f.control.snapshot().handed).toBe(true);
  expect(create).toHaveBeenCalledOnce();
});

it('returns to the original tab after handoff or expiry without extracting, creating or rekeying', async () => {
  const f = fixture();
  await f.control.captureAndHandoff(target);
  const key = f.control.recoveryURL();
  vi.spyOn(Date, 'now').mockReturnValue(Date.now() + 600_000);
  await f.control.captureAndHandoff({ id: 99, url: 'https://unrelated.example/' });
  expect(f.focus).toHaveBeenCalledExactlyOnceWith(42);
  expect(f.extract).toHaveBeenCalledOnce();
  expect(f.create).toHaveBeenCalledOnce();
  expect(f.control.recoveryURL()).toBe(key);
});

it('coalesces return requests and stays busy until activation settles', async () => {
  let finish!: () => void;
  const focus = vi.fn(() => new Promise<void>(resolve => { finish = resolve; }));
  const f = fixture(undefined, undefined, focus);
  await f.control.captureAndHandoff(target);
  const first = f.control.returnToApplication(), second = f.control.returnToApplication();
  expect(first).toBe(second);
  await vi.waitFor(() => expect(focus).toHaveBeenCalledOnce());
  expect(f.control.isBusy()).toBe(true);
  finish(); await Promise.all([first, second]);
  expect(f.control.isBusy()).toBe(false);
});

it('preserves the original operation when the original tab/window cannot be activated', async () => {
  const focus = vi.fn(async () => { throw Error('a browser detail must not become an API contract'); });
  const f = fixture(undefined, undefined, focus);
  await f.control.captureAndHandoff(target);
  const key = f.control.recoveryURL();
  await expect(f.control.captureAndHandoff(target)).rejects.toThrow('ORIGINAL_TAB_UNAVAILABLE');
  expect(f.control.recoveryURL()).toBe(key);
  expect(f.control.snapshot().handed).toBe(true);
  expect(f.extract).toHaveBeenCalledOnce(); expect(f.create).toHaveBeenCalledOnce();
});

it('explicitly retries opening the retained draft after its first tab creation failed, without another extraction', async () => {
  const f = fixture(undefined, vi.fn(async () => { throw Error('creation unavailable'); }));
  await expect(f.control.captureAndHandoff(target)).rejects.toThrow('HANDOFF_UNAVAILABLE');
  const key = f.control.recoveryURL();
  await expect(f.control.captureAndHandoff(target)).rejects.toThrow('HANDOFF_UNAVAILABLE');
  expect(f.control.recoveryURL()).toBe(key);
  expect(f.control.snapshot().outcome).toBe('outcome_unknown');
  expect(f.extract).toHaveBeenCalledOnce(); expect(f.create).toHaveBeenCalledTimes(2); expect(f.focus).not.toHaveBeenCalled();
});
