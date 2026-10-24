import { vi } from 'vitest';

// Browser-owned session data survives module/worker recreation, unlike globals.
export function returnStorage() {
  let values: Record<string, unknown> = {};
  const session = {
    get: vi.fn<(key: string) => Promise<Record<string, unknown>>>(),
    set: vi.fn<(items: Record<string, unknown>) => Promise<void>>(),
    remove: vi.fn<(key: string) => Promise<void>>(),
    setAccessLevel: vi.fn<(options: { accessLevel: string }) => Promise<void>>(),
  };
  const reset = () => {
    values = {};
    session.get.mockReset().mockImplementation(async key => Object.hasOwn(values, key) ? { [key]: structuredClone(values[key]) } : {});
    session.set.mockReset().mockImplementation(async items => { Object.assign(values, structuredClone(items)); });
    session.remove.mockReset().mockImplementation(async key => { delete values[key]; });
    session.setAccessLevel.mockReset().mockResolvedValue();
  };
  reset();
  return { session, reset, contents: () => structuredClone(values) };
}
