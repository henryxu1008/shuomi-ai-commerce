import assert from 'node:assert/strict';
import { test } from 'node:test';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, copyFileSync, existsSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scripts = path.dirname(fileURLToPath(import.meta.url));
const repo = path.dirname(scripts);
const wrapper = `param([string]$BuildScript,[string]$OutputDirectory,[string]$SourceRef='HEAD',[string]$Component='All',[string]$BackendTarget='')
$global:revision=''; $global:tree=''; $global:builtImage=''; $global:builtTargetLabel=''
function global:docker {
 $a=@($args); $global:LASTEXITCODE=0
 if($a[0] -eq 'build') {
  $context=$a[-1]; $global:builtImage=$a[[array]::IndexOf($a,'--tag')+1]; $global:builtTargetLabel=''
  foreach($v in $a) {if($v -like 'org.opencontainers.image.revision=*'){$global:revision=$v.Split('=')[1]};if($v -like 'org.opencontainers.image.tree=*'){$global:tree=$v.Split('=')[1]};if($v -like 'io.shuomi.build.backend-target=*'){$global:builtTargetLabel=$v.Split('=')[1]}}
  [ordered]@{arguments=$a;tracked=(Get-Content -LiteralPath (Join-Path $context 'marker') -Raw);privatePresent=(Test-Path -LiteralPath (Join-Path $context 'private.json'))} | ConvertTo-Json -Compress -Depth 5 | Add-Content -LiteralPath $env:FIXTURE_CALLS
  if($env:FIXTURE_FAULT -eq 'build'){$global:LASTEXITCODE=23}; return
 }
 if($a[0] -eq 'image') {
  $targetLabel=if($env:FIXTURE_FAULT -eq 'target'){'incorrect-target'}else{$global:builtTargetLabel}
  @([ordered]@{Id=('sha256:' + ('1'*64));Size=1234;RootFS=@{Layers=@('base','app')};Config=@{Labels=@{'org.opencontainers.image.revision'=$global:revision;'org.opencontainers.image.tree'=$global:tree;'io.shuomi.build.backend-target'=$targetLabel}}}) | ConvertTo-Json -Depth 6 -Compress; return
 }
 if($a[0] -eq 'history') {
  $copy=if($global:builtImage -like '*current-application*'){'COPY /out/current-application /usr/local/bin/current-application # buildkit'}else{'COPY /app/.next/standalone ./ # buildkit'}
  @{CreatedBy=$copy} | ConvertTo-Json -Compress
  if($env:FIXTURE_FAULT -eq 'stacked'){@{CreatedBy=$copy}|ConvertTo-Json -Compress}; return
 }
 throw ('Unexpected Docker operation: '+($a -join ' '))
}
$selection=@{}; if($BackendTarget){$selection.BackendTarget=$BackendTarget}
& $BuildScript -SourceRef $SourceRef -Component $Component -OutputDirectory $OutputDirectory @selection
`;

function fixture(run) {
  const dir = mkdtempSync(path.join(tmpdir(), 'listingkit-build-test-'));
  try {
    mkdirSync(path.join(dir, 'scripts'));
    mkdirSync(path.join(dir, 'deployments/docker'), { recursive: true });
    copyFileSync(path.join(scripts, 'build-listingkit-images.ps1'), path.join(dir, 'scripts/build.ps1'));
    for (const name of ['Dockerfile.account-compose', 'Dockerfile.listingkit-ui']) {
      copyFileSync(path.join(repo, 'deployments/docker', name), path.join(dir, 'deployments/docker', name));
    }
    writeFileSync(path.join(dir, 'marker'), 'committed source');
    const git = (...args) => execFileSync('git', ['-C', dir, ...args], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
    git('init'); git('add', '.');
    git('-c', 'user.name=Build Test', '-c', 'user.email=build-test@example.invalid', 'commit', '-m', 'fixture');
    const revision = git('rev-parse', 'HEAD');
    writeFileSync(path.join(dir, 'marker'), 'dirty source must not be built');
    writeFileSync(path.join(dir, 'private.json'), 'untracked private fixture');
    writeFileSync(path.join(dir, 'invoke.ps1'), wrapper);
    const output = path.join(dir, 'receipts');
    const calls = path.join(dir, 'calls.jsonl');
    const invoke = (fault = '', source = 'HEAD', component = 'All', backendTarget = '') => spawnSync('pwsh', [
      '-NoProfile', '-File', path.join(dir, 'invoke.ps1'), '-BuildScript', path.join(dir, 'scripts/build.ps1'),
      '-OutputDirectory', output, '-SourceRef', source, '-Component', component, '-BackendTarget', backendTarget,
    ], { encoding: 'utf8', env: { ...process.env, FIXTURE_CALLS: calls, FIXTURE_FAULT: fault } });
    run({ dir, output, calls, revision, invoke });
  } finally {
    assert.equal(path.dirname(path.resolve(dir)), path.resolve(tmpdir()));
    assert.match(path.basename(dir), /^listingkit-build-test-/);
    rmSync(dir, { recursive: true, force: true });
  }
}

test('builds both canonical targets from committed source, excluding dirty and private files', () => fixture(({ output, calls, revision, invoke }) => {
  const result = invoke();
  assert.equal(result.status, 0, result.stderr);
  const operations = readFileSync(calls, 'utf8').trim().split(/\r?\n/).map(JSON.parse);
  assert.equal(operations.length, 2);
  for (const op of operations) {
    assert.equal(op.tracked, 'committed source');
    assert.equal(op.privatePresent, false);
    assert.equal(existsSync(op.arguments.at(-1)), false, 'temporary source cleaned');
    assert.equal(op.arguments.includes('--push'), false);
  }
  assert.ok(operations[0].arguments.includes('data-services-runtime'));
  assert.match(operations[1].arguments.join(' '), /Dockerfile.listingkit-ui/);
  const receipt = JSON.parse(readFileSync(path.join(output, `build-${revision}-All.json`)));
  assert.equal(receipt.sourceCommit, revision);
  assert.equal(receipt.images.length, 2);
  assert.equal(receipt.images[0].sizeBytes, 1234);
  assert.equal(receipt.images[0].target, 'data-services-runtime');
  assert.equal(receipt.images[0].tag, `local/current-application:git-${revision}`);
}));

test('explicit runtime has separate backend artifacts while retaining the same UI', () => fixture(({ output, calls, revision, invoke }) => {
  assert.equal(invoke().status, 0);
  const defaultPath = path.join(output, `build-${revision}-All.json`);
  const previous = readFileSync(defaultPath, 'utf8');
  const result = invoke('', 'HEAD', 'All', 'runtime');
  assert.equal(result.status, 0, result.stderr);
  assert.equal(readFileSync(defaultPath, 'utf8'), previous);
  const defaults = JSON.parse(previous);
  const receipt = JSON.parse(readFileSync(path.join(output, `build-${revision}-All-runtime.json`)));
  assert.equal(receipt.images[0].target, 'runtime');
  assert.equal(receipt.images[0].tag, `local/current-application:git-${revision}-runtime`);
  assert.equal(receipt.images[1].tag, defaults.images[1].tag);
  const operations = readFileSync(calls, 'utf8').trim().split(/\r?\n/).map(JSON.parse);
  assert.equal(operations.length, 4);
  const args = operations[2].arguments;
  assert.equal(args[args.indexOf('--target') + 1], 'runtime');
  assert.ok(args.includes('io.shuomi.build.backend-target=runtime'));
  assert.equal(operations[3].arguments.includes('--target'), false);
}));

test('explicit target failure stops without fallback or a success receipt', () => fixture(({ output, calls, revision, invoke }) => {
  const result = invoke('build', 'HEAD', 'All', 'runtime');
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /image build failed/);
  const operations = readFileSync(calls, 'utf8').trim().split(/\r?\n/).map(JSON.parse);
  assert.equal(operations.length, 1);
  assert.equal(operations[0].arguments[operations[0].arguments.indexOf('--target') + 1], 'runtime');
  assert.equal(existsSync(path.join(output, `build-${revision}-All-runtime.json`)), false);
  assert.equal(existsSync(path.join(output, `build-${revision}-All.json`)), false);
}));

test('UI-only build ignores the backend selection for tags and receipts', () => fixture(({ output, calls, revision, invoke }) => {
  const result = invoke('', 'HEAD', 'UI', 'runtime');
  assert.equal(result.status, 0, result.stderr);
  const operations = readFileSync(calls, 'utf8').trim().split(/\r?\n/).map(JSON.parse);
  assert.equal(operations.length, 1);
  assert.equal(operations[0].arguments.includes('--target'), false);
  const receipt = JSON.parse(readFileSync(path.join(output, `build-${revision}-UI.json`)));
  assert.equal(receipt.images[0].tag, `local/listingkit-ui:git-${revision}`);
  assert.equal(receipt.images[0].target, undefined);
}));

for (const [name, fault, source, error, component = 'All'] of [
  ['unknown source', '', 'missing-ref', /SourceRef must resolve/],
  ['Docker failure', 'build', 'HEAD', /image build failed/],
  ['incorrect backend target label', 'target', 'HEAD', /Backend image target label does not match/],
  ['historical application layers', 'stacked', 'HEAD', /exactly one application COPY/],
  ['historical UI layers', 'stacked', 'HEAD', /exactly one application COPY/, 'UI'],
]) {
  test(`rejects ${name} without writing a success receipt`, () => fixture(({ output, revision, invoke }) => {
    const result = invoke(fault, source, component);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, error);
    assert.equal(existsSync(path.join(output, `build-${revision}-${component}.json`)), false);
  }));
}

test('does not overwrite a previous success receipt', () => fixture(({ output, revision, invoke }) => {
  assert.equal(invoke().status, 0);
  const filename = path.join(output, `build-${revision}-All.json`);
  const previous = readFileSync(filename, 'utf8');
  const result = invoke();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /receipt already exists/);
  assert.equal(readFileSync(filename, 'utf8'), previous);
}));

test('browser base excludes application source and canonical serving target adds one binary', () => {
  const dockerfile = readFileSync(path.join(repo, 'deployments/docker/Dockerfile.account-compose'), 'utf8');
  const stages = dockerfile.split(/^FROM /m);
  const installer = stages.find(x => x.startsWith('go-modules AS playwright-installer-builder'));
  assert.ok(installer);
  assert.doesNotMatch(installer, /COPY (internal|cmd\/current-application)/);
  const base = stages.find(x => x.includes(' AS data-services-runtime-base'));
  assert.ok(base);
  assert.doesNotMatch(base, /\/out\/current-application/);
  const serving = stages.find(x => x.startsWith('data-services-runtime-base AS data-services-runtime'));
  assert.equal((serving.match(/^COPY /gm) || []).length, 1);
  assert.match(serving, /--from=current-application-builder/);
});
