"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { isDocumentationMarkdown, parsePatchDestination, addedMarkdownLinks, checkAddedLinks } = require("./ci-document-links.cjs");

test("only repository Markdown paths are treated as documentation link inputs", () => {
  assert.equal(isDocumentationMarkdown("docs/product/README.md"), true);
  assert.equal(isDocumentationMarkdown("README.md"), true);
  assert.equal(isDocumentationMarkdown("AGENTS.md"), true);
  assert.equal(isDocumentationMarkdown("docs/document-inventory.csv"), false);
  assert.equal(isDocumentationMarkdown("internal/pkg/doc.go"), false);
});

test("new relative documentation references are checked but old and removed lines are ignored", () => {
  const patch = [
    "diff --git a/docs/README.md b/docs/README.md",
    "--- a/docs/README.md",
    "+++ b/docs/README.md",
    "@@ -1,2 +1,3 @@",
    "-[removed](missing-old.md)",
    " [unchanged](missing-unchanged.md)",
    "+[valid](architecture/README.md#section) and [missing](architecture/missing.md)",
    "+[web](https://example.com), [anchor](#section), [email](mailto:foo@example.com)",
    "diff --git a/docs/document-inventory.csv b/docs/document-inventory.csv",
    "--- /dev/null",
    "+++ b/docs/document-inventory.csv",
    "@@ -0,0 +1,1 @@",
    "+path,layer,domain",
  ].join("\n");
  const links = addedMarkdownLinks(patch);
  assert.deepEqual(links.map((x) => x.href), [
    "architecture/README.md#section",
    "architecture/missing.md",
    "https://example.com",
    "#section",
    "mailto:foo@example.com",
  ]);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-"));
  try {
    fs.mkdirSync(path.join(root, "docs", "architecture"), { recursive: true });
    fs.writeFileSync(path.join(root, "docs", "architecture", "README.md"), "ok");
    const file = [
      "[unchanged](missing-unchanged.md)",
      "[valid](architecture/README.md#section) and [missing](architecture/missing.md)",
      "[web](https://example.com), [anchor](#section), [email](mailto:foo@example.com)",
    ].join("\n");
    assert.deepEqual(checkAddedLinks(patch, { repositoryRoot: root, readFile: () => file }), [
      "docs/README.md: unresolved relative link architecture/missing.md",
    ]);
    fs.writeFileSync(path.join(root, "docs", "architecture", "missing.md"), "ok");
    assert.deepEqual(checkAddedLinks(patch, { repositoryRoot: root, readFile: () => file }), []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("unrecognized and malformed relative targets fail closed, while valid encoded names work", () => {
  const patch = [
    "diff --git a/docs/README.md b/docs/README.md",
    "+++ b/docs/README.md",
    "@@ -1,0 +2 @@",
    "+[out](../../outside.md), [bad](bad%ZZ.md), [encoded](file%20name.md)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-"));
  try {
    fs.mkdirSync(path.join(root, "docs"), { recursive: true });
    fs.writeFileSync(path.join(root, "docs", "file name.md"), "ok");
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "# Intro\n[out](../../outside.md), [bad](bad%ZZ.md), [encoded](file%20name.md)",
    }), [
      "docs/README.md: unresolved relative link ../../outside.md",
      "docs/README.md: invalid URL encoding in bad%ZZ.md",
    ]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("new document with directory links is validated without checking removed content", () => {
  const patch = [
    "diff --git a/docs/product/flows/README.md b/docs/product/flows/README.md",
    "--- /dev/null",
    "+++ b/docs/product/flows/README.md",
    "@@ -0,0 +1,2 @@",
    "+# Flows",
    "+[platform](../../../internal/marketplace/) and [products](../README.md)",
  ].join("\n");
  assert.deepEqual(checkAddedLinks(patch, {
    repositoryRoot: "/tmp/example-repo",
    exists: (p) => p === "/tmp/example-repo/internal/marketplace" ||
      p === "/tmp/example-repo/docs/product/README.md",
    readFile: () => "# Flows\n[platform](../../../internal/marketplace/) and [products](../README.md)",
  }), []);
});

test("normalizes Git patch headers for spaces and C-style quoted filenames", () => {
  assert.equal(parsePatchDestination("+++ b/docs/guide notes.md\t"), "docs/guide notes.md");
  assert.equal(parsePatchDestination('+++ "b/docs/a\\"quoted.md"'), 'docs/a"quoted.md');
  assert.equal(parsePatchDestination('+++ "b/docs/tab\\tnotes.md"'), "docs/tab\tnotes.md");
  assert.equal(parsePatchDestination('+++ "b/docs/\\344\\270\\255\\346\\226\\207.md"'), "docs/中文.md");
  assert.equal(parsePatchDestination("+++ /dev/null"), null);
  assert.throws(() => parsePatchDestination('+++ "b/docs/unterminated'), /malformed Git patch filename/);
  assert.throws(() => parsePatchDestination("+++ unknown-target"), /unexpected Git patch destination/);

  const patch = [
    "diff --git a/docs/guide notes.md b/docs/guide notes.md",
    "--- a/docs/guide notes.md\t",
    "+++ b/docs/guide notes.md\t",
    "@@ -1,0 +2 @@",
    "+[invalid](missing.md)",
  ].join("\n");
  assert.deepEqual(addedMarkdownLinks(patch), [
    { filename: "docs/guide notes.md", href: "missing.md" },
  ]);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-spaces-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "# Notes\n[invalid](missing.md)",
    }), [
      "docs/guide notes.md: unresolved relative link missing.md",
    ]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("accepts repository-root links but refuses paths that escape it", () => {
  const patch = [
    "diff --git a/docs/README.md b/docs/README.md",
    "--- a/docs/README.md",
    "+++ b/docs/README.md",
    "@@ -1,0 +2 @@",
    "+[repository](../) and [outside](../../)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-root-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "# Intro\n[repository](../) and [outside](../../)",
    }), [
      "docs/README.md: unresolved relative link ../../",
    ]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("skips literal examples in existing fences and inline code but validates real added links", () => {
  const patch = [
    "diff --git a/docs/guide.md b/docs/guide.md",
    "--- a/docs/guide.md",
    "+++ b/docs/guide.md",
    "@@ -2,0 +3 @@",
    "+[fenced](missing-fence.md)",
    "@@ -4,0 +5 @@",
    "+\x60[inline](missing-inline.md)\x60 and [real](missing-real.md)",
  ].join("\n");
  const content = [
    "# Guide",
    "\x60\x60\x60md",
    "[fenced](missing-fence.md)",
    "\x60\x60\x60",
    "\x60[inline](missing-inline.md)\x60 and [real](missing-real.md)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-code-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => content,
    }), [
      "docs/guide.md: unresolved relative link missing-real.md",
    ]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("skips Markdown links appearing inside newly added fenced blocks", () => {
  const patch = [
    "diff --git a/docs/examples.md b/docs/examples.md",
    "--- /dev/null",
    "+++ b/docs/examples.md",
    "@@ -0,0 +1,3 @@",
    "+\x60\x60\x60md",
    "+[sample](not-a-real-file.md)",
    "+\x60\x60\x60",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-fence-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "\x60\x60\x60md\n[sample](not-a-real-file.md)\n\x60\x60\x60",
    }), []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("an unmatched backtick is literal and must not mask subsequent broken links", () => {
  const patch = [
    "diff --git a/docs/guide.md b/docs/guide.md",
    "+++ b/docs/guide.md",
    "@@ -1,0 +2 @@",
    "+[broken](missing.md)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-unmatched-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "A literal \x60 character\n[broken](missing.md)",
    }), ["docs/guide.md: unresolved relative link missing.md"]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("matched multiline code spans suppress only their examples, not following links", () => {
  const patch = [
    "diff --git a/docs/multiline.md b/docs/multiline.md",
    "+++ b/docs/multiline.md",
    "@@ -0,0 +1,3 @@",
    "+Start \x60open",
    "+[example](not-a-file.md)\x60 after",
    "+[real](missing.md)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-multiline-"));
  try {
    fs.mkdirSync(path.join(root, "docs"));
    assert.deepEqual(checkAddedLinks(patch, {
      repositoryRoot: root,
      readFile: () => "Start \x60open\n[example](not-a-file.md)\x60 after\n[real](missing.md)",
    }), ["docs/multiline.md: unresolved relative link missing.md"]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("CLI validates against PR HEAD documents despite unrelated lines in the checkout", () => {
  const { spawnSync } = require("node:child_process");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-head-"));
  const git = (...args) => {
    const result = spawnSync("git", args, { cwd: root, encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout.trim();
  };
  try {
    git("init");
    git("config", "user.name", "CI Test");
    git("config", "user.email", "ci@example.invalid");
    fs.mkdirSync(path.join(root, "docs"));
    const doc = path.join(root, "docs", "guide.md");
    fs.writeFileSync(doc, "# Guide\nOriginal line\n");
    git("add", ".");
    git("commit", "-m", "base");
    const base = git("rev-parse", "HEAD");
    fs.writeFileSync(doc, "# Guide\nOriginal line\n[broken](missing.md)\n");
    git("add", ".");
    git("commit", "-m", "head");
    const head = git("rev-parse", "HEAD");
    // The checked-out tree differs from the immutable HEAD being reviewed.
    fs.writeFileSync(doc, "# Extra line from main\n# Guide\nOriginal line\n[broken](missing.md)\n");
    const child = spawnSync(process.execPath, [
      path.join(__dirname, "ci-document-links.cjs"), base, head,
    ], { cwd: root, encoding: "utf8" });
    assert.notEqual(child.status, 0);
    assert.match(child.stderr, /docs\/guide\.md: unresolved relative link missing\.md/);

    // A real target in the immutable HEAD must be accepted, even when the
    // worktree document has unrelated uncommitted extra lines.
    fs.writeFileSync(path.join(root, "docs", "missing.md"), "# Target\n");
    git("add", "docs/missing.md");
    git("commit", "-m", "add referenced target");
    const validHead = git("rev-parse", "HEAD");
    const valid = spawnSync(process.execPath, [
      path.join(__dirname, "ci-document-links.cjs"), base, validHead,
    ], { cwd: root, encoding: "utf8" });
    assert.equal(valid.status, 0, valid.stderr);
    assert.match(valid.stdout, /Verified 1 added Markdown links/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
