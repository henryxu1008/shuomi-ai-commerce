const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { classifyChangedPaths, formatOutputs } = require("./ci-change-classifier.cjs");

test("runs all suites for main push mode", () => {
  assert.deepEqual(classifyChangedPaths([], { full: true }), {
    backend: true,
    frontend: true,
    code_health: true,
    release_authority: true,
    capture_extension: true,
    isolated_runtime: true,
  });
});

test("backend-only changes skip frontend", () => {
  assert.deepEqual(classifyChangedPaths(["internal/product/sourcing/service.go"]), {
    backend: true,
    frontend: false,
    code_health: true,
    release_authority: false,
    capture_extension: false,
    isolated_runtime: false,
  });
});

test("frontend-only changes skip backend", () => {
  assert.deepEqual(classifyChangedPaths(["web/listingkit-ui/src/app/page.tsx"]), {
    backend: false,
    frontend: true,
    code_health: true,
    release_authority: false,
    capture_extension: false,
    isolated_runtime: false,
  });
});

test("Markdown authority and repository documentation leave runtime backend checks to code changes", () => {
  for (const path of [
    "AGENTS.md",
    "README.md",
    "docs/architecture/project-target-architecture.md",
    "docs/refactoring/legacy-register.md",
    "docs/product/final-ui-ia-authority.md",
  ]) {
    const result = classifyChangedPaths([path]);
    assert.equal(result.backend, false, path);
    assert.equal(result.frontend, false, path);
    assert.equal(result.code_health, false, `${path} should not run code health`);
  }
});

test("code-health document rename preserves its dedicated policy check", () => {
  for (const changedPath of [
    "docs/refactoring/code-health-decisions.md",
    "docs/engineering/code-health-decisions.md",
    "docs/engineering/code-health-rules.md",
  ]) {
    assert.deepEqual(classifyChangedPaths([changedPath]), {
      backend: false,
      frontend: false,
      code_health: true,
      release_authority: false,
      capture_extension: false,
      isolated_runtime: false,
    }, changedPath);
  }
});

test("release policy and workflow changes run their Go-owned contract tests", () => {
  for (const changedPath of [
    ".github/workflows/listingkit-deploy.yml",
    "policy/listingkit-release-authority/rules.yaml",
  ]) {
    const result = classifyChangedPaths([changedPath]);
    assert.equal(result.release_authority, true, changedPath);
    assert.equal(result.backend, true, changedPath);
    assert.equal(result.frontend, false, changedPath);
  }
});

test("workbench release documentation and manifests run release plus backend contracts", () => {
  for (const changedPath of [
    "deployments/kubernetes/listingkit-workbench/README.md",
    "deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-deployment.yaml",
  ]) {
    const result = classifyChangedPaths([changedPath]);
    assert.equal(result.release_authority, true, changedPath);
    assert.equal(result.backend, true, changedPath);
    assert.equal(result.frontend, false, changedPath);
  }
});

test("scripts run backend and code-health safety nets", () => {
  for (const changedPath of [
    "scripts/listingkit-shein-pod-image-index-backfill/main.go",
    "scripts/listingkit-shein-pod-image-index-backfill/main_test.go",
    "scripts/code-health-audit.config.json",
  ]) {
    const result = classifyChangedPaths([changedPath]);
    assert.equal(result.backend, true, changedPath);
    assert.equal(result.code_health, true, changedPath);
    assert.equal(result.frontend, false, changedPath);
  }
});

test("hack debug nested module runs backend and code-health safety nets", () => {
  const changedPath = "hack/debug/listingkit-phone-onboarding-preflight/main.go";
  const result = classifyChangedPaths([changedPath]);
  assert.equal(result.backend, true);
  assert.equal(result.code_health, true);
  assert.equal(result.frontend, false);
});

test("a document in a mixed PR keeps the affected runtime checks", () => {
  const docs = ["AGENTS.md", "README.md", "docs/product/final-ui-ia-authority.md"];
  const docsOnly = classifyChangedPaths(docs);
  assert.ok(Object.values(docsOnly).every((selected) => selected === false));
  for (const changedPath of ["go.mod", "internal/app/httpapi/server_auth.go", "web/listingkit-ui/pnpm-lock.yaml",
    "policy/listingkit-release-authority/rules.yaml", "Makefile", "deployments/docker/Dockerfile.product-listing-api"]) {
    assert.deepEqual(classifyChangedPaths([...docs, changedPath]), classifyChangedPaths([changedPath]), changedPath);
    assert.ok(Object.values(classifyChangedPaths([changedPath])).some(Boolean), changedPath);
  }
});

test("the CI workflow, its classifier, and unknown paths select the full matrix", () => {
  const full = classifyChangedPaths([], { full: true });
  for (const changedPath of [
    ".github/workflows/ci.yml",
    ".github/scripts/ci-change-classifier.cjs",
    ".github/scripts/ci-change-classifier.test.cjs",
    "docs/architecture/new-contract.json",
    "Docs/product/guide.md",
    " docs/product/guide.md",
    "extensions/other/main.ts",
    "unknown/build-input",
  ]) {
    assert.deepEqual(classifyChangedPaths(["README.md", changedPath]), full, changedPath);
  }
  assert.throws(() => classifyChangedPaths([]));
});

test("repository CI uses current identity while preserving the stable required check", () => {
  const workflowPath = path.join(__dirname, "..", "workflows", "ci.yml");
  const workflow = fs.readFileSync(workflowPath, "utf8").replaceAll("\r\n", "\n");
  assert.match(workflow, /^name: Task Processor CI\n/);
  assert.match(workflow, /group: task-processor-ci-\$\{\{/);
  assert.match(workflow, /f"### Task Processor CI：\{overall\}"/);
  assert.doesNotMatch(workflow, /f"### ListingKit CI：\{overall\}"/);
  assert.match(workflow, /\n  required-gate:\n[\s\S]*?name: Required CI Gate/);
});

test("isolated runtime classification is limited to its owned harness and CI contract", () => {
  for (const changedPath of [
    "scripts/issue357/contract.mjs",
    "scripts/issue357/lifecycle.test.mjs",
    "scripts/issue357-runtime.mjs",
    "web/listingkit-ui/scripts/current-application-final-acceptance.mjs",
    ".github/workflows/ci.yml",
    ".github/scripts/ci-change-classifier.cjs",
    ".github/scripts/ci-change-classifier.test.cjs",
  ]) {
    assert.equal(classifyChangedPaths([changedPath]).isolated_runtime, true, changedPath);
  }

  for (const changedPath of [
    "internal/product/sourcing/service.go",
    "web/listingkit-ui/src/app/page.tsx",
    "extensions/product-capture/src/extractor.ts",
    "README.md",
  ]) {
    assert.equal(classifyChangedPaths([changedPath]).isolated_runtime, false, changedPath);
  }

  assert.equal(classifyChangedPaths([], { full: true }).isolated_runtime, true);
});

test("isolated runtime applicability reaches its job and required gate", () => {
  const workflowPath = path.join(__dirname, "..", "workflows", "ci.yml");
  const workflow = fs.readFileSync(workflowPath, "utf8").replaceAll("\r\n", "\n");
  assert.match(workflow, /isolated_runtime: \$\{\{ steps\.classify\.outputs\.isolated_runtime \}\}/);

  const job = workflow.slice(
    workflow.indexOf("\n  isolated-runtime-contracts:\n"),
    workflow.indexOf("\n  release-authority:\n"),
  );
  assert.match(job, /needs: changes/);
  assert.match(
    job,
    /github\.event_name == 'pull_request' && needs\.changes\.outputs\.isolated_runtime == 'true'/,
  );

  const required = workflow.slice(
    workflow.indexOf("\n  required-gate:\n"),
    workflow.indexOf("\n  notify:\n"),
  );
  assert.match(required, /ISOLATED_RUNTIME_SELECTED: \$\{\{ needs\.changes\.outputs\.isolated_runtime \}\}/);
});

test("classification diff disables rename detection so both source and destination paths are visible", () => {
  const workflowPath = path.join(__dirname, "..", "workflows", "ci.yml");
  const workflow = fs.readFileSync(workflowPath, "utf8").replaceAll("\r\n", "\n");
  assert.match(
    workflow,
    /git diff --no-renames --name-only "\$BASE_SHA\.\.\.\$HEAD_SHA"/,
  );
});

test("pull requests always run repository-wide Go architecture contracts", () => {
  const workflowPath = path.join(__dirname, "..", "workflows", "ci.yml");
  const workflow = fs.readFileSync(workflowPath, "utf8").replaceAll("\r\n", "\n");
  const architectureJob = workflow.slice(
    workflow.indexOf("\n  architecture-contracts:\n"),
    workflow.indexOf("\n  release-authority:\n"),
  );
  const requiredGate = workflow.slice(
    workflow.indexOf("\n  required-gate:\n"),
    workflow.indexOf("\n  notify:\n"),
  );

  assert.match(architectureJob, /name: Architecture Contract Guards/);
  assert.match(architectureJob, /if: \$\{\{ github\.event_name == 'pull_request' \}\}/);
  assert.match(architectureJob, /run: go test \.\/tests\/\.\.\. -count=1/);
  assert.match(architectureJob, /go test -tags issue357 \.\/internal\/app\/httpapi/);
  assert.match(requiredGate, /needs:[\s\S]*- architecture-contracts\b/);
  assert.match(
    requiredGate,
    /ARCHITECTURE_CONTRACT_RESULT:\s*\$\{\{\s*needs\.architecture-contracts\.result\s*\}\}/,
  );
  assert.match(requiredGate, /require_success "architecture contracts" "\$ARCHITECTURE_CONTRACT_RESULT"/);
});

test("normalizes windows paths and emits github outputs", () => {
  const result = classifyChangedPaths(["web\\listingkit-ui\\src\\app.tsx"]);
  assert.equal(result.frontend, true);
  assert.equal(formatOutputs(result), [
    "backend=false",
    "frontend=true",
    "code_health=true",
    "release_authority=false",
    "capture_extension=false",
    "isolated_runtime=false",
  ].join("\n"));
});

test("capture extension has a precise independent classification including its CI owners", () => {
  for (const changedPath of [
    "extensions/product-capture/src/extractor.ts", "extensions\\product-capture\\package-lock.json",
    ".github/workflows/ci.yml", ".github/scripts/ci-change-classifier.cjs", ".github/scripts/ci-change-classifier.test.cjs",
  ]) assert.equal(classifyChangedPaths([changedPath]).capture_extension, true, changedPath);
  for (const changedPath of ["README.md",
    "web/listingkit-ui/src/app.tsx", "internal/product/sourcing/service.go", ".github/workflows/listingkit-deploy.yml"]) {
    assert.equal(classifyChangedPaths([changedPath]).capture_extension, false, changedPath);
  }
  const extension=classifyChangedPaths(["extensions/product-capture/src/extractor.ts"]);
  assert.deepEqual([extension.backend,extension.frontend,extension.code_health,extension.release_authority],[false,false,false,false]);
  assert.equal(classifyChangedPaths([],{full:true}).capture_extension,true);
});

function requiredGate() {
  const workflow=fs.readFileSync(path.join(__dirname,"..","workflows","ci.yml"),"utf8").replaceAll("\r\n","\n");
  const required=workflow.slice(workflow.indexOf("\n  required-gate:\n"),workflow.indexOf("\n  notify:\n"));
  const script=required.slice(required.indexOf("        run: |\n")+"        run: |\n".length).split("\n").map(line=>line.startsWith("          ")?line.slice(10):line).join("\n");
  return {workflow,required,script};
}

test("capture classification output reaches its isolated job and required gate", () => {
  const {workflow,required}=requiredGate();
  assert.match(workflow,/capture_extension: \$\{\{ steps\.classify\.outputs\.capture_extension \}\}/);
  const job=workflow.slice(workflow.indexOf("\n  capture-extension:\n"),workflow.indexOf("\n  code-health:\n"));
  assert.match(job,/needs\.changes\.outputs\.capture_extension == 'true'/);
  assert.match(job,/node-version: "24"/);
  assert.match(job,/working-directory: extensions\/product-capture/);
  assert.match(job,/cache-dependency-path: extensions\/product-capture\/package-lock\.json/);
  for(const command of ["npm ci","npm test","npm run typecheck","npm run lint","npm run build -- --fixture"]) assert.ok(job.includes(command),command);
  assert.match(required,/needs:[\s\S]*- capture-extension\b/);
  assert.match(required,/CAPTURE_EXTENSION_RESULT: \$\{\{ needs\.capture-extension\.result \}\}/);
  assert.match(required,/CAPTURE_EXTENSION_SELECTED: \$\{\{ needs\.changes\.outputs\.capture_extension \}\}/);
});

const bash = process.platform === "win32" ? "C:/Program Files/Git/bin/bash.exe" : "bash";
const conditionalOutputs = {
  ISOLATED_RUNTIME: "isolated_runtime",
  RELEASE: "release_authority",
  BACKEND: "backend",
  FRONTEND: "frontend",
  CAPTURE_EXTENSION: "capture_extension",
  CODE_HEALTH: "code_health",
};
const conditionalJobs = Object.keys(conditionalOutputs);
const requiredOnPR = ["CHANGES", "DEVELOPMENT_ADMISSION_TEST", "ARCHITECTURE_CONTRACT"];

function runShell(script, env) {
  const run = spawnSync(bash, ["-c", script], { env: { ...process.env, ...env }, encoding: "utf8" });
  if (run.error) throw run.error;
  return run;
}

function gateEnvironment(event) {
  const push = event === "push";
  const env = { CI_EVENT: event };
  for (const job of requiredOnPR) env[`${job}_RESULT`] = "success";
  for (const job of conditionalJobs) {
    env[`${job}_SELECTED`] = push ? "true" : "false";
    env[`${job}_RESULT`] = push ? "success" : "skipped";
  }
  if (push) {
    env.ARCHITECTURE_CONTRACT_RESULT = "skipped";
    env.ISOLATED_RUNTIME_RESULT = "skipped";
  }
  return env;
}

test("the actual gate permits explicitly inapplicable jobs and fails closed for missing applicability or required results", () => {
  const { required, script } = requiredGate();
  const env = gateEnvironment("pull_request");
  assert.equal(runShell(script, env).status, 0);
  for (const job of requiredOnPR) {
    assert.notEqual(runShell(script, { ...env, [`${job}_RESULT`]: "skipped" }).status, 0, job);
  }
  for (const job of conditionalJobs) {
    const selected = `${job}_SELECTED`;
    const result = `${job}_RESULT`;
    assert.ok(required.includes(`${selected}: \${{ needs.changes.outputs.${conditionalOutputs[job]} }}`), selected);
    assert.equal(runShell(script, { ...env, [selected]: "true", [result]: "success" }).status, 0, job);
    for (const value of ["skipped", "cancelled", "failure", ""]) {
      assert.notEqual(runShell(script, { ...env, [selected]: "true", [result]: value }).status, 0, `${job}/${value}`);
    }
    for (const value of ["", "unknown"]) {
      assert.notEqual(runShell(script, { ...env, [selected]: value }).status, 0, `${job}/selection=${value}`);
    }
    assert.notEqual(runShell(script, { ...env, [result]: "cancelled" }).status, 0, job);
  }
});

test("the actual main push gate keeps all existing push jobs required", () => {
  const { script, workflow } = requiredGate();
  assert.match(workflow, /node \.github\/scripts\/ci-change-classifier\.cjs --all >> "\$GITHUB_OUTPUT"/);
  const env = gateEnvironment("push");
  assert.equal(runShell(script, env).status, 0);
  for (const job of conditionalJobs) {
    assert.notEqual(runShell(script, { ...env, [`${job}_SELECTED`]: "false" }).status, 0, job);
    if (job !== "ISOLATED_RUNTIME") {
      assert.notEqual(runShell(script, { ...env, [`${job}_RESULT`]: "skipped" }).status, 0, job);
    }
  }
});

test("the actual backend step runs all packages on push and leaves only the architecture subtree out of PR backend tests", () => {
  const { workflow } = requiredGate();
  const backend = workflow.match(/- name: Run backend tests\n        shell: bash\n        run: \|\n([\s\S]*?)\n      - name:/);
  assert.ok(backend, "backend step must expose the event-specific shell command");
  const script = backend[1].split("\n").map((line) => line.slice(10)).join("\n");
  const mockGo = [
    "go() {",
    '  if [[ "$1" == list ]]; then',
    '    if [[ "${LIST_FAIL:-}" == true ]]; then return 12; fi',
    "    printf '%s\\n' task-processor/tests task-processor/tests/child task-processor/testsibling task-processor/internal/app",
    "  else",
    '    printf "%s\\n" "$@"',
    "  fi",
    "}",
  ].join("\n");
  const run = (event, extra = {}) => runShell(`${mockGo}\n${script}`, { GITHUB_EVENT_NAME: event, LIST_FAIL: "false", ...extra });
  const push = run("push", { LIST_FAIL: "true" });
  assert.equal(push.status, 0, push.stderr);
  assert.deepEqual(push.stdout.trim().split(/\r?\n/), ["test", "./...", "-count=1"]);
  const pr = run("pull_request");
  assert.equal(pr.status, 0, pr.stderr);
  assert.deepEqual(pr.stdout.trim().split(/\r?\n/), ["test", "-count=1", "task-processor/testsibling", "task-processor/internal/app"]);
  assert.notEqual(run("pull_request", { LIST_FAIL: "true" }).status, 0);
});
