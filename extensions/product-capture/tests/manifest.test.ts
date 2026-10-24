import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { expect, it } from 'vitest';

const hosts = ['https://detail.1688.com/*', 'https://www.amazon.com/*', 'https://www.amazon.ca/*', 'https://www.amazon.co.uk/*', 'https://www.amazon.de/*', 'https://www.amazon.co.jp/*'];
const receivers = (origin: string) => ({matches:[`${origin}/capture/1688`,`${origin}/capture/amazon`]});

it('builds only activeTab/scripting/storage with exact app messaging, and rejects implicit or unsafe destinations', () => {
  const build=resolve('scripts/build.mjs');
  const env={...process.env};delete env.CAPTURE_APP_URL;
  expect(()=>execFileSync(process.execPath,[build],{env,stdio:'pipe'})).toThrow();
  for(const destination of ['http://app.example.com/capture/1688','https://user:password@app.example.com/capture/1688','https://app.example.com/other']) {
    expect(()=>execFileSync(process.execPath,[build],{env:{...env,CAPTURE_APP_URL:destination},stdio:'pipe'})).toThrow();
  }
  execFileSync(process.execPath,[build,'--fixture'],{env:{...env,CAPTURE_APP_URL:'http://127.0.0.1:4399/capture/1688'},stdio:'pipe'});
  const manifest=JSON.parse(readFileSync('dist-fixture/manifest.json','utf8'));
  expect(manifest.permissions).toEqual(['activeTab','scripting','storage']);
  expect(manifest.commands).toEqual({ 'capture-current-product': { suggested_key: { default: 'Alt+Shift+P' }, description: '采集当前商品并在应用确认' } });
  expect(manifest.action.default_popup).toBe('popup.html');
  // Exact approved origins only; each source parser also validates the product
  // path before injection. No subdomain wildcard or implicit new destination.
  expect(manifest.host_permissions).toEqual(hosts);
  for(const key of ['optional_permissions','optional_host_permissions','web_accessible_resources']) expect(manifest[key]).toBeUndefined();
  expect(manifest.content_scripts).toEqual([{ matches: ['https://detail.1688.com/offer/*', ...hosts.slice(1).flatMap(host => [host.replace('/*','/dp/*'), host.replace('/*','/*/dp/*')])], js:['page.js'], run_at:'document_idle', all_frames:false, world:'ISOLATED' }]);
  expect(manifest.externally_connectable).toEqual(receivers('http://127.0.0.1'));
  expect(manifest.content_security_policy.extension_pages).toContain("connect-src 'none'");
  expect(manifest.content_security_policy.extension_pages).not.toMatch(/unsafe-inline|unsafe-eval/);
  // The fixture build is not what ships. Assert the real build separately, so a permission
  // cannot be added to one and silently missed in the other.
  execFileSync(process.execPath,[build],{env:{...env,CAPTURE_APP_URL:'https://app.example.com/capture/1688'},stdio:'pipe'});
  const shipped=JSON.parse(readFileSync('dist/manifest.json','utf8'));
  expect(shipped.permissions).toEqual(manifest.permissions);
  expect(shipped.commands).toEqual(manifest.commands);
  expect(shipped.content_scripts).toEqual(manifest.content_scripts);
  expect(shipped.host_permissions).toEqual(manifest.host_permissions);
  expect(shipped.externally_connectable).toEqual(receivers('https://app.example.com'));
  execFileSync(process.execPath,[build],{env:{...env,CAPTURE_APP_URL:'https://localhost:31544/capture/1688'},stdio:'pipe'});
  const local=JSON.parse(readFileSync('dist/manifest.json','utf8'));
  expect(local.name).toBe(shipped.name);
  expect(local.permissions).toEqual(shipped.permissions);
  expect(local.host_permissions).toEqual(shipped.host_permissions);
  expect(local.externally_connectable).toEqual(receivers('https://localhost'));
  expect(readFileSync('dist/background.js','utf8')).toContain('https://localhost:31544/capture/1688');
  expect(readFileSync('dist/background.js','utf8')).toContain('https://localhost:31544/capture/amazon');
},20000);
