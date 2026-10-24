import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ReturnSession } from '../src/return-session';
import { returnStorage } from './fixtures/return-storage';

const storage = returnStorage();
const source = 'https://www.amazon.com/dp/B0BDLGZCTY';
const revision = crypto.randomUUID();
const key = 'product-capture:return-location:v1';
const pending = { version: 1, revision, sourceURL: source, phase: 'pending' };
const open = async () => { const value = new ReturnSession(chrome.storage.session); await value.ready; return value; };
beforeEach(() => { storage.reset(); vi.stubGlobal('chrome', { storage }); });
afterEach(() => vi.unstubAllGlobals());

it('stores only bounded original-tab location in trusted session storage and hydrates the same entry', async () => {
  const value = await open();
  expect(value.state).toBe('none');
  await value.prepare(source, revision);
  expect(storage.contents()).toEqual({ [key]: pending });
  await value.bind(42, revision);
  expect(storage.contents()).toEqual({ [key]: { ...pending, phase: 'bound', tabId: 42 } });
  const recreated = await open();
  expect(recreated.state).toBe('available'); expect(recreated.tabID()).toBe(42);
  expect(recreated.revision).toBe(revision); expect(recreated.sourceURL).toBe(source);
  expect(storage.session.setAccessLevel).toHaveBeenCalledWith({ accessLevel: 'TRUSTED_CONTEXTS' });
});

it('does not replace a known original location without explicit clear', async () => {
  const value = await open(); await value.prepare(source, revision); await value.bind(42, revision);
  await expect(value.prepare('https://detail.1688.com/offer/9.html', crypto.randomUUID())).rejects.toThrow('ENTRY_CHANGED');
  expect(value.tabID()).toBe(42); expect(storage.session.set).toHaveBeenCalledTimes(2);
  await value.clear();
  expect(storage.contents()).toEqual({}); expect(value.state).toBe('none');
});

it('hydrates interrupted pending binding as uncertain, never as an empty entry', async () => {
  const value = await open(); await value.prepare(source, revision);
  const recreated = await open();
  expect(recreated.state).toBe('uncertain'); expect(recreated.tabID()).toBeUndefined();
  expect(recreated.revision).toBe(revision); expect(storage.session.remove).not.toHaveBeenCalled();
});

it.each([
  null, undefined, { ...pending, payload: {} }, { ...pending, sourceURL: 'https://www.amazon.com/dp/B0BDLGZCTY?tag=extra' },
  { ...pending, sourceURL: 'javascript:alert(1)' }, { ...pending, sourceURL: 'https://evil.example/dp/B0BDLGZCTY' },
  { ...pending, sourceURL: source+'#key=private' }, { ...pending, revision: 'old' },
  { ...pending, phase: 'bound' }, { ...pending, phase: 'bound', tabId: -1 },
  { ...pending, phase: 'bound', tabId: 1.1 }, { ...pending, tabId: 42 }, { ...pending, version: 2 },
])('does not interpret a malformed session record as initial capture: %#', async record => {
  storage.session.get.mockResolvedValueOnce({ [key]: record });
  const value = await open();
  expect(value.state).toBe('uncertain'); expect(value.tabID()).toBeUndefined();
  await expect(value.prepare(source, revision)).rejects.toThrow('RETURN_STATE_UNAVAILABLE');
  await expect(value.clear()).rejects.toThrow('RETURN_STATE_UNAVAILABLE');
  expect(storage.session.set).not.toHaveBeenCalled(); expect(storage.session.remove).not.toHaveBeenCalled();
});

it.each(['get', 'setAccessLevel'] as const)('fails closed on %s initialization error', async method => {
  storage.session[method].mockRejectedValueOnce(Error('secret underlying error'));
  const value = await open();
  expect(value.state).toBe('uncertain');
  expect(() => value.assertReadable()).toThrow('RETURN_STATE_UNAVAILABLE');
  expect(storage.session.set).not.toHaveBeenCalled();
});

it.each(['prepare', 'bind', 'clear'] as const)('keeps an unknown %s write from enabling new work', async action => {
  const value = await open();
  if (action !== 'prepare') await value.prepare(source, revision);
  if (action === 'clear') storage.session.remove.mockRejectedValueOnce(Error('lost write response'));
  else storage.session.set.mockRejectedValueOnce(Error('lost write response'));
  await expect(action === 'prepare' ? value.prepare(source, revision) : action === 'bind' ? value.bind(42, revision) : value.clear()).rejects.toThrow('RETURN_STATE_UNAVAILABLE');
  expect(value.state).toBe('uncertain'); expect(value.tabID()).toBeUndefined();
  await expect(value.clear()).rejects.toThrow('RETURN_STATE_UNAVAILABLE');
});

it('refuses an old binding completion without overwriting the current navigation record', async () => {
  const value = await open(); await value.prepare(source, revision);
  await expect(value.bind(42, crypto.randomUUID())).rejects.toThrow('ENTRY_CHANGED');
  expect(storage.contents()).toEqual({ [key]: pending });
});
